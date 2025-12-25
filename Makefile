# Tan-ZK Project Makefile

.PHONY: all clean setup build test anvil deploy-local run-sequencer run-prover help check-batch

# Default target: Full fresh setup and test
all: clean setup build test

# Default target for CI dependencies
deps:
	@cd sequencer && go mod download
	@cd prover && go mod download
	@make install-swag

install-swag:
	@echo "⬇️  Installing Swag CLI..."
	@go install github.com/swaggo/swag/cmd/swag@latest

# --- 1. Clean ---

clean:
	@echo "🧹 Cleaning up..."
	@cd contracts && forge clean && rm -rf broadcast/ cache/ deployments/ out/ src/generated/
	@cd sequencer && go clean
	@cd prover && go clean
	@rm -rf build/
	@rm -rf ~/.tan-zk/

# --- 2. Setup (Keys & Verifier) ---

# Full Project Setup: Builds binaries, Generates keys, Compiles contracts
setup: install-swag swagger swagger-prover build-prover
	@echo "🔑 Running Prover Setup..."
	@mkdir -p ~/.tan-zk/keys
	@./build/prover-bin setup --output-dir ~/.tan-zk/keys --contract-output contracts/src/generated
	@echo "📝 Formatting generated contract..."
	@cd contracts && forge fmt src/generated/RollupVerifier.sol
	@echo "🏗️  Building Sequencer & Contracts..."
	@$(MAKE) build-sequencer
	@$(MAKE) build-contracts

# --- 3. Build ---

build: install-swag swagger swagger-prover build-prover build-contracts build-sequencer

build-prover:
	@echo "🏗️  Building Prover..."
	@mkdir -p build
	@cd prover && go build -o ../build/prover-bin ./cmd/prover

build-contracts:
	@echo "🏗️  Building Contracts..."
	@cd contracts && forge build

swagger:
	@echo "📜 Generating Sequencer Swagger Docs..."
	@cd sequencer && $(HOME)/go/bin/swag init -g internal/sequencer/api.go --output docs --parseDependency --parseInternal

swagger-prover:
	@echo "📜 Generating Prover Swagger Docs..."
	@cd prover && $(HOME)/go/bin/swag init -g internal/api/server.go --output docs --parseDependency --parseInternal

build-sequencer:
	@echo "🏗️  Building Sequencer..."
	@mkdir -p build
	@cd sequencer && go build -o ../build/sequencer-bin ./cmd/sequencer

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
	cd contracts && forge script script/Deploy.s.sol --rpc-url $${RPC_PROVER} --broadcast
	@mkdir -p .tan-zk
	@echo "📦 Copying deployment file..."
	@mkdir -p ~/.tan-zk
	@echo "📦 Copying deployment file to ~/.tan-zk/..."
	@cp contracts/deployments/deployments.json ~/.tan-zk/deployments.json
	@if [ ! -f .tan-zk/secrets.env ]; then \
		cp .env .tan-zk/secrets.env; \
		echo "📝 Created .tan-zk/secrets.env from .env"; \
	fi
	@echo "✅ Deployment info saved to .tan-zk/deployments.json"

# Optional: ( New Terminal ) Run Traffic Generator
# Usage: make run-traffic-gen [COUNT=10000]
run-traffic-gen: build-sequencer
	@if [ -f .env ]; then export $$(grep -v '^[[:space:]]*#' .env | xargs); fi; \
	./build/sequencer-bin traffic-gen --count $(or $(COUNT), 10000) --rpc $${RPC_SEQUENCER}

# Terminal 3: Run Sequencer
run-sequencer: build-sequencer
	@mkdir -p ~/.tan-zk/sequencer/data
	@if [ -f .env ]; then export $$(grep -v '^[[:space:]]*#' .env | xargs); fi; \
	./build/sequencer-bin start --rpc-url $${RPC_SEQUENCER:-http://localhost:8545} --state-db-path ~/.tan-zk/sequencer/data

# Terminal 4: Run Prover
# Usage: make run-prover [CONTRACT=...] [KEY=...]
#   If CONTRACT/KEY are omitted, they are auto-loaded from .tan-zk/deployments.json and .tan-zk/secrets.env
run-prover: build-prover
	@echo "🔐 Starting Prover..."
	@./build/prover-bin start --keys-dir ~/.tan-zk/keys $(if $(CONTRACT),--contract $(CONTRACT),) $(if $(KEY),--private-key $(KEY),)

# --- Help ---

help:
	@echo "Tan-ZK Makefile Commands (in execution order):"
	@echo "  make clean           - 1. Clear artifacts"
	@echo "  make clean-global    - 1b. Clear global artifacts (~/.tan-zk)"
	@echo "  make setup           - 2. Generate keys & verifier"
	@echo "  make build           - 3. Build all binaries"
	@echo "  make test            - 4. Run all tests"
	@echo ""
	@echo "Run Workflow:"
	@echo "  make anvil           - 5. Start chain (Term 1)"
	@echo "  make deploy          - 6. Deploy contracts (Term 2)"
	@echo "  make run-sequencer   - 7. Start sequencer (Term 3)"
	@echo "  make run-prover      - 8. Start prover (Term 4)"
	@echo "  make check-batch     - 9. Check latest batch info"
