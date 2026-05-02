# SD Studio — База данных

## SQLite

Путь: `data/presets.db` (настраивается через `DB_PATH`).

Миграции выполняются автоматически при `preset.Open()` (версии 1-10).

## Схема

### presets
Основная таблица пресетов генерации.

| Поле | Тип | Описание |
|------|-----|----------|
| id | INTEGER PK | Auto-increment |
| name | TEXT | Название пресета |
| type_id | INTEGER FK | Ссылка на preset_types.id |
| preset_type | TEXT | Название типа (денормрализовано) |
| prompt | TEXT | Positive prompt |
| negative_prompt | TEXT | Negative prompt |
| sampler | TEXT | Sampler name |
| schedule_type | TEXT | "auto", "karras", "exponential" |
| steps | INTEGER | Количество шагов |
| cfg_scale | REAL | CFG Scale |
| width | INTEGER | Ширина |
| height | INTEGER | Высота |
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
| sort_order | INTEGER | Порядок сортировки |
| created_at | TEXT | ISO 8601 |
| updated_at | TEXT | ISO 8601 |

### preset_types
Типы пресетов (Anime, Realistic, etc.)

| Поле | Тип | Описание |
|------|-----|----------|
| id | INTEGER PK | |
| name | TEXT UNIQUE | Название типа |
| description | TEXT | Описание |
| sort_order | INTEGER | Порядок |
| created_at | TEXT | |

### compound_presets
Compound пресеты (pipeline из нескольких шагов).

| Поле | Тип | Описание |
|------|-----|----------|
| id | INTEGER PK | |
| name | TEXT | Название |
| description | TEXT | Описание |
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
Key-value хранилище настроек.

| Поле | Тип | Описание |
|------|-----|----------|
| key | TEXT PK | Ключ настройки |
| value | TEXT | Значение |

**Ключи:**

| Ключ | Описание | Default |
|------|----------|---------|
| `llm_url` | URL LLM API | `http://localhost:1234` |
| `sd_url` | URL SD WebUI | `http://localhost:7860` |
| `rembg_url` | URL rembg API | `http://localhost:5000` |
| `llm_generate_model` | Модель для промптов | из env |
| `llm_analyze_model` | Модель для vision | из env |
| `llm_backend` | ollama/lmstudio | из env |
| `llm_max_tokens` | Max tokens | `256` |
| `llm_keep_alive` | Ollama keep_alive | `5m` |
| `llm_num_ctx` | Ollama context | `4096` |
| `llm_num_gpu` | Ollama GPU layers | `1` |
| `sd_prompt_instruction` | System prompt для SD | DefaultSDPromptInstruction |
| `kids_mode` | Kids mode on/off | `false` |
| `kids_pin_hash` | SHA256 хэш PIN | — |
| `kids_pin_attempts` | Попытки ввода PIN | `0` |
| `kids_pin_lockout` | Время блокировки | — |
| `kids_categories` | JSON включённых категорий | — |
| `fi_mode` | From Image: режим | `img2img` |
| `fi_preset_id` | From Image: preset | — |
| `fi_gen_mode` | From Image: gen mode | `preset` |
| `fi_denoising` | From Image: denoising | `0.5` |
| `fi_extra_negative` | From Image: negative | — |
| `fi_analyze_mode` | From Image: quick/deep | `quick` |
| `fi_mask_padding` | From Image: mask dilation (px) | `8` |
| `fi_mask_feather` | From Image: mask feathering (px) | `8` |
| `browser_folder` | File browser: папка | — |

### saved_descriptions
Сохранённые пользовательские описания.

| Поле | Тип |
|------|-----|
| id | INTEGER PK |
| text | TEXT |
| name | TEXT |
| negative_prompt | TEXT |
| type | TEXT |
| created_at | TEXT |

### saved_prompts
Сохранённые промпты.

| Поле | Тип |
|------|-----|
| id | INTEGER PK |
| text | TEXT |
| created_at | TEXT |

### saved_scenes
Сохранённые сцены для multi-pass.

| Поле | Тип |
|------|-----|
| id | INTEGER PK |
| name | TEXT |
| scene_json | TEXT (JSON Scene) |
| created_at | TEXT |

### sessions
Сессии генерации.

| Поле | Тип |
|------|-----|
| id | INTEGER PK |
| name | TEXT |
| created_at | TEXT |
| updated_at | TEXT |

### session_items
Элементы сессии (сгенерированные изображения).

| Поле | Тип | Описание |
|------|-----|----------|
| id | INTEGER PK | |
| session_id | INTEGER FK | Ссылка на sessions.id |
| file_name | TEXT | Имя файла PNG |
| thumb_name | TEXT | Имя файла thumbnail |
| source | TEXT | "preset" / "compound" / "remove" / "test" |
| prompt | TEXT | Использованный промпт |
| negative_prompt | TEXT | Использованный negative |
| model | TEXT | SD модель |
| sampler | TEXT | Sampler |
| steps | INTEGER | Шаги |
| cfg_scale | REAL | CFG |
| seed | INTEGER | Seed |
| created_at | TEXT | |

Файлы хранятся в:
- Изображения: `data/sessions/{session_id}/{file_name}`
- Thumbnails: `data/sessions/{session_id}/thumb/{thumb_name}`

### export_presets
Пресеты экспорта изображений.

| Поле | Тип | Описание |
|------|-----|----------|
| id | INTEGER PK | |
| name | TEXT | |
| format | TEXT | "png", "jpg", "webp" |
| width | INTEGER | Target width |
| height | INTEGER | Target height |
| lock_ratio | BOOLEAN | Сохранять пропорции |
| quality | INTEGER | Качество (jpg/webp) |
| interpolation | TEXT | nearest/bilinear/bicubic/lanczos |
| created_at | TEXT | |
| updated_at | TEXT | |

## Raw SQL паттерны

Проект использует `database/sql` без ORM. Типичные паттерны:

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
