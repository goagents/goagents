package tools

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

type HTTPTool struct {
	config *Config
	client *http.Client
}

func NewHTTPTool(config *Config) (*HTTPTool, error) {
	if config.URL == "" {
		return nil, fmt.Errorf("URL is required for HTTP tool")
	}
	
	timeout := 30 * time.Second
	if config.Timeout > 0 {
		timeout = config.Timeout
	}
	
	return &HTTPTool{
		config: config,
		client: &http.Client{
			Timeout: timeout,
		},
	}, nil
}

func (t *HTTPTool) Name() string {
	return t.config.Name
}

func (t *HTTPTool) Type() string {
	return "http"
}

func (t *HTTPTool) Execute(ctx context.Context, args map[string]interface{}) (*Result, error) {
	method := "POST"
	if m, ok := args["method"].(string); ok {
		method = strings.ToUpper(m)
	}
	
	url := t.config.URL
	if endpoint, ok := args["endpoint"].(string); ok {
		url = strings.TrimSuffix(url, "/") + "/" + strings.TrimPrefix(endpoint, "/")
	}
	
	var body io.Reader
	if method != "GET" && method != "HEAD" {
		if data, ok := args["data"]; ok {
			jsonData, err := json.Marshal(data)
			if err != nil {
				return &Result{Error: fmt.Sprintf("failed to marshal request data: %v", err)}, nil
			}
			body = bytes.NewReader(jsonData)
		}
	}
	
	req, err := http.NewRequestWithContext(ctx, method, url, body)
	if err != nil {
		return &Result{Error: fmt.Sprintf("failed to create request: %v", err)}, nil
	}
	
	// Set headers
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("User-Agent", "goagents/1.0")
	
	// Add authentication
	if t.config.Auth != nil {
		switch t.config.Auth.Type {
		case "bearer":
			req.Header.Set("Authorization", "Bearer "+t.config.Auth.Token)
		case "api_key":
			req.Header.Set("X-API-Key", t.config.Auth.APIKey)
		case "basic":
			req.SetBasicAuth(t.config.Auth.APIKey, t.config.Auth.Secret)
		}
	}
	
	// Add custom headers from config
	for key, value := range t.config.Config {
		if strings.HasPrefix(key, "header_") {
			headerName := strings.TrimPrefix(key, "header_")
			req.Header.Set(headerName, value)
		}
	}
	
	resp, err := t.client.Do(req)
	if err != nil {
		return &Result{Error: fmt.Sprintf("request failed: %v", err)}, nil
	}
	defer resp.Body.Close()
	
	responseBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return &Result{Error: fmt.Sprintf("failed to read response: %v", err)}, nil
	}
	
	if resp.StatusCode >= 400 {
		return &Result{
			Error: fmt.Sprintf("HTTP %d: %s", resp.StatusCode, string(responseBody)),
		}, nil
	}
	
	var data interface{}
	if len(responseBody) > 0 {
		if err := json.Unmarshal(responseBody, &data); err != nil {
			// If JSON parsing fails, return as string
			data = string(responseBody)
		}
	}
	
	return &Result{
		Data: data,
		Metadata: map[string]interface{}{
			"status_code": resp.StatusCode,
			"headers":     resp.Header,
			"url":         url,
			"method":      method,
		},
	}, nil
}

func (t *HTTPTool) Close() error {
	return nil
}