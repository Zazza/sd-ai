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
	"image/jpeg"
	"image/png"
	"math"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/chai2010/webp"
	"github.com/wailsapp/wails/v2/pkg/runtime"
	xdraw "golang.org/x/image/draw"

	"go-sd/internal/compositor"
	"go-sd/internal/config"
	"go-sd/internal/kids"
	"go-sd/internal/llm"
	"go-sd/internal/logger"
	"go-sd/internal/preset"
	"go-sd/internal/rembg"
	"go-sd/internal/sd"
)

type App struct {
	ctx          context.Context
	presets      *preset.DB
	llm          *llm.Client
	sd           *sd.Client
	rembgClient  *rembg.Client
	log          *logger.Logger
	config       *config.Config
	dataDir      string
	batchMu      sync.Mutex
	batchRunning bool
}

func NewApp(presets *preset.DB, llmClient *llm.Client, sdClient *sd.Client, cfg *config.Config) *App {
	return &App{
		presets:     presets,
		llm:         llmClient,
		sd:          sdClient,
		rembgClient: rembg.New(""),
		log:         logger.New(nil),
		config:      cfg,
		dataDir:     filepath.Dir(cfg.DBPath),
	}
}

func (a *App) startup(ctx context.Context) {
	a.ctx = ctx
	a.log.SetContext(ctx)
}

const (
	maxPinAttempts = 5
	pinLockoutMins = 5
)

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
		a.presets.SetSetting("kids_pin_attempts", "0")
		a.presets.SetSetting("kids_pin_lockout", "")
		return a.presets.SetSetting("kids_mode", "true")
	}

	storedHash, _ := a.presets.GetSetting("kids_pin_hash")
	if storedHash != "" {
		if err := a.checkPinLockout(); err != nil {
			return err
		}
		if pin == "" {
			return fmt.Errorf("PIN required")
		}
		hash := sha256.Sum256([]byte(pin))
		if hex.EncodeToString(hash[:]) != storedHash {
			a.recordFailedPinAttempt()
			return fmt.Errorf("incorrect PIN")
		}
		a.presets.SetSetting("kids_pin_attempts", "0")
		a.presets.SetSetting("kids_pin_lockout", "")
	}
	return a.presets.SetSetting("kids_mode", "false")
}

type KidsCategoryInfo struct {
	Name     string `json:"name"`
	Label    string `json:"label"`
	AlwaysOn bool   `json:"alwaysOn"`
	Enabled  bool   `json:"enabled"`
}

func (a *App) GetKidsCategories() ([]KidsCategoryInfo, error) {
	var result []KidsCategoryInfo
	for _, cat := range kids.Categories {
		v, _ := a.presets.GetSetting("kids_cat_" + cat.Name)
		enabled := cat.AlwaysOn || v != "false"
		result = append(result, KidsCategoryInfo{
			Name:     cat.Name,
			Label:    cat.Label,
			AlwaysOn: cat.AlwaysOn,
			Enabled:  enabled,
		})
	}
	return result, nil
}

func (a *App) SetKidsCategory(name string, enabled bool, pin string) error {
	if !a.isKidsMode() {
		return fmt.Errorf("Kids Mode is not active")
	}
	var found *kids.Category
	for i := range kids.Categories {
		if kids.Categories[i].Name == name {
			found = &kids.Categories[i]
			break
		}
	}
	if found == nil {
		return fmt.Errorf("unknown category: %s", name)
	}
	if found.AlwaysOn {
		return fmt.Errorf("category %s cannot be disabled", name)
	}
	storedHash, _ := a.presets.GetSetting("kids_pin_hash")
	if storedHash != "" {
		if err := a.checkPinLockout(); err != nil {
			return err
		}
		if pin == "" {
			return fmt.Errorf("PIN required")
		}
		hash := sha256.Sum256([]byte(pin))
		if hex.EncodeToString(hash[:]) != storedHash {
			a.recordFailedPinAttempt()
			return fmt.Errorf("incorrect PIN")
		}
		a.presets.SetSetting("kids_pin_attempts", "0")
		a.presets.SetSetting("kids_pin_lockout", "")
	}
	val := "true"
	if !enabled {
		val = "false"
	}
	return a.presets.SetSetting("kids_cat_"+name, val)
}

func (a *App) checkPinLockout() error {
	lockoutStr, _ := a.presets.GetSetting("kids_pin_lockout")
	if lockoutStr == "" {
		return nil
	}
	lockoutTime, err := time.Parse(time.RFC3339, lockoutStr)
	if err != nil {
		a.presets.SetSetting("kids_pin_lockout", "")
		a.presets.SetSetting("kids_pin_attempts", "0")
		return nil
	}
	if time.Now().Before(lockoutTime) {
		remaining := time.Until(lockoutTime).Truncate(time.Second)
		return fmt.Errorf("PIN locked. Try again in %s", remaining)
	}
	a.presets.SetSetting("kids_pin_lockout", "")
	a.presets.SetSetting("kids_pin_attempts", "0")
	return nil
}

func (a *App) recordFailedPinAttempt() {
	attemptsStr, _ := a.presets.GetSetting("kids_pin_attempts")
	attempts := 0
	if n, err := strconv.Atoi(attemptsStr); err == nil {
		attempts = n
	}
	attempts++
	a.presets.SetSetting("kids_pin_attempts", strconv.Itoa(attempts))
	if attempts >= maxPinAttempts {
		lockoutUntil := time.Now().Add(pinLockoutMins * time.Minute).Format(time.RFC3339)
		a.presets.SetSetting("kids_pin_lockout", lockoutUntil)
	}
}

func (a *App) getKidsDisabledCategories() map[string]bool {
	disabled := make(map[string]bool)
	for _, cat := range kids.Categories {
		if cat.AlwaysOn {
			continue
		}
		v, _ := a.presets.GetSetting("kids_cat_" + cat.Name)
		if v == "false" {
			disabled[cat.Name] = true
		}
	}
	return disabled
}

func (a *App) applyKidsSystemPrompt(systemPrompt string) string {
	if !a.isKidsMode() {
		return systemPrompt
	}
	return systemPrompt + config.KidsModePrompt
}

func (a *App) applyKidsNegative(negativePrompt string) string {
	if !a.isKidsMode() {
		return negativePrompt
	}
	disabled := a.getKidsDisabledCategories()
	kidsNeg := kids.NegativePrompt(disabled)
	if negativePrompt != "" {
		return negativePrompt + ", " + kidsNeg
	}
	return kidsNeg
}

func (a *App) filterKidsInput(text string) (string, error) {
	if !a.isKidsMode() {
		return text, nil
	}
	return kids.FilterInput(text, a.getKidsDisabledCategories())
}

func (a *App) filterKidsOutput(text string) string {
	if !a.isKidsMode() {
		return text
	}
	return kids.FilterOutput(text, a.getKidsDisabledCategories())
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
			a.log.Warn("LLM unavailable: %s", err)
			return
		}
		status.LLM.Available = true
		status.LLM.Model = a.config.SDPromptModel
	}()

	go func() {
		defer wg.Done()
		if err := a.sd.HealthCheck(); err != nil {
			status.SD.Available = false
			a.log.Warn("SD unavailable: %s", err)
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
	a.log.Debug("Service check: LLM=%v SD=%v", status.LLM.Available, status.SD.Available)
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

	var filterErr error
	description, filterErr = a.filterKidsInput(description)
	if filterErr != nil {
		return nil, filterErr
	}
	negative, filterErr = a.filterKidsInput(negative)
	if filterErr != nil {
		return nil, filterErr
	}
	systemPrompt = a.applyKidsSystemPrompt(systemPrompt)

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
	jsonRaw := extractJSON(raw)
	if err := json.Unmarshal([]byte(jsonRaw), &result); err != nil {
		result = GenerateSDPromptResult{
			Prompt:         truncateRepetitive(raw, 1000),
			NegativePrompt: p.NegativePrompt,
		}
		}

	if containsCyrillic(result.Prompt) {
		result.Prompt = extractTagsFromRaw(raw)
	}
	if containsCyrillic(result.NegativePrompt) {
		result.NegativePrompt = extractNegativeFromRaw(raw)
	}

	extractEmbeddedNegative(&result)

	result.Prompt = stripJunk(result.Prompt)
	result.Prompt = truncateRepetitive(result.Prompt, 1000)
	result.NegativePrompt = stripJunk(result.NegativePrompt)
	result.NegativePrompt = truncateRepetitive(result.NegativePrompt, 500)

	result.Prompt = a.filterKidsOutput(result.Prompt)
	result.NegativePrompt = a.filterKidsOutput(result.NegativePrompt)

	return &result, nil
}

func (a *App) GetDefaultPromptInstruction() string {
	return config.DefaultSDPromptInstruction
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
	if len(imageBase64) > 22*1024*1024 {
		return "", fmt.Errorf("image too large (max 16 MB)")
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

	systemPrompt, _ := a.presets.GetSetting("analyze_system_prompt")
	if systemPrompt == "" {
		systemPrompt = config.DefaultAnalyzeSystemPrompt
	}

	maxTokens := 256
	if v, err := a.presets.GetSetting("llm_max_tokens"); err == nil && v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 {
			maxTokens = n
		}
	}

	useChain := true
	if v, err := a.presets.GetSetting("analyze_use_chain"); err == nil {
		useChain = v != "false"
	}

	if !useChain {
		prompt, _ := a.presets.GetSetting("analyze_prompt")
		if prompt == "" {
			prompt = config.DefaultAnalyzePrompt
		}
		tags, err := a.llm.AnalyzeImage(model, systemPrompt+"\n\n"+prompt, imageBase64, maxTokens)
		if err != nil {
			return "", err
		}
		tags = a.filterKidsOutput(tags)
		return tags, nil
	}

	chainPrompts := a.getAnalyzeChainPrompts()
	messages := []llm.Message{
		{Role: "system", Content: systemPrompt},
		{Role: "user", Content: []llm.ContentPart{
			{Type: "text", Text: chainPrompts[0]},
			{Type: "image_url", ImageURL: &llm.ImageURLPart{URL: "data:image/png;base64," + imageBase64}},
		}},
	}

	for i := 0; i < len(chainPrompts); i++ {
		resp, err := a.llm.ChatWithMessages(model, messages, 0.4, maxTokens)
		if err != nil {
			if i == 0 {
				return "", err
			}
			break
		}

		messages = append(messages, llm.Message{Role: "assistant", Content: resp})

		if i+1 < len(chainPrompts) {
			messages = append(messages, llm.Message{
				Role:    "user",
				Content: chainPrompts[i+1],
			})
		}

		if a.ctx != nil {
			runtime.EventsEmit(a.ctx, "analyze:step", i+1, len(chainPrompts))
		}
	}

	lastResp := ""
	for j := len(messages) - 1; j >= 0; j-- {
		if messages[j].Role == "assistant" {
			if s, ok := messages[j].Content.(string); ok {
				lastResp = s
			}
			break
		}
	}

	tags := llm.CleanTags(lastResp)
	tags = a.filterKidsOutput(tags)
	return tags, nil
}

func (a *App) getAnalyzeChainPrompts() []string {
	prompts := make([]string, 4)
	for i := range prompts {
		key := "analyze_chain_" + strconv.Itoa(i+1)
		if v, err := a.presets.GetSetting(key); err == nil && v != "" {
			prompts[i] = v
		} else if i < len(config.DefaultAnalyzeChainPrompts) {
			prompts[i] = config.DefaultAnalyzeChainPrompts[i]
		}
	}
	return prompts
}

type AnalyzePrompts struct {
	SystemPrompt string   `json:"system_prompt"`
	SinglePrompt string   `json:"single_prompt"`
	ChainPrompts []string `json:"chain_prompts"`
}

func (a *App) GetDefaultAnalyzePrompts() *AnalyzePrompts {
	return &AnalyzePrompts{
		SystemPrompt: config.DefaultAnalyzeSystemPrompt,
		SinglePrompt: config.DefaultAnalyzePrompt,
		ChainPrompts: config.DefaultAnalyzeChainPrompts,
	}
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

func (a *App) ReadClipboardImage() (string, error) {
	var data []byte
	var err error

	if os.Getenv("WAYLAND_DISPLAY") != "" {
		data, err = exec.Command("wl-paste", "-t", "image/png").Output()
	} else {
		data, err = exec.Command("xclip", "-selection", "clipboard", "-t", "image/png", "-o").Output()
	}

	if err != nil {
		if os.Getenv("WAYLAND_DISPLAY") != "" {
			return "", fmt.Errorf("failed to read clipboard (install wl-clipboard)")
		}
		return "", fmt.Errorf("failed to read clipboard (install xclip)")
	}

	if len(data) == 0 {
		return "", fmt.Errorf("no image in clipboard")
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
	Image                   any    `json:"image"`
	Parameters              any    `json:"parameters"`
	Info                    any    `json:"info"`
	IsPreview               bool   `json:"is_preview"`
	EffectivePrompt         string `json:"effective_prompt"`
	EffectiveNegativePrompt string `json:"effective_negative_prompt"`
}

func (a *App) GenerateImage(params GenerateImageParams) (*GenerateImageResult, error) {
	a.log.UserAction("Generate image (preset_id=%d)", params.PresetID)
	p, err := a.presets.Get(params.PresetID)
	if err != nil {
		a.log.Error("Generate image: preset not found: %s", err)
		return nil, err
	}

	prompt := p.Prompt
	if params.ExtraPrompt != "" {
		prompt = params.ExtraPrompt
	}
	if p.Loras != "" {
		var loras []preset.LoRAEntry
		if err := json.Unmarshal([]byte(p.Loras), &loras); err == nil {
			for _, l := range loras {
				prompt += fmt.Sprintf(" <lora:%s:%g>", l.Name, l.Weight)
			}
		}
	}

	negativePrompt := p.NegativePrompt
	if params.ExtraNegativePrompt != "" {
		negativePrompt = params.ExtraNegativePrompt
	}

	negativePrompt = a.applyKidsNegative(negativePrompt)

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
		Image:                   result.Images[0],
		Parameters:              result.Parameters,
		Info:                    result.Info,
		IsPreview:               isPreview,
		EffectivePrompt:         prompt,
		EffectiveNegativePrompt: negativePrompt,
	}
	a.saveLastImage(result.Images[0], result.Info, isPreview)
	return img, nil
}

type UpscaleImageParams struct {
	ImageBase64 string `json:"image_base64"`
	GenInfo     string `json:"gen_info"`
	PresetID    int64  `json:"preset_id"`
}

type BatchGenerateParams struct {
	PresetID        int64  `json:"preset_id"`
	Prompt          string `json:"prompt"`
	NegativePrompt  string `json:"negative_prompt"`
	Count           int    `json:"count"`
	OutputFolder    string `json:"output_folder"`
}

type BatchProgress struct {
	Current   int    `json:"current"`
	Total     int    `json:"total"`
	FilePath  string `json:"file_path"`
	Status    string `json:"status"`
}

func (a *App) BatchGenerate(params BatchGenerateParams) error {
	if params.Count <= 0 || params.Count > 100 {
		return fmt.Errorf("count must be between 1 and 100")
	}
	if params.OutputFolder == "" {
		return fmt.Errorf("output folder is required")
	}
	if params.Prompt == "" {
		return fmt.Errorf("prompt is required")
	}

	a.batchMu.Lock()
	if a.batchRunning {
		a.batchMu.Unlock()
		return fmt.Errorf("batch generation is already running")
	}
	a.batchRunning = true
	a.batchMu.Unlock()
	defer func() {
		a.batchMu.Lock()
		a.batchRunning = false
		a.batchMu.Unlock()
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
		p, err = a.presets.Get(params.PresetID)
		if err != nil {
			return fmt.Errorf("preset not found: %w", err)
		}
	}

	prompt := params.Prompt
	var filterErr error
	prompt, filterErr = a.filterKidsInput(prompt)
	if filterErr != nil {
		return filterErr
	}
	if p.Loras != "" {
		var loras []preset.LoRAEntry
		if err := json.Unmarshal([]byte(p.Loras), &loras); err == nil {
			for _, l := range loras {
				prompt += fmt.Sprintf(" <lora:%s:%g>", l.Name, l.Weight)
			}
		}
	}

	negativePrompt := params.NegativePrompt
	negativePrompt, filterErr = a.filterKidsInput(negativePrompt)
	if filterErr != nil {
		return filterErr
	}
	negativePrompt = a.applyKidsNegative(negativePrompt)

	if p.ModelName != "" {
		_ = a.sd.SetModel(p.ModelName)
	}
	if p.VAE != "" {
		_ = a.sd.SetVAE(p.VAE)
	}

	samplerName := p.Sampler
	if len(p.ScheduleType) > 0 {
		st := strings.ToUpper(p.ScheduleType[:1]) + p.ScheduleType[1:]
		samplerName = p.Sampler + " " + st
	}

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
		runtime.EventsEmit(a.ctx, "batch:progress", BatchProgress{
			Current: i + 1,
			Total:   params.Count,
			Status:  "generating",
		})

		result, err := a.sd.Txt2Img(sd.Txt2ImgRequest{
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
		})
		if err != nil {
			runtime.EventsEmit(a.ctx, "batch:progress", BatchProgress{
				Current: i + 1,
				Total:   params.Count,
				Status:  fmt.Sprintf("error: image %d failed", i+1),
			})
			return fmt.Errorf("image %d/%d failed: %w", i+1, params.Count, err)
		}
		if len(result.Images) == 0 {
			runtime.EventsEmit(a.ctx, "batch:progress", BatchProgress{
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

		runtime.EventsEmit(a.ctx, "batch:progress", BatchProgress{
			Current:  i + 1,
			Total:    params.Count,
			FilePath: filePath,
			Status:   "saved",
		})
	}

	runtime.EventsEmit(a.ctx, "batch:progress", BatchProgress{
		Current: params.Count,
		Total:   params.Count,
		Status:  "done",
	})
	return nil
}

type TestGenerateParams struct {
	Mode            string   `json:"mode"`
	SelectedIDs     []int64  `json:"selected_ids"`
	SelectedModels  []string `json:"selected_models"`
	Prompt          string   `json:"prompt"`
	NegativePrompt  string   `json:"negative_prompt"`
	Sampler         string   `json:"sampler"`
	ScheduleType    string   `json:"schedule_type"`
	Steps           int      `json:"steps"`
	CfgScale        float64  `json:"cfg_scale"`
	Width           int      `json:"width"`
	Height          int      `json:"height"`
	Seed            *int64   `json:"seed"`
}

type TestGenerateResultItem struct {
	Name           string `json:"name"`
	Image          string `json:"image"`
	Seed           int64  `json:"seed"`
	Error          string `json:"error,omitempty"`
	Sampler        string `json:"sampler"`
	ScheduleType   string `json:"schedule_type"`
	CfgScale       float64 `json:"cfg_scale"`
	ModelName      string `json:"model_name"`
}

func (a *App) TestGenerate(params TestGenerateParams) ([]TestGenerateResultItem, error) {
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
		runtime.EventsEmit(a.ctx, "test:progress", map[string]any{
			"current": idx + 1,
			"total":   totalItems,
			"status":  "generating",
		})

		item := TestGenerateResultItem{}
		p := &preset.Preset{}
		*p = *defaultPreset

		if params.Mode == "presets" {
			id := params.SelectedIDs[idx]
			loaded, err := a.presets.Get(id)
			if err != nil {
				item.Error = fmt.Sprintf("preset not found: %v", err)
				item.Name = fmt.Sprintf("Preset #%d", id)
				results = append(results, item)
				continue
			}
			p = loaded
			item.Name = p.Name
			if p.ModelName != "" {
				_ = a.sd.SetModel(p.ModelName)
			}
			if p.VAE != "" {
				_ = a.sd.SetVAE(p.VAE)
			}
		} else {
			modelTitle := params.SelectedModels[idx]
			_ = a.sd.SetModel(modelTitle)
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
		prompt, filterErr := a.filterKidsInput(prompt)
		if filterErr != nil {
			return nil, fmt.Errorf("generating image: %w", filterErr)
		}
		if p.Loras != "" && params.Mode == "presets" {
			var loras []preset.LoRAEntry
			if err := json.Unmarshal([]byte(p.Loras), &loras); err == nil {
				for _, l := range loras {
					prompt += fmt.Sprintf(" <lora:%s:%g>", l.Name, l.Weight)
				}
			}
		}

		negPrompt := params.NegativePrompt
		if params.Mode == "presets" && p.NegativePrompt != "" {
			if negPrompt != "" {
				negPrompt = p.NegativePrompt + ", " + negPrompt
			} else {
				negPrompt = p.NegativePrompt
			}
		}
		negPrompt = a.applyKidsNegative(negPrompt)

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

		samplerName := sampler
		if scheduleType != "" {
			st := strings.ToUpper(scheduleType[:1]) + scheduleType[1:]
			samplerName = sampler + " " + st
		}

		clipSkip := 1
		if p.ClipSkip != nil {
			clipSkip = *p.ClipSkip
		}
		batchSize := 1
		batchCount := 1

		result, err := a.sd.Txt2Img(sd.Txt2ImgRequest{
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

		runtime.EventsEmit(a.ctx, "test:progress", map[string]any{
			"current": idx + 1,
			"total":   totalItems,
			"status":  "done",
		})
	}

	return results, nil
}

func (a *App) SelectFolder() (string, error) {
	path, err := runtime.OpenDirectoryDialog(a.ctx, runtime.OpenDialogOptions{
		Title: "Select Output Folder",
	})
	if err != nil {
		return "", err
	}
	return path, nil
}

func (a *App) GetPresetForBatch(presetID int64, description string) (*GenerateSDPromptResult, error) {
	if presetID <= 0 {
		return nil, fmt.Errorf("preset is required")
	}

	if description != "" {
		result, err := a.GenerateSDPrompt(GenerateSDPromptParams{
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

	negativePrompt = a.applyKidsNegative(negativePrompt)

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

	negativePrompt = a.applyKidsNegative(negativePrompt)

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
		"kids_cat_violence":         "true",
		"kids_cat_horror":           "true",
		"kids_cat_weapons":          "true",
		"kids_cat_substances":       "true",
		"kids_cat_mature":           "true",
		"rembg_url":                 "",
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
		"kids_cat_violence": true, "kids_cat_horror": true, "kids_cat_weapons": true,
		"kids_cat_substances": true, "kids_cat_mature": true,
		"rembg_url": true,
		"preview_mode": true, "preview_width": true, "preview_height": true,
		"gen_preset_id": true, "gen_action_pose": true, "gen_characters": true,
		"gen_clothing_details": true,
		"gen_environment": true, "gen_lighting": true, "gen_negative": true,
		"gen_extra_prompt": true, "gen_extra_negative": true,
		"gen_description": true, "gen_type_id": true,
		"gen_mode": true, "gen_compound_preset_id": true,
		"batch_preset_id": true, "batch_compound_preset_id": true, "batch_mode": true,
		"batch_prompt": true, "batch_negative": true, "batch_count": true, "batch_output_folder": true,
		"test_mode": true, "test_prompt": true, "test_negative": true,
		"test_sampler": true, "test_schedule_type": true, "test_steps": true,
		"test_cfg_scale": true, "test_width": true, "test_height": true,
		"fi_mode": true, "fi_preset_id": true, "fi_compound_preset_id": true,
		"fi_gen_mode": true, "fi_denoising": true, "fi_extra_negative": true, "fi_analyze_mode": true,
		"theme": true, "file_browser_path": true,
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
	if v, ok := data["rembg_url"]; ok {
		a.rembgClient.SetURL(v)
	}

	var changed []string
	for k := range data {
		if allowed[k] {
			changed = append(changed, k)
		}
	}
	a.log.UserAction("Settings updated: %s", strings.Join(changed, ", "))

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

func (a *App) CreateDescriptionFull(s preset.SavedDescription) (*preset.SavedDescription, error) {
	s.Text = strings.TrimSpace(s.Text)
	if s.Text == "" {
		return nil, nil
	}
	return a.presets.CreateDescriptionFull(&s)
}

func (a *App) UpdateDescription(s preset.SavedDescription) error {
	return a.presets.UpdateDescription(&s)
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

func (a *App) ClearLastImage() {
	os.Remove(filepath.Join(a.dataDir, "last_image.png"))
	os.Remove(filepath.Join(a.dataDir, "last_image.json"))
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

	a.log.UserAction("Image saved: %s", path)
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
	SourceFile             string   `json:"source_file,omitempty"`
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
	paths, err := runtime.OpenMultipleFilesDialog(a.ctx, runtime.OpenDialogOptions{
		Filters: []runtime.FileFilter{
			{DisplayName: "JSON Files", Pattern: "*.json"},
		},
	})
	if err != nil || len(paths) == 0 {
		return nil, err
	}

	var allPresets []PresetData
	for _, path := range paths {
		info, err := os.Stat(path)
		if err != nil {
			continue
		}
		if info.Size() > 10*1024*1024 {
			continue
		}

		jsonBytes, err := os.ReadFile(path)
		if err != nil {
			continue
		}

		var data PresetExportFile
		if err := json.Unmarshal(jsonBytes, &data); err != nil {
			continue
		}
		if data.Version < 1 || data.Version > 2 {
			continue
		}

		fileName := filepath.Base(path)
		for i := range data.Presets {
			data.Presets[i].SourceFile = fileName
		}
		allPresets = append(allPresets, data.Presets...)
	}

	if len(allPresets) == 0 {
		return nil, fmt.Errorf("no presets found in selected files")
	}

	return &ImportPreview{
		Presets: allPresets,
		Total:   len(allPresets),
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

var reCyrillicCheck = regexp.MustCompile(`[а-яА-ЯёЁ]`)

func containsCyrillic(s string) bool {
	return reCyrillicCheck.MatchString(s)
}

func extractTagsFromRaw(raw string) string {
	var best string
	for _, m := range reQuotedStrings.FindAllString(raw, -1) {
		m = strings.Trim(m, `"`)
		m = strings.TrimSpace(m)
		if len(m) > len(best) && !strings.Contains(m, `"`) && !containsCyrillic(m) && (strings.Contains(m, ", ") || strings.Contains(m, "quality")) {
			best = m
		}
	}
	return best
}

func extractNegativeFromRaw(raw string) string {
	jsonRaw := extractJSON(raw)
	if jsonRaw == "" {
		return ""
	}
	var obj map[string]string
	if err := json.Unmarshal([]byte(jsonRaw), &obj); err != nil {
		return ""
	}
	if np, ok := obj["negative_prompt"]; ok && !containsCyrillic(np) {
		return np
	}
	return ""
}

var reJunkLabels = regexp.MustCompile(`(?i)\b(BASE (POSITIVE|NEGATIVE) PROMPT|USER DESCRIPTION|USER NEGATIVE|MERGED PROMPT|NEGATIVE[_ ]PROMPT|Translation of non-English text|translates to|Merged Prompt)\s*:\s*`)
var reJSONFragments = regexp.MustCompile(`\{[^{}]*"(prompt|negative_prompt)"[^{}]*\}`)
var reQuotedStrings = regexp.MustCompile(`"[^"]{0,500}"`)
var reCyrillic = regexp.MustCompile(`[а-яА-ЯёЁ]+[^,(\[<]*,?`)

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

func stripJunk(s string) string {
	if s == "" {
		return s
	}
	for reJunkLabels.MatchString(s) {
		s = reJunkLabels.ReplaceAllString(s, "")
	}
	for reJSONFragments.MatchString(s) {
		s = reJSONFragments.ReplaceAllString(s, "")
	}
	for strings.Contains(s, `"prompt"`) || strings.Contains(s, `"negative_prompt"`) {
		s = reQuotedStrings.ReplaceAllString(s, "")
	}
	for strings.Contains(s, "  ") {
		s = strings.ReplaceAll(s, "  ", " ")
	}
	s = strings.ReplaceAll(s, ", ,", ",")
	s = strings.ReplaceAll(s, ",,", ",")
	s = strings.Trim(s, " ,.\n\r")
	return s
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
	s = strings.TrimRight(s, " ,.")
	return s
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

func (a *App) ListCompoundPresets() ([]preset.CompoundPreset, error) {
	items, err := a.presets.ListCompoundPresets()
	if err != nil {
		return nil, err
	}
	for i := range items {
		for j := range items[i].Steps {
			p, err := a.presets.Get(items[i].Steps[j].PresetID)
			if err == nil {
				items[i].Steps[j].Preset = p
			}
		}
	}
	return items, nil
}

func (a *App) GetCompoundPreset(id int64) (*preset.CompoundPreset, error) {
	cp, err := a.presets.GetCompoundPreset(id)
	if err != nil {
		return nil, err
	}
	for i := range cp.Steps {
		p, err := a.presets.Get(cp.Steps[i].PresetID)
		if err == nil {
			cp.Steps[i].Preset = p
		}
	}
	return cp, nil
}

func (a *App) CreateCompoundPreset(cp preset.CompoundPreset) (*preset.CompoundPreset, error) {
	if cp.Name == "" {
		return nil, fmt.Errorf("name is required")
	}
	if len(cp.Steps) == 0 {
		return nil, fmt.Errorf("at least one step is required")
	}
	if err := a.presets.CreateCompoundPreset(&cp); err != nil {
		return nil, err
	}
	return a.GetCompoundPreset(cp.ID)
}

func (a *App) UpdateCompoundPreset(cp preset.CompoundPreset) (*preset.CompoundPreset, error) {
	if cp.ID <= 0 {
		return nil, fmt.Errorf("id is required")
	}
	if cp.Name == "" {
		return nil, fmt.Errorf("name is required")
	}
	if len(cp.Steps) == 0 {
		return nil, fmt.Errorf("at least one step is required")
	}
	if err := a.presets.UpdateCompoundPreset(&cp); err != nil {
		return nil, err
	}
	return a.GetCompoundPreset(cp.ID)
}

func (a *App) DeleteCompoundPreset(id int64) error {
	return a.presets.DeleteCompoundPreset(id)
}

type GenerateCompoundImageParams struct {
	CompoundPresetID    int64  `json:"compound_preset_id"`
	ExtraPrompt         string `json:"extra_prompt"`
	ExtraNegativePrompt string `json:"extra_negative_prompt"`
}

func (a *App) GenerateCompoundImage(params GenerateCompoundImageParams) (*GenerateImageResult, error) {
	cp, err := a.presets.GetCompoundPreset(params.CompoundPresetID)
	if err != nil {
		return nil, fmt.Errorf("compound preset not found: %w", err)
	}
	if len(cp.Steps) == 0 {
		return nil, fmt.Errorf("compound preset has no steps")
	}

	var lastImage string
	var lastInfo json.RawMessage

	for stepIdx, step := range cp.Steps {
		p, err := a.presets.Get(step.PresetID)
		if err != nil {
			return nil, fmt.Errorf("step %d: preset not found: %w", stepIdx+1, err)
		}

		runtime.EventsEmit(a.ctx, "compound:progress", map[string]any{
			"current": stepIdx + 1,
			"total":   len(cp.Steps),
			"status":  "generating",
			"step":    stepIdx + 1,
		})

		prompt := p.Prompt
		if params.ExtraPrompt != "" {
			prompt = params.ExtraPrompt
		}
		if p.Loras != "" {
			var loras []preset.LoRAEntry
			if json.Unmarshal([]byte(p.Loras), &loras) == nil {
				for _, l := range loras {
					prompt += fmt.Sprintf(" <lora:%s:%g>", l.Name, l.Weight)
				}
			}
		}

		negativePrompt := p.NegativePrompt
		if params.ExtraNegativePrompt != "" {
			negativePrompt = params.ExtraNegativePrompt
		}
		negativePrompt = a.applyKidsNegative(negativePrompt)

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
			result, err := a.sd.Txt2Img(sd.Txt2ImgRequest{
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
				return nil, fmt.Errorf("step %d (txt2img): %w", stepIdx+1, err)
			}
			if len(result.Images) == 0 {
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
			result, err := a.sd.Img2Img(sd.Img2ImgRequest{
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
				return nil, fmt.Errorf("step %d (img2img): %w", stepIdx+1, err)
			}
			if len(result.Images) == 0 {
				return nil, fmt.Errorf("step %d: no image returned", stepIdx+1)
			}
			lastImage = result.Images[0]
			lastInfo = result.Info
		}
	}

	runtime.EventsEmit(a.ctx, "compound:progress", map[string]any{
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
	a.saveLastImage(lastImage, lastInfo, false)
	return img, nil
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

func (a *App) GenerateFromImage(params GenerateFromImageParams) (*GenerateImageResult, error) {
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

	tags := params.Tags
	if tags == "" {
		var err error
		tags, err = a.AnalyzeImage(params.ImageBase64)
		if err != nil {
			return nil, fmt.Errorf("image analysis failed: %w", err)
		}
	}

	var filterErr error
	tags, filterErr = a.filterKidsInput(tags)
	if filterErr != nil {
		return nil, filterErr
	}
	params.ExtraNegativePrompt, filterErr = a.filterKidsInput(params.ExtraNegativePrompt)
	if filterErr != nil {
		return nil, filterErr
	}

	if params.GenMode == "compound" {
		return a.generateFromImageCompound(params, tags)
	}

	if params.RemoveObject {
		return a.generateRemoveObject(params, tags)
	}

	p, err := a.presets.Get(params.PresetID)
	if err != nil {
		return nil, fmt.Errorf("preset not found: %w", err)
	}

	sdPromptInstruction := config.DefaultSDPromptInstruction
	if v, err := a.presets.GetSetting("sd_prompt_instruction"); err == nil && v != "" {
		sdPromptInstruction = v
	}

	systemPrompt := a.applyKidsSystemPrompt(sdPromptInstruction)

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

	userParts := []string{
		"BASE POSITIVE PROMPT: " + p.Prompt,
		"BASE NEGATIVE PROMPT: " + p.NegativePrompt,
		"USER DESCRIPTION (extracted from image): " + tags,
	}
	if params.ExtraNegativePrompt != "" {
		userParts = append(userParts, "USER NEGATIVE: "+params.ExtraNegativePrompt)
	}
	userMessage := strings.Join(userParts, "\n\n")

	raw, err := a.llm.GenerateSDPrompt(systemPrompt, userMessage, p.PresetType, generateModel, maxTokens)
	if err != nil {
		return nil, err
	}

	var promptResult GenerateSDPromptResult
	jsonRaw := extractJSON(raw)
	if err := json.Unmarshal([]byte(jsonRaw), &promptResult); err != nil {
		promptResult = GenerateSDPromptResult{
			Prompt:         truncateRepetitive(raw, 1000),
			NegativePrompt: p.NegativePrompt,
		}
	}

	if containsCyrillic(promptResult.Prompt) {
		promptResult.Prompt = extractTagsFromRaw(raw)
	}
	if containsCyrillic(promptResult.NegativePrompt) {
		promptResult.NegativePrompt = extractNegativeFromRaw(raw)
	}

	extractEmbeddedNegative(&promptResult)
	promptResult.Prompt = stripJunk(promptResult.Prompt)
	promptResult.Prompt = truncateRepetitive(promptResult.Prompt, 1000)
	promptResult.NegativePrompt = stripJunk(promptResult.NegativePrompt)
	promptResult.NegativePrompt = truncateRepetitive(promptResult.NegativePrompt, 500)

	promptResult.Prompt = a.filterKidsOutput(promptResult.Prompt)
	promptResult.NegativePrompt = a.filterKidsOutput(promptResult.NegativePrompt)

	prompt := promptResult.Prompt
	if p.Loras != "" {
		var loras []preset.LoRAEntry
		if json.Unmarshal([]byte(p.Loras), &loras) == nil {
			for _, l := range loras {
				prompt += fmt.Sprintf(" <lora:%s:%g>", l.Name, l.Weight)
			}
		}
	}

	negativePrompt := promptResult.NegativePrompt
	if params.ExtraNegativePrompt != "" {
		negativePrompt += ", " + params.ExtraNegativePrompt
	}
	negativePrompt = a.applyKidsNegative(negativePrompt)

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

	clipSkip := 1
	if p.ClipSkip != nil {
		clipSkip = *p.ClipSkip
	}
	batchSize := 1
	batchCount := 1

	if params.Mode == "img2img" || params.Mode == "inpaint" {
		denoising := params.DenoisingStrength
		if denoising <= 0 {
			denoising = 0.5
		}
		maskBlur := params.MaskBlur
		if maskBlur <= 0 {
			maskBlur = 4
		}
		result, err := a.sd.Img2Img(sd.Img2ImgRequest{
			InitImages:            []string{params.ImageBase64},
			Prompt:                prompt,
			NegativePrompt:        negativePrompt,
			SamplerName:           samplerName,
			Scheduler:             p.ScheduleType,
			Steps:                 p.Steps,
			CfgScale:              p.CfgScale,
			Width:                 p.Width,
			Height:                p.Height,
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
			return nil, fmt.Errorf("no image generated (%s)", params.Mode)
		}
		img := &GenerateImageResult{
			Image:                   result.Images[0],
			Info:                    result.Info,
			EffectivePrompt:         prompt,
			EffectiveNegativePrompt: negativePrompt,
		}
		a.saveLastImage(result.Images[0], result.Info, false)
		return img, nil
	}

	width := p.Width
	height := p.Height
	hiresFix := p.HiresFix

	isPreview := false
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

	denoisingStrength := p.DenoisingStrength
	if denoisingStrength == nil && p.HiresFix != nil && *p.HiresFix {
		ds := 0.5
		if p.HiresDenoisingStrength != nil {
			ds = *p.HiresDenoisingStrength
		}
		denoisingStrength = &ds
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
	a.saveLastImage(result.Images[0], result.Info, isPreview)
	return img, nil
}

func (a *App) generateRemoveObject(params GenerateFromImageParams, removeDesc string) (*GenerateImageResult, error) {
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

	if params.PresetID > 0 {
		p, err := a.presets.Get(params.PresetID)
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
			width = p.Width
			height = p.Height
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
		_ = a.sd.SetModel(modelName)
	}
	if vae != "" {
		_ = a.sd.SetVAE(vae)
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

	result, err := a.sd.Img2Img(sd.Img2ImgRequest{
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
		InpaintFullResPadding: 32,
		DoNotSaveImages:       true,
		DoNotSaveGrid:         true,
	})
	if err != nil {
		return nil, err
	}
	if len(result.Images) == 0 {
		return nil, fmt.Errorf("no image generated (remove object)")
	}

	img := &GenerateImageResult{
		Image:                   result.Images[0],
		Info:                    result.Info,
		EffectivePrompt:         prompt,
		EffectiveNegativePrompt: negativePrompt,
	}
	a.saveLastImage(result.Images[0], result.Info, false)
	return img, nil
}

func (a *App) generateFromImageCompound(params GenerateFromImageParams, tags string) (*GenerateImageResult, error) {
	cp, err := a.presets.GetCompoundPreset(params.CompoundPresetID)
	if err != nil {
		return nil, fmt.Errorf("compound preset not found: %w", err)
	}
	if len(cp.Steps) == 0 {
		return nil, fmt.Errorf("compound preset has no steps")
	}

	firstPreset, err := a.presets.Get(cp.Steps[0].PresetID)
	if err != nil {
		return nil, fmt.Errorf("step 1: preset not found: %w", err)
	}

	sdPromptInstruction := config.DefaultSDPromptInstruction
	if v, err := a.presets.GetSetting("sd_prompt_instruction"); err == nil && v != "" {
		sdPromptInstruction = v
	}

	systemPrompt := a.applyKidsSystemPrompt(sdPromptInstruction)

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

	userParts := []string{
		"BASE POSITIVE PROMPT: " + firstPreset.Prompt,
		"BASE NEGATIVE PROMPT: " + firstPreset.NegativePrompt,
		"USER DESCRIPTION (extracted from image): " + tags,
	}
	if params.ExtraNegativePrompt != "" {
		userParts = append(userParts, "USER NEGATIVE: "+params.ExtraNegativePrompt)
	}
	userMessage := strings.Join(userParts, "\n\n")

	raw, err := a.llm.GenerateSDPrompt(systemPrompt, userMessage, firstPreset.PresetType, generateModel, maxTokens)
	if err != nil {
		return nil, err
	}

	var promptResult GenerateSDPromptResult
	jsonRaw := extractJSON(raw)
	if err := json.Unmarshal([]byte(jsonRaw), &promptResult); err != nil {
		promptResult = GenerateSDPromptResult{
			Prompt:         truncateRepetitive(raw, 1000),
			NegativePrompt: firstPreset.NegativePrompt,
		}
	}

	if containsCyrillic(promptResult.Prompt) {
		promptResult.Prompt = extractTagsFromRaw(raw)
	}
	extractEmbeddedNegative(&promptResult)
	promptResult.Prompt = stripJunk(promptResult.Prompt)
	promptResult.Prompt = truncateRepetitive(promptResult.Prompt, 1000)
	promptResult.NegativePrompt = stripJunk(promptResult.NegativePrompt)
	promptResult.NegativePrompt = truncateRepetitive(promptResult.NegativePrompt, 500)

	promptResult.Prompt = a.filterKidsOutput(promptResult.Prompt)
	promptResult.NegativePrompt = a.filterKidsOutput(promptResult.NegativePrompt)

	var lastImage string
	var lastInfo json.RawMessage

	for stepIdx, step := range cp.Steps {
		p, err := a.presets.Get(step.PresetID)
		if err != nil {
			return nil, fmt.Errorf("step %d: preset not found: %w", stepIdx+1, err)
		}

		runtime.EventsEmit(a.ctx, "fromimage:progress", map[string]any{
			"current": stepIdx + 1,
			"total":   len(cp.Steps),
			"status":  "generating",
		})

		prompt := p.Prompt
		if stepIdx == 0 {
			prompt = promptResult.Prompt
		}
		if p.Loras != "" {
			var loras []preset.LoRAEntry
			if json.Unmarshal([]byte(p.Loras), &loras) == nil {
				for _, l := range loras {
					prompt += fmt.Sprintf(" <lora:%s:%g>", l.Name, l.Weight)
				}
			}
		}

		negativePrompt := promptResult.NegativePrompt
		if params.ExtraNegativePrompt != "" {
			negativePrompt += ", " + params.ExtraNegativePrompt
		}
		negativePrompt = a.applyKidsNegative(negativePrompt)

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
			result, err := a.sd.Img2Img(sd.Img2ImgRequest{
				InitImages:        []string{params.ImageBase64},
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
				return nil, fmt.Errorf("step %d (img2img): %w", stepIdx+1, err)
			}
			if len(result.Images) == 0 {
				return nil, fmt.Errorf("step %d: no image returned", stepIdx+1)
			}
			lastImage = result.Images[0]
			lastInfo = result.Info
		} else if stepIdx == 0 {
			result, err := a.sd.Txt2Img(sd.Txt2ImgRequest{
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
				return nil, fmt.Errorf("step %d (txt2img): %w", stepIdx+1, err)
			}
			if len(result.Images) == 0 {
				return nil, fmt.Errorf("step %d: no image returned", stepIdx+1)
			}
			lastImage = result.Images[0]
			lastInfo = result.Info
		} else {
			denoising := step.DenoisingStrength
			if denoising <= 0 {
				denoising = 0.5
			}
			result, err := a.sd.Img2Img(sd.Img2ImgRequest{
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
				return nil, fmt.Errorf("step %d (img2img): %w", stepIdx+1, err)
			}
			if len(result.Images) == 0 {
				return nil, fmt.Errorf("step %d: no image returned", stepIdx+1)
			}
			lastImage = result.Images[0]
			lastInfo = result.Info
		}
	}

	runtime.EventsEmit(a.ctx, "fromimage:progress", map[string]any{
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
	a.saveLastImage(lastImage, lastInfo, false)
	return img, nil
}

type BatchCompoundGenerateParams struct {
	CompoundPresetID   int64  `json:"compound_preset_id"`
	ExtraPrompt        string `json:"extra_prompt"`
	ExtraNegativePrompt string `json:"extra_negative_prompt"`
	Count              int    `json:"count"`
	OutputFolder       string `json:"output_folder"`
}

func (a *App) BatchCompoundGenerate(params BatchCompoundGenerateParams) error {
	if params.Count <= 0 || params.Count > 100 {
		return fmt.Errorf("count must be between 1 and 100")
	}
	if params.OutputFolder == "" {
		return fmt.Errorf("output folder is required")
	}

	a.batchMu.Lock()
	if a.batchRunning {
		a.batchMu.Unlock()
		return fmt.Errorf("batch generation is already running")
	}
	a.batchRunning = true
	a.batchMu.Unlock()
	defer func() {
		a.batchMu.Lock()
		a.batchRunning = false
		a.batchMu.Unlock()
	}()

	if err := os.MkdirAll(params.OutputFolder, 0755); err != nil {
		return fmt.Errorf("create output folder: %w", err)
	}

	cp, err := a.presets.GetCompoundPreset(params.CompoundPresetID)
	if err != nil {
		return fmt.Errorf("compound preset not found: %w", err)
	}
	if len(cp.Steps) == 0 {
		return fmt.Errorf("compound preset has no steps")
	}

	for batchIdx := 0; batchIdx < params.Count; batchIdx++ {
		runtime.EventsEmit(a.ctx, "batch:progress", map[string]any{
			"current": batchIdx + 1,
			"total":   params.Count,
			"status":  "generating",
		})

		var lastImage string

		for stepIdx, step := range cp.Steps {
			p, err := a.presets.Get(step.PresetID)
			if err != nil {
				return fmt.Errorf("step %d: preset not found: %w", stepIdx+1, err)
			}

			prompt := p.Prompt
			if params.ExtraPrompt != "" {
				prompt = params.ExtraPrompt
			}
			if p.Loras != "" {
				var loras []preset.LoRAEntry
				if json.Unmarshal([]byte(p.Loras), &loras) == nil {
					for _, l := range loras {
						prompt += fmt.Sprintf(" <lora:%s:%g>", l.Name, l.Weight)
					}
				}
			}

			negativePrompt := p.NegativePrompt
			if params.ExtraNegativePrompt != "" {
				negativePrompt = params.ExtraNegativePrompt
			}
			var filterErr error
			prompt, filterErr = a.filterKidsInput(prompt)
			if filterErr != nil {
				return fmt.Errorf("step %d: %w", stepIdx+1, filterErr)
			}
			negativePrompt, filterErr = a.filterKidsInput(negativePrompt)
			if filterErr != nil {
				return fmt.Errorf("step %d: %w", stepIdx+1, filterErr)
			}
			negativePrompt = a.applyKidsNegative(negativePrompt)

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
				result, err := a.sd.Txt2Img(sd.Txt2ImgRequest{
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
					return fmt.Errorf("batch %d, step %d (txt2img): %w", batchIdx+1, stepIdx+1, err)
				}
				if len(result.Images) == 0 {
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
				result, err := a.sd.Img2Img(sd.Img2ImgRequest{
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
					return fmt.Errorf("batch %d, step %d (img2img): %w", batchIdx+1, stepIdx+1, err)
				}
				if len(result.Images) == 0 {
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

		runtime.EventsEmit(a.ctx, "batch:progress", map[string]any{
			"current":   batchIdx + 1,
			"total":     params.Count,
			"file_path": filePath,
			"status":    "generating",
		})
	}

	runtime.EventsEmit(a.ctx, "batch:progress", map[string]any{
		"current": params.Count,
		"total":   params.Count,
		"status":  "done",
	})
	return nil
}

type TestCompoundGenerateParams struct {
	SelectedIDs        []int64 `json:"selected_ids"`
	Prompt             string  `json:"prompt"`
	NegativePrompt     string  `json:"negative_prompt"`
}

func (a *App) TestCompoundGenerate(params TestCompoundGenerateParams) ([]TestGenerateResultItem, error) {
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
		runtime.EventsEmit(a.ctx, "test:progress", map[string]any{
			"current": idx + 1,
			"total":   totalItems,
			"status":  "generating",
		})

		item := TestGenerateResultItem{}

		cp, err := a.presets.GetCompoundPreset(compoundID)
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
			p, err := a.presets.Get(step.PresetID)
			if err != nil {
				item.Error = fmt.Sprintf("step %d: preset not found", stepIdx+1)
				break
			}

			prompt := params.Prompt
			prompt, filterErr := a.filterKidsInput(prompt)
			if filterErr != nil {
				item.Error = filterErr.Error()
				break
			}
			if p.Loras != "" {
				var loras []preset.LoRAEntry
				if json.Unmarshal([]byte(p.Loras), &loras) == nil {
					for _, l := range loras {
						prompt += fmt.Sprintf(" <lora:%s:%g>", l.Name, l.Weight)
					}
				}
			}

			negPrompt := params.NegativePrompt
			if p.NegativePrompt != "" {
				if negPrompt != "" {
					negPrompt = p.NegativePrompt + ", " + negPrompt
				} else {
					negPrompt = p.NegativePrompt
				}
			}
			negPrompt = a.applyKidsNegative(negPrompt)

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
				result, err := a.sd.Txt2Img(sd.Txt2ImgRequest{
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
				result, err := a.sd.Img2Img(sd.Img2ImgRequest{
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

		runtime.EventsEmit(a.ctx, "test:progress", map[string]any{
			"current": idx + 1,
			"total":   totalItems,
			"status":  "done",
		})
	}

	return results, nil
}

// --- Multi-Pass Scene Generation ---

type DecomposeSceneParams struct {
	Description string `json:"description"`
	PresetID    int64  `json:"preset_id"`
}

func (a *App) DecomposeScene(params DecomposeSceneParams) (*compositor.Scene, error) {
	a.log.UserAction("Decompose scene: %s", truncate(params.Description, 80))
	if params.Description == "" {
		return nil, fmt.Errorf("description is required")
	}
	if params.PresetID <= 0 {
		return nil, fmt.Errorf("preset is required")
	}

	p, err := a.presets.Get(params.PresetID)
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

	raw, err := a.llm.Chat(generateModel, systemPrompt, userMessage, 0.4, maxTokens)
	if err != nil {
		return nil, fmt.Errorf("LLM decomposition failed: %w", err)
	}

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

func (a *App) GenerateMultiPass(scene compositor.Scene) (*compositor.MultiPassResult, error) {
	a.log.UserAction("Multi-pass generation: %d characters", len(scene.Characters))

	emit := func(progress compositor.MultiPassProgress) {
		runtime.EventsEmit(a.ctx, "multipass:progress", progress)
		switch progress.Step {
		case "background":
			a.log.Info("Generating background...")
		case "character":
			a.log.Info("Generating character %d/%d", progress.Character, progress.Total)
		case "rembg":
			a.log.Info("Removing background (character %d/%d)", progress.Character, progress.Total)
		case "done":
			a.log.Info("Multi-pass generation complete")
		}
	}

	rembgURL, _ := a.presets.GetSetting("rembg_url")
	var rembgIf compositor.RembgClient
	if rembgURL != "" {
		a.rembgClient.SetURL(rembgURL)
		rembgIf = a.rembgClient
		a.log.Debug("Rembg enabled: %s", rembgURL)
	} else {
		a.log.Warn("Rembg not configured, using Go-based background removal")
	}

	c := compositor.New(a.sd, rembgIf, a.presets, emit)
	result, err := c.GenerateScene(scene)
	if err != nil {
		a.log.Error("Multi-pass failed: %s", err)
		return nil, err
	}

	if result.Image != "" {
		a.saveLastImage(result.Image, nil, false)
	}

	return result, nil
}

func (a *App) CheckRembg() error {
	rembgURL, _ := a.presets.GetSetting("rembg_url")
	if rembgURL == "" {
		return fmt.Errorf("rembg URL not configured")
	}
	a.rembgClient.SetURL(rembgURL)
	err := a.rembgClient.HealthCheck()
	if err != nil {
		a.log.Error("Rembg check failed: %s", err)
	} else {
		a.log.Info("Rembg connected: %s", rembgURL)
	}
	return err
}

func (a *App) ListSavedScenes() ([]preset.SavedScene, error) {
	items, err := a.presets.ListSavedScenes()
	if err != nil {
		return nil, err
	}
	if items == nil {
		items = []preset.SavedScene{}
	}
	return items, nil
}

func (a *App) GetSavedScene(id int64) (*preset.SavedScene, error) {
	return a.presets.GetSavedScene(id)
}

func (a *App) SaveScene(s preset.SavedScene) (*preset.SavedScene, error) {
	if strings.TrimSpace(s.Name) == "" {
		return nil, fmt.Errorf("scene name is required")
	}
	if s.SceneJSON == "" {
		return nil, fmt.Errorf("scene data is required")
	}
	if err := a.presets.CreateSavedScene(&s); err != nil {
		return nil, err
	}
	return &s, nil
}

func (a *App) UpdateSavedScene(s preset.SavedScene) (*preset.SavedScene, error) {
	if s.ID <= 0 {
		return nil, fmt.Errorf("invalid scene ID")
	}
	if err := a.presets.UpdateSavedScene(&s); err != nil {
		return nil, err
	}
	return &s, nil
}

func (a *App) DeleteSavedScene(id int64) error {
	return a.presets.DeleteSavedScene(id)
}

func truncate(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n] + "..."
}

// --- Export ---

type ExportImageParams struct {
	ImageBase64   string `json:"image_base64"`
	Format        string `json:"format"`
	Width         int    `json:"width"`
	Height        int    `json:"height"`
	LockRatio     bool   `json:"lock_ratio"`
	Quality       int    `json:"quality"`
	Interpolation string `json:"interpolation"`
	Filename      string `json:"filename"`
}

func (a *App) ExportImage(params ExportImageParams) (string, error) {
	if params.ImageBase64 == "" {
		return "", fmt.Errorf("no image provided")
	}

	switch params.Format {
	case "png", "jpeg", "webp":
	default:
		return "", fmt.Errorf("unsupported format: %s", params.Format)
	}
	switch params.Interpolation {
	case "nearest", "linear", "lanczos", "":
	default:
		return "", fmt.Errorf("unsupported interpolation: %s", params.Interpolation)
	}

	const maxBase64Len = 22 * 1024 * 1024 // ~16 MB decoded
	if len(params.ImageBase64) > maxBase64Len {
		return "", fmt.Errorf("image too large (max 16 MB)")
	}

	imgData, err := base64.StdEncoding.DecodeString(params.ImageBase64)
	if err != nil {
		return "", fmt.Errorf("decode base64: %w", err)
	}

	img, _, err := image.Decode(bytes.NewReader(imgData))
	if err != nil {
		return "", fmt.Errorf("decode image: %w", err)
	}

	const maxDim = 8192
	origBounds := img.Bounds()
	origW := origBounds.Dx()
	origH := origBounds.Dy()
	targetW := params.Width
	targetH := params.Height

	if targetW == 0 && targetH == 0 {
		targetW = origW
		targetH = origH
	} else if params.LockRatio {
		if targetW > 0 && targetH == 0 {
			longSide := float64(targetW)
			if origH > origW {
				ratio := longSide / float64(origH)
				targetW = int(float64(origW) * ratio)
				targetH = int(longSide)
			} else {
				ratio := longSide / float64(origW)
				targetH = int(float64(origH) * ratio)
			}
		} else if targetH > 0 && targetW == 0 {
			longSide := float64(targetH)
			if origW > origH {
				ratio := longSide / float64(origW)
				targetH = int(float64(origH) * ratio)
				targetW = int(longSide)
			} else {
				ratio := longSide / float64(origH)
				targetW = int(float64(origW) * ratio)
			}
		} else {
			ratio := math.Min(float64(targetW)/float64(origW), float64(targetH)/float64(origH))
			ratio = math.Min(ratio, 1.0)
			targetW = int(float64(origW) * ratio)
			targetH = int(float64(origH) * ratio)
		}
	}

	if targetW > maxDim {
		targetW = maxDim
	}
	if targetH > maxDim {
		targetH = maxDim
	}
	if targetW < 1 {
		targetW = 1
	}
	if targetH < 1 {
		targetH = 1
	}

	var result image.Image
	if targetW != origW || targetH != origH {
		dst := image.NewRGBA(image.Rect(0, 0, targetW, targetH))
		var scaler xdraw.Scaler
		switch params.Interpolation {
		case "nearest":
			scaler = xdraw.NearestNeighbor
		case "linear":
			scaler = xdraw.BiLinear
		case "lanczos", "":
			scaler = xdraw.CatmullRom
		default:
			scaler = xdraw.CatmullRom
		}
		scaler.Scale(dst, dst.Bounds(), img, origBounds, xdraw.Over, nil)
		result = dst
	} else {
		result = img
	}

	if params.Quality <= 0 {
		params.Quality = 90
	}

	ext := "." + params.Format
	if params.Filename == "" {
		params.Filename = "export_" + time.Now().Format("20060102_150405") + ext
	}
	if !strings.HasSuffix(strings.ToLower(params.Filename), ext) {
		params.Filename += ext
	}

	filterName := "PNG Image"
	filterPattern := "*.png"
	switch params.Format {
	case "jpeg":
		filterName = "JPEG Image"
		filterPattern = "*.jpg"
	case "webp":
		filterName = "WebP Image"
		filterPattern = "*.webp"
	}

	path, err := runtime.SaveFileDialog(a.ctx, runtime.SaveDialogOptions{
		DefaultFilename: params.Filename,
		Filters: []runtime.FileFilter{
			{DisplayName: filterName, Pattern: filterPattern},
		},
	})
	if err != nil || path == "" {
		return "", err
	}

	var buf bytes.Buffer
	switch params.Format {
	case "jpeg":
		err = jpeg.Encode(&buf, result, &jpeg.Options{Quality: params.Quality})
	case "webp":
		webpData, encErr := webp.EncodeRGBA(result, float32(params.Quality))
		if encErr != nil {
			return "", fmt.Errorf("encode webp: %w", encErr)
		}
		buf.Write(webpData)
	default:
		err = png.Encode(&buf, result)
	}
	if err != nil {
		return "", fmt.Errorf("encode %s: %w", params.Format, err)
	}

	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return "", err
	}
	if err := os.WriteFile(path, buf.Bytes(), 0o644); err != nil {
		return "", err
	}

	return path, nil
}

func (a *App) ListExportPresets() ([]preset.ExportPreset, error) {
	return a.presets.ListExportPresets()
}

func (a *App) SaveExportPreset(ep preset.ExportPreset) (*preset.ExportPreset, error) {
	if ep.ID > 0 {
		if err := a.presets.UpdateExportPreset(&ep); err != nil {
			return nil, err
		}
	} else {
		if err := a.presets.CreateExportPreset(&ep); err != nil {
			return nil, err
		}
	}
	return &ep, nil
}

func (a *App) DeleteExportPreset(id int64) error {
	return a.presets.DeleteExportPreset(id)
}

type FileEntry struct {
	Name    string `json:"name"`
	Path    string `json:"path"`
	IsDir   bool   `json:"is_dir"`
	Size    int64  `json:"size"`
	ModTime string `json:"mod_time"`
}

var imageExts = map[string]bool{
	".png":  true,
	".jpg":  true,
	".jpeg": true,
	".webp": true,
}

func (a *App) BrowseDirectory(dirPath string) ([]FileEntry, error) {
	if dirPath == "" {
		return []FileEntry{}, nil
	}
	entries, err := os.ReadDir(dirPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read directory: %w", err)
	}
	var result []FileEntry
	for _, e := range entries {
		name := e.Name()
		ext := strings.ToLower(filepath.Ext(name))
		if e.IsDir() {
			result = append(result, FileEntry{
				Name:  name,
				Path:  filepath.Join(dirPath, name),
				IsDir: true,
			})
			continue
		}
		if !imageExts[ext] {
			continue
		}
		info, err := e.Info()
		if err != nil {
			continue
		}
		result = append(result, FileEntry{
			Name:    name,
			Path:    filepath.Join(dirPath, name),
			IsDir:   false,
			Size:    info.Size(),
			ModTime: info.ModTime().Format("2006-01-02 15:04"),
		})
	}
	return result, nil
}

func (a *App) ReadFileAsBase64(filePath string) (string, error) {
	if filePath == "" {
		return "", nil
	}
	ext := strings.ToLower(filepath.Ext(filePath))
	if !imageExts[ext] {
		return "", fmt.Errorf("unsupported file type: %s", ext)
	}
	info, err := os.Stat(filePath)
	if err != nil {
		return "", fmt.Errorf("failed to read file: %w", err)
	}
	if info.Size() > 16*1024*1024 {
		return "", fmt.Errorf("image too large (max 16 MB)")
	}
	data, err := os.ReadFile(filePath)
	if err != nil {
		return "", fmt.Errorf("failed to read file: %w", err)
	}
	return base64.StdEncoding.EncodeToString(data), nil
}

func (a *App) SelectBrowserFolder() (string, error) {
	path, err := runtime.OpenDirectoryDialog(a.ctx, runtime.OpenDialogOptions{
		Title: "Select Image Folder",
	})
	if err != nil {
		return "", err
	}
	if path != "" {
		_ = a.presets.SetSetting("file_browser_path", path)
	}
	return path, nil
}

func (a *App) SetLastImage(base64Data string) error {
	if base64Data == "" {
		return nil
	}
	if len(base64Data) > 22*1024*1024 {
		return fmt.Errorf("image too large (max 16 MB)")
	}
	a.saveLastImage(base64Data, nil, false)
	return nil
}

func (a *App) ReadThumbnail(filePath string) (string, error) {
	if filePath == "" {
		return "", nil
	}
	ext := strings.ToLower(filepath.Ext(filePath))
	if !imageExts[ext] {
		return "", fmt.Errorf("unsupported file type: %s", ext)
	}
	info, err := os.Stat(filePath)
	if err != nil {
		return "", fmt.Errorf("failed to read file: %w", err)
	}
	if info.Size() > 16*1024*1024 {
		return "", fmt.Errorf("image too large (max 16 MB)")
	}
	data, err := os.ReadFile(filePath)
	if err != nil {
		return "", fmt.Errorf("failed to read file: %w", err)
	}

	img, _, err := image.Decode(bytes.NewReader(data))
	if err != nil {
		return "", fmt.Errorf("decode image: %w", err)
	}

	const thumbSize = 256
	origW := img.Bounds().Dx()
	origH := img.Bounds().Dy()
	if origW <= thumbSize && origH <= thumbSize {
		return base64.StdEncoding.EncodeToString(data), nil
	}

	ratio := math.Min(float64(thumbSize)/float64(origW), float64(thumbSize)/float64(origH))
	tw := int(float64(origW) * ratio)
	th := int(float64(origH) * ratio)
	if tw < 1 {
		tw = 1
	}
	if th < 1 {
		th = 1
	}

	dst := image.NewRGBA(image.Rect(0, 0, tw, th))
	xdraw.CatmullRom.Scale(dst, dst.Bounds(), img, img.Bounds(), xdraw.Over, nil)

	var buf bytes.Buffer
	switch ext {
	case ".jpg", ".jpeg":
		err = jpeg.Encode(&buf, dst, &jpeg.Options{Quality: 80})
	case ".webp":
		err = webp.Encode(&buf, dst, &webp.Options{Quality: 80})
	default:
		err = png.Encode(&buf, dst)
	}
	if err != nil {
		return "", fmt.Errorf("encode thumbnail: %w", err)
	}

	return base64.StdEncoding.EncodeToString(buf.Bytes()), nil
}
