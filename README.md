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

## Improvement Roadmap

Prioritized findings from code review (2026-04-24).

### Critical — Security & Correctness

| # | Issue | Effort |
|---|-------|--------|
| 1 | Kids Mode PIN: SHA-256 without salt (rainbow tables), `!=` hash comparison (timing attack), silent error on hash read | 1h |
| 2 | Kids Mode filter: false positives (`"class"` matches `"ass"`, `"dismissing"` matches `"smoking"`) — bare `strings.Contains` | 1h |
| 3 | `CheckServices` race condition — goroutines write shared `status` without mutex | 30m |
| 4 | `CheckServices` writes `SDPromptModel` into LLM status field (bug) | 10m |
| 5 | No input validation on user-provided URLs and numeric fields | 1h |

### Important — Architecture & Reliability

| # | Issue | Effort |
|---|-------|--------|
| 6 | No interfaces for LLM/SD clients — `app.go` depends on concrete types, untestable | 2–3h |
| 7 | No tests at all (no `*_test.go`, no frontend tests) | 2–3d |
| 8 | Duplicated settings validation in `app.go` and `api/handler.go`; whitelist missing newer fields (`llm_backend`, `llm_keep_alive`) | 1h |
| 9 | No `context.Context` in HTTP calls — requests can hang on app exit | 2h |
| 10 | SQLite `SetMaxOpenConns(1)` — potential bottleneck | varies |

### Nice-to-have — UX & Quality

| # | Issue | Effort |
|---|-------|--------|
| 11 | Frontend: missing error handling, no retry buttons, no progress indicators | 1d |
| 12 | Leftover `HelloWorld.vue` template not removed | 5m |
| 13 | No DB migration system — `CREATE TABLE IF NOT EXISTS` only, no versioning/rollback | 0.5d |
| 14 | Docker container runs as root, no healthcheck, no resource limits | 2h |
| 15 | No CI/CD, no linters (golangci-lint, prettier, eslint) | 0.5d |
| 16 | `log.Printf` logging — no levels, no structured output; response body logged in full (`llm/client.go:104`) | 0.5d |

### Priority Matrix

| Priority | Items | Description |
|----------|-------|-------------|
| **P0** | 1, 3, 4 | Security fixes and bugs |
| **P1** | 5, 6, 8, 9 | Architecture improvements |
| **P2** | 7, 11, 12, 13 | Tests, UX, migrations |
| **P3** | 10, 14, 15, 16 | Ops, quality tooling |

## License

[GNU Affero General Public License v3.0](LICENSE)
