# SD Studio

Desktop-приложение для генерации изображений через [Stable Diffusion WebUI](https://github.com/AUTOMATIC1111/stable-diffusion-webui) с LLM-генерацией промптов.

## Технологии

| Слой | Стек |
|------|------|
| Backend | Go 1.25 |
| Desktop | [Wails](https://wails.io/) v2 |
| Frontend | Vue 3 + Vite |
| БД | SQLite ([modernc.org/sqlite](https://gitlab.com/cznic/sqlite), pure Go) |
| LLM | OpenAI-compatible API |
| SD | Stable Diffusion WebUI API |

## Возможности

- Генерация SD-промптов через LLM по текстовому описанию
- Генерация изображений через Stable Diffusion WebUI API
- Управление пресетами (параметры генерации: sampler, steps, cfg_scale, размер, seed)
- Настройка подключения к LLM и SD API (URL, модели)
- Сохранение описаний для повторного использования

## Требования

### Общие

- [Go](https://go.dev/dl/) >= 1.25
- [Node.js](https://nodejs.org/) >= 18
- npm

### Linux

- `libgtk-3-dev`, `libwebkit2gtk-4.1-dev` (или `libwebkit2gtk-4.0-dev`)

### Windows

- [WebView2](https://developer.microsoft.com/en-us/microsoft-edge/webview2/) (встроена в Windows 10/11)

### macOS

- Xcode Command Line Tools (`xcode-select --install`)
- macOS 10.15+ (Catalina)

## Сборка

### Установка Wails CLI

```bash
go install github.com/wailsapp/wails/v2/cmd/wails@latest
```

### Разработка

```bash
wails dev
```

Frontend hot-reload на `http://localhost:34115`, desktop-окно открывается автоматически.

### Production-сборка

```bash
wails build
```

Бинарник: `build/bin/sd-studio` (Linux/macOS) или `build/bin/sd-studio.exe` (Windows).

### Сборка через Docker

```bash
docker compose up --build
```

## Конфигурация

Приложение подключается к внешним сервисам, указываемым в настройках интерфейса:

| Параметр | По умолчанию | Описание |
|----------|-------------|----------|
| `llm_url` | `http://localhost:11434/v1` | URL LLM API (Ollama, OpenAI-compatible) |
| `sd_url` | `http://localhost:7860` | URL Stable Diffusion WebUI API |
| `llm_model` | — | Модель LLM для генерации промптов |
| `sd_prompt_model` | — | Модель LLM для системного промпта |

Настройки хранятся в SQLite (`data/presets.db`) и переживают переустановку.

## Структура проекта

```
├── main.go              # Entrypoint
├── app.go               # Wails RPC bindings
├── internal/
│   ├── config/          # Конфигурация
│   ├── llm/             # LLM клиент
│   ├── preset/          # SQLite CRUD (пресеты, настройки)
│   ├── sd/              # Stable Diffusion клиент
│   └── api/             # HTTP API
├── frontend/
│   └── src/
│       ├── components/  # Vue компоненты
│       └── wailsjs/     # Auto-generated Wails bindings
└── data/                # SQLite DB (runtime)
```

## Лицензия

Приватный проект.
