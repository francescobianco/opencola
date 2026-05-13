# OpenCola Features

This document describes the core design principles and features of OpenCola. Each feature is motivated by a specific user need or design philosophy, and the document serves as both a specification and a reference for contributors.

## TUI Layout

**Rationale:** A clean, predictable terminal interface reduces cognitive load and makes the tool feel professional. The layout follows a minimal design inspired by modern CLI tools.

**Behavior:**
- On startup, the terminal is cleared and a minimal banner is displayed:
  ```
  OpenCola - minimal coding agent
  by Francesco Bianco <bianco@javanile.org>

  >
  ```
- The status bar occupies the last line at all times (dark blue background, white text)
- On exit, the terminal is cleared and a goodbye message is shown:
  ```
  Goodbye! Thanks for using OpenCola. See you next time!
  ```

## Command History & Navigation

**Rationale:** Developers expect arrow-key navigation through previous commands, just like in any standard shell. Without this, repetitive tasks become tedious.

**Behavior:**
- Up/Down arrow keys navigate through previously entered commands
- History persists across sessions via `~/.config/opencola/history`
- Lightweight readline implementation with no heavy TUI dependencies

## Tab Autocompletion

**Rationale:** Slash commands should be discoverable and fast to type. Tab completion reduces typing effort and helps users discover available commands.

**Behavior:**
- Typing `/con` + `Tab` completes to `/connect`
- If multiple matches exist, they are displayed without losing the current input
- Available commands: `/connect`, `/models`, `/reset`, `/clear`, `/status`, `/help`, `/exit`, `/quit`

## Ctrl+C Interrupt

**Rationale:** Users expect Ctrl+C to always work as an escape hatch. The application should never trap the user in a state where they cannot exit.

**Behavior:**
- Ctrl+C immediately exits the application with a goodbye message
- No confirmation required — it's an emergency exit

## Vim-Style Exit Easter Egg

**Rationale:** Developers love vim. Supporting `:q`, `:q!`, `:wq`, `:wq!` as exit commands is a fun nod to the community and makes the tool feel familiar.

**Behavior:**
- Typing `:q` (or variants) exits the application
- Works alongside `/exit` and `/quit`

## Clear Screen

**Rationale:** Users should be able to clear the terminal without leaving the application, matching the behavior of standard shells.

**Behavior:**
- `/clear` command clears the screen and redraws the banner
- Typing `clear` (without slash) also works, matching shell convention

## Configuration System

**Rationale:** Users need flexible ways to configure providers. Supporting both a JSON config and a simple `.env` file accommodates different workflows.

**Behavior:**
- `~/.config/opencola/config.json` — full configuration with provider history
- `~/.opencolarc` — simple `.env` format with uppercase keys:
  ```
  OPENAI_API_KEY=sk-...
  OPENAI_BASE_URL=https://api.openai.com/v1
  OPENAI_MODEL=gpt-4o
  ```
- The `/connect` command saves to both files simultaneously

## Graceful Degradation

**Rationale:** The application must always start, regardless of whether API keys or provider configurations are present.

**Behavior:**
- The agent starts without requiring any preconfigured environment variables
- When no provider is connected, the user is guided to use `/connect`
- After connecting, `/models` lists available models

## Status Bar

**Rationale:** Users should always know the current state of the application without typing a command.

**Behavior:**
- Occupies the last line of the terminal at all times
- Dark blue background (`rgb(30,64,120)`) with white text
- Format: `OpenCola v0.1.0 | Provider: [name] | Model: [model] | Status: [Connected/Disconnected]`
- Automatically redraws after each command output

## Extensible Provider Architecture

**Rationale:** Users should not be locked into a single LLM provider.

**Design:**
- `provider.Provider` interface with `Name()`, `ModelName()`, `Chat()`, and `ListModels()` methods
- New providers implement the interface and are registered at runtime
- Configuration persists across sessions
