# ContainerScope — Configuration Reference

## Overview

All configuration is loaded from environment variables. You can also use a `.env` file in the backend directory.

## Environment Variables

### Server

| Variable | Default | Description |
|----------|---------|-------------|
| `BACKEND_PORT` | `8080` | HTTP server port |
| `BACKEND_HOST` | `0.0.0.0` | HTTP server bind address |
| `LOG_LEVEL` | `info` | Log level (debug, info, warn, error) |

### Database (Postgres)

| Variable | Default | Description |
|----------|---------|-------------|
| `POSTGRES_HOST` | `localhost` | Postgres host |
| `POSTGRES_PORT` | `5432` | Postgres port |
| `POSTGRES_USER` | `containerscope` | Postgres user |
| `POSTGRES_PASSWORD` | `containerscope` | Postgres password |
| `POSTGRES_DB` | `containerscope` | Postgres database name |
| `POSTGRES_SSLMODE` | `disable` | Postgres SSL mode |

### Authentication

| Variable | Default | Description |
|----------|---------|-------------|
| `JWT_SECRET` | - | JWT signing secret (min 32 chars) |
| `JWT_ACCESS_TTL` | `15m` | Access token time-to-live |
| `JWT_REFRESH_TTL` | `168h` | Refresh token time-to-live (7 days) |

### VictoriaMetrics (Metrics TSDB)

| Variable | Default | Description |
|----------|---------|-------------|
| `VICTORIAMETRICS_URL` | `http://localhost:8428` | VictoriaMetrics API URL |

### ClickHouse (Flow Store)

| Variable | Default | Description |
|----------|---------|-------------|
| `CLICKHOUSE_HOST` | `localhost` | ClickHouse host |
| `CLICKHOUSE_PORT` | `9000` | ClickHouse native port |
| `CLICKHOUSE_USER` | `containerscope` | ClickHouse user |
| `CLICKHOUSE_PASSWORD` | `containerscope` | ClickHouse password |
| `CLICKHOUSE_DB` | `containerscope` | ClickHouse database name |

### MinIO / S3 (Object Store)

| Variable | Default | Description |
|----------|---------|-------------|
| `S3_ENDPOINT` | `localhost:9100` | S3 endpoint |
| `S3_ACCESS_KEY` | `containerscope` | S3 access key |
| `S3_SECRET_KEY` | `containerscope123` | S3 secret key |
| `S3_BUCKET` | `containerscope` | S3 bucket name |
| `S3_USE_SSL` | `false` | Use SSL for S3 |

### Agent

| Variable | Default | Description |
|----------|---------|-------------|
| `BACKEND_URL` | `localhost:8081` | Backend gRPC URL |
| `ENROLLMENT_TOKEN` | - | Enrollment token from connection |

## Infrastructure Services

### Port Map

| Service | Port | Protocol | Description |
|---------|------|----------|-------------|
| Frontend | 3000 | HTTP | React UI |
| Backend API | 8080 | HTTP | REST/GraphQL API |
| gRPC | 8081 | gRPC | Agent communication |
| Postgres | 5432 | TCP | Relational database |
| VictoriaMetrics | 8428 | HTTP | Metrics TSDB |
| ClickHouse HTTP | 8123 | HTTP | Flow queries |
| ClickHouse Native | 9000 | TCP | Flow storage |
| MinIO API | 9100 | HTTP | S3-compatible API |
| MinIO Console | 9001 | HTTP | Web console |

### Docker Compose

```yaml
services:
  postgres:
    image: postgres:16-alpine
    ports:
      - "5432:5432"
    environment:
      POSTGRES_USER: containerscope
      POSTGRES_PASSWORD: containerscope
      POSTGRES_DB: containerscope

  victoriametrics:
    image: victoriametrics/victoria-metrics:v1.96.0
    ports:
      - "8428:8428"

  clickhouse:
    image: clickhouse/clickhouse-server:24.1-alpine
    ports:
      - "8123:8123"
      - "9000:9000"

  minio:
    image: minio/minio:latest
    ports:
      - "9100:9000"
      - "9001:9001"
```

## API Endpoints

| Endpoint | Method | Description |
|----------|--------|-------------|
| `/healthz` | GET | Health check (includes DB ping) |
| `/version` | GET | Returns backend version |
| `/ping` | GET | Heartbeat endpoint |

## Database Migrations

Migrations run automatically on startup. Migration files are located in `backend/migrations/`.

### Migration Files

| File | Description |
|------|-------------|
| `001_initial_schema.up.sql` | Core tables (orgs, users, connections, containers, etc.) |
| `002_metric_samples.up.sql` | Metric samples table |
| `003_edges.up.sql` | Network edges table |
| `004_vulnerabilities.up.sql` | Vulnerability scans and findings |

## Quick Start

### Development

```bash
# Copy environment template
cp .env.example .env

# Start infrastructure
docker compose up -d

# Start backend (in terminal 1)
cd backend && go run ./cmd/server

# Start frontend (in terminal 2)
cd frontend && npm run dev

# Start agent (in terminal 3)
cd agent && ENROLLMENT_TOKEN=<token> go run ./cmd/agent
```

### Production

```bash
# Generate secrets
make -f Makefile.production generate-secrets

# Configure environment
cp .env.production .env
# Edit .env with generated secrets

# Deploy
docker compose -f docker-compose.prod.yml up -d
```

## Configuration Files

### .env.example

```env
# Backend
BACKEND_PORT=8080
BACKEND_HOST=0.0.0.0
LOG_LEVEL=info

# Database
POSTGRES_HOST=localhost
POSTGRES_PORT=5432
POSTGRES_USER=containerscope
POSTGRES_PASSWORD=containerscope
POSTGRES_DB=containerscope
POSTGRES_SSLMODE=disable

# Authentication
JWT_SECRET=change-me-in-production-at-least-32-chars
JWT_ACCESS_TTL=15m
JWT_REFRESH_TTL=168h

# VictoriaMetrics
VICTORIAMETRICS_URL=http://localhost:8428

# ClickHouse
CLICKHOUSE_HOST=localhost
CLICKHOUSE_PORT=9000
CLICKHOUSE_USER=containerscope
CLICKHOUSE_PASSWORD=containerscope
CLICKHOUSE_DB=containerscope

# MinIO
S3_ENDPOINT=localhost:9100
S3_ACCESS_KEY=containerscope
S3_SECRET_KEY=containerscope123
S3_BUCKET=containerscope
S3_USE_SSL=false
```

### .env.production

```env
# Backend
BACKEND_PORT=8080
BACKEND_HOST=0.0.0.0
LOG_LEVEL=warn

# Database
POSTGRES_HOST=postgres
POSTGRES_PORT=5432
POSTGRES_USER=containerscope
POSTGRES_PASSWORD=<strong-password>
POSTGRES_DB=containerscope
POSTGRES_SSLMODE=require

# Authentication
JWT_SECRET=<random-64-char-string>
JWT_ACCESS_TTL=15m
JWT_REFRESH_TTL=168h

# VictoriaMetrics
VICTORIAMETRICS_URL=http://victoriametrics:8428

# ClickHouse
CLICKHOUSE_HOST=clickhouse
CLICKHOUSE_PORT=9000
CLICKHOUSE_USER=containerscope
CLICKHOUSE_PASSWORD=<strong-password>
CLICKHOUSE_DB=containerscope

# MinIO
S3_ENDPOINT=minio:9000
S3_ACCESS_KEY=containerscope
S3_SECRET_KEY=<strong-password>
S3_BUCKET=containerscope
S3_USE_SSL=false
```

## Security Considerations

### JWT Secret

- Generate a random string of at least 32 characters
- Use different secrets for development and production
- Store securely (e.g., environment variable, secrets manager)

```bash
# Generate a random secret
openssl rand -base64 48
```

### Database Passwords

- Use strong passwords in production
- Never commit passwords to version control
- Use environment variables or secrets manager

### CORS

Configure allowed origins in `backend/internal/server/server.go`:

```go
cors.Handler(cors.Options{
    AllowedOrigins:   []string{"https://app.containerscope.io"},
    AllowedMethods:   []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
    AllowedHeaders:   []string{"Accept", "Authorization", "Content-Type"},
    AllowCredentials: true,
    MaxAge:           300,
})
```

## Troubleshooting

### Database Connection Failed

```bash
# Check if Postgres is running
docker compose ps postgres

# Check logs
docker compose logs postgres

# Test connection
psql -h localhost -U containerscope -d containerscope
```

### Migration Failed

```bash
# Check migration status
docker compose exec postgres psql -U containerscope -c "SELECT * FROM schema_migrations;"

# Reset migrations
docker compose down -v
docker compose up -d
```

### Agent Connection Failed

```bash
# Check agent logs
docker compose logs agent

# Verify enrollment token
curl -X POST http://localhost:8080/api/v1/agent/enroll \
  -H "Content-Type: application/json" \
  -d '{"enrollment_token": "your-token"}'
```

## References

- [PostgreSQL Documentation](https://www.postgresql.org/docs/)
- [VictoriaMetrics Documentation](https://docs.victoriametrics.com/)
- [ClickHouse Documentation](https://clickhouse.com/docs/en/)
- [MinIO Documentation](https://min.io/docs/)
- [JWT Documentation](https://jwt.io/introduction)
