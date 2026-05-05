package analyze

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"image"
	"image/color"
	"image/draw"
	"image/png"
	"math"
	"strconv"
	"strings"

	xdraw "golang.org/x/image/draw"

	"go-sd/internal/config"
	"go-sd/internal/filebrowser"
	"go-sd/internal/kids"
	"go-sd/internal/llm"
	"go-sd/internal/preset"
	"go-sd/internal/promptutil"
	"go-sd/internal/sd"
)

type EventEmitter interface {
	Emit(event string, data ...any)
}

type SettingsApplier interface {
	ApplyLLMConfig(mode string)
}

type SessionAdder interface {
	AddToSession(imageBase64 string, info json.RawMessage, source string, isPreview bool, presetID *int64) int64
}

type SDInterruptChecker interface {
	CheckSDInterrupted() error
}

type Service struct {
	llm         llm.Service
	sd          sd.Service
	db          *preset.DB
	kids        *kids.Manager
	cfg         *config.Config
	settings    SettingsApplier
	sessions    SessionAdder
	sdCheck     SDInterruptChecker
	emitter     EventEmitter
}

func New(
	llmSvc llm.Service,
	sdSvc sd.Service,
	db *preset.DB,
	kidsMgr *kids.Manager,
	cfg *config.Config,
	settings SettingsApplier,
	sessions SessionAdder,
	sdCheck SDInterruptChecker,
	emitter EventEmitter,
) *Service {
	return &Service{
		llm:      llmSvc,
		sd:       sdSvc,
		db:       db,
		kids:     kidsMgr,
		cfg:      cfg,
		settings: settings,
		sessions: sessions,
		sdCheck:  sdCheck,
		emitter:  emitter,
	}
}

type AnalyzePrompts struct {
	SystemPrompt string   `json:"system_prompt"`
	SinglePrompt string   `json:"single_prompt"`
	ChainPrompts []string `json:"chain_prompts"`
}

type GenerateImageResult struct {
	Image                   string          `json:"image"`
	Info                    json.RawMessage `json:"info"`
	IsPreview               bool            `json:"is_preview"`
	EffectivePrompt         string          `json:"effective_prompt"`
	EffectiveNegativePrompt string          `json:"effective_negative_prompt"`
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

type GenerateSDPromptResult struct {
	Prompt         string `json:"prompt"`
	NegativePrompt string `json:"negative_prompt"`
}

func (s *Service) GetDefaultAnalyzePrompts() *AnalyzePrompts {
	return &AnalyzePrompts{
		SystemPrompt: config.DefaultAnalyzeSystemPrompt,
		SinglePrompt: config.DefaultAnalyzePrompt,
		ChainPrompts: config.DefaultAnalyzeChainPrompts,
	}
}

func (s *Service) getAnalyzeChainPrompts() []string {
	prompts := make([]string, 4)
	for i := range prompts {
		key := "analyze_chain_" + strconv.Itoa(i+1)
		if v, err := s.db.GetSetting(key); err == nil && v != "" {
			prompts[i] = v
		} else if i < len(config.DefaultAnalyzeChainPrompts) {
			prompts[i] = config.DefaultAnalyzeChainPrompts[i]
		}
	}
	return prompts
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

func (s *Service) AnalyzeImage(imageBase64 string) (string, error) {
	if imageBase64 == "" {
		return "", fmt.Errorf("image is required")
	}
	if len(imageBase64) > 22*1024*1024 {
		return "", fmt.Errorf("image too large (max 16 MB)")
	}

	model := s.getAnalyzeModel()
	s.settings.ApplyLLMConfig("analyze")

	systemPrompt, _ := s.db.GetSetting("analyze_system_prompt")
	if systemPrompt == "" {
		systemPrompt = config.DefaultAnalyzeSystemPrompt
	}

	maxTokens := s.getMaxTokens()

	useChain := true
	if v, err := s.db.GetSetting("analyze_use_chain"); err == nil {
		useChain = v != "false"
	}

	if !useChain {
		prompt, _ := s.db.GetSetting("analyze_prompt")
		if prompt == "" {
			prompt = config.DefaultAnalyzePrompt
		}
		s.emitter.Emit("llm:status", map[string]string{"status": "thinking"})
		tags, err := s.llm.AnalyzeImage(model, systemPrompt+"\n\n"+prompt, imageBase64, maxTokens)
		if err != nil {
			s.emitter.Emit("llm:status", map[string]string{"status": "done"})
			return "", err
		}
		s.emitter.Emit("llm:status", map[string]string{"status": "done"})
		tags = s.kids.FilterOutput(tags)
		return tags, nil
	}

	chainPrompts := s.getAnalyzeChainPrompts()
	messages := []llm.Message{
		{Role: "system", Content: systemPrompt},
		{Role: "user", Content: []llm.ContentPart{
			{Type: "text", Text: chainPrompts[0]},
			{Type: "image_url", ImageURL: &llm.ImageURLPart{URL: "data:image/png;base64," + imageBase64}},
		}},
	}

	for i := 0; i < len(chainPrompts); i++ {
		s.emitter.Emit("llm:status", map[string]string{"status": "thinking"})
		resp, err := s.llm.ChatWithMessages(model, messages, 0.4, maxTokens)
		if err != nil {
			s.emitter.Emit("llm:status", map[string]string{"status": "done"})
			if i == 0 {
				return "", err
			}
			break
		}

		messages = append(messages, llm.Message{Role: "assistant", Content: resp})
		s.emitter.Emit("llm:status", map[string]string{"status": "done"})

		if i+1 < len(chainPrompts) {
			messages = append(messages, llm.Message{
				Role:    "user",
				Content: chainPrompts[i+1],
			})
		}

		s.emitter.Emit("analyze:step", i+1, len(chainPrompts))
	}

	lastResp := ""
	for j := len(messages) - 1; j >= 0; j-- {
		if messages[j].Role == "assistant" {
			if str, ok := messages[j].Content.(string); ok {
				lastResp = str
			}
			break
		}
	}

	tags := llm.CleanTags(lastResp)
	tags = s.kids.FilterOutput(tags)
	return tags, nil
}

func (s *Service) AnalyzeRemoveContext(imageBase64, maskBase64 string) (string, error) {
	imgData, err := base64.StdEncoding.DecodeString(imageBase64)
	if err != nil {
		return "", fmt.Errorf("decode image: %w", err)
	}
	maskData, err := base64.StdEncoding.DecodeString(maskBase64)
	if err != nil {
		return "", fmt.Errorf("decode mask: %w", err)
	}

	img, _, err := image.Decode(bytes.NewReader(imgData))
	if err != nil {
		return "", fmt.Errorf("parse image: %w", err)
	}
	mask, _, err := image.Decode(bytes.NewReader(maskData))
	if err != nil {
		return "", fmt.Errorf("parse mask: %w", err)
	}

	bounds := img.Bounds()
	overlay := image.NewRGBA(bounds)
	draw.Draw(overlay, bounds, img, bounds.Min, draw.Src)

	red := color.NRGBA{R: 255, G: 0, B: 0, A: 140}
	for y := bounds.Min.Y; y < bounds.Max.Y; y++ {
		for x := bounds.Min.X; x < bounds.Max.X; x++ {
			_, _, _, ma := mask.At(x, y).RGBA()
			if ma > 32768 {
				draw.Draw(overlay, image.Rect(x, y, x+1, y+1), &image.Uniform{red}, image.Point{}, draw.Over)
			}
		}
	}

	maxDim := 1024
	w, h := bounds.Dx(), bounds.Dy()
	if w > maxDim || h > maxDim {
		ratio := math.Min(float64(maxDim)/float64(w), float64(maxDim)/float64(h))
		w = int(float64(w) * ratio)
		h = int(float64(h) * ratio)
		if w < 1 {
			w = 1
		}
		if h < 1 {
			h = 1
		}
		scaled := image.NewRGBA(image.Rect(0, 0, w, h))
		xdraw.CatmullRom.Scale(scaled, scaled.Bounds(), overlay, bounds, draw.Over, nil)
		overlay = scaled
	}

	var buf bytes.Buffer
	if err := png.Encode(&buf, overlay); err != nil {
		return "", fmt.Errorf("encode overlay: %w", err)
	}
	overlayBase64 := base64.StdEncoding.EncodeToString(buf.Bytes())

	model := s.getAnalyzeModel()
	s.settings.ApplyLLMConfig("analyze")

	systemPrompt := "You are a vision model analyzing images for inpainting. The red overlay marks the area to remove. Describe what should fill that area to match the surrounding context seamlessly. Output ONLY comma-separated Stable Diffusion tags for the background/content that should replace the red area. No explanation, no extra text."
	userText := "Look at the red overlay area. What should fill this space to blend naturally with the surroundings? Output only SD tags."

	s.emitter.Emit("llm:status", map[string]string{"status": "thinking"})
	result, err := s.llm.ChatVision(model, systemPrompt, userText, overlayBase64, 0.3, 128)
	if err != nil {
		s.emitter.Emit("llm:status", map[string]string{"status": "done"})
		return "", fmt.Errorf("vision analysis failed: %w", err)
	}
	s.emitter.Emit("llm:status", map[string]string{"status": "done"})

	result = strings.TrimSpace(result)
	result = promptutil.TruncateRepetitive(result, 500)
	return result, nil
}

func (s *Service) GenerateRemoveObject(params GenerateFromImageParams) (*GenerateImageResult, error) {
	s.emitter.Emit("remove:stage", "analyzing")

	removeDesc, err := s.AnalyzeRemoveContext(params.ImageBase64, params.MaskBase64)
	if err != nil {
		return nil, fmt.Errorf("context analysis failed: %w", err)
	}

	s.emitter.Emit("remove:stage", "generating")

	removeNegative := "object, items, things, artifacts, distortion"
	if params.ExtraNegativePrompt != "" {
		removeNegative += ", " + params.ExtraNegativePrompt
	}

	var prompt, negativePrompt string
	var samplerName string
	var scheduler string
	var steps int
	var cfgScale float64
	var width, height int
	var seed *int64
	var clipSkip int
	var modelName, vae string
	var loras string

	imgData, _ := base64.StdEncoding.DecodeString(params.ImageBase64)
	if imgCfg, _, err := image.DecodeConfig(bytes.NewReader(imgData)); err == nil {
		width = imgCfg.Width
		height = imgCfg.Height
	}

	if params.PresetID > 0 {
		p, err := s.db.Get(params.PresetID)
		if err == nil {
			if p.Prompt != "" {
				prompt = p.Prompt + ", " + removeDesc + ", seamless background, clean, natural, consistent with surroundings"
			} else {
				prompt = removeDesc + ", seamless background, clean, natural, consistent with surroundings"
			}
			negativePrompt = p.NegativePrompt
			if negativePrompt != "" {
				negativePrompt += ", "
			}
			negativePrompt += removeNegative

			samplerName = p.Sampler
			if p.ScheduleType != "" {
				st := strings.ToUpper(p.ScheduleType[:1]) + p.ScheduleType[1:]
				samplerName = p.Sampler + " " + st
			}
			scheduler = p.ScheduleType
			steps = p.Steps
			cfgScale = p.CfgScale
			if p.Width > 0 {
				width = p.Width
			}
			if p.Height > 0 {
				height = p.Height
			}
			seed = p.Seed
			if p.ClipSkip != nil {
				clipSkip = *p.ClipSkip
			}
			modelName = p.ModelName
			vae = p.VAE
			loras = p.Loras
		}
	}

	if prompt == "" {
		prompt = removeDesc + ", seamless background, clean, natural, consistent with surroundings"
	}
	if negativePrompt == "" {
		negativePrompt = removeNegative
	}

	if loras != "" {
		var loraList []preset.LoRAEntry
		if json.Unmarshal([]byte(loras), &loraList) == nil {
			for _, l := range loraList {
				prompt += fmt.Sprintf(" <lora:%s:%g>", l.Name, l.Weight)
			}
		}
	}

	if samplerName == "" {
		samplerName = "Euler a"
	}
	if steps == 0 {
		steps = 20
	}
	if cfgScale == 0 {
		cfgScale = 7
	}
	if width == 0 {
		width = 512
	}
	if height == 0 {
		height = 512
	}

	if modelName != "" {
		_ = s.sd.SetModel(modelName)
	}
	if vae != "" {
		_ = s.sd.SetVAE(vae)
	}

	denoising := params.DenoisingStrength
	if denoising <= 0 {
		denoising = 0.75
	}
	maskBlur := params.MaskBlur
	if maskBlur <= 0 {
		maskBlur = 8
	}

	batchSize := 1
	batchCount := 1

	result, err := s.sd.Img2Img(sd.Img2ImgRequest{
		InitImages:            []string{params.ImageBase64},
		Prompt:                prompt,
		NegativePrompt:        negativePrompt,
		SamplerName:           samplerName,
		Scheduler:             scheduler,
		Steps:                 steps,
		CfgScale:              cfgScale,
		Width:                 width,
		Height:                height,
		Seed:                  seed,
		DenoisingStrength:     &denoising,
		ClipSkip:              &clipSkip,
		BatchSize:             &batchSize,
		BatchCount:            &batchCount,
		Mask:                  params.MaskBase64,
		MaskBlur:              maskBlur,
		InpaintingFill:        params.InpaintFill,
		InpaintFullRes:        params.InpaintFullRes,
		InpaintFullResPadding: 64,
		DoNotSaveImages:       true,
		DoNotSaveGrid:         true,
	})
	if err != nil {
		return nil, err
	}
	if len(result.Images) == 0 {
		if ierr := s.sdCheck.CheckSDInterrupted(); ierr != nil {
			return nil, ierr
		}
		return nil, fmt.Errorf("no image generated (remove object)")
	}

	img := &GenerateImageResult{
		Image:                   result.Images[0],
		Info:                    result.Info,
		EffectivePrompt:         prompt,
		EffectiveNegativePrompt: negativePrompt,
	}
	s.sessions.AddToSession(result.Images[0], result.Info, "generate", false, nil)
	return img, nil
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

func (s *Service) generateLLMPromptFromTags(p *preset.Preset, tags string, mode, extraNegative string) (*GenerateSDPromptResult, error) {
	sdPromptInstruction := s.getSDPromptInstruction()
	systemPrompt := s.kids.ApplySystemPrompt(sdPromptInstruction)
	maxTokens := s.getMaxTokens()
	generateModel := s.getGenerateModel()

	s.settings.ApplyLLMConfig("generate")

	systemPrompt += fmt.Sprintf(`

RESPONSE LENGTH: your response is limited to ~%d tokens. You MUST fit within this limit.`, maxTokens)

	userParts := []string{
		"BASE POSITIVE PROMPT: " + p.Prompt,
		"BASE NEGATIVE PROMPT: " + p.NegativePrompt,
		"USER DESCRIPTION (extracted from image): " + tags,
	}
	if mode == "inpaint" {
		userParts = []string{
			"MODE: inpaint — user wants to REPLACE the masked area with what they describe below.",
			"BASE POSITIVE PROMPT (style/quality reference only): " + p.Prompt,
			"BASE NEGATIVE PROMPT: " + p.NegativePrompt,
			"USER INSTRUCTION FOR MASKED AREA (THIS IS THE PRIMARY PROMPT): " + tags,
		}
	}
	if mode == "img2img" {
		userParts = []string{
			"MODE: img2img — user wants to TRANSFORM the image into the scene described below. Ignore what is currently in the image. Generate a NEW scene based on the user's description.",
			"BASE POSITIVE PROMPT (style/quality reference): " + p.Prompt,
			"BASE NEGATIVE PROMPT: " + p.NegativePrompt,
			"USER SCENE DESCRIPTION (THIS IS THE PRIMARY PROMPT — generate exactly this scene): " + tags,
		}
	}
	if extraNegative != "" {
		userParts = append(userParts, "USER NEGATIVE: "+extraNegative)
	}
	userMessage := strings.Join(userParts, "\n\n")

	s.emitter.Emit("llm:status", map[string]string{"status": "thinking"})
	raw, err := s.llm.GenerateSDPrompt(systemPrompt, userMessage, p.PresetType, generateModel, maxTokens)
	if err != nil {
		s.emitter.Emit("llm:status", map[string]string{"status": "done"})
		return nil, err
	}
	s.emitter.Emit("llm:status", map[string]string{"status": "done"})

	var promptResult GenerateSDPromptResult
	jsonRaw := promptutil.ExtractJSON(raw)
	if err := json.Unmarshal([]byte(jsonRaw), &promptResult); err != nil {
		promptResult = GenerateSDPromptResult{
			Prompt:         promptutil.TruncateRepetitive(raw, 1000),
			NegativePrompt: p.NegativePrompt,
		}
	}

	if promptutil.ContainsCyrillic(promptResult.Prompt) {
		promptResult.Prompt = promptutil.ExtractTagsFromRaw(raw)
	}
	if promptutil.ContainsCyrillic(promptResult.NegativePrompt) {
		promptResult.NegativePrompt = promptutil.ExtractNegativeFromRaw(raw)
	}

	extractEmbeddedNegative(&promptResult)
	promptResult.Prompt = promptutil.StripJunk(promptResult.Prompt)
	promptResult.Prompt = promptutil.TruncateRepetitive(promptResult.Prompt, 1000)
	promptResult.NegativePrompt = promptutil.StripJunk(promptResult.NegativePrompt)
	promptResult.NegativePrompt = promptutil.TruncateRepetitive(promptResult.NegativePrompt, 500)

	promptResult.Prompt = s.kids.FilterOutput(promptResult.Prompt)
	promptResult.NegativePrompt = s.kids.FilterOutput(promptResult.NegativePrompt)

	return &promptResult, nil
}

func (s *Service) GenerateFromImageCompound(params GenerateFromImageParams, tags string) (*GenerateImageResult, error) {
	cp, err := s.db.GetCompoundPreset(params.CompoundPresetID)
	if err != nil {
		return nil, fmt.Errorf("compound preset not found: %w", err)
	}
	if len(cp.Steps) == 0 {
		return nil, fmt.Errorf("compound preset has no steps")
	}

	firstPreset, err := s.db.Get(cp.Steps[0].PresetID)
	if err != nil {
		return nil, fmt.Errorf("step 1: preset not found: %w", err)
	}

	promptResult, err := s.generateLLMPromptFromTags(firstPreset, tags, params.Mode, params.ExtraNegativePrompt)
	if err != nil {
		return nil, err
	}

	var lastImage string
	var lastInfo json.RawMessage

	imgW, imgH := filebrowser.DecodeImageSize(params.ImageBase64)
	if imgW > 0 && imgH > 0 {
		imgW = imgW / 8 * 8
		imgH = imgH / 8 * 8
	}

	for stepIdx, step := range cp.Steps {
		p, err := s.db.Get(step.PresetID)
		if err != nil {
			return nil, fmt.Errorf("step %d: preset not found: %w", stepIdx+1, err)
		}

		s.emitter.Emit("fromimage:progress", map[string]any{
			"current": stepIdx + 1,
			"total":   len(cp.Steps),
			"status":  "generating",
		})

		prompt := p.Prompt
		if stepIdx == 0 {
			prompt = promptResult.Prompt
		}
		if p.Loras != "" {
			var loraList []preset.LoRAEntry
			if json.Unmarshal([]byte(p.Loras), &loraList) == nil {
				for _, l := range loraList {
					prompt += fmt.Sprintf(" <lora:%s:%g>", l.Name, l.Weight)
				}
			}
		}

		negativePrompt := promptResult.NegativePrompt
		if params.ExtraNegativePrompt != "" {
			negativePrompt += ", " + params.ExtraNegativePrompt
		}
		negativePrompt = s.kids.ApplyNegative(negativePrompt)

		if p.ModelName != "" {
			_ = s.sd.SetModel(p.ModelName)
		}
		if p.VAE != "" {
			_ = s.sd.SetVAE(p.VAE)
		}

		samplerName := p.Sampler
		if p.ScheduleType != "" {
			st := strings.ToUpper(p.ScheduleType[:1]) + p.ScheduleType[1:]
			samplerName = p.Sampler + " " + st
		}

		width := step.Width
		if width == 0 {
			width = p.Width
		}
		height := step.Height
		if height == 0 {
			height = p.Height
		}

		clipSkip := 1
		if p.ClipSkip != nil {
			clipSkip = *p.ClipSkip
		}
		batchSize := 1
		batchCount := 1

		if stepIdx == 0 && params.Mode == "img2img" {
			denoising := params.DenoisingStrength
			if denoising <= 0 {
				denoising = 0.5
			}
			w, h := imgW, imgH
			if w == 0 || h == 0 {
				w, h = width, height
			}
			result, err := s.sd.Img2Img(sd.Img2ImgRequest{
				InitImages:        []string{params.ImageBase64},
				Prompt:            prompt,
				NegativePrompt:    negativePrompt,
				SamplerName:       samplerName,
				Scheduler:         p.ScheduleType,
				Steps:             p.Steps,
				CfgScale:          p.CfgScale,
				Width:             w,
				Height:            h,
				Seed:              p.Seed,
				DenoisingStrength: &denoising,
				ClipSkip:          &clipSkip,
				BatchSize:         &batchSize,
				BatchCount:        &batchCount,
				DoNotSaveImages:   true,
				DoNotSaveGrid:     true,
			})
			if err != nil {
				if ierr := s.sdCheck.CheckSDInterrupted(); ierr != nil {
					return nil, ierr
				}
				return nil, fmt.Errorf("step %d (img2img): %w", stepIdx+1, err)
			}
			if len(result.Images) == 0 {
				if ierr := s.sdCheck.CheckSDInterrupted(); ierr != nil {
					return nil, ierr
				}
				return nil, fmt.Errorf("step %d: no image returned", stepIdx+1)
			}
			lastImage = result.Images[0]
			lastInfo = result.Info
		} else if stepIdx == 0 {
			result, err := s.sd.Txt2Img(sd.Txt2ImgRequest{
				Prompt:          prompt,
				NegativePrompt:  negativePrompt,
				SamplerName:     samplerName,
				Scheduler:       p.ScheduleType,
				Steps:           p.Steps,
				CfgScale:        p.CfgScale,
				Width:           width,
				Height:          height,
				Seed:            p.Seed,
				ClipSkip:        &clipSkip,
				BatchSize:       &batchSize,
				BatchCount:      &batchCount,
				DoNotSaveImages: true,
				DoNotSaveGrid:   true,
			})
			if err != nil {
				if ierr := s.sdCheck.CheckSDInterrupted(); ierr != nil {
					return nil, ierr
				}
				return nil, fmt.Errorf("step %d (txt2img): %w", stepIdx+1, err)
			}
			if len(result.Images) == 0 {
				if ierr := s.sdCheck.CheckSDInterrupted(); ierr != nil {
					return nil, ierr
				}
				return nil, fmt.Errorf("step %d: no image returned", stepIdx+1)
			}
			lastImage = result.Images[0]
			lastInfo = result.Info
		} else {
			denoising := step.DenoisingStrength
			if denoising <= 0 {
				denoising = 0.5
			}
			w, h := width, height
			if imgW > 0 && imgH > 0 && params.Mode == "img2img" {
				w, h = imgW, imgH
			}
			result, err := s.sd.Img2Img(sd.Img2ImgRequest{
				InitImages:        []string{lastImage},
				Prompt:            prompt,
				NegativePrompt:    negativePrompt,
				SamplerName:       samplerName,
				Scheduler:         p.ScheduleType,
				Steps:             p.Steps,
				CfgScale:          p.CfgScale,
				Width:             w,
				Height:            h,
				Seed:              p.Seed,
				DenoisingStrength: &denoising,
				ClipSkip:          &clipSkip,
				BatchSize:         &batchSize,
				BatchCount:        &batchCount,
				DoNotSaveImages:   true,
				DoNotSaveGrid:     true,
			})
			if err != nil {
				if ierr := s.sdCheck.CheckSDInterrupted(); ierr != nil {
					return nil, ierr
				}
				return nil, fmt.Errorf("step %d (img2img): %w", stepIdx+1, err)
			}
			if len(result.Images) == 0 {
				if ierr := s.sdCheck.CheckSDInterrupted(); ierr != nil {
					return nil, ierr
				}
				return nil, fmt.Errorf("step %d: no image returned", stepIdx+1)
			}
			lastImage = result.Images[0]
			lastInfo = result.Info
		}
	}

	s.emitter.Emit("fromimage:progress", map[string]any{
		"current": len(cp.Steps),
		"total":   len(cp.Steps),
		"status":  "done",
	})

	img := &GenerateImageResult{
		Image:                   lastImage,
		Info:                    lastInfo,
		EffectivePrompt:         promptResult.Prompt,
		EffectiveNegativePrompt: promptResult.NegativePrompt,
	}
	s.sessions.AddToSession(lastImage, lastInfo, "compound-from-image", false, nil)
	return img, nil
}
