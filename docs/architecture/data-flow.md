# Sequencer → Prover → L1 Data Flow

> [!IMPORTANT]
> This describes `main`'s Sequencer pipeline — on hold, not the current operated flow.
> Nowa-ZK's current flow is [`v0.3.0`](../project/testnet-v0.3.0-flow.md).

## Overview

This pipeline has three hops: a user's order reaches the **Sequencer**, a matched
fill becomes a sealed **batch**, and the **Prover** turns that batch into a proof settled
on L1. There is no blockchain in the middle — the Sequencer is a single Go process with
an in-memory orderbook and a LevelDB-backed Merkle tree.

---

## Step by step

### 1. Order submission

A client signs an order twice and `POST`s it to the Sequencer's `/order`:

- **Order intent signature** (`signature`): EdDSA over
  `SHA256(maker_address:token_id:amount:price:is_buy:nonce)`. This is what
  `engine.VerifyEdDSASignature` checks before the order enters the book.
- **Circuit signature** (`circuit_signature`): EdDSA over
  `MiMC(OpType, pubX, pubY, baseIndex, quoteIndex)` — the *exact* message the ZK circuit
  verifies later. `baseIndex`/`quoteIndex` are the account's Merkle leaf indices
  (`accountID*256 + tokenID`), so a client must know its own account ID before it can
  produce a valid circuit signature — `GET /account?pubkey=...` onboards a new pubkey
  and returns both indices. See `sequencer/cmd/cli/test_client.go` for a full worked
  example (also runnable via `make test-sequencer-live`).

The fill amount is deliberately **not** part of the circuit-signature message — one
signature authorizes any number of partial fills of the same resting order.

### 2. Matching

`sequencer/internal/engine` maintains one orderbook per token, sorted by price/time
priority (`Orderbook.Bids`/`Asks`). A crossing order emits a `types.Trade` on an internal
channel; `applyTrade` (`sequencer/cmd/sequencer/main.go`) then:

1. Snapshots the pre-update Merkle paths for maker-base, maker-quote, taker-base,
   taker-quote (order matters — the circuit updates leaves in this same sequence).
2. Debits/credits the four balances in the LevelDB SMT.
3. Hands the resulting `StateTransition` (old root, new root, all four paths, both
   signatures) to the `Batcher`.

### 3. Batch sealing

`sequencer/internal/batcher.Batcher.AddTransition` appends the transition to the current
batch and, once it reaches `BatchSize` transitions, seals it: computes the
`WithdrawalHash`/`DepositHash` MiMC accumulators, stores it in `batchHistory`, and starts
a new batch. **`BatchSize = 1` today** — every fill seals its own batch immediately (must
match `prover/circuits.BatchSize`, or the Prover's witness size won't match its compiled
keys — see [troubleshooting.md](../operations/troubleshooting.md)).

### 4. Prover pickup

`prover start` polls `GET /batch/latest` and `GET /batch/:id` against the Sequencer
(`--indexer-url`, default `http://localhost:8080` — the flag name is a holdover from the
legacy design, it just means "where to fetch batches from"). It tracks
`last_processed_batch` in its own BadgerDB store (`--data-dir`, default
`~/.nowa-zk/prover/data`) so it resumes correctly after a restart.

### 5. Proof generation

For the fetched batch, the Prover builds a `circuits.StateTransitionCircuit` witness from
the JSON transition (decoding both EdDSA signatures — note the maker/taker signature
verification described above applies even to deposits, which are authorized by the
sequencer's own signing key rather than skipped, since `gnark` doesn't support cheaply
skipping a signature check conditionally) and runs Groth16 `Prove` + a local `Verify`
before ever touching L1.

### 6. L1 submission

`submitProofWithBlob` (`prover/cmd/prover/start.go`) packs the batch's transitions into
an EIP-4844 blob (`prover/internal/da/blob.go` — cell-proof format, `BlobSidecarVersion1`,
required since the Osaka hardfork), then sends an EIP-4844 transaction calling
`NowaRollup.submitBatch(proof, oldRoot, newRoot, withdrawalHash, depositHash, dataHash)`.
The contract verifies the proof, requires `blobhash(0)` to be present, and advances
`stateRoot`.

### 7. Deposits (parallel path)

`StartDepositWatcher` (`sequencer/cmd/sequencer/deposit_watcher.go`) runs as a goroutine
inside the Sequencer binary, subscribed to `NowaRollup`'s `Deposit` event via
`bindings.NowaRollup.WatchDeposit`. Each event mints an `OpDeposit` transition directly
into the batch queue — deposits flow through the same batching/proving path as trades,
just authorized by the sequencer's signing key instead of the depositor's.

---

## Key Guarantees

- **Deterministic leaf ordering**: the four Merkle paths in a `StateTransition` are taken
  in a fixed order (maker-base, maker-quote, taker-base, taker-quote) matching the
  circuit's `processOperation` — get this order wrong and proof generation fails with an
  unsatisfied constraint, not a silent bug.
- **No dummy padding**: with `BatchSize = 1`, every proof corresponds to exactly one real
  state transition — there's no filler operation to reason about.
- **Resumability**: the Prover's `last_processed_batch` checkpoint means restarting it
  resumes from the next unprocessed batch rather than replaying settled ones.

## Code References

- Matching: [`sequencer/internal/engine/matching.go`](../../sequencer/internal/engine/matching.go)
- Trade application: [`sequencer/cmd/sequencer/main.go`](../../sequencer/cmd/sequencer/main.go)
- Batching: [`sequencer/internal/batcher/batcher.go`](../../sequencer/internal/batcher/batcher.go)
- Deposit watcher: [`sequencer/cmd/sequencer/deposit_watcher.go`](../../sequencer/cmd/sequencer/deposit_watcher.go)
- Circuit: [`prover/circuits/state_circuit.go`](../../prover/circuits/state_circuit.go)
- Prover loop + L1 submission: [`prover/cmd/prover/start.go`](../../prover/cmd/prover/start.go)
- Blob DA: [`prover/internal/da/blob.go`](../../prover/internal/da/blob.go)
- L1 contract: [`contracts/src/NowaRollup.sol`](../../contracts/src/NowaRollup.sol)
