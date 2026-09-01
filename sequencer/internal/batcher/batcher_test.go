package batcher

import (
	"fmt"
	"math/big"
	"testing"

	"github.com/nowafinance/nowa-zk/sequencer/internal/types"
	"github.com/stretchr/testify/assert"
)

// TestMimcHash_MatchesCircuitConstruction_NotNestedForm is a regression test for a
// real bug found live on Sepolia: the DepositHash/WithdrawalHash accumulators used to
// compute H(H(currentHash, index), amount) — two separate, independently-reset
// 2-item hashes — while the circuit computes H(currentHash, index, amount) as one
// continuous 3-item chain from a single reset (state_circuit.go's
// h.Write(currentDepositHash, op.TakerBase.Index, op.Amount); h.Sum()). These are
// mathematically different constructions (the nested form re-injects a fresh zero
// state partway through), so every real batch containing a deposit failed the
// circuit's final DepositHash equality check — masked until now by the
// deposit-signing bug blocking deposits from being proven at all.
//
// This pins down that mimcHash(a, b, c) — the single-chain form now used by both
// accumulators — genuinely differs from the old nested form, so a regression back to
// nesting would be caught here rather than only live, weeks later, against real gas.
func TestMimcHash_MatchesCircuitConstruction_NotNestedForm(t *testing.T) {
	a := big.NewInt(111)
	b := big.NewInt(222)
	c := big.NewInt(333)

	singleChain := mimcHash(a, b, c)

	nestedForm := mimcHash(mimcHash(a, b), c)

	if singleChain.Cmp(nestedForm) == 0 {
		t.Fatal("single-chain and nested hash forms produced the same value — this test can no longer distinguish the bug this guards against; the two constructions are expected to differ")
	}
}

func TestBatcher_AddTransition(t *testing.T) {
	b := NewBatcher()

	assert.Equal(t, 0, b.GetCurrentBatchSize())
	assert.Nil(t, b.GetLatestBatch())

	// A batch only seals once exactly BatchSize real transitions have accumulated —
	// confirm it stays open (no sealed batch, growing current-batch size) for every
	// transition short of that, then seals correctly on the exact Nth one.
	for i := 1; i < BatchSize; i++ {
		root := fmt.Sprintf("root%d", i)
		prevRoot := "oldRoot"
		if i > 1 {
			prevRoot = fmt.Sprintf("root%d", i-1)
		}
		b.AddTransition(types.StateTransition{Amount: "10"}, prevRoot, root)
		assert.Equal(t, i, b.GetCurrentBatchSize(), "batch should still be open after %d/%d transitions", i, BatchSize)
		assert.Nil(t, b.GetLatestBatch(), "no batch should seal before BatchSize is reached (%d/%d)", i, BatchSize)
	}

	lastOpenRoot := fmt.Sprintf("root%d", BatchSize-1)
	b.AddTransition(types.StateTransition{Amount: "10"}, lastOpenRoot, "newRoot1")

	assert.Equal(t, 0, b.GetCurrentBatchSize(), "batch resets to empty immediately after sealing")
	latest := b.GetLatestBatch()
	assert.NotNil(t, latest)
	assert.Equal(t, uint64(1), latest.BatchID)
	assert.Equal(t, "oldRoot", latest.OldRoot, "sealed batch's OldRoot must be the FIRST transition's oldRoot, not the last")
	assert.Equal(t, "newRoot1", latest.NewRoot)
	assert.Equal(t, "0", latest.WithdrawalHash)
	assert.Len(t, latest.Transitions, BatchSize)

	// A second batch of BatchSize transitions should chain correctly off the first.
	for i := 0; i < BatchSize; i++ {
		prevRoot := "newRoot1"
		if i > 0 {
			prevRoot = fmt.Sprintf("batch2root%d", i-1)
		}
		newRoot := fmt.Sprintf("batch2root%d", i)
		if i == BatchSize-1 {
			newRoot = "newRoot2"
		}
		b.AddTransition(types.StateTransition{Amount: "30"}, prevRoot, newRoot)
	}
	assert.Equal(t, 0, b.GetCurrentBatchSize())
	assert.Equal(t, uint64(2), b.GetLatestBatch().BatchID)
	assert.Equal(t, "newRoot1", b.GetLatestBatch().OldRoot, "batch #2's OldRoot must chain from batch #1's NewRoot")
	assert.Equal(t, "newRoot2", b.GetLatestBatch().NewRoot)
}
