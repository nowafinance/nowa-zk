package api

import (
	"encoding/json"
	"fmt"
	"math/big"
	"net/http"

	"github.com/nowafinance/nowa-zk/sequencer/internal/engine"
	"github.com/nowafinance/nowa-zk/sequencer/internal/types"
)

type Server struct {
	engine *engine.Engine
}

func NewServer(eng *engine.Engine) *Server {
	return &Server{
		engine: eng,
	}
}

// Start runs the HTTP server for the Sequencer.
func (s *Server) Start(port string) error {
	http.HandleFunc("/order", s.handleOrder)
	
	fmt.Printf("Sequencer API listening on %s...\n", port)
	return http.ListenAndServe(port, nil)
}

type OrderRequest struct {
	MakerAddress string `json:"maker_address"`
	TokenID      uint32 `json:"token_id"`
	Amount       string `json:"amount"` // string for big.Int parsing
	Price        string `json:"price"`  // string for big.Int parsing
	IsBuy        bool   `json:"is_buy"`
	Nonce        uint64 `json:"nonce"`
	Signature    string `json:"signature"`
}

func (s *Server) handleOrder(w http.ResponseWriter, r *http.Request) {
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
		MakerAddress: req.MakerAddress,
		TokenID:      req.TokenID,
		Amount:       amount,
		Price:        price,
		IsBuy:        req.IsBuy,
		Nonce:        req.Nonce,
		Signature:    req.Signature,
	}

	err := s.engine.ProcessOrder(order)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	w.WriteHeader(http.StatusOK)
	w.Write([]byte(`{"status":"success"}`))
}
