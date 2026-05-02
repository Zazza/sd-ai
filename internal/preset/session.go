package preset

import "database/sql"

func (d *DB) CreateSession(name string) (*SessionInfo, error) {
	result, err := d.db.Exec(`INSERT INTO sessions (name) VALUES (?)`, name)
	if err != nil {
		return nil, err
	}
	id, _ := result.LastInsertId()
	return &SessionInfo{ID: id, Name: name}, nil
}

func (d *DB) ListSessions() ([]SessionInfo, error) {
	rows, err := d.db.Query(`
		SELECT s.id, s.name, s.created_at, s.updated_at,
			(SELECT COUNT(*) FROM session_items si WHERE si.session_id = s.id) as item_count
		FROM sessions s ORDER BY s.created_at ASC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var items []SessionInfo
	for rows.Next() {
		var s SessionInfo
		if err := rows.Scan(&s.ID, &s.Name, &s.CreatedAt, &s.UpdatedAt, &s.ItemCount); err != nil {
			return nil, err
		}
		items = append(items, s)
	}
	return items, rows.Err()
}

func (d *DB) RenameSession(id int64, name string) error {
	_, err := d.db.Exec(`UPDATE sessions SET name=?, updated_at=CURRENT_TIMESTAMP WHERE id=?`, name, id)
	return err
}

func (d *DB) DeleteSession(id int64) error {
	tx, err := d.db.Begin()
	if err != nil {
		return err
	}
	_, err = tx.Exec(`DELETE FROM session_items WHERE session_id = ?`, id)
	if err != nil {
		tx.Rollback()
		return err
	}
	_, err = tx.Exec(`DELETE FROM sessions WHERE id = ?`, id)
	if err != nil {
		tx.Rollback()
		return err
	}
	return tx.Commit()
}

func (d *DB) GetActiveSessionID() (int64, error) {
	var id int64
	err := d.db.QueryRow(`SELECT session_id FROM session_active WHERE id = 1`).Scan(&id)
	if err == sql.ErrNoRows {
		return 0, nil
	}
	return id, err
}

func (d *DB) SetActiveSession(id int64) error {
	_, err := d.db.Exec(`INSERT INTO session_active (id, session_id) VALUES (1, ?) ON CONFLICT(id) DO UPDATE SET session_id = ?`, id, id)
	return err
}

func (d *DB) AddSessionItem(item *SessionItem) (int64, error) {
	_, err := d.db.Exec(`UPDATE session_items SET is_active = 0 WHERE session_id = ?`, item.SessionID)
	if err != nil {
		return 0, err
	}
	result, err := d.db.Exec(`
		INSERT INTO session_items (session_id, file_name, thumb_name, source, prompt, negative_prompt, sampler, steps, cfg_scale, seed, denoising, width, height, is_preview, preset_id, is_active)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, 1)`,
		item.SessionID, item.FileName, item.ThumbName, item.Source,
		item.Prompt, item.NegativePrompt, item.Sampler, item.Steps, item.CfgScale,
		item.Seed, item.Denoising, item.Width, item.Height, item.IsPreview, item.PresetID)
	if err != nil {
		return 0, err
	}
	return result.LastInsertId()
}

func (d *DB) GetSessionItems(sessionID int64) ([]SessionItem, error) {
	rows, err := d.db.Query(`
		SELECT id, session_id, file_name, thumb_name, source, prompt, negative_prompt, sampler, steps, cfg_scale, seed, denoising, width, height, is_preview, preset_id, is_active, created_at
		FROM session_items WHERE session_id = ? ORDER BY created_at ASC`, sessionID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanSessionItems(rows)
}

func (d *DB) GetSessionItem(id int64) (*SessionItem, error) {
	var item SessionItem
	err := d.db.QueryRow(`
		SELECT id, session_id, file_name, thumb_name, source, prompt, negative_prompt, sampler, steps, cfg_scale, seed, denoising, width, height, is_preview, preset_id, is_active, created_at
		FROM session_items WHERE id = ?`, id).
		Scan(&item.ID, &item.SessionID, &item.FileName, &item.ThumbName, &item.Source,
			&item.Prompt, &item.NegativePrompt, &item.Sampler, &item.Steps, &item.CfgScale,
			&item.Seed, &item.Denoising, &item.Width, &item.Height, &item.IsPreview,
			&item.PresetID, &item.IsActive, &item.CreatedAt)
	if err != nil {
		return nil, err
	}
	return &item, nil
}

func (d *DB) GetActiveItem(sessionID int64) (*SessionItem, error) {
	var item SessionItem
	err := d.db.QueryRow(`
		SELECT id, session_id, file_name, thumb_name, source, prompt, negative_prompt, sampler, steps, cfg_scale, seed, denoising, width, height, is_preview, preset_id, is_active, created_at
		FROM session_items WHERE session_id = ? AND is_active = 1`, sessionID).
		Scan(&item.ID, &item.SessionID, &item.FileName, &item.ThumbName, &item.Source,
			&item.Prompt, &item.NegativePrompt, &item.Sampler, &item.Steps, &item.CfgScale,
			&item.Seed, &item.Denoising, &item.Width, &item.Height, &item.IsPreview,
			&item.PresetID, &item.IsActive, &item.CreatedAt)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &item, nil
}

func (d *DB) SetActiveItem(id int64, sessionID int64) error {
	_, err := d.db.Exec(`UPDATE session_items SET is_active = CASE WHEN id = ? THEN 1 ELSE 0 END WHERE session_id = ?`, id, sessionID)
	return err
}

func (d *DB) DeleteSessionItem(id int64) error {
	_, err := d.db.Exec(`DELETE FROM session_items WHERE id = ?`, id)
	return err
}

func (d *DB) ClearSessionItems(sessionID int64) error {
	_, err := d.db.Exec(`DELETE FROM session_items WHERE session_id = ?`, sessionID)
	return err
}

func (d *DB) HasAnyItems() (bool, error) {
	var count int
	err := d.db.QueryRow(`SELECT COUNT(*) FROM session_items`).Scan(&count)
	return count > 0, err
}

func (d *DB) DeleteAllSessions() error {
	tx, err := d.db.Begin()
	if err != nil {
		return err
	}
	_, err = tx.Exec(`DELETE FROM session_items`)
	if err != nil {
		tx.Rollback()
		return err
	}
	_, err = tx.Exec(`DELETE FROM sessions`)
	if err != nil {
		tx.Rollback()
		return err
	}
	result, err := tx.Exec(`INSERT INTO sessions (name) VALUES ('Default')`)
	if err != nil {
		tx.Rollback()
		return err
	}
	id, _ := result.LastInsertId()
	_, err = tx.Exec(`UPDATE session_active SET session_id = ? WHERE id = 1`, id)
	if err != nil {
		tx.Rollback()
		return err
	}
	return tx.Commit()
}

func (d *DB) UpdateSessionItemPaths(id int64, fileName string, thumbName string) error {
	_, err := d.db.Exec(`UPDATE session_items SET file_name = ?, thumb_name = ? WHERE id = ?`, fileName, thumbName, id)
	return err
}

func scanSessionItems(rows *sql.Rows) ([]SessionItem, error) {
	var items []SessionItem
	for rows.Next() {
		var item SessionItem
		if err := rows.Scan(&item.ID, &item.SessionID, &item.FileName, &item.ThumbName,
			&item.Source, &item.Prompt, &item.NegativePrompt, &item.Sampler,
			&item.Steps, &item.CfgScale, &item.Seed, &item.Denoising,
			&item.Width, &item.Height, &item.IsPreview, &item.PresetID,
			&item.IsActive, &item.CreatedAt); err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	return items, rows.Err()
}
