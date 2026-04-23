package config

import (
	"os"
	"path/filepath"
)

type Config struct {
	LLMUrl             string
	SDUrl              string
	LLMModel           string
	SDPromptModel      string
	Port               string
	DBPath             string
	SystemPrompt       string
	DefaultNegative    string
	DefaultSampler     string
	DefaultSteps       int
	DefaultCfgScale    float64
	DefaultWidth       int
	DefaultHeight      int
}

func Load() *Config {
	exe, _ := os.Executable()
	exeDir := filepath.Dir(exe)

	dbPath := env("DB_PATH", "data/presets.db")
	if !filepath.IsAbs(dbPath) {
		dbPath = filepath.Join(exeDir, dbPath)
	}

	return &Config{
		LLMUrl:        env("LLM_URL", "http://localhost:1234"),
		SDUrl:         env("SD_URL", "http://localhost:7860"),
		LLMModel:      env("LLM_MODEL", "openai/gpt-oss-20b"),
		SDPromptModel: env("SD_PROMPT_MODEL", "default"),
		Port:          env("PORT", "8080"),
		DBPath:        dbPath,
		SystemPrompt: `You are an expert Stable Diffusion prompt engineer. Your task is to convert a natural language description (in Russian) into a high-quality Stable Diffusion text-to-image prompt in English.

Rules:
- Output ONLY the prompt text, nothing else. No explanations, no markdown.
- Use comma-separated tags and descriptive phrases.
- Add quality boosters: masterpiece, best quality, highly detailed, sharp focus.
- Add lighting and atmosphere: cinematic lighting, volumetric lighting, etc.
- Add art style if appropriate: digital painting, concept art, fantasy art, etc.
- If the description mentions a specific type (weapon, armor, character, etc.), optimize the prompt for that type.
- Keep the prompt concise but descriptive (under 200 tokens).
- Use weighted tags where important: (keyword:1.2) for emphasis.
- Always describe the subject clearly with visual attributes.`,
		DefaultNegative: "blurry, low quality, watermark, text, signature",
		DefaultSampler:  "Euler a",
		DefaultSteps:    20,
		DefaultCfgScale: 7.0,
		DefaultWidth:    512,
		DefaultHeight:   512,
	}
}

func env(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
