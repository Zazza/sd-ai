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

## External Services

SD Studio requires two external services running on a machine in your local network with their API accessible:

- **Image generation** — [Stable Diffusion WebUI (A1111)](https://github.com/AUTOMATIC1111/stable-diffusion-webui) must be running with the `--api` flag. Configure its URL in Settings (default: `http://localhost:7860`).

- **Prompt generation (LLM)** — Any OpenAI-compatible API server: [llama.cpp](https://github.com/ggerganov/llama.cpp) (`--server` mode), [Ollama](https://ollama.com/), or [LM Studio](https://lmstudio.ai/). Configure URL and model in Settings (default: `http://localhost:11434/v1`).

Both services must be reachable from the machine running SD Studio. If they run on a different host, set the URL accordingly (e.g. `http://192.168.1.100:7860`).

## Features

- **Smart merge prompt generation** — describe what you want in natural language, LLM merges your description with the preset into a proper SD prompt
- **Preset recommendation** — LLM analyzes your description and recommends the best matching preset from your library
- **Editable SD prompt instruction** — customize how LLM formats SD prompts via Settings > Prompt tab
- Generate images through Stable Diffusion WebUI API
- **Batch generation** — generate N images to a specified folder with progress tracking
- **Test page** — multi-select presets or models and generate images directly through SD for comparison
- **Pipelines (Compound Presets)** — multi-step generation chains (txt2img → img2img → ...) for complex workflows
- **Generate From Image** — upload image, analyze via vision LLM (quick or deep mode), generate with preset, inpaint with canvas mask drawing
- **Multi-Scene Editor** — LLM decomposes scene into multiple characters, inpaint-based multi-pass compositing with rembg background removal
- Manage presets (generation parameters: sampler, steps, cfg_scale, size, seed, LoRA)
- Preset types with tags for organization
- Import/Export presets with model validation
- Configure LLM and SD API connections (URL, models, backend-specific params)
- Save descriptions and prompts for reuse
- Saved scenes for re-running multi-scene compositions
- Kids Mode with content filtering, PIN protection, and category toggles
- NSFW preset section
- Event logger with footer log viewer
- Preview mode with upscale to full resolution

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

### Makefile Commands

```bash
make dev       # wails dev
make build     # npm install + wails build
make test      # go vet/test + frontend build
make lint      # go vet + vue-tsc
make tidy      # go mod tidy
make clean     # remove build/
```

### Development

```bash
make dev
```

Frontend hot-reload at `http://localhost:34115`, desktop window opens automatically.

### Production Build

```bash
make build
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
| `sd_prompt_instruction` | built-in default | Instruction sent to LLM for SD prompt formatting (editable in Settings > Prompt) |

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
└── data/                # Runtime data
    ├── presets.db       # SQLite database
    ├── models.json      # Available SD models catalog
    ├── recommended-loras.json  # Recommended LoRAs with default weights
    ├── SD-models.md     # Model descriptions reference
    ├── SD-LORAS.md      # LoRA descriptions reference
    └── presets/         # JSON preset files (import-ready)
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

## Generation Flow

### Standard Generation
```
User selects preset + writes description
         │
         ▼
  ┌─ Has description? ─┐
  No                    Yes
  │                     │
  │              LLM Smart Merge
  │         (preset prompt + description
  │          + sd_prompt_instruction)
  │                     │
  │              Merged SD prompt
  │                     │
  └───────┬─────────────┘
          ▼
   Generate Image via SD API
          │
          ▼
   Preview → Upscale → Download
```

### From Image
```
Upload image → Analyze (quick/deep via vision LLM)
         │
         ▼
  Select preset (manual or LLM-recommended)
         │
         ▼
  Generate (img2img) or Inpaint (canvas mask)
```

### Multi-Scene
```
Describe scene → LLM decomposes into characters
         │
         ▼
  For each character: rembg → inpaint into scene
         │
         ▼
  Save scene for re-use
```

Preset recommendation: user describes what they want → LLM selects best preset from library + suggests additional tags.

## License

[GNU Affero General Public License v3.0](LICENSE)
