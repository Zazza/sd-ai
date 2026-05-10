package main

import (
	"bytes"
	"context"
	"encoding/base64"
	"fmt"
	"image"
	"image/jpeg"
	_ "image/png"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/wailsapp/wails/v2/pkg/runtime"

	"go-sd/internal/compositor"
	"go-sd/internal/config"
	"go-sd/internal/filebrowser"
	"go-sd/internal/generation"
	"go-sd/internal/importexport"
	"go-sd/internal/kids"
	"go-sd/internal/llm"
	"go-sd/internal/logger"
	"go-sd/internal/preset"
	"go-sd/internal/rembg"
	"go-sd/internal/sd"
	"go-sd/internal/session"
	"go-sd/internal/settings"
)

type appEmitter struct {
	ctx *context.Context
}

func (e *appEmitter) Emit(event string, data ...any) {
	if *e.ctx != nil {
		runtime.EventsEmit(*e.ctx, event, data...)
	}
}

type App struct {
	ctx         context.Context
	presets     *preset.DB
	llm         llm.Service
	sd          sd.Service
	rembgClient *rembg.Client
	log         *logger.Logger
	config      *config.Config
	dataDir     string

	kidsMgr     *kids.Manager
	sessions    *session.Service
	settingsSvc *settings.Service
	ieSvc       *importexport.Service
	gen         *generation.Service
	emitter     appEmitter
}

func NewApp(presets *preset.DB, llmClient llm.Service, sdClient sd.Service, cfg *config.Config) *App {
	a := &App{
		presets:     presets,
		llm:         llmClient,
		sd:          sdClient,
		rembgClient: rembg.New(""),
		log:         logger.New(nil),
		config:      cfg,
		dataDir:     filepath.Dir(cfg.DBPath),
		kidsMgr:     kids.NewManager(presets),
	}
	a.emitter = appEmitter{ctx: &a.ctx}
	a.sessions = session.New(presets, a.dataDir, &a.emitter)
	a.settingsSvc = settings.New(presets, llmClient, sdClient, cfg, a.rembgClient, a.log)
	a.ieSvc = importexport.New(presets, sdClient, a.log)
	a.gen = generation.New(
		presets, llmClient, sdClient, cfg,
		a.rembgClient, a.dataDir,
		&a.emitter, a.kidsMgr, a.sessions, a.settingsSvc, a.log,
	)
	return a
}

func (a *App) startup(ctx context.Context) {
	a.ctx = ctx
	a.log.SetContext(ctx)
	a.gen.SetContext(ctx)
}

func (a *App) Version() string {
	return version
}

// --- Kids Mode ---

type KidsCategoryInfo = kids.CategoryInfo

func (a *App) IsKidsModeActive() bool {
	return a.kidsMgr.IsActive()
}

func (a *App) SetKidsMode(enabled bool, pin string) error {
	return a.kidsMgr.SetKidsMode(enabled, pin)
}

func (a *App) GetKidsCategories() ([]KidsCategoryInfo, error) {
	return a.kidsMgr.GetCategories()
}

func (a *App) SetKidsCategory(name string, enabled bool, pin string) error {
	return a.kidsMgr.SetCategory(name, enabled, pin)
}

// --- Service Status ---

type ServiceInfo = settings.ServiceInfo
type ServiceStatus = settings.ServiceStatus

func (a *App) CheckServices() ServiceStatus {
	return a.settingsSvc.CheckServices()
}

func (a *App) CheckRembg() error {
	return a.settingsSvc.CheckRembg()
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

// --- Generation (delegates) ---

type SDProgressEvent = generation.SDProgressEvent
type GenerateSDPromptParams = generation.GenerateSDPromptParams
type GenerateSDPromptResult = generation.GenerateSDPromptResult
type GenerateImageParams = generation.GenerateImageParams
type GenerateImageResult = generation.GenerateImageResult
type RecommendPresetResult = generation.RecommendPresetResult
type AnalyzePrompts = generation.AnalyzePrompts
type UpscaleImageParams = generation.UpscaleImageParams
type BatchGenerateParams = generation.BatchGenerateParams
type BatchProgress = generation.BatchProgress
type TestGenerateParams = generation.TestGenerateParams
type TestGenerateResultItem = generation.TestGenerateResultItem
type UpscalePreviewParams = generation.UpscalePreviewParams
type GenerateCompoundImageParams = generation.GenerateCompoundImageParams
type GenerateFromImageParams = generation.GenerateFromImageParams
type BatchCompoundGenerateParams = generation.BatchCompoundGenerateParams
type TestCompoundGenerateParams = generation.TestCompoundGenerateParams
type DecomposeSceneParams = generation.DecomposeSceneParams

func (a *App) GenerateSDPrompt(params GenerateSDPromptParams) (*GenerateSDPromptResult, error) {
	return a.gen.GenerateSDPrompt(params)
}

func (a *App) GetDefaultPromptInstruction() string {
	return a.gen.GetDefaultPromptInstruction()
}

func (a *App) RecommendPreset(description string) (*RecommendPresetResult, error) {
	return a.gen.RecommendPreset(description)
}

func (a *App) AnalyzeImage(imageBase64 string) (string, error) {
	return a.gen.AnalyzeImage(imageBase64)
}

func (a *App) GetDefaultAnalyzePrompts() *AnalyzePrompts {
	return a.gen.GetDefaultAnalyzePrompts()
}

func (a *App) GenerateImage(params GenerateImageParams) (*GenerateImageResult, error) {
	return a.gen.GenerateImage(params)
}

func (a *App) BatchGenerate(params BatchGenerateParams) error {
	return a.gen.BatchGenerate(params)
}

func (a *App) TestGenerate(params TestGenerateParams) ([]TestGenerateResultItem, error) {
	return a.gen.TestGenerate(params)
}

func (a *App) GetPresetForBatch(presetID int64, description string) (*GenerateSDPromptResult, error) {
	return a.gen.GetPresetForBatch(presetID, description)
}

func (a *App) UpscaleImage(params UpscaleImageParams) (*GenerateImageResult, error) {
	return a.gen.UpscaleImage(params)
}

func (a *App) UpscalePreview(params UpscalePreviewParams) (*GenerateImageResult, error) {
	return a.gen.UpscalePreview(params)
}

func (a *App) GetLastImage() (*GenerateImageResult, error) {
	return a.gen.GetLastImage()
}

func (a *App) ClearLastImage() {
	a.gen.ClearLastImage()
}

func (a *App) InterruptGeneration() error {
	return a.gen.InterruptGeneration()
}

func (a *App) GenerateCompoundImage(params GenerateCompoundImageParams) (*GenerateImageResult, error) {
	return a.gen.GenerateCompoundImage(params)
}

func (a *App) GenerateFromImage(params GenerateFromImageParams) (*GenerateImageResult, error) {
	return a.gen.GenerateFromImage(params)
}

func (a *App) GenerateStereoImage(imageBase64, format string, eyeShift float64) (string, error) {
	return a.gen.GenerateStereoImage(imageBase64, format, eyeShift)
}

func (a *App) BatchCompoundGenerate(params BatchCompoundGenerateParams) error {
	return a.gen.BatchCompoundGenerate(params)
}

func (a *App) TestCompoundGenerate(params TestCompoundGenerateParams) ([]TestGenerateResultItem, error) {
	return a.gen.TestCompoundGenerate(params)
}

func (a *App) DecomposeScene(params DecomposeSceneParams) (*compositor.Scene, error) {
	return a.gen.DecomposeScene(params)
}

func (a *App) GenerateMultiPass(scene compositor.Scene) (*compositor.MultiPassResult, error) {
	return a.gen.GenerateMultiPass(scene)
}

// --- File/Clipboard (Wails runtime) ---

func (a *App) ReadImageFile() (string, error) {
	path, err := runtime.OpenFileDialog(a.ctx, runtime.OpenDialogOptions{
		Filters: []runtime.FileFilter{
			{DisplayName: "Images", Pattern: "*.png;*.jpg;*.jpeg"},
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

func (a *App) SelectFolder() (string, error) {
	path, err := runtime.OpenDirectoryDialog(a.ctx, runtime.OpenDialogOptions{
		Title: "Select Output Folder",
	})
	if err != nil {
		return "", err
	}
	return path, nil
}

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

func (a *App) FastSaveImage(base64Data, filename, format string) (string, error) {
	if base64Data == "" {
		return "", fmt.Errorf("no image data")
	}
	dir, err := a.presets.GetSetting("file_browser_path")
	if err != nil || dir == "" {
		return "", fmt.Errorf("no save directory set — open File Browser and select a folder first")
	}

	if filename == "" {
		filename = "sd-studio-image"
	}

	ext := ".jpg"
	if format == "png" {
		ext = ".png"
	}
	filename = sanitizeFilename(filename) + ext

	path := filepath.Join(dir, filename)
	if _, err := os.Stat(path); err == nil {
		base := filename[:len(filename)-len(ext)]
		for i := 1; ; i++ {
			path = filepath.Join(dir, fmt.Sprintf("%s_%d%s", base, i, ext))
			if _, err := os.Stat(path); err != nil {
				break
			}
		}
	}

	data, err := base64.StdEncoding.DecodeString(base64Data)
	if err != nil {
		return "", err
	}

	if format == "jpg" {
		img, _, err := image.Decode(bytes.NewReader(data))
		if err == nil {
			var buf bytes.Buffer
			if err := jpeg.Encode(&buf, img, &jpeg.Options{Quality: 95}); err != nil {
				return "", err
			}
			data = buf.Bytes()
		}
	}

	if err := os.WriteFile(path, data, 0o644); err != nil {
		return "", err
	}

	a.log.UserAction("Fast save: %s", path)
	return path, nil
}

func sanitizeFilename(name string) string {
	r := strings.NewReplacer(
		"/", "_", "\\", "_", ":", "_", "*", "_",
		"?", "_", "\"", "_", "<", "_", ">", "_", "|", "_",
	)
	result := r.Replace(name)
	if len(result) > 200 {
		result = result[:200]
	}
	return result
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

func (a *App) GetSDLoRAs() ([]sd.LoRA, error) {
	return a.sd.GetLoRAs()
}

// --- LLM Info ---

func (a *App) GetLLMModels() ([]llm.LLMModel, error) {
	return a.llm.GetModels()
}

// --- Settings ---

func (a *App) GetSettings() (map[string]string, error) {
	return a.settingsSvc.GetSettings()
}

func (a *App) UpdateSettings(data map[string]string) error {
	return a.settingsSvc.UpdateSettings(data)
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

// --- Preset Export/Import ---

type PresetExportFile = importexport.ExportFile
type PresetData = importexport.PresetData
type ImportPreview = importexport.ImportPreview
type ValidationWarning = importexport.ValidationWarning

func (a *App) ExportPresets(ids []int64) (string, error) {
	presets, err := a.ieSvc.PrepareExportData(ids)
	if err != nil {
		return "", err
	}
	jsonBytes, err := a.ieSvc.BuildExportFile(presets)
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
	return path, os.WriteFile(path, jsonBytes, 0o644)
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
	for _, p := range paths {
		parsed, err := a.ieSvc.ParseImportFile(p)
		if err != nil {
			continue
		}
		allPresets = append(allPresets, parsed...)
	}
	if len(allPresets) == 0 {
		return nil, fmt.Errorf("no presets found in selected files")
	}
	return &ImportPreview{Presets: allPresets, Total: len(allPresets)}, nil
}

func (a *App) ValidateImportModels(items []PresetData) ([]ValidationWarning, error) {
	return a.ieSvc.ValidateModels(items)
}

func (a *App) ImportPresets(items []PresetData) ([]preset.Preset, error) {
	return a.ieSvc.ImportItems(items)
}

// --- Compound Presets ---

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

// --- Export Image ---

type ExportImageParams = importexport.ExportImageParams

func (a *App) ExportImage(params ExportImageParams) (string, error) {
	processed, err := a.ieSvc.ProcessExportImage(params)
	if err != nil {
		return "", err
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

	return path, importexport.WriteImageToPath(processed, path)
}

func (a *App) ListExportPresets() ([]preset.ExportPreset, error) {
	return a.ieSvc.ListExportPresets()
}

func (a *App) SaveExportPreset(ep preset.ExportPreset) (*preset.ExportPreset, error) {
	return a.ieSvc.SaveExportPreset(ep)
}

func (a *App) DeleteExportPreset(id int64) error {
	return a.ieSvc.DeleteExportPreset(id)
}

// --- File Browser ---

type FileEntry = filebrowser.FileEntry

func (a *App) BrowseDirectory(dirPath string) ([]FileEntry, error) {
	return filebrowser.BrowseDirectory(dirPath)
}

func (a *App) ReadFileAsBase64(filePath string) (string, error) {
	return filebrowser.ReadFileAsBase64(filePath)
}

func (a *App) ReadThumbnail(filePath string) (string, error) {
	return filebrowser.ReadThumbnail(filePath)
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
	a.sessions.AddToSession(base64Data, nil, "file-browser", false, nil)
	return nil
}

// --- Scene Management ---

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

// --- Session Management ---

func (a *App) CreateSession(name string) (*preset.SessionInfo, error) {
	return a.sessions.CreateSession(name)
}

func (a *App) ListSessions() ([]preset.SessionInfo, error) {
	return a.sessions.ListSessions()
}

func (a *App) SwitchSession(id int64) error {
	return a.sessions.SwitchSession(id)
}

func (a *App) RenameSession(id int64, name string) error {
	return a.sessions.RenameSession(id, name)
}

func (a *App) DeleteSession(id int64) error {
	return a.sessions.DeleteSession(id)
}

func (a *App) GetSessionItems() ([]preset.SessionItem, error) {
	return a.sessions.GetSessionItems()
}

func (a *App) GetActiveSessionItem() (*preset.SessionItem, error) {
	return a.sessions.GetActiveSessionItem()
}

func (a *App) SetActiveSessionItem(id int64) error {
	return a.sessions.SetActiveSessionItem(id)
}

func (a *App) DeleteSessionItem(id int64) error {
	return a.sessions.DeleteSessionItem(id)
}

func (a *App) ClearSession() error {
	return a.sessions.ClearSession()
}

func (a *App) GetSessionImage(id int64) (string, error) {
	return a.sessions.GetSessionImage(id)
}

func (a *App) GetSessionThumb(id int64) (string, error) {
	return a.sessions.GetSessionThumb(id)
}

func (a *App) HasSessionItems() (bool, error) {
	return a.sessions.HasSessionItems()
}

func (a *App) ConfirmClose(action string) {
	a.sessions.ConfirmClose(action)
	if a.ctx != nil {
		runtime.Quit(a.ctx)
	}
}
