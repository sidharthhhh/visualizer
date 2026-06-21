package store

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

type Store struct {
	pool *pgxpool.Pool
}

func New(pool *pgxpool.Pool) *Store {
	return &Store{pool: pool}
}

type User struct {
	ID           uuid.UUID `json:"id"`
	Email        string    `json:"email"`
	Name         string    `json:"name"`
	PasswordHash string    `json:"-"`
	CreatedAt    time.Time `json:"created_at"`
}

type Org struct {
	ID        uuid.UUID `json:"id"`
	Name      string    `json:"name"`
	Slug      string    `json:"slug"`
	Plan      string    `json:"plan"`
	CreatedAt time.Time `json:"created_at"`
}

type Membership struct {
	UserID    uuid.UUID `json:"user_id"`
	OrgID     uuid.UUID `json:"org_id"`
	Role      string    `json:"role"`
	CreatedAt time.Time `json:"created_at"`
}

type MemberWithUser struct {
	Membership
	User User `json:"user"`
}

func (s *Store) CreateUser(ctx context.Context, email, name, passwordHash string) (*User, error) {
	var u User
	err := s.pool.QueryRow(ctx,
		`INSERT INTO users (email, name, password_hash) VALUES ($1, $2, $3)
		 RETURNING id, email, name, password_hash, created_at`,
		email, name, passwordHash,
	).Scan(&u.ID, &u.Email, &u.Name, &u.PasswordHash, &u.CreatedAt)
	if err != nil {
		return nil, fmt.Errorf("creating user: %w", err)
	}
	return &u, nil
}

func (s *Store) GetUserByEmail(ctx context.Context, email string) (*User, error) {
	var u User
	err := s.pool.QueryRow(ctx,
		`SELECT id, email, name, password_hash, created_at FROM users WHERE email = $1`, email,
	).Scan(&u.ID, &u.Email, &u.Name, &u.PasswordHash, &u.CreatedAt)
	if err != nil {
		return nil, fmt.Errorf("getting user by email: %w", err)
	}
	return &u, nil
}

func (s *Store) GetUserByID(ctx context.Context, id uuid.UUID) (*User, error) {
	var u User
	err := s.pool.QueryRow(ctx,
		`SELECT id, email, name, password_hash, created_at FROM users WHERE id = $1`, id,
	).Scan(&u.ID, &u.Email, &u.Name, &u.PasswordHash, &u.CreatedAt)
	if err != nil {
		return nil, fmt.Errorf("getting user by id: %w", err)
	}
	return &u, nil
}

func (s *Store) CreateOrg(ctx context.Context, name, slug string) (*Org, error) {
	var o Org
	err := s.pool.QueryRow(ctx,
		`INSERT INTO orgs (name, slug) VALUES ($1, $2)
		 RETURNING id, name, slug, plan, created_at`,
		name, slug,
	).Scan(&o.ID, &o.Name, &o.Slug, &o.Plan, &o.CreatedAt)
	if err != nil {
		return nil, fmt.Errorf("creating org: %w", err)
	}
	return &o, nil
}

func (s *Store) GetOrgByID(ctx context.Context, id uuid.UUID) (*Org, error) {
	var o Org
	err := s.pool.QueryRow(ctx,
		`SELECT id, name, slug, plan, created_at FROM orgs WHERE id = $1`, id,
	).Scan(&o.ID, &o.Name, &o.Slug, &o.Plan, &o.CreatedAt)
	if err != nil {
		return nil, fmt.Errorf("getting org: %w", err)
	}
	return &o, nil
}

func (s *Store) ListOrgsByUser(ctx context.Context, userID uuid.UUID) ([]Org, error) {
	rows, err := s.pool.Query(ctx,
		`SELECT o.id, o.name, o.slug, o.plan, o.created_at
		 FROM orgs o JOIN memberships m ON o.id = m.org_id
		 WHERE m.user_id = $1 ORDER BY o.name`, userID,
	)
	if err != nil {
		return nil, fmt.Errorf("listing orgs: %w", err)
	}
	defer rows.Close()

	orgs := make([]Org, 0)
	for rows.Next() {
		var o Org
		if err := rows.Scan(&o.ID, &o.Name, &o.Slug, &o.Plan, &o.CreatedAt); err != nil {
			return nil, fmt.Errorf("scanning org: %w", err)
		}
		orgs = append(orgs, o)
	}
	return orgs, nil
}

func (s *Store) CreateMembership(ctx context.Context, userID, orgID uuid.UUID, role string) (*Membership, error) {
	var m Membership
	err := s.pool.QueryRow(ctx,
		`INSERT INTO memberships (user_id, org_id, role) VALUES ($1, $2, $3)
		 RETURNING user_id, org_id, role, created_at`,
		userID, orgID, role,
	).Scan(&m.UserID, &m.OrgID, &m.Role, &m.CreatedAt)
	if err != nil {
		return nil, fmt.Errorf("creating membership: %w", err)
	}
	return &m, nil
}

func (s *Store) GetMembership(ctx context.Context, userID, orgID uuid.UUID) (*Membership, error) {
	var m Membership
	err := s.pool.QueryRow(ctx,
		`SELECT user_id, org_id, role, created_at FROM memberships WHERE user_id = $1 AND org_id = $2`,
		userID, orgID,
	).Scan(&m.UserID, &m.OrgID, &m.Role, &m.CreatedAt)
	if err != nil {
		return nil, fmt.Errorf("getting membership: %w", err)
	}
	return &m, nil
}

func (s *Store) ListMembers(ctx context.Context, orgID uuid.UUID) ([]MemberWithUser, error) {
	rows, err := s.pool.Query(ctx,
		`SELECT m.user_id, m.org_id, m.role, m.created_at,
		        u.id, u.email, u.name, u.created_at
		 FROM memberships m JOIN users u ON m.user_id = u.id
		 WHERE m.org_id = $1 ORDER BY u.name`, orgID,
	)
	if err != nil {
		return nil, fmt.Errorf("listing members: %w", err)
	}
	defer rows.Close()

	members := make([]MemberWithUser, 0)
	for rows.Next() {
		var m MemberWithUser
		if err := rows.Scan(
			&m.UserID, &m.OrgID, &m.Role, &m.CreatedAt,
			&m.User.ID, &m.User.Email, &m.User.Name, &m.User.CreatedAt,
		); err != nil {
			return nil, fmt.Errorf("scanning member: %w", err)
		}
		members = append(members, m)
	}
	return members, nil
}

func (s *Store) UpdateMemberRole(ctx context.Context, userID, orgID uuid.UUID, role string) error {
	_, err := s.pool.Exec(ctx,
		`UPDATE memberships SET role = $3 WHERE user_id = $1 AND org_id = $2`,
		userID, orgID, role,
	)
	if err != nil {
		return fmt.Errorf("updating role: %w", err)
	}
	return nil
}

func (s *Store) RemoveMember(ctx context.Context, userID, orgID uuid.UUID) error {
	_, err := s.pool.Exec(ctx,
		`DELETE FROM memberships WHERE user_id = $1 AND org_id = $2`,
		userID, orgID,
	)
	if err != nil {
		return fmt.Errorf("removing member: %w", err)
	}
	return nil
}

func (s *Store) CreateAuditLog(ctx context.Context, orgID, actorID uuid.UUID, action, target string, metadata map[string]interface{}) error {
	_, err := s.pool.Exec(ctx,
		`INSERT INTO audit_logs (org_id, actor_id, action, target, metadata) VALUES ($1, $2, $3, $4, $5)`,
		orgID, actorID, action, target, metadata,
	)
	if err != nil {
		return fmt.Errorf("creating audit log: %w", err)
	}
	return nil
}

type AuditLog struct {
	ID        uuid.UUID              `json:"id"`
	OrgID     uuid.UUID              `json:"org_id"`
	ActorID   *uuid.UUID             `json:"actor_id,omitempty"`
	Action    string                 `json:"action"`
	Target    *string                `json:"target,omitempty"`
	Metadata  map[string]interface{} `json:"metadata,omitempty"`
	CreatedAt time.Time              `json:"created_at"`
}

func (s *Store) ListAuditLogs(ctx context.Context, orgID uuid.UUID) ([]AuditLog, error) {
	rows, err := s.pool.Query(ctx,
		`SELECT id, org_id, actor_id, action, target, metadata, created_at
		 FROM audit_logs WHERE org_id = $1 ORDER BY created_at DESC LIMIT 100`, orgID,
	)
	if err != nil {
		return nil, fmt.Errorf("listing audit logs: %w", err)
	}
	defer rows.Close()

	logs := make([]AuditLog, 0)
	for rows.Next() {
		var l AuditLog
		if err := rows.Scan(&l.ID, &l.OrgID, &l.ActorID, &l.Action, &l.Target, &l.Metadata, &l.CreatedAt); err != nil {
			return nil, fmt.Errorf("scanning audit log: %w", err)
		}
		logs = append(logs, l)
	}
	return logs, nil
}

type Connection struct {
	ID         uuid.UUID              `json:"id"`
	OrgID      uuid.UUID              `json:"org_id"`
	Type       string                 `json:"type"`
	Name       string                 `json:"name"`
	Status     string                 `json:"status"`
	AgentToken string                 `json:"agent_token,omitempty"`
	LastSeenAt *time.Time             `json:"last_seen_at,omitempty"`
	Metadata   map[string]interface{} `json:"metadata,omitempty"`
	CreatedAt  time.Time              `json:"created_at"`
}

type Host struct {
	ID           uuid.UUID `json:"id"`
	ConnectionID uuid.UUID `json:"connection_id"`
	Hostname     string    `json:"hostname"`
	OS           string    `json:"os"`
	Kernel       string    `json:"kernel"`
	CPUCores     int32     `json:"cpu_cores"`
	MemTotal     int64     `json:"mem_total"`
	AgentVersion string    `json:"agent_version"`
	CreatedAt    time.Time `json:"created_at"`
}

func (s *Store) CreateConnection(ctx context.Context, orgID uuid.UUID, connType, name string) (*Connection, error) {
	var c Connection
	token := uuid.New().String()
	err := s.pool.QueryRow(ctx,
		`INSERT INTO connections (org_id, type, name, agent_token) VALUES ($1, $2, $3, $4)
		 RETURNING id, org_id, type, name, status, agent_token, created_at`,
		orgID, connType, name, token,
	).Scan(&c.ID, &c.OrgID, &c.Type, &c.Name, &c.Status, &c.AgentToken, &c.CreatedAt)
	if err != nil {
		return nil, fmt.Errorf("creating connection: %w", err)
	}
	return &c, nil
}

func (s *Store) GetConnectionByID(ctx context.Context, id uuid.UUID) (*Connection, error) {
	var c Connection
	err := s.pool.QueryRow(ctx,
		`SELECT id, org_id, type, name, status, last_seen_at, created_at FROM connections WHERE id = $1`, id,
	).Scan(&c.ID, &c.OrgID, &c.Type, &c.Name, &c.Status, &c.LastSeenAt, &c.CreatedAt)
	if err != nil {
		return nil, fmt.Errorf("getting connection: %w", err)
	}
	return &c, nil
}

func (s *Store) GetConnectionByToken(ctx context.Context, token string) (*Connection, error) {
	var c Connection
	err := s.pool.QueryRow(ctx,
		`SELECT id, org_id, type, name, status, last_seen_at, created_at FROM connections WHERE agent_token = $1`, token,
	).Scan(&c.ID, &c.OrgID, &c.Type, &c.Name, &c.Status, &c.LastSeenAt, &c.CreatedAt)
	if err != nil {
		return nil, fmt.Errorf("getting connection by token: %w", err)
	}
	return &c, nil
}

func (s *Store) ListConnections(ctx context.Context, orgID uuid.UUID) ([]Connection, error) {
	rows, err := s.pool.Query(ctx,
		`SELECT id, org_id, type, name, status, last_seen_at, created_at
		 FROM connections WHERE org_id = $1 ORDER BY name`, orgID,
	)
	if err != nil {
		return []Connection{}, fmt.Errorf("listing connections: %w", err)
	}
	defer rows.Close()

	conns := make([]Connection, 0)
	for rows.Next() {
		var c Connection
		if err := rows.Scan(&c.ID, &c.OrgID, &c.Type, &c.Name, &c.Status, &c.LastSeenAt, &c.CreatedAt); err != nil {
			return []Connection{}, fmt.Errorf("scanning connection: %w", err)
		}
		conns = append(conns, c)
	}
	return conns, nil
}

func (s *Store) UpdateConnectionStatus(ctx context.Context, id uuid.UUID, status string) error {
	_, err := s.pool.Exec(ctx,
		`UPDATE connections SET status = $1, updated_at = NOW() WHERE id = $2`,
		status, id,
	)
	if err != nil {
		return fmt.Errorf("updating connection status: %w", err)
	}
	return nil
}

func (s *Store) UpdateConnectionLastSeen(ctx context.Context, id uuid.UUID) error {
	_, err := s.pool.Exec(ctx,
		`UPDATE connections SET last_seen_at = NOW() WHERE id = $1`,
		id,
	)
	if err != nil {
		return fmt.Errorf("updating last seen: %w", err)
	}
	return nil
}

func (s *Store) CreateHost(ctx context.Context, connID uuid.UUID, hostname, os, kernel string, cpuCores int32, memTotal int64, agentVersion string) (*Host, error) {
	var h Host
	err := s.pool.QueryRow(ctx,
		`INSERT INTO hosts (connection_id, hostname, os, kernel, cpu_cores, mem_total, agent_version)
		 VALUES ($1, $2, $3, $4, $5, $6, $7)
		 RETURNING id, connection_id, hostname, os, kernel, cpu_cores, mem_total, agent_version, created_at`,
		connID, hostname, os, kernel, cpuCores, memTotal, agentVersion,
	).Scan(&h.ID, &h.ConnectionID, &h.Hostname, &h.OS, &h.Kernel, &h.CPUCores, &h.MemTotal, &h.AgentVersion, &h.CreatedAt)
	if err != nil {
		return nil, fmt.Errorf("creating host: %w", err)
	}
	return &h, nil
}

func (s *Store) ListHosts(ctx context.Context, connID uuid.UUID) ([]Host, error) {
	rows, err := s.pool.Query(ctx,
		`SELECT id, connection_id, hostname, os, kernel, cpu_cores, mem_total, agent_version, created_at
		 FROM hosts WHERE connection_id = $1 ORDER BY hostname`, connID,
	)
	if err != nil {
		return nil, fmt.Errorf("listing hosts: %w", err)
	}
	defer rows.Close()

	hosts := make([]Host, 0)
	for rows.Next() {
		var h Host
		if err := rows.Scan(&h.ID, &h.ConnectionID, &h.Hostname, &h.OS, &h.Kernel, &h.CPUCores, &h.MemTotal, &h.AgentVersion, &h.CreatedAt); err != nil {
			return nil, fmt.Errorf("scanning host: %w", err)
		}
		hosts = append(hosts, h)
	}
	return hosts, nil
}

type Container struct {
	ID           uuid.UUID              `json:"id"`
	ConnectionID uuid.UUID              `json:"connection_id"`
	RuntimeID    string                 `json:"runtime_id"`
	Name         string                 `json:"name"`
	Image        string                 `json:"image"`
	State        string                 `json:"state"`
	Labels       map[string]interface{} `json:"labels,omitempty"`
	Ports        []interface{}          `json:"ports,omitempty"`
	CreatedAt    time.Time              `json:"created_at"`
	UpdatedAt    time.Time              `json:"updated_at"`
}

type Network struct {
	ID           uuid.UUID `json:"id"`
	ConnectionID uuid.UUID `json:"connection_id"`
	Name         string    `json:"name"`
	Driver       string    `json:"driver"`
	Scope        string    `json:"scope"`
	Subnet       string    `json:"subnet"`
	CreatedAt    time.Time `json:"created_at"`
}

func (s *Store) ListContainers(ctx context.Context, connID uuid.UUID) ([]Container, error) {
	rows, err := s.pool.Query(ctx,
		`SELECT id, connection_id, runtime_id, name, image, state, labels, ports, created_at, updated_at
		 FROM containers WHERE connection_id = $1 AND removed_at IS NULL ORDER BY name`, connID,
	)
	if err != nil {
		return nil, fmt.Errorf("listing containers: %w", err)
	}
	defer rows.Close()

	containers := make([]Container, 0)
	for rows.Next() {
		var c Container
		if err := rows.Scan(&c.ID, &c.ConnectionID, &c.RuntimeID, &c.Name, &c.Image, &c.State, &c.Labels, &c.Ports, &c.CreatedAt, &c.UpdatedAt); err != nil {
			return nil, fmt.Errorf("scanning container: %w", err)
		}
		containers = append(containers, c)
	}
	return containers, nil
}

func (s *Store) GetContainerByRuntimeID(ctx context.Context, connID uuid.UUID, runtimeID string) (*Container, error) {
	var c Container
	err := s.pool.QueryRow(ctx,
		`SELECT id, connection_id, runtime_id, name, image, state, labels, ports, created_at, updated_at
		 FROM containers WHERE connection_id = $1 AND runtime_id = $2`,
		connID, runtimeID,
	).Scan(&c.ID, &c.ConnectionID, &c.RuntimeID, &c.Name, &c.Image, &c.State, &c.Labels, &c.Ports, &c.CreatedAt, &c.UpdatedAt)
	if err != nil {
		return nil, fmt.Errorf("getting container: %w", err)
	}
	return &c, nil
}

func (s *Store) ListNetworks(ctx context.Context, connID uuid.UUID) ([]Network, error) {
	rows, err := s.pool.Query(ctx,
		`SELECT id, connection_id, name, COALESCE(driver, ''), COALESCE(scope, 'local'), COALESCE(subnet, ''), created_at
		 FROM networks WHERE connection_id = $1 ORDER BY name`, connID,
	)
	if err != nil {
		return nil, fmt.Errorf("listing networks: %w", err)
	}
	defer rows.Close()

	networks := make([]Network, 0)
	for rows.Next() {
		var n Network
		if err := rows.Scan(&n.ID, &n.ConnectionID, &n.Name, &n.Driver, &n.Scope, &n.Subnet, &n.CreatedAt); err != nil {
			return nil, fmt.Errorf("scanning network: %w", err)
		}
		networks = append(networks, n)
	}
	return networks, nil
}

type Edge struct {
	ID             uuid.UUID `json:"id"`
	ConnectionID   uuid.UUID `json:"connection_id"`
	SrcContainerID string    `json:"src_container_id"`
	DstContainerID *string   `json:"dst_container_id,omitempty"`
	DstIP          string    `json:"dst_ip"`
	DstPort        int       `json:"dst_port"`
	Protocol       string    `json:"protocol"`
	FirstSeen      time.Time `json:"first_seen"`
	LastSeen       time.Time `json:"last_seen"`
}

func (s *Store) ListEdges(ctx context.Context, connID uuid.UUID) ([]Edge, error) {
	rows, err := s.pool.Query(ctx,
		`SELECT id, connection_id, src_container_id, dst_container_id, dst_ip, dst_port, protocol, first_seen, last_seen
		 FROM edges WHERE connection_id = $1 ORDER BY last_seen DESC`, connID,
	)
	if err != nil {
		return nil, fmt.Errorf("listing edges: %w", err)
	}
	defer rows.Close()

	edges := make([]Edge, 0)
	for rows.Next() {
		var e Edge
		if err := rows.Scan(&e.ID, &e.ConnectionID, &e.SrcContainerID, &e.DstContainerID, &e.DstIP, &e.DstPort, &e.Protocol, &e.FirstSeen, &e.LastSeen); err != nil {
			return nil, fmt.Errorf("scanning edge: %w", err)
		}
		edges = append(edges, e)
	}
	return edges, nil
}

func (s *Store) ReplaceEdges(ctx context.Context, connID uuid.UUID, edges []Edge) error {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("beginning transaction: %w", err)
	}
	defer tx.Rollback(ctx)

	_, err = tx.Exec(ctx, `DELETE FROM edges WHERE connection_id = $1`, connID)
	if err != nil {
		return fmt.Errorf("deleting old edges: %w", err)
	}

	for _, e := range edges {
		_, err = tx.Exec(ctx,
			`INSERT INTO edges (connection_id, src_container_id, dst_container_id, dst_ip, dst_port, protocol)
			 VALUES ($1, $2, $3, $4, $5, $6)`,
			connID, e.SrcContainerID, e.DstContainerID, e.DstIP, e.DstPort, e.Protocol,
		)
		if err != nil {
			return fmt.Errorf("inserting edge: %w", err)
		}
	}

	return tx.Commit(ctx)
}

type VulnerabilityScan struct {
	ID            uuid.UUID `json:"id"`
	ConnectionID  uuid.UUID `json:"connection_id"`
	Image         string    `json:"image"`
	ImageID       string    `json:"image_id"`
	ScanTime      time.Time `json:"scan_time"`
	CriticalCount int       `json:"critical_count"`
	HighCount     int       `json:"high_count"`
	MediumCount   int       `json:"medium_count"`
	LowCount      int       `json:"low_count"`
	TotalCount    int       `json:"total_count"`
	CreatedAt     time.Time `json:"created_at"`
}

type Vulnerability struct {
	ID          uuid.UUID `json:"id"`
	ScanID      uuid.UUID `json:"scan_id"`
	VulnID      string    `json:"vuln_id"`
	Severity    string    `json:"severity"`
	Package     string    `json:"package"`
	Version     string    `json:"version"`
	FixedIn     string    `json:"fixed_in"`
	Title       string    `json:"title"`
	Description string    `json:"description"`
	CreatedAt   time.Time `json:"created_at"`
}

func (s *Store) CreateVulnScan(ctx context.Context, scan *VulnerabilityScan) error {
	err := s.pool.QueryRow(ctx,
		`INSERT INTO vulnerability_scans (connection_id, image, image_id, scan_time, critical_count, high_count, medium_count, low_count, total_count)
		 VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
		 RETURNING id, created_at`,
		scan.ConnectionID, scan.Image, scan.ImageID, scan.ScanTime,
		scan.CriticalCount, scan.HighCount, scan.MediumCount, scan.LowCount, scan.TotalCount,
	).Scan(&scan.ID, &scan.CreatedAt)
	if err != nil {
		return fmt.Errorf("creating vuln scan: %w", err)
	}
	return nil
}

func (s *Store) CreateVulnerability(ctx context.Context, vuln *Vulnerability) error {
	err := s.pool.QueryRow(ctx,
		`INSERT INTO vulnerabilities (scan_id, vuln_id, severity, package, version, fixed_in, title, description)
		 VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
		 RETURNING id, created_at`,
		vuln.ScanID, vuln.VulnID, vuln.Severity, vuln.Package, vuln.Version,
		vuln.FixedIn, vuln.Title, vuln.Description,
	).Scan(&vuln.ID, &vuln.CreatedAt)
	if err != nil {
		return fmt.Errorf("creating vulnerability: %w", err)
	}
	return nil
}

func (s *Store) ListVulnScans(ctx context.Context, connID uuid.UUID) ([]VulnerabilityScan, error) {
	rows, err := s.pool.Query(ctx,
		`SELECT id, connection_id, image, image_id, scan_time, critical_count, high_count, medium_count, low_count, total_count, created_at
		 FROM vulnerability_scans WHERE connection_id = $1 ORDER BY scan_time DESC`, connID,
	)
	if err != nil {
		return nil, fmt.Errorf("listing vuln scans: %w", err)
	}
	defer rows.Close()

	scans := make([]VulnerabilityScan, 0)
	for rows.Next() {
		var scan VulnerabilityScan
		if err := rows.Scan(&scan.ID, &scan.ConnectionID, &scan.Image, &scan.ImageID, &scan.ScanTime,
			&scan.CriticalCount, &scan.HighCount, &scan.MediumCount, &scan.LowCount, &scan.TotalCount, &scan.CreatedAt); err != nil {
			return nil, fmt.Errorf("scanning vuln scan: %w", err)
		}
		scans = append(scans, scan)
	}
	return scans, nil
}

func (s *Store) ListVulnerabilities(ctx context.Context, scanID uuid.UUID) ([]Vulnerability, error) {
	rows, err := s.pool.Query(ctx,
		`SELECT id, scan_id, vuln_id, severity, package, version, fixed_in, title, description, created_at
		 FROM vulnerabilities WHERE scan_id = $1 ORDER BY severity, vuln_id`, scanID,
	)
	if err != nil {
		return nil, fmt.Errorf("listing vulnerabilities: %w", err)
	}
	defer rows.Close()

	vulns := make([]Vulnerability, 0)
	for rows.Next() {
		var v Vulnerability
		if err := rows.Scan(&v.ID, &v.ScanID, &v.VulnID, &v.Severity, &v.Package, &v.Version,
			&v.FixedIn, &v.Title, &v.Description, &v.CreatedAt); err != nil {
			return nil, fmt.Errorf("scanning vulnerability: %w", err)
		}
		vulns = append(vulns, v)
	}
	return vulns, nil
}

func (s *Store) GetLatestScanForImage(ctx context.Context, connID uuid.UUID, image string) (*VulnerabilityScan, error) {
	var scan VulnerabilityScan
	err := s.pool.QueryRow(ctx,
		`SELECT id, connection_id, image, image_id, scan_time, critical_count, high_count, medium_count, low_count, total_count, created_at
		 FROM vulnerability_scans WHERE connection_id = $1 AND image = $2 ORDER BY scan_time DESC LIMIT 1`, connID, image,
	).Scan(&scan.ID, &scan.ConnectionID, &scan.Image, &scan.ImageID, &scan.ScanTime,
		&scan.CriticalCount, &scan.HighCount, &scan.MediumCount, &scan.LowCount, &scan.TotalCount, &scan.CreatedAt)
	if err != nil {
		return nil, fmt.Errorf("getting latest scan: %w", err)
	}
	return &scan, nil
}
