package batcher

import (
	"testing"

	"github.com/nowafinance/nowa-zk/sequencer/internal/types"
	"github.com/stretchr/testify/assert"
)

func TestBatcher_AddTransition(t *testing.T) {
	b := NewBatcher()

	assert.Equal(t, 0, b.GetCurrentBatchSize())
	assert.Nil(t, b.GetLatestBatch())

	// BatchSize=1: first real transition seals as batch ID 1.
	b.AddTransition(types.StateTransition{Amount: "10"}, "oldRoot", "newRoot1")

	assert.Equal(t, 0, b.GetCurrentBatchSize())
	latest := b.GetLatestBatch()
	assert.NotNil(t, latest)
	assert.Equal(t, uint64(1), latest.BatchID)
	assert.Equal(t, "oldRoot", latest.OldRoot)
	assert.Equal(t, "newRoot1", latest.NewRoot)
	assert.Equal(t, "0", latest.WithdrawalHash)
	assert.Len(t, latest.Transitions, 1)

	b.AddTransition(types.StateTransition{Amount: "30"}, "newRoot1", "newRoot2")
	assert.Equal(t, 0, b.GetCurrentBatchSize())
	assert.Equal(t, uint64(2), b.GetLatestBatch().BatchID)
	assert.Equal(t, "newRoot2", b.GetLatestBatch().NewRoot)
}
