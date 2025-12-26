package api

import (
	"encoding/json"
	"fmt"
	"math/big"
	"strconv"

	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/cors"
	"github.com/gofiber/fiber/v2/middleware/logger"
	"github.com/gofiber/swagger"
	"github.com/tannetwork/zk-sequencer/prover/bindings"
	_ "github.com/tannetwork/zk-sequencer/prover/docs" // Swagger docs
	"github.com/tannetwork/zk-sequencer/prover/internal/storage"
)

// APIServer serves the Prover API
type APIServer struct {
	app      *fiber.App
	registry *bindings.BatchRegistry
	store    *storage.ProverStore
	port     int
}

// BatchResponse represents the batch data returned by API
type BatchResponse struct {
	BatchNumber  uint64   `json:"batchNumber"`
	BatchHash    string   `json:"batchHash"`
	OldStateRoot string   `json:"oldStateRoot"`
	NewStateRoot string   `json:"newStateRoot"`
	Submitter    string   `json:"submitter"`
	Timestamp    uint64   `json:"timestamp"`
	VerifiedAt   uint64   `json:"verifiedAt"`
	Status       uint8    `json:"status"`
	TxHash       string   `json:"txHash,omitempty"`
	TxHashes     []string `json:"txHashes,omitempty"` // L2 transaction hashes
}

// StatusResponse represents the proof status
type StatusResponse struct {
	Status      string `json:"status"`
	BatchNumber uint64 `json:"batch_number"`
	Timestamp   int64  `json:"timestamp,omitempty"`
	Note        string `json:"note,omitempty"`
}

// NewAPIServer creates a new API server
func NewAPIServer(registry *bindings.BatchRegistry, store *storage.ProverStore, port int) *APIServer {
	app := fiber.New(fiber.Config{
		DisableStartupMessage: true,
		AppName:               "Tan-ZK Prover",
		JSONEncoder:           json.Marshal,
		JSONDecoder:           json.Unmarshal,
	})

	// Middleware
	app.Use(logger.New())
	app.Use(cors.New())

	return &APIServer{
		app:      app,
		registry: registry,
		store:    store,
		port:     port,
	}
}

// Start starts the API server
// @title Tan-ZK Prover API
// @version 1.0
// @description API for the Tan-ZK Rollup Prover
// @BasePath /
func (api *APIServer) Start() error {
	// Swagger
	api.app.Get("/swagger/*", swagger.New(swagger.Config{
		Title:       "Tan-ZK Prover API",
		CustomStyle: `.swagger-ui .topbar { display: none }`,
	}))

	// Root endpoint
	api.app.Get("/", api.handleRoot)

	// Batches
	api.app.Get("/batches/latest", api.handleLatestBatch)
	api.app.Get("/batches/:id", api.handleGetBatch)

	// Status
	api.app.Get("/status/:id", api.handleGetStatus)

	addr := fmt.Sprintf(":%d", api.port)
	fmt.Printf("🌍 Starting Prover API on %s\n", addr)
	fmt.Printf("📖 Swagger UI available at http://localhost%s/swagger/index.html\n", addr)

	return api.app.Listen(addr)
}

// handleRoot godoc
// @Summary List available endpoints
// @Description Redirects to Swagger UI
// @Tags Meta
// @Success 302
// @Router / [get]
func (api *APIServer) handleRoot(c *fiber.Ctx) error {
	return c.Redirect("/swagger/index.html")
}

// handleLatestBatch godoc
// @Summary Get latest verified batch
// @Description Get details of the latest verified batch from the contract
// @Tags Batches
// @Produce json
// @Success 200 {object} BatchResponse
// @Failure 404 {string} string "No batches found"
// @Failure 500 {string} string "Internal Server Error"
// @Router /batches/latest [get]
func (api *APIServer) handleLatestBatch(c *fiber.Ctx) error {
	totalBatches, err := api.registry.TotalBatches(&bind.CallOpts{})
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).SendString(fmt.Sprintf("Failed to get total batches: %v", err))
	}

	if totalBatches.Cmp(big.NewInt(0)) == 0 {
		return c.Status(fiber.StatusNotFound).SendString("No batches found")
	}

	batch, err := api.registry.GetBatch(&bind.CallOpts{}, totalBatches)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).SendString(fmt.Sprintf("Failed to get batch: %v", err))
	}

	// Try to get tx hash and tx hashes from local storage
	txHash := ""
	var txHashes []string
	if proofData, err := api.store.GetProof(totalBatches.Uint64()); err == nil && proofData != nil {
		txHash = proofData.TxHash
		txHashes = proofData.TxHashes
	}

	resp := BatchResponse{
		BatchNumber:  totalBatches.Uint64(),
		BatchHash:    common.BytesToHash(batch.BatchHash[:]).Hex(),
		OldStateRoot: common.BytesToHash(batch.OldStateRoot[:]).Hex(),
		NewStateRoot: common.BytesToHash(batch.NewStateRoot[:]).Hex(),
		Submitter:    batch.Submitter.Hex(),
		Timestamp:    batch.Timestamp.Uint64(),
		VerifiedAt:   batch.VerifiedAt.Uint64(),
		Status:       batch.Status,
		TxHash:       txHash,
		TxHashes:     txHashes,
	}

	return c.JSON(resp)
}

// handleGetBatch godoc
// @Summary Get batch by ID
// @Description Get details of a specific batch
// @Tags Batches
// @Param id path int true "Batch ID"
// @Produce json
// @Success 200 {object} BatchResponse
// @Failure 400 {string} string "Invalid batch ID"
// @Failure 500 {string} string "Internal Server Error"
// @Router /batches/{id} [get]
func (api *APIServer) handleGetBatch(c *fiber.Ctx) error {
	idStr := c.Params("id")
	batchID, ok := new(big.Int).SetString(idStr, 10)
	if !ok {
		return c.Status(fiber.StatusBadRequest).SendString("Invalid batch ID")
	}

	batch, err := api.registry.GetBatch(&bind.CallOpts{}, batchID)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).SendString(fmt.Sprintf("Failed to get batch: %v", err))
	}

	// Try to get tx hash and tx hashes from local storage
	txHash := ""
	var txHashes []string
	if proofData, err := api.store.GetProof(batchID.Uint64()); err == nil && proofData != nil {
		txHash = proofData.TxHash
		txHashes = proofData.TxHashes
	}

	resp := BatchResponse{
		BatchNumber:  batchID.Uint64(),
		BatchHash:    common.BytesToHash(batch.BatchHash[:]).Hex(),
		OldStateRoot: common.BytesToHash(batch.OldStateRoot[:]).Hex(),
		NewStateRoot: common.BytesToHash(batch.NewStateRoot[:]).Hex(),
		Submitter:    batch.Submitter.Hex(),
		Timestamp:    batch.Timestamp.Uint64(),
		VerifiedAt:   batch.VerifiedAt.Uint64(),
		Status:       batch.Status,
		TxHash:       txHash,
		TxHashes:     txHashes,
	}

	return c.JSON(resp)
}

// handleGetStatus godoc
// @Summary Get proof status
// @Description Get proof generation status for a batch
// @Tags Proofs
// @Param id path int true "Batch ID"
// @Produce json
// @Success 200 {object} StatusResponse
// @Failure 400 {string} string "Invalid batch ID"
// @Router /status/{id} [get]
func (api *APIServer) handleGetStatus(c *fiber.Ctx) error {
	idStr := c.Params("id")
	batchNumber, err := strconv.ParseUint(idStr, 10, 64)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).SendString("Invalid batch ID")
	}

	// Check ProverStore specifically for local proof status
	proof, err := api.store.GetProof(batchNumber)
	if err == nil && proof != nil {
		return c.JSON(StatusResponse{
			Status:      "COMPLETED",
			BatchNumber: batchNumber,
			Timestamp:   proof.Timestamp,
		})
	}

	// If not found in local store, check contract
	batchID := new(big.Int).SetUint64(batchNumber)
	batch, err := api.registry.GetBatch(&bind.CallOpts{}, batchID)
	if err == nil && batch.VerifiedAt.Uint64() > 0 {
		return c.JSON(StatusResponse{
			Status:      "VERIFIED_ON_CHAIN",
			BatchNumber: batchNumber,
			Note:        "Proof verified but local proof data not found",
		})
	}

	// Default to PENDING
	return c.JSON(StatusResponse{
		Status:      "PENDING",
		BatchNumber: batchNumber,
	})
}
