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
	Width          int     `json:"width"`
	Height         int     `json:"height"`
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
	TypeID                 *int64   `json:"type_id"`
	Tags                   string   `json:"tags"`
	Loras                  string   `json:"loras"`
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

type LoRAEntry struct {
	Name   string  `json:"name"`
	Weight float64 `json:"weight"`
}

type SavedDescription struct {
	ID        int64  `json:"id"`
	Text      string `json:"text"`
	CreatedAt string `json:"created_at"`
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
	Width             int     `json:"width"`
	Height            int     `json:"height"`
	DenoisingStrength float64 `json:"denoising_strength"`
	Preset            *Preset `json:"preset,omitempty"`
}

type SavedScene struct {
	ID        int64  `json:"id"`
	Name      string `json:"name"`
	SceneJSON string `json:"scene_json"`
	CreatedAt string `json:"created_at"`
}
