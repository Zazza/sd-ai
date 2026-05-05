[English](frontend-en.md) | [Русский](frontend-ru.md)

# SD Studio — Frontend (Vue 3)

## Структура

```
frontend/src/
├── main.js                        # Vue app entrypoint
├── App.vue                        # Root component (sidebar + router)
├── api.js                         # API-слой (Wails bindings proxy)
├── style.css                      # Global styles
├── components/
│   ├── GeneratePage.vue           # Основная генерация (txt2img)
│   ├── GenerateFromImagePage.vue  # Генерация из изображения
│   ├── SceneEditorPage.vue        # Multi-pass редактор сцен
│   ├── PresetsPage.vue            # Управление пресетами
│   ├── UnifiedPresetsPage.vue     # Unified: presets + types + compounds
│   ├── PresetForm.vue             # Форма создания/редактирования пресета
│   ├── SettingsPage.vue           # Настройки (LLM, SD, prompts)
│   ├── BatchPage.vue              # Пакетная генерация
│   ├── TestPage.vue               # Тестовая генерация
│   ├── ExportPage.vue             # Экспорт изображений
│   ├── FileBrowserPage.vue        # Файловый браузер
│   ├── CompoundPresetsPage.vue    # Compound пресеты
│   ├── ImageViewer.vue            # Fullscreen просмотр изображений
│   ├── ImportModal.vue            # Модальное окно импорта
│   ├── SavedDescriptionsModal.vue # Сохранённые описания
│   ├── PinModal.vue               # PIN-код для kids mode
│   ├── ToggleSwitch.vue           # Toggle компонент
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

## i18n — Интернационализация

Все UI-строки вынесены в `i18n/en.js` — плоский объект с ~300 ключами.

```javascript
import { t } from '../i18n/index.js'

// Использование
t('generate.btn_generate')                    // → "Generate"
t('settings.label_llm_url')                   // → "LLM URL"
t('fi.progress_analyze', { step: 1, total: 3 }) // → "Analyzing 1/3..."
```

**Конвенция именования ключей:** `{component}.{context}` — например `generate.btn_generate`, `settings.label_llm_url`.

**Добавление нового языка:**
1. Создать `i18n/{lang}.js` с таким же набором ключей
2. Добавить переключатель в `i18n/index.js`

## api.js — API-слой

Все вызовы Wails bindings проходят через `api.js`. Это единственная точка импорта для компонентов.

```javascript
import { GenerateImage, ... } from './wailsjs/go/main/App.js'

export const api = {
  generateImage: (presetId, extraPrompt, extraNegativePrompt) =>
    GenerateImage({ preset_id: presetId, ... }),
  // ...
}
```

**Добавление нового binding:**
1. Метод в `app.go` с заглавной буквы
2. Запустить `wails dev` для генерации bindings (или добавить вручную в `App.js`, `App.d.ts`, `models.ts`)
3. Добавить в `api.js`

## Events (Backend → Frontend)

```javascript
import { EventsOn, EventsOff } from '../wailsjs/runtime/runtime'

// Подписка
EventsOn("remove:stage", (stage) => { ... })

// Отписка
EventsOff("remove:stage")
```

### Все events

| Event | Data | Страница |
|-------|------|----------|
| `analyze:step` | `(step, total)` | GenerateFromImage |
| `remove:stage` | `"analyzing"` / `"generating"` | GenerateFromImage |
| `multipass:progress` | `{step, character, total}` | SceneEditor |
| `batch:progress` | `(current, total, fileName)` | Batch |
| `batch:done` | — | Batch |
| `batch:error` | `error` | Batch |
| `session:added` | — | GenerateFromImage |

## Основные компоненты

### GeneratePage.vue
Основная страница генерации.

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
1. Выбор пресета → заполнение описания
2. `api.generateSdPrompt()` → превью промпта
3. `api.generateImage()` → генерация
4. Результат: image (base64) + info (JSON)
5. Скачать / upscale / добавить в сессию

### GenerateFromImagePage.vue
Генерация из изображения. 4 режима.

**Режимы:**
```javascript
const mode = ref('img2img')  // img2img | inpaint | remove
```

**Mask Canvas (inpaint/remove):**
- HTML5 Canvas поверх изображения
- Рисование белой полупрозрачной кистью
- `getMaskBase64()` — конвертация в ч/б PNG (белый = маска) с dilation + feathering
- **Mask Padding** (0–64px) — dilation: blur + threshold, расширяет маску
- **Mask Feather** (0–64px) — Gaussian blur краёв маски для плавного перехода
- Fullscreen редактор маски
- Undo (история состояний canvas)

**Drag & Drop:**
- Перетаскивание изображения на drop zone
- Ctrl+V вставка из буфера
- Кнопка "Last Generated" (из сессии)

### SceneEditorPage.vue
Multi-pass компоновка персонажей.

**Flow:**
1. Описание сцены → `api.decomposeScene()` → Scene объект
2. Редактирование: drag персонажей, промпты, позиции
3. `api.generateMultiPass(scene)` → результат с композицией
4. Progress: `multipass:progress` events

### PresetsPage.vue / UnifiedPresetsPage.vue
CRUD пресетов.

**PresetForm.vue** — форма с полями:
- Name, Type, Prompt, Negative Prompt
- Model, VAE, Sampler, Scheduler
- Steps, Cfg Scale, Width, Height, Seed, Clip Skip
- LoRA: динамический список `{name, weight}`

### SettingsPage.vue
Настройки приложения. Табы:
- **General:** LLM URL, SD URL, Backend
- **Models:** LLM generate model, LLM analyze model
- **Prompt:** sd_prompt_instruction (textarea)
- **Image Browser:** default folder
- **Kids Mode:** PIN, категории

### BatchPage.vue
Пакетная генерация по файлам из папки.

**Flow:**
1. Выбор папки → сканирование файлов
2. Выбор пресета + описание
3. `api.batchGenerate()` — асинхронная генерация
4. Progress через `batch:progress` events

### ImageViewer.vue
Fullscreen просмотр изображений (зум, навигация).

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

## Клавиатурные сокращения

| Shortcut | Действие | Контекст |
|----------|----------|----------|
| `Ctrl+Enter` | Generate | GeneratePage, GenerateFromImagePage |
| `Escape` | Close fullscreen mask editor | GenerateFromImagePage |

## Testing

### Инфраструктура
- **Vitest** + `@vue/test-utils` + `happy-dom`
- Запуск: `npm test` (или `npm run test:watch`)
- Конфиг: `vitest.config.js`

### Mocks
Все Wails bindings мокаются в `src/__tests__/setup.js` через `vi.mock()`. Компоненты не обращаются к реальному бэкенду.

**Runtime mocks** (`src/__tests__/mocks/runtime.js`):
- `EventsOn`, `EventsOff`, `EventsEmit` — можно emit'ить события в тестах
- `clearEventMocks()` — очистка между тестами

**Wails binding mocks** (`src/__tests__/mocks/wails.js`):
- `mockWailsBinding(name, fn)` — переопределение мока для конкретного теста
- `clearWailsMocks()` — очистка

### Написание тестов
```javascript
import { mount } from '@vue/test-utils'
import MyComponent from '../MyComponent.vue'

test('renders correctly', () => {
  const wrapper = mount(MyComponent, { props: { ... } })
  expect(wrapper.text()).toContain('expected text')
})
```

### Формат base64 изображений
Все изображения передаются как чистый base64 (без `data:image/...;base64,` префикса). Преобразование для `<img>`:

```vue
<img :src="'data:image/png;base64,' + imageBase64" />
```

### Error handling
```javascript
try {
  const result = await api.generateImage(presetId, description)
  if (!result || !result.image) {
    error.value = 'No image returned'
  }
} catch (e) {
  error.value = String(e)  // Показать пользователю
}
```

### Preset persistence (GenerateFromImagePage)
Состояние (mode, preset_id, denoising, etc.) сохраняется в settings через `fi_*` ключи:
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

### Shared state
Между страницами через `inject('sharedGenState')`:
```javascript
// Из GeneratePage в GenerateFromImagePage
const shared = inject('sharedGenState', null)
if (shared) {
  shared.description = tags.value
}
```
