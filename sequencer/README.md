# ZK Sequencer

The sequencer component of the Tan-ZK Sequencer system. This service is responsible for ingesting transactions from the Tan-ZK blockchain, batching them, and coordinating with the prover to generate zero-knowledge proofs.

## Overview

The sequencer connects to Tan-ZK blockchain nodes via JSON-RPC, collects transactions, builds batches, and submits proofs to the BatchRegistry contract.

## Project Structure

```
sequencer/
├── cmd/
│   └── sequencer/          # Main application entry point
├── pkg/
│   └── rpc/                # RPC client for Tan-ZK blockchain
├── internal/
│   └── sequencer/          # Core sequencer logic
├── .env.example            # Environment configuration template
└── README.md               # This file
```

## Features

### ✅ Implemented

- **RPC Client** (`pkg/rpc/`): JSON-RPC client for Tan-ZK blockchain
  - HTTP/HTTPS JSON-RPC support
  - Configurable timeouts and retries
  - Exponential backoff retry mechanism
  - Environment variable configuration
  - Integration tests with real RPC endpoints

### 🚧 In Progress

- WebSocket support for real-time block subscriptions
- Transaction fetching and processing
- Batch building and execution traces
- State synchronization

## Quick Start

### Prerequisites

- Go 1.22 or later
- Access to a Tan-ZK RPC endpoint (or use Anvil for local testing)

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

2. **Edit `.env` with your RPC endpoint**:
```bash
# Required: Your Tan-ZK RPC endpoint
TAN_ZK_RPC_URL=http://localhost:8545

# Optional: Customize timeouts and retries
RPC_TIMEOUT=30
RPC_MAX_RETRIES=3
RPC_RETRY_BACKOFF=100
RPC_MAX_RETRY_BACKOFF=10
```

### Running the Sequencer

```bash
# Build
go build ./cmd/sequencer

# Run
./sequencer
```

Or directly:
```bash
go run ./cmd/sequencer
```

## RPC Client Usage

### Basic Example

```go
package main

import (
    "context"
    "log"
    
    "github.com/tannetwork/zk-sequencer/sequencer/pkg/rpc"
)

func main() {
    // Create client from environment variables
    client, err := rpc.NewClientFromEnv()
    if err != nil {
        log.Fatal(err)
    }
    defer client.Close()

    ctx := context.Background()

    // Get chain ID
    chainID, err := client.ChainID(ctx)
    if err != nil {
        log.Fatal(err)
    }
    log.Printf("Chain ID: %d", chainID)

    // Get latest block number
    blockNum, err := client.BlockNumber(ctx)
    if err != nil {
        log.Fatal(err)
    }
    log.Printf("Latest block: %d", blockNum)
}
```

### With Custom Configuration

```go
import (
    "time"
    "github.com/tannetwork/zk-sequencer/sequencer/pkg/rpc"
)

// Create client with explicit URL
client, err := rpc.NewClient(
    "http://localhost:8545",
    rpc.WithTimeout(60*time.Second),
    rpc.WithMaxRetries(5),
)
```

### Environment Variables

The RPC client supports configuration via environment variables:

| Variable | Description | Default |
|----------|-------------|---------|
| `TAN_ZK_RPC_URL` | RPC endpoint URL (required) | - |
| `TAN_ZK_WS_URL` | WebSocket endpoint URL | - |
| `RPC_TIMEOUT` | Request timeout in seconds | 30 |
| `RPC_MAX_RETRIES` | Maximum retry attempts | 3 |
| `RPC_RETRY_BACKOFF` | Initial retry backoff in milliseconds | 100 |
| `RPC_MAX_RETRY_BACKOFF` | Maximum retry backoff in seconds | 10 |

## Testing

**⚠️ Important**: Always run tests from the `sequencer/` directory (where `go.mod` is located).

### Quick Start (Using Helper Script)

```bash
cd sequencer

# Unit tests (mock server, no RPC needed)
./test.sh unit

# Integration tests (requires .env with RPC URL)
./test.sh integration

# Run all tests
./test.sh all
```

### Unit Tests (Mock Server)

Unit tests use a mock HTTP server and don't require a real RPC endpoint:

```bash
cd sequencer
go test ./pkg/rpc/... -short -v
```

Run with coverage:

```bash
go test ./pkg/rpc/... -cover
```

### Integration Tests (Real RPC Endpoint)

Integration tests require a real Tan-ZK RPC endpoint:

1. **Set up `.env` file**:
```bash
cd sequencer
cp .env.example .env
# Edit .env with your RPC URL: TAN_ZK_RPC_URL=http://localhost:8545
```

2. **Run integration tests**:
```bash
cd sequencer
go test ./pkg/rpc/... -tags=integration -v
```

**Note**: If `TAN_ZK_RPC_URL` is not set, integration tests will skip gracefully.

### Testing with Anvil (Local Ethereum Node)

If you don't have a Tan-ZK node yet, you can test with Anvil (local Ethereum test node):

1. **Install Anvil** (part of Foundry):
```bash
# If you have Foundry installed
anvil --port 8545
```

2. **Update `.env`**:
```bash
TAN_ZK_RPC_URL=http://localhost:8545
```

3. **Run integration tests**:
```bash
go test ./pkg/rpc/... -tags=integration -v
```

### Common Test Errors

**Error**: `pattern ./pkg/rpc/...: directory prefix pkg/rpc does not contain main module`

**Solution**: Make sure you're in the `sequencer/` directory:
```bash
cd sequencer  # ← Important!
go test ./pkg/rpc/... -v
```

## Error Handling

The RPC client provides specific error types:

- `ErrInvalidConfig`: Configuration validation failed
- `ErrConnectionFailed`: Connection to RPC endpoint failed
- `ErrRPCError`: RPC call returned an error
- `ErrTimeout`: Request timed out
- `ErrMaxRetriesExceeded`: Maximum retries exceeded

Example error handling:

```go
blockNum, err := client.BlockNumber(ctx)
if err != nil {
    var rpcErr rpc.ErrRPCError
    if errors.As(err, &rpcErr) {
        log.Printf("RPC error [%d]: %s", rpcErr.Code, rpcErr.Message)
    } else {
        log.Fatal(err)
    }
}
```

## Development

### Project Structure

- **`cmd/sequencer/`**: Main application entry point
- **`pkg/rpc/`**: RPC client package for Tan-ZK blockchain
- **`internal/sequencer/`**: Internal sequencer logic (not for external use)

### Adding New Features

1. Create a feature branch: `git checkout -b feat/feature-name`
2. Implement the feature
3. Add tests (unit + integration if applicable)
4. Update documentation
5. Submit a pull request

### Code Style

- Follow Go standard formatting: `go fmt ./...`
- Run linters: `golangci-lint run`
- Write tests for new functionality
- Document exported functions and types

## Roadmap

### Milestone 1.3 - Tan-ZK RPC Client ✅
- [x] Core RPC client structure
- [x] Environment variable configuration
- [x] Integration tests with real RPC
- [ ] WebSocket support (#38)
- [ ] Block subscription (#39)
- [ ] Transaction fetching (#40)
- [ ] Account state queries (#41)
- [ ] SubmitBatchProof stub (#42)
- [ ] Robust error handling (#44)

### Future Milestones

- **Milestone 1.4**: State Synchronization
- **Milestone 2.1**: Transaction Pool
- **Milestone 2.2**: Batch Builder
- **Milestone 2.3**: Sequencer Service
- **Milestone 2.4**: REST API

See [docs/milestone.md](../docs/milestone.md) for complete roadmap.

## Troubleshooting

### Connection Issues

**Problem**: Cannot connect to RPC endpoint

**Solutions**:
- Verify RPC endpoint is running: `curl http://localhost:8545`
- Check firewall/network settings
- Verify URL in `.env` file is correct
- Check if endpoint requires authentication

### Timeout Issues

**Problem**: Requests timing out

**Solutions**:
- Increase `RPC_TIMEOUT` in `.env`
- Check network latency
- Verify RPC endpoint is responsive
- Check for rate limiting

### Integration Tests Skipped

**Problem**: Integration tests are skipped

**Solutions**:
- Ensure `.env` file exists with `TAN_ZK_RPC_URL` set
- Use `-tags=integration` flag when running tests
- Don't use `-short` flag (it skips integration tests)

## Contributing

See [CONTRIBUTING.md](../CONTRIBUTING.md) for contribution guidelines.

## License

Apache 2.0 - See [LICENSE](../LICENSE) file.

## Related Documentation

- [Project README](../README.md)
- [Milestones](../docs/milestone.md)
- [Security Policy](../SECURITY.md)
