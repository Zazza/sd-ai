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
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"go-sd/internal/config"
	"go-sd/internal/kids"
	"go-sd/internal/llm"
	"go-sd/internal/logger"
	"go-sd/internal/preset"
	"go-sd/internal/sd"
)

type mockLLM struct {
	mu             sync.Mutex
	chatFn         func(model, systemPrompt, userMessage string, temperature float64, maxTokens int) (string, error)
	chatVisionFn   func(model, systemPrompt, userText, imageBase64 string, temperature float64, maxTokens int) (string, error)
	chatMsgFn      func(model string, messages []llm.Message, temperature float64, maxTokens int) (string, error)
	genSDPromptFn  func(systemPrompt, userMessage, presetType, model string, maxTokens int) (string, error)
	analyzeImageFn func(model, systemPrompt, imageBase64 string, maxTokens int) (string, error)
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

func (m *mockLLM) GetModels() ([]llm.LLMModel, error)    { return nil, nil }
func (m *mockLLM) HealthCheck() error                      { return nil }
func (m *mockLLM) SetURL(baseURL string)                   {}
func (m *mockLLM) SetBackend(backend string)                {}
func (m *mockLLM) SetBackendConfig(cfg llm.BackendConfig)   {}

type mockSD struct {
	mu       sync.Mutex
	txt2img  func(req sd.Txt2ImgRequest) (*sd.Txt2ImgResponse, error)
	img2img  func(req sd.Img2ImgRequest) (*sd.Txt2ImgResponse, error)
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
			Width:          512,
			Height:         512,
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
		nil,
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
	assert.Nil(t, result)
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

func TestRecommendPreset_EmptyDescription(t *testing.T) {
	t.Parallel()
	db := openTestDB(t)
	svc := newTestService(t, db, &mockLLM{}, &mockSD{})

	_, err := svc.RecommendPreset("")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "description is required")
}

func TestRecommendPreset_NoPresets(t *testing.T) {
	t.Parallel()
	db := openTestDB(t)
	svc := newTestService(t, db, &mockLLM{}, &mockSD{})

	_, err := svc.RecommendPreset("a cat")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "no presets available")
}

func TestRecommendPreset_Success(t *testing.T) {
	t.Parallel()
	db := openTestDB(t)
	makeTestPreset(t, db, &preset.Preset{
		Name:           "Anime Girl",
		PresetType:     "anime",
		Prompt:         "1girl",
		NegativePrompt: "lowres",
		Sampler:        "Euler a",
		Steps:          20,
		CfgScale:       7.0,
		Width:          512,
		Height:         512,
	})

	llmSvc := &mockLLM{
		genSDPromptFn: func(systemPrompt, userMessage, presetType, model string, maxTokens int) (string, error) {
			return `{"preset_id": 1, "preset_name": "Anime Girl", "extra_prompt": "cat ears", "reasoning": "best match"}`, nil
		},
	}

	svc := newTestService(t, db, llmSvc, &mockSD{})

	result, err := svc.RecommendPreset("anime girl with cat ears")
	require.NoError(t, err)
	require.NotNil(t, result)
	assert.Equal(t, int64(1), result.PresetID)
	assert.Equal(t, "Anime Girl", result.PresetName)
	assert.Equal(t, "cat ears", result.ExtraPrompt)
}

func TestRecommendPreset_LLMError(t *testing.T) {
	t.Parallel()
	db := openTestDB(t)
	makeTestPreset(t, db, nil)

	llmSvc := &mockLLM{
		genSDPromptFn: func(systemPrompt, userMessage, presetType, model string, maxTokens int) (string, error) {
			return "", fmt.Errorf("timeout")
		},
	}

	svc := newTestService(t, db, llmSvc, &mockSD{})
	_, err := svc.RecommendPreset("a cat")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "timeout")
}

func TestRecommendPreset_InvalidJSONResponse(t *testing.T) {
	t.Parallel()
	db := openTestDB(t)
	makeTestPreset(t, db, nil)

	llmSvc := &mockLLM{
		genSDPromptFn: func(systemPrompt, userMessage, presetType, model string, maxTokens int) (string, error) {
			return "not json", nil
		},
	}

	svc := newTestService(t, db, llmSvc, &mockSD{})
	_, err := svc.RecommendPreset("a cat")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to parse LLM response")
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
			assert.Equal(t, "custom prompt", req.Prompt)
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
		Width:          512,
		Height:         512,
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
		Width:          512,
		Height:         512,
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

func TestDecomposeScene_EmptyDescription(t *testing.T) {
	t.Parallel()
	db := openTestDB(t)
	svc := newTestService(t, db, &mockLLM{}, &mockSD{})

	_, err := svc.DecomposeScene(DecomposeSceneParams{Description: "", PresetID: 1})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "description is required")
}

func TestDecomposeScene_NoPreset(t *testing.T) {
	t.Parallel()
	db := openTestDB(t)
	svc := newTestService(t, db, &mockLLM{}, &mockSD{})

	_, err := svc.DecomposeScene(DecomposeSceneParams{Description: "two warriors fighting", PresetID: 0})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "preset is required")
}

func TestDecomposeScene_PresetNotFound(t *testing.T) {
	t.Parallel()
	db := openTestDB(t)
	svc := newTestService(t, db, &mockLLM{}, &mockSD{})

	_, err := svc.DecomposeScene(DecomposeSceneParams{Description: "scene", PresetID: 999})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "preset not found")
}

func TestDecomposeScene_Success(t *testing.T) {
	t.Parallel()
	db := openTestDB(t)
	makeTestPreset(t, db, &preset.Preset{
		Name:           "scene-preset",
		Prompt:         "masterpiece, best quality",
		NegativePrompt: "lowres",
		Sampler:        "Euler a",
		Steps:          20,
		CfgScale:       7.0,
		Width:          768,
		Height:         512,
	})

	llmSvc := &mockLLM{
		chatFn: func(model, systemPrompt, userMessage string, temperature float64, maxTokens int) (string, error) {
			return `{
				"background_prompt": "battlefield, smoke, fire",
				"negative_prompt": "blurry, low quality",
				"characters": [
					{"name": "warrior1", "prompt": "warrior, heavy armor, sword", "position": {"x": 0.3, "y": 0.5}, "scale": 0.4},
					{"name": "warrior2", "prompt": "warrior, dark armor, axe", "position": {"x": 0.7, "y": 0.5}, "scale": 0.4}
				],
				"width": 768,
				"height": 512
			}`, nil
		},
	}

	svc := newTestService(t, db, llmSvc, &mockSD{})

	scene, err := svc.DecomposeScene(DecomposeSceneParams{
		Description: "two warriors fighting on a battlefield",
		PresetID:    1,
	})
	require.NoError(t, err)
	require.NotNil(t, scene)
	assert.Len(t, scene.Characters, 2)
	assert.Equal(t, "battlefield, smoke, fire", scene.BackgroundPrompt)
	assert.Equal(t, int64(1), scene.PresetID)
	assert.Equal(t, 768, scene.Width)
	assert.Equal(t, 512, scene.Height)
}

func TestDecomposeScene_LLMError(t *testing.T) {
	t.Parallel()
	db := openTestDB(t)
	makeTestPreset(t, db, nil)

	llmSvc := &mockLLM{
		chatFn: func(model, systemPrompt, userMessage string, temperature float64, maxTokens int) (string, error) {
			return "", fmt.Errorf("LLM unavailable")
		},
	}

	svc := newTestService(t, db, llmSvc, &mockSD{})

	_, err := svc.DecomposeScene(DecomposeSceneParams{
		Description: "scene",
		PresetID:    1,
	})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "LLM decomposition failed")
}

func TestDecomposeScene_InvalidJSONResponse(t *testing.T) {
	t.Parallel()
	db := openTestDB(t)
	makeTestPreset(t, db, nil)

	llmSvc := &mockLLM{
		chatFn: func(model, systemPrompt, userMessage string, temperature float64, maxTokens int) (string, error) {
			return "not valid json for a scene", nil
		},
	}

	svc := newTestService(t, db, llmSvc, &mockSD{})

	_, err := svc.DecomposeScene(DecomposeSceneParams{
		Description: "scene",
		PresetID:    1,
	})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to parse scene")
}

func TestAnalyzeImage_EmptyImage(t *testing.T) {
	t.Parallel()
	db := openTestDB(t)
	svc := newTestService(t, db, &mockLLM{}, &mockSD{})

	_, err := svc.AnalyzeImage("")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "image is required")
}

func TestAnalyzeImage_TooLarge(t *testing.T) {
	t.Parallel()
	db := openTestDB(t)
	svc := newTestService(t, db, &mockLLM{}, &mockSD{})

	bigData := make([]byte, 23*1024*1024)
	encoded := base64.StdEncoding.EncodeToString(bigData)

	_, err := svc.AnalyzeImage(encoded)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "image too large")
}

func TestAnalyzeImage_SingleMode(t *testing.T) {
	t.Parallel()
	db := openTestDB(t)
	require.NoError(t, db.SetSetting("analyze_use_chain", "false"))

	llmSvc := &mockLLM{
		analyzeImageFn: func(model, systemPrompt, imageBase64 string, maxTokens int) (string, error) {
			return "1girl, blue eyes, long hair", nil
		},
	}

	svc := newTestService(t, db, llmSvc, &mockSD{})
	img := makePNGBase64(t, 64, 64)

	result, err := svc.AnalyzeImage(img)
	require.NoError(t, err)
	assert.Contains(t, result, "1girl")
}

func TestAnalyzeImage_ChainMode(t *testing.T) {
	t.Parallel()
	db := openTestDB(t)

	callCount := 0
	llmSvc := &mockLLM{
		chatMsgFn: func(model string, messages []llm.Message, temperature float64, maxTokens int) (string, error) {
			callCount++
			return fmt.Sprintf("response for step %d", callCount), nil
		},
	}

	svc := newTestService(t, db, llmSvc, &mockSD{})
	img := makePNGBase64(t, 64, 64)

	result, err := svc.AnalyzeImage(img)
	require.NoError(t, err)
	assert.NotEmpty(t, result)
	assert.Equal(t, 4, callCount)
}

func TestAnalyzeImage_ChainMode_LLMError(t *testing.T) {
	t.Parallel()
	db := openTestDB(t)

	llmSvc := &mockLLM{
		chatMsgFn: func(model string, messages []llm.Message, temperature float64, maxTokens int) (string, error) {
			return "", fmt.Errorf("vision model error")
		},
	}

	svc := newTestService(t, db, llmSvc, &mockSD{})
	img := makePNGBase64(t, 64, 64)

	_, err := svc.AnalyzeImage(img)
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
	assert.Len(t, prompts.ChainPrompts, 4)
}

func TestBatchGenerate_InvalidCount(t *testing.T) {
	t.Parallel()
	db := openTestDB(t)
	svc := newTestService(t, db, &mockLLM{}, &mockSD{})
	svc.ctx = context.Background()

	tests := []struct {
		name  string
		count int
	}{
		{"zero", 0},
		{"negative", -1},
		{"too large", 101},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := svc.BatchGenerate(BatchGenerateParams{
				Prompt:       "test",
				Count:        tt.count,
				OutputFolder: t.TempDir(),
			})
			assert.Error(t, err)
			assert.Contains(t, err.Error(), "count must be between 1 and 100")
		})
	}
}

func TestBatchGenerate_NoOutputFolder(t *testing.T) {
	t.Parallel()
	db := openTestDB(t)
	svc := newTestService(t, db, &mockLLM{}, &mockSD{})
	svc.ctx = context.Background()

	err := svc.BatchGenerate(BatchGenerateParams{
		Prompt: "test",
		Count:  1,
	})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "output folder is required")
}

func TestBatchGenerate_NoPrompt(t *testing.T) {
	t.Parallel()
	db := openTestDB(t)
	svc := newTestService(t, db, &mockLLM{}, &mockSD{})
	svc.ctx = context.Background()

	err := svc.BatchGenerate(BatchGenerateParams{
		Count:        1,
		OutputFolder: t.TempDir(),
	})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "prompt is required")
}

func TestBatchGenerate_Success(t *testing.T) {
	db := openTestDB(t)
	makeTestPreset(t, db, nil)

	sdSvc := &mockSD{
		txt2img: func(req sd.Txt2ImgRequest) (*sd.Txt2ImgResponse, error) {
			return &sd.Txt2ImgResponse{
				Images: []string{makePNGBase64(t, 8, 8)},
			}, nil
		},
	}

	svc := newTestService(t, db, &mockLLM{}, sdSvc)
	svc.ctx = context.Background()

	outputDir := t.TempDir()
	err := svc.BatchGenerate(BatchGenerateParams{
		PresetID:     1,
		Prompt:       "1girl",
		Count:        2,
		OutputFolder: outputDir,
	})
	require.NoError(t, err)

	files, err := os.ReadDir(outputDir)
	require.NoError(t, err)
	assert.Len(t, files, 2)
}

func TestBatchGenerate_SDError(t *testing.T) {
	db := openTestDB(t)
	makeTestPreset(t, db, nil)

	sdSvc := &mockSD{
		txt2img: func(req sd.Txt2ImgRequest) (*sd.Txt2ImgResponse, error) {
			return nil, fmt.Errorf("SD error")
		},
	}

	svc := newTestService(t, db, &mockLLM{}, sdSvc)
	svc.ctx = context.Background()

	err := svc.BatchGenerate(BatchGenerateParams{
		PresetID:     1,
		Prompt:       "1girl",
		Count:        1,
		OutputFolder: t.TempDir(),
	})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "image 1/1 failed")
}

func TestBatchGenerate_AlreadyRunning(t *testing.T) {
	db := openTestDB(t)
	makeTestPreset(t, db, nil)

	blockCh := make(chan struct{})
	doneCh := make(chan struct{})
	sdSvc := &mockSD{
		txt2img: func(req sd.Txt2ImgRequest) (*sd.Txt2ImgResponse, error) {
			<-blockCh
			return &sd.Txt2ImgResponse{Images: []string{makePNGBase64(t, 8, 8)}}, nil
		},
	}

	outputDir := t.TempDir()

	svc := newTestService(t, db, &mockLLM{}, sdSvc)
	svc.ctx = context.Background()

	go func() {
		_ = svc.BatchGenerate(BatchGenerateParams{
			PresetID:     1,
			Prompt:       "1girl",
			Count:        1,
			OutputFolder: outputDir,
		})
		close(doneCh)
	}()

	var locked bool
	for i := 0; i < 1000; i++ {
		svc.batchMu.Lock()
		locked = svc.batchRunning
		svc.batchMu.Unlock()
		if locked {
			break
		}
	}

	if locked {
		err := svc.BatchGenerate(BatchGenerateParams{
			Prompt:       "1girl",
			Count:        1,
			OutputFolder: outputDir,
		})
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "already running")
	}

	close(blockCh)
	<-doneCh
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

	_, err := svc.TestGenerate(TestGenerateParams{Mode: "presets", SelectedIDs: []int64{1}})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "prompt is required")
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

func TestGetPresetForBatch_NoPreset(t *testing.T) {
	t.Parallel()
	db := openTestDB(t)
	svc := newTestService(t, db, &mockLLM{}, &mockSD{})

	_, err := svc.GetPresetForBatch(0, "desc")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "preset is required")
}

func TestGetPresetForBatch_NoDescription(t *testing.T) {
	t.Parallel()
	db := openTestDB(t)
	makeTestPreset(t, db, nil)
	svc := newTestService(t, db, &mockLLM{}, &mockSD{})

	result, err := svc.GetPresetForBatch(1, "")
	assert.NoError(t, err)
	assert.Nil(t, result)
}

func TestGetPresetForBatch_WithDescription(t *testing.T) {
	t.Parallel()
	db := openTestDB(t)
	makeTestPreset(t, db, nil)

	llmSvc := &mockLLM{
		genSDPromptFn: func(systemPrompt, userMessage, presetType, model string, maxTokens int) (string, error) {
			return `{"prompt": "merged", "negative_prompt": "neg"}`, nil
		},
	}

	svc := newTestService(t, db, llmSvc, &mockSD{})

	result, err := svc.GetPresetForBatch(1, "cat girl")
	require.NoError(t, err)
	require.NotNil(t, result)
	assert.Equal(t, "merged", result.Prompt)
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
			result := buildSamplerName(tt.sampler, tt.scheduleType)
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
		assert.Equal(t, 256, svc.getMaxTokens())
	})

	t.Run("from setting", func(t *testing.T) {
		t.Parallel()
		db := openTestDB(t)
		require.NoError(t, db.SetSetting("llm_max_tokens", "512"))
		svc := newTestService(t, db, &mockLLM{}, &mockSD{})
		assert.Equal(t, 512, svc.getMaxTokens())
	})

	t.Run("invalid setting uses default", func(t *testing.T) {
		t.Parallel()
		db := openTestDB(t)
		require.NoError(t, db.SetSetting("llm_max_tokens", "not-a-number"))
		svc := newTestService(t, db, &mockLLM{}, &mockSD{})
		assert.Equal(t, 256, svc.getMaxTokens())
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

func TestSaveLastImage(t *testing.T) {
	t.Parallel()
	tmpDir := t.TempDir()
	db := openTestDB(t)
	svc := newTestService(t, db, &mockLLM{}, &mockSD{})
	svc.dataDir = tmpDir

	t.Run("saves png and meta", func(t *testing.T) {
		pngData := makePNGBase64(t, 8, 8)
		svc.saveLastImage(pngData, json.RawMessage(`{"seed": 42}`), false)

		_, err := os.Stat(filepath.Join(tmpDir, "last_image.png"))
		assert.NoError(t, err)
		_, err = os.Stat(filepath.Join(tmpDir, "last_image.json"))
		assert.NoError(t, err)
	})

	t.Run("empty image does nothing", func(t *testing.T) {
		svc.saveLastImage("", nil, false)
	})

	t.Run("invalid base64 does nothing", func(t *testing.T) {
		svc.saveLastImage("!!!invalid!!!", nil, false)
	})
}

func TestGetAnalyzeChainPrompts(t *testing.T) {
	t.Parallel()

	t.Run("default prompts", func(t *testing.T) {
		t.Parallel()
		db := openTestDB(t)
		svc := newTestService(t, db, &mockLLM{}, &mockSD{})
		prompts := svc.getAnalyzeChainPrompts()
		assert.Len(t, prompts, 4)
		assert.NotEmpty(t, prompts[0])
	})

	t.Run("custom prompts from settings", func(t *testing.T) {
		t.Parallel()
		db := openTestDB(t)
		require.NoError(t, db.SetSetting("analyze_chain_1", "custom prompt 1"))
		require.NoError(t, db.SetSetting("analyze_chain_3", "custom prompt 3"))
		svc := newTestService(t, db, &mockLLM{}, &mockSD{})
		prompts := svc.getAnalyzeChainPrompts()
		assert.Len(t, prompts, 4)
		assert.Equal(t, "custom prompt 1", prompts[0])
		assert.Equal(t, "custom prompt 3", prompts[2])
	})
}

func TestBatchCompoundGenerate_InvalidCount(t *testing.T) {
	t.Parallel()
	db := openTestDB(t)
	svc := newTestService(t, db, &mockLLM{}, &mockSD{})
	svc.ctx = context.Background()

	tests := []struct {
		name  string
		count int
	}{
		{"zero", 0},
		{"too large", 101},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := svc.BatchCompoundGenerate(BatchCompoundGenerateParams{
				Count:        tt.count,
				OutputFolder: t.TempDir(),
			})
			assert.Error(t, err)
		})
	}
}

func TestBatchCompoundGenerate_NoOutputFolder(t *testing.T) {
	t.Parallel()
	db := openTestDB(t)
	svc := newTestService(t, db, &mockLLM{}, &mockSD{})
	svc.ctx = context.Background()

	err := svc.BatchCompoundGenerate(BatchCompoundGenerateParams{Count: 1})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "output folder is required")
}

func TestBatchCompoundGenerate_CompoundNotFound(t *testing.T) {
	t.Parallel()
	db := openTestDB(t)
	svc := newTestService(t, db, &mockLLM{}, &mockSD{})
	svc.ctx = context.Background()

	err := svc.BatchCompoundGenerate(BatchCompoundGenerateParams{
		CompoundPresetID: 999,
		Count:            1,
		OutputFolder:     t.TempDir(),
	})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "compound preset not found")
}

func TestBatchCompoundGenerate_Success(t *testing.T) {
	db := openTestDB(t)
	p := makeTestPreset(t, db, nil)

	cp := &preset.CompoundPreset{
		Name:        "test-compound",
		Description: "test",
		Steps: []preset.CompoundPresetStep{
			{PresetID: p.ID, Width: 512, Height: 512, DenoisingStrength: 0.5},
		},
	}
	require.NoError(t, db.CreateCompoundPreset(cp))

	sdSvc := &mockSD{
		txt2img: func(req sd.Txt2ImgRequest) (*sd.Txt2ImgResponse, error) {
			return &sd.Txt2ImgResponse{
				Images: []string{makePNGBase64(t, 8, 8)},
			}, nil
		},
	}

	svc := newTestService(t, db, &mockLLM{}, sdSvc)
	svc.ctx = context.Background()

	outputDir := t.TempDir()
	err := svc.BatchCompoundGenerate(BatchCompoundGenerateParams{
		CompoundPresetID: cp.ID,
		ExtraPrompt:      "test",
		Count:            1,
		OutputFolder:     outputDir,
	})
	require.NoError(t, err)

	files, err := os.ReadDir(outputDir)
	require.NoError(t, err)
	assert.Len(t, files, 1)
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
			{PresetID: p.ID, Width: 512, Height: 512},
			{PresetID: p.ID, Width: 768, Height: 768, DenoisingStrength: 0.6},
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

	svc := newTestService(t, db, &mockLLM{}, sdSvc)
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

func TestTestCompoundGenerate_Validation(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name   string
		params TestCompoundGenerateParams
		errMsg string
	}{
		{"no ids", TestCompoundGenerateParams{Prompt: "test"}, "select at least one"},
		{"too many ids", TestCompoundGenerateParams{SelectedIDs: make([]int64, 21), Prompt: "test"}, "maximum 20"},
		{"no prompt", TestCompoundGenerateParams{SelectedIDs: []int64{1}}, "prompt is required"},
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
			{PresetID: p.ID, Width: 512, Height: 512},
		},
	}
	require.NoError(t, db.CreateCompoundPreset(cp))

	sdSvc := &mockSD{
		txt2img: func(req sd.Txt2ImgRequest) (*sd.Txt2ImgResponse, error) {
			return &sd.Txt2ImgResponse{Images: []string{"result-img"}}, nil
		},
	}

	svc := newTestService(t, db, &mockLLM{}, sdSvc)
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
