# SD Studio — Service Setup Guide

## Contents

- [Stable Diffusion WebUI](#stable-diffusion-webui)
- [Ollama](#ollama)
- [LM Studio](#lm-studio)
- [llama.cpp](#llamacpp)
- [Rembg](#rembg)

---

## Stable Diffusion WebUI

Image generation engine. SD Studio connects via REST API.

### Installation (Python + Git)

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

LoRA → `models/Lora/`, VAE → `models/VAE/`.

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

In SD Studio → Settings → Connection: select **Ollama**, URL `http://localhost:11434`.

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

In LM Studio: Settings → Advanced → **Enable CORS** and set server host to `0.0.0.0`.

### Verification

```bash
curl http://localhost:1234/v1/models
```

In SD Studio → Settings → Connection: select **LM Studio**, URL `http://localhost:1234`.

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

In SD Studio → Settings → Connection: select **llama.cpp**, URL `http://localhost:8081`.

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

Settings → Rembg → enter URL (e.g. `http://192.168.1.100:7000`) → **Test** → **Save**.

If rembg is not configured, SD Studio falls back to built-in Go-based background removal (lower quality, visible edge artifacts).

---

## Quick Reference

| Service | Default Port | SD Studio URL |
|---------|-------------|---------------|
| Stable Diffusion | 7860 | `http://localhost:7860` |
| Ollama | 11434 | `http://localhost:11434` |
| LM Studio | 1234 | `http://localhost:1234` |
| llama.cpp | 8081 | `http://localhost:8081` |
| Rembg | 7000 | `http://localhost:7000` |
