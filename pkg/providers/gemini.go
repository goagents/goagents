package providers

import (
	"context"
	"fmt"
	"strings"

	"github.com/google/generative-ai-go/genai"
	"google.golang.org/api/option"
)

type GeminiProvider struct {
	config *GeminiConfig
	client *genai.Client
}

func NewGeminiProvider(config *GeminiConfig) *GeminiProvider {
	ctx := context.Background()
	client, err := genai.NewClient(ctx, option.WithAPIKey(config.APIKey))
	if err != nil {
		// For now, return a provider with nil client - errors will be handled in methods
		return &GeminiProvider{
			config: config,
			client: nil,
		}
	}
	
	return &GeminiProvider{
		config: config,
		client: client,
	}
}

func (p *GeminiProvider) Name() string {
	return "gemini"
}

func (p *GeminiProvider) Chat(ctx context.Context, req *ChatRequest) (*ChatResponse, error) {
	if p.client == nil {
		return nil, fmt.Errorf("gemini client not initialized")
	}
	
	model := p.client.GenerativeModel(req.Model)
	
	// Configure generation settings
	if req.Temperature > 0 {
		temp := float32(req.Temperature)
		model.Temperature = &temp
	}
	if req.TopP > 0 {
		topP := float32(req.TopP)
		model.TopP = &topP
	}
	if req.MaxTokens > 0 {
		maxTokens := int32(req.MaxTokens)
		model.MaxOutputTokens = &maxTokens
	}
	
	// Convert messages to parts
	parts := p.convertMessagesToParts(req.Messages)
	
	resp, err := model.GenerateContent(ctx, parts...)
	if err != nil {
		return nil, fmt.Errorf("gemini API error: %w", err)
	}
	
	return p.convertFromGeminiResponse(resp, req.Model), nil
}

func (p *GeminiProvider) Stream(ctx context.Context, req *ChatRequest) (<-chan *StreamChunk, error) {
	chunks := make(chan *StreamChunk, 10)
	
	go func() {
		defer close(chunks)
		
		if p.client == nil {
			chunks <- &StreamChunk{Error: "gemini client not initialized"}
			return
		}
		
		model := p.client.GenerativeModel(req.Model)
		
		// Configure generation settings
		if req.Temperature > 0 {
			temp := float32(req.Temperature)
			model.Temperature = &temp
		}
		if req.TopP > 0 {
			topP := float32(req.TopP)
			model.TopP = &topP
		}
		if req.MaxTokens > 0 {
			maxTokens := int32(req.MaxTokens)
			model.MaxOutputTokens = &maxTokens
		}
		
		// Convert messages to parts
		parts := p.convertMessagesToParts(req.Messages)
		
		iter := model.GenerateContentStream(ctx, parts...)
		
		var fullContent strings.Builder
		chunkIndex := 0
		
		for {
			resp, err := iter.Next()
			if err != nil {
				if err.Error() == "iterator done" {
					break
				}
				chunks <- &StreamChunk{Error: fmt.Sprintf("streaming error: %v", err)}
				return
			}
			
			for _, candidate := range resp.Candidates {
				if candidate.Content != nil {
					for _, part := range candidate.Content.Parts {
						if textPart, ok := part.(genai.Text); ok {
							text := string(textPart)
							fullContent.WriteString(text)
							
							select {
							case <-ctx.Done():
								return
							case chunks <- &StreamChunk{
								ID:      fmt.Sprintf("chunk_%d", chunkIndex),
								Delta:   text,
								Content: fullContent.String(),
								Done:    false,
							}:
								chunkIndex++
							}
						}
					}
				}
			}
		}
		
		// Send final chunk
		select {
		case <-ctx.Done():
			return
		case chunks <- &StreamChunk{
			ID:      fmt.Sprintf("final_chunk_%d", chunkIndex),
			Delta:   "",
			Content: fullContent.String(),
			Done:    true,
		}:
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
	if p.client != nil {
		return p.client.Close()
	}
	return nil
}

func (p *GeminiProvider) convertMessagesToParts(messages []Message) []genai.Part {
	var parts []genai.Part
	
	for _, msg := range messages {
		// For now, simply concatenate all messages as text parts
		// The Gemini API handles conversation differently than chat completions
		if msg.Role == "system" {
			parts = append(parts, genai.Text(fmt.Sprintf("System: %s", msg.Content)))
		} else if msg.Role == "user" {
			parts = append(parts, genai.Text(msg.Content))
		} else if msg.Role == "assistant" {
			parts = append(parts, genai.Text(fmt.Sprintf("Assistant: %s", msg.Content)))
		}
	}
	
	return parts
}

func (p *GeminiProvider) convertFromGeminiResponse(resp *genai.GenerateContentResponse, model string) *ChatResponse {
	chatResp := &ChatResponse{
		ID:    fmt.Sprintf("gemini-%d", resp.UsageMetadata.TotalTokenCount),
		Model: model,
	}
	
	if resp.UsageMetadata != nil {
		chatResp.Usage = &Usage{
			PromptTokens:     int(resp.UsageMetadata.PromptTokenCount),
			CompletionTokens: int(resp.UsageMetadata.CandidatesTokenCount),
			TotalTokens:      int(resp.UsageMetadata.TotalTokenCount),
		}
	}
	
	// Extract content from candidates
	var content strings.Builder
	for _, candidate := range resp.Candidates {
		if candidate.Content != nil {
			for _, part := range candidate.Content.Parts {
				if textPart, ok := part.(genai.Text); ok {
					content.WriteString(string(textPart))
				}
			}
		}
	}
	chatResp.Content = content.String()
	
	return chatResp
}