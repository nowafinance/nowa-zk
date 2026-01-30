# CODEME

## Quick Start Guide

Follow these steps in order to set up and run the Nowa-ZK system:

### 1. Build Required Files

#### Generate Prover Keys and Verifier Contract

```bash
cd prover

# Generate proving/verification keys and RollupVerifier.sol
go run ./cmd/prover setup
```

**Output:**
- `prover/keys/rollup.pk` - Proving key
- `prover/keys/rollup.vk` - Verification key
- `contracts/src/generated/RollupVerifier.sol` - Verifier contract

#### Build Contracts

```bash
cd ../contracts

# Build all contracts (including generated RollupVerifier)
forge build
```

---

### 2. Configure Environment

Create a `.env` file in the `contracts` directory:

```bash
cd contracts
```

**Create `.env` file:**

```bash
cat > .env << 'EOF'
# RPC URL (your network endpoint)
RPC=https://eth-sepolia.g.alchemy.com/v2/YOUR_API_KEY

# Deployer Private Key
PRIVATE_KEY=0xYOUR_PRIVATE_KEY_HERE

# Etherscan API Key (for contract verification)
ETHERSCAN_API_KEY=YOUR_ETHERSCAN_API_KEY
EOF
```

> **⚠️ Security Warning:** Never commit `.env` to version control. Keep your private key secure!

**Edit the values:**
- `RPC`: Your Ethereum node RPC URL (Alchemy, Infura, etc.)
- `PRIVATE_KEY`: Your deployer wallet private key
- `ETHERSCAN_API_KEY`: Your Etherscan API key for contract verification

---

### 3. Deploy Contracts

```bash
cd contracts
set -a
source .env
set +a

# Deploy to your network
forge script script/Deploy.s.sol:Deploy --rpc-url $RPC --private-key $PRIVATE_KEY --broadcast

# Optional: Verify on Etherscan
# forge script script/Deploy.s.sol:Deploy --rpc-url $RPC --private-key $PRIVATE_KEY --broadcast --verify
```

**Save the deployed contract address** - you'll need it for the prover service.

---

### 4. Start Sequencer

Open a new terminal:

```bash
cd sequencer

# Set RPC to your network
export RPC=https://eth-sepolia.g.alchemy.com/v2/YOUR_API_KEY

# Start the sequencer
go run ./cmd/sequencer start
```

**To clear sequencer data (if needed):**
```bash
go run ./cmd/sequencer clear
```

---

### 5. Start Prover Service

Open another terminal:

```bash
cd prover

# Start prover with deployed contract address and private key
go run ./cmd/prover start --contract 0xYOUR_CONTRACT_ADDRESS --private-key 0xYOUR_PRIVATE_KEY
```

**Parameters:**
- `--contract`: The deployed RollupContract address from step 3
- `--private-key`: Private key with funds to submit proofs

---

## Docker Deployment

To run the entire infrastructure using Docker:

```bash
# Build and start all services (Sequencer + Prover)
docker-compose up --build
```

**Note:** Make sure to configure your `.env` file before running Docker.

---

## Utilities

### Check Latest Batch Status

```bash
cd sequencer
go run ./cmd/sequencer check-batch
```

### Rebuild Everything

```bash
# Clean and rebuild prover
cd prover
rm -rf keys/
go run ./cmd/prover setup

# Rebuild contracts
cd ../contracts
forge clean
forge build
```

---

## Troubleshooting

- **"Keys not found"**: Run `go run ./cmd/prover setup` first
- **"Contract not found"**: Verify deployment was successful and contract address is correct
- **"Insufficient funds"**: Ensure deployer/prover wallet has enough ETH
- **Build fails**: Make sure `forge` and Go are properly installed
