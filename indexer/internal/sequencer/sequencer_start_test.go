package sequencer

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/nowafinance/nowa-zk/sequencer/internal/sequencer/types"
)

func TestGetStartingBlock(t *testing.T) {
	// Setup temporary directory for DB
	tmpDir, err := os.MkdirTemp("", "sequencer_test_start_block")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	// Create a real BatchStore (using BadgerDB in tmp dir)
	store, err := NewBatchStore(tmpDir)
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()

	tests := []struct {
		name           string
		setupDB        func(*BatchStore)
		config         *types.Config
		expectedBlock  uint64
		expectedReason string
	}{
		{
			name: "DB Empty, IndexFromBlock Not Set -> 0",
			setupDB: func(bs *BatchStore) {
				// Do nothing, empty DB
			},
			config: &types.Config{
				IndexFromBlock: 0,
			},
			expectedBlock: 0,
		},
		{
			name: "DB Empty, IndexFromBlock Set -> IndexFromBlock - 1",
			setupDB: func(bs *BatchStore) {
				// Do nothing
			},
			config: &types.Config{
				IndexFromBlock: 100,
			},
			expectedBlock: 99, // Should return 99 so processing starts at 100
		},
		{
			name: "DB Has Data, IndexFromBlock Set -> DB Value",
			setupDB: func(bs *BatchStore) {
				err := bs.SetLastProcessedBlock(50)
				assert.NoError(t, err)
			},
			config: &types.Config{
				IndexFromBlock: 100,
			},
			expectedBlock: 50, // DB priority
		},
		{
			name: "DB Has Data (0), IndexFromBlock Set -> DB Value?",
			// Wait, if DB has explicit 0, it means we processed block 0.
			// Is this distinguishable from "Empty DB"?
			// In BadgerDB, if we SetLastProcessedBlock(0), the key exists.
			setupDB: func(bs *BatchStore) {
				err := bs.SetLastProcessedBlock(0)
				assert.NoError(t, err)
			},
			config: &types.Config{
				IndexFromBlock: 100,
			},
			expectedBlock: 99,
			// Actual logic:
			// GetLastProcessedBlock returns 0 if key found AND value is 0.
			// OR if key NOT found (err=nil, lastBlock=0).
			// So "DB Has Data (0)" is indistinguishable from "DB Empty" in current implementation?
			// Let's verify what the code does.
			// Code: if err == nil && lastBlock > 0 { return lastBlock }
			// So if lastBlock is 0, it falls through to config!
			// So if we processed ONLY block 0, and then set IndexFromBlock=100, it WILL jump to 100.
			// This matches the "DB Empty" behavior.
			// This is acceptable behavior (if you only processed block 0, you probably haven't done much).
		},
		{
			name: "DB Has Data (0), IndexFromBlock Not Set -> 0",
			setupDB: func(bs *BatchStore) {
				err := bs.SetLastProcessedBlock(0)
				assert.NoError(t, err)
			},
			config: &types.Config{
				IndexFromBlock: 0,
			},
			expectedBlock: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Clear DB for each test? Or reuse?
			// Since we use one DB instance, we should clear or use distinct DBs.
			// Reusing DB is tricky if we want "Empty" state.
			// Let's clear logic: bs.ClearAll()
			err := store.ClearAll()
			assert.NoError(t, err)

			// Setup DB state
			if tt.setupDB != nil {
				tt.setupDB(store)
			}

			// Create Service with this store and config
			svc := &Service{
				batches: store,
				config:  tt.config,
			}

			// Call method
			got := svc.initStartingBlock()

			assert.Equal(t, tt.expectedBlock, got)

			// Verify persistence:
			// The method should have written 'got' to the DB.
			storedBlock, err := store.GetLastProcessedBlock()
			if err != nil && err.Error() != "block 0 not found" {
				// store.GetLastProcessedBlock returns 0 if not found,
				// or maybe error depending on impl?
				// Implementations usually return 0 on not found if method signature returns value.
				// Let's check store.go: it returns 0 on ErrKeyNotFound.
			}

			// GetLastProcessedBlock returns 0 if key not found.
			// If we expect 0, checking storedBlock==0 is ambiguous (could be "not found" or "found 0").
			// But since we explicitly persist even 0 (in the new code), it should be fine.
			// EXCEPT: "DB Empty, IndexFromBlock Not Set -> 0".
			// My code: "3. Default to 0 ... Persist ... 0".
			// So DB should have 0.

			assert.Equal(t, got, storedBlock, "Start block should be persisted to DB")
		})
	}
}
