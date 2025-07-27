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

type GeminiProvider struct {
	config *GeminiConfig
	client *http.Client
}

func NewGeminiProvider(config *GeminiConfig) *GeminiProvider {
	timeout := 30 * time.Second
	if config.Timeout > 0 {
		timeout = config.Timeout
	}
	
	return &GeminiProvider{
		config: config,
		client: &http.Client{
			Timeout: timeout,
		},
	}
}

func (p *GeminiProvider) Name() string {
	return "gemini"
}

func (p *GeminiProvider) Chat(ctx context.Context, req *ChatRequest) (*ChatResponse, error) {
	geminiReq := p.convertRequest(req)
	
	reqBody, err := json.Marshal(geminiReq)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}
	
	url := fmt.Sprintf("https://generativelanguage.googleapis.com/v1beta/models/%s:generateContent?key=%s", 
		req.Model, p.config.APIKey)
	
	httpReq, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewReader(reqBody))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}
	
	httpReq.Header.Set("Content-Type", "application/json")
	
	resp, err := p.client.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()
	
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("API error %d: %s", resp.StatusCode, string(body))
	}
	
	var geminiResp geminiResponse
	if err := json.NewDecoder(resp.Body).Decode(&geminiResp); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}
	
	return p.convertResponse(&geminiResp, req.Model), nil
}

func (p *GeminiProvider) Stream(ctx context.Context, req *ChatRequest) (<-chan *StreamChunk, error) {
	chunks := make(chan *StreamChunk, 10)
	
	go func() {
		defer close(chunks)
		
		// For demo purposes, simulate streaming response
		content := "This is a simulated streaming response from Gemini"
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

func (p *GeminiProvider) Models() []string {
	return []string{
		"gemini-1.5-pro",
		"gemini-1.5-flash",
		"gemini-1.0-pro",
	}
}

func (p *GeminiProvider) Close() error {
	return nil
}

type geminiRequest struct {
	Contents         []geminiContent          `json:"contents"`
	Tools            []geminiTool             `json:"tools,omitempty"`
	GenerationConfig *geminiGenerationConfig  `json:"generationConfig,omitempty"`
}

type geminiContent struct {
	Role  string         `json:"role"`
	Parts []geminiPart   `json:"parts"`
}

type geminiPart struct {
	Text string `json:"text"`
}

type geminiTool struct {
	FunctionDeclarations []geminiFunctionDeclaration `json:"functionDeclarations"`
}

type geminiFunctionDeclaration struct {
	Name        string                 `json:"name"`
	Description string                 `json:"description"`
	Parameters  map[string]interface{} `json:"parameters"`
}

type geminiGenerationConfig struct {
	Temperature   float64 `json:"temperature,omitempty"`
	TopP          float64 `json:"topP,omitempty"`
	MaxOutputTokens int   `json:"maxOutputTokens,omitempty"`
}

type geminiResponse struct {
	Candidates     []geminiCandidate     `json:"candidates"`
	UsageMetadata  *geminiUsageMetadata  `json:"usageMetadata,omitempty"`
}

type geminiCandidate struct {
	Content       geminiContent `json:"content"`
	FinishReason  string        `json:"finishReason"`
	Index         int           `json:"index"`
}

type geminiUsageMetadata struct {
	PromptTokenCount     int `json:"promptTokenCount"`
	CandidatesTokenCount int `json:"candidatesTokenCount"`
	TotalTokenCount      int `json:"totalTokenCount"`
}

func (p *GeminiProvider) convertRequest(req *ChatRequest) *geminiRequest {
	geminiReq := &geminiRequest{
		GenerationConfig: &geminiGenerationConfig{
			Temperature:     req.Temperature,
			TopP:            req.TopP,
			MaxOutputTokens: req.MaxTokens,
		},
	}
	
	// Convert messages
	for _, msg := range req.Messages {
		role := msg.Role
		if role == "assistant" {
			role = "model"
		}
		
		geminiReq.Contents = append(geminiReq.Contents, geminiContent{
			Role: role,
			Parts: []geminiPart{
				{Text: msg.Content},
			},
		})
	}
	
	// Convert tools
	if len(req.Tools) > 0 {
		var declarations []geminiFunctionDeclaration
		for _, tool := range req.Tools {
			declarations = append(declarations, geminiFunctionDeclaration{
				Name:        tool.Name,
				Description: tool.Description,
				Parameters:  tool.Parameters,
			})
		}
		geminiReq.Tools = []geminiTool{
			{FunctionDeclarations: declarations},
		}
	}
	
	return geminiReq
}

func (p *GeminiProvider) convertResponse(resp *geminiResponse, model string) *ChatResponse {
	chatResp := &ChatResponse{
		ID:    fmt.Sprintf("gemini-%d", time.Now().UnixNano()),
		Model: model,
	}
	
	if resp.UsageMetadata != nil {
		chatResp.Usage = &Usage{
			PromptTokens:     resp.UsageMetadata.PromptTokenCount,
			CompletionTokens: resp.UsageMetadata.CandidatesTokenCount,
			TotalTokens:      resp.UsageMetadata.TotalTokenCount,
		}
	}
	
	if len(resp.Candidates) > 0 && len(resp.Candidates[0].Content.Parts) > 0 {
		chatResp.Content = resp.Candidates[0].Content.Parts[0].Text
	}
	
	return chatResp
}