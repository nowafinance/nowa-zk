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

Milestone 1.2 — Smart Contract Foundation (Foundry)
- Due date: 2025-11-14
- Description: Core contracts with tests and local deploy.
- Scope:
  - Initialize Foundry project (forge init)
  - Implement BatchRegistry, StateManager, IVerifier interface
  - Unit tests (Forge tests with fuzzing)
  - script/Deploy.s.sol for localhost
- Acceptance:
  - Contracts compile without warnings
  - Core functions tested with forge test
  - 100% coverage on critical functions

Milestone 3.3.5 — Gnark Verifier Contract Integration
- Due date: [Week 6, Day 3]
- Description: Update contracts to use Gnark-generated verifier
- Scope:
  - Move generated Verifier.sol to contracts/src/generated/
  - Update BatchRegistry to import generated verifier
  - Test proof verification on-chain
  - Gas optimization for verification
- Acceptance:
  - Gnark proof verifies on-chain successfully
  - Gas costs documented (target: < 500k gas)

Milestone 2.2 — Batch Builder & Execution Traces
- Description: Build batches, metadata, and prover inputs.
- Scope:
  - Select N txs, compute batch hash/metadata
  - Simulate state transitions; generate execution traces
  - Format traces for Gnark circuit (witness data)
  - Store batches locally with traces
- Acceptance:
  - Deterministic batch hash; persisted batches
  - Execution traces compatible with circuit witness format 

Milestone 3.2 — Batch Circuit (100 → 1000 txs)
- Description: Scale circuit with chained state updates.
- Scope:
  - **Phase 1**: Process 100 txs; intermediate roots (Week 5)
  - **Phase 2**: Scale to 500 txs with optimizations (Week 6)
  - **Phase 3**: Reach 1000 txs target (Week 7)
  - Optimize hotspots at each phase; regenerate pk/vk
- Acceptance:
  - 100 txs: < 2 min proving time
  - 500 txs: < 5 min proving time
  - 1000 txs: < 10 min proving time

Milestone 4.2 — End‑to‑End Integration with Tan-ZK
- Description: Full flow on local Tan-ZK devnet.
- Scope:
  - Clone and start local Tan-ZK node (separate repo)
  - Deploy contracts to Tan-ZK EVM using Foundry
  - Configure sequencer to connect to Tan-ZK RPC
  - Run sequencer+prover; submit 1000 real txs to Tan-ZK
  - Generate proof and verify on Tan-ZK chain
  - scripts/test-e2e.sh to automate entire flow
- Acceptance:
  - Script completes end-to-end without manual intervention
  - Batch verified on-chain on Tan-ZK
  - State roots match between sequencer and Tan-ZK

Milestone 5.3 — Docker & CI/CD
- Due date: [Week 8, Day 5]
- Description: Production containers and automation.
- Scope:
  - Dockerfile.sequencer (multi-stage, optimized)
  - Dockerfile.prover (multi-stage, with keys)
  - docker-compose.yml (all services + monitoring)
  - GitHub Actions for build/test/release
  - Container registry publishing
- Acceptance:
  - docker-compose up runs full stack
  - CI builds and pushes images on merge
  - Release workflow creates tagged builds

Milestone 5.2 — Documentation & API Specs
- Description: Dev/architecture/API/ops docs + OpenAPI.
- Scope:
  - docs/ structure complete (getting-started, architecture, api, ops)
  - Quickstart guide tested by external contributor
  - OpenAPI 3.0 spec for sequencer and prover APIs
  - Configuration reference with all env vars
  - Troubleshooting common issues
  - Architecture diagrams (system, data flow, components)
- Acceptance:
  - New contributor can run E2E locally using only docs
  - OpenAPI spec validates and generates client SDKs
  - All API endpoints documented with examples