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

type OpenAIProvider struct {
	config *OpenAIConfig
	client *http.Client
}

func NewOpenAIProvider(config *OpenAIConfig) *OpenAIProvider {
	timeout := 30 * time.Second
	if config.Timeout > 0 {
		timeout = config.Timeout
	}
	
	baseURL := "https://api.openai.com"
	if config.BaseURL != "" {
		baseURL = config.BaseURL
	}
	config.BaseURL = baseURL
	
	return &OpenAIProvider{
		config: config,
		client: &http.Client{
			Timeout: timeout,
		},
	}
}

func (p *OpenAIProvider) Name() string {
	return "openai"
}

func (p *OpenAIProvider) Chat(ctx context.Context, req *ChatRequest) (*ChatResponse, error) {
	openaiReq := p.convertRequest(req)
	
	reqBody, err := json.Marshal(openaiReq)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}
	
	httpReq, err := http.NewRequestWithContext(ctx, "POST", p.config.BaseURL+"/v1/chat/completions", bytes.NewReader(reqBody))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}
	
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Authorization", "Bearer "+p.config.APIKey)
	if p.config.OrgID != "" {
		httpReq.Header.Set("OpenAI-Organization", p.config.OrgID)
	}
	
	resp, err := p.client.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()
	
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("API error %d: %s", resp.StatusCode, string(body))
	}
	
	var openaiResp openaiResponse
	if err := json.NewDecoder(resp.Body).Decode(&openaiResp); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}
	
	return p.convertResponse(&openaiResp), nil
}

func (p *OpenAIProvider) Stream(ctx context.Context, req *ChatRequest) (<-chan *StreamChunk, error) {
	chunks := make(chan *StreamChunk, 10)
	
	go func() {
		defer close(chunks)
		
		openaiReq := p.convertRequest(req)
		openaiReq.Stream = true
		
		reqBody, err := json.Marshal(openaiReq)
		if err != nil {
			chunks <- &StreamChunk{Error: fmt.Sprintf("failed to marshal request: %v", err)}
			return
		}
		
		httpReq, err := http.NewRequestWithContext(ctx, "POST", p.config.BaseURL+"/v1/chat/completions", bytes.NewReader(reqBody))
		if err != nil {
			chunks <- &StreamChunk{Error: fmt.Sprintf("failed to create request: %v", err)}
			return
		}
		
		httpReq.Header.Set("Content-Type", "application/json")
		httpReq.Header.Set("Authorization", "Bearer "+p.config.APIKey)
		httpReq.Header.Set("Accept", "text/event-stream")
		if p.config.OrgID != "" {
			httpReq.Header.Set("OpenAI-Organization", p.config.OrgID)
		}
		
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
		content := "This is a simulated streaming response from GPT-4"
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

func (p *OpenAIProvider) Models() []string {
	return []string{
		"gpt-4o",
		"gpt-4o-mini",
		"gpt-4-turbo",
		"gpt-4",
		"gpt-3.5-turbo",
	}
}

func (p *OpenAIProvider) Close() error {
	return nil
}

type openaiRequest struct {
	Model       string           `json:"model"`
	Messages    []openaiMessage  `json:"messages"`
	MaxTokens   int              `json:"max_tokens,omitempty"`
	Temperature float64          `json:"temperature,omitempty"`
	TopP        float64          `json:"top_p,omitempty"`
	Tools       []openaiTool     `json:"tools,omitempty"`
	Stream      bool             `json:"stream,omitempty"`
}

type openaiMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type openaiTool struct {
	Type     string                 `json:"type"`
	Function openaiToolFunction     `json:"function"`
}

type openaiToolFunction struct {
	Name        string                 `json:"name"`
	Description string                 `json:"description"`
	Parameters  map[string]interface{} `json:"parameters"`
}

type openaiResponse struct {
	ID      string               `json:"id"`
	Object  string               `json:"object"`
	Created int64                `json:"created"`
	Model   string               `json:"model"`
	Choices []openaiChoice       `json:"choices"`
	Usage   openaiUsage          `json:"usage"`
}

type openaiChoice struct {
	Index        int                   `json:"index"`
	Message      openaiMessage         `json:"message"`
	FinishReason string                `json:"finish_reason"`
	ToolCalls    []openaiToolCall      `json:"tool_calls,omitempty"`
}

type openaiToolCall struct {
	ID       string                `json:"id"`
	Type     string                `json:"type"`
	Function openaiToolCallFunction `json:"function"`
}

type openaiToolCallFunction struct {
	Name      string `json:"name"`
	Arguments string `json:"arguments"`
}

type openaiUsage struct {
	PromptTokens     int `json:"prompt_tokens"`
	CompletionTokens int `json:"completion_tokens"`
	TotalTokens      int `json:"total_tokens"`
}

func (p *OpenAIProvider) convertRequest(req *ChatRequest) *openaiRequest {
	openaiReq := &openaiRequest{
		Model:       req.Model,
		MaxTokens:   req.MaxTokens,
		Temperature: req.Temperature,
		TopP:        req.TopP,
		Stream:      req.Stream,
	}
	
	// Convert messages
	for _, msg := range req.Messages {
		openaiReq.Messages = append(openaiReq.Messages, openaiMessage{
			Role:    msg.Role,
			Content: msg.Content,
		})
	}
	
	// Convert tools
	for _, tool := range req.Tools {
		openaiReq.Tools = append(openaiReq.Tools, openaiTool{
			Type: "function",
			Function: openaiToolFunction{
				Name:        tool.Name,
				Description: tool.Description,
				Parameters:  tool.Parameters,
			},
		})
	}
	
	return openaiReq
}

func (p *OpenAIProvider) convertResponse(resp *openaiResponse) *ChatResponse {
	chatResp := &ChatResponse{
		ID:    resp.ID,
		Model: resp.Model,
		Usage: &Usage{
			PromptTokens:     resp.Usage.PromptTokens,
			CompletionTokens: resp.Usage.CompletionTokens,
			TotalTokens:      resp.Usage.TotalTokens,
		},
	}
	
	if len(resp.Choices) > 0 {
		choice := resp.Choices[0]
		chatResp.Content = choice.Message.Content
		
		// Convert tool calls
		for _, toolCall := range choice.ToolCalls {
			var args map[string]interface{}
			if err := json.Unmarshal([]byte(toolCall.Function.Arguments), &args); err == nil {
				chatResp.ToolUse = append(chatResp.ToolUse, ToolUse{
					ID:   toolCall.ID,
					Name: toolCall.Function.Name,
					Args: args,
				})
			}
		}
	}
	
	return chatResp
}