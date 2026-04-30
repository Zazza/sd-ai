package preset

const exportPresetColumns = `id, name, format, width, height, lock_ratio, quality, interpolation, created_at, updated_at`

func scanExportPreset(scanner interface{ Scan(...any) error }, ep *ExportPreset) error {
	var lockRatio int
	err := scanner.Scan(&ep.ID, &ep.Name, &ep.Format, &ep.Width, &ep.Height, &lockRatio, &ep.Quality, &ep.Interpolation, &ep.CreatedAt, &ep.UpdatedAt)
	if err != nil {
		return err
	}
	ep.LockRatio = lockRatio != 0
	return nil
}

func (d *DB) ListExportPresets() ([]ExportPreset, error) {
	rows, err := d.db.Query(`SELECT `+exportPresetColumns+` FROM export_presets ORDER BY created_at ASC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var items []ExportPreset
	for rows.Next() {
		var ep ExportPreset
		if err := scanExportPreset(rows, &ep); err != nil {
			return nil, err
		}
		items = append(items, ep)
	}
	return items, rows.Err()
}

func (d *DB) GetExportPreset(id int64) (*ExportPreset, error) {
	var ep ExportPreset
	err := scanExportPreset(d.db.QueryRow(`SELECT `+exportPresetColumns+` FROM export_presets WHERE id = ?`, id), &ep)
	if err != nil {
		return nil, err
	}
	return &ep, nil
}

func (d *DB) CreateExportPreset(ep *ExportPreset) error {
	result, err := d.db.Exec(
		`INSERT INTO export_presets (name, format, width, height, lock_ratio, quality, interpolation) VALUES (?, ?, ?, ?, ?, ?, ?)`,
		ep.Name, ep.Format, ep.Width, ep.Height, ep.LockRatio, ep.Quality, ep.Interpolation,
	)
	if err != nil {
		return err
	}
	ep.ID, _ = result.LastInsertId()
	return nil
}

func (d *DB) UpdateExportPreset(ep *ExportPreset) error {
	_, err := d.db.Exec(
		`UPDATE export_presets SET name=?, format=?, width=?, height=?, lock_ratio=?, quality=?, interpolation=?, updated_at=datetime('now') WHERE id=?`,
		ep.Name, ep.Format, ep.Width, ep.Height, ep.LockRatio, ep.Quality, ep.Interpolation, ep.ID,
	)
	return err
}

func (d *DB) DeleteExportPreset(id int64) error {
	_, err := d.db.Exec(`DELETE FROM export_presets WHERE id = ?`, id)
	return err
}
