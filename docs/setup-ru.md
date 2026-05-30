[English](setup-en.md) | [Русский](setup-ru.md)

# SD Studio — Руководство по настройке сервисов

## Содержание

- [Stable Diffusion WebUI](#stable-diffusion-webui)
- [Ollama](#ollama)
- [LM Studio](#lm-studio)
- [llama.cpp](#llamacpp)
- [Rembg](#rembg)
- [SD Studio Server](#sd-studio-server)

---

## Stable Diffusion WebUI

Движок генерации изображений. SD Studio подключается через REST API. Поддерживаются два варианта: **A1111** (стандартный) и **Forge** (ускоренный).

### Вариант 1: A1111 (стандартный, рекомендуется)

```bash
git clone https://github.com/AUTOMATIC1111/stable-diffusion-webui.git
cd stable-diffusion-webui

# Запуск с сетевым доступом
./webui.sh --listen --api --enable-insecure-extension-access
```

Windows:
```bat
webui.bat --listen --api --enable-insecure-extension-access
```

### Вариант 2: Forge (быстрее, с известными ограничениями)

[Stable Diffusion WebUI Forge](https://github.com/lllyasviel/stable-diffusion-webui-forge) — оптимизированный форк A1111, работает значительно быстрее.

```bash
git clone https://github.com/lllyasviel/stable-diffusion-webui-forge.git
cd stable-diffusion-webui-forge

# Запуск с сетевым доступом
./webui.sh --listen --api --enable-insecure-extension-access
```

Windows:
```bat
webui.bat --listen --api --enable-insecure-extension-access
```

> **Известная проблема Forge:** Hires Fix может вызывать ошибку `argument of type 'NoneType' is not iterable`. SD Studio автоматически повторит генерацию без Hires Fix и покажет предупреждение. Для полноценной работы Hires Fix используйте A1111.

### Флаги

| Флаг | Описание |
|------|----------|
| `--listen` | Разрешить подключения с других устройств |
| `--api` | Включить REST API |
| `--enable-insecure-extension-access` | Разрешить установку расширений |
| `--port 7860` | Порт (по умолчанию: 7860) |
| `--xformers` | Ускорить генерацию (NVIDIA) |
| `--medvram` | Экономить VRAM (4–6 ГБ) |
| `--lowvram` | Минимальное использование VRAM (2–4 ГБ) |

### Проверка

Откройте `http://localhost:7860/sdapi/v1/sd-models` — должен вернуться JSON со списком моделей.

### Модели

Скачайте `.safetensors` в `models/Stable-diffusion/`:

- [Civitai](https://civitai.com/) — основной репозиторий моделей
- [HuggingFace](https://huggingface.co/models?pipeline_tag=text-to-image)

LoRA -> `models/Lora/`, VAE -> `models/VAE/`.

---

## Ollama

Локальный LLM-сервер. SD Studio использует его для генерации промптов и анализа изображений.

### Установка

**macOS / Linux:**
```bash
curl -fsSL https://ollama.com/install.sh | sh
```

**macOS (альтернатива):** Скачайте с [ollama.com/download](https://ollama.com/download)

**Linux (вручную):**
```bash
curl -L https://ollama.com/download/ollama-linux-amd64 -o /usr/local/bin/ollama
chmod +x /usr/local/bin/ollama
```

### Запуск

```bash
# Автозапуск (приложение macOS / systemd)
ollama serve

# Запуск модели
ollama run llama3.2-vision
```

### Рекомендуемые модели

```bash
# Для генерации промптов (обязательно)
ollama pull llama3.2

# Для анализа изображений (нужна vision-модель)
ollama pull llama3.2-vision
ollama pull llava:13b
ollama pull minicpm-v
```

### Сетевой доступ

Ollama по умолчанию слушает `localhost`. Чтобы разрешить удалённые подключения:

```bash
OLLAMA_HOST=0.0.0.0:11434 ollama serve
```

Или задайте переменную окружения:
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

В SD Studio -> Настройки -> Подключение: выберите **Ollama**, URL `http://localhost:11434`.

---

## LM Studio

GUI-приложение для LLM. Проще в настройке, чем Ollama.

### Установка

Скачайте с [lmstudio.ai](https://lmstudio.ai/) (macOS / Windows / Linux).

### Настройка

1. Откройте LM Studio
2. Скачайте модель (вкладка Search):
   - `Llama 3.2 3B Instruct` — для генерации промптов
   - `Llama 3.2 11B Vision Instruct` — для анализа изображений
3. Перейдите на вкладку **Local Server** (значок стрелки на левой панели)
4. Выберите модель из выпадающего списка
5. Нажмите **Start Server**
6. Порт по умолчанию: `1234`

### Сетевой доступ

В LM Studio: Settings -> Advanced -> **Enable CORS** и укажите хост сервера `0.0.0.0`.

### Проверка

```bash
curl http://localhost:1234/v1/models
```

В SD Studio -> Настройки -> Подключение: выберите **LM Studio**, URL `http://localhost:1234`.

---

## llama.cpp

Минимальный LLM-сервер на C++. Подходит для слабых машин без Python.

### Установка

**Сборка из исходников:**
```bash
git clone https://github.com/ggerganov/llama.cpp
cd llama.cpp

# С поддержкой CUDA
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
| `-m` | Путь к файлу модели .gguf |
| `--host` | Адрес привязки (`0.0.0.0` для сетевого доступа) |
| `--port` | Номер порта |
| `-ngl` | Слоёв на GPU (99 = все) |
| `-c` | Размер контекста |

### Модели

Скачайте файлы `.gguf` с [HuggingFace](https://huggingface.co/models?search=gguf):

- [Llama 3.2 3B](https://huggingface.co/bartowski/Llama-3.2-3B-Instruct-GGUF)
- [Llama 3.2 Vision](https://huggingface.co/bartowski/Llama-3.2-11B-Vision-Instruct-GGUF) — для анализа

### Проверка

```bash
curl http://localhost:8081/v1/models
```

В SD Studio -> Настройки -> Подключение: выберите **llama.cpp**, URL `http://localhost:8081`.

> **Примечание:** llama.cpp не поддерживает переключение моделей через API. Модель задаётся при запуске сервера.

---

## Rembg

AI-удаление фона. Используется для чистой вырезки персонажей при генерации нескольких персонажей. Работает как самостоятельный HTTP-сервис.

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

**Windows (если команда `rembg` не найдена):**
```bat
python -m rembg s --host 0.0.0.0 --port 7000
```

### Запуск сервера

```bash
rembg s --host 0.0.0.0 --port 7000 --log_level info
```

При первом запуске модель скачивается автоматически (~180 МБ, сохраняется в `~/.u2net/`).

### Модели

Модель по умолчанию — `u2net`. Укажите другую:

```bash
rembg s --host 0.0.0.0 --port 7000 -m birefnet-general
```

| Модель | Размер | Качество | Скорость |
|--------|--------|----------|----------|
| `u2net` | 176 МБ | Хорошее | Средняя |
| `u2netp` | 4 МБ | Приемлемое | Быстрая |
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

Настройки -> Rembg -> введите URL (например, `http://192.168.1.100:7000`) -> **Тест** -> **Сохранить**.

Если rembg не настроен, SD Studio использует встроенное удаление фона на Go (более низкое качество, заметные артефакты по краям).

---

## SD Studio Server

Автономный сервис для автоматического управления всеми AI-компонентами (SD WebUI, Ollama, Rembg) — установка, запуск, мониторинг, оптимизация GPU и управление моделями. Идеально для headless-развертываний и серверов.

### Установка

```bash
cd server
go build -o sd-studio-server .
```

Или через Docker:
```bash
docker compose up --build
```

### Первый запуск

```bash
./sd-studio-server --data ~/sd-studio-server
```

При первом запуске интерактивный мастер настройки проведёт через:
- Выбор компонентов для установки (SD WebUI, Ollama, Rembg)
- Выбор директории данных
- Настройку GPU-бэкенда (Forge / A1111)

### Конфигурация

Конфигурация хранится в `{data-dir}/server-config.yaml`:

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

### Режимы запуска

| Режим | Команда | Описание |
|-------|---------|----------|
| TUI (интерактивный) | `./sd-studio-server` | Терминальный дашборд с управлением сервисами |
| Headless | `./sd-studio-server --headless` | Только вывод логов, без TUI |
| Кастомный порт | `./sd-studio-server --port 9090` | Переопределить HTTP-порт |
| Кастомный конфиг | `./sd-studio-server --config /path/to/config.yaml` | Использовать указанный файл конфигурации |

### GPU Proxy

При `proxy.enabled: true` сервер работает как умный обратный прокси:
- **Очередь с приоритетом** — запросы от студии получают более высокий приоритет
- **Ограничение GPU-слотов** — только N параллельных запросов разделяют GPU
- **Охлаждение VRAM** — ожидание >= 50% свободной VRAM между задачами (предотвращает OOM)
- Конфигурация на уровне эндпоинтов (отдельные порты прокси для SD и Ollama)

### API-эндпоинты

| Эндпоинт | Описание |
|-----------|----------|
| `GET /` | Информация о сервере |
| `GET /api/health` | Статус здоровья сервисов |
| `GET /api/processes` | Статус процессов |
| `POST /api/processes/:name/start` | Запустить сервис |
| `POST /api/processes/:name/stop` | Остановить сервис |
| `POST /api/processes/:name/restart` | Перезапустить сервис |
| `GET /api/gpu` | Информация о GPU |
| `GET /api/models` | Доступные модели |
| `POST /api/models/download` | Скачать модель |
| `DELETE /api/models/:name` | Удалить модель |
| `GET /api/backends` | Доступные бэкенды |
| `POST /api/backends/switch` | Переключить бэкенд |

### Обнаружение через mDNS

При `mdns: true` сервер анонсирует себя как `_sd-studio._tcp` в локальной сети. Десктопные приложения могут автоматически обнаруживать сервер.

### Проверка

```bash
curl http://localhost:8080/
```

---

## Краткая справка

| Сервис | Порт по умолчанию | URL в SD Studio |
|--------|-------------------|-----------------|
| Stable Diffusion | 7860 | `http://localhost:7860` |
| Ollama | 11434 | `http://localhost:11434` |
| LM Studio | 1234 | `http://localhost:1234` |
| llama.cpp | 8081 | `http://localhost:8081` |
| Rembg | 7000 | `http://localhost:7000` |
| SD Studio Server | 8080 | `http://localhost:8080` |
