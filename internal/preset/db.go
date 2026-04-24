package preset

import (
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	_ "modernc.org/sqlite"
)

type DB struct {
	db *sql.DB
}

func Open(dbPath string) (*DB, error) {
	if err := os.MkdirAll(filepath.Dir(dbPath), 0755); err != nil {
		return nil, fmt.Errorf("create db dir: %w", err)
	}

	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		return nil, fmt.Errorf("open db: %w", err)
	}

	db.SetMaxOpenConns(1)

	if err := migrate(db); err != nil {
		return nil, fmt.Errorf("migrate: %w", err)
	}

	if err := migrateV2(db); err != nil {
		return nil, fmt.Errorf("migrate v2: %w", err)
	}

	if err := migrateV3(db); err != nil {
		return nil, fmt.Errorf("migrate v3: %w", err)
	}

	return &DB{db: db}, nil
}

func migrate(db *sql.DB) error {
	_, err := db.Exec(`
		CREATE TABLE IF NOT EXISTS presets (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			name TEXT NOT NULL,
			preset_type TEXT NOT NULL DEFAULT '',
			prompt TEXT NOT NULL DEFAULT '',
			negative_prompt TEXT NOT NULL DEFAULT '',
			sampler TEXT NOT NULL DEFAULT 'Euler a',
			steps INTEGER NOT NULL DEFAULT 20,
			cfg_scale REAL NOT NULL DEFAULT 7.0,
			width INTEGER NOT NULL DEFAULT 512,
			height INTEGER NOT NULL DEFAULT 512,
			model_name TEXT NOT NULL DEFAULT '',
			seed INTEGER,
			created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
			updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
		);
		CREATE TABLE IF NOT EXISTS settings (
			key TEXT PRIMARY KEY,
			value TEXT NOT NULL DEFAULT ''
		);
		CREATE TABLE IF NOT EXISTS saved_descriptions (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			text TEXT NOT NULL,
			created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
		);
		CREATE TABLE IF NOT EXISTS saved_prompts (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			text TEXT NOT NULL,
			created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
		)
	`)
	return err
}

func migrateV2(db *sql.DB) error {
	columns := []struct {
		name string
		typ  string
	}{
		{"denoising_strength", "REAL"},
		{"clip_skip", "INTEGER"},
		{"batch_size", "INTEGER"},
		{"batch_count", "INTEGER"},
		{"hires_fix", "INTEGER"},
		{"hires_upscale", "REAL"},
		{"hires_denoising_strength", "REAL"},
		{"hires_upscaler", "TEXT DEFAULT ''"},
		{"vae", "TEXT DEFAULT ''"},
	}
	for _, col := range columns {
		_, err := db.Exec("ALTER TABLE presets ADD COLUMN " + col.name + " " + col.typ)
		if err != nil && strings.Contains(err.Error(), "duplicate column name") {
			continue
		}
		if err != nil {
			return err
		}
	}
	return nil
}

func migrateV3(db *sql.DB) error {
	_, err := db.Exec("ALTER TABLE presets ADD COLUMN schedule_type TEXT DEFAULT ''")
	if err != nil && strings.Contains(err.Error(), "duplicate column name") {
		return nil
	}
	if err != nil {
		return err
	}

	suffixes := []string{"Karras", "Exponential", "Polyexponential"}
	for _, s := range suffixes {
		pattern := "% " + s
		_, _ = db.Exec("UPDATE presets SET sampler = SUBSTR(sampler, 1, LENGTH(sampler) - ?), schedule_type = ? WHERE sampler LIKE ? AND schedule_type = ''",
			len(s)+1, s, pattern)
	}
	return nil
}

func (d *DB) Close() error {
	return d.db.Close()
}

const presetColumns = `id, name, preset_type, prompt, negative_prompt, sampler, schedule_type, steps, cfg_scale, width, height, model_name, seed, denoising_strength, clip_skip, batch_size, batch_count, hires_fix, hires_upscale, hires_denoising_strength, hires_upscaler, vae, created_at, updated_at`

func scanPreset(scanner interface{ Scan(...any) error }, p *Preset) error {
	var seed sql.NullInt64
	var denoisingStrength sql.NullFloat64
	var clipSkip sql.NullInt64
	var batchSize sql.NullInt64
	var batchCount sql.NullInt64
	var hiresFix sql.NullInt64
	var hiresUpscale sql.NullFloat64
	var hiresDenoisingStrength sql.NullFloat64

	err := scanner.Scan(
		&p.ID, &p.Name, &p.PresetType, &p.Prompt, &p.NegativePrompt,
		&p.Sampler, &p.ScheduleType, &p.Steps, &p.CfgScale, &p.Width, &p.Height,
		&p.ModelName, &seed,
		&denoisingStrength, &clipSkip, &batchSize, &batchCount,
		&hiresFix, &hiresUpscale, &hiresDenoisingStrength,
		&p.HiresUpscaler, &p.VAE,
		&p.CreatedAt, &p.UpdatedAt,
	)
	if err != nil {
		return err
	}

	if seed.Valid {
		p.Seed = &seed.Int64
	}
	if denoisingStrength.Valid {
		p.DenoisingStrength = &denoisingStrength.Float64
	}
	if clipSkip.Valid {
		v := int(clipSkip.Int64)
		p.ClipSkip = &v
	}
	if batchSize.Valid {
		v := int(batchSize.Int64)
		p.BatchSize = &v
	}
	if batchCount.Valid {
		v := int(batchCount.Int64)
		p.BatchCount = &v
	}
	if hiresFix.Valid {
		v := hiresFix.Int64 != 0
		p.HiresFix = &v
	}
	if hiresUpscale.Valid {
		p.HiresUpscale = &hiresUpscale.Float64
	}
	if hiresDenoisingStrength.Valid {
		p.HiresDenoisingStrength = &hiresDenoisingStrength.Float64
	}
	return nil
}

func (d *DB) List() ([]Preset, error) {
	rows, err := d.db.Query(`SELECT `+presetColumns+` FROM presets ORDER BY created_at DESC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var presets []Preset
	for rows.Next() {
		var p Preset
		if err := scanPreset(rows, &p); err != nil {
			return nil, err
		}
		presets = append(presets, p)
	}
	return presets, rows.Err()
}

func (d *DB) ListByType(presetType string) ([]Preset, error) {
	rows, err := d.db.Query(`SELECT `+presetColumns+` FROM presets WHERE preset_type = ? ORDER BY created_at DESC`, presetType)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var presets []Preset
	for rows.Next() {
		var p Preset
		if err := scanPreset(rows, &p); err != nil {
			return nil, err
		}
		presets = append(presets, p)
	}
	return presets, rows.Err()
}

func (d *DB) Get(id int64) (*Preset, error) {
	var p Preset
	err := scanPreset(d.db.QueryRow(`SELECT `+presetColumns+` FROM presets WHERE id = ?`, id), &p)
	if err != nil {
		return nil, err
	}
	return &p, nil
}

func (d *DB) Create(p *Preset) error {
	result, err := d.db.Exec(`INSERT INTO presets (name, preset_type, prompt, negative_prompt, sampler, schedule_type, steps, cfg_scale, width, height, model_name, seed, denoising_strength, clip_skip, batch_size, batch_count, hires_fix, hires_upscale, hires_denoising_strength, hires_upscaler, vae) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		p.Name, p.PresetType, p.Prompt, p.NegativePrompt, p.Sampler, p.ScheduleType, p.Steps, p.CfgScale, p.Width, p.Height, p.ModelName, p.Seed,
		p.DenoisingStrength, p.ClipSkip, p.BatchSize, p.BatchCount, p.HiresFix, p.HiresUpscale, p.HiresDenoisingStrength, p.HiresUpscaler, p.VAE)
	if err != nil {
		return err
	}
	p.ID, _ = result.LastInsertId()
	return nil
}

func (d *DB) Update(p *Preset) error {
	_, err := d.db.Exec(`UPDATE presets SET name=?, preset_type=?, prompt=?, negative_prompt=?, sampler=?, schedule_type=?, steps=?, cfg_scale=?, width=?, height=?, model_name=?, seed=?, denoising_strength=?, clip_skip=?, batch_size=?, batch_count=?, hires_fix=?, hires_upscale=?, hires_denoising_strength=?, hires_upscaler=?, vae=?, updated_at=CURRENT_TIMESTAMP WHERE id=?`,
		p.Name, p.PresetType, p.Prompt, p.NegativePrompt, p.Sampler, p.ScheduleType, p.Steps, p.CfgScale, p.Width, p.Height, p.ModelName, p.Seed,
		p.DenoisingStrength, p.ClipSkip, p.BatchSize, p.BatchCount, p.HiresFix, p.HiresUpscale, p.HiresDenoisingStrength, p.HiresUpscaler, p.VAE, p.ID)
	return err
}

func (d *DB) Delete(id int64) error {
	_, err := d.db.Exec(`DELETE FROM presets WHERE id = ?`, id)
	return err
}

func (d *DB) GetByIDs(ids []int64) ([]Preset, error) {
	if len(ids) == 0 {
		return nil, nil
	}
	placeholders := ""
	args := make([]any, len(ids))
	for i, id := range ids {
		if i > 0 {
			placeholders += ","
		}
		placeholders += "?"
		args[i] = id
	}
	rows, err := d.db.Query(`SELECT `+presetColumns+` FROM presets WHERE id IN (`+placeholders+`) ORDER BY created_at DESC`, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var presets []Preset
	for rows.Next() {
		var p Preset
		if err := scanPreset(rows, &p); err != nil {
			return nil, err
		}
		presets = append(presets, p)
	}
	return presets, rows.Err()
}

func (d *DB) CreateBatch(items []Preset) ([]Preset, error) {
	tx, err := d.db.Begin()
	if err != nil {
		return nil, err
	}

	var created []Preset
	for _, item := range items {
		result, err := tx.Exec(`INSERT INTO presets (name, preset_type, prompt, negative_prompt, sampler, schedule_type, steps, cfg_scale, width, height, model_name, seed, denoising_strength, clip_skip, batch_size, batch_count, hires_fix, hires_upscale, hires_denoising_strength, hires_upscaler, vae) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
			item.Name, item.PresetType, item.Prompt, item.NegativePrompt, item.Sampler, item.ScheduleType, item.Steps, item.CfgScale, item.Width, item.Height, item.ModelName, item.Seed,
			item.DenoisingStrength, item.ClipSkip, item.BatchSize, item.BatchCount, item.HiresFix, item.HiresUpscale, item.HiresDenoisingStrength, item.HiresUpscaler, item.VAE)
		if err != nil {
			tx.Rollback()
			return nil, err
		}
		id, _ := result.LastInsertId()
		p := item
		p.ID = id
		created = append(created, p)
	}

	if err := tx.Commit(); err != nil {
		return nil, err
	}
	return created, nil
}

func (d *DB) GetSetting(key string) (string, error) {
	var value string
	err := d.db.QueryRow(`SELECT value FROM settings WHERE key = ?`, key).Scan(&value)
	if err == sql.ErrNoRows {
		return "", nil
	}
	return value, err
}

func (d *DB) GetAllSettings() (map[string]string, error) {
	rows, err := d.db.Query(`SELECT key, value FROM settings`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	settings := make(map[string]string)
	for rows.Next() {
		var k, v string
		if err := rows.Scan(&k, &v); err != nil {
			return nil, err
		}
		settings[k] = v
	}
	return settings, rows.Err()
}

func (d *DB) SetSetting(key, value string) error {
	_, err := d.db.Exec(`INSERT INTO settings (key, value) VALUES (?, ?) ON CONFLICT(key) DO UPDATE SET value = ?`, key, value, value)
	return err
}

func (d *DB) ListDescriptions() ([]SavedDescription, error) {
	rows, err := d.db.Query(`SELECT id, text, created_at FROM saved_descriptions ORDER BY created_at DESC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var items []SavedDescription
	for rows.Next() {
		var s SavedDescription
		if err := rows.Scan(&s.ID, &s.Text, &s.CreatedAt); err != nil {
			return nil, err
		}
		items = append(items, s)
	}
	return items, rows.Err()
}

func (d *DB) CreateDescription(text string) (*SavedDescription, error) {
	result, err := d.db.Exec(`INSERT INTO saved_descriptions (text) VALUES (?)`, text)
	if err != nil {
		return nil, err
	}
	id, _ := result.LastInsertId()
	return &SavedDescription{ID: id, Text: text}, nil
}

func (d *DB) DeleteDescription(id int64) error {
	_, err := d.db.Exec(`DELETE FROM saved_descriptions WHERE id = ?`, id)
	return err
}

func (d *DB) ListPrompts() ([]SavedPrompt, error) {
	rows, err := d.db.Query(`SELECT id, text, created_at FROM saved_prompts ORDER BY created_at DESC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var items []SavedPrompt
	for rows.Next() {
		var s SavedPrompt
		if err := rows.Scan(&s.ID, &s.Text, &s.CreatedAt); err != nil {
			return nil, err
		}
		items = append(items, s)
	}
	return items, rows.Err()
}

func (d *DB) CreatePrompt(text string) (*SavedPrompt, error) {
	result, err := d.db.Exec(`INSERT INTO saved_prompts (text) VALUES (?)`, text)
	if err != nil {
		return nil, err
	}
	id, _ := result.LastInsertId()
	return &SavedPrompt{ID: id, Text: text}, nil
}

func (d *DB) DeletePrompt(id int64) error {
	_, err := d.db.Exec(`DELETE FROM saved_prompts WHERE id = ?`, id)
	return err
}
