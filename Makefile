# Tan-ZK Project Makefile

.PHONY: all clean setup build test anvil deploy-local run-sequencer run-prover help check-batch

# Default target: Full fresh setup and test
all: clean setup build test

# --- 1. Clean ---

clean:
	@echo "🧹 Cleaning up..."
	@cd contracts && forge clean && rm -rf broadcast/ cache/ deployments/ out/ src/generated/
	@cd sequencer && go clean && rm -rf sequencer-bin
	@cd prover && go clean && rm -rf prover-bin
	@rm -rf .tan-zk/sequencer/data
	@rm -rf .tan-zk/prover/data
	@rm -rf .tan-zk/

# --- 2. Setup (Keys & Verifier) ---

# Generates prover keys and RollupVerifier.sol
setup: build-prover
	@echo "🔑 Running Prover Setup..."
	@mkdir -p .tan-zk/keys
	@cd prover && ./prover-bin setup --output-dir ../.tan-zk/keys --contract-output ../contracts/src/generated

# --- 3. Build ---

build: build-prover build-contracts build-sequencer

build-prover:
	@echo "🏗️  Building Prover..."
	@cd prover && go build -o prover-bin ./cmd/prover

build-contracts:
	@echo "🏗️  Building Contracts..."
	@cd contracts && forge build

build-sequencer:
	@echo "🏗️  Building Sequencer..."
	@cd sequencer && go build -o sequencer-bin ./cmd/sequencer

# --- 4. Test ---

test: test-contracts test-sequencer test-prover

test-contracts:
	@echo "🧪 Testing Contracts..."
	@cd contracts && forge test

test-sequencer:
	@echo "🧪 Testing Sequencer..."
	@cd sequencer && go test ./...

test-prover:
	@echo "🧪 Testing Prover..."
	@cd prover && go test ./...

# --- 5. Run (Development Workflow) ---

# Terminal 1: Start local Anvil chain
anvil:
	@anvil --port 8545

# Terminal 2: Deploy contracts to local Anvil chain
deploy-local:
	@echo "🚀 Deploying Contracts to Local Anvil..."
	@mkdir -p contracts/deployments
	@cd contracts && forge script script/Deploy.s.sol --rpc-url http://localhost:8545 --broadcast
	@mkdir -p .tan-zk
	@cp contracts/deployments/31337.json .tan-zk/deployments.json
	@if [ ! -f .tan-zk/secrets.env ]; then \
		echo "PRIVATE_KEY=0xac0974bec39a17e36ba4a6b4d238ff944bacb478cbed5efcae784d7bf4f2ff80" > .tan-zk/secrets.env; \
		echo "📝 Created .tan-zk/secrets.env with default Anvil key"; \
	fi
	@echo "✅ Deployment info saved to .tan-zk/deployments.json"

# Optional: ( New Terminal ) Run Traffic Generator
# Usage: make run-traffic-gen [COUNT=10000]
run-traffic-gen: build-sequencer
	@./sequencer/sequencer-bin traffic-gen --count $(or $(COUNT), 10000)

# Terminal 3: Run Sequencer
run-sequencer: build-sequencer
	@mkdir -p .tan-zk/sequencer/data
	@./sequencer/sequencer-bin start --reset --rpc-url http://localhost:8545 --state-db-path .tan-zk/sequencer/data


# Terminal 4: Run Prover
# Usage: make run-prover [CONTRACT=...] [KEY=...]
#   If CONTRACT/KEY are omitted, they are auto-loaded from .tan-zk/deployments.json and .tan-zk/secrets.env
run-prover: build-prover
	@echo "🔐 Starting Prover..."
	@./prover/prover-bin start --keys-dir .tan-zk/keys $(if $(CONTRACT),--contract $(CONTRACT),) $(if $(KEY),--private-key $(KEY),)

# --- Help ---

help:
	@echo "Tan-ZK Makefile Commands (in execution order):"
	@echo "  make clean           - 1. Clear artifacts"
	@echo "  make setup           - 2. Generate keys & verifier"
	@echo "  make build           - 3. Build all binaries"
	@echo "  make test            - 4. Run all tests"
	@echo ""
	@echo "Run Workflow:"
	@echo "  make anvil           - 5. Start chain (Term 1)"
	@echo "  make deploy-local    - 6. Deploy contracts (Term 2)"
	@echo "  make run-sequencer   - 7. Start sequencer (Term 3)"
	@echo "  make run-prover      - 8. Start prover (Term 4)"
	@echo "  make check-batch     - 9. Check latest batch info"

# --- 6. Utilities ---

check-batch:
	@echo "🔍 Checking Batch Registry..."
	@TOTAL=$$(cast call 0xCf7Ed3AccA5a467e9e704C703E8D87F634fB0Fc9 "totalBatches()(uint256)" --rpc-url http://localhost:8545); \
	echo "📊 Total Batches: $$TOTAL"; \
	if [ "$$TOTAL" -gt 0 ]; then \
		echo "📄 Latest Batch Details:"; \
		cast call 0xCf7Ed3AccA5a467e9e704C703E8D87F634fB0Fc9 "getBatch(uint256)(tuple(bytes32,bytes32,bytes32,address,uint256,uint256,uint8))" $$TOTAL --rpc-url http://localhost:8545; \
	else \
		echo "⚠️  No batches registered yet."; \
	fi
