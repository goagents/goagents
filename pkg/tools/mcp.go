package tools

import (
	"context"
	"fmt"
	"time"
)

type MCPTool struct {
	config *Config
	client *MCPClient
}

type MCPClient struct {
	serverAddr string
	timeout    time.Duration
}

type MCPRequest struct {
	ID     string                 `json:"id"`
	Method string                 `json:"method"`
	Params map[string]interface{} `json:"params"`
}

type MCPResponse struct {
	ID     string      `json:"id"`
	Result interface{} `json:"result,omitempty"`
	Error  *MCPError   `json:"error,omitempty"`
}

type MCPError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
	Data    interface{} `json:"data,omitempty"`
}

func NewMCPTool(config *Config) (*MCPTool, error) {
	if config.Server == "" {
		return nil, fmt.Errorf("server is required for MCP tool")
	}
	
	timeout := 30 * time.Second
	if config.Timeout > 0 {
		timeout = config.Timeout
	}
	
	client := &MCPClient{
		serverAddr: config.Server,
		timeout:    timeout,
	}
	
	return &MCPTool{
		config: config,
		client: client,
	}, nil
}

func (t *MCPTool) Name() string {
	return t.config.Name
}

func (t *MCPTool) Type() string {
	return "mcp"
}

func (t *MCPTool) Execute(ctx context.Context, args map[string]interface{}) (*Result, error) {
	method := "call_tool"
	if m, ok := args["method"].(string); ok {
		method = m
	}
	
	// Prepare MCP request
	req := &MCPRequest{
		ID:     fmt.Sprintf("req-%d", time.Now().UnixNano()),
		Method: method,
		Params: args,
	}
	
	resp, err := t.client.Call(ctx, req)
	if err != nil {
		return &Result{Error: fmt.Sprintf("MCP call failed: %v", err)}, nil
	}
	
	if resp.Error != nil {
		return &Result{
			Error: fmt.Sprintf("MCP error %d: %s", resp.Error.Code, resp.Error.Message),
		}, nil
	}
	
	return &Result{
		Data: resp.Result,
		Metadata: map[string]interface{}{
			"server": t.config.Server,
			"method": method,
			"id":     resp.ID,
		},
	}, nil
}

func (t *MCPTool) Close() error {
	return nil
}

func (c *MCPClient) Call(ctx context.Context, req *MCPRequest) (*MCPResponse, error) {
	// For demo purposes, simulate MCP server communication
	// In a real implementation, this would use the MCP protocol over stdio, HTTP, or WebSocket
	
	time.Sleep(100 * time.Millisecond) // Simulate network latency
	
	// Mock response based on method
	var result interface{}
	switch req.Method {
	case "list_tools":
		result = map[string]interface{}{
			"tools": []map[string]interface{}{
				{
					"name":        "file_read",
					"description": "Read a file from the filesystem",
					"parameters": map[string]interface{}{
						"type": "object",
						"properties": map[string]interface{}{
							"path": map[string]interface{}{
								"type":        "string",
								"description": "Path to the file to read",
							},
						},
						"required": []string{"path"},
					},
				},
				{
					"name":        "web_search",
					"description": "Search the web for information",
					"parameters": map[string]interface{}{
						"type": "object",
						"properties": map[string]interface{}{
							"query": map[string]interface{}{
								"type":        "string",
								"description": "Search query",
							},
						},
						"required": []string{"query"},
					},
				},
			},
		}
	case "call_tool":
		toolName, _ := req.Params["name"].(string)
		switch toolName {
		case "file_read":
			path, _ := req.Params["path"].(string)
			result = map[string]interface{}{
				"content": fmt.Sprintf("Mock file content for: %s", path),
				"path":    path,
			}
		case "web_search":
			query, _ := req.Params["query"].(string)
			result = map[string]interface{}{
				"results": []map[string]interface{}{
					{
						"title": "Mock Search Result 1",
						"url":   "https://example.com/1",
						"snippet": fmt.Sprintf("Mock result for query: %s", query),
					},
					{
						"title": "Mock Search Result 2",
						"url":   "https://example.com/2",
						"snippet": fmt.Sprintf("Another mock result for: %s", query),
					},
				},
				"query": query,
			}
		default:
			return &MCPResponse{
				ID: req.ID,
				Error: &MCPError{
					Code:    -32601,
					Message: fmt.Sprintf("Tool not found: %s", toolName),
				},
			}, nil
		}
	default:
		return &MCPResponse{
			ID: req.ID,
			Error: &MCPError{
				Code:    -32601,
				Message: fmt.Sprintf("Method not found: %s", req.Method),
			},
		}, nil
	}
	
	return &MCPResponse{
		ID:     req.ID,
		Result: result,
	}, nil
}