package settings

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"go-sd/internal/config"
	"go-sd/internal/llm"
	"go-sd/internal/logger"
	"go-sd/internal/preset"
	"go-sd/internal/rembg"
	"go-sd/internal/sd"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type mockLLMService struct {
	healthErr      error
	setURLCalled   bool
	setURLVal      string
	setBackendVal  string
	setBackendCfg  llm.BackendConfig
	setBackendCfgV bool
}

func (m *mockLLMService) Chat(string, string, string, float64, int) (string, error) {
	return "", nil
}
func (m *mockLLMService) ChatVision(string, string, string, string, float64, int) (string, error) {
	return "", nil
}
func (m *mockLLMService) ChatWithMessages(string, []llm.Message, float64, int) (string, error) {
	return "", nil
}
func (m *mockLLMService) GenerateSDPrompt(string, string, string, string, int) (string, error) {
	return "", nil
}
func (m *mockLLMService) AnalyzeImage(string, string, string, int) (string, error) {
	return "", nil
}
func (m *mockLLMService) GetModels() ([]llm.LLMModel, error) {
	return nil, nil
}
func (m *mockLLMService) HealthCheck() error {
	return m.healthErr
}
func (m *mockLLMService) SetURL(baseURL string) {
	m.setURLCalled = true
	m.setURLVal = baseURL
}
func (m *mockLLMService) SetBackend(backend string) {
	m.setBackendVal = backend
}
func (m *mockLLMService) SetBackendConfig(cfg llm.BackendConfig) {
	m.setBackendCfg = cfg
	m.setBackendCfgV = true
}

type mockSDService struct {
	healthErr    error
	options      map[string]interface{}
	optionsErr   error
	setURLCalled bool
	setURLVal    string
}

func (m *mockSDService) Txt2Img(sd.Txt2ImgRequest) (*sd.Txt2ImgResponse, error) {
	return nil, nil
}
func (m *mockSDService) Img2Img(sd.Img2ImgRequest) (*sd.Txt2ImgResponse, error) {
	return nil, nil
}
func (m *mockSDService) GetModels() ([]sd.SDModel, error) {
	return nil, nil
}
func (m *mockSDService) GetSamplers() ([]sd.Sampler, error) {
	return nil, nil
}
func (m *mockSDService) GetSchedulers() ([]sd.Scheduler, error) {
	return nil, nil
}
func (m *mockSDService) GetUpscalers() ([]sd.Upscaler, error) {
	return nil, nil
}
func (m *mockSDService) GetVAEs() ([]sd.VAE, error) {
	return nil, nil
}
func (m *mockSDService) GetLoRAs() ([]sd.LoRA, error) {
	return nil, nil
}
func (m *mockSDService) GetOptions() (map[string]interface{}, error) {
	if m.optionsErr != nil {
		return nil, m.optionsErr
	}
	return m.options, nil
}
func (m *mockSDService) GetProgress() (*sd.ProgressResponse, error) {
	return nil, nil
}
func (m *mockSDService) Interrupt() error {
	return nil
}
func (m *mockSDService) HealthCheck() error {
	return m.healthErr
}
func (m *mockSDService) SetURL(baseURL string) {
	m.setURLCalled = true
	m.setURLVal = baseURL
}
func (m *mockSDService) SetModel(string) error { return nil }
func (m *mockSDService) SetVAE(string) error   { return nil }
func (m *mockSDService) UpscaleImage(string, string, float64) (string, error) { return "", nil }

func testService(t *testing.T, llmSvc *mockLLMService, sdSvc *mockSDService) (*Service, *preset.DB) {
	t.Helper()
	db, err := preset.Open(":memory:")
	require.NoError(t, err)
	t.Cleanup(func() { db.Close() })

	if llmSvc == nil {
		llmSvc = &mockLLMService{}
	}
	if sdSvc == nil {
		sdSvc = &mockSDService{}
	}

	cfg := &config.Config{
		LLMUrl:        "http://localhost:1234",
		SDUrl:         "http://localhost:7860",
		LLMModel:      "test-model",
		SDPromptModel: "default",
		VisionModel:   "vision-model",
		LLMBackend:    "lmstudio",
	}

	rembgClient := rembg.New("")
	log := logger.New(nil)

	svc := New(db, llmSvc, sdSvc, cfg, rembgClient, log)
	return svc, db
}

func TestNew_ServiceCreation(t *testing.T) {
	t.Parallel()
	svc, _ := testService(t, nil, nil)
	assert.NotNil(t, svc)
}

func TestCheckServices_BothAvailable(t *testing.T) {
	t.Parallel()
	sdSvc := &mockSDService{
		options: map[string]interface{}{
			"sd_model_checkpoint": "test-model.safetensors",
		},
	}
	llmSvc := &mockLLMService{healthErr: nil}
	svc, db := testService(t, llmSvc, sdSvc)

	require.NoError(t, db.SetSetting("llm_generate_model", "custom-gen-model"))
	require.NoError(t, db.SetSetting("llm_analyze_model", "custom-vision-model"))

	status := svc.CheckServices()

	assert.True(t, status.LLM.Available)
	assert.Equal(t, "custom-gen-model", status.LLM.Model)
	assert.Equal(t, "custom-vision-model", status.LLM.VisionModel)
	assert.True(t, status.SD.Available)
	assert.Equal(t, "test-model.safetensors", status.SD.Model)
}

func TestCheckServices_LLMUnhealthy(t *testing.T) {
	t.Parallel()
	llmSvc := &mockLLMService{healthErr: fmt.Errorf("connection refused")}
	sdSvc := &mockSDService{}
	svc, _ := testService(t, llmSvc, sdSvc)

	status := svc.CheckServices()

	assert.False(t, status.LLM.Available)
	assert.Empty(t, status.LLM.Model)
	assert.True(t, status.SD.Available)
}

func TestCheckServices_SDUnhealthy(t *testing.T) {
	t.Parallel()
	llmSvc := &mockLLMService{}
	sdSvc := &mockSDService{healthErr: fmt.Errorf("sd down")}
	svc, _ := testService(t, llmSvc, sdSvc)

	status := svc.CheckServices()

	assert.True(t, status.LLM.Available)
	assert.False(t, status.SD.Available)
}

func TestCheckServices_BothUnhealthy(t *testing.T) {
	t.Parallel()
	llmSvc := &mockLLMService{healthErr: fmt.Errorf("llm error")}
	sdSvc := &mockSDService{healthErr: fmt.Errorf("sd error")}
	svc, _ := testService(t, llmSvc, sdSvc)

	status := svc.CheckServices()

	assert.False(t, status.LLM.Available)
	assert.False(t, status.SD.Available)
}

func TestCheckServices_SDOptionsError(t *testing.T) {
	t.Parallel()
	llmSvc := &mockLLMService{}
	sdSvc := &mockSDService{optionsErr: fmt.Errorf("options error")}
	svc, _ := testService(t, llmSvc, sdSvc)

	status := svc.CheckServices()

	assert.True(t, status.SD.Available)
	assert.Empty(t, status.SD.Model)
}

func TestCheckServices_SDModelNotString(t *testing.T) {
	t.Parallel()
	sdSvc := &mockSDService{
		options: map[string]interface{}{
			"sd_model_checkpoint": 42,
		},
	}
	llmSvc := &mockLLMService{}
	svc, _ := testService(t, llmSvc, sdSvc)

	status := svc.CheckServices()

	assert.True(t, status.SD.Available)
	assert.Empty(t, status.SD.Model)
}

func TestCheckServices_LLMFallbackModels(t *testing.T) {
	t.Parallel()
	llmSvc := &mockLLMService{}
	sdSvc := &mockSDService{}
	svc, _ := testService(t, llmSvc, sdSvc)

	status := svc.CheckServices()

	assert.True(t, status.LLM.Available)
	assert.Equal(t, "default", status.LLM.Model)
	assert.Equal(t, "vision-model", status.LLM.VisionModel)
}

func TestCheckRembg_NoURLConfigured(t *testing.T) {
	t.Parallel()
	svc, _ := testService(t, nil, nil)

	err := svc.CheckRembg()
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "rembg URL not configured")
}

func TestCheckRembg_WithValidServer(t *testing.T) {
	t.Parallel()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/api" {
			w.WriteHeader(http.StatusOK)
			return
		}
		w.WriteHeader(http.StatusNotFound)
	}))
	defer server.Close()

	svc, db := testService(t, nil, nil)
	require.NoError(t, db.SetSetting("rembg_url", server.URL))

	err := svc.CheckRembg()
	assert.NoError(t, err)
}

func TestCheckRembg_ServerUnreachable(t *testing.T) {
	t.Parallel()
	svc, db := testService(t, nil, nil)
	require.NoError(t, db.SetSetting("rembg_url", "http://localhost:1"))

	err := svc.CheckRembg()
	assert.Error(t, err)
}

func TestCheckRembg_ServerReturnsError(t *testing.T) {
	t.Parallel()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer server.Close()

	svc, db := testService(t, nil, nil)
	require.NoError(t, db.SetSetting("rembg_url", server.URL))

	err := svc.CheckRembg()
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "rembg status 500")
}

func TestGetSettings_ReturnsDefaults(t *testing.T) {
	t.Parallel()
	svc, _ := testService(t, nil, nil)

	settings, err := svc.GetSettings()
	require.NoError(t, err)
	assert.NotEmpty(t, settings)

	assert.Equal(t, "http://localhost:1234", settings["llm_url"])
	assert.Equal(t, "http://localhost:7860", settings["sd_url"])
	assert.Equal(t, "test-model", settings["llm_model"])
	assert.Equal(t, "default", settings["sd_prompt_model"])
	assert.Equal(t, "vision-model", settings["vision_model"])
	assert.Equal(t, "lmstudio", settings["llm_backend"])
	assert.Equal(t, "5m", settings["llm_keep_alive"])
	assert.Equal(t, "false", settings["kids_mode"])
	assert.Equal(t, "", settings["rembg_url"])
	assert.Equal(t, "512", settings["preview_width"])
	assert.Equal(t, "512", settings["preview_height"])
}

func TestGetSettings_OverridesWithStoredValues(t *testing.T) {
	t.Parallel()
	svc, db := testService(t, nil, nil)

	require.NoError(t, db.SetSetting("llm_url", "http://custom-llm:9999"))
	require.NoError(t, db.SetSetting("kids_mode", "true"))
	require.NoError(t, db.SetSetting("preview_width", "1024"))

	settings, err := svc.GetSettings()
	require.NoError(t, err)

	assert.Equal(t, "http://custom-llm:9999", settings["llm_url"])
	assert.Equal(t, "true", settings["kids_mode"])
	assert.Equal(t, "1024", settings["preview_width"])
}

func TestGetSettings_AllDefaultKeysPresent(t *testing.T) {
	t.Parallel()
	svc, _ := testService(t, nil, nil)

	settings, err := svc.GetSettings()
	require.NoError(t, err)

	expectedKeys := []string{
		"llm_url", "sd_url", "llm_model", "sd_prompt_model", "vision_model",
		"llm_backend", "llm_keep_alive", "llm_num_ctx", "llm_num_gpu",
		"llm_max_tokens", "llm_generate_model", "llm_analyze_model",
		"llm_generate_temperature", "llm_generate_num_ctx",
		"llm_generate_num_predict", "llm_generate_top_p",
		"llm_generate_num_thread", "llm_analyze_temperature",
		"llm_analyze_num_ctx", "llm_analyze_num_predict",
		"llm_analyze_top_p", "llm_analyze_num_thread",
		"kids_mode", "kids_cat_violence", "kids_cat_horror",
		"kids_cat_weapons", "kids_cat_substances", "kids_cat_mature",
		"rembg_url", "preview_mode", "preview_width", "preview_height",
	}
	for _, k := range expectedKeys {
		_, ok := settings[k]
		assert.True(t, ok, "missing default key: %s", k)
	}
}

func TestUpdateSettings_AllowedSettings(t *testing.T) {
	t.Parallel()
	llmSvc := &mockLLMService{}
	sdSvc := &mockSDService{}
	svc, _ := testService(t, llmSvc, sdSvc)

	data := map[string]string{
		"llm_url":        "http://new-llm:4000",
		"sd_url":         "http://new-sd:9000",
		"llm_model":      "new-model",
		"sd_prompt_model": "new-prompt-model",
		"vision_model":   "new-vision",
		"llm_backend":    "ollama",
		"kids_mode":      "true",
	}

	err := svc.UpdateSettings(data)
	require.NoError(t, err)

	assert.True(t, llmSvc.setURLCalled)
	assert.Equal(t, "http://new-llm:4000", llmSvc.setURLVal)
	assert.True(t, sdSvc.setURLCalled)
	assert.Equal(t, "http://new-sd:9000", sdSvc.setURLVal)
	assert.Equal(t, "ollama", llmSvc.setBackendVal)

	settings, err := svc.GetSettings()
	require.NoError(t, err)
	assert.Equal(t, "http://new-llm:4000", settings["llm_url"])
	assert.Equal(t, "http://new-sd:9000", settings["sd_url"])
	assert.Equal(t, "new-model", settings["llm_model"])
	assert.Equal(t, "true", settings["kids_mode"])
}

func TestUpdateSettings_InvalidLLMUrl(t *testing.T) {
	t.Parallel()
	svc, _ := testService(t, nil, nil)

	data := map[string]string{
		"llm_url": ":::not-a-url",
	}

	err := svc.UpdateSettings(data)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "invalid llm_url")
}

func TestUpdateSettings_InvalidSDUrl(t *testing.T) {
	t.Parallel()
	svc, _ := testService(t, nil, nil)

	data := map[string]string{
		"sd_url": "://bad",
	}

	err := svc.UpdateSettings(data)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "invalid sd_url")
}

func TestUpdateSettings_InvalidRembgUrl(t *testing.T) {
	t.Parallel()
	svc, _ := testService(t, nil, nil)

	data := map[string]string{
		"rembg_url": "::invalid",
	}

	err := svc.UpdateSettings(data)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "invalid rembg_url")
}

func TestUpdateSettings_EmptyUrlAllowed(t *testing.T) {
	t.Parallel()
	svc, _ := testService(t, nil, nil)

	data := map[string]string{
		"llm_url":   "",
		"sd_url":    "",
		"rembg_url": "",
	}

	err := svc.UpdateSettings(data)
	assert.NoError(t, err)
}

func TestUpdateSettings_NumericFieldsValid(t *testing.T) {
	t.Parallel()
	svc, _ := testService(t, nil, nil)

	data := map[string]string{
		"llm_num_ctx":   "8192",
		"llm_num_gpu":   "1",
		"llm_max_tokens": "512",
		"preview_width":  "1024",
		"preview_height": "768",
	}

	err := svc.UpdateSettings(data)
	require.NoError(t, err)

	settings, err := svc.GetSettings()
	require.NoError(t, err)
	assert.Equal(t, "8192", settings["llm_num_ctx"])
	assert.Equal(t, "1", settings["llm_num_gpu"])
	assert.Equal(t, "512", settings["llm_max_tokens"])
	assert.Equal(t, "1024", settings["preview_width"])
	assert.Equal(t, "768", settings["preview_height"])
}

func TestUpdateSettings_NumericFieldsZeroOnInvalid(t *testing.T) {
	t.Parallel()
	svc, _ := testService(t, nil, nil)

	data := map[string]string{
		"llm_num_ctx": "abc",
		"preview_width": "-100",
	}

	err := svc.UpdateSettings(data)
	require.NoError(t, err)

	settings, err := svc.GetSettings()
	require.NoError(t, err)
	assert.Equal(t, "0", settings["llm_num_ctx"])
	assert.Equal(t, "0", settings["preview_width"])
}

func TestUpdateSettings_NumericFieldsEmptyBecomesZero(t *testing.T) {
	t.Parallel()
	svc, _ := testService(t, nil, nil)

	data := map[string]string{
		"llm_num_ctx":    "",
		"preview_width":  "",
	}

	err := svc.UpdateSettings(data)
	require.NoError(t, err)

	settings, err := svc.GetSettings()
	require.NoError(t, err)
	assert.Equal(t, "0", settings["llm_num_ctx"])
	assert.Equal(t, "0", settings["preview_width"])
}

func TestUpdateSettings_DisallowedSettingIgnored(t *testing.T) {
	t.Parallel()
	svc, db := testService(t, nil, nil)

	data := map[string]string{
		"malicious_key": "evil_value",
	}

	err := svc.UpdateSettings(data)
	require.NoError(t, err)

	val, err := db.GetSetting("malicious_key")
	require.NoError(t, err)
	assert.Empty(t, val)
}

func TestUpdateSettings_RembgURL(t *testing.T) {
	t.Parallel()
	svc, db := testService(t, nil, nil)

	data := map[string]string{
		"rembg_url": "http://rembg:7000",
	}

	err := svc.UpdateSettings(data)
	require.NoError(t, err)

	val, err := db.GetSetting("rembg_url")
	require.NoError(t, err)
	assert.Equal(t, "http://rembg:7000", val)
}

func TestUpdateSettings_ConfigUpdated(t *testing.T) {
	t.Parallel()
	llmSvc := &mockLLMService{}
	sdSvc := &mockSDService{}
	svc, _ := testService(t, llmSvc, sdSvc)

	data := map[string]string{
		"llm_model":       "updated-model",
		"sd_prompt_model": "updated-prompt",
		"vision_model":    "updated-vision",
	}

	err := svc.UpdateSettings(data)
	require.NoError(t, err)

	assert.Equal(t, "updated-model", svc.cfg.LLMModel)
	assert.Equal(t, "updated-prompt", svc.cfg.SDPromptModel)
	assert.Equal(t, "updated-vision", svc.cfg.VisionModel)
}

func TestUpdateSettings_LLMGenerateModelOverridesSDPromptModel(t *testing.T) {
	t.Parallel()
	svc, _ := testService(t, nil, nil)

	data := map[string]string{
		"llm_generate_model": "gen-model-v2",
	}

	err := svc.UpdateSettings(data)
	require.NoError(t, err)

	assert.Equal(t, "gen-model-v2", svc.cfg.SDPromptModel)
}

func TestUpdateSettings_LLMAnalyzeModelOverridesVisionModel(t *testing.T) {
	t.Parallel()
	svc, _ := testService(t, nil, nil)

	data := map[string]string{
		"llm_analyze_model": "analyze-vision-v2",
	}

	err := svc.UpdateSettings(data)
	require.NoError(t, err)

	assert.Equal(t, "analyze-vision-v2", svc.cfg.VisionModel)
}

func TestApplyLLMConfig_GenerateMode(t *testing.T) {
	t.Parallel()
	llmSvc := &mockLLMService{}
	svc, db := testService(t, llmSvc, nil)

	require.NoError(t, db.SetSetting("llm_keep_alive", "10m"))
	require.NoError(t, db.SetSetting("llm_generate_num_ctx", "2048"))
	require.NoError(t, db.SetSetting("llm_generate_num_predict", "128"))
	require.NoError(t, db.SetSetting("llm_generate_top_p", "0.8"))
	require.NoError(t, db.SetSetting("llm_generate_num_thread", "4"))
	require.NoError(t, db.SetSetting("llm_num_gpu", "2"))

	svc.ApplyLLMConfig("generate")

	assert.True(t, llmSvc.setBackendCfgV)
	assert.Equal(t, "10m", llmSvc.setBackendCfg.KeepAlive)
	assert.Equal(t, 2048, llmSvc.setBackendCfg.NumCtx)
	assert.Equal(t, 128, llmSvc.setBackendCfg.NumPredict)
	assert.Equal(t, 0.8, llmSvc.setBackendCfg.TopP)
	assert.Equal(t, 4, llmSvc.setBackendCfg.NumThread)
	assert.Equal(t, 2, llmSvc.setBackendCfg.NumGPU)
}

func TestApplyLLMConfig_AnalyzeMode(t *testing.T) {
	t.Parallel()
	llmSvc := &mockLLMService{}
	svc, db := testService(t, llmSvc, nil)

	require.NoError(t, db.SetSetting("llm_keep_alive", "15m"))
	require.NoError(t, db.SetSetting("llm_analyze_num_ctx", "8192"))
	require.NoError(t, db.SetSetting("llm_analyze_num_predict", "512"))
	require.NoError(t, db.SetSetting("llm_analyze_top_p", "0.95"))
	require.NoError(t, db.SetSetting("llm_analyze_num_thread", "8"))
	require.NoError(t, db.SetSetting("llm_num_gpu", "1"))

	svc.ApplyLLMConfig("analyze")

	assert.True(t, llmSvc.setBackendCfgV)
	assert.Equal(t, "15m", llmSvc.setBackendCfg.KeepAlive)
	assert.Equal(t, 8192, llmSvc.setBackendCfg.NumCtx)
	assert.Equal(t, 512, llmSvc.setBackendCfg.NumPredict)
	assert.Equal(t, 0.95, llmSvc.setBackendCfg.TopP)
	assert.Equal(t, 8, llmSvc.setBackendCfg.NumThread)
	assert.Equal(t, 1, llmSvc.setBackendCfg.NumGPU)
}

func TestApplyLLMConfig_DefaultIsGenerate(t *testing.T) {
	t.Parallel()
	llmSvc := &mockLLMService{}
	svc, db := testService(t, llmSvc, nil)

	require.NoError(t, db.SetSetting("llm_generate_num_ctx", "4096"))
	require.NoError(t, db.SetSetting("llm_analyze_num_ctx", "8192"))

	svc.ApplyLLMConfig("")

	assert.True(t, llmSvc.setBackendCfgV)
	assert.Equal(t, 4096, llmSvc.setBackendCfg.NumCtx)
}

func TestApplyLLMConfig_EmptySettingsIgnored(t *testing.T) {
	t.Parallel()
	llmSvc := &mockLLMService{}
	svc, db := testService(t, llmSvc, nil)

	require.NoError(t, db.SetSetting("llm_keep_alive", "5m"))

	svc.ApplyLLMConfig("generate")

	assert.True(t, llmSvc.setBackendCfgV)
	assert.Equal(t, "5m", llmSvc.setBackendCfg.KeepAlive)
	assert.Equal(t, 0, llmSvc.setBackendCfg.NumCtx)
	assert.Equal(t, 0, llmSvc.setBackendCfg.NumPredict)
	assert.Equal(t, 0.0, llmSvc.setBackendCfg.TopP)
	assert.Equal(t, 0, llmSvc.setBackendCfg.NumThread)
}

func TestApplyLLMConfig_InvalidNumbersIgnored(t *testing.T) {
	t.Parallel()
	llmSvc := &mockLLMService{}
	svc, db := testService(t, llmSvc, nil)

	require.NoError(t, db.SetSetting("llm_generate_num_ctx", "not-a-number"))
	require.NoError(t, db.SetSetting("llm_generate_top_p", "bad-float"))
	require.NoError(t, db.SetSetting("llm_num_gpu", "xyz"))

	svc.ApplyLLMConfig("generate")

	assert.True(t, llmSvc.setBackendCfgV)
	assert.Equal(t, 0, llmSvc.setBackendCfg.NumCtx)
	assert.Equal(t, 0.0, llmSvc.setBackendCfg.TopP)
	assert.Equal(t, 0, llmSvc.setBackendCfg.NumGPU)
}
