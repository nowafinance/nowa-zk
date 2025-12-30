package sequencer

import (
	"sync"
	"time"

	"github.com/tannetwork/zk-sequencer/sequencer/pkg/logger"
	"github.com/tannetwork/zk-sequencer/sequencer/pkg/rpc"
)

// processBlockRange fetches and processes a range of blocks (up to 100)
// Ensures deterministic transaction ordering: sorted by block number, then transaction index
// Returns true if all blocks were fully processed
func (s *Service) processBlockRange(startBlock, endBlock uint64) bool {
	// Check if context is already canceled
	select {
	case <-s.ctx.Done():
		return false
	default:
	}

	blockCount := endBlock - startBlock + 1
	logger.Info("📦 Processing block range #%d to #%d (%d blocks)", startBlock, endBlock, blockCount)

	// Collect all transactions from all blocks in order
	type blockTxs struct {
		blockNum  uint64
		txs       []*rpc.Transaction
		blockHash string
	}
	allBlocks := make([]blockTxs, 0, blockCount)

	// Container for parallel fetch results
	type fetchResult struct {
		block *rpc.Block
		err   error
	}
	results := make([]fetchResult, blockCount)
	var wg sync.WaitGroup

	// logger.Info("⚡ Fetching %d blocks", blockCount)

	// Fetch all blocks in range with UNLIMITED PARALLELISM
	// Each block gets its own goroutine - all 100 fetch simultaneously
	for i := 0; i < int(blockCount); i++ {
		wg.Add(1)
		go func(index int) {
			defer wg.Done()

			blockNum := startBlock + uint64(index)

			// Log which block we're fetching
			// logger.Info("🔍 Fetching block #%d", blockNum)

			// Check context
			if s.ctx.Err() != nil {
				results[index] = fetchResult{err: s.ctx.Err()}
				return
			}

			// Retry individual block up to 5 times with exponential backoff
			var block *rpc.Block
			var err error
			maxRetries := 5 // Increased from 3 for robustness

			for attempt := 1; attempt <= maxRetries; attempt++ {
				block, err = s.rpcClient.GetBlockByNumber(s.ctx, blockNum, true)
				if err == nil {
					break // Success!
				}

				if s.ctx.Err() != nil {
					results[index] = fetchResult{err: s.ctx.Err()}
					return
				}

				// Exponential backoff: 1s, 2s, 4s, 8s, 16s
				if attempt < maxRetries {
					backoff := time.Duration(1<<uint(attempt-1)) * time.Second
					logger.Warn("Block %d fetch attempt %d/%d failed, retrying in %v...", blockNum, attempt, maxRetries, backoff)
					time.Sleep(backoff)
				}
			}

			if err != nil {
				logger.Error("Block %d failed after %d attempts: %v", blockNum, maxRetries, err)
				results[index] = fetchResult{err: err}
				return
			}

			// Enrich transactions with contract addresses (also parallelized!)
			for _, tx := range block.Transactions {
				if tx.IsContractDeployment {
					if tx.ContractAddress == "" || tx.ContractAddress == "0x" {
						receipt, err := s.rpcClient.GetTransactionReceipt(s.ctx, tx.Hash)
						if err == nil && receipt != nil && receipt.ContractAddress != "" {
							tx.ContractAddress = receipt.ContractAddress
						} else if tx.ContractAddress == "" && s.ctx.Err() == nil {
							tx.ContractAddress = rpc.ComputeContractAddress(tx.From, tx.Nonce)
						}
					}
					if tx.ContractAddress != "" && tx.ContractAddress != "0x" {
						tx.To = tx.ContractAddress
					}
				}
			}

			results[index] = fetchResult{block: block, err: nil}
		}(i)
	}

	// Wait for all fetches to complete (will be much faster now!)
	wg.Wait()
	logger.Info("✅ All %d blocks fetched, now processing sequentially...", blockCount)

	// Sequential Verification and Processing
	// Must process in order to detect reorgs and maintain chain integrity
	for i, res := range results {
		blockNum := startBlock + uint64(i)

		// Check for fetch errors
		if res.err != nil {
			logger.Error("🚨 Block %d failed to fetch: %v", blockNum, res.err)
			logger.Error("⏸️  Sequencer will retry entire range on next poll")
			return false
		}

		block := res.block

		// Check for reorg
		if storedHash, err := s.batches.GetBlockHash(blockNum); err == nil {
			if storedHash != block.Hash {
				logger.Warn("Reorg detected at block #%d, handling reorg...", blockNum)
				forkPoint, err := s.findForkPoint(blockNum)
				if err != nil {
					logger.Error("Failed to find fork point: %v", err)
					return false
				}
				if err := s.handleReorg(forkPoint); err != nil {
					logger.Error("Failed to handle reorg: %v", err)
					return false
				}
				if err := s.batches.SetLastProcessedBlock(forkPoint); err != nil {
					logger.Warn("Failed to update last processed block: %v", err)
				}
				return false
			}
		}

		// Store block hash for future reorg detection
		if err := s.batches.SetBlockHash(blockNum, block.Hash); err != nil {
			logger.Warn("Failed to store block hash: %v", err)
		}

		allBlocks = append(allBlocks, blockTxs{
			blockNum:  blockNum,
			txs:       block.Transactions,
			blockHash: block.Hash,
		})
	}

	// Process all transactions in order (by block number)
	// This ensures deterministic ordering even after crash/restart
	totalTxs := 0
	for _, block := range allBlocks {
		select {
		case <-s.ctx.Done():
			return false
		default:
		}

		totalTxs += len(block.txs)

		// Process transactions from this block
		remainingTxs := block.txs
		for len(remainingTxs) > 0 {
			select {
			case <-s.ctx.Done():
				return false
			default:
			}

			// Check for incomplete batch first - fill it completely before creating new ones
			incompleteBatch, err := s.batches.GetIncompleteBatch(s.config.BatchSize)
			if err == nil && incompleteBatch != nil {
				spaceLeft := s.config.BatchSize - len(incompleteBatch.Transactions)
				if spaceLeft > 0 {
					txsToAdd := remainingTxs
					if len(txsToAdd) > spaceLeft {
						txsToAdd = remainingTxs[:spaceLeft]
					}

					if err := s.batchBuilder.AppendToBatch(incompleteBatch, txsToAdd, block.blockNum); err != nil {
						logger.Error("Failed to append to batch: %v", err)
						return false
					}

					if err := s.batches.SaveBatch(incompleteBatch); err != nil {
						logger.Error("Failed to save batch: %v", err)
						return false
					}

					// Log batch status
					if len(incompleteBatch.Transactions) >= s.config.BatchSize {
						logger.Info("🎉 Batch #%d COMPLETE: %d transactions (batch hash: %s)",
							incompleteBatch.Number, len(incompleteBatch.Transactions), incompleteBatch.Hash)
						logger.Info("✅ Ready for proof generation!")
					} else {
						logger.Info("📝 Batch #%d incomplete: %d/%d transactions from block #%d",
							incompleteBatch.Number, len(incompleteBatch.Transactions), s.config.BatchSize, block.blockNum)
					}

					if s.api != nil {
						s.api.NotifyNewBatch(incompleteBatch)
					}

					remainingTxs = remainingTxs[len(txsToAdd):]

					// Continue processing remaining transactions
					continue
				}
			}

			// No incomplete batch exists, or it just got filled
			// Create new batches ONLY if we have enough transactions
			if len(remainingTxs) >= s.config.BatchSize {
				// Create full batch
				txsForNewBatch := remainingTxs[:s.config.BatchSize]
				s.createBatchFromTransactions(txsForNewBatch, block.blockNum)
				remainingTxs = remainingTxs[s.config.BatchSize:]
			} else if len(remainingTxs) > 0 {
				// Not enough for full batch - create incomplete batch
				// This will be filled in next iteration
				s.createBatchFromTransactions(remainingTxs, block.blockNum)
				remainingTxs = nil
			}
		}
	}

	logger.Info("✅ Block range #%d to #%d processed: %d total transactions across %d blocks",
		startBlock, endBlock, totalTxs, blockCount)
	return true
}
