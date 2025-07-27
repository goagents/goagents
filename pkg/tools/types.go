package tools

import (
	"context"
	"fmt"
	"time"
)

type Tool interface {
	Name() string
	Type() string
	Execute(ctx context.Context, args map[string]interface{}) (*Result, error)
	Close() error
}

type Result struct {
	Data     interface{}            `json:"data"`
	Error    string                 `json:"error,omitempty"`
	Metadata map[string]interface{} `json:"metadata,omitempty"`
}

type Config struct {
	Type     string            `json:"type"`
	Name     string            `json:"name"`
	URL      string            `json:"url,omitempty"`
	Endpoint string            `json:"endpoint,omitempty"`
	Server   string            `json:"server,omitempty"`
	Auth     *AuthConfig       `json:"auth,omitempty"`
	Config   map[string]string `json:"config,omitempty"`
	Timeout  time.Duration     `json:"timeout,omitempty"`
}

type AuthConfig struct {
	Type   string `json:"type"`
	Token  string `json:"token,omitempty"`
	APIKey string `json:"api_key,omitempty"`
	Secret string `json:"secret,omitempty"`
}

type Manager struct {
	tools map[string]Tool
}

func NewManager() *Manager {
	return &Manager{
		tools: make(map[string]Tool),
	}
}

func (m *Manager) RegisterTool(tool Tool) {
	m.tools[tool.Name()] = tool
}

func (m *Manager) GetTool(name string) (Tool, bool) {
	tool, exists := m.tools[name]
	return tool, exists
}

func (m *Manager) ListTools() []Tool {
	tools := make([]Tool, 0, len(m.tools))
	for _, tool := range m.tools {
		tools = append(tools, tool)
	}
	return tools
}

func (m *Manager) Execute(ctx context.Context, name string, args map[string]interface{}) (*Result, error) {
	tool, exists := m.tools[name]
	if !exists {
		return &Result{Error: "tool not found: " + name}, nil
	}
	
	return tool.Execute(ctx, args)
}

func (m *Manager) Close() error {
	for _, tool := range m.tools {
		if err := tool.Close(); err != nil {
			return err
		}
	}
	return nil
}

func CreateTool(config *Config) (Tool, error) {
	switch config.Type {
	case "http":
		return NewHTTPTool(config)
	case "websocket":
		return NewWebSocketTool(config)
	case "mcp":
		return NewMCPTool(config)
	default:
		return nil, fmt.Errorf("unsupported tool type: %s", config.Type)
	}
}