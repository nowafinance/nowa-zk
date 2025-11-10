package sequencer

import (
	"context"
	"fmt"
	"log"
	"sync"
	"time"

	"github.com/tannetwork/zk-sequencer/sequencer/pkg/rpc"
)

// Service is the main sequencer service
type Service struct {
	rpcClient    *rpc.Client
	wsClient     *rpc.WebSocketClient
	txPool       *TransactionPool
	batchBuilder *BatchBuilder
	batches      *BatchStore
	api          *APIServer
	ctx          context.Context
	cancel       context.CancelFunc
	wg           sync.WaitGroup
	config       *Config
}

// Config holds sequencer configuration
type Config struct {
	RPCURL          string
	WSURL           string
	BatchSize       int
	BatchInterval   time.Duration
	APIPort         int
	StateDBPath     string
}

// DefaultConfig returns default configuration
func DefaultConfig() *Config {
	return &Config{
		BatchSize:     100,              // 100 transactions per batch
		BatchInterval: 10 * time.Second, // Create batch every 10 seconds
		APIPort:       8080,              // REST API port
		StateDBPath:   "./data",         // Local storage path
	}
}

// New creates a new sequencer service
func New() *Service {
	ctx, cancel := context.WithCancel(context.Background())
	return &Service{
		ctx:    ctx,
		cancel: cancel,
		config: DefaultConfig(),
	}
}

// NewWithConfig creates a new sequencer service with custom config
func NewWithConfig(config *Config) *Service {
	ctx, cancel := context.WithCancel(context.Background())
	return &Service{
		ctx:    ctx,
		cancel: cancel,
		config: config,
	}
}

// Start initializes and starts the sequencer service
func (s *Service) Start() error {
	log.Println("🚀 Starting ZK Sequencer...")

	// Initialize RPC client
	var err error
	s.rpcClient, err = rpc.NewClientFromEnv()
	if err != nil {
		return fmt.Errorf("failed to create RPC client: %w", err)
	}
	log.Println("✅ Connected to Tan-ZK RPC endpoint")

	// Initialize WebSocket client (optional)
	if s.config.WSURL != "" {
		s.wsClient, err = rpc.NewWebSocketClientFromEnv()
		if err != nil {
			log.Printf("⚠️  WebSocket connection failed: %v (continuing with HTTP polling)", err)
		} else {
			log.Println("✅ Connected to Tan-ZK WebSocket endpoint")
		}
	}

	// Initialize transaction pool
	s.txPool = NewTransactionPool()
	log.Println("✅ Transaction pool initialized")

	// Initialize batch builder
	s.batchBuilder = NewBatchBuilder(s.txPool, s.config.BatchSize)
	log.Println("✅ Batch builder initialized")

	// Initialize batch store
	s.batches = NewBatchStore(s.config.StateDBPath)
	log.Println("✅ Batch store initialized")

	// Initialize REST API server
	s.api = NewAPIServer(s.batches, s.config.APIPort)
	log.Printf("✅ REST API server initialized on port %d", s.config.APIPort)

	// Start REST API server
	s.wg.Add(1)
	go func() {
		defer s.wg.Done()
		if err := s.api.Start(); err != nil {
			log.Printf("❌ API server error: %v", err)
		}
	}()

	// Start block subscription
	s.wg.Add(1)
	go s.subscribeToBlocks()

	// Start batch creation loop
	s.wg.Add(1)
	go s.batchCreationLoop()

	log.Println("✅ ZK Sequencer is running!")
	log.Println("📡 Listening for new blocks and building batches...")
	log.Printf("🌐 REST API available at http://localhost:%d", s.config.APIPort)

	return nil
}

// Stop gracefully stops the sequencer service
func (s *Service) Stop() error {
	log.Println("🛑 Stopping ZK Sequencer...")

	// Cancel context to stop all goroutines
	s.cancel()

	// Close RPC clients
	if s.rpcClient != nil {
		s.rpcClient.Close()
	}
	if s.wsClient != nil {
		s.wsClient.Close()
	}

	// Wait for all goroutines to finish
	s.wg.Wait()

	log.Println("✅ ZK Sequencer stopped")
	return nil
}

// subscribeToBlocks subscribes to new blocks and processes transactions
func (s *Service) subscribeToBlocks() {
	defer s.wg.Done()

	// Use WebSocket if available, otherwise poll
	if s.wsClient != nil {
		s.subscribeViaWebSocket()
	} else {
		s.pollBlocks()
	}
}

// subscribeViaWebSocket subscribes to blocks via WebSocket
func (s *Service) subscribeViaWebSocket() {
	headerChan, err := s.wsClient.SubscribeNewHeads(s.ctx)
	if err != nil {
		log.Printf("❌ Failed to subscribe to blocks: %v, falling back to polling", err)
		s.pollBlocks()
		return
	}

	log.Println("📡 Subscribed to new block headers via WebSocket")

	for {
		select {
		case <-s.ctx.Done():
			return
		case header, ok := <-headerChan:
			if !ok {
				log.Println("⚠️  WebSocket channel closed, falling back to polling")
				s.pollBlocks()
				return
			}
			s.processBlock(header.Number.Uint64())
		}
	}
}

// pollBlocks polls for new blocks periodically
func (s *Service) pollBlocks() {
	log.Println("📡 Polling for new blocks (every 2 seconds)")

	lastBlockNum := uint64(0)

	ticker := time.NewTicker(2 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-s.ctx.Done():
			return
		case <-ticker.C:
			blockNum, err := s.rpcClient.BlockNumber(s.ctx)
			if err != nil {
				log.Printf("⚠️  Failed to get block number: %v", err)
				continue
			}

			if blockNum > lastBlockNum {
				// Process all new blocks
				for i := lastBlockNum + 1; i <= blockNum; i++ {
					s.processBlock(i)
				}
				lastBlockNum = blockNum
			}
		}
	}
}

// processBlock fetches and processes a block
func (s *Service) processBlock(blockNum uint64) {
	log.Printf("📦 Processing block #%d", blockNum)

	// Fetch block with transactions
	block, err := s.rpcClient.GetBlockByNumber(s.ctx, blockNum, true)
	if err != nil {
		log.Printf("⚠️  Failed to fetch block %d: %v", blockNum, err)
		return
	}

	// Add transactions to pool
	for _, tx := range block.Transactions {
		s.txPool.AddTransaction(tx)
	}

	log.Printf("✅ Block #%d processed: %d transactions added to pool (pool size: %d)",
		blockNum, len(block.Transactions), s.txPool.Size())
}

// batchCreationLoop periodically creates batches from the transaction pool
func (s *Service) batchCreationLoop() {
	defer s.wg.Done()

	ticker := time.NewTicker(s.config.BatchInterval)
	defer ticker.Stop()

	log.Printf("🔄 Batch creation loop started (interval: %v)", s.config.BatchInterval)

	for {
		select {
		case <-s.ctx.Done():
			return
		case <-ticker.C:
			s.createBatch()
		}
	}
}

// createBatch creates a new batch from the transaction pool
func (s *Service) createBatch() {
	if s.txPool.Size() == 0 {
		return // No transactions to batch
	}

	batch, err := s.batchBuilder.BuildBatch()
	if err != nil {
		log.Printf("⚠️  Failed to build batch: %v", err)
		return
	}

	// Store batch
	if err := s.batches.SaveBatch(batch); err != nil {
		log.Printf("⚠️  Failed to save batch: %v", err)
		return
	}

	log.Printf("✅ Batch #%d created: %d transactions (batch hash: %s)",
		batch.Number, len(batch.Transactions), batch.Hash)

	// TODO: Request proof from prover
	// This will be implemented when prover service is ready
}

// Info returns service information
func (s *Service) Info() string {
	if s.txPool == nil {
		return "zk-sequencer initialized (not started)"
	}
	return fmt.Sprintf("zk-sequencer running - Pool: %d txs, Batches: %d",
		s.txPool.Size(), s.batches.Count())
}
