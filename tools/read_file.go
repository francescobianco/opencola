package tools

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
)

type ReadFileTool struct{}

func (r ReadFileTool) Name() string {
	return "read_file"
}

func (r ReadFileTool) Description() string {
	return "Read the contents of a file"
}

func (r ReadFileTool) Parameters() map[string]any {
	return map[string]any{
		"type": "object",
		"properties": map[string]any{
			"path": map[string]any{
				"type":        "string",
				"description": "Path to the file to read",
			},
		},
		"required": []string{"path"},
	}
}

type readFileArgs struct {
	Path string `json:"path"`
}

func (r ReadFileTool) Execute(ctx context.Context, args string) (string, error) {
	var parsed readFileArgs
	if err := json.Unmarshal([]byte(args), &parsed); err != nil {
		return "", err
	}

	content, err := os.ReadFile(filepath.Clean(parsed.Path))
	if err != nil {
		return "", err
	}
	return string(content), nil
}
