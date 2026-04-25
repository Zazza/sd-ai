package main

import (
	"context"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/wailsapp/wails/v2/pkg/runtime"

	"go-sd/internal/config"
	"go-sd/internal/kids"
	"go-sd/internal/llm"
	"go-sd/internal/preset"
	"go-sd/internal/sd"
)

type App struct {
	ctx      context.Context
	presets  *preset.DB
	llm      *llm.Client
	sd       *sd.Client
	config   *config.Config
	dataDir  string
}

func NewApp(presets *preset.DB, llmClient *llm.Client, sdClient *sd.Client, cfg *config.Config) *App {
	return &App{
		presets: presets,
		llm:     llmClient,
		sd:      sdClient,
		config:  cfg,
		dataDir: filepath.Dir(cfg.DBPath),
	}
}

func (a *App) startup(ctx context.Context) {
	a.ctx = ctx
}

func (a *App) isKidsMode() bool {
	v, _ := a.presets.GetSetting("kids_mode")
	return v == "true"
}

func (a *App) IsKidsModeActive() bool {
	return a.isKidsMode()
}

func (a *App) SetKidsMode(enabled bool, pin string) error {
	if enabled {
		if pin != "" {
			if len(pin) != 4 {
				return fmt.Errorf("PIN must be 4 digits")
			}
			hash := sha256.Sum256([]byte(pin))
			if err := a.presets.SetSetting("kids_pin_hash", hex.EncodeToString(hash[:])); err != nil {
				return err
			}
		}
		return a.presets.SetSetting("kids_mode", "true")
	}

	storedHash, _ := a.presets.GetSetting("kids_pin_hash")
	if storedHash != "" {
		if pin == "" {
			return fmt.Errorf("PIN required")
		}
		hash := sha256.Sum256([]byte(pin))
		if hex.EncodeToString(hash[:]) != storedHash {
			return fmt.Errorf("incorrect PIN")
		}
	}
	return a.presets.SetSetting("kids_mode", "false")
}

// --- Service Status ---

type ServiceInfo struct {
	Available bool   `json:"available"`
	Model     string `json:"model"`
}

type ServiceStatus struct {
	LLM ServiceInfo `json:"llm"`
	SD  ServiceInfo `json:"sd"`
}

func (a *App) CheckServices() ServiceStatus {
	var status ServiceStatus
	var wg sync.WaitGroup
	wg.Add(2)

	go func() {
		defer wg.Done()
		if err := a.llm.HealthCheck(); err != nil {
			status.LLM.Available = false
			return
		}
		status.LLM.Available = true
		status.LLM.Model = a.config.SDPromptModel
	}()

	go func() {
		defer wg.Done()
		if err := a.sd.HealthCheck(); err != nil {
			status.SD.Available = false
			return
		}
		status.SD.Available = true
		opts, err := a.sd.GetOptions()
		if err == nil {
			if m, ok := opts["sd_model_checkpoint"].(string); ok {
				status.SD.Model = m
			}
		}
	}()

	wg.Wait()
	return status
}

// --- Presets ---

func (a *App) ListPresets() ([]preset.Preset, error) {
	presets, err := a.presets.List()
	if err != nil {
		return nil, err
	}
	if presets == nil {
		presets = []preset.Preset{}
	}
	return presets, nil
}

func (a *App) ListPresetsByType(presetType string) ([]preset.Preset, error) {
	presets, err := a.presets.ListByType(presetType)
	if err != nil {
		return nil, err
	}
	if presets == nil {
		presets = []preset.Preset{}
	}
	return presets, nil
}

func (a *App) GetPreset(id int64) (*preset.Preset, error) {
	return a.presets.Get(id)
}

func (a *App) CreatePreset(p preset.Preset) (*preset.Preset, error) {
	if err := a.presets.Create(&p); err != nil {
		return nil, err
	}
	return &p, nil
}

func (a *App) UpdatePreset(p preset.Preset) (*preset.Preset, error) {
	if err := a.presets.Update(&p); err != nil {
		return nil, err
	}
	return &p, nil
}

func (a *App) DeletePreset(id int64) error {
	return a.presets.Delete(id)
}

// --- Generation ---

func (a *App) GenerateSDPrompt(description, presetType string) (string, error) {
	description = strings.TrimSpace(description)
	if description == "" {
		return "", nil
	}

	systemPrompt := a.config.SystemPrompt

	if a.isKidsMode() {
		filtered := kids.FilterInput(description)
		if filtered == "" {
			return "", fmt.Errorf("description contains restricted content")
		}
		description = filtered
		systemPrompt += config.KidsModePrompt
	}

	maxTokens := 1024
	if v, err := a.presets.GetSetting("llm_max_tokens"); err == nil && v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 {
			maxTokens = n
		}
	}

	result, err := a.llm.GenerateSDPrompt(systemPrompt, description, presetType, a.config.SDPromptModel, maxTokens)
	if err != nil {
		return "", err
	}

	if a.isKidsMode() {
		result = kids.FilterOutput(result)
	}

	return result, nil
}

type GenerateImageParams struct {
	PresetID            int64  `json:"preset_id"`
	ExtraPrompt         string `json:"extra_prompt"`
	ExtraNegativePrompt string `json:"extra_negative_prompt"`
}

type GenerateImageResult struct {
	Image      any  `json:"image"`
	Parameters any  `json:"parameters"`
	Info       any  `json:"info"`
	IsPreview  bool `json:"is_preview"`
}

func (a *App) GenerateImage(params GenerateImageParams) (*GenerateImageResult, error) {
	p, err := a.presets.Get(params.PresetID)
	if err != nil {
		return nil, err
	}

	prompt := p.Prompt
	if params.ExtraPrompt != "" {
		prompt += ", " + params.ExtraPrompt
	}

	negativePrompt := p.NegativePrompt
	if params.ExtraNegativePrompt != "" {
		negativePrompt += ", " + params.ExtraNegativePrompt
	}

	if a.isKidsMode() {
		negativePrompt += ", " + config.KidsModeNegativePrompt
	}

	if p.ModelName != "" {
		_ = a.sd.SetModel(p.ModelName)
	}

	if p.VAE != "" {
		_ = a.sd.SetVAE(p.VAE)
	}

	samplerName := p.Sampler
	if p.ScheduleType != "" {
		st := strings.ToUpper(p.ScheduleType[:1]) + p.ScheduleType[1:]
		samplerName = p.Sampler + " " + st
	}

	batchSize := 1
	if p.BatchSize != nil {
		batchSize = *p.BatchSize
	}
	batchCount := 1
	if p.BatchCount != nil {
		batchCount = *p.BatchCount
	}
	clipSkip := 1
	if p.ClipSkip != nil {
		clipSkip = *p.ClipSkip
	}

	denoisingStrength := p.DenoisingStrength
	if denoisingStrength == nil && p.HiresFix != nil && *p.HiresFix {
		ds := 0.5
		if p.HiresDenoisingStrength != nil {
			ds = *p.HiresDenoisingStrength
		}
		denoisingStrength = &ds
	}

	isPreview := false
	width := p.Width
	height := p.Height
	var hiresFix *bool
	if p.HiresFix != nil {
		hiresFix = p.HiresFix
	}

	if v, _ := a.presets.GetSetting("preview_mode"); v == "true" {
		isPreview = true
		maxW, maxH := 512, 512
		if pw, _ := a.presets.GetSetting("preview_width"); pw != "" {
			if n, err := strconv.Atoi(pw); err == nil && n > 0 {
				maxW = n
			}
		}
		if ph, _ := a.presets.GetSetting("preview_height"); ph != "" {
			if n, err := strconv.Atoi(ph); err == nil && n > 0 {
				maxH = n
			}
		}
		targetRatio := float64(p.Width) / float64(p.Height)
		maxRatio := float64(maxW) / float64(maxH)
		if maxRatio > targetRatio {
			height = maxH
			width = int(float64(maxH) * targetRatio)
		} else {
			width = maxW
			height = int(float64(maxW) / targetRatio)
		}
		width = (width / 8) * 8
		height = (height / 8) * 8
		if width < 64 {
			width = 64
		}
		if height < 64 {
			height = 64
		}
		hiresFix = nil
	}

	result, err := a.sd.Txt2Img(sd.Txt2ImgRequest{
		Prompt:                 prompt,
		NegativePrompt:         negativePrompt,
		SamplerName:            samplerName,
		Scheduler:              p.ScheduleType,
		Steps:                  p.Steps,
		CfgScale:               p.CfgScale,
		Width:                  width,
		Height:                 height,
		Seed:                   p.Seed,
		DenoisingStrength:      denoisingStrength,
		ClipSkip:               &clipSkip,
		BatchSize:              &batchSize,
		BatchCount:             &batchCount,
		HiresFix:               hiresFix,
		HiresUpscale:           p.HiresUpscale,
		HiresDenoisingStrength: p.HiresDenoisingStrength,
		HiresUpscaler:          p.HiresUpscaler,
		DoNotSaveImages:        true,
		DoNotSaveGrid:          true,
	})
	if err != nil {
		return nil, err
	}

	if len(result.Images) == 0 {
		reason := "empty response"
		if result.Error != "" {
			reason = result.Error
		} else if len(result.Info) > 0 {
			var info struct {
				Reason string `json:"reason"`
			}
			if json.Unmarshal(result.Info, &info) == nil && info.Reason != "" {
				reason = info.Reason
			}
		}
		return nil, fmt.Errorf("no image generated: %s (sampler=%s, scheduler=%s, model=%s)",
			reason, p.Sampler, p.ScheduleType, p.ModelName)
	}

	img := &GenerateImageResult{
		Image:      result.Images[0],
		Parameters: result.Parameters,
		Info:       result.Info,
		IsPreview:  isPreview,
	}
	a.saveLastImage(result.Images[0], result.Info, isPreview)
	return img, nil
}

type UpscalePreviewParams struct {
	PreviewImageBase64 string   `json:"preview_image_base64"`
	PresetID           int64    `json:"preset_id"`
	Seed               int64    `json:"seed"`
	DenoisingStrength  *float64 `json:"denoising_strength,omitempty"`
}

func (a *App) UpscalePreview(params UpscalePreviewParams) (*GenerateImageResult, error) {
	p, err := a.presets.Get(params.PresetID)
	if err != nil {
		return nil, err
	}

	prompt := p.Prompt
	negativePrompt := p.NegativePrompt

	if a.isKidsMode() {
		negativePrompt += ", " + config.KidsModeNegativePrompt
	}

	if p.ModelName != "" {
		_ = a.sd.SetModel(p.ModelName)
	}

	if p.VAE != "" {
		_ = a.sd.SetVAE(p.VAE)
	}

	samplerName := p.Sampler
	if p.ScheduleType != "" {
		st := strings.ToUpper(p.ScheduleType[:1]) + p.ScheduleType[1:]
		samplerName = p.Sampler + " " + st
	}

	batchSize := 1
	if p.BatchSize != nil {
		batchSize = *p.BatchSize
	}
	batchCount := 1
	if p.BatchCount != nil {
		batchCount = *p.BatchCount
	}
	clipSkip := 1
	if p.ClipSkip != nil {
		clipSkip = *p.ClipSkip
	}

	denoisingStrength := 0.55
	if params.DenoisingStrength != nil && *params.DenoisingStrength > 0 {
		denoisingStrength = *params.DenoisingStrength
	}

	result, err := a.sd.Img2Img(sd.Img2ImgRequest{
		InitImages:        []string{params.PreviewImageBase64},
		Prompt:            prompt,
		NegativePrompt:    negativePrompt,
		SamplerName:       samplerName,
		Scheduler:         p.ScheduleType,
		Steps:             p.Steps,
		CfgScale:          p.CfgScale,
		Width:             p.Width,
		Height:            p.Height,
		Seed:              &params.Seed,
		DenoisingStrength: &denoisingStrength,
		ClipSkip:          &clipSkip,
		BatchSize:         &batchSize,
		BatchCount:        &batchCount,
		DoNotSaveImages:   true,
		DoNotSaveGrid:     true,
	})
	if err != nil {
		return nil, err
	}

	if len(result.Images) == 0 {
		return nil, fmt.Errorf("no image generated during upscale")
	}

	img := &GenerateImageResult{
		Image:      result.Images[0],
		Parameters: result.Parameters,
		Info:       result.Info,
		IsPreview:  false,
	}
	a.saveLastImage(result.Images[0], result.Info, false)
	return img, nil
}

// --- SD Info ---

func (a *App) GetSDModels() ([]sd.SDModel, error) {
	return a.sd.GetModels()
}

func (a *App) GetSDSamplers() ([]sd.Sampler, error) {
	return a.sd.GetSamplers()
}

func (a *App) GetSDSchedulers() ([]sd.Scheduler, error) {
	return a.sd.GetSchedulers()
}

func (a *App) GetSDUpscalers() ([]sd.Upscaler, error) {
	return a.sd.GetUpscalers()
}

func (a *App) GetSDVAEs() ([]sd.VAE, error) {
	return a.sd.GetVAEs()
}

// --- LLM Info ---

func (a *App) GetLLMModels() ([]llm.LLMModel, error) {
	return a.llm.GetModels()
}

// --- Settings ---

func (a *App) GetSettings() (map[string]string, error) {
	settings, err := a.presets.GetAllSettings()
	if err != nil {
		return nil, err
	}

	defaults := map[string]string{
		"llm_url":         a.config.LLMUrl,
		"sd_url":          a.config.SDUrl,
		"llm_model":       a.config.LLMModel,
		"sd_prompt_model": a.config.SDPromptModel,
		"llm_backend":     a.config.LLMBackend,
		"llm_keep_alive":  "5m",
		"llm_num_ctx":     "4096",
		"llm_num_gpu":     "0",
		"llm_max_tokens":  "1024",
		"kids_mode":       "false",
		"preview_mode":    "false",
		"preview_width":   "512",
		"preview_height":  "512",
	}
	for k, v := range defaults {
		if _, ok := settings[k]; !ok {
			settings[k] = v
		}
	}
	return settings, nil
}

func (a *App) UpdateSettings(data map[string]string) error {
	allowed := map[string]bool{
		"llm_url": true, "sd_url": true, "llm_model": true, "sd_prompt_model": true,
		"llm_backend": true, "llm_keep_alive": true, "llm_num_ctx": true, "llm_num_gpu": true, "llm_max_tokens": true,
		"kids_mode": true, "kids_pin_hash": true,
		"preview_mode": true, "preview_width": true, "preview_height": true,
		"gen_preset_id": true, "gen_description": true, "gen_extra_prompt": true, "gen_extra_negative": true,
	}

	for k, v := range data {
		if !allowed[k] {
			continue
		}
		if err := a.presets.SetSetting(k, v); err != nil {
			return err
		}
	}

	if v, ok := data["llm_url"]; ok {
		a.llm.SetURL(v)
		a.config.LLMUrl = v
	}
	if v, ok := data["sd_url"]; ok {
		a.sd.SetURL(v)
		a.config.SDUrl = v
	}
	if v, ok := data["llm_model"]; ok {
		a.config.LLMModel = v
	}
	if v, ok := data["sd_prompt_model"]; ok {
		a.config.SDPromptModel = v
	}
	if v, ok := data["llm_backend"]; ok {
		a.llm.SetBackend(v)
		a.config.LLMBackend = v
	}

	var cfg llm.BackendConfig
	if v, ok := data["llm_keep_alive"]; ok {
		cfg.KeepAlive = v
	}
	if v, ok := data["llm_num_ctx"]; ok {
		if n, err := strconv.Atoi(v); err == nil {
			cfg.NumCtx = n
		}
	}
	if v, ok := data["llm_num_gpu"]; ok {
		if n, err := strconv.Atoi(v); err == nil {
			cfg.NumGPU = n
		}
	}
	a.llm.SetBackendConfig(cfg)

	return nil
}

// --- Saved Descriptions ---

func (a *App) ListDescriptions() ([]preset.SavedDescription, error) {
	items, err := a.presets.ListDescriptions()
	if err != nil {
		return nil, err
	}
	if items == nil {
		items = []preset.SavedDescription{}
	}
	return items, nil
}

func (a *App) CreateDescription(text string) (*preset.SavedDescription, error) {
	text = strings.TrimSpace(text)
	if text == "" {
		return nil, nil
	}
	return a.presets.CreateDescription(text)
}

func (a *App) DeleteDescription(id int64) error {
	return a.presets.DeleteDescription(id)
}

// --- Saved Prompts ---

func (a *App) ListPrompts() ([]preset.SavedPrompt, error) {
	items, err := a.presets.ListPrompts()
	if err != nil {
		return nil, err
	}
	if items == nil {
		items = []preset.SavedPrompt{}
	}
	return items, nil
}

func (a *App) CreatePrompt(text string) (*preset.SavedPrompt, error) {
	text = strings.TrimSpace(text)
	if text == "" {
		return nil, nil
	}
	return a.presets.CreatePrompt(text)
}

func (a *App) DeletePrompt(id int64) error {
	return a.presets.DeletePrompt(id)
}

// --- Last Image Persistence ---

type lastImageMeta struct {
	IsPreview bool            `json:"is_preview"`
	Info      json.RawMessage `json:"info"`
}

func (a *App) saveLastImage(imageBase64 string, info json.RawMessage, isPreview bool) {
	if imageBase64 == "" {
		return
	}

	pngData, err := base64.StdEncoding.DecodeString(imageBase64)
	if err != nil {
		return
	}

	if err := os.MkdirAll(a.dataDir, 0o755); err != nil {
		return
	}

	pngPath := filepath.Join(a.dataDir, "last_image.png")
	if err := os.WriteFile(pngPath, pngData, 0o644); err != nil {
		return
	}

	meta := lastImageMeta{IsPreview: isPreview, Info: info}
	metaBytes, err := json.Marshal(meta)
	if err != nil {
		return
	}

	metaPath := filepath.Join(a.dataDir, "last_image.json")
	_ = os.WriteFile(metaPath, metaBytes, 0o644)
}

func (a *App) GetLastImage() (*GenerateImageResult, error) {
	pngPath := filepath.Join(a.dataDir, "last_image.png")
	pngData, err := os.ReadFile(pngPath)
	if err != nil {
		return nil, nil
	}

	metaPath := filepath.Join(a.dataDir, "last_image.json")
	metaBytes, err := os.ReadFile(metaPath)

	var meta lastImageMeta
	if err == nil {
		_ = json.Unmarshal(metaBytes, &meta)
	}

	return &GenerateImageResult{
		Image:     base64.StdEncoding.EncodeToString(pngData),
		Parameters: nil,
		Info:      meta.Info,
		IsPreview: meta.IsPreview,
	}, nil
}

// --- Save Image ---

func (a *App) SaveImage(base64Data, defaultName string) (string, error) {
	if base64Data == "" {
		return "", nil
	}

	if defaultName == "" {
		defaultName = "sd-studio-image.png"
	}
	if !strings.HasSuffix(strings.ToLower(defaultName), ".png") {
		defaultName += ".png"
	}

	path, err := runtime.SaveFileDialog(a.ctx, runtime.SaveDialogOptions{
		DefaultFilename: defaultName,
		Filters: []runtime.FileFilter{
			{DisplayName: "PNG Image", Pattern: "*.png"},
		},
	})
	if err != nil || path == "" {
		return "", err
	}

	data, err := base64.StdEncoding.DecodeString(base64Data)
	if err != nil {
		return "", err
	}

	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return "", err
	}

	if err := os.WriteFile(path, data, 0o644); err != nil {
		return "", err
	}

	return path, nil
}

// --- Preset Export/Import ---

type PresetExportFile struct {
	Version    int          `json:"version"`
	ExportedAt time.Time    `json:"exported_at"`
	Presets    []PresetData `json:"presets"`
}

type PresetData struct {
	Name           string  `json:"name"`
	PresetType     string  `json:"preset_type"`
	Prompt         string  `json:"prompt"`
	NegativePrompt string  `json:"negative_prompt"`
	Sampler        string  `json:"sampler"`
	ScheduleType   string  `json:"schedule_type"`
	Steps          int     `json:"steps"`
	CfgScale       float64 `json:"cfg_scale"`
	Width          int     `json:"width"`
	Height         int     `json:"height"`
	ModelName      string  `json:"model_name"`
	Seed                   *int64   `json:"seed"`
	DenoisingStrength      *float64 `json:"denoising_strength"`
	ClipSkip               *int     `json:"clip_skip"`
	BatchSize              *int     `json:"batch_size"`
	BatchCount             *int     `json:"batch_count"`
	HiresFix               *bool    `json:"hires_fix"`
	HiresUpscale           *float64 `json:"hires_upscale"`
	HiresDenoisingStrength *float64 `json:"hires_denoising_strength"`
	HiresUpscaler          string   `json:"hires_upscaler"`
	VAE                    string   `json:"vae"`
}

type ImportPreview struct {
	Presets []PresetData `json:"presets"`
	Total   int          `json:"total"`
}

func (a *App) ExportPresets(ids []int64) (string, error) {
	if len(ids) == 0 {
		return "", fmt.Errorf("no presets selected")
	}

	selected, err := a.presets.GetByIDs(ids)
	if err != nil {
		return "", err
	}

	data := PresetExportFile{
		Version:    1,
		ExportedAt: time.Now().UTC(),
		Presets:    make([]PresetData, len(selected)),
	}
	for i, p := range selected {
		data.Presets[i] = PresetData{
			Name:                   p.Name,
			PresetType:             p.PresetType,
			Prompt:                 p.Prompt,
			NegativePrompt:         p.NegativePrompt,
			Sampler:                p.Sampler,
			ScheduleType:           p.ScheduleType,
			Steps:                  p.Steps,
			CfgScale:               p.CfgScale,
			Width:                  p.Width,
			Height:                 p.Height,
			ModelName:              p.ModelName,
			Seed:                   p.Seed,
			DenoisingStrength:      p.DenoisingStrength,
			ClipSkip:               p.ClipSkip,
			BatchSize:              p.BatchSize,
			BatchCount:             p.BatchCount,
			HiresFix:               p.HiresFix,
			HiresUpscale:           p.HiresUpscale,
			HiresDenoisingStrength: p.HiresDenoisingStrength,
			HiresUpscaler:          p.HiresUpscaler,
			VAE:                    p.VAE,
		}
	}

	jsonBytes, err := json.MarshalIndent(data, "", "  ")
	if err != nil {
		return "", err
	}

	path, err := runtime.SaveFileDialog(a.ctx, runtime.SaveDialogOptions{
		DefaultFilename: "sd-studio-presets.json",
		Filters: []runtime.FileFilter{
			{DisplayName: "JSON Files", Pattern: "*.json"},
		},
	})
	if err != nil || path == "" {
		return "", err
	}

	if err := os.WriteFile(path, jsonBytes, 0o644); err != nil {
		return "", err
	}

	return path, nil
}

func (a *App) OpenImportFile() (*ImportPreview, error) {
	path, err := runtime.OpenFileDialog(a.ctx, runtime.OpenDialogOptions{
		Filters: []runtime.FileFilter{
			{DisplayName: "JSON Files", Pattern: "*.json"},
		},
	})
	if err != nil || path == "" {
		return nil, err
	}

	info, err := os.Stat(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read file")
	}
	if info.Size() > 10*1024*1024 {
		return nil, fmt.Errorf("file too large (max 10 MB)")
	}

	jsonBytes, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read file")
	}

	var data PresetExportFile
	if err := json.Unmarshal(jsonBytes, &data); err != nil {
		return nil, fmt.Errorf("invalid file format: %w", err)
	}

	if data.Version != 1 {
		return nil, fmt.Errorf("unsupported file version: %d", data.Version)
	}

	if len(data.Presets) == 0 {
		return nil, fmt.Errorf("no presets found in file")
	}

	return &ImportPreview{
		Presets: data.Presets,
		Total:   len(data.Presets),
	}, nil
}

func splitCompositeSampler(sampler, scheduleType string) (string, string) {
	if scheduleType != "" {
		return sampler, scheduleType
	}
	knownSchedulers := []string{"Karras", "Exponential", "Polyexponential"}
	for _, s := range knownSchedulers {
		if strings.HasSuffix(sampler, " "+s) {
			return sampler[:len(sampler)-len(s)-1], s
		}
	}
	return sampler, ""
}

func (a *App) ImportPresets(items []PresetData) ([]preset.Preset, error) {
	if len(items) == 0 {
		return nil, fmt.Errorf("no presets selected")
	}
	if len(items) > 500 {
		return nil, fmt.Errorf("too many presets (max 500)")
	}

	for _, item := range items {
		if strings.TrimSpace(item.Name) == "" {
			return nil, fmt.Errorf("preset name is required")
		}
		if item.Steps < 1 || item.Steps > 150 {
			return nil, fmt.Errorf("invalid steps for %q: must be 1-150", item.Name)
		}
		if item.Width < 64 || item.Width > 2048 || item.Height < 64 || item.Height > 2048 {
			return nil, fmt.Errorf("invalid dimensions for %q: must be 64-2048", item.Name)
		}
		if item.CfgScale < 0 || item.CfgScale > 30 {
			return nil, fmt.Errorf("invalid cfg_scale for %q: must be 0-30", item.Name)
		}
		if item.DenoisingStrength != nil && (*item.DenoisingStrength < 0 || *item.DenoisingStrength > 1) {
			return nil, fmt.Errorf("invalid denoising_strength for %q: must be 0-1", item.Name)
		}
		if item.ClipSkip != nil && (*item.ClipSkip < 1 || *item.ClipSkip > 12) {
			return nil, fmt.Errorf("invalid clip_skip for %q: must be 1-12", item.Name)
		}
		if item.BatchSize != nil && (*item.BatchSize < 1 || *item.BatchSize > 8) {
			return nil, fmt.Errorf("invalid batch_size for %q: must be 1-8", item.Name)
		}
		if item.BatchCount != nil && (*item.BatchCount < 1 || *item.BatchCount > 8) {
			return nil, fmt.Errorf("invalid batch_count for %q: must be 1-8", item.Name)
		}
		if item.HiresUpscale != nil && (*item.HiresUpscale < 1 || *item.HiresUpscale > 4) {
			return nil, fmt.Errorf("invalid hires_upscale for %q: must be 1-4", item.Name)
		}
		if item.HiresDenoisingStrength != nil && (*item.HiresDenoisingStrength < 0 || *item.HiresDenoisingStrength > 1) {
			return nil, fmt.Errorf("invalid hires_denoising_strength for %q: must be 0-1", item.Name)
		}
	}

	batch := make([]preset.Preset, len(items))
	for i, item := range items {
		sampler, scheduleType := splitCompositeSampler(item.Sampler, item.ScheduleType)
		batch[i] = preset.Preset{
			Name:                   item.Name,
			PresetType:             item.PresetType,
			Prompt:                 item.Prompt,
			NegativePrompt:         item.NegativePrompt,
			Sampler:                sampler,
			ScheduleType:           scheduleType,
			Steps:                  item.Steps,
			CfgScale:               item.CfgScale,
			Width:                  item.Width,
			Height:                 item.Height,
			ModelName:              item.ModelName,
			Seed:                   item.Seed,
			DenoisingStrength:      item.DenoisingStrength,
			ClipSkip:               item.ClipSkip,
			BatchSize:              item.BatchSize,
			BatchCount:             item.BatchCount,
			HiresFix:               item.HiresFix,
			HiresUpscale:           item.HiresUpscale,
			HiresDenoisingStrength: item.HiresDenoisingStrength,
			HiresUpscaler:          item.HiresUpscaler,
			VAE:                    item.VAE,
		}
	}

	created, err := a.presets.CreateBatch(batch)
	if err != nil {
		return nil, err
	}
	return created, nil
}
