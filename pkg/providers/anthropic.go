package providers

import (
	"context"
	"fmt"
	"strings"

	"github.com/anthropics/anthropic-sdk-go"
	"github.com/anthropics/anthropic-sdk-go/option"
)

type AnthropicProvider struct {
	config *AnthropicConfig
	client *anthropic.Client
}

func NewAnthropicProvider(config *AnthropicConfig) *AnthropicProvider {
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
	
	opts := []option.RequestOption{
		option.WithAPIKey(config.APIKey),
	}
	
	if config.BaseURL != "" {
		opts = append(opts, option.WithBaseURL(config.BaseURL))
	}
	
	client := anthropic.NewClient(opts...)
	
	return &AnthropicProvider{
		config: config,
		client: &client,
	}
}

func (p *AnthropicProvider) Name() string {
	return "anthropic"
}

func (p *AnthropicProvider) Chat(ctx context.Context, req *ChatRequest) (*ChatResponse, error) {
	messageReq := p.convertToMessageRequest(req)
	
	resp, err := p.client.Messages.New(ctx, messageReq)
	if err != nil {
		return nil, fmt.Errorf("anthropic API error: %w", err)
	}
	
	return p.convertFromMessageResponse(resp, req.Model), nil
}

func (p *AnthropicProvider) Stream(ctx context.Context, req *ChatRequest) (<-chan *StreamChunk, error) {
	chunks := make(chan *StreamChunk, 10)
	
	go func() {
		defer close(chunks)
		
		messageReq := p.convertToMessageRequest(req)
		
		stream := p.client.Messages.NewStreaming(ctx, messageReq)
		
		var fullContent strings.Builder
		chunkIndex := 0
		message := anthropic.Message{}
		
		for stream.Next() {
			event := stream.Current()
			err := message.Accumulate(event)
			if err != nil {
				chunks <- &StreamChunk{Error: fmt.Sprintf("accumulation error: %v", err)}
				return
			}
			
			switch eventVariant := event.AsAny().(type) {
			case anthropic.ContentBlockDeltaEvent:
				switch deltaVariant := eventVariant.Delta.AsAny().(type) {
				case anthropic.TextDelta:
					if deltaVariant.Text != "" {
						fullContent.WriteString(deltaVariant.Text)
						
						select {
						case <-ctx.Done():
							return
						case chunks <- &StreamChunk{
							ID:      fmt.Sprintf("chunk_%d", chunkIndex),
							Delta:   deltaVariant.Text,
							Content: fullContent.String(),
							Done:    false,
						}:
							chunkIndex++
						}
					}
				}
			}
		}
		
		if err := stream.Err(); err != nil {
			chunks <- &StreamChunk{Error: fmt.Sprintf("streaming error: %v", err)}
			return
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

func (p *AnthropicProvider) Models() []string {
	return []string{
		"claude-3-5-sonnet-20241022",
		"claude-3-5-haiku-20241022",
		"claude-3-opus-20240229",
		"claude-3-sonnet-20240229",
		"claude-3-haiku-20240307",
	}
}

func (p *AnthropicProvider) Close() error {
	return nil
}

func (p *AnthropicProvider) convertToMessageRequest(req *ChatRequest) anthropic.MessageNewParams {
	maxTokens := int64(req.MaxTokens)
	if maxTokens == 0 {
		maxTokens = 4096
	}
	
	messageReq := anthropic.MessageNewParams{
		Model:     anthropic.Model(req.Model),
		MaxTokens: maxTokens,
	}
	
	if req.Temperature > 0 {
		messageReq.Temperature = anthropic.Float(req.Temperature)
	}
	
	if req.TopP > 0 {
		messageReq.TopP = anthropic.Float(req.TopP)
	}
	
	// Convert messages
	var messages []anthropic.MessageParam
	var systemMessage string
	
	for _, msg := range req.Messages {
		if msg.Role == "system" {
			systemMessage = msg.Content
		} else {
			var messageParam anthropic.MessageParam
			if msg.Role == "user" {
				messageParam = anthropic.NewUserMessage(anthropic.NewTextBlock(msg.Content))
			} else if msg.Role == "assistant" {
				messageParam = anthropic.NewAssistantMessage(anthropic.NewTextBlock(msg.Content))
			}
			messages = append(messages, messageParam)
		}
	}
	
	messageReq.Messages = messages
	
	if systemMessage != "" {
		messageReq.System = []anthropic.TextBlockParam{{
			Type: "text",
			Text: systemMessage,
		}}
	}
	
	// Convert tools - skip for now to get basic functionality working
	
	return messageReq
}


func (p *AnthropicProvider) convertFromMessageResponse(resp *anthropic.Message, model string) *ChatResponse {
	chatResp := &ChatResponse{
		ID:    resp.ID,
		Model: model,
		Usage: &Usage{
			PromptTokens:     int(resp.Usage.InputTokens),
			CompletionTokens: int(resp.Usage.OutputTokens),
			TotalTokens:      int(resp.Usage.InputTokens + resp.Usage.OutputTokens),
		},
	}
	
	// Extract content from response
	var content strings.Builder
	for _, block := range resp.Content {
		switch contentBlock := block.AsAny().(type) {
		case anthropic.TextBlock:
			content.WriteString(contentBlock.Text)
		}
	}
	chatResp.Content = content.String()
	
	return chatResp
}