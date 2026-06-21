-- +migrate Up
CREATE TABLE vulnerability_scans (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    connection_id UUID NOT NULL REFERENCES connections(id) ON DELETE CASCADE,
    image TEXT NOT NULL,
    image_id TEXT,
    scan_time TIMESTAMPTZ NOT NULL,
    critical_count INTEGER NOT NULL DEFAULT 0,
    high_count INTEGER NOT NULL DEFAULT 0,
    medium_count INTEGER NOT NULL DEFAULT 0,
    low_count INTEGER NOT NULL DEFAULT 0,
    total_count INTEGER NOT NULL DEFAULT 0,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE vulnerabilities (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    scan_id UUID NOT NULL REFERENCES vulnerability_scans(id) ON DELETE CASCADE,
    vuln_id TEXT NOT NULL,
    severity TEXT NOT NULL,
    package TEXT NOT NULL,
    version TEXT NOT NULL,
    fixed_in TEXT,
    title TEXT,
    description TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_vuln_scans_connection ON vulnerability_scans(connection_id);
CREATE INDEX idx_vuln_scans_image ON vulnerability_scans(image);
CREATE INDEX idx_vuln_scans_time ON vulnerability_scans(scan_time);
CREATE INDEX idx_vulnerabilities_scan ON vulnerabilities(scan_id);
CREATE INDEX idx_vulnerabilities_severity ON vulnerabilities(severity);
CREATE INDEX idx_vulnerabilities_vuln_id ON vulnerabilities(vuln_id);

-- +migrate Down
DROP TABLE IF EXISTS vulnerabilities;
DROP TABLE IF EXISTS vulnerability_scans;
