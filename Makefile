# Nowa-ZK Project Makefile

.PHONY: all clean-artifacts setup build test test-sequencer test-sequencer-live anvil deploy-local run-indexer run-prover help check-batch

# Default target: Full fresh setup and test
all: clean-artifacts setup build test

# Default target for CI dependencies
deps:
	@cd indexer && go mod download
	@cd prover && go mod download
	@make install-swag

install-swag:
	@echo "⬇️  Installing Swag CLI..."
	@go install github.com/swaggo/swag/cmd/swag@latest

# --- 1. Clean ---

clean-artifacts:
	@echo "🧹 Cleaning build artifacts..."
	@cd contracts && forge clean && rm -rf broadcast/ cache/ deployments/ out/ src/generated/
	@cd indexer && go clean
	@cd prover && go clean
	@rm -rf build/

clean-data:
	@echo "🗑️  Clearing databases (Indexer & Prover)..."
	@rm -rf ~/.nowa-zk/indexer/data
	@rm -rf ~/.nowa-zk/prover/data
	@echo "✅ Databases cleared! (Keys and deployments were kept)"

clean-global: clean-artifacts
	@echo "💥 Wiping all global configurations, keys, and data..."
	@rm -rf ~/.nowa-zk/
	@echo "✅ Global state wiped!"

# --- 2. Setup (Keys & Verifier) ---

# Full Project Setup: Builds binaries, Generates keys, Compiles contracts
setup: install-swag swagger swagger-prover build-prover
	@echo "🔑 Running Prover Setup..."
	@mkdir -p ~/.nowa-zk/keys
	@./build/prover-bin setup --output-dir ~/.nowa-zk/keys --contract-output contracts/src/generated
	@echo "📝 Formatting generated contract..."
	@cd contracts && forge fmt src/generated/Verifier.sol src/generated/StateVerifier.sol
	@echo "🏗️  Building Indexer & Contracts..."
	@$(MAKE) build-indexer
	@$(MAKE) build-contracts

# --- 3. Build ---

build: install-swag swagger swagger-prover build-prover build-contracts build-indexer

build-prover:
	@echo "🏗️  Building Prover..."
	@mkdir -p build
	@cd prover && go build -o ../build/prover-bin ./cmd/prover

build-contracts:
	@echo "🏗️  Building Contracts..."
	@cd contracts && forge build

swagger:
	@echo "📜 Generating Indexer Swagger Docs..."
	@cd indexer && $(HOME)/go/bin/swag init -g internal/indexer/api.go --output docs --parseDependency --parseInternal

swagger-prover:
	@echo "📜 Generating Prover Swagger Docs..."
	@cd prover && $(HOME)/go/bin/swag init -g internal/api/server.go --output docs --parseDependency --parseInternal

build-indexer:
	@echo "🏗️  Building Indexer..."
	@mkdir -p build
	@cd indexer && go build -o ../build/indexer-bin ./cmd/indexer

build-sequencer:
	@echo "🏗️  Building Sequencer..."
	@mkdir -p build
	@cd sequencer && go build -o ../build/sequencer-bin ./cmd/sequencer

# --- 4. Test ---

test: test-contracts test-indexer test-prover test-sequencer

test-contracts:
	@echo "🧪 Testing Contracts..."
	@cd contracts && forge test

test-indexer:
	@echo "🧪 Testing Indexer..."
	@cd indexer && go test ./...

test-prover:
	@echo "🧪 Testing Prover..."
	@cd prover && go test ./...

test-sequencer:
	@echo "🧪 Testing Sequencer (match + balance updates)..."
	@cd sequencer && go test ./... -count=1

# Posts Alice buy + Bob sell against a running sequencer (make run-sequencer).
test-sequencer-live:
	@echo "🧪 Live sequencer smoke (requires make run-sequencer on :8080)..."
	@cd sequencer && go run ./cmd/cli/test_client.go

# --- 5. Run (Development Workflow) ---

# Terminal 1: Start local Anvil chain ( for testing, or If don't have a target blockchain )
anvil:
	@anvil --port 8545

# Deploy to custom network using .env configuration
deploy:
	@if [ ! -f contracts/src/generated/Verifier.sol ]; then \
		echo "⚠️  Verifier.sol not found. Running setup..."; \
		$(MAKE) setup; \
	fi
	@echo "🚀 Deploying Contracts..."
	@mkdir -p contracts/deployments
	@# Load .env variables from ROOT .env
	@if [ -f .env ]; then export $$(grep -v '^#' .env | xargs); fi; \
	cd contracts && forge script script/Deploy.s.sol --rpc-url $${L1_RPC_URL} --broadcast
	@mkdir -p .nowa-zk
	@echo "📦 Copying deployment file..."
	@mkdir -p ~/.nowa-zk
	@echo "📦 Copying deployment file to ~/.nowa-zk/..."
	@cp contracts/deployments/deployments.json ~/.nowa-zk/deployments.json
	@if [ ! -f .nowa-zk/secrets.env ]; then \
		cp .env .nowa-zk/secrets.env; \
		echo "📝 Created .nowa-zk/secrets.env from .env"; \
	fi
	@echo "✅ Deployment info saved to .nowa-zk/deployments.json"

# Optional: ( New Terminal ) Run Traffic Generator
# Usage: make run-traffic-gen [COUNT=10000]
run-traffic-gen: build-indexer
	@if [ -f .env ]; then export $$(grep -v '^[[:space:]]*#' .env | xargs); fi; \
	./build/indexer-bin traffic-gen --count $(or $(COUNT), 10000) --rpc $${L2_RPC_URL}

# Terminal 3a: Run Sequencer (off-chain matching + batches for prover on :8080)
run-sequencer: build-sequencer
	@echo "⚡ Starting Sequencer on :8080..."
	@mkdir -p ~/.nowa-zk/sequencer
	@if [ -f .env ]; then export $$(grep -v '^[[:space:]]*#' .env | xargs); fi; \
	cd ~/.nowa-zk/sequencer && \
	ROLLUP_CONTRACT_ADDRESS=$${ROLLUP_CONTRACT_ADDRESS:-$$(jq -r '.NowaRollup' ~/.nowa-zk/deployments.json 2>/dev/null)} \
	L1_RPC_URL=$${L1_RPC_URL} \
	$(CURDIR)/build/sequencer-bin

# Wipe sequencer LevelDB (correct path — make run-sequencer cds here)
clean-sequencer-state:
	@echo "🧹 Clearing ~/.nowa-zk/sequencer/nowa_state_db"
	@rm -rf ~/.nowa-zk/sequencer/nowa_state_db

# Terminal 3: Run Indexer
run-indexer: build-indexer
	@mkdir -p ~/.nowa-zk/indexer/data
	@if [ -f .env ]; then export $$(grep -v '^[[:space:]]*#' .env | xargs); fi; \
	./build/indexer-bin start --rpc-url $${L2_RPC_URL:-http://localhost:8545} --state-db-path ~/.nowa-zk/indexer/data

# Terminal 4: Run Prover (fetches batches from sequencer at :8080 by default)
# Usage: make run-prover [CONTRACT=...] [KEY=...]
#   If CONTRACT/KEY are omitted, they are auto-loaded from ~/.nowa-zk/deployments.json and .env
run-prover: build-prover
	@echo "🔐 Starting Prover..."
	@if [ -f .env ]; then export $$(grep -v '^[[:space:]]*#' .env | xargs); fi; \
	CONTRACT_ADDR="$(CONTRACT)"; \
	if [ -z "$$CONTRACT_ADDR" ]; then \
		CONTRACT_ADDR="$${ROLLUP_CONTRACT_ADDRESS:-}"; \
	fi; \
	if [ -z "$$CONTRACT_ADDR" ] && [ -f "$$HOME/.nowa-zk/deployments.json" ]; then \
		CONTRACT_ADDR=$$(jq -r '.NowaRollup // empty' "$$HOME/.nowa-zk/deployments.json"); \
	fi; \
	if [ -z "$$CONTRACT_ADDR" ] || [ "$$CONTRACT_ADDR" = "null" ]; then \
		echo "❌ Contract address required. Set CONTRACT=0x... or deploy (see ~/.nowa-zk/deployments.json)"; \
		exit 1; \
	fi; \
	echo "📄 Using NowaRollup $$CONTRACT_ADDR"; \
	./build/prover-bin start --keys-dir ~/.nowa-zk/keys --indexer-url http://localhost:8080 \
		--contract $$CONTRACT_ADDR \
		$(if $(KEY),--private-key $(KEY),) $(if $(CLEAR_HALT),--clear-halt,)

# --- Help ---

help:
	@echo "Nowa-ZK Makefile Commands (in execution order):"
	@echo "  make clean-artifacts - 1a. Clear build artifacts"
	@echo "  make clean-data      - 1b. Clear indexer/prover databases only"
	@echo "  make clean-global    - 1c. Clear global artifacts (~/.nowa-zk/)"
	@echo "  make setup           - 2. Generate keys & verifier"
	@echo "  make build           - 3. Build all binaries"
	@echo "  make test            - 4. Run all tests"
	@echo "  make test-sequencer  - 4b. Sequencer unit/integration (match + balances)"
	@echo "  make test-sequencer-live - 4c. Smoke orders vs running sequencer :8080"
	@echo ""
	@echo "Run Workflow:"
	@echo "  make anvil           - 5. Start chain (Term 1)"
	@echo "  make deploy          - 6. Deploy contracts (Term 2)"
	@echo "  make verify-contracts - 6b. Verify contracts on Etherscan"
	@echo "  make run-sequencer   - 7. Start sequencer (Term 3) — batches for prover"
	@echo "  make run-indexer     - 7b. Start indexer (optional / legacy L2 indexing)"
	@echo "  make run-prover      - 8. Start prover (Term 4)"
	@echo "  make check-batch     - 9. Check latest batch info"

# Verify contracts on Etherscan (Sepolia)
verify-contracts:
	@echo "🔍 Starting Contract Verification..."
	@if [ -f .env ]; then export $$(grep -v '^[[:space:]]*#' .env | xargs); fi; \
	if [ -z "$$ETHERSCAN_API_KEY" ]; then \
		echo "❌ Error: ETHERSCAN_API_KEY is not set in .env"; \
		exit 1; \
	fi; \
	VERIFIER_ADDR=$$(jq -r '.Verifier' ~/.nowa-zk/deployments.json); \
	ROLLUP_ADDR=$$(jq -r '.NowaRollup' ~/.nowa-zk/deployments.json); \
	if [ -z "$$VERIFIER_ADDR" ] || [ "$$VERIFIER_ADDR" = "null" ]; then \
		echo "❌ Error: Verifier address missing in ~/.nowa-zk/deployments.json"; \
		exit 1; \
	fi; \
	if [ -z "$$ROLLUP_ADDR" ] || [ "$$ROLLUP_ADDR" = "null" ]; then \
		echo "❌ Error: NowaRollup address missing in ~/.nowa-zk/deployments.json"; \
		exit 1; \
	fi; \
	echo "🔍 Verifying Verifier at $$VERIFIER_ADDR..."; \
	cd contracts && forge verify-contract --chain-id 11155111 --watch --etherscan-api-key $$ETHERSCAN_API_KEY $$VERIFIER_ADDR src/generated/Verifier.sol:Verifier; \
	echo "🔍 Verifying NowaRollup at $$ROLLUP_ADDR..."; \
	forge verify-contract --chain-id 11155111 --watch --constructor-args $$(cast abi-encode "constructor(address,bytes32)" $$VERIFIER_ADDR $$(cast --to-bytes32 0)) --etherscan-api-key $$ETHERSCAN_API_KEY $$ROLLUP_ADDR src/NowaRollup.sol:NowaRollup
	@echo "✅ Verification complete!"
