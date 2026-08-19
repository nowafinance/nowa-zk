# Release Status

> [!IMPORTANT]
> **This file tracks `main`'s in-progress rollup work, not the flow currently live on
> testnet.** Nowa-ZK's current flow — deployed and verified on Sepolia today — is
> [`v0.3.0`](https://github.com/nowafinance/nowa-zk/releases/tag/v0.3.0) (Cosmos L2 +
> Indexer + `TradeRegistry.sol`, branch `release/v0.3.0`); see
> [testnet-v0.3.0-flow.md](testnet-v0.3.0-flow.md) for that flow. Everything below
> describes `main`'s Sequencer/`NowaRollup.sol` work toward a more complete ZK rollup,
> which hasn't reached testnet yet.

What's actually been released, what's built but unreleased, and what's genuinely
incomplete — a snapshot of *shipped* state, not a forward plan.

*Last verified against code and GitHub Releases: 2026-08-18.*

---

## 1. Published releases

| Tag | Title | Contents |
|---|---|---|
| `v0.0.1` | Initial Alpha | Initial project structure only — no feature list. |
| `v0.0.2` | Genesis Release | First working system: old-model `nowa-sequencer` (the Cosmos indexer, not today's Sequencer), Gnark prover, `StateManager`/`BatchRegistry`/`VerifierAdapter` contracts, systemd deployment docs. |
| `v0.1.0` | End-to-End App-Specific ZK Rollup & MiMC Integration | Pivoted from generic rollup to trade-specific. MiMC hashing, `TradeRegistry.sol` (replacing `BatchRegistry`), Foundry E2E tests, automated trusted setup. |
| `v0.2.0` | L1 Data Availability & ZK Trade Signatures | Closes Phase 1+2: L1 DA (raw payloads on-chain), finalized MiMC trade signature circuit, `TradeRegistry.sol` + `Verifier.sol` deployed, modular circuits + fuzz tests. |
| `v0.4.0` | The High-Frequency Sequencer & Universal State Transition | Published 2026-08-17 on `main`. Closes Phase 3+4: Sequencer (matching engine, LevelDB SMT, REST API, EdDSA signatures), Universal State Transition Circuit (`TradeRegistry` → `NowaRollup`), partial L1 bridge (`deposit()` + Deposit Watcher, dual-token swaps), EIP-4844 blob DA with post-Osaka cell proofs, exhaustive test suite, full docs rewrite. See §3 below for what it does *not* close. |
| `v0.3.0` | Testnet | **Currently "Latest" on GitHub** (badge is repo-wide, not branch-scoped). Published 2026-08-18 on `release/v0.3.0` — not in this branch's linear history despite the version number. **This is Nowa-ZK's current flow** (Cosmos L2 + Indexer + `TradeRegistry.sol`), verified live on Sepolia and actually running on testnet, unlike everything else in this table. See [testnet-v0.3.0-flow.md](testnet-v0.3.0-flow.md). |

`v0.0.3` is a bare git tag between `v0.0.2` and `v0.1.0` — never published as a GitHub
Release, no notes. On `main`'s own history, Phase 3 (the SMT circuit) was folded into
`v0.4.0` since it was never independently shippable on its own — `main` has no
`v0.3.0` tag of its own; the `v0.3.0` release above belongs entirely to the separate
`release/v0.3.0` branch's numbering.

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
  no manual relay needed. Two real bugs found and fixed while verifying this
  end-to-end (both would have broken every real deposit, not just an edge case):
  the watcher was building its tree lookup key from raw, uncompressed `pubX||pubY`
  instead of the actual compressed EdDSA pubkey format every client uses (deposits
  were landing under a key the depositor could never look up again), and new
  deposit accounts were getting a phantom 1,000,000 "lab credit" on top of the real
  deposited amount. Both fixed and regression-tested
  (`sequencer/cmd/sequencer/deposit_watcher_test.go`).
- **Dual-token swaps** — trades aren't limited to one fixed pair.
- **Escape hatch (deposit-bound) — verified live end-to-end, not just Foundry-tested.**
  `NowaRollup.emergencyWithdraw()` force-withdraws via Merkle proof once the
  Sequencer has stalled for a fixed, non-owner-adjustable `escapeTimeout`, using
  `MiMC.sol` (byte-compatible with the circuit's hash, verified against
  `gnark-crypto`'s actual round-constant generation) to fold the proof up to
  `stateRoot`. Beyond the 15/15 Foundry suite, this was proven on real Sepolia
  infrastructure: `GET /proof` (new Sequencer endpoint, `sequencer/internal/api`)
  serves the Merkle proof for a live account, and `sequencer/cmd/claim-escape` (new
  CLI) fetches it and submits the withdrawal — real deposit → real proof fetch →
  real `emergencyWithdraw()` call → wallet balance 0 → 500, `escapeWithdrawn` set, a
  second claim correctly reverting "Already withdrawn". Deposit-bound scope
  limitation unchanged, see below.
- **Test coverage** — matching engine, SMT, batching, full trade application, a real
  Groth16 prove/verify cycle, a `--count N` live-integration tool
  (`sequencer/cmd/cli/test_client.go`) that stress-tests batching end to end, a
  full Foundry suite for `NowaRollup.sol` including the escape hatch, and the
  deposit-watcher regression tests above.
- **Documentation** — rewritten this session to match the Sequencer-centric
  architecture (previous docs described the retired Cosmos-indexer design).

### ❌ Not done — real gaps, not roadmap aspirations

- **Escape hatch is deposit-bound, not fully permissionless.** L2 accounts are keyed
  by BabyJubJub pubkeys, not Ethereum addresses, so the contract can't verify
  on-chain that a caller controls the L2 private key — eligibility is instead tied
  to whoever originally deposited into that pubkey (`depositorOf`, first-depositor-
  wins). Covers "get back what you deposited"; does **not** cover balance a pubkey
  only ever received via L2 trades and never deposited into directly. Full on-chain
  EdDSA verification (BabyJubJub curve arithmetic in Solidity) would close this but
  is a separate, larger piece of work — not started. See
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
- **No deposit can currently be proven and settled.** The Universal State Transition
  Circuit unconditionally verifies a Maker EdDSA signature on every operation,
  including deposits — its own comments say a designated sequencer keypair should
  sign deposits, but `deposit_watcher.go` submits a dummy zero pubkey/signature, so
  a real batch containing a deposit fails circuit constraints. Found live this
  session while verifying the blob-reconstruction tool below; not yet fixed. Blocks
  proving *any* deposit-containing batch end-to-end on `main`, separate from the two
  Deposit Watcher bugs already fixed (§2 above).

## Related
- [Architecture Overview](../architecture/overview.md) — component-level known gaps
- [Current Flow — `v0.3.0`](./testnet-v0.3.0-flow.md) — what's actually operated today
