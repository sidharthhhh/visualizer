# ContainerScope — Architecture Specification

## Overview
ContainerScope is a multi-tenant DevOps observability and security visualizer for **Docker** and
**Kubernetes**. It renders infrastructure topology, live network packet flow, per-container
resource metrics, and security posture in a high-performance **3D** UI. The system uses an
**Agent → Backend → Frontend** model backed by purpose-fit relational, time-series, and analytical
databases.

The UI is the headline feature (see `ContainerScope_UI_UX_SPEC_3D.md`). This document covers the
system architecture, stack, transport, and canonical data model.

---

## High-level architecture

```text
                         ┌───────────────────────────────────────────┐
                         │                 FRONTEND (Web)             │
                         │  React + TS. 3D topology (Three.js /       │
                         │  3d-force-graph / R3F), live WebSocket     │
                         │  stream, dashboards, 2D fallback           │
                         └───────────────▲───────────────────────────┘
                                         │ REST / GraphQL / WS
                         ┌───────────────┴───────────────────────────┐
                         │                 BACKEND (API)              │
                         │  Go (or Node/TS). Auth, RBAC, orgs,        │
                         │  ingest, query, scan orchestration,        │
                         │  WS fan-out                                │
                         └───┬──────────┬──────────┬─────────┬────────┘
                             │          │          │         │
                   ┌─────────▼──┐  ┌────▼─────┐ ┌──▼──────┐ ┌▼──────────┐
                   │ Postgres   │  │ Metrics  │ │ Flow DB │ │ Object/   │
                   │ (orgs,     │  │ TSDB     │ │ (Click- │ │ blob      │
                   │ users,     │  │ (Victoria│ │ House:  │ │ store     │
                   │ topology,  │  │ Metrics /│ │ flow    │ │ S3/MinIO: │
                   │ findings,  │  │ Prom):   │ │ events) │ │ SBOMs,    │
                   │ RBAC,audit)│  │ cpu/mem  │ │         │ │ reports)  │
                   └────────────┘  └──────────┘ └─────────┘ └───────────┘
                             ▲
                             │ gRPC over mTLS, buffered + compressed
                   ┌─────────┴───────────────────────────────────────────┐
                   │                 AGENT (one per host / node)          │
                   │  Go. Collects: container topology (Docker API /      │
                   │  kubelet + k8s API), metrics (cgroups v2/cAdvisor),  │
                   │  network flows (eBPF, /proc fallback), triggers      │
                   │  image scans (Trivy).                                │
                   └──────────────────────────────────────────────────────┘
```

> **Data placement note:** vulnerability **Findings/Scans live in Postgres** (relational, queried
> with topology and org scope). The **Metrics TSDB holds only time-series** (CPU/mem/net/disk).
> SBOMs and raw scan reports are blobs in the **object store**. Keeping these straight avoids the
> common mistake of stuffing findings into the TSDB.

---

## Component responsibilities

### Agent (one per host / k8s node)
- **Topology:** Docker Engine API (read-only socket) and/or kubelet + Kubernetes API (client-go).
- **Metrics:** cgroups v2 / cAdvisor for per-container CPU, memory, disk I/O, network I/O.
- **Network flows:** eBPF (`cilium/ebpf`) for accurate, low-overhead flow + latency capture;
  **falls back to `/proc/net` sampling** on unsupported kernels (Docker Desktop, older hosts).
- **Scanning:** runs Trivy on observed images; ships results to backend.
- **Transport:** streams to backend over gRPC with mTLS; buffers + compresses; survives reconnects.
- **Footprint target:** < 2% host CPU overhead.

### Backend (API)
- Auth, RBAC, org/tenant management, agent enrollment, audit logging.
- Ingest pipeline: topology deltas, metric samples, flow events, scan results.
- Query APIs: REST + GraphQL; WebSocket fan-out for live topology/flow pushes.
- Scan orchestration (enqueue, dedupe by image digest, persist findings).
- Alert rule evaluation and notification dispatch.

### Frontend (Web)
- 3D topology viewport (Three.js / `3d-force-graph` for MVP → react-three-fiber for scale).
- Live WebSocket-driven updates; metrics dashboards; flow explorer; security views.
- 2D Cytoscape fallback + reduced-motion mode for accessibility/low-power devices.

---

## Technology stack

| Concern | Choice | Rationale |
|---|---|---|
| Agent + Backend | **Go** | Best ecosystem for Docker API, k8s `client-go`, and `cilium/ebpf` |
| Frontend | **React + TypeScript + Vite** | Standard, fast, large ecosystem |
| 3D engine | **Three.js** | The standard for production WebGL |
| 3D graph (MVP) | **`3d-force-graph`** | Force-directed 3D graph on Three.js; built-in directional link particles |
| 3D graph (scale) | **react-three-fiber + Drei** | Full control: InstancedMesh particles, shaders, LOD, bloom |
| 2D fallback | **Cytoscape.js** | Accessibility / low-power / "flatten to 2D" |
| Charts | **D3 / visx** | Sparklines + detailed metric charts |
| Relational DB | **PostgreSQL** | Orgs, users, RBAC, topology state, findings, audit |
| Metrics TSDB | **VictoriaMetrics** (or Prometheus) | High-volume CPU/mem/net/disk time-series |
| Flow store | **ClickHouse** | High-volume flow events, fast aggregation |
| Object store | **S3 / MinIO** | SBOMs, scan reports |
| Vuln scanning | **Trivy** (primary), **Grype** (optional 2nd engine) | Trivy = CVE+SBOM+misconfig in one; Grype = deep SBOM matching |
| Agent ↔ Backend | **gRPC + mTLS** | Efficient, secure, streaming |
| Backend ↔ Frontend | **REST + GraphQL + WebSocket** | Flexible reads + live push |

---

## Communication & security

- **Agent ↔ Backend:** gRPC over **mTLS**; enrollment via per-connection token, then certificate.
  Buffered and compressed; backpressure-aware; reconnect with state resync.
- **Backend ↔ Frontend:** REST (`/api/v1`) + GraphQL for reads; **WebSocket** for live topology and
  flow deltas.
- **Tenancy isolation:** every query is org-scoped; cross-org reads are impossible by construction
  and covered by isolation tests.
- **Least privilege:** read-only Docker socket mounts; documented eBPF capabilities; no secrets in
  logs; tokens encrypted at rest; signed agent releases.

---

## Data model (canonical entities)

**Tenancy & access**
- **Org** — tenant boundary (`id, name, slug, plan`).
- **User** — `id, email, name, credential`.
- **Membership** — `user_id, org_id, role(owner|admin|member|viewer)`.
- **AuditLog** — `id, org_id, actor_id, action, target, ts, metadata`.

**Infrastructure**
- **Connection** — attached source (`id, org_id, type(docker|k8s), name, status, agent_token,
  last_seen_at, metadata`).
- **Host/Node** — `id, connection_id, hostname, os, kernel, cpu_cores, mem_total, agent_version`.
- **Container** — `id, connection_id, host_id, runtime_id, name, image, image_digest, state,
  labels, ports` (+ `pod_id, namespace, owner_kind` for k8s).
- **Pod / Deployment / Service / Namespace** — k8s objects with parent/child links.
- **Network** — `id, connection_id, name, driver, scope, subnet`.

**Telemetry**
- **Edge** — aggregated link (`src_container_id, dst_container_id, dst_ip, dst_port, protocol,
  first_seen, last_seen`).
- **FlowEvent** *(ClickHouse)* — `ts, connection_id, src, dst, proto, bytes, packets, l7, latency_ms`.
- **MetricSample** *(TSDB)* — `entity_id, metric(cpu|mem|net_rx|net_tx|disk_r|disk_w), ts, value`.

**Security**
- **Image** — `digest, repo, tag, os, size, layers`.
- **Scan** *(Postgres)* — `id, image_digest, engine, started_at, finished_at, status, summary`.
- **Finding** *(Postgres)* — `id, scan_id, cve_id, severity, package, installed_version,
  fixed_version, cvss, fixable, description`.
- **Misconfig** *(Postgres)* — `id, connection_id, entity_id, rule_id, severity, title, remediation`.
- **SBOM / raw report** *(object store)* — referenced by `Scan`.

---

## Companion documents
- `ContainerScope_BUILD_SPEC.md` — full chunked build plan (23 chunks, DoD per chunk).
- `ContainerScope_UI_UX_SPEC_3D.md` — the 3D/layered UI & UX specification.
