package flows

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/ClickHouse/clickhouse-go/v2"
	"github.com/ClickHouse/clickhouse-go/v2/lib/driver"
)

type FlowEvent struct {
	Timestamp    time.Time
	ConnectionID string
	SrcIP        string
	DstIP        string
	SrcPort      uint16
	DstPort      uint16
	Protocol     string
	Bytes        uint64
	Packets      uint64
	LatencyMs    float64
}

type Client struct {
	conn   driver.Conn
	logger *slog.Logger
}

func NewClient(host string, port int, database, user, password string, logger *slog.Logger) (*Client, error) {
	conn, err := clickhouse.Open(&clickhouse.Options{
		Addr: []string{fmt.Sprintf("%s:%d", host, port)},
		Auth: clickhouse.Auth{
			Database: database,
			Username: user,
			Password: password,
		},
		Settings: clickhouse.Settings{
			"max_execution_time": 60,
		},
		Compression: &clickhouse.Compression{
			Method: clickhouse.CompressionLZ4,
		},
	})
	if err != nil {
		return nil, fmt.Errorf("connecting to clickhouse: %w", err)
	}

	return &Client{conn: conn, logger: logger}, nil
}

func (c *Client) Close() error {
	return c.conn.Close()
}

func (c *Client) Ping(ctx context.Context) error {
	return c.conn.Ping(ctx)
}

func (c *Client) InsertFlows(ctx context.Context, flows []FlowEvent) error {
	batch, err := c.conn.PrepareBatch(ctx, `
		INSERT INTO flow_events (
			timestamp, connection_id, src_ip, dst_ip, src_port, dst_port,
			protocol, bytes, packets, latency_ms
		)
	`)
	if err != nil {
		return fmt.Errorf("preparing batch: %w", err)
	}

	for _, f := range flows {
		if err := batch.Append(
			f.Timestamp,
			f.ConnectionID,
			f.SrcIP,
			f.DstIP,
			f.SrcPort,
			f.DstPort,
			f.Protocol,
			f.Bytes,
			f.Packets,
			f.LatencyMs,
		); err != nil {
			return fmt.Errorf("appending flow: %w", err)
		}
	}

	return batch.Send()
}

func (c *Client) QueryFlows(ctx context.Context, connectionID string, start, end time.Time, limit int) ([]FlowEvent, error) {
	query := `
		SELECT timestamp, connection_id, src_ip, dst_ip, src_port, dst_port,
		       protocol, bytes, packets, latency_ms
		FROM flow_events
		WHERE connection_id = $1 AND timestamp >= $2 AND timestamp <= $3
		ORDER BY timestamp DESC
		LIMIT $4
	`

	rows, err := c.conn.Query(ctx, query, connectionID, start, end, limit)
	if err != nil {
		return nil, fmt.Errorf("querying flows: %w", err)
	}
	defer rows.Close()

	var flows []FlowEvent
	for rows.Next() {
		var f FlowEvent
		if err := rows.Scan(
			&f.Timestamp,
			&f.ConnectionID,
			&f.SrcIP,
			&f.DstIP,
			&f.SrcPort,
			&f.DstPort,
			&f.Protocol,
			&f.Bytes,
			&f.Packets,
			&f.LatencyMs,
		); err != nil {
			return nil, fmt.Errorf("scanning flow: %w", err)
		}
		flows = append(flows, f)
	}

	return flows, nil
}

func (c *Client) GetEdgeBandwidth(ctx context.Context, connectionID string, duration time.Duration) (map[string]uint64, error) {
	query := `
		SELECT src_ip, dst_ip, dst_port, protocol, sum(bytes) as total_bytes
		FROM flow_events
		WHERE connection_id = $1 AND timestamp >= $2
		GROUP BY src_ip, dst_ip, dst_port, protocol
	`

	start := time.Now().Add(-duration)
	rows, err := c.conn.Query(ctx, query, connectionID, start)
	if err != nil {
		return nil, fmt.Errorf("querying bandwidth: %w", err)
	}
	defer rows.Close()

	result := make(map[string]uint64)
	for rows.Next() {
		var srcIP, dstIP, protocol string
		var dstPort uint16
		var totalBytes uint64
		if err := rows.Scan(&srcIP, &dstIP, &dstPort, &protocol, &totalBytes); err != nil {
			return nil, fmt.Errorf("scanning bandwidth: %w", err)
		}
		key := fmt.Sprintf("%s:%s:%d:%s", srcIP, dstIP, dstPort, protocol)
		result[key] = totalBytes
	}

	return result, nil
}

func (c *Client) InitSchema(ctx context.Context) error {
	query := `
		CREATE TABLE IF NOT EXISTS flow_events (
			timestamp DateTime64(3),
			connection_id String,
			src_ip String,
			dst_ip String,
			src_port UInt16,
			dst_port UInt16,
			protocol String,
			bytes UInt64,
			packets UInt64,
			latency_ms Float64
		) ENGINE = MergeTree()
		PARTITION BY toYYYYMM(timestamp)
		ORDER BY (connection_id, timestamp)
		TTL timestamp + INTERVAL 90 DAY
	`

	return c.conn.Exec(ctx, query)
}
