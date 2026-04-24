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
	LLMBackend         string
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

const KidsModePrompt = `

SAFETY RULES (mandatory):
- Do NOT generate any tags related to violence, gore, blood, weapons harm, death, horror, torture, abuse.
- Do NOT generate any tags related to nudity, sexual content, erotic, suggestive poses.
- Do NOT generate any tags related to drugs, alcohol, smoking, self-harm, suicide.
- Only produce safe, family-friendly, child-appropriate content tags.
- If the request cannot be made safe, respond with: "safe landscape, beautiful nature, sunny day, clear sky, peaceful meadow, colorful flowers, butterflies, gentle breeze, warm sunlight, family-friendly, wholesome, cute animals playing, rainbow"`

const KidsModeNegativePrompt = "nsfw, nude, naked, porn, erotic, sexual, violence, gore, blood, horror, torture, death, kill, murder, weapon harm, drugs, alcohol, smoking, self-harm, suicide, abuse, assault, mature content, explicit, suggestive, provocative, disturbing, frightening, scary, creepy, disgusting, obscene"

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
		LLMBackend:    env("LLM_BACKEND", "lmstudio"),
		LLMModel:      env("LLM_MODEL", "openai/gpt-oss-20b"),
		SDPromptModel: env("SD_PROMPT_MODEL", "default"),
		Port:          env("PORT", "8080"),
		DBPath:        dbPath,
		SystemPrompt: `You are an expert Stable Diffusion prompt engineer. Your task is to convert a natural language description into a high-quality Stable Diffusion text-to-image prompt in English.

Rules:
- CRITICAL: Output ONLY comma-separated tags and phrases. NO thinking, NO reasoning, NO explanations, NO headers, NO lists, NO markdown, NO analysis. Start your response immediately with the first tag.
- Use comma-separated tags and descriptive phrases.
- Add quality boosters: masterpiece, best quality, highly detailed, sharp focus.
- Add lighting and atmosphere: cinematic lighting, volumetric lighting, etc.
- Add art style if appropriate: digital painting, concept art, fantasy art, etc.
- If the description mentions a specific type (weapon, armor, character, etc.), optimize the prompt for that type.
- Keep the prompt under 150 tokens. Be concise — prioritize the most important visual details over listing every color.
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
