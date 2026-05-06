package importexport

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"image"
	"image/jpeg"
	"image/png"
	"math"
	"os"
	"path/filepath"
	"strings"
	"time"

	xdraw "golang.org/x/image/draw"

	"go-sd/internal/logger"
	"go-sd/internal/preset"
	"go-sd/internal/promptutil"
	"go-sd/internal/sd"
)

type ExportFile struct {
	Version    int          `json:"version"`
	ExportedAt time.Time    `json:"exported_at"`
	Presets    []PresetData `json:"presets"`
}

type PresetData struct {
	Name                   string   `json:"name"`
	PresetType             string   `json:"preset_type"`
	TypeName               string   `json:"type_name"`
	Prompt                 string   `json:"prompt"`
	NegativePrompt         string   `json:"negative_prompt"`
	Sampler                string   `json:"sampler"`
	ScheduleType           string   `json:"schedule_type"`
	Steps                  int      `json:"steps"`
	CfgScale               float64  `json:"cfg_scale"`
	Width                  int      `json:"width"`
	Height                 int      `json:"height"`
	ModelName              string   `json:"model_name"`
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

type ValidationWarning struct {
	PresetName string   `json:"preset_name"`
	Warnings   []string `json:"warnings"`
}

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

type ProcessedImage struct {
	Data     []byte
	Filename string
	Format   string
}

type Service struct {
	db  *preset.DB
	sd  sd.Service
	log *logger.Logger
}

func New(db *preset.DB, sdClient sd.Service, log *logger.Logger) *Service {
	return &Service{
		db:  db,
		sd:  sdClient,
		log: log,
	}
}

func (s *Service) PrepareExportData(ids []int64) ([]PresetData, error) {
	if len(ids) == 0 {
		return nil, fmt.Errorf("no presets selected")
	}

	selected, err := s.db.GetByIDs(ids)
	if err != nil {
		return nil, err
	}

	typeMap := make(map[int64]string)
	types, _ := s.db.ListPresetTypes()
	for _, t := range types {
		typeMap[t.ID] = t.Name
	}

	result := make([]PresetData, len(selected))
	for i, p := range selected {
		typeName := p.PresetType
		if p.TypeID != nil {
			if n, ok := typeMap[*p.TypeID]; ok {
				typeName = n
			}
		}
		result[i] = PresetData{
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

	return result, nil
}

func (s *Service) BuildExportFile(presets []PresetData) ([]byte, error) {
	data := ExportFile{
		Version:    2,
		ExportedAt: time.Now().UTC(),
		Presets:    presets,
	}
	return json.MarshalIndent(data, "", "  ")
}

func (s *Service) ParseImportFile(filePath string) ([]PresetData, error) {
	info, err := os.Stat(filePath)
	if err != nil {
		return nil, fmt.Errorf("stat file: %w", err)
	}
	if info.Size() > 10*1024*1024 {
		return nil, fmt.Errorf("file too large (max 10 MB)")
	}

	jsonBytes, err := os.ReadFile(filePath)
	if err != nil {
		return nil, fmt.Errorf("read file: %w", err)
	}

	var data ExportFile
	if err := json.Unmarshal(jsonBytes, &data); err != nil {
		return nil, fmt.Errorf("parse json: %w", err)
	}
	if data.Version < 1 || data.Version > 2 {
		return nil, fmt.Errorf("unsupported version: %d", data.Version)
	}

	fileName := filepath.Base(filePath)
	for i := range data.Presets {
		data.Presets[i].SourceFile = fileName
	}

	return data.Presets, nil
}

func (s *Service) ValidateModels(items []PresetData) ([]ValidationWarning, error) {
	if len(items) == 0 {
		return nil, nil
	}

	var warnings []ValidationWarning

	sdModels, _ := s.sd.GetModels()
	modelSet := make(map[string]bool)
	for _, m := range sdModels {
		modelSet[m.Name] = true
	}

	loras, _ := s.sd.GetLoRAs()
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

func (s *Service) ImportItems(items []PresetData) ([]preset.Preset, error) {
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
		existing, err := s.db.ListPresetTypes()
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
			if err := s.db.CreatePresetType(pt); err == nil {
				typeCache[typeName] = &pt.ID
			}
		}
	}

	batch := make([]preset.Preset, len(items))
	for i, item := range items {
		sampler, scheduleType := promptutil.SplitCompositeSampler(item.Sampler, item.ScheduleType)
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

	return s.db.CreateBatch(batch)
}

func (s *Service) ProcessExportImage(params ExportImageParams) (*ProcessedImage, error) {
	if params.ImageBase64 == "" {
		return nil, fmt.Errorf("no image provided")
	}

	switch params.Format {
	case "png", "jpeg":
	default:
		return nil, fmt.Errorf("unsupported format: %s", params.Format)
	}
	switch params.Interpolation {
	case "nearest", "linear", "lanczos", "":
	default:
		return nil, fmt.Errorf("unsupported interpolation: %s", params.Interpolation)
	}

	const maxBase64Len = 22 * 1024 * 1024
	if len(params.ImageBase64) > maxBase64Len {
		return nil, fmt.Errorf("image too large (max 16 MB)")
	}

	imgData, err := base64.StdEncoding.DecodeString(params.ImageBase64)
	if err != nil {
		return nil, fmt.Errorf("decode base64: %w", err)
	}

	img, _, err := image.Decode(bytes.NewReader(imgData))
	if err != nil {
		return nil, fmt.Errorf("decode image: %w", err)
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

	var buf bytes.Buffer
	switch params.Format {
	case "jpeg":
		err = jpeg.Encode(&buf, result, &jpeg.Options{Quality: params.Quality})
	default:
		err = png.Encode(&buf, result)
	}
	if err != nil {
		return nil, fmt.Errorf("encode %s: %w", params.Format, err)
	}

	return &ProcessedImage{
		Data:     buf.Bytes(),
		Filename: params.Filename,
		Format:   params.Format,
	}, nil
}

func (s *Service) ListExportPresets() ([]preset.ExportPreset, error) {
	return s.db.ListExportPresets()
}

func (s *Service) SaveExportPreset(ep preset.ExportPreset) (*preset.ExportPreset, error) {
	if ep.ID > 0 {
		if err := s.db.UpdateExportPreset(&ep); err != nil {
			return nil, err
		}
	} else {
		if err := s.db.CreateExportPreset(&ep); err != nil {
			return nil, err
		}
	}
	return &ep, nil
}

func (s *Service) DeleteExportPreset(id int64) error {
	return s.db.DeleteExportPreset(id)
}

func WriteImageToPath(img *ProcessedImage, path string) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	return os.WriteFile(path, img.Data, 0o644)
}
