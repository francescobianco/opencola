# OpenCola Features

This document describes the core design principles and features of OpenCola. Each feature is motivated by a specific user need or design philosophy, and the document serves as both a specification and a reference for contributors.

## TUI Layout

**Rationale:** A clean, predictable terminal interface reduces cognitive load and makes the tool feel professional. The layout follows a minimal design inspired by modern CLI tools.

**Behavior:**
- On startup, the terminal is cleared before rendering
- Free space is left above the banner (variable, depends on terminal height — roughly one-third)
- Banner displays with **OpenCola** in bold, rest in normal weight:
  ```
  OpenCola - minimal coding agent
  by Francesco Bianco <bianco@javanile.org>

  >
  ```
- One blank line below the prompt before user input
- The status bar occupies the last line at all times
- On exit, the terminal is cleared and a goodbye message is shown:
  ```
  Goodbye! Thanks for using OpenCola. See you next time!
  ```

## Status Bar

**Rationale:** Users should always know the current state of the application without typing a command. A persistent status bar on the last terminal line provides at-a-glance context.

**Behavior:**
- Occupies the last line of the terminal at all times
- The logo portion (`OpenCola v0.1.0`) uses **inverted colors**: white background with dark blue text
- The rest of the bar uses dark blue background (`rgb(30,64,120)`) with white text
- Format: `[OpenCola v0.1.0] Provider: [name] Model: [model] Status: [Connected/Disconnected]`
- Automatically redraws after each command output
- The prompt `> ` is rendered on the line above the status bar
- Model shows the selected model name instead of "none" after selection

## Provider System

**Rationale:** Users should be able to connect to different LLM providers, including OpenCode's own variants. Provider selection should be discoverable and secure.

**Behavior:**
- Available providers: `opencode` (default), `opencode-go`, `opencode-zen`
- Provider names are autocompleted with Tab (e.g., `/connect opencode-[Tab]`)
- API key is **never** entered inline — a separate prompt appears after selecting the provider:
  ```
  > /connect opencode-go [Enter]
  Please enter your API key for opencode-go:
  ```
- Each provider has its own base URL configured automatically
- Side providers (`opencode-go`, `opencode-zen`) only show their own models

## Model Selection Menu

**Rationale:** Users need an intuitive way to choose from available models. A compact interactive menu is faster and more discoverable than typing model names.

**Behavior:**
- `/models` fetches available models from the connected provider
- Displays a compact menu (4 rows max) below the prompt
- Cursor `> ` appears to the left of the current selection
- Up/Down arrow keys navigate the list
- Enter confirms the selection
- Selected model appears in the status bar

## Command History & Navigation

**Rationale:** Developers expect arrow-key navigation through previous commands, just like in any standard shell.

**Behavior:**
- Up/Down arrow keys navigate through previously entered commands
- History persists across sessions via `~/.config/opencola/history`
- Lightweight readline implementation with no heavy TUI dependencies

## Tab Autocompletion

**Rationale:** Commands and provider names should be discoverable and fast to type.

**Behavior:**
- Typing `/con` + `Tab` completes to `/connect`
- Typing `/connect open` + `Tab` completes to `/connect opencode`
- If multiple matches exist, they are displayed without losing the current input

## Ctrl+C Interrupt

**Rationale:** Users expect Ctrl+C to always work as an escape hatch.

**Behavior:**
- Ctrl+C immediately exits the application with a goodbye message

## Vim-Style Exit Easter Egg

**Rationale:** Developers love vim. Supporting `:q`, `:q!`, `:wq`, `:wq!` is a fun nod to the community.

**Behavior:**
- Typing `:q` (or variants) exits the application

## Clear Screen

**Rationale:** Users should be able to clear the terminal without leaving the application.

**Behavior:**
- `/clear` command clears the screen and redraws the banner
- Typing `clear` (without slash) also works

## Configuration System

**Rationale:** Users need flexible ways to configure providers.

**Behavior:**
- `~/.config/opencola/config.json` — full configuration with provider history
- `~/.opencolarc` — simple `.env` format with uppercase keys:
  ```
  OPENAI_API_KEY=sk-...
  OPENAI_BASE_URL=https://api.openai.com/v1
  OPENAI_MODEL=gpt-4o
  ```

## Graceful Degradation

**Rationale:** The application must always start, regardless of configuration.

**Behavior:**
- Starts without requiring any preconfigured environment variables
- Guides users to use `/connect` when no provider is connected

## Extensible Provider Architecture

**Rationale:** Users should not be locked into a single LLM provider.

**Design:**
- `provider.Provider` interface with `Name()`, `ModelName()`, `SetModel()`, `Chat()`, and `ListModels()` methods
- New providers implement the interface and are registered at runtime
