.PHONY: dev dev-infra dev-backend dev-frontend test lint build clean help

# Default target
help:
	@echo "ContainerScope - DevOps Observability Platform"
	@echo ""
	@echo "Usage:"
	@echo "  make dev           Start all infrastructure + backend + frontend"
	@echo "  make dev-infra     Start infrastructure services only"
	@echo "  make dev-backend   Start backend server"
	@echo "  make dev-frontend  Start frontend dev server"
	@echo "  make test          Run all tests"
	@echo "  make lint          Run all linters"
	@echo "  make build         Build all binaries"
	@echo "  make clean         Stop and remove containers"

# Infrastructure
dev-infra:
	docker compose up -d

dev-infra-down:
	docker compose down

# Development
dev: dev-infra
	@echo "Infrastructure started. Run 'make dev-backend' and 'make dev-frontend' in separate terminals."

dev-backend:
	cd backend && go run ./cmd/server

dev-frontend:
	cd frontend && npm run dev

# Testing
test: test-backend test-frontend

test-backend:
	cd backend && go test ./...

test-frontend:
	cd frontend && npm run test

# Linting
lint: lint-backend lint-frontend

lint-backend:
	cd backend && go vet ./...

lint-frontend:
	cd frontend && npm run lint

# Type checking
typecheck: typecheck-frontend

typecheck-frontend:
	cd frontend && npm run typecheck

# Building
build: build-backend build-agent build-frontend

build-backend:
	cd backend && go build -o ../bin/containerscope-backend ./cmd/server

build-agent:
	cd agent && go build -o ../bin/containerscope-agent ./cmd/agent

build-frontend:
	cd frontend && npm run build

# Clean
clean:
	docker compose down -v
	rm -rf bin/
	cd frontend && rm -rf dist/ node_modules/
