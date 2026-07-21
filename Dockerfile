# --- Stage 1: Builder ---
FROM golang:1.24 AS builder

WORKDIR /app

# Copy dependency files first for caching
COPY indexer/go.mod indexer/go.sum ./indexer/
COPY prover/go.mod prover/go.sum ./prover/

# Download dependencies
WORKDIR /app/indexer
RUN go mod download
WORKDIR /app/prover
RUN go mod download

# Copy source code
WORKDIR /app
COPY indexer ./indexer
COPY prover ./prover

# Build Indexer
WORKDIR /app/indexer
RUN CGO_ENABLED=1 GOOS=linux go build -o /app/bin/indexer ./cmd/indexer

# Build Prover
WORKDIR /app/prover
RUN CGO_ENABLED=1 GOOS=linux go build -o /app/bin/prover ./cmd/prover

# --- Stage 2: Runtime ---
FROM ubuntu:22.04

# Install basic runtime dependencies
RUN apt-get update && apt-get install -y \
    ca-certificates \
    curl \
    libc6 \
    && rm -rf /var/lib/apt/lists/*

WORKDIR /app

# Create data directories
RUN mkdir -p /app/data /app/keys /app/config

# Copy binaries from builder
COPY --from=builder /app/bin/indexer /usr/local/bin/indexer
COPY --from=builder /app/bin/prover /usr/local/bin/prover

# Expose ports
# Indexer API & WebSocket
EXPOSE 8080 8546
# Prover RPC (if applicable) or metrics
EXPOSE 9000

# Default entrypoint (can be overridden)
CMD ["indexer", "start"]
