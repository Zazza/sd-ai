# SD Studio — Алгоритмы и потоки данных

## 1. Txt2Img — Генерация с нуля

```
Frontend                    app.go                     LLM              SD WebUI
   │                          │                          │                  │
   │  GenerateImage(params)   │                          │                  │
   │─────────────────────────>│                          │                  │
   │                          │  preset.Get(id)          │                  │
   │                          │──────────>               │                  │
   │                          │                          │                  │
   │                          │  GenerateSDPrompt()      │                  │
   │                          │─────────────────────────>│                  │
   │                          │  SD tags (string)        │                  │
   │                          │<─────────────────────────│                  │
   │                          │                          │                  │
   │                          │  sd.Txt2Img(req)         │                  │
   │                          │─────────────────────────────────────────────>│
   │                          │  image (base64)          │                  │
   │                          │<─────────────────────────────────────────────│
   │                          │                          │                  │
   │  GenerateImageResult     │                          │                  │
   │<─────────────────────────│                          │                  │
```

**LLM-генерация промпта** использует `sd_prompt_instruction` из settings как system prompt. LLM получает описание пользователя + базовый промпт пресета и генерирует SD-теги.

## 2. Generate From Image — Генерация из изображения

```
Frontend                    app.go                     LLM              SD WebUI
   │                          │                          │                  │
   │  GenerateFromImage()     │                          │                  │
   │─────────────────────────>│                          │                  │
   │                          │                          │                  │
   │                   ┌──── mode? ────┐                 │                  │
   │                   │              │                  │                  │
   │              txt2img/       inpaint/            remove              │
   │              img2img        (user mask)                              │
   │                   │              │                  │                  │
   │                   ▼              ▼                  ▼                  │
   │              AnalyzeImage   mask from       analyzeRemoveContext     │
   │              (LLM vision)   canvas          (red overlay → LLM)     │
   │                   │              │                  │                 │
   │                   └──────────────┴──────────────────┘                │
   │                          │                          │                 │
   │                          │  sd.Img2Img(req)         │                 │
   │                          │────────────────────────────────────────────>│
   │                          │  image (base64)          │                 │
   │                          │<───────────────────────────────────────────│
```

### Режимы GenerateFromImage

| Режим | Маска | Промпт | LLM шаг |
|-------|-------|--------|---------|
| `txt2img` | — | tags + preset | AnalyzeImage (vision) |
| `img2img` | — | tags + preset | AnalyzeImage (vision) |
| `inpaint` | User canvas | tags + preset | AnalyzeImage (vision) |
| `remove` | User canvas | auto (background) | analyzeRemoveContext (vision + red overlay) |

### Smart Remove (remove mode)
1. Пользователь рисует маску на canvas
2. Backend: красный полупрозрачный overlay по маске поверх оригинала
3. LLM vision анализирует overlay → возвращает SD-теги описания фона
4. SD inpaint с авто-промптом (без пресета)

### Mask Processing (inpaint/remove)
Маска обрабатывается на frontend перед отправкой в SD:

```
Binary mask (user-drawn)
       │
       ▼
  Dilation (maskPadding px)
  blur(maskPadding) → threshold (alpha > 0 → white)
  Расширяет маску за пределы нарисованного
       │
       ▼
  Feathering (maskFeather px)
  blur(maskFeather) → soft gradient edges
  Плавный переход на границах маски
       │
       ▼
  PNG base64 → SD WebUI inpaint
```

Параметры по умолчанию: padding=8px, feather=8px. Настраиваются слайдерами в UI.

## 3. Multi-Pass — Компоновка персонажей

```
Frontend                    app.go                    LLM           Compositor      SD WebUI
   │                          │                         │              │              │
   │  DecomposeScene()        │                         │              │              │
   │─────────────────────────>│                         │              │              │
   │                          │  LLM: decompose         │              │              │
   │                          │────────────────────────>│              │              │
   │                          │  Scene JSON             │              │              │
   │                          │<────────────────────────│              │              │
   │  Scene (user edits)      │                         │              │              │
   │<─────────────────────────│                         │              │              │
   │                          │                         │              │              │
   │  GenerateMultiPass()     │                         │              │              │
   │─────────────────────────>│                         │              │              │
   │                          │                         │              │              │
   │                          │     GenerateScene()     │              │              │
   │                          │────────────────────────────────────────>│              │
   │                          │                         │              │              │
   │                          │                         │     Pass 1:  │  txt2img     │
   │                          │                         │  background  │─────────────>│
   │                          │                         │              │<─────────────│
   │                          │                         │              │              │
   │                          │                         │  Pass 2-N:   │  txt2img     │
   │                          │                         │  characters  │─────────────>│
   │                          │                         │  (rembg)     │<─────────────│
   │                          │                         │              │              │
   │                          │                         │  Composite:  │              │
   │                          │                         │  bg + chars  │              │
   │                          │                         │              │              │
   │  MultiPassResult         │                         │              │              │
   │<─────────────────────────│<───────────────────────────────────────│              │
```

### Алгоритм компоновки
1. LLM декомпозирует сцену: background + N персонажей (max 10)
2. Пользователь редактирует позиции/промпты в SceneEditor
3. Генерация по проходам:
   - Pass 1: background (txt2img)
   - Pass 2..N: каждый персонаж (txt2img → rembg удаление фона)
4. Composite: `draw.Draw` накладывает персонажей на background по позициям
5. Размеры: 64-2048, кратные 64

## 4. SD Client — Retry с Backoff

```
doPost(url, body)
    │
    ├─ Attempt 1: POST → status 500 → sleep(2s)
    ├─ Attempt 2: POST → timeout   → sleep(4s)
    ├─ Attempt 3: POST → status 500 → return error + SD response body
    │
    └─ Если успех (< 500) → decode JSON → return Txt2ImgResponse
```

**Retry conditions:**
- HTTP 500, 502, 503, 504
- Network errors: `*url.Error` (timeout, connection refused, EOF)
- NOT retried: 4xx, 200 с `result.Error`

**Параметры:**
- Max attempts: 3
- Initial delay: 2s
- Multiplier: 2x (2s → 4s)
- Только для `Txt2Img` и `Img2Img`

## 5. Preset Resolution

Пресет содержит все параметры генерации. Логика резолва (одинаковая для всех режимов):

```
1. Загрузить пресет по ID
2. Извлечь: Prompt, NegativePrompt, Sampler, Steps, CfgScale,
   Width, Height, Seed, ClipSkip, ModelName, VAE, Loras
3. Prompt = preset.Prompt + ", " + generated tags
4. NegativePrompt = preset.NegativePrompt + ", " + extra negatives
5. LoRAs: JSON → <lora:name:weight> теги в конец prompt
6. SamplerName = Sampler + " " + ScheduleType (если есть)
7. Установить модель/VAE через SD API (SetModel/SetVAE)
8. Дефолты если без пресета:
   - Sampler: "Euler a"
   - Steps: 20
   - CfgScale: 7
   - Width/Height: из изображения или 512x512
```

## 6. LLM Integration

### Бэкенды

| Бэкенд | KeepAlive | Options | Особенности |
|--------|-----------|---------|-------------|
| Ollama | Да (настройка) | num_ctx, num_gpu | Автовыгрузка моделей |
| LM Studio | Нет | — | — |

### Режимы LLM

| Режим | Модель (setting) | Назначение |
|-------|------------------|-----------|
| `generate` | `llm_generate_model` | Генерация SD-промптов |
| `analyze` | `llm_analyze_model` | Vision анализ изображений |

### Промпт-инженерия

**SD Prompt Generation:**
- System: `sd_prompt_instruction` из settings (пользователь редактирует в UI)
- User: `BASE POSITIVE PROMPT: ... \n BASE NEGATIVE PROMPT: ... \n USER DESCRIPTION: ...`
- Output: JSON `{prompt, negative_prompt}` или plain text tags

**Image Analysis (AnalyzeImage):**
- System: `DefaultAnalyzeSystemPrompt` / `DefaultAnalyzeChainPrompts`
- Mode: `quick` (один запрос) или `deep` (цепочка промптов)
- Output: comma-separated SD tags

**Remove Context (analyzeRemoveContext):**
- Red overlay на маску → LLM vision
- Output: SD-теги описания окружения

## 7. Sessions

Механизм хранения истории генераций:

```
sessions                     session_items
├── id, name                 ├── id, session_id
├── created_at               ├── file_name, thumb_name
└── updated_at               ├── source (preset/compound/remove/...)
                              ├── prompt, negative_prompt
                              ├── model, sampler, steps, cfg, seed
                              └── created_at
```

- Изображения хранятся в `data/sessions/{session_id}/`
- Thumbnails в `data/sessions/{session_id}/thumb/`
- Frontend: сессия → grid изображений → zoom через ImageViewer
