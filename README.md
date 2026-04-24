# SD Studio

Desktop application for image generation via [Stable Diffusion WebUI](https://github.com/AUTOMATIC1111/stable-diffusion-webui) with LLM-powered prompt generation.

## Tech Stack

| Layer | Stack |
|-------|-------|
| Backend | Go 1.25 |
| Desktop | [Wails](https://wails.io/) v2 |
| Frontend | Vue 3 + Vite |
| Database | SQLite ([modernc.org/sqlite](https://gitlab.com/cznic/sqlite), pure Go) |
| LLM | OpenAI-compatible API |
| SD | Stable Diffusion WebUI API |

## Features

- Generate SD prompts via LLM from text descriptions (any language)
- Generate images through Stable Diffusion WebUI API
- Manage presets (generation parameters: sampler, steps, cfg_scale, size, seed)
- Configure LLM and SD API connections (URL, models)
- Save descriptions for reuse
- Kids Mode with content filtering and PIN protection

## Requirements

### General

- [Go](https://go.dev/dl/) >= 1.25
- [Node.js](https://nodejs.org/) >= 18
- npm

### Linux

- `libgtk-3-dev`, `libwebkit2gtk-4.1-dev` (or `libwebkit2gtk-4.0-dev`)

### Windows

- [WebView2](https://developer.microsoft.com/en-us/microsoft-edge/webview2/) (built into Windows 10/11)

### macOS

- Xcode Command Line Tools (`xcode-select --install`)
- macOS 10.15+ (Catalina)

## Build

### Install Wails CLI

```bash
go install github.com/wailsapp/wails/v2/cmd/wails@latest
```

### Development

```bash
wails dev
```

Frontend hot-reload at `http://localhost:34115`, desktop window opens automatically.

### Production Build

```bash
wails build
```

Binary: `build/bin/sd-studio` (Linux/macOS) or `build/bin/sd-studio.exe` (Windows).

### Docker Build

```bash
docker compose up --build
```

## Configuration

The application connects to external services configured via the settings UI:

| Parameter | Default | Description |
|-----------|---------|-------------|
| `llm_url` | `http://localhost:11434/v1` | LLM API URL (Ollama, OpenAI-compatible) |
| `sd_url` | `http://localhost:7860` | Stable Diffusion WebUI API URL |
| `llm_model` | — | LLM model for prompt generation |
| `sd_prompt_model` | — | LLM model for system prompt |

Settings are stored in SQLite (`data/presets.db`) and persist across reinstalls.

## Project Structure

```
├── main.go              # Entrypoint
├── app.go               # Wails RPC bindings
├── internal/
│   ├── config/          # Configuration
│   ├── llm/             # LLM client
│   ├── preset/          # SQLite CRUD (presets, settings)
│   ├── sd/              # Stable Diffusion client
│   └── api/             # HTTP API
├── frontend/
│   └── src/
│       ├── components/  # Vue components
│       └── wailsjs/     # Auto-generated Wails bindings
└── data/                # SQLite DB (runtime)
```

## License

Private project.
