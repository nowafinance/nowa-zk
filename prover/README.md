# Prover

ZK Prover for Tan-ZK Sequencer - Generates proofs and submits to smart contract.

## Quick Start

```bash
# Build
go build -o prover-bin ./cmd/prover

# Generate keys and Solidity verifier
./prover-bin setup

# Start prover (fetch batches and submit proofs)
./prover-bin start --contract 0x... --private-key YOUR_KEY
```

## Commands

### Setup

Generates proving/verifying keys and exports Solidity verifier contract.

```bash
./prover-bin setup
# Options:
#   -o, --output-dir string         Directory for keys (default: ./keys)
#   -c, --contract-output string    Directory for Solidity file (default: ../contracts/src/generated)
```

**Output:**
- `keys/rollup.pk` - Proving key
- `keys/rollup.vk` - Verifying key  
- `keys/rollup.r1cs` - Compiled circuit
- `contracts/src/generated/RollupVerifier.sol` - Solidity verifier

### Start

Fetches batches from sequencer API, generates proofs, submits to contract.

```bash
./prover-bin start --contract 0xABC... --private-key abc123...
# Options:
#   -k, --keys-dir string        Keys directory (default: ./keys)
#   -s, --sequencer-url string   Sequencer API URL (default: http://localhost:8080)
#   -r, --rpc-url string         Ethereum RPC URL (default: http://localhost:8545)
#   -c, --contract string        Contract address (REQUIRED)
#   -p, --private-key string     Private key for submitting txs (REQUIRED)
#   -i, --poll-interval int      Poll interval in seconds (default: 10)
```

## Circuit

**BatchCircuit** processes 128 transactions per batch:
- Computes transaction hashes (MiMC)
- Builds Merkle tree
- Verifies root matches public input

**Transaction fields:**
- `From` - Sender address
- `To` - Recipient address
- `Amount` - Transfer amount
- `Nonce` - Sender nonce
- `InputHash` - Hash of transaction data

## API

The prover exposes an HTTP API (default port 8081) for monitoring and status checks.

### Endpoints

- `GET /status/:id` - Check proof generation status for a batch.
- `GET /proof/:id` - Retrieve the full binary proof data for a batch.
- `GET /batches/latest` - Get details of the latest verified batch from L1.
- `GET /batches/:id` - Get details of a specific batch from L1.

## testing

```bash
# Run all tests
go test ./...

# Test circuits only
go test ./circuits -v

# With coverage
go test ./... -coverprofile=coverage.out
go tool cover -html=coverage.out
```

## Workflow

1. **Setup** - Generate keys once
2. **Deploy** - Deploy `RollupVerifier.sol` to chain
3. **Start** - Run prover to process batches
4. Prover polls sequencer API for new batches
5. Generates ZK proof for each batch
6. Submits proof to smart contract for verification

See [/CODEME.md](../CODEME.md) for full command reference.