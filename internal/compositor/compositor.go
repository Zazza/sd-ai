package compositor

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"image"
	"image/draw"
	"image/png"
	"strings"

	"go-sd/internal/preset"
	"go-sd/internal/sd"
)

type SDGenerator interface {
	Txt2Img(req sd.Txt2ImgRequest) (*sd.Txt2ImgResponse, error)
	Img2Img(req sd.Img2ImgRequest) (*sd.Txt2ImgResponse, error)
	SetModel(modelName string) error
	SetVAE(vaeName string) error
}

type PresetGetter interface {
	Get(id int64) (*preset.Preset, error)
}

type ProgressEmitter func(progress MultiPassProgress)

type Compositor struct {
	sd      SDGenerator
	presets PresetGetter
	emit    ProgressEmitter
}

func New(sdClient SDGenerator, presetDB PresetGetter, emit ProgressEmitter) *Compositor {
	return &Compositor{
		sd:      sdClient,
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

	bgPrompt := p.Prompt + ", " + scene.BackgroundPrompt + loraSuffix
	bgNeg := scene.NegativePrompt
	if bgNeg == "" {
		bgNeg = p.NegativePrompt
	}

	bgReq := buildTxt2ImgRequest(p, bgPrompt, bgNeg, scene.Width, scene.Height)
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

	originalBgBase64 := bgResp.Images[0]

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

		charPrompt := p.Prompt + ", " + char.Prompt + loraSuffix
		charNeg := scene.NegativePrompt
		if charNeg == "" {
			charNeg = p.NegativePrompt
		}

		scale := char.Scale
		if scale <= 0 {
			scale = 0.4
		}

		mask := createCharacterMask(bgImage.Bounds(), char.Position, scale)
		maskBase64, err := encodeImageToBase64(mask)
		if err != nil {
			return nil, fmt.Errorf("encode mask for character %q: %w", char.Name, err)
		}

		samplerName := p.Sampler
		if p.ScheduleType != "" {
			st := strings.ToUpper(p.ScheduleType[:1]) + p.ScheduleType[1:]
			samplerName = p.Sampler + " " + st
		}

		clipSkip := 1
		if p.ClipSkip != nil {
			clipSkip = *p.ClipSkip
		}

		denoising := 0.65
		inpaintReq := sd.Img2ImgRequest{
			InitImages:           []string{originalBgBase64},
			Mask:                 maskBase64,
			Prompt:               charPrompt,
			NegativePrompt:       charNeg,
			SamplerName:          samplerName,
			Scheduler:            p.ScheduleType,
			Steps:                p.Steps,
			CfgScale:             p.CfgScale,
			Width:                scene.Width,
			Height:               scene.Height,
			Seed:                 p.Seed,
			DenoisingStrength:    &denoising,
			ClipSkip:             &clipSkip,
			MaskBlur:             8,
			InpaintingFill:       1,
			InpaintFullRes:       true,
			InpaintFullResPadding: 32,
			DoNotSaveImages:      true,
			DoNotSaveGrid:        true,
		}

		inpaintResp, err := c.sd.Img2Img(inpaintReq)
		if err != nil {
			return nil, fmt.Errorf("character %q inpaint failed: %w", char.Name, err)
		}
		if len(inpaintResp.Images) == 0 {
			return nil, fmt.Errorf("no image for character %q", char.Name)
		}

		charResult, err := decodeBase64PNG(inpaintResp.Images[0])
		if err != nil {
			return nil, fmt.Errorf("decode character %q: %w", char.Name, err)
		}

		applyMaskedRegion(bgImage, charResult, mask)

		result.Characters = append(result.Characters, struct {
			Name  string `json:"name"`
			Image string `json:"image,omitempty"`
		}{Name: char.Name, Image: inpaintResp.Images[0]})
	}

	if c.emit != nil {
		c.emit(MultiPassProgress{Step: "done"})
	}

	finalBase64, err := encodeImageToBase64(bgImage)
	if err != nil {
		return nil, fmt.Errorf("encode result: %w", err)
	}
	result.Image = finalBase64

	return result, nil
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
	return &scene, nil
}

func extractJSON(s string) string {
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

func buildTxt2ImgRequest(p *preset.Preset, prompt, negativePrompt string, width, height int) sd.Txt2ImgRequest {
	samplerName := p.Sampler
	if p.ScheduleType != "" {
		st := strings.ToUpper(p.ScheduleType[:1]) + p.ScheduleType[1:]
		samplerName = p.Sampler + " " + st
	}

	clipSkip := 1
	if p.ClipSkip != nil {
		clipSkip = *p.ClipSkip
	}

	return sd.Txt2ImgRequest{
		Prompt:                 prompt,
		NegativePrompt:         negativePrompt,
		SamplerName:            samplerName,
		Scheduler:              p.ScheduleType,
		Steps:                  p.Steps,
		CfgScale:               p.CfgScale,
		Width:                  width,
		Height:                 height,
		Seed:                   p.Seed,
		ClipSkip:               &clipSkip,
		DoNotSaveImages:        true,
		DoNotSaveGrid:          true,
	}
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
