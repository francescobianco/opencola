package provider

import (
	"context"
	"encoding/json"
	"fmt"
	"os"

	openai "github.com/sashabaranov/go-openai"
)

type OpenAIProvider struct {
	client *openai.Client
	model  string
	name   string
}

func NewOpenAI(apiKey, model, baseURL string) *OpenAIProvider {
	if apiKey == "" {
		apiKey = os.Getenv("OPENAI_API_KEY")
	}
	if model == "" {
		model = "gpt-4o"
	}

	cfg := openai.DefaultConfig(apiKey)
	if baseURL != "" {
		cfg.BaseURL = baseURL
	}

	return &OpenAIProvider{
		client: openai.NewClientWithConfig(cfg),
		model:  model,
		name:   "openai",
	}
}

func (p *OpenAIProvider) Name() string {
	return p.name
}

func (p *OpenAIProvider) Chat(ctx context.Context, messages []Message, tools []ToolDefinition) (Response, error) {
	var openaiMessages []openai.ChatCompletionMessage
	for _, m := range messages {
		openaiMessages = append(openaiMessages, openai.ChatCompletionMessage{
			Role:    m.Role,
			Content: m.Content,
		})
	}

	var openaiTools []openai.Tool
	for _, t := range tools {
		openaiTools = append(openaiTools, openai.Tool{
			Type: openai.ToolTypeFunction,
			Function: &openai.FunctionDefinition{
				Name:        t.Name,
				Description: t.Description,
				Parameters:  t.Parameters,
			},
		})
	}

	req := openai.ChatCompletionRequest{
		Model:    p.model,
		Messages: openaiMessages,
	}
	if len(openaiTools) > 0 {
		req.Tools = openaiTools
	}

	resp, err := p.client.CreateChatCompletion(ctx, req)
	if err != nil {
		return Response{}, fmt.Errorf("openai chat completion: %w", err)
	}

	if len(resp.Choices) == 0 {
		return Response{}, fmt.Errorf("no choices in response")
	}

	choice := resp.Choices[0].Message
	var toolCalls []ToolCall
	for _, tc := range choice.ToolCalls {
		toolCalls = append(toolCalls, ToolCall{
			Name:      tc.Function.Name,
			Arguments: tc.Function.Arguments,
		})
	}

	return Response{
		Content:   choice.Content,
		ToolCalls: toolCalls,
	}, nil
}

func (p *OpenAIProvider) ListModels(ctx context.Context) ([]ModelInfo, error) {
	models, err := p.client.ListModels(ctx)
	if err != nil {
		return nil, fmt.Errorf("list models: %w", err)
	}

	var result []ModelInfo
	for _, m := range models.Models {
		result = append(result, ModelInfo{
			ID:   m.ID,
			Name: m.ID,
		})
	}
	return result, nil
}

func toJSON(v any) json.RawMessage {
	b, _ := json.Marshal(v)
	return b
}
