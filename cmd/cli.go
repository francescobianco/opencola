package cmd

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"strconv"
	"strings"

	"github.com/opencola/opencola/agent"
	"github.com/opencola/opencola/config"
	"github.com/opencola/opencola/provider"
	"github.com/opencola/opencola/tools"
	"github.com/peterh/liner"
)

const version = "0.1.0"

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

	line := liner.NewLiner()
	defer line.Close()

	line.SetCtrlCAborts(true)

	historyPath := config.DefaultHistoryPath()
	if f, err := os.Open(historyPath); err == nil {
		line.ReadHistory(f)
		f.Close()
	}
	defer func() {
		if f, err := os.Create(historyPath); err == nil {
			line.WriteHistory(f)
			f.Close()
		}
	}()

	fmt.Print(banner)
	drawStatusBar(ag)

	ctx := context.Background()

	for {
		input, err := line.Prompt("> ")
		if err != nil {
			break
		}

		input = strings.TrimSpace(input)
		if input == "" {
			continue
		}

		line.AppendHistory(input)

		if strings.HasPrefix(input, "/") {
			handleCommand(input, ag, cfg, cfgPath, ctx)
			drawStatusBar(ag)
			continue
		}

		output, err := ag.Run(ctx, input)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			drawStatusBar(ag)
			continue
		}

		fmt.Println(output)
		fmt.Println()
		drawStatusBar(ag)
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
		drawStatusBar(ag)

	case "/exit", "/quit":
		os.Exit(0)

	default:
		fmt.Printf("Unknown command: %s. Type /help for available commands.\n", cmd)
	}
}

func drawStatusBar(ag *agent.Agent) {
	status := "Disconnected"
	provName := "none"
	modelName := "none"

	if ag.IsConnected() {
		status = "Connected"
		provName = ag.ProviderName()
		modelName = ag.ModelName()
	}

	clearLine()
	fmt.Printf("\033[38;5;240m\033[48;5;250m OpenCola v%s \033[0m\033[38;5;240m\033[48;5;250m| Provider: \033[38;5;27m\033[48;5;250m%s \033[0m\033[38;5;240m\033[48;5;250m| Model: \033[38;5;34m\033[48;5;250m%s \033[0m\033[38;5;240m\033[48;5;250m| Status: \033[48;5;250m%s \033[0m",
		version, provName, modelName, status)

	if status == "Connected" {
		fmt.Printf("\033[38;5;22m\033[48;5;250m %s\033[0m", status)
	} else {
		fmt.Printf("\033[38;5;160m\033[48;5;250m %s\033[0m", status)
	}

	fmt.Println()
}

func clearLine() {
	fmt.Print("\033[2K\033[G")
}

func getTerminalHeight() int {
	cmd := exec.Command("stty", "size")
	cmd.Stdin = os.Stdin
	out, err := cmd.Output()
	if err != nil {
		return 24
	}
	parts := strings.Fields(string(out))
	if len(parts) < 2 {
		return 24
	}
	h, _ := strconv.Atoi(parts[0])
	return h
}
