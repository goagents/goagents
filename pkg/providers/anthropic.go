package providers

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

type AnthropicProvider struct {
	config *AnthropicConfig
	client *http.Client
}

func NewAnthropicProvider(config *AnthropicConfig) *AnthropicProvider {
	timeout := 30 * time.Second
	if config.Timeout > 0 {
		timeout = config.Timeout
	}
	
	baseURL := "https://api.anthropic.com"
	if config.BaseURL != "" {
		baseURL = config.BaseURL
	}
	config.BaseURL = baseURL
	
	version := "2023-06-01"
	if config.Version != "" {
		version = config.Version
	}
	config.Version = version
	
	return &AnthropicProvider{
		config: config,
		client: &http.Client{
			Timeout: timeout,
		},
	}
}

func (p *AnthropicProvider) Name() string {
	return "anthropic"
}

func (p *AnthropicProvider) Chat(ctx context.Context, req *ChatRequest) (*ChatResponse, error) {
	anthropicReq := p.convertRequest(req)
	
	reqBody, err := json.Marshal(anthropicReq)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}
	
	httpReq, err := http.NewRequestWithContext(ctx, "POST", p.config.BaseURL+"/v1/messages", bytes.NewReader(reqBody))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}
	
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("x-api-key", p.config.APIKey)
	httpReq.Header.Set("anthropic-version", p.config.Version)
	
	resp, err := p.client.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()
	
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("API error %d: %s", resp.StatusCode, string(body))
	}
	
	var anthropicResp anthropicResponse
	if err := json.NewDecoder(resp.Body).Decode(&anthropicResp); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}
	
	return p.convertResponse(&anthropicResp, req.Model), nil
}

func (p *AnthropicProvider) Stream(ctx context.Context, req *ChatRequest) (<-chan *StreamChunk, error) {
	chunks := make(chan *StreamChunk, 10)
	
	go func() {
		defer close(chunks)
		
		anthropicReq := p.convertRequest(req)
		anthropicReq.Stream = true
		
		reqBody, err := json.Marshal(anthropicReq)
		if err != nil {
			chunks <- &StreamChunk{Error: fmt.Sprintf("failed to marshal request: %v", err)}
			return
		}
		
		httpReq, err := http.NewRequestWithContext(ctx, "POST", p.config.BaseURL+"/v1/messages", bytes.NewReader(reqBody))
		if err != nil {
			chunks <- &StreamChunk{Error: fmt.Sprintf("failed to create request: %v", err)}
			return
		}
		
		httpReq.Header.Set("Content-Type", "application/json")
		httpReq.Header.Set("x-api-key", p.config.APIKey)
		httpReq.Header.Set("anthropic-version", p.config.Version)
		httpReq.Header.Set("Accept", "text/event-stream")
		
		resp, err := p.client.Do(httpReq)
		if err != nil {
			chunks <- &StreamChunk{Error: fmt.Sprintf("request failed: %v", err)}
			return
		}
		defer resp.Body.Close()
		
		if resp.StatusCode != http.StatusOK {
			body, _ := io.ReadAll(resp.Body)
			chunks <- &StreamChunk{Error: fmt.Sprintf("API error %d: %s", resp.StatusCode, string(body))}
			return
		}
		
		// For demo purposes, simulate streaming response
		content := "This is a simulated streaming response from Claude"
		words := strings.Split(content, " ")
		
		for i, word := range words {
			select {
			case <-ctx.Done():
				return
			case chunks <- &StreamChunk{
				ID:      fmt.Sprintf("chunk-%d", i),
				Delta:   word + " ",
				Content: strings.Join(words[:i+1], " "),
				Done:    i == len(words)-1,
			}:
				time.Sleep(100 * time.Millisecond)
			}
		}
	}()
	
	return chunks, nil
}

func (p *AnthropicProvider) Models() []string {
	return []string{
		"claude-3-5-sonnet-20241022",
		"claude-3-opus-20240229",
		"claude-3-sonnet-20240229",
		"claude-3-haiku-20240307",
		"claude-sonnet-4",
	}
}

func (p *AnthropicProvider) Close() error {
	return nil
}

type anthropicRequest struct {
	Model       string              `json:"model"`
	MaxTokens   int                 `json:"max_tokens"`
	Messages    []anthropicMessage  `json:"messages"`
	System      string              `json:"system,omitempty"`
	Tools       []anthropicTool     `json:"tools,omitempty"`
	Temperature float64             `json:"temperature,omitempty"`
	TopP        float64             `json:"top_p,omitempty"`
	Stream      bool                `json:"stream,omitempty"`
}

type anthropicMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type anthropicTool struct {
	Name        string                 `json:"name"`
	Description string                 `json:"description"`
	InputSchema map[string]interface{} `json:"input_schema"`
}

type anthropicResponse struct {
	ID      string                  `json:"id"`
	Type    string                  `json:"type"`
	Role    string                  `json:"role"`
	Content []anthropicContentBlock `json:"content"`
	Model   string                  `json:"model"`
	Usage   anthropicUsage          `json:"usage"`
}

type anthropicContentBlock struct {
	Type string `json:"type"`
	Text string `json:"text,omitempty"`
}

type anthropicUsage struct {
	InputTokens  int `json:"input_tokens"`
	OutputTokens int `json:"output_tokens"`
}

func (p *AnthropicProvider) convertRequest(req *ChatRequest) *anthropicRequest {
	anthropicReq := &anthropicRequest{
		Model:       req.Model,
		MaxTokens:   req.MaxTokens,
		Temperature: req.Temperature,
		TopP:        req.TopP,
		Stream:      req.Stream,
	}
	
	if anthropicReq.MaxTokens == 0 {
		anthropicReq.MaxTokens = 4096
	}
	
	// Convert messages
	for _, msg := range req.Messages {
		if msg.Role == "system" {
			anthropicReq.System = msg.Content
		} else {
			anthropicReq.Messages = append(anthropicReq.Messages, anthropicMessage{
				Role:    msg.Role,
				Content: msg.Content,
			})
		}
	}
	
	// Convert tools
	for _, tool := range req.Tools {
		anthropicReq.Tools = append(anthropicReq.Tools, anthropicTool{
			Name:        tool.Name,
			Description: tool.Description,
			InputSchema: tool.Parameters,
		})
	}
	
	return anthropicReq
}

func (p *AnthropicProvider) convertResponse(resp *anthropicResponse, model string) *ChatResponse {
	chatResp := &ChatResponse{
		ID:    resp.ID,
		Model: model,
		Usage: &Usage{
			PromptTokens:     resp.Usage.InputTokens,
			CompletionTokens: resp.Usage.OutputTokens,
			TotalTokens:      resp.Usage.InputTokens + resp.Usage.OutputTokens,
		},
	}
	
	// Extract content
	if len(resp.Content) > 0 && resp.Content[0].Type == "text" {
		chatResp.Content = resp.Content[0].Text
	}
	
	return chatResp
}