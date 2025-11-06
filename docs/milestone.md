Milestone 1.1 — Repository Structure
- Due date: 2025-11-10
- Description: Initialize repo layout, Hardhat scaffolding, CI, and baseline docs.
- Scope:
  - Create dirs: contracts, sequencer, prover, docs, docker, scripts, .github/workflows
  - Initialize Hardhat skeleton in contracts/
  - Add CI (contracts, sequencer, prover)
  - Update README and ROADMAP
- Acceptance:
  - CI jobs green
  - contracts compiles; Go modules build

Milestone 1.2 — Smart Contract Foundation (Hardhat)
- Due date: 2025-11-14
- Description: Core contracts with tests and local deploy.
- Scope:
  - Implement BatchRegistry, StateManager, IVerifier interface
  - Unit tests (Hardhat) 
  - scripts/deploy.ts for localhost
- Acceptance:
  - Contracts compile without warnings
  - Core functions tested locally

Milestone 1.3 — Tan‑ZK RPC Client
- Description: Implement Go RPC client for Tan‑ZK.
- Scope:
  - JSON‑RPC/WebSocket connection
  - Subscribe new blocks, fetch txs
  - Query account state; stub SubmitBatchProof
  - Basic integration tests with local node
- Acceptance:
  - Reliable block streaming and tx fetch with retries

Milestone 1.4 — State Synchronization
- Description: Basic state tracking with persistence.
- Scope:
  - LevelDB/BadgerDB persistence
  - Sparse Merkle Tree placeholder
  - Current state root calculation
  - Simple reorg handling
- Acceptance:
  - Deterministic state root on replay; reorg tests pass

Milestone 2.1 — Transaction Pool
- Description: In‑memory pool with validation and ordering.
- Scope:
  - Validation, nonce ordering, priority queue, limits/eviction
  - Metrics for pool size/throughput
- Acceptance:
  - Unit tests for add/evict/order; metrics exposed

Milestone 2.2 — Batch Builder
- Description: Build batches and metadata.
- Scope:
  - Select N txs, compute batch hash/metadata
  - Simulate state transitions; execution traces
  - Store batches locally
- Acceptance:
  - Deterministic batch hash; persisted batches

Milestone 2.3 — Sequencer Service
- Description: Main loop and lifecycle.
- Scope:
  - Ingest blocks → txpool
  - Periodic batch creation; request proof from prover
  - Proof submitter stub; config; graceful shutdown
- Acceptance:
  - Service stable for extended local runs

Milestone 2.4 — REST API
- Description: Read‑only API for status/inspection.
- Scope:
  - Endpoints: status, batch/:number, batch/latest, metrics, state/root
  - OpenAPI draft
- Acceptance:
  - Valid JSON responses; basic load test OK

Milestone 3.1 — Gnark Project Setup
- Description: Prover boot, simple circuit, keys flow.
- Scope:
  - Gnark setup; simple tx circuit
  - Local proof generation/verification
  - Plan Solidity verifier export path
- Acceptance:
  - Proofs generate for sample inputs; tests pass

Milestone 3.2 — Batch Circuit (100 txs)
- Description: Scale circuit with chained state updates.
- Scope:
  - Process 100 txs; intermediate roots
  - Optimize hotspots; generate pk/vk
- Acceptance:
  - Proof within target time for 100 txs

Milestone 3.3 — Solidity Verifier Export
- Description: Export and integrate verifier contract.
- Scope:
  - Export Solidity verifier from vk
  - Deploy verifier; wire into BatchRegistry
- Acceptance:
  - On‑chain verification passes for sample proofs

Milestone 3.4 — Prover HTTP Service
- Description: Prover API and worker queue.
- Scope:
  - POST /prove, GET /status/:id, GET /proof/:id, metrics, health
  - Concurrent workers; basic persistence
- Acceptance:
  - Sequencer → Prover request/response flow reliable

---

Milestone 4.1 — Circuit Optimization (towards 1000 txs)
- Description: Optimize constraints and runtime.
- Scope:
  - Batch ECDSA, Merkle proof reuse, profiling
- Acceptance:
  - Measurable performance gains vs baseline

Milestone 4.2 — End‑to‑End Integration
- Description: Full flow on local/devnet.
- Scope:
  - Start local Tan‑ZK; deploy contracts (Hardhat)
  - Run sequencer+prover; generate/verify proofs
  - scripts/test-e2e.sh to automate
- Acceptance:
  - Script completes; batch verified on‑chain

Milestone 4.3 — Performance & Monitoring
- Description: Metrics, dashboards, and tuning.
- Scope:
  - Prometheus metrics; Grafana dashboards
  - Throughput/latency bottleneck identification
- Acceptance:
  - Dashboards usable; perf targets tracked

Milestone 5.1 — Testing Hardening
- Description: Unit, integration, load, chaos.
- Scope:
  - ≥90% unit coverage core paths
  - E2E, failure recovery, high‑load tests
- Acceptance:
  - CI green; flakiness addressed

Milestone 5.2 — Documentation
- Description: Dev/architecture/API/ops docs.
- Scope:
  - docs/ filled; quickstart; configuration; troubleshooting
  - OpenAPI published in repo
- Acceptance:
  - New contributor can run E2E locally using docs