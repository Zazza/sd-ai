[English](backend-en.md) | [Русский](backend-ru.md)

# SD Studio — Backend (Go)

## Constructors (Dependency Injection)

All core components are created in `main.go` and injected via constructors:

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
    SDPromptModel  string  // env: SD_PROMPT_MODEL (alias for LLMModel)
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

### Prompt Constants
- `DefaultSDPromptInstruction` — system prompt for SD prompt generation
- `DefaultAnalyzeSystemPrompt` — prompt for image analysis
- `DefaultAnalyzeChainPrompts` — chain of prompts for deep analysis
- `KidsModePrompt` — prompt for kids mode (safe mode)

## internal/llm

### Client
HTTP client for OpenAI-compatible API.

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

### BackendConfig (for Ollama)
```go
type BackendConfig struct {
    KeepAlive string  // "5m", "0" (unload immediately)
    NumCtx    int     // context size
    NumGPU    int     // GPU layers
}
```

### Utilities
- `CleanTags(s string) string` — tag cleanup and trimming
- `stripThinkTags(s string) string` — removal of `<think/>` blocks

## internal/sd

### Client
HTTP client for SD WebUI API with retry support.

```go
type Client struct {
    baseURL           string
    httpClient        *http.Client
    retryMaxAttempts  int
    retryDelay        time.Duration
}

func New(baseURL string) *Client
func (c *Client) Txt2Img(req Txt2ImgRequest) (*Txt2ImgResponse, error)    // with retry
func (c *Client) Img2Img(req Img2ImgRequest) (*Txt2ImgResponse, error)    // with retry
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
- `Txt2Img` and `Img2Img` use `doPost()` with retry
- Retry: 500/502/503/504 + network errors
- Max 3 attempts, exponential backoff 2s -> 4s
- When attempts are exhausted: error includes the SD response body

### Requests

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
    // ... hires parameters
}
```

**Img2ImgRequest (additional fields):**
```go
type Img2ImgRequest struct {
    InitImages             []string  // base64
    Mask                   string    // base64 PNG
    MaskBlur               int
    InpaintingFill         int       // 0=Fill, 1=Original, 2=Latent Noise
    InpaintFullRes         bool
    InpaintFullResPadding  int
    // ... all Txt2ImgRequest fields
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
SQLite CRUD for all application data.

```go
type DB struct { ... }

func Open(dbPath string) (*DB, error)
func (db *DB) Close() error
```

### CRUD Methods
Presets: `ListPresets`, `Get`, `Create`, `Update`, `Delete`, `ListByType`
Types: `ListPresetTypes`, `GetPresetType`, `CreatePresetType`, `UpdatePresetType`, `DeletePresetType`
Compound: `ListCompoundPresets`, `GetCompoundPreset`, `CreateCompoundPreset`, `UpdateCompoundPreset`, `DeleteCompoundPreset`
Settings: `GetSetting(key)`, `SetSetting(key, value)`
Descriptions: `ListDescriptions`, `CreateDescription`, `UpdateDescription`, `DeleteDescription`
Prompts: `ListPrompts`, `CreatePrompt`, `DeletePrompt`
Scenes: `ListSavedScenes`, `GetSavedScene`, `SaveScene`, `UpdateSavedScene`, `DeleteSavedScene`
Sessions: `CreateSession`, `ListSessions`, `SwitchSession`, `DeleteSession`, etc.
- `AddSessionItem` deactivates previous items (`is_active=0`) before inserting a new one
Export: `ListExportPresets`, `SaveExportPreset`, `DeleteExportPreset`

### Migrations
Versions 1-10 in `db.go`. Automatically executed on `Open()`.

### Models

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
Multi-pass scene generation with characters.

```go
type Compositor struct { ... }

func New(sdClient SDGenerator, rembgClient RembgClient, presetDB PresetGetter, emit ProgressEmitter) *Compositor
func (c *Compositor) GenerateScene(scene Scene) (*MultiPassResult, error)
func DecomposeSceneFromJSON(jsonStr string) (*Scene, error)
```

### Interfaces
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

### Compositing
- `RemoveWhiteBackground(img)` — removes white background (threshold 240)
- `CompositeOver(background, character, pos, scale)` — overlays a character onto the background

## internal/rembg

```go
type Client struct { ... }
func New(baseURL string) *Client
func (c *Client) Remove(imageBase64 string) (string, error)
```

## internal/generation

Generation service that handles all image generation modes. Extracted from `app.go` to isolate generation logic from Wails bindings.

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

Methods:
- `GenerateImage` — txt2img generation with preset resolution, LLM prompt generation, and optional batch
- `GenerateFromImage` — img2img generation with init image analysis and inpainting
- `GenerateSDPrompt` — converts text description to SD prompt via LLM
- `RecommendPreset` — suggests best preset for a given description
- `GenerateCompoundImage` — multi-step generation using compound presets
- `GenerateScene` — multi-pass scene generation with character compositing
- `UpscaleImage` — image upscaling via SD WebUI extras API
- `StartSDPolling` / `StopSDPolling` — periodic SD WebUI health/status polling

## internal/queue

Job queue with retry logic and paused state.

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

Key features:
- Job states: pending -> running -> completed/failed/paused
- Retry with exponential backoff (5s -> 10s -> 20s -> ... -> 60s max)
- Paused state when max retries (3) exhausted
- `IsRetryableError()` checks for transient errors (connection refused, timeout, EOF, etc.)
- Worker polls every 5s for scheduled retries
- DB migration v24 adds retry_count, max_retries, next_retry_at columns
- Frontend events: `queue:updated`, `queue:job-progress`

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

### Job Model

```go
type Job struct {
    ID          int64
    Type        JobType
    Status      JobStatus
    Params      string    // JSON-encoded params
    Source      string
    Result      string    // JSON-encoded result
    Error       string
    RetryCount  int
    MaxRetries  int
    NextRetryAt *time.Time
    CreatedAt   time.Time
    UpdatedAt   time.Time
}
```

## internal/logger

Event logger with LogBridge for capturing all log output.

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

`InstallBridge()` hooks `log.SetOutput` to capture all standard log output (from LLM, SD, queue packages) and re-emit as `log:entry` events visible in the frontend footer log panel.

## Window Layout Persistence

`SaveWindowLayout(footerHeight)` and `GetFooterHeight()` bindings persist window state:
- Window size (width, height), position (x, y), maximised state
- Footer panel height
- Stored in SQLite `settings` table
- Restored on startup via `restoreWindowLayout()`
- Saved on `beforeunload` event and after footer drag-resize

## app.go Helper Functions

### Preset Resolution
Common pattern for all generation modes:
```go
// Load preset -> extract parameters -> set model/VAE
p, _ := a.presets.Get(presetID)
samplerName, steps, cfgScale, width, height = ...
a.sd.SetModel(p.ModelName)
a.sd.SetVAE(p.VAE)
```

### Image Processing
- `analyzeRemoveContext(image, mask)` — red overlay + LLM vision

### LLM Config
- `applyLLMConfig(mode)` — loads Ollama-specific settings from settings
- Modes: `"generate"` (text), `"analyze"` (vision)

### Events
```go
runtime.EventsEmit(a.ctx, "event:name", data)
```
- `analyze:step` — analysis progress
- `remove:stage` — "analyzing" / "generating"
- `multipass:progress` — multi-pass progress
- `batch:progress` / `batch:done` / `batch:error`
- `session:added` — new image added to session
- `queue:updated` — queue state changed
- `queue:job-progress` — job execution progress
- `log:entry` — log entry from LogBridge

### Kids Mode
- `isKidsMode()` — checks if enabled
- `filterKidsInput(text)` — input filtering
- `applyKidsSystemPrompt(prompt)` — system prompt modification
