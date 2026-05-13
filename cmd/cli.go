package cmd

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"slices"
	"strings"
	"sync"
	"time"

	"github.com/opencola/opencola/agent"
	"github.com/opencola/opencola/config"
	"github.com/opencola/opencola/provider"
	"github.com/opencola/opencola/tools"
	"golang.org/x/term"
)

const version = "0.1.0"
const author = "by Francesco Bianco <bianco@javanile.org>"

var providers = []string{"opencode", "opencode-go", "opencode-zen"}

const spinnerSeed = "...|...||....|.|...||......"

var spinnerBuffer = []byte(spinnerSeed)

type TUI struct {
	ag          *agent.Agent
	cfg         *config.Config
	cfgPath     string
	envCfg      *config.EnvConfig
	envPath     string
	ctx         context.Context
	input       *InputReader
	spinning    bool
	spinnerMu   sync.Mutex
	spinnerDone chan struct{}
}

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

	tui := &TUI{
		ag:          ag,
		cfg:         cfg,
		cfgPath:     cfgPath,
		envCfg:      envCfg,
		envPath:     envPath,
		ctx:         context.Background(),
		input:       NewInputReader(),
		spinning:    true,
		spinnerDone: make(chan struct{}),
	}

	tui.input.LoadHistory(config.DefaultHistoryPath())
	defer tui.input.SaveHistory(config.DefaultHistoryPath())

	tui.renderInitialLayout()
	tui.startSpinner()
	defer tui.stopSpinner()

	for {
		tui.renderPrompt()
		line, err := tui.input.ReadLine()
		if err != nil {
			tui.printGoodbye()
			break
		}

		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}

		if tui.input.IsVimQuit(line) {
			tui.printGoodbye()
			break
		}

		fmt.Println()

		if strings.HasPrefix(line, "/") {
			if tui.handleCommand(line) {
				tui.printGoodbye()
				break
			}
			tui.renderStatusBar()
			continue
		}

		if strings.ToLower(line) == "clear" {
			tui.renderInitialLayout()
			continue
		}

		output, err := tui.ag.RunWithHooks(tui.ctx, line, agent.RunHooks{
			OnLog: tui.renderEventLog,
		})
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			tui.renderStatusBar()
			continue
		}

		tui.renderEventLog("assistant response ready")
		fmt.Println(output)
		fmt.Println()
		tui.renderStatusBar()
	}

	return nil
}

func (t *TUI) renderInitialLayout() {
	height := getTerminalHeight()

	fmt.Print("\033[2J\033[H")

	freeLines := height - 7
	if freeLines < 0 {
		freeLines = 0
	}

	for i := 0; i < freeLines; i++ {
		fmt.Println()
	}

	fmt.Printf("\033[1mOpenCola\033[0m - minimal coding agent\n")
	fmt.Println(author)
	fmt.Println("Type /help for a list of commands.")
	fmt.Println()

	fmt.Printf("\033[%d;1H", height-2)
	fmt.Print("\033[2K")
	fmt.Print("> ")

	t.renderStatusBar()
}

func (t *TUI) renderPrompt() {
	height := getTerminalHeight()
	fmt.Printf("\033[%d;1H\033[2K> ", height-2)
}

func (t *TUI) renderStatusBar() {
	t.spinnerMu.Lock()
	frame := string(spinnerBuffer[:3])
	t.spinnerMu.Unlock()

	status := "Disconnected"
	provName := "none"
	modelName := "none"

	if t.ag.IsConnected() {
		status = "Connected"
		provName = t.ag.ProviderName()
		modelName = t.ag.ModelName()
	}

	width := getTerminalWidth()
	height := getTerminalHeight()

	logo := fmt.Sprintf(" OpenCola v%s ", version)
	rest := fmt.Sprintf(" Provider: %s  Model: %s  Status: %s ", provName, modelName, status)

	bar := frame + logo + rest
	if len(bar) > width {
		bar = bar[:width]
	}

	padding := strings.Repeat(" ", width-len(bar))
	bar += padding

	frameLen := len(frame)
	logoLen := len(logo)

	fmt.Printf("\033[%d;1H", height)
	fmt.Print("\033[2K")

	fmt.Printf("\033[48;2;30;64;120m\033[38;2;255;200;50m%s\033[0m", bar[:frameLen])

	fmt.Printf("\033[48;2;255;255;255m\033[38;2;30;64;120m%s\033[0m", bar[frameLen:frameLen+logoLen])
	fmt.Printf("\033[48;2;30;64;120m\033[38;2;255;255;255m%s\033[0m", bar[frameLen+logoLen:])
}

func (t *TUI) renderEventLog(message string) {
	ts := time.Now().Format("15:04:05")
	height := getTerminalHeight()
	fmt.Printf("\033[%d;1H\033[2K", height-2)
	fmt.Printf("[%s] %s\n", ts, message)
}

func (t *TUI) handleCommand(input string) bool {
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

		t.ag.SetProvider(prov)
		t.cfg.AddProvider(config.ProviderConfig{
			Name:     prov.Name(),
			Provider: provType,
			APIKey:   apiKey,
			BaseURL:  baseURL,
		})
		t.cfg.Save(t.cfgPath)

		t.envCfg.APIKey = apiKey
		t.envCfg.BaseURL = baseURL
		t.envCfg.Save(t.envPath)

		fmt.Printf("Connected to %s\n", prov.Name())

	case "/models":
		if !t.ag.IsConnected() {
			fmt.Println("Not connected to any provider. Use /connect first.")
			return false
		}

		models, err := t.ag.ListModels(t.ctx)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			return false
		}

		selected := showModelMenu(models)
		if selected != "" {
			t.ag.SetModel(selected)
			fmt.Printf("Selected model: %s\n", selected)
		}

	case "/reset":
		t.ag.Reset()
		fmt.Println("Session reset")

	case "/clear":
		t.renderInitialLayout()

	case "/status":
		t.renderStatusBar()

	case "/spin":
		t.toggleSpinner()

	case "/exit", "/quit":
		return true

	default:
		fmt.Printf("Unknown command: %s. Type /help for available commands.\n", cmd)
	}

	return false
}

func (t *TUI) startSpinner() {
	go func() {
		ticker := time.NewTicker(50 * time.Millisecond)
		defer ticker.Stop()

		for {
			select {
			case <-t.spinnerDone:
				return
			case <-ticker.C:
				t.spinnerMu.Lock()
				if t.spinning {
					first := spinnerBuffer[0]
					copy(spinnerBuffer[:len(spinnerBuffer)-1], spinnerBuffer[1:])
					spinnerBuffer[len(spinnerBuffer)-1] = first
				}
				t.spinnerMu.Unlock()
				if t.spinning {
					t.renderStatusBar()
				}
			}
		}
	}()
}

func (t *TUI) stopSpinner() {
	select {
	case <-t.spinnerDone:
	default:
		close(t.spinnerDone)
	}
}

func (t *TUI) toggleSpinner() {
	t.spinnerMu.Lock()
	defer t.spinnerMu.Unlock()

	t.spinning = !t.spinning
	spinnerBuffer = []byte(spinnerSeed)
}

func (t *TUI) printGoodbye() {
	t.stopSpinner()
	fmt.Print("\033[2J\033[H")
	fmt.Println()
	fmt.Println("Goodbye! Thanks for using OpenCola. See you next time!")
	fmt.Println()
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
	offset := 0

	fd := int(os.Stdin.Fd())
	state, _ := term.MakeRaw(fd)
	defer term.Restore(fd, state)

	reader := bufio.NewReader(os.Stdin)

	renderMenu := func() {
		fmt.Println()
		for i := 0; i < maxShow; i++ {
			idx := offset + i
			if idx >= len(models) {
				break
			}
			cursor := "  "
			if idx == selected {
				cursor = "> "
			}
			fmt.Printf("%s %s\n", cursor, models[idx].ID)
		}
	}

	clearMenu := func() {
		fmt.Printf("\033[%dA", maxShow+1)
		for i := 0; i < maxShow+1; i++ {
			fmt.Print("\033[2K\033[G")
		}
	}

	renderMenu()

	for {
		b, _, _ := reader.ReadRune()
		switch b {
		case 'A':
			if selected > 0 {
				selected--
				if selected < offset {
					offset = selected
				}
			}
			clearMenu()
			renderMenu()

		case 'B':
			if selected < len(models)-1 {
				selected++
				if selected >= offset+maxShow {
					offset = selected - maxShow + 1
				}
			}
			clearMenu()
			renderMenu()

		case '\r', '\n':
			clearMenu()
			fmt.Print("\033[1A\033[2K\033[G")
			return models[selected].ID
		}
	}
}

func newProvider(name, apiKey, model, baseURL string) provider.Provider {
	return provider.NewOpenAI(name, apiKey, model, baseURL)
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
