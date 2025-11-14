# ZK-Sequencer Roadmap

## Snapshot

- Goal: end-to-end sequencer + prover for the Tan-ZK L1 network.
- Repo: https://github.com/tannetwork/tan-zk
- Status: Phase 1 and Phase 2 complete; Phase 3 (ZK prover stack) is next.
- Architecture: read-only sequencer (follows L1), batches transactions, prepares data for prover, submits proof + calldata back to Tan-ZK.

---

## Completed Work

### Phase 1 — Foundation & Infrastructure
- Repository scaffold, CI, docs, tooling.
- Foundry contracts: BatchRegistry, StateManager, IVerifier interface, deploy scripts, unit/fuzz tests.
- Outcome: repo ready for contributors; contracts compile & test cleanly.

### Phase 2 — Sequencer Core
- Tan-ZK RPC client (JSON-RPC/WebSocket, block streaming, historical balance lookup).
- Sequencer service: direct block processing, incremental batch builder, Badger persistence, SMT placeholder, reorg handling, restart safety.
- REST API + WebSocket: /status, /batch/:n, /batch/latest, /metrics, /state/root, batch notifications.
- Structured logging, error package, Cobra CLI (`sequencer start --reset`), .env config loader.
- Outcome: sequencer can run continuously, recover from restarts, and expose status endpoints.

---

## Phase 3 — Zero-Knowledge Prover (Next Focus)

| Milestone | Objective | Key Tasks | Acceptance |
| --- | --- | --- | --- |
| 3.1 Gnark Project Setup | Bootstrap prover repo + workflow | Init Go module, add sample tx circuit, integrate gnark tests, script proving/verifying for fixture data, define key-generation + Solidity export plan | Local proofs succeed for fixture txs; CI job runs gnark tests |
| 3.2 Batch Circuit (100 txs) | Chain 100 tx executions with intermediate roots | Encode state transitions, integrate SMT root inputs, optimize field operations, produce proving/verifying keys | Proof completes <2 min for 100 txs with deterministic state root |
| 3.3 Solidity Verifier Export | Deploy verifier + connect to contracts | Export verifier from vk, benchmark gas (<500k), deploy to Tan-ZK devnet, hook into BatchRegistry | On-chain verification passes for sample proofs |
| 3.4 Prover HTTP Service | Externalize proving workflow | REST API (POST /prove, GET /status/:id, GET /proof/:id), worker queue, basic persistence, metrics/health | Sequencer ↔ Prover flow reliable under load, proofs retrievable via API |

Open design items before Phase 3 closes:
- Circuit spec for batch witness format (link once ready).
- Interface contract between sequencer batches and prover API (JSON schema). 

---

## Future Phases (High-Level until Phase 3 lands)

### Phase 4 — Integration & Testing
- 4.1 Local Integration: Docker Compose stack, automated E2E script, performance benchmarks.
- 4.2 Tan-ZK End-to-End: run against Tan-ZK devnet, submit ≥1000 real txs, generate/verify proof on-chain, scripted flow.

### Phase 5 — Production Readiness
- 5.1 Monitoring & Observability: Prometheus metrics, Grafana dashboards, alerting, structured logs.
- 5.2 Documentation & API Specs: docs/ revamp (getting-started, ops, troubleshooting), validated OpenAPI specs, contributor-tested quickstart.
- 5.3 Docker & CI/CD: hardened Dockerfiles, GH Actions for build/test/release, registry publishing, compose stack.
- 5.4 Security & Hardening: contract + service audits, rate limiting, key management guidelines, remediation of findings.

Detailed task lists for Phases 4-5 will be expanded after Phase 3 milestones are underway.

---

## Tracking & References

- Milestone source of truth: `docs/milestone.md` (mirrors GitHub milestones).
- Issues/PRs: every milestone has summary issue(s) linked for historical context.
- Progress cadence: update roadmap + milestone doc whenever a milestone starts or finishes.
- Next review checkpoint: after Milestone 3.1 lands, reassess timelines for 3.2–3.4 and prepare Phase 4 breakdown.

---

## Notes

- Transaction pool milestone (Phase 2.1) intentionally skipped in current read-only architecture; will revisit when sequencer accepts direct user txs.
- SMT implementation is a placeholder sufficient for determinism; production-grade SMT will arrive alongside Phase 3 circuit work.
- Reorg handling is "simple" (single-fork rollback); advanced fork-choice logic can be scheduled post Phase 3 if Tan-ZK consensus requires.
