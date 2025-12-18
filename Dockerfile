# Port Forward Manager - Docker Image
# Multi-stage build for minimal image size

# ============ Build Stage ============
FROM golang:1.24-alpine AS builder

# Install build dependencies
RUN apk add --no-cache git ca-certificates

WORKDIR /app

# Copy go mod files first for better caching
COPY go.mod go.sum ./
RUN go mod download

# Copy source code
COPY . .

# Build the CLI-only binary (no GUI dependencies)
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -tags nogui \
    -ldflags="-w -s" \
    -o pfm .

# ============ Runtime Stage ============
FROM alpine:3.19

# Install ca-certificates for HTTPS connections
RUN apk add --no-cache ca-certificates tzdata

# Create non-root user (optional, comment out if you need to bind privileged ports)
# RUN adduser -D -H -s /sbin/nologin pfm
# USER pfm

WORKDIR /app

# Copy binary from builder
COPY --from=builder /app/pfm /usr/local/bin/pfm

# Create data directory for persistent storage
RUN mkdir -p /data

# Environment variables
ENV HOME=/data
ENV PFM_DATA_DIR=/data

# Expose common ports (adjust as needed)
# These are just examples - actual ports depend on your rules
EXPOSE 8080 9000 10000-10100

# Health check
HEALTHCHECK --interval=30s --timeout=5s --start-period=5s --retries=3 \
    CMD pfm status || exit 1

# Volume for persistent data
VOLUME ["/data"]

# Default command: run as service
ENTRYPOINT ["pfm"]
CMD ["service", "run"]
