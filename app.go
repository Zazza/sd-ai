package main

import (
	"context"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"

	"github.com/wailsapp/wails/v2/pkg/runtime"

	"go-sd/internal/config"
	"go-sd/internal/kids"
	"go-sd/internal/llm"
	"go-sd/internal/preset"
	"go-sd/internal/sd"
)

type App struct {
	ctx     context.Context
	presets *preset.DB
	llm     *llm.Client
	sd      *sd.Client
	config  *config.Config
}

func NewApp(presets *preset.DB, llmClient *llm.Client, sdClient *sd.Client, cfg *config.Config) *App {
	return &App{presets: presets, llm: llmClient, sd: sdClient, config: cfg}
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

type ServiceStatus struct {
	LLM struct {
		Available bool   `json:"available"`
		Model     string `json:"model"`
	} `json:"llm"`
	SD struct {
		Available bool   `json:"available"`
		Model     string `json:"model"`
	} `json:"sd"`
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

	result, err := a.llm.GenerateSDPrompt(systemPrompt, description, presetType, a.config.SDPromptModel)
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
	Image      any `json:"image"`
	Parameters any `json:"parameters"`
	Info       any `json:"info"`
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

	result, err := a.sd.Txt2Img(sd.Txt2ImgRequest{
		Prompt:         prompt,
		NegativePrompt: negativePrompt,
		SamplerName:    p.Sampler,
		Steps:          p.Steps,
		CfgScale:       p.CfgScale,
		Width:          p.Width,
		Height:         p.Height,
		Seed:           p.Seed,
	})
	if err != nil {
		return nil, err
	}

	if len(result.Images) == 0 {
		return nil, nil
	}

	return &GenerateImageResult{
		Image:      result.Images[0],
		Parameters: result.Parameters,
		Info:       result.Info,
	}, nil
}

// --- SD Info ---

func (a *App) GetSDModels() ([]sd.SDModel, error) {
	return a.sd.GetModels()
}

func (a *App) GetSDSamplers() ([]sd.Sampler, error) {
	return a.sd.GetSamplers()
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
		"kids_mode":       "false",
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
		"llm_backend": true, "llm_keep_alive": true, "llm_num_ctx": true, "llm_num_gpu": true,
		"kids_mode": true, "kids_pin_hash": true,
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
