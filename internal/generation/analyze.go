package generation

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
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	xdraw "golang.org/x/image/draw"

	"go-sd/internal/config"
	"go-sd/internal/filebrowser"
	"go-sd/internal/llm"
	"go-sd/internal/preset"
	"go-sd/internal/promptutil"
	"go-sd/internal/sd"
)

// --- AnalyzeImage ---

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

func (s *Service) GetDefaultAnalyzePrompts() *AnalyzePrompts {
	return &AnalyzePrompts{
		SystemPrompt: config.DefaultAnalyzeSystemPrompt,
		SinglePrompt: config.DefaultAnalyzePrompt,
		ChainPrompts: config.DefaultAnalyzeChainPrompts,
	}
}

// --- AnalyzeRemoveContext ---

func (s *Service) analyzeRemoveContext(imageBase64, maskBase64 string) (string, error) {
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

	systemPrompt := "You analyze images for object removal via inpainting. The red overlay marks the area to REMOVE. Your task: describe ONLY the background/surface that should fill the red area to blend seamlessly. DO NOT describe objects, people, or the scene. Output ONLY short comma-separated tags for texture/color/material of the surrounding area.\n\nExample outputs:\n- gray concrete wall, subtle cracks, warm lighting\n- blue sky, soft white clouds\n- green grass, natural texture, sunlight\n- wooden floor, planks, warm brown tone\n\nNO sentences. NO explanation. NO scene description. ONLY tags."
	userText := "What background/surface should replace the red area? Tags only."

	s.emitter.Emit("llm:status", map[string]string{"status": "thinking"})
	result, err := s.llm.ChatVision(model, systemPrompt, userText, overlayBase64, 0.2, 64)
	if err != nil {
		s.emitter.Emit("llm:status", map[string]string{"status": "done"})
		return "", fmt.Errorf("vision analysis failed: %w", err)
	}
	s.emitter.Emit("llm:status", map[string]string{"status": "done"})

	result = llm.CleanTags(result)
	result = promptutil.StripJunk(result)
	result = promptutil.TruncateRepetitive(result, 200)
	return result, nil
}

// --- generateRemoveObject ---

func (s *Service) generateRemoveObject(params GenerateFromImageParams) (*GenerateImageResult, error) {
	s.emitter.Emit("remove:stage", "analyzing")

	removeDesc, err := s.analyzeRemoveContext(params.ImageBase64, params.MaskBase64)
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

	imgData, _ := base64.StdEncoding.DecodeString(params.ImageBase64)
	if imgCfg, _, err := image.DecodeConfig(bytes.NewReader(imgData)); err == nil {
		width = imgCfg.Width
		height = imgCfg.Height
	}

	if removeDesc != "" {
		prompt = removeDesc + ", seamless blend, clean edges"
	} else {
		prompt = "seamless background, clean, natural, consistent with surroundings"
	}
	negativePrompt = removeNegative

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

	denoising := params.DenoisingStrength
	if denoising <= 0 {
		denoising = 0.6
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
		if ierr := s.checkSDInterrupted(); ierr != nil {
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
	s.sessions.AddToSession(result.Images[0], result.Info, "compound", false, nil)
	return img, nil
}

// --- generateLLMPromptFromTags ---

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

// --- GenerateFromImage ---

func (s *Service) GenerateFromImage(params GenerateFromImageParams) (*GenerateImageResult, error) {
	s.StartSDPolling()
	defer s.StopSDPolling()
	if params.ImageBase64 == "" {
		return nil, fmt.Errorf("image is required")
	}
	if len(params.ImageBase64) > 22*1024*1024 {
		return nil, fmt.Errorf("image too large (max 16 MB)")
	}
	if params.GenMode != "preset" && params.GenMode != "compound" {
		return nil, fmt.Errorf("gen_mode must be preset or compound")
	}
	if !params.RemoveObject {
		if params.GenMode == "preset" && params.PresetID <= 0 {
			return nil, fmt.Errorf("preset is required")
		}
		if params.GenMode == "compound" && params.CompoundPresetID <= 0 {
			return nil, fmt.Errorf("compound preset is required")
		}
	}
	if params.Mode != "txt2img" && params.Mode != "img2img" && params.Mode != "inpaint" {
		return nil, fmt.Errorf("mode must be txt2img, img2img or inpaint")
	}
	if params.Mode == "inpaint" && params.MaskBase64 == "" {
		return nil, fmt.Errorf("mask is required for inpaint mode")
	}
	if params.DenoisingStrength <= 0 {
		params.DenoisingStrength = 0.5
	}
	if params.DenoisingStrength > 1.0 {
		params.DenoisingStrength = 1.0
	}

	var filterErr error
	params.ExtraNegativePrompt, filterErr = s.kids.FilterInput(params.ExtraNegativePrompt)
	if filterErr != nil {
		return nil, filterErr
	}

	if params.RemoveObject {
		return s.generateRemoveObject(params)
	}

	tags := params.Tags

	tags, filterErr = s.kids.FilterInput(tags)
	if filterErr != nil {
		return nil, filterErr
	}

	if params.GenMode == "compound" {
		return s.generateFromImageCompound(params, tags)
	}

	p, err := s.db.Get(params.PresetID)
	if err != nil {
		return nil, fmt.Errorf("preset not found: %w", err)
	}

	var prompt, negativePrompt string
	if tags == "" {
		prompt = p.Prompt
		negativePrompt = p.NegativePrompt
		if params.ExtraNegativePrompt != "" {
			negativePrompt += ", " + params.ExtraNegativePrompt
		}
		negativePrompt = s.kids.ApplyNegative(negativePrompt)
	} else {
		promptResult, err := s.generateLLMPromptFromTags(p, tags, params.Mode, params.ExtraNegativePrompt)
		if err != nil {
			return nil, err
		}

		prompt = promptResult.Prompt
		if params.Mode == "inpaint" {
			prompt += ", " + tags
		}
		negativePrompt = promptResult.NegativePrompt
		if params.ExtraNegativePrompt != "" {
			negativePrompt += ", " + params.ExtraNegativePrompt
		}
		negativePrompt = s.kids.ApplyNegative(negativePrompt)
	}

	prompt = appendLorasToPrompt(prompt, p.Loras)

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

	if params.Mode == "img2img" || params.Mode == "inpaint" {
		imgW, imgH := filebrowser.DecodeImageSize(params.ImageBase64)
		if imgW > 0 && imgH > 0 {
			imgW = imgW / 8 * 8
			imgH = imgH / 8 * 8
		} else {
			imgW = p.Width
			imgH = p.Height
		}
		denoising := params.DenoisingStrength
		if denoising <= 0 {
			denoising = 0.5
		}
		maskBlur := params.MaskBlur
		if maskBlur <= 0 {
			maskBlur = 4
		}
		result, err := s.sd.Img2Img(sd.Img2ImgRequest{
			InitImages:            []string{params.ImageBase64},
			Prompt:                prompt,
			NegativePrompt:        negativePrompt,
			SamplerName:           samplerName,
			Scheduler:             p.ScheduleType,
			Steps:                 p.Steps,
			CfgScale:              p.CfgScale,
			Width:                 imgW,
			Height:                imgH,
			Seed:                  p.Seed,
			DenoisingStrength:     &denoising,
			ClipSkip:              &clipSkip,
			BatchSize:             &batchSize,
			BatchCount:            &batchCount,
			Mask:                  params.MaskBase64,
			MaskBlur:              maskBlur,
			InpaintingFill:        params.InpaintFill,
			InpaintFullRes:        params.InpaintFullRes,
			InpaintFullResPadding: 32,
			DoNotSaveImages:       true,
			DoNotSaveGrid:         true,
		})
		if err != nil {
			return nil, err
		}
		if len(result.Images) == 0 {
			if ierr := s.checkSDInterrupted(); ierr != nil {
				return nil, ierr
			}
			return nil, fmt.Errorf("no image generated (%s)", params.Mode)
		}
		img := &GenerateImageResult{
			Image:                   result.Images[0],
			Info:                    result.Info,
			EffectivePrompt:         prompt,
			EffectiveNegativePrompt: negativePrompt,
		}
		s.sessions.AddToSession(result.Images[0], result.Info, "from-image", false, nil)
		return img, nil
	}

	width := p.Width
	height := p.Height
	hiresFix := p.HiresFix

	isPreview := false
	if pw, ph, preview := s.getPreviewDimensions(p.Width, p.Height); preview {
		isPreview = true
		width = pw
		height = ph
		hiresFix = nil
	}

	denoisingStrength := p.DenoisingStrength
	if denoisingStrength == nil && p.HiresFix != nil && *p.HiresFix {
		ds := 0.5
		if p.HiresDenoisingStrength != nil {
			ds = *p.HiresDenoisingStrength
		}
		denoisingStrength = &ds
	}

	result, err := s.sd.Txt2Img(sd.Txt2ImgRequest{
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
		if ierr := s.checkSDInterrupted(); ierr != nil {
			return nil, ierr
		}
		return nil, fmt.Errorf("no image generated (txt2img)")
	}

	img := &GenerateImageResult{
		Image:                   result.Images[0],
		Parameters:              result.Parameters,
		Info:                    result.Info,
		IsPreview:               isPreview,
		EffectivePrompt:         prompt,
		EffectiveNegativePrompt: negativePrompt,
	}
	s.sessions.AddToSession(result.Images[0], result.Info, "generate", isPreview, nil)
	return img, nil
}

// --- generateFromImageCompound ---

func (s *Service) generateFromImageCompound(params GenerateFromImageParams, tags string) (*GenerateImageResult, error) {
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
		prompt = appendLorasToPrompt(prompt, p.Loras)

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

		samplerName := buildSamplerName(p.Sampler, p.ScheduleType)

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
				if ierr := s.checkSDInterrupted(); ierr != nil {
					return nil, ierr
				}
				return nil, fmt.Errorf("step %d (img2img): %w", stepIdx+1, err)
			}
			if len(result.Images) == 0 {
				if ierr := s.checkSDInterrupted(); ierr != nil {
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
				if ierr := s.checkSDInterrupted(); ierr != nil {
					return nil, ierr
				}
				return nil, fmt.Errorf("step %d (txt2img): %w", stepIdx+1, err)
			}
			if len(result.Images) == 0 {
				if ierr := s.checkSDInterrupted(); ierr != nil {
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
				if ierr := s.checkSDInterrupted(); ierr != nil {
					return nil, ierr
				}
				return nil, fmt.Errorf("step %d (img2img): %w", stepIdx+1, err)
			}
			if len(result.Images) == 0 {
				if ierr := s.checkSDInterrupted(); ierr != nil {
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

// --- GenerateCompoundImage ---

func (s *Service) GenerateCompoundImage(params GenerateCompoundImageParams) (*GenerateImageResult, error) {
	s.StartSDPolling()
	defer s.StopSDPolling()
	cp, err := s.db.GetCompoundPreset(params.CompoundPresetID)
	if err != nil {
		return nil, fmt.Errorf("compound preset not found: %w", err)
	}
	if len(cp.Steps) == 0 {
		return nil, fmt.Errorf("compound preset has no steps")
	}

	var lastImage string
	var lastInfo json.RawMessage

	for stepIdx, step := range cp.Steps {
		p, err := s.db.Get(step.PresetID)
		if err != nil {
			return nil, fmt.Errorf("step %d: preset not found: %w", stepIdx+1, err)
		}

		s.emitter.Emit("compound:progress", map[string]any{
			"current": stepIdx + 1,
			"total":   len(cp.Steps),
			"status":  "generating",
			"step":    stepIdx + 1,
		})

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

		if stepIdx == 0 {
			batchSize := 1
			batchCount := 1
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
				if ierr := s.checkSDInterrupted(); ierr != nil {
					return nil, ierr
				}
				return nil, fmt.Errorf("step %d (txt2img): %w", stepIdx+1, err)
			}
			if len(result.Images) == 0 {
				if ierr := s.checkSDInterrupted(); ierr != nil {
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
			batchSize := 1
			batchCount := 1
			result, err := s.sd.Img2Img(sd.Img2ImgRequest{
				InitImages:        []string{lastImage},
				Prompt:            prompt,
				NegativePrompt:    negativePrompt,
				SamplerName:       samplerName,
				Scheduler:         p.ScheduleType,
				Steps:             p.Steps,
				CfgScale:          p.CfgScale,
				Width:             width,
				Height:            height,
				Seed:              p.Seed,
				DenoisingStrength: &denoising,
				ClipSkip:          &clipSkip,
				BatchSize:         &batchSize,
				BatchCount:        &batchCount,
				DoNotSaveImages:   true,
				DoNotSaveGrid:     true,
			})
			if err != nil {
				if ierr := s.checkSDInterrupted(); ierr != nil {
					return nil, ierr
				}
				return nil, fmt.Errorf("step %d (img2img): %w", stepIdx+1, err)
			}
			if len(result.Images) == 0 {
				if ierr := s.checkSDInterrupted(); ierr != nil {
					return nil, ierr
				}
				return nil, fmt.Errorf("step %d: no image returned", stepIdx+1)
			}
			lastImage = result.Images[0]
			lastInfo = result.Info
		}
	}

	s.emitter.Emit("compound:progress", map[string]any{
		"current": len(cp.Steps),
		"total":   len(cp.Steps),
		"status":  "done",
	})

	img := &GenerateImageResult{
		Image:                   lastImage,
		Info:                    lastInfo,
		IsPreview:               false,
		EffectivePrompt:         "",
		EffectiveNegativePrompt: "",
	}
	s.sessions.AddToSession(lastImage, lastInfo, "compound", false, nil)
	return img, nil
}

// --- BatchCompoundGenerate ---

func (s *Service) BatchCompoundGenerate(params BatchCompoundGenerateParams) error {
	s.StartSDPolling()
	defer s.StopSDPolling()
	if params.Count <= 0 || params.Count > 100 {
		return fmt.Errorf("count must be between 1 and 100")
	}
	if params.OutputFolder == "" {
		return fmt.Errorf("output folder is required")
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

	cp, err := s.db.GetCompoundPreset(params.CompoundPresetID)
	if err != nil {
		return fmt.Errorf("compound preset not found: %w", err)
	}
	if len(cp.Steps) == 0 {
		return fmt.Errorf("compound preset has no steps")
	}

	for batchIdx := 0; batchIdx < params.Count; batchIdx++ {
		s.emitter.Emit("batch:progress", map[string]any{
			"current": batchIdx + 1,
			"total":   params.Count,
			"status":  "generating",
		})

		var lastImage string

		for stepIdx, step := range cp.Steps {
			p, err := s.db.Get(step.PresetID)
			if err != nil {
				return fmt.Errorf("step %d: preset not found: %w", stepIdx+1, err)
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
			var filterErr error
			prompt, filterErr = s.kids.FilterInput(prompt)
			if filterErr != nil {
				return fmt.Errorf("step %d: %w", stepIdx+1, filterErr)
			}
			negativePrompt, filterErr = s.kids.FilterInput(negativePrompt)
			if filterErr != nil {
				return fmt.Errorf("step %d: %w", stepIdx+1, filterErr)
			}
			negativePrompt = s.kids.ApplyNegative(negativePrompt)

			if p.ModelName != "" {
				_ = s.sd.SetModel(p.ModelName)
			}
			if p.VAE != "" {
				_ = s.sd.SetVAE(p.VAE)
			}

			samplerName := buildSamplerName(p.Sampler, p.ScheduleType)

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

			if stepIdx == 0 {
				batchSize := 1
				batchCount := 1
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
					if ierr := s.checkSDInterrupted(); ierr != nil {
						return ierr
					}
					return fmt.Errorf("batch %d, step %d (txt2img): %w", batchIdx+1, stepIdx+1, err)
				}
				if len(result.Images) == 0 {
					if ierr := s.checkSDInterrupted(); ierr != nil {
						return ierr
					}
					return fmt.Errorf("batch %d, step %d: no image returned", batchIdx+1, stepIdx+1)
				}
				lastImage = result.Images[0]
			} else {
				denoising := step.DenoisingStrength
				if denoising <= 0 {
					denoising = 0.5
				}
				batchSize := 1
				batchCount := 1
				result, err := s.sd.Img2Img(sd.Img2ImgRequest{
					InitImages:        []string{lastImage},
					Prompt:            prompt,
					NegativePrompt:    negativePrompt,
					SamplerName:       samplerName,
					Scheduler:         p.ScheduleType,
					Steps:             p.Steps,
					CfgScale:          p.CfgScale,
					Width:             width,
					Height:            height,
					Seed:              p.Seed,
					DenoisingStrength: &denoising,
					ClipSkip:          &clipSkip,
					BatchSize:         &batchSize,
					BatchCount:        &batchCount,
					DoNotSaveImages:   true,
					DoNotSaveGrid:     true,
				})
				if err != nil {
					if ierr := s.checkSDInterrupted(); ierr != nil {
						return ierr
					}
					return fmt.Errorf("batch %d, step %d (img2img): %w", batchIdx+1, stepIdx+1, err)
				}
				if len(result.Images) == 0 {
					if ierr := s.checkSDInterrupted(); ierr != nil {
						return ierr
					}
					return fmt.Errorf("batch %d, step %d: no image returned", batchIdx+1, stepIdx+1)
				}
				lastImage = result.Images[0]
			}
		}

		imgData, err := base64.StdEncoding.DecodeString(lastImage)
		if err != nil {
			return fmt.Errorf("batch %d: decode image: %w", batchIdx+1, err)
		}

		filename := fmt.Sprintf("compound_%s_%d_%d.png", cp.Name, time.Now().Unix(), batchIdx+1)
		filePath := filepath.Join(params.OutputFolder, filename)
		if err := os.WriteFile(filePath, imgData, 0644); err != nil {
			return fmt.Errorf("batch %d: save file: %w", batchIdx+1, err)
		}

		s.emitter.Emit("batch:progress", map[string]any{
			"current":   batchIdx + 1,
			"total":     params.Count,
			"file_path": filePath,
			"status":    "generating",
		})
	}

	s.emitter.Emit("batch:progress", map[string]any{
		"current": params.Count,
		"total":   params.Count,
		"status":  "done",
	})
	return nil
}

// --- TestCompoundGenerate ---

func (s *Service) TestCompoundGenerate(params TestCompoundGenerateParams) ([]TestGenerateResultItem, error) {
	s.StartSDPolling()
	defer s.StopSDPolling()
	if len(params.SelectedIDs) == 0 {
		return nil, fmt.Errorf("select at least one compound preset")
	}
	if len(params.SelectedIDs) > 20 {
		return nil, fmt.Errorf("maximum 20 compound presets at once")
	}
	if params.Prompt == "" {
		return nil, fmt.Errorf("prompt is required")
	}

	totalItems := len(params.SelectedIDs)
	results := make([]TestGenerateResultItem, 0, totalItems)

	for idx, compoundID := range params.SelectedIDs {
		s.emitter.Emit("test:progress", map[string]any{
			"current": idx + 1,
			"total":   totalItems,
			"status":  "generating",
		})

		item := TestGenerateResultItem{}

		cp, err := s.db.GetCompoundPreset(compoundID)
		if err != nil {
			item.Error = fmt.Sprintf("compound preset not found: %v", err)
			item.Name = fmt.Sprintf("Compound #%d", compoundID)
			results = append(results, item)
			continue
		}
		item.Name = cp.Name

		if len(cp.Steps) == 0 {
			item.Error = "no steps in compound preset"
			results = append(results, item)
			continue
		}

		var lastImage string

		for stepIdx, step := range cp.Steps {
			p, err := s.db.Get(step.PresetID)
			if err != nil {
				item.Error = fmt.Sprintf("step %d: preset not found", stepIdx+1)
				break
			}

			prompt := params.Prompt
			prompt, filterErr := s.kids.FilterInput(prompt)
			if filterErr != nil {
				item.Error = filterErr.Error()
				break
			}
			prompt = appendLorasToPrompt(prompt, p.Loras)

			negPrompt := params.NegativePrompt
			if p.NegativePrompt != "" {
				if negPrompt != "" {
					negPrompt = p.NegativePrompt + ", " + negPrompt
				} else {
					negPrompt = p.NegativePrompt
				}
			}
			negPrompt = s.kids.ApplyNegative(negPrompt)

			if p.ModelName != "" {
				_ = s.sd.SetModel(p.ModelName)
			}
			if p.VAE != "" {
				_ = s.sd.SetVAE(p.VAE)
			}

			samplerName := buildSamplerName(p.Sampler, p.ScheduleType)

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

			if stepIdx == 0 {
				result, err := s.sd.Txt2Img(sd.Txt2ImgRequest{
					Prompt:          prompt,
					NegativePrompt:  negPrompt,
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
					item.Error = fmt.Sprintf("step %d: %v", stepIdx+1, err)
					break
				}
				if len(result.Images) == 0 {
					item.Error = fmt.Sprintf("step %d: no image", stepIdx+1)
					break
				}
				lastImage = result.Images[0]
			} else {
				denoising := step.DenoisingStrength
				if denoising <= 0 {
					denoising = 0.5
				}
				result, err := s.sd.Img2Img(sd.Img2ImgRequest{
					InitImages:        []string{lastImage},
					Prompt:            prompt,
					NegativePrompt:    negPrompt,
					SamplerName:       samplerName,
					Scheduler:         p.ScheduleType,
					Steps:             p.Steps,
					CfgScale:          p.CfgScale,
					Width:             width,
					Height:            height,
					Seed:              p.Seed,
					DenoisingStrength: &denoising,
					ClipSkip:          &clipSkip,
					BatchSize:         &batchSize,
					BatchCount:        &batchCount,
					DoNotSaveImages:   true,
					DoNotSaveGrid:     true,
				})
				if err != nil {
					item.Error = fmt.Sprintf("step %d: %v", stepIdx+1, err)
					break
				}
				if len(result.Images) == 0 {
					item.Error = fmt.Sprintf("step %d: no image", stepIdx+1)
					break
				}
				lastImage = result.Images[0]
			}
		}

		if item.Error == "" {
			item.Image = lastImage
		}
		item.Sampler = ""
		item.ScheduleType = ""
		item.CfgScale = 0
		item.ModelName = ""

		results = append(results, item)

		s.emitter.Emit("test:progress", map[string]any{
			"current": idx + 1,
			"total":   totalItems,
			"status":  "done",
		})
	}

	return results, nil
}
