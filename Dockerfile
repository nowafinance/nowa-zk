# --- Stage 1: Builder ---
FROM golang:1.24 AS builder

WORKDIR /app

# Copy dependency files first for caching
COPY sequencer/go.mod sequencer/go.sum ./sequencer/
COPY prover/go.mod prover/go.sum ./prover/

# Download dependencies
WORKDIR /app/sequencer
RUN go mod download
WORKDIR /app/prover
RUN go mod download

# Copy source code
WORKDIR /app
COPY sequencer ./sequencer
COPY prover ./prover

# Build Sequencer
WORKDIR /app/sequencer
RUN CGO_ENABLED=1 GOOS=linux go build -o /app/bin/sequencer ./cmd/sequencer

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
COPY --from=builder /app/bin/sequencer /usr/local/bin/sequencer
COPY --from=builder /app/bin/prover /usr/local/bin/prover

# Expose ports
# Sequencer API & WebSocket
EXPOSE 8080 8546
# Prover RPC (if applicable) or metrics
EXPOSE 9000

# Default entrypoint (can be overridden)
CMD ["sequencer", "start"]
