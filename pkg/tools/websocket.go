package tools

import (
	"context"
	"fmt"
	"net/http"
	"sync"
	"time"

	"github.com/gorilla/websocket"
)

type WebSocketTool struct {
	config *Config
	conn   *websocket.Conn
	mu     sync.RWMutex
}

func NewWebSocketTool(config *Config) (*WebSocketTool, error) {
	if config.Endpoint == "" {
		return nil, fmt.Errorf("endpoint is required for WebSocket tool")
	}
	
	return &WebSocketTool{
		config: config,
	}, nil
}

func (t *WebSocketTool) Name() string {
	return t.config.Name
}

func (t *WebSocketTool) Type() string {
	return "websocket"
}

func (t *WebSocketTool) Execute(ctx context.Context, args map[string]interface{}) (*Result, error) {
	if err := t.ensureConnected(ctx); err != nil {
		return &Result{Error: fmt.Sprintf("failed to connect: %v", err)}, nil
	}
	
	// Prepare message
	message := map[string]interface{}{
		"id":   fmt.Sprintf("msg-%d", time.Now().UnixNano()),
		"type": "request",
		"data": args,
	}
	
	t.mu.Lock()
	err := t.conn.WriteJSON(message)
	t.mu.Unlock()
	
	if err != nil {
		return &Result{Error: fmt.Sprintf("failed to send message: %v", err)}, nil
	}
	
	// Wait for response
	timeout := 30 * time.Second
	if t.config.Timeout > 0 {
		timeout = t.config.Timeout
	}
	
	responseCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()
	
	responseCh := make(chan map[string]interface{}, 1)
	errorCh := make(chan error, 1)
	
	go func() {
		t.mu.RLock()
		conn := t.conn
		t.mu.RUnlock()
		
		if conn == nil {
			errorCh <- fmt.Errorf("connection closed")
			return
		}
		
		var response map[string]interface{}
		if err := conn.ReadJSON(&response); err != nil {
			errorCh <- err
			return
		}
		
		responseCh <- response
	}()
	
	select {
	case <-responseCtx.Done():
		return &Result{Error: "request timeout"}, nil
	case err := <-errorCh:
		return &Result{Error: fmt.Sprintf("failed to read response: %v", err)}, nil
	case response := <-responseCh:
		return &Result{
			Data: response,
			Metadata: map[string]interface{}{
				"endpoint": t.config.Endpoint,
				"tool":     t.config.Name,
			},
		}, nil
	}
}

func (t *WebSocketTool) ensureConnected(ctx context.Context) error {
	t.mu.Lock()
	defer t.mu.Unlock()
	
	if t.conn != nil {
		// Check if connection is still alive
		if err := t.conn.WriteMessage(websocket.PingMessage, nil); err == nil {
			return nil
		}
		// Connection is dead, close and reconnect
		t.conn.Close()
		t.conn = nil
	}
	
	headers := http.Header{}
	
	// Add authentication
	if t.config.Auth != nil {
		switch t.config.Auth.Type {
		case "bearer":
			headers.Set("Authorization", "Bearer "+t.config.Auth.Token)
		case "api_key":
			headers.Set("X-API-Key", t.config.Auth.APIKey)
		}
	}
	
	// Add custom headers from config
	for key, value := range t.config.Config {
		if key == "subprotocol" {
			continue // Handle subprotocols separately
		}
		headers.Set(key, value)
	}
	
	dialer := websocket.Dialer{
		HandshakeTimeout: 10 * time.Second,
	}
	
	conn, _, err := dialer.DialContext(ctx, t.config.Endpoint, headers)
	if err != nil {
		return fmt.Errorf("failed to dial WebSocket: %w", err)
	}
	
	t.conn = conn
	return nil
}

func (t *WebSocketTool) Close() error {
	t.mu.Lock()
	defer t.mu.Unlock()
	
	if t.conn != nil {
		err := t.conn.Close()
		t.conn = nil
		return err
	}
	
	return nil
}