package compositor

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"image"
	"image/draw"
	"image/png"
	"math/rand"
	"strings"
	"unicode/utf8"

	"go-sd/internal/preset"
	"go-sd/internal/promptutil"
	"go-sd/internal/rembg"
	"go-sd/internal/sd"
)

type SDGenerator interface {
	Txt2Img(req sd.Txt2ImgRequest) (*sd.Txt2ImgResponse, error)
	Img2Img(req sd.Img2ImgRequest) (*sd.Txt2ImgResponse, error)
	SetModel(modelName string) error
	SetVAE(vaeName string) error
}

type RembgClient interface {
	RemoveBackgroundBase64(base64Image string) (string, error)
}

type PresetGetter interface {
	Get(id int64) (*preset.Preset, error)
	GetResolution(id int64) (*preset.Resolution, error)
	GetHiresProfile(id int64) (*preset.HiresProfile, error)
}

type ProgressEmitter func(progress MultiPassProgress)

type Compositor struct {
	sd      SDGenerator
	rembg   RembgClient
	presets PresetGetter
	emit    ProgressEmitter
}

func New(sdClient SDGenerator, rembgClient RembgClient, presetDB PresetGetter, emit ProgressEmitter) *Compositor {
	return &Compositor{
		sd:      sdClient,
		rembg:   rembgClient,
		presets: presetDB,
		emit:    emit,
	}
}

func (c *Compositor) GenerateScene(scene Scene) (*MultiPassResult, error) {
	if len(scene.Characters) == 0 {
		return nil, fmt.Errorf("scene must have at least one character")
	}
	if len(scene.Characters) > 10 {
		return nil, fmt.Errorf("too many characters (max 10)")
	}
	if scene.Width < 64 || scene.Width > 2048 || scene.Height < 64 || scene.Height > 2048 {
		return nil, fmt.Errorf("invalid dimensions %dx%d (must be 64-2048)", scene.Width, scene.Height)
	}

	if scene.ResolutionID != nil && *scene.ResolutionID > 0 {
		if r, err := c.presets.GetResolution(*scene.ResolutionID); err == nil {
			scene.Width = r.Width
			scene.Height = r.Height
		}
	}

	hiresEnabled := false
	var hiresUpscale float64
	var hiresDenoise float64
	hiresUpscaler := ""
	if scene.HiresProfileID != nil && *scene.HiresProfileID > 0 {
		if h, err := c.presets.GetHiresProfile(*scene.HiresProfileID); err == nil {
			hiresEnabled = true
			hiresUpscale = h.Upscale
			hiresDenoise = h.DenoisingStrength
			hiresUpscaler = h.Upscaler
			if hiresUpscaler == "" {
				hiresUpscaler = "R-ESRGAN 4x+"
			}
		}
	}

	for _, ch := range scene.Characters {
		if len(ch.Prompt) > 2000 {
			return nil, fmt.Errorf("character %q prompt too long (max 2000 chars)", ch.Name)
		}
	}

	p, err := c.presets.Get(scene.PresetID)
	if err != nil {
		return nil, fmt.Errorf("preset not found: %w", err)
	}

	if p.ModelName != "" {
		_ = c.sd.SetModel(p.ModelName)
	}
	if p.VAE != "" {
		_ = c.sd.SetVAE(p.VAE)
	} else {
		_ = c.sd.SetVAE("Automatic")
	}

	loraSuffix := ""
	if p.Loras != "" {
		var loras []preset.LoRAEntry
		if err := json.Unmarshal([]byte(p.Loras), &loras); err == nil {
			for _, l := range loras {
				loraSuffix += fmt.Sprintf(" <lora:%s:%g>", l.Name, l.Weight)
			}
		}
	}

	if c.emit != nil {
		c.emit(MultiPassProgress{Step: "background"})
	}

	stylePrefix := extractStyleTags(p.Prompt)
	bgPrompt := stylePrefix + ", " + scene.BackgroundPrompt + ", no people, no characters, empty scene, landscape" + loraSuffix
	bgNeg := scene.NegativePrompt
	if bgNeg == "" {
		bgNeg = p.NegativePrompt
	} else if p.NegativePrompt != "" {
		bgNeg = bgNeg + ", " + p.NegativePrompt
	}

	bgReq := buildTxt2ImgRequest(p, bgPrompt, bgNeg, scene.Width, scene.Height)
	if hiresEnabled {
		bgReq.HiresFix = &hiresEnabled
		bgReq.HiresUpscale = &hiresUpscale
		bgReq.HiresDenoisingStrength = &hiresDenoise
		bgReq.HiresUpscaler = hiresUpscaler
	}
	bgResp, err := c.sd.Txt2Img(bgReq)
	if err != nil {
		return nil, fmt.Errorf("background generation failed: %w", err)
	}
	if len(bgResp.Images) == 0 {
		return nil, fmt.Errorf("no background image generated")
	}

	bgDecoded, err := decodeBase64PNG(bgResp.Images[0])
	if err != nil {
		return nil, fmt.Errorf("decode background: %w", err)
	}
	bgImage := imageToRGBA(bgDecoded)

	resultImage := image.NewRGBA(bgImage.Bounds())
	draw.Draw(resultImage, bgImage.Bounds(), bgImage, bgImage.Bounds().Min, draw.Src)

	result := &MultiPassResult{
		Background: bgResp.Images[0],
	}

	for i, char := range scene.Characters {
		if c.emit != nil {
			c.emit(MultiPassProgress{
				Step:      "character",
				Character: i + 1,
				Total:     len(scene.Characters),
			})
		}

		stylePrompt := extractStyleTags(p.Prompt)
		charPrompt := stylePrompt + ", " + char.Prompt + ", plain white background" + poseSuffix(char.Prompt) + loraSuffix
		charNeg := scene.NegativePrompt
		if charNeg == "" {
			charNeg = p.NegativePrompt
		} else if p.NegativePrompt != "" {
			charNeg = charNeg + ", " + p.NegativePrompt
		}

		for j, other := range scene.Characters {
			if j != i {
				charNeg += ", " + other.Name
			}
		}

		charW, charH := 512, 768
		if scene.Width > 512 {
			charW = 640
			charH = 960
		}

		charSeed := rand.Int63()
		charReq := buildTxt2ImgRequestWithSeed(p, charPrompt, charNeg, charW, charH, &charSeed)
		charResp, err := c.sd.Txt2Img(charReq)
		if err != nil {
			return nil, fmt.Errorf("character %q generation failed: %w", char.Name, err)
		}
		if len(charResp.Images) == 0 {
			return nil, fmt.Errorf("no image for character %q", char.Name)
		}

		var charRGBA *image.RGBA
		if c.rembg != nil {
			if c.emit != nil {
				c.emit(MultiPassProgress{
					Step:      "rembg",
					Character: i + 1,
					Total:     len(scene.Characters),
				})
			}

			rembgBase64, err := c.rembg.RemoveBackgroundBase64(charResp.Images[0])
			if err != nil {
				return nil, fmt.Errorf("rembg failed for %q: %w", char.Name, err)
			}

			rembgDecoded, err := decodeBase64PNG(rembgBase64)
			if err != nil {
				return nil, fmt.Errorf("decode rembg result for %q: %w", char.Name, err)
			}
			charRGBA = imageToRGBA(rembgDecoded)
		} else {
			charDecoded, err := decodeBase64PNG(charResp.Images[0])
			if err != nil {
				return nil, fmt.Errorf("decode character %q: %w", char.Name, err)
			}
			charRGBA = RemoveWhiteBackground(charDecoded)
		}

		scale := char.Scale
		if scale <= 0 {
			scale = 0.4
		}
		composited := CompositeOver(resultImage, charRGBA, char.Position, scale)
		resultImage = composited

		result.Characters = append(result.Characters, struct {
			Name  string `json:"name"`
			Image string `json:"image,omitempty"`
		}{Name: char.Name, Image: charResp.Images[0]})
	}

	if c.emit != nil {
		c.emit(MultiPassProgress{Step: "refine"})
	}

	compositeBase64, err := encodeImageToBase64(resultImage)
	if err != nil {
		return nil, fmt.Errorf("encode composite: %w", err)
	}

	refinePrompt := buildRefinePrompt(scene, stylePrefix)
	refineNeg := p.NegativePrompt
	if scene.NegativePrompt != "" {
		refineNeg = scene.NegativePrompt + ", " + p.NegativePrompt
	}

	denoise := 0.35
	if scene.RefineDenoise > 0 {
		denoise = scene.RefineDenoise
	}
	if denoise < 0.1 {
		denoise = 0.1
	}
	if denoise > 0.8 {
		denoise = 0.8
	}

	refineSteps := p.Steps
	if refineSteps > 15 {
		refineSteps = 15
	}

	refineReq := sd.Img2ImgRequest{
		InitImages:        []string{compositeBase64},
		Prompt:            refinePrompt,
		NegativePrompt:    refineNeg,
		SamplerName:       promptutil.BuildSamplerName(p.Sampler, p.ScheduleType),
		Scheduler:         p.ScheduleType,
		Steps:             refineSteps,
		CfgScale:          p.CfgScale,
		Width:             scene.Width,
		Height:            scene.Height,
		DenoisingStrength: &denoise,
		Seed:              p.Seed,
		ClipSkip:          buildClipSkip(p),
		DoNotSaveImages:   true,
		DoNotSaveGrid:     true,
	}

	refineResp, err := c.sd.Img2Img(refineReq)
	if err != nil {
		return nil, fmt.Errorf("refinement pass failed: %w", err)
	}
	if len(refineResp.Images) == 0 {
		return nil, fmt.Errorf("refinement pass returned no images")
	}

	result.Image = refineResp.Images[0]

	if c.emit != nil {
		c.emit(MultiPassProgress{Step: "done"})
	}

	return result, nil
}

func positionLabel(x float64) string {
	if x < 0.33 {
		return "on the left"
	}
	if x > 0.66 {
		return "on the right"
	}
	return "in the center"
}

func truncatePrompt(s string, maxRunes int) string {
	if utf8.RuneCountInString(s) <= maxRunes {
		return s
	}
	runes := []rune(s)
	return string(runes[:maxRunes])
}

func buildRefinePrompt(scene Scene, stylePrefix string) string {
	if scene.RefinePrompt != "" {
		return scene.RefinePrompt
	}

	parts := []string{stylePrefix, scene.BackgroundPrompt}
	for _, ch := range scene.Characters {
		label := positionLabel(ch.Position.X)
		truncated := truncatePrompt(ch.Prompt, 80)
		parts = append(parts, fmt.Sprintf("character %s %s: %s", label, ch.Name, truncated))
	}
	return strings.Join(parts, ", ")
}

func DecomposeSceneFromJSON(jsonStr string) (*Scene, error) {
	jsonStr = strings.TrimSpace(jsonStr)
	jsonStr = extractJSON(jsonStr)

	var scene Scene
	if err := json.Unmarshal([]byte(jsonStr), &scene); err != nil {
		return nil, fmt.Errorf("parse scene: %w", err)
	}
	if len(scene.Characters) == 0 {
		return nil, fmt.Errorf("scene must have at least one character")
	}
	for i := range scene.Characters {
		if scene.Characters[i].Scale <= 0 {
			scene.Characters[i].Scale = 0.4
		}
	}
	if scene.RefineDenoise <= 0 {
		scene.RefineDenoise = 0.35
	}
	return &scene, nil
}

func extractJSON(s string) string {
	s = strings.TrimSpace(s)
	s = strings.TrimPrefix(s, "```json")
	s = strings.TrimPrefix(s, "```")
	s = strings.TrimSuffix(s, "```")
	s = strings.TrimSpace(s)

	start := strings.Index(s, "{")
	end := strings.LastIndex(s, "}")
	if start >= 0 && end > start {
		return s[start : end+1]
	}
	return s
}

func imageToRGBA(img image.Image) *image.RGBA {
	if rgba, ok := img.(*image.RGBA); ok {
		return rgba
	}
	bounds := img.Bounds()
	rgba := image.NewRGBA(bounds)
	draw.Draw(rgba, bounds, img, bounds.Min, draw.Src)
	return rgba
}

func poseSuffix(prompt string) string {
	lower := strings.ToLower(prompt)
	poseHints := []string{
		"looking", "facing", "facing away", "turned", "glancing",
		"from behind", "from side", "from left", "from right",
		"profile", "back to", "standing pose", "sitting", "kneeling",
		"lying", "crouching", "walking", "running", "fighting stance",
		"portrait", "close-up", "upper body", "full body",
	}
	for _, h := range poseHints {
		if strings.Contains(lower, h) {
			return ""
		}
	}
	return ", full body, standing pose"
}

func extractStyleTags(prompt string) string {
	parts := strings.Split(prompt, ",")
	var style []string
	skipWords := []string{"man", "woman", "boy", "girl", "person", "character", "warrior", "knight",
		"wizard", "bear", "wolf", "dragon", "animal", "creature", "fighting", "holding",
		"standing", "sitting", "walking", "running", "sword", "axe", "shield", "weapon",
		"riding", "attacking", "defending", "scene", "forest", "mountain", "battle",
		"forest", "woods", "field", "river", "castle", "village", "city", "sky"}

	for _, part := range parts {
		p := strings.TrimSpace(strings.ToLower(part))
		skip := false
		for _, sw := range skipWords {
			if strings.Contains(p, sw) {
				skip = true
				break
			}
		}
		if !skip && p != "" {
			style = append(style, strings.TrimSpace(part))
		}
	}
	if len(style) == 0 {
		return prompt
	}
	return strings.Join(style, ", ")
}

func buildClipSkip(p *preset.Preset) *int {
	clipSkip := 1
	if p.ClipSkip != nil {
		clipSkip = *p.ClipSkip
	}
	return &clipSkip
}

func buildTxt2ImgRequestWithSeed(p *preset.Preset, prompt, negativePrompt string, width, height int, seed *int64) sd.Txt2ImgRequest {
	return sd.Txt2ImgRequest{
		Prompt:                 prompt,
		NegativePrompt:         negativePrompt,
		SamplerName:            promptutil.BuildSamplerName(p.Sampler, p.ScheduleType),
		Scheduler:              p.ScheduleType,
		Steps:                  p.Steps,
		CfgScale:               p.CfgScale,
		Width:                  width,
		Height:                 height,
		Seed:                   seed,
		ClipSkip:               buildClipSkip(p),
		DoNotSaveImages:        true,
		DoNotSaveGrid:          true,
	}
}

func buildTxt2ImgRequest(p *preset.Preset, prompt, negativePrompt string, width, height int) sd.Txt2ImgRequest {
	return buildTxt2ImgRequestWithSeed(p, prompt, negativePrompt, width, height, p.Seed)
}

func decodeBase64PNG(data string) (image.Image, error) {
	pngData, err := base64.StdEncoding.DecodeString(data)
	if err != nil {
		return nil, fmt.Errorf("base64 decode: %w", err)
	}
	img, err := png.Decode(bytes.NewReader(pngData))
	if err != nil {
		return nil, fmt.Errorf("png decode: %w", err)
	}
	return img, nil
}

func encodeImageToBase64(img image.Image) (string, error) {
	var buf bytes.Buffer
	if err := png.Encode(&buf, img); err != nil {
		return "", err
	}
	return base64.StdEncoding.EncodeToString(buf.Bytes()), nil
}

// ensure rembg.Client satisfies RembgClient interface
var _ RembgClient = (*rembg.Client)(nil)
