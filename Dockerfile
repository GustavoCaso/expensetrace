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

# Install runtime dependencies and create non-root user
RUN apk add --no-cache ca-certificates tzdata sqlite-libs && \
    rm -rf /var/cache/apk/* && \
    addgroup -g 1234 expensetrace && \
    adduser -u 1234 -G expensetrace -s /bin/sh -D expensetrace && \
    mkdir -p /app /data && \
    chown -R expensetrace:expensetrace /app /data

# Copy binary from builder with proper ownership
COPY --from=builder --chown=expensetrace:expensetrace /app/expensetrace /app/

# Environment variables with defaults pointing to /data directory
ENV EXPENSETRACE_DB=/data/expensetrace.db \
    EXPENSETRACE_PORT=8080 \
    EXPENSETRACE_LOG_LEVEL=info \
    EXPENSETRACE_LOG_FORMAT=text \
    EXPENSETRACE_LOG_OUTPUT=stdout \
    EXPENSETRACE_ALLOW_EMBEDDING=false

# Expose the default port
EXPOSE 8080

# Switch to non-root user
USER expensetrace

# Set working directory to data directory
WORKDIR /data

# Run the expensetrace binary directly
ENTRYPOINT ["/app/expensetrace"]
# Default to web mode (can be overridden with: docker run ... expensetrace tui)
CMD ["web"]
