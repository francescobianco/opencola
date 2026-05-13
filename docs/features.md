# OpenCola Features

This document describes the core design principles and features of OpenCola. Each feature is motivated by a specific user need or design philosophy, and the document serves as both a specification and a reference for contributors.

## Command History

**Rationale:** Developers expect arrow-key navigation through previous commands, just like in any standard shell. Without this, repetitive tasks become tedious and the tool feels incomplete.

**Behavior:**
- Up/Down arrow keys navigate through previously entered commands
- History persists across sessions via `~/.config/opencola/history`
- Powered by a lightweight readline library (`peterh/liner`) with no heavy TUI dependencies

## Status Bar

**Rationale:** Users should always know the current state of the application without typing a command. A persistent status bar on the last terminal line provides at-a-glance context about the connection, provider, and model.

**Behavior:**
- Occupies the last line of the terminal at all times
- Light gray background (`#e4e4e4`) with dark gray default text
- Key information uses distinct colors: provider name (blue), model name (green), status (green/red)
- Format: `OpenCola v0.1.0 | Provider: [name] | Model: [model] | Status: [Connected/Disconnected]`
- Automatically redraws after each command output

## Graceful Degradation

**Rationale:** The application must always start, regardless of whether API keys or provider configurations are present. Users should not be blocked from exploring the tool just because they haven't configured a provider yet.

**Behavior:**
- The agent starts without requiring any preconfigured environment variables
- When no provider is connected, the user is guided to use `/connect` to set up a connection
- After connecting, `/models` lists available models from the provider

## Slash Commands

**Rationale:** Commands should be discoverable and consistent, following the pattern established by tools like Slack, Discord, and modern CLI applications.

**Available commands:**
| Command | Description |
|---------|-------------|
| `/connect <provider> <api_key> [base_url]` | Connect to an LLM provider |
| `/models` | List available models from the connected provider |
| `/reset` | Clear the current conversation |
| `/status` | Show current connection status |
| `/help` | Show available commands |
| `/exit` | Exit the application |

## Extensible Provider Architecture

**Rationale:** Users should not be locked into a single LLM provider. The provider interface allows swapping backends without changing the rest of the codebase.

**Design:**
- `provider.Provider` interface with `Name()`, `ModelName()`, `Chat()`, and `ListModels()` methods
- New providers implement the interface and are registered at runtime
- Configuration persists across sessions via `~/.config/opencola/config.json`

## Minimalist Pure-Terminal Interface

**Rationale:** The TUI should not interfere with standard terminal workflows. Users must be able to copy/paste text freely and interact with their terminal normally.

**Behavior:**
- Output is plain text that works with terminal scrollback, grep, and pipes
- The only ANSI escape sequences used are for the status bar (last line) and do not affect text selection in the main output area
- No alternate screen buffer or mouse tracking
- No ncurses or tcell dependencies — stdin/stdout with a lightweight readline wrapper
