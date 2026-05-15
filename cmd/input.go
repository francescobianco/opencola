package cmd

import (
	"bufio"
	"fmt"
	"os"
	"slices"
	"strings"

	"github.com/francescobianco/opencola/provider"
	"golang.org/x/term"
)

var slashCommands = []string{
	"/connect",
	"/model",
	"/models",
	"/reset",
	"/clear",
	"/status",
	"/help",
	"/exit",
	"/quit",
	"/spin",
}

var plainCommands = []string{
	"clear",
	"quit",
	"exit",
}

type InputReader struct {
	history           []string
	historyIndex      int
	buffer            []rune
	cursorPos         int
	promptRow         int
	renderedRows      int
	renderedModelRows int
	originalState     *term.State
	modelOptions      []provider.ModelInfo
	currentModel      string
	modelMenuActive   bool
	modelMenuSelected int
	modelMenuOffset   int
	modelMenuMaxRows  int
}

func NewInputReader() *InputReader {
	return &InputReader{
		history:          make([]string, 0),
		historyIndex:     -1,
		buffer:           make([]rune, 0),
		cursorPos:        0,
		promptRow:        0,
		renderedRows:     1,
		modelMenuMaxRows: 6,
	}
}

func (r *InputReader) SetPromptRow(row int) {
	r.promptRow = row
}

func (r *InputReader) SetModelOptions(models []provider.ModelInfo, current string) {
	r.modelOptions = append(r.modelOptions[:0], models...)
	r.currentModel = current
}

func (r *InputReader) LoadHistory(path string) {
	f, err := os.Open(path)
	if err != nil {
		return
	}
	defer f.Close()

	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		r.history = append(r.history, scanner.Text())
	}
	r.historyIndex = len(r.history)
}

func (r *InputReader) SaveHistory(path string) {
	dir := path[:lastIndex(path, "/")]
	if dir != "" {
		os.MkdirAll(dir, 0755)
	}

	f, err := os.Create(path)
	if err != nil {
		return
	}
	defer f.Close()

	maxHistory := 1000
	start := 0
	if len(r.history) > maxHistory {
		start = len(r.history) - maxHistory
	}

	for _, line := range r.history[start:] {
		fmt.Fprintln(f, line)
	}
}

func lastIndex(s, sep string) int {
	for i := len(s) - 1; i >= 0; i-- {
		if s[i:i+1] == sep {
			return i
		}
	}
	return -1
}

func (r *InputReader) ReadLine() (string, error) {
	fd := int(os.Stdin.Fd())
	state, err := term.MakeRaw(fd)
	if err != nil {
		return "", err
	}
	r.originalState = state
	defer term.Restore(fd, state)

	// Enable kitty keyboard protocol for shift+enter detection
	fmt.Print("\x1b[>1u")
	defer fmt.Print("\x1b[<1u")

	r.buffer = make([]rune, 0)
	r.cursorPos = 0
	r.historyIndex = len(r.history)
	r.modelMenuActive = false
	r.modelMenuOffset = 0

	r.renderLine()

	reader := bufio.NewReader(os.Stdin)

	for {
		b, _, err := reader.ReadRune()
		if err != nil {
			return string(r.buffer), err
		}

		switch b {
		case '\r':
			if r.modelMenuActive {
				r.acceptModelSelection()
			}
			line := string(r.buffer)
			if line != "" {
				r.history = append(r.history, line)
			}
			r.historyIndex = len(r.history)
			fmt.Print("\r\n")
			return line, nil

		case '\n':
			r.insertRune('\n')

		case 3:
			return "", fmt.Errorf("interrupt")

		case 127, '\b':
			if r.cursorPos > 0 {
				r.buffer = append(r.buffer[:r.cursorPos-1], r.buffer[r.cursorPos:]...)
				r.cursorPos--
				r.updateModelMenuState()
				r.renderLine()
			}

		case 1:
			r.cursorPos = 0
			r.updateModelMenuState()
			r.renderLine()

		case 5:
			r.cursorPos = len(r.buffer)
			r.updateModelMenuState()
			r.renderLine()

		case '\t':
			r.autocomplete()

		case 27:
			b2, _, _ := reader.ReadRune()
			if b2 == '[' {
				b3, _, _ := reader.ReadRune()
				switch b3 {
				case 'A':
					if r.modelMenuActive {
						r.moveModelSelection(-1)
						break
					}
					if r.historyIndex > 0 {
						r.historyIndex--
						r.buffer = []rune(r.history[r.historyIndex])
						r.cursorPos = len(r.buffer)
						r.updateModelMenuState()
						r.renderLine()
					}

				case 'B':
					if r.modelMenuActive {
						r.moveModelSelection(1)
						break
					}
					if r.historyIndex < len(r.history)-1 {
						r.historyIndex++
						r.buffer = []rune(r.history[r.historyIndex])
						r.cursorPos = len(r.buffer)
						r.updateModelMenuState()
						r.renderLine()
					} else {
						r.historyIndex = len(r.history)
						r.buffer = make([]rune, 0)
						r.cursorPos = 0
						r.updateModelMenuState()
						r.renderLine()
					}

				case 'C':
					if r.cursorPos < len(r.buffer) {
						r.cursorPos++
						r.updateModelMenuState()
						r.renderLine()
					}

				case 'D':
					if r.cursorPos > 0 {
						r.cursorPos--
						r.updateModelMenuState()
						r.renderLine()
					}

				case '3':
					reader.ReadRune()
					if r.cursorPos < len(r.buffer) {
						r.buffer = append(r.buffer[:r.cursorPos], r.buffer[r.cursorPos+1:]...)
						r.updateModelMenuState()
						r.renderLine()
					}

				case '1':
					seq := []rune{b3}
					for {
						next, _, err := reader.ReadRune()
						if err != nil {
							break
						}
						seq = append(seq, next)
						if (next >= '@' && next <= '~') || len(seq) > 12 {
							break
						}
					}
					if string(seq) == "13;2u" || string(seq) == "13;2~" {
						r.insertRune('\n')
					}

				case '2':
					seq := []rune{b3}
					for {
						next, _, err := reader.ReadRune()
						if err != nil {
							break
						}
						seq = append(seq, next)
						if (next >= '@' && next <= '~') || len(seq) > 12 {
							break
						}
					}
					if string(seq) == "7;2;13~" {
						r.insertRune('\n')
					}
				}
			}

		default:
			if b >= 32 {
				r.insertRune(b)
			}
		}
	}
}

func (r *InputReader) insertRune(ch rune) {
	r.buffer = append(r.buffer, 0)
	copy(r.buffer[r.cursorPos+1:], r.buffer[r.cursorPos:])
	r.buffer[r.cursorPos] = ch
	r.cursorPos++
	r.updateModelMenuState()
	r.renderLine()
}

func (r *InputReader) setBuffer(value string) {
	r.buffer = []rune(value)
	r.cursorPos = len(r.buffer)
	r.updateModelMenuState()
	r.renderLine()
}

func (r *InputReader) updateModelMenuState() {
	input := string(r.buffer)
	fields := strings.Fields(input)
	isModelCommand := input == "/model" || input == "/models" ||
		strings.HasPrefix(input, "/model ") || strings.HasPrefix(input, "/models ")

	if !isModelCommand || len(r.modelOptions) == 0 {
		r.modelMenuActive = false
		r.modelMenuOffset = 0
		return
	}

	selectedSlug := ""
	if len(fields) >= 2 {
		selectedSlug = fields[1]
	} else if r.currentModel != "" && r.currentModel != "none" {
		selectedSlug = r.currentModel
	}

	selected := r.findModelIndex(selectedSlug)
	if selected < 0 {
		selected = 0
	}
	r.modelMenuActive = true
	r.modelMenuSelected = selected
	r.ensureModelSelectionVisible()

	if len(fields) < 2 && selectedSlug != "" && input != "/model" {
		cmd := fields[0]
		if cmd == "" {
			cmd = "/model"
		}
		r.buffer = []rune(cmd + " " + selectedSlug)
		r.cursorPos = len(r.buffer)
	}
}

func (r *InputReader) findModelIndex(slug string) int {
	if slug == "" {
		return -1
	}
	for i, model := range r.modelOptions {
		if model.ID == slug {
			return i
		}
	}
	return -1
}

func (r *InputReader) ensureModelSelectionVisible() {
	maxRows := r.modelMenuVisibleRows()
	if r.modelMenuSelected < r.modelMenuOffset {
		r.modelMenuOffset = r.modelMenuSelected
	}
	if r.modelMenuSelected >= r.modelMenuOffset+maxRows {
		r.modelMenuOffset = r.modelMenuSelected - maxRows + 1
	}
	if r.modelMenuOffset < 0 {
		r.modelMenuOffset = 0
	}
}

func (r *InputReader) modelMenuVisibleRows() int {
	rows := r.modelMenuMaxRows
	if rows <= 0 {
		rows = 6
	}
	if len(r.modelOptions) < rows {
		rows = len(r.modelOptions)
	}
	return rows
}

func (r *InputReader) moveModelSelection(delta int) {
	if len(r.modelOptions) == 0 {
		return
	}
	r.modelMenuSelected += delta
	if r.modelMenuSelected < 0 {
		r.modelMenuSelected = 0
	}
	if r.modelMenuSelected >= len(r.modelOptions) {
		r.modelMenuSelected = len(r.modelOptions) - 1
	}
	r.ensureModelSelectionVisible()
	r.fillSelectedModel()
	r.renderLine()
}

func (r *InputReader) acceptModelSelection() {
	if len(r.modelOptions) == 0 {
		return
	}
	r.fillSelectedModel()
	r.modelMenuActive = false
}

func (r *InputReader) fillSelectedModel() {
	if r.modelMenuSelected < 0 || r.modelMenuSelected >= len(r.modelOptions) {
		return
	}
	cmd := "/model"
	fields := strings.Fields(string(r.buffer))
	if len(fields) > 0 && fields[0] == "/models" {
		cmd = "/models"
	}
	value := cmd + " " + r.modelOptions[r.modelMenuSelected].ID
	r.buffer = []rune(value)
	r.cursorPos = len(r.buffer)
}

func (r *InputReader) autocomplete() {
	input := string(r.buffer[:r.cursorPos])
	if !strings.HasPrefix(input, "/") {
		for _, cmd := range plainCommands {
			if strings.HasPrefix(cmd, strings.ToLower(input)) {
				r.setBuffer(cmd + " ")
				return
			}
		}
		return
	}

	if strings.HasPrefix(input, "/connect ") {
		provInput := strings.TrimPrefix(input, "/connect ")
		var matches []string
		for _, slug := range providerSlugs() {
			if strings.HasPrefix(slug, provInput) {
				matches = append(matches, "/connect "+slug)
			}
		}

		if len(matches) == 1 {
			r.setBuffer(matches[0] + " ")
		} else if len(matches) > 1 {
			fmt.Println()
			for _, m := range matches {
				fmt.Printf("  %s\n", m)
			}
			r.renderLine()
		}
		return
	}

	var matches []string
	for _, cmd := range slashCommands {
		if strings.HasPrefix(cmd, input) {
			matches = append(matches, cmd)
		}
	}

	if input == "/mode" {
		r.setBuffer("/model ")
		return
	}

	if len(matches) == 1 {
		r.setBuffer(matches[0] + " ")
	} else if len(matches) > 1 {
		fmt.Println()
		for _, m := range matches {
			fmt.Printf("  %s\n", m)
		}
		r.renderLine()
	}
}

func (r *InputReader) visualRowsForLine(textLen, termWidth int) int {
	if textLen == 0 {
		return 1
	}
	available := termWidth - 2
	if available < 1 {
		available = 1
	}
	rows := (textLen + available - 1) / available
	if rows < 1 {
		rows = 1
	}
	return rows
}

func (r *InputReader) renderLine() {
	terminalMu.Lock()
	defer terminalMu.Unlock()

	height := getTerminalHeight()
	termWidth := getTerminalWidth()
	inputRow := r.promptRow
	if inputRow <= 0 || inputRow >= height {
		inputRow = height - 1
	}
	if inputRow < 1 {
		inputRow = 1
	}

	lines := strings.Split(string(r.buffer), "\n")

	// Calcola il numero totale di righe visuali (considerando il wrapping)
	totalVisualRows := 0
	visualRowsPerLine := make([]int, len(lines))
	for i, line := range lines {
		textLen := len([]rune(line))
		visualRowsPerLine[i] = r.visualRowsForLine(textLen, termWidth)
		totalVisualRows += visualRowsPerLine[i]
	}

	clearRows := r.renderedRows
	if totalVisualRows > clearRows {
		clearRows = totalVisualRows
	}
	menuRows := 0
	if r.modelMenuActive {
		menuRows = r.modelMenuVisibleRows()
	}
	clearMenuRows := r.renderedModelRows
	if menuRows > clearMenuRows {
		clearMenuRows = menuRows
	}
	startRow := inputRow - clearRows - clearMenuRows + 1
	if startRow < 1 {
		startRow = 1
	}
	for row := startRow; row <= inputRow; row++ {
		fmt.Printf("\033[%d;1H\033[2K", row)
	}

	// Determina quali linee mostrare se non entrano tutte nello schermo
	firstLine := 0
	visualRowsShown := 0
	if totalVisualRows > inputRow {
		remaining := inputRow
		for i := len(lines) - 1; i >= 0; i-- {
			if visualRowsPerLine[i] <= remaining {
				remaining -= visualRowsPerLine[i]
				firstLine = i
			} else {
				firstLine = i
				break
			}
		}
		for i := firstLine; i < len(lines); i++ {
			visualRowsShown += visualRowsPerLine[i]
		}
	} else {
		visualRowsShown = totalVisualRows
	}

	drawStartRow := inputRow - visualRowsShown + 1
	if drawStartRow < 1 {
		drawStartRow = 1
	}

	r.renderModelMenu(drawStartRow, termWidth)

	// Stampa con wrapping manuale e indentazione costante
	currentRow := drawStartRow
	for i := firstLine; i < len(lines); i++ {
		prefix := "  "
		if i == 0 {
			prefix = "> "
		}

		runes := []rune(lines[i])
		available := termWidth - 2
		if available < 1 {
			available = 1
		}

		if len(runes) == 0 {
			fmt.Printf("\033[%d;1H%s", currentRow, prefix)
			currentRow++
			continue
		}

		for len(runes) > 0 {
			chunkLen := available
			if len(runes) < chunkLen {
				chunkLen = len(runes)
			}
			fmt.Printf("\033[%d;1H%s%s", currentRow, prefix, string(runes[:chunkLen]))
			runes = runes[chunkLen:]
			currentRow++
			// Per le righe successive dello stesso blocco, mantieni l'indentazione
			prefix = "  "
		}
	}

	cursorRow, cursorCol := r.cursorPosition(lines, firstLine, drawStartRow, termWidth)
	fmt.Printf("\033[%d;%dH", cursorRow, cursorCol)
	r.renderedRows = totalVisualRows
	r.renderedModelRows = menuRows
}

func (r *InputReader) renderModelMenu(inputStartRow int, termWidth int) {
	if !r.modelMenuActive || len(r.modelOptions) == 0 {
		return
	}

	rows := r.modelMenuVisibleRows()
	startRow := inputStartRow - rows
	if startRow < 1 {
		startRow = 1
	}

	for i := 0; i < rows; i++ {
		idx := r.modelMenuOffset + i
		if idx >= len(r.modelOptions) {
			break
		}
		model := r.modelOptions[idx]
		label := model.Name
		if label == "" {
			label = model.ID
		}

		cursor := "  "
		if idx == r.modelMenuSelected {
			cursor = "> "
		}

		line := fmt.Sprintf("%s%s", cursor, label)
		if label != model.ID {
			line = fmt.Sprintf("%s%s (%s)", cursor, label, model.ID)
		}
		if model.ID == r.currentModel {
			line += " *"
		}
		if len([]rune(line)) > termWidth {
			line = string([]rune(line)[:termWidth])
		}
		fmt.Printf("\033[%d;1H\033[2K%s", startRow+i, line)
	}
}

func (r *InputReader) cursorPosition(lines []string, firstLine int, drawStartRow int, termWidth int) (int, int) {
	lineIndex := 0
	colInLine := 0
	for i, ch := range r.buffer {
		if i == r.cursorPos {
			break
		}
		if ch == '\n' {
			lineIndex++
			colInLine = 0
		} else {
			colInLine++
		}
	}

	if r.cursorPos == len(r.buffer) {
		lineIndex = len(lines) - 1
		if lineIndex < 0 {
			lineIndex = 0
		}
		colInLine = len([]rune(lines[lineIndex]))
	}

	if lineIndex < firstLine {
		lineIndex = firstLine
		colInLine = 0
	}

	// Calcola la riga visuale sommando le righe delle linee precedenti
	row := drawStartRow
	for i := firstLine; i < lineIndex; i++ {
		textLen := len([]rune(lines[i]))
		row += r.visualRowsForLine(textLen, termWidth)
	}

	// Calcola l'offset nella linea corrente
	if lineIndex >= len(lines) {
		lineIndex = len(lines) - 1
		if lineIndex < 0 {
			lineIndex = 0
		}
	}
	textLen := len([]rune(lines[lineIndex]))
	if colInLine > textLen {
		colInLine = textLen
	}

	available := termWidth - 2
	if available < 1 {
		available = 1
	}

	visualRowInLine := 0
	if available > 0 {
		visualRowInLine = colInLine / available
	}
	visualCol := colInLine % available

	row += visualRowInLine
	col := 2 + visualCol + 1 // prefix (2) + col (0-based) + 1 (1-based)

	return row, col
}

func (r *InputReader) IsVimQuit(input string) bool {
	return slices.Contains([]string{":q", ":q!", ":wq", ":wq!"}, strings.TrimSpace(input))
}
