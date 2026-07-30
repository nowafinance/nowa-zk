package batcher

import (
	"sync"

	"github.com/nowafinance/nowa-zk/sequencer/internal/types"
)

const BatchSize = 5 // Must exactly match the ZK Circuit

type Batcher struct {
	mu           sync.Mutex
	currentBatch *types.ZKBatch
	latestBatch  *types.ZKBatch
	batchCounter uint64
}

func NewBatcher() *Batcher {
	return &Batcher{
		currentBatch: &types.ZKBatch{
			BatchID: 0,
		},
	}
}

// AddTransition adds a state transition to the current batch.
// If the batch hits BatchSize, it seals it and starts a new one.
func (b *Batcher) AddTransition(t types.StateTransition, oldRoot, newRoot string) {
	b.mu.Lock()
	defer b.mu.Unlock()

	// If this is the first transition in the batch, set the OldRoot
	if len(b.currentBatch.Transitions) == 0 {
		b.currentBatch.OldRoot = oldRoot
	}

	b.currentBatch.Transitions = append(b.currentBatch.Transitions, t)

	// Update the NewRoot after every transition so the final one is the batch's NewRoot
	b.currentBatch.NewRoot = newRoot

	if len(b.currentBatch.Transitions) == BatchSize {
		// Batch is full! Seal it.
		// Set WithdrawalHash to zero for now (we can implement later)
		b.currentBatch.WithdrawalHash = "0"

		// Move current to latest
		b.latestBatch = b.currentBatch

		b.batchCounter++
		b.currentBatch = &types.ZKBatch{
			BatchID: b.batchCounter,
		}
	}
}

// GetLatestBatch returns the most recently sealed batch.
func (b *Batcher) GetLatestBatch() *types.ZKBatch {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.latestBatch
}

// GetCurrentBatchSize returns the number of transitions in the current batch.
func (b *Batcher) GetCurrentBatchSize() int {
	b.mu.Lock()
	defer b.mu.Unlock()
	return len(b.currentBatch.Transitions)
}
