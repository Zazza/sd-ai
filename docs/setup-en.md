[English](setup-en.md) | [Русский](setup-ru.md)

# SD Studio — Service Setup Guide

## Contents

- [Stable Diffusion WebUI](#stable-diffusion-webui)
- [Ollama](#ollama)
- [LM Studio](#lm-studio)
- [llama.cpp](#llamacpp)
- [Rembg](#rembg)
- [SD Studio Server](#sd-studio-server)

---

## Stable Diffusion WebUI

Image generation engine. SD Studio connects via REST API. Two variants are supported: **A1111** (standard) and **Forge** (faster).

### Option 1: A1111 (standard, recommended)

```bash
git clone https://github.com/AUTOMATIC1111/stable-diffusion-webui.git
cd stable-diffusion-webui

# Start with network access
./webui.sh --listen --api --enable-insecure-extension-access
```

Windows:
```bat
webui.bat --listen --api --enable-insecure-extension-access
```

### Option 2: Forge (faster, with known limitations)

[Stable Diffusion WebUI Forge](https://github.com/lllyasviel/stable-diffusion-webui-forge) is an optimized A1111 fork with significantly faster generation.

```bash
git clone https://github.com/lllyasviel/stable-diffusion-webui-forge.git
cd stable-diffusion-webui-forge

# Start with network access
./webui.sh --listen --api --enable-insecure-extension-access
```

Windows:
```bat
webui.bat --listen --api --enable-insecure-extension-access
```

> **Known Forge issue:** Hires Fix may trigger `argument of type 'NoneType' is not iterable` error. SD Studio automatically retries generation without Hires Fix and shows a warning. For full Hires Fix support, use A1111.

### Flags

| Flag | Description |
|------|-------------|
| `--listen` | Allow connections from other devices |
| `--api` | Enable REST API |
| `--enable-insecure-extension-access` | Allow extension installation |
| `--port 7860` | Port (default: 7860) |
| `--xformers` | Accelerate generation (NVIDIA) |
| `--medvram` | Save VRAM (4–6 GB) |
| `--lowvram` | Minimal VRAM usage (2–4 GB) |

### Verification

Open `http://localhost:7860/sdapi/v1/sd-models` — should return JSON with model list.

### Models

Download `.safetensors` into `models/Stable-diffusion/`:

- [Civitai](https://civitai.com/) — main model repository
- [HuggingFace](https://huggingface.co/models?pipeline_tag=text-to-image)

LoRA -> `models/Lora/`, VAE -> `models/VAE/`.

---

## Ollama

Local LLM server. SD Studio uses it for prompt generation and image analysis.

### Installation

**macOS / Linux:**
```bash
curl -fsSL https://ollama.com/install.sh | sh
```

**macOS (alternative):** Download from [ollama.com/download](https://ollama.com/download)

**Linux (manual):**
```bash
curl -L https://ollama.com/download/ollama-linux-amd64 -o /usr/local/bin/ollama
chmod +x /usr/local/bin/ollama
```

### Running

```bash
# Auto-start (macOS app / systemd)
ollama serve

# Run a model
ollama run llama3.2-vision
```

### Recommended Models

```bash
# For prompt generation (required)
ollama pull llama3.2

# For image analysis (vision model required)
ollama pull llama3.2-vision
ollama pull llava:13b
ollama pull minicpm-v
```

### Network Access

Ollama listens on `localhost` by default. To allow remote connections:

```bash
OLLAMA_HOST=0.0.0.0:11434 ollama serve
```

Or set environment variable:
```bash
export OLLAMA_HOST=0.0.0.0:11434
```

systemd (`/etc/systemd/system/ollama.service`):
```ini
[Service]
Environment="OLLAMA_HOST=0.0.0.0:11434"
```

### Verification

```bash
curl http://localhost:11434/api/tags
```

In SD Studio -> Settings -> Connection: select **Ollama**, URL `http://localhost:11434`.

---

## LM Studio

GUI application for LLMs. Easier to set up than Ollama.

### Installation

Download from [lmstudio.ai](https://lmstudio.ai/) (macOS / Windows / Linux).

### Setup

1. Open LM Studio
2. Download a model (Search tab):
   - `Llama 3.2 3B Instruct` — for prompt generation
   - `Llama 3.2 11B Vision Instruct` — for image analysis
3. Go to **Local Server** tab (arrow icon on the left panel)
4. Select a model from the dropdown
5. Click **Start Server**
6. Default port: `1234`

### Network Access

In LM Studio: Settings -> Advanced -> **Enable CORS** and set server host to `0.0.0.0`.

### Verification

```bash
curl http://localhost:1234/v1/models
```

In SD Studio -> Settings -> Connection: select **LM Studio**, URL `http://localhost:1234`.

---

## llama.cpp

Minimal C++ LLM server. Good for low-end machines without Python.

### Installation

**Build from source:**
```bash
git clone https://github.com/ggerganov/llama.cpp
cd llama.cpp

# With CUDA
cmake -B build -DGGML_CUDA=ON && cmake --build build --config Release

# CPU only
cmake -B build && cmake --build build --config Release
```

**macOS (Homebrew):**
```bash
brew install llama.cpp
```

### Running the Server

```bash
# CPU
./llama-server -m model.gguf --host 0.0.0.0 --port 8081 -c 4096

# With GPU (CUDA)
./llama-server -m model.gguf --host 0.0.0.0 --port 8081 -ngl 99 -c 4096
```

| Parameter | Description |
|-----------|-------------|
| `-m` | Path to .gguf model file |
| `--host` | Bind address (`0.0.0.0` for network) |
| `--port` | Port number |
| `-ngl` | GPU layers (99 = all) |
| `-c` | Context size |

### Models

Download `.gguf` files from [HuggingFace](https://huggingface.co/models?search=gguf):

- [Llama 3.2 3B](https://huggingface.co/bartowski/Llama-3.2-3B-Instruct-GGUF)
- [Llama 3.2 Vision](https://huggingface.co/bartowski/Llama-3.2-11B-Vision-Instruct-GGUF) — for analysis

### Verification

```bash
curl http://localhost:8081/v1/models
```

In SD Studio -> Settings -> Connection: select **llama.cpp**, URL `http://localhost:8081`.

> **Note:** llama.cpp does not support model switching via API. The model is set at server startup.

---

## Rembg

AI background removal. Used for clean character extraction in multi-character generation. Runs as a standalone HTTP service.

### Installation

**CPU:**
```bash
pip install "rembg[cli]"
```

**GPU (NVIDIA CUDA):**
```bash
pip install "rembg[gpu,cli]"
```

> Requires Python 3.10+. For GPU: CUDA Toolkit + cuDNN.

**Windows (if `rembg` command not found):**
```bat
python -m rembg s --host 0.0.0.0 --port 7000
```

### Running the Server

```bash
rembg s --host 0.0.0.0 --port 7000 --log_level info
```

On first run, the model downloads automatically (~180 MB, saved to `~/.u2net/`).

### Models

Default model is `u2net`. Specify a different one:

```bash
rembg s --host 0.0.0.0 --port 7000 -m birefnet-general
```

| Model | Size | Quality | Speed |
|-------|------|---------|-------|
| `u2net` | 176 MB | Good | Medium |
| `u2netp` | 4 MB | Fair | Fast |
| `isnet-general-use` | 176 MB | Good | Medium |
| `birefnet-general` | 176 MB | Excellent | Slower |
| `birefnet-general-lite` | 88 MB | Good | Medium |
| `birefnet-portrait` | 176 MB | Excellent for portraits | Medium |
| `isnet-anime` | 176 MB | Excellent for anime | Medium |

### Verification

```bash
# Check API
curl http://localhost:7000/api

# Test background removal
curl -s -F file=@test.png http://localhost:7000/api/remove -o result.png
```

### SD Studio Configuration

Settings -> Rembg -> enter URL (e.g. `http://192.168.1.100:7000`) -> **Test** -> **Save**.

If rembg is not configured, SD Studio falls back to built-in Go-based background removal (lower quality, visible edge artifacts).

---

## SD Studio Server

A standalone service that automatically manages all AI components (SD WebUI, Ollama, Rembg) — installation, startup, health monitoring, GPU optimization, and model management. Ideal for headless or server deployments.

### Installation

```bash
cd server
go build -o sd-studio-server .
```

Or use Docker:
```bash
docker compose up --build
```

### First Run

```bash
./sd-studio-server --data ~/sd-studio-server
```

On first run, an interactive setup wizard will guide you through:
- Selecting which components to install (SD WebUI, Ollama, Rembg)
- Choosing data directory
- Configuring GPU backend (Forge / A1111)

### Configuration

Configuration is stored in `{data-dir}/server-config.yaml`:

```yaml
port: 8080
data_dir: ~/sd-studio-server
active_sd: forge
mdns: true
proxy:
  enabled: true
  gpu_slots: 1
  endpoints:
    sd:
      listen_addr: ":7860"
      target_url: "http://localhost:7860"
    ollama:
      listen_addr: ":11434"
      target_url: "http://localhost:11434"
```

### Running Modes

| Mode | Command | Description |
|------|---------|-------------|
| TUI (interactive) | `./sd-studio-server` | Terminal dashboard with service controls |
| Headless | `./sd-studio-server --headless` | Log output only, no TUI |
| Custom port | `./sd-studio-server --port 9090` | Override HTTP port |
| Custom config | `./sd-studio-server --config /path/to/config.yaml` | Use specific config file |

### GPU Proxy

When `proxy.enabled: true`, the server acts as a smart reverse proxy:
- **Priority queue** — studio requests get higher priority than external
- **GPU slot limiting** — only N concurrent requests share the GPU
- **VRAM cooldown** — waits for >= 50% free VRAM between jobs (prevents OOM)
- Per-endpoint configuration (separate proxy ports for SD and Ollama)

### API Endpoints

| Endpoint | Description |
|----------|-------------|
| `GET /` | Server info |
| `GET /api/health` | Service health status |
| `GET /api/processes` | Process status |
| `POST /api/processes/:name/start` | Start a service |
| `POST /api/processes/:name/stop` | Stop a service |
| `POST /api/processes/:name/restart` | Restart a service |
| `GET /api/gpu` | GPU info |
| `GET /api/models` | Available models |
| `POST /api/models/download` | Download a model |
| `DELETE /api/models/:name` | Delete a model |
| `GET /api/backends` | Available backends |
| `POST /api/backends/switch` | Switch backend |

### mDNS Discovery

When `mdns: true`, the server advertises itself as `_sd-studio._tcp` on the local network. Desktop apps can auto-discover the server.

### Verification

```bash
curl http://localhost:8080/
```

---

## Quick Reference

| Service | Default Port | SD Studio URL |
|---------|-------------|---------------|
| Stable Diffusion | 7860 | `http://localhost:7860` |
| Ollama | 11434 | `http://localhost:11434` |
| LM Studio | 1234 | `http://localhost:1234` |
| llama.cpp | 8081 | `http://localhost:8081` |
| Rembg | 7000 | `http://localhost:7000` |
| SD Studio Server | 8080 | `http://localhost:8080` |
