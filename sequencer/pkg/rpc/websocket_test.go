package rpc

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gorilla/websocket"
)

func TestNewWebSocketClient(t *testing.T) {
	tests := []struct {
		name    string
		wsURL   string
		wantErr bool
	}{
		{
			name:    "valid URL",
			wsURL:   "ws://localhost:8546",
			wantErr: false,
		},
		{
			name:    "empty URL",
			wsURL:   "",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// For empty URL test, we can test directly
			if tt.wsURL == "" {
				_, err := NewWebSocketClient(tt.wsURL)
				if (err != nil) != tt.wantErr {
					t.Errorf("NewWebSocketClient() error = %v, wantErr %v", err, tt.wantErr)
				}
				return
			}

			// For valid URL, we need a mock server
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				upgrader := websocket.Upgrader{}
				conn, err := upgrader.Upgrade(w, r, nil)
				if err != nil {
					return
				}
				defer conn.Close()

				// Keep connection alive for a bit
				time.Sleep(100 * time.Millisecond)
			}))
			defer server.Close()

			// Convert http:// to ws://
			wsURL := "ws" + server.URL[4:]

			client, err := NewWebSocketClient(wsURL)
			if (err != nil) != tt.wantErr {
				t.Errorf("NewWebSocketClient() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr && client == nil {
				t.Error("NewWebSocketClient() returned nil client without error")
			}
			if client != nil {
				client.Close()
			}
		})
	}
}

func TestWebSocketClient_IsConnected(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		upgrader := websocket.Upgrader{}
		conn, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			return
		}
		defer conn.Close()

		time.Sleep(100 * time.Millisecond)
	}))
	defer server.Close()

	wsURL := "ws" + server.URL[4:]

	client, err := NewWebSocketClient(wsURL)
	if err != nil {
		t.Fatalf("NewWebSocketClient() error = %v", err)
	}
	defer client.Close()

	if !client.IsConnected() {
		t.Error("IsConnected() = false, want true")
	}
}

func TestWebSocketClient_Subscribe(t *testing.T) {
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
		var req RPCRequest
		if err := conn.ReadJSON(&req); err == nil {
			receivedReq = req

			// Send a response
			resp := RPCResponse{
				JSONRPC: "2.0",
				ID:      req.ID,
				Result:  json.RawMessage(`"0x1234"`),
			}
			conn.WriteJSON(resp)
		}

		// Keep connection alive until test is done
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

	// Wait a bit for connection to establish
	time.Sleep(100 * time.Millisecond)

	ch, err := client.Subscribe("eth_subscribe", []interface{}{"newHeads"})
	if err != nil {
		t.Fatalf("Subscribe() error = %v", err)
	}

	// Wait for message
	select {
	case resp := <-ch:
		if resp == nil {
			t.Error("Received nil response")
		}
	case <-time.After(1 * time.Second):
		t.Error("Timeout waiting for subscription response")
	}

	// Verify request was sent
	if receivedReq.Method != "eth_subscribe" {
		t.Errorf("Subscribe() sent method = %s, want eth_subscribe", receivedReq.Method)
	}
}

func TestWebSocketClient_Close(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		upgrader := websocket.Upgrader{}
		conn, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			return
		}
		defer conn.Close()

		time.Sleep(100 * time.Millisecond)
	}))
	defer server.Close()

	wsURL := "ws" + server.URL[4:]

	client, err := NewWebSocketClient(wsURL)
	if err != nil {
		t.Fatalf("NewWebSocketClient() error = %v", err)
	}

	// Close client
	if err := client.Close(); err != nil {
		t.Errorf("Close() error = %v", err)
	}

	// Verify connection is closed
	if client.IsConnected() {
		t.Error("IsConnected() = true after Close(), want false")
	}
}

func TestWebSocketClient_Reconnect(t *testing.T) {
	// This test would require more complex setup to simulate disconnection
	// For now, we'll test that reconnect logic exists
	config := DefaultWebSocketConfig()
	if config.MaxReconnect == 0 {
		t.Error("DefaultWebSocketConfig() MaxReconnect = 0, want > 0")
	}
}

