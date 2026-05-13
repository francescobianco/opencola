package cmd

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"slices"
	"strings"

	"github.com/opencola/opencola/agent"
	"github.com/opencola/opencola/config"
	"github.com/opencola/opencola/provider"
	"github.com/opencola/opencola/tools"
	"golang.org/x/term"
)

const version = "0.1.0"
const author = "by Francesco Bianco <bianco@javanile.org>"

var providers = []string{"opencode", "opencode-go", "opencode-zen"}

func Run() error {
	fmt.Print("\033[2J\033[H")

	cfgPath := config.DefaultConfigPath()
	cfg, err := config.Load(cfgPath)
	if err != nil {
		cfg = &config.Config{}
	}

	envPath := config.DefaultEnvPath()
	envCfg := config.LoadEnv(envPath)

	if envCfg.APIKey != "" && cfg.ActiveProvider() == nil {
		cfg.AddProvider(config.ProviderConfig{
			Name:     "opencode",
			Provider: "opencode",
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
		prov := newProvider(p.Provider, p.APIKey, p.Model, p.BaseURL)
		ag.SetProvider(prov)
	}

	printBanner()

	input := NewInputReader()
	input.LoadHistory(config.DefaultHistoryPath())
	defer input.SaveHistory(config.DefaultHistoryPath())

	ctx := context.Background()

	for {
		drawStatusBar(ag)
		line, err := input.ReadLine()
		if err != nil {
			printGoodbye()
			break
		}

		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}

		if input.IsVimQuit(line) {
			printGoodbye()
			break
		}

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
			printBanner()
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

func printBanner() {
	height := getTerminalHeight()
	for i := 0; i < height/3; i++ {
		fmt.Println()
	}
	fmt.Printf("\033[1mOpenCola\033[0m - minimal coding agent\n")
	fmt.Println(author)
	fmt.Println()
	fmt.Print("> ")
}

func handleCommand(input string, ag *agent.Agent, cfg *config.Config, cfgPath string, envCfg *config.EnvConfig, envPath string, ctx context.Context) bool {
	parts := strings.Fields(input)
	cmd := strings.ToLower(parts[0])

	switch cmd {
	case "/help":
		fmt.Println("Available commands:")
		fmt.Println("  /connect <provider>                     - Connect to a provider")
		fmt.Println("  /models                                 - Select a model")
		fmt.Println("  /reset                                  - Reset conversation")
		fmt.Println("  /clear                                  - Clear the screen")
		fmt.Println("  /status                                 - Show current status")
		fmt.Println("  /exit, /quit, :q                        - Exit the program")
		fmt.Println("  /help                                   - Show this help")

	case "/connect":
		if len(parts) < 2 {
			fmt.Println("Usage: /connect <provider>")
			fmt.Printf("  providers: %s\n", strings.Join(providers, ", "))
			return false
		}

		provType := parts[1]
		if !slices.Contains(providers, provType) {
			fmt.Printf("Unknown provider: %s\n", provType)
			fmt.Printf("Available: %s\n", strings.Join(providers, ", "))
			return false
		}

		fmt.Printf("Please enter your API key for %s: ", provType)

		fd := int(os.Stdin.Fd())
		state, _ := term.MakeRaw(fd)
		apiKey, _ := bufio.NewReader(os.Stdin).ReadString('\n')
		term.Restore(fd, state)
		apiKey = strings.TrimSpace(apiKey)
		fmt.Println()

		if apiKey == "" {
			fmt.Println("API key is required")
			return false
		}

		baseURL := getProviderBaseURL(provType)
		prov := newProvider(provType, apiKey, "", baseURL)

		ag.SetProvider(prov)
		cfg.AddProvider(config.ProviderConfig{
			Name:     prov.Name(),
			Provider: provType,
			APIKey:   apiKey,
			BaseURL:  baseURL,
		})
		cfg.Save(cfgPath)

		envCfg.APIKey = apiKey
		envCfg.BaseURL = baseURL
		envCfg.Save(envPath)

		fmt.Printf("Connected to %s\n", prov.Name())

	case "/models":
		if !ag.IsConnected() {
			fmt.Println("Not connected to any provider. Use /connect first.")
			return false
		}

		models, err := ag.ListModels(ctx)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			return false
		}

		selected := showModelMenu(models)
		if selected != "" {
			ag.SetModel(selected)
			fmt.Printf("Selected model: %s\n", selected)
		}

	case "/reset":
		ag.Reset()
		fmt.Println("Session reset")

	case "/clear":
		fmt.Print("\033[2J\033[H")
		printBanner()

	case "/status":
		drawStatusBar(ag)

	case "/exit", "/quit":
		return true

	default:
		fmt.Printf("Unknown command: %s. Type /help for available commands.\n", cmd)
	}

	return false
}

func showModelMenu(models []provider.ModelInfo) string {
	if len(models) == 0 {
		fmt.Println("No models available")
		return ""
	}

	maxShow := 4
	if len(models) < maxShow {
		maxShow = len(models)
	}

	selected := 0

	fd := int(os.Stdin.Fd())
	state, _ := term.MakeRaw(fd)
	defer term.Restore(fd, state)

	reader := bufio.NewReader(os.Stdin)

	for {
		fmt.Println()
		for i := 0; i < maxShow; i++ {
			idx := i
			if idx >= len(models) {
				break
			}
			cursor := "  "
			if i == selected {
				cursor = "> "
			}
			fmt.Printf("%s %s\n", cursor, models[idx].ID)
		}

		b, _, _ := reader.ReadRune()
		switch b {
		case 'A':
			if selected > 0 {
				selected--
			}
			fmt.Printf("\033[%dA", maxShow)
			for i := 0; i < maxShow; i++ {
				fmt.Print("\033[2K\033[G")
			}

		case 'B':
			if selected < maxShow-1 && selected < len(models)-1 {
				selected++
			}
			fmt.Printf("\033[%dA", maxShow)
			for i := 0; i < maxShow; i++ {
				fmt.Print("\033[2K\033[G")
			}

		case '\r', '\n':
			for i := 0; i < maxShow; i++ {
				fmt.Print("\033[2K\033[G")
			}
			fmt.Print("\033[1A\033[2K\033[G")
			return models[selected].ID
		}
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

	width := getTerminalWidth()
	height := getTerminalHeight()

	logo := fmt.Sprintf(" OpenCola v%s ", version)
	rest := fmt.Sprintf(" Provider: %s  Model: %s  Status: %s ", provName, modelName, status)

	bar := logo + rest
	if len(bar) > width {
		bar = bar[:width]
	}

	padding := strings.Repeat(" ", width-len(bar))
	bar += padding

	logoLen := len(logo)
	restPart := bar[logoLen:]

	fmt.Printf("\033[%d;1H", height)
	fmt.Printf("\033[2K")
	fmt.Printf("\033[48;2;255;255;255m\033[38;2;30;64;120m%s\033[0m", logo)
	fmt.Printf("\033[48;2;30;64;120m\033[38;2;255;255;255m%s\033[0m", restPart)
	fmt.Printf("\033[%d;1H", height-1)
	fmt.Printf("\033[2K")
	fmt.Print("> ")
}

func printGoodbye() {
	fmt.Print("\033[2J\033[H")
	fmt.Println()
	fmt.Println("Goodbye! Thanks for using OpenCola. See you next time!")
	fmt.Println()
}

func newProvider(name, apiKey, model, baseURL string) provider.Provider {
	return provider.NewOpenAI(apiKey, model, baseURL)
}

func getProviderBaseURL(name string) string {
	switch name {
	case "opencode-go":
		return "https://go.opencode.ai/v1"
	case "opencode-zen":
		return "https://zen.opencode.ai/v1"
	default:
		return "https://api.openai.com/v1"
	}
}
