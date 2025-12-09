# CODEME

## Quick Start (Makefile)

```bash
# Build everything
make build

# Run all tests
make test

# Setup prover keys & contracts
make setup

# Clean everything
make clean
```

## Run Workflow (Makefile)

The Makefile provides shortcuts for the entire development lifecycle:

```bash
# 1. Start Local Anvil Chain (Terminal 1)
make anvil

# 2. Deploy Contracts (Terminal 2)
make deploy-local

# 3. Start Sequencer (Terminal 3)
make run-sequencer

# 4. Start Prover (Terminal 4)
make run-prover

# Utilities
make check-batch  # Check latest batch status
```

## Manual Workflow

### 1. Setup (Prover)

```bash
cd prover

# Generate keys and RollupVerifier.sol
go run ./cmd/prover setup
# Output: keys/rollup.pk, keys/rollup.vk, ../contracts/src/generated/RollupVerifier.sol
```

### 2. Contracts

```bash
cd ../contracts

# Build (includes generated Verifier)
forge build

# Deploy (Local Anvil)
source .env
forge script script/Deploy.s.sol --rpc-url $RPC --broadcast

# Deploy (Testnet)
# forge script script/Deploy.s.sol --rpc-url $RPC --private-key $PRIVATE_KEY --broadcast --verify
```

### 3. Sequencer

```bash
cd ../sequencer

# Start (Local)
export RPC=http://localhost:8545
go run ./cmd/sequencer start

# Clear Data
# go run ./cmd/sequencer clear
```

### 4. Prover Service (New Terminal)

```bash
cd ../prover

# Start
go run ./cmd/prover start --contract 0xYOUR_CONTRACT_ADDRESS --private-key 0xYOUR_PRIVATE_KEY
```

## Docker

```bash
# Start Infrastructure (Anvil + Sequencer + Prover)
docker-compose up --build
```
