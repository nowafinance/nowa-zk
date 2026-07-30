# Project Milestones - ZK Indexer

## Phase 1: Foundation & Infrastructure

### Milestone 1.1 — Repository Structure ✅
- **Status**: COMPLETED
- **Due date**: 2025-11-10
- **Description**: Initialize repo layout, Hardhat scaffolding, CI, and baseline docs.
- **Scope**:
  - Create dirs: contracts, indexer, prover, docs, docker, scripts, .github/workflows
  - Initialize Hardhat skeleton in contracts/
  - Add CI (contracts, indexer, prover)
  - Update README and ROADMAP
- **Acceptance**:
  - CI jobs green
  - contracts compiles; Go modules build

### Milestone 1.2 — Smart Contract Foundation (Foundry) ✅
- **Status**: COMPLETED
- **Due date**: 2025-11-14
- **Description**: Core contracts with tests and local deploy.
- **Scope**:
  - Initialize Foundry project (forge init)
  - Implement TradeRegistry, StateManager, IVerifier interface
  - Unit tests (Forge tests with fuzzing)
  - script/Deploy.s.sol for localhost
- **Acceptance**:
  - Contracts compile without warnings
  - Core functions tested with forge test
  - 100% coverage on critical functions

### Milestone 1.3 — Nowa-ZK RPC Client ✅
- **Status**: COMPLETED
- **Due date**: TBD
- **Description**: Implement Go RPC client for Nowa-ZK blockchain.
- **Scope**:
  - JSON-RPC/WebSocket connection to Nowa-ZK node
  - Subscribe to new blocks, fetch transactions
  - Query account state
  - Stub SubmitBatchProof method
  - Basic integration tests with local node
- **Acceptance**:
  - Reliable block streaming and transaction fetch with retries
  - Integration tests pass with local Nowa-ZK node
  - Error handling and reconnection logic implemented

### Milestone 1.4 — State Synchronization ✅
- **Status**: COMPLETED
- **Due date**: TBD
- **Description**: Basic state tracking with persistence.
- **Scope**:
  - LevelDB/BadgerDB persistence
  - Sparse Merkle Tree placeholder
  - Current state root calculation
  - Simple reorg handling
- **Acceptance**:
  - Deterministic state root on replay
  - Reorg tests pass

---

## Phase 2: Indexer Core

### Milestone 2.1 — Transaction Pool ✅
- **Status**: COMPLETED
- **Due date**: TBD
- **Description**: In-memory pool with validation and ordering.
- **Scope**:
  - Validation, nonce ordering, priority queue
  - Limits and eviction policies
  - Metrics for pool size/throughput
- **Acceptance**:
  - Unit tests for add/evict/order
  - Metrics exposed

### Milestone 2.2 — Batch Builder ✅
- **Status**: COMPLETED
- **Due date**: TBD
- **Description**: Build batches, metadata, and prover inputs.
- **Scope**:
  - Select N txs, compute batch hash/metadata
  - Simulate state transitions; generate execution traces
  - Format traces for Gnark circuit (witness data)
  - Store batches locally with traces
- **Acceptance**:
  - Deterministic batch hash; persisted batches
  - Execution traces compatible with circuit witness format

### Milestone 2.3 — Indexer Service ✅
- **Status**: COMPLETED
- **Due date**: TBD
- **Description**: Main loop and lifecycle management.
- **Scope**:
  - Ingest blocks → transaction pool
  - Periodic batch creation
  - Request proof from prover
  - Proof submitter stub
  - Configuration management
  - Graceful shutdown
- **Acceptance**:
  - Service stable for extended local runs
  - Clean shutdown on SIGTERM/SIGINT

### Milestone 2.4 — REST API ✅
- **Status**: COMPLETED
- **Due date**: TBD
- **Description**: Read-only API for status/inspection.
- **Scope**:
  - Endpoints: status, batch/:number, batch/latest, metrics, state/root
  - OpenAPI 3.0 specification
- **Acceptance**:
  - Valid JSON responses
  - Basic load test OK
  - OpenAPI spec validates

---

## Phase 3: Zero-Knowledge Prover

### Milestone 3.1 — Gnark Project Setup ✅
- **Status**: COMPLETED
- **Due date**: TBD
- **Description**: Prover bootstrap, simple circuit, keys flow.
- **Scope**:
  - Gnark setup; simple transaction circuit
  - Local proof generation/verification
  - Plan Solidity verifier export path
- **Acceptance**:
  - Proofs generate for sample inputs
  - Tests pass

### Milestone 3.2 — Batch Circuit (100 → 1000 txs)
- **Status**: IN PROGRESS
- **Due date**: TBD
- **Description**: Scale circuit with chained state updates.
- **Scope**:
  - **Phase 1**: Process 100 txs; intermediate roots (Week 5)
  - **Phase 2**: Scale to 500 txs with optimizations (Week 6)
  - **Phase 3**: Reach 1000 txs target (Week 7)
  - Optimize hotspots at each phase; regenerate pk/vk
- **Acceptance**:
  - 100 txs: < 2 min proving time
  - 500 txs: < 5 min proving time
  - 1000 txs: < 10 min proving time

### Milestone 3.3 — Prover HTTP Service
- **Status**: IN PROGRESS
- **Due date**: TBD
- **Description**: Prover API and worker queue.
- **Scope**:
  - POST /prove
  - ✅ GET /status/:id
  - ✅ GET /proof/:id
  - Metrics and health endpoints
  - Concurrent workers
  - ✅ Basic persistence
- **Acceptance**:
  - Indexer → Prover request/response flow reliable



---

## Phase 4: Integration & Testing

### Milestone 4.1 — Documentation & API Specs
- **Status**: IN PROGRESS
- **Due date**: TBD
- **Description**: Dev/architecture/API/ops docs + OpenAPI.
- **Scope**:
  - docs/ structure complete (getting-started, architecture, api, ops)
  - Quickstart guide tested by external contributor
  - ✅ Cloud setup guide (systemd)
  - OpenAPI 3.0 spec for indexer and prover APIs
  - Configuration reference with all env vars
- **Acceptance**:
  - New contributor can run E2E locally using only docs
  - All API endpoints documented with examples

### Milestone 4.2— Docker & CI/CD
- **Status**: IN PROGRESS
- **Due date**: TBD
- **Description**: Production containers and automation.
- **Scope**:
  - ✅ Unified Dockerfile (multi-stage)
  - ✅ docker-compose.yml (all services)
  - ❌ Fix CI/CD Pipeline (Currently Failing)
  - Verify Docker build on production env
- **Acceptance**:
  - docker-compose up runs full stack
  - CI builds green

### Milestone 5.1 — Local Integration Testing
- **Status**: PLANNED
- **Due date**: TBD
- **Description**: End-to-end testing on localhost.
- **Scope**:
  - Docker Compose setup for all services
  - Automated integration test suite
  - Performance benchmarks
- **Acceptance**:
  - E2E flow completes without errors
  - Performance meets targets

### Milestone 5.2 — End-to-End Integration with nowa-zk
- **Status**: PLANNED
- **Due date**: TBD
- **Description**: Full flow on local Nowa-ZK devnet.
- **Scope**:
  - Clone and start local Nowa-ZK node (separate repo)
  - Deploy contracts to Nowa-ZK EVM using Foundry
  - Configure indexer to connect to Nowa-ZK RPC
  - Run indexer+prover; submit 1000 real txs to nowa-zk
  - Generate proof and verify on Nowa-ZK chain
  - scripts/test-e2e.sh to automate entire flow
- **Acceptance**:
  - Script completes end-to-end without manual intervention
  - Batch verified on-chain on nowa-zk
  - State roots match between indexer and nowa-zk

### Milestone 5.3 — Monitoring & Observability
- **Status**: PLANNED
- **Due date**: TBD
- **Description**: Production monitoring setup.
- **Scope**:
  - Prometheus metrics
  - Grafana dashboards
  - Structured logging
  - Alerting rules
- **Acceptance**:
  - All critical metrics exposed
  - Dashboards functional
  - Alerts trigger correctly

### Milestone 5.4 — Security Audit & Hardening
- **Status**: PLANNED
- **Due date**: TBD
- **Description**: Security review and production hardening.
- **Scope**:
  - Third-party smart contract audit
  - Security review of indexer/prover
  - Rate limiting and DDoS protection
  - Key management best practices
- **Acceptance**:
  - Audit report completed
  - All critical/high findings resolved
  - Security documentation complete

---

## Progress Tracking

**Completed**: 9/19 milestones (47%)
- ✅ 1.1 Repository Structure
- ✅ 1.2 Smart Contract Foundation
- ✅ 1.3 Nowa-ZK RPC Client
- ✅ 1.4 State Synchronization
- ✅ 2.1 Transaction Pool
- ✅ 2.2 Batch Builder
- ✅ 2.3 Indexer Service
- ✅ 2.4 REST API
- ✅ 3.1 Gnark Project Setup

**In Progress**: 4 milestones
- 🔄 3.2 Batch Circuit (100 → 1000 txs)
- 🔄 3.3 Prover HTTP Service (Partial)
- 🔄 4.1 Documentation & API Specs (Cloud Guide Done)
- 🔄 4.2 Docker & CI/CD (CI Failing / Untested)

**Remaining**: 6 milestones