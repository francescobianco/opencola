# TUI Reference

This document defines the exact terminal UI layout for OpenCola. It serves as the single source of truth for how the interface should look and behave.

## Initial Layout

On startup, the terminal is cleared and the layout is rendered as follows:

```
.
. [ free space above the banner — variable, depends on terminal height ]
.
.
OpenCola - minimal coding agent
by Francesco Bianco <bianco@javanile.org>
Type /help for a list of commands.
. [ one blank line below the banner ]
>
. [ one blank line below the prompt ]
 OpenCola v0.1.0  Provider: none  Model: none  Status: Disconnected
```

**Positioning rules:**
- The banner is at a fixed position from the bottom, not centered
- Counting from the bottom: status bar (1 row), blank line (1 row), prompt (1 row), blank line (1 row), banner (3 rows)
- All remaining rows above the banner are free space — variable, depends on terminal height
- Exactly one blank line separates the banner from the prompt
- The prompt (`> `) is always one line above the blank line that sits above the status bar
- The status bar occupies the last line of the terminal at all times

## Status Bar

The status bar is always visible on the last terminal row.

**Layout:**
```
[spinner] [OpenCola v0.1.0] Provider: [name]  Model: [model]  Status: [state]
```

**Colors:**
- The spinner has **yellow foreground** (`rgb(255,200,50)`) on dark blue background (`rgb(30,64,120)`)
- The logo (`OpenCola v0.1.0`) has **inverted colors**: white background, dark blue text
- The rest of the bar: dark blue background (`rgb(30,64,120)`), white text

**Spinner:**
- 3 characters wide, positioned at the far left of the status bar
- Continuous flow animation using character left-rotation (pop first, append last): `...`, `|..`, `||.`, `.||`, `..|`
- 50ms interval between frames (full cycle in 250ms)
- Toggled by the hidden `/spin` command (not documented in `/help`)
- Resets to initial state on each toggle
- When active, only the status bar animates; the prompt remains visible and fully interactive

## Banner

```
OpenCola - minimal coding agent
by Francesco Bianco <bianco@javanile.org>
Type /help for a list of commands.
```

- "OpenCola" is rendered in **bold**
- The rest of the text is normal weight and color

## Bye Bye Screen

On exit, the terminal is cleared and a goodbye message is displayed:

```
Goodbye! Thanks for using OpenCola. See you next time!
```
