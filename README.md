# ContainerScope

A multi-tenant DevOps observability and security visualizer for Docker and Kubernetes.

![Architecture](https://img.shields.io/badge/Architecture-Microservices-blue)
![Go](https://img.shields.io/badge/Go-1.22+-00ADD8?logo=go)
![React](https://img.shields.io/badge/React-18-61DAFB?logo=react)
![TypeScript](https://img.shields.io/badge/TypeScript-5.3-3178C6?logo=typescript)

## Overview

ContainerScope is an enterprise-grade platform for monitoring, analyzing, and securing containerized infrastructure. It provides real-time topology visualization, performance metrics, vulnerability scanning, and intelligent alerting.

## Quick Start

```bash
# Clone and start
git clone https://github.com/containerscope/containerscope.git
cd containerscope
cp .env.example .env
docker compose up -d

# Access
open http://localhost:3000

# Default credentials
# Email: admin@containerscope.io
# Password: admin123
```

## Architecture

```
┌─────────────────────────────────────────────────────────────┐
│                      Frontend (React + TS)                   │
│  ┌──────────┐  ┌──────────┐  ┌──────────┐  ┌──────────┐   │
│  │ Dashboard │  │ Topology │  │ Security │  │  Alerts  │   │
│  └──────────┘  └──────────┘  └──────────┘  └──────────┘   │
└────────────────────────┬────────────────────────────────────┘
                         │ REST + WebSocket
┌────────────────────────┴────────────────────────────────────┐
│                      Backend (Go + Chi)                       │
│  ┌──────────┐  ┌──────────┐  ┌──────────┐  ┌──────────┐   │
│  │   Auth   │  │ Topology │  │ Metrics  │  │  Alerts  │   │
│  │   Orgs   │  │  Flows   │  │   Vulns  │  │  Agent   │   │
│  └──────────┘  └──────────┘  └──────────┘  └──────────┘   │
└────────────────────────┬────────────────────────────────────┘
                         │
┌────────────────────────┴────────────────────────────────────┐
│                    Data Layer                                │
│  ┌──────────┐  ┌──────────┐  ┌──────────┐  ┌──────────┐   │
│  │ Postgres │  │ Victoria │  │ClickHouse│  │   MinIO  │   │
│  │          │  │ Metrics  │  │          │  │          │   │
│  └──────────┘  └──────────┘  └──────────┘  └──────────┘   │
└─────────────────────────────────────────────────────────────┘
                         │
┌────────────────────────┴────────────────────────────────────┐
│                      Agent (Go)                              │
│  ┌──────────┐  ┌──────────┐  ┌──────────┐  ┌──────────┐   │
│  │  Docker  │  │   K8s    │  │  eBPF    │  │  Trivy   │   │
│  │ Collector│  │ Collector│  │  Flows   │  │ Scanner  │   │
│  └──────────┘  └──────────┘  └──────────┘  └──────────┘   │
└─────────────────────────────────────────────────────────────┘
```

## Tech Stack

### Backend
| Technology | Purpose |
|------------|---------|
| Go 1.22+ | Primary language |
| Chi | HTTP router |
| pgx/v5 | PostgreSQL driver |
| golang-migrate | Database migrations |
| JWT | Authentication |
| gRPC | Agent communication |

### Frontend
| Technology | Purpose |
|------------|---------|
| React 18 | UI framework |
| TypeScript 5.3 | Type safety |
| Vite | Build tool |
| Tailwind CSS | Styling |
| shadcn/ui | UI components |
| Framer Motion | Animations |
| Recharts | Charts |
| Cytoscape.js | 2D topology |
| Three.js | 3D topology |
| Zustand | State management |

### Data Stores
| Technology | Purpose |
|------------|---------|
| PostgreSQL 16 | Relational data |
| VictoriaMetrics | Time-series metrics |
| ClickHouse | Flow events |
| MinIO | Object storage |

## Project Structure

```
containerscope/
├── agent/                      # Go agent
│   ├── cmd/agent/              # Entry point
│   └── internal/
│       ├── compliance/         # Misconfiguration checks
│       ├── docker/             # Docker collector
│       ├── ebpf/               # eBPF flow capture
│       ├── flows/              # Flow processing
│       ├── grpc/               # gRPC client
│       ├── host/               # Host info
│       ├── kubernetes/         # K8s collector
│       ├── metrics/            # Metrics collector
│       └── scanner/            # Trivy integration
├── backend/                    # Go backend
│   ├── cmd/server/             # Entry point
│   ├── handlers/               # HTTP handlers
│   ├── internal/
│   │   ├── alerts/             # Alert engine
│   │   ├── auth/               # JWT, passwords
│   │   ├── config/             # Configuration
│   │   ├── db/                 # Database
│   │   ├── flows/              # ClickHouse client
│   │   ├── graphql/            # GraphQL handler
│   │   ├── logger/             # Logging
│   │   ├── metrics/            # VictoriaMetrics client
│   │   ├── middleware/          # Auth, RBAC, security
│   │   ├── server/             # HTTP server
│   │   ├── store/              # Data store
│   │   └── ws/                 # WebSocket hub
│   ├── migrations/             # SQL migrations
│   └── proto/agent/            # gRPC proto
├── frontend/                   # React + TypeScript
│   └── src/
│       ├── components/         # UI components
│       ├── contexts/           # React contexts
│       ├── hooks/              # Custom hooks
│       ├── lib/                # API, store
│       └── pages/              # Page components
├── sdks/                       # Client SDKs
├── deploy/                     # Deployment configs
├── docs/                       # Documentation
├── docker-compose.yml          # Development
└── docker-compose.prod.yml     # Production
```

## API Documentation

### Authentication

All API endpoints require JWT authentication:
```
Authorization: Bearer <token>
```

### REST API

#### Auth
| Method | Endpoint | Description |
|--------|----------|-------------|
| POST | `/api/v1/auth/register` | Register new user |
| POST | `/api/v1/auth/login` | Login, get tokens |
| POST | `/api/v1/auth/refresh` | Refresh access token |
| GET | `/api/v1/auth/me` | Get current user |

#### Organizations
| Method | Endpoint | Description |
|--------|----------|-------------|
| POST | `/api/v1/orgs` | Create organization |
| GET | `/api/v1/orgs` | List organizations |
| GET | `/api/v1/orgs/{id}` | Get organization |
| GET | `/api/v1/orgs/{id}/members` | List members |
| POST | `/api/v1/orgs/{id}/invite` | Invite member |
| PUT | `/api/v1/orgs/{id}/members/{id}/role` | Update role |
| DELETE | `/api/v1/orgs/{id}/members/{id}` | Remove member |

#### Connections
| Method | Endpoint | Description |
|--------|----------|-------------|
| POST | `/api/v1/orgs/{id}/connections` | Create connection |
| GET | `/api/v1/orgs/{id}/connections` | List connections |
| GET | `/api/v1/orgs/{id}/connections/{id}` | Get connection |
| GET | `/api/v1/orgs/{id}/connections/{id}/hosts` | List hosts |

#### Topology
| Method | Endpoint | Description |
|--------|----------|-------------|
| GET | `/api/v1/orgs/{id}/connections/{id}/topology` | Get topology |

#### Metrics
| Method | Endpoint | Description |
|--------|----------|-------------|
| GET | `/api/v1/orgs/{id}/connections/{id}/metrics` | Range query |
| GET | `/api/v1/orgs/{id}/connections/{id}/metrics/instant` | Instant values |

#### Flows
| Method | Endpoint | Description |
|--------|----------|-------------|
| GET | `/api/v1/orgs/{id}/connections/{id}/flows` | Query flows |
| GET | `/api/v1/orgs/{id}/connections/{id}/bandwidth` | Edge bandwidth |

#### Vulnerabilities
| Method | Endpoint | Description |
|--------|----------|-------------|
| GET | `/api/v1/orgs/{id}/connections/{id}/vulns` | List scans |
| POST | `/api/v1/orgs/{id}/connections/{id}/vulns` | Record scan |
| GET | `/api/v1/orgs/{id}/connections/{id}/vulns/{id}` | Get vulns |
| GET | `/api/v1/orgs/{id}/connections/{id}/vulns/dashboard` | Dashboard |

#### Alerts
| Method | Endpoint | Description |
|--------|----------|-------------|
| GET | `/api/v1/orgs/{id}/connections/{id}/alerts` | List alerts |
| GET | `/api/v1/orgs/{id}/connections/{id}/alerts/firing` | Firing alerts |
| GET | `/api/v1/orgs/{id}/connections/{id}/alerts/rules` | List rules |
| POST | `/api/v1/orgs/{id}/connections/{id}/alerts/rules` | Create rule |
| GET | `/api/v1/orgs/{id}/connections/{id}/alerts/channels` | List channels |
| POST | `/api/v1/orgs/{id}/connections/{id}/alerts/channels` | Create channel |
| POST | `/api/v1/orgs/{id}/connections/{id}/alerts/silence` | Silence alert |

#### Agent
| Method | Endpoint | Description |
|--------|----------|-------------|
| POST | `/api/v1/agent/enroll` | Agent enrollment |
| POST | `/api/v1/agent/heartbeat` | Agent heartbeat |
| POST | `/api/v1/agent/topology` | Sync topology |

#### System
| Method | Endpoint | Description |
|--------|----------|-------------|
| GET | `/healthz` | Health check |
| GET | `/version` | Version info |
| GET | `/ping` | Heartbeat |

### WebSocket

Endpoint: `/ws/orgs/{orgID}/connections/{connectionID}`

**Message Types:**
- `topology_update` - Full topology changed
- `container_add` - New container
- `container_del` - Container removed
- `container_update` - Container changed
- `status_change` - Connection status

## Database Schema

### Tables

| Table | Description |
|-------|-------------|
| `orgs` | Organizations |
| `users` | Users |
| `memberships` | User-Org relationships |
| `connections` | Agent connections |
| `hosts` | Host information |
| `containers` | Container state |
| `networks` | Docker networks |
| `edges` | Network edges |
| `metric_samples` | Resource metrics |
| `vulnerability_scans` | Vuln scan results |
| `vulnerabilities` | Individual vulns |
| `audit_logs` | Audit trail |

## Development

### Backend
```bash
cd backend
go mod tidy
go build ./...
go test ./...
go vet ./...
```

### Frontend
```bash
cd frontend
npm install
npm run dev        # Development
npm run build      # Production
npm run typecheck  # TypeScript check
npm run lint       # ESLint
```

### Agent
```bash
cd agent
go mod tidy
go build ./...
```

### Docker
```bash
docker compose up -d           # Development
docker compose build           # Build images
docker compose -f docker-compose.prod.yml up -d  # Production
```

## Configuration

### Environment Variables

| Variable | Default | Description |
|----------|---------|-------------|
| `BACKEND_PORT` | `8080` | Backend HTTP port |
| `BACKEND_HOST` | `0.0.0.0` | Backend bind address |
| `POSTGRES_HOST` | `localhost` | Postgres host |
| `POSTGRES_PORT` | `5432` | Postgres port |
| `POSTGRES_USER` | `containerscope` | Postgres user |
| `POSTGRES_PASSWORD` | `containerscope` | Postgres password |
| `POSTGRES_DB` | `containerscope` | Postgres database |
| `JWT_SECRET` | - | JWT signing secret (min 32 chars) |
| `JWT_ACCESS_TTL` | `15m` | Access token TTL |
| `JWT_REFRESH_TTL` | `168h` | Refresh token TTL |
| `VICTORIAMETRICS_URL` | `http://localhost:8428` | VictoriaMetrics URL |
| `CLICKHOUSE_HOST` | `localhost` | ClickHouse host |
| `CLICKHOUSE_PORT` | `9000` | ClickHouse port |
| `CLICKHOUSE_USER` | `containerscope` | ClickHouse user |
| `CLICKHOUSE_PASSWORD` | `containerscope` | ClickHouse password |
| `CLICKHOUSE_DB` | `containerscope` | ClickHouse database |

## Port Map

| Service | Port | Description |
|---------|------|-------------|
| Frontend | 3000 | React UI |
| Backend | 8080 | REST API |
| gRPC | 8081 | Agent communication |
| Postgres | 5432 | Database |
| VictoriaMetrics | 8428 | Metrics |
| ClickHouse HTTP | 8123 | Flow queries |
| ClickHouse Native | 9000 | Flow storage |
| MinIO API | 9100 | Object storage |
| MinIO Console | 9001 | Web console |

## Documentation

- [Architecture Spec](doc/ContainerScope_ARCHITECTURE_SPEC.md)
- [Build Spec](doc/ContainerScope_BUILD_SPEC.md)
- [Build Plan](doc/ContainerScope_BUILD_PLAN.md)
- [UI/UX Spec](doc/ContainerScope_UI_UX_SPEC_3D.md)
- [Configuration](docs/config.md)
- [API Reference](docs/api.md)
- [Metrics](docs/metrics.md)
- [eBPF](docs/ebpf.md)
- [Enterprise](docs/enterprise.md)
- [Time Travel](docs/time-travel.md)

## License

MIT License
