[English](algorithms-en.md) | [Русский](algorithms-ru.md)

# SD Studio — Algorithms and Data Flows

## 1. Txt2Img — Generation from Scratch

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

**LLM prompt generation** uses `sd_prompt_instruction` from settings as the system prompt. The LLM receives the user's description + the preset's base prompt and generates SD tags.

## 2. Generate From Image — Generation from an Image

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

### GenerateFromImage Modes

| Mode | Mask | Prompt | LLM Step |
|------|------|--------|----------|
| `txt2img` | — | tags + preset | AnalyzeImage (vision) |
| `img2img` | — | tags + preset | AnalyzeImage (vision) |
| `inpaint` | User canvas | tags + preset | AnalyzeImage (vision) |
| `remove` | User canvas | auto (background) | analyzeRemoveContext (vision + red overlay) |

### Smart Remove (remove mode)
1. The user draws a mask on the canvas
2. Backend: a semi-transparent red overlay is applied on top of the original image along the mask
3. LLM vision analyzes the overlay and returns SD tags describing the background
4. SD inpaint runs with an auto-generated prompt (without a preset)

### Mask Processing (inpaint/remove)
The mask is processed on the frontend before being sent to SD:

```
Binary mask (user-drawn)
       │
       ▼
  Dilation (maskPadding px)
  blur(maskPadding) → threshold (alpha > 0 → white)
  Expands the mask beyond the drawn area
       │
       ▼
  Feathering (maskFeather px)
  blur(maskFeather) → soft gradient edges
  Smooth transition at the mask boundaries
       │
       ▼
  PNG base64 → SD WebUI inpaint
```

Default parameters: padding=8px, feather=8px. Configurable via sliders in the UI.

## 3. Multi-Pass — Character Composition

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

### Composition Algorithm
1. LLM decomposes the scene: background + N characters (max 10)
2. The user edits positions/prompts in SceneEditor
3. Pass-by-pass generation:
   - Pass 1: background (txt2img)
   - Pass 2..N: each character (txt2img → rembg background removal)
4. Composite: `draw.Draw` overlays characters onto the background at their positions
5. Dimensions: 64-2048, multiples of 64

## 4. SD Client — Retry with Backoff

```
doPost(url, body)
    │
    ├─ Attempt 1: POST → status 500 → sleep(2s)
    ├─ Attempt 2: POST → timeout   → sleep(4s)
    ├─ Attempt 3: POST → status 500 → return error + SD response body
    │
    └─ On success (< 500) → decode JSON → return Txt2ImgResponse
```

**Retry conditions:**
- HTTP 500, 502, 503, 504
- Network errors: `*url.Error` (timeout, connection refused, EOF)
- NOT retried: 4xx, 200 with `result.Error`

**Parameters:**
- Max attempts: 3
- Initial delay: 2s
- Multiplier: 2x (2s → 4s)
- Only for `Txt2Img` and `Img2Img`

## 5. Preset Resolution

A preset contains all generation parameters. The resolution logic (the same for all modes):

```
1. Load preset by ID
2. Extract: Prompt, NegativePrompt, Sampler, Steps, CfgScale,
   Width, Height, Seed, ClipSkip, ModelName, VAE, Loras
3. Prompt = preset.Prompt + ", " + generated tags
4. NegativePrompt = preset.NegativePrompt + ", " + extra negatives
5. LoRAs: JSON → <lora:name:weight> tags appended to prompt
6. SamplerName = Sampler + " " + ScheduleType (if present)
7. Set model/VAE via SD API (SetModel/SetVAE)
8. Defaults when no preset:
   - Sampler: "Euler a"
   - Steps: 20
   - CfgScale: 7
   - Width/Height: from image or 512x512
```

## 6. LLM Integration

### Backends

| Backend | KeepAlive | Options | Features |
|---------|-----------|---------|----------|
| Ollama | Yes (configurable) | num_ctx, num_gpu | Automatic model unloading |
| LM Studio | No | — | — |

### LLM Modes

| Mode | Model (setting) | Purpose |
|------|-----------------|---------|
| `generate` | `llm_generate_model` | SD prompt generation |
| `analyze` | `llm_analyze_model` | Image vision analysis |

### Prompt Engineering

**SD Prompt Generation:**
- System: `sd_prompt_instruction` from settings (user-editable in the UI)
- User: `BASE POSITIVE PROMPT: ... \n BASE NEGATIVE PROMPT: ... \n USER DESCRIPTION: ...`
- Output: JSON `{prompt, negative_prompt}` or plain text tags

**Image Analysis (AnalyzeImage):**
- System: `DefaultAnalyzeSystemPrompt` / `DefaultAnalyzeChainPrompts`
- Mode: `quick` (single request) or `deep` (prompt chain)
- Output: comma-separated SD tags

**Remove Context (analyzeRemoveContext):**
- Red overlay on mask → LLM vision
- Output: SD tags describing the surroundings

## 7. Sessions

A mechanism for storing generation history:

```
sessions                     session_items
├── id, name                 ├── id, session_id
├── created_at               ├── file_name, thumb_name
└── updated_at               ├── source (preset/compound/remove/...)
                             ├── prompt, negative_prompt
                             ├── model, sampler, steps, cfg, seed
                             ├── is_active (only 1 per session)
                             └── created_at
```

- `AddSessionItem` deactivates all previous items in the session before inserting a new one
- `SetActiveItem(id, sessionID)` switches the active item using `CASE WHEN`
- Images are stored in `data/sessions/{session_id}/`
- Thumbnails in `data/sessions/{session_id}/thumb/`
- Frontend: session → image grid → zoom via ImageViewer
