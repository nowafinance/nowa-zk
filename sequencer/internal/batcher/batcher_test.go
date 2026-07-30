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

	for i := 0; i < 4; i++ {
		b.AddTransition(types.StateTransition{Amount: "10"}, "oldRoot", "newRoot1")
	}

	assert.Equal(t, 4, b.GetCurrentBatchSize())
	assert.Nil(t, b.GetLatestBatch())

	// 5th transition should seal the batch
	b.AddTransition(types.StateTransition{Amount: "20"}, "newRoot1", "newRoot2")

	assert.Equal(t, 0, b.GetCurrentBatchSize())
	
	latest := b.GetLatestBatch()
	assert.NotNil(t, latest)
	assert.Equal(t, uint64(0), latest.BatchID)
	assert.Equal(t, "oldRoot", latest.OldRoot)
	assert.Equal(t, "newRoot2", latest.NewRoot)
	assert.Equal(t, "0", latest.WithdrawalHash)
	assert.Len(t, latest.Transitions, 5)

	// Add 1 more to next batch
	b.AddTransition(types.StateTransition{Amount: "30"}, "newRoot2", "newRoot3")
	assert.Equal(t, 1, b.GetCurrentBatchSize())
	
	// Latest should still be BatchID 0
	assert.Equal(t, uint64(0), b.GetLatestBatch().BatchID)
}
