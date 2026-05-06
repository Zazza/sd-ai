package api

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"go-sd/internal/config"
	"go-sd/internal/llm"
	"go-sd/internal/preset"
	"go-sd/internal/sd"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type mockLLM struct {
	prompt    string
	models    []llm.LLMModel
	err       error
	url       string
	backend   string
	backendCfg llm.BackendConfig
}

func (m *mockLLM) Chat(_, _, _ string, _ float64, _ int) (string, error) {
	return m.prompt, m.err
}

func (m *mockLLM) ChatVision(_, _, _, _ string, _ float64, _ int) (string, error) {
	return m.prompt, m.err
}

func (m *mockLLM) ChatWithMessages(_ string, _ []llm.Message, _ float64, _ int) (string, error) {
	return m.prompt, m.err
}

func (m *mockLLM) GenerateSDPrompt(_, _, _, _ string, _ int) (string, error) {
	return m.prompt, m.err
}

func (m *mockLLM) AnalyzeImage(_, _, _ string, _ int) (string, error) {
	return m.prompt, m.err
}

func (m *mockLLM) GetModels() ([]llm.LLMModel, error) {
	return m.models, m.err
}

func (m *mockLLM) HealthCheck() error {
	return m.err
}

func (m *mockLLM) SetURL(u string) {
	m.url = u
}

func (m *mockLLM) SetBackend(b string) {
	m.backend = b
}

func (m *mockLLM) SetBackendConfig(cfg llm.BackendConfig) {
	m.backendCfg = cfg
}

type mockSD struct {
	models     []sd.SDModel
	samplers   []sd.Sampler
	schedulers []sd.Scheduler
	result     *sd.Txt2ImgResponse
	err        error
	url        string
	modelName  string
}

func (m *mockSD) Txt2Img(_ sd.Txt2ImgRequest) (*sd.Txt2ImgResponse, error) {
	return m.result, m.err
}

func (m *mockSD) Img2Img(_ sd.Img2ImgRequest) (*sd.Txt2ImgResponse, error) {
	return m.result, m.err
}

func (m *mockSD) GetModels() ([]sd.SDModel, error) {
	return m.models, m.err
}

func (m *mockSD) GetSamplers() ([]sd.Sampler, error) {
	return m.samplers, m.err
}

func (m *mockSD) GetSchedulers() ([]sd.Scheduler, error) {
	return m.schedulers, m.err
}

func (m *mockSD) GetUpscalers() ([]sd.Upscaler, error) {
	return nil, nil
}

func (m *mockSD) GetVAEs() ([]sd.VAE, error) {
	return nil, nil
}

func (m *mockSD) GetLoRAs() ([]sd.LoRA, error) {
	return nil, nil
}

func (m *mockSD) GetOptions() (map[string]interface{}, error) {
	return nil, nil
}

func (m *mockSD) GetProgress() (*sd.ProgressResponse, error) {
	return nil, nil
}

func (m *mockSD) Interrupt() error {
	return nil
}

func (m *mockSD) HealthCheck() error {
	return m.err
}

func (m *mockSD) SetURL(u string) {
	m.url = u
}

func (m *mockSD) SetModel(name string) error {
	m.modelName = name
	return nil
}

func (m *mockSD) SetVAE(_ string) error {
	return nil
}

func openTestDB(t *testing.T) *preset.DB {
	t.Helper()
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "test.db")
	db, err := preset.Open(dbPath)
	require.NoError(t, err)
	t.Cleanup(func() { db.Close() })
	return db
}

func defaultConfig() *config.Config {
	return &config.Config{
		LLMUrl:        "http://localhost:1234",
		SDUrl:         "http://localhost:7860",
		LLMModel:      "test-model",
		SDPromptModel: "default",
		LLMBackend:    "lmstudio",
		SystemPrompt:  "system",
	}
}

func setupHandler(t *testing.T, llmClient *mockLLM, sdClient *mockSD) *Handler {
	t.Helper()
	db := openTestDB(t)
	cfg := defaultConfig()
	return NewHandler(db, llmClient, sdClient, cfg)
}

func setupServer(t *testing.T, h *Handler) *httptest.Server {
	t.Helper()
	mux := http.NewServeMux()
	h.RegisterRoutes(mux)
	ts := httptest.NewServer(mux)
	t.Cleanup(func() { ts.Close() })
	return ts
}

func doRequest(t *testing.T, method, url string, body any) *http.Response {
	t.Helper()
	var reader io.Reader
	if body != nil {
		b, err := json.Marshal(body)
		require.NoError(t, err)
		reader = bytes.NewReader(b)
	}
	req, err := http.NewRequest(method, url, reader)
	require.NoError(t, err)
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	resp, err := http.DefaultClient.Do(req)
	require.NoError(t, err)
	return resp
}

func decodeJSON(t *testing.T, resp *http.Response, target any) {
	t.Helper()
	defer resp.Body.Close()
	err := json.NewDecoder(resp.Body).Decode(target)
	require.NoError(t, err)
}

func decodeError(t *testing.T, resp *http.Response) string {
	t.Helper()
	var m map[string]string
	decodeJSON(t, resp, &m)
	return m["error"]
}

func TestNewHandler_NilValues(t *testing.T) {
	t.Parallel()
	h := NewHandler(nil, nil, nil, nil)
	assert.NotNil(t, h)
}

func TestRegisterRoutes_AllEndpoints(t *testing.T) {
	t.Parallel()
	h := setupHandler(t, &mockLLM{}, &mockSD{})
	mux := http.NewServeMux()
	h.RegisterRoutes(mux)
	ts := httptest.NewServer(mux)
	defer ts.Close()

	unregisteredResp := doRequest(t, "GET", ts.URL+"/api/nonexistent-route", nil)
	unregisteredResp.Body.Close()
	assert.Equal(t, http.StatusNotFound, unregisteredResp.StatusCode)

	endpoints := []struct {
		method string
		path   string
	}{
		{"GET", "/api/presets"},
		{"GET", "/api/presets/type/character"},
		{"POST", "/api/presets"},
		{"DELETE", "/api/presets/1"},
		{"POST", "/api/generate-sd-prompt"},
		{"GET", "/api/sd/models"},
		{"GET", "/api/sd/samplers"},
		{"GET", "/api/sd/schedulers"},
		{"GET", "/api/llm/models"},
		{"GET", "/api/settings"},
		{"PUT", "/api/settings"},
		{"GET", "/api/descriptions"},
		{"POST", "/api/descriptions"},
		{"DELETE", "/api/descriptions/1"},
	}
	for _, ep := range endpoints {
		var body io.Reader
		if ep.method == "POST" || ep.method == "PUT" {
			body = bytes.NewReader([]byte(`{}`))
		}
		req, err := http.NewRequest(ep.method, ts.URL+ep.path, body)
		require.NoError(t, err)
		if body != nil {
			req.Header.Set("Content-Type", "application/json")
		}
		resp, err := http.DefaultClient.Do(req)
		require.NoError(t, err)
		resp.Body.Close()
		assert.NotEqual(t, http.StatusNotFound, resp.StatusCode,
			"route %s %s should be registered", ep.method, ep.path)
	}
}

func TestListPresets_Empty(t *testing.T) {
	t.Parallel()
	h := setupHandler(t, &mockLLM{}, &mockSD{})
	ts := setupServer(t, h)

	resp := doRequest(t, "GET", ts.URL+"/api/presets", nil)
	assert.Equal(t, http.StatusOK, resp.StatusCode)

	var result []preset.Preset
	decodeJSON(t, resp, &result)
	assert.Equal(t, []preset.Preset{}, result)
}

func TestListPresets_WithData(t *testing.T) {
	t.Parallel()
	db := openTestDB(t)
	p := &preset.Preset{Name: "test", Prompt: "prompt", Sampler: "Euler a", Steps: 20, CfgScale: 7, Width: 512, Height: 512}
	require.NoError(t, db.Create(p))

	h := NewHandler(db, &mockLLM{}, &mockSD{}, defaultConfig())
	ts := setupServer(t, h)

	resp := doRequest(t, "GET", ts.URL+"/api/presets", nil)
	assert.Equal(t, http.StatusOK, resp.StatusCode)

	var result []preset.Preset
	decodeJSON(t, resp, &result)
	require.Len(t, result, 1)
	assert.Equal(t, "test", result[0].Name)
}

func TestListPresetsByType_Filter(t *testing.T) {
	t.Parallel()
	db := openTestDB(t)
	character := &preset.Preset{Name: "char", PresetType: "character", Prompt: "p", Sampler: "Euler a", Steps: 20, CfgScale: 7, Width: 512, Height: 512}
	bg := &preset.Preset{Name: "bg", PresetType: "background", Prompt: "p", Sampler: "Euler a", Steps: 20, CfgScale: 7, Width: 512, Height: 512}
	require.NoError(t, db.Create(character))
	require.NoError(t, db.Create(bg))

	h := NewHandler(db, &mockLLM{}, &mockSD{}, defaultConfig())
	ts := setupServer(t, h)

	resp := doRequest(t, "GET", ts.URL+"/api/presets/type/character", nil)
	assert.Equal(t, http.StatusOK, resp.StatusCode)

	var result []preset.Preset
	decodeJSON(t, resp, &result)
	require.Len(t, result, 1)
	assert.Equal(t, "character", result[0].PresetType)
}

func TestListPresetsByType_EmptyResult(t *testing.T) {
	t.Parallel()
	h := setupHandler(t, &mockLLM{}, &mockSD{})
	ts := setupServer(t, h)

	resp := doRequest(t, "GET", ts.URL+"/api/presets/type/nonexistent", nil)
	assert.Equal(t, http.StatusOK, resp.StatusCode)

	var result []preset.Preset
	decodeJSON(t, resp, &result)
	assert.Equal(t, []preset.Preset{}, result)
}

func TestGetPreset_Valid(t *testing.T) {
	t.Parallel()
	db := openTestDB(t)
	p := &preset.Preset{Name: "target", Prompt: "p", Sampler: "Euler a", Steps: 20, CfgScale: 7, Width: 512, Height: 512}
	require.NoError(t, db.Create(p))

	h := NewHandler(db, &mockLLM{}, &mockSD{}, defaultConfig())
	ts := setupServer(t, h)

	resp := doRequest(t, "GET", fmt.Sprintf("%s/api/presets/%d", ts.URL, p.ID), nil)
	assert.Equal(t, http.StatusOK, resp.StatusCode)

	var result preset.Preset
	decodeJSON(t, resp, &result)
	assert.Equal(t, "target", result.Name)
}

func TestGetPreset_InvalidID(t *testing.T) {
	t.Parallel()
	h := setupHandler(t, &mockLLM{}, &mockSD{})
	ts := setupServer(t, h)

	resp := doRequest(t, "GET", ts.URL+"/api/presets/abc", nil)
	assert.Equal(t, http.StatusBadRequest, resp.StatusCode)
	assert.Equal(t, "invalid id", decodeError(t, resp))
}

func TestGetPreset_NotFound(t *testing.T) {
	t.Parallel()
	h := setupHandler(t, &mockLLM{}, &mockSD{})
	ts := setupServer(t, h)

	resp := doRequest(t, "GET", ts.URL+"/api/presets/9999", nil)
	assert.Equal(t, http.StatusNotFound, resp.StatusCode)
	assert.Equal(t, "preset not found", decodeError(t, resp))
}

func TestCreatePreset_Valid(t *testing.T) {
	t.Parallel()
	h := setupHandler(t, &mockLLM{}, &mockSD{})
	ts := setupServer(t, h)

	body := map[string]any{
		"name": "new preset", "prompt": "test prompt", "sampler": "Euler a",
		"steps": 20, "cfg_scale": 7.0, "width": 512, "height": 512,
	}
	resp := doRequest(t, "POST", ts.URL+"/api/presets", body)
	assert.Equal(t, http.StatusCreated, resp.StatusCode)

	var result preset.Preset
	decodeJSON(t, resp, &result)
	assert.Equal(t, "new preset", result.Name)
	assert.NotZero(t, result.ID)
}

func TestCreatePreset_InvalidJSON(t *testing.T) {
	t.Parallel()
	h := setupHandler(t, &mockLLM{}, &mockSD{})
	ts := setupServer(t, h)

	req, err := http.NewRequest("POST", ts.URL+"/api/presets", bytes.NewReader([]byte("not json")))
	require.NoError(t, err)
	req.Header.Set("Content-Type", "application/json")
	resp, err := http.DefaultClient.Do(req)
	require.NoError(t, err)

	assert.Equal(t, http.StatusBadRequest, resp.StatusCode)
	assert.Equal(t, "invalid json", decodeError(t, resp))
}

func TestUpdatePreset_Valid(t *testing.T) {
	t.Parallel()
	db := openTestDB(t)
	p := &preset.Preset{Name: "original", Prompt: "p", Sampler: "Euler a", Steps: 20, CfgScale: 7, Width: 512, Height: 512}
	require.NoError(t, db.Create(p))

	h := NewHandler(db, &mockLLM{}, &mockSD{}, defaultConfig())
	ts := setupServer(t, h)

	body := map[string]any{
		"name": "updated", "prompt": "new prompt", "sampler": "DPM++ 2M",
		"steps": 30, "cfg_scale": 8.0, "width": 768, "height": 768,
	}
	resp := doRequest(t, "PUT", fmt.Sprintf("%s/api/presets/%d", ts.URL, p.ID), body)
	assert.Equal(t, http.StatusOK, resp.StatusCode)

	var result preset.Preset
	decodeJSON(t, resp, &result)
	assert.Equal(t, "updated", result.Name)
	assert.Equal(t, p.ID, result.ID)
}

func TestUpdatePreset_InvalidID(t *testing.T) {
	t.Parallel()
	h := setupHandler(t, &mockLLM{}, &mockSD{})
	ts := setupServer(t, h)

	resp := doRequest(t, "PUT", ts.URL+"/api/presets/abc", map[string]any{"name": "x"})
	assert.Equal(t, http.StatusBadRequest, resp.StatusCode)
	assert.Equal(t, "invalid id", decodeError(t, resp))
}

func TestUpdatePreset_InvalidJSON(t *testing.T) {
	t.Parallel()
	h := setupHandler(t, &mockLLM{}, &mockSD{})
	ts := setupServer(t, h)

	req, err := http.NewRequest("PUT", ts.URL+"/api/presets/1", bytes.NewReader([]byte("bad")))
	require.NoError(t, err)
	req.Header.Set("Content-Type", "application/json")
	resp, err := http.DefaultClient.Do(req)
	require.NoError(t, err)

	assert.Equal(t, http.StatusBadRequest, resp.StatusCode)
	assert.Equal(t, "invalid json", decodeError(t, resp))
}

func TestDeletePreset_Valid(t *testing.T) {
	t.Parallel()
	db := openTestDB(t)
	p := &preset.Preset{Name: "to-delete", Prompt: "p", Sampler: "Euler a", Steps: 20, CfgScale: 7, Width: 512, Height: 512}
	require.NoError(t, db.Create(p))

	h := NewHandler(db, &mockLLM{}, &mockSD{}, defaultConfig())
	ts := setupServer(t, h)

	resp := doRequest(t, "DELETE", fmt.Sprintf("%s/api/presets/%d", ts.URL, p.ID), nil)
	assert.Equal(t, http.StatusOK, resp.StatusCode)

	var result map[string]bool
	decodeJSON(t, resp, &result)
	assert.True(t, result["deleted"])

	_, err := db.Get(p.ID)
	assert.Error(t, err)
}

func TestDeletePreset_InvalidID(t *testing.T) {
	t.Parallel()
	h := setupHandler(t, &mockLLM{}, &mockSD{})
	ts := setupServer(t, h)

	resp := doRequest(t, "DELETE", ts.URL+"/api/presets/abc", nil)
	assert.Equal(t, http.StatusBadRequest, resp.StatusCode)
	assert.Equal(t, "invalid id", decodeError(t, resp))
}

func TestDeletePreset_NonExistentStill200(t *testing.T) {
	t.Parallel()
	h := setupHandler(t, &mockLLM{}, &mockSD{})
	ts := setupServer(t, h)

	resp := doRequest(t, "DELETE", ts.URL+"/api/presets/9999", nil)
	assert.Equal(t, http.StatusOK, resp.StatusCode)
}

func TestGenerateSDPrompt_Valid(t *testing.T) {
	t.Parallel()
	llmClient := &mockLLM{prompt: "best quality, 1girl, solo"}
	h := setupHandler(t, llmClient, &mockSD{})
	ts := setupServer(t, h)

	body := map[string]any{"description": "anime girl in garden", "preset_type": "character"}
	resp := doRequest(t, "POST", ts.URL+"/api/generate-sd-prompt", body)
	assert.Equal(t, http.StatusOK, resp.StatusCode)

	var result map[string]string
	decodeJSON(t, resp, &result)
	assert.Equal(t, "best quality, 1girl, solo", result["prompt"])
}

func TestGenerateSDPrompt_EmptyDescription(t *testing.T) {
	t.Parallel()
	h := setupHandler(t, &mockLLM{}, &mockSD{})
	ts := setupServer(t, h)

	tests := []struct {
		name        string
		description string
	}{
		{"empty", ""},
		{"whitespace", "   "},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			body := map[string]any{"description": tt.description}
			resp := doRequest(t, "POST", ts.URL+"/api/generate-sd-prompt", body)
			assert.Equal(t, http.StatusBadRequest, resp.StatusCode)
			assert.Equal(t, "description is required", decodeError(t, resp))
		})
	}
}

func TestGenerateSDPrompt_InvalidJSON(t *testing.T) {
	t.Parallel()
	h := setupHandler(t, &mockLLM{}, &mockSD{})
	ts := setupServer(t, h)

	req, err := http.NewRequest("POST", ts.URL+"/api/generate-sd-prompt", bytes.NewReader([]byte("not json")))
	require.NoError(t, err)
	req.Header.Set("Content-Type", "application/json")
	resp, err := http.DefaultClient.Do(req)
	require.NoError(t, err)

	assert.Equal(t, http.StatusBadRequest, resp.StatusCode)
	assert.Equal(t, "invalid json", decodeError(t, resp))
}

func TestGenerateSDPrompt_LLMError(t *testing.T) {
	t.Parallel()
	llmClient := &mockLLM{err: fmt.Errorf("connection refused")}
	h := setupHandler(t, llmClient, &mockSD{})
	ts := setupServer(t, h)

	body := map[string]any{"description": "test"}
	resp := doRequest(t, "POST", ts.URL+"/api/generate-sd-prompt", body)
	assert.Equal(t, http.StatusInternalServerError, resp.StatusCode)
	assert.Contains(t, decodeError(t, resp), "LLM error")
}

func TestGenerateSDPrompt_UsesMaxTokens(t *testing.T) {
	t.Parallel()
	db := openTestDB(t)
	require.NoError(t, db.SetSetting("llm_max_tokens", "2048"))

	llmClient := &mockLLM{prompt: "generated"}
	h := NewHandler(db, llmClient, &mockSD{}, defaultConfig())
	ts := setupServer(t, h)

	body := map[string]any{"description": "test", "preset_type": ""}
	resp := doRequest(t, "POST", ts.URL+"/api/generate-sd-prompt", body)
	assert.Equal(t, http.StatusOK, resp.StatusCode)
}

func TestGenerateImage_Valid(t *testing.T) {
	t.Parallel()
	db := openTestDB(t)
	p := &preset.Preset{Name: "gen", Prompt: "base prompt", NegativePrompt: "bad", Sampler: "Euler a", Steps: 20, CfgScale: 7, Width: 512, Height: 512}
	require.NoError(t, db.Create(p))

	sdClient := &mockSD{
		result: &sd.Txt2ImgResponse{
			Images:     []string{"base64data"},
			Parameters: json.RawMessage(`{}`),
			Info:       json.RawMessage(`{}`),
		},
	}
	h := NewHandler(db, &mockLLM{}, sdClient, defaultConfig())
	ts := setupServer(t, h)

	body := map[string]any{
		"preset_id":             p.ID,
		"extra_prompt":          "extra",
		"extra_negative_prompt": "extra neg",
	}
	resp := doRequest(t, "POST", ts.URL+"/api/generate", body)
	assert.Equal(t, http.StatusOK, resp.StatusCode)

	var result map[string]any
	decodeJSON(t, resp, &result)
	assert.Equal(t, "base64data", result["image"])
}

func TestGenerateImage_PresetNotFound(t *testing.T) {
	t.Parallel()
	h := setupHandler(t, &mockLLM{}, &mockSD{})
	ts := setupServer(t, h)

	body := map[string]any{"preset_id": 9999}
	resp := doRequest(t, "POST", ts.URL+"/api/generate", body)
	assert.Equal(t, http.StatusNotFound, resp.StatusCode)
	assert.Equal(t, "preset not found", decodeError(t, resp))
}

func TestGenerateImage_InvalidJSON(t *testing.T) {
	t.Parallel()
	h := setupHandler(t, &mockLLM{}, &mockSD{})
	ts := setupServer(t, h)

	req, err := http.NewRequest("POST", ts.URL+"/api/generate", bytes.NewReader([]byte("bad")))
	require.NoError(t, err)
	req.Header.Set("Content-Type", "application/json")
	resp, err := http.DefaultClient.Do(req)
	require.NoError(t, err)

	assert.Equal(t, http.StatusBadRequest, resp.StatusCode)
}

func TestGenerateImage_SDError(t *testing.T) {
	t.Parallel()
	db := openTestDB(t)
	p := &preset.Preset{Name: "gen", Prompt: "p", Sampler: "Euler a", Steps: 20, CfgScale: 7, Width: 512, Height: 512}
	require.NoError(t, db.Create(p))

	sdClient := &mockSD{err: fmt.Errorf("SD unavailable")}
	h := NewHandler(db, &mockLLM{}, sdClient, defaultConfig())
	ts := setupServer(t, h)

	body := map[string]any{"preset_id": p.ID}
	resp := doRequest(t, "POST", ts.URL+"/api/generate", body)
	assert.Equal(t, http.StatusInternalServerError, resp.StatusCode)
	assert.Contains(t, decodeError(t, resp), "SD error")
}

func TestGenerateImage_NoImagesReturned(t *testing.T) {
	t.Parallel()
	db := openTestDB(t)
	p := &preset.Preset{Name: "gen", Prompt: "p", Sampler: "Euler a", Steps: 20, CfgScale: 7, Width: 512, Height: 512}
	require.NoError(t, db.Create(p))

	sdClient := &mockSD{result: &sd.Txt2ImgResponse{Images: []string{}}}
	h := NewHandler(db, &mockLLM{}, sdClient, defaultConfig())
	ts := setupServer(t, h)

	body := map[string]any{"preset_id": p.ID}
	resp := doRequest(t, "POST", ts.URL+"/api/generate", body)
	assert.Equal(t, http.StatusInternalServerError, resp.StatusCode)
	assert.Equal(t, "no images returned", decodeError(t, resp))
}

func TestGenerateImage_WithModelName(t *testing.T) {
	t.Parallel()
	db := openTestDB(t)
	p := &preset.Preset{Name: "gen", Prompt: "p", ModelName: "sdxl", Sampler: "Euler a", Steps: 20, CfgScale: 7, Width: 512, Height: 512}
	require.NoError(t, db.Create(p))

	sdClient := &mockSD{
		result: &sd.Txt2ImgResponse{Images: []string{"img"}, Parameters: json.RawMessage(`{}`), Info: json.RawMessage(`{}`)},
	}
	h := NewHandler(db, &mockLLM{}, sdClient, defaultConfig())
	ts := setupServer(t, h)

	body := map[string]any{"preset_id": p.ID}
	resp := doRequest(t, "POST", ts.URL+"/api/generate", body)
	assert.Equal(t, http.StatusOK, resp.StatusCode)
	assert.Equal(t, "sdxl", sdClient.modelName)
}

func TestGenerateImage_WithScheduleType(t *testing.T) {
	t.Parallel()
	db := openTestDB(t)
	p := &preset.Preset{Name: "gen", Prompt: "p", Sampler: "DPM++ 2M", ScheduleType: "karras", Steps: 20, CfgScale: 7, Width: 512, Height: 512}
	require.NoError(t, db.Create(p))

	sdClient := &mockSD{
		result: &sd.Txt2ImgResponse{Images: []string{"img"}, Parameters: json.RawMessage(`{}`), Info: json.RawMessage(`{}`)},
	}
	h := NewHandler(db, &mockLLM{}, sdClient, defaultConfig())
	ts := setupServer(t, h)

	body := map[string]any{"preset_id": p.ID}
	resp := doRequest(t, "POST", ts.URL+"/api/generate", body)
	assert.Equal(t, http.StatusOK, resp.StatusCode)
}

func TestGenerateImage_HiresFixDefaults(t *testing.T) {
	t.Parallel()
	db := openTestDB(t)
	hf := true
	p := &preset.Preset{Name: "gen", Prompt: "p", Sampler: "Euler a", Steps: 20, CfgScale: 7, Width: 512, Height: 512, HiresFix: &hf}
	require.NoError(t, db.Create(p))

	sdClient := &mockSD{
		result: &sd.Txt2ImgResponse{Images: []string{"img"}, Parameters: json.RawMessage(`{}`), Info: json.RawMessage(`{}`)},
	}
	h := NewHandler(db, &mockLLM{}, sdClient, defaultConfig())
	ts := setupServer(t, h)

	body := map[string]any{"preset_id": p.ID}
	resp := doRequest(t, "POST", ts.URL+"/api/generate", body)
	assert.Equal(t, http.StatusOK, resp.StatusCode)
}

func TestGetSDModels_Success(t *testing.T) {
	t.Parallel()
	sdClient := &mockSD{
		models: []sd.SDModel{{Title: "SDXL", Name: "sdxl.safetensors"}},
	}
	h := setupHandler(t, &mockLLM{}, sdClient)
	ts := setupServer(t, h)

	resp := doRequest(t, "GET", ts.URL+"/api/sd/models", nil)
	assert.Equal(t, http.StatusOK, resp.StatusCode)

	var result []sd.SDModel
	decodeJSON(t, resp, &result)
	require.Len(t, result, 1)
	assert.Equal(t, "SDXL", result[0].Title)
}

func TestGetSDModels_Error(t *testing.T) {
	t.Parallel()
	sdClient := &mockSD{err: fmt.Errorf("unreachable")}
	h := setupHandler(t, &mockLLM{}, sdClient)
	ts := setupServer(t, h)

	resp := doRequest(t, "GET", ts.URL+"/api/sd/models", nil)
	assert.Equal(t, http.StatusInternalServerError, resp.StatusCode)
}

func TestGetSDSamplers_Success(t *testing.T) {
	t.Parallel()
	sdClient := &mockSD{
		samplers: []sd.Sampler{{Name: "Euler a"}, {Name: "DPM++ 2M"}},
	}
	h := setupHandler(t, &mockLLM{}, sdClient)
	ts := setupServer(t, h)

	resp := doRequest(t, "GET", ts.URL+"/api/sd/samplers", nil)
	assert.Equal(t, http.StatusOK, resp.StatusCode)

	var result []sd.Sampler
	decodeJSON(t, resp, &result)
	require.Len(t, result, 2)
}

func TestGetSDSamplers_Error(t *testing.T) {
	t.Parallel()
	sdClient := &mockSD{err: fmt.Errorf("fail")}
	h := setupHandler(t, &mockLLM{}, sdClient)
	ts := setupServer(t, h)

	resp := doRequest(t, "GET", ts.URL+"/api/sd/samplers", nil)
	assert.Equal(t, http.StatusInternalServerError, resp.StatusCode)
}

func TestGetSDSchedulers_Success(t *testing.T) {
	t.Parallel()
	sdClient := &mockSD{
		schedulers: []sd.Scheduler{{Name: "Automatic"}, {Name: "Karras"}},
	}
	h := setupHandler(t, &mockLLM{}, sdClient)
	ts := setupServer(t, h)

	resp := doRequest(t, "GET", ts.URL+"/api/sd/schedulers", nil)
	assert.Equal(t, http.StatusOK, resp.StatusCode)

	var result []sd.Scheduler
	decodeJSON(t, resp, &result)
	require.Len(t, result, 2)
}

func TestGetSDSchedulers_Error(t *testing.T) {
	t.Parallel()
	sdClient := &mockSD{err: fmt.Errorf("fail")}
	h := setupHandler(t, &mockLLM{}, sdClient)
	ts := setupServer(t, h)

	resp := doRequest(t, "GET", ts.URL+"/api/sd/schedulers", nil)
	assert.Equal(t, http.StatusInternalServerError, resp.StatusCode)
}

func TestGetLLMModels_Success(t *testing.T) {
	t.Parallel()
	llmClient := &mockLLM{
		models: []llm.LLMModel{{ID: "gpt-test", Object: "model"}},
	}
	h := setupHandler(t, llmClient, &mockSD{})
	ts := setupServer(t, h)

	resp := doRequest(t, "GET", ts.URL+"/api/llm/models", nil)
	assert.Equal(t, http.StatusOK, resp.StatusCode)

	var result []llm.LLMModel
	decodeJSON(t, resp, &result)
	require.Len(t, result, 1)
	assert.Equal(t, "gpt-test", result[0].ID)
}

func TestGetLLMModels_Error(t *testing.T) {
	t.Parallel()
	llmClient := &mockLLM{err: fmt.Errorf("unreachable")}
	h := setupHandler(t, llmClient, &mockSD{})
	ts := setupServer(t, h)

	resp := doRequest(t, "GET", ts.URL+"/api/llm/models", nil)
	assert.Equal(t, http.StatusInternalServerError, resp.StatusCode)
}

func TestGetSettings_Defaults(t *testing.T) {
	t.Parallel()
	h := setupHandler(t, &mockLLM{}, &mockSD{})
	ts := setupServer(t, h)

	resp := doRequest(t, "GET", ts.URL+"/api/settings", nil)
	assert.Equal(t, http.StatusOK, resp.StatusCode)

	var result map[string]string
	decodeJSON(t, resp, &result)
	assert.Equal(t, "http://localhost:1234", result["llm_url"])
	assert.Equal(t, "http://localhost:7860", result["sd_url"])
	assert.Equal(t, "test-model", result["llm_model"])
	assert.Equal(t, "lmstudio", result["llm_backend"])
	assert.Equal(t, "5m", result["llm_keep_alive"])
}

func TestGetSettings_WithOverrides(t *testing.T) {
	t.Parallel()
	db := openTestDB(t)
	require.NoError(t, db.SetSetting("llm_url", "http://custom:9999"))

	h := NewHandler(db, &mockLLM{}, &mockSD{}, defaultConfig())
	ts := setupServer(t, h)

	resp := doRequest(t, "GET", ts.URL+"/api/settings", nil)
	assert.Equal(t, http.StatusOK, resp.StatusCode)

	var result map[string]string
	decodeJSON(t, resp, &result)
	assert.Equal(t, "http://custom:9999", result["llm_url"])
}

func TestUpdateSettings_Valid(t *testing.T) {
	t.Parallel()
	llmClient := &mockLLM{}
	sdClient := &mockSD{}
	db := openTestDB(t)
	cfg := defaultConfig()
	h := NewHandler(db, llmClient, sdClient, cfg)
	ts := setupServer(t, h)

	body := map[string]string{
		"llm_url": "http://new-llm:1234",
		"sd_url":  "http://new-sd:7860",
	}
	resp := doRequest(t, "PUT", ts.URL+"/api/settings", body)
	assert.Equal(t, http.StatusOK, resp.StatusCode)

	var result map[string]bool
	decodeJSON(t, resp, &result)
	assert.True(t, result["saved"])

	assert.Equal(t, "http://new-llm:1234", llmClient.url)
	assert.Equal(t, "http://new-sd:7860", sdClient.url)
	assert.Equal(t, "http://new-llm:1234", cfg.LLMUrl)
	assert.Equal(t, "http://new-sd:7860", cfg.SDUrl)
}

func TestUpdateSettings_InvalidURL(t *testing.T) {
	t.Parallel()
	h := setupHandler(t, &mockLLM{}, &mockSD{})
	ts := setupServer(t, h)

	tests := []struct {
		name  string
		field string
		value string
	}{
		{"invalid sd_url", "sd_url", "http://[::1]:namedport"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			body := map[string]string{tt.field: tt.value}
			resp := doRequest(t, "PUT", ts.URL+"/api/settings", body)
			assert.Equal(t, http.StatusBadRequest, resp.StatusCode)
			assert.Contains(t, decodeError(t, resp), "invalid "+tt.field)
		})
	}
}

func TestUpdateSettings_InvalidNumeric(t *testing.T) {
	t.Parallel()
	h := setupHandler(t, &mockLLM{}, &mockSD{})
	ts := setupServer(t, h)

	tests := []struct {
		name  string
		field string
		value string
	}{
		{"negative llm_num_ctx", "llm_num_ctx", "-1"},
		{"non-numeric llm_max_tokens", "llm_max_tokens", "abc"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			body := map[string]string{tt.field: tt.value}
			resp := doRequest(t, "PUT", ts.URL+"/api/settings", body)
			assert.Equal(t, http.StatusBadRequest, resp.StatusCode)
			assert.Contains(t, decodeError(t, resp), "invalid "+tt.field)
		})
	}
}

func TestUpdateSettings_InvalidJSON(t *testing.T) {
	t.Parallel()
	h := setupHandler(t, &mockLLM{}, &mockSD{})
	ts := setupServer(t, h)

	req, err := http.NewRequest("PUT", ts.URL+"/api/settings", bytes.NewReader([]byte("bad")))
	require.NoError(t, err)
	req.Header.Set("Content-Type", "application/json")
	resp, err := http.DefaultClient.Do(req)
	require.NoError(t, err)

	assert.Equal(t, http.StatusBadRequest, resp.StatusCode)
	assert.Equal(t, "invalid json", decodeError(t, resp))
}

func TestUpdateSettings_DisallowedKeyIgnored(t *testing.T) {
	t.Parallel()
	db := openTestDB(t)
	h := NewHandler(db, &mockLLM{}, &mockSD{}, defaultConfig())
	ts := setupServer(t, h)

	body := map[string]string{
		"evil_injection": "hacked",
		"llm_model":      "new-model",
	}
	resp := doRequest(t, "PUT", ts.URL+"/api/settings", body)
	assert.Equal(t, http.StatusOK, resp.StatusCode)

	val, _ := db.GetSetting("evil_injection")
	assert.Equal(t, "", val)

	val, _ = db.GetSetting("llm_model")
	assert.Equal(t, "new-model", val)
}

func TestUpdateSettings_UpdatesConfigFields(t *testing.T) {
	t.Parallel()
	cfg := defaultConfig()
	db := openTestDB(t)
	h := NewHandler(db, &mockLLM{}, &mockSD{}, cfg)
	ts := setupServer(t, h)

	body := map[string]string{
		"llm_model":       "updated-model",
		"sd_prompt_model": "updated-sd-model",
	}
	resp := doRequest(t, "PUT", ts.URL+"/api/settings", body)
	assert.Equal(t, http.StatusOK, resp.StatusCode)

	assert.Equal(t, "updated-model", cfg.LLMModel)
	assert.Equal(t, "updated-sd-model", cfg.SDPromptModel)
}

func TestUpdateSettings_EmptyURLAllowed(t *testing.T) {
	t.Parallel()
	h := setupHandler(t, &mockLLM{}, &mockSD{})
	ts := setupServer(t, h)

	body := map[string]string{"llm_url": ""}
	resp := doRequest(t, "PUT", ts.URL+"/api/settings", body)
	assert.Equal(t, http.StatusOK, resp.StatusCode)
}

func TestUpdateSettings_ValidNumeric(t *testing.T) {
	t.Parallel()
	h := setupHandler(t, &mockLLM{}, &mockSD{})
	ts := setupServer(t, h)

	body := map[string]string{"llm_num_ctx": "4096"}
	resp := doRequest(t, "PUT", ts.URL+"/api/settings", body)
	assert.Equal(t, http.StatusOK, resp.StatusCode)
}

func TestUpdateSettings_EmptyNumericAllowed(t *testing.T) {
	t.Parallel()
	h := setupHandler(t, &mockLLM{}, &mockSD{})
	ts := setupServer(t, h)

	body := map[string]string{"llm_num_ctx": ""}
	resp := doRequest(t, "PUT", ts.URL+"/api/settings", body)
	assert.Equal(t, http.StatusOK, resp.StatusCode)
}

func TestListDescriptions_Empty(t *testing.T) {
	t.Parallel()
	h := setupHandler(t, &mockLLM{}, &mockSD{})
	ts := setupServer(t, h)

	resp := doRequest(t, "GET", ts.URL+"/api/descriptions", nil)
	assert.Equal(t, http.StatusOK, resp.StatusCode)

	var result []preset.SavedDescription
	decodeJSON(t, resp, &result)
	assert.Equal(t, []preset.SavedDescription{}, result)
}

func TestCreateDescription_Valid(t *testing.T) {
	t.Parallel()
	h := setupHandler(t, &mockLLM{}, &mockSD{})
	ts := setupServer(t, h)

	body := map[string]string{"text": "a beautiful landscape"}
	resp := doRequest(t, "POST", ts.URL+"/api/descriptions", body)
	assert.Equal(t, http.StatusCreated, resp.StatusCode)

	var result preset.SavedDescription
	decodeJSON(t, resp, &result)
	assert.Equal(t, "a beautiful landscape", result.Text)
	assert.NotZero(t, result.ID)
}

func TestCreateDescription_EmptyText(t *testing.T) {
	t.Parallel()
	h := setupHandler(t, &mockLLM{}, &mockSD{})
	ts := setupServer(t, h)

	tests := []struct {
		name string
		text string
	}{
		{"empty", ""},
		{"whitespace", "   "},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			body := map[string]string{"text": tt.text}
			resp := doRequest(t, "POST", ts.URL+"/api/descriptions", body)
			assert.Equal(t, http.StatusBadRequest, resp.StatusCode)
			assert.Equal(t, "text is required", decodeError(t, resp))
		})
	}
}

func TestCreateDescription_InvalidJSON(t *testing.T) {
	t.Parallel()
	h := setupHandler(t, &mockLLM{}, &mockSD{})
	ts := setupServer(t, h)

	req, err := http.NewRequest("POST", ts.URL+"/api/descriptions", bytes.NewReader([]byte("bad")))
	require.NoError(t, err)
	req.Header.Set("Content-Type", "application/json")
	resp, err := http.DefaultClient.Do(req)
	require.NoError(t, err)

	assert.Equal(t, http.StatusBadRequest, resp.StatusCode)
	assert.Equal(t, "invalid json", decodeError(t, resp))
}

func TestDeleteDescription_Valid(t *testing.T) {
	t.Parallel()
	db := openTestDB(t)
	saved, err := db.CreateDescription("to delete")
	require.NoError(t, err)

	h := NewHandler(db, &mockLLM{}, &mockSD{}, defaultConfig())
	ts := setupServer(t, h)

	resp := doRequest(t, "DELETE", fmt.Sprintf("%s/api/descriptions/%d", ts.URL, saved.ID), nil)
	assert.Equal(t, http.StatusOK, resp.StatusCode)

	var result map[string]bool
	decodeJSON(t, resp, &result)
	assert.True(t, result["deleted"])
}

func TestDeleteDescription_InvalidID(t *testing.T) {
	t.Parallel()
	h := setupHandler(t, &mockLLM{}, &mockSD{})
	ts := setupServer(t, h)

	resp := doRequest(t, "DELETE", ts.URL+"/api/descriptions/abc", nil)
	assert.Equal(t, http.StatusBadRequest, resp.StatusCode)
	assert.Equal(t, "invalid id", decodeError(t, resp))
}

func TestDeleteDescription_NonExistentStill200(t *testing.T) {
	t.Parallel()
	h := setupHandler(t, &mockLLM{}, &mockSD{})
	ts := setupServer(t, h)

	resp := doRequest(t, "DELETE", ts.URL+"/api/descriptions/9999", nil)
	assert.Equal(t, http.StatusOK, resp.StatusCode)
}

func TestListDescriptions_WithData(t *testing.T) {
	t.Parallel()
	db := openTestDB(t)
	_, err := db.CreateDescription("first")
	require.NoError(t, err)
	_, err = db.CreateDescription("second")
	require.NoError(t, err)

	h := NewHandler(db, &mockLLM{}, &mockSD{}, defaultConfig())
	ts := setupServer(t, h)

	resp := doRequest(t, "GET", ts.URL+"/api/descriptions", nil)
	assert.Equal(t, http.StatusOK, resp.StatusCode)

	var result []preset.SavedDescription
	decodeJSON(t, resp, &result)
	require.Len(t, result, 2)
}

func TestWriteJSON_ContentType(t *testing.T) {
	t.Parallel()
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, http.StatusOK, map[string]string{"ok": "true"})
	})
	ts := httptest.NewServer(handler)
	defer ts.Close()

	resp, err := http.Get(ts.URL + "/")
	require.NoError(t, err)
	defer resp.Body.Close()

	assert.Equal(t, "application/json", resp.Header.Get("Content-Type"))
}

func TestWriteError_Format(t *testing.T) {
	t.Parallel()
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		writeError(w, http.StatusBadRequest, "something went wrong")
	})
	ts := httptest.NewServer(handler)
	defer ts.Close()

	resp, err := http.Get(ts.URL + "/")
	require.NoError(t, err)
	defer resp.Body.Close()

	assert.Equal(t, http.StatusBadRequest, resp.StatusCode)
	assert.Equal(t, "application/json", resp.Header.Get("Content-Type"))

	var m map[string]string
	err = json.NewDecoder(resp.Body).Decode(&m)
	require.NoError(t, err)
	assert.Equal(t, "something went wrong", m["error"])
}

func TestWriteHTML_ContentType(t *testing.T) {
	t.Parallel()
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		writeHTML(w, "<h1>Hello</h1>")
	})
	ts := httptest.NewServer(handler)
	defer ts.Close()

	resp, err := http.Get(ts.URL + "/")
	require.NoError(t, err)
	defer resp.Body.Close()

	assert.Equal(t, "text/html; charset=utf-8", resp.Header.Get("Content-Type"))
	assert.Equal(t, http.StatusOK, resp.StatusCode)

	body, _ := io.ReadAll(resp.Body)
	assert.Equal(t, "<h1>Hello</h1>", string(body))
}

func TestMain(m *testing.M) {
	os.Exit(m.Run())
}
