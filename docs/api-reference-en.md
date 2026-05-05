[English](api-reference-en.md) | [Русский](api-reference-ru.md)

# SD Studio — API Reference (Wails Bindings)

## Common Types

### GenerateImageResult
Used by all generation methods as the result.

```typescript
interface GenerateImageResult {
  image: string                    // base64 PNG
  parameters: any                  // JSON от SD WebUI
  info: any                        // JSON с деталями генерации
  is_preview: boolean
  effective_prompt: string         // итоговый positive prompt
  effective_negative_prompt: string // итоговый negative prompt
}
```

### ServiceStatus
```typescript
interface ServiceStatus {
  sd: boolean                      // SD WebUI доступен
  llm: boolean                     // LLM API доступен
  rembg: boolean                   // rembg доступен
}
```

---

## Generation

### GenerateImage
Generate an image from scratch using a preset.

```typescript
api.generateImage(presetId: number, extraPrompt?: string, extraNegativePrompt?: string)
  → Promise<GenerateImageResult>
```

**Parameters:**
| Parameter | Type | Required | Description |
|-----------|------|----------|-------------|
| presetId | number | Yes | Preset ID |
| extraPrompt | string | No | Additional prompt |
| extraNegativePrompt | string | No | Additional negative prompt |

**Flow:** LLM generates prompt → SD txt2img → result

---

### GenerateFromImage
Generation based on an uploaded image.

```typescript
api.generateFromImage(params: GenerateFromImageParams)
  → Promise<GenerateImageResult>
```

```typescript
interface GenerateFromImageParams {
  image_base64: string              // Required
  mode: "txt2img" | "img2img" | "inpaint"
  gen_mode: "preset" | "compound"
  preset_id: number
  compound_preset_id: number
  denoising_strength: number        // 0.05 - 1.0, default 0.5
  tags: string                      // LLM tags or manual
  extra_negative_prompt: string
  mask_base64: string               // Required for inpaint/remove
  mask_blur: number                 // default 4
  inpaint_fill: 0 | 1 | 2          // 0=Fill, 1=Original, 2=Latent Noise
  inpaint_full_res: boolean
  remove_object: boolean            // true = Smart Remove mode
}
```

**Events:**
```javascript
EventsOn("analyze:step", (step, total) => { ... })  // Deep analysis progress
EventsOn("remove:stage", (stage) => { ... })         // "analyzing" | "generating"
```

---

### GenerateSDPrompt
Generate an SD prompt via LLM.

```typescript
api.generateSdPrompt(params: GenerateSDPromptParams)
  → Promise<GenerateSDPromptResult>
```

```typescript
interface GenerateSDPromptParams {
  preset_id: number
  description: string
  negative: string
}

interface GenerateSDPromptResult {
  prompt: string
  negative_prompt: string
}
```

---

### UpscaleImage / UpscalePreview
Upscale an image.

```typescript
api.upscaleImage(imageBase64: string, genInfo: any, presetId: number)
  → Promise<GenerateImageResult>

api.upscalePreview(previewImageBase64: string, presetId: number, seed: number)
  → Promise<GenerateImageResult>
```

---

### BatchGenerate
Batch generation. Runs asynchronously, emits events.

```typescript
api.batchGenerate(params: BatchGenerateParams): Promise<void>
```

**Events:**
```javascript
EventsOn("batch:progress", (current, total, fileName) => { ... })
EventsOn("batch:done", () => { ... })
EventsOn("batch:error", (err) => { ... })
```

---

## Multi-Pass Composition

### DecomposeScene
LLM decomposes a scene description into background + characters.

```typescript
api.decomposeScene(params: DecomposeSceneParams)
  → Promise<Scene>
```

```typescript
interface DecomposeSceneParams {
  description: string
  preset_id: number
  width?: number
  height?: number
}

interface Scene {
  background_prompt: string
  negative_prompt: string
  characters: CharacterSlot[]
  width: number
  height: number
  preset_id: number
}

interface CharacterSlot {
  name: string
  prompt: string
  position: { x: number; y: number }    // 0.0 - 1.0
  scale: number                          // 0.1 - 2.0
}
```

### GenerateMultiPass
Generate a scene in multiple passes.

```typescript
api.generateMultiPass(scene: Scene)
  → Promise<MultiPassResult>
```

**Events:**
```javascript
EventsOn("multipass:progress", (data) => {
  // data.step: "background" | "character" | "compositing"
  // data.character: 1-based index
  // data.total: total characters
})
```

---

## Presets (CRUD)

```typescript
api.listPresets(): Promise<Preset[]>
api.listPresetsByType(type: string): Promise<Preset[]>
api.getPresetType(id: number): Promise<PresetType>
api.createPreset(data: Preset): Promise<Preset>
api.updatePreset(id: number, data: Preset): Promise<Preset>
api.deletePreset(id: number): Promise<void>

api.listPresetTypes(): Promise<PresetType[]>
api.createPresetType(data: PresetType): Promise<PresetType>
api.updatePresetType(data: Presetype): Promise<PresetType>
api.deletePresetType(id: number): Promise<void>
```

### Preset
```typescript
interface Preset {
  id: number
  name: string
  type_id: number
  preset_type: string
  prompt: string
  negative_prompt: string
  sampler: string
  schedule_type: string              // "auto", "karras", "exponential"
  steps: number
  cfg_scale: number
  width: number
  height: number
  model_name: string
  vae: string
  seed: number | null
  clip_skip: number | null
  loras: string                      // JSON: [{name, weight}]
}
```

### Compound Preset
```typescript
api.listCompoundPresets(): Promise<CompoundPreset[]>
api.createCompoundPreset(data: CompoundPreset): Promise<CompoundPreset>
api.updateCompoundPreset(data: CompoundPreset): Promise<CompoundPreset>
api.deleteCompoundPreset(id: number): Promise<void>
api.generateCompoundImage(params): Promise<GenerateImageResult>
api.testCompoundGenerate(params): Promise<TestGenerateResultItem[]>
```

---

## SD WebUI Proxy

```typescript
api.getModels(): Promise<SDModel[]>
api.getSamplers(): Promise<Sampler[]>
api.getSchedulers(): Promise<Scheduler[]>
api.getUpscalers(): Promise<Upscaler[]>
api.getVAEs(): Promise<VAE[]>
api.getLoRAs(): Promise<LoRA[]>
api.getLLMModels(): Promise<LLMModel[]>
```

---

## Images

```typescript
api.analyzeImage(imageBase64: string): Promise<string>  // returns SD tags
api.readImageFile(): Promise<string>                     // file dialog → base64
api.readClipboardImage(): Promise<string>                // clipboard → base64
api.saveImage(base64: string, defaultName: string): Promise<string>
api.getLastImage(): Promise<GenerateImageResult | null>
api.clearLastImage(): Promise<void>
api.setLastImage(base64: string): Promise<void>
```

---

## Settings

```typescript
api.getSettings(): Promise<Record<string, string>>
api.updateSettings(data: Record<string, string>): Promise<void>
```

**Settings keys:**
| Key | Description |
|-----|-------------|
| `llm_url` | LLM API URL |
| `sd_url` | SD WebUI URL |
| `llm_generate_model` | Model for prompt generation |
| `llm_analyze_model` | Model for vision analysis |
| `sd_prompt_instruction` | System prompt for SD prompt generation |
| `llm_backend` | `ollama` / `lmstudio` |
| `kids_mode` | `true` / `false` |

---

## Sessions

```typescript
api.createSession(name: string): Promise<SessionInfo>
api.listSessions(): Promise<SessionInfo[]>
api.switchSession(id: number): Promise<void>
api.renameSession(id: number, name: string): Promise<void>
api.deleteSession(id: number): Promise<void>
api.getSessionItems(): Promise<SessionItem[]>
api.getActiveSessionItem(): Promise<SessionItem | null>
api.setActiveSessionItem(id: number): Promise<void>
api.deleteSessionItem(id: number): Promise<void>
api.clearSession(): Promise<void>
api.getSessionImage(id: number): Promise<string>       // base64
api.getSessionThumb(id: number): Promise<string>       // base64
api.hasSessionItems(): Promise<boolean>
```

---

## File Browser

```typescript
api.browseDirectory(dirPath: string): Promise<FileEntry[]>
api.readFileAsBase64(filePath: string): Promise<string>
api.readThumbnail(filePath: string): Promise<string>
api.selectBrowserFolder(): Promise<string>
api.selectFolder(): Promise<string>
```

---

## Export / Import

```typescript
api.exportPresets(ids: number[]): Promise<string>       // JSON string
api.openImportFile(): Promise<ImportPreview>
api.validateImportModels(items: PresetData[]): Promise<ValidationWarning[]>
api.importPresets(items: PresetData[]): Promise<Preset[]>

api.exportImage(params: ExportImageParams): Promise<string>
api.listExportPresets(): Promise<ExportPreset[]>
api.saveExportPreset(data: ExportPreset): Promise<ExportPreset>
api.deleteExportPreset(id: number): Promise<void>
```

---

## Kids Mode

```typescript
api.setKidsMode(enabled: boolean, pin: string): Promise<void>
api.isKidsModeActive(): Promise<boolean>
api.getKidsCategories(): Promise<KidsCategoryInfo[]>
api.setKidsCategory(name: string, enabled: boolean, pin: string): Promise<void>
```
