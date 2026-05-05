package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"go-sd/internal/config"
	"go-sd/internal/llm"
	"go-sd/internal/preset"
	"go-sd/internal/sd"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func newTestApp(t *testing.T) (*App, *preset.DB) {
	t.Helper()
	db, err := preset.Open(":memory:")
	require.NoError(t, err)
	t.Cleanup(func() { db.Close() })
	cfg := &config.Config{LLMModel: "test-llm-model", SDPromptModel: "test-sd-model"}
	app := NewApp(db, llm.New("http://localhost:1234", "lmstudio"), sd.New("http://localhost:7860"), cfg)
	return app, db
}

func TestCheckServices_LLMModelField(t *testing.T) {
	llmSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode(map[string]interface{}{"status": "ok"})
	}))
	defer llmSrv.Close()

	sdSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/sdapi/v1/options" {
			json.NewEncoder(w).Encode(map[string]interface{}{"sd_model_checkpoint": "sd-1.5"})
			return
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer sdSrv.Close()

	db, err := preset.Open(":memory:")
	require.NoError(t, err)
	defer db.Close()

	cfg := &config.Config{LLMModel: "my-llm-model", SDPromptModel: "my-sd-model"}
	app := NewApp(db, llm.New(llmSrv.URL, "lmstudio"), sd.New(sdSrv.URL), cfg)

	status := app.CheckServices()
	assert.True(t, status.LLM.Available)
	assert.Equal(t, "my-sd-model", status.LLM.Model, "LLM model should use SDPromptModel from config as fallback")
	assert.True(t, status.SD.Available)
	assert.Equal(t, "sd-1.5", status.SD.Model)
}

func TestUpdateSettings_InvalidURL(t *testing.T) {
	app, _ := newTestApp(t)
	err := app.UpdateSettings(map[string]string{"llm_url": "::not-a-url"})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "invalid llm_url")
}

func TestUpdateSettings_ValidURL(t *testing.T) {
	app, _ := newTestApp(t)
	err := app.UpdateSettings(map[string]string{"llm_url": "http://localhost:1234"})
	assert.NoError(t, err)
}

func TestUpdateSettings_InvalidNumeric(t *testing.T) {
	app, _ := newTestApp(t)
	err := app.UpdateSettings(map[string]string{"llm_num_ctx": "abc"})
	assert.NoError(t, err)
	v, _ := app.presets.GetSetting("llm_num_ctx")
	assert.Equal(t, "0", v)
}

func TestUpdateSettings_NegativeNumeric(t *testing.T) {
	app, _ := newTestApp(t)
	err := app.UpdateSettings(map[string]string{"llm_max_tokens": "-1"})
	assert.NoError(t, err)
	v, _ := app.presets.GetSetting("llm_max_tokens")
	assert.Equal(t, "0", v)
}

func TestUpdateSettings_ValidNumeric(t *testing.T) {
	app, _ := newTestApp(t)
	err := app.UpdateSettings(map[string]string{"llm_num_ctx": "4096"})
	assert.NoError(t, err)
}

func TestLLMInterface(t *testing.T) {
	var _ llm.Service = llm.New("http://localhost:1234", "lmstudio")
}

func TestSDInterface(t *testing.T) {
	var _ sd.Service = sd.New("http://localhost:7860")
}
