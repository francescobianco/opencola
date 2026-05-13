package cmd

import (
	"bufio"
	"fmt"
	"os"
	"slices"
	"strings"

	"golang.org/x/term"
)

var slashCommands = []string{
	"/connect",
	"/models",
	"/reset",
	"/clear",
	"/status",
	"/help",
	"/exit",
	"/quit",
}

type InputReader struct {
	history       []string
	historyIndex  int
	buffer        []rune
	cursorPos     int
	originalState *term.State
	Spinning      bool
}

func NewInputReader() *InputReader {
	return &InputReader{
		history:      make([]string, 0),
		historyIndex: -1,
		buffer:       make([]rune, 0),
		cursorPos:    0,
	}
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

	r.buffer = make([]rune, 0)
	r.cursorPos = 0
	r.historyIndex = len(r.history)

	r.renderLine()

	reader := bufio.NewReader(os.Stdin)

	for {
		b, _, err := reader.ReadRune()
		if err != nil {
			return string(r.buffer), err
		}

		switch b {
		case '\r', '\n':
			line := string(r.buffer)
			if line != "" {
				r.history = append(r.history, line)
			}
			r.historyIndex = len(r.history)
			return line, nil

		case 3:
			return "", fmt.Errorf("interrupt")

		case 127, '\b':
			if r.cursorPos > 0 {
				r.buffer = append(r.buffer[:r.cursorPos-1], r.buffer[r.cursorPos:]...)
				r.cursorPos--
				r.renderLine()
			}

		case 1:
			r.cursorPos = 0
			r.renderLine()

		case 5:
			r.cursorPos = len(r.buffer)
			r.renderLine()

		case '\t':
			r.autocomplete()

		case 27:
			b2, _, _ := reader.ReadRune()
			if b2 == '[' {
				b3, _, _ := reader.ReadRune()
				switch b3 {
				case 'A':
					if r.historyIndex > 0 {
						r.historyIndex--
						r.buffer = []rune(r.history[r.historyIndex])
						r.cursorPos = len(r.buffer)
						r.renderLine()
					}

				case 'B':
					if r.historyIndex < len(r.history)-1 {
						r.historyIndex++
						r.buffer = []rune(r.history[r.historyIndex])
						r.cursorPos = len(r.buffer)
						r.renderLine()
					} else {
						r.historyIndex = len(r.history)
						r.buffer = make([]rune, 0)
						r.cursorPos = 0
						r.renderLine()
					}

				case 'C':
					if r.cursorPos < len(r.buffer) {
						r.cursorPos++
						r.renderLine()
					}

				case 'D':
					if r.cursorPos > 0 {
						r.cursorPos--
						r.renderLine()
					}

				case '3':
					reader.ReadRune()
					if r.cursorPos < len(r.buffer) {
						r.buffer = append(r.buffer[:r.cursorPos], r.buffer[r.cursorPos+1:]...)
						r.renderLine()
					}
				}
			}

		default:
			if b >= 32 {
				r.buffer = append(r.buffer, 0)
				copy(r.buffer[r.cursorPos+1:], r.buffer[r.cursorPos:])
				r.buffer[r.cursorPos] = b
				r.cursorPos++
				r.renderLine()
			}
		}
	}
}

func (r *InputReader) autocomplete() {
	input := string(r.buffer[:r.cursorPos])
	if !strings.HasPrefix(input, "/") {
		return
	}

	if strings.HasPrefix(input, "/connect ") {
		provInput := strings.TrimPrefix(input, "/connect ")
		var matches []string
		for _, p := range providers {
			if strings.HasPrefix(p, provInput) {
				matches = append(matches, "/connect "+p)
			}
		}

		if len(matches) == 1 {
			r.buffer = []rune(matches[0] + " ")
			r.cursorPos = len(r.buffer)
			r.renderLine()
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

	if len(matches) == 1 {
		r.buffer = []rune(matches[0] + " ")
		r.cursorPos = len(r.buffer)
		r.renderLine()
	} else if len(matches) > 1 {
		fmt.Println()
		for _, m := range matches {
			fmt.Printf("  %s\n", m)
		}
		r.renderLine()
	}
}

func (r *InputReader) renderLine() {
	height := getTerminalHeight()
	fmt.Printf("\033[%d;1H\033[2K", height-2)
	if r.Spinning {
		return
	}
	fmt.Print("> ")
	fmt.Print(string(r.buffer))
	if r.cursorPos < len(r.buffer) {
		fmt.Printf("\033[%dG", 2+r.cursorPos)
	}
}

func (r *InputReader) IsVimQuit(input string) bool {
	return slices.Contains([]string{":q", ":q!", ":wq", ":wq!"}, strings.TrimSpace(input))
}
