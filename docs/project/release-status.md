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
| `v0.2.0` | L1 Data Availability & ZK Trade Signatures | Closes Phase 1+2: L1 DA (raw payloads on-chain), finalized MiMC trade signature circuit, `TradeRegistry.sol` + `Verifier.sol` deployed, modular circuits + fuzz tests. |
| `v0.4.0` | The High-Frequency Sequencer & Universal State Transition | **Currently "Latest" on GitHub**, published 2026-08-17. Closes Phase 3+4: Sequencer (matching engine, LevelDB SMT, REST API, EdDSA signatures), Universal State Transition Circuit (`TradeRegistry` → `NowaRollup`), partial L1 bridge (`deposit()` + Deposit Watcher, dual-token swaps), EIP-4844 blob DA with post-Osaka cell proofs, exhaustive test suite, full docs rewrite. See §3 below for what it does *not* close. |

`v0.0.3` is a bare git tag between `v0.0.2` and `v0.1.0` — never published as a GitHub
Release, no notes. There is no `v0.3.0` — Phase 3 (the SMT circuit) was folded into
`v0.4.0` since it was never independently shippable on its own.

---

## 2. Shipped in `v0.4.0` (previously unreleased since `v0.2.0`)

```
5baa928  feat(phase-3): integrate Universal State Circuit with Sparse Merkle Trees
a05a620  feat(sequencer): migrate to EdDSA, implement LevelDB SMT, high-frequency matching engine
0aaf75a  feat: implement dual-token swaps and L1 deposit watcher for production ZK-rollup
5fbf3dc  test: implement exhaustive production-grade test suite and edge case handling
```
Plus the docs overhaul, the EIP-4844 post-Osaka DA fix, and the `test_client.go`
`--count`/lock rewrite from this session — all merged to `main` and included in
`v0.4.0`. Nothing currently sitting unreleased.

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

**`v0.5.0`** — completes Phase 5 (The L1 Bridge & Escape Hatch). `v0.4.0` shipped the
"happy path" half (`deposit()` + Deposit Watcher); the half that actually matters for
calling this trustless is still unbuilt:

- Rewrite `withdraw()` to accept `(tokenId, amount, merkleProof)` from **any caller**
  and verify it against the current `stateRoot` — not `onlyOwner`.
- A timeout/liveness check (e.g. "usable only if no `submitBatch()` in N days") so
  this stays a fallback, not the primary withdrawal path.
- Update [architecture/overview.md](../architecture/overview.md#known-gaps-as-of-2026-08-17)'s
  Known Gaps list once it lands — this is currently the top entry there.

No draft release notes yet — unlike `v0.4.0`, this work doesn't exist in code yet, so
there's nothing to bundle into a release description until it's actually built.

## Related
- [Marketing Roadmap](./roadmap-marketing.md) — forward-looking phase tracking
- [Technical Roadmap](./roadmap-technical.md) — internal engineering detail
- [Architecture Overview](../architecture/overview.md) — component-level known gaps
