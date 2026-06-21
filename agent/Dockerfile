# Build stage
FROM golang:1-alpine AS builder

RUN apk add --no-cache git ca-certificates

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY . .

RUN go mod tidy && CGO_ENABLED=0 GOOS=linux go build -ldflags="-w -s" -o /containerscope-agent ./cmd/agent

# Runtime stage
FROM alpine:3.19

RUN apk add --no-cache ca-certificates tzdata

RUN adduser -D -u 1000 containerscope

COPY --from=builder /containerscope-agent /usr/local/bin/containerscope-agent

USER root

EXPOSE 9090

HEALTHCHECK --interval=30s --timeout=3s --start-period=5s --retries=3 \
  CMD pgrep containerscope-agent || exit 1

ENTRYPOINT ["containerscope-agent"]
