package preset

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"io/fs"
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

	if err := migrateV4(db); err != nil {
		return nil, fmt.Errorf("migrate v4: %w", err)
	}

	if err := migrateV5(db); err != nil {
		return nil, fmt.Errorf("migrate v5: %w", err)
	}

	if err := migrateV6(db); err != nil {
		return nil, fmt.Errorf("migrate v6: %w", err)
	}

	if err := migrateV7(db); err != nil {
		return nil, fmt.Errorf("migrate v7: %w", err)
	}

	if err := migrateV8(db); err != nil {
		return nil, fmt.Errorf("migrate v8: %w", err)
	}

	if err := migrateV9(db); err != nil {
		return nil, fmt.Errorf("migrate v9: %w", err)
	}

	if err := migrateV10(db); err != nil {
		return nil, fmt.Errorf("migrate v10: %w", err)
	}

	if err := migrateV11(db); err != nil {
		return nil, fmt.Errorf("migrate v11: %w", err)
	}

	if err := migrateV12(db); err != nil {
		return nil, fmt.Errorf("migrate v12: %w", err)
	}

	if err := migrateV13(db); err != nil {
		return nil, fmt.Errorf("migrate v13: %w", err)
	}

	if err := migrateV14(db); err != nil {
		return nil, fmt.Errorf("migrate v14: %w", err)
	}

	if err := migrateV15(db); err != nil {
		return nil, fmt.Errorf("migrate v15: %w", err)
	}

	if err := migrateV16(db); err != nil {
		return nil, fmt.Errorf("migrate v16: %w", err)
	}

	if err := migrateV17(db); err != nil {
		return nil, fmt.Errorf("migrate v17: %w", err)
	}

	if err := migrateV18(db); err != nil {
		return nil, fmt.Errorf("migrate v18: %w", err)
	}

	if err := migrateV19(db); err != nil {
		return nil, fmt.Errorf("migrate v19: %w", err)
	}

	if err := migrateV20(db); err != nil {
		return nil, fmt.Errorf("migrate v20: %w", err)
	}

	if err := migrateV21(db); err != nil {
		return nil, fmt.Errorf("migrate v21: %w", err)
	}

	if err := migrateV22(db); err != nil {
		return nil, fmt.Errorf("migrate v22: %w", err)
	}

	if err := migrateV23(db); err != nil {
		return nil, fmt.Errorf("migrate v23: %w", err)
	}

	if err := migrateV24(db); err != nil {
		return nil, fmt.Errorf("migrate v24: %w", err)
	}

	if err := migrateV25(db); err != nil {
		return nil, fmt.Errorf("migrate v25: %w", err)
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

func addColumnIfNotExists(db *sql.DB, table, col, typ string) error {
	_, err := db.Exec("ALTER TABLE " + table + " ADD COLUMN " + col + " " + typ)
	if err != nil && strings.Contains(err.Error(), "duplicate column name") {
		return nil
	}
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
		if err := addColumnIfNotExists(db, "presets", col.name, col.typ); err != nil {
			return err
		}
	}
	return nil
}

func migrateV3(db *sql.DB) error {
	if err := addColumnIfNotExists(db, "presets", "schedule_type", "TEXT DEFAULT ''"); err != nil {
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

func migrateV4(db *sql.DB) error {
	fixes := map[string]string{
		"karras":        "Karras",
		"exponential":   "Exponential",
		"polyexponential": "Polyexponential",
	}
	for lower, proper := range fixes {
		_, _ = db.Exec("UPDATE presets SET schedule_type = ? WHERE schedule_type = ?", proper, lower)
	}
	return nil
}

func (d *DB) Close() error {
	return d.db.Close()
}

func (d *DB) DB() *sql.DB {
	return d.db
}

const presetColumns = `id, name, preset_type, prompt, negative_prompt, sampler, schedule_type, steps, cfg_scale, model_name, seed, denoising_strength, clip_skip, batch_size, batch_count, vae, type_id, tags, loras, is_bundled, created_at, updated_at`

func scanPreset(scanner interface{ Scan(...any) error }, p *Preset) error {
	var seed sql.NullInt64
	var denoisingStrength sql.NullFloat64
	var clipSkip sql.NullInt64
	var batchSize sql.NullInt64
	var batchCount sql.NullInt64
	var typeID sql.NullInt64

	err := scanner.Scan(
		&p.ID, &p.Name, &p.PresetType, &p.Prompt, &p.NegativePrompt,
		&p.Sampler, &p.ScheduleType, &p.Steps, &p.CfgScale,
		&p.ModelName, &seed,
		&denoisingStrength, &clipSkip, &batchSize, &batchCount,
		&p.VAE,
		&typeID, &p.Tags, &p.Loras, &p.IsBundled,
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
	if typeID.Valid {
		p.TypeID = &typeID.Int64
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

func (d *DB) GetBundledInstallStatus(sdModels []string, loraModels []string) ([]PresetInstallStatus, error) {
	sdSet := make(map[string]bool, len(sdModels))
	for _, m := range sdModels {
		sdSet[strings.TrimSuffix(m, ".safetensors")] = true
	}
	loraSet := make(map[string]bool, len(loraModels))
	for _, m := range loraModels {
		loraSet[strings.TrimSuffix(m, ".safetensors")] = true
	}

	rows, err := d.db.Query(`SELECT id, name, model_name, loras FROM presets WHERE is_bundled = 1`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var result []PresetInstallStatus
	for rows.Next() {
		var id int64
		var name, modelName, lorasJSON string
		if err := rows.Scan(&id, &name, &modelName, &lorasJSON); err != nil {
			return nil, err
		}

		status := PresetInstallStatus{ID: id, Name: name}

		if modelName != "" && !sdSet[modelName] {
			status.MissingSD = append(status.MissingSD, modelName)
		}

		if lorasJSON != "" && lorasJSON != "[]" {
			var loras []LoRAEntry
			if json.Unmarshal([]byte(lorasJSON), &loras) == nil {
				for _, l := range loras {
					if !loraSet[l.Name] {
						status.MissingLoRA = append(status.MissingLoRA, l.Name)
					}
				}
			}
		}

		status.Installed = len(status.MissingSD) == 0 && len(status.MissingLoRA) == 0
		result = append(result, status)
	}
	return result, rows.Err()
}

func (d *DB) Create(p *Preset) error {
	result, err := d.db.Exec(`INSERT INTO presets (name, preset_type, prompt, negative_prompt, sampler, schedule_type, steps, cfg_scale, model_name, seed, denoising_strength, clip_skip, batch_size, batch_count, vae, type_id, tags, loras) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		p.Name, p.PresetType, p.Prompt, p.NegativePrompt, p.Sampler, p.ScheduleType, p.Steps, p.CfgScale, p.ModelName, p.Seed,
		p.DenoisingStrength, p.ClipSkip, p.BatchSize, p.BatchCount, p.VAE,
		p.TypeID, p.Tags, p.Loras)
	if err != nil {
		return err
	}
	p.ID, _ = result.LastInsertId()
	return nil
}

func (d *DB) Update(p *Preset) error {
	_, err := d.db.Exec(`UPDATE presets SET name=?, preset_type=?, prompt=?, negative_prompt=?, sampler=?, schedule_type=?, steps=?, cfg_scale=?, model_name=?, seed=?, denoising_strength=?, clip_skip=?, batch_size=?, batch_count=?, vae=?, type_id=?, tags=?, loras=?, updated_at=CURRENT_TIMESTAMP WHERE id=?`,
		p.Name, p.PresetType, p.Prompt, p.NegativePrompt, p.Sampler, p.ScheduleType, p.Steps, p.CfgScale, p.ModelName, p.Seed,
		p.DenoisingStrength, p.ClipSkip, p.BatchSize, p.BatchCount, p.VAE,
		p.TypeID, p.Tags, p.Loras, p.ID)
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
		result, err := tx.Exec(`INSERT INTO presets (name, preset_type, prompt, negative_prompt, sampler, schedule_type, steps, cfg_scale, model_name, seed, denoising_strength, clip_skip, batch_size, batch_count, vae, type_id, tags, loras) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
			item.Name, item.PresetType, item.Prompt, item.NegativePrompt, item.Sampler, item.ScheduleType, item.Steps, item.CfgScale, item.ModelName, item.Seed,
			item.DenoisingStrength, item.ClipSkip, item.BatchSize, item.BatchCount, item.VAE,
			item.TypeID, item.Tags, item.Loras)
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
	rows, err := d.db.Query(`SELECT id, text, name, negative_prompt, type, created_at FROM saved_descriptions ORDER BY created_at DESC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var items []SavedDescription
	for rows.Next() {
		var s SavedDescription
		if err := rows.Scan(&s.ID, &s.Text, &s.Name, &s.NegativePrompt, &s.Type, &s.CreatedAt); err != nil {
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

func (d *DB) CreateDescriptionFull(s *SavedDescription) (*SavedDescription, error) {
	result, err := d.db.Exec(`INSERT INTO saved_descriptions (text, name, negative_prompt, type) VALUES (?, ?, ?, ?)`,
		s.Text, s.Name, s.NegativePrompt, s.Type)
	if err != nil {
		return nil, err
	}
	s.ID, _ = result.LastInsertId()
	return s, nil
}

func (d *DB) UpdateDescription(s *SavedDescription) error {
	_, err := d.db.Exec(`UPDATE saved_descriptions SET text=?, name=?, negative_prompt=?, type=? WHERE id=?`,
		s.Text, s.Name, s.NegativePrompt, s.Type, s.ID)
	return err
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

func migrateV5(db *sql.DB) error {
	_, err := db.Exec(`
		CREATE TABLE IF NOT EXISTS preset_types (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			name TEXT NOT NULL,
			description TEXT NOT NULL DEFAULT '',
			sort_order INTEGER NOT NULL DEFAULT 0,
			created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
		)
	`)
	if err != nil {
		return err
	}

	newCols := []struct {
		name string
		typ  string
	}{
		{"type_id", "INTEGER"},
		{"tags", "TEXT DEFAULT ''"},
		{"loras", "TEXT DEFAULT ''"},
	}
	for _, col := range newCols {
		if err := addColumnIfNotExists(db, "presets", col.name, col.typ); err != nil {
			return err
		}
	}
	return nil
}

func (d *DB) ListPresetTypes() ([]PresetType, error) {
	rows, err := d.db.Query(`SELECT id, name, description, sort_order, created_at FROM preset_types ORDER BY sort_order, name`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var items []PresetType
	for rows.Next() {
		var pt PresetType
		if err := rows.Scan(&pt.ID, &pt.Name, &pt.Description, &pt.SortOrder, &pt.CreatedAt); err != nil {
			return nil, err
		}
		items = append(items, pt)
	}
	return items, rows.Err()
}

func (d *DB) GetPresetType(id int64) (*PresetType, error) {
	var pt PresetType
	err := d.db.QueryRow(`SELECT id, name, description, sort_order, created_at FROM preset_types WHERE id = ?`, id).
		Scan(&pt.ID, &pt.Name, &pt.Description, &pt.SortOrder, &pt.CreatedAt)
	if err != nil {
		return nil, err
	}
	return &pt, nil
}

func (d *DB) CreatePresetType(pt *PresetType) error {
	result, err := d.db.Exec(`INSERT INTO preset_types (name, description, sort_order) VALUES (?, ?, ?)`,
		pt.Name, pt.Description, pt.SortOrder)
	if err != nil {
		return err
	}
	pt.ID, _ = result.LastInsertId()
	return nil
}

func (d *DB) UpdatePresetType(pt *PresetType) error {
	_, err := d.db.Exec(`UPDATE preset_types SET name=?, description=?, sort_order=? WHERE id=?`,
		pt.Name, pt.Description, pt.SortOrder, pt.ID)
	return err
}

func (d *DB) DeletePresetType(id int64) error {
	_, err := d.db.Exec(`DELETE FROM preset_types WHERE id = ?`, id)
	return err
}

func (d *DB) GetAllTags() ([]string, error) {
	rows, err := d.db.Query(`SELECT DISTINCT tags FROM presets WHERE tags != ''`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	tagSet := make(map[string]bool)
	for rows.Next() {
		var tagsStr string
		if err := rows.Scan(&tagsStr); err != nil {
			return nil, err
		}
		for _, t := range strings.Split(tagsStr, ",") {
			t = strings.TrimSpace(t)
			if t != "" {
				tagSet[t] = true
			}
		}
	}

	var tags []string
	for t := range tagSet {
		tags = append(tags, t)
	}
	return tags, rows.Err()
}

func migrateV6(db *sql.DB) error {
	_, err := db.Exec(`
		CREATE TABLE IF NOT EXISTS compound_presets (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			name TEXT NOT NULL,
			description TEXT NOT NULL DEFAULT '',
			created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
			updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
		);
		CREATE TABLE IF NOT EXISTS compound_preset_steps (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			compound_preset_id INTEGER NOT NULL,
			step_order INTEGER NOT NULL,
			preset_id INTEGER NOT NULL,
			width INTEGER NOT NULL DEFAULT 512,
			height INTEGER NOT NULL DEFAULT 512,
			denoising_strength REAL NOT NULL DEFAULT 0.5,
			FOREIGN KEY (compound_preset_id) REFERENCES compound_presets(id) ON DELETE CASCADE,
			FOREIGN KEY (preset_id) REFERENCES presets(id)
		)
	`)
	return err
}

func (d *DB) ListCompoundPresets() ([]CompoundPreset, error) {
	rows, err := d.db.Query(`SELECT id, name, description, created_at, updated_at FROM compound_presets ORDER BY created_at DESC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var items []CompoundPreset
	for rows.Next() {
		var cp CompoundPreset
		if err := rows.Scan(&cp.ID, &cp.Name, &cp.Description, &cp.CreatedAt, &cp.UpdatedAt); err != nil {
			return nil, err
		}
		items = append(items, cp)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	for i := range items {
		steps, err := d.getCompoundSteps(items[i].ID)
		if err != nil {
			return nil, err
		}
		items[i].Steps = steps
	}
	return items, nil
}

func (d *DB) GetCompoundPreset(id int64) (*CompoundPreset, error) {
	var cp CompoundPreset
	err := d.db.QueryRow(`SELECT id, name, description, created_at, updated_at FROM compound_presets WHERE id = ?`, id).
		Scan(&cp.ID, &cp.Name, &cp.Description, &cp.CreatedAt, &cp.UpdatedAt)
	if err != nil {
		return nil, err
	}

	steps, err := d.getCompoundSteps(id)
	if err != nil {
		return nil, err
	}
	cp.Steps = steps
	return &cp, nil
}

func (d *DB) getCompoundSteps(compoundPresetID int64) ([]CompoundPresetStep, error) {
	rows, err := d.db.Query(
		`SELECT id, compound_preset_id, step_order, preset_id, denoising_strength, resolution_id FROM compound_preset_steps WHERE compound_preset_id = ? ORDER BY step_order`,
		compoundPresetID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var steps []CompoundPresetStep
	for rows.Next() {
		var s CompoundPresetStep
		var resolutionID sql.NullInt64
		if err := rows.Scan(&s.ID, &s.CompoundPresetID, &s.StepOrder, &s.PresetID, &s.DenoisingStrength, &resolutionID); err != nil {
			return nil, err
		}
		if resolutionID.Valid {
			s.ResolutionID = &resolutionID.Int64
		}
		steps = append(steps, s)
	}
	return steps, rows.Err()
}

func (d *DB) CreateCompoundPreset(cp *CompoundPreset) error {
	tx, err := d.db.Begin()
	if err != nil {
		return err
	}

	result, err := tx.Exec(`INSERT INTO compound_presets (name, description) VALUES (?, ?)`, cp.Name, cp.Description)
	if err != nil {
		tx.Rollback()
		return err
	}
	cp.ID, _ = result.LastInsertId()

	for i, step := range cp.Steps {
		_, err := tx.Exec(
			`INSERT INTO compound_preset_steps (compound_preset_id, step_order, preset_id, denoising_strength, resolution_id) VALUES (?, ?, ?, ?, ?)`,
			cp.ID, i+1, step.PresetID, step.DenoisingStrength, step.ResolutionID,
		)
		if err != nil {
			tx.Rollback()
			return err
		}
	}

	return tx.Commit()
}

func (d *DB) UpdateCompoundPreset(cp *CompoundPreset) error {
	tx, err := d.db.Begin()
	if err != nil {
		return err
	}

	_, err = tx.Exec(`UPDATE compound_presets SET name=?, description=?, updated_at=CURRENT_TIMESTAMP WHERE id=?`,
		cp.Name, cp.Description, cp.ID)
	if err != nil {
		tx.Rollback()
		return err
	}

	_, err = tx.Exec(`DELETE FROM compound_preset_steps WHERE compound_preset_id = ?`, cp.ID)
	if err != nil {
		tx.Rollback()
		return err
	}

	for i, step := range cp.Steps {
		_, err := tx.Exec(
			`INSERT INTO compound_preset_steps (compound_preset_id, step_order, preset_id, denoising_strength, resolution_id) VALUES (?, ?, ?, ?, ?)`,
			cp.ID, i+1, step.PresetID, step.DenoisingStrength, step.ResolutionID,
		)
		if err != nil {
			tx.Rollback()
			return err
		}
	}

	return tx.Commit()
}

func (d *DB) DeleteCompoundPreset(id int64) error {
	tx, err := d.db.Begin()
	if err != nil {
		return err
	}
	_, err = tx.Exec(`DELETE FROM compound_preset_steps WHERE compound_preset_id = ?`, id)
	if err != nil {
		tx.Rollback()
		return err
	}
	_, err = tx.Exec(`DELETE FROM compound_presets WHERE id = ?`, id)
	if err != nil {
		tx.Rollback()
		return err
	}
	return tx.Commit()
}

func (d *DB) GetCompoundPresetsByIDs(ids []int64) ([]CompoundPreset, error) {
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
	rows, err := d.db.Query(`SELECT id, name, description, created_at, updated_at FROM compound_presets WHERE id IN (`+placeholders+`) ORDER BY created_at DESC`, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var items []CompoundPreset
	for rows.Next() {
		var cp CompoundPreset
		if err := rows.Scan(&cp.ID, &cp.Name, &cp.Description, &cp.CreatedAt, &cp.UpdatedAt); err != nil {
			return nil, err
		}
		items = append(items, cp)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	for i := range items {
		steps, err := d.getCompoundSteps(items[i].ID)
		if err != nil {
			return nil, err
		}
		items[i].Steps = steps
	}
	return items, nil
}

func migrateV7(db *sql.DB) error {
	_, err := db.Exec(`
		CREATE TABLE IF NOT EXISTS saved_scenes (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			name TEXT NOT NULL,
			scene_json TEXT NOT NULL,
			created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
		)
	`)
	return err
}

func migrateV8(db *sql.DB) error {
	newCols := []struct {
		name string
		typ  string
	}{
		{"name", "TEXT DEFAULT ''"},
		{"negative_prompt", "TEXT DEFAULT ''"},
		{"type", "TEXT DEFAULT ''"},
	}
	for _, col := range newCols {
		if err := addColumnIfNotExists(db, "saved_descriptions", col.name, col.typ); err != nil {
			return err
		}
	}
	return nil
}

func (d *DB) ListSavedScenes() ([]SavedScene, error) {
	rows, err := d.db.Query(`SELECT id, name, scene_json, created_at FROM saved_scenes ORDER BY created_at DESC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var items []SavedScene
	for rows.Next() {
		var s SavedScene
		if err := rows.Scan(&s.ID, &s.Name, &s.SceneJSON, &s.CreatedAt); err != nil {
			return nil, err
		}
		items = append(items, s)
	}
	return items, rows.Err()
}

func (d *DB) GetSavedScene(id int64) (*SavedScene, error) {
	var s SavedScene
	err := d.db.QueryRow(`SELECT id, name, scene_json, created_at FROM saved_scenes WHERE id = ?`, id).
		Scan(&s.ID, &s.Name, &s.SceneJSON, &s.CreatedAt)
	if err != nil {
		return nil, err
	}
	return &s, nil
}

func (d *DB) CreateSavedScene(s *SavedScene) error {
	result, err := d.db.Exec(`INSERT INTO saved_scenes (name, scene_json) VALUES (?, ?)`, s.Name, s.SceneJSON)
	if err != nil {
		return err
	}
	s.ID, _ = result.LastInsertId()
	return nil
}

func (d *DB) UpdateSavedScene(s *SavedScene) error {
	_, err := d.db.Exec(`UPDATE saved_scenes SET name=?, scene_json=? WHERE id=?`, s.Name, s.SceneJSON, s.ID)
	return err
}

func (d *DB) DeleteSavedScene(id int64) error {
	_, err := d.db.Exec(`DELETE FROM saved_scenes WHERE id = ?`, id)
	return err
}

func migrateV9(db *sql.DB) error {
	_, err := db.Exec(`
		CREATE TABLE IF NOT EXISTS export_presets (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			name TEXT NOT NULL,
			format TEXT NOT NULL DEFAULT 'png',
			width INTEGER NOT NULL DEFAULT 0,
			height INTEGER NOT NULL DEFAULT 0,
			lock_ratio INTEGER NOT NULL DEFAULT 1,
			quality INTEGER NOT NULL DEFAULT 90,
			interpolation TEXT NOT NULL DEFAULT 'lanczos',
			created_at TEXT NOT NULL DEFAULT (datetime('now')),
			updated_at TEXT NOT NULL DEFAULT (datetime('now'))
		)
	`)
	if err != nil {
		return err
	}

	var count int
	if err := db.QueryRow(`SELECT COUNT(*) FROM export_presets`).Scan(&count); err != nil {
		return err
	}
	if count > 0 {
		return nil
	}

	defaults := []ExportPreset{
		{Name: "Quality Photo", Format: "png", Width: 0, Height: 0, LockRatio: true, Quality: 0, Interpolation: "lanczos"},
		{Name: "Web Optimized", Format: "jpeg", Width: 1920, Height: 0, LockRatio: true, Quality: 85, Interpolation: "lanczos"},
		{Name: "Social Media", Format: "jpeg", Width: 1080, Height: 1080, LockRatio: false, Quality: 90, Interpolation: "lanczos"},
		{Name: "Thumbnail", Format: "jpeg", Width: 256, Height: 256, LockRatio: false, Quality: 75, Interpolation: "lanczos"},
	}
	for _, d := range defaults {
		_, err := db.Exec(
			`INSERT INTO export_presets (name, format, width, height, lock_ratio, quality, interpolation) VALUES (?, ?, ?, ?, ?, ?, ?)`,
			d.Name, d.Format, d.Width, d.Height, d.LockRatio, d.Quality, d.Interpolation,
		)
		if err != nil {
			return err
		}
	}
	return nil
}

func migrateV10(db *sql.DB) error {
	_, err := db.Exec(`
		CREATE TABLE IF NOT EXISTS sessions (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			name TEXT NOT NULL,
			created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
			updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
		);
		CREATE TABLE IF NOT EXISTS session_items (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			session_id INTEGER NOT NULL REFERENCES sessions(id) ON DELETE CASCADE,
			file_name TEXT NOT NULL DEFAULT '',
			thumb_name TEXT NOT NULL DEFAULT '',
			source TEXT NOT NULL DEFAULT '',
			prompt TEXT NOT NULL DEFAULT '',
			negative_prompt TEXT NOT NULL DEFAULT '',
			sampler TEXT NOT NULL DEFAULT '',
			steps INTEGER NOT NULL DEFAULT 0,
			cfg_scale REAL NOT NULL DEFAULT 0,
			seed INTEGER,
			denoising REAL NOT NULL DEFAULT 0,
			width INTEGER NOT NULL DEFAULT 0,
			height INTEGER NOT NULL DEFAULT 0,
			is_preview INTEGER NOT NULL DEFAULT 0,
			preset_id INTEGER,
			is_active INTEGER NOT NULL DEFAULT 0,
			created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
		);
		CREATE TABLE IF NOT EXISTS session_active (
			id INTEGER PRIMARY KEY CHECK (id = 1),
			session_id INTEGER REFERENCES sessions(id)
		)
	`)
	if err != nil {
		return err
	}

	var count int
	if err := db.QueryRow(`SELECT COUNT(*) FROM sessions`).Scan(&count); err != nil {
		return err
	}
	if count > 0 {
		return nil
	}

	result, err := db.Exec(`INSERT INTO sessions (name) VALUES ('Default')`)
	if err != nil {
		return err
	}
	sessionID, _ := result.LastInsertId()
	_, err = db.Exec(`INSERT INTO session_active (id, session_id) VALUES (1, ?)`, sessionID)
	return err
}

type bundledPresetsFile struct {
	Version  int              `json:"version"`
	Presets  []bundledPreset  `json:"presets"`
}

type bundledPreset struct {
	Name           string  `json:"name"`
	PresetType     string  `json:"preset_type"`
	TypeName       string  `json:"type_name"`
	Tags           string  `json:"tags"`
	Loras          string  `json:"loras"`
	Prompt         string  `json:"prompt"`
	NegativePrompt string  `json:"negative_prompt"`
	Sampler        string  `json:"sampler"`
	ScheduleType   string  `json:"schedule_type"`
	Steps          int     `json:"steps"`
	CfgScale       float64 `json:"cfg_scale"`
	ModelName      string  `json:"model_name"`
	ClipSkip       *int    `json:"clip_skip"`
	VAE            string  `json:"vae"`
}

func migrateV11(db *sql.DB) error {
	return addColumnIfNotExists(db, "presets", "is_bundled", "INTEGER NOT NULL DEFAULT 0")
}


func migrateV12(db *sql.DB) error {
	_, err := db.Exec(`
		CREATE TABLE IF NOT EXISTS resolutions (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			name TEXT NOT NULL,
			width INTEGER NOT NULL,
			height INTEGER NOT NULL,
			is_builtin INTEGER NOT NULL DEFAULT 0,
			created_at TEXT NOT NULL DEFAULT (datetime('now'))
		)
	`)
	if err != nil {
		return err
	}

	var count int
	if err := db.QueryRow(`SELECT COUNT(*) FROM resolutions`).Scan(&count); err != nil {
		return err
	}
	if count > 0 {
		return nil
	}

	builtins := []struct {
		name   string
		width  int
		height int
	}{
		{"Square 512x512", 512, 512},
		{"Square 768x768", 768, 768},
		{"Square 1024x1024", 1024, 1024},
		{"Portrait 512x768", 512, 768},
		{"Portrait 768x1024", 768, 1024},
		{"Portrait 768x1152", 768, 1152},
		{"Portrait 1024x1536", 1024, 1536},
		{"Landscape 768x512", 768, 512},
		{"Landscape 1024x768", 1024, 768},
		{"Landscape 1152x768", 1152, 768},
		{"Landscape 1536x1024", 1536, 1024},
		{"Widescreen 912x512", 912, 512},
		{"Widescreen 1344x768", 1344, 768},
		{"Widescreen 1024x576", 1024, 576},
	}
	for _, r := range builtins {
		_, err := db.Exec(
			`INSERT INTO resolutions (name, width, height, is_builtin) VALUES (?, ?, ?, 1)`,
			r.name, r.width, r.height,
		)
		if err != nil {
			return err
		}
	}
	return nil
}

func migrateV13(db *sql.DB) error {
	_, err := db.Exec(`
		CREATE TABLE IF NOT EXISTS hires_profiles (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			name TEXT NOT NULL,
			upscale REAL NOT NULL DEFAULT 2.0,
			denoising_strength REAL NOT NULL DEFAULT 0.45,
			upscaler TEXT NOT NULL DEFAULT '',
			is_builtin INTEGER NOT NULL DEFAULT 0,
			created_at TEXT NOT NULL DEFAULT (datetime('now'))
		)
	`)
	if err != nil {
		return err
	}

	var count int
	if err := db.QueryRow(`SELECT COUNT(*) FROM hires_profiles`).Scan(&count); err != nil {
		return err
	}
	if count > 0 {
		return nil
	}

	builtins := []struct {
		name              string
		upscale           float64
		denoisingStrength float64
		upscaler          string
	}{
		{"Light", 1.5, 0.3, "R-ESRGAN 4x+"},
		{"Standard", 2.0, 0.45, "R-ESRGAN 4x+"},
		{"Heavy", 2.5, 0.55, "R-ESRGAN 4x+"},
		{"Max", 3.0, 0.6, "R-ESRGAN 4x+"},
	}
	for _, h := range builtins {
		_, err := db.Exec(
			`INSERT INTO hires_profiles (name, upscale, denoising_strength, upscaler, is_builtin) VALUES (?, ?, ?, ?, 1)`,
			h.name, h.upscale, h.denoisingStrength, h.upscaler,
		)
		if err != nil {
			return err
		}
	}
	return nil
}

func migrateV14(db *sql.DB) error {
	newCols := []struct {
		name string
		typ  string
	}{
		{"default_resolution_id", "INTEGER REFERENCES resolutions(id)"},
		{"default_hires_profile_id", "INTEGER REFERENCES hires_profiles(id)"},
	}
	for _, col := range newCols {
		if err := addColumnIfNotExists(db, "presets", col.name, col.typ); err != nil {
			return err
		}
	}
	return nil
}

func migrateV15(db *sql.DB) error {
	return addColumnIfNotExists(db, "compound_preset_steps", "resolution_id", "INTEGER REFERENCES resolutions(id)")
}

func migrateV16(db *sql.DB) error {
	cols := []string{"default_resolution_id", "default_hires_profile_id"}
	for _, col := range cols {
		_, err := db.Exec("ALTER TABLE presets DROP COLUMN " + col)
		if err != nil && !strings.Contains(err.Error(), "no such column") {
			return err
		}
	}
	return nil
}

func migrateV17(db *sql.DB) error {
	cols := []string{"width", "height"}
	for _, col := range cols {
		_, err := db.Exec("ALTER TABLE presets DROP COLUMN " + col)
		if err != nil && !strings.Contains(err.Error(), "no such column") {
			return err
		}
	}
	return nil
}

func migrateV18(db *sql.DB) error {
	_, err := db.Exec("UPDATE hires_profiles SET upscaler = 'Latent' WHERE upscaler = '' OR upscaler IS NULL")
	return err
}

func migrateV19(db *sql.DB) error {
	cols := []string{"width", "height"}
	for _, col := range cols {
		_, err := db.Exec("ALTER TABLE compound_preset_steps DROP COLUMN " + col)
		if err != nil && !strings.Contains(err.Error(), "no such column") {
			return err
		}
	}
	return nil
}

func migrateV20(db *sql.DB) error {
	_, err := db.Exec(`DELETE FROM presets WHERE is_bundled = 1`)
	if err != nil {
		return err
	}
	_, err = db.Exec(`DELETE FROM preset_types WHERE id NOT IN (SELECT DISTINCT type_id FROM presets WHERE type_id IS NOT NULL)`)
	if err != nil {
		return err
	}
	return nil
}

func migrateV21(db *sql.DB) error {
	for _, col := range []string{"hires_fix", "hires_upscale", "hires_denoising_strength", "hires_upscaler"} {
		_, err := db.Exec("ALTER TABLE presets DROP COLUMN " + col)
		if err != nil && !strings.Contains(err.Error(), "no such column") {
			return err
		}
	}
	return nil
}

func (d *DB) SeedBundled(fsys fs.FS) error {
	var bundledCount int
	if err := d.db.QueryRow(`SELECT COUNT(*) FROM presets WHERE is_bundled = 1`).Scan(&bundledCount); err != nil {
		return err
	}
	if bundledCount > 0 {
		return nil
	}

	entries, err := fs.ReadDir(fsys, "data/presets")
	if err != nil {
		return fmt.Errorf("read bundled presets: %w", err)
	}

	var files []bundledPresetsFile
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".json") {
			continue
		}
		raw, err := fs.ReadFile(fsys, "data/presets/"+entry.Name())
		if err != nil {
			continue
		}
		var file bundledPresetsFile
		if err := json.Unmarshal(raw, &file); err != nil || file.Version != 2 {
			continue
		}
		files = append(files, file)
	}

	typeCache := make(map[string]int64)
	existing, _ := d.ListPresetTypes()
	for _, t := range existing {
		typeCache[t.Name] = t.ID
	}
	for _, f := range files {
		for _, p := range f.Presets {
			typeName := p.TypeName
			if typeName == "" {
				typeName = p.PresetType
			}
			if typeName == "" {
				continue
			}
			if _, ok := typeCache[typeName]; ok {
				continue
			}
			pt := &PresetType{Name: typeName}
			if err := d.CreatePresetType(pt); err != nil {
				return err
			}
			typeCache[typeName] = pt.ID
		}
	}

	stmt, err := d.db.Prepare(`INSERT INTO presets (name, preset_type, tags, loras, prompt, negative_prompt, sampler, schedule_type, steps, cfg_scale, model_name, clip_skip, vae, type_id, is_bundled) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, 1)`)
	if err != nil {
		return err
	}
	defer stmt.Close()

	for _, f := range files {
		for _, p := range f.Presets {
			if p.Name == "" {
				continue
			}
			presetType := p.TypeName
			if presetType == "" {
				presetType = p.PresetType
			}
			var typeID *int64
			if id, ok := typeCache[presetType]; ok {
				typeID = &id
			}
			if _, err := stmt.Exec(p.Name, presetType, p.Tags, p.Loras, p.Prompt, p.NegativePrompt, p.Sampler, p.ScheduleType, p.Steps, p.CfgScale, p.ModelName, p.ClipSkip, p.VAE, typeID); err != nil {
				return err
			}
		}
	}

	return nil
}

func migrateV22(db *sql.DB) error {
	_, err := db.Exec("UPDATE hires_profiles SET upscaler = 'R-ESRGAN 4x+' WHERE upscaler = 'Latent'")
	return err
}

func migrateV23(db *sql.DB) error {
	_, err := db.Exec(`
		CREATE TABLE IF NOT EXISTS job_queue (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			type TEXT NOT NULL,
			status TEXT NOT NULL DEFAULT 'pending',
			params TEXT NOT NULL DEFAULT '{}',
			progress REAL NOT NULL DEFAULT 0,
			progress_detail TEXT NOT NULL DEFAULT '{}',
			result TEXT NOT NULL DEFAULT '',
			error TEXT NOT NULL DEFAULT '',
			source TEXT NOT NULL DEFAULT '',
			created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
			started_at DATETIME,
			completed_at DATETIME
		);
		CREATE INDEX IF NOT EXISTS idx_job_queue_status ON job_queue(status)
	`)
	return err
}

func migrateV24(db *sql.DB) error {
	cols := []struct {
		name string
		typ  string
	}{
		{"retry_count", "INTEGER NOT NULL DEFAULT 0"},
		{"max_retries", "INTEGER NOT NULL DEFAULT 3"},
		{"next_retry_at", "DATETIME"},
	}
	for _, col := range cols {
		if err := addColumnIfNotExists(db, "job_queue", col.name, col.typ); err != nil {
			return err
		}
	}
	_, err := db.Exec(`CREATE INDEX IF NOT EXISTS idx_job_queue_next_retry ON job_queue(next_retry_at) WHERE status = 'pending' AND next_retry_at IS NOT NULL`)
	return err
}

func migrateV25(db *sql.DB) error {
	_, err := db.Exec(`UPDATE hires_profiles SET upscale = 3.0, denoising_strength = 0.6 WHERE name = 'Max' AND is_builtin = 1`)
	return err
}
