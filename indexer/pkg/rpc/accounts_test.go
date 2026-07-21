package rpc

import (
	"context"
	"encoding/json"
	"math/big"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestGetBalance(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		response := RPCResponse{
			JSONRPC: "2.0",
			ID:      1,
			Result:  json.RawMessage(`"0x1bc16d674ec80000"`), // 2 ETH
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

	balance, err := client.GetBalance(context.Background(), "0x1111111111111111111111111111111111111111", nil)
	if err != nil {
		t.Fatalf("GetBalance() error = %v", err)
	}

	expected := big.NewInt(0)
	expected.SetString("1bc16d674ec80000", 16) // 2 ETH in wei
	if balance.Cmp(expected) != 0 {
		t.Errorf("GetBalance() = %s, want %s", balance.String(), expected.String())
	}
}

func TestGetTransactionCount(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		response := RPCResponse{
			JSONRPC: "2.0",
			ID:      1,
			Result:  json.RawMessage(`"0x5"`), // Nonce 5
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

	nonce, err := client.GetTransactionCount(context.Background(), "0x1111111111111111111111111111111111111111", nil)
	if err != nil {
		t.Fatalf("GetTransactionCount() error = %v", err)
	}

	if nonce != 5 {
		t.Errorf("GetTransactionCount() = %d, want 5", nonce)
	}
}

func TestGetCode(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		response := RPCResponse{
			JSONRPC: "2.0",
			ID:      1,
			Result:  json.RawMessage(`"0x6080604052348015600f57600080fd5b50"`), // Sample contract code
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

	code, err := client.GetCode(context.Background(), "0x2222222222222222222222222222222222222222", nil)
	if err != nil {
		t.Fatalf("GetCode() error = %v", err)
	}

	if code != "0x6080604052348015600f57600080fd5b50" {
		t.Errorf("GetCode() = %s, want 0x6080604052348015600f57600080fd5b50", code)
	}
}

func TestGetAccountState(t *testing.T) {
	callCount := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		callCount++
		var response RPCResponse

		// Parse request to determine which method
		var req RPCRequest
		json.NewDecoder(r.Body).Decode(&req)

		switch req.Method {
		case "eth_getBalance":
			response = RPCResponse{
				JSONRPC: "2.0",
				ID:      req.ID,
				Result:  json.RawMessage(`"0x1bc16d674ec80000"`),
			}
		case "eth_getTransactionCount":
			response = RPCResponse{
				JSONRPC: "2.0",
				ID:      req.ID,
				Result:  json.RawMessage(`"0x5"`),
			}
		case "eth_getCode":
			response = RPCResponse{
				JSONRPC: "2.0",
				ID:      req.ID,
				Result:  json.RawMessage(`"0x6080604052348015600f57600080fd5b50"`),
			}
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

	state, err := client.GetAccountState(context.Background(), "0x1111111111111111111111111111111111111111", nil)
	if err != nil {
		t.Fatalf("GetAccountState() error = %v", err)
	}

	expectedBalance := big.NewInt(0)
	expectedBalance.SetString("1bc16d674ec80000", 16)
	if state.Balance.Cmp(expectedBalance) != 0 {
		t.Errorf("GetAccountState() balance = %s, want %s", state.Balance.String(), expectedBalance.String())
	}
	if state.Nonce != 5 {
		t.Errorf("GetAccountState() nonce = %d, want 5", state.Nonce)
	}
	if state.Code != "0x6080604052348015600f57600080fd5b50" {
		t.Errorf("GetAccountState() code = %s, want 0x6080604052348015600f57600080fd5b50", state.Code)
	}
}

