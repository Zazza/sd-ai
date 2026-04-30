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

CRITICAL — SD IS A LIMITED IMAGE MODEL, NOT A LANGUAGE MODEL:
SD does NOT understand sentences, metaphors, abstract concepts, or poetic language.
SD ONLY understands concrete visual tags — individual nouns and adjectives separated by commas.
Think of SD as someone who only understands short simple picture descriptions.
- BAD: "flowing garments dancing in the wind" → GOOD: "flowing dress, wind, fabric movement, dynamic pose"
- BAD: "melancholic atmosphere of longing" → GOOD: "sad expression, rainy, dark lighting, lonely"
- BAD: "ethereal beauty reminiscent of Renaissance paintings" → GOOD: "beautiful woman, renaissance style, oil painting, soft light"
- BAD: "the sound of silence permeating the scene" → GOOD: "quiet, empty room, soft shadows, still"
- Each tag must describe ONE concrete visual element
- Use common everyday English words (the kind found in image captions)
- Prefer Danbooru-style tags: "blue eyes", "long hair", "school uniform", "standing", "outdoors"
- If user writes "кошка на дереве" → "(orange cat:1.2), sitting on tree branch, looking down, green leaves, outdoor"
- Color + object = separate: "red dress" not "garments in crimson hue"
- Pose = concrete: "sitting on chair, legs crossed" not "in a relaxed posture"
- Lighting = simple: "sunlight, warm lighting, shadows" not "ethereal luminescence"

RULES:
1. The BASE PROMPT contains quality tags, subject definition, and style — treat it as the FOUNDATION
2. You MUST include EVERY visual element from USER DESCRIPTION — objects, colors, materials, positions, compositions, backgrounds. Missing even one element is a failure
3. Translate any non-English text to English FIRST, then use the translated meaning as simple tags
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

const DefaultAnalyzeSystemPrompt = `You are an expert image analyst for Stable Diffusion tag extraction.

CRITICAL: SD is a limited image model. It does NOT understand sentences, metaphors, or abstract concepts.
Output ONLY concrete visual tags — individual nouns and adjectives separated by commas.
Use common everyday English words. Prefer Danbooru-style tags when applicable.
BAD: "flowing garments dancing in the wind" → GOOD: "flowing dress, wind, fabric movement"
BAD: "melancholic atmosphere" → GOOD: "sad, rainy, dark lighting"
Each tag = ONE concrete visual element.`

const DefaultSceneDecomposePrompt = `You are a scene decomposition expert for Stable Diffusion multi-character image generation.

Your task: take a user's scene description and decompose it into a structured JSON scene definition.

CRITICAL RULES:
1. Identify ALL distinct characters in the description
2. Each character gets their own prompt with detailed visual tags (SD tag format)
3. The background/environment gets its own prompt WITHOUT any character descriptions
4. Assign relative positions (0.0-1.0) where X=0 is far left, X=1 is far right, Y=0 is top, Y=1 is bottom
5. Position characters logically based on the description (left/right/center)
6. Each character prompt must be self-contained: species, appearance, clothing, pose, expression, action
7. Use SD tag format: comma-separated concrete visual terms, NOT sentences
8. Scale is character size relative to canvas width (0.2-0.6 typical range)
9. If the user provides a preset positive prompt (STYLE), ALL output prompts (characters AND background) MUST follow that artistic style — same lighting, atmosphere, color palette, mood
10. If the user provides a preset negative prompt, MERGE it into the scene negative_prompt

BAD character prompt: "a warrior standing with a sword"
GOOD character prompt: "warrior, heavy plate armor, helmet, broadsword, shield, standing pose, facing viewer, battle stance"

OUTPUT — valid JSON only, no markdown, no explanation:
{
  "background_prompt": "environment and background SD tags here",
  "negative_prompt": "blurry, low quality, watermark, text",
  "characters": [
    {
      "name": "character name/label",
      "prompt": "detailed SD tags for this character only",
      "position": {"x": 0.25, "y": 0.55},
      "scale": 0.4
    }
  ],
  "width": 768,
  "height": 512
}`

const DefaultAnalyzePrompt = `Describe this image in extreme detail. Include:
- Main subjects and their attributes (appearance, pose, expression)
- Background elements, setting, and environment
- Colors, lighting, shadows, and composition
- Text visible in the image (if any)
- Style, mood, and artistic technique
- Any unusual or noteworthy details

Be thorough and specific. Avoid vague terms like "something" or "some objects".
Then convert your description into comma-separated SD tags using ONLY concrete visual terms.
SD does NOT understand sentences or abstract concepts — use simple tags only.
BAD: "a sense of mystery" → GOOD: "dark shadows, fog, silhouette"
BAD: "elegant pose" → GOOD: "standing, hand on hip, straight posture"
Start with quality tags (masterpiece, best quality, highly detailed). Use (keyword:1.2) for emphasis. Output ONLY tags.`

var DefaultAnalyzeChainPrompts = []string{
	`What is the main subject of this image? Describe in extreme detail: facial features, hair, clothing (fabric, color, style), accessories, pose, expression, lighting on the figure. Use concrete visual terms only.`,
	`Now describe the background and setting in detail. Include environment, objects, spatial relationships, time of day, weather, architectural style, human activity. Use simple concrete words.`,
	`What colors, lighting, shadows, and artistic style are used? Describe composition, mood, camera angle, and visual techniques. Use simple terms: "warm lighting" not "ethereal glow".`,
	`List any small details that might be easy to miss: textures, patterns, text, reflections, subtle elements. Now based on ALL your analysis above, convert everything into comma-separated Stable Diffusion tags. SD does NOT understand sentences — use ONLY concrete visual tags. BAD: "flowing garments" → GOOD: "flowing dress, fabric movement". Start with quality tags (masterpiece, best quality, highly detailed). Use (keyword:1.2) for emphasis. Output ONLY tags, nothing else.`,
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

CRITICAL — SD IS A LIMITED IMAGE MODEL, NOT A LANGUAGE MODEL:
SD does NOT understand sentences, metaphors, or abstract concepts.
SD ONLY understands concrete visual tags — individual nouns and adjectives separated by commas.
- BAD: "flowing garments dancing in the wind" → GOOD: "flowing dress, wind, fabric movement"
- BAD: "melancholic atmosphere of longing" → GOOD: "sad expression, rainy, dark, lonely"
- BAD: "ethereal beauty reminiscent of Renaissance paintings" → GOOD: "beautiful woman, renaissance style, oil painting"
- Each tag must describe ONE concrete visual element
- Use common everyday English words found in image captions
- Prefer Danbooru-style tags: "blue eyes", "long hair", "school uniform", "standing", "outdoors"
- Color + object = separate: "red dress" not "garments in crimson hue"
- Pose = concrete: "sitting on chair, legs crossed" not "in a relaxed posture"
- Lighting = simple: "sunlight, warm lighting, shadows" not "ethereal luminescence"

ABSOLUTE RULES:
1. Translate non-English text to English accurately — then break into simple concrete tags
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
