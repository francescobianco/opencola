# TUI Reference

This document defines the exact terminal UI layout for OpenCola. It serves as the single source of truth for how the interface should look and behave.

## Initial Layout

On startup, the terminal is cleared and the layout is rendered as follows:

```
.
.
. [ free space above the banner — variable, depends on terminal height ]
.
.
OpenCola - minimal coding agent
by Francesco Bianco <bianco@javanile.org>
. [ one blank line below the banner ]
>
. [ one blank line below the prompt ]
 OpenCola v0.1.0  Provider: none  Model: none  Status: Disconnected
```

**Positioning rules:**
- The banner is vertically centered in the available space between the top of the terminal and the status bar area
- Free space above the banner is variable — it scales with terminal height
- Exactly one blank line separates the banner from the prompt
- The prompt (`> `) is always one line above the status bar
- The status bar occupies the last line of the terminal at all times

## Status Bar

The status bar is always visible on the last terminal row.

**Layout:**
```
[spinner] [OpenCola v0.1.0] Provider: [name]  Model: [model]  Status: [state]
```

**Colors:**
- The spinner uses default terminal colors
- The logo (`OpenCola v0.1.0`) has **inverted colors**: white background, dark blue text
- The rest of the bar: dark blue background (`rgb(30,64,120)`), white text

**Spinner:**
- 3 characters wide, positioned at the far left of the status bar
- 9-frame animation sequence: ` - `, ` : `, ` = `, `-=−`, `=|=`, `-=−`, ` = `, ` : `, ` - `
- 100ms interval between frames
- Toggled by the hidden `/spin` command (not documented in `/help`)
- Resets to frame 0 when activated

## Banner

```
OpenCola - minimal coding agent
by Francesco Bianco <bianco@javanile.org>
```

- "OpenCola" is rendered in **bold**
- The rest of the text is normal weight and color

## Bye Bye Screen

On exit, the terminal is cleared and a goodbye message is displayed:

```
Goodbye! Thanks for using OpenCola. See you next time!
```
