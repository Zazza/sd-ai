package compositor

type Position struct {
	X float64 `json:"x"`
	Y float64 `json:"y"`
}

type CharacterSlot struct {
	Name     string   `json:"name"`
	Prompt   string   `json:"prompt"`
	Position Position `json:"position"`
	Scale    float64  `json:"scale"`
}

type Scene struct {
	BackgroundPrompt string          `json:"background_prompt"`
	NegativePrompt   string          `json:"negative_prompt"`
	Characters       []CharacterSlot `json:"characters"`
	Width            int             `json:"width"`
	Height           int             `json:"height"`
	PresetID         int64           `json:"preset_id"`
}

type MultiPassProgress struct {
	Step      string `json:"step"`
	Character int    `json:"character,omitempty"`
	Total     int    `json:"total,omitempty"`
}

type MultiPassResult struct {
	Image      string `json:"image"`
	Background string `json:"background,omitempty"`
	Characters []struct {
		Name  string `json:"name"`
		Image string `json:"image,omitempty"`
	} `json:"characters,omitempty"`
}
