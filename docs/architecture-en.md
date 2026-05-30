[English](architecture-en.md) | [Русский](architecture-ru.md)

# SD Studio — Architecture

## Stack

| Layer | Technology |
|------|-----------|
| Backend | Go 1.25 |
| Desktop | Wails v2 |
| Frontend | Vue 3 + Vite |
| Database | SQLite (modernc.org/sqlite, no ORM) |
| AI — text | LLM (Ollama / LM Studio / OpenAI-compatible API) |
| AI — images | Stable Diffusion WebUI API |
| Containerization | Docker |

## Project Structure

```
sd-ai/
├── main.go                    # Entrypoint, Wails config
├── app.go                     # Facade: all Wails bindings
├── internal/
│   ├── config/                # Configuration (env, defaults)
│   ├── llm/                   # LLM HTTP client (OpenAI-compatible)
│   ├── sd/                    # Stable Diffusion WebUI HTTP client
│   ├── preset/                # SQLite CRUD (presets, settings, sessions)
│   ├── generation/            # Image generation service
│   ├── queue/                 # Job queue with retry and paused state
│   ├── compositor/            # Multi-pass generation (background + characters)
│   ├── session/               # Session management
│   ├── importexport/          # Preset import/export
│   ├── settings/              # Settings service
│   ├── promptutil/            # Prompt utilities (ExtractJSON, StripJunk)
│   ├── filebrowser/           # File browser backend
│   ├── serverclient/          # Server API client
│   ├── rembg/                 # Background removal (rembg API)
│   ├── kids/                  # Kids mode (content filtering)
│   └── logger/                # Event logger with LogBridge
├── server/                    # Standalone service orchestrator
│   ├── main.go                # Server entrypoint (TUI / headless)
│   ├── config/                # Server configuration (YAML)
│   ├── gpu/                   # GPU monitoring (nvidia-smi)
│   ├── gpuproxy/              # GPU proxy with priority queue
│   ├── health/                # Service health monitoring
│   ├── installer/             # Component installer
│   ├── models/                # Model management
│   ├── process/               # Process lifecycle management
│   ├── tui/                   # Terminal UI (bubbletea)
│   └── api/                   # HTTP API handlers
├── frontend/
│   └── src/
│       ├── components/        # Vue components
│       ├── composables/       # Vue composables (useQueue, etc.)
│       ├── api.js             # API layer (Wails bindings mapping)
│       ├── i18n/              # Internationalization
│       └── wailsjs/           # Auto-generated Wails bindings
├── data/                      # Runtime data (SQLite, presets)
└── docs/                      # Documentation
```

## Testing

| Package | Framework | Tests | Coverage |
|-------|-----------|--------|----------|
| `internal/config` | Go testing | 6 | env, defaults |
| `internal/llm` | Go testing + httptest | ~52 | Chat, GetModels, HealthCheck, CleanTags, StripThinkTags |
| `internal/sd` | Go testing + httptest | ~30 | Retry, Txt2Img, Img2Img, HealthCheck, Get* |
| `internal/preset` | Go testing + :memory: SQLite + testify | ~50 | CRUD, sessions, compound presets, settings, tags |
| `internal/kids` | Go testing + testify | ~11 | Filter false positives, blocked/safe content |
| Frontend | Vitest + @vue/test-utils | — | infra ready (mocks, setup) |
```

## Architectural Principles

### Facade (app.go)
`App` is the single entry point for the frontend. All Wails bindings are exported methods on `*App` (capitalized). The frontend never accesses internal packages directly.

### Client (HTTP wrappers)
`llm.Client`, `sd.Client`, `rembg.Client` — stateless HTTP clients for external services. Constructor injection via `New*`.

### Repository (preset.DB)
`preset.DB` encapsulates SQLite. Raw SQL via `database/sql`. Migrations are defined in Go code (`CREATE TABLE IF NOT EXISTS`).

### Dependency Injection
All dependencies are injected through constructors. `NewApp(presets, llmClient, sdClient, cfg)`.

### Events (Wails runtime)
Backend → Frontend communication via `runtime.EventsEmit(ctx, eventName, data)`.

## Module Diagram

```
┌─────────────────────────────────────────────────┐
│                  frontend (Vue 3)                │
│  components ← composables ← api.js ← wailsjs    │
└──────────────────────┬──────────────────────────┘
                       │ Wails RPC
┌──────────────────────▼──────────────────────────┐
│                  app.go (Facade)                  │
├──────┬────────┬──────────┬──────────┬────────────┤
│ llm  │   sd   │ preset   │ queue    │ generation │
│Client│ Client │   DB     │ Service  │  Service   │
├──────┴────────┴──────────┴──────────┴────────────┤
│  Ollama/LM Studio  │  SD WebUI  │  SQLite        │
└────────────────────┴────────────┴────────────────┘
```

## Server Component

The `server/` package is a standalone Go service for managing AI infrastructure:

```
┌─────────────────────────────────────────┐
│           SD Studio Server               │
├────────┬──────────┬──────────┬──────────┤
│Process │  GPU     │  Health  │   TUI    │
│Manager │ Monitor  │ Monitor  │Dashboard │
├────────┴──────────┴──────────┴──────────┤
│  GPU Proxy (priority queue, VRAM guard)  │
├─────────────────────────────────────────┤
│  HTTP API + mDNS discovery              │
└─────────────────────────────────────────┘
```

### Key Features
- **Process Management** — lifecycle of SD WebUI, Ollama, Rembg (start/stop/restart)
- **GPU Monitoring** — nvidia-smi polling, VRAM tracking, auto-optimization
- **GPU Proxy** — priority queue with VRAM cooldown (prevents OOM on low-VRAM GPUs)
- **Health Monitoring** — periodic HTTP checks for all services
- **TUI** — terminal dashboard via bubbletea with service status and controls
- **mDNS** — automatic service discovery via `_sd-studio._tcp`
- **Model Management** — download/delete SD checkpoints and LLM models
- **Backend Switching** — switch between A1111 and Forge at runtime
- **Installer** — automatic installation of Python, SD WebUI, Ollama

## Database Diagram

```
presets ────┬── preset_types (type_id)
            ├── loras (JSON в поле)
            └── settings (model, vae, sampler)

compound_presets ──── compound_preset_steps

sessions ──── session_items

settings (key-value)
saved_descriptions
saved_prompts
saved_scenes
export_presets
```

## Configuration

All configuration is done via environment variables (env):

| Variable | Description | Default |
|-----------|----------|---------|
| `LLM_URL` | LLM API URL | `http://localhost:1234` |
| `SD_URL` | SD WebUI API URL | `http://localhost:7860` |
| `LLM_MODEL` | Model for prompt generation | — |
| `LLM_BACKEND` | `ollama` or `lmstudio` | `lmstudio` |
| `PORT` | Application port | `8080` |
| `DB_PATH` | SQLite database path | `data/presets.db` |

Settings are also stored in the `settings` table (key-value) and override env vars via the UI.
