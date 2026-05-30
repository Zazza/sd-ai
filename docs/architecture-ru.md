[English](architecture-en.md) | [Русский](architecture-ru.md)

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
│   ├── generation/            # Сервис генерации изображений
│   ├── queue/                 # Очередь задач с повторами и паузой
│   ├── compositor/            # Multi-pass генерация (background + characters)
│   ├── session/               # Управление сессиями
│   ├── importexport/          # Импорт/экспорт пресетов
│   ├── settings/              # Сервис настроек
│   ├── promptutil/            # Утилиты промптов (ExtractJSON, StripJunk)
│   ├── filebrowser/           # Файловый браузер
│   ├── serverclient/          # Клиент API сервера
│   ├── rembg/                 # Удаление фона (rembg API)
│   ├── kids/                  # Kids mode (фильтрация контента)
│   └── logger/                # Логирование событий с LogBridge
├── server/                    # Автономный оркестратор сервисов
│   ├── main.go                # Entrypoint сервера (TUI / headless)
│   ├── config/                # Конфигурация сервера (YAML)
│   ├── gpu/                   # Мониторинг GPU (nvidia-smi)
│   ├── gpuproxy/              # GPU-прокси с приоритетной очередью
│   ├── health/                # Мониторинг здоровья сервисов
│   ├── installer/             # Установщик компонентов
│   ├── models/                # Управление моделями
│   ├── process/               # Управление жизненным циклом процессов
│   ├── tui/                   # Терминальный UI (bubbletea)
│   └── api/                   # HTTP API обработчики
├── frontend/
│   └── src/
│       ├── components/        # Vue компоненты
│       ├── composables/       # Vue composables (useQueue и др.)
│       ├── api.js             # API-слой (маппинг Wails bindings)
│       ├── i18n/              # Интернационализация
│       └── wailsjs/           # Auto-generated Wails bindings
├── data/                      # Runtime-данные (SQLite, пресеты)
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

## Компонент Server

Пакет `server/` — автономный Go-сервис для управления AI-инфраструктурой:

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

### Ключевые возможности
- **Управление процессами** — жизненный цикл SD WebUI, Ollama, Rembg (start/stop/restart)
- **Мониторинг GPU** — опрос nvidia-smi, отслеживание VRAM, автооптимизация
- **GPU-прокси** — приоритетная очередь с VRAM cooldown (предотвращает OOM на GPU с малой VRAM)
- **Мониторинг здоровья** — периодические HTTP-проверки всех сервисов
- **TUI** — терминальный дашборд на bubbletea со статусом сервисов и управлением
- **mDNS** — автоматическое обнаружение сервисов через `_sd-studio._tcp`
- **Управление моделями** — загрузка/удаление SD-чекпоинтов и LLM-моделей
- **Переключение бэкендов** — переключение между A1111 и Forge во время работы
- **Установщик** — автоматическая установка Python, SD WebUI, Ollama

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
