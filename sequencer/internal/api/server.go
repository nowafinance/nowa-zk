package api

import (
	"encoding/json"
	"fmt"
	"math/big"
	"net/http"
	"strconv"
	"strings"

	"github.com/nowafinance/nowa-zk/sequencer/internal/batcher"
	"github.com/nowafinance/nowa-zk/sequencer/internal/engine"
	"github.com/nowafinance/nowa-zk/sequencer/internal/state"
	"github.com/nowafinance/nowa-zk/sequencer/internal/types"
)

type Server struct {
	engine  *engine.Engine
	batcher *batcher.Batcher
	tree    *state.LevelDBMerkleTree
}

func NewServer(eng *engine.Engine, batch *batcher.Batcher, tree *state.LevelDBMerkleTree) *Server {
	return &Server{
		engine:  eng,
		batcher: batch,
		tree:    tree,
	}
}

func writeCORS(w http.ResponseWriter, methods string) {
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Methods", methods)
	w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
}

// Start runs the HTTP server for the Sequencer.
func (s *Server) Start(port string) error {
	http.HandleFunc("/order", s.handleOrder)
	http.HandleFunc("/orderbook", s.handleOrderbook)
	http.HandleFunc("/balance", s.handleBalance)
	http.HandleFunc("/account", s.handleAccount)
	http.HandleFunc("/proof", s.handleProof)
	http.HandleFunc("/batch/latest", s.handleBatchLatest)
	http.HandleFunc("/batch/count", s.handleBatchCount)
	http.HandleFunc("/batch/", s.handleBatchByID)

	fmt.Printf("Sequencer API listening on %s...\n", port)
	return http.ListenAndServe(port, nil)
}

type OrderRequest struct {
	MakerAddress     string `json:"maker_address"`
	TokenID          uint32 `json:"token_id"`
	Amount           string `json:"amount"`
	Price            string `json:"price"`
	IsBuy            bool   `json:"is_buy"`
	Nonce            uint64 `json:"nonce"`
	Signature        string `json:"signature"`
	CircuitSignature string `json:"circuit_signature"`
}

type orderView struct {
	MakerAddress string `json:"maker_address"`
	TokenID      uint32 `json:"token_id"`
	Amount       string `json:"amount"`
	Price        string `json:"price"`
	IsBuy        bool   `json:"is_buy"`
	Nonce        uint64 `json:"nonce"`
}

func orderToView(o *types.Order) orderView {
	return orderView{
		MakerAddress: o.MakerAddress,
		TokenID:      o.TokenID,
		Amount:       o.Amount.String(),
		Price:        o.Price.String(),
		IsBuy:        o.IsBuy,
		Nonce:        o.Nonce,
	}
}

func (s *Server) handleOrder(w http.ResponseWriter, r *http.Request) {
	writeCORS(w, "POST, OPTIONS")
	if r.Method == http.MethodOptions {
		w.WriteHeader(http.StatusOK)
		return
	}
	if r.Method != http.MethodPost {
		http.Error(w, "Only POST is allowed", http.StatusMethodNotAllowed)
		return
	}

	var req OrderRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid JSON payload", http.StatusBadRequest)
		return
	}

	amount, ok := new(big.Int).SetString(req.Amount, 10)
	if !ok {
		http.Error(w, "Invalid amount format", http.StatusBadRequest)
		return
	}
	price, ok := new(big.Int).SetString(req.Price, 10)
	if !ok {
		http.Error(w, "Invalid price format", http.StatusBadRequest)
		return
	}

	order := &types.Order{
		MakerAddress:     req.MakerAddress,
		TokenID:          req.TokenID,
		Amount:           amount,
		Price:            price,
		IsBuy:            req.IsBuy,
		Nonce:            req.Nonce,
		Signature:        req.Signature,
		CircuitSignature: req.CircuitSignature,
	}

	if order.CircuitSignature == "" {
		http.Error(w, "circuit_signature required", http.StatusBadRequest)
		return
	}

	if err := s.engine.ProcessOrder(order); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	w.Write([]byte(`{"status":"success"}`))
}

// GET /orderbook?token_id=1
func (s *Server) handleOrderbook(w http.ResponseWriter, r *http.Request) {
	writeCORS(w, "GET, OPTIONS")
	if r.Method == http.MethodOptions {
		w.WriteHeader(http.StatusOK)
		return
	}
	if r.Method != http.MethodGet {
		http.Error(w, "Only GET is allowed", http.StatusMethodNotAllowed)
		return
	}

	tokenID := uint32(1)
	if q := r.URL.Query().Get("token_id"); q != "" {
		v, err := strconv.ParseUint(q, 10, 32)
		if err != nil {
			http.Error(w, "invalid token_id", http.StatusBadRequest)
			return
		}
		tokenID = uint32(v)
	}

	bids, asks := s.engine.SnapshotOrderbook(tokenID)
	bidViews := make([]orderView, len(bids))
	askViews := make([]orderView, len(asks))
	for i, o := range bids {
		bidViews[i] = orderToView(o)
	}
	for i, o := range asks {
		askViews[i] = orderToView(o)
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"token_id": tokenID,
		"bids":     bidViews,
		"asks":     askViews,
	})
}

// GET/POST /account?pubkey=0x... — open base+quote leaves; return indices for circuit auth.
func (s *Server) handleAccount(w http.ResponseWriter, r *http.Request) {
	writeCORS(w, "GET, POST, OPTIONS")
	if r.Method == http.MethodOptions {
		w.WriteHeader(http.StatusOK)
		return
	}
	if r.Method != http.MethodGet && r.Method != http.MethodPost {
		http.Error(w, "GET or POST only", http.StatusMethodNotAllowed)
		return
	}
	if s.tree == nil {
		http.Error(w, "state tree unavailable", http.StatusServiceUnavailable)
		return
	}

	pubKey := strings.TrimSpace(r.URL.Query().Get("pubkey"))
	if pubKey == "" && r.Method == http.MethodPost {
		var body struct {
			PubKey string `json:"pubkey"`
		}
		_ = json.NewDecoder(r.Body).Decode(&body)
		pubKey = strings.TrimSpace(body.PubKey)
	}
	if pubKey == "" {
		http.Error(w, "pubkey required", http.StatusBadRequest)
		return
	}

	baseID := uint32(1)
	quoteID := uint32(2)
	if q := r.URL.Query().Get("base_token"); q != "" {
		if v, err := strconv.ParseUint(q, 10, 32); err == nil {
			baseID = uint32(v)
		}
	}
	if q := r.URL.Query().Get("quote_token"); q != "" {
		if v, err := strconv.ParseUint(q, 10, 32); err == nil {
			quoteID = uint32(v)
		}
	}

	open := func(tokenID uint32) (map[string]interface{}, error) {
		acc, err := openBalance(s.tree, s.batcher, pubKey, tokenID)
		if err != nil {
			return nil, err
		}
		idx := (acc.AccountID * 256) + uint64(acc.TokenID)
		return map[string]interface{}{
			"token_id":   tokenID,
			"account_id": acc.AccountID,
			"index":      idx,
			"balance":    acc.Balance.String(),
			"nonce":      acc.Nonce,
			"pub_key_x":  acc.PubKeyX.String(),
			"pub_key_y":  acc.PubKeyY.String(),
		}, nil
	}

	base, err := open(baseID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	quote, err := open(quoteID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]interface{}{
		"pubkey": pubKey,
		"base":   base,
		"quote":  quote,
	})
}

// GET /balance?pubkey=0x...&token_id=1
func (s *Server) handleBalance(w http.ResponseWriter, r *http.Request) {
	writeCORS(w, "GET, OPTIONS")
	if r.Method == http.MethodOptions {
		w.WriteHeader(http.StatusOK)
		return
	}
	if r.Method != http.MethodGet {
		http.Error(w, "Only GET is allowed", http.StatusMethodNotAllowed)
		return
	}
	if s.tree == nil {
		http.Error(w, "state tree unavailable", http.StatusServiceUnavailable)
		return
	}

	pubKey := strings.TrimSpace(r.URL.Query().Get("pubkey"))
	if pubKey == "" {
		http.Error(w, "pubkey required", http.StatusBadRequest)
		return
	}
	tokenID := uint32(1)
	if q := r.URL.Query().Get("token_id"); q != "" {
		v, err := strconv.ParseUint(q, 10, 32)
		if err != nil {
			http.Error(w, "invalid token_id", http.StatusBadRequest)
			return
		}
		tokenID = uint32(v)
	}

	accID, err := s.tree.GetAccountID(pubKey)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	acc, err := s.tree.GetBalance(accID, tokenID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	balance := "0"
	nonce := uint64(0)
	exists := acc != nil
	if exists {
		balance = acc.Balance.String()
		nonce = acc.Nonce
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"pubkey":     pubKey,
		"account_id": accID,
		"token_id":   tokenID,
		"balance":    balance,
		"nonce":      nonce,
		"exists":     exists,
	})
}

// GET /proof?pubkey=0x...&token_id=1
//
// Returns everything needed to call NowaRollup.emergencyWithdraw() for this leaf:
// its current balance/nonce plus the 28-level Merkle path to the current root. This
// is the escape hatch's proof-serving endpoint — it only helps while the Sequencer
// itself is still reachable (e.g. the Prover died but the matching engine didn't);
// see docs/architecture/overview.md for the fully-offline reconstruction story.
func (s *Server) handleProof(w http.ResponseWriter, r *http.Request) {
	writeCORS(w, "GET, OPTIONS")
	if r.Method == http.MethodOptions {
		w.WriteHeader(http.StatusOK)
		return
	}
	if r.Method != http.MethodGet {
		http.Error(w, "Only GET is allowed", http.StatusMethodNotAllowed)
		return
	}
	if s.tree == nil {
		http.Error(w, "state tree unavailable", http.StatusServiceUnavailable)
		return
	}

	pubKey := strings.TrimSpace(r.URL.Query().Get("pubkey"))
	if pubKey == "" {
		http.Error(w, "pubkey required", http.StatusBadRequest)
		return
	}
	tokenID := uint32(1)
	if q := r.URL.Query().Get("token_id"); q != "" {
		v, err := strconv.ParseUint(q, 10, 32)
		if err != nil {
			http.Error(w, "invalid token_id", http.StatusBadRequest)
			return
		}
		tokenID = uint32(v)
	}

	// Same lookup pattern as /balance — GetAccountID auto-creates a fresh account ID
	// for a never-before-seen pubkey (an existing quirk of the tree, not new here).
	accID, err := s.tree.GetAccountID(pubKey)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	acc, err := s.tree.GetBalance(accID, tokenID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	balance := "0"
	nonce := uint64(0)
	pubX, pubY := "0", "0"
	if acc != nil {
		balance = acc.Balance.String()
		nonce = acc.Nonce
		pubX = acc.PubKeyX.String()
		pubY = acc.PubKeyY.String()
	}

	index := (accID * 256) + uint64(tokenID)
	path, bits := s.tree.GetPath(index)

	siblings := make([]string, 28)
	pathBits := make([]int, 28)
	for i := 0; i < 28; i++ {
		siblings[i] = path[i].String()
		pathBits[i] = int(bits[i])
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"pubkey":     pubKey,
		"account_id": accID,
		"token_id":   tokenID,
		"index":      index,
		"balance":    balance,
		"nonce":      nonce,
		"pub_key_x":  pubX,
		"pub_key_y":  pubY,
		"siblings":   siblings,
		"path_bits":  pathBits,
	})
}

func (s *Server) handleBatchLatest(w http.ResponseWriter, r *http.Request) {
	writeCORS(w, "GET, OPTIONS")
	if r.Method == http.MethodOptions {
		w.WriteHeader(http.StatusOK)
		return
	}
	if r.Method != http.MethodGet {
		http.Error(w, "Only GET is allowed", http.StatusMethodNotAllowed)
		return
	}

	b := s.batcher.GetLatestBatch()
	if b == nil {
		http.Error(w, "No batches available yet", http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(b); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

func (s *Server) handleBatchCount(w http.ResponseWriter, r *http.Request) {
	writeCORS(w, "GET, OPTIONS")
	if r.Method == http.MethodOptions {
		w.WriteHeader(http.StatusOK)
		return
	}
	if r.Method != http.MethodGet {
		http.Error(w, "Only GET is allowed", http.StatusMethodNotAllowed)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	fmt.Fprintf(w, `{"count":%d}`, s.batcher.GetBatchCount())
}

func (s *Server) handleBatchByID(w http.ResponseWriter, r *http.Request) {
	writeCORS(w, "GET, OPTIONS")
	if r.Method == http.MethodOptions {
		w.WriteHeader(http.StatusOK)
		return
	}
	if r.Method != http.MethodGet {
		http.Error(w, "Only GET is allowed", http.StatusMethodNotAllowed)
		return
	}

	idStr := r.URL.Path[len("/batch/"):]
	if idStr == "" {
		http.Error(w, "Missing batch ID", http.StatusBadRequest)
		return
	}
	id, err := strconv.ParseUint(idStr, 10, 64)
	if err != nil {
		http.Error(w, "Invalid batch ID: must be a non-negative integer", http.StatusBadRequest)
		return
	}

	b, ok := s.batcher.GetBatch(id)
	if !ok {
		http.Error(w, fmt.Sprintf("Batch %d not found", id), http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(b); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}
