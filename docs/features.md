# OpenCola Features

This document describes the core design principles and features of OpenCola. Each feature is motivated by a specific user need or design philosophy.

## Graceful Degradation

**Rationale:** The application must always start, regardless of whether API keys or provider configurations are present. Users should not be blocked from exploring the tool just because they haven't configured a provider yet.

**Behavior:**
- The agent starts without requiring any preconfigured environment variables
- When no provider is connected, the user is guided to use `/connect` to set up a connection
- After connecting, `/models` lists available models from the provider

## Consistent Terminal UI

**Rationale:** A predictable interface reduces cognitive load. The TUI structure remains the same throughout the session — only the content changes, not the layout.

**Behavior:**
- A single, consistent prompt format (`> `) is used for all input
- Status is displayed in a minimal `[provider]` indicator before each prompt
- The structure never changes: banner → status → prompt → output → status → prompt

## Minimalist Pure-Terminal Interface

**Rationale:** The TUI should not interfere with standard terminal workflows. Users must be able to copy/paste text freely and interact with their terminal normally.

**Behavior:**
- No ANSI escape sequences that break text selection
- No alternate screen buffer or mouse tracking
- Plain text output that works with terminal scrollback, grep, and pipes
- No ncurses or tcell dependencies — just stdin/stdout

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
- `provider.Provider` interface with `Chat()` and `ListModels()` methods
- New providers implement the interface and are registered at runtime
- Configuration persists across sessions via `~/.config/opencola/config.json`
