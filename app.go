package main

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"image"
	"image/color"
	"image/draw"
	_ "image/jpeg"
	"image/png"
	"math"
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

func (a *App) ListPresetTypes() ([]preset.PresetType, error) {
	items, err := a.presets.ListPresetTypes()
	if err != nil {
		return nil, err
	}
	if items == nil {
		items = []preset.PresetType{}
	}
	return items, nil
}

func (a *App) GetPresetType(id int64) (*preset.PresetType, error) {
	return a.presets.GetPresetType(id)
}

func (a *App) CreatePresetType(pt preset.PresetType) (*preset.PresetType, error) {
	if err := a.presets.CreatePresetType(&pt); err != nil {
		return nil, err
	}
	return &pt, nil
}

func (a *App) UpdatePresetType(pt preset.PresetType) (*preset.PresetType, error) {
	if err := a.presets.UpdatePresetType(&pt); err != nil {
		return nil, err
	}
	return &pt, nil
}

func (a *App) DeletePresetType(id int64) error {
	return a.presets.DeletePresetType(id)
}

func (a *App) GetAllTags() ([]string, error) {
	tags, err := a.presets.GetAllTags()
	if err != nil {
		return nil, err
	}
	if tags == nil {
		tags = []string{}
	}
	return tags, nil
}

func (a *App) GetSDLoRAs() ([]sd.LoRA, error) {
	return a.sd.GetLoRAs()
}

// --- Generation ---

type GenerateSDPromptParams struct {
	PresetID    int64  `json:"preset_id"`
	Description string `json:"description"`
	Negative    string `json:"negative"`
}

type GenerateSDPromptResult struct {
	Prompt         string `json:"prompt"`
	NegativePrompt string `json:"negative_prompt"`
}

func (a *App) GenerateSDPrompt(params GenerateSDPromptParams) (*GenerateSDPromptResult, error) {
	if params.PresetID <= 0 {
		return nil, fmt.Errorf("preset is required")
	}

	p, err := a.presets.Get(params.PresetID)
	if err != nil {
		return nil, fmt.Errorf("preset not found: %w", err)
	}

	description := strings.TrimSpace(params.Description)
	negative := strings.TrimSpace(params.Negative)

	if description == "" && negative == "" {
		return nil, nil
	}

	sdPromptInstruction := config.DefaultSDPromptInstruction
	if v, err := a.presets.GetSetting("sd_prompt_instruction"); err == nil && v != "" {
		sdPromptInstruction = v
	}

	systemPrompt := sdPromptInstruction

	if a.isKidsMode() {
		description = kids.FilterInput(description)
		negative = kids.FilterInput(negative)
		systemPrompt += config.KidsModePrompt
	}

	maxTokens := 256
	if v, err := a.presets.GetSetting("llm_max_tokens"); err == nil && v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 {
			maxTokens = n
		}
	}

	generateModel := a.config.SDPromptModel
	if v, err := a.presets.GetSetting("llm_generate_model"); err == nil && v != "" {
		generateModel = v
	}

	a.applyLLMConfig("generate")

	systemPrompt += fmt.Sprintf(`

RESPONSE LENGTH: your response is limited to ~%d tokens. You MUST fit within this limit.`, maxTokens)

	var userParts []string
	userParts = append(userParts, "BASE POSITIVE PROMPT: "+p.Prompt)
	userParts = append(userParts, "BASE NEGATIVE PROMPT: "+p.NegativePrompt)
	if description != "" {
		userParts = append(userParts, "USER DESCRIPTION: "+description)
	}
	if negative != "" {
		userParts = append(userParts, "USER NEGATIVE: "+negative)
	}
	userMessage := strings.Join(userParts, "\n\n")

	raw, err := a.llm.GenerateSDPrompt(systemPrompt, userMessage, p.PresetType, generateModel, maxTokens)
	if err != nil {
		return nil, err
	}

	var result GenerateSDPromptResult
	if err := json.Unmarshal([]byte(extractJSON(raw)), &result); err != nil {
		result = GenerateSDPromptResult{
			Prompt:         truncateRepetitive(raw, 1000),
			NegativePrompt: p.NegativePrompt,
		}
	}

	result.Prompt = truncateRepetitive(result.Prompt, 1000)
	result.NegativePrompt = truncateRepetitive(result.NegativePrompt, 500)

	if a.isKidsMode() {
		result.Prompt = kids.FilterOutput(result.Prompt)
		result.NegativePrompt = kids.FilterOutput(result.NegativePrompt)
	}

	return &result, nil
}

type RecommendPresetResult struct {
	PresetID    int64  `json:"preset_id"`
	PresetName  string `json:"preset_name"`
	ExtraPrompt string `json:"extra_prompt"`
	Reasoning   string `json:"reasoning"`
}

func (a *App) RecommendPreset(description string) (*RecommendPresetResult, error) {
	if strings.TrimSpace(description) == "" {
		return nil, fmt.Errorf("description is required")
	}

	allPresets, err := a.presets.List()
	if err != nil {
		return nil, fmt.Errorf("load presets: %w", err)
	}
	if len(allPresets) == 0 {
		return nil, fmt.Errorf("no presets available")
	}

	typesMap := make(map[int64]string)
	types, _ := a.presets.ListPresetTypes()
	if types != nil {
		for _, t := range types {
			typesMap[t.ID] = t.Name
		}
	}

	var presetList []string
	for _, p := range allPresets {
		typeName := ""
		if p.TypeID != nil {
			typeName = typesMap[*p.TypeID]
		}
		entry := fmt.Sprintf("ID:%d | Name:%q | Type:%q | Tags:%q", p.ID, p.Name, typeName, p.Tags)
		presetList = append(presetList, entry)
	}

	systemPrompt := `You are a Stable Diffusion preset recommender. Given a user's description of what they want to generate, you must select the BEST matching preset from the available list and suggest any additional prompt enhancements.

RULES:
1. Select EXACTLY ONE preset that best matches the user's description
2. Consider: subject matter, style, quality, and technical aspects
3. In extra_prompt, suggest additional SD tags that would improve the result based on the user's description
4. Keep extra_prompt as comma-separated SD tags only
5. Translate non-English to English

OUTPUT — valid JSON only, no markdown:
{"preset_id": 123, "preset_name": "exact name", "extra_prompt": "additional tags", "reasoning": "why this preset"}`

	userMessage := "AVAILABLE PRESETS:\n" + strings.Join(presetList, "\n") + "\n\nUSER DESCRIPTION: " + strings.TrimSpace(description)

	generateModel := a.config.SDPromptModel
	if v, err := a.presets.GetSetting("llm_generate_model"); err == nil && v != "" {
		generateModel = v
	}

	a.applyLLMConfig("generate")

	maxTokens := 512
	if v, err := a.presets.GetSetting("llm_max_tokens"); err == nil && v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 {
			maxTokens = n
		}
	}

	raw, err := a.llm.GenerateSDPrompt(systemPrompt, userMessage, "", generateModel, maxTokens)
	if err != nil {
		return nil, err
	}

	var result RecommendPresetResult
	if err := json.Unmarshal([]byte(extractJSON(raw)), &result); err != nil {
		return nil, fmt.Errorf("failed to parse LLM response: %w", err)
	}

	return &result, nil
}

func (a *App) AnalyzeImage(imageBase64 string) (string, error) {
	if imageBase64 == "" {
		return "", fmt.Errorf("image is required")
	}

	model := a.config.VisionModel
	if v, err := a.presets.GetSetting("llm_analyze_model"); err == nil && v != "" {
		model = v
	}
	if model == "" {
		model = a.config.SDPromptModel
		if v, err := a.presets.GetSetting("llm_generate_model"); err == nil && v != "" {
			model = v
		}
	}

	a.applyLLMConfig("analyze")

	systemPrompt := `You are an SD tag extractor. Describe the given image as comma-separated Stable Diffusion tags.
Output ONLY tags, nothing else. Start with quality tags (masterpiece, best quality, highly detailed).
Then describe: subject, pose, clothing, expression, lighting, background, style, camera angle.
Use (keyword:1.2) for emphasis on important elements.`

	maxTokens := 256
	if v, err := a.presets.GetSetting("llm_max_tokens"); err == nil && v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 {
			maxTokens = n
		}
	}

	result, err := a.llm.AnalyzeImage(model, systemPrompt, imageBase64, maxTokens)
	if err != nil {
		return "", err
	}

	return result, nil
}

func (a *App) ReadImageFile() (string, error) {
	path, err := runtime.OpenFileDialog(a.ctx, runtime.OpenDialogOptions{
		Filters: []runtime.FileFilter{
			{DisplayName: "Images", Pattern: "*.png;*.jpg;*.jpeg;*.webp"},
		},
	})
	if err != nil || path == "" {
		return "", err
	}

	data, err := os.ReadFile(path)
	if err != nil {
		return "", fmt.Errorf("failed to read file: %w", err)
	}

	if len(data) > 16*1024*1024 {
		return "", fmt.Errorf("image too large (max 16 MB)")
	}

	return base64.StdEncoding.EncodeToString(data), nil
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
	if p.Loras != "" {
		var loras []preset.LoRAEntry
		if err := json.Unmarshal([]byte(p.Loras), &loras); err == nil {
			for _, l := range loras {
				prompt += fmt.Sprintf(" <lora:%s:%g>", l.Name, l.Weight)
			}
		}
	}
	if params.ExtraPrompt != "" {
		prompt += " BREAK " + params.ExtraPrompt
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

type UpscaleImageParams struct {
	ImageBase64 string `json:"image_base64"`
	GenInfo     string `json:"gen_info"`
	PresetID    int64  `json:"preset_id"`
}

func (a *App) UpscaleImage(params UpscaleImageParams) (*GenerateImageResult, error) {
	if params.ImageBase64 == "" {
		return nil, fmt.Errorf("image is required")
	}
	if len(params.ImageBase64) > 67*1024*1024 {
		return nil, fmt.Errorf("image too large (max 50 MB)")
	}

	var info struct {
		Prompt         string  `json:"prompt"`
		NegativePrompt string  `json:"negative_prompt"`
		SamplerName    string  `json:"sampler_name"`
		Scheduler      string  `json:"scheduler"`
		Seed           int64   `json:"seed"`
		Width          int     `json:"width"`
		Height         int     `json:"height"`
		Steps          int     `json:"steps"`
		CfgScale       float64 `json:"cfg_scale"`
		ClipSkip       int     `json:"clip_skip"`
	}
	if err := json.Unmarshal([]byte(params.GenInfo), &info); err != nil {
		return nil, fmt.Errorf("parse gen_info: %w", err)
	}

	if info.Width <= 0 || info.Height <= 0 {
		return nil, fmt.Errorf("invalid dimensions in gen_info: %dx%d", info.Width, info.Height)
	}

	const maxDim = 2048
	if info.Width > maxDim || info.Height > maxDim {
		return nil, fmt.Errorf("image is already %dx%d (max %d for upscale)", info.Width, info.Height, maxDim)
	}

	prompt := info.Prompt
	negativePrompt := info.NegativePrompt

	if a.isKidsMode() {
		negativePrompt += ", " + config.KidsModeNegativePrompt
	}

	samplerName, scheduler := splitCompositeSampler(info.SamplerName, info.Scheduler)
	steps := 30
	if info.Steps > 0 {
		steps = info.Steps
	}
	cfgScale := 7.0
	if info.CfgScale > 0 {
		cfgScale = info.CfgScale
	}
	clipSkip := 1
	if info.ClipSkip > 0 {
		clipSkip = info.ClipSkip
	}

	if params.PresetID > 0 {
		p, err := a.presets.Get(params.PresetID)
		if err != nil {
			return nil, err
		}
		if p.Prompt != "" {
			prompt = p.Prompt
		}
		if p.NegativePrompt != "" {
			negativePrompt = p.NegativePrompt
		}
		if p.Sampler != "" {
			samplerName = p.Sampler
			if p.ScheduleType != "" {
				st := strings.ToUpper(p.ScheduleType[:1]) + p.ScheduleType[1:]
				samplerName = p.Sampler + " " + st
			}
		}
		if p.ScheduleType != "" {
			scheduler = p.ScheduleType
		}
		if p.Steps > 0 {
			steps = p.Steps
		}
		if p.CfgScale > 0 {
			cfgScale = p.CfgScale
		}
		if p.ClipSkip != nil {
			clipSkip = *p.ClipSkip
		}
		if p.ModelName != "" {
			_ = a.sd.SetModel(p.ModelName)
		}
		if p.VAE != "" {
			_ = a.sd.SetVAE(p.VAE)
		}
	}

	denoisingStrength := 0.4
	newWidth := info.Width * 2
	newHeight := info.Height * 2
	if newWidth > maxDim*2 {
		newWidth = maxDim * 2
	}
	if newHeight > maxDim*2 {
		newHeight = maxDim * 2
	}
	seed := info.Seed

	result, err := a.sd.Img2Img(sd.Img2ImgRequest{
		InitImages:        []string{params.ImageBase64},
		Prompt:            prompt,
		NegativePrompt:    negativePrompt,
		SamplerName:       samplerName,
		Scheduler:         scheduler,
		Steps:             steps,
		CfgScale:          cfgScale,
		Width:             newWidth,
		Height:            newHeight,
		Seed:              &seed,
		DenoisingStrength: &denoisingStrength,
		ClipSkip:          &clipSkip,
		BatchSize:         intPtr(1),
		BatchCount:        intPtr(1),
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

func intPtr(v int) *int { return &v }

func padToAspectRatio(imageBase64 string, targetW, targetH int) (string, error) {
	imgData, err := base64.StdEncoding.DecodeString(imageBase64)
	if err != nil {
		return "", fmt.Errorf("decode base64: %w", err)
	}

	img, _, err := image.Decode(bytes.NewReader(imgData))
	if err != nil {
		return "", fmt.Errorf("decode image: %w", err)
	}

	imgW := img.Bounds().Dx()
	imgH := img.Bounds().Dy()

	targetRatio := float64(targetW) / float64(targetH)
	imgRatio := float64(imgW) / float64(imgH)

	if math.Abs(targetRatio-imgRatio) < 0.01 {
		return imageBase64, nil
	}

	var padW, padH int
	if imgRatio > targetRatio {
		padW = imgW
		padH = int(float64(imgW) / targetRatio)
	} else {
		padH = imgH
		padW = int(float64(imgH) * targetRatio)
	}
	padW = (padW / 8) * 8
	padH = (padH / 8) * 8

	canvas := image.NewRGBA(image.Rect(0, 0, padW, padH))
	draw.Draw(canvas, canvas.Bounds(), &image.Uniform{color.Black}, image.Point{}, draw.Src)

	offsetX := (padW - imgW) / 2
	offsetY := (padH - imgH) / 2
	draw.Draw(canvas, image.Rect(offsetX, offsetY, offsetX+imgW, offsetY+imgH), img, image.Point{}, draw.Over)

	var buf bytes.Buffer
	if err := png.Encode(&buf, canvas); err != nil {
		return "", fmt.Errorf("encode image: %w", err)
	}

	return base64.StdEncoding.EncodeToString(buf.Bytes()), nil
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

	initImage := params.PreviewImageBase64
	padded, err := padToAspectRatio(initImage, p.Width, p.Height)
	if err == nil {
		initImage = padded
	}

	result, err := a.sd.Img2Img(sd.Img2ImgRequest{
		InitImages:        []string{initImage},
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
		"llm_url":                   a.config.LLMUrl,
		"sd_url":                    a.config.SDUrl,
		"llm_model":                 a.config.LLMModel,
		"sd_prompt_model":           a.config.SDPromptModel,
		"vision_model":              a.config.VisionModel,
		"llm_backend":               a.config.LLMBackend,
		"llm_keep_alive":            "5m",
		"llm_num_ctx":               "4096",
		"llm_num_gpu":               "0",
		"llm_max_tokens":            "256",
		"llm_generate_model":        a.config.SDPromptModel,
		"llm_analyze_model":         a.config.VisionModel,
		"llm_generate_temperature":  "0.4",
		"llm_generate_num_ctx":      "4096",
		"llm_generate_num_predict":  "256",
		"llm_generate_top_p":        "0.9",
		"llm_generate_num_thread":   "0",
		"llm_analyze_temperature":   "0.4",
		"llm_analyze_num_ctx":       "4096",
		"llm_analyze_num_predict":   "256",
		"llm_analyze_top_p":         "0.9",
		"llm_analyze_num_thread":    "0",
		"kids_mode":                 "false",
		"preview_mode":              "false",
		"preview_width":             "512",
		"preview_height":            "512",
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
		"vision_model": true,
		"llm_backend": true, "llm_keep_alive": true, "llm_num_ctx": true, "llm_num_gpu": true, "llm_max_tokens": true,
		"llm_generate_model": true, "llm_analyze_model": true,
		"llm_generate_temperature": true, "llm_generate_num_ctx": true, "llm_generate_num_predict": true,
		"llm_generate_top_p": true, "llm_generate_num_thread": true,
		"llm_analyze_temperature": true, "llm_analyze_num_ctx": true, "llm_analyze_num_predict": true,
		"llm_analyze_top_p": true, "llm_analyze_num_thread": true,
		"kids_mode": true, "kids_pin_hash": true,
		"preview_mode": true, "preview_width": true, "preview_height": true,
		"gen_preset_id": true, "gen_action_pose": true, "gen_characters": true,
		"gen_clothing_details": true,
		"gen_environment": true, "gen_lighting": true, "gen_negative": true,
		"gen_extra_prompt": true, "gen_extra_negative": true,
		"gen_type_id": true,
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
	if v, ok := data["vision_model"]; ok {
		a.config.VisionModel = v
	}
	if v, ok := data["llm_backend"]; ok {
		a.llm.SetBackend(v)
		a.config.LLMBackend = v
	}
	if v, ok := data["llm_generate_model"]; ok {
		a.config.SDPromptModel = v
	}
	if v, ok := data["llm_analyze_model"]; ok {
		a.config.VisionModel = v
	}

	return nil
}

func (a *App) applyLLMConfig(mode string) {
	prefix := "llm_generate_"
	if mode == "analyze" {
		prefix = "llm_analyze_"
	}

	var cfg llm.BackendConfig
	if v, err := a.presets.GetSetting("llm_keep_alive"); err == nil {
		cfg.KeepAlive = v
	}
	if v, err := a.presets.GetSetting(prefix + "num_ctx"); err == nil && v != "" {
		if n, err := strconv.Atoi(v); err == nil {
			cfg.NumCtx = n
		}
	}
	if v, err := a.presets.GetSetting(prefix + "num_predict"); err == nil && v != "" {
		if n, err := strconv.Atoi(v); err == nil {
			cfg.NumPredict = n
		}
	}
	if v, err := a.presets.GetSetting(prefix + "top_p"); err == nil && v != "" {
		if n, err := strconv.ParseFloat(v, 64); err == nil {
			cfg.TopP = n
		}
	}
	if v, err := a.presets.GetSetting(prefix + "num_thread"); err == nil && v != "" {
		if n, err := strconv.Atoi(v); err == nil {
			cfg.NumThread = n
		}
	}
	if v, err := a.presets.GetSetting("llm_num_gpu"); err == nil && v != "" {
		if n, err := strconv.Atoi(v); err == nil {
			cfg.NumGPU = n
		}
	}
	a.llm.SetBackendConfig(cfg)
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
	TypeName       string  `json:"type_name"`
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
	Tags                   string   `json:"tags"`
	Loras                  string   `json:"loras"`
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
		Version:    2,
		ExportedAt: time.Now().UTC(),
		Presets:    make([]PresetData, len(selected)),
	}

	typeMap := make(map[int64]string)
	types, _ := a.presets.ListPresetTypes()
	for _, t := range types {
		typeMap[t.ID] = t.Name
	}

	for i, p := range selected {
		typeName := p.PresetType
		if p.TypeID != nil {
			if n, ok := typeMap[*p.TypeID]; ok {
				typeName = n
			}
		}
		data.Presets[i] = PresetData{
			Name:                   p.Name,
			PresetType:             p.PresetType,
			TypeName:               typeName,
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
			Tags:                   p.Tags,
			Loras:                  p.Loras,
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

	if data.Version < 1 || data.Version > 2 {
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

type ValidationWarning struct {
	PresetName string   `json:"preset_name"`
	Warnings   []string `json:"warnings"`
}

func (a *App) ValidateImportModels(items []PresetData) ([]ValidationWarning, error) {
	if len(items) == 0 {
		return nil, nil
	}

	var warnings []ValidationWarning

	sdModels, _ := a.sd.GetModels()
	modelSet := make(map[string]bool)
	for _, m := range sdModels {
		modelSet[m.Name] = true
	}

	loras, _ := a.sd.GetLoRAs()
	loraSet := make(map[string]bool)
	for _, l := range loras {
		loraSet[l.Name] = true
	}

	for _, item := range items {
		var w []string
		if item.ModelName != "" && !modelSet[item.ModelName] {
			w = append(w, "Model not found: "+item.ModelName)
		}
		if item.Loras != "" {
			var loraEntries []preset.LoRAEntry
			if err := json.Unmarshal([]byte(item.Loras), &loraEntries); err == nil {
				for _, l := range loraEntries {
					if !loraSet[l.Name] {
						w = append(w, "LoRA not found: "+l.Name)
					}
				}
			}
		}
		if len(w) > 0 {
			warnings = append(warnings, ValidationWarning{
				PresetName: item.Name,
				Warnings:   w,
			})
		}
	}

	return warnings, nil
}

func extractJSON(s string) string {
	s = strings.TrimSpace(s)
	s = strings.TrimPrefix(s, "```json")
	s = strings.TrimPrefix(s, "```")
	s = strings.TrimSuffix(s, "```")
	s = strings.TrimSpace(s)
	s = strings.ReplaceAll(s, `\_`, "_")
	start := strings.Index(s, "{")
	if start < 0 {
		return s
	}
	end := strings.LastIndex(s, "}")
	if end <= start {
		return s
	}
	return s[start : end+1]
}

func truncateRepetitive(s string, maxLen int) string {
	if s == "" {
		return s
	}
	parts := strings.Split(s, ", ")
	result := make([]string, 0, len(parts))
	prevPrefix := ""
	repeatCount := 0
	for _, part := range parts {
		prefix := part
		if idx := strings.Index(part, ":"); idx > 0 {
			prefix = part[:idx]
		}
		prefix = strings.ToLower(strings.TrimSpace(prefix))
		if prefix == prevPrefix && prefix != "" {
			repeatCount++
			if repeatCount >= 3 {
				break
			}
		} else {
			prevPrefix = prefix
			repeatCount = 0
		}
		result = append(result, part)
	}
	s = strings.Join(result, ", ")
	if len(s) > maxLen {
		if idx := strings.LastIndex(s[:maxLen], ","); idx > 0 {
			s = s[:idx]
		} else {
			s = s[:maxLen]
		}
	}
	return strings.TrimSpace(s)
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

	typeCache := make(map[string]*int64)
	for _, item := range items {
		typeName := item.TypeName
		if typeName == "" {
			typeName = item.PresetType
		}
		if typeName == "" {
			continue
		}
		if _, ok := typeCache[typeName]; ok {
			continue
		}
		existing, err := a.presets.ListPresetTypes()
		if err == nil {
			for _, t := range existing {
				if t.Name == typeName {
					typeCache[typeName] = &t.ID
					break
				}
			}
		}
		if _, ok := typeCache[typeName]; !ok {
			pt := &preset.PresetType{Name: typeName}
			if err := a.presets.CreatePresetType(pt); err == nil {
				typeCache[typeName] = &pt.ID
			}
		}
	}

	batch := make([]preset.Preset, len(items))
	for i, item := range items {
		sampler, scheduleType := splitCompositeSampler(item.Sampler, item.ScheduleType)
		p := preset.Preset{
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
			Tags:                   item.Tags,
			Loras:                  item.Loras,
		}

		typeName := item.TypeName
		if typeName == "" {
			typeName = item.PresetType
		}
		if typeName != "" {
			if id, ok := typeCache[typeName]; ok {
				p.TypeID = id
			}
		}

		batch[i] = p
	}

	created, err := a.presets.CreateBatch(batch)
	if err != nil {
		return nil, err
	}
	return created, nil
}
