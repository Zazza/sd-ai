package main

import (
	"context"
	"strings"

	"go-sd/internal/config"
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
	return a.llm.GenerateSDPrompt(a.config.SystemPrompt, description, presetType, a.config.SDPromptModel)
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
