# Nowa-ZK Project Makefile

.PHONY: all clean-artifacts setup build test anvil deploy-local run-indexer run-prover help check-batch

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
	@cd contracts && forge fmt src/generated/RollupVerifier.sol
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

# --- 4. Test ---

test: test-contracts test-indexer test-prover

test-contracts:
	@echo "🧪 Testing Contracts..."
	@cd contracts && forge test

test-indexer:
	@echo "🧪 Testing Indexer..."
	@cd indexer && go test ./...

test-prover:
	@echo "🧪 Testing Prover..."
	@cd prover && go test ./...

# --- 5. Run (Development Workflow) ---

# Terminal 1: Start local Anvil chain ( for testing, or If don't have a target blockchain )
anvil:
	@anvil --port 8545

# Deploy to custom network using .env configuration
deploy:
	@if [ ! -f contracts/src/generated/RollupVerifier.sol ]; then \
		echo "⚠️  RollupVerifier.sol not found. Running setup..."; \
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

# Terminal 3: Run Indexer
run-indexer: build-indexer
	@mkdir -p ~/.nowa-zk/indexer/data
	@if [ -f .env ]; then export $$(grep -v '^[[:space:]]*#' .env | xargs); fi; \
	./build/indexer-bin start --rpc-url $${L2_RPC_URL:-http://localhost:8545} --state-db-path ~/.nowa-zk/indexer/data

# Terminal 4: Run Prover
# Usage: make run-prover [CONTRACT=...] [KEY=...]
#   If CONTRACT/KEY are omitted, they are auto-loaded from .nowa-zk/deployments.json and .nowa-zk/secrets.env
run-prover: build-prover
	@echo "🔐 Starting Prover..."
	@./build/prover-bin start --keys-dir ~/.nowa-zk/keys $(if $(CONTRACT),--contract $(CONTRACT),) $(if $(KEY),--private-key $(KEY),)

# --- Help ---

help:
	@echo "Nowa-ZK Makefile Commands (in execution order):"
	@echo "  make clean-artifacts - 1a. Clear build artifacts"
	@echo "  make clean-data      - 1b. Clear indexer/prover databases only"
	@echo "  make clean-global    - 1c. Clear global artifacts (~/.nowa-zk/)"
	@echo "  make setup           - 2. Generate keys & verifier"
	@echo "  make build           - 3. Build all binaries"
	@echo "  make test            - 4. Run all tests"
	@echo ""
	@echo "Run Workflow:"
	@echo "  make anvil           - 5. Start chain (Term 1)"
	@echo "  make deploy          - 6. Deploy contracts (Term 2)"
	@echo "  make run-indexer   - 7. Start indexer (Term 3)"
	@echo "  make run-prover      - 8. Start prover (Term 4)"
	@echo "  make check-batch     - 9. Check latest batch info"
