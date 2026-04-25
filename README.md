# infai

```text
██╗███╗   ██╗███████╗ █████╗ ██╗
██║████╗  ██║██╔════╝██╔══██╗██║
██║██╔██╗ ██║█████╗  ███████║██║
██║██║╚██╗██║██╔══╝  ██╔══██║██║
██║██║ ╚████║██║     ██║  ██║██║
╚═╝╚═╝  ╚═══╝╚═╝     ╚═╝  ╚═╝╚═╝
```

**Zero-management launch templates for [llama.cpp](https://github.com/ggerganov/llama.cpp).**  
Create one-click runnable profiles, manage your inference engine and model locations simply, and monitor performance with live server logs.

## Features

- **One-Click Launch** — Instant start with pre-configured, named profiles (e.g., `text-only`, `low-vram`).
- **Zero Template Management** — Stop messing with shell aliases and scripts; manage everything through a clean TUI.
- **Live Inference Logs** — Monitor your server in real-time with a built-in scrollable viewport.
- **Smart UI** — Easy pickers for quantization types and context size units (K, M).
- **Easy Path Management** — Quickly configure multiple scan folders and your `llama-server` binary path.
- **Themes** — 11+ themes (Tokyonight, Gruvbox, Rose Pine, etc.) to match your terminal setup.

## Install

Requires Go 1.23+ and a C compiler (for SQLite).

```bash
go install github.com/dipankardas011/infai@latest
```

## Key Bindings

| Screen | Keys | Action |
|--------|------|--------|
| **Home** | `a`, `f`, `c` | All models, manage folders, configure executor |
| **Model List** | `enter`, `/`, `r` | Select, Filter, Rescan |
| **Profile List** | `enter`, `e`, `d` | Launch, Edit, Delete |
| **Editor** | `tab`, `space`, `ctrl+s`| Navigate, Toggle, Save |
| **Logs** | `s`, `esc`, `↑↓` | Stop, Back, Scroll |

## Configuration

Settings and profiles are stored in a local SQLite database:
- **Linux**: `~/.config/infai/config.db`
- **macOS**: `~/Library/Application Support/infai/config.db`
- **Windows**: `%AppData%\infai\config.db`

## License
MIT
