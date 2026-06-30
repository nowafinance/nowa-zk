package sequencer

import (
	"fmt"
	"log"

	"github.com/nowafinance/nowa-zk/sequencer/internal/prover"
)

// ProverClient interface for easier testing
type ProverClient interface {
	GetLatestProvenBatch() (uint64, error)
}

// CleanupOldBatches performs intelligent cleanup based on prover's finalized batches
func (bs *BatchStore) CleanupOldBatches(proverClient ProverClient) error {
	// 1. Get last deleted batch from DB (avoid re-scanning)
	lastDeleted, err := bs.GetLastDeletedBatch()
	if err != nil {
		return fmt.Errorf("failed to get last deleted batch: %w", err)
	}
	log.Printf("🧹 Cleanup: Last deleted batch: %d", lastDeleted)

	// 2. Query prover for latest proven batch on L1
	latestProven, err := proverClient.GetLatestProvenBatch()
	if err != nil {
		// Don't fail if prover is unavailable - just skip cleanup this round
		log.Printf("⚠️  Cleanup skipped: Unable to reach prover API (%v). Will retry later.", err)
		return nil // Return nil so sequencer doesn't crash
	}
	log.Printf("🧹 Cleanup: Latest proven batch: %d", latestProven)

	// Nothing to cleanup
	if latestProven == 0 || latestProven <= lastDeleted {
		log.Printf("🧹 No batches to cleanup (proven=%d, last_deleted=%d)", latestProven, lastDeleted)
		return nil
	}

	// 3. Delete batches in range (lastDeleted+1 to latestProven-1)
	// Note: Prover already has all necessary data (tx hashes, metadata, etc.)
	// So we can completely delete sequencer data without saving metadata
	deleteCount := 0
	for batchNum := lastDeleted + 1; batchNum < latestProven; batchNum++ {
		// Delete batch completely (no metadata save needed)
		if err := bs.DeleteBatch(batchNum); err != nil {
			log.Printf("⚠️  Warning: failed to delete batch %d: %v", batchNum, err)
			continue
		}

		deleteCount++
	}

	// 4. Update tracker
	if deleteCount > 0 {
		if err := bs.SetLastDeletedBatch(latestProven - 1); err != nil {
			return fmt.Errorf("failed to update tracker: %w", err)
		}
	}

	log.Printf("✅ Cleanup complete: Deleted %d batches (%d to %d)",
		deleteCount, lastDeleted+1, latestProven-1)

	return nil
}

// NewProverClient creates a prover client from config
func NewProverClient(apiURL string) ProverClient {
	return prover.NewClient(apiURL)
}
