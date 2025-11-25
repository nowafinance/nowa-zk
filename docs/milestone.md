# Project Milestones - ZK Sequencer

## Phase 1: Foundation & Infrastructure

### Milestone 1.1 — Repository Structure ✅
- **Status**: COMPLETED
- **Due date**: 2025-11-10
- **Description**: Initialize repo layout, Hardhat scaffolding, CI, and baseline docs.
- **Scope**:
  - Create dirs: contracts, sequencer, prover, docs, docker, scripts, .github/workflows
  - Initialize Hardhat skeleton in contracts/
  - Add CI (contracts, sequencer, prover)
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
  - Implement BatchRegistry, StateManager, IVerifier interface
  - Unit tests (Forge tests with fuzzing)
  - script/Deploy.s.sol for localhost
- **Acceptance**:
  - Contracts compile without warnings
  - Core functions tested with forge test
  - 100% coverage on critical functions

### Milestone 1.3 — Tan-ZK RPC Client ✅
- **Status**: COMPLETED
- **Due date**: TBD
- **Description**: Implement Go RPC client for Tan-ZK blockchain.
- **Scope**:
  - JSON-RPC/WebSocket connection to Tan-ZK node
  - Subscribe to new blocks, fetch transactions
  - Query account state
  - Stub SubmitBatchProof method
  - Basic integration tests with local node
- **Acceptance**:
  - Reliable block streaming and transaction fetch with retries
  - Integration tests pass with local Tan-ZK node
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

## Phase 2: Sequencer Core

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

### Milestone 2.3 — Sequencer Service ✅
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

### Milestone 3.3 — Solidity Verifier Export
- **Status**: PLANNED
- **Due date**: TBD
- **Description**: Export and integrate verifier contract.
- **Scope**:
  - Export Solidity verifier from verification key
  - Deploy verifier contract
  - Wire into BatchRegistry
  - Move generated Verifier.sol to contracts/src/generated/
  - Gas optimization for verification
- **Acceptance**:
  - On-chain verification passes for sample proofs
  - Gas costs documented (target: < 500k gas)

### Milestone 3.4 — Prover HTTP Service
- **Status**: PLANNED
- **Due date**: TBD
- **Description**: Prover API and worker queue.
- **Scope**:
  - POST /prove, GET /status/:id, GET /proof/:id
  - Metrics and health endpoints
  - Concurrent workers
  - Basic persistence
- **Acceptance**:
  - Sequencer → Prover request/response flow reliable

---

## Phase 4: Integration & Testing

### Milestone 4.1 — Local Integration Testing
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

### Milestone 4.2 — End-to-End Integration with Tan-ZK
- **Status**: PLANNED
- **Due date**: TBD
- **Description**: Full flow on local Tan-ZK devnet.
- **Scope**:
  - Clone and start local Tan-ZK node (separate repo)
  - Deploy contracts to Tan-ZK EVM using Foundry
  - Configure sequencer to connect to Tan-ZK RPC
  - Run sequencer+prover; submit 1000 real txs to Tan-ZK
  - Generate proof and verify on Tan-ZK chain
  - scripts/test-e2e.sh to automate entire flow
- **Acceptance**:
  - Script completes end-to-end without manual intervention
  - Batch verified on-chain on Tan-ZK
  - State roots match between sequencer and Tan-ZK

---

## Phase 5: Production Readiness

### Milestone 5.1 — Monitoring & Observability
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

### Milestone 5.2 — Documentation & API Specs
- **Status**: PLANNED
- **Due date**: TBD
- **Description**: Dev/architecture/API/ops docs + OpenAPI.
- **Scope**:
  - docs/ structure complete (getting-started, architecture, api, ops)
  - Quickstart guide tested by external contributor
  - OpenAPI 3.0 spec for sequencer and prover APIs
  - Configuration reference with all env vars
  - Troubleshooting common issues
  - Architecture diagrams (system, data flow, components)
- **Acceptance**:
  - New contributor can run E2E locally using only docs
  - OpenAPI spec validates and generates client SDKs
  - All API endpoints documented with examples

### Milestone 5.3 — Docker & CI/CD
- **Status**: PLANNED
- **Due date**: TBD
- **Description**: Production containers and automation.
- **Scope**:
  - Dockerfile.sequencer (multi-stage, optimized)
  - Dockerfile.prover (multi-stage, with keys)
  - docker-compose.yml (all services + monitoring)
  - GitHub Actions for build/test/release
  - Container registry publishing
- **Acceptance**:
  - docker-compose up runs full stack
  - CI builds and pushes images on merge
  - Release workflow creates tagged builds

### Milestone 5.4 — Security Audit & Hardening
- **Status**: PLANNED
- **Due date**: TBD
- **Description**: Security review and production hardening.
- **Scope**:
  - Third-party smart contract audit
  - Security review of sequencer/prover
  - Rate limiting and DDoS protection
  - Key management best practices
- **Acceptance**:
  - Audit report completed
  - All critical/high findings resolved
  - Security documentation complete

---

## Progress Tracking

**Completed**: 9/20+ milestones (45%)
- ✅ 1.1 Repository Structure
- ✅ 1.2 Smart Contract Foundation
- ✅ 1.3 Tan-ZK RPC Client
- ✅ 1.4 State Synchronization
- ✅ 2.1 Transaction Pool
- ✅ 2.2 Batch Builder
- ✅ 2.3 Sequencer Service
- ✅ 2.4 REST API
- ✅ 3.1 Gnark Project Setup

**In Progress**: 1 milestone
- 🔄 3.2 Batch Circuit (100 → 1000 txs)

**Remaining**: 10 milestones