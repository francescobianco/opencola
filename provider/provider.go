package provider

import "context"

type Message struct {
	Role    string
	Content string
}

type ToolCall struct {
	Name      string
	Arguments string
}

type ToolDefinition struct {
	Name        string
	Description string
	Parameters  map[string]any
}

type Response struct {
	Content   string
	ToolCalls []ToolCall
}

type ModelInfo struct {
	ID   string
	Name string
}

type Provider interface {
	Name() string
	ModelName() string
	SetModel(model string)
	Chat(ctx context.Context, messages []Message, tools []ToolDefinition) (Response, error)
	ListModels(ctx context.Context) ([]ModelInfo, error)
}
