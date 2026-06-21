-- +migrate Up
-- Performance optimization indexes

-- Users table
CREATE INDEX IF NOT EXISTS idx_users_email_lower ON users (LOWER(email));
CREATE INDEX IF NOT EXISTS idx_users_created_at ON users (created_at);

-- Orgs table
CREATE INDEX IF NOT EXISTS idx_orgs_slug_lower ON orgs (LOWER(slug));
CREATE INDEX IF NOT EXISTS idx_orgs_created_at ON orgs (created_at);

-- Memberships table
CREATE INDEX IF NOT EXISTS idx_memberships_user_org ON memberships (user_id, org_id);
CREATE INDEX IF NOT EXISTS idx_memberships_role ON memberships (role);

-- Connections table
CREATE INDEX IF NOT EXISTS idx_connections_org_status ON connections (org_id, status);
CREATE INDEX IF NOT EXISTS idx_connections_agent_token ON connections (agent_token) WHERE agent_token IS NOT NULL;
CREATE INDEX IF NOT EXISTS idx_connections_last_seen ON connections (last_seen_at);
CREATE INDEX IF NOT EXISTS idx_connections_type ON connections (type);

-- Hosts table
CREATE INDEX IF NOT EXISTS idx_hosts_connection_hostname ON hosts (connection_id, hostname);

-- Containers table
CREATE INDEX IF NOT EXISTS idx_containers_connection_state ON containers (connection_id, state);
CREATE INDEX IF NOT EXISTS idx_containers_runtime_id ON containers (runtime_id);
CREATE INDEX IF NOT EXISTS idx_containers_image ON containers (image);
CREATE INDEX IF NOT EXISTS idx_containers_removed ON containers (removed_at) WHERE removed_at IS NOT NULL;
CREATE INDEX IF NOT EXISTS idx_containers_name_search ON containers USING gin (name gin_trgm_ops);

-- Networks table
CREATE INDEX IF NOT EXISTS idx_networks_connection_driver ON networks (connection_id, driver);

-- Edges table
CREATE INDEX IF NOT EXISTS idx_edges_last_seen ON edges (last_seen_at);
CREATE INDEX IF NOT EXISTS idx_edges_protocol ON edges (protocol);
CREATE INDEX IF NOT EXISTS idx_edges_dst_ip_port ON edges (dst_ip, dst_port);

-- Metric samples table
CREATE INDEX IF NOT EXISTS idx_metrics_connection_metric_time ON metric_samples (connection_id, metric, timestamp DESC);
CREATE INDEX IF NOT EXISTS idx_metrics_runtime_metric_time ON metric_samples (runtime_id, metric, timestamp DESC);

-- Vulnerability scans table
CREATE INDEX IF NOT EXISTS idx_vuln_scans_image_time ON vulnerability_scans (image, scan_time DESC);

-- Vulnerabilities table
CREATE INDEX IF NOT EXISTS idx_vuln_scan_severity ON vulnerabilities (scan_id, severity);

-- Audit logs table
CREATE INDEX IF NOT EXISTS idx_audit_actor ON audit_logs (actor_id);
CREATE INDEX IF NOT EXISTS idx_audit_action ON audit_logs (action);
CREATE INDEX IF NOT EXISTS idx_audit_org_action_time ON audit_logs (org_id, action, created_at DESC);

-- Enable pg_trgm extension for text search
CREATE EXTENSION IF NOT EXISTS pg_trgm;

-- +migrate Down
DROP INDEX IF EXISTS idx_users_email_lower;
DROP INDEX IF EXISTS idx_users_created_at;
DROP INDEX IF EXISTS idx_orgs_slug_lower;
DROP INDEX IF EXISTS idx_orgs_created_at;
DROP INDEX IF EXISTS idx_memberships_user_org;
DROP INDEX IF EXISTS idx_memberships_role;
DROP INDEX IF EXISTS idx_connections_org_status;
DROP INDEX IF EXISTS idx_connections_agent_token;
DROP INDEX IF EXISTS idx_connections_last_seen;
DROP INDEX IF EXISTS idx_connections_type;
DROP INDEX IF EXISTS idx_hosts_connection_hostname;
DROP INDEX IF EXISTS idx_containers_connection_state;
DROP INDEX IF EXISTS idx_containers_runtime_id;
DROP INDEX IF EXISTS idx_containers_image;
DROP INDEX IF EXISTS idx_containers_removed;
DROP INDEX IF EXISTS idx_containers_name_search;
DROP INDEX IF EXISTS idx_networks_connection_driver;
DROP INDEX IF EXISTS idx_edges_last_seen;
DROP INDEX IF EXISTS idx_edges_protocol;
DROP INDEX IF EXISTS idx_edges_dst_ip_port;
DROP INDEX IF EXISTS idx_metrics_connection_metric_time;
DROP INDEX IF EXISTS idx_metrics_runtime_metric_time;
DROP INDEX IF EXISTS idx_vuln_scans_image_time;
DROP INDEX IF EXISTS idx_vuln_scan_severity;
DROP INDEX IF EXISTS idx_audit_actor;
DROP INDEX IF EXISTS idx_audit_action;
DROP INDEX IF EXISTS idx_audit_org_action_time;
