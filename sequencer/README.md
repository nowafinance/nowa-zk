# ZK Sequencer

The sequencer service for the Tan-ZK network. It continuously monitors the Tan-ZK blockchain, collects transactions, builds batches, and provides REST API endpoints for the prover to fetch batch data for proof generation.

## What It Does

The ZK Sequencer is a continuously running service that:

- **Connects to Tan-ZK Node**: Establishes JSON-RPC and WebSocket connections to Tan-ZK blockchain nodes
- **Monitors Blocks**: Subscribes to new blocks via WebSocket or polls periodically
- **Collects Transactions**: Fetches transactions from new blocks and adds them to the transaction pool
- **Builds Batches**: Periodically creates batches of transactions (default: 100 transactions per batch, every 10 seconds)
- **Stores Batches**: Saves batches locally with execution traces for the prover
- **REST API**: Provides HTTP endpoints for the prover to fetch batch data

## Quick Start

### Prerequisites

- **Go 1.22 or later** - [Install Go](https://go.dev/doc/install)
- **Access to a Tan-ZK RPC endpoint** (or use Anvil for local testing)

### Installation

```bash
# Clone the repository
git clone https://github.com/tannetwork/zk-sequencer.git
cd zk-sequencer/sequencer

# Install dependencies
go mod download
```

### Configuration

1. **Copy the environment template**:
```bash
cp .env.example .env
```

2. **Edit `.env` with your Tan-ZK node settings**:
```bash
# Required: Tan-ZK RPC endpoint (HTTP/HTTPS)
TAN_ZK_RPC_URL=http://localhost:8545

# Optional: WebSocket endpoint for real-time block subscriptions
TAN_ZK_WS_URL=ws://localhost:8546

# Optional: Request timeout (seconds)
RPC_TIMEOUT=30

# Optional: Maximum retry attempts
RPC_MAX_RETRIES=3
```

### Running the Sequencer

**Build and run:**
```bash
# Build the binary
go build ./cmd/sequencer

# Run it
./sequencer
```

**Or run directly:**
```bash
go run ./cmd/sequencer
```

The sequencer will:
- Connect to your Tan-ZK node
- Start monitoring for new blocks
- Begin building batches from transactions
- Start the REST API server on port 8080

**Stop the sequencer**: Press `Ctrl+C` for graceful shutdown.

## How It Works

### Block Monitoring

The sequencer monitors new blocks in two ways:

1. **WebSocket (Preferred)**: If `TAN_ZK_WS_URL` is configured, it subscribes to `eth_subscribe` for real-time block notifications
2. **HTTP Polling (Fallback)**: If WebSocket is unavailable, it polls `eth_blockNumber` every 2 seconds

### Transaction Collection

When a new block is detected:
1. The sequencer fetches the full block with all transactions
2. Each transaction is added to the in-memory transaction pool
3. The pool maintains up to 10,000 transactions

### Batch Building

Every 10 seconds (configurable), the sequencer:
1. Takes up to 100 transactions from the pool (configurable)
2. Generates execution traces for each transaction
3. Computes state roots (old and new)
4. Creates a batch with metadata
5. Stores the batch locally in `./data/` directory

### Batch Storage

Batches are stored as JSON files in `./data/batch_<number>.json` with:
- Batch number and hash
- All transactions
- Execution traces
- State roots
- Timestamp
- Status (pending, proving, ready, submitted)

## REST API Endpoints

The sequencer exposes REST API endpoints for the prover to fetch batch data:

### Health & Status

**GET `/health`**
- Returns: `{"status": "ok"}`

**GET `/status`**
- Returns: `{"status": "running", "batch_count": 5}`

### Batch Endpoints

**GET `/batch/latest`**
- Returns the latest batch with all transactions and traces

**GET `/batch/{number}`**
- Returns a specific batch by number
- Example: `GET /batch/1`

**GET `/batches`**
- Returns all batches
- Returns: `{"batches": [...], "count": 5}`

### Prover Endpoints

**GET `/prover/batch/latest`**
- Returns the latest batch with execution traces for proof generation

**GET `/prover/batch/{number}`**
- Returns a specific batch with execution traces
- Example: `GET /prover/batch/1`

### Example API Usage

```bash
# Check health
curl http://localhost:8080/health

# Get status
curl http://localhost:8080/status

# Get latest batch
curl http://localhost:8080/batch/latest

# Get batch #1
curl http://localhost:8080/batch/1

# Get latest batch for prover (with traces)
curl http://localhost:8080/prover/batch/latest
```

## Configuration Options

The sequencer can be configured via environment variables:

| Variable | Description | Required | Default |
|----------|-------------|----------|---------|
| `TAN_ZK_RPC_URL` | HTTP/HTTPS RPC endpoint URL | ✅ Yes | - |
| `TAN_ZK_WS_URL` | WebSocket endpoint URL | No | - |
| `RPC_TIMEOUT` | Request timeout in seconds | No | 30 |
| `RPC_MAX_RETRIES` | Maximum retry attempts | No | 3 |
| `RPC_RETRY_BACKOFF` | Initial retry backoff in milliseconds | No | 100 |
| `RPC_MAX_RETRY_BACKOFF` | Maximum retry backoff in seconds | No | 10 |

**Note**: Batch size and interval are currently hardcoded but will be configurable in future versions.

## Project Structure

```
sequencer/
├── cmd/
│   └── sequencer/          # Main application entry point
│       └── main.go
├── pkg/
│   └── rpc/                # RPC client for Tan-ZK blockchain
│       ├── client.go       # HTTP RPC client
│       ├── websocket.go    # WebSocket client
│       ├── blocks.go       # Block operations
│       ├── transactions.go # Transaction operations
│       └── accounts.go     # Account queries
├── internal/
│   └── sequencer/          # Core sequencer logic
│       ├── sequencer.go    # Main service
│       ├── pool.go         # Transaction pool
│       ├── batch.go        # Batch builder
│       ├── store.go        # Batch storage
│       └── api.go          # REST API server
├── .env.example            # Environment configuration template
└── README.md               # This file
```

## Local Development with Anvil

If you don't have a Tan-ZK node, you can test with Anvil (local Ethereum test node):

1. **Start Anvil**:
```bash
anvil --port 8545
```

2. **Update `.env`**:
```bash
TAN_ZK_RPC_URL=http://localhost:8545
```

3. **Start the sequencer**:
```bash
go run ./cmd/sequencer
```

4. **Send test transactions** (in another terminal):
```bash
# Anvil provides test accounts with funds
# You can use cast or send transactions via RPC
```

## What Happens When You Start

When you start the sequencer, you'll see:

```
🚀 Starting ZK Sequencer...
✅ Connected to Tan-ZK RPC endpoint
✅ Connected to Tan-ZK WebSocket endpoint (if configured)
✅ Transaction pool initialized
✅ Batch builder initialized
✅ Batch store initialized
✅ REST API server initialized on port 8080
✅ ZK Sequencer is running!
📡 Listening for new blocks and building batches...
🌐 REST API available at http://localhost:8080
📦 Processing block #12345
✅ Block #12345 processed: 10 transactions added to pool (pool size: 10)
✅ Batch #1 created: 10 transactions (batch hash: 0xabcd...)
```

The sequencer will continue running until you press `Ctrl+C`.

## Integration with Prover

The prover service can fetch batches from the sequencer:

1. **Poll for new batches**: The prover can periodically check `/prover/batch/latest`
2. **Fetch specific batch**: Use `/prover/batch/{number}` to get a batch with execution traces
3. **Generate proof**: Use the batch data and traces to generate ZK proofs
4. **Submit proof**: Once proof is generated, submit it back to the sequencer (future feature)

## Next Steps

- The sequencer is now functional and continuously running
- It monitors blocks, collects transactions, and builds batches
- REST API is available for the prover to fetch batch data
- Future enhancements will include proof submission and on-chain verification
