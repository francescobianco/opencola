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
const author = "by Francesco Bianco <bianco@javanile.org>"

const banner = `OpenCola - minimal coding agent
`

func Run() error {
	cfgPath := config.DefaultConfigPath()
	cfg, err := config.Load(cfgPath)
	if err != nil {
		cfg = &config.Config{}
	}

	envPath := config.DefaultEnvPath()
	envCfg := config.LoadEnv(envPath)

	if envCfg.APIKey != "" && cfg.ActiveProvider() == nil {
		cfg.AddProvider(config.ProviderConfig{
			Name:     "openai",
			Provider: "openai",
			APIKey:   envCfg.APIKey,
			BaseURL:  envCfg.BaseURL,
			Model:    envCfg.Model,
		})
		cfg.Save(cfgPath)
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
	fmt.Println(author)
	fmt.Println()

	input := NewInputReader()
	input.LoadHistory(config.DefaultHistoryPath())
	defer input.SaveHistory(config.DefaultHistoryPath())

	ctx := context.Background()

	for {
		drawStatusBar(ag)
		line, err := input.ReadLine()
		if err != nil {
			fmt.Println()
			printGoodbye()
			break
		}

		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}

		if input.IsVimQuit(line) {
			fmt.Println()
			printGoodbye()
			break
		}

		fmt.Println()

		if strings.HasPrefix(line, "/") {
			if handleCommand(line, ag, cfg, cfgPath, envCfg, envPath, ctx) {
				printGoodbye()
				break
			}
			drawStatusBar(ag)
			continue
		}

		if strings.ToLower(line) == "clear" {
			fmt.Print("\033[2J\033[H")
			fmt.Print(banner)
			fmt.Println(author)
			fmt.Println()
			drawStatusBar(ag)
			continue
		}

		output, err := ag.Run(ctx, line)
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

func handleCommand(input string, ag *agent.Agent, cfg *config.Config, cfgPath string, envCfg *config.EnvConfig, envPath string, ctx context.Context) bool {
	parts := strings.Fields(input)
	cmd := strings.ToLower(parts[0])

	switch cmd {
	case "/help":
		fmt.Println("Available commands:")
		fmt.Println("  /connect <provider> <api_key> [base_url]  - Connect to a provider")
		fmt.Println("  /models                                   - List available models")
		fmt.Println("  /reset                                    - Reset conversation")
		fmt.Println("  /clear                                    - Clear the screen")
		fmt.Println("  /status                                   - Show current status")
		fmt.Println("  /exit, /quit, :q                          - Exit the program")
		fmt.Println("  /help                                     - Show this help")

	case "/connect":
		if len(parts) < 3 {
			fmt.Println("Usage: /connect <provider> <api_key> [base_url]")
			fmt.Println("  provider: openai")
			return false
		}

		provType := parts[1]
		apiKey := parts[2]
		baseURL := ""
		model := ""
		if len(parts) > 3 {
			baseURL = parts[3]
		}
		if len(parts) > 4 {
			model = parts[4]
		}

		var prov provider.Provider
		switch provType {
		case "openai":
			prov = provider.NewOpenAI(apiKey, model, baseURL)
		default:
			fmt.Printf("Unknown provider: %s\n", provType)
			return false
		}

		ag.SetProvider(prov)
		cfg.AddProvider(config.ProviderConfig{
			Name:     prov.Name(),
			Provider: provType,
			APIKey:   apiKey,
			BaseURL:  baseURL,
			Model:    model,
		})
		if err := cfg.Save(cfgPath); err != nil {
			fmt.Fprintf(os.Stderr, "Failed to save config: %v\n", err)
		}

		envCfg.APIKey = apiKey
		envCfg.BaseURL = baseURL
		envCfg.Model = model
		if err := envCfg.Save(envPath); err != nil {
			fmt.Fprintf(os.Stderr, "Failed to save env config: %v\n", err)
		}

		fmt.Printf("Connected to %s\n", prov.Name())

	case "/models":
		models, err := ag.ListModels(ctx)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			return false
		}
		fmt.Println("Available models:")
		for _, m := range models {
			fmt.Printf("  %s\n", m.ID)
		}

	case "/reset":
		ag.Reset()
		fmt.Println("Session reset")

	case "/clear":
		fmt.Print("\033[2J\033[H")
		fmt.Print(banner)
		fmt.Println(author)
		fmt.Println()

	case "/status":
		drawStatusBar(ag)

	case "/exit", "/quit":
		printGoodbye()
		return true

	default:
		fmt.Printf("Unknown command: %s. Type /help for available commands.\n", cmd)
	}

	return false
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

	width := getTerminalWidth()
	height := getTerminalHeight()

	bar := fmt.Sprintf(" OpenCola v%s  |  Provider: %s  |  Model: %s  |  Status: %s ",
		version, provName, modelName, status)

	if len(bar) > width {
		bar = bar[:width]
	}

	padding := strings.Repeat(" ", width-len(bar))
	bar += padding

	fmt.Printf("\033[%d;1H", height)
	fmt.Printf("\033[2K")
	fmt.Printf("\033[48;2;30;64;120m\033[38;2;255;255;255m%s\033[0m", bar)
	fmt.Printf("\033[%d;1H", height-1)
	fmt.Printf("\033[2K")
}

func printGoodbye() {
	fmt.Println()
	fmt.Println("Goodbye! Thanks for using OpenCola. See you next time!")
}
