[English](frontend-en.md) | [Русский](frontend-ru.md)

# SD Studio — Frontend (Vue 3)

## Structure

```
frontend/src/
├── main.js                        # Vue app entrypoint
├── App.vue                        # Root component (sidebar + router)
├── api.js                         # API layer (Wails bindings proxy)
├── style.css                      # Global styles
├── components/
│   ├── GeneratePage.vue           # Main generation (txt2img)
│   ├── GenerateFromImagePage.vue  # Generation from image
│   ├── SceneEditorPage.vue        # Multi-pass scene editor
│   ├── PresetsPage.vue            # Preset management
│   ├── UnifiedPresetsPage.vue     # Unified: presets + types + compounds
│   ├── PresetForm.vue             # Preset create/edit form
│   ├── SettingsPage.vue           # Settings (LLM, SD, prompts)
│   ├── BatchPage.vue              # Batch generation
│   ├── TestPage.vue               # Test generation
│   ├── ExportPage.vue             # Image export
│   ├── FileBrowserPage.vue        # File browser
│   ├── CompoundPresetsPage.vue    # Compound presets
│   ├── ImageViewer.vue            # Fullscreen image viewer
│   ├── ImportModal.vue            # Import modal
│   ├── SavedDescriptionsModal.vue # Saved descriptions
│   ├── ResolutionSelector.vue     # Resolution profile selector
│   ├── HiresProfileSelector.vue   # Hires profile selector
│   ├── PinModal.vue               # PIN code for kids mode
│   ├── ToggleSwitch.vue           # Toggle component
│   └── AppFooter.vue              # Footer
├── i18n/
│   ├── index.js                   # Re-export entry point
│   └── en.js                      # English strings (~300 keys)
├── assets/
│   └── main.css                   # CSS variables, global styles
└── wailsjs/                       # Auto-generated Wails bindings
    ├── runtime/runtime.js         # EventsOn, EventsOff, etc.
    └── go/
        ├── main/App.js            # JS bindings
        ├── main/App.d.ts          # TypeScript types
        └── models.ts              # Data models
```

## i18n — Internationalization

All UI strings are extracted into `i18n/en.js` — a flat object with ~300 keys.

```javascript
import { t } from '../i18n/index.js'

// Usage
t('generate.btn_generate')                    // → "Generate"
t('settings.label_llm_url')                   // → "LLM URL"
t('fi.progress_analyze', { step: 1, total: 3 }) // → "Analyzing 1/3..."
```

**Key naming convention:** `{component}.{context}` — for example, `generate.btn_generate`, `settings.label_llm_url`.

**Adding a new language:**
1. Create `i18n/{lang}.js` with the same set of keys
2. Add a switch in `i18n/index.js`

## api.js — API Layer

All Wails binding calls go through `api.js`. This is the single import point for components.

```javascript
import { GenerateImage, ... } from './wailsjs/go/main/App.js'

export const api = {
  generateImage: (presetId, extraPrompt, extraNegativePrompt) =>
    GenerateImage({ preset_id: presetId, ... }),
  // ...
}
```

**Adding a new binding:**
1. Add a method in `app.go` with an uppercase first letter
2. Run `wails dev` to generate bindings (or add manually in `App.js`, `App.d.ts`, `models.ts`)
3. Add to `api.js`

## Events (Backend → Frontend)

```javascript
import { EventsOn, EventsOff } from '../wailsjs/runtime/runtime'

// Subscribe
EventsOn("remove:stage", (stage) => { ... })

// Unsubscribe
EventsOff("remove:stage")
```

### All Events

| Event | Data | Page |
|-------|------|------|
| `analyze:step` | `(step, total)` | GenerateFromImage |
| `remove:stage` | `"analyzing"` / `"generating"` | GenerateFromImage |
| `multipass:progress` | `{step, character, total}` | SceneEditor |
| `batch:progress` | `(current, total, fileName)` | Batch |
| `batch:done` | — | Batch |
| `batch:error` | `error` | Batch |
| `session:added` | — | GenerateFromImage |

## Main Components

### GeneratePage.vue
The main generation page.

**State:**
```javascript
const selectedPresetId = ref(null)
const description = ref('')
const extraPrompt = ref('')
const extraNegativePrompt = ref('')
const generating = ref(false)
const generatedImage = ref('')
const genInfo = ref(null)
```

**Flow:**
1. Select a preset → fill in the description
2. `api.generateSdPrompt()` → prompt preview
3. `api.generateImage()` → generation
4. Result: image (base64) + info (JSON)
5. Download / upscale / add to session

### GenerateFromImagePage.vue
Generation from an image. 4 modes.

**Modes:**
```javascript
const mode = ref('img2img')  // img2img | inpaint | remove
```

**Mask Canvas (inpaint/remove):**
- HTML5 Canvas overlaid on the image
- Drawing with a white semi-transparent brush
- `getMaskBase64()` — converts to black-and-white PNG (white = mask) with dilation + feathering
- **Mask Padding** (0–64px) — dilation: blur + threshold, expands the mask
- **Mask Feather** (0–64px) — Gaussian blur on mask edges for a smooth transition
- Fullscreen mask editor
- Undo (canvas state history)

**Drag & Drop:**
- Drag an image onto the drop zone
- Ctrl+V paste from clipboard
- "Last Generated" button (from session)

### SceneEditorPage.vue
Multi-pass character composition.

**Flow:**
1. Scene description → `api.decomposeScene()` → Scene object
2. Editing: drag characters, prompts, positions
3. `api.generateMultiPass(scene)` → composition result
4. Progress: `multipass:progress` events

### PresetsPage.vue / UnifiedPresetsPage.vue
Preset CRUD.

**PresetForm.vue** — form with fields:
- Name, Type, Prompt, Negative Prompt
- Model, VAE, Sampler, Scheduler
- Steps, Cfg Scale, Width, Height, Seed, Clip Skip
- LoRA: dynamic list `{name, weight}`

### SettingsPage.vue
Application settings. Tabs:
- **General:** LLM URL, SD URL, Backend
- **Models:** LLM generate model, LLM analyze model
- **Prompt:** sd_prompt_instruction (textarea)
- **Image Browser:** default folder
- **Kids Mode:** PIN, categories

### BatchPage.vue
Batch generation from files in a folder.

**Flow:**
1. Select a folder → scan files
2. Select a preset + description
3. `api.batchGenerate()` — async generation
4. Progress via `batch:progress` events

### ImageViewer.vue
Fullscreen image viewer (zoom, navigation).

```vue
<ImageViewer :image-base64="image" @close="showViewer = false" />
```

## CSS Variables

```css
:root {
  --bg-primary: #1a1a2e;
  --bg-secondary: #16213e;
  --surface-2: #1e2a4a;
  --text-primary: #e0e0e0;
  --text-dim: #888;
  --accent: #4f8fff;
  --border: #2a3a5e;
  --radius-sm: 6px;
  --error-bg: #2e1a1a;
  --error-text: #ff6b6b;
}
```

## Keyboard Shortcuts

| Shortcut | Action | Context |
|----------|--------|---------|
| `Ctrl+Enter` | Generate | GeneratePage, GenerateFromImagePage |
| `Escape` | Close fullscreen mask editor | GenerateFromImagePage |

## Testing

### Infrastructure
- **Vitest** + `@vue/test-utils` + `happy-dom`
- Run: `npm test` (or `npm run test:watch`)
- Config: `vitest.config.js`

### Mocks
All Wails bindings are mocked in `src/__tests__/setup.js` via `vi.mock()`. Components do not access the real backend.

**Runtime mocks** (`src/__tests__/mocks/runtime.js`):
- `EventsOn`, `EventsOff`, `EventsEmit` — events can be emitted in tests
- `clearEventMocks()` — cleanup between tests

**Wails binding mocks** (`src/__tests__/mocks/wails.js`):
- `mockWailsBinding(name, fn)` — override a mock for a specific test
- `clearWailsMocks()` — cleanup

### Writing Tests
```javascript
import { mount } from '@vue/test-utils'
import MyComponent from '../MyComponent.vue'

test('renders correctly', () => {
  const wrapper = mount(MyComponent, { props: { ... } })
  expect(wrapper.text()).toContain('expected text')
})
```

### Base64 Image Format
All images are passed as pure base64 (without the `data:image/...;base64,` prefix). Conversion for `<img>`:

```vue
<img :src="'data:image/png;base64,' + imageBase64" />
```

### Error Handling
```javascript
try {
  const result = await api.generateImage(presetId, description)
  if (!result || !result.image) {
    error.value = 'No image returned'
  }
} catch (e) {
  error.value = String(e)  // Show to user
}
```

### Preset Persistence (GenerateFromImagePage)
State (mode, preset_id, denoising, etc.) is saved to settings via `fi_*` keys:
```javascript
function saveFIState() {
  api.updateSettings({
    fi_mode: mode.value,
    fi_preset_id: String(selectedPresetId.value),
    fi_denoising: String(denoisingStrength.value),
    // ...
  })
}
```

### Shared State
Between pages via `inject('sharedGenState')`:
```javascript
// From GeneratePage to GenerateFromImagePage
const shared = inject('sharedGenState', null)
if (shared) {
  shared.description = tags.value
}
```
