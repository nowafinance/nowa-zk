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
	app.Use(cors.New(cors.Config{
		AllowOrigins: "*",
		AllowMethods: "GET,POST,HEAD,PUT,DELETE,PATCH,OPTIONS",
		AllowHeaders: "Origin,Content-Type,Accept,Authorization",
	}))

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
	api.app.Get("/batches", api.handleGetBatches)
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

// handleGetBatches godoc
// @Summary Get batches with pagination
// @Description Retrieve multiple batch metadata entries with pagination
// @Tags Batches
// @Param page query int false "Page number (default: 1)"
// @Param limit query int false "Items per page - 10, 25, 50, or 100 (default: 25)"
// @Produce json
// @Success 200 {object} map[string]interface{} "Paginated batch list"
// @Failure 400 {string} string "Invalid parameters"
// @Failure 500 {string} string "Server error"
// @Router /batches [get]
func (api *APIServer) handleGetBatches(c *fiber.Ctx) error {
	// Parse pagination parameters
	page := c.QueryInt("page", 1)
	limit := c.QueryInt("limit", 25)

	// Validate page
	if page < 1 {
		return c.Status(fiber.StatusBadRequest).SendString("Page must be >= 1")
	}

	// Validate and normalize limit
	validLimits := []int{10, 25, 50, 100}
	validLimit := false
	for _, vl := range validLimits {
		if limit == vl {
			validLimit = true
			break
		}
	}
	if !validLimit {
		// Default to closest valid limit
		if limit < 10 {
			limit = 10
		} else if limit < 25 {
			limit = 25
		} else if limit < 50 {
			limit = 50
		} else {
			limit = 100
		}
	}

	// Calculate offset
	offset := uint64((page - 1) * limit)

	// Fetch batches from storage
	batches, err := api.store.GetBatches(offset, uint64(limit))
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).SendString(fmt.Sprintf("Error fetching batches: %v", err))
	}

	// Return paginated response
	return c.JSON(fiber.Map{
		"page":    page,
		"limit":   limit,
		"count":   len(batches),
		"batches": batches,
	})
}
