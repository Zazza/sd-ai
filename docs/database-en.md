[English](database-en.md) | [Русский](database-ru.md)

# SD Studio — Database

## SQLite

Path: `data/presets.db` (configurable via `DB_PATH`).

Migrations run automatically on `preset.Open()` (versions 1-10).

## Schema

### presets
Main table for generation presets.

| Field | Type | Description |
|-------|------|-------------|
| id | INTEGER PK | Auto-increment |
| name | TEXT | Preset name |
| type_id | INTEGER FK | Reference to preset_types.id |
| preset_type | TEXT | Type name (denormalized) |
| prompt | TEXT | Positive prompt |
| negative_prompt | TEXT | Negative prompt |
| sampler | TEXT | Sampler name |
| schedule_type | TEXT | "auto", "karras", "exponential" |
| steps | INTEGER | Number of steps |
| cfg_scale | REAL | CFG Scale |
| width | INTEGER | Width |
| height | INTEGER | Height |
| model_name | TEXT | SD model filename |
| vae | TEXT | VAE filename |
| seed | INTEGER | Seed (NULL = random) |
| clip_skip | INTEGER | CLIP Skip (NULL = default) |
| loras | TEXT | JSON: `[{"name":"lora1","weight":0.8}]` |
| hires_fix | BOOLEAN | Enable Hires fix |
| hires_upscaler | TEXT | Hires upscaler name |
| hires_scale | REAL | Hires upscale factor |
| hires_denoising | REAL | Hires denoising strength |
| hires_resize_x | INTEGER | Hires target width |
| hires_resize_y | INTEGER | Hires target height |
| hires_steps | INTEGER | Hires steps |
| batch_size | INTEGER | Batch size |
| batch_count | INTEGER | Batch count |
| sort_order | INTEGER | Sort order |
| created_at | TEXT | ISO 8601 |
| updated_at | TEXT | ISO 8601 |

### preset_types
Preset types (Anime, Realistic, etc.)

| Field | Type | Description |
|-------|------|-------------|
| id | INTEGER PK | |
| name | TEXT UNIQUE | Type name |
| description | TEXT | Description |
| sort_order | INTEGER | Sort order |
| created_at | TEXT | |

### compound_presets
Compound presets (multi-step pipeline).

| Field | Type | Description |
|-------|------|-------------|
| id | INTEGER PK | |
| name | TEXT | Name |
| description | TEXT | Description |
| steps | TEXT | JSON: `[{step_type, preset_id, ...}]` |
| created_at | TEXT | |
| updated_at | TEXT | |

**compound_preset_steps (JSON):**
```json
[
  {"step_type": "txt2img", "preset_id": 1, "order": 0},
  {"step_type": "upscale", "preset_id": 2, "order": 1}
]
```

### settings
Key-value settings store.

| Field | Type | Description |
|-------|------|-------------|
| key | TEXT PK | Setting key |
| value | TEXT | Value |

**Keys:**

| Key | Description | Default |
|-----|-------------|---------|
| `llm_url` | LLM API URL | `http://localhost:1234` |
| `sd_url` | SD WebUI URL | `http://localhost:7860` |
| `rembg_url` | rembg API URL | `http://localhost:5000` |
| `llm_generate_model` | Model for prompts | from env |
| `llm_analyze_model` | Model for vision | from env |
| `llm_backend` | ollama/lmstudio | from env |
| `llm_max_tokens` | Max tokens | `256` |
| `llm_keep_alive` | Ollama keep_alive | `5m` |
| `llm_num_ctx` | Ollama context | `4096` |
| `llm_num_gpu` | Ollama GPU layers | `1` |
| `sd_prompt_instruction` | System prompt for SD | DefaultSDPromptInstruction |
| `kids_mode` | Kids mode on/off | `false` |
| `kids_pin_hash` | SHA256 hash of PIN | — |
| `kids_pin_attempts` | PIN entry attempts | `0` |
| `kids_pin_lockout` | Lockout time | — |
| `kids_categories` | JSON of enabled categories | — |
| `fi_mode` | From Image: mode | `img2img` |
| `fi_preset_id` | From Image: preset | — |
| `fi_gen_mode` | From Image: gen mode | `preset` |
| `fi_denoising` | From Image: denoising | `0.5` |
| `fi_extra_negative` | From Image: negative | — |
| `fi_analyze_mode` | From Image: quick/deep | `quick` |
| `fi_mask_padding` | From Image: mask dilation (px) | `8` |
| `fi_mask_feather` | From Image: mask feathering (px) | `8` |
| `browser_folder` | File browser: folder | — |

### saved_descriptions
Saved user descriptions.

| Field | Type |
|-------|------|
| id | INTEGER PK |
| text | TEXT |
| name | TEXT |
| negative_prompt | TEXT |
| type | TEXT |
| created_at | TEXT |

### saved_prompts
Saved prompts.

| Field | Type |
|-------|------|
| id | INTEGER PK |
| text | TEXT |
| created_at | TEXT |

### saved_scenes
Saved scenes for multi-pass.

| Field | Type |
|-------|------|
| id | INTEGER PK |
| name | TEXT |
| scene_json | TEXT (JSON Scene) |
| created_at | TEXT |

### sessions
Generation sessions.

| Field | Type |
|-------|------|
| id | INTEGER PK |
| name | TEXT |
| created_at | TEXT |
| updated_at | TEXT |

### session_items
Session items (generated images).

| Field | Type | Description |
|-------|------|-------------|
| id | INTEGER PK | |
| session_id | INTEGER FK | Reference to sessions.id |
| file_name | TEXT | PNG file name |
| thumb_name | TEXT | Thumbnail file name |
| source | TEXT | "preset" / "compound" / "remove" / "test" |
| prompt | TEXT | Prompt used |
| negative_prompt | TEXT | Negative prompt used |
| model | TEXT | SD model |
| sampler | TEXT | Sampler |
| steps | INTEGER | Steps |
| cfg_scale | REAL | CFG |
| seed | INTEGER | Seed |
| is_active | BOOLEAN | Active session item (only one per session_id) |
| preset_id | INTEGER FK | Reference to presets.id |
| created_at | TEXT | |

Files are stored at:
- Images: `data/sessions/{session_id}/{file_name}`
- Thumbnails: `data/sessions/{session_id}/thumb/{thumb_name}`

### export_presets
Image export presets.

| Field | Type | Description |
|-------|------|-------------|
| id | INTEGER PK | |
| name | TEXT | |
| format | TEXT | "png", "jpg", "webp" |
| width | INTEGER | Target width |
| height | INTEGER | Target height |
| lock_ratio | BOOLEAN | Preserve aspect ratio |
| quality | INTEGER | Quality (jpg/webp) |
| interpolation | TEXT | nearest/bilinear/bicubic/lanczos |
| created_at | TEXT | |
| updated_at | TEXT | |

## Raw SQL Patterns

The project uses `database/sql` without an ORM. Typical patterns:

**Query:**
```go
rows, err := db.Query("SELECT id, name FROM presets WHERE type_id = ?", typeID)
defer rows.Close()
for rows.Next() {
    var p Preset
    rows.Scan(&p.ID, &p.Name)
}
```

**QueryRow:**
```go
var value string
err := db.QueryRow("SELECT value FROM settings WHERE key = ?", key).Scan(&value)
if err == sql.ErrNoRows { ... }
```

**Exec:**
```go
result, err := db.Exec("INSERT INTO presets (name, prompt) VALUES (?, ?)", name, prompt)
id, _ := result.LastInsertId()
```

**Transaction:**
```go
tx, _ := db.Begin()
tx.Exec(...)
tx.Exec(...)
tx.Commit()
```
