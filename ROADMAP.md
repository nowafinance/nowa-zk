# Nowa-ZK Roadmap

## Snapshot
- **Goal**: End-to-end ZK Sequencer and Prover for an App-Specific Orderbook/Validium.
- **Architecture**: Trades execute and store DA on a Cosmos-Ethereum chain. The Indexer batches trades, the Prover generates ZK proofs, and L1 verification happens on Ethereum (Sepolia).
- **Status**: Core ZK Circuit and L1 Settlement layers are complete.

---

## Phase 1 — Core ZK Circuit & Sequencer (Completed)
- Built the foundational ZK-SNARK circuit using Groth16 (`gnark`).
- Validates EIP-712 signatures and multiple trade executions.
- Designed the Indexer daemon to fetch and chunk trades from the Cosmos execution layer into fixed batch sizes.

## Phase 2 — L1 Settlement & Verification (Completed)
- Designed `TradeRegistry.sol` and auto-generated `TradeVerifier.sol`.
- Submits `proof` and `publicInputs` to Ethereum to act as the ultimate cryptographic anchor.
- Completed integration testing of the Go Prover directly interacting with L1 Smart Contracts.

## Phase 3 — Public Input Hashing & Scalability (Planned / Next)
- **Goal:** Scale the maximum trades per batch from 25 to 1000+ without hitting the EIP-170 (24 KB) Ethereum contract size limit.
- **Task:** Implement Fiat-Shamir / Public Input Hashing in the `gnark` circuit (compressing all trade inputs into a single `ExpectedHash` public input).
- **Task:** Update `TradeRegistry.sol` to perform `keccak256` on calldata trades and verify the single hash against the ZK Proof.

## Phase 4 — L1-L2 Bridge & Vault Contracts (Planned)
- **Goal:** Enable users to securely bridge real assets (like USDC) into the ZK trading engine.
- **Task:** Build L1 Deposit Vaults that map to L2 balances.
- **Task:** Build Trustless ZK-Withdrawals (Ethereum allows withdrawals based on the proven state root).

## Phase 5 — Production & Decentralization
- **5.1 Monitoring & Observability:** Prometheus metrics, Grafana dashboards, alerting, structured logs.
- **5.2 Docker & CI/CD:** Hardened Dockerfiles, GH Actions for build/test/release.
- **5.3 Security & Hardening:** Contract and circuit audits, rate limiting, and key management guidelines.
- **5.4 Decentralized Provers:** Transition from a single centralized prover to a permissionless proving network.
