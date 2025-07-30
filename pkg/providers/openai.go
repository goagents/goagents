package providers

import (
	"context"
	"fmt"
	"strings"

	"github.com/openai/openai-go"
	"github.com/openai/openai-go/option"
)

type OpenAIProvider struct {
	config *OpenAIConfig
	client *openai.Client
}

func NewOpenAIProvider(config *OpenAIConfig) *OpenAIProvider {
	baseURL := "https://api.openai.com"
	if config.BaseURL != "" {
		baseURL = config.BaseURL
	}
	config.BaseURL = baseURL
	
	opts := []option.RequestOption{
		option.WithAPIKey(config.APIKey),
	}
	
	if config.BaseURL != "" {
		opts = append(opts, option.WithBaseURL(config.BaseURL))
	}
	
	client := openai.NewClient(opts...)
	
	return &OpenAIProvider{
		config: config,
		client: &client,
	}
}

func (p *OpenAIProvider) Name() string {
	return "openai"
}

func (p *OpenAIProvider) Chat(ctx context.Context, req *ChatRequest) (*ChatResponse, error) {
	params := p.convertToChatCompletionParams(req)
	
	resp, err := p.client.Chat.Completions.New(ctx, params)
	if err != nil {
		return nil, fmt.Errorf("openai API error: %w", err)
	}
	
	return p.convertFromChatCompletion(resp), nil
}

func (p *OpenAIProvider) Stream(ctx context.Context, req *ChatRequest) (<-chan *StreamChunk, error) {
	chunks := make(chan *StreamChunk, 10)
	
	go func() {
		defer close(chunks)
		
		params := p.convertToChatCompletionParams(req)
		
		stream := p.client.Chat.Completions.NewStreaming(ctx, params)
		
		var fullContent strings.Builder
		chunkIndex := 0
		acc := openai.ChatCompletionAccumulator{}
		
		for stream.Next() {
			chunk := stream.Current()
			acc.AddChunk(chunk)
			
			if len(chunk.Choices) > 0 {
				delta := chunk.Choices[0].Delta.Content
				if delta != "" {
					fullContent.WriteString(delta)
					
					select {
					case <-ctx.Done():
						return
					case chunks <- &StreamChunk{
						ID:      chunk.ID,
						Delta:   delta,
						Content: fullContent.String(),
						Done:    false,
					}:
						chunkIndex++
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

func (p *OpenAIProvider) convertToChatCompletionParams(req *ChatRequest) openai.ChatCompletionNewParams {
	params := openai.ChatCompletionNewParams{
		Model: req.Model,
	}
	
	if req.MaxTokens > 0 {
		params.MaxTokens = openai.Int(int64(req.MaxTokens))
	}
	
	if req.Temperature > 0 {
		params.Temperature = openai.Float(req.Temperature)
	}
	
	if req.TopP > 0 {
		params.TopP = openai.Float(req.TopP)
	}
	
	// Convert messages
	messages := []openai.ChatCompletionMessageParamUnion{}
	for _, msg := range req.Messages {
		switch msg.Role {
		case "system":
			messages = append(messages, openai.SystemMessage(msg.Content))
		case "user":
			messages = append(messages, openai.UserMessage(msg.Content))
		case "assistant":
			messages = append(messages, openai.AssistantMessage(msg.Content))
		}
	}
	params.Messages = messages
	
	// Convert tools - skip for now to get basic functionality working
	
	return params
}

func (p *OpenAIProvider) convertFromChatCompletion(resp *openai.ChatCompletion) *ChatResponse {
	chatResp := &ChatResponse{
		ID:    resp.ID,
		Model: resp.Model,
	}
	
	if resp.Usage.PromptTokens > 0 {
		chatResp.Usage = &Usage{
			PromptTokens:     int(resp.Usage.PromptTokens),
			CompletionTokens: int(resp.Usage.CompletionTokens),
			TotalTokens:      int(resp.Usage.TotalTokens),
		}
	}
	
	if len(resp.Choices) > 0 {
		choice := resp.Choices[0]
		if choice.Message.Content != "" {
			chatResp.Content = choice.Message.Content
		}
		
		// Convert tool calls
		for _, toolCall := range choice.Message.ToolCalls {
			if toolCall.Function.Name != "" {
				chatResp.ToolUse = append(chatResp.ToolUse, ToolUse{
					ID:   toolCall.ID,
					Name: toolCall.Function.Name,
					Args: map[string]interface{}{
						"arguments": toolCall.Function.Arguments,
					},
				})
			}
		}
	}
	
	return chatResp
}