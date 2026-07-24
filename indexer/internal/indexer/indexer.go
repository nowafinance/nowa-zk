package indexer

import (
	"context"
	"fmt"
	"sync"
	"time"
	"strings"
	"github.com/nowafinance/nowa-zk/indexer/internal/indexer/types"
	"github.com/nowafinance/nowa-zk/indexer/pkg/errors"
	"github.com/nowafinance/nowa-zk/indexer/pkg/logger"
	"github.com/nowafinance/nowa-zk/indexer/pkg/rpc"
)

// Service is the main indexer service
type Service struct {
	rpcClient    *rpc.Client
	wsClient     *rpc.WebSocketClient
	batchBuilder *BatchBuilder
	batches      *BatchStore
	api          *APIServer
	ctx          context.Context
	cancel       context.CancelFunc
	wg           sync.WaitGroup
	config       *types.Config
}

// DefaultConfig returns default configuration
func DefaultConfig() *types.Config {
	return types.DefaultConfig()
}

// New creates a new indexer service
func New() *Service {
	ctx, cancel := context.WithCancel(context.Background())
	return &Service{
		ctx:    ctx,
		cancel: cancel,
		config: DefaultConfig(),
	}
}

// NewWithConfig creates a new indexer service with custom config
func NewWithConfig(config *types.Config) *Service {
	ctx, cancel := context.WithCancel(context.Background())
	return &Service{
		ctx:    ctx,
		cancel: cancel,
		config: config,
	}
}

// Start initializes and starts the indexer service
func (s *Service) Start() error {
	logger.Info("🚀 Starting ZK Indexer...")

	// Initialize RPC client
	var err error
	if s.config.RPCURL != "" {
		s.rpcClient, err = rpc.NewClient(s.config.RPCURL)
		if err != nil {
			return errors.ErrRPC("failed to create RPC client", err)
		}
	} else {
		s.rpcClient, err = rpc.NewClientFromEnv()
		if err != nil {
			return errors.ErrRPC("failed to create RPC client", err)
		}
	}
	logger.Info("✅ Connected to Nowa-ZK RPC endpoint")

	// Initialize WebSocket client (optional)
	if s.config.WSURL != "" {
		s.wsClient, err = rpc.NewWebSocketClient(s.config.WSURL)
		if err != nil {
			logger.Warn("WebSocket connection failed: %v (continuing with HTTP polling)", err)
		} else {
			logger.Info("✅ Connected to Nowa-ZK WebSocket endpoint")
		}
	}

	// Initialize batch store first (needed for batch builder initialization)
	s.batches, err = NewBatchStore(s.config.StateDBPath)
	if err != nil {
		return errors.ErrBatchStore("failed to create batch store", err)
	}
	logger.Info("✅ Batch store initialized")

	// Initialize batch builder with last batch number and state root
	initialBatchNum := uint64(1)
	initialStateRoot := "0x0000000000000000000000000000000000000000000000000000000000000001"

	// Check for incomplete batch first (to resume filling it)
	incompleteBatch, err := s.batches.GetIncompleteBatch(s.config.BatchSize)
	if err == nil && incompleteBatch != nil {
		// Resume from incomplete batch
		initialBatchNum = incompleteBatch.Number
		initialStateRoot = incompleteBatch.NewStateRoot
		logger.Info("📦 Resuming incomplete batch #%d (%d/%d transactions) with state root %s",
			incompleteBatch.Number, len(incompleteBatch.Transactions), s.config.BatchSize, initialStateRoot)
	} else if lastBatch, err := s.batches.GetLatestBatch(); err == nil {
		// No incomplete batch, resume from last complete batch
		initialBatchNum = lastBatch.Number + 1
		initialStateRoot = lastBatch.NewStateRoot
		logger.Info("📦 Resuming from batch #%d with state root %s", initialBatchNum, initialStateRoot)
	}

	s.batchBuilder = NewBatchBuilder(initialBatchNum, initialStateRoot, s.rpcClient, s.ctx)
	logger.Info("✅ Batch builder initialized")

	// Initialize REST API server
	s.api = NewAPIServer(s.batches, s.config.APIPort)
	logger.Info("✅ REST API server initialized on port %d", s.config.APIPort)

	// Start REST API server
	s.wg.Add(1)
	go func() {
		defer s.wg.Done()
		if err := s.api.Start(); err != nil {
			logger.Error("API server error: %v", err)
		}
	}()

	// Start block subscription
	s.wg.Add(1)
	go s.subscribeToBlocks()

	// Start cleanup job (if ProverAPI is configured)
	if s.config.ProverAPI != "" {
		s.wg.Add(1)
		go s.runCleanupJob()
		logger.Info("🧹 Cleanup job scheduled (queries prover at %s every 5 minutes)", s.config.ProverAPI)
	}

	logger.Info("✅ ZK Indexer is running!")
	logger.Info("📡 Listening for new blocks and building batches...")
	logger.Info("🌐 REST API available at http://localhost:%d", s.config.APIPort)

	return nil
}

// Stop gracefully stops the indexer service
func (s *Service) Stop() error {
	logger.Info("🛑 Stopping ZK Indexer...")

	// Cancel context to stop all goroutines
	s.cancel()

	// Stop API server first (with timeout)
	if s.api != nil {
		if err := s.api.Stop(); err != nil {
			logger.Warn("Error stopping API server: %v", err)
		}
	}

	// Close batch store
	if s.batches != nil {
		if err := s.batches.Close(); err != nil {
			logger.Warn("Error closing batch store: %v", err)
		}
	}

	// Close RPC clients
	if s.rpcClient != nil {
		s.rpcClient.Close()
	}
	if s.wsClient != nil {
		s.wsClient.Close()
	}

	// Wait for all goroutines to finish (with timeout)
	done := make(chan struct{})
	go func() {
		s.wg.Wait()
		close(done)
	}()

	select {
	case <-done:
		logger.Info("✅ ZK Indexer stopped")
	case <-time.After(5 * time.Second):
		logger.Warn("Timeout waiting for goroutines to finish")
	}

	return nil
}

// ClearAllData clears all batches, block hashes, and state data
func (s *Service) ClearAllData() error {
	// Open batch store temporarily to clear data
	store, err := NewBatchStore(s.config.StateDBPath)
	if err != nil {
		return fmt.Errorf("failed to open batch store for clearing: %w", err)
	}
	defer store.Close()

	// Clear all data
	if err := store.ClearAll(); err != nil {
		return fmt.Errorf("failed to clear data: %w", err)
	}

	return nil
}

// subscribeToBlocks subscribes to new blocks and processes transactions
func (s *Service) subscribeToBlocks() {
	defer s.wg.Done()

	// Always use polling to prevent missing values in low network
	s.pollBlocks()
}

// subscribeViaWebSocket subscribes to blocks via WebSocket
func (s *Service) subscribeViaWebSocket() {
	// First, catch up on any unprocessed blocks
	s.catchUpBlocks()

	// Then subscribe to new blocks
	headerChan, err := s.wsClient.SubscribeNewHeads(s.ctx)
	if err != nil {
		logger.Error("Failed to subscribe to blocks: %v, falling back to polling", err)
		s.pollBlocks()
		return
	}

	logger.Info("📡 Subscribed to new block headers via WebSocket")

	for {
		select {
		case <-s.ctx.Done():
			return
		case header, ok := <-headerChan:
			if !ok {
				logger.Warn("WebSocket channel closed, falling back to polling")
				s.pollBlocks()
				return
			}
			blockNum := header.Number.Uint64()
			// Process block - only update last processed block if successful
			if s.processBlock(blockNum) {
				// Update last processed block only after successful processing
				if err := s.batches.SetLastProcessedBlock(blockNum); err != nil {
					logger.Warn("Failed to save last processed block: %v", err)
				}
			} else {
				logger.Warn("Block #%d processing incomplete (will retry)", blockNum)
			}
		}
	}
}

// catchUpBlocks processes any unprocessed blocks on startup
func (s *Service) catchUpBlocks() {
	// Check context first
	select {
	case <-s.ctx.Done():
		logger.Warn("Context canceled, skipping catch-up")
		return
	default:
	}

	// Determine starting block
	lastBlockNum := s.initStartingBlock()

	// Get current block number
	currentBlockNum, err := s.rpcClient.BlockNumber(s.ctx)
	if err != nil {
		// If context canceled, just skip catch-up and continue
		if s.ctx.Err() != nil {
			logger.Warn("Context canceled during catch-up, continuing to subscription")
			return
		}
		logger.Warn("Failed to get current block number: %v, skipping catch-up", err)
		return
	}

	if currentBlockNum > lastBlockNum {
		blocksBehind := currentBlockNum - lastBlockNum
		logger.Info("📦 Catching up on %d unprocessed blocks...", blocksBehind)

		// Process blocks in chunks of 100 for deterministic ordering
		for startBlock := lastBlockNum + 1; startBlock <= currentBlockNum; {
			select {
			case <-s.ctx.Done():
				logger.Warn("Context canceled during catch-up, stopping at block #%d", startBlock-1)
				return
			default:
				// Process up to 100 blocks at once
				endBlock := startBlock + 99
				if endBlock > currentBlockNum {
					endBlock = currentBlockNum
				}

				if s.processBlockRange(startBlock, endBlock) {
					// Successfully processed range, update last processed block
					if err := s.batches.SetLastProcessedBlock(endBlock); err != nil {
						logger.Warn("Failed to save last processed block: %v", err)
					}
					startBlock = endBlock + 1
				} else {
					logger.Warn("Stopping catch-up at block #%d (will retry)", startBlock)
					return
				}
			}
		}
		logger.Info("✅ Caught up to block #%d", currentBlockNum)
	} else {
		logger.Info("✅ Already up to date at block #%d", currentBlockNum)
	}
}

// pollBlocks polls for new blocks periodically
func (s *Service) pollBlocks() {
	logger.Info("📡 Polling for new blocks every 2 seconds (processing 100 blocks per batch)")

	// Determine starting block - initialize ONCE
	lastBlockNum := s.initStartingBlock()

	ticker := time.NewTicker(2 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-s.ctx.Done():
			return
		case <-ticker.C:
			// Get current block number from chain
			blockNum, err := s.rpcClient.BlockNumber(s.ctx)
			if err != nil {
				logger.Warn("Failed to get block number: %v", err)
				continue
			}

			if blockNum > lastBlockNum {
				// Process blocks in chunks of 100
				for startBlock := lastBlockNum + 1; startBlock <= blockNum; startBlock += 100 {
					select {
					case <-s.ctx.Done():
						return
					default:
						// Process up to 100 blocks at once
						endBlock := startBlock + 99
						if endBlock > blockNum {
							endBlock = blockNum
						}

						// Process block range - only advance if successfully processed
						if s.processBlockRange(startBlock, endBlock) {
							// Update last processed block only after successful processing
							if err := s.batches.SetLastProcessedBlock(endBlock); err != nil {
								logger.Warn("Failed to save last processed block: %v", err)
							}
							lastBlockNum = endBlock
						} else {
							// Block range processing failed, stop and retry on next poll
							logger.Warn("Stopping block processing at block #%d (will retry)", startBlock)
							break
						}
					}
				}
			}
		}
	}
}

// processBlock fetches and processes a block
// Returns true if block was fully processed, false if incomplete batch prevented processing
func (s *Service) processBlock(blockNum uint64) bool {
	// Check if context is already canceled
	select {
	case <-s.ctx.Done():
		return false
	default:
	}

	logger.Info("📦 Processing block #%d", blockNum)

	// Fetch block with transactions
	block, err := s.rpcClient.GetBlockByNumber(s.ctx, blockNum, true)
	if err != nil {
		// Don't log context canceled errors during shutdown
		if s.ctx.Err() == nil {
			logger.Warn("Failed to fetch block %d: %v", blockNum, err)
		}
		return false
	}

	// Check for reorg: compare block hash with stored hash
	if storedHash, err := s.batches.GetBlockHash(blockNum); err == nil {
		if storedHash != block.Hash {
			// Reorg detected!
			logger.Warn("Reorg detected at block #%d: stored hash %s != new hash %s", blockNum, storedHash, block.Hash)

			// Find fork point and rollback
			forkPoint, err := s.findForkPoint(blockNum)
			if err != nil {
				logger.Error("Failed to find fork point: %v", err)
				return false
			}

			if err := s.handleReorg(forkPoint); err != nil {
				logger.Error("Failed to handle reorg: %v", err)
				return false
			}

			logger.Info("Rolled back to block #%d, will re-process from block #%d", forkPoint, forkPoint+1)

			// Update last processed block to fork point
			if err := s.batches.SetLastProcessedBlock(forkPoint); err != nil {
				logger.Warn("Failed to update last processed block: %v", err)
			}

			// Return false to indicate we need to re-process from fork point
			return false
		}
	}

	// Store block hash for future reorg detection
	if err := s.batches.SetBlockHash(blockNum, block.Hash); err != nil {
		logger.Warn("Failed to store block hash: %v", err)
	}

	// Filter transactions for executeBatchTradeMultiFill directed to Nowa_Orderbook
	var filteredTxs []*rpc.Transaction
	targetAddress := strings.ToLower(s.config.TargetContract)
	targetSelector := "0x7864fe7a"

	for _, tx := range block.Transactions {
		// Check context before processing each transaction
		select {
		case <-s.ctx.Done():
			return false
		default:
		}

		if strings.ToLower(tx.To) == targetAddress && strings.HasPrefix(tx.Data, targetSelector) {
			filteredTxs = append(filteredTxs, tx)
		}
	}
	block.Transactions = filteredTxs

	// Process transactions sequentially to fill incomplete batches first
	remainingTxs := block.Transactions
	for len(remainingTxs) > 0 {
		select {
		case <-s.ctx.Done():
			return false
		default:
		}

		// Check for incomplete batch first
		incompleteBatch, err := s.batches.GetIncompleteBatch(s.config.BatchSize)
		if err == nil && incompleteBatch != nil {
			// Fill incomplete batch
			spaceLeft := s.config.BatchSize - len(incompleteBatch.Trades)
			if spaceLeft > 0 {
				var txsToAdd []*rpc.Transaction
				tradesAdded := 0
				
				for _, tx := range remainingTxs {
					trades, _ := ParseTradesFromTxData(tx.Data)
					if tradesAdded > 0 && tradesAdded+len(trades) > spaceLeft {
						break // Can't fit more trades without going over (if we already added some)
					}
					txsToAdd = append(txsToAdd, tx)
					tradesAdded += len(trades)
					if tradesAdded >= spaceLeft {
						break
					}
				}

				if len(txsToAdd) > 0 {
					// Append to incomplete batch
					if err := s.batchBuilder.AppendToBatch(incompleteBatch, txsToAdd, blockNum); err != nil {
						logger.Error("Failed to append to batch: %v", err)
						return false
					}

					// Save updated batch
					if err := s.batches.SaveBatch(incompleteBatch); err != nil {
						logger.Error("Failed to save batch: %v", err)
						return false
					}

					// Notify WebSocket clients
					if s.api != nil {
						s.api.NotifyNewBatch(incompleteBatch)
					}

					logger.Info("📝 Appended %d transactions (%d trades) to batch #%d (total trades: %d/%d)",
						len(txsToAdd), tradesAdded, incompleteBatch.Number, len(incompleteBatch.Trades), s.config.BatchSize)

					remainingTxs = remainingTxs[len(txsToAdd):]
				}

				// If batch is now complete, log it
				if len(incompleteBatch.Trades) >= s.config.BatchSize {
					logger.Info("✅ Batch #%d completed with %d trades", incompleteBatch.Number, len(incompleteBatch.Trades))
					continue
				} else {
					if len(remainingTxs) == 0 {
						logger.Info("⏸️  Batch #%d incomplete (%d/%d trades), all transactions from block added",
							incompleteBatch.Number, len(incompleteBatch.Trades), s.config.BatchSize)
					}
				}
			}
		} else {
			// No incomplete batch, create new batch with remaining transactions
			if len(remainingTxs) > 0 {
				var txsForNewBatch []*rpc.Transaction
				tradesAdded := 0
				
				for _, tx := range remainingTxs {
					trades, _ := ParseTradesFromTxData(tx.Data)
					if tradesAdded > 0 && tradesAdded+len(trades) > s.config.BatchSize {
						break
					}
					txsForNewBatch = append(txsForNewBatch, tx)
					tradesAdded += len(trades)
					if tradesAdded >= s.config.BatchSize {
						break
					}
				}

				s.createBatchFromTransactions(txsForNewBatch, blockNum)
				remainingTxs = remainingTxs[len(txsForNewBatch):]
			}
		}
	}

	logger.Info("✅ Block #%d processed: %d relevant transactions", blockNum, len(block.Transactions))
	return true
}

// findForkPoint finds the last block where hash matches (common ancestor)
func (s *Service) findForkPoint(currentBlock uint64) (uint64, error) {
	// Start from current block and go backwards
	for blockNum := currentBlock; blockNum > 0; blockNum-- {
		// Get block from blockchain
		block, err := s.rpcClient.GetBlockByNumber(s.ctx, blockNum, false)
		if err != nil {
			return 0, fmt.Errorf("failed to get block %d: %w", blockNum, err)
		}

		// Check if we have this block hash stored
		storedHash, err := s.batches.GetBlockHash(blockNum)
		if err != nil {
			// Block not stored yet, continue backwards
			continue
		}

		// If hashes match, this is the fork point
		if storedHash == block.Hash {
			return blockNum, nil
		}
	}

	// No matching block found, fork point is genesis (0)
	return 0, nil
}

// handleReorg handles blockchain reorganization by rolling back batches
func (s *Service) handleReorg(forkPoint uint64) error {
	logger.Info("Handling reorg: rolling back to block #%d", forkPoint)

	// Get latest batch to determine rollback point
	latestBatch, err := s.batches.GetLatestBatch()
	if err != nil {
		// No batches to rollback
		return nil
	}

	if latestBatch != nil {
		logger.Info("Rolling back batches after batch #%d", latestBatch.Number)

		if err := s.batches.DeleteBatchesAfter(latestBatch.Number); err != nil {
			return fmt.Errorf("failed to delete batches: %w", err)
		}
	}

	return nil
}

// createBatchFromTransactions creates a new batch directly from provided transactions
// Only finalizes the batch if it has EXACTLY or GREATER THAN config.BatchSize trades
func (s *Service) createBatchFromTransactions(txs []*rpc.Transaction, blockNumber uint64) {
	// Check if context is canceled
	select {
	case <-s.ctx.Done():
		return
	default:
	}

	if len(txs) == 0 {
		return // No transactions to batch
	}

	batch, err := s.batchBuilder.BuildBatch(txs, blockNumber)
	if err != nil {
		if s.ctx.Err() == nil {
			logger.Error("Failed to build batch: %v", err)
		}
		return
	}

	// Store batch
	if err := s.batches.SaveBatch(batch); err != nil {
		if s.ctx.Err() == nil {
			logger.Error("Failed to save batch: %v", err)
		}
		return
	}

	if len(batch.Trades) >= s.config.BatchSize {
		// Notify WebSocket clients about new batch
		if s.api != nil {
			s.api.NotifyNewBatch(batch)
		}

		logger.Info("🎉 Batch #%d COMPLETE: %d trades (batch hash: %s)",
			batch.Number, len(batch.Trades), batch.Hash)
		logger.Info("✅ Ready for proof generation!")
	} else {
		// Batch is incomplete, just save it silently
		logger.Info("📝 Batch #%d incomplete: %d/%d trades",
			batch.Number, len(batch.Trades), s.config.BatchSize)
	}
}

// Info returns service information
func (s *Service) Info() string {
	if s.batches == nil {
		return "zk-indexer initialized (not started)"
	}
	return "zk-indexer running"
}

// initStartingBlock determines the starting block number and persists it to the DB if needed.
// Priority:
// 1. Database (Last Processed Block)
// 2. Configuration (INDEX_FROM_BLOCK)
// 3. Default (0)
// Returns the block number representing the *last processed block*.
func (s *Service) initStartingBlock() uint64 {
	// 1. Try to load from database
	lastBlock, err := s.batches.GetLastProcessedBlock()
	if err == nil && lastBlock > 0 {
		logger.Info("📦 Resuming from block #%d (found in DB)", lastBlock)
		return lastBlock
	}

	if err != nil {
		logger.Warn("Failed to get last processed block from DB: %v (will check config)", err)
	}

	var startFrom uint64

	// 2. Check configuration (Env / flag)
	if s.config.IndexFromBlock > 0 {
		// We subtract 1 because the caller will start processing at lastBlock + 1
		startFrom = s.config.IndexFromBlock - 1
		logger.Info("📦 Configuration overrides start block: starting from #%d (INDEX_FROM_BLOCK=%d)",
			s.config.IndexFromBlock, s.config.IndexFromBlock)
	} else {
		// 3. Default to 0
		logger.Info("📦 Starting from block #0 (default)")
		startFrom = 0
	}

	// Persist this decision to DB so it is "initialized"
	if err := s.batches.SetLastProcessedBlock(startFrom); err != nil {
		logger.Warn("Failed to persist initial start block #%d to DB: %v", startFrom, err)
	} else {
		logger.Info("💾 Persisted initial start block #%d to DB", startFrom)
	}

	return startFrom
}

// runCleanupJob periodically queries prover and deletes old batch data
func (s *Service) runCleanupJob() {
	defer s.wg.Done()

	// Create prover client
	proverClient := NewProverClient(s.config.ProverAPI)

	// Run cleanup every 5 minutes
	ticker := time.NewTicker(5 * time.Minute)
	defer ticker.Stop()

	logger.Info("🧹 Cleanup job started (interval: 5 minutes)")

	for {
		select {
		case <-s.ctx.Done():
			logger.Info("🧹 Cleanup job stopped")
			return
		case <-ticker.C:
			logger.Info("🧹 Running scheduled cleanup...")
			if err := s.batches.CleanupOldBatches(proverClient); err != nil {
				logger.Warn("Cleanup failed: %v", err)
			}
		}
	}
}
