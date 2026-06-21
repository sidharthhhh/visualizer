#!/bin/bash

# ===========================================
# ContainerScope Production Deployment
# ===========================================

set -e

echo "=========================================="
echo "  ContainerScope Production Deployment"
echo "=========================================="
echo ""

# Check if Docker is installed
if ! command -v docker &> /dev/null; then
    echo "ERROR: Docker is not installed"
    exit 1
fi

if ! command -v docker &> /dev/null || ! docker compose version &> /dev/null; then
    echo "ERROR: Docker Compose is not installed"
    exit 1
fi

# Check if .env file exists
if [ ! -f .env ]; then
    echo "Creating .env file from template..."
    cp .env.example .env 2>/dev/null || cat > .env << 'EOF'
# Database Passwords (CHANGE THESE!)
POSTGRES_PASSWORD=SecureP@ss2024!
CLICKHOUSE_PASSWORD=SecureP@ss2024!
MINIO_ROOT_PASSWORD=SecureP@ss2024!

# JWT Secret (CHANGE THIS! Min 32 characters)
JWT_SECRET=ProductionSecretKey2024!MustBe32CharsLong!
EOF
    echo "WARNING: Please edit .env and change the default passwords!"
    echo ""
fi

# Stop existing containers
echo "Stopping existing containers..."
docker compose down 2>/dev/null || true

# Build and start
echo ""
echo "Building and starting services..."
docker compose up -d --build

# Wait for services to be healthy
echo ""
echo "Waiting for services to start..."
sleep 30

# Check health
echo ""
echo "Checking service health..."

# Backend
if curl -s http://localhost:8080/healthz | grep -q '"status":"ok"'; then
    echo "✓ Backend: Healthy"
else
    echo "✗ Backend: Starting..."
fi

# Frontend
if curl -s http://localhost:3000/health | grep -q "ok"; then
    echo "✓ Frontend: Healthy"
else
    echo "✗ Frontend: Starting..."
fi

# Postgres
if docker exec containerscope-postgres pg_isready -U containerscope > /dev/null 2>&1; then
    echo "✓ Postgres: Healthy"
else
    echo "✗ Postgres: Starting..."
fi

echo ""
echo "=========================================="
echo "  Deployment Complete!"
echo "=========================================="
echo ""
echo "Services:"
echo "  Frontend:  http://localhost:3000"
echo "  Backend:   http://localhost:8080"
echo "  gRPC:      http://localhost:8081"
echo "  MinIO:     http://localhost:9001"
echo ""
echo "Default Login:"
echo "  Email:     admin@containerscope.io"
echo "  Password:  admin123"
echo ""
echo "For Apache reverse proxy, see: docs/apache-config.md"
echo ""
