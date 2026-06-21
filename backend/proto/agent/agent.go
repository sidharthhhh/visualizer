package agent

import (
	"context"
	"fmt"
	"log/slog"
	"net"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/containerscope/backend/internal/ws"
)

type AgentServer struct {
	pool   *pgxpool.Pool
	logger *slog.Logger
	hub    *ws.Hub
}

func NewAgentServer(pool *pgxpool.Pool, logger *slog.Logger, hub *ws.Hub) *AgentServer {
	return &AgentServer{pool: pool, logger: logger, hub: hub}
}

func (s *AgentServer) Start(addr string) error {
	lis, err := net.Listen("tcp", addr)
	if err != nil {
		return err
	}

	grpcServer := grpc.NewServer()
	RegisterAgentServiceServer(grpcServer, s)

	s.logger.Info("gRPC server starting", "addr", addr)
	return grpcServer.Serve(lis)
}

func (s *AgentServer) Enroll(ctx context.Context, req *EnrollRequest) (*EnrollResponse, error) {
	if req.EnrollmentToken == "" {
		return nil, status.Error(codes.InvalidArgument, "enrollment token is required")
	}

	conn, err := s.getConnectionByToken(ctx, req.EnrollmentToken)
	if err != nil {
		return nil, status.Error(codes.NotFound, "invalid enrollment token")
	}

	if req.HostInfo != nil {
		if err := s.createHost(ctx, conn.ID, req.HostInfo); err != nil {
			s.logger.Error("creating host", "error", err)
		}
	}

	if err := s.updateConnectionStatus(ctx, conn.ID, "connected"); err != nil {
		s.logger.Error("updating connection status", "error", err)
	}

	if s.hub != nil {
		s.hub.Broadcast(conn.OrgID, conn.ID, &ws.Message{
			Type: ws.TypeStatusChange,
			Payload: map[string]interface{}{
				"connection_id": conn.ID.String(),
				"status":        "connected",
			},
		})
	}

	return &EnrollResponse{
		ConnectionId: conn.ID.String(),
		Status:       "connected",
	}, nil
}

func (s *AgentServer) Heartbeat(ctx context.Context, req *HeartbeatRequest) (*HeartbeatResponse, error) {
	if req.ConnectionId == "" {
		return nil, status.Error(codes.InvalidArgument, "connection id is required")
	}

	connID, err := uuid.Parse(req.ConnectionId)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid connection id")
	}

	if err := s.updateLastSeen(ctx, connID); err != nil {
		s.logger.Error("updating last seen", "error", err)
		return nil, status.Error(codes.Internal, "failed to update heartbeat")
	}

	return &HeartbeatResponse{
		Status: "ok",
	}, nil
}

func (s *AgentServer) SyncTopology(ctx context.Context, req *TopologySnapshot) (*TopologyResponse, error) {
	if req.ConnectionId == "" {
		return nil, status.Error(codes.InvalidArgument, "connection id is required")
	}

	connID, err := uuid.Parse(req.ConnectionId)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid connection id")
	}

	conn, err := s.getConnectionByID(ctx, connID)
	if err != nil {
		return nil, status.Error(codes.NotFound, "connection not found")
	}

	for _, ctr := range req.Containers {
		if err := s.upsertContainer(ctx, connID, ctr); err != nil {
			s.logger.Error("upserting container", "error", err, "runtime_id", ctr.RuntimeId)
		}
	}

	for _, net := range req.Networks {
		if err := s.upsertNetwork(ctx, connID, net); err != nil {
			s.logger.Error("upserting network", "error", err, "name", net.Name)
		}
	}

	if err := s.markRemovedContainers(ctx, connID, req.Containers); err != nil {
		s.logger.Error("marking removed containers", "error", err)
	}

	if s.hub != nil {
		s.hub.Broadcast(conn.OrgID, connID, &ws.Message{
			Type: ws.TypeTopologyUpdate,
			Payload: map[string]interface{}{
				"connection_id": connID.String(),
				"containers":    len(req.Containers),
				"networks":      len(req.Networks),
			},
		})
	}

	s.logger.Info("topology synced",
		"connection_id", connID,
		"containers", len(req.Containers),
		"networks", len(req.Networks),
	)

	return &TopologyResponse{Status: "ok"}, nil
}

func (s *AgentServer) StreamTopologyEvents(stream AgentService_StreamTopologyEventsServer) error {
	for {
		event, err := stream.Recv()
		if err != nil {
			return err
		}

		connID, err := uuid.Parse(event.ConnectionId)
		if err != nil {
			s.logger.Error("invalid connection id", "error", err)
			continue
		}

		conn, err := s.getConnectionByID(stream.Context(), connID)
		if err != nil {
			s.logger.Error("getting connection", "error", err)
			continue
		}

		if event.Container != nil {
			if err := s.upsertContainer(stream.Context(), connID, event.Container); err != nil {
				s.logger.Error("upserting container from event", "error", err)
			}

			if s.hub != nil {
				var msgType ws.MessageType
				switch event.EventType {
				case "create":
					msgType = ws.TypeContainerAdd
				case "destroy":
					msgType = ws.TypeContainerDel
				default:
					msgType = ws.TypeContainerUpd
				}

				s.hub.Broadcast(conn.OrgID, connID, &ws.Message{
					Type: msgType,
					Payload: map[string]interface{}{
						"runtime_id": event.Container.RuntimeId,
						"name":       event.Container.Name,
						"image":      event.Container.Image,
						"state":      event.Container.State,
					},
				})
			}
		}

		s.logger.Debug("topology event received",
			"type", event.EventType,
			"connection_id", connID,
		)
	}
}

func (s *AgentServer) StreamMetrics(stream AgentService_StreamMetricsServer) error {
	for {
		batch, err := stream.Recv()
		if err != nil {
			return err
		}

		connID, err := uuid.Parse(batch.ConnectionId)
		if err != nil {
			s.logger.Error("invalid connection id", "error", err)
			continue
		}

		for _, sample := range batch.Samples {
			if err := s.insertMetric(stream.Context(), connID, sample); err != nil {
				s.logger.Error("inserting metric", "error", err)
			}
		}

		s.logger.Debug("metrics received",
			"connection_id", connID,
			"samples", len(batch.Samples),
		)
	}
}

func (s *AgentServer) insertMetric(ctx context.Context, connID uuid.UUID, sample *MetricSample) error {
	_, err := s.pool.Exec(ctx,
		`INSERT INTO metric_samples (connection_id, runtime_id, metric, value, timestamp)
		 VALUES ($1, $2, $3, $4, $5)`,
		connID, sample.RuntimeId, sample.Metric, sample.Value, time.Unix(sample.Timestamp, 0),
	)
	return err
}

func (s *AgentServer) SyncEdges(ctx context.Context, req *EdgeSnapshot) (*EdgeResponse, error) {
	if req.ConnectionId == "" {
		return nil, status.Error(codes.InvalidArgument, "connection id is required")
	}

	connID, err := uuid.Parse(req.ConnectionId)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid connection id")
	}

	if err := s.replaceEdges(ctx, connID, req.Edges); err != nil {
		s.logger.Error("replacing edges", "error", err)
		return nil, status.Error(codes.Internal, "failed to sync edges")
	}

	s.logger.Info("edges synced",
		"connection_id", connID,
		"edges", len(req.Edges),
	)

	return &EdgeResponse{Status: "ok"}, nil
}

func (s *AgentServer) replaceEdges(ctx context.Context, connID uuid.UUID, edges []*EdgeInfo) error {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("beginning transaction: %w", err)
	}
	defer tx.Rollback(ctx)

	_, err = tx.Exec(ctx,
		`DELETE FROM edges WHERE connection_id = $1`,
		connID,
	)
	if err != nil {
		return fmt.Errorf("deleting old edges: %w", err)
	}

	for _, edge := range edges {
		_, err = tx.Exec(ctx,
			`INSERT INTO edges (connection_id, src_container_id, dst_container_id, dst_ip, dst_port, protocol)
			 VALUES ($1, $2, $3, $4, $5, $6)`,
			connID, edge.SrcContainerId, edge.DstContainerId, edge.DstIp, edge.DstPort, edge.Protocol,
		)
		if err != nil {
			return fmt.Errorf("inserting edge: %w", err)
		}
	}

	return tx.Commit(ctx)
}

type Connection struct {
	ID     uuid.UUID
	OrgID  uuid.UUID
	Type   string
	Name   string
	Status string
}

func (s *AgentServer) getConnectionByToken(ctx context.Context, token string) (*Connection, error) {
	var conn Connection
	err := s.pool.QueryRow(ctx,
		`SELECT id, org_id, type, name, status FROM connections WHERE agent_token = $1`,
		token,
	).Scan(&conn.ID, &conn.OrgID, &conn.Type, &conn.Name, &conn.Status)
	if err != nil {
		return nil, err
	}
	return &conn, nil
}

func (s *AgentServer) getConnectionByID(ctx context.Context, id uuid.UUID) (*Connection, error) {
	var conn Connection
	err := s.pool.QueryRow(ctx,
		`SELECT id, org_id, type, name, status FROM connections WHERE id = $1`,
		id,
	).Scan(&conn.ID, &conn.OrgID, &conn.Type, &conn.Name, &conn.Status)
	if err != nil {
		return nil, err
	}
	return &conn, nil
}

func (s *AgentServer) createHost(ctx context.Context, connID uuid.UUID, info *HostInfo) error {
	_, err := s.pool.Exec(ctx,
		`INSERT INTO hosts (connection_id, hostname, os, kernel, cpu_cores, mem_total, agent_version)
		 VALUES ($1, $2, $3, $4, $5, $6, $7)
		 ON CONFLICT (connection_id) DO UPDATE SET
		   hostname = EXCLUDED.hostname,
		   os = EXCLUDED.os,
		   kernel = EXCLUDED.kernel,
		   cpu_cores = EXCLUDED.cpu_cores,
		   mem_total = EXCLUDED.mem_total,
		   agent_version = EXCLUDED.agent_version`,
		connID, info.Hostname, info.Os, info.Kernel, info.CpuCores, info.MemTotal, info.AgentVersion,
	)
	return err
}

func (s *AgentServer) updateConnectionStatus(ctx context.Context, connID uuid.UUID, status string) error {
	_, err := s.pool.Exec(ctx,
		`UPDATE connections SET status = $1, last_seen_at = NOW(), updated_at = NOW() WHERE id = $2`,
		status, connID,
	)
	return err
}

func (s *AgentServer) updateLastSeen(ctx context.Context, connID uuid.UUID) error {
	_, err := s.pool.Exec(ctx,
		`UPDATE connections SET last_seen_at = NOW() WHERE id = $1`,
		connID,
	)
	return err
}

func (s *AgentServer) upsertContainer(ctx context.Context, connID uuid.UUID, ctr *ContainerInfo) error {
	_, err := s.pool.Exec(ctx,
		`INSERT INTO containers (connection_id, runtime_id, name, image, state, labels, ports)
		 VALUES ($1, $2, $3, $4, $5, $6, $7)
		 ON CONFLICT (connection_id, runtime_id) DO UPDATE SET
		   name = EXCLUDED.name,
		   image = EXCLUDED.image,
		   state = EXCLUDED.state,
		   labels = EXCLUDED.labels,
		   ports = EXCLUDED.ports,
		   updated_at = NOW(),
		   removed_at = NULL`,
		connID, ctr.RuntimeId, ctr.Name, ctr.Image, ctr.State, ctr.Labels, ctr.Ports,
	)
	return err
}

func (s *AgentServer) upsertNetwork(ctx context.Context, connID uuid.UUID, net *NetworkInfo) error {
	_, err := s.pool.Exec(ctx,
		`INSERT INTO networks (connection_id, name, driver, scope, subnet)
		 VALUES ($1, $2, $3, $4, $5)
		 ON CONFLICT (connection_id, name) DO UPDATE SET
		   driver = EXCLUDED.driver,
		   scope = EXCLUDED.scope,
		   subnet = EXCLUDED.subnet`,
		connID, net.Name, net.Driver, net.Scope, net.Subnet,
	)
	return err
}

func (s *AgentServer) markRemovedContainers(ctx context.Context, connID uuid.UUID, active []*ContainerInfo) error {
	activeIDs := make([]string, 0, len(active))
	for _, ctr := range active {
		activeIDs = append(activeIDs, ctr.RuntimeId)
	}

	_, err := s.pool.Exec(ctx,
		`UPDATE containers SET removed_at = NOW(), updated_at = NOW()
		 WHERE connection_id = $1 AND removed_at IS NULL AND runtime_id != ALL($2)`,
		connID, activeIDs,
	)
	return err
}

type HostInfo struct {
	Hostname     string
	Os           string
	Kernel       string
	CpuCores     int32
	MemTotal     int64
	AgentVersion string
}

type EnrollRequest struct {
	EnrollmentToken string
	HostInfo        *HostInfo
}

type EnrollResponse struct {
	ConnectionId string
	Status       string
}

type HeartbeatRequest struct {
	ConnectionId string
	Timestamp    int64
}

type HeartbeatResponse struct {
	Status string
}

type TopologySnapshot struct {
	ConnectionId string
	Containers   []*ContainerInfo
	Networks     []*NetworkInfo
	Timestamp    int64
}

type ContainerInfo struct {
	RuntimeId string
	Name      string
	Image     string
	State     string
	Labels    map[string]string
	Ports     []*PortMapping
	CreatedAt int64
}

type PortMapping struct {
	HostPort      string
	ContainerPort string
	Protocol      string
}

type NetworkInfo struct {
	Name   string
	Driver string
	Scope  string
	Subnet string
}

type TopologyEvent struct {
	ConnectionId string
	EventType    string
	Container    *ContainerInfo
	Timestamp    int64
}

type TopologyResponse struct {
	Status string
}

type MetricBatch struct {
	ConnectionId string
	Samples      []*MetricSample
	Timestamp    int64
}

type MetricSample struct {
	RuntimeId string
	Metric    string
	Value     float64
	Timestamp int64
}

type MetricResponse struct {
	Status string
}

type EdgeSnapshot struct {
	ConnectionId string
	Edges        []*EdgeInfo
	Timestamp    int64
}

type EdgeInfo struct {
	SrcContainerId string
	DstContainerId string
	DstIp          string
	DstPort        int32
	Protocol       string
}

type EdgeResponse struct {
	Status string
}

type AgentServiceServer interface {
	Enroll(context.Context, *EnrollRequest) (*EnrollResponse, error)
	Heartbeat(context.Context, *HeartbeatRequest) (*HeartbeatResponse, error)
	SyncTopology(context.Context, *TopologySnapshot) (*TopologyResponse, error)
	StreamTopologyEvents(AgentService_StreamTopologyEventsServer) error
	StreamMetrics(AgentService_StreamMetricsServer) error
	SyncEdges(context.Context, *EdgeSnapshot) (*EdgeResponse, error)
}

type AgentService_StreamTopologyEventsServer interface {
	Recv() (*TopologyEvent, error)
	Send(*TopologyResponse) error
	Context() context.Context
	grpc.ServerStream
}

type AgentService_StreamMetricsServer interface {
	Recv() (*MetricBatch, error)
	Send(*MetricResponse) error
	Context() context.Context
	grpc.ServerStream
}

func RegisterAgentServiceServer(s *grpc.Server, srv AgentServiceServer) {
	s.RegisterService(&grpc.ServiceDesc{
		ServiceName: "containerscope.agent.AgentService",
		HandlerType: (*AgentServiceServer)(nil),
		Methods: []grpc.MethodDesc{
			{
				MethodName: "Enroll",
				Handler: func(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
					in := new(EnrollRequest)
					if err := dec(in); err != nil {
						return nil, err
					}
					if interceptor == nil {
						return srv.(AgentServiceServer).Enroll(ctx, in)
					}
					info := &grpc.UnaryServerInfo{
						Server:     srv,
						FullMethod: "/containerscope.agent.AgentService/Enroll",
					}
					handler := func(ctx context.Context, req interface{}) (interface{}, error) {
						return srv.(AgentServiceServer).Enroll(ctx, req.(*EnrollRequest))
					}
					return interceptor(ctx, in, info, handler)
				},
			},
			{
				MethodName: "Heartbeat",
				Handler: func(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
					in := new(HeartbeatRequest)
					if err := dec(in); err != nil {
						return nil, err
					}
					if interceptor == nil {
						return srv.(AgentServiceServer).Heartbeat(ctx, in)
					}
					info := &grpc.UnaryServerInfo{
						Server:     srv,
						FullMethod: "/containerscope.agent.AgentService/Heartbeat",
					}
					handler := func(ctx context.Context, req interface{}) (interface{}, error) {
						return srv.(AgentServiceServer).Heartbeat(ctx, req.(*HeartbeatRequest))
					}
					return interceptor(ctx, in, info, handler)
				},
			},
			{
				MethodName: "SyncTopology",
				Handler: func(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
					in := new(TopologySnapshot)
					if err := dec(in); err != nil {
						return nil, err
					}
					if interceptor == nil {
						return srv.(AgentServiceServer).SyncTopology(ctx, in)
					}
					info := &grpc.UnaryServerInfo{
						Server:     srv,
						FullMethod: "/containerscope.agent.AgentService/SyncTopology",
					}
					handler := func(ctx context.Context, req interface{}) (interface{}, error) {
						return srv.(AgentServiceServer).SyncTopology(ctx, req.(*TopologySnapshot))
					}
					return interceptor(ctx, in, info, handler)
				},
			},
			{
				MethodName: "SyncEdges",
				Handler: func(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
					in := new(EdgeSnapshot)
					if err := dec(in); err != nil {
						return nil, err
					}
					if interceptor == nil {
						return srv.(AgentServiceServer).SyncEdges(ctx, in)
					}
					info := &grpc.UnaryServerInfo{
						Server:     srv,
						FullMethod: "/containerscope.agent.AgentService/SyncEdges",
					}
					handler := func(ctx context.Context, req interface{}) (interface{}, error) {
						return srv.(AgentServiceServer).SyncEdges(ctx, req.(*EdgeSnapshot))
					}
					return interceptor(ctx, in, info, handler)
				},
			},
		},
		Streams: []grpc.StreamDesc{
			{
				StreamName:    "StreamTopologyEvents",
				Handler:       _AgentService_StreamTopologyEvents_Handler,
				ServerStreams: true,
				ClientStreams: true,
			},
			{
				StreamName:    "StreamMetrics",
				Handler:       _AgentService_StreamMetrics_Handler,
				ClientStreams: true,
			},
		},
	}, srv)
}

func _AgentService_StreamTopologyEvents_Handler(srv interface{}, stream grpc.ServerStream) error {
	return srv.(AgentServiceServer).StreamTopologyEvents(&agentServiceStreamTopologyEventsServer{stream})
}

type agentServiceStreamTopologyEventsServer struct {
	grpc.ServerStream
}

func (x *agentServiceStreamTopologyEventsServer) Recv() (*TopologyEvent, error) {
	m := new(TopologyEvent)
	if err := x.ServerStream.RecvMsg(m); err != nil {
		return nil, err
	}
	return m, nil
}

func (x *agentServiceStreamTopologyEventsServer) Send(resp *TopologyResponse) error {
	return x.ServerStream.SendMsg(resp)
}

func _AgentService_StreamMetrics_Handler(srv interface{}, stream grpc.ServerStream) error {
	return srv.(AgentServiceServer).StreamMetrics(&agentServiceStreamMetricsServer{stream})
}

type agentServiceStreamMetricsServer struct {
	grpc.ServerStream
}

func (x *agentServiceStreamMetricsServer) Recv() (*MetricBatch, error) {
	m := new(MetricBatch)
	if err := x.ServerStream.RecvMsg(m); err != nil {
		return nil, err
	}
	return m, nil
}

func (x *agentServiceStreamMetricsServer) Send(resp *MetricResponse) error {
	return x.ServerStream.SendMsg(resp)
}

type AgentClient struct {
	conn   *grpc.ClientConn
	logger *slog.Logger
}

func NewAgentClient(addr string, logger *slog.Logger) (*AgentClient, error) {
	conn, err := grpc.Dial(addr, grpc.WithInsecure())
	if err != nil {
		return nil, err
	}
	return &AgentClient{conn: conn, logger: logger}, nil
}

func (c *AgentClient) Close() error {
	return c.conn.Close()
}

func (c *AgentClient) Enroll(token string, info *HostInfo) (*EnrollResponse, error) {
	client := NewAgentServiceClient(c.conn)
	return client.Enroll(context.Background(), &EnrollRequest{
		EnrollmentToken: token,
		HostInfo:        info,
	})
}

func (c *AgentClient) Heartbeat(connectionID string) (*HeartbeatResponse, error) {
	client := NewAgentServiceClient(c.conn)
	return client.Heartbeat(context.Background(), &HeartbeatRequest{
		ConnectionId: connectionID,
		Timestamp:    time.Now().Unix(),
	})
}

func (c *AgentClient) SyncTopology(connectionID string, containers []*ContainerInfo, networks []*NetworkInfo) (*TopologyResponse, error) {
	client := NewAgentServiceClient(c.conn)
	return client.SyncTopology(context.Background(), &TopologySnapshot{
		ConnectionId: connectionID,
		Containers:   containers,
		Networks:     networks,
		Timestamp:    time.Now().Unix(),
	})
}

type AgentServiceClient interface {
	Enroll(ctx context.Context, req *EnrollRequest) (*EnrollResponse, error)
	Heartbeat(ctx context.Context, req *HeartbeatRequest) (*HeartbeatResponse, error)
	SyncTopology(ctx context.Context, req *TopologySnapshot) (*TopologyResponse, error)
}

func NewAgentServiceClient(cc *grpc.ClientConn) AgentServiceClient {
	return &agentServiceClient{cc}
}

type agentServiceClient struct {
	cc *grpc.ClientConn
}

func (c *agentServiceClient) Enroll(ctx context.Context, req *EnrollRequest) (*EnrollResponse, error) {
	out := new(EnrollResponse)
	err := c.cc.Invoke(ctx, "/containerscope.agent.AgentService/Enroll", req, out)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *agentServiceClient) Heartbeat(ctx context.Context, req *HeartbeatRequest) (*HeartbeatResponse, error) {
	out := new(HeartbeatResponse)
	err := c.cc.Invoke(ctx, "/containerscope.agent.AgentService/Heartbeat", req, out)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *agentServiceClient) SyncTopology(ctx context.Context, req *TopologySnapshot) (*TopologyResponse, error) {
	out := new(TopologyResponse)
	err := c.cc.Invoke(ctx, "/containerscope.agent.AgentService/SyncTopology", req, out)
	if err != nil {
		return nil, err
	}
	return out, nil
}
