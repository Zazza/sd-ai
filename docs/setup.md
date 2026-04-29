# SD Studio — Установка сервисов

## Содержание

- [Stable Diffusion WebUI](#stable-diffusion-webui)
- [Ollama](#ollama)
- [LM Studio](#lm-studio)
- [llama.cpp](#llamacpp)
- [Rembg](#rembg)

---

## Stable Diffusion WebUI

Генерация изображений. SD Studio подключается по API.

### Установка (Python + Git)

```bash
# Клонирование
git clone https://github.com/AUTOMATIC1111/stable-diffusion-webui.git
cd stable-diffusion-webui

# Запуск с доступом по сети
./webui.sh --listen --api --enable-insecure-extension-access
```

Windows:
```bat
webui.bat --listen --api --enable-insecure-extension-access
```

### Флаги

| Флаг | Описание |
|------|----------|
| `--listen` | Доступ с других устройств в сети |
| `--api` | Включить REST API |
| `--enable-insecure-extension-access` | Разрешить установку расширений |
| `--port 7860` | Порт (по умолчанию 7860) |
| `--xformers` | Ускорение генерации (NVIDIA) |
| `--medvram` | Экономия VRAM (4–6 ГБ) |
| `--lowvram` | Минимальное потребление VRAM (2–4 ГБ) |

### Проверка

Открыть `http://localhost:7860/sdapi/v1/sd-models` — должен вернуть JSON со списком моделей.

### Модели

Скачать `.safetensors` в папку `models/Stable-diffusion/`:

- [Civitai](https://civitai.com/) — основное хранилище моделей
- [HuggingFace](https://huggingface.co/models?pipeline_tag=text-to-image)

LoRA — в `models/Lora/`, VAE — в `models/VAE/`.

---

## Ollama

Локальный LLM-сервер. SD Studio использует его для генерации промптов и анализа изображений.

### Установка

**macOS / Linux:**
```bash
curl -fsSL https://ollama.com/install.sh | sh
```

**macOS (альтернатива):** Скачать с [ollama.com/download](https://ollama.com/download)

**Linux (ручная установка):**
```bash
curl -L https://ollama.com/download/ollama-linux-amd64 -o /usr/local/bin/ollama
chmod +x /usr/local/bin/ollama
```

### Запуск

```bash
# Автозапуск (macOS app / systemd)
ollama serve

# Запуск модели
ollama run llama3.2-vision
```

### Модели для SD Studio

```bash
# Для генерации промптов (обязательно)
ollama pull llama3.2

# Для анализа изображений (требуется vision-модель)
ollama pull llama3.2-vision
ollama pull llava:13b
ollama pull minicpm-v
```

### Доступ по сети

Ollama слушает только `localhost` по умолчанию. Для доступа с другого устройства:

```bash
# Linux/macOS
OLLAMA_HOST=0.0.0.0:11434 ollama serve
```

Или переменная окружения:
```bash
export OLLAMA_HOST=0.0.0.0:11434
```

systemd (`/etc/systemd/system/ollama.service`):
```ini
[Service]
Environment="OLLAMA_HOST=0.0.0.0:11434"
```

### Проверка

```bash
curl http://localhost:11434/api/tags
```

В SD Studio Settings → Connection: выбрать **Ollama**, URL `http://localhost:11434`.

---

## LM Studio

GUI-приложение для LLM. Проще в настройке, чем Ollama.

### Установка

Скачать с [lmstudio.ai](https://lmstudio.ai/) (macOS / Windows / Linux).

### Настройка

1. Открыть LM Studio
2. Скачать модель (вкладка Search):
   - `Llama 3.2 3B Instruct` — для генерации промптов
   - `Llama 3.2 11B Vision Instruct` — для анализа изображений
3. Перейти на вкладку **Local Server** (значок ➔ на левой панели)
4. Выбрать модель из выпадающего списка
5. Нажать **Start Server**
6. Порт по умолчанию: `1234`

### Доступ по сети

В LM Studio: Settings → Advanced → **Enable CORS** и в настройках сервера указать `0.0.0.0`.

### Проверка

```bash
curl http://localhost:1234/v1/models
```

В SD Studio Settings → Connection: выбрать **LM Studio**, URL `http://localhost:1234`.

---

## llama.cpp

Минимальный LLM-сервер на C++. Для слабых машин без Python.

### Установка

**Сборка из исходников:**
```bash
git clone https://github.com/ggerganov/llama.cpp
cd llama.cpp

# С CUDA
cmake -B build -DGGML_CUDA=ON && cmake --build build --config Release

# Только CPU
cmake -B build && cmake --build build --config Release
```

**macOS (Homebrew):**
```bash
brew install llama.cpp
```

### Запуск сервера

```bash
# CPU
./llama-server -m model.gguf --host 0.0.0.0 --port 8081 -c 4096

# С GPU (CUDA)
./llama-server -m model.gguf --host 0.0.0.0 --port 8081 -ngl 99 -c 4096
```

| Параметр | Описание |
|----------|----------|
| `-m` | Путь к .gguf файлу модели |
| `--host` | Адрес привязки (`0.0.0.0` для сети) |
| `--port` | Порт |
| `-ngl` | Количество слоёв на GPU (99 = все) |
| `-c` | Размер контекста |

### Где брать модели

Скачать `.gguf` файлы с [HuggingFace](https://huggingface.co/models?search=gguf):

- [Llama 3.2 3B](https://huggingface.co/bartowski/Llama-3.2-3B-Instruct-GGUF)
- [Llama 3.2 Vision](https://huggingface.co/bartowski/Llama-3.2-11B-Vision-Instruct-GGUF) — для анализа

### Проверка

```bash
curl http://localhost:8081/v1/models
```

В SD Studio Settings → Connection: выбрать **llama.cpp**, URL `http://localhost:8081`.

> **Примечание:** llama.cpp не поддерживает смену модели через API. Модель задаётся при запуске сервера.

---

## Rembg

AI-удаление фона. Используется для чистого вырезания персонажей при multi-character генерации. Работает как отдельный HTTP-сервис.

### Установка

**CPU:**
```bash
pip install "rembg[cli]"
```

**GPU (NVIDIA CUDA):**
```bash
pip install "rembg[gpu,cli]"
```

> Требуется Python 3.10+. Для GPU: CUDA Toolkit + cuDNN.

### Запуск сервера

```bash
rembg s --host 0.0.0.0 --port 7000 --log_level info
```

При первом запуске скачается модель (~180 МБ, сохраняется в `~/.u2net/`).

### Модели

По умолчанию используется `u2net`. Можно указать другую:

```bash
rembg s --host 0.0.0.0 --port 7000 -m birefnet-general
```

| Модель | Размер | Качество | Скорость |
|--------|--------|----------|----------|
| `u2net` | 176 МБ | Хорошее | Средняя |
| `u2netp` | 4 МБ | Среднее | Быстрая |
| `isnet-general-use` | 176 МБ | Хорошее | Средняя |
| `birefnet-general` | 176 МБ | Отличное | Медленнее |
| `birefnet-general-lite` | 88 МБ | Хорошее | Средняя |
| `birefnet-portrait` | 176 МБ | Отличное для портретов | Средняя |
| `isnet-anime` | 176 МБ | Отличное для аниме | Средняя |

### Проверка

```bash
# Проверка API
curl http://localhost:7000/api

# Тест удаления фона
curl -s -F file=@test.png http://localhost:7000/api/remove -o result.png
```

### Настройка в SD Studio

Settings → Rembg → ввести URL (`http://192.168.1.100:7000`) → **Test** → **Save**.

Если rembg не настроен, SD Studio использует встроенный Go-алгоритм удаления фона (худшее качество, артефакты на краях).

---

## Быстрая сводка по подключению

| Сервис | Порт по умолчанию | URL для SD Studio |
|--------|-------------------|-------------------|
| Stable Diffusion | 7860 | `http://localhost:7860` |
| Ollama | 11434 | `http://localhost:11434` |
| LM Studio | 1234 | `http://localhost:1234` |
| llama.cpp | 8081 | `http://localhost:8081` |
| Rembg | 7000 | `http://localhost:7000` |
