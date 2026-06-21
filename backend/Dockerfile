# Build stage
FROM golang:1-alpine AS builder

RUN apk add --no-cache git ca-certificates

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY . .

RUN go mod tidy && CGO_ENABLED=0 GOOS=linux go build -ldflags="-w -s" -o /containerscope ./cmd/server

# Copy migrations
COPY migrations/ /app/migrations/

# Runtime stage
FROM alpine:3.19

RUN apk add --no-cache ca-certificates tzdata

RUN adduser -D -u 1000 containerscope

WORKDIR /app

COPY --from=builder /containerscope /usr/local/bin/containerscope
COPY --from=builder /app/migrations/ /app/migrations/

EXPOSE 8080 8081

HEALTHCHECK --interval=30s --timeout=3s --start-period=5s --retries=3 \
  CMD wget -qO- http://localhost:8080/healthz || exit 1

ENTRYPOINT ["containerscope"]
