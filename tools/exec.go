package tools

import (
	"context"
	"encoding/json"
	"os/exec"
)

type ExecTool struct{}

func (e ExecTool) Name() string {
	return "exec"
}

func (e ExecTool) Description() string {
	return "Execute a shell command"
}

func (e ExecTool) Parameters() map[string]any {
	return map[string]any{
		"type": "object",
		"properties": map[string]any{
			"command": map[string]any{
				"type":        "string",
				"description": "Shell command to execute",
			},
		},
		"required": []string{"command"},
	}
}

type execArgs struct {
	Command string `json:"command"`
}

func (e ExecTool) Execute(ctx context.Context, args string) (string, error) {
	var parsed execArgs
	if err := json.Unmarshal([]byte(args), &parsed); err != nil {
		return "", err
	}

	cmd := exec.CommandContext(ctx, "sh", "-c", parsed.Command)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return string(output), err
	}
	return string(output), nil
}
