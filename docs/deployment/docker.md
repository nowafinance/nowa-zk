# 🐳 Docker Setup Guide

This guide describes how to run the **nowa-zk** system (Indexer & Prover) using Docker, connecting to an **external blockchain network** (e.g., Sepolia, or any EVM-compatible chain).

---

## Prerequisites

*   **Docker** installed (includes `docker compose`)
*   **Go 1.24.10+** (for generating keys)
*   **Foundry** (for deploying contracts)

---

## Setup Steps

### 0. Install Dependencies

If you haven't installed the required tools yet:

```bash
sudo apt update
sudo apt install -y make git build-essential curl

# Install Go 1.24.10
curl -OL https://go.dev/dl/go1.24.10.linux-amd64.tar.gz
sudo rm -rf /usr/local/go && sudo tar -C /usr/local -xzf go1.24.10.linux-amd64.tar.gz
export PATH=$PATH:/usr/local/go/bin
echo 'export PATH=$PATH:/usr/local/go/bin' >> ~/.bashrc
source ~/.bashrc

# Install Foundry (Forge, Cast, Anvil)
curl -L https://foundry.paradigm.xyz | bash
source ~/.bashrc
foundryup
```

### 1. Generate Prover Keys

Keys must be generated on the host machine first.

```bash
cd prover

# Build prover binary
go build -o ../build/prover-bin ./cmd/prover

# Generate keys and verifier contract
../build/prover-bin setup --output-dir ../keys --contract-output ../contracts/src/generated

cd ..
```

**Output:**
- `keys/rollup.pk` - Proving key
- `keys/rollup.vk` - Verification key
- `contracts/src/generated/RollupVerifier.sol` - Verifier contract

---

### 2. Configure Environment

Create a `.env` file in the project root:

```bash
cat > .env << 'EOF'
# RPC URL of your target blockchain
RPC=https://eth-sepolia.g.alchemy.com/v2/YOUR_API_KEY

# Private Key for deployment and proof submission
PRIVATE_KEY=0xYOUR_PRIVATE_KEY_HERE

# Start indexing from this block
INDEX_FROM_BLOCK=0

# Optional: Etherscan API key for verification
ETHERSCAN_API_KEY=YOUR_ETHERSCAN_KEY
EOF
```

**Edit the values:**
- `RPC`: Your Ethereum node RPC URL (Alchemy, Infura, etc.)
- `PRIVATE_KEY`: Your wallet private key (must have funds)
- `INDEX_FROM_BLOCK`: Block number to start indexing from

> [!WARNING]
> Never commit `.env` to version control. Keep your private key secure!

---

### 3. Build Contracts

```bash
cd contracts

# Build contracts (includes generated verifier)
forge build

cd ..
```

---

### 4. Deploy Contracts

Deploy contracts to your target network:

```bash
cd contracts
set -a
source ../.env
set +a

# Deploy
forge script script/Deploy.s.sol:Deploy --rpc-url $RPC --private-key $PRIVATE_KEY --broadcast

# Optional: Verify on Etherscan
# forge script script/Deploy.s.sol:Deploy --rpc-url $RPC --private-key $PRIVATE_KEY --broadcast --verify

cd ..
```

**Deployment addresses saved to:** `contracts/deployments/deployments.json`

> [!TIP]
> The deployment script automatically saves contract addresses to `deployments/deployments.json`. Docker Compose will mount this file so the Prover can auto-load contract addresses.

---

### 5. Start Docker Services

Start the Indexer and Prover containers:

```bash
docker compose up -d
```

This will:
- Build Docker images for both services
- Start the Indexer on port `8080`
- Start the Prover service
- Automatically load configuration from `.env`

---

## Verifying the System

### Check Container Status

```bash
# List running containers
docker compose ps

# Check if healthy
docker compose ps
```

### View Logs

```bash
# Follow indexer logs
docker compose logs -f indexer

# Follow prover logs
docker compose logs -f prover

# View both
docker compose logs -f
```

### Test API Endpoints

The Indexer API is exposed on port **8080**:

```bash
# Check indexer status
curl http://localhost:8080/status

# Check latest batch
curl http://localhost:8080/batch/latest

# View Swagger docs
open http://localhost:8080/swagger/index.html
```

---

## Managing Services

### Stop Services

```bash
docker compose down
```

### Restart Services

```bash
docker compose restart
```

### Rebuild After Code Changes

```bash
# Rebuild and restart
docker compose up -d --build
```

### View Resource Usage

```bash
docker compose stats
```

---

## Troubleshooting

### "Connection refused" or RPC errors

*   Ensure your RPC endpoint is accessible from within Docker
*   Verify `.env` has correct RPC URL
*   Check if your RPC provider has rate limits

### "Contract address required"

*   Ensure you ran `forge script` for deployment
*   Check that deployment was successful
*   Verify `.env` has correct PRIVATE_KEY with funds

### "Keys not found"

*   Make sure you ran step 1 (Generate Prover Keys)
*   Verify `keys/` directory exists with `.pk` and `.vk` files
*   Check that docker-compose mounts keys correctly

### Indexer not processing blocks

```bash
# Check logs for errors
docker compose logs indexer | grep -i error

# Verify RPC connection
docker compose exec indexer curl -X POST $RPC \
  -H "Content-Type: application/json" \
  -d '{"jsonrpc":"2.0","method":"eth_blockNumber","params":[],"id":1}'
```

### Reset Everything

To wipe all container data and start fresh:

```bash
# Stop and remove containers and volumes
docker compose down -v

# Remove local state data
rm -rf dir_data/

# Restart
docker compose up -d
```

---

## Advanced Configuration

### Custom State Persistence Path

By default, state is stored in `dir_data/`. To change this, edit `docker-compose.yml`:

```yaml
volumes:
  - ./your-custom-path:/app/data
```

### Resource Limits

Limit CPU and memory usage in `docker-compose.yml`:

```yaml
services:
  prover:
    deploy:
      resources:
        limits:
          cpus: '4'
          memory: 8G
```

### Using Different Networks

To connect to different networks, just update `.env`:

```bash
# For Mainnet
RPC=https://eth-mainnet.g.alchemy.com/v2/YOUR_API_KEY

# For custom network
RPC=https://your-custom-node.example.com
```

Then restart:

```bash
docker compose down
docker compose up -d
```
