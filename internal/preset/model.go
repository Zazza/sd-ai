package preset

import "time"

type Preset struct {
	ID             int64     `json:"id"`
	Name           string    `json:"name"`
	PresetType     string    `json:"preset_type"`
	Prompt         string    `json:"prompt"`
	NegativePrompt string    `json:"negative_prompt"`
	Sampler        string    `json:"sampler"`
	Steps          int       `json:"steps"`
	CfgScale       float64   `json:"cfg_scale"`
	Width          int       `json:"width"`
	Height         int       `json:"height"`
	ModelName      string    `json:"model_name"`
	Seed           *int64    `json:"seed"`
	CreatedAt      time.Time `json:"created_at"`
	UpdatedAt      time.Time `json:"updated_at"`
}

type SavedDescription struct {
	ID        int64     `json:"id"`
	Text      string    `json:"text"`
	CreatedAt time.Time `json:"created_at"`
}

type SavedPrompt struct {
	ID        int64     `json:"id"`
	Text      string    `json:"text"`
	CreatedAt time.Time `json:"created_at"`
}
