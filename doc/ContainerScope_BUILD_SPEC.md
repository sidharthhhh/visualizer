# ContainerScope — Build Specification for an LLM Implementer

> A multi-tenant DevOps observability + security visualizer for **Docker** and **Kubernetes**.
> Org → connect hosts/clusters → live visual map of containers, networks, and packet flow,
> with per-container resource metrics and image vulnerability scanning. **The UI is the product.**

This document is written so an LLM coding agent can build the system **chunk by chunk**.
Each chunk is independently buildable, has a clear definition of done (DoD), and lists exact
files to create. Do not skip ahead. Finish a chunk's DoD before starting the next.

---

## 0. How to use this document (read first)

- Work **one chunk at a time**, top to bottom. Each chunk = one or more PRs.
- After each chunk: it must **compile, run, and pass its DoD checklist** before moving on.
- Prefer **boring, well-supported libraries** over clever ones.
- Every chunk says *what to build*, *where*, and *how to verify*.
- When a chunk says "stub", build a fake that returns realistic mock data so the layer above
  can be built and demoed independently. Replace stubs in later chunks.
- Keep secrets out of code. Use `.env` + a secrets manager interface from day one.

---

## 1. Product summary & scope

### What it does
1. **Orgs & access** — multi-tenant orgs, RBAC, invite teammates, connect infra.
2. **Connect infra** — attach one or many Docker hosts and Kubernetes clusters to an org.
3. **Live topology visualizer** — containers/pods as nodes, networks as zones, edges = real
   connections; animated packet/byte flow along edges. This is the headline feature.
4. **Resource metrics** — per-container CPU, memory, disk I/O, network I/O; node rollups.
5. **Network & packet inspection** — who-talks-to-whom, bandwidth, protocol, latency; L7 where possible.
6. **Vulnerability & security** — image CVE scanning, SBOM, misconfig/CIS checks, security score.
7. **SDKs & API** — REST + GraphQL + WebSocket; SDKs (Go, Python, TS) + CI/CD plugins + webhooks.

### Explicit non-goals (v1)
- Not an orchestrator. We **observe**, we don't deploy/schedule workloads (read + light actions only).
- Not a full APM/tracing product. We do infra + network + security, not deep app traces.
- Not a log-management product (we surface logs, we don't store them long-term in v1).

### Market context (why this is worth building)
- Weave Scope was the reference tool for Docker+k8s topology with connection mapping, but it is
  effectively unmaintained after Weaveworks wound down — leaving a real gap.
- eBPF tools exist for pieces (Cilium/Hubble for k8s flows, Caretta for service maps, Netdata for
  L4 metrics, libebpfflow/Trayce for Docker traffic) but **nothing modern unifies Docker + k8s +
  packet-flow + vuln scanning behind one strong multi-tenant UI.** That unification + UI is the wedge.

### The hard, differentiating parts (build a rough version EARLY to de-risk)
- **eBPF-based flow capture** (Chunk 12). Hardest + most differentiating. Most tools stay shallow here.
- **Real-time animated topology at scale** (Chunk 6–7). Many tools render a static blob; smooth,
  filterable, 1000+ node rendering is the moat.

---

## 2. Final architecture (target state)

```
                         ┌───────────────────────────────────────────┐
                         │                 FRONTEND (Web)            │
                         │  React + TS, graph canvas (WebGL), live   │
                         │  WebSocket stream, dashboards             │
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
                   │ Postgres   │  │ TS-DB    │ │ Flow DB │ │ Object/   │
                   │ (orgs,     │  │ (metrics:│ │ (Click- │ │ blob (SBOM│
                   │ topology,  │  │ Victoria │ │ House:  │ │ , reports)│
                   │ users,     │  │ Metrics/ │ │ flows)  │ │           │
                   │ findings)  │  │ Prom)    │ │         │ │           │
                   └────────────┘  └──────────┘ └─────────┘ └───────────┘
                             ▲
                             │ gRPC / HTTPS (mTLS), buffered, compressed
                   ┌─────────┴───────────────────────────────────────────┐
                   │                 AGENT (one per host/node)            │
                   │  Go. Collects: container topology (Docker API /      │
                   │  kubelet+k8s API), metrics (cgroups/cAdvisor),       │
                   │  network flows (eBPF), triggers image scans (Trivy). │
                   └──────────────────────────────────────────────────────┘
```

**Default stack choice (recommended):**
- **Agent + Backend: Go.** Best ecosystem for Docker API, client-go (k8s), and eBPF (cilium/ebpf).
- **Frontend: React + TypeScript + Vite.** Graph rendering via **Cytoscape.js** (MVP) →
  **Sigma.js/PixiJS (WebGL)** for scale. D3 for charts.
- **Datastores:** Postgres (relational/state), VictoriaMetrics or Prometheus (metrics),
  ClickHouse (flow logs — high volume), S3-compatible object store (SBOMs, scan reports).
- **Transport:** Agent→Backend over gRPC with mTLS. Backend→Frontend over WebSocket.
- **Scanning:** Trivy as the primary engine (image CVE + SBOM + misconfig in one binary);
  Grype as an optional second engine for SBOM-based deep matching.

> If the team is JS-only and eBPF is deferred, the Backend may be Node/TS. The **Agent should still
> be Go** the moment eBPF is on the table — eBPF in Go via `cilium/ebpf` is the path of least pain.

---

## 3. Data model (canonical entities)

Build these as the shared vocabulary across all chunks.

- **Org** — tenant boundary. `id, name, slug, created_at, plan`.
- **User** — `id, email, name, password_hash/sso_subject, created_at`.
- **Membership** — `user_id, org_id, role(owner|admin|member|viewer)`.
- **Connection** — an attached infra source. `id, org_id, type(docker|k8s), name, status,
  agent_token, last_seen_at, metadata(jsonb)`.
- **Host/Node** — `id, connection_id, hostname, os, kernel, cpu_cores, mem_total, agent_version`.
- **Container** — `id, connection_id, host_id, runtime_id, name, image, image_digest, state,
  created_at, labels(jsonb), ports(jsonb)`. For k8s also: `pod_id, namespace, owner_kind`.
- **Pod / Deployment / Service / Namespace** — k8s objects, parent/child links.
- **Network** — `id, connection_id, name, driver, scope, subnet`.
- **Edge (Flow link)** — `src_container_id, dst_container_id, dst_ip, dst_port, protocol,
  first_seen, last_seen` (aggregated view of flows).
- **MetricSample** — time-series: `entity_id, metric(cpu|mem|net_rx|net_tx|disk_r|disk_w), ts, value`.
- **FlowEvent** — raw/aggregated network event: `ts, connection_id, src, dst, proto, bytes,
  packets, l7(jsonb optional), latency_ms`.
- **Image** — `digest, repo, tag, os, size, layers(jsonb)`.
- **Scan** — `id, image_digest, engine, started_at, finished_at, status, summary(jsonb)`.
- **Finding** — `id, scan_id, cve_id, severity, package, installed_version, fixed_version,
  cvss, fixable(bool), description`.
- **Misconfig** — `id, connection_id, entity_id, rule_id, severity, title, remediation`.
- **AuditLog** — `id, org_id, actor_id, action, target, ts, metadata`.

---

## 4. The chunks

> Legend — each chunk: **Goal · Build · DoD (definition of done)**.

### PHASE A — Foundations

#### Chunk 1 — Repo, tooling, CI skeleton
- **Goal:** A monorepo that builds and runs nothing-yet, with CI green.
- **Build:**
  - Monorepo layout:
    ```
    /backend        (Go module)
    /agent          (Go module)
    /frontend       (Vite + React + TS)
    /sdks           (go/ python/ ts/ — empty stubs)
    /deploy         (docker-compose.yml, k8s manifests, helm later)
    /docs
    ```
  - `docker-compose.yml` that stands up Postgres + (VictoriaMetrics OR Prometheus) + ClickHouse +
    MinIO (S3) — all empty/idle.
  - Linting + formatting + a CI workflow (build + lint + test) for each package.
  - `Makefile` / `Taskfile` with `make dev`, `make test`, `make lint`.
- **DoD:** `make dev` brings up all infra containers; `make test` passes (even with zero tests);
  CI is green on a pushed branch.

#### Chunk 2 — Backend skeleton + health + config
- **Goal:** Backend boots, reads config, exposes `/healthz` and `/version`.
- **Build:**
  - HTTP server (Go: chi/echo/gin — pick one, document it).
  - Config loader (env + file), structured logging, request IDs, graceful shutdown.
  - DB connection pool to Postgres; migration tooling (golang-migrate or goose).
  - `/healthz` (checks DB), `/version`.
- **DoD:** `curl /healthz` returns 200 with DB OK; migrations run on boot; config documented in `docs/config.md`.

#### Chunk 3 — Auth, Orgs, RBAC
- **Goal:** Users can sign up, create an org, invite members with roles.
- **Build:**
  - Migrations for `users, orgs, memberships, audit_logs`.
  - Email+password auth (argon2id hashes), JWT access + refresh tokens, session revocation.
  - Endpoints: register, login, refresh, logout, me; create org, list my orgs, invite member,
    accept invite, list members, change role, remove member.
  - **RBAC middleware**: every org-scoped route requires membership; enforce role
    (owner > admin > member > viewer). Centralize in one authz helper.
  - AuditLog write on every mutating action.
  - Leave an interface seam for **SSO/SAML/OIDC** (implement in Phase E).
- **DoD:** Full auth + org + invite flow works via API tests; a viewer cannot mutate; audit rows appear.

---

### PHASE B — Get real data from one Docker host

#### Chunk 4 — Agent skeleton + secure enrollment
- **Goal:** A Go agent binary that registers to the backend and heartbeats.
- **Build:**
  - Agent reads an **enrollment token** (generated when a Connection is created) + backend URL.
  - Agent ↔ Backend transport: gRPC over TLS (mTLS in Phase E; start with token-auth TLS).
  - Backend endpoints: create Connection (returns token), agent enroll, agent heartbeat.
  - Connection status state machine: `pending → connected → degraded → disconnected`.
  - Agent ships `Host/Node` info (hostname, os, kernel, cpu, mem, agent version) on enroll.
- **DoD:** Start agent with a token → Connection flips to `connected`, host row created, heartbeat
  updates `last_seen_at`; killing the agent flips status to `disconnected` after timeout.

#### Chunk 5 — Docker topology collection
- **Goal:** Agent reports live containers + networks from the Docker Engine API.
- **Build:**
  - Agent uses the Docker API (mount `/var/run/docker.sock` read-only) to list containers,
    images, networks; subscribe to Docker events for create/start/stop/destroy.
  - Map to canonical `Container`, `Image`, `Network` entities; send full snapshot on connect,
    then incremental deltas on events.
  - Backend persists topology per connection; exposes:
    `GET /orgs/:id/connections/:cid/topology` → nodes + edges (edges empty for now).
  - Reconciliation: containers gone for N minutes are marked `removed`.
- **DoD:** Run 3–4 containers on the host; API returns them with image, state, ports, labels;
  `docker stop` reflects in the API within seconds.

---

### PHASE C — The visualizer (headline feature)

#### Chunk 6 — Frontend shell + topology graph (static)
- **Goal:** Log in, pick an org/connection, see the container topology as an interactive graph.
- **Build:**
  - Frontend app shell: auth screens, org switcher, left nav, connection picker.
  - Graph canvas with **Cytoscape.js**: containers = nodes, networks = grouping/zones.
  - Node styling by state (running/stopped/unhealthy); labels = container name + image.
  - Click a node → side panel with details (image, ports, labels, created, host).
  - Layouts: force-directed + grouped-by-network; zoom/pan; search/filter box.
  - Data via the topology REST endpoint (poll for now).
- **DoD:** A real Docker host's containers render as a clean, navigable graph; clicking a node
  shows its details; search filters nodes. **This is the first demo-able milestone.**

#### Chunk 7 — Live updates over WebSocket
- **Goal:** The graph updates in real time without refresh.
- **Build:**
  - Backend WebSocket endpoint per org/connection; on agent deltas, fan-out topology patches.
  - Frontend subscribes; nodes appear/disappear/recolor smoothly (animate transitions).
  - Connection/agent status surfaced in the UI (connected/degraded badges).
  - Backpressure & reconnect logic on both ends; resync on reconnect.
- **DoD:** `docker run` / `docker stop` on the host animates the graph live within ~1–2s; dropping
  and restoring the WS reconciles state with no ghosts.

---

### PHASE D — Metrics

#### Chunk 8 — Per-container resource metrics (collection + storage)
- **Goal:** CPU, memory, disk I/O, network I/O per container, stored as time-series.
- **Build:**
  - Agent collects from cgroups (v2) / Docker stats / cAdvisor lib at a fixed interval.
  - Metrics shipped to backend → written to **VictoriaMetrics/Prometheus** with labels
    (org, connection, host, container, metric).
  - Backend query endpoints: instantaneous + range queries per entity.
  - Define retention + downsampling policy in `docs/metrics.md`.
- **DoD:** For each running container, CPU% and memory are queryable over a time range and match
  `docker stats` within reasonable tolerance.

#### Chunk 9 — Metrics in the UI (sparklines + dashboards)
- **Goal:** See "how much CPU is **this** container using" at a glance and in detail.
- **Build:**
  - Sparklines on graph nodes (live CPU/mem mini-charts).
  - Node side-panel: detailed charts (CPU, mem, net rx/tx, disk r/w) with range picker.
  - Host/connection dashboard: top consumers, "noisy neighbor" highlight, totals.
  - "Color graph by CPU/mem" heatmap mode.
- **DoD:** Load-test a container (e.g., `stress`) and watch its node heat up and its charts spike
  live; top-consumers list ranks correctly.

---

### PHASE E — Network & packet flow (the moat)

#### Chunk 10 — Connection edges from /proc (no eBPF yet)
- **Goal:** Draw real "who talks to whom" edges cheaply before tackling eBPF.
- **Build:**
  - Agent samples `/proc/net/{tcp,tcp6,udp}` + socket→container mapping (via cgroup/pid namespace)
    to derive active connections between containers and to external IPs.
  - Backend aggregates into `Edge` records; topology endpoint now returns edges.
  - Frontend draws edges; external endpoints shown as distinct nodes (e.g., DB, internet).
- **DoD:** Two containers talking (e.g., app→db) show a connecting edge; the edge disappears when
  traffic stops; external calls render as edges to external nodes.

#### Chunk 11 — Flow storage + bandwidth on edges
- **Goal:** Edges carry volume/direction/protocol; queryable flow history.
- **Build:**
  - Introduce **ClickHouse** for `FlowEvent` (high volume). Agent ships periodic per-flow
    byte/packet counters.
  - Edge thickness = throughput; color = protocol; arrow = direction.
  - Flow explorer view: table of flows with filters (src, dst, port, proto, time).
- **DoD:** Generating traffic visibly thickens the right edge; the flow explorer lists matching
  flows with byte/packet counts; history is queryable over a time window.

#### Chunk 12 — eBPF flow capture (the hard, differentiating part)
- **Goal:** Accurate, low-overhead flow + latency (and optional L7) via eBPF.
- **Build (start rough, iterate):**
  - Agent loads eBPF programs via **cilium/ebpf** to trace TCP/UDP connect/accept/send/recv and
    retransmits; attribute to container via cgroup id / pid namespace. (Reference patterns:
    libebpfflow, Hubble, Caretta.)
  - Emit flow events with bytes/packets/latency; feed the same `FlowEvent` pipeline as Chunk 11
    (eBPF replaces/augments the /proc sampler).
  - Kernel/version guardrails: feature-detect; **fall back to /proc sampler** when eBPF
    unavailable (older kernels, Docker Desktop, restricted hosts).
  - Document required privileges (CAP_BPF/CAP_SYS_ADMIN, mounts) clearly.
  - **Stretch:** L7 visibility (HTTP/gRPC/DNS/SQL) — gate behind a flag; this is deep work.
- **DoD:** On a supported Linux host, eBPF flows match reality with sub-1% CPU overhead target;
  latency appears on edges; on an unsupported host it cleanly falls back to /proc with a UI note.

#### Chunk 13 — Packet-flow animation (the "wow")
- **Goal:** Animated particles along edges showing live direction + volume — the headline visual.
- **Build:**
  - Move graph rendering to **WebGL (Sigma.js/PixiJS)** if Cytoscape can't keep ~60fps with
    animated edges at scale.
  - Particle density/speed ∝ throughput; direction ∝ flow direction; protocol-colored.
  - Performance budget: smooth at 500–1000 nodes; LOD (level of detail) — simplify when zoomed out.
- **DoD:** Live traffic produces smooth animated flow along edges at target node counts without
  frame drops; toggling animation off restores a static view.

---

### PHASE F — Kubernetes

#### Chunk 14 — Kubernetes topology
- **Goal:** Attach a cluster; see pods/deployments/services/namespaces as a nested topology.
- **Build:**
  - Agent runs as a **DaemonSet**; uses client-go to watch pods/services/deployments/endpoints
    (+ kubelet for node-local container stats).
  - Namespaces = zones; pods contain containers; services link to their pods; ingress shown.
  - Reuse the same canonical entities + topology/WS pipeline.
- **DoD:** A real cluster renders with namespaces/pods/services; scaling a deployment animates new
  pods in live; clicking a service shows its backing pods.

#### Chunk 15 — Kubernetes flows + metrics
- **Goal:** Pod-to-pod and pod-to-service traffic + per-pod metrics.
- **Build:**
  - eBPF/`proc` flow capture per node attributing to pods; service-aware aggregation.
  - Per-pod CPU/mem incl. requests/limits vs actual usage.
  - Network policy **visualization** + "what would this policy block" dry-run simulator.
- **DoD:** Pod-to-pod edges animate; requests/limits vs usage shown per pod; selecting a
  NetworkPolicy highlights what it allows/blocks on the graph.

---

### PHASE G — Security

#### Chunk 16 — Image vulnerability scanning (Trivy)
- **Goal:** Scan images of running containers; list CVEs with severity + fixes.
- **Build:**
  - Scan orchestration: backend enqueues scans; a **scanner worker** runs **Trivy**
    (`trivy image --format json`) and parses results into `Scan` + `Finding` rows.
  - Trigger scans: on new image observed, on schedule, and on-demand from UI.
  - Store **SBOM** (Trivy CycloneDX/SPDX) in object storage; dedupe scans by image digest.
  - UI: per-image findings (CVE, severity, package, installed→fixed version, CVSS, fixable),
    filters, and CVE detail. Findings overlaid as badges on graph nodes.
  - **Optional:** Grype as a second engine for SBOM-based deep matching (`engine` field already exists).
- **DoD:** A known-vulnerable image (e.g., an old nginx/node tag) returns a CVE list matching
  Trivy CLI; the node shows a severity badge; SBOM is downloadable.

#### Chunk 17 — Misconfig / runtime hardening checks
- **Goal:** Flag risky runtime config: privileged containers, root user, exposed secrets/ports,
  CIS Docker/Kubernetes benchmark items.
- **Build:**
  - Agent + backend rules engine producing `Misconfig` findings (privileged, hostNetwork,
    writable root fs, added caps, secrets in env, dangerous mounts, exposed dashboards).
  - Trivy config/k8s misconfig scanning for manifests where available.
  - **Security score** per connection/org (weighted by severity); trend over time.
- **DoD:** Launch a `--privileged` container → a high-severity misconfig appears with remediation;
  the connection's security score drops and recovers when fixed.

#### Chunk 18 — Alerting & notifications
- **Goal:** Tell people when something is wrong (new critical CVE, resource threshold, status).
- **Build:**
  - Rule definitions (threshold/condition + target). Evaluation loop.
  - Channels: email, Slack, generic webhook. Per-org notification settings.
  - Alert history + ack/resolve.
- **DoD:** A CPU threshold and a "new CRITICAL CVE" rule both fire to a test Slack webhook and
  appear in alert history; acking updates state.

---

### PHASE H — Platform, SDKs, extensibility

#### Chunk 19 — Public API hardening + GraphQL
- **Goal:** A stable, documented external API surface.
- **Build:**
  - Versioned REST (`/api/v1`) covering topology, metrics, flows, scans, findings, orgs.
  - GraphQL schema for flexible reads (topology + metrics + findings in one query).
  - **API keys/PATs** scoped to org + role; rate limiting; pagination; consistent errors.
  - OpenAPI spec + GraphQL schema published; auto-generated docs.
- **DoD:** An external script can authenticate with an API key and pull topology + findings via
  both REST and GraphQL; OpenAPI validates; rate limiting works.

#### Chunk 20 — SDKs (Go, Python, TypeScript)
- **Goal:** First-class client libraries.
- **Build:**
  - Generate base clients from OpenAPI; hand-polish ergonomics.
  - Each SDK: auth, list connections, fetch topology, query metrics, trigger/fetch scans,
    stream live updates (WS) where feasible.
  - Examples + README per SDK; publish to module registries (later).
- **DoD:** A 15-line example in each language fetches topology and prints the top-CPU container and
  any critical CVEs.

#### Chunk 21 — CI/CD plugins + webhooks
- **Goal:** Fit DevSecOps pipelines; fail builds on critical CVEs.
- **Build:**
  - GitHub Action + GitLab CI template that scan an image via the platform (or push results) and
    fail on threshold; SARIF output for code-scanning tabs.
  - Outbound webhooks for events (scan finished, new critical finding, status change).
  - Optional: admission-controller pattern for deploy-time gating (k8s).
- **DoD:** A sample pipeline fails on a critical CVE and passes once the image is patched; SARIF
  shows in GitHub's security tab; webhooks deliver on events.

#### Chunk 22 — Enterprise readiness
- **Goal:** Make it adoptable by serious DevOps orgs ("industry standard").
- **Build:**
  - **SSO/SAML/OIDC** (wire the seam from Chunk 3); SCIM optional.
  - **mTLS** agent↔backend; secret rotation; agent auto-update channel.
  - Multi-region/HA notes; backup/restore runbooks; data retention controls per org.
  - **Audit log export**; RBAC custom roles; org-level data isolation tests.
  - Helm chart for the whole platform; air-gapped install + offline vuln DB mirror (Trivy/Grype
    support offline DBs).
  - Observability of the platform itself (its own metrics/logs/traces).
- **DoD:** SSO login works against a test IdP; agents use mTLS; Helm installs the full stack; an
  air-gapped scan works against a mirrored vuln DB.

#### Chunk 23 — Time-travel / replay (high-value differentiator)
- **Goal:** Scrub backward through topology + traffic to debug past incidents.
- **Build:**
  - Snapshot topology state + retain flow/metric history; a time slider drives graph state.
  - Replay flow animation for a chosen window.
- **DoD:** Pick a 10-minute window in the past and watch the topology + flows replay accurately.

---

## 5. Cross-cutting requirements (apply to every chunk)

- **Multi-tenancy isolation:** every query is org-scoped; add tests that prove org A can't read
  org B's data.
- **Security:** least-privilege agent; read-only socket mounts; no secrets in logs; encrypt
  tokens at rest; sign agent releases.
- **Performance budgets:** agent CPU overhead target < 2% per host; UI smooth to 500–1000 nodes;
  flow ingestion sized for ClickHouse.
- **Testing:** unit tests per package; integration tests using a real Docker host + a kind/k3d
  cluster in CI; an end-to-end "golden path" test (enroll → topology → metric → scan).
- **Docs:** every chunk updates `/docs` (config, API, deploy, agent install).
- **Telemetry of the product itself:** structured logs + metrics from day one.

---

## 6. Tech choices cheat-sheet

| Concern | Primary | Why |
|---|---|---|
| Agent + Backend | Go | Docker API, client-go, cilium/ebpf all first-class in Go |
| Frontend | React + TS + Vite | Standard, fast, big ecosystem |
| Graph (MVP) | Cytoscape.js | Quick to get a good interactive graph |
| Graph (scale) | Sigma.js / PixiJS (WebGL) | 60fps with animated edges at 1000+ nodes |
| Charts | D3 / visx | Custom sparklines + dashboards |
| Relational DB | Postgres | State, orgs, topology, findings |
| Metrics TSDB | VictoriaMetrics (or Prometheus) | Efficient time-series + retention |
| Flow store | ClickHouse | High-volume flow events, fast aggregation |
| Object store | S3 / MinIO | SBOMs, scan reports |
| Vuln scan | Trivy (primary), Grype (optional 2nd engine) | Trivy = CVE+SBOM+misconfig in one; Grype = deep SBOM matching |
| Agent↔Backend | gRPC + mTLS | Efficient, secure, streaming |
| Backend↔Frontend | WebSocket | Live topology/flow push |
| eBPF | cilium/ebpf | Mature Go eBPF tooling |

---

## 7. Suggested milestones (for demos / fundraising / pilots)

1. **M1 (Chunks 1–7):** Live animated-ready Docker topology with real-time updates. *Demo-able.*
2. **M2 (Chunks 8–9):** + per-container metrics, heatmaps, top consumers.
3. **M3 (Chunks 10–13):** + real network edges and eBPF packet-flow animation. *The "wow" demo.*
4. **M4 (Chunks 14–15):** + Kubernetes parity.
5. **M5 (Chunks 16–18):** + vulnerability scanning, misconfig, alerting. *Sellable security story.*
6. **M6 (Chunks 19–23):** + API/SDKs, CI/CD, enterprise readiness, time-travel. *Industry-standard.*

---

## 8. First action for the implementer

Start **Chunk 1**. Produce the monorepo, `docker-compose.yml`, Makefile, and green CI. Do not
write product features until `make dev` + `make test` + CI all pass. Then proceed to Chunk 2.
