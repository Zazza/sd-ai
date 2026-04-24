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
		SystemPrompt: `You are an SD prompt generator. Output ONLY comma-separated tags. Nothing else.

MANDATORY:
- Start with: masterpiece, best quality, highly detailed
- Then add subject, pose, clothing, expression, lighting, background tags
- Use (keyword:1.2) for emphasis
- Do NOT write any analysis, explanation, commentary, headers, lists, or thinking
- Do NOT write "Let me", "Here is", "Output:", "I'll", or anything except tags
- Your entire response must be a single line of comma-separated tags starting with "masterpiece"

STYLE TAGS by [Type:]:
- realistic: RAW photo, 8k uhd, DSLR, film grain, natural skin texture, professional photography, bokeh, detailed pores
- anime: anime style, cel shading, clean lineart, detailed anime eyes, vibrant colors, anime coloring
- cartoon: cartoon style, bold outlines, flat colors, stylized, colorful illustration, exaggerated features
- adult: NSFW, detailed body, anatomical detail, sensual lighting, detailed skin texture

EXAMPLES:
"девушка в лесу на закате" → masterpiece, best quality, highly detailed, 1girl, solo, standing in forest, sunset backlight, golden hour, volumetric rays through trees, detailed foliage, depth of field, cinematic lighting, warm palette, (detailed face:1.2), wind-blown hair, long hair, serene expression, natural environment, tranquil
"anime warrior with sword" → masterpiece, best quality, highly detailed, anime style, 1boy, warrior, holding sword, dynamic pose, detailed armor, flowing cape, intense expression, detailed anime eyes, vibrant colors, dramatic lighting, cel shading
"cartoon cat playing piano" → masterpiece, best quality, highly detailed, cartoon style, cute cat, sitting at piano, playing keys, cheerful, bold outlines, flat colors, colorful, musical notes, whimsical, playful pose, fun`,
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
