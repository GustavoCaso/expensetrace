# syntax=docker/dockerfile:1.4

# Build stage
FROM golang:1.25.0-alpine3.21 AS builder

WORKDIR /app

# Install build dependencies
RUN apk add --no-cache build-base sqlite-dev

# Copy dependency files first for better layer caching
COPY go.mod go.sum ./

# Download dependencies with cache mounts for faster rebuilds
RUN --mount=type=cache,target=/go/pkg/mod \
    --mount=type=cache,target=/root/.cache/go-build \
    go mod download

# Copy source code
COPY . .

# Build with cache mounts and optimizations
RUN --mount=type=cache,target=/go/pkg/mod \
    --mount=type=cache,target=/root/.cache/go-build \
    CGO_ENABLED=1 GOOS=linux go build \
    -ldflags="-s -w" \
    -trimpath \
    -o expensetrace /app/cmd/

# Final stage
FROM alpine:3.21.0

# Install runtime dependencies and shadow package (for usermod/groupmod) and su-exec (for privilege dropping)
RUN apk add --no-cache ca-certificates tzdata sqlite-libs shadow su-exec && \
    rm -rf /var/cache/apk/* && \
    addgroup -g 1000 expensetrace && \
    adduser -u 1000 -G expensetrace -s /bin/sh -D expensetrace && \
    mkdir -p /app /data && \
    chown -R expensetrace:expensetrace /app /data

# Copy binary and entrypoint script from builder
COPY --from=builder --chown=root:root /app/expensetrace /app/
COPY --chmod=755 scripts/entrypoint.sh /usr/local/bin/

# Environment variables with defaults pointing to /data directory
ENV EXPENSETRACE_DB=/data/expensetrace.db \
    EXPENSETRACE_PORT=8080 \
    EXPENSETRACE_LOG_LEVEL=info \
    EXPENSETRACE_LOG_FORMAT=text \
    EXPENSETRACE_LOG_OUTPUT=stdout \
    EXPENSETRACE_ALLOW_EMBEDDING=false

# Expose the default port
EXPOSE 8080

# Set working directory to data directory
WORKDIR /data

# Use entrypoint script to handle PUID/PGID and drop privileges
ENTRYPOINT ["/usr/local/bin/entrypoint.sh"]
