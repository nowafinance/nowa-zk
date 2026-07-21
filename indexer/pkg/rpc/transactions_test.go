package rpc

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestGetTransactionByHash(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		response := RPCResponse{
			JSONRPC: "2.0",
			ID:      1,
			Result: json.RawMessage(`{
				"hash": "0xabcd1234",
				"from": "0x1111111111111111111111111111111111111111",
				"to": "0x2222222222222222222222222222222222222222",
				"value": "0x1bc16d674ec80000",
				"input": "0x",
				"nonce": "0x5",
				"gas": "0x5208",
				"gasPrice": "0x3b9aca00"
			}`),
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(response)
	}))
	defer server.Close()

	client, err := NewClient(server.URL)
	if err != nil {
		t.Fatalf("NewClient() error = %v", err)
	}
	defer client.Close()

	tx, err := client.GetTransactionByHash(context.Background(), "0xabcd1234")
	if err != nil {
		t.Fatalf("GetTransactionByHash() error = %v", err)
	}

	if tx.Hash != "0xabcd1234" {
		t.Errorf("GetTransactionByHash() hash = %s, want 0xabcd1234", tx.Hash)
	}
	if tx.From != "0x1111111111111111111111111111111111111111" {
		t.Errorf("GetTransactionByHash() from = %s, want 0x1111...", tx.From)
	}
	if tx.Nonce != 5 {
		t.Errorf("GetTransactionByHash() nonce = %d, want 5", tx.Nonce)
	}
}

func TestGetTransactionsByBlock(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		response := RPCResponse{
			JSONRPC: "2.0",
			ID:      1,
			Result: json.RawMessage(`{
				"number": "0x1234",
				"hash": "0xabcd",
				"transactions": [
					{
						"hash": "0xtx1",
						"from": "0x1111",
						"to": "0x2222",
						"value": "0x0",
						"input": "0x",
						"nonce": "0x1",
						"gas": "0x5208",
						"gasPrice": "0x3b9aca00"
					},
					{
						"hash": "0xtx2",
						"from": "0x3333",
						"to": "0x4444",
						"value": "0x0",
						"input": "0x",
						"nonce": "0x2",
						"gas": "0x5208",
						"gasPrice": "0x3b9aca00"
					}
				]
			}`),
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(response)
	}))
	defer server.Close()

	client, err := NewClient(server.URL)
	if err != nil {
		t.Fatalf("NewClient() error = %v", err)
	}
	defer client.Close()

	txs, err := client.GetTransactionsByBlock(context.Background(), 0x1234)
	if err != nil {
		t.Fatalf("GetTransactionsByBlock() error = %v", err)
	}

	if len(txs) != 2 {
		t.Errorf("GetTransactionsByBlock() returned %d transactions, want 2", len(txs))
	}
	if txs[0].Hash != "0xtx1" {
		t.Errorf("GetTransactionsByBlock() first tx hash = %s, want 0xtx1", txs[0].Hash)
	}
}

