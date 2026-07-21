# Indexer

ZK Indexer monitors blockchain, builds transaction batches, provides API for prover.

## Quick Start

```bash
# Build
go build -o indexer-bin ./cmd/indexer

# Start indexer
export RPC_URL=http://localhost:8545
./indexer-bin start

# Clear all data
rm -rf ./data/
```

## Configuration

Environment variables:

```bash
RPC_URL=http://localhost:8545      # Ethereum RPC endpoint (REQUIRED)
WS_URL=ws://localhost:8546          # WebSocket endpoint (optional)
API_PORT=8080                       # API server port (default: 8080)
BATCH_SIZE=100                      # Transactions per batch (default: 100)
STATE_DB_PATH=./data/state          # Database path (default: ./data/state)
```

## API Endpoints

### REST API

```bash
GET /health                 # Health check
GET /status                 # Service status
GET /batches/latest         # Latest batch
GET /batches/:id            # Get batch by number
GET /batches                # All batches
GET /prover/batch/latest    # Latest batch for prover
GET /prover/batch/:id       # Batch for prover by number
```

### WebSocket

```bash
WS /ws   # Real-time batch notifications
```

### Examples

```bash
curl http://localhost:8080/health
curl http://localhost:8080/batches/latest
curl http://localhost:8080/prover/batch/1
```

## How It Works

1. **Monitors** blockchain for new blocks
2. **Processes** transactions from each block
3. **Builds** batches incrementally (128 txs per batch)
4. **Computes** state roots using Sparse Merkle Tree  
5. **Persists** batches to BadgerDB
6. **Serves** batches via API for prover
7. **Handles** chain reorganizations automatically

## Features

- ✅ Incremental batch filling
- ✅ BadgerDB persistence
- ✅ SMT state root calculation
- ✅ Reorg handling with rollback
- ✅ Balance validation
- ✅ Thread-safe RPC client
- ✅ REST + WebSocket API

## Testing

```bash
# Run all tests
go test ./...

# With race detection
go test -race ./...

# With coverage
go test ./... -coverprofile=coverage.out
```

## Local Development

```bash
# Terminal 1: Start local chain
anvil --port 8545

# Terminal 2: Start indexer
export RPC_URL=http://localhost:8545
./indexer-bin start
```

See [/CODEME.md](../CODEME.md) for full commands.
