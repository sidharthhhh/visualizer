# ContainerScope — Metrics Reference

## Metric Types

| Metric | Description | Unit |
|--------|-------------|------|
| `cpu` | CPU usage percentage | % |
| `mem` | Memory usage | bytes |
| `net_rx` | Network bytes received | bytes |
| `net_tx` | Network bytes transmitted | bytes |
| `disk_r` | Disk bytes read | bytes |
| `disk_w` | Disk bytes written | bytes |

## Labels

All metrics include the following labels:

| Label | Description |
|-------|-------------|
| `connection_id` | UUID of the connection |
| `runtime_id` | Docker container ID |
| `org_id` | UUID of the organization |

## Collection

- **Source:** Docker Engine API (`/containers/{id}/stats`)
- **Interval:** 15 seconds (configurable)
- **Agent:** Collects and streams to backend via gRPC

## Storage

- **Backend:** VictoriaMetrics (Prometheus-compatible)
- **Retention:** 90 days (configurable)
- **Downsampling:** Not implemented in v1

## Query API

### Get Container Metrics (Range)

```
GET /api/v1/orgs/{orgID}/connections/{connectionID}/metrics?runtime_id={id}&metric={type}&start={RFC3339}&end={RFC3339}
```

**Parameters:**
- `runtime_id` (required): Docker container ID
- `metric` (optional): Metric type (default: `cpu`)
- `start` (optional): Start time in RFC3339 format (default: 1 hour ago)
- `end` (optional): End time in RFC3339 format (default: now)

**Response:**
```json
{
  "metric": "cpu",
  "results": [...],
  "start": "2024-01-01T00:00:00Z",
  "end": "2024-01-01T01:00:00Z",
  "step": 15000000000
}
```

### Get Container Metrics (Instant)

```
GET /api/v1/orgs/{orgID}/connections/{connectionID}/metrics/instant?runtime_id={id}
```

**Parameters:**
- `runtime_id` (required): Docker container ID

**Response:**
```json
{
  "cpu": [1704067200, "12.5"],
  "mem": [1704067200, "1073741824"],
  "net_rx": [1704067200, "1048576"],
  "net_tx": [1704067200, "524288"],
  "disk_r": [1704067200, "0"],
  "disk_w": [1704067200, "0"]
}
```

## Prometheus Query Examples

```promql
# CPU usage for a container
cpu{connection_id="xxx",runtime_id="yyy"}

# Memory usage over time
mem{connection_id="xxx",runtime_id="yyy"}[1h]

# Network throughput rate
rate(net_rx{connection_id="xxx"}[5m])
```

## Performance

- Agent overhead target: < 2% host CPU
- Collection interval: 15s (balance between accuracy and overhead)
- Batch size: Up to 100 samples per gRPC message
