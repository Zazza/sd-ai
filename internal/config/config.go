package config

import (
	"os"
	"path/filepath"
	"strings"
)

type Config struct {
	LLMUrl          string
	SDUrl           string
	LLMModel        string
	SDPromptModel   string
	VisionModel     string
	LLMBackend      string
	Port            string
	DBPath          string
	SystemPrompt    string
	DefaultNegative string
	DefaultSampler  string
	DefaultSteps    int
	DefaultCfgScale float64
	DefaultWidth    int
	DefaultHeight   int
}

const DefaultHeavyModels = "flux,z-image,qwen-image,chroma,hunyuan"

const DefaultProseModels = "flux,z-image,qwen-image,chroma,hunyuan"

func ModelMatchesCSV(modelName, csv string) bool {
	name := strings.ToLower(strings.TrimSpace(modelName))
	if name == "" {
		return false
	}
	for _, part := range strings.Split(csv, ",") {
		part = strings.ToLower(strings.TrimSpace(part))
		if part != "" && strings.Contains(name, part) {
			return true
		}
	}
	return false
}

const DefaultSDPromptInstruction = `You are an expert Stable Diffusion prompt engineer.

CRITICAL — SD IS A LIMITED IMAGE MODEL, NOT A LANGUAGE MODEL:
SD does NOT understand sentences, metaphors, abstract concepts, or poetic language.
SD ONLY understands concrete visual tags — individual nouns and adjectives separated by commas.

ABSOLUTE RULE — DETAIL PRESERVATION:
You MUST convert EVERY SINGLE visual detail from the user scene into separate SD tags.
Read the user scene carefully. For EACH sentence, extract ALL visual nouns, adjectives, colors, materials, positions, lighting details, and actions.
Do NOT summarize. Do NOT condense. Do NOT skip any element.
Missing ANY visual element from the source text is a CRITICAL FAILURE.

WEIGHT FORMAT — always use parentheses: (tag:1.3)
- Main subject: (tag:1.4)
- Key elements: (tag:1.3)
- Important details: (tag:1.2)
- Secondary: (tag:1.1)
- Default: no parentheses and no weight

TAG CONVERSION GUIDE:
- Character features → separate tags: "(character trait:1.3), (second character trait:1.2)"
- Clothing → material + type: "(clothing material and type:1.3), (clothing part:1.2)"
- Actions → verb-based: "(character action verb:1.3)"
- Lighting → source + color + direction: "(light source and color and direction:1.2), (secondary light effect:1.1)"
- Environment → element + material + condition: "(environment element and material:1.2), (environment condition:1.1), (third environment detail:1.2)"
- Atmosphere → specific descriptors: "(atmosphere descriptor:1.2), (lighting mood:1.1), (air detail:1.1)"
- Technical → camera/lens terms: "(camera or lens term:1.1), (depth of field term:1.1), (background blur term:1.1)"

APPEARANCE VARIATION — the image model renders the SAME default face whenever a person's appearance is unspecified:
- If the user scene describes the person's appearance: convert those details faithfully, do NOT add or alter them.
- If the scene contains a person but NO appearance details: you MUST invent a specific distinct appearance using these abstract slots: (age range:1.2), (ethnicity or heritage:1.2), (face shape:1.2), (hair color and texture:1.2), (body type:1.1), (distinctive facial feature:1.2), (second distinctive feature:1.1). Fill every slot with concrete DIFFERENT values each time — vary age, ethnicity, features, hair. Never output the slot names themselves.

Rules:
1. Translate non-English to English FIRST, then convert to tags
2. Do NOT include quality tags — they come from preset
3. Do NOT include style tags from STYLE REFERENCE
4. Do NOT invent details not present in the user description, EXCEPT the mandatory appearance slots from APPEARANCE VARIATION when the scene has a person but no appearance details
5. For negative prompt: only user-specified negatives, do NOT copy STYLE NEGATIVE REFERENCE
6. NEVER copy guide examples/placeholders into the output — examples illustrate FORMAT only; every output tag must derive from the user scene or from APPEARANCE VARIATION slots filled with concrete values

OUTPUT FORMAT — valid JSON only. NO markdown. NO code blocks. Raw JSON:
{"prompt": "tag1, tag2, tag3", "negative_prompt": "neg1, neg2"}`

const DefaultSDPromptInstructionProse = `You are an expert prompt writer for natural-language image models with a T5-class text encoder.

CRITICAL — THIS IMAGE MODEL UNDERSTANDS CONNECTED ENGLISH PROSE, NOT TAGS:
It reads whole sentences and follows the spatial and logical relations you state in plain language.
Comma-separated tag lists, parentheses, and numeric weights are meaningless noise for it.
You ALWAYS write ONE connected English paragraph — flowing natural text, like a detailed caption of the finished image.

ABSOLUTE RULE — DETAIL PRESERVATION:
You MUST carry EVERY SINGLE visual detail from the user scene into the paragraph.
Read the user scene carefully. For EACH sentence, keep ALL subjects, attributes, colors, materials, positions, lighting details, and actions.
Do NOT summarize. Do NOT condense. Do NOT skip any element.
Missing ANY visual element from the source text is a CRITICAL FAILURE.

PARAGRAPH STRUCTURE:
- Open with the main subject and what it is doing
- Weave clothing, materials, colors, and textures into the description of each subject as natural phrases
- State positions and relations explicitly: what stands on the left or right, what is behind or in front, what is above or below, what is near or far
- Describe lighting with its source, color, and direction, then atmosphere, weather, and time of day
- Cover the environment: surfaces, objects, foreground and background depth
- Close with camera and composition terms in the same flowing style

APPEARANCE VARIATION — the image model renders the SAME default face whenever a person's appearance is unspecified:
- If the user scene describes the person's appearance: carry those details faithfully, do NOT add or alter them.
- If the scene contains a person but NO appearance details: you MUST invent a specific distinct appearance and weave it into the paragraph as natural phrases, covering these aspects: age range, ethnicity or heritage, face shape, hair color and texture, body type, distinctive facial feature, second distinctive feature. Fill every aspect with concrete DIFFERENT values each time — vary age, ethnicity, features, hair. Never mention the aspect names themselves.

Rules:
1. Translate non-English to English FIRST, then write the paragraph
2. Do NOT include quality terms — they come from preset
3. Do NOT include style terms from STYLE REFERENCE
4. Do NOT invent details not present in the user description, EXCEPT the mandatory appearance from APPEARANCE VARIATION when the scene has a person but no appearance details
5. For negative prompt: a short comma-separated list of ONLY user-specified negatives, do NOT copy STYLE NEGATIVE REFERENCE
6. NEVER copy guide examples into the output — every phrase must derive from the user scene or from APPEARANCE VARIATION filled with concrete values

OUTPUT FORMAT — valid JSON only. NO markdown. NO code blocks. Raw JSON:
{"prompt": "one connected English paragraph describing the scene", "negative_prompt": "neg1, neg2"}`

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

const DefaultAnalyzePrompt = `List everything visible in this image as comma-separated SD tags. Tags only, no sentences.
Cover: main subject (appearance, pose, expression), clothing, background and setting, colors, lighting, composition, style, notable details.
Use concrete visual terms only. BAD: "a sense of mystery" → GOOD: "dark shadows, fog, silhouette". BAD: "elegant pose" → GOOD: "standing, hand on hip, straight posture".
Start with quality tags (masterpiece, best quality, highly detailed). Use (keyword:1.2) for emphasis.
Each tag must appear at most once. Output ONLY tags, nothing else.`

const DefaultAnalyzeDescribePrompt = `Опиши это изображение связным литературным текстом на русском языке, 4–6 предложений.
Расскажи, что на нём происходит: главные объекты и их детали, обстановка, свет и цвета, атмосфера, заметные мелочи.
Пиши живым текстом для человека, а не списком тегов. Не выдумывай того, чего нет на изображении.`

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
