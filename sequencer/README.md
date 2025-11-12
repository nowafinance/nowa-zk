# ZK Sequencer

Continuously monitors the Tan-ZK blockchain, collects transactions, builds batches incrementally, and provides REST API endpoints for the prover.

## Quick Start

### Prerequisites
- Go 1.24+
- Access to Tan-ZK RPC endpoint (or use Anvil for local testing)

### Installation

```bash
cd sequencer
go mod download
cp .env.example .env
# Edit .env with your RPC endpoint
go build ./cmd/sequencer
./sequencer start
```

### Configuration

Create `.env` file:
```bash
TAN_ZK_RPC_URL=http://localhost:8545    # Required
TAN_ZK_WS_URL=ws://localhost:8546       # Optional
BATCH_SIZE=100                          # Transactions per batch
API_PORT=8080                           # REST API port
STATE_DB_PATH=./data                    # BadgerDB storage path
```

### Commands

```bash
# Start sequencer (resumes from last processed block)
./sequencer start

# Start with custom options
./sequencer start --rpc-url http://localhost:8545 --batch-size 200

# Start and clear all data (reset to block 0)
./sequencer start --reset

# Start with reset and custom options
./sequencer start --reset --rpc-url http://localhost:8545 --batch-size 200

# Help
./sequencer --help
./sequencer start --help
```

## How It Works

1. **Block Monitoring**: Subscribes to new blocks via WebSocket (or polls via HTTP)
2. **Transaction Processing**: Fetches transactions, enriches with contract addresses, queries balances
3. **State Root**: Updates Sparse Merkle Tree (SMT) with balance changes, computes state root
4. **Batch Building**: Creates batches incrementally (no pooling), appends to incomplete batches
5. **Reorg Handling**: Detects forks by comparing block hashes, rolls back and re-processes
6. **Storage**: Persists batches in BadgerDB with resume support

## Features

- ✅ Incremental batch creation (no transaction pooling)
- ✅ BadgerDB persistence with resume support
- ✅ Sparse Merkle Tree (SMT) for state root calculation
- ✅ Blockchain reorg handling (fork detection and rollback)
- ✅ REST API and WebSocket endpoints
- ✅ Structured logging and error handling
- ✅ Cobra CLI with start command and --reset flag

## API Endpoints

### REST API

- `GET /health` - Health check
- `GET /status` - Sequencer status
- `GET /batch/latest` - Latest batch
- `GET /batch/{number}` - Get batch by number
- `GET /batches` - All batches
- `GET /prover/batch/latest` - Latest batch for prover (with traces)
- `GET /prover/batch/{number}` - Batch for prover by number

### WebSocket

- `WS /ws/batches` - Real-time batch notifications

**Example:**
```bash
curl http://localhost:8080/health
curl http://localhost:8080/batch/latest
```

## Architecture

```
sequencer/
├── cmd/sequencer/          # CLI (Cobra)
├── pkg/
│   ├── errors/            # Structured errors
│   ├── logger/            # Structured logging
│   ├── config/            # Configuration
│   ├── smt/               # Sparse Merkle Tree
│   └── rpc/               # RPC client
└── internal/sequencer/    # Core logic
    ├── sequencer.go       # Main service
    ├── batch.go           # Batch builder
    ├── store.go           # BadgerDB storage
    └── api.go             # REST API & WebSocket
```

## Local Development

**Using Anvil:**
```bash
# Terminal 1: Start Anvil
anvil --port 8545

# Terminal 2: Start Sequencer
cd sequencer
./sequencer start --rpc-url http://localhost:8545
```

## Troubleshooting

- **Won't start**: Check `TAN_ZK_RPC_URL` in `.env`
- **No batches**: Ensure blocks contain transactions
- **Database errors**: Check `STATE_DB_PATH` is writable
- **WebSocket issues**: Sequencer falls back to HTTP polling

## Status

- ✅ Block monitoring and transaction collection
- ✅ Incremental batch creation
- ✅ BadgerDB persistence
- ✅ SMT state root calculation
- ✅ Reorg handling
- 🔄 Proof submission (in progress)
