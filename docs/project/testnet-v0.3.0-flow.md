# Current Flow (Testnet): `v0.3.0`

> **This is Nowa-ZK's current, operated flow — deployed and verified live on Sepolia.**
> It's the one actually running on testnet today. The rest of `docs/` additionally
> describes `main` — built groundwork toward a more complete ZK rollup (on-chain state
> root tracking, EIP-4844 blob DA, an escape hatch), **on hold** pending prioritization,
> not something currently being worked. See
> [§ How this relates to `main`](#how-this-relates-to-main) below.

Tag: [`v0.3.0` — Testnet](https://github.com/nowafinance/nowa-zk/releases/tag/v0.3.0),
built off branch `release/v0.3.0`, published 2026-08-18. Verified end-to-end live on
Ethereum Sepolia (see [§ Verified on Sepolia](#verified-on-sepolia)).

---

## The flow

**1. Execution (Cosmos L2) — `Nowa_Orderbook.sol`**
Users submit EIP-712-signed orders. Matched trades execute on-chain via
`executeBatchTradeMultiFill`, which validates each order's ECDSA signature, checks
fund-locking, and updates filled quantities and balances — **the L2 chain is the
source of truth for exchange state** (balances, fills) in this flow, not L1.

**2. Indexer**
Watches L2 blocks for `executeBatchTradeMultiFill` calls, recovers each trade's
signer info — `(messageHash, pubKeyX, pubKeyY, sigR, sigS)` — from calldata, and
groups trades into batches of **125**.

**3. Prover**
Pulls each 125-trade batch and splits it into **6 chunks of 25**. For each chunk it
generates one Groth16 proof (`gnark`, BN254) that verifies all 25 ECDSA signatures
inside the circuit and folds `(messageHash, pubKeyX, pubKeyY)` per trade into a single
on-chain-recomputable MiMC `BatchRoot`.

**4. L1 settlement — `TradeRegistry.sol`**
Each chunk's plaintext trade data `(messageHash, pubKeyX, pubKeyY)` — never the raw
signatures — is submitted to L1 alongside its proof via `registerTrades(batchNumber,
chunkIndex, proof, commitments, commitmentPok, publicInputs, trades)`. The contract:
- Independently recomputes the MiMC `BatchRoot` on-chain from the submitted plaintext
  (packs up to 25 trades into a 150-element array — 6 field elements per trade:
  `messageHash` and both pubkey coordinates each split into two 128-bit halves — and
  hashes it), and requires it match `publicInputs[0]`.
- Verifies the Groth16 proof via the generated `TradeVerifier`.
- Tracks verification per `(batchNumber, chunkIndex)` in `isChunkVerified` /
  `chunkBatchRoot`, and emits `TradesVerified` / `TradesSettled`.

**Worth knowing:** `TradeRegistry.sol` has **no `stateRoot` and tracks no
balances/fills.** It anchors, per chunk, that a specific set of trades carried valid
signatures — `Nowa_Orderbook.sol` on the L2 chain remains the source of truth for
actual account balances. There's no L1-enforced force-withdrawal against this flow's
L2 state today, since L1 never holds state to check a withdrawal against.

---

## Verified on Sepolia

Per the release notes: the full loop was run live — batches built by the indexer,
proofs generated and locally verified, chunks submitted and mined on L1, and the
Prover confirmed to resume correctly after a kill (on restart it detects
already-verified chunks via `isChunkVerified` and skips re-submission).

Known limitation, stated in the release notes: proving throughput (~2–4 min per
125-trade batch, chunks proved sequentially) can fall behind indexer batch production
under sustained load.

---

## How this relates to `main`

This flow lives on `release/v0.3.0`. `main` carries a broader rewrite — an off-chain
Go Sequencer, `NowaRollup.sol` tracking a full on-chain `stateRoot`, EIP-4844 blob data
availability, and a deposit-bound escape hatch (`emergencyWithdraw()` + tooling) —
extending toward a more complete ZK rollup than this flow currently is. That work is
real and tested, live-verified on Sepolia in isolation, but **on hold pending
prioritization** — this `v0.3.0` flow is what's actually deployed and what "Nowa-ZK on
testnet" means today.

| | `v0.3.0` (**this flow, live**) | `main` (in progress) |
|---|---|---|
| Execution | Cosmos L2 (`Nowa_Orderbook.sol`) | Off-chain Go Sequencer |
| Order signatures | ECDSA (EIP-712) | EdDSA (BabyJubJub) |
| Batching | Indexer groups 125 trades, proved in 6×25 chunks | Sequencer seals batches (currently size 1) |
| L1 contract | `TradeRegistry.sol` — signature-validity anchor only | `NowaRollup.sol` — tracks a full on-chain `stateRoot` |
| Data availability | Plaintext trade fields in calldata | EIP-4844 blobs |
| Escape hatch | Not yet applicable (L1 has no state to exit against) | `emergencyWithdraw()` + `claim-escape` / `reconstruct-proof` tools |

The two branches' contract/service code diverges enough (different signature scheme,
different L1 contract, different data model) that merging one into the other isn't a
normal fast-forward — treat a `git pull`/merge between them as something that needs a
deliberate reconciliation plan, not a routine sync. None is planned right now.

If you're operating or building against what's live on testnet today, use
`release/v0.3.0` and this document. `main` and the rest of `docs/` are there for
whenever the more complete rollup work gets picked back up.
