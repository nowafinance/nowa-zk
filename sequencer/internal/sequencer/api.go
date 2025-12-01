package sequencer

import (
	"context"
	"encoding/json"
	"fmt"
	"html/template"
	"net/http"
	"strconv"
	"sync"
	"time"

	"github.com/gorilla/mux"
	"github.com/gorilla/websocket"
	"github.com/tannetwork/zk-sequencer/sequencer/internal/sequencer/types"
	"github.com/tannetwork/zk-sequencer/sequencer/pkg/logger"
)

// APIServer provides REST API and WebSocket for the prover
type APIServer struct {
	store         *BatchStore
	port          int
	server        *http.Server
	wsUpgrader    websocket.Upgrader
	wsClients     map[*websocket.Conn]bool
	wsClientsMu   sync.RWMutex
	batchNotifier chan *types.Batch
}

// NewAPIServer creates a new API server
func NewAPIServer(store *BatchStore, port int) *APIServer {
	return &APIServer{
		store:         store,
		port:          port,
		wsClients:     make(map[*websocket.Conn]bool),
		batchNotifier: make(chan *types.Batch, 100),
		wsUpgrader: websocket.Upgrader{
			CheckOrigin: func(r *http.Request) bool {
				return true // Allow all origins in development
			},
		},
	}
}

// Start starts the API server
func (api *APIServer) Start() error {
	router := mux.NewRouter()

	// Root endpoint
	router.HandleFunc("/", api.handleRoot).Methods("GET")

	// Health check
	router.HandleFunc("/health", api.handleHealth).Methods("GET")

	// Status endpoint
	router.HandleFunc("/status", api.handleStatus).Methods("GET")

	// Batch endpoints
	router.HandleFunc("/batch/latest", api.handleLatestBatch).Methods("GET")
	router.HandleFunc("/batch/{number}", api.handleGetBatch).Methods("GET")
	router.HandleFunc("/batches", api.handleGetAllBatches).Methods("GET")

	// Prover endpoints
	router.HandleFunc("/prover/batch/latest", api.handleLatestBatchForProver).Methods("GET")
	router.HandleFunc("/prover/batch/{number}", api.handleGetBatchForProver).Methods("GET")

	// WebSocket endpoint for real-time batch updates
	router.HandleFunc("/ws/batches", api.handleWebSocket)

	// Start broadcast loop for WebSocket notifications
	go api.broadcastLoop()

	addr := fmt.Sprintf(":%d", api.port)
	api.server = &http.Server{
		Addr:    addr,
		Handler: router,
	}

	logger.Info("🌐 REST API server listening on %s", addr)
	logger.Info("🔌 WebSocket endpoint available at ws://localhost%s/ws/batches", addr)
	return api.server.ListenAndServe()
}

// Stop stops the API server gracefully
func (api *APIServer) Stop() error {
	if api.server != nil {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		if err := api.server.Shutdown(ctx); err != nil {
			// If graceful shutdown fails, force close
			api.server.Close()
			return err
		}
	}
	return nil
}

const rootTemplate = `
<!DOCTYPE html>
<html lang="en">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>Tan-ZK Sequencer API</title>
    <style>
        body {
            font-family: -apple-system, BlinkMacSystemFont, "Segoe UI", Roboto, Helvetica, Arial, sans-serif;
            background-color: #ffffff;
            color: #333;
            margin: 0;
            padding: 40px;
        }
        .container {
            max-width: 900px;
            margin: 0 auto;
        }
        h1 {
            font-size: 24px;
            margin-bottom: 20px;
            color: #000;
        }
        table {
            width: 100%;
            border-collapse: collapse;
            border: 1px solid #e0e0e0;
        }
        th, td {
            text-align: left;
            padding: 12px 16px;
            border-bottom: 1px solid #e0e0e0;
        }
        th {
            background-color: #f9f9f9;
            font-weight: 600;
            color: #555;
            font-size: 14px;
            text-transform: uppercase;
            letter-spacing: 0.5px;
        }
        tr:last-child td {
            border-bottom: none;
        }
        tr:hover {
            background-color: #f5f5f5;
        }
        .method {
            font-family: monospace;
            font-weight: bold;
            color: #007bff;
        }
        .path {
            font-family: monospace;
            color: #d63384;
        }
        .footer {
            margin-top: 40px;
            font-size: 12px;
            color: #888;
            text-align: center;
        }
    </style>
</head>
<body>
    <div class="container">
        <h1>Tan-ZK Sequencer API</h1>
        <table>
            <thead>
                <tr>
                    <th>Method</th>
                    <th>Path</th>
                    <th>Description</th>
                </tr>
            </thead>
            <tbody>
                {{range .endpoints}}
                <tr>
                    <td><span class="method">{{.method}}</span></td>
                    <td><span class="path">{{.path}}</span></td>
                    <td>{{.description}}</td>
                </tr>
                {{end}}
            </tbody>
        </table>
        <div class="footer">
            Tan-ZK Sequencer v1.0.0
        </div>
    </div>
</body>
</html>
`

func (api *APIServer) handleRoot(w http.ResponseWriter, r *http.Request) {
	endpoints := []map[string]string{
		{"path": "/", "method": "GET", "description": "List all available endpoints"},
		{"path": "/health", "method": "GET", "description": "Health check"},
		{"path": "/status", "method": "GET", "description": "Get sequencer status"},
		{"path": "/batch/latest", "method": "GET", "description": "Get latest batch"},
		{"path": "/batch/{number}", "method": "GET", "description": "Get batch by number"},
		{"path": "/batches", "method": "GET", "description": "Get all batches"},
		{"path": "/prover/batch/latest", "method": "GET", "description": "Get latest batch for prover"},
		{"path": "/prover/batch/{number}", "method": "GET", "description": "Get batch by number for prover"},
		{"path": "/ws/batches", "method": "WS", "description": "WebSocket for real-time batch updates"},
	}

	tmpl, err := template.New("root").Parse(rootTemplate)
	if err != nil {
		logger.Error("Failed to parse root template: %v", err)
		http.Error(w, "internal server error", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if err := tmpl.Execute(w, map[string]interface{}{
		"endpoints": endpoints,
	}); err != nil {
		logger.Error("Failed to execute root template: %v", err)
		http.Error(w, "internal server error", http.StatusInternalServerError)
	}
}

func (api *APIServer) handleHealth(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(map[string]string{"status": "ok"}); err != nil {
		logger.Error("Failed to encode health response: %v", err)
		http.Error(w, "internal server error", http.StatusInternalServerError)
	}
}

func (api *APIServer) handleStatus(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(map[string]interface{}{
		"status":      "running",
		"batch_count": api.store.Count(),
	}); err != nil {
		logger.Error("Failed to encode status response: %v", err)
		http.Error(w, "internal server error", http.StatusInternalServerError)
	}
}

func (api *APIServer) handleLatestBatch(w http.ResponseWriter, r *http.Request) {
	batch, err := api.store.GetLatestBatch()
	if err != nil {
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(batch); err != nil {
		logger.Error("Failed to encode batch response: %v", err)
		http.Error(w, "internal server error", http.StatusInternalServerError)
	}
}

func (api *APIServer) handleGetBatch(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	number, err := strconv.ParseUint(vars["number"], 10, 64)
	if err != nil {
		http.Error(w, "invalid batch number", http.StatusBadRequest)
		return
	}

	batch, err := api.store.GetBatch(number)
	if err != nil {
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(batch); err != nil {
		logger.Error("Failed to encode batch response: %v", err)
		http.Error(w, "internal server error", http.StatusInternalServerError)
	}
}

func (api *APIServer) handleGetAllBatches(w http.ResponseWriter, r *http.Request) {
	batches := api.store.GetAllBatches()
	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(map[string]interface{}{
		"batches": batches,
		"count":   len(batches),
	}); err != nil {
		logger.Error("Failed to encode batches response: %v", err)
		http.Error(w, "internal server error", http.StatusInternalServerError)
	}
}

func (api *APIServer) handleGetBatchForProver(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	number, err := strconv.ParseUint(vars["number"], 10, 64)
	if err != nil {
		http.Error(w, "invalid batch number", http.StatusBadRequest)
		return
	}

	batch, err := api.store.GetBatch(number)
	if err != nil {
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}

	// Return batch with traces for prover
	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(batch); err != nil {
		logger.Error("Failed to encode batch response: %v", err)
		http.Error(w, "internal server error", http.StatusInternalServerError)
	}
}

func (api *APIServer) handleLatestBatchForProver(w http.ResponseWriter, r *http.Request) {
	batch, err := api.store.GetLatestBatch()
	if err != nil {
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}

	// Return batch with traces for prover
	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(batch); err != nil {
		logger.Error("Failed to encode batch response: %v", err)
		http.Error(w, "internal server error", http.StatusInternalServerError)
	}
}

// NotifyNewBatch notifies all WebSocket clients about a new batch
func (api *APIServer) NotifyNewBatch(batch *types.Batch) {
	select {
	case api.batchNotifier <- batch:
	default:
		// Channel full, skip notification
	}
}

// handleWebSocket handles WebSocket connections for real-time batch updates
func (api *APIServer) handleWebSocket(w http.ResponseWriter, r *http.Request) {
	conn, err := api.wsUpgrader.Upgrade(w, r, nil)
	if err != nil {
		logger.Error("WebSocket upgrade failed: %v", err)
		return
	}

	// Register client
	api.wsClientsMu.Lock()
	api.wsClients[conn] = true
	api.wsClientsMu.Unlock()

	logger.Info("WebSocket client connected (total clients: %d)", len(api.wsClients))

	// Send welcome message
	welcomeMsg := map[string]interface{}{
		"type":    "welcome",
		"message": "Connected to sequencer WebSocket",
		"time":    time.Now().Unix(),
	}
	if err := conn.WriteJSON(welcomeMsg); err != nil {
		logger.Error("Failed to send welcome message: %v", err)
		conn.Close()
		return
	}

	// Handle incoming messages (ping/pong, subscriptions, etc.)
	for {
		var msg map[string]interface{}
		if err := conn.ReadJSON(&msg); err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				logger.Error("WebSocket error: %v", err)
			}
			break
		}

		// Handle ping
		if msgType, ok := msg["type"].(string); ok && msgType == "ping" {
			if err := conn.WriteJSON(map[string]interface{}{
				"type": "pong",
				"time": time.Now().Unix(),
			}); err != nil {
				logger.Error("Failed to send pong: %v", err)
				break
			}
		}
	}

	// Unregister client
	api.wsClientsMu.Lock()
	delete(api.wsClients, conn)
	api.wsClientsMu.Unlock()
	conn.Close()

	logger.Info("WebSocket client disconnected (remaining clients: %d)", len(api.wsClients))
}

// broadcastLoop broadcasts batch notifications to all connected clients
func (api *APIServer) broadcastLoop() {
	for batch := range api.batchNotifier {
		api.wsClientsMu.RLock()
		clients := make([]*websocket.Conn, 0, len(api.wsClients))
		for conn := range api.wsClients {
			clients = append(clients, conn)
		}
		api.wsClientsMu.RUnlock()

		// Prepare batch notification message
		msg := map[string]interface{}{
			"type":  "new_batch",
			"batch": batch,
			"time":  time.Now().Unix(),
		}

		// Broadcast to all clients
		for _, conn := range clients {
			if err := conn.WriteJSON(msg); err != nil {
				logger.Error("Failed to send WebSocket message: %v", err)
				// Remove dead connection
				api.wsClientsMu.Lock()
				delete(api.wsClients, conn)
				api.wsClientsMu.Unlock()
				conn.Close()
			}
		}
	}
}
