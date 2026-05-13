
        O P E N C O L A   v 0 . 1 . 0
        Minimal coding agent for terminal
             by Francesco Bianco

> /connect opencode
Please enter your API key for opencode:
> ****************************
Connected to opencode

> What is the capital of France?
According to my knowledge, the capital of France is Paris.
... OpenCola v0.1.0  Provider: opencode  Model: gpt-4o  Status: Connected

---

## Install

```bash
git clone https://github.com/francescobianco/opencola.git
cd opencola
make build
```

## Quick Start

```bash
./opencola
/connect opencode
/models
What is the capital of France?
/exit
```

## Commands

| Command | Description |
|---|---|
| `/connect <provider>` | Connect to an LLM provider |
| `/models` | Select a model from the connected provider |
| `/reset` | Clear conversation history |
| `/clear` | Clear the screen |
| `/status` | Redraw the status bar |
| `/exit`, `/quit`, `:q` | Exit the program |

## Features

- **TUI-first** — clean terminal interface with persistent status bar
- **Provider system** — opencode, opencode-go, opencode-zen
- **Prompt hiding** — the input line clears during processing, spinner signals activity
- **Tab completion** — commands and provider names autocomplete
- **Command history** — persists across sessions via `~/.config/opencola/history`
- **Vim easter egg** — `:q`, `:wq`, `:q!` all work
- **Lightweight** — pure Go, zero heavy TUI dependencies (only `golang.org/x/term`)

## Configuration

```
~/.config/opencola/config.json    — full JSON config with provider history
~/.opencolarc                     — simple .env format
```

## Project Structure

```
cmd/cli.go       — TUI layout, commands, spinner, status bar
cmd/input.go     — readline with history, tab completion, arrow keys
cmd/terminal.go  — terminal size via ioctl
agent/agent.go   — agent loop, tool orchestration
provider/        — Provider interface + OpenAI SDK implementation
config/          — JSON and .env config loading
session/         — conversation message history
tools/           — read_file, write_file, exec tools
tests/           — tmux-based TUI tests with reference captures
```

## License

MIT
