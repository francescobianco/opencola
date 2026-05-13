package cmd

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/opencola/opencola/agent"
	"github.com/opencola/opencola/config"
	"github.com/opencola/opencola/provider"
	"github.com/opencola/opencola/tools"
)

const banner = `
opencola - coding agent
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

	scanner := bufio.NewScanner(os.Stdin)
	ctx := context.Background()

	fmt.Print(banner)
	printStatus(ag)

	for {
		fmt.Print("> ")
		if !scanner.Scan() {
			break
		}

		input := strings.TrimSpace(scanner.Text())
		if input == "" {
			continue
		}

		if strings.HasPrefix(input, "/") {
			handleCommand(input, ag, cfg, cfgPath, ctx)
			printStatus(ag)
			continue
		}

		output, err := ag.Run(ctx, input)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			continue
		}

		fmt.Println(output)
		fmt.Println()
	}

	return scanner.Err()
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
		printStatus(ag)

	case "/exit", "/quit":
		os.Exit(0)

	default:
		fmt.Printf("Unknown command: %s. Type /help for available commands.\n", cmd)
	}
}

func printStatus(ag *agent.Agent) {
	status := "disconnected"
	if ag.IsConnected() {
		status = ag.ProviderName()
	}
	fmt.Printf("[%s]\n", status)
}
