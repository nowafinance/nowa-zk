package rpc

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gorilla/websocket"
)

func TestGetBlockByNumber(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		response := RPCResponse{
			JSONRPC: "2.0",
			ID:      1,
			Result: json.RawMessage(`{
				"number": "0x1234",
				"hash": "0xabcd",
				"parentHash": "0xef01",
				"timestamp": "0x5f5e100",
				"stateRoot": "0x1234567890abcdef",
				"transactions": []
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

	block, err := client.GetBlockByNumber(context.Background(), 0x1234, false)
	if err != nil {
		t.Fatalf("GetBlockByNumber() error = %v", err)
	}

	if block.Number.Uint64() != 0x1234 {
		t.Errorf("GetBlockByNumber() number = %d, want %d", block.Number.Uint64(), 0x1234)
	}
	if block.Hash != "0xabcd" {
		t.Errorf("GetBlockByNumber() hash = %s, want 0xabcd", block.Hash)
	}
}

func TestGetBlockByHash(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		response := RPCResponse{
			JSONRPC: "2.0",
			ID:      1,
			Result: json.RawMessage(`{
				"number": "0x5678",
				"hash": "0x1234",
				"parentHash": "0xabcd",
				"timestamp": "0x5f5e100",
				"stateRoot": "0xabcdef1234567890",
				"transactions": []
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

	block, err := client.GetBlockByHash(context.Background(), "0x1234", false)
	if err != nil {
		t.Fatalf("GetBlockByHash() error = %v", err)
	}

	if block.Hash != "0x1234" {
		t.Errorf("GetBlockByHash() hash = %s, want 0x1234", block.Hash)
	}
	if block.Number.Uint64() != 0x5678 {
		t.Errorf("GetBlockByHash() number = %d, want %d", block.Number.Uint64(), 0x5678)
	}
}

func TestWebSocketClient_SubscribeNewHeads(t *testing.T) {
	var receivedReq RPCRequest
	done := make(chan bool)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		upgrader := websocket.Upgrader{}
		conn, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			return
		}
		defer conn.Close()

		// Read subscription request
		if err := conn.ReadJSON(&receivedReq); err == nil {
			// Send subscription confirmation first
			confirmResp := RPCResponse{
				JSONRPC: "2.0",
				ID:      receivedReq.ID,
				Result:  json.RawMessage(`"0x1"`), // Subscription ID
			}
			conn.WriteJSON(confirmResp)

			// Wait a bit, then send subscription notification
			time.Sleep(50 * time.Millisecond)

			// Send a block header notification
			notification := map[string]interface{}{
				"jsonrpc": "2.0",
				"method":  "eth_subscription",
				"params": map[string]interface{}{
					"subscription": "0x1",
					"result": map[string]interface{}{
						"number":     "0x1234",
						"hash":       "0xabcd",
						"parentHash": "0xef01",
						"timestamp":  "0x5f5e100",
						"stateRoot":  "0x1234567890abcdef",
					},
				},
			}
			conn.WriteJSON(notification)
		}

		// Keep connection alive
		<-done
	}))
	defer server.Close()

	wsURL := "ws" + server.URL[4:]

	client, err := NewWebSocketClient(wsURL)
	if err != nil {
		t.Fatalf("NewWebSocketClient() error = %v", err)
	}
	defer func() {
		close(done)
		client.Close()
	}()

	// Wait for connection
	time.Sleep(100 * time.Millisecond)

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	headerChan, err := client.SubscribeNewHeads(ctx)
	if err != nil {
		t.Fatalf("SubscribeNewHeads() error = %v", err)
	}

	// Wait for block header
	select {
	case header := <-headerChan:
		if header == nil {
			t.Error("Received nil block header")
		} else if header.Number.Uint64() != 0x1234 {
			t.Errorf("SubscribeNewHeads() number = %d, want %d", header.Number.Uint64(), 0x1234)
		}
	case <-ctx.Done():
		t.Error("Timeout waiting for block header")
	}

	// Verify subscription request
	if receivedReq.Method != "eth_subscribe" {
		t.Errorf("SubscribeNewHeads() sent method = %s, want eth_subscribe", receivedReq.Method)
	}
}

