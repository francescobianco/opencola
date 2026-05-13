package cmd

import (
	"bufio"
	"context"
	"fmt"
	"os"
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

type providerProfile struct {
	Slug         string
	Label        string
	BaseURL      string
	DefaultModel string
	Models       []provider.ModelInfo
}

var providerProfiles = []providerProfile{
	{
		Slug:         "opencode",
		Label:        "OpenCode",
		BaseURL:      "https://opencode.ai/zen/v1",
		DefaultModel: "qwen3.6-plus",
		Models:       openCodeZenChatModels(),
	},
	{
		Slug:         "opencode-go",
		Label:        "OpenCode Go",
		BaseURL:      "https://opencode.ai/zen/go/v1",
		DefaultModel: "kimi-k2.6",
		Models:       openCodeGoChatModels(),
	},
	{
		Slug:         "opencode-zen",
		Label:        "OpenCode Zen",
		BaseURL:      "https://opencode.ai/zen/v1",
		DefaultModel: "qwen3.6-plus",
		Models:       openCodeZenChatModels(),
	},
}

const spinnerSeed = "...|...||....|.|...||......"

var spinnerBuffer = []byte(spinnerSeed)
var terminalMu sync.Mutex

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
		profile := mustProviderProfile("opencode")
		cfg.AddProvider(config.ProviderConfig{
			Name:     profile.Label,
			Provider: profile.Slug,
			APIKey:   envCfg.APIKey,
			BaseURL:  profile.BaseURL,
			Model:    firstNonEmpty(envCfg.Model, profile.DefaultModel),
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
		prov := newProvider(p.Provider, p.APIKey, p.Model, normalizeProviderBaseURL(p.Provider, p.BaseURL))
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
		spinning:    false,
		spinnerDone: make(chan struct{}),
	}

	tui.input.LoadHistory(config.DefaultHistoryPath())
	defer tui.input.SaveHistory(config.DefaultHistoryPath())

	tui.renderInitialLayout(true)
	tui.startSpinner()
	defer tui.stopSpinner()

	for {
		tui.renderPrompt()
		line, err := tui.input.ReadLine()
		if err != nil {
			tui.printGoodbye()
			break
		}
		tui.input.SetPromptRow(tui.contentBottomRow())

		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}

		if tui.input.IsVimQuit(line) {
			tui.printGoodbye()
			break
		}

		if strings.ToLower(line) == "quit" || strings.ToLower(line) == "exit" {
			tui.printGoodbye()
			break
		}

		if strings.HasPrefix(line, "/") {
			if tui.handleCommand(line) {
				tui.printGoodbye()
				break
			}
			tui.renderStatusBar()
			continue
		}

		if strings.ToLower(line) == "clear" {
			tui.renderInitialLayout(false)
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

func (t *TUI) renderInitialLayout(showBanner bool) {
	height := getTerminalHeight()

	fmt.Print("\033[2J\033[H")
	t.reserveStatusLine()

	if showBanner {
		freeLines := height - 8
		if freeLines < 0 {
			freeLines = 0
		}

		for i := 0; i < freeLines; i++ {
			fmt.Println()
		}

		fmt.Printf("\033[1;97mOpenCola\033[0m small coding agent\n")
		fmt.Println(author)
		fmt.Println("type /help for a list of commands")
		fmt.Println()
	}

	t.input.SetPromptRow(t.initialPromptRow())
	fmt.Printf("\033[%d;1H", t.initialPromptRow())
	fmt.Print("\033[2K")
	fmt.Print("> ")
	fmt.Printf("\033[%d;1H\033[2K", t.contentBottomRow())

	t.renderStatusBar()
}

func (t *TUI) renderPrompt() {
	row := t.input.promptRow
	if row <= 0 {
		row = t.contentBottomRow()
		t.input.SetPromptRow(row)
	}
	terminalMu.Lock()
	defer terminalMu.Unlock()
	t.reserveStatusLine()
	fmt.Printf("\033[%d;1H\033[2K> ", row)
}

func (t *TUI) contentBottomRow() int {
	height := getTerminalHeight()
	if height < 2 {
		return 1
	}
	return height - 1
}

func (t *TUI) initialPromptRow() int {
	row := getTerminalHeight() - 2
	if row < 1 {
		return 1
	}
	return row
}

func (t *TUI) reserveStatusLine() {
	height := getTerminalHeight()
	if height > 1 {
		fmt.Printf("\033[1;%dr", height-1)
	}
}

func (t *TUI) renderStatusBar() {
	t.spinnerMu.Lock()
	spinning := t.spinning
	frame := ""
	if spinning {
		frame = string(spinnerBuffer[:3])
	}
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
	if spinning {
		rest += frame
	}

	bar := logo + rest
	if len(bar) > width {
		bar = bar[:width]
	}

	padding := strings.Repeat(" ", width-len(bar))
	bar += padding

	logoLen := len(logo)
	if logoLen > len(bar) {
		logoLen = len(bar)
	}

	terminalMu.Lock()
	defer terminalMu.Unlock()

	fmt.Print("\033[?25l")
	fmt.Print("\033[s")
	fmt.Print("\033[r")
	fmt.Printf("\033[%d;1H", height)
	fmt.Print("\033[2K")

	fmt.Printf("\033[1;97m\033[48;2;0;70;180m%s\033[0m", bar[:logoLen])
	fmt.Printf("\033[48;2;45;55;65m\033[38;2;190;200;210m%s\033[0m", bar[logoLen:])
	t.reserveStatusLine()
	fmt.Print("\033[u")
	fmt.Print("\033[?25h")
}

func (t *TUI) renderEventLog(message string) {
	ts := time.Now().Format("15:04:05")
	terminalMu.Lock()
	fmt.Printf("\033[%d;1H\033[2K", t.contentBottomRow())
	fmt.Printf("[%s] %s\r\n", ts, message)
	terminalMu.Unlock()
	t.renderStatusBar()
}

func (t *TUI) handleCommand(input string) bool {
	parts := strings.Fields(input)
	cmd := strings.ToLower(parts[0])

	switch cmd {
	case "/help":
		fmt.Println("Available commands:")
		fmt.Println("  /connect <provider>        Connect to a provider")
		fmt.Println("  /models                    Select a model")
		fmt.Println("  /reset                     Reset conversation")
		fmt.Println("  /clear                     Clear the screen")
		fmt.Println("  /status                    Show current status")
		fmt.Println("  /exit, /quit, :q           Exit the program")

	case "/connect":
		if len(parts) < 2 {
			fmt.Println("Usage: /connect <provider>")
			fmt.Printf("  providers: %s\n", strings.Join(providerLabels(), ", "))
			return false
		}

		provType := parts[1]
		profile, ok := findProviderProfile(provType)
		if !ok {
			fmt.Printf("Unknown provider: %s\n", provType)
			fmt.Printf("Available: %s\n", strings.Join(providerLabels(), ", "))
			return false
		}

		apiKey, err := readAPIKey(profile.Label)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error reading API key: %v\n", err)
			return false
		}
		apiKey = strings.TrimSpace(apiKey)

		if apiKey == "" {
			fmt.Println("API key is required")
			return false
		}

		baseURL := profile.BaseURL
		prov := newProvider(profile.Slug, apiKey, profile.DefaultModel, baseURL)

		t.ag.SetProvider(prov)
		t.cfg.AddProvider(config.ProviderConfig{
			Name:     prov.Name(),
			Provider: profile.Slug,
			APIKey:   apiKey,
			BaseURL:  baseURL,
			Model:    profile.DefaultModel,
		})
		t.cfg.Save(t.cfgPath)

		t.envCfg.APIKey = apiKey
		t.envCfg.BaseURL = baseURL
		t.envCfg.Save(t.envPath)

		fmt.Printf("Connected to %s\n", profile.Label)

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
		t.renderInitialLayout(false)

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

func readAPIKey(providerName string) (string, error) {
	terminalMu.Lock()
	defer terminalMu.Unlock()

	fmt.Printf("Please enter your API key for %s: ", providerName)

	fd := int(os.Stdin.Fd())
	state, err := term.MakeRaw(fd)
	if err != nil {
		fmt.Println()
		return "", err
	}
	defer term.Restore(fd, state)

	var input strings.Builder
	reader := bufio.NewReader(os.Stdin)

	for {
		ch, _, err := reader.ReadRune()
		if err != nil {
			fmt.Println()
			return input.String(), err
		}

		switch ch {
		case '\r', '\n':
			fmt.Print("\r\n")
			return input.String(), nil
		case 3:
			fmt.Print("\r\n")
			return "", fmt.Errorf("interrupt")
		case 127, '\b':
			if input.Len() > 0 {
				value := []rune(input.String())
				value = value[:len(value)-1]
				input.Reset()
				input.WriteString(string(value))
			}
		default:
			if ch >= 32 {
				input.WriteRune(ch)
			}
		}
	}
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
				spinning := t.spinning
				t.spinnerMu.Unlock()
				if spinning {
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
	fmt.Print("\033[r")
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
			name := models[idx].Name
			if name == "" {
				name = models[idx].ID
			}
			if name == models[idx].ID {
				fmt.Printf("%s %s\n", cursor, models[idx].ID)
			} else {
				fmt.Printf("%s %s (%s)\n", cursor, name, models[idx].ID)
			}
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

func newProvider(slug, apiKey, model, baseURL string) provider.Provider {
	profile, ok := findProviderProfile(slug)
	if !ok {
		return provider.NewOpenAI(slug, apiKey, model, baseURL)
	}
	if model == "" {
		model = profile.DefaultModel
	}
	return provider.NewOpenAI(profile.Label, apiKey, model, normalizeProviderBaseURL(slug, baseURL), profile.Models)
}

func normalizeProviderBaseURL(slug string, baseURL string) string {
	profile, ok := findProviderProfile(slug)
	if !ok {
		return firstNonEmpty(baseURL, "https://api.openai.com/v1")
	}
	return profile.BaseURL
}

func findProviderProfile(slug string) (providerProfile, bool) {
	for _, profile := range providerProfiles {
		if profile.Slug == slug {
			return profile, true
		}
	}
	return providerProfile{}, false
}

func mustProviderProfile(slug string) providerProfile {
	profile, ok := findProviderProfile(slug)
	if !ok {
		panic("unknown provider profile: " + slug)
	}
	return profile
}

func providerSlugs() []string {
	slugs := make([]string, 0, len(providerProfiles))
	for _, profile := range providerProfiles {
		slugs = append(slugs, profile.Slug)
	}
	return slugs
}

func providerLabels() []string {
	labels := make([]string, 0, len(providerProfiles))
	for _, profile := range providerProfiles {
		labels = append(labels, fmt.Sprintf("%s (%s)", profile.Label, profile.Slug))
	}
	return labels
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return value
		}
	}
	return ""
}

func openCodeZenChatModels() []provider.ModelInfo {
	return []provider.ModelInfo{
		{ID: "qwen3.6-plus", Name: "Qwen3.6 Plus"},
		{ID: "qwen3.5-plus", Name: "Qwen3.5 Plus"},
		{ID: "minimax-m2.7", Name: "MiniMax M2.7"},
		{ID: "minimax-m2.5", Name: "MiniMax M2.5"},
		{ID: "minimax-m2.5-free", Name: "MiniMax M2.5 Free"},
		{ID: "glm-5.1", Name: "GLM 5.1"},
		{ID: "glm-5", Name: "GLM 5"},
		{ID: "kimi-k2.5", Name: "Kimi K2.5"},
		{ID: "kimi-k2.6", Name: "Kimi K2.6"},
		{ID: "big-pickle", Name: "Big Pickle"},
		{ID: "deepseek-v4-flash-free", Name: "DeepSeek V4 Flash Free"},
		{ID: "ring-2.6-1t-free", Name: "Ring 2.6 1T Free"},
		{ID: "nemotron-3-super-free", Name: "Nemotron 3 Super Free"},
	}
}

func openCodeGoChatModels() []provider.ModelInfo {
	return []provider.ModelInfo{
		{ID: "glm-5.1", Name: "GLM 5.1"},
		{ID: "glm-5", Name: "GLM 5"},
		{ID: "kimi-k2.5", Name: "Kimi K2.5"},
		{ID: "kimi-k2.6", Name: "Kimi K2.6"},
		{ID: "deepseek-v4-pro", Name: "DeepSeek V4 Pro"},
		{ID: "deepseek-v4-flash", Name: "DeepSeek V4 Flash"},
		{ID: "mimo-v2.5", Name: "MiMo-V2.5"},
		{ID: "mimo-v2.5-pro", Name: "MiMo-V2.5-Pro"},
		{ID: "qwen3.6-plus", Name: "Qwen3.6 Plus"},
		{ID: "qwen3.5-plus", Name: "Qwen3.5 Plus"},
	}
}
