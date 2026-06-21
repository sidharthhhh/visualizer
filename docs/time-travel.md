# ContainerScope Time Travel

## Overview

Time Travel allows you to replay historical topology states, compare snapshots, and analyze infrastructure changes over time.

## Features

### Topology Snapshots

- Automatic snapshots on topology changes
- Configurable snapshot interval
- Manual snapshot triggers
- Retention policy

### Timeline Replay

- Slider-based timeline navigation
- Play/pause controls
- Speed control (1x, 2x, 5x, 10x)
- Jump to specific timestamps

### Snapshot Comparison

- Side-by-side comparison
- Diff view (added/removed/changed)
- Container lifecycle tracking
- Network flow changes

### Data Export

- Export topology at specific time
- Export time range
- CSV/JSON formats
- API access

## Usage

### Web UI

1. Navigate to Topology view
2. Click "Time Travel" button
3. Use timeline slider to navigate
4. Click "Compare" to select two timestamps

### API

```bash
# Get topology at specific time
curl -H "Authorization: Bearer $TOKEN" \
  "https://api.containerscope.example.com/api/v1/orgs/{orgId}/connections/{connId}/topology?at=2024-01-15T10:30:00Z"

# Get snapshots in time range
curl -H "Authorization: Bearer $TOKEN" \
  "https://api.containerscope.example.com/api/v1/orgs/{orgId}/connections/{connId}/snapshots?start=2024-01-15T00:00:00Z&end=2024-01-16T00:00:00Z"

# Compare two snapshots
curl -H "Authorization: Bearer $TOKEN" \
  "https://api.containerscope.example.com/api/v1/orgs/{orgId}/connections/{connId}/compare?from=2024-01-15T10:00:00Z&to=2024-01-15T11:00:00Z"
```

### SDK

```go
client := containerscope.NewClient("https://api.example.com", containerscope.WithAPIKey("key"))

// Get topology at specific time
topology, err := client.GetTopologyAt(orgID, connID, time.Now().Add(-1*time.Hour))

// Compare snapshots
diff, err := client.CompareTopology(orgID, connID, t1, t2)
```

## Configuration

```yaml
time_travel:
  enabled: true
  snapshot_interval: 5m
  retention: 30d
  max_snapshots: 10000
```

## Storage

- Snapshots stored in PostgreSQL
- Compressed for efficiency
- Incremental storage (only changes)
- Automatic cleanup based on retention
