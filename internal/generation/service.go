package generation

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"image"
	"image/color"
	"image/draw"
	"image/png"
	"math"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"

	"go-sd/internal/compositor"
	"go-sd/internal/config"
	"go-sd/internal/kids"
	"go-sd/internal/llm"
	"go-sd/internal/logger"
	"go-sd/internal/preset"
	"go-sd/internal/promptutil"
	"go-sd/internal/rembg"
	"go-sd/internal/sd"
)

// --- Interfaces ---

type EventEmitter interface {
	Emit(event string, data ...any)
}

type SettingsApplier interface {
	ApplyLLMConfig(mode string)
}

type SessionAdder interface {
	AddToSession(imageBase64 string, info json.RawMessage, source string, isPreview bool, presetID *int64) int64
}

// --- Types ---

type SDProgressEvent struct {
	Progress    float64 `json:"progress"`
	ETARelative float64 `json:"eta_relative"`
	Job         string  `json:"job"`
	JobCount    int     `json:"job_count"`
	JobNo       int     `json:"job_no"`
	Steps       int     `json:"steps"`
	SamplerName string  `json:"sampler_name"`
	Preview     string  `json:"preview,omitempty"`
}

type GenerateSDPromptParams struct {
	PresetID    int64  `json:"preset_id"`
	Description string `json:"description"`
	Negative    string `json:"negative"`
}

type GenerateSDPromptResult struct {
	Prompt         string `json:"prompt"`
	NegativePrompt string `json:"negative_prompt"`
}

type RecommendPresetResult struct {
	PresetID    int64  `json:"preset_id"`
	PresetName  string `json:"preset_name"`
	ExtraPrompt string `json:"extra_prompt"`
	Reasoning   string `json:"reasoning"`
}

type AnalyzePrompts struct {
	SystemPrompt string   `json:"system_prompt"`
	SinglePrompt string   `json:"single_prompt"`
	ChainPrompts []string `json:"chain_prompts"`
}

type GenerateImageParams struct {
	PresetID            int64  `json:"preset_id"`
	ExtraPrompt         string `json:"extra_prompt"`
	ExtraNegativePrompt string `json:"extra_negative_prompt"`
}

type GenerateImageResult struct {
	Image                   any    `json:"image"`
	Parameters              any    `json:"parameters"`
	Info                    any    `json:"info"`
	IsPreview               bool   `json:"is_preview"`
	HiresFixSkipped         bool   `json:"hires_fix_skipped"`
	HiresFixManual          bool   `json:"hires_fix_manual"`
	EffectivePrompt         string `json:"effective_prompt"`
	EffectiveNegativePrompt string `json:"effective_negative_prompt"`
}

type UpscaleImageParams struct {
	ImageBase64 string `json:"image_base64"`
	GenInfo     string `json:"gen_info"`
	PresetID    int64  `json:"preset_id"`
}

type BatchGenerateParams struct {
	PresetID       int64  `json:"preset_id"`
	Prompt         string `json:"prompt"`
	NegativePrompt string `json:"negative_prompt"`
	Count          int    `json:"count"`
	OutputFolder   string `json:"output_folder"`
}

type BatchProgress struct {
	Current  int    `json:"current"`
	Total    int    `json:"total"`
	FilePath string `json:"file_path"`
	Status   string `json:"status"`
}

type TestGenerateParams struct {
	Mode           string   `json:"mode"`
	SelectedIDs    []int64  `json:"selected_ids"`
	SelectedModels []string `json:"selected_models"`
	Prompt         string   `json:"prompt"`
	NegativePrompt string   `json:"negative_prompt"`
	Sampler        string   `json:"sampler"`
	ScheduleType   string   `json:"schedule_type"`
	Steps          int      `json:"steps"`
	CfgScale       float64  `json:"cfg_scale"`
	Width          int      `json:"width"`
	Height         int      `json:"height"`
	Seed           *int64   `json:"seed"`
}

type TestGenerateResultItem struct {
	Name         string  `json:"name"`
	Image        string  `json:"image"`
	Seed         int64   `json:"seed"`
	Error        string  `json:"error,omitempty"`
	Sampler      string  `json:"sampler"`
	ScheduleType string  `json:"schedule_type"`
	CfgScale     float64 `json:"cfg_scale"`
	ModelName    string  `json:"model_name"`
}

type UpscalePreviewParams struct {
	PreviewImageBase64 string   `json:"preview_image_base64"`
	PresetID           int64    `json:"preset_id"`
	Seed               int64    `json:"seed"`
	DenoisingStrength  *float64 `json:"denoising_strength,omitempty"`
}

type GenerateCompoundImageParams struct {
	CompoundPresetID    int64  `json:"compound_preset_id"`
	ExtraPrompt         string `json:"extra_prompt"`
	ExtraNegativePrompt string `json:"extra_negative_prompt"`
}

type GenerateFromImageParams struct {
	ImageBase64         string  `json:"image_base64"`
	Mode                string  `json:"mode"`
	GenMode             string  `json:"gen_mode"`
	PresetID            int64   `json:"preset_id"`
	CompoundPresetID    int64   `json:"compound_preset_id"`
	DenoisingStrength   float64 `json:"denoising_strength"`
	Tags                string  `json:"tags"`
	ExtraNegativePrompt string  `json:"extra_negative_prompt"`
	MaskBase64          string  `json:"mask_base64"`
	MaskBlur            int     `json:"mask_blur"`
	InpaintFill         int     `json:"inpaint_fill"`
	InpaintFullRes      bool    `json:"inpaint_full_res"`
	RemoveObject        bool    `json:"remove_object"`
}

type BatchCompoundGenerateParams struct {
	CompoundPresetID    int64  `json:"compound_preset_id"`
	ExtraPrompt         string `json:"extra_prompt"`
	ExtraNegativePrompt string `json:"extra_negative_prompt"`
	Count               int    `json:"count"`
	OutputFolder        string `json:"output_folder"`
}

type TestCompoundGenerateParams struct {
	SelectedIDs    []int64 `json:"selected_ids"`
	Prompt         string  `json:"prompt"`
	NegativePrompt string  `json:"negative_prompt"`
}

type DecomposeSceneParams struct {
	Description string `json:"description"`
	PresetID    int64  `json:"preset_id"`
}

type lastImageMeta struct {
	IsPreview bool            `json:"is_preview"`
	Info      json.RawMessage `json:"info"`
}

// --- Service ---

type Service struct {
	db        *preset.DB
	llm       llm.Service
	sd        sd.Service
	cfg       *config.Config
	rembg     *rembg.Client
	dataDir   string
	emitter   EventEmitter
	kids      *kids.Manager
	sessions  SessionAdder
	settings  SettingsApplier
	log       *logger.Logger

	ctx            context.Context
	sdPollingMu    sync.Mutex
	sdPollingCancel context.CancelFunc
	sdInterrupted  bool

	batchMu      sync.Mutex
	batchRunning bool
}

func New(
	db *preset.DB,
	llmSvc llm.Service,
	sdSvc sd.Service,
	cfg *config.Config,
	rembgClient *rembg.Client,
	dataDir string,
	emitter EventEmitter,
	kidsMgr *kids.Manager,
	sessions SessionAdder,
	settings SettingsApplier,
	log *logger.Logger,
) *Service {
	return &Service{
		db:       db,
		llm:      llmSvc,
		sd:       sdSvc,
		cfg:      cfg,
		rembg:    rembgClient,
		dataDir:  dataDir,
		emitter:  emitter,
		kids:     kidsMgr,
		sessions: sessions,
		settings: settings,
		log:      log,
	}
}

func (s *Service) SetContext(ctx context.Context) {
	s.ctx = ctx
}

// --- SD Polling ---

func (s *Service) StartSDPolling() {
	s.sdPollingMu.Lock()
	defer s.sdPollingMu.Unlock()
	s.sdInterrupted = false
	if s.sdPollingCancel != nil {
		return
	}
	ctx, cancel := context.WithCancel(s.ctx)
	s.sdPollingCancel = cancel
	go func() {
		ticker := time.NewTicker(500 * time.Millisecond)
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				prog, err := s.sd.GetProgress()
				if err != nil {
					continue
				}
				s.emitter.Emit("sd:progress", SDProgressEvent{
					Progress:    prog.Progress,
					ETARelative: prog.ETARelative,
					Job:         prog.State.Job,
					JobCount:    prog.State.JobCount,
					JobNo:       prog.State.JobNo,
					Steps:       prog.State.Sampling.Steps,
					SamplerName: prog.State.Sampling.SamplerName,
					Preview:     prog.CurrentImage,
				})
			}
		}
	}()
}

func (s *Service) StopSDPolling() {
	s.sdPollingMu.Lock()
	defer s.sdPollingMu.Unlock()
	if s.sdPollingCancel != nil {
		s.sdPollingCancel()
		s.sdPollingCancel = nil
	}
}

func (s *Service) InterruptGeneration() error {
	s.sdPollingMu.Lock()
	s.sdInterrupted = true
	s.sdPollingMu.Unlock()
	return s.sd.Interrupt()
}

func (s *Service) checkSDInterrupted() error {
	s.sdPollingMu.Lock()
	interrupted := s.sdInterrupted
	s.sdPollingMu.Unlock()
	if interrupted {
		return fmt.Errorf("interrupted")
	}
	return nil
}

func (s *Service) manualHiresUpscale(base64Img string, req sd.Txt2ImgRequest, scale float64, denoiseStrength float64) (*sd.Txt2ImgResponse, error) {
	targetW := int(float64(req.Width) * scale)
	targetH := int(float64(req.Height) * scale)
	i2iReq := sd.Img2ImgRequest{
		InitImages:        []string{base64Img},
		Prompt:            req.Prompt,
		NegativePrompt:    req.NegativePrompt,
		SamplerName:       req.SamplerName,
		Scheduler:         req.Scheduler,
		Steps:             req.Steps,
		CfgScale:          req.CfgScale,
		Width:             targetW,
		Height:            targetH,
		Seed:              req.Seed,
		DenoisingStrength: &denoiseStrength,
		ClipSkip:          req.ClipSkip,
		BatchSize:         req.BatchSize,
		BatchCount:        req.BatchCount,
		DoNotSaveImages:   true,
		DoNotSaveGrid:     true,
	}
	return s.sd.Img2Img(i2iReq)
}

// --- Helpers ---

func (s *Service) getMaxTokens() int {
	maxTokens := 256
	if v, err := s.db.GetSetting("llm_max_tokens"); err == nil && v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 {
			maxTokens = n
		}
	}
	return maxTokens
}

func (s *Service) getGenerateModel() string {
	generateModel := s.cfg.SDPromptModel
	if v, err := s.db.GetSetting("llm_generate_model"); err == nil && v != "" {
		generateModel = v
	}
	return generateModel
}

func (s *Service) getSDPromptInstruction() string {
	sdPromptInstruction := config.DefaultSDPromptInstruction
	if v, err := s.db.GetSetting("sd_prompt_instruction"); err == nil && v != "" {
		sdPromptInstruction = v
	}
	return sdPromptInstruction
}

func (s *Service) getAnalyzeModel() string {
	model := s.cfg.VisionModel
	if v, err := s.db.GetSetting("llm_analyze_model"); err == nil && v != "" {
		model = v
	}
	if model == "" {
		model = s.cfg.SDPromptModel
		if v, err := s.db.GetSetting("llm_generate_model"); err == nil && v != "" {
			model = v
		}
	}
	return model
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

func extractEmbeddedNegative(result *GenerateSDPromptResult) {
	idx := strings.Index(result.Prompt, "negative_prompt")
	if idx <= 0 {
		return
	}
	embeddedNeg := result.Prompt[idx:]
	result.Prompt = strings.TrimRight(result.Prompt[:idx], " ,\n\r\t")
	embeddedNeg = strings.TrimPrefix(embeddedNeg, "negative_prompt")
	embeddedNeg = strings.TrimLeft(embeddedNeg, `: "'`)
	embeddedNeg = strings.TrimRight(embeddedNeg, `"}'`)
	embeddedNeg = strings.Trim(embeddedNeg, " ,\n\r\t")
	if embeddedNeg == "" {
		return
	}
	if result.NegativePrompt != "" {
		result.NegativePrompt = embeddedNeg + ", " + result.NegativePrompt
	} else {
		result.NegativePrompt = embeddedNeg
	}
}

func buildSamplerName(sampler, scheduleType string) string {
	if scheduleType != "" {
		st := strings.ToUpper(scheduleType[:1]) + scheduleType[1:]
		return sampler + " " + st
	}
	return sampler
}

func appendLorasToPrompt(prompt, lorasJSON string) string {
	if lorasJSON == "" {
		return prompt
	}
	var loras []preset.LoRAEntry
	if json.Unmarshal([]byte(lorasJSON), &loras) == nil {
		for _, l := range loras {
			prompt += fmt.Sprintf(" <lora:%s:%g>", l.Name, l.Weight)
		}
	}
	return prompt
}

func (s *Service) getPreviewDimensions(presetW, presetH int) (int, int, bool) {
	if v, _ := s.db.GetSetting("preview_mode"); v != "true" {
		return presetW, presetH, false
	}
	maxW, maxH := 512, 512
	if pw, _ := s.db.GetSetting("preview_width"); pw != "" {
		if n, err := strconv.Atoi(pw); err == nil && n > 0 {
			maxW = n
		}
	}
	if ph, _ := s.db.GetSetting("preview_height"); ph != "" {
		if n, err := strconv.Atoi(ph); err == nil && n > 0 {
			maxH = n
		}
	}
	targetRatio := float64(presetW) / float64(presetH)
	maxRatio := float64(maxW) / float64(maxH)
	w, h := presetW, presetH
	if maxRatio > targetRatio {
		h = maxH
		w = int(float64(maxH) * targetRatio)
	} else {
		w = maxW
		h = int(float64(maxW) / targetRatio)
	}
	w = (w / 8) * 8
	h = (h / 8) * 8
	if w < 64 {
		w = 64
	}
	if h < 64 {
		h = 64
	}
	return w, h, true
}

// --- GenerateSDPrompt ---

func (s *Service) GenerateSDPrompt(params GenerateSDPromptParams) (*GenerateSDPromptResult, error) {
	if params.PresetID <= 0 {
		return nil, fmt.Errorf("preset is required")
	}

	p, err := s.db.Get(params.PresetID)
	if err != nil {
		return nil, fmt.Errorf("preset not found: %w", err)
	}

	description := strings.TrimSpace(params.Description)
	negative := strings.TrimSpace(params.Negative)

	if description == "" && negative == "" {
		return &GenerateSDPromptResult{
			Prompt:         p.Prompt,
			NegativePrompt: p.NegativePrompt,
		}, nil
	}

	systemPrompt := s.getSDPromptInstruction()

	var filterErr error
	description, filterErr = s.kids.FilterInput(description)
	if filterErr != nil {
		return nil, filterErr
	}
	negative, filterErr = s.kids.FilterInput(negative)
	if filterErr != nil {
		return nil, filterErr
	}
	systemPrompt = s.kids.ApplySystemPrompt(systemPrompt)

	maxTokens := s.getMaxTokens()
	generateModel := s.getGenerateModel()

	s.settings.ApplyLLMConfig("generate")

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

	s.emitter.Emit("llm:status", map[string]string{"status": "thinking"})
	raw, err := s.llm.GenerateSDPrompt(systemPrompt, userMessage, p.PresetType, generateModel, maxTokens)
	if err != nil {
		s.emitter.Emit("llm:status", map[string]string{"status": "done"})
		return nil, err
	}
	s.emitter.Emit("llm:status", map[string]string{"status": "done"})

	var result GenerateSDPromptResult
	jsonRaw := promptutil.ExtractJSON(raw)
	if err := json.Unmarshal([]byte(jsonRaw), &result); err != nil {
		result = GenerateSDPromptResult{
			Prompt:         promptutil.TruncateRepetitive(raw, 1000),
			NegativePrompt: p.NegativePrompt,
		}
	}

	if promptutil.ContainsCyrillic(result.Prompt) {
		result.Prompt = promptutil.ExtractTagsFromRaw(raw)
	}
	if promptutil.ContainsCyrillic(result.NegativePrompt) {
		result.NegativePrompt = promptutil.ExtractNegativeFromRaw(raw)
	}

	extractEmbeddedNegative(&result)

	result.Prompt = promptutil.StripJunk(result.Prompt)
	result.Prompt = promptutil.TruncateRepetitive(result.Prompt, 1000)
	result.NegativePrompt = promptutil.StripJunk(result.NegativePrompt)
	result.NegativePrompt = promptutil.TruncateRepetitive(result.NegativePrompt, 500)

	result.Prompt = s.kids.FilterOutput(result.Prompt)
	result.NegativePrompt = s.kids.FilterOutput(result.NegativePrompt)

	return &result, nil
}

func (s *Service) GetDefaultPromptInstruction() string {
	return config.DefaultSDPromptInstruction
}

// --- RecommendPreset ---

func (s *Service) RecommendPreset(description string) (*RecommendPresetResult, error) {
	if strings.TrimSpace(description) == "" {
		return nil, fmt.Errorf("description is required")
	}

	allPresets, err := s.db.List()
	if err != nil {
		return nil, fmt.Errorf("load presets: %w", err)
	}
	if len(allPresets) == 0 {
		return nil, fmt.Errorf("no presets available")
	}

	typesMap := make(map[int64]string)
	types, _ := s.db.ListPresetTypes()
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

	generateModel := s.getGenerateModel()
	s.settings.ApplyLLMConfig("generate")

	maxTokens := 512
	if v, err := s.db.GetSetting("llm_max_tokens"); err == nil && v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 {
			maxTokens = n
		}
	}

	s.emitter.Emit("llm:status", map[string]string{"status": "thinking"})
	raw, err := s.llm.GenerateSDPrompt(systemPrompt, userMessage, "", generateModel, maxTokens)
	if err != nil {
		s.emitter.Emit("llm:status", map[string]string{"status": "done"})
		return nil, err
	}
	s.emitter.Emit("llm:status", map[string]string{"status": "done"})

	var result RecommendPresetResult
	if err := json.Unmarshal([]byte(promptutil.ExtractJSON(raw)), &result); err != nil {
		return nil, fmt.Errorf("failed to parse LLM response: %w", err)
	}

	return &result, nil
}

// --- GenerateImage ---

func (s *Service) GenerateImage(params GenerateImageParams) (*GenerateImageResult, error) {
	s.log.UserAction("Generate image (preset_id=%d)", params.PresetID)
	s.StartSDPolling()
	defer s.StopSDPolling()
	p, err := s.db.Get(params.PresetID)
	if err != nil {
		s.log.Error("Generate image: preset not found: %s", err)
		return nil, err
	}

	prompt := p.Prompt
	if params.ExtraPrompt != "" {
		prompt = params.ExtraPrompt
	}
	prompt = appendLorasToPrompt(prompt, p.Loras)

	negativePrompt := p.NegativePrompt
	if params.ExtraNegativePrompt != "" {
		negativePrompt = params.ExtraNegativePrompt
	}

	negativePrompt = s.kids.ApplyNegative(negativePrompt)

	if p.ModelName != "" {
		_ = s.sd.SetModel(p.ModelName)
	}

	if p.VAE != "" {
		_ = s.sd.SetVAE(p.VAE)
	}

	samplerName := buildSamplerName(p.Sampler, p.ScheduleType)

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

	width, height := p.Width, p.Height
	var hiresFix *bool
	if p.HiresFix != nil {
		hiresFix = p.HiresFix
	}

	isPreview := false
	if pw, ph, preview := s.getPreviewDimensions(p.Width, p.Height); preview {
		isPreview = true
		width = pw
		height = ph
		hiresFix = nil
	}

	req := sd.Txt2ImgRequest{
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
	}

	result, err := s.sd.Txt2Img(req)
	hiresSkipped := false
	hiresManual := false
	if err != nil {
		if ierr := s.checkSDInterrupted(); ierr != nil {
			return nil, ierr
		}
		if hiresFix != nil {
			s.log.Warn("SD error with hires fix enabled, retrying with manual upscale: %s", err)
			time.Sleep(3 * time.Second)
			req.HiresFix = nil
			req.HiresUpscale = nil
			req.HiresDenoisingStrength = nil
			req.HiresUpscaler = ""
			req.HiresResizeX = 0
			req.HiresResizeY = 0
			req.HiresSecondPassSteps = 0
			result, err = s.sd.Txt2Img(req)
			if err == nil && len(result.Images) > 0 {
				scale := 2.0
				if p.HiresUpscale != nil {
					scale = *p.HiresUpscale
				}
				ds := 0.5
				if p.HiresDenoisingStrength != nil {
					ds = *p.HiresDenoisingStrength
				}
				s.log.Info("Manual hires upscale: %.1fx, denoise=%.2f", scale, ds)
				hrResult, hrErr := s.manualHiresUpscale(result.Images[0], req, scale, ds)
				if hrErr != nil {
					s.log.Warn("Manual hires upscale failed, using base image: %s", hrErr)
					hiresSkipped = true
				} else if len(hrResult.Images) > 0 {
					result = hrResult
					hiresManual = true
				} else {
					hiresSkipped = true
				}
			} else {
				hiresSkipped = true
			}
		}
		if err != nil {
			return nil, err
		}
	}

	if len(result.Images) == 0 {
		if ierr := s.checkSDInterrupted(); ierr != nil {
			return nil, ierr
		}
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
		Image:                   result.Images[0],
		Parameters:              result.Parameters,
		Info:                    result.Info,
		IsPreview:               isPreview,
		HiresFixSkipped:         hiresSkipped,
		HiresFixManual:          hiresManual,
		EffectivePrompt:         prompt,
		EffectiveNegativePrompt: negativePrompt,
	}
	s.sessions.AddToSession(result.Images[0], result.Info, "generate", isPreview, nil)
	return img, nil
}

// --- BatchGenerate ---

func (s *Service) BatchGenerate(params BatchGenerateParams) error {
	s.StartSDPolling()
	defer s.StopSDPolling()
	if params.Count <= 0 || params.Count > 100 {
		return fmt.Errorf("count must be between 1 and 100")
	}
	if params.OutputFolder == "" {
		return fmt.Errorf("output folder is required")
	}
	if params.Prompt == "" {
		return fmt.Errorf("prompt is required")
	}

	s.batchMu.Lock()
	if s.batchRunning {
		s.batchMu.Unlock()
		return fmt.Errorf("batch generation is already running")
	}
	s.batchRunning = true
	s.batchMu.Unlock()
	defer func() {
		s.batchMu.Lock()
		s.batchRunning = false
		s.batchMu.Unlock()
	}()

	if err := os.MkdirAll(params.OutputFolder, 0755); err != nil {
		return fmt.Errorf("create output folder: %w", err)
	}

	p := &preset.Preset{
		Prompt:         "",
		NegativePrompt: "",
		Sampler:        "Euler a",
		Steps:          20,
		CfgScale:       7.0,
		Width:          512,
		Height:         512,
	}
	if params.PresetID > 0 {
		var err error
		p, err = s.db.Get(params.PresetID)
		if err != nil {
			return fmt.Errorf("preset not found: %w", err)
		}
	}

	prompt := params.Prompt
	var filterErr error
	prompt, filterErr = s.kids.FilterInput(prompt)
	if filterErr != nil {
		return filterErr
	}
	prompt = appendLorasToPrompt(prompt, p.Loras)

	negativePrompt := params.NegativePrompt
	negativePrompt, filterErr = s.kids.FilterInput(negativePrompt)
	if filterErr != nil {
		return filterErr
	}
	negativePrompt = s.kids.ApplyNegative(negativePrompt)

	if p.ModelName != "" {
		_ = s.sd.SetModel(p.ModelName)
	}
	if p.VAE != "" {
		_ = s.sd.SetVAE(p.VAE)
	}

	samplerName := buildSamplerName(p.Sampler, p.ScheduleType)

	clipSkip := 1
	if p.ClipSkip != nil {
		clipSkip = *p.ClipSkip
	}
	batchSize := 1
	batchCount := 1

	denoisingStrength := p.DenoisingStrength
	if denoisingStrength == nil && p.HiresFix != nil && *p.HiresFix {
		ds := 0.5
		if p.HiresDenoisingStrength != nil {
			ds = *p.HiresDenoisingStrength
		}
		denoisingStrength = &ds
	}
	var hiresFix *bool
	if p.HiresFix != nil {
		hiresFix = p.HiresFix
	}

	timestamp := time.Now().Format("20060102_150405")

	for i := 0; i < params.Count; i++ {
		s.emitter.Emit("batch:progress", BatchProgress{
			Current: i + 1,
			Total:   params.Count,
			Status:  "generating",
		})

		req := sd.Txt2ImgRequest{
			Prompt:                 prompt,
			NegativePrompt:         negativePrompt,
			SamplerName:            samplerName,
			Scheduler:              p.ScheduleType,
			Steps:                  p.Steps,
			CfgScale:               p.CfgScale,
			Width:                  p.Width,
			Height:                 p.Height,
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
		}

		result, err := s.sd.Txt2Img(req)
		if err != nil {
			if ierr := s.checkSDInterrupted(); ierr != nil {
				s.emitter.Emit("batch:progress", BatchProgress{
					Current: i + 1,
					Total:   params.Count,
					Status:  "interrupted",
				})
				return ierr
			}
			if hiresFix != nil {
				s.log.Warn("SD error with hires fix enabled, retrying with manual upscale: %s", err)
				time.Sleep(3 * time.Second)
				req.HiresFix = nil
				req.HiresUpscale = nil
				req.HiresDenoisingStrength = nil
				req.HiresUpscaler = ""
				req.HiresResizeX = 0
				req.HiresResizeY = 0
				req.HiresSecondPassSteps = 0
				result, err = s.sd.Txt2Img(req)
				if err == nil && len(result.Images) > 0 {
					scale := 2.0
					if p.HiresUpscale != nil {
						scale = *p.HiresUpscale
					}
					ds := 0.5
					if p.HiresDenoisingStrength != nil {
						ds = *p.HiresDenoisingStrength
					}
					s.log.Info("Manual hires upscale: %.1fx, denoise=%.2f", scale, ds)
					hrResult, hrErr := s.manualHiresUpscale(result.Images[0], req, scale, ds)
					if hrErr == nil && len(hrResult.Images) > 0 {
						result = hrResult
					}
				}
			}
			if err != nil {
				s.emitter.Emit("batch:progress", BatchProgress{
					Current: i + 1,
					Total:   params.Count,
					Status:  fmt.Sprintf("error: image %d failed", i+1),
				})
				return fmt.Errorf("image %d/%d failed: %w", i+1, params.Count, err)
			}
		}
		if len(result.Images) == 0 {
			if ierr := s.checkSDInterrupted(); ierr != nil {
				s.emitter.Emit("batch:progress", BatchProgress{
					Current: i + 1,
					Total:   params.Count,
					Status:  "interrupted",
				})
				return ierr
			}
			s.emitter.Emit("batch:progress", BatchProgress{
				Current: i + 1,
				Total:   params.Count,
				Status:  "error: no image returned",
			})
			return fmt.Errorf("image %d/%d: no image returned", i+1, params.Count)
		}

		if len(result.Images[0]) > 67*1024*1024 {
			return fmt.Errorf("image %d too large (max 50 MB)", i+1)
		}

		imgData, err := base64.StdEncoding.DecodeString(result.Images[0])
		if err != nil {
			return fmt.Errorf("decode image %d: %w", i+1, err)
		}

		fileName := fmt.Sprintf("batch_%s_%03d.png", timestamp, i+1)
		filePath := filepath.Join(params.OutputFolder, fileName)
		if err := os.WriteFile(filePath, imgData, 0644); err != nil {
			return fmt.Errorf("save image %d: %w", i+1, err)
		}

		s.emitter.Emit("batch:progress", BatchProgress{
			Current:  i + 1,
			Total:    params.Count,
			FilePath: filePath,
			Status:   "saved",
		})
	}

	s.emitter.Emit("batch:progress", BatchProgress{
		Current: params.Count,
		Total:   params.Count,
		Status:  "done",
	})
	return nil
}

// --- TestGenerate ---

func (s *Service) TestGenerate(params TestGenerateParams) ([]TestGenerateResultItem, error) {
	s.StartSDPolling()
	defer s.StopSDPolling()
	if params.Mode != "presets" && params.Mode != "models" {
		return nil, fmt.Errorf("mode must be 'presets' or 'models'")
	}
	totalItems := len(params.SelectedIDs)
	if params.Mode == "models" {
		totalItems = len(params.SelectedModels)
	}
	if totalItems == 0 {
		return nil, fmt.Errorf("select at least one item")
	}
	if totalItems > 50 {
		return nil, fmt.Errorf("maximum 50 items at once")
	}
	if params.Prompt == "" {
		return nil, fmt.Errorf("prompt is required")
	}
	if params.Width > 2048 || params.Height > 2048 {
		return nil, fmt.Errorf("maximum dimension is 2048")
	}
	if params.Steps > 150 {
		return nil, fmt.Errorf("maximum steps is 150")
	}

	defaultPreset := &preset.Preset{
		Sampler:  "Euler a",
		Steps:    20,
		CfgScale: 7.0,
		Width:    512,
		Height:   512,
	}

	results := make([]TestGenerateResultItem, 0, totalItems)

	for idx := 0; idx < totalItems; idx++ {
		s.emitter.Emit("test:progress", map[string]any{
			"current": idx + 1,
			"total":   totalItems,
			"status":  "generating",
		})

		item := TestGenerateResultItem{}
		p := &preset.Preset{}
		*p = *defaultPreset

		if params.Mode == "presets" {
			id := params.SelectedIDs[idx]
			loaded, err := s.db.Get(id)
			if err != nil {
				item.Error = fmt.Sprintf("preset not found: %v", err)
				item.Name = fmt.Sprintf("Preset #%d", id)
				results = append(results, item)
				continue
			}
			p = loaded
			item.Name = p.Name
			if p.ModelName != "" {
				_ = s.sd.SetModel(p.ModelName)
			}
			if p.VAE != "" {
				_ = s.sd.SetVAE(p.VAE)
			}
		} else {
			modelTitle := params.SelectedModels[idx]
			_ = s.sd.SetModel(modelTitle)
			item.Name = modelTitle
		}

		if p.Sampler == "" {
			p.Sampler = defaultPreset.Sampler
		}
		if p.Steps == 0 {
			p.Steps = defaultPreset.Steps
		}
		if p.CfgScale == 0 {
			p.CfgScale = defaultPreset.CfgScale
		}
		if p.Width == 0 {
			p.Width = defaultPreset.Width
		}
		if p.Height == 0 {
			p.Height = defaultPreset.Height
		}

		prompt := params.Prompt
		prompt, filterErr := s.kids.FilterInput(prompt)
		if filterErr != nil {
			return nil, fmt.Errorf("generating image: %w", filterErr)
		}
		if p.Loras != "" && params.Mode == "presets" {
			prompt = appendLorasToPrompt(prompt, p.Loras)
		}

		negPrompt := params.NegativePrompt
		if params.Mode == "presets" && p.NegativePrompt != "" {
			if negPrompt != "" {
				negPrompt = p.NegativePrompt + ", " + negPrompt
			} else {
				negPrompt = p.NegativePrompt
			}
		}
		negPrompt = s.kids.ApplyNegative(negPrompt)

		sampler := p.Sampler
		scheduleType := p.ScheduleType
		steps := p.Steps
		cfgScale := p.CfgScale
		width := p.Width
		height := p.Height
		seed := p.Seed

		if params.Sampler != "" {
			sampler = params.Sampler
		}
		if params.ScheduleType != "" {
			scheduleType = params.ScheduleType
		}
		if params.Steps > 0 {
			steps = params.Steps
		}
		if params.CfgScale > 0 {
			cfgScale = params.CfgScale
		}
		if params.Width > 0 {
			width = params.Width
		}
		if params.Height > 0 {
			height = params.Height
		}
		if params.Seed != nil {
			seed = params.Seed
		}

		samplerName := buildSamplerName(sampler, scheduleType)

		clipSkip := 1
		if p.ClipSkip != nil {
			clipSkip = *p.ClipSkip
		}
		batchSize := 1
		batchCount := 1

		result, err := s.sd.Txt2Img(sd.Txt2ImgRequest{
			Prompt:          prompt,
			NegativePrompt:  negPrompt,
			SamplerName:     samplerName,
			Scheduler:       scheduleType,
			Steps:           steps,
			CfgScale:        cfgScale,
			Width:           width,
			Height:          height,
			Seed:            seed,
			ClipSkip:        &clipSkip,
			BatchSize:       &batchSize,
			BatchCount:      &batchCount,
			DoNotSaveImages: true,
			DoNotSaveGrid:   true,
		})
		if err != nil {
			item.Error = err.Error()
			item.Sampler = sampler
			item.ScheduleType = scheduleType
			item.CfgScale = cfgScale
			if p.ModelName != "" {
				item.ModelName = p.ModelName
			}
			results = append(results, item)
			continue
		}
		if len(result.Images) == 0 {
			if ierr := s.checkSDInterrupted(); ierr != nil {
				return nil, ierr
			}
			item.Error = "no image returned"
			results = append(results, item)
			continue
		}

		var infoSeed int64
		var infoModel string
		if len(result.Info) > 0 {
			var info struct {
				Seed    int64  `json:"seed"`
				SDModel string `json:"sd_model_name"`
			}
			if json.Unmarshal(result.Info, &info) == nil {
				infoSeed = info.Seed
				infoModel = info.SDModel
			}
		}

		item.Image = result.Images[0]
		item.Seed = infoSeed
		item.Sampler = sampler
		item.ScheduleType = scheduleType
		item.CfgScale = cfgScale
		item.ModelName = infoModel
		if item.ModelName == "" && p.ModelName != "" {
			item.ModelName = p.ModelName
		}

		results = append(results, item)

		s.emitter.Emit("test:progress", map[string]any{
			"current": idx + 1,
			"total":   totalItems,
			"status":  "done",
		})
	}

	return results, nil
}

// --- GetPresetForBatch ---

func (s *Service) GetPresetForBatch(presetID int64, description string) (*GenerateSDPromptResult, error) {
	if presetID <= 0 {
		return nil, fmt.Errorf("preset is required")
	}

	if description != "" {
		result, err := s.GenerateSDPrompt(GenerateSDPromptParams{
			PresetID:    presetID,
			Description: description,
		})
		if err != nil {
			return nil, err
		}
		return result, nil
	}

	return nil, nil
}

// --- UpscaleImage ---

func (s *Service) UpscaleImage(params UpscaleImageParams) (*GenerateImageResult, error) {
	s.StartSDPolling()
	defer s.StopSDPolling()
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

	negativePrompt = s.kids.ApplyNegative(negativePrompt)

	samplerName, scheduler := promptutil.SplitCompositeSampler(info.SamplerName, info.Scheduler)
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
		p, err := s.db.Get(params.PresetID)
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
			_ = s.sd.SetModel(p.ModelName)
		}
		if p.VAE != "" {
			_ = s.sd.SetVAE(p.VAE)
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

	result, err := s.sd.Img2Img(sd.Img2ImgRequest{
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
		if ierr := s.checkSDInterrupted(); ierr != nil {
			return nil, ierr
		}
		return nil, fmt.Errorf("no image generated during upscale")
	}

	img := &GenerateImageResult{
		Image:      result.Images[0],
		Parameters: result.Parameters,
		Info:       result.Info,
		IsPreview:  false,
	}
	s.sessions.AddToSession(result.Images[0], result.Info, "upscale", false, nil)
	return img, nil
}

// --- UpscalePreview ---

func (s *Service) UpscalePreview(params UpscalePreviewParams) (*GenerateImageResult, error) {
	s.StartSDPolling()
	defer s.StopSDPolling()
	p, err := s.db.Get(params.PresetID)
	if err != nil {
		return nil, err
	}

	prompt := p.Prompt
	negativePrompt := p.NegativePrompt

	negativePrompt = s.kids.ApplyNegative(negativePrompt)

	if p.ModelName != "" {
		_ = s.sd.SetModel(p.ModelName)
	}

	if p.VAE != "" {
		_ = s.sd.SetVAE(p.VAE)
	}

	samplerName := buildSamplerName(p.Sampler, p.ScheduleType)

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

	result, err := s.sd.Img2Img(sd.Img2ImgRequest{
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
		if ierr := s.checkSDInterrupted(); ierr != nil {
			return nil, ierr
		}
		return nil, fmt.Errorf("no image generated during upscale")
	}

	img := &GenerateImageResult{
		Image:      result.Images[0],
		Parameters: result.Parameters,
		Info:       result.Info,
		IsPreview:  false,
	}
	s.sessions.AddToSession(result.Images[0], result.Info, "upscale-preview", false, nil)
	return img, nil
}

// --- Last Image Persistence ---

func (s *Service) saveLastImage(imageBase64 string, info json.RawMessage, isPreview bool) {
	if imageBase64 == "" {
		return
	}

	pngData, err := base64.StdEncoding.DecodeString(imageBase64)
	if err != nil {
		return
	}

	if err := os.MkdirAll(s.dataDir, 0o755); err != nil {
		return
	}

	pngPath := filepath.Join(s.dataDir, "last_image.png")
	if err := os.WriteFile(pngPath, pngData, 0o644); err != nil {
		return
	}

	meta := lastImageMeta{IsPreview: isPreview, Info: info}
	metaBytes, err := json.Marshal(meta)
	if err != nil {
		return
	}

	metaPath := filepath.Join(s.dataDir, "last_image.json")
	_ = os.WriteFile(metaPath, metaBytes, 0o644)
}

func (s *Service) GetLastImage() (*GenerateImageResult, error) {
	pngPath := filepath.Join(s.dataDir, "last_image.png")
	pngData, err := os.ReadFile(pngPath)
	if err != nil {
		return nil, nil
	}

	metaPath := filepath.Join(s.dataDir, "last_image.json")
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

func (s *Service) ClearLastImage() {
	os.Remove(filepath.Join(s.dataDir, "last_image.png"))
	os.Remove(filepath.Join(s.dataDir, "last_image.json"))
}

// --- DecomposeScene ---

func (s *Service) DecomposeScene(params DecomposeSceneParams) (*compositor.Scene, error) {
	s.log.UserAction("Decompose scene: %s", promptutil.Truncate(params.Description, 80))
	if params.Description == "" {
		return nil, fmt.Errorf("description is required")
	}
	if params.PresetID <= 0 {
		return nil, fmt.Errorf("preset is required")
	}

	p, err := s.db.Get(params.PresetID)
	if err != nil {
		return nil, fmt.Errorf("preset not found: %w", err)
	}

	systemPrompt := config.DefaultSceneDecomposePrompt

	userMessage := params.Description
	userMessage += fmt.Sprintf("\n\nPreset dimensions: %dx%d", p.Width, p.Height)
	if p.Prompt != "" {
		userMessage += fmt.Sprintf("\nPreset positive prompt (STYLE — all character and background prompts MUST follow this style): %s", p.Prompt)
	}
	if p.NegativePrompt != "" {
		userMessage += fmt.Sprintf("\nPreset negative prompt (MERGE into scene negative_prompt): %s", p.NegativePrompt)
	}

	maxTokens := 1024
	if v, err := s.db.GetSetting("llm_max_tokens"); err == nil && v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 {
			maxTokens = n
		}
	}

	generateModel := s.getGenerateModel()
	s.settings.ApplyLLMConfig("generate")

	s.emitter.Emit("llm:status", map[string]string{"status": "thinking"})
	raw, err := s.llm.Chat(generateModel, systemPrompt, userMessage, 0.4, maxTokens)
	if err != nil {
		s.emitter.Emit("llm:status", map[string]string{"status": "done"})
		return nil, fmt.Errorf("LLM decomposition failed: %w", err)
	}
	s.emitter.Emit("llm:status", map[string]string{"status": "done"})

	scene, err := compositor.DecomposeSceneFromJSON(raw)
	if err != nil {
		return nil, fmt.Errorf("failed to parse scene from LLM response: %w", err)
	}

	scene.PresetID = params.PresetID
	if scene.Width == 0 {
		scene.Width = p.Width
	}
	if scene.Height == 0 {
		scene.Height = p.Height
	}

	return scene, nil
}

// --- GenerateMultiPass ---

func (s *Service) GenerateMultiPass(scene compositor.Scene) (*compositor.MultiPassResult, error) {
	s.log.UserAction("Multi-pass generation: %d characters", len(scene.Characters))

	emit := func(progress compositor.MultiPassProgress) {
		s.emitter.Emit("multipass:progress", progress)
		switch progress.Step {
		case "background":
			s.log.Info("Generating background...")
		case "character":
			s.log.Info("Generating character %d/%d", progress.Character, progress.Total)
		case "rembg":
			s.log.Info("Removing background (character %d/%d)", progress.Character, progress.Total)
		case "done":
			s.log.Info("Multi-pass generation complete")
		}
	}

	rembgURL, _ := s.db.GetSetting("rembg_url")
	var rembgIf compositor.RembgClient
	if rembgURL != "" {
		s.rembg.SetURL(rembgURL)
		rembgIf = s.rembg
		s.log.Debug("Rembg enabled: %s", rembgURL)
	} else {
		s.log.Warn("Rembg not configured, using Go-based background removal")
	}

	c := compositor.New(s.sd, rembgIf, s.db, emit)
	result, err := c.GenerateScene(scene)
	if err != nil {
		s.log.Error("Multi-pass failed: %s", err)
		return nil, err
	}

	if result.Image != "" {
		s.sessions.AddToSession(result.Image, nil, "scene", false, nil)
	}

	return result, nil
}
