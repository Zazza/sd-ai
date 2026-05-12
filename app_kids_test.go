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

func TestApp_ResolutionCRUD(t *testing.T) {
	t.Parallel()
	app, _ := newTestApp(t)

	r := preset.Resolution{Name: "Test Res", Width: 512, Height: 512}
	created, err := app.CreateResolution(r)
	require.NoError(t, err)
	assert.Greater(t, created.ID, int64(0))
	assert.Equal(t, "Test Res", created.Name)

	got, err := app.GetResolution(created.ID)
	require.NoError(t, err)
	assert.Equal(t, "Test Res", got.Name)

	created.Name = "Updated Res"
	created.Width = 768
	created.Height = 768
	updated, err := app.UpdateResolution(*created)
	require.NoError(t, err)
	assert.Equal(t, "Updated Res", updated.Name)
	assert.Equal(t, 768, updated.Width)

	err = app.DeleteResolution(created.ID)
	require.NoError(t, err)

	_, err = app.GetResolution(created.ID)
	assert.Error(t, err)
}

func TestApp_ResolutionValidation(t *testing.T) {
	t.Parallel()
	app, _ := newTestApp(t)

	tests := []struct {
		name    string
		r       preset.Resolution
		wantErr string
	}{
		{"empty name", preset.Resolution{Name: "", Width: 512, Height: 512}, "name is required"},
		{"whitespace name", preset.Resolution{Name: "   ", Width: 512, Height: 512}, "name is required"},
		{"width too small", preset.Resolution{Name: "Res", Width: 63, Height: 512}, "width and height must be between 64 and 4096"},
		{"width too large", preset.Resolution{Name: "Res", Width: 4097, Height: 512}, "width and height must be between 64 and 4096"},
		{"height too small", preset.Resolution{Name: "Res", Width: 512, Height: 32}, "width and height must be between 64 and 4096"},
		{"height too large", preset.Resolution{Name: "Res", Width: 512, Height: 5000}, "width and height must be between 64 and 4096"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := app.CreateResolution(tt.r)
			require.Error(t, err)
			assert.Contains(t, err.Error(), tt.wantErr)
		})
	}
}

func TestApp_ResolutionUpdateValidation(t *testing.T) {
	t.Parallel()
	app, _ := newTestApp(t)

	tests := []struct {
		name    string
		r       preset.Resolution
		wantErr string
	}{
		{"zero id", preset.Resolution{ID: 0, Name: "Res", Width: 512, Height: 512}, "id is required"},
		{"negative id", preset.Resolution{ID: -1, Name: "Res", Width: 512, Height: 512}, "id is required"},
		{"empty name", preset.Resolution{ID: 1, Name: "", Width: 512, Height: 512}, "name is required"},
		{"width out of range", preset.Resolution{ID: 1, Name: "Res", Width: 10, Height: 512}, "width and height must be between 64 and 4096"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := app.UpdateResolution(tt.r)
			require.Error(t, err)
			assert.Contains(t, err.Error(), tt.wantErr)
		})
	}
}

func TestApp_ListResolutionsEmptyNotNil(t *testing.T) {
	t.Parallel()
	app, db := newTestApp(t)

	items, err := db.ListResolutions()
	require.NoError(t, err)
	assert.NotEmpty(t, items)

	result, err := app.ListResolutions()
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestApp_HiresProfileCRUD(t *testing.T) {
	t.Parallel()
	app, _ := newTestApp(t)

	h := preset.HiresProfile{Name: "Test Profile", Upscale: 2.0, DenoisingStrength: 0.5, Upscaler: "R-ESRGAN 4x+"}
	created, err := app.CreateHiresProfile(h)
	require.NoError(t, err)
	assert.Greater(t, created.ID, int64(0))
	assert.Equal(t, "Test Profile", created.Name)

	got, err := app.GetHiresProfile(created.ID)
	require.NoError(t, err)
	assert.Equal(t, "Test Profile", got.Name)

	created.Name = "Updated Profile"
	created.Upscale = 3.0
	created.DenoisingStrength = 0.6
	updated, err := app.UpdateHiresProfile(*created)
	require.NoError(t, err)
	assert.Equal(t, "Updated Profile", updated.Name)
	assert.Equal(t, 3.0, updated.Upscale)

	err = app.DeleteHiresProfile(created.ID)
	require.NoError(t, err)

	_, err = app.GetHiresProfile(created.ID)
	assert.Error(t, err)
}

func TestApp_HiresProfileValidation(t *testing.T) {
	t.Parallel()
	app, _ := newTestApp(t)

	tests := []struct {
		name    string
		h       preset.HiresProfile
		wantErr string
	}{
		{"empty name", preset.HiresProfile{Name: "", Upscale: 2.0, DenoisingStrength: 0.5}, "name is required"},
		{"whitespace name", preset.HiresProfile{Name: "   ", Upscale: 2.0, DenoisingStrength: 0.5}, "name is required"},
		{"upscale too low", preset.HiresProfile{Name: "P", Upscale: 0.5, DenoisingStrength: 0.5}, "upscale must be between 1.0 and 4.0"},
		{"upscale too high", preset.HiresProfile{Name: "P", Upscale: 5.0, DenoisingStrength: 0.5}, "upscale must be between 1.0 and 4.0"},
		{"denoising too low", preset.HiresProfile{Name: "P", Upscale: 2.0, DenoisingStrength: -0.1}, "denoising_strength must be between 0.0 and 1.0"},
		{"denoising too high", preset.HiresProfile{Name: "P", Upscale: 2.0, DenoisingStrength: 1.1}, "denoising_strength must be between 0.0 and 1.0"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := app.CreateHiresProfile(tt.h)
			require.Error(t, err)
			assert.Contains(t, err.Error(), tt.wantErr)
		})
	}
}

func TestApp_HiresProfileUpdateValidation(t *testing.T) {
	t.Parallel()
	app, _ := newTestApp(t)

	tests := []struct {
		name    string
		h       preset.HiresProfile
		wantErr string
	}{
		{"zero id", preset.HiresProfile{ID: 0, Name: "P", Upscale: 2.0, DenoisingStrength: 0.5}, "id is required"},
		{"negative id", preset.HiresProfile{ID: -1, Name: "P", Upscale: 2.0, DenoisingStrength: 0.5}, "id is required"},
		{"empty name", preset.HiresProfile{ID: 1, Name: "", Upscale: 2.0, DenoisingStrength: 0.5}, "name is required"},
		{"upscale out of range", preset.HiresProfile{ID: 1, Name: "P", Upscale: 0.0, DenoisingStrength: 0.5}, "upscale must be between 1.0 and 4.0"},
		{"denoising out of range", preset.HiresProfile{ID: 1, Name: "P", Upscale: 2.0, DenoisingStrength: 2.0}, "denoising_strength must be between 0.0 and 1.0"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := app.UpdateHiresProfile(tt.h)
			require.Error(t, err)
			assert.Contains(t, err.Error(), tt.wantErr)
		})
	}
}

func TestApp_ListHiresProfilesEmptyNotNil(t *testing.T) {
	t.Parallel()
	app, db := newTestApp(t)

	items, err := db.ListHiresProfiles()
	require.NoError(t, err)
	assert.NotEmpty(t, items)

	result, err := app.ListHiresProfiles()
	require.NoError(t, err)
	assert.NotNil(t, result)
}
