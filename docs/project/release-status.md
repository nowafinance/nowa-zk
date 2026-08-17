# Release Status

What's actually been released, what's built but unreleased, and what's genuinely
incomplete — kept separate from `roadmap-marketing.md`/`roadmap-technical.md` (which
track phases going forward) because this tracks what's *shipped* right now.

*Last verified against code and GitHub Releases: 2026-08-17.*

---

## 1. Published releases

| Tag | Title | Contents |
|---|---|---|
| `v0.0.1` | Initial Alpha | Initial project structure only — no feature list. |
| `v0.0.2` | Genesis Release | First working system: old-model `nowa-sequencer` (the Cosmos indexer, not today's Sequencer), Gnark prover, `StateManager`/`BatchRegistry`/`VerifierAdapter` contracts, systemd deployment docs. |
| `v0.1.0` | End-to-End App-Specific ZK Rollup & MiMC Integration | Pivoted from generic rollup to trade-specific. MiMC hashing, `TradeRegistry.sol` (replacing `BatchRegistry`), Foundry E2E tests, automated trusted setup. |
| `v0.2.0` | L1 Data Availability & ZK Trade Signatures | **Currently "Latest" on GitHub.** Closes Phase 1+2: L1 DA (raw payloads on-chain), finalized MiMC trade signature circuit, `TradeRegistry.sol` + `Verifier.sol` deployed, modular circuits + fuzz tests. |

`v0.0.3` is a bare git tag between `v0.0.2` and `v0.1.0` — never published as a GitHub
Release, no notes.

> [!WARNING]
> **`v0.2.0` is stale.** Its own release notes end with *"What's Next: Phase 3
> (Universal State Transition) & Phase 4 (Off-Chain Sequencer)"* — both are done in
> code (see §2) but were never released. Anyone installing "the latest release" today
> gets the old Cosmos-indexer/`TradeRegistry` architecture, not what's actually in
> `main`.

---

## 2. Shipped but unreleased (since `v0.2.0`)

```
5baa928  feat(phase-3): integrate Universal State Circuit with Sparse Merkle Trees
a05a620  feat(sequencer): migrate to EdDSA, implement LevelDB SMT, high-frequency matching engine
0aaf75a  feat: implement dual-token swaps and L1 deposit watcher for production ZK-rollup
5fbf3dc  test: implement exhaustive production-grade test suite and edge case handling
```
Plus this round of uncommitted work (docs overhaul, an EIP-4844 post-Osaka DA fix,
`test_client.go` rewrite) — not yet committed, let alone released. See
[Suggested next release](#4-suggested-next-release) below.

---

## 3. Completed vs. incomplete, as of today

### ✅ Completed and verified working

- **Sequencer** (`sequencer/`) — order matching (price/time priority, partial fills),
  LevelDB Sparse Merkle Tree (depth 28), REST API, EdDSA (BabyJubJub) signatures.
- **Universal State Transition Circuit** — one circuit handles Trade / Transfer /
  Withdrawal / Deposit via Merkle inclusion proofs.
- **Prover** — Groth16 (BN254) proof generation + local verification. Watched live:
  witness solve → prove (~1-2s) → verify → submit, end to end.
- **L1 settlement** (`NowaRollup.sol`) — Groth16 proof verification, `stateRoot`
  tracking, batch counting. Watched a real batch settle on Sepolia this session.
- **Data Availability** — EIP-4844 blobs, cell-proof format (`BlobSidecarVersion1`,
  fixed this session for post-Osaka compatibility). Enforced on-chain
  (`submitBatch()` requires a blob to be present) — not optional.
- **L1 → L2 deposits** — `deposit()` + a live Deposit Watcher goroutine, automatic,
  no manual relay needed.
- **Dual-token swaps** — trades aren't limited to one fixed pair.
- **Test coverage** — matching engine, SMT, batching, full trade application, a real
  Groth16 prove/verify cycle, and a `--count N` live-integration tool
  (`sequencer/cmd/cli/test_client.go`) that stress-tests batching end to end.
- **Documentation** — rewritten this session to match the Sequencer-centric
  architecture (previous docs described the retired Cosmos-indexer design).

### ❌ Not done — real gaps, not roadmap aspirations

- **No escape hatch — the single biggest gap.** `NowaRollup.withdraw()` is
  `onlyOwner` and ignores its Merkle-proof parameter. Not "no emergency exit" —
  **no user-initiated withdrawal at all**, ever, under any condition. 100% of
  deposited funds' recovery currently depends on one private key. See
  [architecture/overview.md](../architecture/overview.md#known-gaps-as-of-2026-08-17).
- **`stateRoot` bootstrap isn't automated.** A fresh `NowaRollup` starts at
  `stateRoot = 0`; no real Sequencer tree (empty or not) ever roots to `0`. Must be
  manually synced via `setStateRoot()` after every deploy — undocumented until this
  session, and we hit the revert live. See
  [operations/troubleshooting.md](../operations/troubleshooting.md).
- **Account onboarding isn't a tracked state transition.** `GET /account` lab-credits
  a new pubkey by writing directly to the Merkle tree, invisible to the batcher — a
  new account appearing between two submitted batches permanently breaks the chain
  for the later one. Hit live: batch #1 settled, batch #2 never could. Workaround
  shipped (`test_client.go --count`/lock file); underlying Sequencer gap remains.
- **Batch size is 1**, not the 25/128 in `BENCHMARKS.md` or the recursive-proving plan
  in `FAQ-ZK.md` — every fill is its own proof and its own L1 transaction.
- **No WebSocket API** — order placement and orderbook streaming are HTTP-only.
- **Generated contract bindings are stale** — `sequencer/internal/bindings/nowa_rollup.go`
  is missing `SetStateRoot`/`SetProver` and its `SubmitBatch` signature lacks the
  `_dataHash` parameter the real contract has. Not on the live path (the Prover packs
  calldata manually instead), but misleading if anyone reaches for these bindings.
- **`prover/internal/api/server.go` is dead code** — a Fiber API bound to a legacy
  `BatchRegistry` contract, never invoked by `prover/cmd/prover` (the actual binary).
- **Indexer (`indexer/`) is legacy/optional**, off the live trading path entirely —
  kept buildable, not maintained against the current architecture.

---

## 4. Suggested next release

**`v0.4.0`** — bundles Phase 3 (SMT) + Phase 4 (Sequencer), since the SMT circuit was
never independently shippable on its own; it only became a real system once wired
into the Sequencer. Skip a standalone `v0.3.0` for that reason.

Draft description:

```markdown
## v0.4.0: The High-Frequency Sequencer & Universal State Transition

Replaces the Cosmos-indexer execution model with a real off-chain matching engine,
and upgrades the circuit from a stateless trade verifier into a full stateful
rollup engine.

### ✨ What's New
- **Sequencer** (new component): in-memory matching engine, persistent depth-28
  Sparse Merkle Tree, REST API, EdDSA signatures (replacing ECDSA/EIP-712).
- **Universal State Transition Circuit**: one circuit now handles Trade, Transfer,
  Withdrawal, and Deposit. Contract renamed TradeRegistry → NowaRollup.
- **L1 Bridge (partial)**: deposit() + a live Deposit Watcher, dual-token swaps.
- **Data Availability**: EIP-4844 blobs with KZG cell proofs (post-Osaka compatible).
- **Hardening**: exhaustive test suite, full documentation rewrite.

### ⚠️ Known Gaps (documented, not hidden)
- No escape hatch — withdraw() is owner-gated, not trustless yet.
- Batch size is 1 — one proof per fill.
- No WebSocket API.

### Upgrade Notes
Circuit changed → requires new keys and a fresh contract deployment
(`make setup && make deploy`) — old deployments cannot verify proofs from this circuit.
```

Before tagging: commit the current working-tree changes (docs, the DA fix,
`test_client.go`) first — a release should point at a committed state.

## Related
- [Marketing Roadmap](./roadmap-marketing.md) — forward-looking phase tracking
- [Technical Roadmap](./roadmap-technical.md) — internal engineering detail
- [Architecture Overview](../architecture/overview.md) — component-level known gaps
