package tools

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
)

type WriteFileTool struct{}

func (w WriteFileTool) Name() string {
	return "write_file"
}

func (w WriteFileTool) Description() string {
	return "Write content to a file"
}

func (w WriteFileTool) Parameters() map[string]any {
	return map[string]any{
		"type": "object",
		"properties": map[string]any{
			"path": map[string]any{
				"type":        "string",
				"description": "Path to the file to write",
			},
			"content": map[string]any{
				"type":        "string",
				"description": "Content to write to the file",
			},
		},
		"required": []string{"path", "content"},
	}
}

type writeFileArgs struct {
	Path    string `json:"path"`
	Content string `json:"content"`
}

func (w WriteFileTool) Execute(ctx context.Context, args string) (string, error) {
	var parsed writeFileArgs
	if err := json.Unmarshal([]byte(args), &parsed); err != nil {
		return "", err
	}

	if parsed.Path == "" {
		return "", ErrInvalidArgs
	}

	dir := filepath.Dir(parsed.Path)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return "", err
	}

	if err := os.WriteFile(filepath.Clean(parsed.Path), []byte(parsed.Content), 0644); err != nil {
		return "", err
	}
	return "File written successfully", nil
}

var ErrInvalidArgs = &InvalidArgsError{}

type InvalidArgsError struct{}

func (e *InvalidArgsError) Error() string {
	return "invalid arguments"
}
