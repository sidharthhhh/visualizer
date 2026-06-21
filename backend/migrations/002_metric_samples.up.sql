-- +migrate Up
CREATE TABLE metric_samples (
    id BIGSERIAL,
    connection_id UUID NOT NULL REFERENCES connections(id) ON DELETE CASCADE,
    runtime_id TEXT NOT NULL,
    metric TEXT NOT NULL,
    value DOUBLE PRECISION NOT NULL,
    timestamp TIMESTAMPTZ NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_metric_samples_connection ON metric_samples(connection_id);
CREATE INDEX idx_metric_samples_runtime ON metric_samples(runtime_id);
CREATE INDEX idx_metric_samples_metric ON metric_samples(metric);
CREATE INDEX idx_metric_samples_timestamp ON metric_samples(timestamp);
CREATE INDEX idx_metric_samples_composite ON metric_samples(connection_id, runtime_id, metric, timestamp);

-- +migrate Down
DROP TABLE IF EXISTS metric_samples;
