# ContainerScope — eBPF Flow Capture

## Overview

ContainerScope uses eBPF for accurate, low-overhead network flow capture when available. On unsupported systems, it automatically falls back to `/proc` sampling.

## Requirements

### Kernel Version
- **Minimum:** Linux 5.4+
- **Recommended:** Linux 5.10+ for best performance

### Privileges
eBPF requires one of:
- `CAP_BPF` capability (Linux 5.8+)
- `CAP_SYS_ADMIN` capability (older kernels)

### Docker Agent
```bash
docker run --privileged \
  -v /var/run/docker.sock:/var/run/docker.sock:ro \
  -v /sys/kernel/debug:/sys/kernel/debug:ro \
  containerscope/agent
```

### Kubernetes DaemonSet
```yaml
securityContext:
  privileged: true
  capabilities:
    add:
      - CAP_BPF
      - CAP_SYS_ADMIN
volumeMounts:
  - name: debug
    mountPath: /sys/kernel/debug
volumes:
  - name: debug
    hostPath:
      path: /sys/kernel/debug
```

## Detection

The agent automatically detects eBPF availability:

1. Checks if running on Linux
2. Verifies kernel version >= 5.4
3. Attempts to remove memlock limits
4. Falls back to `/proc` if any check fails

## Fallback Mode

When eBPF is unavailable, the agent uses `/proc/net/tcp` and `/proc/net/tcp6` sampling:

- **Interval:** 15 seconds
- **Accuracy:** Lower (misses short-lived connections)
- **Overhead:** Minimal
- **Latency:** Not available

## eBPF Programs

The agent loads these eBPF programs:

| Program | Type | Purpose |
|---------|------|---------|
| `tcp_connect` | kprobe | Trace TCP connection establishment |
| `tcp_close` | kprobe | Trace TCP connection close |
| `tcp_retransmit` | kprobe | Trace TCP retransmissions |

## Metrics Collected

| Metric | eBPF | /proc |
|--------|------|-------|
| Connection tracking | ✓ | ✓ |
| Bytes transferred | ✓ | ✗ |
| Packet count | ✓ | ✗ |
| Latency (RTT) | ✓ | ✗ |
| Retransmissions | ✓ | ✗ |
| L7 protocol | Stretch | ✗ |

## Performance

- **Target:** < 1% CPU overhead
- **Buffer:** 1000 events
- **Batch:** Events batched before sending to backend

## Troubleshooting

### "eBPF not available"
- Check kernel version: `uname -r`
- Check capabilities: `capsh --print`
- Check if debugfs is mounted: `mount | grep debug`

### "permission denied"
- Run with `--privileged` or add `CAP_BPF`
- Check if `/sys/kernel/debug` is accessible

### "kernel too old"
- Upgrade kernel to 5.4+
- Agent will automatically use `/proc` fallback

## UI Indicator

When using `/proc` fallback, the UI shows a badge:
- **eBPF:** Green badge "eBPF"
- **/proc:** Yellow badge "/proc fallback"
