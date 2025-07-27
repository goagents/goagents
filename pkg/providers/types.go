package providers

import (
	"context"
	"time"
)

type Provider interface {
	Name() string
	Chat(ctx context.Context, req *ChatRequest) (*ChatResponse, error)
	Stream(ctx context.Context, req *ChatRequest) (<-chan *StreamChunk, error)
	Models() []string
	Close() error
}

type ChatRequest struct {
	Model       string             `json:"model"`
	Messages    []Message          `json:"messages"`
	Tools       []Tool             `json:"tools,omitempty"`
	MaxTokens   int                `json:"max_tokens,omitempty"`
	Temperature float64            `json:"temperature,omitempty"`
	TopP        float64            `json:"top_p,omitempty"`
	Stream      bool               `json:"stream,omitempty"`
	Metadata    map[string]string  `json:"metadata,omitempty"`
}

type ChatResponse struct {
	ID      string    `json:"id"`
	Content string    `json:"content"`
	Usage   *Usage    `json:"usage,omitempty"`
	ToolUse []ToolUse `json:"tool_use,omitempty"`
	Model   string    `json:"model"`
	Error   string    `json:"error,omitempty"`
}

type StreamChunk struct {
	ID      string    `json:"id"`
	Content string    `json:"content"`
	Delta   string    `json:"delta"`
	Done    bool      `json:"done"`
	Usage   *Usage    `json:"usage,omitempty"`
	ToolUse []ToolUse `json:"tool_use,omitempty"`
	Error   string    `json:"error,omitempty"`
}

type Message struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type Tool struct {
	Name        string                 `json:"name"`
	Description string                 `json:"description"`
	Parameters  map[string]interface{} `json:"parameters"`
}

type ToolUse struct {
	ID   string                 `json:"id"`
	Name string                 `json:"name"`
	Args map[string]interface{} `json:"args"`
}

type Usage struct {
	PromptTokens     int `json:"prompt_tokens"`
	CompletionTokens int `json:"completion_tokens"`
	TotalTokens      int `json:"total_tokens"`
}

type Config struct {
	Anthropic *AnthropicConfig `json:"anthropic,omitempty"`
	OpenAI    *OpenAIConfig    `json:"openai,omitempty"`
	Gemini    *GeminiConfig    `json:"gemini,omitempty"`
}

type AnthropicConfig struct {
	APIKey  string        `json:"api_key"`
	BaseURL string        `json:"base_url,omitempty"`
	Version string        `json:"version,omitempty"`
	Timeout time.Duration `json:"timeout,omitempty"`
}

type OpenAIConfig struct {
	APIKey  string        `json:"api_key"`
	BaseURL string        `json:"base_url,omitempty"`
	OrgID   string        `json:"org_id,omitempty"`
	Timeout time.Duration `json:"timeout,omitempty"`
}

type GeminiConfig struct {
	APIKey    string        `json:"api_key"`
	ProjectID string        `json:"project_id,omitempty"`
	Timeout   time.Duration `json:"timeout,omitempty"`
}

type Manager struct {
	providers map[string]Provider
}

func NewManager() *Manager {
	return &Manager{
		providers: make(map[string]Provider),
	}
}

func (m *Manager) RegisterProvider(name string, provider Provider) {
	m.providers[name] = provider
}

func (m *Manager) GetProvider(name string) (Provider, bool) {
	provider, exists := m.providers[name]
	return provider, exists
}

func (m *Manager) ListProviders() []string {
	names := make([]string, 0, len(m.providers))
	for name := range m.providers {
		names = append(names, name)
	}
	return names
}

func (m *Manager) Close() error {
	for _, provider := range m.providers {
		if err := provider.Close(); err != nil {
			return err
		}
	}
	return nil
}