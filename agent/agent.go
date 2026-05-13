package agent

import (
	"context"
	"fmt"
	"strings"

	"github.com/opencola/opencola/provider"
	"github.com/opencola/opencola/session"
	"github.com/opencola/opencola/tools"
)

type Agent struct {
	provider provider.Provider
	tools    map[string]tools.Tool
	session  *session.Session
}

func New(p provider.Provider, toolList []tools.Tool) *Agent {
	toolMap := make(map[string]tools.Tool)
	for _, t := range toolList {
		toolMap[t.Name()] = t
	}
	return &Agent{
		provider: p,
		tools:    toolMap,
		session:  session.New(),
	}
}

func (a *Agent) SetProvider(p provider.Provider) {
	a.provider = p
	a.session.Clear()
}

func (a *Agent) IsConnected() bool {
	return a.provider != nil
}

func (a *Agent) ListModels(ctx context.Context) ([]provider.ModelInfo, error) {
	if a.provider == nil {
		return nil, fmt.Errorf("not connected to any provider")
	}
	return a.provider.ListModels(ctx)
}

func (a *Agent) ProviderName() string {
	if a.provider == nil {
		return "none"
	}
	return a.provider.Name()
}

func (a *Agent) ModelName() string {
	if a.provider == nil {
		return "none"
	}
	return a.provider.ModelName()
}

func (a *Agent) SetModel(model string) {
	if a.provider != nil {
		a.provider.SetModel(model)
	}
}

func (a *Agent) Run(ctx context.Context, input string) (string, error) {
	if a.provider == nil {
		return "", fmt.Errorf("not connected to any provider. Use /connect to connect first")
	}

	a.session.AddMessage("user", input)

	var result strings.Builder

	for {
		tools := a.getToolDefinitions()
		resp, err := a.provider.Chat(ctx, a.session.Messages, tools)
		if err != nil {
			return "", fmt.Errorf("chat: %w", err)
		}

		if resp.Content != "" {
			result.WriteString(resp.Content)
			a.session.AddMessage("assistant", resp.Content)
		}

		if len(resp.ToolCalls) == 0 {
			break
		}

		for _, tc := range resp.ToolCalls {
			tool, ok := a.tools[tc.Name]
			if !ok {
				a.session.AddMessage("assistant", fmt.Sprintf("unknown tool: %s", tc.Name))
				continue
			}

			output, err := tool.Execute(ctx, tc.Arguments)
			if err != nil {
				output = fmt.Sprintf("error: %v", err)
			}

			result.WriteString(fmt.Sprintf("\n[%s] %s", tc.Name, output))
			a.session.AddMessage("assistant", fmt.Sprintf("tool %s: %s", tc.Name, output))
		}
	}

	return result.String(), nil
}

func (a *Agent) getToolDefinitions() []provider.ToolDefinition {
	var defs []provider.ToolDefinition
	for _, t := range a.tools {
		defs = append(defs, provider.ToolDefinition{
			Name:        t.Name(),
			Description: t.Description(),
			Parameters:  t.Parameters(),
		})
	}
	return defs
}

func (a *Agent) Reset() {
	a.session.Clear()
}
