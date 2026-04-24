package preset

import (
	"database/sql"
	"fmt"
	"os"
	"path/filepath"

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

func (d *DB) Close() error {
	return d.db.Close()
}

func (d *DB) List() ([]Preset, error) {
	rows, err := d.db.Query(`SELECT id, name, preset_type, prompt, negative_prompt, sampler, steps, cfg_scale, width, height, model_name, seed, created_at, updated_at FROM presets ORDER BY created_at DESC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var presets []Preset
	for rows.Next() {
		var p Preset
		var seed sql.NullInt64
		if err := rows.Scan(&p.ID, &p.Name, &p.PresetType, &p.Prompt, &p.NegativePrompt, &p.Sampler, &p.Steps, &p.CfgScale, &p.Width, &p.Height, &p.ModelName, &seed, &p.CreatedAt, &p.UpdatedAt); err != nil {
			return nil, err
		}
		if seed.Valid {
			p.Seed = &seed.Int64
		}
		presets = append(presets, p)
	}
	return presets, rows.Err()
}

func (d *DB) ListByType(presetType string) ([]Preset, error) {
	rows, err := d.db.Query(`SELECT id, name, preset_type, prompt, negative_prompt, sampler, steps, cfg_scale, width, height, model_name, seed, created_at, updated_at FROM presets WHERE preset_type = ? ORDER BY created_at DESC`, presetType)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var presets []Preset
	for rows.Next() {
		var p Preset
		var seed sql.NullInt64
		if err := rows.Scan(&p.ID, &p.Name, &p.PresetType, &p.Prompt, &p.NegativePrompt, &p.Sampler, &p.Steps, &p.CfgScale, &p.Width, &p.Height, &p.ModelName, &seed, &p.CreatedAt, &p.UpdatedAt); err != nil {
			return nil, err
		}
		if seed.Valid {
			p.Seed = &seed.Int64
		}
		presets = append(presets, p)
	}
	return presets, rows.Err()
}

func (d *DB) Get(id int64) (*Preset, error) {
	var p Preset
	var seed sql.NullInt64
	err := d.db.QueryRow(`SELECT id, name, preset_type, prompt, negative_prompt, sampler, steps, cfg_scale, width, height, model_name, seed, created_at, updated_at FROM presets WHERE id = ?`, id).
		Scan(&p.ID, &p.Name, &p.PresetType, &p.Prompt, &p.NegativePrompt, &p.Sampler, &p.Steps, &p.CfgScale, &p.Width, &p.Height, &p.ModelName, &seed, &p.CreatedAt, &p.UpdatedAt)
	if err != nil {
		return nil, err
	}
	if seed.Valid {
		p.Seed = &seed.Int64
	}
	return &p, nil
}

func (d *DB) Create(p *Preset) error {
	result, err := d.db.Exec(`INSERT INTO presets (name, preset_type, prompt, negative_prompt, sampler, steps, cfg_scale, width, height, model_name, seed) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		p.Name, p.PresetType, p.Prompt, p.NegativePrompt, p.Sampler, p.Steps, p.CfgScale, p.Width, p.Height, p.ModelName, p.Seed)
	if err != nil {
		return err
	}
	p.ID, _ = result.LastInsertId()
	return nil
}

func (d *DB) Update(p *Preset) error {
	_, err := d.db.Exec(`UPDATE presets SET name=?, preset_type=?, prompt=?, negative_prompt=?, sampler=?, steps=?, cfg_scale=?, width=?, height=?, model_name=?, seed=?, updated_at=CURRENT_TIMESTAMP WHERE id=?`,
		p.Name, p.PresetType, p.Prompt, p.NegativePrompt, p.Sampler, p.Steps, p.CfgScale, p.Width, p.Height, p.ModelName, p.Seed, p.ID)
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
	rows, err := d.db.Query(`SELECT id, name, preset_type, prompt, negative_prompt, sampler, steps, cfg_scale, width, height, model_name, seed, created_at, updated_at FROM presets WHERE id IN (`+placeholders+`) ORDER BY created_at DESC`, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var presets []Preset
	for rows.Next() {
		var p Preset
		var seed sql.NullInt64
		if err := rows.Scan(&p.ID, &p.Name, &p.PresetType, &p.Prompt, &p.NegativePrompt, &p.Sampler, &p.Steps, &p.CfgScale, &p.Width, &p.Height, &p.ModelName, &seed, &p.CreatedAt, &p.UpdatedAt); err != nil {
			return nil, err
		}
		if seed.Valid {
			p.Seed = &seed.Int64
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
		result, err := tx.Exec(`INSERT INTO presets (name, preset_type, prompt, negative_prompt, sampler, steps, cfg_scale, width, height, model_name, seed) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
			item.Name, item.PresetType, item.Prompt, item.NegativePrompt, item.Sampler, item.Steps, item.CfgScale, item.Width, item.Height, item.ModelName, item.Seed)
		if err != nil {
			tx.Rollback()
			return nil, err
		}
		id, _ := result.LastInsertId()
		created = append(created, Preset{
			ID:             id,
			Name:           item.Name,
			PresetType:     item.PresetType,
			Prompt:         item.Prompt,
			NegativePrompt: item.NegativePrompt,
			Sampler:        item.Sampler,
			Steps:          item.Steps,
			CfgScale:       item.CfgScale,
			Width:          item.Width,
			Height:         item.Height,
			ModelName:      item.ModelName,
			Seed:           item.Seed,
		})
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
