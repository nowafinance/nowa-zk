# Nowa-ZK System Overview

## Architecture Diagram

```text
┌─────────────────────────────────────────────────────────────┐
│                         L1 (Ethereum)                        │
│  ┌────────────────────────────────────────────────────────┐ │
│  │              NowaRollup Smart Contract                  │ │
│  │  • deposit() — locks ERC20, credits L2 balance           │ │
│  │  • submitBatch() — verifies Groth16 proof + EIP-4844     │ │
│  │    blob DA, advances stateRoot                           │ │
│  │  • withdraw() — operator-gated placeholder (NOT an       │ │
│  │    escape hatch yet — see Known Gaps below)              │ │
│  └────────────────────────────────────────────────────────┘ │
└─────────────────────────────────────────────────────────────┘
                    ▲                          │
                    │ submitBatch()             │ Deposit event
                    │ (proof + blob)            ▼
┌───────────────────┴──────────┐   ┌───────────────────────────┐
│           Prover              │   │      Deposit Watcher       │
│  • Fetches sealed batches      │   │  (goroutine inside the     │
│    from the Sequencer's REST   │   │   Sequencer binary)        │
│    API (:8080)                 │   │  • Watches NowaRollup's    │
│  • Generates a Groth16 proof   │   │    Deposit event           │
│    (gnark, BN254)              │   │  • Mints the deposit into  │
│  • Builds an EIP-4844 blob     │   │    the L2 Merkle tree as   │
│    sidecar (cell proofs,       │   │    an OpDeposit transition │
│    post-Osaka format)          │   └─────────────┬─────────────┘
│  • Submits proof + blob to L1  │                 │
└───────────────┬────────────────┘                 │
                 │ GET /batch/latest, /batch/:id    │
                 ▼                                  ▼
┌─────────────────────────────────────────────────────────────┐
│                          Sequencer                            │
│  • In-memory limit-order matching engine (per-token order-    │
│    book, price/time priority) — sequencer/internal/engine     │
│  • LevelDB Sparse Merkle Tree (depth 28) of account state —   │
│    sequencer/internal/state                                   │
│  • Batcher seals matched fills into ZK batches for the        │
│    Prover — sequencer/internal/batcher                        │
│  • REST API on :8080 — sequencer/internal/api                 │
│  • Deposit Watcher — sequencer/cmd/sequencer/deposit_watcher.go│
└─────────────────────────────────────────────────────────────┘
                             ▲
                             │ HTTP orders (EdDSA-signed)
                             │
                       User / Frontend
```

## Component Responsibilities

### Sequencer (`sequencer/`) — the execution layer
- **Purpose**: Matches orders and maintains the canonical L2 account state off-chain.
- **Matching**: In-memory orderbook per token (`sequencer/internal/engine/matching.go`), price/time priority, supports partial fills.
- **State**: A depth-28 Sparse Merkle Tree persisted in LevelDB (`sequencer/internal/state/merkle_db.go`). Leaf index = `accountID*256 + tokenID`.
- **Batching**: Every matched fill becomes a `StateTransition` fed to the `Batcher` (`sequencer/internal/batcher`). Current `BatchSize = 1` — one real fill per proof, no dummy padding.
- **Signatures**: Two independent EdDSA (BabyJubJub/BN254) signatures per order — an "order intent" signature the matching engine checks (SHA256-based), and a separate "circuit signature" over `MiMC(OpType, pubX, pubY, baseIndex, quoteIndex)`, which is what the ZK circuit actually verifies. See `sequencer/cmd/cli/test_client.go` for a worked example of producing both.
- **Deposits**: `deposit_watcher.go` subscribes to the `NowaRollup` contract's `Deposit` event on L1 and mints the corresponding `OpDeposit` transition into the tree — set `ROLLUP_CONTRACT_ADDRESS` to enable it (`make run-sequencer` auto-loads it from `~/.nowa-zk/deployments.json`).
- **API**: Plain `net/http` on `:8080` — `POST /order`, `GET /orderbook`, `/balance`, `/account`, `/batch/latest`, `/batch/count`, `/batch/:id`. No WebSocket streaming yet (roadmap item).

### Prover (`prover/`)
- **Purpose**: Turns sealed Sequencer batches into a Groth16 proof and settles them on L1.
- **Circuit**: `prover/circuits/state_circuit.go` — a single `StateTransitionCircuit` that handles all four op types (Trade / Transfer / Withdrawal / Deposit) via Merkle inclusion proofs against the depth-28 SMT.
- **Proof system**: Groth16 on BN254 (`gnark`). Keys live in `~/.nowa-zk/keys/` (produced by `make setup` / `prover setup`) — **not** `prover/keys/` in the repo, which is git-ignored and may hold stale local artifacts.
- **Data Availability**: Builds an EIP-4844 blob sidecar per batch (`prover/internal/da/blob.go`) using KZG **cell proofs** (`BlobSidecarVersion1`) — required post-Osaka; the old single-proof `BlobSidecarVersion0` format is rejected by current L1 nodes.
- **Submission**: Signs and sends `submitBatch(proof, oldRoot, newRoot, withdrawalHash, depositHash, dataHash)` as an EIP-4844 blob transaction to `NowaRollup`.

### L1 Smart Contracts (`contracts/`) — Ethereum Sepolia
- **`NowaRollup.sol`**: token registry (`registerToken`), `deposit`, `submitBatch` (Groth16 verification + blob DA requirement via `blobhash(0)`), and a placeholder `withdraw` (operator-gated — see Known Gaps).
- **`generated/Verifier.sol`**: Groth16 verifier, auto-generated from the circuit's verifying key by `prover setup` / `make setup`. **Regenerating it (because the circuit changed) requires redeploying `NowaRollup` too** — the verifying key is baked into the deployed bytecode and the two must match.
- Deployed addresses live in `contracts/deployments/deployments.json` (per-run, via `forge script`) and are copied to `~/.nowa-zk/deployments.json`, which the Sequencer and Prover both auto-load from.

### Indexer (`indexer/`) — legacy, optional
An earlier design ran trade execution on a Cosmos-SDK EVM chain, with this Indexer polling blocks and building 25-trade batches for the Prover. That model has been replaced by the Sequencer above (see the Makefile's own `run-indexer` help text: "optional / legacy L2 indexing"). The Sequencer does **not** call the Indexer, and the Prover talks directly to the Sequencer's REST API. `indexer/` still builds and has tests, but it's off the path real trades take today. See the archived [indexer-batch-flow-legacy.md](../archived-files/indexer-batch-flow-legacy.md) and [cleanup-system-legacy-indexer.md](../archived-files/cleanup-system-legacy-indexer.md) if you need its internals.

## Data Flow Summary

1. **User** → EdDSA-signs and `POST /order`s to the **Sequencer**.
2. **Sequencer** → matches the order; on a fill, updates the Merkle tree and seals a batch (`BatchSize = 1`).
3. **Prover** → polls `GET /batch/latest`, generates a Groth16 proof + EIP-4844 blob, submits `submitBatch()` to L1.
4. **L1 (`NowaRollup`)** → verifies the proof, checks the blob is present, advances `stateRoot`.
5. **Deposit Watcher** (inside the Sequencer) → watches `NowaRollup`'s `Deposit` event and mints the deposit into the L2 tree as the next batch's first transition.

## Known Gaps (as of 2026-08-17)

See [Release Status](../project/release-status.md) for how these gaps line up against
what's actually been published as a release.

- **No escape hatch.** `NowaRollup.withdraw()` is explicitly a placeholder — `onlyOwner`, ignores the Merkle proof parameter. If the Sequencer disappears, there is currently no on-chain path for a user to reclaim funds with a Merkle proof of their L2 balance. This is the single biggest gap between the documented design intent (`FAQ-ZK.md` §7) and the current code.
- **Account onboarding isn't a tracked state transition.** `GET /account`/`/balance` (and the first-trade path inside `applyTrade`) lab-credit a new pubkey by writing directly to the Merkle tree (`sequencer/internal/api/account.go`'s `openBalance`) with no corresponding `StateTransition` — invisible to the batcher and Prover. If a new account gets onboarded *between* two batches that both get submitted, the later batch's `old_root` silently stops matching the earlier one's `new_root`, and it becomes permanently unsubmittable (no recovery once `batchCount > 0` — `setStateRoot()` only works pre-genesis). We hit this live: batch #1 settled, a second `test_client.go` run's batch #2 never could. `cmd/cli/test_client.go` now works around this by registering its accounts exactly once and refusing to run twice (see [testing.md](../testing.md#stress-testing-with-many-trades---count)), but the underlying gap is in the Sequencer itself — lab-credits (and likely any account's very first appearance in general) should really be their own tracked op type, the same way `OpDeposit` is.
- **Batch size is 1**, not the 25/128 described in `BENCHMARKS.md` or the recursive-proving roadmap in `FAQ-ZK.md` — every fill gets its own proof and its own L1 submission today.
- **No WebSocket API** on the Sequencer — order placement and orderbook streaming are HTTP-only.

## Technology Stack

| Component | Technologies |
|-----------|-------------|
| Sequencer | Go, LevelDB (Sparse Merkle Tree), `net/http` |
| Prover | Go, `gnark` (Groth16, BN254), BadgerDB (submission checkpointing) |
| Smart Contracts | Solidity, Foundry |
| ZK Circuit | `gnark` constraint system, MiMC hashing, EdDSA (BabyJubJub) signatures |
| Indexer (legacy) | Go, BadgerDB, Fiber API |
