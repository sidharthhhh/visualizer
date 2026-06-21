# ContainerScope — Build Plan

Step-by-step build plan for ContainerScope, divided into actionable chunks across distinct phases.
Each chunk must be built, compiled, run, and verified against its **Definition of Done (DoD)**
before moving to the next. Full DoD detail per chunk lives in `ContainerScope_BUILD_SPEC.md`;
UI specifics live in `ContainerScope_UI_UX_SPEC_3D.md`; system design in
`ContainerScope_ARCHITECTURE_SPEC.md`.

## Milestones
- **M1 (Chunks 1–7):** Live, animated-ready Docker topology with real-time updates. *First demo.*
- **M2 (Chunks 8–9):** Per-container metrics, heatmaps, top consumers.
- **M3 (Chunks 10–13):** Real network edges and **3D eBPF packet-flow animation**. *The "wow."*
- **M4 (Chunks 14–15):** Kubernetes parity.
- **M5 (Chunks 16–18):** Vulnerability scanning, misconfig, alerting.
- **M6 (Chunks 19–23):** API/SDKs, CI/CD, enterprise readiness, time-travel.

## Phase A: Foundations
- **Chunk 1:** Repo, tooling, CI skeleton. Monorepo, docker-compose (Postgres + VictoriaMetrics + ClickHouse + MinIO), linters, CI workflows.
- **Chunk 2:** Backend skeleton — health checks, config loader, structured logging, DB migrations, HTTP server.
- **Chunk 3:** Auth, Orgs, RBAC. Multi-tenant orgs, JWT (access + refresh), role-based access (owner/admin/member/viewer), audit logs, SSO seam.

## Phase B: Get real data from one Docker host
- **Chunk 4:** Agent skeleton + secure enrollment. gRPC over TLS (mTLS in Phase H), token enroll, heartbeat, connection status state machine.
- **Chunk 5:** Docker topology collection via Docker Engine API — containers, images, networks, live events; snapshot + deltas.

## Phase C: The visualizer
- **Chunk 6:** Frontend shell + topology graph. App shell, org/connection picker, **2D Cytoscape.js** graph first (prove data + interaction), then swap in **`3d-force-graph` (Three.js)** for the 3D view. Node side panel.
- **Chunk 7:** Live updates over WebSocket. Real-time node add/remove/recolor with animated transitions; reconnect + resync.

## Phase D: Metrics
- **Chunk 8:** Per-container resource metrics (CPU, memory, disk I/O, network I/O) from cgroups v2/cAdvisor, stored in **VictoriaMetrics/Prometheus**.
- **Chunk 9:** Metrics in the UI — live sparkline billboards on 3D nodes, detailed D3/visx charts in the side panel, CPU/Mem **heatmap modes**, top-consumers dashboard.

## Phase E: Network & packet flow
- **Chunk 10:** Connection edges from `/proc/net` — draw real "who-talks-to-whom" edges cheaply (no eBPF yet).
- **Chunk 11:** Flow storage in **ClickHouse** + bandwidth on edges (thickness = throughput, color = protocol, direction arrows); flow explorer grid.
- **Chunk 12:** **eBPF flow capture** (`cilium/ebpf`) for accurate, low-overhead flows + latency; feature-detect with **`/proc` fallback**. The hardest, most differentiating piece.
- **Chunk 13:** **3D packet-flow animation.** Move to react-three-fiber + **InstancedMesh** particle pool with GPU-shader motion; LOD (Drei `<Detailed>`) + selective bloom. Target 60fps at 500–1000 nodes.

## Phase F: Kubernetes
- **Chunk 14:** Kubernetes topology — agent as DaemonSet, watches via client-go; namespaces as zones, pods/services/ingress as nested 3D objects.
- **Chunk 15:** Kubernetes flows + metrics — pod-to-pod/service traffic, requests/limits vs usage, **NetworkPolicy visualization + "what-if" simulator**.

## Phase G: Security
- **Chunk 16:** Image vulnerability scanning with **Trivy** (CVEs + SBOM), dedupe by digest; findings overlaid as **severity halos/badges** on nodes; SBOM to object store.
- **Chunk 17:** Misconfig + runtime hardening checks (privileged/root/exposed, CIS Docker/Kubernetes), per-connection **security score** with trend.
- **Chunk 18:** Alerting + notifications — threshold/condition rules, channels (email, Slack, webhook), alert history + ack/resolve.

## Phase H: Platform, SDKs, extensibility
- **Chunk 19:** Public API hardening + **GraphQL** schema; versioned REST, API keys/PATs, rate limiting, OpenAPI docs.
- **Chunk 20:** SDKs for **Go, Python, TypeScript** — auth, topology, metrics, scans, live stream; examples per language.

- **Chunk 21:** CI/CD plugins (GitHub Action, GitLab CI), outbound webhooks, **SARIF** integration; fail builds on critical CVEs.
- **Chunk 22:** Enterprise readiness — **SSO/SAML/OIDC**, **mTLS**, agent auto-update, Helm chart, air-gapped install + offline vuln DB, multi-region/HA, RBAC custom roles.
- **Chunk 23:** **Time-travel / replay** — scrub backward through topology + flows via a time slider driving the 3D scene.
