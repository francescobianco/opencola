package cmd

import (
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/opencola/opencola/agent"
	"github.com/opencola/opencola/config"
	"github.com/opencola/opencola/provider"
	"github.com/opencola/opencola/tools"
)

const version = "0.1.0"

const banner = `opencola - coding agent
Type /help for available commands
`

func Run() error {
	cfgPath := config.DefaultConfigPath()
	cfg, err := config.Load(cfgPath)
	if err != nil {
		cfg = &config.Config{}
	}

	toolList := []tools.Tool{
		tools.ReadFileTool{},
		tools.WriteFileTool{},
		tools.ExecTool{},
	}

	ag := agent.New(nil, toolList)

	if p := cfg.ActiveProvider(); p != nil && p.APIKey != "" {
		prov := provider.NewOpenAI(p.APIKey, p.Model, p.BaseURL)
		ag.SetProvider(prov)
	}

	fmt.Print(banner)

	input := NewInputReader()
	input.LoadHistory(config.DefaultHistoryPath())
	defer input.SaveHistory(config.DefaultHistoryPath())

	ctx := context.Background()

	for {
		renderStatusBar(ag)
		line, err := input.ReadLine()
		if err != nil {
			break
		}

		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}

		fmt.Println()

		if strings.HasPrefix(line, "/") {
			handleCommand(line, ag, cfg, cfgPath, ctx)
			continue
		}

		output, err := ag.Run(ctx, line)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			continue
		}

		fmt.Println(output)
		fmt.Println()
	}

	return nil
}

func handleCommand(input string, ag *agent.Agent, cfg *config.Config, cfgPath string, ctx context.Context) {
	parts := strings.Fields(input)
	cmd := strings.ToLower(parts[0])

	switch cmd {
	case "/help":
		fmt.Println("Available commands:")
		fmt.Println("  /connect <provider> <api_key> [base_url]  - Connect to a provider")
		fmt.Println("  /models                                   - List available models")
		fmt.Println("  /reset                                    - Reset conversation")
		fmt.Println("  /status                                   - Show current status")
		fmt.Println("  /exit                                     - Exit the program")
		fmt.Println("  /help                                     - Show this help")

	case "/connect":
		if len(parts) < 3 {
			fmt.Println("Usage: /connect <provider> <api_key> [base_url]")
			fmt.Println("  provider: openai")
			return
		}

		provType := parts[1]
		apiKey := parts[2]
		baseURL := ""
		if len(parts) > 3 {
			baseURL = parts[3]
		}

		var prov provider.Provider
		switch provType {
		case "openai":
			prov = provider.NewOpenAI(apiKey, "", baseURL)
		default:
			fmt.Printf("Unknown provider: %s\n", provType)
			return
		}

		ag.SetProvider(prov)
		cfg.AddProvider(config.ProviderConfig{
			Name:     prov.Name(),
			Provider: provType,
			APIKey:   apiKey,
			BaseURL:  baseURL,
		})
		if err := cfg.Save(cfgPath); err != nil {
			fmt.Fprintf(os.Stderr, "Failed to save config: %v\n", err)
		}
		fmt.Printf("Connected to %s\n", prov.Name())

	case "/models":
		models, err := ag.ListModels(ctx)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			return
		}
		fmt.Println("Available models:")
		for _, m := range models {
			fmt.Printf("  %s\n", m.ID)
		}

	case "/reset":
		ag.Reset()
		fmt.Println("Session reset")

	case "/status":
		renderStatusBar(ag)

	case "/exit", "/quit":
		os.Exit(0)

	default:
		fmt.Printf("Unknown command: %s. Type /help for available commands.\n", cmd)
	}
}

func renderStatusBar(ag *agent.Agent) {
	status := "Disconnected"
	provName := "none"
	modelName := "none"

	if ag.IsConnected() {
		status = "Connected"
		provName = ag.ProviderName()
		modelName = ag.ModelName()
	}

	width := getTerminalWidth()

	bar := fmt.Sprintf(" OpenCola v%s  |  Provider: %s  |  Model: %s  |  Status: %s ",
		version, provName, modelName, status)

	if len(bar) > width {
		bar = bar[:width]
	}

	padding := strings.Repeat(" ", width-len(bar))
	bar += padding

	fmt.Printf("\033[s")
	fmt.Printf("\033[%d;1H", getTerminalHeight())
	fmt.Printf("\033[48;2;30;64;120m\033[38;2;255;255;255m%s\033[0m", bar)
	fmt.Printf("\033[u")
	fmt.Printf("\033[1A")
}
