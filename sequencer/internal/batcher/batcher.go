package batcher

import (
	"encoding/binary"
	"fmt"
	"math/big"
	"sync"

	"github.com/consensys/gnark-crypto/ecc/bn254/fr"
	"github.com/consensys/gnark-crypto/ecc/bn254/fr/mimc"
	"github.com/nowafinance/nowa-zk/sequencer/internal/types"
)

const BatchSize = 25 // must match prover/circuits.BatchSize — real fills only, no dummy padding.
// A batch only seals once exactly this many real transitions have accumulated —
// on low trade volume, nothing settles to L1 until BatchSize fills happen.

const (
	OpTrade    = 0
	OpTransfer = 1
	OpWithdraw = 2
	OpDeposit  = 3
)

type Batcher struct {
	mu              sync.Mutex
	currentBatch    *types.ZKBatch
	latestBatch     *types.ZKBatch
	batchCounter    uint64
	batchHistory    map[uint64]*types.ZKBatch

	// Running hash accumulators for the current batch
	currentDepositHash    *big.Int
	currentWithdrawalHash *big.Int
}

func NewBatcher() *Batcher {
	return &Batcher{
		// Prover treats lastProcessed=0 as "none yet" and requests batch 1 next.
		currentBatch: &types.ZKBatch{
			BatchID: 1,
		},
		batchCounter:          1,
		batchHistory:          make(map[uint64]*types.ZKBatch),
		currentDepositHash:    big.NewInt(0),
		currentWithdrawalHash: big.NewInt(0),
	}
}

// mimcHash hashes N *big.Int values using MiMC (BN254) as a single continuous chain
// from one fresh state — matching prover/circuits/state_circuit.go's
// h.Write(a, b, c); h.Sum() exactly. This must NOT be composed by nesting two calls
// (e.g. mimcHash(mimcHash(a, b), c)) to fold in a third item — that's a different,
// incompatible construction: MiMC's Miyaguchi-Preneel state threads through every
// Write in one chain, so f(f(f(0,a),b),c) (one chain, three items) is mathematically
// different from f(f(0, f(f(0,a),b)), c) (two chains, the second re-injecting a fresh
// zero state) — confirmed live on Sepolia: the DepositHash accumulator used the
// nested form here while the circuit used the single-chain form, and every batch
// containing a deposit failed the circuit's final DepositHash equality check as a
// result once deposits could reach proving at all (this was masked until now by the
// deposit-signing bug blocking every deposit from being proven in the first place).
func mimcHash(items ...*big.Int) *big.Int {
	h := mimc.NewMiMC()
	for _, item := range items {
		var f fr.Element
		f.SetBigInt(item)
		b := f.Bytes()
		h.Write(b[:])
	}
	var res fr.Element
	res.SetBytes(h.Sum(nil))
	out := new(big.Int)
	res.BigInt(out)
	return out
}

// AddTransition adds a state transition to the current batch.
// If the batch hits BatchSize, it seals it and starts a new one.
func (b *Batcher) AddTransition(t types.StateTransition, oldRoot, newRoot string) {
	b.mu.Lock()
	defer b.mu.Unlock()

	// If this is the first transition in the batch, set the OldRoot.
	if len(b.currentBatch.Transitions) == 0 {
		b.currentBatch.OldRoot = oldRoot
	}

	b.currentBatch.Transitions = append(b.currentBatch.Transitions, t)

	// Update the NewRoot after every transition so the final one is the batch's NewRoot.
	b.currentBatch.NewRoot = newRoot

	// Accumulate DepositHash: H(currentHash, leafIndex, amount)  for OpDeposit.
	// Mirrors the circuit's deposit accumulator.
	if t.OpType == OpDeposit {
		leafIndex := new(big.Int).SetUint64(t.TakerBase.Index)
		amount, ok := new(big.Int).SetString(t.Amount, 10)
		if !ok {
			amount = big.NewInt(0)
		}
		b.currentDepositHash = mimcHash(b.currentDepositHash, leafIndex, amount)
	}

	// Accumulate WithdrawalHash: H(currentHash, leafIndex, amount) for OpWithdraw.
	if t.OpType == OpWithdraw {
		leafIndex := new(big.Int).SetUint64(t.MakerBase.Index)
		amount, ok := new(big.Int).SetString(t.Amount, 10)
		if !ok {
			amount = big.NewInt(0)
		}
		b.currentWithdrawalHash = mimcHash(b.currentWithdrawalHash, leafIndex, amount)
	}

	if len(b.currentBatch.Transitions) == BatchSize {
		// Batch is full — seal it.
		b.currentBatch.WithdrawalHash = b.currentWithdrawalHash.String()
		b.currentBatch.DepositHash = b.currentDepositHash.String()

		// Store in history and expose as latest.
		sealedID := b.currentBatch.BatchID
		b.batchHistory[sealedID] = b.currentBatch
		b.latestBatch = b.currentBatch

		// Start a new empty batch.
		b.batchCounter++
		b.currentBatch = &types.ZKBatch{
			BatchID: b.batchCounter,
		}
		// Reset accumulators for the next batch.
		b.currentDepositHash = big.NewInt(0)
		b.currentWithdrawalHash = big.NewInt(0)
	}
}

// GetLatestBatch returns the most recently sealed batch.
func (b *Batcher) GetLatestBatch() *types.ZKBatch {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.latestBatch
}

// GetBatch returns a specific sealed batch by its ID.
func (b *Batcher) GetBatch(id uint64) (*types.ZKBatch, bool) {
	b.mu.Lock()
	defer b.mu.Unlock()
	batch, ok := b.batchHistory[id]
	return batch, ok
}

// GetBatchCount returns the number of sealed batches.
func (b *Batcher) GetBatchCount() uint64 {
	b.mu.Lock()
	defer b.mu.Unlock()
	return uint64(len(b.batchHistory))
}

// GetCurrentBatchSize returns the number of transitions in the current (unsealed) batch.
func (b *Batcher) GetCurrentBatchSize() int {
	b.mu.Lock()
	defer b.mu.Unlock()
	return len(b.currentBatch.Transitions)
}

// batchIDToKey encodes batch ID as big-endian bytes for storage keys.
func batchIDToKey(id uint64) []byte {
	b := make([]byte, 8)
	binary.BigEndian.PutUint64(b, id)
	return b
}

// DebugDepositHash returns a string representation of the current deposit hash.
// Used for logging/debugging only.
func (b *Batcher) DebugDepositHash() string {
	b.mu.Lock()
	defer b.mu.Unlock()
	return fmt.Sprintf("%s", b.currentDepositHash.String())
}
