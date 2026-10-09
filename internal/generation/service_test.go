package generation

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"image"
	"image/color"
	"image/png"
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"go-sd/internal/config"
	"go-sd/internal/kids"
	"go-sd/internal/llm"
	"go-sd/internal/logger"
	"go-sd/internal/preset"
	"go-sd/internal/promptutil"
	"go-sd/internal/sd"
)

type mockLLM struct {
	mu             sync.Mutex
	chatFn         func(model, systemPrompt, userMessage string, temperature float64, maxTokens int) (string, error)
	chatVisionFn   func(model, systemPrompt, userText, imageBase64 string, temperature float64, maxTokens int) (string, error)
	chatMsgFn      func(model string, messages []llm.Message, temperature float64, maxTokens int) (string, error)
	genSDPromptFn  func(systemPrompt, userMessage, presetType, model string, maxTokens int) (string, error)
	analyzeImageFn func(model, systemPrompt, imageBase64 string, maxTokens int) (string, error)
	describeFn     func(model, prompt, imageBase64 string, maxTokens int) (string, error)
}

func (m *mockLLM) Chat(model, systemPrompt, userMessage string, temperature float64, maxTokens int) (string, error) {
	m.mu.Lock()
	fn := m.chatFn
	m.mu.Unlock()
	if fn != nil {
		return fn(model, systemPrompt, userMessage, temperature, maxTokens)
	}
	return "", fmt.Errorf("not implemented")
}

func (m *mockLLM) ChatVision(model, systemPrompt, userText, imageBase64 string, temperature float64, maxTokens int) (string, error) {
	m.mu.Lock()
	fn := m.chatVisionFn
	m.mu.Unlock()
	if fn != nil {
		return fn(model, systemPrompt, userText, imageBase64, temperature, maxTokens)
	}
	return "", fmt.Errorf("not implemented")
}

func (m *mockLLM) ChatWithMessages(model string, messages []llm.Message, temperature float64, maxTokens int) (string, error) {
	m.mu.Lock()
	fn := m.chatMsgFn
	m.mu.Unlock()
	if fn != nil {
		return fn(model, messages, temperature, maxTokens)
	}
	return "", fmt.Errorf("not implemented")
}

func (m *mockLLM) GenerateSDPrompt(systemPrompt, userMessage, presetType, model string, maxTokens int) (string, error) {
	m.mu.Lock()
	fn := m.genSDPromptFn
	m.mu.Unlock()
	if fn != nil {
		return fn(systemPrompt, userMessage, presetType, model, maxTokens)
	}
	return "", fmt.Errorf("not implemented")
}

func (m *mockLLM) AnalyzeImage(model, systemPrompt, imageBase64 string, maxTokens int) (string, error) {
	m.mu.Lock()
	fn := m.analyzeImageFn
	m.mu.Unlock()
	if fn != nil {
		return fn(model, systemPrompt, imageBase64, maxTokens)
	}
	return "", fmt.Errorf("not implemented")
}

func (m *mockLLM) AnalyzeImageDescribe(model, prompt, imageBase64 string, maxTokens int) (string, error) {
	m.mu.Lock()
	fn := m.describeFn
	m.mu.Unlock()
	if fn != nil {
		return fn(model, prompt, imageBase64, maxTokens)
	}
	return "", fmt.Errorf("not implemented")
}

func (m *mockLLM) GetModels() ([]llm.LLMModel, error)    { return nil, nil }
func (m *mockLLM) HealthCheck() error                      { return nil }
func (m *mockLLM) SetURL(baseURL string)                   {}
func (m *mockLLM) SetBackend(backend string)                {}
func (m *mockLLM) SetBackendConfig(cfg llm.BackendConfig)   {}
func (m *mockLLM) Backend() string                          { return "" }
func (m *mockLLM) ListLoaded() ([]llm.LoadedModel, error)   { return nil, nil }
func (m *mockLLM) UnloadAll(ctx context.Context) error      { return nil }

type mockSD struct {
	mu       sync.Mutex
	txt2img  func(req sd.Txt2ImgRequest) (*sd.Txt2ImgResponse, error)
	img2img  func(req sd.Img2ImgRequest) (*sd.Txt2ImgResponse, error)
	upscale  func(base64Img string, upscaler string, scale float64) (string, error)
	setModel func(modelName string) error
	setVAE   func(vaeName string) error
	progress func() (*sd.ProgressResponse, error)
}

func (m *mockSD) Txt2Img(req sd.Txt2ImgRequest) (*sd.Txt2ImgResponse, error) {
	m.mu.Lock()
	fn := m.txt2img
	m.mu.Unlock()
	if fn != nil {
		return fn(req)
	}
	return nil, fmt.Errorf("not implemented")
}

func (m *mockSD) Img2Img(req sd.Img2ImgRequest) (*sd.Txt2ImgResponse, error) {
	m.mu.Lock()
	fn := m.img2img
	m.mu.Unlock()
	if fn != nil {
		return fn(req)
	}
	return nil, fmt.Errorf("not implemented")
}

func (m *mockSD) GetModels() ([]sd.SDModel, error)          { return nil, nil }
func (m *mockSD) GetSamplers() ([]sd.Sampler, error)        { return nil, nil }
func (m *mockSD) GetSchedulers() ([]sd.Scheduler, error)    { return nil, nil }
func (m *mockSD) GetUpscalers() ([]sd.Upscaler, error)      { return nil, nil }
func (m *mockSD) GetVAEs() ([]sd.VAE, error)                { return nil, nil }
func (m *mockSD) GetLoRAs() ([]sd.LoRA, error)              { return nil, nil }
func (m *mockSD) GetOptions() (map[string]interface{}, error) { return nil, nil }

func (m *mockSD) GetProgress() (*sd.ProgressResponse, error) {
	m.mu.Lock()
	fn := m.progress
	m.mu.Unlock()
	if fn != nil {
		return fn()
	}
	return &sd.ProgressResponse{}, nil
}

func (m *mockSD) Interrupt() error { return nil }
func (m *mockSD) HealthCheck() error { return nil }
func (m *mockSD) SetURL(baseURL string) {}

func (m *mockSD) SetModel(modelName string) error {
	m.mu.Lock()
	fn := m.setModel
	m.mu.Unlock()
	if fn != nil {
		return fn(modelName)
	}
	return nil
}

func (m *mockSD) SetVAE(vaeName string) error {
	m.mu.Lock()
	fn := m.setVAE
	m.mu.Unlock()
	if fn != nil {
		return fn(vaeName)
	}
	return nil
}

func (m *mockSD) UpscaleImage(base64Img string, upscaler string, scale float64) (string, error) {
	m.mu.Lock()
	fn := m.upscale
	m.mu.Unlock()
	if fn != nil {
		return fn(base64Img, upscaler, scale)
	}
	return "", fmt.Errorf("not implemented")
}
func (m *mockSD) MemoryInfo() (*sd.MemoryStats, error) { return nil, nil }

type mockEmitter struct {
	mu     sync.Mutex
	events []emittedEvent
}

type emittedEvent struct {
	name string
	data []any
}

func (e *mockEmitter) Emit(event string, data ...any) {
	e.mu.Lock()
	defer e.mu.Unlock()
	e.events = append(e.events, emittedEvent{name: event, data: data})
}

func (e *mockEmitter) hasEvent(name string) bool {
	e.mu.Lock()
	defer e.mu.Unlock()
	for _, ev := range e.events {
		if ev.name == name {
			return true
		}
	}
	return false
}

type mockSettings struct {
	applied []string
}

func (m *mockSettings) ApplyLLMConfig(mode string) {
	m.applied = append(m.applied, mode)
}

type mockSessions struct {
	mu     sync.Mutex
	items  []sessionItem
}

type sessionItem struct {
	imageBase64 string
	info        json.RawMessage
	source      string
	isPreview   bool
}

func (m *mockSessions) AddToSession(imageBase64 string, info json.RawMessage, source string, isPreview bool, presetID *int64) int64 {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.items = append(m.items, sessionItem{
		imageBase64: imageBase64,
		info:        info,
		source:      source,
		isPreview:   isPreview,
	})
	return int64(len(m.items))
}

func tinyPNGBase64(t *testing.T) string {
	t.Helper()
	img := image.NewRGBA(image.Rect(0, 0, 512, 512))
	img.Set(0, 0, color.Black)
	var buf bytes.Buffer
	require.NoError(t, png.Encode(&buf, img))
	return base64.StdEncoding.EncodeToString(buf.Bytes())
}

func openTestDB(t *testing.T) *preset.DB {
	t.Helper()
	db, err := preset.Open(":memory:")
	require.NoError(t, err)
	t.Cleanup(func() { db.Close() })
	return db
}

func makeTestPreset(t *testing.T, db *preset.DB, p *preset.Preset) *preset.Preset {
	t.Helper()
	if p == nil {
		p = &preset.Preset{
			Name:           "test-preset",
			PresetType:     "anime",
			Prompt:         "masterpiece, best quality, 1girl",
			NegativePrompt: "lowres, bad anatomy",
			Sampler:        "Euler a",
			Steps:          20,
			CfgScale:       7.0,
		}
	}
	err := db.Create(p)
	require.NoError(t, err)
	return p
}

func newTestService(t *testing.T, db *preset.DB, llmSvc *mockLLM, sdSvc *mockSD) *Service {
	t.Helper()
	emitter := &mockEmitter{}
	settings := &mockSettings{}
	sessions := &mockSessions{}
	kidsMgr := kids.NewManager(db)
	tmpDir := t.TempDir()
	log := logger.New(nil)

	return New(
		db,
		llmSvc,
		sdSvc,
		&config.Config{
			SDPromptModel: "test-model",
			VisionModel:   "test-vision",
		},
		tmpDir,
		emitter,
		kidsMgr,
		sessions,
		settings,
		log,
	)
}

func makePNGBase64(t *testing.T, w, h int) string {
	t.Helper()
	img := image.NewRGBA(image.Rect(0, 0, w, h))
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			img.Set(x, y, color.NRGBA{R: 128, G: 128, B: 128, A: 255})
		}
	}
	var buf bytes.Buffer
	require.NoError(t, png.Encode(&buf, img))
	return base64.StdEncoding.EncodeToString(buf.Bytes())
}

func TestGenerateSDPrompt_InvalidPresetID(t *testing.T) {
	t.Parallel()
	db := openTestDB(t)
	svc := newTestService(t, db, &mockLLM{}, &mockSD{})

	_, err := svc.GenerateSDPrompt(GenerateSDPromptParams{PresetID: 0, Description: "test"})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "preset is required")
}

func TestGenerateSDPrompt_PresetNotFound(t *testing.T) {
	t.Parallel()
	db := openTestDB(t)
	svc := newTestService(t, db, &mockLLM{}, &mockSD{})

	_, err := svc.GenerateSDPrompt(GenerateSDPromptParams{PresetID: 999, Description: "test"})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "preset not found")
}

func TestGenerateSDPrompt_EmptyDescriptionAndNegative(t *testing.T) {
	t.Parallel()
	db := openTestDB(t)
	makeTestPreset(t, db, nil)
	svc := newTestService(t, db, &mockLLM{}, &mockSD{})

	result, err := svc.GenerateSDPrompt(GenerateSDPromptParams{PresetID: 1, Description: "", Negative: ""})
	assert.NoError(t, err)
	require.NotNil(t, result)
	assert.Equal(t, "masterpiece, best quality, 1girl", result.Prompt)
	assert.Equal(t, "lowres, bad anatomy", result.NegativePrompt)
}

func TestGenerateSDPrompt_Success(t *testing.T) {
	t.Parallel()
	db := openTestDB(t)
	makeTestPreset(t, db, nil)

	llmSvc := &mockLLM{
		genSDPromptFn: func(systemPrompt, userMessage, presetType, model string, maxTokens int) (string, error) {
			return `{"prompt": "merged prompt, 1girl, cat ears", "negative_prompt": "lowres, bad anatomy, blurry"}`, nil
		},
	}

	svc := newTestService(t, db, llmSvc, &mockSD{})

	result, err := svc.GenerateSDPrompt(GenerateSDPromptParams{
		PresetID:    1,
		Description: "a girl with cat ears",
	})
	require.NoError(t, err)
	require.NotNil(t, result)
	assert.Contains(t, result.Prompt, "1girl")
	assert.Contains(t, result.NegativePrompt, "lowres")
}

func TestGenerateSDPrompt_LLMError(t *testing.T) {
	t.Parallel()
	db := openTestDB(t)
	makeTestPreset(t, db, nil)

	llmSvc := &mockLLM{
		genSDPromptFn: func(systemPrompt, userMessage, presetType, model string, maxTokens int) (string, error) {
			return "", fmt.Errorf("LLM connection refused")
		},
	}

	svc := newTestService(t, db, llmSvc, &mockSD{})

	_, err := svc.GenerateSDPrompt(GenerateSDPromptParams{
		PresetID:    1,
		Description: "a girl",
	})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "LLM connection refused")
}

func TestGenerateSDPrompt_InvalidJSONResponse(t *testing.T) {
	t.Parallel()
	db := openTestDB(t)
	makeTestPreset(t, db, nil)

	llmSvc := &mockLLM{
		genSDPromptFn: func(systemPrompt, userMessage, presetType, model string, maxTokens int) (string, error) {
			return "plain text tags without json structure, 1girl, cat ears", nil
		},
	}

	svc := newTestService(t, db, llmSvc, &mockSD{})

	result, err := svc.GenerateSDPrompt(GenerateSDPromptParams{
		PresetID:    1,
		Description: "cat girl",
	})
	require.NoError(t, err)
	require.NotNil(t, result)
	assert.NotEmpty(t, result.Prompt)
}

func TestGenerateSDPrompt_WithNegative(t *testing.T) {
	t.Parallel()
	db := openTestDB(t)
	makeTestPreset(t, db, nil)

	llmSvc := &mockLLM{
		genSDPromptFn: func(systemPrompt, userMessage, presetType, model string, maxTokens int) (string, error) {
			assert.Contains(t, userMessage, "USER NEGATIVE: blurry")
			return `{"prompt": "prompt", "negative_prompt": "neg"}`, nil
		},
	}

	svc := newTestService(t, db, llmSvc, &mockSD{})
	result, err := svc.GenerateSDPrompt(GenerateSDPromptParams{
		PresetID:    1,
		Description: "test",
		Negative:    "blurry",
	})
	require.NoError(t, err)
	require.NotNil(t, result)
}

func TestGenerateSDPrompt_EmitsLLMStatus(t *testing.T) {
	t.Parallel()
	db := openTestDB(t)
	makeTestPreset(t, db, nil)

	llmSvc := &mockLLM{
		genSDPromptFn: func(systemPrompt, userMessage, presetType, model string, maxTokens int) (string, error) {
			return `{"prompt": "p", "negative_prompt": "n"}`, nil
		},
	}

	svc := newTestService(t, db, llmSvc, &mockSD{})
	_, err := svc.GenerateSDPrompt(GenerateSDPromptParams{PresetID: 1, Description: "x"})
	require.NoError(t, err)

	emitter := svc.emitter.(*mockEmitter)
	assert.True(t, emitter.hasEvent("llm:status"))
}

func TestGetDefaultPromptInstruction(t *testing.T) {
	t.Parallel()
	db := openTestDB(t)
	svc := newTestService(t, db, &mockLLM{}, &mockSD{})

	result := svc.GetDefaultPromptInstruction()
	assert.Contains(t, result, "Stable Diffusion")
	assert.Contains(t, result, "prompt")
}

func TestGetDefaultPromptInstruction_NoVividExamples(t *testing.T) {
	t.Parallel()
	db := openTestDB(t)
	svc := newTestService(t, db, &mockLLM{}, &mockSD{})

	instruction := strings.ToLower(svc.GetDefaultPromptInstruction())
	for _, banned := range []string{"beard", "mushroom", "cloak", "artifact", "moss", "85mm", "bokeh"} {
		assert.NotContains(t, instruction, banned)
	}
}

func TestGenerateSDPrompt_FiltersInstructionExampleLeak(t *testing.T) {
	t.Parallel()
	db := openTestDB(t)
	makeTestPreset(t, db, nil)

	instruction := "Convert the scene to SD tags. Examples: (thick beard:1.3), (giant luminescent mushrooms:1.2), (blue glow from artifact on face:1.2), (worn leather cloak:1.1)"
	require.NoError(t, db.SetSetting("sd_prompt_instruction", instruction))

	llmSvc := &mockLLM{
		genSDPromptFn: func(systemPrompt, userMessage, presetType, model string, maxTokens int) (string, error) {
			return `{"prompt": "(thick beard:1.3), giant luminescent mushrooms, (blue glow from artifact on face:1.2), sunny meadow", "negative_prompt": "lowres"}`, nil
		},
	}

	svc := newTestService(t, db, llmSvc, &mockSD{})

	result, err := svc.GenerateSDPrompt(GenerateSDPromptParams{
		PresetID:    1,
		Description: "sunny meadow with flowers",
	})
	require.NoError(t, err)
	require.NotNil(t, result)

	lower := strings.ToLower(result.Prompt)
	for _, leak := range []string{"thick beard", "beard", "mushroom", "artifact", "blue glow"} {
		assert.NotContains(t, lower, leak)
	}
	assert.Contains(t, result.Prompt, "sunny meadow")
}

func TestGenerateSDPrompt_KeepsUserRequestedTag(t *testing.T) {
	t.Parallel()
	db := openTestDB(t)
	makeTestPreset(t, db, nil)

	instruction := "Convert the scene to SD tags. Examples: (thick beard:1.3), (glowing moss:1.2)"
	require.NoError(t, db.SetSetting("sd_prompt_instruction", instruction))

	llmSvc := &mockLLM{
		genSDPromptFn: func(systemPrompt, userMessage, presetType, model string, maxTokens int) (string, error) {
			return `{"prompt": "(thick beard:1.3), forest", "negative_prompt": ""}`, nil
		},
	}

	svc := newTestService(t, db, llmSvc, &mockSD{})

	result, err := svc.GenerateSDPrompt(GenerateSDPromptParams{
		PresetID:    1,
		Description: "a man with a thick beard in a forest",
	})
	require.NoError(t, err)
	require.NotNil(t, result)

	assert.Contains(t, result.Prompt, "thick beard")
	assert.Contains(t, result.Prompt, "forest")
}

func TestGenerateSDPrompt_PlaceholderLeakFiltered(t *testing.T) {
	t.Parallel()
	db := openTestDB(t)
	makeTestPreset(t, db, nil)

	llmSvc := &mockLLM{
		genSDPromptFn: func(systemPrompt, userMessage, presetType, model string, maxTokens int) (string, error) {
			return `{"prompt": "(character trait:1.3), real tag", "negative_prompt": ""}`, nil
		},
	}

	svc := newTestService(t, db, llmSvc, &mockSD{})

	result, err := svc.GenerateSDPrompt(GenerateSDPromptParams{
		PresetID:    1,
		Description: "a warrior standing on a hill at sunset",
	})
	require.NoError(t, err)
	require.NotNil(t, result)

	lower := strings.ToLower(result.Prompt)
	assert.NotContains(t, lower, "character trait")
	assert.Contains(t, result.Prompt, "real tag")
}

func TestGenerateSDPrompt_FiltersNegativePromptLeak(t *testing.T) {
	t.Parallel()
	db := openTestDB(t)
	makeTestPreset(t, db, nil)

	instruction := "Convert the scene to SD tags. Examples: (thick beard:1.3), (glowing moss:1.2)"
	require.NoError(t, db.SetSetting("sd_prompt_instruction", instruction))

	llmSvc := &mockLLM{
		genSDPromptFn: func(systemPrompt, userMessage, presetType, model string, maxTokens int) (string, error) {
			return `{"prompt": "sunny meadow", "negative_prompt": "(thick beard:1.3), glowing moss, lowres"}`, nil
		},
	}

	svc := newTestService(t, db, llmSvc, &mockSD{})

	result, err := svc.GenerateSDPrompt(GenerateSDPromptParams{
		PresetID:    1,
		Description: "sunny meadow with flowers",
	})
	require.NoError(t, err)
	require.NotNil(t, result)

	assert.Equal(t, "sunny meadow", result.Prompt)
	assert.Equal(t, "lowres", result.NegativePrompt)
}

func TestGenerateSDPrompt_ShortGenericKeySubtracted(t *testing.T) {
	t.Parallel()
	db := openTestDB(t)
	makeTestPreset(t, db, nil)

	llmSvc := &mockLLM{
		genSDPromptFn: func(systemPrompt, userMessage, presetType, model string, maxTokens int) (string, error) {
			return `{"prompt": "(tag:1.3), garage interior", "negative_prompt": ""}`, nil
		},
	}

	svc := newTestService(t, db, llmSvc, &mockSD{})

	result, err := svc.GenerateSDPrompt(GenerateSDPromptParams{
		PresetID:    1,
		Description: "vintage car garage",
	})
	require.NoError(t, err)
	require.NotNil(t, result)

	assert.Equal(t, "garage interior", result.Prompt)
}

func TestGenerateLLMPromptFromTags_FiltersInstructionLeak(t *testing.T) {
	t.Parallel()
	db := openTestDB(t)
	p := makeTestPreset(t, db, nil)

	instruction := "Convert the scene to SD tags. Examples: (thick beard:1.3), (glowing moss:1.2)"
	require.NoError(t, db.SetSetting("sd_prompt_instruction", instruction))

	llmSvc := &mockLLM{
		genSDPromptFn: func(systemPrompt, userMessage, presetType, model string, maxTokens int) (string, error) {
			return `{"prompt": "sunny meadow", "negative_prompt": "(thick beard:1.3), lowres"}`, nil
		},
	}

	svc := newTestService(t, db, llmSvc, &mockSD{})

	result, err := svc.generateLLMPromptFromTags(p, "sunny meadow with flowers", "txt2img", "")
	require.NoError(t, err)
	require.NotNil(t, result)

	assert.Equal(t, "sunny meadow", result.Prompt)
	assert.Equal(t, "lowres", result.NegativePrompt)
}

func TestFilterInstructionExamples(t *testing.T) {
	t.Parallel()
	db := openTestDB(t)
	svc := newTestService(t, db, &mockLLM{}, &mockSD{})

	tests := []struct {
		name        string
		prompt      string
		instruction string
		userInput   string
		want        string
	}{
		{
			name:        "leaked example tags subtracted",
			prompt:      "(thick beard:1.3), giant luminescent mushrooms, sunny meadow",
			instruction: "Examples: (thick beard:1.3), (giant luminescent mushrooms:1.2)",
			userInput:   "sunny meadow with flowers",
			want:        "sunny meadow",
		},
		{
			name:        "user-requested example tag kept",
			prompt:      "(thick beard:1.3), forest",
			instruction: "Examples: (thick beard:1.3)",
			userInput:   "a man with a thick beard in a forest",
			want:        "(thick beard:1.3), forest",
		},
		{
			name:        "empty instruction passthrough",
			prompt:      "cat, dog",
			instruction: "",
			userInput:   "a cat",
			want:        "cat, dog",
		},
		{
			name:        "instruction without examples passthrough",
			prompt:      "cat, dog",
			instruction: "No weighted examples here at all",
			userInput:   "a cat",
			want:        "cat, dog",
		},
		{
			name:        "case-insensitive user match keeps tag",
			prompt:      "(thick beard:1.3), forest",
			instruction: "Examples: (thick beard:1.3)",
			userInput:   "a man with a THICK BEARD",
			want:        "(thick beard:1.3), forest",
		},
		{
			name:        "short generic key subtracted despite substring match",
			prompt:      "(tag:1.3), garage interior",
			instruction: "WEIGHT FORMAT — always use parentheses: (tag:1.3)",
			userInput:   "vintage car garage",
			want:        "garage interior",
		},
		{
			name:        "four char key subtracted unconditionally",
			prompt:      "(hood:1.2), forest",
			instruction: "Examples: (hood:1.2)",
			userInput:   "hooded figure in a forest",
			want:        "forest",
		},
		{
			name:        "five char key still guarded by substring match",
			prompt:      "(sword:1.2), forest",
			instruction: "Examples: (sword:1.2)",
			userInput:   "a knight with a sword in a forest",
			want:        "(sword:1.2), forest",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tt.want, svc.filterInstructionExamples(tt.prompt, tt.instruction, tt.userInput))
		})
	}
}

func TestGenerateImage_PresetNotFound(t *testing.T) {
	t.Parallel()
	db := openTestDB(t)
	svc := newTestService(t, db, &mockLLM{}, &mockSD{})
	svc.ctx = context.Background()

	_, err := svc.GenerateImage(GenerateImageParams{PresetID: 999})
	assert.Error(t, err)
}

func TestGenerateImage_Success(t *testing.T) {
	t.Parallel()
	db := openTestDB(t)
	makeTestPreset(t, db, nil)

	sdSvc := &mockSD{
		txt2img: func(req sd.Txt2ImgRequest) (*sd.Txt2ImgResponse, error) {
			return &sd.Txt2ImgResponse{
				Images:     []string{"base64imagedata"},
				Info:       json.RawMessage(`{"seed": 42}`),
				Parameters: json.RawMessage(`{}`),
			}, nil
		},
	}

	svc := newTestService(t, db, &mockLLM{}, sdSvc)
	svc.ctx = context.Background()

	result, err := svc.GenerateImage(GenerateImageParams{PresetID: 1})
	require.NoError(t, err)
	require.NotNil(t, result)
	assert.Equal(t, "base64imagedata", result.Image)
	assert.Contains(t, result.EffectivePrompt, "1girl")
	assert.False(t, result.IsPreview)
}

func TestGenerateImage_WithExtraPrompt(t *testing.T) {
	t.Parallel()
	db := openTestDB(t)
	makeTestPreset(t, db, nil)

	sdSvc := &mockSD{
		txt2img: func(req sd.Txt2ImgRequest) (*sd.Txt2ImgResponse, error) {
			assert.Equal(t, "custom prompt, BREAK, masterpiece, best quality, 1girl", req.Prompt)
			return &sd.Txt2ImgResponse{Images: []string{"img"}}, nil
		},
	}

	svc := newTestService(t, db, &mockLLM{}, sdSvc)
	svc.ctx = context.Background()

	result, err := svc.GenerateImage(GenerateImageParams{
		PresetID:    1,
		ExtraPrompt: "custom prompt",
	})
	require.NoError(t, err)
	require.NotNil(t, result)
}

func TestGenerateImage_EmptyImages(t *testing.T) {
	t.Parallel()
	db := openTestDB(t)
	makeTestPreset(t, db, nil)

	sdSvc := &mockSD{
		txt2img: func(req sd.Txt2ImgRequest) (*sd.Txt2ImgResponse, error) {
			return &sd.Txt2ImgResponse{Images: []string{}}, nil
		},
	}

	svc := newTestService(t, db, &mockLLM{}, sdSvc)
	svc.ctx = context.Background()

	_, err := svc.GenerateImage(GenerateImageParams{PresetID: 1})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "no image generated")
}

func TestGenerateImage_SDError(t *testing.T) {
	t.Parallel()
	db := openTestDB(t)
	makeTestPreset(t, db, nil)

	sdSvc := &mockSD{
		txt2img: func(req sd.Txt2ImgRequest) (*sd.Txt2ImgResponse, error) {
			return nil, fmt.Errorf("SD server error")
		},
	}

	svc := newTestService(t, db, &mockLLM{}, sdSvc)
	svc.ctx = context.Background()

	_, err := svc.GenerateImage(GenerateImageParams{PresetID: 1})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "SD server error")
}

func TestGenerateImage_WithLoRAs(t *testing.T) {
	t.Parallel()
	db := openTestDB(t)
	makeTestPreset(t, db, &preset.Preset{
		Name:           "lora-preset",
		Prompt:         "1girl",
		NegativePrompt: "lowres",
		Sampler:        "Euler a",
		Steps:          20,
		CfgScale:       7.0,
		Loras:          `[{"name":"test-lora","weight":0.8}]`,
	})

	sdSvc := &mockSD{
		txt2img: func(req sd.Txt2ImgRequest) (*sd.Txt2ImgResponse, error) {
			assert.Contains(t, req.Prompt, "<lora:test-lora:0.8>")
			return &sd.Txt2ImgResponse{Images: []string{"img"}}, nil
		},
	}

	svc := newTestService(t, db, &mockLLM{}, sdSvc)
	svc.ctx = context.Background()

	result, err := svc.GenerateImage(GenerateImageParams{PresetID: 1})
	require.NoError(t, err)
	require.NotNil(t, result)
}

func TestGenerateImage_SetsModelAndVAE(t *testing.T) {
	t.Parallel()
	db := openTestDB(t)
	makeTestPreset(t, db, &preset.Preset{
		Name:           "model-preset",
		Prompt:         "1girl",
		NegativePrompt: "lowres",
		Sampler:        "Euler a",
		Steps:          20,
		CfgScale:       7.0,
		ModelName:      "sd-xl",
		VAE:            "vae-ft-mse",
	})

	var setModelCalled, setVAECalled bool
	sdSvc := &mockSD{
		setModel: func(modelName string) error {
			setModelCalled = true
			assert.Equal(t, "sd-xl", modelName)
			return nil
		},
		setVAE: func(vaeName string) error {
			setVAECalled = true
			assert.Equal(t, "vae-ft-mse", vaeName)
			return nil
		},
		txt2img: func(req sd.Txt2ImgRequest) (*sd.Txt2ImgResponse, error) {
			return &sd.Txt2ImgResponse{Images: []string{"img"}}, nil
		},
	}

	svc := newTestService(t, db, &mockLLM{}, sdSvc)
	svc.ctx = context.Background()

	_, err := svc.GenerateImage(GenerateImageParams{PresetID: 1})
	require.NoError(t, err)
	assert.True(t, setModelCalled)
	assert.True(t, setVAECalled)
}

func TestGenerateImage_VAEFallback(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name         string
		presetVAE    string
		expectedVAE  string
	}{
		{
			name:        "explicit_vae_passed_through",
			presetVAE:   "sdxl_vae",
			expectedVAE: "sdxl_vae",
		},
		{
			name:        "empty_vae_falls_back_to_automatic",
			presetVAE:   "",
			expectedVAE: "Automatic",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			db := openTestDB(t)
			makeTestPreset(t, db, &preset.Preset{
				Name:           "vae-test",
				Prompt:         "1girl",
				NegativePrompt: "lowres",
				Sampler:        "Euler a",
				Steps:          20,
				CfgScale:       7.0,
				VAE:            tc.presetVAE,
			})

			var capturedVAE string
			sdSvc := &mockSD{
				setVAE: func(vaeName string) error {
					capturedVAE = vaeName
					return nil
				},
				txt2img: func(req sd.Txt2ImgRequest) (*sd.Txt2ImgResponse, error) {
					return &sd.Txt2ImgResponse{Images: []string{"img"}}, nil
				},
			}

			svc := newTestService(t, db, &mockLLM{}, sdSvc)
			svc.ctx = context.Background()

			_, err := svc.GenerateImage(GenerateImageParams{PresetID: 1})
			require.NoError(t, err)
			assert.Equal(t, tc.expectedVAE, capturedVAE)
		})
	}
}

func TestUpscaleImage_EmptyImage(t *testing.T) {
	t.Parallel()
	db := openTestDB(t)
	svc := newTestService(t, db, &mockLLM{}, &mockSD{})
	svc.ctx = context.Background()

	_, err := svc.UpscaleImage(UpscaleImageParams{})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "image is required")
}

func TestUpscaleImage_ImageTooLarge(t *testing.T) {
	t.Parallel()
	db := openTestDB(t)
	svc := newTestService(t, db, &mockLLM{}, &mockSD{})
	svc.ctx = context.Background()

	bigImage := make([]byte, 68*1024*1024)
	encoded := base64.StdEncoding.EncodeToString(bigImage)

	_, err := svc.UpscaleImage(UpscaleImageParams{ImageBase64: encoded})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "image too large")
}

func TestUpscaleImage_InvalidGenInfo(t *testing.T) {
	t.Parallel()
	db := openTestDB(t)
	svc := newTestService(t, db, &mockLLM{}, &mockSD{})
	svc.ctx = context.Background()

	_, err := svc.UpscaleImage(UpscaleImageParams{
		ImageBase64: "dGVzdA==",
		GenInfo:     "not json",
	})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "parse gen_info")
}

func TestUpscaleImage_InvalidDimensions(t *testing.T) {
	t.Parallel()
	db := openTestDB(t)
	svc := newTestService(t, db, &mockLLM{}, &mockSD{})
	svc.ctx = context.Background()

	genInfo := `{"prompt": "test", "negative_prompt": "neg", "sampler_name": "Euler", "seed": 1, "width": 0, "height": 0, "steps": 20, "cfg_scale": 7.0, "clip_skip": 1}`

	_, err := svc.UpscaleImage(UpscaleImageParams{
		ImageBase64: "dGVzdA==",
		GenInfo:     genInfo,
	})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "invalid dimensions")
}

func TestUpscaleImage_AlreadyMaxSize(t *testing.T) {
	t.Parallel()
	db := openTestDB(t)
	svc := newTestService(t, db, &mockLLM{}, &mockSD{})
	svc.ctx = context.Background()

	genInfo := `{"prompt": "test", "negative_prompt": "neg", "sampler_name": "Euler", "seed": 1, "width": 2049, "height": 2049, "steps": 20, "cfg_scale": 7.0, "clip_skip": 1}`

	_, err := svc.UpscaleImage(UpscaleImageParams{
		ImageBase64: "dGVzdA==",
		GenInfo:     genInfo,
	})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "already")
}

func TestUpscaleImage_Success(t *testing.T) {
	t.Parallel()
	db := openTestDB(t)
	svc := newTestService(t, db, &mockLLM{}, &mockSD{})
	svc.ctx = context.Background()

	genInfo := `{"prompt": "test prompt", "negative_prompt": "neg", "sampler_name": "Euler", "scheduler": "", "seed": 42, "width": 512, "height": 512, "steps": 20, "cfg_scale": 7.0, "clip_skip": 1}`

	sdSvc := &mockSD{
		img2img: func(req sd.Img2ImgRequest) (*sd.Txt2ImgResponse, error) {
			assert.Equal(t, 1024, req.Width)
			assert.Equal(t, 1024, req.Height)
			assert.Equal(t, "test prompt", req.Prompt)
			return &sd.Txt2ImgResponse{Images: []string{"upscaled-img"}}, nil
		},
	}
	svc.sd = sdSvc

	result, err := svc.UpscaleImage(UpscaleImageParams{
		ImageBase64: "dGVzdA==",
		GenInfo:     genInfo,
	})
	require.NoError(t, err)
	require.NotNil(t, result)
	assert.Equal(t, "upscaled-img", result.Image)
}

func TestUpscaleImage_SDReturnsEmpty(t *testing.T) {
	t.Parallel()
	db := openTestDB(t)
	sdSvc := &mockSD{
		img2img: func(req sd.Img2ImgRequest) (*sd.Txt2ImgResponse, error) {
			return &sd.Txt2ImgResponse{Images: []string{}}, nil
		},
	}
	svc := newTestService(t, db, &mockLLM{}, sdSvc)
	svc.ctx = context.Background()

	genInfo := `{"prompt": "p", "negative_prompt": "n", "sampler_name": "Euler", "seed": 1, "width": 512, "height": 512, "steps": 20, "cfg_scale": 7.0, "clip_skip": 1}`

	_, err := svc.UpscaleImage(UpscaleImageParams{
		ImageBase64: "dGVzdA==",
		GenInfo:     genInfo,
	})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "no image generated during upscale")
}

func TestGetLastImage_NoFile(t *testing.T) {
	t.Parallel()
	db := openTestDB(t)
	svc := newTestService(t, db, &mockLLM{}, &mockSD{})

	result, err := svc.GetLastImage()
	assert.NoError(t, err)
	assert.Nil(t, result)
}

func TestGetLastImage_WithFile(t *testing.T) {
	t.Parallel()
	tmpDir := t.TempDir()
	db := openTestDB(t)
	svc := newTestService(t, db, &mockLLM{}, &mockSD{})
	svc.dataDir = tmpDir

	pngData := []byte("fake png data")
	require.NoError(t, os.WriteFile(filepath.Join(tmpDir, "last_image.png"), pngData, 0644))
	meta := lastImageMeta{IsPreview: true, Info: json.RawMessage(`{"seed": 42}`)}
	metaBytes, _ := json.Marshal(meta)
	require.NoError(t, os.WriteFile(filepath.Join(tmpDir, "last_image.json"), metaBytes, 0644))

	result, err := svc.GetLastImage()
	require.NoError(t, err)
	require.NotNil(t, result)
	assert.True(t, result.IsPreview)
	assert.Equal(t, base64.StdEncoding.EncodeToString(pngData), result.Image)
}

func TestClearLastImage(t *testing.T) {
	t.Parallel()
	tmpDir := t.TempDir()
	db := openTestDB(t)
	svc := newTestService(t, db, &mockLLM{}, &mockSD{})
	svc.dataDir = tmpDir

	require.NoError(t, os.WriteFile(filepath.Join(tmpDir, "last_image.png"), []byte("data"), 0644))
	require.NoError(t, os.WriteFile(filepath.Join(tmpDir, "last_image.json"), []byte("{}"), 0644))

	svc.ClearLastImage()

	_, err := os.Stat(filepath.Join(tmpDir, "last_image.png"))
	assert.True(t, os.IsNotExist(err))
	_, err = os.Stat(filepath.Join(tmpDir, "last_image.json"))
	assert.True(t, os.IsNotExist(err))
}

func TestAnalyzeImage_EmptyImage(t *testing.T) {
	t.Parallel()
	db := openTestDB(t)
	svc := newTestService(t, db, &mockLLM{}, &mockSD{})

	_, err := svc.AnalyzeImage("", "quick")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "image is required")
}

func TestAnalyzeImage_TooLarge(t *testing.T) {
	t.Parallel()
	db := openTestDB(t)
	svc := newTestService(t, db, &mockLLM{}, &mockSD{})

	bigData := make([]byte, 23*1024*1024)
	encoded := base64.StdEncoding.EncodeToString(bigData)

	_, err := svc.AnalyzeImage(encoded, "quick")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "image too large")
}

func TestAnalyzeImage_QuickMode(t *testing.T) {
	t.Parallel()
	db := openTestDB(t)

	llmSvc := &mockLLM{
		analyzeImageFn: func(model, systemPrompt, imageBase64 string, maxTokens int) (string, error) {
			assert.Contains(t, systemPrompt, "image analyst")
			return "masterpiece, 1girl, blue eyes, long hair", nil
		},
	}

	svc := newTestService(t, db, llmSvc, &mockSD{})
	img := makePNGBase64(t, 64, 64)

	result, err := svc.AnalyzeImage(img, "quick")
	require.NoError(t, err)
	assert.Contains(t, result, "1girl")
}

func TestAnalyzeImage_QuickMode_CustomPrompt(t *testing.T) {
	t.Parallel()
	db := openTestDB(t)
	require.NoError(t, db.SetSetting("analyze_prompt", "custom quick prompt"))

	llmSvc := &mockLLM{
		analyzeImageFn: func(model, systemPrompt, imageBase64 string, maxTokens int) (string, error) {
			assert.Contains(t, systemPrompt, "custom quick prompt")
			return "masterpiece, tag", nil
		},
	}

	svc := newTestService(t, db, llmSvc, &mockSD{})
	img := makePNGBase64(t, 64, 64)

	result, err := svc.AnalyzeImage(img, "quick")
	require.NoError(t, err)
	assert.NotEmpty(t, result)
}

func TestAnalyzeImage_DescribeMode(t *testing.T) {
	t.Parallel()
	db := openTestDB(t)

	prose := "Ночной город после дождя, мост отражается в мокром асфальте."
	llmSvc := &mockLLM{
		describeFn: func(model, prompt, imageBase64 string, maxTokens int) (string, error) {
			assert.Contains(t, prompt, "литературным текстом")
			return "  " + prose + "  ", nil
		},
	}

	svc := newTestService(t, db, llmSvc, &mockSD{})
	img := makePNGBase64(t, 64, 64)

	result, err := svc.AnalyzeImage(img, "describe")
	require.NoError(t, err)
	assert.Equal(t, prose, result)
}

func TestAnalyzeImage_DescribeMode_CustomPrompt(t *testing.T) {
	t.Parallel()
	db := openTestDB(t)
	require.NoError(t, db.SetSetting("analyze_describe_prompt", "custom describe prompt"))

	llmSvc := &mockLLM{
		describeFn: func(model, prompt, imageBase64 string, maxTokens int) (string, error) {
			assert.Equal(t, "custom describe prompt", prompt)
			return "текст описания", nil
		},
	}

	svc := newTestService(t, db, llmSvc, &mockSD{})
	img := makePNGBase64(t, 64, 64)

	result, err := svc.AnalyzeImage(img, "describe")
	require.NoError(t, err)
	assert.NotEmpty(t, result)
}

func TestAnalyzeImage_QuickMode_LLMError(t *testing.T) {
	t.Parallel()
	db := openTestDB(t)

	llmSvc := &mockLLM{
		analyzeImageFn: func(model, systemPrompt, imageBase64 string, maxTokens int) (string, error) {
			return "", fmt.Errorf("vision model error")
		},
	}

	svc := newTestService(t, db, llmSvc, &mockSD{})
	img := makePNGBase64(t, 64, 64)

	_, err := svc.AnalyzeImage(img, "quick")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "vision model error")
}

func TestGetDefaultAnalyzePrompts(t *testing.T) {
	t.Parallel()
	db := openTestDB(t)
	svc := newTestService(t, db, &mockLLM{}, &mockSD{})

	prompts := svc.GetDefaultAnalyzePrompts()
	require.NotNil(t, prompts)
	assert.NotEmpty(t, prompts.SystemPrompt)
	assert.NotEmpty(t, prompts.SinglePrompt)
	assert.NotEmpty(t, prompts.DescribePrompt)
}

func TestTestGenerate_InvalidMode(t *testing.T) {
	t.Parallel()
	db := openTestDB(t)
	svc := newTestService(t, db, &mockLLM{}, &mockSD{})
	svc.ctx = context.Background()

	_, err := svc.TestGenerate(TestGenerateParams{Mode: "invalid"})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "mode must be")
}

func TestTestGenerate_NoItems(t *testing.T) {
	t.Parallel()
	db := openTestDB(t)
	svc := newTestService(t, db, &mockLLM{}, &mockSD{})
	svc.ctx = context.Background()

	_, err := svc.TestGenerate(TestGenerateParams{Mode: "presets", Prompt: "test"})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "select at least one item")
}

func TestTestGenerate_NoPrompt(t *testing.T) {
	t.Parallel()
	db := openTestDB(t)
	svc := newTestService(t, db, &mockLLM{}, &mockSD{})
	svc.ctx = context.Background()

	_, err := svc.TestGenerate(TestGenerateParams{Mode: "presets"})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "select at least one item")
}

func TestTestGenerate_DimensionsTooLarge(t *testing.T) {
	t.Parallel()
	db := openTestDB(t)
	svc := newTestService(t, db, &mockLLM{}, &mockSD{})
	svc.ctx = context.Background()

	_, err := svc.TestGenerate(TestGenerateParams{
		Mode:        "presets",
		SelectedIDs: []int64{1},
		Prompt:      "test",
		Width:       4096,
	})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "maximum dimension")
}

func TestTestGenerate_StepsTooLarge(t *testing.T) {
	t.Parallel()
	db := openTestDB(t)
	svc := newTestService(t, db, &mockLLM{}, &mockSD{})
	svc.ctx = context.Background()

	_, err := svc.TestGenerate(TestGenerateParams{
		Mode:        "presets",
		SelectedIDs: []int64{1},
		Prompt:      "test",
		Steps:       200,
	})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "maximum steps")
}

func TestTestGenerate_Success(t *testing.T) {
	t.Parallel()
	db := openTestDB(t)
	makeTestPreset(t, db, nil)

	sdSvc := &mockSD{
		txt2img: func(req sd.Txt2ImgRequest) (*sd.Txt2ImgResponse, error) {
			return &sd.Txt2ImgResponse{
				Images: []string{"test-img-base64"},
				Info:   json.RawMessage(`{"seed": 12345, "sd_model_name": "model-v1"}`),
			}, nil
		},
	}

	svc := newTestService(t, db, &mockLLM{}, sdSvc)
	svc.ctx = context.Background()

	results, err := svc.TestGenerate(TestGenerateParams{
		Mode:        "presets",
		SelectedIDs: []int64{1},
		Prompt:      "1girl, blue hair",
	})
	require.NoError(t, err)
	require.Len(t, results, 1)
	assert.Equal(t, "test-img-base64", results[0].Image)
	assert.Equal(t, int64(12345), results[0].Seed)
	assert.Equal(t, "model-v1", results[0].ModelName)
}

func TestTestGenerate_PresetNotFound(t *testing.T) {
	t.Parallel()
	db := openTestDB(t)
	svc := newTestService(t, db, &mockLLM{}, &mockSD{})
	svc.ctx = context.Background()

	results, err := svc.TestGenerate(TestGenerateParams{
		Mode:        "presets",
		SelectedIDs: []int64{999},
		Prompt:      "test",
	})
	require.NoError(t, err)
	require.Len(t, results, 1)
	assert.Contains(t, results[0].Error, "preset not found")
}

func TestTestGenerate_WithInitImage_UsesImg2Img(t *testing.T) {
	t.Parallel()
	db := openTestDB(t)
	makeTestPreset(t, db, nil)

	var calledImg2Img bool
	sdSvc := &mockSD{
		img2img: func(req sd.Img2ImgRequest) (*sd.Txt2ImgResponse, error) {
			calledImg2Img = true
			assert.NotEmpty(t, req.InitImages)
			assert.Contains(t, req.InitImages[0], "data:image/png;base64,")
			assert.Contains(t, req.Prompt, "1girl, blue hair")
			assert.NotNil(t, req.DenoisingStrength)
			assert.InDelta(t, 0.5, *req.DenoisingStrength, 0.01)
			return &sd.Txt2ImgResponse{
				Images: []string{"img2img-result"},
				Info:   json.RawMessage(`{"seed": 99}`),
			}, nil
		},
		txt2img: func(req sd.Txt2ImgRequest) (*sd.Txt2ImgResponse, error) {
			t.Fatal("Txt2Img should not be called when InitImage is set")
			return nil, nil
		},
	}

	svc := newTestService(t, db, &mockLLM{}, sdSvc)
	svc.ctx = context.Background()

	results, err := svc.TestGenerate(TestGenerateParams{
		Mode:        "presets",
		SelectedIDs: []int64{1},
		Prompt:      "1girl, blue hair",
		InitImage:   "iVBORw0KGgo=",
	})
	require.NoError(t, err)
	require.Len(t, results, 1)
	assert.Equal(t, "img2img-result", results[0].Image)
	assert.True(t, calledImg2Img)
}

func TestBuildSamplerName(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name         string
		sampler      string
		scheduleType string
		expected     string
	}{
		{"no schedule", "Euler a", "", "Euler a"},
		{"with Karras", "Euler a", "Karras", "Euler a Karras"},
		{"with Exponential", "DPM++ 2M", "Exponential", "DPM++ 2M Exponential"},
		{"lowercase schedule", "Euler", "karras", "Euler Karras"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			result := promptutil.BuildSamplerName(tt.sampler, tt.scheduleType)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestAppendLorasToPrompt(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		prompt   string
		loras    string
		expected string
	}{
		{"empty loras", "test prompt", "", "test prompt"},
		{"single lora", "test prompt", `[{"name":"detail","weight":0.8}]`, "test prompt <lora:detail:0.8>"},
		{"multiple loras", "test prompt", `[{"name":"a","weight":1},{"name":"b","weight":0.5}]`, "test prompt <lora:a:1> <lora:b:0.5>"},
		{"invalid json", "test prompt", "not json", "test prompt"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			result := appendLorasToPrompt(tt.prompt, tt.loras)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestExtractEmbeddedNegative(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name              string
		prompt            string
		negativePrompt    string
		expectedPrompt    string
		expectedNegative  string
	}{
		{
			"no embedded negative",
			"1girl, blue hair",
			"lowres",
			"1girl, blue hair",
			"lowres",
		},
		{
			"embedded negative",
			`1girl, blue hair, negative_prompt: "blurry, bad"`,
			"lowres",
			"1girl, blue hair",
			"blurry, bad, lowres",
		},
		{
			"embedded negative empty existing",
			`1girl, blue hair, negative_prompt: "blurry"`,
			"",
			"1girl, blue hair",
			"blurry",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			result := &GenerateSDPromptResult{
				Prompt:         tt.prompt,
				NegativePrompt: tt.negativePrompt,
			}
			extractEmbeddedNegative(result)
			assert.Equal(t, tt.expectedPrompt, result.Prompt)
			assert.Equal(t, tt.expectedNegative, result.NegativePrompt)
		})
	}
}

func TestPadToAspectRatio(t *testing.T) {
	t.Parallel()

	img := makePNGBase64(t, 100, 100)

	t.Run("same ratio no padding", func(t *testing.T) {
		t.Parallel()
		result, err := padToAspectRatio(img, 200, 200)
		assert.NoError(t, err)
		assert.Equal(t, img, result)
	})

	t.Run("different ratio adds padding", func(t *testing.T) {
		t.Parallel()
		result, err := padToAspectRatio(img, 200, 100)
		assert.NoError(t, err)
		assert.NotEqual(t, img, result)
	})

	t.Run("invalid base64", func(t *testing.T) {
		t.Parallel()
		_, err := padToAspectRatio("not-valid-base64!!!", 512, 512)
		assert.Error(t, err)
	})
}

func TestGetPreviewDimensions(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name           string
		previewMode    string
		previewW       string
		previewH       string
		presetW        int
		presetH        int
		expectPreview  bool
	}{
		{"preview off", "false", "", "", 512, 512, false},
		{"preview on square", "true", "512", "512", 1024, 1024, true},
		{"preview on landscape", "true", "512", "512", 1024, 768, true},
		{"preview on portrait", "true", "512", "512", 768, 1024, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			db := openTestDB(t)
			if tt.previewMode != "" {
				require.NoError(t, db.SetSetting("preview_mode", tt.previewMode))
			}
			if tt.previewW != "" {
				require.NoError(t, db.SetSetting("preview_width", tt.previewW))
			}
			if tt.previewH != "" {
				require.NoError(t, db.SetSetting("preview_height", tt.previewH))
			}

			svc := newTestService(t, db, &mockLLM{}, &mockSD{})
			w, h, preview := svc.getPreviewDimensions(tt.presetW, tt.presetH)
			assert.Equal(t, tt.expectPreview, preview)
			if preview {
				assert.LessOrEqual(t, w, 512)
				assert.LessOrEqual(t, h, 512)
				assert.Equal(t, 0, w%8)
				assert.Equal(t, 0, h%8)
				assert.GreaterOrEqual(t, w, 64)
				assert.GreaterOrEqual(t, h, 64)
			}
		})
	}
}

func TestGetMaxTokens(t *testing.T) {
	t.Parallel()

	t.Run("default", func(t *testing.T) {
		t.Parallel()
		db := openTestDB(t)
		svc := newTestService(t, db, &mockLLM{}, &mockSD{})
		assert.Equal(t, 512, svc.getMaxTokens())
	})

	t.Run("from setting", func(t *testing.T) {
		t.Parallel()
		db := openTestDB(t)
		require.NoError(t, db.SetSetting("llm_max_tokens", "1024"))
		svc := newTestService(t, db, &mockLLM{}, &mockSD{})
		assert.Equal(t, 1024, svc.getMaxTokens())
	})

	t.Run("invalid setting uses default", func(t *testing.T) {
		t.Parallel()
		db := openTestDB(t)
		require.NoError(t, db.SetSetting("llm_max_tokens", "not-a-number"))
		svc := newTestService(t, db, &mockLLM{}, &mockSD{})
		assert.Equal(t, 512, svc.getMaxTokens())
	})
}

func TestGetGenerateModel(t *testing.T) {
	t.Parallel()

	t.Run("default from config", func(t *testing.T) {
		t.Parallel()
		db := openTestDB(t)
		svc := newTestService(t, db, &mockLLM{}, &mockSD{})
		assert.Equal(t, "test-model", svc.getGenerateModel())
	})

	t.Run("from setting override", func(t *testing.T) {
		t.Parallel()
		db := openTestDB(t)
		require.NoError(t, db.SetSetting("llm_generate_model", "custom-model"))
		svc := newTestService(t, db, &mockLLM{}, &mockSD{})
		assert.Equal(t, "custom-model", svc.getGenerateModel())
	})
}

func TestGetAnalyzeModel(t *testing.T) {
	t.Parallel()

	t.Run("default vision model", func(t *testing.T) {
		t.Parallel()
		db := openTestDB(t)
		svc := newTestService(t, db, &mockLLM{}, &mockSD{})
		assert.Equal(t, "test-vision", svc.getAnalyzeModel())
	})

	t.Run("setting override", func(t *testing.T) {
		t.Parallel()
		db := openTestDB(t)
		require.NoError(t, db.SetSetting("llm_analyze_model", "custom-vision"))
		svc := newTestService(t, db, &mockLLM{}, &mockSD{})
		assert.Equal(t, "custom-vision", svc.getAnalyzeModel())
	})

	t.Run("falls back to generate model when vision empty", func(t *testing.T) {
		t.Parallel()
		db := openTestDB(t)
		svc := newTestService(t, db, &mockLLM{}, &mockSD{})
		svc.cfg.VisionModel = ""
		require.NoError(t, db.SetSetting("llm_generate_model", "fallback-model"))
		assert.Equal(t, "fallback-model", svc.getAnalyzeModel())
	})
}

func TestGetSDPromptInstruction(t *testing.T) {
	t.Parallel()

	t.Run("default", func(t *testing.T) {
		t.Parallel()
		db := openTestDB(t)
		svc := newTestService(t, db, &mockLLM{}, &mockSD{})
		result := svc.getSDPromptInstruction()
		assert.Contains(t, result, "Stable Diffusion")
	})

	t.Run("custom override", func(t *testing.T) {
		t.Parallel()
		db := openTestDB(t)
		require.NoError(t, db.SetSetting("sd_prompt_instruction", "custom instruction"))
		svc := newTestService(t, db, &mockLLM{}, &mockSD{})
		assert.Equal(t, "custom instruction", svc.getSDPromptInstruction())
	})
}

func TestInterruptGeneration(t *testing.T) {
	t.Parallel()
	db := openTestDB(t)
	svc := newTestService(t, db, &mockLLM{}, &mockSD{})

	err := svc.InterruptGeneration()
	assert.NoError(t, err)

	assert.Error(t, svc.checkSDInterrupted())
}

func TestSetContext(t *testing.T) {
	t.Parallel()
	db := openTestDB(t)
	svc := newTestService(t, db, &mockLLM{}, &mockSD{})

	ctx := context.Background()
	svc.SetContext(ctx)
	assert.Equal(t, ctx, svc.ctx)
}

func TestIntPtr(t *testing.T) {
	t.Parallel()

	result := intPtr(5)
	require.NotNil(t, result)
	assert.Equal(t, 5, *result)
}

func TestGenerateFromImage_Validation(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		params  GenerateFromImageParams
		errMsg  string
	}{
		{"empty image", GenerateFromImageParams{ImageBase64: ""}, "image is required"},
		{"invalid gen_mode", GenerateFromImageParams{ImageBase64: "dGVzdA==", GenMode: "bad"}, "gen_mode must be preset or compound"},
		{"invalid mode", GenerateFromImageParams{ImageBase64: "dGVzdA==", GenMode: "preset", Mode: "bad", PresetID: 1}, "mode must be txt2img, img2img or inpaint"},
		{"inpaint without mask", GenerateFromImageParams{ImageBase64: "dGVzdA==", GenMode: "preset", Mode: "inpaint", PresetID: 1}, "mask is required for inpaint mode"},
		{"preset mode no preset", GenerateFromImageParams{ImageBase64: "dGVzdA==", GenMode: "preset", Mode: "txt2img"}, "preset is required"},
		{"compound mode no compound", GenerateFromImageParams{ImageBase64: "dGVzdA==", GenMode: "compound", Mode: "txt2img"}, "compound preset is required"},
		{"invalid output_mode", GenerateFromImageParams{ImageBase64: "dGVzdA==", GenMode: "preset", Mode: "img2img", PresetID: 1, OutputMode: "bad"}, "output_mode must be standard or quality"},
		{"quality with txt2img mode", GenerateFromImageParams{ImageBase64: "dGVzdA==", GenMode: "preset", Mode: "txt2img", PresetID: 1, OutputMode: "quality"}, "quality output mode requires img2img preset mode"},
		{"quality with compound gen_mode", GenerateFromImageParams{ImageBase64: "dGVzdA==", GenMode: "compound", Mode: "img2img", CompoundPresetID: 1, OutputMode: "quality"}, "quality output mode requires img2img preset mode"},
		{"quality with remove object", GenerateFromImageParams{ImageBase64: "dGVzdA==", GenMode: "preset", Mode: "img2img", PresetID: 1, RemoveObject: true, OutputMode: "quality"}, "quality output mode requires img2img preset mode"},
		{"quality without resolution", GenerateFromImageParams{ImageBase64: "dGVzdA==", GenMode: "preset", Mode: "img2img", PresetID: 1, OutputMode: "quality"}, "quality output mode requires resolution"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			db := openTestDB(t)
			svc := newTestService(t, db, &mockLLM{}, &mockSD{})
			svc.ctx = context.Background()

			_, err := svc.GenerateFromImage(tt.params)
			assert.Error(t, err)
			assert.Contains(t, err.Error(), tt.errMsg)
		})
	}
}

func TestGenerateFromImage_ImageTooLarge(t *testing.T) {
	t.Parallel()
	db := openTestDB(t)
	svc := newTestService(t, db, &mockLLM{}, &mockSD{})
	svc.ctx = context.Background()

	bigData := make([]byte, 23*1024*1024)
	encoded := base64.StdEncoding.EncodeToString(bigData)

	_, err := svc.GenerateFromImage(GenerateFromImageParams{
		ImageBase64: encoded,
		GenMode:     "preset",
		Mode:        "txt2img",
		PresetID:    1,
	})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "image too large")
}

func TestGenerateCompoundImage_CompoundNotFound(t *testing.T) {
	t.Parallel()
	db := openTestDB(t)
	svc := newTestService(t, db, &mockLLM{}, &mockSD{})
	svc.ctx = context.Background()

	_, err := svc.GenerateCompoundImage(GenerateCompoundImageParams{CompoundPresetID: 999})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "compound preset not found")
}

func TestGenerateCompoundImage_EmptySteps(t *testing.T) {
	t.Parallel()
	db := openTestDB(t)

	cp := &preset.CompoundPreset{Name: "empty", Description: "no steps"}
	require.NoError(t, db.CreateCompoundPreset(cp))

	svc := newTestService(t, db, &mockLLM{}, &mockSD{})
	svc.ctx = context.Background()

	_, err := svc.GenerateCompoundImage(GenerateCompoundImageParams{CompoundPresetID: cp.ID})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "no steps")
}

func TestGenerateCompoundImage_Success(t *testing.T) {
	db := openTestDB(t)
	p := makeTestPreset(t, db, nil)

	cp := &preset.CompoundPreset{
		Name:        "two-step",
		Description: "two steps",
		Steps: []preset.CompoundPresetStep{
			{PresetID: p.ID},
			{PresetID: p.ID, DenoisingStrength: 0.6},
		},
	}
	require.NoError(t, db.CreateCompoundPreset(cp))

	sdSvc := &mockSD{
		txt2img: func(req sd.Txt2ImgRequest) (*sd.Txt2ImgResponse, error) {
			return &sd.Txt2ImgResponse{Images: []string{"step1-img"}}, nil
		},
		img2img: func(req sd.Img2ImgRequest) (*sd.Txt2ImgResponse, error) {
			assert.Equal(t, "step1-img", req.InitImages[0])
			return &sd.Txt2ImgResponse{Images: []string{"step2-img"}}, nil
		},
	}

	llmSvc := &mockLLM{
		genSDPromptFn: func(systemPrompt, userMessage, presetType, model string, maxTokens int) (string, error) {
			return `{"prompt":"llm-scene-tags","negative_prompt":"llm-neg"}`, nil
		},
	}

	svc := newTestService(t, db, llmSvc, sdSvc)
	svc.ctx = context.Background()

	result, err := svc.GenerateCompoundImage(GenerateCompoundImageParams{
		CompoundPresetID:    cp.ID,
		ExtraPrompt:         "custom prompt",
		ExtraNegativePrompt: "extra neg",
	})
	require.NoError(t, err)
	require.NotNil(t, result)
	assert.Equal(t, "step2-img", result.Image)
}

func TestGenerateFromImage_CompoundHires(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name          string
		steps         int
		hiresProfile  bool
		expectUpscale int
		expectedImage string
		expectedInfo  string
	}{
		{"multi-step with hires", 2, true, 1, "hires-img", `{"seed":3}`},
		{"multi-step without hires", 2, false, 0, "step2-img", `{"seed":2}`},
		{"single-step with hires", 1, true, 1, "hires-img", `{"seed":3}`},
		{"single-step without hires", 1, false, 0, "step1-img", `{"seed":1}`},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			db := openTestDB(t)
			p := makeTestPreset(t, db, nil)

			steps := make([]preset.CompoundPresetStep, 0, tt.steps)
			for i := 0; i < tt.steps; i++ {
				steps = append(steps, preset.CompoundPresetStep{PresetID: p.ID, DenoisingStrength: 0.6})
			}
			cp := &preset.CompoundPreset{Name: "from-image-cp", Description: "from image", Steps: steps}
			require.NoError(t, db.CreateCompoundPreset(cp))

			var hiresID *int64
			if tt.hiresProfile {
				hp := &preset.HiresProfile{Name: "cp-hires", Upscale: 2.0, DenoisingStrength: 0.5, Upscaler: "R-ESRGAN 4x+"}
				require.NoError(t, db.CreateHiresProfile(hp))
				hiresID = &hp.ID
			}

			srcImg := makePNGBase64(t, 512, 512)

			upscaleCalls := 0
			sdSvc := &mockSD{
				upscale: func(base64Img string, upscaler string, scale float64) (string, error) {
					upscaleCalls++
					assert.Equal(t, "R-ESRGAN 4x+", upscaler)
					assert.InDelta(t, 2.0, scale, 0.01)
					return "upscaled-b64", nil
				},
				img2img: func(req sd.Img2ImgRequest) (*sd.Txt2ImgResponse, error) {
					switch req.InitImages[0] {
					case srcImg:
						return &sd.Txt2ImgResponse{Images: []string{"step1-img"}, Info: json.RawMessage(`{"seed":1}`)}, nil
					case "step1-img":
						return &sd.Txt2ImgResponse{Images: []string{"step2-img"}, Info: json.RawMessage(`{"seed":2}`)}, nil
					case "upscaled-b64":
						return &sd.Txt2ImgResponse{Images: []string{"hires-img"}, Info: json.RawMessage(`{"seed":3}`)}, nil
					}
					return nil, fmt.Errorf("unexpected init image: %q", req.InitImages[0])
				},
			}

			svc := newTestService(t, db, &mockLLM{
				genSDPromptFn: func(systemPrompt, userMessage, presetType, model string, maxTokens int) (string, error) {
					return `{"prompt":"llm-scene-tags","negative_prompt":"llm-neg"}`, nil
				},
			}, sdSvc)
			svc.ctx = context.Background()

			result, err := svc.GenerateFromImage(GenerateFromImageParams{
				ImageBase64:      srcImg,
				Mode:             "img2img",
				GenMode:          "compound",
				CompoundPresetID: cp.ID,
				Tags:             "custom prompt",
				HiresProfileID:   hiresID,
			})
			require.NoError(t, err)
			require.NotNil(t, result)
			assert.Equal(t, tt.expectedImage, result.Image)
			assert.Equal(t, tt.expectUpscale, upscaleCalls)
			assert.NotEmpty(t, result.Info)
			assert.Equal(t, tt.expectedInfo, string(result.Info))
		})
	}
}

func TestTestCompoundGenerate_Validation(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name   string
		params TestCompoundGenerateParams
		errMsg string
	}{
		{"no ids", TestCompoundGenerateParams{Prompt: "test"}, "select at least one"},
		{"too many ids", TestCompoundGenerateParams{SelectedIDs: make([]int64, 21), Prompt: "test"}, "maximum 20"},
		{"no prompt no preset", TestCompoundGenerateParams{}, "select at least one compound preset"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			db := openTestDB(t)
			svc := newTestService(t, db, &mockLLM{}, &mockSD{})
			svc.ctx = context.Background()

			_, err := svc.TestCompoundGenerate(tt.params)
			assert.Error(t, err)
			assert.Contains(t, err.Error(), tt.errMsg)
		})
	}
}

func TestTestCompoundGenerate_CompoundNotFound(t *testing.T) {
	t.Parallel()
	db := openTestDB(t)
	svc := newTestService(t, db, &mockLLM{}, &mockSD{})
	svc.ctx = context.Background()

	results, err := svc.TestCompoundGenerate(TestCompoundGenerateParams{
		SelectedIDs: []int64{999},
		Prompt:      "test",
	})
	require.NoError(t, err)
	require.Len(t, results, 1)
	assert.Contains(t, results[0].Error, "compound preset not found")
}

func TestTestCompoundGenerate_Success(t *testing.T) {
	t.Parallel()
	db := openTestDB(t)
	p := makeTestPreset(t, db, nil)

	cp := &preset.CompoundPreset{
		Name:        "test-cp",
		Description: "test",
		Steps: []preset.CompoundPresetStep{
			{PresetID: p.ID},
		},
	}
	require.NoError(t, db.CreateCompoundPreset(cp))

	sdSvc := &mockSD{
		txt2img: func(req sd.Txt2ImgRequest) (*sd.Txt2ImgResponse, error) {
			return &sd.Txt2ImgResponse{Images: []string{"result-img"}}, nil
		},
	}

	svc := newTestService(t, db, &mockLLM{
		genSDPromptFn: func(systemPrompt, userMessage, presetType, model string, maxTokens int) (string, error) {
			return `{"prompt":"1girl, solo, standing","negative_prompt":"lowres, bad"}`, nil
		},
	}, sdSvc)
	svc.ctx = context.Background()

	results, err := svc.TestCompoundGenerate(TestCompoundGenerateParams{
		SelectedIDs:    []int64{cp.ID},
		Prompt:         "1girl",
		NegativePrompt: "lowres",
	})
	require.NoError(t, err)
	require.Len(t, results, 1)
	assert.Equal(t, "test-cp", results[0].Name)
	assert.Equal(t, "result-img", results[0].Image)
	assert.Empty(t, results[0].Error)
}

func floatPtr(v float64) *float64 { return &v }

func TestGenerateImage_ForgeHiresFallback(t *testing.T) {
	t.Parallel()
	db := openTestDB(t)
	makeTestPreset(t, db, &preset.Preset{
		Name:           "forge-preset",
		Prompt:         "1girl, masterpiece",
		NegativePrompt: "lowres",
		Sampler:        "Euler a",
		Steps:          20,
		CfgScale:       7.0,
	})

	hp := &preset.HiresProfile{Name: "test-hires", Upscale: 2.0, DenoisingStrength: 0.5, Upscaler: "Latent"}
	require.NoError(t, db.CreateHiresProfile(hp))

	callCount := 0
	var firstCallReq, secondCallReq sd.Txt2ImgRequest
	sdSvc := &mockSD{
		txt2img: func(req sd.Txt2ImgRequest) (*sd.Txt2ImgResponse, error) {
			callCount++
			if callCount == 1 {
				firstCallReq = req
				return nil, fmt.Errorf(
					"request failed after 3 attempts: status 500\nSD response: {\"error\":\"TypeError\",\"detail\":\"\",\"body\":\"\",\"message\":\"argument of type 'NoneType' is not iterable\"}",
				)
			}
			secondCallReq = req
			return &sd.Txt2ImgResponse{
				Images:     []string{"fallback-image-data"},
				Info:       json.RawMessage(`{"seed":123}`),
				Parameters: json.RawMessage(`{}`),
			}, nil
		},
	}

	svc := newTestService(t, db, &mockLLM{}, sdSvc)
	svc.ctx = context.Background()

	result, err := svc.GenerateImage(GenerateImageParams{PresetID: 1, HiresProfileID: &hp.ID})
	require.NoError(t, err)
	require.NotNil(t, result)
	assert.Equal(t, "fallback-image-data", result.Image)
	assert.Equal(t, 2, callCount)

	assert.NotNil(t, firstCallReq.HiresFix)
	assert.True(t, *firstCallReq.HiresFix)
	assert.NotNil(t, firstCallReq.HiresUpscale)
	assert.NotNil(t, firstCallReq.HiresDenoisingStrength)
	assert.Equal(t, "Latent", firstCallReq.HiresUpscaler)

	assert.Nil(t, secondCallReq.HiresFix)
	assert.Nil(t, secondCallReq.HiresUpscale)
	assert.Nil(t, secondCallReq.HiresDenoisingStrength)
	assert.Equal(t, "", secondCallReq.HiresUpscaler)
	assert.Zero(t, secondCallReq.HiresResizeX)
	assert.Zero(t, secondCallReq.HiresResizeY)
	assert.Zero(t, secondCallReq.HiresSecondPassSteps)
}

func TestGenerateImage_ForgeHiresManualUpscale(t *testing.T) {
	t.Parallel()
	db := openTestDB(t)
	makeTestPreset(t, db, &preset.Preset{
		Name:           "forge-manual",
		Prompt:         "1girl, masterpiece",
		NegativePrompt: "lowres",
		Sampler:        "Euler a",
		Steps:          20,
		CfgScale:       7.0,
	})

	hp := &preset.HiresProfile{Name: "manual-hires", Upscale: 2.0, DenoisingStrength: 0.5, Upscaler: "Latent"}
	require.NoError(t, db.CreateHiresProfile(hp))

	validB64 := tinyPNGBase64(t)

	txt2imgCalls := 0
	sdSvc := &mockSD{
		txt2img: func(req sd.Txt2ImgRequest) (*sd.Txt2ImgResponse, error) {
			txt2imgCalls++
			if txt2imgCalls == 1 {
				return nil, fmt.Errorf("status 500: hires not supported")
			}
			return &sd.Txt2ImgResponse{
				Images:     []string{validB64},
				Info:       json.RawMessage(`{"seed":123}`),
				Parameters: json.RawMessage(`{}`),
			}, nil
		},
		img2img: func(req sd.Img2ImgRequest) (*sd.Txt2ImgResponse, error) {
			assert.Equal(t, 1024, req.Width)
			assert.Equal(t, 1024, req.Height)
			assert.NotNil(t, req.DenoisingStrength)
			assert.InDelta(t, 0.5, *req.DenoisingStrength, 0.01)
			return &sd.Txt2ImgResponse{
				Images:     []string{"upscaled-image-data"},
				Info:       json.RawMessage(`{"seed":123}`),
				Parameters: json.RawMessage(`{}`),
			}, nil
		},
	}

	svc := newTestService(t, db, &mockLLM{}, sdSvc)
	svc.ctx = context.Background()

	result, err := svc.GenerateImage(GenerateImageParams{PresetID: 1, HiresProfileID: &hp.ID})
	require.NoError(t, err)
	require.NotNil(t, result)
	assert.Equal(t, "upscaled-image-data", result.Image)
	assert.False(t, result.HiresFixSkipped)
	assert.True(t, result.HiresFixManual)
	assert.Equal(t, 2, txt2imgCalls)
}

func TestGenerateImage_ForgeErrorNoHiresFix(t *testing.T) {
	t.Parallel()
	db := openTestDB(t)
	makeTestPreset(t, db, &preset.Preset{
		Name:           "no-hires-preset",
		Prompt:         "1girl",
		NegativePrompt: "lowres",
		Sampler:        "Euler a",
		Steps:          20,
		CfgScale:       7.0,
	})

	callCount := 0
	sdSvc := &mockSD{
		txt2img: func(req sd.Txt2ImgRequest) (*sd.Txt2ImgResponse, error) {
			callCount++
			return nil, fmt.Errorf(
				"request failed after 3 attempts: status 500\nSD response: {\"error\":\"TypeError\",\"detail\":\"\",\"body\":\"\",\"message\":\"argument of type 'NoneType' is not iterable\"}",
			)
		},
	}

	svc := newTestService(t, db, &mockLLM{}, sdSvc)
	svc.ctx = context.Background()

	_, err := svc.GenerateImage(GenerateImageParams{PresetID: 1})
	assert.Error(t, err)
	assert.Equal(t, 1, callCount)
}

func TestGenerateImage_HiresFallbackStillFails(t *testing.T) {
	t.Parallel()
	db := openTestDB(t)
	makeTestPreset(t, db, &preset.Preset{
		Name:           "hires-fail-preset",
		Prompt:         "1girl",
		NegativePrompt: "lowres",
		Sampler:        "Euler a",
		Steps:          20,
		CfgScale:       7.0,
	})

	hp := &preset.HiresProfile{Name: "fail-hires", Upscale: 2.0, DenoisingStrength: 0.5, Upscaler: "Latent"}
	require.NoError(t, db.CreateHiresProfile(hp))

	callCount := 0
	sdSvc := &mockSD{
		txt2img: func(req sd.Txt2ImgRequest) (*sd.Txt2ImgResponse, error) {
			callCount++
			return nil, fmt.Errorf("SD server error")
		},
	}

	svc := newTestService(t, db, &mockLLM{}, sdSvc)
	svc.ctx = context.Background()

	_, err := svc.GenerateImage(GenerateImageParams{PresetID: 1, HiresProfileID: &hp.ID})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "SD server error")
	assert.Equal(t, 2, callCount)
}

func TestGetSDPromptInstructionFor(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		modelName string
		setup     func(t *testing.T, db *preset.DB)
		wantFn    func(t *testing.T, svc *Service) string
	}{
		{
			name:      "flux model uses default prose instruction",
			modelName: "flux1-dev-fp8",
			wantFn: func(t *testing.T, svc *Service) string {
				return config.DefaultSDPromptInstructionProse
			},
		},
		{
			name:      "tagged model uses tagged instruction",
			modelName: "epicrealismXL_pureFix",
			wantFn: func(t *testing.T, svc *Service) string {
				return svc.getSDPromptInstruction()
			},
		},
		{
			name:      "empty model name uses tagged instruction",
			modelName: "",
			wantFn: func(t *testing.T, svc *Service) string {
				return svc.getSDPromptInstruction()
			},
		},
		{
			name:      "custom prose instruction from db",
			modelName: "flux-dev",
			setup: func(t *testing.T, db *preset.DB) {
				require.NoError(t, db.SetSetting("sd_prompt_instruction_prose", "CUSTOM PROSE"))
			},
			wantFn: func(t *testing.T, svc *Service) string {
				return "CUSTOM PROSE"
			},
		},
		{
			name:      "custom prose models csv overrides default prose list",
			modelName: "flux-dev",
			setup: func(t *testing.T, db *preset.DB) {
				require.NoError(t, db.SetSetting("prose_models", "megatron"))
			},
			wantFn: func(t *testing.T, svc *Service) string {
				return svc.getSDPromptInstruction()
			},
		},
		{
			name:      "model matching custom prose models csv uses prose",
			modelName: "megatron-xl",
			setup: func(t *testing.T, db *preset.DB) {
				require.NoError(t, db.SetSetting("prose_models", "megatron"))
			},
			wantFn: func(t *testing.T, svc *Service) string {
				return config.DefaultSDPromptInstructionProse
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			db := openTestDB(t)
			if tt.setup != nil {
				tt.setup(t, db)
			}
			svc := newTestService(t, db, &mockLLM{}, &mockSD{})
			assert.Equal(t, tt.wantFn(t, svc), svc.getSDPromptInstructionFor(tt.modelName))
		})
	}
}

func TestUserSceneLabel(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		modelName string
		contains  string
	}{
		{name: "prose model asks for connected english paragraph", modelName: "flux1-dev-fp8", contains: "connected English paragraph"},
		{name: "tagged model asks for sd tags with weights", modelName: "sd_xl_base_1.0", contains: "SD tags with weights"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			db := openTestDB(t)
			svc := newTestService(t, db, &mockLLM{}, &mockSD{})
			assert.Contains(t, svc.userSceneLabel(tt.modelName), tt.contains)
		})
	}
}

func TestPromptTruncateLimit(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		modelName string
		want      int
	}{
		{name: "prose model limit", modelName: "flux1-dev-fp8", want: 2000},
		{name: "tagged model limit", modelName: "riMix", want: 1000},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			db := openTestDB(t)
			svc := newTestService(t, db, &mockLLM{}, &mockSD{})
			assert.Equal(t, tt.want, svc.promptTruncateLimit(tt.modelName))
		})
	}
}

func TestGenerateImage_AppliesOverrides(t *testing.T) {
	t.Parallel()
	db := openTestDB(t)
	makeTestPreset(t, db, &preset.Preset{
		Name:           "override-preset",
		Prompt:         "1girl",
		NegativePrompt: "lowres",
		Sampler:        "Euler a",
		ScheduleType:   "Karras",
		Steps:          20,
		CfgScale:       7.0,
		ClipSkip:       intPtr(1),
	})

	var capturedReq sd.Txt2ImgRequest
	sdSvc := &mockSD{
		txt2img: func(req sd.Txt2ImgRequest) (*sd.Txt2ImgResponse, error) {
			capturedReq = req
			return &sd.Txt2ImgResponse{Images: []string{"img"}}, nil
		},
	}

	svc := newTestService(t, db, &mockLLM{}, sdSvc)
	svc.ctx = context.Background()

	_, err := svc.GenerateImage(GenerateImageParams{
		PresetID: 1,
		Overrides: &PresetOverrides{
			Sampler:      strPtr("DPM++ 2M"),
			ScheduleType: strPtr("Exponential"),
			Steps:        intPtr(33),
			CfgScale:     floatPtr(4.5),
			ClipSkip:     intPtr(3),
		},
	})
	require.NoError(t, err)
	assert.Equal(t, "DPM++ 2M Exponential", capturedReq.SamplerName)
	assert.Equal(t, "Exponential", capturedReq.Scheduler)
	assert.Equal(t, 33, capturedReq.Steps)
	assert.Equal(t, 4.5, capturedReq.CfgScale)
	require.NotNil(t, capturedReq.ClipSkip)
	assert.Equal(t, 3, *capturedReq.ClipSkip)
}

func TestGenerateImage_OverrideModel_SetsModel(t *testing.T) {
	t.Parallel()
	db := openTestDB(t)
	makeTestPreset(t, db, &preset.Preset{
		Name:           "model-override-preset",
		Prompt:         "1girl",
		NegativePrompt: "lowres",
		Sampler:        "Euler a",
		Steps:          20,
		CfgScale:       7.0,
		ModelName:      "sd-xl",
	})

	var setModels []string
	sdSvc := &mockSD{
		setModel: func(modelName string) error {
			setModels = append(setModels, modelName)
			return nil
		},
		txt2img: func(req sd.Txt2ImgRequest) (*sd.Txt2ImgResponse, error) {
			return &sd.Txt2ImgResponse{Images: []string{"img"}}, nil
		},
	}

	svc := newTestService(t, db, &mockLLM{}, sdSvc)
	svc.ctx = context.Background()

	_, err := svc.GenerateImage(GenerateImageParams{
		PresetID: 1,
		Overrides: &PresetOverrides{
			ModelName: strPtr("flux1-dev-fp8"),
		},
	})
	require.NoError(t, err)
	assert.Equal(t, []string{"flux1-dev-fp8"}, setModels)
}

func TestGenerateFromImage_AppliesOverrides(t *testing.T) {
	t.Parallel()
	db := openTestDB(t)
	makeTestPreset(t, db, &preset.Preset{
		Name:           "from-image-override-preset",
		Prompt:         "1girl",
		NegativePrompt: "lowres",
		Sampler:        "Euler a",
		ScheduleType:   "Karras",
		Steps:          20,
		CfgScale:       7.0,
		ClipSkip:       intPtr(1),
		Loras:          `[{"name":"test-lora","weight":0.8}]`,
	})

	var capturedReq sd.Img2ImgRequest
	sdSvc := &mockSD{
		img2img: func(req sd.Img2ImgRequest) (*sd.Txt2ImgResponse, error) {
			capturedReq = req
			return &sd.Txt2ImgResponse{Images: []string{"remix-img"}}, nil
		},
	}

	svc := newTestService(t, db, &mockLLM{}, sdSvc)
	svc.ctx = context.Background()

	_, err := svc.GenerateFromImage(GenerateFromImageParams{
		ImageBase64:       makePNGBase64(t, 64, 64),
		GenMode:           "preset",
		Mode:              "img2img",
		PresetID:          1,
		DenoisingStrength: 0.6,
		Overrides: &PresetOverrides{
			Sampler:      strPtr("DPM++ 2M"),
			ScheduleType: strPtr("Exponential"),
			Steps:        intPtr(33),
			CfgScale:     floatPtr(4.5),
			ClipSkip:     intPtr(3),
			Loras:        strPtr("[]"),
		},
	})
	require.NoError(t, err)
	assert.Equal(t, "DPM++ 2M Exponential", capturedReq.SamplerName)
	assert.Equal(t, "Exponential", capturedReq.Scheduler)
	assert.Equal(t, 33, capturedReq.Steps)
	assert.Equal(t, 4.5, capturedReq.CfgScale)
	require.NotNil(t, capturedReq.ClipSkip)
	assert.Equal(t, 3, *capturedReq.ClipSkip)
	assert.NotContains(t, capturedReq.Prompt, "<lora:")
}

func TestGenerateSDPrompt_OverrideModel_UsesProseInstruction(t *testing.T) {
	t.Parallel()
	db := openTestDB(t)
	makeTestPreset(t, db, &preset.Preset{
		Name:           "prose-override-preset",
		PresetType:     "anime",
		Prompt:         "masterpiece, best quality, 1girl",
		NegativePrompt: "lowres, bad anatomy",
		Sampler:        "Euler a",
		Steps:          20,
		CfgScale:       7.0,
		ModelName:      "epicrealismXL_pureFix",
	})

	var capturedSystemPrompt, capturedUserMessage string
	llmSvc := &mockLLM{
		genSDPromptFn: func(systemPrompt, userMessage, presetType, model string, maxTokens int) (string, error) {
			capturedSystemPrompt = systemPrompt
			capturedUserMessage = userMessage
			return `{"prompt": "a woman standing in a garden", "negative_prompt": "lowres"}`, nil
		},
	}

	svc := newTestService(t, db, llmSvc, &mockSD{})

	_, err := svc.GenerateSDPrompt(GenerateSDPromptParams{
		PresetID:    1,
		Description: "a woman standing in a garden",
		Overrides: &PresetOverrides{
			ModelName: strPtr("flux1-dev-fp8"),
		},
	})
	require.NoError(t, err)
	assert.True(t, strings.HasPrefix(capturedSystemPrompt, config.DefaultSDPromptInstructionProse))
	assert.Contains(t, capturedUserMessage, "connected English paragraph")
}

func int64Ptr(v int64) *int64 { return &v }

func TestGenerateFromImage_ExplicitSeed(t *testing.T) {
	t.Parallel()
	db := openTestDB(t)
	makeTestPreset(t, db, &preset.Preset{
		Name:     "seed-preset",
		Prompt:   "1girl",
		Sampler:  "Euler a",
		Steps:    20,
		CfgScale: 7.0,
		Seed:     int64Ptr(111),
	})

	var capturedReq sd.Img2ImgRequest
	sdSvc := &mockSD{
		img2img: func(req sd.Img2ImgRequest) (*sd.Txt2ImgResponse, error) {
			capturedReq = req
			return &sd.Txt2ImgResponse{Images: []string{"img"}}, nil
		},
	}
	svc := newTestService(t, db, &mockLLM{}, sdSvc)
	svc.ctx = context.Background()

	_, err := svc.GenerateFromImage(GenerateFromImageParams{
		ImageBase64:       makePNGBase64(t, 64, 64),
		GenMode:           "preset",
		Mode:              "img2img",
		PresetID:          1,
		DenoisingStrength: 0.6,
		Seed:              int64Ptr(9999),
	})
	require.NoError(t, err)
	require.NotNil(t, capturedReq.Seed)
	assert.Equal(t, int64(9999), *capturedReq.Seed)

	_, err = svc.GenerateFromImage(GenerateFromImageParams{
		ImageBase64:       makePNGBase64(t, 64, 64),
		GenMode:           "preset",
		Mode:              "img2img",
		PresetID:          1,
		DenoisingStrength: 0.6,
	})
	require.NoError(t, err)
	require.NotNil(t, capturedReq.Seed)
	assert.Equal(t, int64(111), *capturedReq.Seed)
}

func TestFitImageToResolution(t *testing.T) {
	t.Parallel()
	db := openTestDB(t)
	svc := newTestService(t, db, &mockLLM{}, &mockSD{})
	svc.ctx = context.Background()

	orig := &preset.Preset{Name: "r", Prompt: "x"}
	r := &preset.Resolution{Width: 1024, Height: 1024}
	if err := db.CreateResolution(r); err != nil {
		t.Fatal(err)
	}

	cases := []struct {
		name         string
		w, h         int
		resID        *int64
		wantW, wantH int
	}{
		{"no resolution keeps original", 768, 1152, nil, 768, 1152},
		{"landscape fits into square", 2048, 1024, &r.ID, 1024, 512},
		{"portrait fits into square", 1024, 2048, &r.ID, 512, 1024},
		{"smaller upscales to fit", 512, 512, &r.ID, 1024, 1024},
		{"odd dims rounded to 8", 1000, 700, &r.ID, 1024, 712},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			w, h := svc.fitImageToResolution(tc.w, tc.h, tc.resID)
			if w != tc.wantW || h != tc.wantH {
				t.Errorf("got %dx%d, want %dx%d", w, h, tc.wantW, tc.wantH)
			}
			_ = orig
		})
	}
}

func TestFitImageScale(t *testing.T) {
	t.Parallel()

	db := openTestDB(t)
	r := &preset.Resolution{Width: 1024, Height: 1024}
	require.NoError(t, db.CreateResolution(r))
	svc := newTestService(t, db, &mockLLM{}, &mockSD{})

	tests := []struct {
		name      string
		w         int
		h         int
		resID     *int64
		wantScale float64
		wantOK    bool
	}{
		{name: "nil resolution id", w: 768, h: 1152, resID: nil, wantScale: 1, wantOK: false},
		{name: "zero resolution id", w: 768, h: 1152, resID: int64Ptr(0), wantScale: 1, wantOK: false},
		{name: "missing resolution", w: 768, h: 1152, resID: int64Ptr(999), wantScale: 1, wantOK: false},
		{name: "invalid image dims", w: 0, h: 512, resID: &r.ID, wantScale: 1, wantOK: false},
		{name: "landscape limited by width", w: 2048, h: 1024, resID: &r.ID, wantScale: 0.5, wantOK: true},
		{name: "portrait limited by height", w: 1024, h: 2048, resID: &r.ID, wantScale: 0.5, wantOK: true},
		{name: "small image scale above one", w: 512, h: 512, resID: &r.ID, wantScale: 2, wantOK: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			scale, ok := svc.fitImageScale(tt.w, tt.h, tt.resID)
			assert.InDelta(t, tt.wantScale, scale, 0.0001)
			assert.Equal(t, tt.wantOK, ok)
		})
	}
}

func TestFitImageToResolutionDownOnly(t *testing.T) {
	t.Parallel()

	db := openTestDB(t)
	r := &preset.Resolution{Width: 1024, Height: 1024}
	require.NoError(t, db.CreateResolution(r))
	svc := newTestService(t, db, &mockLLM{}, &mockSD{})

	tests := []struct {
		name  string
		w     int
		h     int
		resID *int64
		wantW int
		wantH int
	}{
		{name: "no resolution keeps original", w: 768, h: 1152, resID: nil, wantW: 768, wantH: 1152},
		{name: "missing resolution keeps original", w: 768, h: 1152, resID: int64Ptr(999), wantW: 768, wantH: 1152},
		{name: "landscape shrinks into box", w: 2048, h: 1024, resID: &r.ID, wantW: 1024, wantH: 512},
		{name: "portrait shrinks into box", w: 1024, h: 2048, resID: &r.ID, wantW: 512, wantH: 1024},
		{name: "small image stays native", w: 512, h: 512, resID: &r.ID, wantW: 512, wantH: 512},
		{name: "odd dims rounded to 8", w: 1500, h: 1000, resID: &r.ID, wantW: 1024, wantH: 680},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			w, h := svc.fitImageToResolutionDownOnly(tt.w, tt.h, tt.resID)
			assert.Equal(t, tt.wantW, w)
			assert.Equal(t, tt.wantH, h)
		})
	}
}

func TestQualityPostProcess(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name                 string
		srcB64               string
		srcW                 int
		srcH                 int
		interrupted          bool
		upscaleFn            func(base64Img string, upscaler string, scale float64) (string, error)
		img2imgFn            func(req sd.Img2ImgRequest) (*sd.Txt2ImgResponse, error)
		wantImg              string
		wantInfo             string
		wantUpscaleSkipped   bool
		wantRedenoiseSkipped bool
		wantErr              string
		wantUpscaleCalls     int
		wantImg2ImgCalls     int
	}{
		{
			name:   "success returns redenoised image and info",
			srcB64: "step1-img",
			srcW:   512,
			srcH:   512,
			upscaleFn: func(base64Img string, upscaler string, scale float64) (string, error) {
				return "upscaled-b64", nil
			},
			img2imgFn: func(req sd.Img2ImgRequest) (*sd.Txt2ImgResponse, error) {
				return &sd.Txt2ImgResponse{
					Images: []string{"final-img"},
					Info:   json.RawMessage(`{"seed": 77}`),
				}, nil
			},
			wantImg:          "final-img",
			wantInfo:         `{"seed": 77}`,
			wantUpscaleCalls: 1,
			wantImg2ImgCalls: 1,
		},
		{
			name:   "upscale failure returns step1 image",
			srcB64: "not-valid-base64!!!",
			srcW:   512,
			srcH:   512,
			upscaleFn: func(base64Img string, upscaler string, scale float64) (string, error) {
				return "", fmt.Errorf("esrgan unavailable")
			},
			img2imgFn: func(req sd.Img2ImgRequest) (*sd.Txt2ImgResponse, error) {
				return nil, fmt.Errorf("img2img must not run")
			},
			wantImg:            "not-valid-base64!!!",
			wantUpscaleSkipped: true,
			wantUpscaleCalls:   1,
			wantImg2ImgCalls:   0,
		},
		{
			name:   "redenoise error keeps upscaled image",
			srcB64: "step1-img",
			srcW:   512,
			srcH:   512,
			upscaleFn: func(base64Img string, upscaler string, scale float64) (string, error) {
				return "upscaled-b64", nil
			},
			img2imgFn: func(req sd.Img2ImgRequest) (*sd.Txt2ImgResponse, error) {
				return nil, fmt.Errorf("sd img2img down")
			},
			wantImg:              "upscaled-b64",
			wantRedenoiseSkipped: true,
			wantUpscaleCalls:     1,
			wantImg2ImgCalls:     1,
		},
		{
			name:   "redenoise empty images keeps upscaled image",
			srcB64: "step1-img",
			srcW:   512,
			srcH:   512,
			upscaleFn: func(base64Img string, upscaler string, scale float64) (string, error) {
				return "upscaled-b64", nil
			},
			img2imgFn: func(req sd.Img2ImgRequest) (*sd.Txt2ImgResponse, error) {
				return &sd.Txt2ImgResponse{}, nil
			},
			wantImg:              "upscaled-b64",
			wantRedenoiseSkipped: true,
			wantUpscaleCalls:     1,
			wantImg2ImgCalls:     1,
		},
		{
			name:        "interrupted redenoise propagates error",
			srcB64:      "step1-img",
			srcW:        512,
			srcH:        512,
			interrupted: true,
			upscaleFn: func(base64Img string, upscaler string, scale float64) (string, error) {
				return "upscaled-b64", nil
			},
			img2imgFn: func(req sd.Img2ImgRequest) (*sd.Txt2ImgResponse, error) {
				return nil, fmt.Errorf("job cancelled")
			},
			wantImg:          "",
			wantErr:          "interrupted",
			wantUpscaleCalls: 1,
			wantImg2ImgCalls: 1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			db := openTestDB(t)
			r := &preset.Resolution{Width: 1024, Height: 1024}
			require.NoError(t, db.CreateResolution(r))

			upscaleCalls := 0
			img2imgCalls := 0
			sdSvc := &mockSD{
				upscale: func(base64Img string, upscaler string, scale float64) (string, error) {
					upscaleCalls++
					return tt.upscaleFn(base64Img, upscaler, scale)
				},
				img2img: func(req sd.Img2ImgRequest) (*sd.Txt2ImgResponse, error) {
					img2imgCalls++
					return tt.img2imgFn(req)
				},
			}

			svc := newTestService(t, db, &mockLLM{}, sdSvc)
			svc.ctx = context.Background()
			if tt.interrupted {
				require.NoError(t, svc.InterruptGeneration())
			}

			img, info, upscaleSkipped, redenoiseSkipped, err := svc.qualityPostProcess(
				tt.srcB64,
				sd.Txt2ImgRequest{Prompt: "p", NegativePrompt: "n"},
				tt.srcW,
				tt.srcH,
				&r.ID,
			)

			assert.Equal(t, tt.wantUpscaleCalls, upscaleCalls)
			assert.Equal(t, tt.wantImg2ImgCalls, img2imgCalls)
			assert.Equal(t, tt.wantUpscaleSkipped, upscaleSkipped)
			assert.Equal(t, tt.wantRedenoiseSkipped, redenoiseSkipped)

			if tt.wantErr != "" {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.wantErr)
			} else {
				require.NoError(t, err)
			}
			assert.Equal(t, tt.wantImg, img)
			assert.Equal(t, tt.wantInfo, string(info))
		})
	}
}

func TestQualityPostProcess_Params(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		srcW      int
		srcH      int
		wantW     int
		wantH     int
		wantScale float64
	}{
		{name: "upscale small image", srcW: 512, srcH: 512, wantW: 1024, wantH: 1024, wantScale: 2},
		{name: "downscale large image", srcW: 2048, srcH: 1024, wantW: 1024, wantH: 512, wantScale: 0.5},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			db := openTestDB(t)
			r := &preset.Resolution{Width: 1024, Height: 1024}
			require.NoError(t, db.CreateResolution(r))

			var capturedUpscaleImg, capturedUpscaler string
			var capturedScale float64
			var capturedI2I sd.Img2ImgRequest

			sdSvc := &mockSD{
				upscale: func(base64Img string, upscaler string, scale float64) (string, error) {
					capturedUpscaleImg = base64Img
					capturedUpscaler = upscaler
					capturedScale = scale
					return "upscaled-b64", nil
				},
				img2img: func(req sd.Img2ImgRequest) (*sd.Txt2ImgResponse, error) {
					capturedI2I = req
					return &sd.Txt2ImgResponse{
						Images: []string{"final-img"},
						Info:   json.RawMessage(`{"seed": 1}`),
					}, nil
				},
			}

			svc := newTestService(t, db, &mockLLM{}, sdSvc)
			svc.ctx = context.Background()

			img, _, _, _, err := svc.qualityPostProcess("step1-img", sd.Txt2ImgRequest{Prompt: "p"}, tt.srcW, tt.srcH, &r.ID)
			require.NoError(t, err)
			assert.Equal(t, "final-img", img)

			assert.Equal(t, "step1-img", capturedUpscaleImg)
			assert.Equal(t, "R-ESRGAN 4x+", capturedUpscaler)
			assert.InDelta(t, tt.wantScale, capturedScale, 0.0001)

			assert.Equal(t, tt.wantW, capturedI2I.Width)
			assert.Equal(t, tt.wantH, capturedI2I.Height)
			require.Len(t, capturedI2I.InitImages, 1)
			assert.Equal(t, "upscaled-b64", capturedI2I.InitImages[0])
			require.NotNil(t, capturedI2I.DenoisingStrength)
			assert.InDelta(t, 0.35, *capturedI2I.DenoisingStrength, 0.001)
		})
	}
}
