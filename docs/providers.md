# Provider System

This document describes how provider connection, authentication, and model selection work in OpenCola.

## Available Providers

- `opencode` — default provider
- `opencode-go` — side provider (Go-specific models)
- `opencode-zen` — side provider (Zen-specific models)

## Provider Selection

Provider names are autocompleted with Tab during `/connect`:

```
/connect open[TAB]     →  /connect opencode
/connect opencode-[TAB] →  displays: opencode-go, opencode-zen
```

## API Key Handling

The API key is never entered inline. After selecting a provider, a separate prompt appears:

```
> /connect opencode-go
Please enter your API key for opencode-go:
```

The key is stored in `~/.config/opencola/config.json` and `~/.opencolarc`.

## Model Listing

When `/models` is called, only models available for the connected provider are fetched. Side providers (`opencode-go`, `opencode-zen`) only expose their own model sets.

## Model Selection Menu

The model menu appears below the prompt and displays up to 4 models at a time:

- A `> ` cursor marks the current selection
- Up/Down arrow keys navigate the list (scrolling if more than 4 models)
- Enter confirms the selection
- The selected model appears in the status bar
