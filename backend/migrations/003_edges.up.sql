-- +migrate Up
CREATE TABLE edges (
    id BIGSERIAL,
    connection_id UUID NOT NULL REFERENCES connections(id) ON DELETE CASCADE,
    src_container_id TEXT NOT NULL,
    dst_container_id TEXT,
    dst_ip TEXT NOT NULL,
    dst_port INTEGER NOT NULL,
    protocol TEXT NOT NULL,
    first_seen TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    last_seen TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_edges_connection ON edges(connection_id);
CREATE INDEX idx_edges_src ON edges(src_container_id);
CREATE INDEX idx_edges_dst ON edges(dst_container_id);
CREATE INDEX idx_edges_composite ON edges(connection_id, src_container_id, dst_container_id);

-- +migrate Down
DROP TABLE IF EXISTS edges;
