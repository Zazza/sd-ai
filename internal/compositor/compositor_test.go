package compositor

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"image"
	"image/color"
	"image/png"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"go-sd/internal/preset"
	"go-sd/internal/sd"
)

type mockSDGenerator struct {
	txt2imgFunc func(req sd.Txt2ImgRequest) (*sd.Txt2ImgResponse, error)
	img2imgFunc func(req sd.Img2ImgRequest) (*sd.Txt2ImgResponse, error)
	setModelErr error
	setVAEErr   error
}

func (m *mockSDGenerator) Txt2Img(req sd.Txt2ImgRequest) (*sd.Txt2ImgResponse, error) {
	if m.txt2imgFunc != nil {
		return m.txt2imgFunc(req)
	}
	return nil, fmt.Errorf("Txt2Img not mocked")
}

func (m *mockSDGenerator) Img2Img(req sd.Img2ImgRequest) (*sd.Txt2ImgResponse, error) {
	if m.img2imgFunc != nil {
		return m.img2imgFunc(req)
	}
	return nil, fmt.Errorf("Img2Img not mocked")
}

func (m *mockSDGenerator) SetModel(modelName string) error {
	return m.setModelErr
}

func (m *mockSDGenerator) SetVAE(vaeName string) error {
	return m.setVAEErr
}

type mockRembgClient struct {
	removeFunc func(base64Image string) (string, error)
}

func (m *mockRembgClient) RemoveBackgroundBase64(base64Image string) (string, error) {
	if m.removeFunc != nil {
		return m.removeFunc(base64Image)
	}
	return "", fmt.Errorf("RemoveBackgroundBase64 not mocked")
}

type mockPresetGetter struct {
	getFunc func(id int64) (*preset.Preset, error)
}

func (m *mockPresetGetter) Get(id int64) (*preset.Preset, error) {
	if m.getFunc != nil {
		return m.getFunc(id)
	}
	return nil, fmt.Errorf("preset not found")
}

func makeBase64PNG(w, h int) string {
	img := image.NewRGBA(image.Rect(0, 0, w, h))
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			img.SetRGBA(x, y, color.RGBA{R: 128, G: 128, B: 128, A: 255})
		}
	}
	var buf bytes.Buffer
	png.Encode(&buf, img)
	return base64.StdEncoding.EncodeToString(buf.Bytes())
}

func makeWhiteBase64PNG(w, h int) string {
	img := image.NewRGBA(image.Rect(0, 0, w, h))
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			img.SetRGBA(x, y, color.RGBA{R: 255, G: 255, B: 255, A: 255})
		}
	}
	var buf bytes.Buffer
	png.Encode(&buf, img)
	return base64.StdEncoding.EncodeToString(buf.Bytes())
}

func makeTestPreset() *preset.Preset {
	steps := 20
	return &preset.Preset{
		ID:             1,
		Name:           "test",
		Prompt:         "masterpiece, best quality",
		NegativePrompt: "blurry, low quality",
		Sampler:        "Euler a",
		ScheduleType:   "",
		Steps:          steps,
		CfgScale:       7.0,
		Width:          512,
		Height:         512,
		ModelName:      "test-model",
		VAE:            "test-vae",
	}
}

func TestNew_ReturnsCompositor(t *testing.T) {
	t.Parallel()

	sdClient := &mockSDGenerator{}
	rembgClient := &mockRembgClient{}
	presetDB := &mockPresetGetter{}

	c := New(sdClient, rembgClient, presetDB, nil)

	assert.NotNil(t, c)
	assert.Equal(t, sdClient, c.sd)
	assert.Equal(t, rembgClient, c.rembg)
	assert.Equal(t, presetDB, c.presets)
}

func TestNew_WithProgressEmitter(t *testing.T) {
	t.Parallel()

	var emitted []MultiPassProgress
	emit := func(p MultiPassProgress) {
		emitted = append(emitted, p)
	}

	sdClient := &mockSDGenerator{
		txt2imgFunc: func(req sd.Txt2ImgRequest) (*sd.Txt2ImgResponse, error) {
			return &sd.Txt2ImgResponse{Images: []string{makeBase64PNG(64, 64)}}, nil
		},
		img2imgFunc: func(req sd.Img2ImgRequest) (*sd.Txt2ImgResponse, error) {
			return &sd.Txt2ImgResponse{Images: []string{makeBase64PNG(64, 64)}}, nil
		},
	}
	presetDB := &mockPresetGetter{
		getFunc: func(id int64) (*preset.Preset, error) {
			return makeTestPreset(), nil
		},
	}

	c := New(sdClient, nil, presetDB, emit)

	scene := Scene{
		BackgroundPrompt: "forest",
		Width:            64,
		Height:           64,
		PresetID:         1,
		Characters: []CharacterSlot{
			{Name: "warrior", Prompt: "warrior, armor", Position: Position{X: 0.5, Y: 0.5}, Scale: 0.4},
		},
	}

	_, err := c.GenerateScene(scene)
	require.NoError(t, err)

	assert.True(t, len(emitted) > 0)
	assert.Equal(t, "background", emitted[0].Step)
	foundChar := false
	foundRefine := false
	foundDone := false
	for _, e := range emitted {
		if e.Step == "character" {
			foundChar = true
		}
		if e.Step == "refine" {
			foundRefine = true
		}
		if e.Step == "done" {
			foundDone = true
		}
	}
	assert.True(t, foundChar, "expected 'character' progress step")
	assert.True(t, foundRefine, "expected 'refine' progress step")
	assert.True(t, foundDone, "expected 'done' progress step")
}

func TestGenerateScene_NoCharacters(t *testing.T) {
	t.Parallel()

	c := New(&mockSDGenerator{}, nil, &mockPresetGetter{}, nil)

	scene := Scene{
		BackgroundPrompt: "forest",
		Width:            512,
		Height:           512,
		PresetID:         1,
		Characters:       []CharacterSlot{},
	}

	_, err := c.GenerateScene(scene)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "at least one character")
}

func TestGenerateScene_TooManyCharacters(t *testing.T) {
	t.Parallel()

	c := New(&mockSDGenerator{}, nil, &mockPresetGetter{}, nil)

	chars := make([]CharacterSlot, 11)
	for i := range chars {
		chars[i] = CharacterSlot{Name: fmt.Sprintf("char%d", i), Prompt: "test prompt"}
	}

	scene := Scene{
		BackgroundPrompt: "forest",
		Width:            512,
		Height:           512,
		PresetID:         1,
		Characters:       chars,
	}

	_, err := c.GenerateScene(scene)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "too many characters")
}

func TestGenerateScene_InvalidDimensions(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name   string
		width  int
		height int
	}{
		{"width_too_small", 32, 512},
		{"height_too_small", 512, 32},
		{"width_too_large", 3000, 512},
		{"height_too_large", 512, 3000},
		{"both_zero", 0, 0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			c := New(&mockSDGenerator{}, nil, &mockPresetGetter{}, nil)
			scene := Scene{
				BackgroundPrompt: "forest",
				Width:            tt.width,
				Height:           tt.height,
				PresetID:         1,
				Characters:       []CharacterSlot{{Name: "test", Prompt: "test"}},
			}

			_, err := c.GenerateScene(scene)
			assert.Error(t, err)
			assert.Contains(t, err.Error(), "invalid dimensions")
		})
	}
}

func TestGenerateScene_CharacterPromptTooLong(t *testing.T) {
	t.Parallel()

	c := New(&mockSDGenerator{}, nil, &mockPresetGetter{}, nil)

	scene := Scene{
		BackgroundPrompt: "forest",
		Width:            512,
		Height:           512,
		PresetID:         1,
		Characters: []CharacterSlot{
			{Name: "test", Prompt: strings.Repeat("a", 2001)},
		},
	}

	_, err := c.GenerateScene(scene)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "prompt too long")
}

func TestGenerateScene_PresetNotFound(t *testing.T) {
	t.Parallel()

	presetDB := &mockPresetGetter{
		getFunc: func(id int64) (*preset.Preset, error) {
			return nil, fmt.Errorf("not found")
		},
	}

	c := New(&mockSDGenerator{}, nil, presetDB, nil)

	scene := Scene{
		BackgroundPrompt: "forest",
		Width:            512,
		Height:           512,
		PresetID:         999,
		Characters:       []CharacterSlot{{Name: "test", Prompt: "test"}},
	}

	_, err := c.GenerateScene(scene)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "preset not found")
}

func TestGenerateScene_BackgroundGenerationFails(t *testing.T) {
	t.Parallel()

	sdClient := &mockSDGenerator{
		txt2imgFunc: func(req sd.Txt2ImgRequest) (*sd.Txt2ImgResponse, error) {
			return nil, fmt.Errorf("SD server error")
		},
	}
	presetDB := &mockPresetGetter{
		getFunc: func(id int64) (*preset.Preset, error) {
			return makeTestPreset(), nil
		},
	}

	c := New(sdClient, nil, presetDB, nil)

	scene := Scene{
		BackgroundPrompt: "forest",
		Width:            64,
		Height:           64,
		PresetID:         1,
		Characters:       []CharacterSlot{{Name: "test", Prompt: "test"}},
	}

	_, err := c.GenerateScene(scene)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "background generation failed")
}

func TestGenerateScene_BackgroundNoImages(t *testing.T) {
	t.Parallel()

	sdClient := &mockSDGenerator{
		txt2imgFunc: func(req sd.Txt2ImgRequest) (*sd.Txt2ImgResponse, error) {
			return &sd.Txt2ImgResponse{Images: []string{}}, nil
		},
	}
	presetDB := &mockPresetGetter{
		getFunc: func(id int64) (*preset.Preset, error) {
			return makeTestPreset(), nil
		},
	}

	c := New(sdClient, nil, presetDB, nil)

	scene := Scene{
		BackgroundPrompt: "forest",
		Width:            64,
		Height:           64,
		PresetID:         1,
		Characters:       []CharacterSlot{{Name: "test", Prompt: "test"}},
	}

	_, err := c.GenerateScene(scene)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "no background image generated")
}

func TestGenerateScene_CharacterGenerationFails(t *testing.T) {
	t.Parallel()

	callCount := 0
	sdClient := &mockSDGenerator{
		txt2imgFunc: func(req sd.Txt2ImgRequest) (*sd.Txt2ImgResponse, error) {
			callCount++
			if callCount == 1 {
				return &sd.Txt2ImgResponse{Images: []string{makeBase64PNG(64, 64)}}, nil
			}
			return nil, fmt.Errorf("character gen error")
		},
		img2imgFunc: func(req sd.Img2ImgRequest) (*sd.Txt2ImgResponse, error) {
			return &sd.Txt2ImgResponse{Images: []string{makeBase64PNG(64, 64)}}, nil
		},
	}
	presetDB := &mockPresetGetter{
		getFunc: func(id int64) (*preset.Preset, error) {
			return makeTestPreset(), nil
		},
	}

	c := New(sdClient, nil, presetDB, nil)

	scene := Scene{
		BackgroundPrompt: "forest",
		Width:            64,
		Height:           64,
		PresetID:         1,
		Characters:       []CharacterSlot{{Name: "hero", Prompt: "hero, armor"}},
	}

	_, err := c.GenerateScene(scene)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "character \"hero\" generation failed")
}

func TestGenerateScene_SuccessWithRembg(t *testing.T) {
	t.Parallel()

	callCount := 0
	sdClient := &mockSDGenerator{
		txt2imgFunc: func(req sd.Txt2ImgRequest) (*sd.Txt2ImgResponse, error) {
			callCount++
			return &sd.Txt2ImgResponse{Images: []string{makeBase64PNG(64, 64)}}, nil
		},
		img2imgFunc: func(req sd.Img2ImgRequest) (*sd.Txt2ImgResponse, error) {
			return &sd.Txt2ImgResponse{Images: []string{makeBase64PNG(64, 64)}}, nil
		},
	}

	rembgClient := &mockRembgClient{
		removeFunc: func(base64Image string) (string, error) {
			return makeBase64PNG(64, 64), nil
		},
	}

	presetDB := &mockPresetGetter{
		getFunc: func(id int64) (*preset.Preset, error) {
			return makeTestPreset(), nil
		},
	}

	c := New(sdClient, rembgClient, presetDB, nil)

	scene := Scene{
		BackgroundPrompt: "forest, trees",
		NegativePrompt:   "blurry",
		Width:            64,
		Height:           64,
		PresetID:         1,
		Characters: []CharacterSlot{
			{Name: "warrior", Prompt: "warrior, armor", Position: Position{X: 0.5, Y: 0.5}, Scale: 0.4},
		},
	}

	result, err := c.GenerateScene(scene)
	require.NoError(t, err)
	assert.NotEmpty(t, result.Image)
	assert.NotEmpty(t, result.Background)
	assert.Len(t, result.Characters, 1)
	assert.Equal(t, "warrior", result.Characters[0].Name)
}

func TestGenerateScene_SuccessWithoutRembg(t *testing.T) {
	t.Parallel()

	sdClient := &mockSDGenerator{
		txt2imgFunc: func(req sd.Txt2ImgRequest) (*sd.Txt2ImgResponse, error) {
			return &sd.Txt2ImgResponse{Images: []string{makeBase64PNG(64, 64)}}, nil
		},
		img2imgFunc: func(req sd.Img2ImgRequest) (*sd.Txt2ImgResponse, error) {
			return &sd.Txt2ImgResponse{Images: []string{makeBase64PNG(64, 64)}}, nil
		},
	}

	presetDB := &mockPresetGetter{
		getFunc: func(id int64) (*preset.Preset, error) {
			return makeTestPreset(), nil
		},
	}

	c := New(sdClient, nil, presetDB, nil)

	scene := Scene{
		BackgroundPrompt: "forest",
		Width:            64,
		Height:           64,
		PresetID:         1,
		Characters: []CharacterSlot{
			{Name: "knight", Prompt: "knight, sword", Position: Position{X: 0.3, Y: 0.6}, Scale: 0.3},
		},
	}

	result, err := c.GenerateScene(scene)
	require.NoError(t, err)
	assert.NotEmpty(t, result.Image)
	assert.Len(t, result.Characters, 1)
	assert.Equal(t, "knight", result.Characters[0].Name)
}

func TestGenerateScene_MultipleCharacters(t *testing.T) {
	t.Parallel()

	sdClient := &mockSDGenerator{
		txt2imgFunc: func(req sd.Txt2ImgRequest) (*sd.Txt2ImgResponse, error) {
			return &sd.Txt2ImgResponse{Images: []string{makeBase64PNG(64, 64)}}, nil
		},
		img2imgFunc: func(req sd.Img2ImgRequest) (*sd.Txt2ImgResponse, error) {
			return &sd.Txt2ImgResponse{Images: []string{makeBase64PNG(64, 64)}}, nil
		},
	}

	presetDB := &mockPresetGetter{
		getFunc: func(id int64) (*preset.Preset, error) {
			return makeTestPreset(), nil
		},
	}

	c := New(sdClient, nil, presetDB, nil)

	scene := Scene{
		BackgroundPrompt: "battlefield",
		Width:            64,
		Height:           64,
		PresetID:         1,
		Characters: []CharacterSlot{
			{Name: "warrior", Prompt: "warrior, sword", Position: Position{X: 0.25, Y: 0.5}, Scale: 0.3},
			{Name: "mage", Prompt: "mage, staff", Position: Position{X: 0.75, Y: 0.5}, Scale: 0.3},
		},
	}

	result, err := c.GenerateScene(scene)
	require.NoError(t, err)
	assert.Len(t, result.Characters, 2)
	assert.Equal(t, "warrior", result.Characters[0].Name)
	assert.Equal(t, "mage", result.Characters[1].Name)
}

func TestGenerateScene_RembgFailure(t *testing.T) {
	t.Parallel()

	callCount := 0
	sdClient := &mockSDGenerator{
		txt2imgFunc: func(req sd.Txt2ImgRequest) (*sd.Txt2ImgResponse, error) {
			callCount++
			return &sd.Txt2ImgResponse{Images: []string{makeBase64PNG(64, 64)}}, nil
		},
		img2imgFunc: func(req sd.Img2ImgRequest) (*sd.Txt2ImgResponse, error) {
			return &sd.Txt2ImgResponse{Images: []string{makeBase64PNG(64, 64)}}, nil
		},
	}

	rembgClient := &mockRembgClient{
		removeFunc: func(base64Image string) (string, error) {
			return "", fmt.Errorf("rembg service unavailable")
		},
	}

	presetDB := &mockPresetGetter{
		getFunc: func(id int64) (*preset.Preset, error) {
			return makeTestPreset(), nil
		},
	}

	c := New(sdClient, rembgClient, presetDB, nil)

	scene := Scene{
		BackgroundPrompt: "forest",
		Width:            64,
		Height:           64,
		PresetID:         1,
		Characters:       []CharacterSlot{{Name: "hero", Prompt: "hero"}},
	}

	_, err := c.GenerateScene(scene)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "rembg failed")
}

func TestGenerateScene_RefinementFails(t *testing.T) {
	t.Parallel()

	sdClient := &mockSDGenerator{
		txt2imgFunc: func(req sd.Txt2ImgRequest) (*sd.Txt2ImgResponse, error) {
			return &sd.Txt2ImgResponse{Images: []string{makeBase64PNG(64, 64)}}, nil
		},
		img2imgFunc: func(req sd.Img2ImgRequest) (*sd.Txt2ImgResponse, error) {
			return nil, fmt.Errorf("refinement error")
		},
	}

	presetDB := &mockPresetGetter{
		getFunc: func(id int64) (*preset.Preset, error) {
			return makeTestPreset(), nil
		},
	}

	c := New(sdClient, nil, presetDB, nil)

	scene := Scene{
		BackgroundPrompt: "forest",
		Width:            64,
		Height:           64,
		PresetID:         1,
		Characters:       []CharacterSlot{{Name: "hero", Prompt: "hero"}},
	}

	_, err := c.GenerateScene(scene)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "refinement pass failed")
}

func TestGenerateScene_RefinementNoImages(t *testing.T) {
	t.Parallel()

	sdClient := &mockSDGenerator{
		txt2imgFunc: func(req sd.Txt2ImgRequest) (*sd.Txt2ImgResponse, error) {
			return &sd.Txt2ImgResponse{Images: []string{makeBase64PNG(64, 64)}}, nil
		},
		img2imgFunc: func(req sd.Img2ImgRequest) (*sd.Txt2ImgResponse, error) {
			return &sd.Txt2ImgResponse{Images: []string{}}, nil
		},
	}

	presetDB := &mockPresetGetter{
		getFunc: func(id int64) (*preset.Preset, error) {
			return makeTestPreset(), nil
		},
	}

	c := New(sdClient, nil, presetDB, nil)

	scene := Scene{
		BackgroundPrompt: "forest",
		Width:            64,
		Height:           64,
		PresetID:         1,
		Characters:       []CharacterSlot{{Name: "hero", Prompt: "hero"}},
	}

	_, err := c.GenerateScene(scene)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "refinement pass returned no images")
}

func TestGenerateScene_DefaultScale(t *testing.T) {
	t.Parallel()

	sdClient := &mockSDGenerator{
		txt2imgFunc: func(req sd.Txt2ImgRequest) (*sd.Txt2ImgResponse, error) {
			return &sd.Txt2ImgResponse{Images: []string{makeBase64PNG(64, 64)}}, nil
		},
		img2imgFunc: func(req sd.Img2ImgRequest) (*sd.Txt2ImgResponse, error) {
			return &sd.Txt2ImgResponse{Images: []string{makeBase64PNG(64, 64)}}, nil
		},
	}

	presetDB := &mockPresetGetter{
		getFunc: func(id int64) (*preset.Preset, error) {
			return makeTestPreset(), nil
		},
	}

	c := New(sdClient, nil, presetDB, nil)

	scene := Scene{
		BackgroundPrompt: "forest",
		Width:            64,
		Height:           64,
		PresetID:         1,
		Characters: []CharacterSlot{
			{Name: "hero", Prompt: "hero", Scale: 0},
		},
	}

	result, err := c.GenerateScene(scene)
	require.NoError(t, err)
	assert.NotEmpty(t, result.Image)
}

func TestGenerateScene_LorasInPreset(t *testing.T) {
	t.Parallel()

	loras := []preset.LoRAEntry{{Name: "detail", Weight: 0.8}}
	lorasJSON, _ := json.Marshal(loras)
	p := makeTestPreset()
	p.Loras = string(lorasJSON)

	var capturedPrompt string
	sdClient := &mockSDGenerator{
		txt2imgFunc: func(req sd.Txt2ImgRequest) (*sd.Txt2ImgResponse, error) {
			capturedPrompt = req.Prompt
			return &sd.Txt2ImgResponse{Images: []string{makeBase64PNG(64, 64)}}, nil
		},
		img2imgFunc: func(req sd.Img2ImgRequest) (*sd.Txt2ImgResponse, error) {
			return &sd.Txt2ImgResponse{Images: []string{makeBase64PNG(64, 64)}}, nil
		},
	}

	presetDB := &mockPresetGetter{
		getFunc: func(id int64) (*preset.Preset, error) {
			return p, nil
		},
	}

	c := New(sdClient, nil, presetDB, nil)

	scene := Scene{
		BackgroundPrompt: "forest",
		Width:            64,
		Height:           64,
		PresetID:         1,
		Characters:       []CharacterSlot{{Name: "hero", Prompt: "hero"}},
	}

	_, err := c.GenerateScene(scene)
	require.NoError(t, err)
	assert.Contains(t, capturedPrompt, "<lora:detail:0.8>")
}

func TestDecomposeSceneFromJSON_ValidInput(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		input    string
		wantName string
		wantLen  int
	}{
		{
			name:     "valid_scene",
			input:    `{"background_prompt":"forest","characters":[{"name":"hero","prompt":"warrior","position":{"x":0.5,"y":0.5},"scale":0.4}]}`,
			wantName: "hero",
			wantLen:  1,
		},
		{
			name:     "wrapped_in_text",
			input:    `Here is the scene: {"background_prompt":"forest","characters":[{"name":"mage","prompt":"mage","position":{"x":0.5,"y":0.5},"scale":0.3}]}`,
			wantName: "mage",
			wantLen:  1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			scene, err := DecomposeSceneFromJSON(tt.input)
			require.NoError(t, err)
			assert.Len(t, scene.Characters, tt.wantLen)
			assert.Equal(t, tt.wantName, scene.Characters[0].Name)
		})
	}
}

func TestDecomposeSceneFromJSON_NoCharacters(t *testing.T) {
	t.Parallel()

	_, err := DecomposeSceneFromJSON(`{"background_prompt":"forest","characters":[]}`)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "at least one character")
}

func TestDecomposeSceneFromJSON_InvalidJSON(t *testing.T) {
	t.Parallel()

	_, err := DecomposeSceneFromJSON(`not json at all`)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "parse scene")
}

func TestDecomposeSceneFromJSON_DefaultScale(t *testing.T) {
	t.Parallel()

	input := `{"background_prompt":"forest","characters":[{"name":"hero","prompt":"warrior","position":{"x":0.5,"y":0.5},"scale":0}]}`

	scene, err := DecomposeSceneFromJSON(input)
	require.NoError(t, err)
	assert.Equal(t, 0.4, scene.Characters[0].Scale)
}

func TestRemoveWhiteBackground_AllWhite(t *testing.T) {
	t.Parallel()

	img := image.NewRGBA(image.Rect(0, 0, 100, 100))
	for y := 0; y < 100; y++ {
		for x := 0; x < 100; x++ {
			img.SetRGBA(x, y, color.RGBA{R: 255, G: 255, B: 255, A: 255})
		}
	}

	result := RemoveWhiteBackground(img)
	assert.NotNil(t, result)

	allTransparent := true
	for y := 0; y < result.Bounds().Dy(); y++ {
		for x := 0; x < result.Bounds().Dx(); x++ {
			if result.RGBAAt(x+result.Bounds().Min.X, y+result.Bounds().Min.Y).A > 0 {
				allTransparent = false
			}
		}
	}
	assert.True(t, allTransparent)
}

func TestRemoveWhiteBackground_ColoredCenter(t *testing.T) {
	t.Parallel()

	img := image.NewRGBA(image.Rect(0, 0, 100, 100))
	for y := 0; y < 100; y++ {
		for x := 0; x < 100; x++ {
			img.SetRGBA(x, y, color.RGBA{R: 255, G: 255, B: 255, A: 255})
		}
	}
	for y := 25; y < 75; y++ {
		for x := 25; x < 75; x++ {
			img.SetRGBA(x, y, color.RGBA{R: 255, G: 0, B: 0, A: 255})
		}
	}

	result := RemoveWhiteBackground(img)
	assert.NotNil(t, result)
	assert.True(t, result.Bounds().Dx() > 0)
	assert.True(t, result.Bounds().Dy() > 0)

	foundOpaque := false
	for y := result.Bounds().Min.Y; y < result.Bounds().Max.Y; y++ {
		for x := result.Bounds().Min.X; x < result.Bounds().Max.X; x++ {
			if result.RGBAAt(x, y).A > 0 {
				foundOpaque = true
			}
		}
	}
	assert.True(t, foundOpaque, "expected non-transparent pixels from colored center")
}

func TestCompositeOver_CenterPlacement(t *testing.T) {
	t.Parallel()

	bg := image.NewRGBA(image.Rect(0, 0, 200, 200))
	for y := 0; y < 200; y++ {
		for x := 0; x < 200; x++ {
			bg.SetRGBA(x, y, color.RGBA{R: 0, G: 128, B: 0, A: 255})
		}
	}

	char := image.NewRGBA(image.Rect(0, 0, 50, 80))
	for y := 0; y < 80; y++ {
		for x := 0; x < 50; x++ {
			char.SetRGBA(x, y, color.RGBA{R: 255, G: 0, B: 0, A: 255})
		}
	}

	result := CompositeOver(bg, char, Position{X: 0.5, Y: 0.5}, 0.5)
	assert.NotNil(t, result)
	assert.Equal(t, 200, result.Bounds().Dx())
	assert.Equal(t, 200, result.Bounds().Dy())
}

func TestCompositeOver_OffScreen(t *testing.T) {
	t.Parallel()

	bg := image.NewRGBA(image.Rect(0, 0, 200, 200))
	char := image.NewRGBA(image.Rect(0, 0, 50, 50))
	for y := 0; y < 50; y++ {
		for x := 0; x < 50; x++ {
			char.SetRGBA(x, y, color.RGBA{R: 255, G: 0, B: 0, A: 255})
		}
	}

	result := CompositeOver(bg, char, Position{X: -1.0, Y: -1.0}, 0.5)
	assert.NotNil(t, result)
	assert.Equal(t, 200, result.Bounds().Dx())
}

func TestCompositeOver_ZeroScale(t *testing.T) {
	t.Parallel()

	bg := image.NewRGBA(image.Rect(0, 0, 200, 200))
	char := image.NewRGBA(image.Rect(0, 0, 50, 50))

	result := CompositeOver(bg, char, Position{X: 0.5, Y: 0.5}, 0)
	assert.NotNil(t, result)
	assert.Equal(t, 200, result.Bounds().Dx())
}

func TestExtractStyleTags(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name   string
		input  string
		expect string
	}{
		{"empty", "", ""},
		{"only_style", "masterpiece, best quality", "masterpiece, best quality"},
		{"with_skip_words", "masterpiece, man, forest", "masterpiece"},
		{"all_skip_words", "man, woman, warrior", "man, woman, warrior"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			result := extractStyleTags(tt.input)
			assert.Equal(t, tt.expect, result)
		})
	}
}

func TestPoseSuffix(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name   string
		input  string
		expect string
	}{
		{"standing_pose", "warrior standing pose", ""},
		{"no_pose", "warrior with sword", ", full body, standing pose"},
		{"sitting", "girl sitting on chair", ""},
		{"profile", "woman profile view", ""},
		{"running", "man running", ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			result := poseSuffix(tt.input)
			assert.Equal(t, tt.expect, result)
		})
	}
}

func TestBuildSamplerName(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name         string
		sampler      string
		scheduleType string
		expect       string
	}{
		{"no_schedule", "Euler a", "", "Euler a"},
		{"with_karras", "Euler a", "karras", "Euler a Karras"},
		{"with_auto", "DPM++ 2M", "automatic", "DPM++ 2M Automatic"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			p := &preset.Preset{Sampler: tt.sampler, ScheduleType: tt.scheduleType}
			result := buildSamplerName(p)
			assert.Equal(t, tt.expect, result)
		})
	}
}

func TestBuildClipSkip(t *testing.T) {
	t.Parallel()

	t.Run("nil_clip_skip", func(t *testing.T) {
		t.Parallel()
		p := &preset.Preset{}
		result := buildClipSkip(p)
		assert.NotNil(t, result)
		assert.Equal(t, 1, *result)
	})

	t.Run("set_clip_skip", func(t *testing.T) {
		t.Parallel()
		cs := 3
		p := &preset.Preset{ClipSkip: &cs}
		result := buildClipSkip(p)
		assert.Equal(t, 3, *result)
	})
}

func TestDecodeBase64PNG_InvalidBase64(t *testing.T) {
	t.Parallel()

	_, err := decodeBase64PNG("not-valid-base64!!!")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "base64 decode")
}

func TestDecodeBase64PNG_InvalidPNG(t *testing.T) {
	t.Parallel()

	_, err := decodeBase64PNG(base64.StdEncoding.EncodeToString([]byte("not a png")))
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "png decode")
}

func TestDecodeBase64PNG_ValidPNG(t *testing.T) {
	t.Parallel()

	input := makeBase64PNG(10, 10)
	img, err := decodeBase64PNG(input)
	require.NoError(t, err)
	assert.Equal(t, 10, img.Bounds().Dx())
	assert.Equal(t, 10, img.Bounds().Dy())
}

func TestEncodeImageToBase64_RoundTrip(t *testing.T) {
	t.Parallel()

	original := image.NewRGBA(image.Rect(0, 0, 16, 16))
	for y := 0; y < 16; y++ {
		for x := 0; x < 16; x++ {
			original.SetRGBA(x, y, color.RGBA{R: uint8(x * 16), G: uint8(y * 16), B: 128, A: 255})
		}
	}

	encoded, err := encodeImageToBase64(original)
	require.NoError(t, err)
	assert.NotEmpty(t, encoded)

	decoded, err := decodeBase64PNG(encoded)
	require.NoError(t, err)
	assert.Equal(t, original.Bounds(), decoded.Bounds())
}

func TestGenerateScene_CharacterNoImages(t *testing.T) {
	t.Parallel()

	callCount := 0
	sdClient := &mockSDGenerator{
		txt2imgFunc: func(req sd.Txt2ImgRequest) (*sd.Txt2ImgResponse, error) {
			callCount++
			if callCount == 1 {
				return &sd.Txt2ImgResponse{Images: []string{makeBase64PNG(64, 64)}}, nil
			}
			return &sd.Txt2ImgResponse{Images: []string{}}, nil
		},
		img2imgFunc: func(req sd.Img2ImgRequest) (*sd.Txt2ImgResponse, error) {
			return &sd.Txt2ImgResponse{Images: []string{makeBase64PNG(64, 64)}}, nil
		},
	}
	presetDB := &mockPresetGetter{
		getFunc: func(id int64) (*preset.Preset, error) {
			return makeTestPreset(), nil
		},
	}

	c := New(sdClient, nil, presetDB, nil)

	scene := Scene{
		BackgroundPrompt: "forest",
		Width:            64,
		Height:           64,
		PresetID:         1,
		Characters:       []CharacterSlot{{Name: "hero", Prompt: "hero, armor"}},
	}

	_, err := c.GenerateScene(scene)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "no image for character \"hero\"")
}

func TestGenerateScene_InvalidBackgroundBase64(t *testing.T) {
	t.Parallel()

	sdClient := &mockSDGenerator{
		txt2imgFunc: func(req sd.Txt2ImgRequest) (*sd.Txt2ImgResponse, error) {
			return &sd.Txt2ImgResponse{Images: []string{"not-valid-base64!!!"}}, nil
		},
	}
	presetDB := &mockPresetGetter{
		getFunc: func(id int64) (*preset.Preset, error) {
			return makeTestPreset(), nil
		},
	}

	c := New(sdClient, nil, presetDB, nil)

	scene := Scene{
		BackgroundPrompt: "forest",
		Width:            64,
		Height:           64,
		PresetID:         1,
		Characters:       []CharacterSlot{{Name: "hero", Prompt: "hero"}},
	}

	_, err := c.GenerateScene(scene)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "decode background")
}

func TestGenerateScene_InvalidRembgBase64(t *testing.T) {
	t.Parallel()

	sdClient := &mockSDGenerator{
		txt2imgFunc: func(req sd.Txt2ImgRequest) (*sd.Txt2ImgResponse, error) {
			return &sd.Txt2ImgResponse{Images: []string{makeBase64PNG(64, 64)}}, nil
		},
		img2imgFunc: func(req sd.Img2ImgRequest) (*sd.Txt2ImgResponse, error) {
			return &sd.Txt2ImgResponse{Images: []string{makeBase64PNG(64, 64)}}, nil
		},
	}

	rembgClient := &mockRembgClient{
		removeFunc: func(base64Image string) (string, error) {
			return "not-valid-base64!!!", nil
		},
	}

	presetDB := &mockPresetGetter{
		getFunc: func(id int64) (*preset.Preset, error) {
			return makeTestPreset(), nil
		},
	}

	c := New(sdClient, rembgClient, presetDB, nil)

	scene := Scene{
		BackgroundPrompt: "forest",
		Width:            64,
		Height:           64,
		PresetID:         1,
		Characters:       []CharacterSlot{{Name: "hero", Prompt: "hero"}},
	}

	_, err := c.GenerateScene(scene)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "decode rembg result")
}

func TestGenerateScene_InvalidCharacterBase64(t *testing.T) {
	t.Parallel()

	callCount := 0
	sdClient := &mockSDGenerator{
		txt2imgFunc: func(req sd.Txt2ImgRequest) (*sd.Txt2ImgResponse, error) {
			callCount++
			if callCount == 1 {
				return &sd.Txt2ImgResponse{Images: []string{makeBase64PNG(64, 64)}}, nil
			}
			return &sd.Txt2ImgResponse{Images: []string{"not-valid-base64!!!"}}, nil
		},
		img2imgFunc: func(req sd.Img2ImgRequest) (*sd.Txt2ImgResponse, error) {
			return &sd.Txt2ImgResponse{Images: []string{makeBase64PNG(64, 64)}}, nil
		},
	}
	presetDB := &mockPresetGetter{
		getFunc: func(id int64) (*preset.Preset, error) {
			return makeTestPreset(), nil
		},
	}

	c := New(sdClient, nil, presetDB, nil)

	scene := Scene{
		BackgroundPrompt: "forest",
		Width:            64,
		Height:           64,
		PresetID:         1,
		Characters:       []CharacterSlot{{Name: "hero", Prompt: "hero"}},
	}

	_, err := c.GenerateScene(scene)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "decode character \"hero\"")
}

func TestGenerateScene_PresetNoModelVAE(t *testing.T) {
	t.Parallel()

	p := makeTestPreset()
	p.ModelName = ""
	p.VAE = ""

	sdClient := &mockSDGenerator{
		txt2imgFunc: func(req sd.Txt2ImgRequest) (*sd.Txt2ImgResponse, error) {
			return &sd.Txt2ImgResponse{Images: []string{makeBase64PNG(64, 64)}}, nil
		},
		img2imgFunc: func(req sd.Img2ImgRequest) (*sd.Txt2ImgResponse, error) {
			return &sd.Txt2ImgResponse{Images: []string{makeBase64PNG(64, 64)}}, nil
		},
	}
	presetDB := &mockPresetGetter{
		getFunc: func(id int64) (*preset.Preset, error) {
			return p, nil
		},
	}

	c := New(sdClient, nil, presetDB, nil)

	scene := Scene{
		BackgroundPrompt: "forest",
		Width:            64,
		Height:           64,
		PresetID:         1,
		Characters:       []CharacterSlot{{Name: "hero", Prompt: "hero", Scale: 0.4}},
	}

	result, err := c.GenerateScene(scene)
	require.NoError(t, err)
	assert.NotEmpty(t, result.Image)
}

func TestGenerateScene_NegativePromptMerging(t *testing.T) {
	t.Parallel()

	var capturedBgReq *sd.Txt2ImgRequest
	var capturedCharReq *sd.Txt2ImgRequest
	txt2imgCount := 0
	sdClient := &mockSDGenerator{
		txt2imgFunc: func(req sd.Txt2ImgRequest) (*sd.Txt2ImgResponse, error) {
			txt2imgCount++
			if txt2imgCount == 1 {
				capturedBgReq = &req
			} else {
				capturedCharReq = &req
			}
			return &sd.Txt2ImgResponse{Images: []string{makeBase64PNG(64, 64)}}, nil
		},
		img2imgFunc: func(req sd.Img2ImgRequest) (*sd.Txt2ImgResponse, error) {
			return &sd.Txt2ImgResponse{Images: []string{makeBase64PNG(64, 64)}}, nil
		},
	}
	presetDB := &mockPresetGetter{
		getFunc: func(id int64) (*preset.Preset, error) {
			return makeTestPreset(), nil
		},
	}

	c := New(sdClient, nil, presetDB, nil)

	scene := Scene{
		BackgroundPrompt: "forest",
		NegativePrompt:   "ugly",
		Width:            64,
		Height:           64,
		PresetID:         1,
		Characters:       []CharacterSlot{{Name: "hero", Prompt: "hero"}},
	}

	_, err := c.GenerateScene(scene)
	require.NoError(t, err)

	require.NotNil(t, capturedBgReq)
	assert.Contains(t, capturedBgReq.NegativePrompt, "ugly")
	assert.Contains(t, capturedBgReq.NegativePrompt, "blurry, low quality")

	require.NotNil(t, capturedCharReq)
	assert.Contains(t, capturedCharReq.NegativePrompt, "ugly")
}

func TestGenerateScene_CharactersExcludedFromEachOther(t *testing.T) {
	t.Parallel()

	var charPrompts []string
	sdClient := &mockSDGenerator{
		txt2imgFunc: func(req sd.Txt2ImgRequest) (*sd.Txt2ImgResponse, error) {
			charPrompts = append(charPrompts, req.NegativePrompt)
			return &sd.Txt2ImgResponse{Images: []string{makeBase64PNG(64, 64)}}, nil
		},
		img2imgFunc: func(req sd.Img2ImgRequest) (*sd.Txt2ImgResponse, error) {
			return &sd.Txt2ImgResponse{Images: []string{makeBase64PNG(64, 64)}}, nil
		},
	}
	presetDB := &mockPresetGetter{
		getFunc: func(id int64) (*preset.Preset, error) {
			return makeTestPreset(), nil
		},
	}

	c := New(sdClient, nil, presetDB, nil)

	scene := Scene{
		BackgroundPrompt: "forest",
		Width:            64,
		Height:           64,
		PresetID:         1,
		Characters: []CharacterSlot{
			{Name: "warrior", Prompt: "warrior"},
			{Name: "mage", Prompt: "mage"},
			{Name: "thief", Prompt: "thief"},
		},
	}

	_, err := c.GenerateScene(scene)
	require.NoError(t, err)
	require.Len(t, charPrompts, 4)

	assert.Contains(t, charPrompts[1], "mage")
	assert.Contains(t, charPrompts[1], "thief")

	assert.Contains(t, charPrompts[2], "warrior")
	assert.Contains(t, charPrompts[2], "thief")

	assert.Contains(t, charPrompts[3], "warrior")
	assert.Contains(t, charPrompts[3], "mage")
}

func TestGenerateScene_ProgressStepsWithRembg(t *testing.T) {
	t.Parallel()

	var steps []MultiPassProgress
	emit := func(p MultiPassProgress) {
		steps = append(steps, p)
	}

	sdClient := &mockSDGenerator{
		txt2imgFunc: func(req sd.Txt2ImgRequest) (*sd.Txt2ImgResponse, error) {
			return &sd.Txt2ImgResponse{Images: []string{makeBase64PNG(64, 64)}}, nil
		},
		img2imgFunc: func(req sd.Img2ImgRequest) (*sd.Txt2ImgResponse, error) {
			return &sd.Txt2ImgResponse{Images: []string{makeBase64PNG(64, 64)}}, nil
		},
	}

	rembgClient := &mockRembgClient{
		removeFunc: func(base64Image string) (string, error) {
			return makeBase64PNG(64, 64), nil
		},
	}

	presetDB := &mockPresetGetter{
		getFunc: func(id int64) (*preset.Preset, error) {
			return makeTestPreset(), nil
		},
	}

	c := New(sdClient, rembgClient, presetDB, emit)

	scene := Scene{
		BackgroundPrompt: "forest",
		Width:            64,
		Height:           64,
		PresetID:         1,
		Characters: []CharacterSlot{
			{Name: "hero", Prompt: "hero", Scale: 0.4},
			{Name: "sidekick", Prompt: "sidekick", Scale: 0.3},
		},
	}

	_, err := c.GenerateScene(scene)
	require.NoError(t, err)

	assert.Equal(t, "background", steps[0].Step)

	rembgSteps := 0
	for _, s := range steps {
		if s.Step == "rembg" {
			rembgSteps++
		}
	}
	assert.Equal(t, 2, rembgSteps)

	assert.Equal(t, "refine", steps[len(steps)-2].Step)
	assert.Equal(t, "done", steps[len(steps)-1].Step)
}

func TestGenerateScene_PresetNegativePromptOnly(t *testing.T) {
	t.Parallel()

	var capturedNeg string
	sdClient := &mockSDGenerator{
		txt2imgFunc: func(req sd.Txt2ImgRequest) (*sd.Txt2ImgResponse, error) {
			capturedNeg = req.NegativePrompt
			return &sd.Txt2ImgResponse{Images: []string{makeBase64PNG(64, 64)}}, nil
		},
		img2imgFunc: func(req sd.Img2ImgRequest) (*sd.Txt2ImgResponse, error) {
			return &sd.Txt2ImgResponse{Images: []string{makeBase64PNG(64, 64)}}, nil
		},
	}
	presetDB := &mockPresetGetter{
		getFunc: func(id int64) (*preset.Preset, error) {
			return makeTestPreset(), nil
		},
	}

	c := New(sdClient, nil, presetDB, nil)

	scene := Scene{
		BackgroundPrompt: "forest",
		Width:            64,
		Height:           64,
		PresetID:         1,
		Characters:       []CharacterSlot{{Name: "hero", Prompt: "hero"}},
	}

	_, err := c.GenerateScene(scene)
	require.NoError(t, err)
	assert.Equal(t, "blurry, low quality", capturedNeg)
}

func TestGenerateScene_InvalidLorasIgnored(t *testing.T) {
	t.Parallel()

	p := makeTestPreset()
	p.Loras = "not valid json"

	sdClient := &mockSDGenerator{
		txt2imgFunc: func(req sd.Txt2ImgRequest) (*sd.Txt2ImgResponse, error) {
			return &sd.Txt2ImgResponse{Images: []string{makeBase64PNG(64, 64)}}, nil
		},
		img2imgFunc: func(req sd.Img2ImgRequest) (*sd.Txt2ImgResponse, error) {
			return &sd.Txt2ImgResponse{Images: []string{makeBase64PNG(64, 64)}}, nil
		},
	}
	presetDB := &mockPresetGetter{
		getFunc: func(id int64) (*preset.Preset, error) {
			return p, nil
		},
	}

	c := New(sdClient, nil, presetDB, nil)

	scene := Scene{
		BackgroundPrompt: "forest",
		Width:            64,
		Height:           64,
		PresetID:         1,
		Characters:       []CharacterSlot{{Name: "hero", Prompt: "hero"}},
	}

	result, err := c.GenerateScene(scene)
	require.NoError(t, err)
	assert.NotEmpty(t, result.Image)
}

func TestGenerateScene_CharacterPromptAtLimit(t *testing.T) {
	t.Parallel()

	sdClient := &mockSDGenerator{
		txt2imgFunc: func(req sd.Txt2ImgRequest) (*sd.Txt2ImgResponse, error) {
			return &sd.Txt2ImgResponse{Images: []string{makeBase64PNG(64, 64)}}, nil
		},
		img2imgFunc: func(req sd.Img2ImgRequest) (*sd.Txt2ImgResponse, error) {
			return &sd.Txt2ImgResponse{Images: []string{makeBase64PNG(64, 64)}}, nil
		},
	}
	presetDB := &mockPresetGetter{
		getFunc: func(id int64) (*preset.Preset, error) {
			return makeTestPreset(), nil
		},
	}

	c := New(sdClient, nil, presetDB, nil)

	scene := Scene{
		BackgroundPrompt: "forest",
		Width:            64,
		Height:           64,
		PresetID:         1,
		Characters: []CharacterSlot{
			{Name: "hero", Prompt: strings.Repeat("a", 2000), Scale: 0.4},
		},
	}

	_, err := c.GenerateScene(scene)
	assert.NoError(t, err)
}

func TestGenerateScene_BoundaryDimensions(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name   string
		width  int
		height int
	}{
		{"min_64x64", 64, 64},
		{"max_2048x2048", 2048, 2048},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			sdClient := &mockSDGenerator{
				txt2imgFunc: func(req sd.Txt2ImgRequest) (*sd.Txt2ImgResponse, error) {
					return &sd.Txt2ImgResponse{Images: []string{makeBase64PNG(64, 64)}}, nil
				},
				img2imgFunc: func(req sd.Img2ImgRequest) (*sd.Txt2ImgResponse, error) {
					return &sd.Txt2ImgResponse{Images: []string{makeBase64PNG(64, 64)}}, nil
				},
			}
			presetDB := &mockPresetGetter{
				getFunc: func(id int64) (*preset.Preset, error) {
					return makeTestPreset(), nil
				},
			}

			c := New(sdClient, nil, presetDB, nil)

			scene := Scene{
				BackgroundPrompt: "forest",
				Width:            tt.width,
				Height:           tt.height,
				PresetID:         1,
				Characters:       []CharacterSlot{{Name: "hero", Prompt: "hero", Scale: 0.4}},
			}

			_, err := c.GenerateScene(scene)
			assert.NoError(t, err)
		})
	}
}

func TestGenerateScene_MaxCharacters(t *testing.T) {
	t.Parallel()

	sdClient := &mockSDGenerator{
		txt2imgFunc: func(req sd.Txt2ImgRequest) (*sd.Txt2ImgResponse, error) {
			return &sd.Txt2ImgResponse{Images: []string{makeBase64PNG(64, 64)}}, nil
		},
		img2imgFunc: func(req sd.Img2ImgRequest) (*sd.Txt2ImgResponse, error) {
			return &sd.Txt2ImgResponse{Images: []string{makeBase64PNG(64, 64)}}, nil
		},
	}
	presetDB := &mockPresetGetter{
		getFunc: func(id int64) (*preset.Preset, error) {
			return makeTestPreset(), nil
		},
	}

	c := New(sdClient, nil, presetDB, nil)

	chars := make([]CharacterSlot, 10)
	for i := range chars {
		chars[i] = CharacterSlot{
			Name:     fmt.Sprintf("char%d", i),
			Prompt:   fmt.Sprintf("character %d", i),
			Position: Position{X: 0.5, Y: 0.5},
			Scale:    0.2,
		}
	}

	scene := Scene{
		BackgroundPrompt: "epic scene",
		Width:            64,
		Height:           64,
		PresetID:         1,
		Characters:       chars,
	}

	result, err := c.GenerateScene(scene)
	require.NoError(t, err)
	assert.Len(t, result.Characters, 10)
}

func TestDecomposeSceneFromJSON_WhitespaceInput(t *testing.T) {
	t.Parallel()

	input := `  {"background_prompt":"castle","characters":[{"name":"king","prompt":"king","position":{"x":0.5,"y":0.5},"scale":0.5}]}  `

	scene, err := DecomposeSceneFromJSON(input)
	require.NoError(t, err)
	assert.Equal(t, "castle", scene.BackgroundPrompt)
	assert.Equal(t, "king", scene.Characters[0].Name)
}

func TestDecomposeSceneFromJSON_MultipleCharacters(t *testing.T) {
	t.Parallel()

	input := `{"background_prompt":"battle","characters":[{"name":"a","prompt":"a","position":{"x":0.3,"y":0.5},"scale":0.4},{"name":"b","prompt":"b","position":{"x":0.7,"y":0.5},"scale":0.4}]}`

	scene, err := DecomposeSceneFromJSON(input)
	require.NoError(t, err)
	assert.Len(t, scene.Characters, 2)
	assert.Equal(t, "a", scene.Characters[0].Name)
	assert.Equal(t, "b", scene.Characters[1].Name)
}

func TestDecomposeSceneFromJSON_EmptyString(t *testing.T) {
	t.Parallel()

	_, err := DecomposeSceneFromJSON("")
	assert.Error(t, err)
}

func TestDecomposeSceneFromJSON_PartialJSON(t *testing.T) {
	t.Parallel()

	_, err := DecomposeSceneFromJSON("some text without braces")
	assert.Error(t, err)
}

func TestRemoveWhiteBackground_ColoredBorderStrip(t *testing.T) {
	t.Parallel()

	img := image.NewRGBA(image.Rect(0, 0, 50, 50))
	for y := 0; y < 50; y++ {
		for x := 0; x < 50; x++ {
			img.SetRGBA(x, y, color.RGBA{R: 255, G: 255, B: 255, A: 255})
		}
	}
	for x := 5; x < 45; x++ {
		img.SetRGBA(x, 25, color.RGBA{R: 0, G: 255, B: 0, A: 255})
	}

	result := RemoveWhiteBackground(img)
	assert.NotNil(t, result)

	found := false
	for y := result.Bounds().Min.Y; y < result.Bounds().Max.Y; y++ {
		for x := result.Bounds().Min.X; x < result.Bounds().Max.X; x++ {
			if result.RGBAAt(x, y).A > 0 {
				found = true
			}
		}
	}
	assert.True(t, found, "colored strip should survive")
}

func TestRemoveWhiteBackground_FullyColored(t *testing.T) {
	t.Parallel()

	img := image.NewRGBA(image.Rect(0, 0, 50, 50))
	for y := 0; y < 50; y++ {
		for x := 0; x < 50; x++ {
			img.SetRGBA(x, y, color.RGBA{R: 128, G: 64, B: 32, A: 255})
		}
	}

	result := RemoveWhiteBackground(img)
	assert.NotNil(t, result)

	opaqueCount := 0
	for y := result.Bounds().Min.Y; y < result.Bounds().Max.Y; y++ {
		for x := result.Bounds().Min.X; x < result.Bounds().Max.X; x++ {
			if result.RGBAAt(x, y).A > 0 {
				opaqueCount++
			}
		}
	}
	assert.True(t, opaqueCount > 0, "colored image should retain opaque pixels")
}

func TestCompositeOver_CornerPlacement(t *testing.T) {
	t.Parallel()

	bg := image.NewRGBA(image.Rect(0, 0, 200, 200))
	for y := 0; y < 200; y++ {
		for x := 0; x < 200; x++ {
			bg.SetRGBA(x, y, color.RGBA{R: 100, G: 100, B: 100, A: 255})
		}
	}

	char := image.NewRGBA(image.Rect(0, 0, 40, 40))
	for y := 0; y < 40; y++ {
		for x := 0; x < 40; x++ {
			char.SetRGBA(x, y, color.RGBA{R: 255, G: 0, B: 0, A: 200})
		}
	}

	tests := []struct {
		name string
		pos  Position
	}{
		{"top_left", Position{X: 0.0, Y: 0.0}},
		{"top_right", Position{X: 1.0, Y: 0.0}},
		{"bottom_left", Position{X: 0.0, Y: 1.0}},
		{"bottom_right", Position{X: 1.0, Y: 1.0}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			result := CompositeOver(bg, char, tt.pos, 0.3)
			assert.NotNil(t, result)
			assert.Equal(t, 200, result.Bounds().Dx())
			assert.Equal(t, 200, result.Bounds().Dy())
		})
	}
}

func TestCompositeOver_LargeScaleCharacter(t *testing.T) {
	t.Parallel()

	bg := image.NewRGBA(image.Rect(0, 0, 100, 100))
	char := image.NewRGBA(image.Rect(0, 0, 20, 20))
	for y := 0; y < 20; y++ {
		for x := 0; x < 20; x++ {
			char.SetRGBA(x, y, color.RGBA{R: 255, G: 0, B: 0, A: 255})
		}
	}

	result := CompositeOver(bg, char, Position{X: 0.5, Y: 0.5}, 1.5)
	assert.NotNil(t, result)
	assert.Equal(t, 100, result.Bounds().Dx())
}

func TestCompositeOver_SemiTransparentCharacter(t *testing.T) {
	t.Parallel()

	bg := image.NewRGBA(image.Rect(0, 0, 100, 100))
	for y := 0; y < 100; y++ {
		for x := 0; x < 100; x++ {
			bg.SetRGBA(x, y, color.RGBA{R: 0, G: 0, B: 255, A: 255})
		}
	}

	char := image.NewRGBA(image.Rect(0, 0, 50, 50))
	for y := 0; y < 50; y++ {
		for x := 0; x < 50; x++ {
			char.SetRGBA(x, y, color.RGBA{R: 255, G: 0, B: 0, A: 128})
		}
	}

	result := CompositeOver(bg, char, Position{X: 0.5, Y: 0.5}, 0.5)
	assert.NotNil(t, result)

	centerX := result.Bounds().Dx() / 2
	centerY := result.Bounds().Dy() / 2
	c := result.RGBAAt(centerX, centerY)
	assert.True(t, c.R > 0, "expected blended red at center")
	assert.True(t, c.B > 0, "expected blended blue at center")
}

func TestImageToRGBA_AlreadyRGBA(t *testing.T) {
	t.Parallel()

	original := image.NewRGBA(image.Rect(0, 0, 10, 10))
	result := imageToRGBA(original)
	assert.Equal(t, original, result)
}

func TestImageToRGBA_NonRGBA(t *testing.T) {
	t.Parallel()

	original := image.NewNRGBA(image.Rect(0, 0, 10, 10))
	for y := 0; y < 10; y++ {
		for x := 0; x < 10; x++ {
			original.SetNRGBA(x, y, color.NRGBA{R: 100, G: 200, B: 50, A: 255})
		}
	}

	result := imageToRGBA(original)
	assert.NotNil(t, result)
	assert.Equal(t, 10, result.Bounds().Dx())
	assert.Equal(t, 10, result.Bounds().Dy())
}

func TestIsNearWhite(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name   string
		r      uint8
		g      uint8
		b      uint8
		expect bool
	}{
		{"pure_white", 255, 255, 255, true},
		{"near_white", 250, 248, 252, true},
		{"light_gray", 220, 220, 220, true},
		{"red", 255, 0, 0, false},
		{"black", 0, 0, 0, false},
		{"medium_gray", 128, 128, 128, false},
		{"bright_saturated", 255, 230, 230, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			img := image.NewNRGBA(image.Rect(0, 0, 1, 1))
			img.SetNRGBA(0, 0, color.NRGBA{R: tt.r, G: tt.g, B: tt.b, A: 255})
			result := isNearWhite(img, 0, 0, image.Point{})
			assert.Equal(t, tt.expect, result)
		})
	}
}

func TestCropToBoundingBox_NoOpaquePixels(t *testing.T) {
	t.Parallel()

	img := image.NewRGBA(image.Rect(0, 0, 20, 20))
	result := cropToBoundingBox(img)
	assert.Equal(t, img.Bounds(), result.Bounds())
}

func TestCropToBoundingBox_WithOpaquePixels(t *testing.T) {
	t.Parallel()

	img := image.NewRGBA(image.Rect(0, 0, 100, 100))
	img.SetRGBA(20, 30, color.RGBA{R: 255, G: 0, B: 0, A: 255})
	img.SetRGBA(40, 50, color.RGBA{R: 0, G: 255, B: 0, A: 255})

	result := cropToBoundingBox(img)
	assert.NotNil(t, result)
	assert.True(t, result.Bounds().Dx() > 0)
	assert.True(t, result.Bounds().Dy() > 0)
}

func TestComputeDistanceField(t *testing.T) {
	t.Parallel()

	removed := [][]bool{
		{true, true, true},
		{true, false, true},
		{true, true, true},
	}

	dist := computeDistanceField(removed, 3, 3)
	assert.Equal(t, 0.0, dist[0][0])
	assert.Equal(t, 0.0, dist[1][0])
	assert.True(t, dist[1][1] > 0)
}

func TestClamp(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name         string
		v, lo, hi    int
		expect       int
	}{
		{"below_lo", -5, 0, 10, 0},
		{"above_hi", 15, 0, 10, 10},
		{"in_range", 5, 0, 10, 5},
		{"at_lo", 0, 0, 10, 0},
		{"at_hi", 10, 0, 10, 10},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			result := clamp(tt.v, tt.lo, tt.hi)
			assert.Equal(t, tt.expect, result)
		})
	}
}

func TestBuildTxt2ImgRequest(t *testing.T) {
	t.Parallel()

	p := makeTestPreset()
	req := buildTxt2ImgRequest(p, "test prompt", "test neg", 512, 768)

	assert.Equal(t, "test prompt", req.Prompt)
	assert.Equal(t, "test neg", req.NegativePrompt)
	assert.Equal(t, 512, req.Width)
	assert.Equal(t, 768, req.Height)
	assert.Equal(t, 20, req.Steps)
	assert.Equal(t, 7.0, req.CfgScale)
	assert.True(t, req.DoNotSaveImages)
	assert.True(t, req.DoNotSaveGrid)
}

func TestBuildTxt2ImgRequestWithSeed(t *testing.T) {
	t.Parallel()

	p := makeTestPreset()
	seed := int64(42)
	req := buildTxt2ImgRequestWithSeed(p, "prompt", "neg", 512, 512, &seed)

	assert.NotNil(t, req.Seed)
	assert.Equal(t, int64(42), *req.Seed)
}

func TestExtractJSON(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name   string
		input  string
		expect string
	}{
		{"clean_json", `{"key":"val"}`, `{"key":"val"}`},
		{"with_prefix", `here is {"key":"val"}`, `{"key":"val"}`},
		{"with_suffix", `{"key":"val"} end`, `{"key":"val"}`},
		{"with_both", `text {"key":"val"} more`, `{"key":"val"}`},
		{"no_braces", `no json`, `no json`},
		{"empty", ``, ``},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			result := extractJSON(tt.input)
			assert.Equal(t, tt.expect, result)
		})
	}
}

func TestBlurMask(t *testing.T) {
	t.Parallel()

	mask := [][]float64{
		{1.0, 0.0, 1.0},
		{0.0, 1.0, 0.0},
		{1.0, 0.0, 1.0},
	}

	result := blurMask(mask, 3, 3, 1)
	assert.NotNil(t, result)
	assert.Len(t, result, 3)

	for _, row := range result {
		assert.Len(t, row, 3)
	}
}

func TestScaleImage(t *testing.T) {
	t.Parallel()

	src := image.NewRGBA(image.Rect(0, 0, 100, 100))
	for y := 0; y < 100; y++ {
		for x := 0; x < 100; x++ {
			src.SetRGBA(x, y, color.RGBA{R: uint8(x * 2), G: uint8(y * 2), B: 128, A: 255})
		}
	}

	result := scaleImage(src, 50, 50)
	assert.Equal(t, 50, result.Bounds().Dx())
	assert.Equal(t, 50, result.Bounds().Dy())
}

func TestGenerateScene_LorasMultiple(t *testing.T) {
	t.Parallel()

	loras := []preset.LoRAEntry{
		{Name: "detail", Weight: 0.8},
		{Name: "anime", Weight: 0.5},
	}
	lorasJSON, _ := json.Marshal(loras)
	p := makeTestPreset()
	p.Loras = string(lorasJSON)

	var capturedPrompt string
	sdClient := &mockSDGenerator{
		txt2imgFunc: func(req sd.Txt2ImgRequest) (*sd.Txt2ImgResponse, error) {
			capturedPrompt = req.Prompt
			return &sd.Txt2ImgResponse{Images: []string{makeBase64PNG(64, 64)}}, nil
		},
		img2imgFunc: func(req sd.Img2ImgRequest) (*sd.Txt2ImgResponse, error) {
			return &sd.Txt2ImgResponse{Images: []string{makeBase64PNG(64, 64)}}, nil
		},
	}
	presetDB := &mockPresetGetter{
		getFunc: func(id int64) (*preset.Preset, error) {
			return p, nil
		},
	}

	c := New(sdClient, nil, presetDB, nil)

	scene := Scene{
		BackgroundPrompt: "forest",
		Width:            64,
		Height:           64,
		PresetID:         1,
		Characters:       []CharacterSlot{{Name: "hero", Prompt: "hero"}},
	}

	_, err := c.GenerateScene(scene)
	require.NoError(t, err)
	assert.Contains(t, capturedPrompt, "<lora:detail:0.8>")
	assert.Contains(t, capturedPrompt, "<lora:anime:0.5>")
}

func TestGenerateScene_LargerDimensions(t *testing.T) {
	t.Parallel()

	sdClient := &mockSDGenerator{
		txt2imgFunc: func(req sd.Txt2ImgRequest) (*sd.Txt2ImgResponse, error) {
			return &sd.Txt2ImgResponse{Images: []string{makeBase64PNG(64, 64)}}, nil
		},
		img2imgFunc: func(req sd.Img2ImgRequest) (*sd.Txt2ImgResponse, error) {
			return &sd.Txt2ImgResponse{Images: []string{makeBase64PNG(64, 64)}}, nil
		},
	}

	presetDB := &mockPresetGetter{
		getFunc: func(id int64) (*preset.Preset, error) {
			return makeTestPreset(), nil
		},
	}

	c := New(sdClient, nil, presetDB, nil)

	scene := Scene{
		BackgroundPrompt: "landscape",
		Width:            768,
		Height:           512,
		PresetID:         1,
		Characters: []CharacterSlot{
			{Name: "hero", Prompt: "hero, armor", Position: Position{X: 0.5, Y: 0.5}, Scale: 0.4},
		},
	}

	result, err := c.GenerateScene(scene)
	require.NoError(t, err)
	assert.NotEmpty(t, result.Image)
}
