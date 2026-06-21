# ContainerScope API Reference

## Base URL

```
http://localhost:8080  # Development
https://api.containerscope.io  # Production
```

## Authentication

### JWT Tokens

Most endpoints require JWT authentication via Bearer token:

```bash
curl -H "Authorization: Bearer <token>" http://localhost:8080/api/v1/auth/me
```

### API Keys

For machine access, use API keys:

```bash
curl -H "X-API-Key: <api_key>" http://localhost:8080/api/v1/orgs
```

Or as query parameter:
```bash
curl "http://localhost:8080/api/v1/orgs?api_key=<api_key>"
```

## REST API

### Auth

#### Register User

```http
POST /api/v1/auth/register
Content-Type: application/json

{
  "email": "user@example.com",
  "name": "John Doe",
  "password": "securepassword"
}
```

**Response (201):**
```json
{
  "id": "uuid",
  "email": "user@example.com",
  "name": "John Doe",
  "created_at": "2024-01-01T00:00:00Z"
}
```

#### Login

```http
POST /api/v1/auth/login
Content-Type: application/json

{
  "email": "user@example.com",
  "password": "securepassword"
}
```

**Response (200):**
```json
{
  "access_token": "eyJhbGciOiJIUzI1NiIs...",
  "refresh_token": "eyJhbGciOiJIUzI1NiIs..."
}
```

#### Refresh Token

```http
POST /api/v1/auth/refresh
Content-Type: application/json

{
  "refresh_token": "eyJhbGciOiJIUzI1NiIs..."
}
```

**Response (200):**
```json
{
  "access_token": "eyJhbGciOiJIUzI1NiIs...",
  "refresh_token": "eyJhbGciOiJIUzI1NiIs..."
}
```

#### Get Current User

```http
GET /api/v1/auth/me
Authorization: Bearer <token>
```

**Response (200):**
```json
{
  "id": "uuid",
  "email": "user@example.com",
  "name": "John Doe",
  "created_at": "2024-01-01T00:00:00Z"
}
```

### Organizations

#### Create Organization

```http
POST /api/v1/orgs
Authorization: Bearer <token>
Content-Type: application/json

{
  "name": "My Company",
  "slug": "my-company"
}
```

**Response (201):**
```json
{
  "org": {
    "id": "uuid",
    "name": "My Company",
    "slug": "my-company",
    "plan": "free",
    "created_at": "2024-01-01T00:00:00Z"
  },
  "membership": {
    "user_id": "uuid",
    "org_id": "uuid",
    "role": "owner",
    "created_at": "2024-01-01T00:00:00Z"
  }
}
```

#### List Organizations

```http
GET /api/v1/orgs
Authorization: Bearer <token>
```

**Response (200):**
```json
[
  {
    "id": "uuid",
    "name": "My Company",
    "slug": "my-company",
    "plan": "free",
    "created_at": "2024-01-01T00:00:00Z"
  }
]
```

#### Get Organization

```http
GET /api/v1/orgs/{orgID}
Authorization: Bearer <token>
```

**Response (200):**
```json
{
  "id": "uuid",
  "name": "My Company",
  "slug": "my-company",
  "plan": "free",
  "created_at": "2024-01-01T00:00:00Z"
}
```

#### List Members

```http
GET /api/v1/orgs/{orgID}/members
Authorization: Bearer <token>
```

**Response (200):**
```json
[
  {
    "user_id": "uuid",
    "org_id": "uuid",
    "role": "owner",
    "created_at": "2024-01-01T00:00:00Z",
    "user": {
      "id": "uuid",
      "email": "user@example.com",
      "name": "John Doe",
      "created_at": "2024-01-01T00:00:00Z"
    }
  }
]
```

#### Invite Member

```http
POST /api/v1/orgs/{orgID}/invite
Authorization: Bearer <token>
Content-Type: application/json

{
  "email": "newuser@example.com",
  "role": "member"
}
```

**Response (201):**
```json
{
  "user_id": "uuid",
  "org_id": "uuid",
  "role": "member",
  "created_at": "2024-01-01T00:00:00Z"
}
```

#### Update Member Role

```http
PUT /api/v1/orgs/{orgID}/members/{userID}/role
Authorization: Bearer <token>
Content-Type: application/json

{
  "role": "admin"
}
```

**Response (200):**
```json
{
  "status": "ok"
}
```

#### Remove Member

```http
DELETE /api/v1/orgs/{orgID}/members/{userID}
Authorization: Bearer <token>
```

**Response (200):**
```json
{
  "status": "ok"
}
```

### Connections

#### Create Connection

```http
POST /api/v1/orgs/{orgID}/connections
Authorization: Bearer <token>
Content-Type: application/json

{
  "type": "docker",
  "name": "my-docker-host"
}
```

**Response (201):**
```json
{
  "id": "uuid",
  "org_id": "uuid",
  "type": "docker",
  "name": "my-docker-host",
  "status": "pending",
  "agent_token": "uuid",
  "created_at": "2024-01-01T00:00:00Z"
}
```

#### List Connections

```http
GET /api/v1/orgs/{orgID}/connections
Authorization: Bearer <token>
```

**Response (200):**
```json
[
  {
    "id": "uuid",
    "org_id": "uuid",
    "type": "docker",
    "name": "my-docker-host",
    "status": "connected",
    "last_seen_at": "2024-01-01T00:00:00Z",
    "created_at": "2024-01-01T00:00:00Z"
  }
]
```

#### Get Connection

```http
GET /api/v1/orgs/{orgID}/connections/{connectionID}
Authorization: Bearer <token>
```

**Response (200):**
```json
{
  "id": "uuid",
  "org_id": "uuid",
  "type": "docker",
  "name": "my-docker-host",
  "status": "connected",
  "last_seen_at": "2024-01-01T00:00:00Z",
  "created_at": "2024-01-01T00:00:00Z"
}
```

#### List Hosts

```http
GET /api/v1/orgs/{orgID}/connections/{connectionID}/hosts
Authorization: Bearer <token>
```

**Response (200):**
```json
[
  {
    "id": "uuid",
    "connection_id": "uuid",
    "hostname": "my-server",
    "os": "linux",
    "kernel": "amd64",
    "cpu_cores": 8,
    "mem_total": 17179869184,
    "agent_version": "0.1.0",
    "created_at": "2024-01-01T00:00:00Z"
  }
]
```

### Topology

#### Get Topology

```http
GET /api/v1/orgs/{orgID}/connections/{connectionID}/topology
Authorization: Bearer <token>
```

**Response (200):**
```json
{
  "containers": [
    {
      "id": "uuid",
      "connection_id": "uuid",
      "runtime_id": "abc123...",
      "name": "my-container",
      "image": "nginx:latest",
      "state": "running",
      "labels": {},
      "ports": [
        {
          "host_port": "8080",
          "container_port": "80",
          "protocol": "tcp"
        }
      ],
      "created_at": "2024-01-01T00:00:00Z",
      "updated_at": "2024-01-01T00:00:00Z"
    }
  ],
  "networks": [
    {
      "id": "uuid",
      "connection_id": "uuid",
      "name": "bridge",
      "driver": "bridge",
      "scope": "local",
      "subnet": "172.17.0.0/16",
      "created_at": "2024-01-01T00:00:00Z"
    }
  ],
  "edges": [
    {
      "id": "uuid",
      "connection_id": "uuid",
      "src_container_id": "uuid",
      "dst_container_id": "uuid",
      "dst_ip": "10.0.0.5",
      "dst_port": 5432,
      "protocol": "tcp",
      "first_seen": "2024-01-01T00:00:00Z",
      "last_seen": "2024-01-01T00:00:00Z"
    }
  ]
}
```

### Metrics

#### Get Container Metrics (Range)

```http
GET /api/v1/orgs/{orgID}/connections/{connectionID}/metrics?runtime_id={id}&metric=cpu&start=2024-01-01T00:00:00Z&end=2024-01-02T00:00:00Z
Authorization: Bearer <token>
```

**Query Parameters:**
- `runtime_id` (required): Container runtime ID
- `metric` (optional): cpu, mem, net_rx, net_tx, disk_r, disk_w
- `start` (optional): RFC3339 timestamp
- `end` (optional): RFC3339 timestamp

**Response (200):**
```json
{
  "metric": "cpu",
  "results": [
    {
      "timestamp": "2024-01-01T00:00:00Z",
      "value": 45.2
    }
  ],
  "start": "2024-01-01T00:00:00Z",
  "end": "2024-01-02T00:00:00Z",
  "step": 15000000000
}
```

#### Get Container Metrics (Instant)

```http
GET /api/v1/orgs/{orgID}/connections/{connectionID}/metrics/instant?runtime_id={id}
Authorization: Bearer <token>
```

**Response (200):**
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

### Flows

#### List Flows

```http
GET /api/v1/orgs/{orgID}/connections/{connectionID}/flows?limit=500
Authorization: Bearer <token>
```

**Query Parameters:**
- `start` (optional): RFC3339 timestamp
- `end` (optional): RFC3339 timestamp
- `limit` (optional): Max results (default: 1000)

**Response (200):**
```json
{
  "flows": [
    {
      "timestamp": "2024-01-01T00:00:00Z",
      "connection_id": "uuid",
      "src_ip": "10.0.0.1",
      "dst_ip": "10.0.0.5",
      "src_port": 45678,
      "dst_port": 5432,
      "protocol": "tcp",
      "bytes": 1024,
      "packets": 10,
      "latency_ms": 1.5
    }
  ],
  "start": "2024-01-01T00:00:00Z",
  "end": "2024-01-02T00:00:00Z",
  "limit": 500
}
```

#### Get Bandwidth

```http
GET /api/v1/orgs/{orgID}/connections/{connectionID}/bandwidth?duration=5m
Authorization: Bearer <token>
```

**Query Parameters:**
- `duration` (optional): Duration string (default: 5m)

**Response (200):**
```json
{
  "bandwidth": {
    "10.0.0.1:10.0.0.5:5432:tcp": 1048576
  },
  "duration": "5m0s"
}
```

### Vulnerabilities

#### List Scans

```http
GET /api/v1/orgs/{orgID}/connections/{connectionID}/vulns
Authorization: Bearer <token>
```

**Response (200):**
```json
[
  {
    "id": "uuid",
    "connection_id": "uuid",
    "image": "nginx:latest",
    "scan_time": "2024-01-01T00:00:00Z",
    "critical_count": 2,
    "high_count": 5,
    "medium_count": 10,
    "low_count": 15,
    "total_count": 32,
    "created_at": "2024-01-01T00:00:00Z"
  }
]
```

#### Record Scan

```http
POST /api/v1/orgs/{orgID}/connections/{connectionID}/vulns
Authorization: Bearer <token>
Content-Type: application/json

{
  "image": "nginx:latest",
  "critical_count": 2,
  "high_count": 5,
  "medium_count": 10,
  "low_count": 15,
  "total_count": 32,
  "vulnerabilities": [
    {
      "vuln_id": "CVE-2024-1234",
      "severity": "CRITICAL",
      "package": "openssl",
      "version": "1.1.1",
      "fixed_in": "1.1.2",
      "title": "OpenSSL vulnerability",
      "description": "..."
    }
  ]
}
```

**Response (201):**
```json
{
  "id": "uuid",
  "connection_id": "uuid",
  "image": "nginx:latest",
  "scan_time": "2024-01-01T00:00:00Z",
  "critical_count": 2,
  "high_count": 5,
  "medium_count": 10,
  "low_count": 15,
  "total_count": 32,
  "created_at": "2024-01-01T00:00:00Z"
}
```

#### Get Scan Vulnerabilities

```http
GET /api/v1/orgs/{orgID}/connections/{connectionID}/vulns/{scanID}
Authorization: Bearer <token>
```

**Response (200):**
```json
[
  {
    "id": "uuid",
    "scan_id": "uuid",
    "vuln_id": "CVE-2024-1234",
    "severity": "CRITICAL",
    "package": "openssl",
    "version": "1.1.1",
    "fixed_in": "1.1.2",
    "title": "OpenSSL vulnerability",
    "description": "...",
    "created_at": "2024-01-01T00:00:00Z"
  }
]
```

#### Get Vulnerability Dashboard

```http
GET /api/v1/orgs/{orgID}/connections/{connectionID}/vulns/dashboard
Authorization: Bearer <token>
```

**Response (200):**
```json
{
  "total_scans": 5,
  "critical": 10,
  "high": 25,
  "medium": 50,
  "low": 75,
  "affected_images": {
    "nginx:latest": 5,
    "redis:alpine": 3
  }
}
```

### Alerts

#### List Alerts

```http
GET /api/v1/orgs/{orgID}/connections/{connectionID}/alerts
Authorization: Bearer <token>
```

**Response (200):**
```json
[
  {
    "id": "uuid",
    "rule_id": "uuid",
    "rule_name": "High CPU",
    "severity": "critical",
    "status": "firing",
    "labels": {},
    "annotations": {},
    "starts_at": "2024-01-01T00:00:00Z",
    "ends_at": null,
    "fired_count": 5,
    "last_sent_at": "2024-01-01T00:00:00Z"
  }
]
```

#### List Firing Alerts

```http
GET /api/v1/orgs/{orgID}/connections/{connectionID}/alerts/firing
Authorization: Bearer <token>
```

#### List Alert Rules

```http
GET /api/v1/orgs/{orgID}/connections/{connectionID}/alerts/rules
Authorization: Bearer <token>
```

**Response (200):**
```json
[
  {
    "id": "uuid",
    "name": "High CPU",
    "description": "CPU usage above 90%",
    "severity": "critical",
    "condition": {
      "type": "metric",
      "metric": "cpu",
      "threshold": 90,
      "operator": ">",
      "duration": "5m"
    },
    "channels": ["webhook-1"],
    "enabled": true
  }
]
```

#### Create Alert Rule

```http
POST /api/v1/orgs/{orgID}/connections/{connectionID}/alerts/rules
Authorization: Bearer <token>
Content-Type: application/json

{
  "id": "high-cpu",
  "name": "High CPU",
  "description": "CPU usage above 90%",
  "severity": "critical",
  "condition": {
    "type": "metric",
    "metric": "cpu",
    "threshold": 90,
    "operator": ">",
    "duration": "5m"
  },
  "channels": ["webhook-1"],
  "enabled": true
}
```

#### List Notification Channels

```http
GET /api/v1/orgs/{orgID}/connections/{connectionID}/alerts/channels
Authorization: Bearer <token>
```

**Response (200):**
```json
[
  {
    "id": "webhook-1",
    "name": "Slack Webhook",
    "type": "slack",
    "enabled": true
  }
]
```

#### Create Notification Channel

```http
POST /api/v1/orgs/{orgID}/connections/{connectionID}/alerts/channels
Authorization: Bearer <token>
Content-Type: application/json

{
  "id": "webhook-1",
  "name": "Slack Webhook",
  "type": "slack",
  "config": {
    "slack_url": "https://hooks.slack.com/services/xxx"
  },
  "enabled": true
}
```

#### Silence Alert

```http
POST /api/v1/orgs/{orgID}/connections/{connectionID}/alerts/silence
Authorization: Bearer <token>
Content-Type: application/json

{
  "alert_id": "uuid"
}
```

**Response (200):**
```json
{
  "status": "silenced"
}
```

### Agent

#### Agent Enrollment

```http
POST /api/v1/agent/enroll
Content-Type: application/json

{
  "enrollment_token": "uuid",
  "hostname": "my-server",
  "os": "linux",
  "kernel": "amd64",
  "cpu_cores": 8,
  "mem_total": 17179869184,
  "agent_version": "0.1.0"
}
```

**Response (200):**
```json
{
  "connection_id": "uuid",
  "status": "connected"
}
```

#### Agent Heartbeat

```http
POST /api/v1/agent/heartbeat
Content-Type: application/json

{
  "connection_id": "uuid"
}
```

**Response (200):**
```json
{
  "status": "ok"
}
```

#### Sync Topology

```http
POST /api/v1/agent/topology
Content-Type: application/json

{
  "connection_id": "uuid",
  "containers": [
    {
      "runtime_id": "abc123",
      "name": "my-container",
      "image": "nginx:latest",
      "state": "running",
      "labels": {}
    }
  ],
  "networks": [
    {
      "name": "bridge",
      "driver": "bridge",
      "subnet": "172.17.0.0/16"
    }
  ]
}
```

**Response (200):**
```json
{
  "status": "ok"
}
```

### System

#### Health Check

```http
GET /healthz
```

**Response (200):**
```json
{
  "status": "ok",
  "db": "ok"
}
```

#### Version

```http
GET /version
```

**Response (200):**
```json
{
  "version": "0.1.0"
}
```

## WebSocket

### Endpoint

```
ws://localhost:8080/ws/orgs/{orgID}/connections/{connectionID}
```

### Message Types

#### topology_update

Full topology changed:

```json
{
  "type": "topology_update",
  "payload": {
    "connection_id": "uuid",
    "containers": 10,
    "networks": 3
  }
}
```

#### container_add

New container started:

```json
{
  "type": "container_add",
  "payload": {
    "runtime_id": "abc123",
    "name": "my-container",
    "image": "nginx:latest",
    "state": "running"
  }
}
```

#### container_del

Container removed:

```json
{
  "type": "container_del",
  "payload": {
    "runtime_id": "abc123"
  }
}
```

#### container_update

Container state changed:

```json
{
  "type": "container_update",
  "payload": {
    "runtime_id": "abc123",
    "name": "my-container",
    "image": "nginx:latest",
    "state": "stopped"
  }
}
```

#### status_change

Connection status changed:

```json
{
  "type": "status_change",
  "payload": {
    "connection_id": "uuid",
    "status": "connected"
  }
}
```

## GraphQL API

### Endpoint

```
POST http://localhost:8080/graphql
```

### Example Queries

#### Get Topology

```graphql
query {
  topology(orgId: "uuid", connectionId: "uuid") {
    containers {
      id
      name
      image
      state
    }
    edges {
      srcContainerId
      dstContainerId
      protocol
    }
  }
}
```

#### Get Container Metrics

```graphql
query {
  metrics(
    orgId: "uuid"
    connectionId: "uuid"
    runtimeId: "abc123"
    metric: "cpu"
  ) {
    timestamp
    value
  }
}
```

### Example Mutations

#### Create Connection

```graphql
mutation {
  createConnection(orgId: "uuid", type: "docker", name: "my-host") {
    id
    name
    status
  }
}
```

## Error Responses

All error responses follow this format:

```json
{
  "error": "error message"
}
```

### HTTP Status Codes

| Code | Description |
|------|-------------|
| 200 | Success |
| 201 | Created |
| 400 | Bad Request |
| 401 | Unauthorized |
| 403 | Forbidden |
| 404 | Not Found |
| 409 | Conflict |
| 429 | Too Many Requests |
| 500 | Internal Server Error |

## Rate Limiting

- 100 requests per minute per API key
- 1000 requests per minute per user
- WebSocket connections: 10 per user

## Pagination

List endpoints support pagination via query parameters:

```http
GET /api/v1/orgs?limit=20&offset=0
```

- `limit`: Number of results (default: 50, max: 100)
- `offset`: Number of results to skip

## Filtering

Some endpoints support filtering:

```http
GET /api/v1/orgs/{orgID}/connections?status=connected
GET /api/v1/orgs/{orgID}/connections/{id}/flows?protocol=tcp
```

## Sorting

Some endpoints support sorting:

```http
GET /api/v1/orgs?sort=name&order=asc
```

- `sort`: Field to sort by
- `order`: asc or desc (default: asc)
