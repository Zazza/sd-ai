package serverclient

import (
	_ "embed"
	"encoding/json"
	"fmt"
)

//go:embed catalog.json
var catalogData []byte

type CatalogModel struct {
	Name        string  `json:"name"`
	Description string  `json:"description"`
	SizeGB      float64 `json:"size_gb"`
	Recommended bool    `json:"recommended,omitempty"`
}

type CatalogSDModel struct {
	Name        string  `json:"name"`
	Base        string  `json:"base"`
	Category    string  `json:"category"`
	Description string  `json:"description"`
	URL         string  `json:"url,omitempty"`
	SizeGB      float64 `json:"size_gb"`
	Recommended bool    `json:"recommended,omitempty"`
}

type CatalogLoRA struct {
	Name          string  `json:"name"`
	Base          string  `json:"base"`
	Category      string  `json:"category"`
	DefaultWeight float64 `json:"default_weight"`
	Description   string  `json:"description"`
	URL           string  `json:"url,omitempty"`
	SizeMB        float64 `json:"size_mb"`
}

type Catalog struct {
	LLMGenerate []CatalogModel   `json:"llm_generate"`
	LLMVision   []CatalogModel   `json:"llm_vision"`
	SDModels    []CatalogSDModel `json:"sd_models"`
	LoRA        []CatalogLoRA    `json:"lora"`
}

func LoadCatalog() (*Catalog, error) {
	var catalog Catalog
	if err := json.Unmarshal(catalogData, &catalog); err != nil {
		return nil, fmt.Errorf("parse catalog: %w", err)
	}
	return &catalog, nil
}
