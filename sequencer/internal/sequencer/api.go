package sequencer

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strconv"
	"time"

	"github.com/gorilla/mux"
)

// APIServer provides REST API for the prover
type APIServer struct {
	store *BatchStore
	port  int
	server *http.Server
}

// NewAPIServer creates a new API server
func NewAPIServer(store *BatchStore, port int) *APIServer {
	return &APIServer{
		store: store,
		port:  port,
	}
}

// Start starts the API server
func (api *APIServer) Start() error {
	router := mux.NewRouter()

	// Health check
	router.HandleFunc("/health", api.handleHealth).Methods("GET")

	// Status endpoint
	router.HandleFunc("/status", api.handleStatus).Methods("GET")

	// Batch endpoints
	router.HandleFunc("/batch/latest", api.handleLatestBatch).Methods("GET")
	router.HandleFunc("/batch/{number}", api.handleGetBatch).Methods("GET")
	router.HandleFunc("/batches", api.handleGetAllBatches).Methods("GET")

	// Prover endpoints
	router.HandleFunc("/prover/batch/{number}", api.handleGetBatchForProver).Methods("GET")
	router.HandleFunc("/prover/batch/latest", api.handleLatestBatchForProver).Methods("GET")

	addr := fmt.Sprintf(":%d", api.port)
	api.server = &http.Server{
		Addr:    addr,
		Handler: router,
	}

	log.Printf("🌐 REST API server listening on %s", addr)
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

func (api *APIServer) handleHealth(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
}

func (api *APIServer) handleStatus(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"status":      "running",
		"batch_count": api.store.Count(),
	})
}

func (api *APIServer) handleLatestBatch(w http.ResponseWriter, r *http.Request) {
	batch, err := api.store.GetLatestBatch()
	if err != nil {
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(batch)
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
	json.NewEncoder(w).Encode(batch)
}

func (api *APIServer) handleGetAllBatches(w http.ResponseWriter, r *http.Request) {
	batches := api.store.GetAllBatches()
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"batches": batches,
		"count":   len(batches),
	})
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
	json.NewEncoder(w).Encode(batch)
}

func (api *APIServer) handleLatestBatchForProver(w http.ResponseWriter, r *http.Request) {
	batch, err := api.store.GetLatestBatch()
	if err != nil {
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}

	// Return batch with traces for prover
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(batch)
}

