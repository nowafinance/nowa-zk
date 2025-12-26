package sequencer

import (
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/cors"
	"github.com/gofiber/fiber/v2/middleware/logger"
	"github.com/gofiber/swagger"
	"github.com/gofiber/websocket/v2"
	_ "github.com/tannetwork/zk-sequencer/sequencer/docs" // Swagger docs
	"github.com/tannetwork/zk-sequencer/sequencer/internal/sequencer/types"
	pkgLogger "github.com/tannetwork/zk-sequencer/sequencer/pkg/logger"
)

// APIServer provides REST API and WebSocket for the prover
type APIServer struct {
	store         *BatchStore
	port          int
	app           *fiber.App
	wsClients     map[*websocket.Conn]bool
	wsClientsMu   sync.RWMutex
	batchNotifier chan *types.Batch
}

// NewAPIServer creates a new API server
func NewAPIServer(store *BatchStore, port int) *APIServer {
	app := fiber.New(fiber.Config{
		DisableStartupMessage: true,
		AppName:               "Tan-ZK Sequencer",
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
		store:         store,
		port:          port,
		app:           app,
		wsClients:     make(map[*websocket.Conn]bool),
		batchNotifier: make(chan *types.Batch, 100),
	}
}

// Start starts the API server
// @title Tan-ZK Sequencer API
// @version 1.0
// @description REST API for the Tan-ZK Rollup Sequencer
// @BasePath /
func (api *APIServer) Start() error {
	// Swagger
	api.app.Get("/swagger/*", swagger.New(swagger.Config{
		Title:       "Tan-ZK Sequencer API",
		CustomStyle: `.swagger-ui .topbar { display: none }`,
	}))

	// Root endpoint
	api.app.Get("/", api.handleRoot)

	// Health check
	api.app.Get("/health", api.handleHealth)

	// Status endpoint
	api.app.Get("/status", api.handleStatus)

	// Batch endpoints
	api.app.Get("/batch/latest", api.handleLatestBatch)
	api.app.Get("/batch/:number", api.handleGetBatch)

	// Prover endpoints
	api.app.Get("/prover/batch/latest", api.handleLatestBatchForProver)
	api.app.Get("/prover/batch/:number", api.handleGetBatchForProver)

	// WebSocket endpoint
	api.app.Use("/ws", func(c *fiber.Ctx) error {
		if websocket.IsWebSocketUpgrade(c) {
			c.Locals("allowed", true)
			return c.Next()
		}
		return fiber.ErrUpgradeRequired
	})
	api.app.Get("/ws/batches", websocket.New(api.handleWebSocket))

	// Start broadcast loop
	go api.broadcastLoop()

	addr := fmt.Sprintf(":%d", api.port)
	pkgLogger.Info("🌐 REST API server listening on %s", addr)
	pkgLogger.Info("🔌 WebSocket endpoint available at ws://localhost%s/ws/batches", addr)
	pkgLogger.Info("📖 Swagger UI available at http://localhost%s/swagger/index.html", addr)

	return api.app.Listen(addr)
}

// Stop stops the API server gracefully
func (api *APIServer) Stop() error {
	if api.app != nil {
		return api.app.Shutdown()
	}
	return nil
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

// handleHealth godoc
// @Summary Health check
// @Description Checks if the service is up
// @Tags Meta
// @Produce json
// @Success 200 {object} map[string]string
// @Router /health [get]
func (api *APIServer) handleHealth(c *fiber.Ctx) error {
	return c.JSON(fiber.Map{"status": "ok"})
}

// handleStatus godoc
// @Summary Get sequencer status
// @Description Returns current status, batch count, and batch data range
// @Tags Meta
// @Produce json
// @Success 200 {object} map[string]interface{}
// @Router /status [get]
func (api *APIServer) handleStatus(c *fiber.Ctx) error {
	batchCount := api.store.Count()
	lastDeleted, _ := api.store.GetLastDeletedBatch()

	// Determine batch range
	var oldestBatch uint64 = 1
	if lastDeleted > 0 {
		oldestBatch = lastDeleted + 1 // First batch after cleanup
	}

	response := fiber.Map{
		"status":      "running",
		"batch_count": batchCount,
	}

	// Add batch range info if we have batches
	if batchCount > 0 {
		response["oldest"] = oldestBatch
		response["newest"] = batchCount
		response["total_in_db"] = batchCount - oldestBatch + 1
		if lastDeleted > 0 {
			response["last_deleted"] = lastDeleted
		}
	}

	return c.JSON(response)
}

// handleLatestBatch godoc
// @Summary Get latest batch
// @Description Returns the most recently created batch
// @Tags Batches
// @Produce json
// @Success 200 {object} types.Batch
// @Failure 404 {string} string "Not found"
// @Router /batch/latest [get]
func (api *APIServer) handleLatestBatch(c *fiber.Ctx) error {
	batch, err := api.store.GetLatestBatch()
	if err != nil {
		return c.Status(fiber.StatusNotFound).SendString(err.Error())
	}
	return c.JSON(batch)
}

// handleGetBatch godoc
// @Summary Get batch by number
// @Description Returns a specific batch by its number
// @Tags Batches
// @Param number path int true "Batch Number"
// @Produce json
// @Success 200 {object} types.Batch
// @Failure 400 {string} string "Invalid ID"
// @Failure 404 {string} string "Not found"
// @Router /batch/{number} [get]
func (api *APIServer) handleGetBatch(c *fiber.Ctx) error {
	number, err := c.ParamsInt("number")
	if err != nil {
		return c.Status(fiber.StatusBadRequest).SendString("invalid batch number")
	}

	batch, err := api.store.GetBatch(uint64(number))
	if err != nil {
		return c.Status(fiber.StatusNotFound).SendString(err.Error())
	}
	return c.JSON(batch)
}

// handleLatestBatchForProver godoc
// @Summary Get latest batch for prover
// @Description Returns the latest batch including execution traces for the prover
// @Tags Prover
// @Produce json
// @Success 200 {object} types.Batch
// @Failure 404 {string} string "Not found"
// @Router /prover/batch/latest [get]
func (api *APIServer) handleLatestBatchForProver(c *fiber.Ctx) error {
	// Same as standard handler for now, but logical separation for the future
	return api.handleLatestBatch(c)
}

// handleGetBatchForProver godoc
// @Summary Get batch by number for prover
// @Description Returns a specific batch including execution traces for the prover
// @Tags Prover
// @Param number path int true "Batch Number"
// @Produce json
// @Success 200 {object} types.Batch
// @Failure 400 {string} string "Invalid ID"
// @Failure 404 {string} string "Not found"
// @Router /prover/batch/{number} [get]
func (api *APIServer) handleGetBatchForProver(c *fiber.Ctx) error {
	return api.handleGetBatch(c)
}

// NotifyNewBatch notifies all WebSocket clients about a new batch
func (api *APIServer) NotifyNewBatch(batch *types.Batch) {
	select {
	case api.batchNotifier <- batch:
	default:
		// Channel full, skip notification
	}
}

// handleWebSocket handles WebSocket connections
func (api *APIServer) handleWebSocket(c *websocket.Conn) {
	// Register client
	api.wsClientsMu.Lock()
	api.wsClients[c] = true
	api.wsClientsMu.Unlock()

	pkgLogger.Info("WebSocket client connected (total clients: %d)", len(api.wsClients))

	// Send welcome message
	welcomeMsg := fiber.Map{
		"type":    "welcome",
		"message": "Connected to sequencer WebSocket",
		"time":    time.Now().Unix(),
	}
	if err := c.WriteJSON(welcomeMsg); err != nil {
		return // Connection likely closed
	}

	// Read loop (keep connection alive)
	var (
		mt  int
		msg []byte
		err error
	)
	for {
		if mt, msg, err = c.ReadMessage(); err != nil {
			break
		}

		// Handle simple ping/pong
		var parsed map[string]interface{}
		if json.Unmarshal(msg, &parsed) == nil {
			if parsed["type"] == "ping" {
				c.WriteJSON(fiber.Map{
					"type": "pong",
					"time": time.Now().Unix(),
				})
			}
		}

		// Currently we don't process other incoming messages, just echo or ignore
		_ = mt
	}

	// Cleanup
	api.wsClientsMu.Lock()
	delete(api.wsClients, c)
	api.wsClientsMu.Unlock()

	pkgLogger.Info("WebSocket client disconnected")
}

// broadcastLoop broadcasts batch notifications
func (api *APIServer) broadcastLoop() {
	for batch := range api.batchNotifier {
		api.wsClientsMu.RLock()
		// Copy connections to avoid holding lock during write
		clients := make([]*websocket.Conn, 0, len(api.wsClients))
		for conn := range api.wsClients {
			clients = append(clients, conn)
		}
		api.wsClientsMu.RUnlock()

		msg := fiber.Map{
			"type":  "new_batch",
			"batch": batch,
			"time":  time.Now().Unix(),
		}

		for _, conn := range clients {
			if err := conn.WriteJSON(msg); err != nil {
				// We don't remove here, the read loop will handle disconnection
				conn.Close()
			}
		}
	}
}
