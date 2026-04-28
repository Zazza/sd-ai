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
	VisionModel        string
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

const DefaultSDPromptInstruction = `You are an expert Stable Diffusion prompt engineer. Your task is to merge a BASE PROMPT (from a preset) with a user DESCRIPTION into a single optimized SD prompt.

RULES:
1. The BASE PROMPT contains quality tags, subject definition, and style — treat it as the FOUNDATION
2. You MUST include EVERY visual element from USER DESCRIPTION — objects, colors, materials, positions, compositions, backgrounds. Missing even one element is a failure
3. Translate any non-English text to English FIRST, then use the translated meaning as tags
4. MERGE intelligently: integrate user elements INTO the base prompt structure, not append at the end
5. Token order matters: quality tags first, then main subject, then details, then background/atmosphere
6. Use SD weighting syntax: (keyword:1.2) for emphasis, (keyword:0.8) for de-emphasis
7. Use BREAK to separate major concept sections if needed
8. Keep total prompt under 75 tokens per chunk (SD processes in 75-token blocks)
9. For negative prompt: merge base negative with user-specified negatives, remove duplicates
10. Do NOT invent details not present in the base prompt or user description

OUTPUT FORMAT — valid JSON only, no markdown:
{"prompt": "merged positive prompt here", "negative_prompt": "merged negative prompt here"}`

const KidsModePrompt = `

SAFETY RULES (mandatory):
- Do NOT generate any tags related to violence, gore, blood, weapons harm, death, horror, torture, abuse.
- Do NOT generate any tags related to nudity, sexual content, erotic, suggestive poses.
- Do NOT generate any tags related to drugs, alcohol, smoking, self-harm, suicide.
- Only produce safe, family-friendly, child-appropriate content tags.
- If the request cannot be made safe, respond with: "safe landscape, beautiful nature, sunny day, clear sky, peaceful meadow, colorful flowers, butterflies, gentle breeze, warm sunlight, family-friendly, wholesome, cute animals playing, rainbow"`

const KidsModeNegativePrompt = "nsfw, nude, naked, porn, erotic, sexual, violence, gore, blood, horror, torture, death, kill, murder, weapon harm, drugs, alcohol, smoking, self-harm, suicide, abuse, assault, mature content, explicit, suggestive, provocative, disturbing, frightening, scary, creepy, disgusting, obscene"

const DefaultAnalyzeSystemPrompt = `You are an expert image analyst for Stable Diffusion tag extraction. Be precise, thorough, and specific in your descriptions.`

const DefaultAnalyzePrompt = `Describe this image in extreme detail. Include:
- Main subjects and their attributes (appearance, pose, expression)
- Background elements, setting, and environment
- Colors, lighting, shadows, and composition
- Text visible in the image (if any)
- Style, mood, and artistic technique
- Any unusual or noteworthy details

Be thorough and specific. Avoid vague terms like "something" or "some objects".
Then convert your description into comma-separated SD tags. Start with quality tags (masterpiece, best quality, highly detailed). Use (keyword:1.2) for emphasis. Output ONLY tags.`

var DefaultAnalyzeChainPrompts = []string{
	`What is the main subject of this image? Describe in extreme detail: facial features, hair, clothing (fabric, color, style), accessories, pose, expression, lighting on the figure.`,
	`Now describe the background and setting in detail. Include environment, objects, spatial relationships, time of day, weather, architectural style, human activity.`,
	`What colors, lighting, shadows, and artistic style are used? Describe composition, mood, camera angle, and visual techniques.`,
	`List any small details that might be easy to miss: textures, patterns, text, reflections, subtle elements. Now based on ALL your analysis above, convert everything into comma-separated Stable Diffusion tags. Start with quality tags (masterpiece, best quality, highly detailed). Use (keyword:1.2) for emphasis. Output ONLY tags, nothing else.`,
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
		LLMBackend:    env("LLM_BACKEND", "lmstudio"),
		LLMModel:      env("LLM_MODEL", "openai/gpt-oss-20b"),
		SDPromptModel: env("SD_PROMPT_MODEL", "default"),
		Port:          env("PORT", "8080"),
		DBPath:        dbPath,
		SystemPrompt: `You convert user descriptions into SD (Stable Diffusion) comma-separated tags.
You receive a STYLE REFERENCE (read-only context) and user descriptions in labeled fields.

ABSOLUTE RULES:
1. Translate non-English text to English accurately — preserve the EXACT meaning
2. Output ONLY tags derived from the user's field content
3. Do NOT output field labels or category names as tags
4. Do NOT copy tags from the STYLE REFERENCE
5. Do NOT add quality tags or subject tags — they come from preset
6. Do NOT invent details not present in user input
7. If NEGATIVE field is empty, output "negative_prompt": ""
8. Skip fields with no user content

CHARACTERS field — IMPORTANT:
- These are ADDITIONAL characters that must appear ALONGSIDE the main subject from the preset
- Describe each character fully for SD: species, size, pose, action, expression
- Use high emphasis: (bear:1.3), (large angry bear:1.2)
- Example: user writes "медведь" → output "(angry bear:1.3), large brown bear, on all fours, growling, facing viewer"
- This ensures SD treats it as a separate visible entity, not a background detail

OUTPUT — valid JSON only, no markdown, no explanation:
{"prompt": "tag1, tag2", "negative_prompt": "neg1, neg2"}`,
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
