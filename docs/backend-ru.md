[English](backend-en.md) | [Русский](backend-ru.md)

# SD Studio — Backend (Go)

## Конструкторы (Dependency Injection)

Все основные компоненты создаются в `main.go` и внедряются через конструкторы:

```go
cfg := config.Load()
presets, _ := preset.Open(cfg.DBPath)
llmClient := llm.New(cfg.LLMUrl, cfg.LLMBackend)
sdClient := sd.New(cfg.SDUrl)
app := NewApp(presets, llmClient, sdClient, cfg)
```

## internal/config

### Config
```go
type Config struct {
    LLMUrl         string  // env: LLM_URL
    SDUrl          string  // env: SD_URL
    LLMModel       string  // env: LLM_MODEL
    SDPromptModel  string  // env: SD_PROMPT_MODEL (алиас для LLMModel)
    VisionModel    string  // env: VISION_MODEL
    LLMBackend     string  // env: LLM_BACKEND (ollama/lmstudio)
    Port           string  // env: PORT
    DBPath         string  // env: DB_PATH
    SystemPrompt   string  // env: SYSTEM_PROMPT
    DefaultNegative string // env: DEFAULT_NEGATIVE
    DefaultSampler  string
    DefaultSteps    int
    DefaultCfgScale float64
    DefaultWidth    int
    DefaultHeight   int
}
```

### Константы промптов
- `DefaultSDPromptInstruction` — системный промпт для генерации SD промптов
- `DefaultAnalyzeSystemPrompt` — промпт для анализа изображений
- `DefaultAnalyzeChainPrompts` — цепочка промптов для глубокого анализа
- `KidsModePrompt` — промпт для безопасного режима

## internal/llm

### Client
HTTP-клиент для OpenAI-совместимого API.

```go
type Client struct { ... }

func New(baseURL, backend string) *Client
func (c *Client) Chat(model, systemPrompt, userMessage string, temperature float64, maxTokens int) (string, error)
func (c *Client) ChatVision(model, systemPrompt, userText, imageBase64 string, temperature float64, maxTokens int) (string, error)
func (c *Client) AnalyzeImage(model, systemPrompt, imageBase64 string, maxTokens int) (string, error)
func (c *Client) GenerateSDPrompt(systemPrompt, description, presetType, model string, maxTokens int) (string, error)
func (c *Client) ChatWithMessages(model string, messages []Message, temperature float64, maxTokens int) (string, error)
func (c *Client) GetModels() ([]LLMModel, error)
func (c *Client) HealthCheck() error
func (c *Client) SetURL(baseURL string)
func (c *Client) SetBackend(backend string)
func (c *Client) SetBackendConfig(cfg BackendConfig)
```

### BackendConfig (для Ollama)
```go
type BackendConfig struct {
    KeepAlive string  // "5m", "0" (выгрузить сразу)
    NumCtx    int     // размер контекста
    NumGPU    int     // GPU слои
}
```

### Утилиты
- `CleanTags(s string) string` — очистка и обрезка тегов
- `stripThinkTags(s string) string` — удаление `<think/>` блоков

## internal/sd

### Client
HTTP-клиент для SD WebUI API с retry.

```go
type Client struct {
    baseURL           string
    httpClient        *http.Client
    retryMaxAttempts  int
    retryDelay        time.Duration
}

func New(baseURL string) *Client
func (c *Client) Txt2Img(req Txt2ImgRequest) (*Txt2ImgResponse, error)    // с retry
func (c *Client) Img2Img(req Img2ImgRequest) (*Txt2ImgResponse, error)    // с retry
func (c *Client) GetModels() ([]SDModel, error)
func (c *Client) GetSamplers() ([]Sampler, error)
func (c *Client) GetSchedulers() ([]Scheduler, error)
func (c *Client) GetUpscalers() ([]Upscaler, error)
func (c *Client) GetVAEs() ([]VAE, error)
func (c *Client) GetLoRAs() ([]LoRA, error)
func (c *Client) SetModel(modelName string) error
func (c *Client) SetVAE(vaeName string) error
func (c *Client) HealthCheck() error
func (c *Client) GetOptions() (map[string]interface{}, error)
func (c *Client) SetURL(baseURL string)
```

### Retry
- `Txt2Img` и `Img2Img` используют `doPost()` с retry
- Retry: 500/502/503/504 + network errors
- Max 3 attempts, exponential backoff 2s -> 4s
- При исчерпании попыток: ошибка включает тело ответа SD

### Запросы

**Txt2ImgRequest:**
```go
type Txt2ImgRequest struct {
    Prompt, NegativePrompt, SamplerName, Scheduler string
    Steps                  int
    CfgScale               float64
    Width, Height          int
    Seed                   *int64
    DenoisingStrength      *float64
    ClipSkip               *int
    BatchSize, BatchCount  *int
    HiresFix               *bool
    // ... hires параметры
}
```

**Img2ImgRequest (дополнительно):**
```go
type Img2ImgRequest struct {
    InitImages             []string  // base64
    Mask                   string    // base64 PNG
    MaskBlur               int
    InpaintingFill         int       // 0=Fill, 1=Original, 2=Latent Noise
    InpaintFullRes         bool
    InpaintFullResPadding  int
    // ... все поля Txt2ImgRequest
}
```

**Txt2ImgResponse:**
```go
type Txt2ImgResponse struct {
    Images     []string        // base64 PNG
    Parameters json.RawMessage
    Info       json.RawMessage
    Error      string
}
```

## internal/preset

### DB
SQLite CRUD для всех данных приложения.

```go
type DB struct { ... }

func Open(dbPath string) (*DB, error)
func (db *DB) Close() error
```

### CRUD методы
Пресеты: `ListPresets`, `Get`, `Create`, `Update`, `Delete`, `ListByType`
Типы: `ListPresetTypes`, `GetPresetType`, `CreatePresetType`, `UpdatePresetType`, `DeletePresetType`
Compound: `ListCompoundPresets`, `GetCompoundPreset`, `CreateCompoundPreset`, `UpdateCompoundPreset`, `DeleteCompoundPreset`
Settings: `GetSetting(key)`, `SetSetting(key, value)`
Descriptions: `ListDescriptions`, `CreateDescription`, `UpdateDescription`, `DeleteDescription`
Prompts: `ListPrompts`, `CreatePrompt`, `DeletePrompt`
Scenes: `ListSavedScenes`, `GetSavedScene`, `SaveScene`, `UpdateSavedScene`, `DeleteSavedScene`
Sessions: `CreateSession`, `ListSessions`, `SwitchSession`, `DeleteSession`, etc.
- `AddSessionItem` деактивирует предыдущие элементы (`is_active=0`) перед вставкой нового
Export: `ListExportPresets`, `SaveExportPreset`, `DeleteExportPreset`

### Миграции
Версии 1-10 в `db.go`. Автоматически выполняются при `Open()`.

### Модели

**Preset:**
```go
type Preset struct {
    ID             int64
    Name           string
    TypeID         int64
    PresetType     string
    Prompt         string
    NegativePrompt string
    Sampler        string
    ScheduleType   string
    Steps          int
    CfgScale       float64
    Width, Height  int
    ModelName      string
    VAE            string
    Seed           *int64
    ClipSkip       *int
    Loras          string   // JSON: [{"name":"lora1","weight":0.8}]
}
```

**LoRAEntry:**
```go
type LoRAEntry struct {
    Name   string  `json:"name"`
    Weight float64 `json:"weight"`
}
```

## internal/compositor

### Compositor
Multi-pass генерация сцен с персонажами.

```go
type Compositor struct { ... }

func New(sdClient SDGenerator, rembgClient RembgClient, presetDB PresetGetter, emit ProgressEmitter) *Compositor
func (c *Compositor) GenerateScene(scene Scene) (*MultiPassResult, error)
func DecomposeSceneFromJSON(jsonStr string) (*Scene, error)
```

### Интерфейсы
```go
type SDGenerator interface {
    Txt2Img(req sd.Txt2ImgRequest) (*sd.Txt2ImgResponse, error)
    Img2Img(req sd.Img2ImgRequest) (*sd.Txt2ImgResponse, error)
    SetModel(modelName string) error
    SetVAE(vaeName string) error
}

type RembgClient interface {
    Remove(imageBase64 string) (string, error)
}

type PresetGetter interface {
    Get(id int64) (*preset.Preset, error)
}

type ProgressEmitter interface {
    Emit(step string, character, total int)
}
```

### Компоновка
- `RemoveWhiteBackground(img)` — удаляет белый фон (порог 240)
- `CompositeOver(background, character, pos, scale)` — накладывает персонажа на фон

## internal/rembg

```go
type Client struct { ... }
func New(baseURL string) *Client
func (c *Client) Remove(imageBase64 string) (string, error)
```

## internal/generation

Сервис генерации, обрабатывающий все режимы генерации изображений. Извлечён из `app.go` для изоляции логики генерации от Wails bindings.

```go
type Service struct { ... }

func New(presetDB *preset.DB, llmClient llm.Service, sdClient sd.Service, cfg *config.Config, ...) *Service
func (s *Service) SetContext(ctx context.Context)
func (s *Service) GenerateImage(params GenerateImageParams) (*GenerateImageResult, error)
func (s *Service) GenerateFromImage(params GenerateFromImageParams) (*GenerateImageResult, error)
func (s *Service) GenerateSDPrompt(description, presetType string) (string, error)
func (s *Service) RecommendPreset(description string) (*RecommendPresetResult, error)
func (s *Service) GenerateCompoundImage(params CompoundParams) (*GenerateImageResult, error)
func (s *Service) GenerateScene(scene SavedScene) (*MultiPassResult, error)
func (s *Service) UpscaleImage(image, mode string, scale float64) (*GenerateImageResult, error)
func (s *Service) StartSDPolling()
func (s *Service) StopSDPolling()
```

Методы:
- `GenerateImage` — генерация txt2img с разрешением пресета, генерацией промпта через LLM и опциональным батчем
- `GenerateFromImage` — генерация img2img с анализом исходного изображения и инпейнтингом
- `GenerateSDPrompt` — преобразование текстового описания в SD промпт через LLM
- `RecommendPreset` — подбор лучшего пресета для заданного описания
- `GenerateCompoundImage` — многошаговая генерация с использованием составных пресетов
- `GenerateScene` — multi-pass генерация сцен с компоновкой персонажей
- `UpscaleImage` — апскейлинг изображений через extras API SD WebUI
- `StartSDPolling` / `StopSDPolling` — периодический опрос состояния SD WebUI

## internal/queue

Очередь задач с логикой повторных попыток и состоянием паузы.

```go
type Service struct { ... }

func NewService(store *Store, processor Processor, emit EventEmitter) *Service
func (s *Service) Start(ctx context.Context)
func (s *Service) Enqueue(jobType JobType, params any, source string) (int64, error)
func (s *Service) GetJobs() ([]*Job, error)
func (s *Service) RemoveJob(id int64) error
func (s *Service) CancelJob(id int64) error
func (s *Service) PauseQueue()
func (s *Service) ResumeQueue()
func (s *Service) ResumePausedJobs() (int, error)
```

Основные возможности:
- Состояния задач: pending -> running -> completed/failed/paused
- Повторные попытки с экспоненциальной задержкой (5с -> 10с -> 20с -> ... -> 60с макс)
- Состояние paused при исчерпании лимита попыток (3)
- `IsRetryableError()` проверяет транзиентные ошибки (connection refused, timeout, EOF и т.д.)
- Воркер опрашивает БД каждые 5с для запланированных повторных попыток
- Миграция БД v24 добавляет колонки retry_count, max_retries, next_retry_at
- События фронтенда: `queue:updated`, `queue:job-progress`

### Store

```go
type Store struct { ... }

func NewStore(db *preset.DB) *Store
func (s *Store) CreateJob(job *Job) error
func (s *Store) UpdateJob(job *Job) error
func (s *Store) GetPendingJobs() ([]*Job, error)
func (s *Store) GetScheduledJobs() ([]*Job, error)
func (s *Store) GetJobByID(id int64) (*Job, error)
func (s *Store) DeleteJob(id int64) error
```

### Модель Job

```go
type Job struct {
    ID          int64
    Type        JobType
    Status      JobStatus
    Params      string    // JSON-закодированные параметры
    Source      string
    Result      string    // JSON-закодированный результат
    Error       string
    RetryCount  int
    MaxRetries  int
    NextRetryAt *time.Time
    CreatedAt   time.Time
    UpdatedAt   time.Time
}
```

## internal/logger

Логгер событий с LogBridge для перехвата всех log-сообщений.

```go
type Logger struct { ... }

func New(ctx context.Context) *Logger
func (l *Logger) SetContext(ctx context.Context)
func (l *Logger) InstallBridge()
func (l *Logger) Error(format string, args ...interface{})
func (l *Logger) Warn(format string, args ...interface{})
func (l *Logger) Info(format string, args ...interface{})
func (l *Logger) Debug(format string, args ...interface{})
```

`InstallBridge()` перехватывает `log.SetOutput` для захвата всех стандартных log-сообщений (из пакетов LLM, SD, queue) и повторной отправки как событий `log:entry`, отображаемых в панели логов подвала фронтенда.

## Сохранение Layout окна

Привязки `SaveWindowLayout(footerHeight)` и `GetFooterHeight()` сохраняют состояние окна:
- Размер окна (ширина, высота), позиция (x, y), состояние максимизации
- Высота панели подвала (footer)
- Хранятся в таблице `settings` SQLite
- Восстанавливаются при запуске через `restoreWindowLayout()`
- Сохраняются при событии `beforeunload` и после перетаскивания/изменения размера footer

## Вспомогательные функции app.go

### Preset Resolution
Общий паттерн для всех режимов генерации:
```go
// Загрузить пресет -> извлечь параметры -> установить модель/VAE
p, _ := a.presets.Get(presetID)
samplerName, steps, cfgScale, width, height = ...
a.sd.SetModel(p.ModelName)
a.sd.SetVAE(p.VAE)
```

### Image Processing
- `analyzeRemoveContext(image, mask)` — красный overlay + LLM vision

### LLM Config
- `applyLLMConfig(mode)` — загружает Ollama-специфичные настройки из settings
- Режимы: `"generate"` (текст), `"analyze"` (vision)

### Events
```go
runtime.EventsEmit(a.ctx, "event:name", data)
```
- `analyze:step` — прогресс анализа
- `remove:stage` — "analyzing" / "generating"
- `multipass:progress` — progress multi-pass
- `batch:progress` / `batch:done` / `batch:error`
- `session:added` — новое изображение в сессии
- `queue:updated` — изменение состояния очереди
- `queue:job-progress` — прогресс выполнения задачи
- `log:entry` — запись лога от LogBridge

### Kids Mode
- `isKidsMode()` — проверка включён
- `filterKidsInput(text)` — фильтрация входных данных
- `applyKidsSystemPrompt(prompt)` — модификация system prompt
