package importexport

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"image"
	"image/png"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go-sd/internal/logger"
	"go-sd/internal/preset"
	"go-sd/internal/sd"
)

type mockSDService struct {
	models []sd.SDModel
	loras  []sd.LoRA
	err    error
}

func (m *mockSDService) Txt2Img(req sd.Txt2ImgRequest) (*sd.Txt2ImgResponse, error) {
	return nil, m.err
}
func (m *mockSDService) Img2Img(req sd.Img2ImgRequest) (*sd.Txt2ImgResponse, error) {
	return nil, m.err
}
func (m *mockSDService) GetModels() ([]sd.SDModel, error) {
	return m.models, m.err
}
func (m *mockSDService) GetSamplers() ([]sd.Sampler, error)          { return nil, nil }
func (m *mockSDService) GetSchedulers() ([]sd.Scheduler, error)      { return nil, nil }
func (m *mockSDService) GetUpscalers() ([]sd.Upscaler, error)        { return nil, nil }
func (m *mockSDService) GetVAEs() ([]sd.VAE, error)                  { return nil, nil }
func (m *mockSDService) GetLoRAs() ([]sd.LoRA, error)                { return m.loras, m.err }
func (m *mockSDService) GetOptions() (map[string]interface{}, error)  { return nil, nil }
func (m *mockSDService) GetProgress() (*sd.ProgressResponse, error)   { return nil, nil }
func (m *mockSDService) Interrupt() error                             { return nil }
func (m *mockSDService) HealthCheck() error                           { return nil }
func (m *mockSDService) SetURL(baseURL string)                        {}
func (m *mockSDService) SetModel(modelName string) error              { return nil }
func (m *mockSDService) SetVAE(vaeName string) error                  { return nil }
func (m *mockSDService) UpscaleImage(base64Img string, upscaler string, scale float64) (string, error) { return "", nil }

func openTestDB(t *testing.T) *preset.DB {
	t.Helper()
	db, err := preset.Open(":memory:")
	require.NoError(t, err)
	t.Cleanup(func() { db.Close() })
	return db
}

func newTestService(t *testing.T, mockSD sd.Service) *Service {
	t.Helper()
	db := openTestDB(t)
	log := logger.New(nil)
	return New(db, mockSD, log)
}

func makePreset(name, pType, model string, steps int) PresetData {
	return PresetData{
		Name:       name,
		PresetType: pType,
		Prompt:     "test prompt",
		Sampler:    "Euler a",
		Steps:      steps,
		CfgScale:   7.0,
		ModelName:  model,
	}
}

func createPresetInDB(t *testing.T, db *preset.DB, name string) int64 {
	t.Helper()
	p := &preset.Preset{
		Name:       name,
		PresetType: "test",
		Prompt:     "prompt",
		Sampler:    "Euler a",
		Steps:      20,
		CfgScale:   7.0,
		ModelName:  "model.safetensors",
	}
	err := db.Create(p)
	require.NoError(t, err)
	return p.ID
}

func encodePNG(t *testing.T, w, h int) string {
	t.Helper()
	img := image.NewRGBA(image.Rect(0, 0, w, h))
	var buf bytes.Buffer
	err := png.Encode(&buf, img)
	require.NoError(t, err)
	return base64.StdEncoding.EncodeToString(buf.Bytes())
}

func TestNew_ServiceCreated(t *testing.T) {
	t.Parallel()
	db := openTestDB(t)
	log := logger.New(nil)
	svc := New(db, &mockSDService{}, log)
	assert.NotNil(t, svc)
}

func TestPrepareExportData_EmptyIDs(t *testing.T) {
	t.Parallel()
	svc := newTestService(t, &mockSDService{})
	_, err := svc.PrepareExportData(nil)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "no presets selected")
}

func TestPrepareExportData_ValidIDs(t *testing.T) {
	t.Parallel()
	db := openTestDB(t)
	svc := New(db, &mockSDService{}, logger.New(nil))

	id := createPresetInDB(t, db, "export-me")
	result, err := svc.PrepareExportData([]int64{id})
	require.NoError(t, err)
	assert.Len(t, result, 1)
	assert.Equal(t, "export-me", result[0].Name)
	assert.Equal(t, "model.safetensors", result[0].ModelName)
	assert.Equal(t, 20, result[0].Steps)
}

func TestPrepareExportData_NonexistentIDs(t *testing.T) {
	t.Parallel()
	svc := newTestService(t, &mockSDService{})
	result, err := svc.PrepareExportData([]int64{9999})
	require.NoError(t, err)
	assert.Len(t, result, 0)
}

func TestPrepareExportData_WithPresetType(t *testing.T) {
	t.Parallel()
	db := openTestDB(t)
	svc := New(db, &mockSDService{}, logger.New(nil))

	pt := &preset.PresetType{Name: "Portrait"}
	err := db.CreatePresetType(pt)
	require.NoError(t, err)

	p := &preset.Preset{
		Name:       "typed-preset",
		PresetType: "test",
		TypeID:     &pt.ID,
		Prompt:     "prompt",
		Sampler:    "Euler a",
		Steps:      20,
		CfgScale:   7.0,
	}
	err = db.Create(p)
	require.NoError(t, err)

	result, err := svc.PrepareExportData([]int64{p.ID})
	require.NoError(t, err)
	assert.Len(t, result, 1)
	assert.Equal(t, "Portrait", result[0].TypeName)
}

func TestBuildExportFile_ValidPresets(t *testing.T) {
	t.Parallel()
	svc := newTestService(t, &mockSDService{})
	presets := []PresetData{makePreset("p1", "type1", "model1", 20)}
	data, err := svc.BuildExportFile(presets)
	require.NoError(t, err)

	var ef ExportFile
	require.NoError(t, json.Unmarshal(data, &ef))
	assert.Equal(t, 2, ef.Version)
	assert.Len(t, ef.Presets, 1)
	assert.Equal(t, "p1", ef.Presets[0].Name)
}

func TestBuildExportFile_EmptyPresets(t *testing.T) {
	t.Parallel()
	svc := newTestService(t, &mockSDService{})
	data, err := svc.BuildExportFile([]PresetData{})
	require.NoError(t, err)

	var ef ExportFile
	require.NoError(t, json.Unmarshal(data, &ef))
	assert.Equal(t, 2, ef.Version)
	assert.Empty(t, ef.Presets)
}

func TestBuildExportFile_MultiplePresets(t *testing.T) {
	t.Parallel()
	svc := newTestService(t, &mockSDService{})
	presets := []PresetData{
		makePreset("a", "t1", "m1", 20),
		makePreset("b", "t2", "m2", 30),
	}
	data, err := svc.BuildExportFile(presets)
	require.NoError(t, err)

	var ef ExportFile
	require.NoError(t, json.Unmarshal(data, &ef))
	assert.Len(t, ef.Presets, 2)
}

func TestParseImportFile_ValidV2(t *testing.T) {
	t.Parallel()
	svc := newTestService(t, &mockSDService{})

	ef := ExportFile{
		Version: 2,
		Presets: []PresetData{makePreset("imported", "test", "model", 20)},
	}
	raw, err := json.Marshal(ef)
	require.NoError(t, err)

	path := filepath.Join(t.TempDir(), "export.json")
	require.NoError(t, os.WriteFile(path, raw, 0o644))

	result, err := svc.ParseImportFile(path)
	require.NoError(t, err)
	assert.Len(t, result, 1)
	assert.Equal(t, "imported", result[0].Name)
	assert.Equal(t, "export.json", result[0].SourceFile)
}

func TestParseImportFile_ValidV1(t *testing.T) {
	t.Parallel()
	svc := newTestService(t, &mockSDService{})

	ef := ExportFile{
		Version: 1,
		Presets: []PresetData{makePreset("v1preset", "test", "m", 20)},
	}
	raw, err := json.Marshal(ef)
	require.NoError(t, err)

	path := filepath.Join(t.TempDir(), "v1.json")
	require.NoError(t, os.WriteFile(path, raw, 0o644))

	result, err := svc.ParseImportFile(path)
	require.NoError(t, err)
	assert.Len(t, result, 1)
}

func TestParseImportFile_UnsupportedVersion(t *testing.T) {
	t.Parallel()
	svc := newTestService(t, &mockSDService{})

	ef := ExportFile{Version: 99, Presets: []PresetData{}}
	raw, err := json.Marshal(ef)
	require.NoError(t, err)

	path := filepath.Join(t.TempDir(), "bad.json")
	require.NoError(t, os.WriteFile(path, raw, 0o644))

	_, err = svc.ParseImportFile(path)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "unsupported version")
}

func TestParseImportFile_FileNotFound(t *testing.T) {
	t.Parallel()
	svc := newTestService(t, &mockSDService{})
	_, err := svc.ParseImportFile("/nonexistent/file.json")
	assert.Error(t, err)
}

func TestParseImportFile_InvalidJSON(t *testing.T) {
	t.Parallel()
	svc := newTestService(t, &mockSDService{})

	path := filepath.Join(t.TempDir(), "invalid.json")
	require.NoError(t, os.WriteFile(path, []byte("not json"), 0o644))

	_, err := svc.ParseImportFile(path)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "parse json")
}

func TestParseImportFile_TooLarge(t *testing.T) {
	t.Parallel()
	svc := newTestService(t, &mockSDService{})

	path := filepath.Join(t.TempDir(), "big.json")
	bigData := make([]byte, 11*1024*1024)
	require.NoError(t, os.WriteFile(path, bigData, 0o644))

	_, err := svc.ParseImportFile(path)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "too large")
}

func TestValidateModels_AllFound(t *testing.T) {
	t.Parallel()
	mock := &mockSDService{
		models: []sd.SDModel{{Name: "model.safetensors"}},
		loras:  []sd.LoRA{{Name: "lora1"}},
	}
	svc := newTestService(t, mock)

	loras, _ := json.Marshal([]preset.LoRAEntry{{Name: "lora1", Weight: 0.8}})
	items := []PresetData{
		{Name: "p1", ModelName: "model.safetensors", Loras: string(loras)},
	}
	warnings, err := svc.ValidateModels(items)
	require.NoError(t, err)
	assert.Empty(t, warnings)
}

func TestValidateModels_ModelNotFound(t *testing.T) {
	t.Parallel()
	mock := &mockSDService{
		models: []sd.SDModel{{Name: "existing.safetensors"}},
	}
	svc := newTestService(t, mock)

	items := []PresetData{{Name: "p1", ModelName: "missing.safetensors"}}
	warnings, err := svc.ValidateModels(items)
	require.NoError(t, err)
	require.Len(t, warnings, 1)
	assert.Equal(t, "p1", warnings[0].PresetName)
	assert.Contains(t, warnings[0].Warnings[0], "Model not found")
}

func TestValidateModels_LoRANotFound(t *testing.T) {
	t.Parallel()
	mock := &mockSDService{
		models: []sd.SDModel{{Name: "model.safetensors"}},
		loras:  []sd.LoRA{{Name: "existing-lora"}},
	}
	svc := newTestService(t, mock)

	loras, _ := json.Marshal([]preset.LoRAEntry{{Name: "missing-lora", Weight: 1.0}})
	items := []PresetData{
		{Name: "p1", ModelName: "model.safetensors", Loras: string(loras)},
	}
	warnings, err := svc.ValidateModels(items)
	require.NoError(t, err)
	require.Len(t, warnings, 1)
	assert.Contains(t, warnings[0].Warnings[0], "LoRA not found")
}

func TestValidateModels_EmptyItems(t *testing.T) {
	t.Parallel()
	svc := newTestService(t, &mockSDService{})
	warnings, err := svc.ValidateModels(nil)
	require.NoError(t, err)
	assert.Nil(t, warnings)
}

func TestValidateModels_EmptyModelName(t *testing.T) {
	t.Parallel()
	mock := &mockSDService{models: []sd.SDModel{}}
	svc := newTestService(t, mock)

	items := []PresetData{{Name: "p1", ModelName: ""}}
	warnings, err := svc.ValidateModels(items)
	require.NoError(t, err)
	assert.Empty(t, warnings)
}

func TestImportItems_EmptyItems(t *testing.T) {
	t.Parallel()
	svc := newTestService(t, &mockSDService{})
	_, err := svc.ImportItems(nil)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "no presets selected")
}

func TestImportItems_TooManyItems(t *testing.T) {
	t.Parallel()
	svc := newTestService(t, &mockSDService{})
	items := make([]PresetData, 501)
	for i := range items {
		items[i] = makePreset("p", "t", "m", 20)
	}
	_, err := svc.ImportItems(items)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "too many presets")
}

func TestImportItems_EmptyName(t *testing.T) {
	t.Parallel()
	svc := newTestService(t, &mockSDService{})
	items := []PresetData{{Name: "   ", Steps: 20, CfgScale: 7.0}}
	_, err := svc.ImportItems(items)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "preset name is required")
}

func TestImportItems_ValidBatch(t *testing.T) {
	t.Parallel()
	svc := newTestService(t, &mockSDService{})
	items := []PresetData{
		makePreset("import-a", "type1", "model1", 20),
		makePreset("import-b", "type2", "model2", 30),
	}
	result, err := svc.ImportItems(items)
	require.NoError(t, err)
	assert.Len(t, result, 2)
	assert.Equal(t, "import-a", result[0].Name)
	assert.Equal(t, "import-b", result[1].Name)
	assert.True(t, result[0].ID > 0)
	assert.True(t, result[1].ID > 0)
}

func TestImportItems_WithTypeName(t *testing.T) {
	t.Parallel()
	svc := newTestService(t, &mockSDService{})
	items := []PresetData{
		{
			Name:       "typed",
			TypeName:   "Landscape",
			PresetType: "test",
			Prompt:     "prompt",
			Sampler:    "Euler a",
			Steps:      20,
			CfgScale:   7.0,
		},
	}
	result, err := svc.ImportItems(items)
	require.NoError(t, err)
	assert.Len(t, result, 1)
	assert.NotNil(t, result[0].TypeID)
}

func TestImportItems_InvalidSteps_Table(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name    string
		preset  PresetData
		errMsg  string
	}{
		{
			"steps_zero",
			PresetData{Name: "p", Steps: 0, CfgScale: 7.0},
			"invalid steps",
		},
		{
			"steps_too_high",
			PresetData{Name: "p", Steps: 200, CfgScale: 7.0},
			"invalid steps",
		},
		{
			"cfg_negative",
			PresetData{Name: "p", Steps: 20, CfgScale: -1.0},
			"invalid cfg_scale",
		},
		{
			"cfg_too_high",
			PresetData{Name: "p", Steps: 20, CfgScale: 50.0},
			"invalid cfg_scale",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			svc := newTestService(t, &mockSDService{})
			_, err := svc.ImportItems([]PresetData{tt.preset})
			assert.Error(t, err)
			assert.Contains(t, err.Error(), tt.errMsg)
		})
	}
}

func TestImportItems_InvalidOptionalFields_Table(t *testing.T) {
	t.Parallel()
	base := func() PresetData {
		return PresetData{Name: "p", Steps: 20, CfgScale: 7.0}
	}

	tests := []struct {
		name   string
		modify func(p *PresetData)
		errMsg string
	}{
		{
			"denoising_negative",
			func(p *PresetData) { v := -0.5; p.DenoisingStrength = &v },
			"invalid denoising_strength",
		},
		{
			"denoising_over_one",
			func(p *PresetData) { v := 1.5; p.DenoisingStrength = &v },
			"invalid denoising_strength",
		},
		{
			"clip_skip_zero",
			func(p *PresetData) { v := 0; p.ClipSkip = &v },
			"invalid clip_skip",
		},
		{
			"clip_skip_too_high",
			func(p *PresetData) { v := 20; p.ClipSkip = &v },
			"invalid clip_skip",
		},
		{
			"batch_size_zero",
			func(p *PresetData) { v := 0; p.BatchSize = &v },
			"invalid batch_size",
		},
		{
			"batch_size_too_high",
			func(p *PresetData) { v := 16; p.BatchSize = &v },
			"invalid batch_size",
		},
		{
			"batch_count_zero",
			func(p *PresetData) { v := 0; p.BatchCount = &v },
			"invalid batch_count",
		},
		{
			"batch_count_too_high",
			func(p *PresetData) { v := 16; p.BatchCount = &v },
			"invalid batch_count",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			svc := newTestService(t, &mockSDService{})
			p := base()
			tt.modify(&p)
			_, err := svc.ImportItems([]PresetData{p})
			assert.Error(t, err)
			assert.Contains(t, err.Error(), tt.errMsg)
		})
	}
}

func TestImportItems_ValidOptionalFields(t *testing.T) {
	t.Parallel()
	svc := newTestService(t, &mockSDService{})
	ds := 0.7
	cs := 2
	bs := 4
	bc := 2
	items := []PresetData{{
		Name:                   "full-optional",
		Steps:                  30,
		CfgScale:               7.5,
		DenoisingStrength:      &ds,
		ClipSkip:               &cs,
		BatchSize:              &bs,
		BatchCount:             &bc,
	}}
	result, err := svc.ImportItems(items)
	require.NoError(t, err)
	assert.Len(t, result, 1)
	assert.Equal(t, ds, *result[0].DenoisingStrength)
	assert.Equal(t, cs, *result[0].ClipSkip)
	assert.Equal(t, bs, *result[0].BatchSize)
	assert.Equal(t, bc, *result[0].BatchCount)
}

func TestProcessExportImage_NoImage(t *testing.T) {
	t.Parallel()
	svc := newTestService(t, &mockSDService{})
	_, err := svc.ProcessExportImage(ExportImageParams{})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "no image provided")
}

func TestProcessExportImage_UnsupportedFormat(t *testing.T) {
	t.Parallel()
	svc := newTestService(t, &mockSDService{})
	_, err := svc.ProcessExportImage(ExportImageParams{
		ImageBase64: "abc",
		Format:      "bmp",
	})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "unsupported format")
}

func TestProcessExportImage_UnsupportedInterpolation(t *testing.T) {
	t.Parallel()
	svc := newTestService(t, &mockSDService{})
	_, err := svc.ProcessExportImage(ExportImageParams{
		ImageBase64:   "abc",
		Format:        "png",
		Interpolation: "bicubic",
	})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "unsupported interpolation")
}

func TestProcessExportImage_InvalidBase64(t *testing.T) {
	t.Parallel()
	svc := newTestService(t, &mockSDService{})
	_, err := svc.ProcessExportImage(ExportImageParams{
		ImageBase64: "!!!not-base64!!!",
		Format:      "png",
	})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "decode base64")
}

func TestProcessExportImage_PNG_NoResize(t *testing.T) {
	t.Parallel()
	svc := newTestService(t, &mockSDService{})
	b64 := encodePNG(t, 64, 64)

	img, err := svc.ProcessExportImage(ExportImageParams{
		ImageBase64: b64,
		Format:      "png",
	})
	require.NoError(t, err)
	assert.NotEmpty(t, img.Data)
	assert.Equal(t, "png", img.Format)
	assert.Contains(t, img.Filename, ".png")
}

func TestProcessExportImage_JPEG_WithResize(t *testing.T) {
	t.Parallel()
	svc := newTestService(t, &mockSDService{})
	b64 := encodePNG(t, 256, 256)

	img, err := svc.ProcessExportImage(ExportImageParams{
		ImageBase64:   b64,
		Format:        "jpeg",
		Width:         128,
		Height:        128,
		Quality:       85,
		Interpolation: "lanczos",
		Filename:      "test",
	})
	require.NoError(t, err)
	assert.NotEmpty(t, img.Data)
	assert.Equal(t, "jpeg", img.Format)
	assert.Contains(t, img.Filename, ".jpeg")
}

func TestProcessExportImage_DefaultFilename(t *testing.T) {
	t.Parallel()
	svc := newTestService(t, &mockSDService{})
	b64 := encodePNG(t, 64, 64)

	img, err := svc.ProcessExportImage(ExportImageParams{
		ImageBase64: b64,
		Format:      "png",
	})
	require.NoError(t, err)
	assert.Contains(t, img.Filename, "export_")
	assert.True(t, strings.HasSuffix(img.Filename, ".png"))
}

func TestProcessExportImage_LockRatio_WidthOnly(t *testing.T) {
	t.Parallel()
	svc := newTestService(t, &mockSDService{})
	b64 := encodePNG(t, 100, 200)

	img, err := svc.ProcessExportImage(ExportImageParams{
		ImageBase64: b64,
		Format:      "png",
		Width:       50,
		LockRatio:   true,
	})
	require.NoError(t, err)
	assert.NotEmpty(t, img.Data)
}

func TestProcessExportImage_LockRatio_HeightOnly(t *testing.T) {
	t.Parallel()
	svc := newTestService(t, &mockSDService{})
	b64 := encodePNG(t, 200, 100)

	img, err := svc.ProcessExportImage(ExportImageParams{
		ImageBase64: b64,
		Format:      "png",
		Height:      50,
		LockRatio:   true,
	})
	require.NoError(t, err)
	assert.NotEmpty(t, img.Data)
}

func TestProcessExportImage_LockRatio_BothDimensions(t *testing.T) {
	t.Parallel()
	svc := newTestService(t, &mockSDService{})
	b64 := encodePNG(t, 512, 512)

	img, err := svc.ProcessExportImage(ExportImageParams{
		ImageBase64: b64,
		Format:      "png",
		Width:       256,
		Height:      128,
		LockRatio:   true,
	})
	require.NoError(t, err)
	assert.NotEmpty(t, img.Data)
}

func TestProcessExportImage_NoDimensions(t *testing.T) {
	t.Parallel()
	svc := newTestService(t, &mockSDService{})
	b64 := encodePNG(t, 64, 64)

	img, err := svc.ProcessExportImage(ExportImageParams{
		ImageBase64: b64,
		Format:      "png",
	})
	require.NoError(t, err)
	assert.NotEmpty(t, img.Data)
}

func TestProcessExportImage_Interpolation_Table(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name          string
		interpolation string
	}{
		{"nearest", "nearest"},
		{"linear", "linear"},
		{"lanczos", "lanczos"},
		{"empty_default", ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			svc := newTestService(t, &mockSDService{})
			b64 := encodePNG(t, 64, 64)
			img, err := svc.ProcessExportImage(ExportImageParams{
				ImageBase64:   b64,
				Format:        "png",
				Width:         32,
				Height:        32,
				Interpolation: tt.interpolation,
			})
			require.NoError(t, err)
			assert.NotEmpty(t, img.Data)
		})
	}
}

func TestProcessExportImage_WebpNotSupported(t *testing.T) {
	t.Parallel()
	svc := newTestService(t, &mockSDService{})
	b64 := encodePNG(t, 64, 64)

	_, err := svc.ProcessExportImage(ExportImageParams{
		ImageBase64: b64,
		Format:      "webp",
	})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "unsupported format")
}

func TestProcessExportImage_DefaultQuality(t *testing.T) {
	t.Parallel()
	svc := newTestService(t, &mockSDService{})
	b64 := encodePNG(t, 64, 64)

	img, err := svc.ProcessExportImage(ExportImageParams{
		ImageBase64: b64,
		Format:      "jpeg",
		Quality:     0,
	})
	require.NoError(t, err)
	assert.NotEmpty(t, img.Data)
}

func TestProcessExportImage_FilenameHasNoExtension(t *testing.T) {
	t.Parallel()
	svc := newTestService(t, &mockSDService{})
	b64 := encodePNG(t, 64, 64)

	img, err := svc.ProcessExportImage(ExportImageParams{
		ImageBase64: b64,
		Format:      "jpeg",
		Filename:    "myphoto",
	})
	require.NoError(t, err)
	assert.Equal(t, "myphoto.jpeg", img.Filename)
}

func TestProcessExportImage_FilenameAlreadyHasExtension(t *testing.T) {
	t.Parallel()
	svc := newTestService(t, &mockSDService{})
	b64 := encodePNG(t, 64, 64)

	img, err := svc.ProcessExportImage(ExportImageParams{
		ImageBase64: b64,
		Format:      "jpeg",
		Filename:    "myphoto.jpeg",
	})
	require.NoError(t, err)
	assert.Equal(t, "myphoto.jpeg", img.Filename)
}

func TestListExportPresets_DefaultSeeded(t *testing.T) {
	t.Parallel()
	svc := newTestService(t, &mockSDService{})
	presets, err := svc.ListExportPresets()
	require.NoError(t, err)
	assert.Len(t, presets, 4)
	assert.Equal(t, "Quality Photo", presets[0].Name)
}

func TestSaveExportPreset_New(t *testing.T) {
	t.Parallel()
	svc := newTestService(t, &mockSDService{})
	ep := preset.ExportPreset{
		Name:          "Custom",
		Format:        "png",
		Width:         1024,
		Height:        1024,
		Quality:       95,
		Interpolation: "lanczos",
	}
	result, err := svc.SaveExportPreset(ep)
	require.NoError(t, err)
	assert.True(t, result.ID > 0)
	assert.Equal(t, "Custom", result.Name)
}

func TestSaveExportPreset_Update(t *testing.T) {
	t.Parallel()
	svc := newTestService(t, &mockSDService{})

	ep := preset.ExportPreset{
		Name:   "Original",
		Format: "png",
	}
	created, err := svc.SaveExportPreset(ep)
	require.NoError(t, err)

	created.Name = "Updated"
	updated, err := svc.SaveExportPreset(*created)
	require.NoError(t, err)
	assert.Equal(t, "Updated", updated.Name)
	assert.Equal(t, created.ID, updated.ID)
}

func TestDeleteExportPreset_Valid(t *testing.T) {
	t.Parallel()
	svc := newTestService(t, &mockSDService{})

	ep := preset.ExportPreset{Name: "ToDelete", Format: "png"}
	created, err := svc.SaveExportPreset(ep)
	require.NoError(t, err)

	err = svc.DeleteExportPreset(created.ID)
	assert.NoError(t, err)

	presets, _ := svc.ListExportPresets()
	for _, p := range presets {
		assert.NotEqual(t, created.ID, p.ID)
	}
}

func TestDeleteExportPreset_Nonexistent(t *testing.T) {
	t.Parallel()
	svc := newTestService(t, &mockSDService{})
	err := svc.DeleteExportPreset(99999)
	assert.NoError(t, err)
}

func TestWriteImageToPath_Valid(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	path := filepath.Join(dir, "sub", "dir", "test.png")
	img := &ProcessedImage{
		Data:     []byte("fake-image-data"),
		Filename: "test.png",
		Format:   "png",
	}
	err := WriteImageToPath(img, path)
	require.NoError(t, err)

	data, err := os.ReadFile(path)
	require.NoError(t, err)
	assert.Equal(t, []byte("fake-image-data"), data)
}

func TestWriteImageToPath_OverwriteExisting(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	path := filepath.Join(dir, "test.png")

	require.NoError(t, os.WriteFile(path, []byte("old"), 0o644))

	img := &ProcessedImage{Data: []byte("new")}
	err := WriteImageToPath(img, path)
	require.NoError(t, err)

	data, _ := os.ReadFile(path)
	assert.Equal(t, []byte("new"), data)
}

func TestRoundTrip_ExportParseImport(t *testing.T) {
	t.Parallel()
	db := openTestDB(t)
	svc := New(db, &mockSDService{}, logger.New(nil))

	p := &preset.Preset{
		Name:       "roundtrip",
		PresetType: "test",
		Prompt:     "rt prompt",
		Sampler:    "Euler a",
		Steps:      25,
		CfgScale:   8.0,
		ModelName:  "model.safetensors",
	}
	require.NoError(t, db.Create(p))

	presets, err := svc.PrepareExportData([]int64{p.ID})
	require.NoError(t, err)
	assert.Len(t, presets, 1)

	fileData, err := svc.BuildExportFile(presets)
	require.NoError(t, err)

	path := filepath.Join(t.TempDir(), "export.json")
	require.NoError(t, os.WriteFile(path, fileData, 0o644))

	parsed, err := svc.ParseImportFile(path)
	require.NoError(t, err)
	assert.Len(t, parsed, 1)
	assert.Equal(t, "roundtrip", parsed[0].Name)
	assert.Equal(t, "export.json", parsed[0].SourceFile)

	imported, err := svc.ImportItems(parsed)
	require.NoError(t, err)
	assert.Len(t, imported, 1)
	assert.Equal(t, "roundtrip", imported[0].Name)
	assert.Equal(t, 25, imported[0].Steps)
	assert.Equal(t, 8.0, imported[0].CfgScale)
	assert.True(t, imported[0].ID > 0)
	assert.NotEqual(t, p.ID, imported[0].ID)
}
