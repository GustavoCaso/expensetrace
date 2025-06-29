# syntax=docker/dockerfile:1.3-labs

# Build stage
FROM golang:1.23-alpine AS builder

WORKDIR /app
COPY . .
RUN apk add build-base
RUN go mod download
RUN CGO_ENABLED=1 GOOS=linux go build -o expensetrace .

# Final stage
FROM alpine:latest
WORKDIR /app
COPY --from=builder /app/expensetrace .
COPY start.sh .
RUN chmod +x /app/start.sh

ARG EXPENSETRACE_DB=expensetrace.db
ARG EXPENSETRACE_CONFIG=expense.yml
ARG EXPENSETRACE_PORT=8080
ARG EXPENSETRACE_SUBCOMMAND=web
ARG EXPENSETRACE_ALLOW_EMBEDDING="false"

# Install SQLite dependencies
RUN apk add --no-cache sqlite

# Environment variables with defaults
ENV EXPENSETRACE_DB=${EXPENSETRACE_DB} \
    EXPENSETRACE_CONFIG=${EXPENSETRACE_CONFIG} \
    EXPENSETRACE_PORT=${EXPENSETRACE_PORT} \
    EXPENSETRACE_SUBCOMMAND=${EXPENSETRACE_SUBCOMMAND} \
    EXPENSETRACE_ALLOW_EMBEDDING=${EXPENSETRACE_ALLOW_EMBEDDING}

# Use the startup script as entrypoint
ENTRYPOINT ["/app/start.sh"]
