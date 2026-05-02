# SD Studio — Архитектура

## Стек

| Слой | Технология |
|------|-----------|
| Backend | Go 1.25 |
| Desktop | Wails v2 |
| Frontend | Vue 3 + Vite |
| БД | SQLite (modernc.org/sqlite, без ORM) |
| AI — текст | LLM (Ollama / LM Studio / OpenAI-совместимый API) |
| AI — изображения | Stable Diffusion WebUI API |
| Контейнеризация | Docker |

## Структура проекта

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
│       ├── assets/            # CSS
│       └── wailsjs/           # Auto-generated Wails bindings
├── data/                      # SQLite DB
└── docs/                      # Документация
```

## Testing

| Пакет | Фреймворк | Тестов | Покрытие |
|-------|-----------|--------|----------|
| `internal/config` | Go testing | 6 | env, defaults |
| `internal/llm` | Go testing + httptest | ~52 | Chat, GetModels, HealthCheck, CleanTags, StripThinkTags |
| `internal/sd` | Go testing + httptest | ~30 | Retry, Txt2Img, Img2Img, HealthCheck, Get* |
| `internal/preset` | Go testing + :memory: SQLite + testify | ~50 | CRUD, sessions, compound presets, settings, tags |
| `internal/kids` | Go testing + testify | ~11 | Filter false positives, blocked/safe content |
| Frontend | Vitest + @vue/test-utils | — | infra ready (mocks, setup) |
```

## Архитектурные принципы

### Facade (app.go)
`App` — единая точка входа для frontend. Все Wails bindings — методы на `*App` с заглавной буквы. Frontend не обращается к internal напрямую.

### Client (HTTP-обёртки)
`llm.Client`, `sd.Client`, `rembg.Client` — Stateless HTTP-клиенты для внешних сервисов. Constructor injection через `New*`.

### Repository (preset.DB)
`preset.DB` инкапсулирует SQLite. Raw SQL через `database/sql`. Миграции в Go-коде (`CREATE TABLE IF NOT EXISTS`).

### Dependency Injection
Все зависимости внедряются через конструкторы. `NewApp(presets, llmClient, sdClient, cfg)`.

### Events (Wails runtime)
Backend → Frontend коммуникация через `runtime.EventsEmit(ctx, eventName, data)`.

## Диаграмма модулей

```
┌─────────────────────────────────────────┐
│              frontend (Vue 3)            │
│  components ← api.js ← wailsjs/bindings │
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

## Диаграмма БД

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

## Конфигурация

Все через переменные окружения (env):

| Переменная | Описание | Default |
|-----------|----------|---------|
| `LLM_URL` | URL LLM API | `http://localhost:1234` |
| `SD_URL` | URL SD WebUI API | `http://localhost:7860` |
| `LLM_MODEL` | Модель для генерации промптов | — |
| `LLM_BACKEND` | `ollama` или `lmstudio` | `lmstudio` |
| `PORT` | Порт приложения | `8080` |
| `DB_PATH` | Путь к SQLite | `data/presets.db` |

Настройки также хранятся в таблице `settings` (key-value) и переопределяют env через UI.
