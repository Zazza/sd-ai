package preset

type Preset struct {
	ID             int64   `json:"id"`
	Name           string  `json:"name"`
	PresetType     string  `json:"preset_type"`
	Prompt         string  `json:"prompt"`
	NegativePrompt string  `json:"negative_prompt"`
	Sampler        string  `json:"sampler"`
	ScheduleType   string  `json:"schedule_type"`
	Steps          int     `json:"steps"`
	CfgScale       float64 `json:"cfg_scale"`
	ModelName              string   `json:"model_name"`
	Seed                   *int64   `json:"seed"`
	DenoisingStrength      *float64 `json:"denoising_strength"`
	ClipSkip               *int     `json:"clip_skip"`
	BatchSize              *int     `json:"batch_size"`
	BatchCount             *int     `json:"batch_count"`
	VAE                    string   `json:"vae"`
	TypeID                 *int64   `json:"type_id"`
	Tags                   string   `json:"tags"`
	Loras                  string   `json:"loras"`
	IsBundled              bool     `json:"is_bundled"`
	CreatedAt              string   `json:"created_at"`
	UpdatedAt              string   `json:"updated_at"`
}

type PresetType struct {
	ID          int64  `json:"id"`
	Name        string `json:"name"`
	Description string `json:"description"`
	SortOrder   int    `json:"sort_order"`
	CreatedAt   string `json:"created_at"`
}

type PresetInstallStatus struct {
	ID          int64    `json:"id"`
	Name        string   `json:"name"`
	Installed   bool     `json:"installed"`
	MissingSD   []string `json:"missing_sd"`
	MissingLoRA []string `json:"missing_lora"`
}

type LoRAEntry struct {
	Name   string  `json:"name"`
	Weight float64 `json:"weight"`
}

type SavedDescription struct {
	ID             int64  `json:"id"`
	Text           string `json:"text"`
	Name           string `json:"name"`
	NegativePrompt string `json:"negative_prompt"`
	Type           string `json:"type"`
	CreatedAt      string `json:"created_at"`
}

type SavedPrompt struct {
	ID        int64  `json:"id"`
	Text      string `json:"text"`
	CreatedAt string `json:"created_at"`
}

type CompoundPreset struct {
	ID          int64                `json:"id"`
	Name        string               `json:"name"`
	Description string               `json:"description"`
	Steps       []CompoundPresetStep `json:"steps"`
	CreatedAt   string               `json:"created_at"`
	UpdatedAt   string               `json:"updated_at"`
}

type CompoundPresetStep struct {
	ID                int64   `json:"id"`
	CompoundPresetID  int64   `json:"compound_preset_id"`
	StepOrder         int     `json:"step_order"`
	PresetID          int64   `json:"preset_id"`
	DenoisingStrength float64 `json:"denoising_strength"`
	ResolutionID      *int64  `json:"resolution_id"`
	Preset            *Preset `json:"preset,omitempty"`
}

type SavedScene struct {
	ID        int64  `json:"id"`
	Name      string `json:"name"`
	SceneJSON string `json:"scene_json"`
	CreatedAt string `json:"created_at"`
}

type SessionInfo struct {
	ID        int64 `json:"id"`
	Name      string `json:"name"`
	ItemCount int    `json:"item_count"`
	CreatedAt string `json:"created_at"`
	UpdatedAt string `json:"updated_at"`
}

type SessionItem struct {
	ID             int64   `json:"id"`
	SessionID      int64   `json:"session_id"`
	FileName       string  `json:"file_name"`
	ThumbName      string  `json:"thumb_name"`
	Source         string  `json:"source"`
	Prompt         string  `json:"prompt"`
	NegativePrompt string  `json:"negative_prompt"`
	Sampler        string  `json:"sampler"`
	Steps          int     `json:"steps"`
	CfgScale       float64 `json:"cfg_scale"`
	Seed           *int64  `json:"seed"`
	Denoising      float64 `json:"denoising"`
	Width          int     `json:"width"`
	Height         int     `json:"height"`
	IsPreview      bool    `json:"is_preview"`
	PresetID       *int64  `json:"preset_id"`
	IsActive       bool    `json:"is_active"`
	CreatedAt      string  `json:"created_at"`
}

type ExportPreset struct {
	ID            int64  `json:"id"`
	Name          string `json:"name"`
	Format        string `json:"format"`
	Width         int    `json:"width"`
	Height        int    `json:"height"`
	LockRatio     bool   `json:"lock_ratio"`
	Quality       int    `json:"quality"`
	Interpolation string `json:"interpolation"`
	CreatedAt     string `json:"created_at"`
	UpdatedAt     string `json:"updated_at"`
}

type Resolution struct {
	ID        int64  `json:"id"`
	Name      string `json:"name"`
	Width     int    `json:"width"`
	Height    int    `json:"height"`
	IsBuiltin bool   `json:"is_builtin"`
	CreatedAt string `json:"created_at"`
}

type HiresProfile struct {
	ID                int64   `json:"id"`
	Name              string  `json:"name"`
	Upscale           float64 `json:"upscale"`
	DenoisingStrength float64 `json:"denoising_strength"`
	Upscaler          string  `json:"upscaler"`
	IsBuiltin         bool    `json:"is_builtin"`
	CreatedAt         string  `json:"created_at"`
}
