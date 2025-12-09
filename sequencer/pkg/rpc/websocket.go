package rpc

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"sync"
	"time"

	"github.com/gorilla/websocket"
	"github.com/joho/godotenv"
)

// WebSocketClient handles WebSocket connections for real-time updates
type WebSocketClient struct {
	url            string
	dialer         *websocket.Dialer
	conn           *websocket.Conn
	connMutex      sync.RWMutex
	pingInterval   time.Duration
	maxReconnect   int
	reconnectDelay time.Duration
	subscribers    map[int]chan *RPCResponse
	subMutex       sync.RWMutex
	nextReqID      int
	reqMutex       sync.Mutex
	ctx            context.Context
	cancel         context.CancelFunc
	wg             sync.WaitGroup
}

// WebSocketConfig holds configuration for WebSocket client
type WebSocketConfig struct {
	URL            string
	PingInterval   time.Duration
	MaxReconnect   int
	ReconnectDelay time.Duration
}

// DefaultWebSocketConfig returns default WebSocket configuration
func DefaultWebSocketConfig() *WebSocketConfig {
	return &WebSocketConfig{
		PingInterval:   30 * time.Second,
		MaxReconnect:   5,
		ReconnectDelay: 1 * time.Second,
	}
}

// NewWebSocketClient creates a new WebSocket client
func NewWebSocketClient(wsURL string, opts ...WebSocketOption) (*WebSocketClient, error) {
	if wsURL == "" {
		return nil, ErrInvalidConfig{Field: "WSURL", Reason: "cannot be empty"}
	}

	config := DefaultWebSocketConfig()
	config.URL = wsURL

	// Apply options
	for _, opt := range opts {
		opt(config)
	}

	ctx, cancel := context.WithCancel(context.Background())

	client := &WebSocketClient{
		url:            config.URL,
		dialer:         websocket.DefaultDialer,
		pingInterval:   config.PingInterval,
		maxReconnect:   config.MaxReconnect,
		reconnectDelay: config.ReconnectDelay,
		subscribers:    make(map[int]chan *RPCResponse),
		nextReqID:      1,
		ctx:            ctx,
		cancel:         cancel,
	}

	// Connect immediately
	if err := client.connect(); err != nil {
		cancel()
		return nil, fmt.Errorf("failed to connect: %w", err)
	}

	// Start message handler
	client.wg.Add(1)
	go client.messageHandler()

	// Start ping ticker
	client.wg.Add(1)
	go client.pingTicker()

	return client, nil
}

// NewWebSocketClientFromEnv creates a WebSocket client from environment variables
func NewWebSocketClientFromEnv() (*WebSocketClient, error) {
	_ = godotenv.Load()

	wsURL := os.Getenv("WS")
	if wsURL == "" {
		return nil, fmt.Errorf("WS environment variable is required")
	}

	return NewWebSocketClient(wsURL)
}

// WebSocketOption is a function that modifies WebSocket configuration
type WebSocketOption func(*WebSocketConfig)

// WithPingInterval sets the ping interval
func WithPingInterval(interval time.Duration) WebSocketOption {
	return func(c *WebSocketConfig) {
		c.PingInterval = interval
	}
}

// WithMaxReconnect sets the maximum reconnection attempts
func WithMaxReconnect(max int) WebSocketOption {
	return func(c *WebSocketConfig) {
		c.MaxReconnect = max
	}
}

// WithReconnectDelay sets the reconnection delay
func WithReconnectDelay(delay time.Duration) WebSocketOption {
	return func(c *WebSocketConfig) {
		c.ReconnectDelay = delay
	}
}

// connect establishes a WebSocket connection
func (w *WebSocketClient) connect() error {
	w.connMutex.Lock()
	defer w.connMutex.Unlock()

	// Close existing connection if any
	if w.conn != nil {
		w.conn.Close()
	}

	conn, _, err := w.dialer.Dial(w.url, http.Header{})
	if err != nil {
		return ErrConnectionFailed{URL: w.url, Cause: err}
	}

	w.conn = conn
	return nil
}

// Close closes the WebSocket connection
func (w *WebSocketClient) Close() error {
	w.cancel()
	w.connMutex.Lock()
	if w.conn != nil {
		w.conn.Close()
		w.conn = nil
	}
	w.connMutex.Unlock()

	// Close all subscriber channels
	w.subMutex.Lock()
	for _, ch := range w.subscribers {
		close(ch)
	}
	w.subscribers = make(map[int]chan *RPCResponse)
	w.subMutex.Unlock()

	w.wg.Wait()
	return nil
}

// IsConnected returns true if the WebSocket is connected
func (w *WebSocketClient) IsConnected() bool {
	w.connMutex.RLock()
	defer w.connMutex.RUnlock()
	return w.conn != nil
}

// reconnect attempts to reconnect with exponential backoff
func (w *WebSocketClient) reconnect() error {
	// Close existing connection if any
	w.connMutex.Lock()
	if w.conn != nil {
		w.conn.Close()
		w.conn = nil
	}
	w.connMutex.Unlock()

	delay := w.reconnectDelay
	for attempt := 0; attempt < w.maxReconnect; attempt++ {
		select {
		case <-w.ctx.Done():
			return w.ctx.Err()
		case <-time.After(delay):
			// Try to reconnect
		}

		if err := w.connect(); err == nil {
			return nil // Successfully reconnected
		}

		// Exponential backoff
		delay *= 2
		if delay > 30*time.Second {
			delay = 30 * time.Second
		}
	}

	return fmt.Errorf("max reconnection attempts (%d) exceeded", w.maxReconnect)
}

// messageHandler handles incoming WebSocket messages
func (w *WebSocketClient) messageHandler() {
	defer w.wg.Done()
	defer func() {
		// Recover from any panics and mark connection as failed
		if r := recover(); r != nil {
			// Mark connection as nil to prevent further reads
			w.connMutex.Lock()
			if w.conn != nil {
				w.conn.Close()
				w.conn = nil
			}
			w.connMutex.Unlock()
			// Try to reconnect
			_ = w.reconnect()
		}
	}()

	for {
		select {
		case <-w.ctx.Done():
			return
		default:
			w.connMutex.RLock()
			conn := w.conn
			w.connMutex.RUnlock()

			if conn == nil {
				// Try to reconnect
				if err := w.reconnect(); err != nil {
					return
				}
				// Small delay after reconnect
				time.Sleep(100 * time.Millisecond)
				continue
			}

			// Set read deadline
			if err := conn.SetReadDeadline(time.Now().Add(60 * time.Second)); err != nil {
				// Connection is broken, mark as nil and try to reconnect
				w.connMutex.Lock()
				w.conn = nil
				w.connMutex.Unlock()
				if err := w.reconnect(); err != nil {
					return
				}
				continue
			}

			_, message, err := conn.ReadMessage()
			if err != nil {
				// Mark connection as nil on any read error
				w.connMutex.Lock()
				w.conn = nil
				w.connMutex.Unlock()

				// Check if context is canceled
				if w.ctx.Err() != nil {
					return
				}

				// Check if it's a close error or connection error
				if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) ||
					websocket.IsCloseError(err, websocket.CloseNormalClosure, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
					// Try to reconnect
					if err := w.reconnect(); err != nil {
						return
					}
				} else {
					// Other read error, try to reconnect
					if err := w.reconnect(); err != nil {
						return
					}
				}
				// Small delay before retrying
				time.Sleep(100 * time.Millisecond)
				continue
			}

			// Parse message - could be RPC response or subscription notification
			var rawMsg map[string]interface{}
			if err := json.Unmarshal(message, &rawMsg); err != nil {
				continue // Skip invalid messages
			}

			// Check if it's a subscription notification (has method but no ID)
			if method, hasMethod := rawMsg["method"].(string); hasMethod {
				if method == "eth_subscription" {
					// This is a subscription notification
					// Wrap it in RPCResponse format for dispatch
					resp := RPCResponse{
						JSONRPC: "2.0",
						ID:      0, // Notifications don't have IDs
					}
					if resultBytes, err := json.Marshal(rawMsg); err == nil {
						resp.Result = json.RawMessage(resultBytes)
					}
					w.dispatchMessage(&resp)
					continue
				}
			}

			// Try parsing as regular RPC response
			var resp RPCResponse
			if err := json.Unmarshal(message, &resp); err != nil {
				continue // Skip invalid messages
			}

			// Dispatch to subscribers
			w.dispatchMessage(&resp)
		}
	}
}

// dispatchMessage sends a message to all subscribers
func (w *WebSocketClient) dispatchMessage(resp *RPCResponse) {
	w.subMutex.RLock()
	defer w.subMutex.RUnlock()

	// Check if this is a subscription notification
	// Subscription notifications have method="eth_subscription" in the Result field
	// (because we wrapped the entire notification in resp.Result)
	if len(resp.Result) > 0 {
		var notification map[string]interface{}
		if err := json.Unmarshal(resp.Result, &notification); err == nil {
			if method, ok := notification["method"].(string); ok && method == "eth_subscription" {
				// This is a subscription notification, broadcast to all subscribers
				for _, ch := range w.subscribers {
					select {
					case ch <- resp:
					default:
						// Channel is full, skip
					}
				}
				return
			}
		}
	}

	// Regular RPC response - find subscriber by ID
	if resp.ID > 0 {
		if ch, ok := w.subscribers[resp.ID]; ok {
			select {
			case ch <- resp:
			default:
				// Channel is full, skip
			}
		}
	}
}

// pingTicker sends ping messages to keep connection alive
func (w *WebSocketClient) pingTicker() {
	defer w.wg.Done()

	ticker := time.NewTicker(w.pingInterval)
	defer ticker.Stop()

	for {
		select {
		case <-w.ctx.Done():
			return
		case <-ticker.C:
			w.connMutex.RLock()
			conn := w.conn
			w.connMutex.RUnlock()

			if conn != nil {
				conn.SetWriteDeadline(time.Now().Add(10 * time.Second))
				if err := conn.WriteMessage(websocket.PingMessage, nil); err != nil {
					// Connection lost, reconnect will be handled by messageHandler
				}
			}
		}
	}
}

// Subscribe creates a subscription and returns a channel for receiving messages
func (w *WebSocketClient) Subscribe(method string, params []interface{}) (<-chan *RPCResponse, error) {
	// Generate unique request ID
	w.reqMutex.Lock()
	reqID := w.nextReqID
	w.nextReqID++
	w.reqMutex.Unlock()

	// Create subscriber channel
	ch := make(chan *RPCResponse, 100) // Buffered channel
	w.subMutex.Lock()
	w.subscribers[reqID] = ch
	w.subMutex.Unlock()

	// Send subscription request
	req := RPCRequest{
		JSONRPC: "2.0",
		Method:  method,
		Params:  params,
		ID:      reqID,
	}

	w.connMutex.RLock()
	conn := w.conn
	w.connMutex.RUnlock()

	if conn == nil {
		return nil, fmt.Errorf("not connected")
	}

	if err := conn.WriteJSON(req); err != nil {
		// Clean up subscription
		w.subMutex.Lock()
		delete(w.subscribers, reqID)
		close(ch)
		w.subMutex.Unlock()
		return nil, err
	}

	return ch, nil
}
