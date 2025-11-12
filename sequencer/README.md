# ZK Sequencer

The sequencer service for the Tan-ZK network. It continuously monitors the Tan-ZK blockchain, collects transactions, builds batches incrementally, and provides REST API endpoints for the prover to fetch batch data for proof generation.

## What It Does

The ZK Sequencer is a continuously running service that:

- **Connects to Tan-ZK Node**: Establishes JSON-RPC and WebSocket connections to Tan-ZK blockchain nodes
- **Monitors Blocks**: Subscribes to new blocks via WebSocket or polls periodically
- **Collects Transactions**: Fetches transactions from new blocks and processes them immediately
- **Builds Batches Incrementally**: Creates batches immediately from transactions (no pooling), appending to incomplete batches until full
- **Stores Batches**: Persists batches in BadgerDB with execution traces for the prover
- **REST API**: Provides HTTP endpoints and WebSocket for the prover to fetch batch data
- **Graceful Shutdown**: Handles interruptions cleanly and resumes from last processed block

## Quick Start

### Prerequisites

- **Go 1.24 or later** - [Install Go](https://go.dev/doc/install)
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

# Sequencer Configuration
BATCH_SIZE=100              # Transactions per batch
BATCH_INTERVAL=5            # Batch check interval (seconds)
API_PORT=8080               # REST API server port
STATE_DB_PATH=./data        # Path to BadgerDB storage

# Optional: RPC Client Configuration
RPC_TIMEOUT=30              # Request timeout (seconds)
RPC_MAX_RETRIES=3           # Maximum retry attempts
RPC_RETRY_BACKOFF=1000      # Initial retry backoff (milliseconds)
RPC_MAX_RETRY_BACKOFF=5     # Maximum retry backoff (seconds)
```

### Running the Sequencer

**Build the binary:**
```bash
go build ./cmd/sequencer
```

**Start the sequencer:**
```bash
# Start with defaults from .env
./sequencer start

# Start with custom RPC URL
./sequencer start --rpc-url http://localhost:8545

# Start with custom database path
./sequencer start --state-db-path /custom/path/data

# Start with all custom options
./sequencer start \
  --rpc-url http://localhost:8545 \
  --ws-url ws://localhost:8546 \
  --batch-size 200 \
  --api-port 9090 \
  --state-db-path ./custom-data
```

**Restart the sequencer:**
```bash
# Restart with defaults
./sequencer restart

# Restart with custom options
./sequencer restart --rpc-url http://localhost:8545 --batch-size 150
```

**View help:**
```bash
./sequencer --help
./sequencer start --help
./sequencer restart --help
```

**Stop the sequencer**: Press `Ctrl+C` for graceful shutdown.

## How It Works

### Block Monitoring

The sequencer monitors new blocks in two ways:

1. **WebSocket (Preferred)**: If `TAN_ZK_WS_URL` is configured, it subscribes to `eth_subscribe` for real-time block notifications
2. **HTTP Polling (Fallback)**: If WebSocket is unavailable, it polls `eth_blockNumber` every 2 seconds

### Transaction Processing

When a new block is detected:
1. The sequencer fetches the full block with all transactions
2. Transactions are enriched with contract addresses for deployments
3. Transactions are immediately processed into batches (no pooling)

### Incremental Batch Building

The sequencer uses **incremental batch creation**:

1. **Check for incomplete batch**: If a batch exists with fewer than `BATCH_SIZE` transactions, new transactions are appended to it
2. **Create new batch**: If no incomplete batch exists, a new batch is created immediately (even with just 1 transaction)
3. **Complete batches**: When a batch reaches `BATCH_SIZE` transactions, it's marked complete and a new batch starts
4. **Persist immediately**: All batches (complete or incomplete) are saved to BadgerDB immediately
5. **Resume on restart**: On restart, the sequencer resumes filling incomplete batches from where it left off

**Key Features:**
- ✅ No transaction pooling - batches created immediately
- ✅ No transactions missed - all transactions are persisted
- ✅ Sequential block processing - blocks processed one at a time
- ✅ Resume support - restarts continue from last processed block

### Batch Storage

**Storage Type**: BadgerDB (fast key-value store)

Batches are stored in BadgerDB with:
- Batch number and hash
- All transactions (including contract addresses for deployments)
- Execution traces
- State roots (old and new)
- Timestamp
- Status (pending, proving, ready, submitted)

**Storage Details**:
- **Database**: BadgerDB (embedded key-value store)
- **Format**: JSON-serialized batches in BadgerDB
- **Location**: `./data/db/` directory (created automatically)
- **In-Memory Cache**: Batches are also cached in memory for fast API access
- **Persistence**: Last processed block number is also persisted for resume support

**Example storage structure:**
```
sequencer/
└── data/
    └── db/                    # BadgerDB files
        ├── 000001.vlog
        ├── MANIFEST
        └── ...
```

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

### WebSocket Endpoint

**WS `/ws/batches`**
- Real-time WebSocket connection for batch updates
- Automatically receives notifications when new batches are created
- Supports ping/pong for connection keepalive

**Connection URL**: `ws://localhost:8080/ws/batches`

**Message Types**:
- `welcome`: Initial connection message
- `new_batch`: Notification when a new batch is created or updated
- `ping`/`pong`: Keepalive messages

**Example Usage (JavaScript)**:
```javascript
const ws = new WebSocket('ws://localhost:8080/ws/batches');

ws.onmessage = (event) => {
  const msg = JSON.parse(event.data);
  if (msg.type === 'new_batch') {
    console.log('New batch:', msg.batch);
  }
};
```

**Example Usage (Python)**:
```python
import websocket
import json

def on_message(ws, message):
    msg = json.loads(message)
    if msg['type'] == 'new_batch':
        print('New batch:', msg['batch'])

ws = websocket.WebSocketApp('ws://localhost:8080/ws/batches',
                           on_message=on_message)
ws.run_forever()
```

**Example Usage (Go)**:
```go
import "github.com/gorilla/websocket"

conn, _, err := websocket.DefaultDialer.Dial("ws://localhost:8080/ws/batches", nil)
if err != nil {
    log.Fatal(err)
}
defer conn.Close()

for {
    var msg map[string]interface{}
    if err := conn.ReadJSON(&msg); err != nil {
        break
    }
    if msg["type"] == "new_batch" {
        fmt.Println("New batch:", msg["batch"])
    }
}
```

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

The sequencer can be configured via environment variables (`.env` file) or command-line flags:

### Environment Variables

| Variable | Description | Required | Default |
|----------|-------------|----------|---------|
| `TAN_ZK_RPC_URL` | HTTP/HTTPS RPC endpoint URL | ✅ Yes | - |
| `TAN_ZK_WS_URL` | WebSocket endpoint URL | No | - |
| `BATCH_SIZE` | Transactions per batch | No | 100 |
| `BATCH_INTERVAL` | Batch check interval (seconds) | No | 5 |
| `API_PORT` | REST API server port | No | 8080 |
| `STATE_DB_PATH` | Path to BadgerDB storage | No | `./data` |
| `RPC_TIMEOUT` | Request timeout (seconds) | No | 30 |
| `RPC_MAX_RETRIES` | Maximum retry attempts | No | 3 |
| `RPC_RETRY_BACKOFF` | Initial retry backoff (ms) | No | 1000 |
| `RPC_MAX_RETRY_BACKOFF` | Maximum retry backoff (s) | No | 5 |

### Command-Line Flags

All environment variables can be overridden via command-line flags:

- `--rpc-url`: Override `TAN_ZK_RPC_URL`
- `--ws-url`: Override `TAN_ZK_WS_URL`
- `--batch-size`: Override `BATCH_SIZE`
- `--api-port`: Override `API_PORT`
- `--state-db-path`: Override `STATE_DB_PATH`

**Example:**
```bash
./sequencer start --rpc-url http://localhost:8545 --batch-size 200
```

## Architecture

### Modular Design

The sequencer is built with a modular architecture:

```
sequencer/
├── cmd/
│   └── sequencer/              # CLI application (Cobra commands)
│       ├── main.go             # Entry point
│       ├── root.go             # Root command
│       ├── start.go            # Start command
│       └── restart.go          # Restart command
├── pkg/
│   ├── errors/                 # Error handling package
│   │   └── errors.go          # Structured errors with codes
│   ├── logger/                 # Logging package
│   │   └── logger.go          # Structured logging (Info, Warn, Error, Debug)
│   ├── config/                 # Configuration package
│   │   └── config.go          # Environment and CLI config loading
│   └── rpc/                    # RPC client for Tan-ZK blockchain
│       ├── client.go           # HTTP RPC client
│       ├── websocket.go        # WebSocket client
│       ├── blocks.go           # Block operations
│       ├── transactions.go     # Transaction operations
│       └── accounts.go         # Account queries
├── internal/
│   └── sequencer/              # Core sequencer logic
│       ├── sequencer.go        # Main service
│       ├── batch.go            # Batch builder
│       ├── store.go            # BadgerDB batch storage
│       ├── api.go              # REST API and WebSocket server
│       └── types/
│           ├── batch.go        # Batch type definitions
│           └── config.go       # Configuration types
├── data/                       # BadgerDB storage directory
│   └── db/                     # BadgerDB files
├── .env.example                # Environment configuration template
└── README.md                   # This file
```

### Key Components

1. **Error Handling** (`pkg/errors`): Structured errors with error codes for better debugging
2. **Logging** (`pkg/logger`): Structured logging with Info, Warn, Error, and Debug levels
3. **Configuration** (`pkg/config`): Centralized config loading from `.env` and CLI flags
4. **RPC Client** (`pkg/rpc`): JSON-RPC and WebSocket client for Tan-ZK blockchain
5. **Sequencer Service** (`internal/sequencer`): Core batch building and block processing logic
6. **BadgerDB Storage** (`internal/sequencer/store.go`): Persistent batch storage with resume support

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
./sequencer start
```

4. **Send test transactions** (in another terminal):
```bash
# Anvil provides test accounts with funds
# You can use cast or send transactions via RPC
```

## What Happens When You Start

When you start the sequencer, you'll see structured logs:

```
[INFO] 🚀 Starting ZK Sequencer...
[INFO] ✅ Connected to Tan-ZK RPC endpoint
[INFO] ✅ Connected to Tan-ZK WebSocket endpoint (if configured)
[INFO] ✅ Batch store initialized
[INFO] 📦 Resuming from batch #5 with state root 0x...
[INFO] ✅ Batch builder initialized
[INFO] ✅ REST API server initialized on port 8080
[INFO] ✅ ZK Sequencer is running!
[INFO] 📡 Listening for new blocks and building batches...
[INFO] 🌐 REST API available at http://localhost:8080
[INFO] 📦 Processing block #12345
[INFO] ✅ Batch #5 created: 2 transactions (batch hash: 0xabcd...)
[INFO] ✅ Block #12345 processed: 2 transactions
```

The sequencer will continue running until you press `Ctrl+C`, at which point it will gracefully shutdown and save the last processed block.

## Logging

The sequencer uses structured logging with the following levels:

- **INFO**: General information (startup, batch creation, block processing)
- **WARN**: Warnings (connection failures, retries)
- **ERROR**: Errors (failed operations, critical issues)
- **DEBUG**: Debug information (detailed operation logs)

All logs include timestamps and file locations for easier debugging.

## Error Handling

The sequencer uses structured error handling with error codes:

- `CONFIG_ERROR`: Configuration-related errors
- `RPC_ERROR`: RPC client errors
- `WEBSOCKET_ERROR`: WebSocket connection errors
- `BATCH_STORE_ERROR`: Batch storage errors
- `BATCH_BUILDER_ERROR`: Batch building errors
- `API_ERROR`: API server errors
- `BLOCK_PROCESSING_ERROR`: Block processing errors

Errors are wrapped with context for better debugging and error tracking.

## Integration with Prover

The prover service can fetch batches from the sequencer:

1. **Poll for new batches**: The prover can periodically check `/prover/batch/latest`
2. **WebSocket subscriptions**: Connect to `/ws/batches` for real-time batch notifications
3. **Fetch specific batch**: Use `/prover/batch/{number}` to get a batch with execution traces
4. **Generate proof**: Use the batch data and traces to generate ZK proofs
5. **Submit proof**: Once proof is generated, submit it back to the sequencer (future feature)

## Troubleshooting

### Sequencer won't start

- Check that `TAN_ZK_RPC_URL` is set correctly in `.env`
- Verify the RPC endpoint is accessible: `curl $TAN_ZK_RPC_URL`
- Check logs for specific error messages

### No batches being created

- Ensure blocks contain transactions
- Check that the RPC endpoint is returning blocks
- Verify block processing logs in the console

### Database errors

- Ensure `STATE_DB_PATH` directory is writable
- Check disk space availability
- Try removing `data/db/` directory and restarting (will lose existing batches)

### WebSocket connection issues

- Verify `TAN_ZK_WS_URL` is correct
- Check firewall settings
- Sequencer will fallback to HTTP polling if WebSocket fails

## Next Steps

- ✅ Sequencer is functional and continuously running
- ✅ Monitors blocks, collects transactions, and builds batches incrementally
- ✅ REST API and WebSocket available for the prover
- ✅ BadgerDB storage with resume support
- ✅ Structured logging and error handling
- ✅ Cobra CLI with start/restart commands
- 🔄 Future enhancements: Sparse Merkle Tree for state roots (Milestone 1.4)
- 🔄 Future enhancements: Reorg handling (Milestone 1.4)
- 🔄 Future enhancements: Proof submission and on-chain verification
