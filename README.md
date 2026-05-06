# SD Studio

**AI-powered desktop app for image generation with Stable Diffusion + LLM.**

[Русский](README-ru.md) | [Changelog](CHANGELOG.md)

SD Studio bridges your local Stable Diffusion and LLM into a single workflow: describe what you want in plain language, get a properly crafted SD prompt, and generate — no manual tag wrangling. Built for power users who want full control over their local pipeline.

> **Note:** SD Studio connects to your local [Stable Diffusion WebUI](https://github.com/AUTOMATIC1111/stable-diffusion-webui) and an [OpenAI-compatible LLM](https://ollama.com). Both run on your hardware — no cloud, no subscriptions, no data leaving your machine. You can also use [Stable Diffusion WebUI Forge](https://github.com/lllyasviel/stable-diffusion-webui-forge) for faster generation (see [Setup Guide](docs/setup-en.md) for details and known limitations).

---

## Highlights

- **LLM Prompt Engineering** — describe in natural language, LLM merges your intent with preset into a production-ready SD prompt
- **Smart Remove** — draw a mask, LLM analyzes the context, inpaints the background automatically
- **Multi-Scene Composition** — describe a scene, LLM decomposes it into characters, composites via multi-pass inpaint
- **Pipelines** — chain multiple generation steps (txt2img → img2img → inpaint) into a single workflow
- **Session Management** — organize work into sessions with full generation history
- **Kids Mode** — PIN-protected content filtering with category controls

## Screenshots

<p align="center">
  <img src="docs/screenshots/main-generation.png" width="45%" alt="Text-to-Image generation">
  <img src="docs/screenshots/from-image-inpaint.png" width="45%" alt="Inpainting with mask editor">
</p>
<p align="center">
  <img src="docs/screenshots/scene-editor.png" width="45%" alt="Multi-scene composition">
  <img src="docs/screenshots/batch-generation.png" width="45%" alt="Batch generation">
</p>

## Features

### Generation

| Feature | Description |
|---------|-------------|
| Text-to-Image | Generate from text with presets, LoRA, custom samplers |
| Image-to-Image | Transform existing images with denoising control |
| Inpainting | Canvas mask editor with fullscreen mode, brush controls, undo |
| Smart Remove | AI-powered object removal — draw mask, context auto-analyzed by LLM vision |
| Batch Generation | Generate N images with progress tracking |
| Pipelines | Multi-step compound presets for complex workflows |
| Upscale | Preview mode with one-click upscale to full resolution |

### LLM Integration

| Feature | Description |
|---------|-------------|
| Smart Merge | Natural language description → merged SD prompt via LLM |
| Vision Analysis | Upload image, analyze via vision LLM (quick or deep chain mode) |
| Preset Recommendation | LLM picks the best preset from your library for a given description |
| Multi-Scene Decomposition | LLM breaks scene description into individual characters |
| Customizable Instruction | Edit the system prompt that shapes LLM output format |

### Workflow & Management

| Feature | Description |
|---------|-------------|
| Sessions | Project-based sessions with full generation history and navigation |
| Presets | Save, organize by type, import/export with model validation |
| File Browser | Thumbnail grid, fullscreen viewer, quick send to generation |
| Export | Resize, convert (PNG/JPEG/WebP), quality/interpolation control |
| Saved Descriptions | Reuse prompts and descriptions across sessions |
| Light/Dark Theme | System-aware theme with manual toggle |

### Safety

| Feature | Description |
|---------|-------------|
| Kids Mode | PIN protection, content filtering by category, safe prompt modification |

## Requirements

### External Services

SD Studio connects to two services on your local network:

- **Stable Diffusion WebUI** (A1111 or [Forge](https://github.com/lllyasviel/stable-diffusion-webui-forge)) — runs with `--api` flag. [Setup guide](docs/setup-en.md). Default: `http://localhost:7860`
- **LLM API** — any OpenAI-compatible server: [Ollama](https://ollama.com/), [llama.cpp](https://github.com/ggerganov/llama.cpp), or [LM Studio](https://lmstudio.ai/). Default: `http://localhost:11434/v1`

Optional: [Rembg](https://github.com/danielgatis/rembg) for background removal in multi-scene mode.

### Development

- [Go](https://go.dev/dl/) >= 1.25
- [Node.js](https://nodejs.org/) >= 18
- [Wails CLI](https://wails.io/) v2

| Platform | Requirements |
|----------|-------------|
| macOS | Xcode CLI tools (`xcode-select --install`), 10.15+ |
| Linux | `libgtk-3-dev`, `libwebkit2gtk-4.1-dev` |
| Windows | [WebView2](https://developer.microsoft.com/en-us/microsoft-edge/webview2/) (built into Win 10/11) |

## Download

Pre-built releases are available on the [Releases page](https://github.com/Zazza/sd-ai/releases).

## Quick Start

```bash
# Install Wails CLI
go install github.com/wailsapp/wails/v2/cmd/wails@latest

# Clone and run
git clone https://github.com/Zazza/sd-ai.git
cd sd-ai
make setup   # first-time: download deps
make dev     # launch with hot-reload
```

Frontend hot-reload at `http://localhost:34115`.

### Build

```bash
make build   # production binary → build/bin/sd-studio
```

### Docker

```bash
docker compose up --build
```

## Tech Stack

| Layer | Stack |
|-------|-------|
| Backend | Go 1.25 |
| Desktop | [Wails](https://wails.io/) v2 |
| Frontend | Vue 3 + Vite |
| Database | SQLite (pure Go, no CGo) |
| LLM | OpenAI-compatible API |
| Image Gen | Stable Diffusion WebUI API |

## How It Works

```
User writes description in plain language
         │
         ▼
  LLM Smart Merge
  (description + preset + instruction → SD prompt)
         │
         ▼
  Generate via Stable Diffusion WebUI
         │
         ▼
  Preview → Upscale → Export
```

**From Image:** Upload → Vision LLM analyzes → Inpaint/Remove with mask editor

**Multi-Scene:** Describe scene → LLM decomposes → Multi-pass inpaint compositing

**Smart Remove:** Draw mask → LLM vision analyzes context → Auto-inpaint background

## Project Structure

```
├── main.go              # Entrypoint
├── app.go               # Wails RPC bindings
├── internal/
│   ├── config/          # Configuration
│   ├── llm/             # LLM client
│   ├── preset/          # SQLite CRUD
│   ├── sd/              # Stable Diffusion client
│   ├── compositor/      # Multi-scene compositing
│   ├── kids/            # Kids mode filtering
│   ├── rembg/           # Background removal client
│   ├── logger/          # Event logger
│   └── api/             # HTTP API
├── frontend/
│   └── src/
│       ├── components/  # Vue components
│       └── wailsjs/     # Auto-generated Wails bindings
└── data/                # Runtime data (SQLite, presets)
```

## License

[GNU Affero General Public License v3.0](LICENSE)
