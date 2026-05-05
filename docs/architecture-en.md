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
├── app.go                     # Facade: все Wails bindings
├── internal/
│   ├── config/                # Конфигурация (env, defaults)
│   ├── llm/                   # LLM HTTP-клиент (OpenAI-compatible)
│   ├── sd/                    # Stable Diffusion WebUI HTTP-клиент
│   ├── preset/                # SQLite CRUD (presets, settings, sessions)
│   ├── compositor/            # Multi-pass генерация (background + characters)
│   ├── rembg/                 # Удаление фона (rembg API)
│   ├── kids/                  # Kids mode (фильтрация контента)
│   └── logger/                # Логирование через Wails runtime
├── frontend/
│   └── src/
│       ├── components/        # Vue компоненты
│       ├── api.js             # API-слой (маппинг Wails bindings)
│       ├── i18n/              # Интернационализация (en.js + index.js)
│       ├── assets/            # CSS
│       └── wailsjs/           # Auto-generated Wails bindings
├── data/                      # SQLite DB
└── docs/                      # Документация
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
┌─────────────────────────────────────────┐
│              frontend (Vue 3)            │
│  components ← i18n/t() ← api.js ← wailsjs/bindings │
└──────────────────┬──────────────────────┘
                   │ Wails RPC
┌──────────────────▼──────────────────────┐
│                app.go (Facade)           │
│  Wails bindings + business logic        │
├────────┬──────────┬──────────┤
│ llm    │   sd     │ preset   │
│ Client │  Client  │   DB     │
├────────┴──────────┴──────────┤
│  Ollama/LM Studio │ SD WebUI │ SQLite   │
└───────────────────┴──────────┴──────────┘
```

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
