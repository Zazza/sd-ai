package preset

import (
	"database/sql"
	"errors"
)

const resolutionColumns = `id, name, width, height, is_builtin, created_at`

func scanResolution(scanner interface{ Scan(...any) error }, r *Resolution) error {
	var isBuiltin int
	err := scanner.Scan(&r.ID, &r.Name, &r.Width, &r.Height, &isBuiltin, &r.CreatedAt)
	if err != nil {
		return err
	}
	r.IsBuiltin = isBuiltin != 0
	return nil
}

func (d *DB) ListResolutions() ([]Resolution, error) {
	rows, err := d.db.Query(`SELECT `+resolutionColumns+` FROM resolutions ORDER BY created_at ASC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var items []Resolution
	for rows.Next() {
		var r Resolution
		if err := scanResolution(rows, &r); err != nil {
			return nil, err
		}
		items = append(items, r)
	}
	return items, rows.Err()
}

func (d *DB) GetResolution(id int64) (*Resolution, error) {
	var r Resolution
	err := scanResolution(d.db.QueryRow(`SELECT `+resolutionColumns+` FROM resolutions WHERE id = ?`, id), &r)
	if err != nil {
		return nil, err
	}
	return &r, nil
}

func (d *DB) CreateResolution(r *Resolution) error {
	result, err := d.db.Exec(
		`INSERT INTO resolutions (name, width, height, is_builtin) VALUES (?, ?, ?, 0)`,
		r.Name, r.Width, r.Height,
	)
	if err != nil {
		return err
	}
	r.ID, _ = result.LastInsertId()
	r.IsBuiltin = false
	return nil
}

func (d *DB) UpdateResolution(r *Resolution) error {
	_, err := d.db.Exec(
		`UPDATE resolutions SET name=?, width=?, height=? WHERE id=? AND is_builtin = 0`,
		r.Name, r.Width, r.Height, r.ID,
	)
	if err != nil {
		return err
	}
	return nil
}

func (d *DB) DeleteResolution(id int64) error {
	var isBuiltin int
	err := d.db.QueryRow(`SELECT is_builtin FROM resolutions WHERE id = ?`, id).Scan(&isBuiltin)
	if err != nil {
		if err == sql.ErrNoRows {
			return errors.New("resolution not found")
		}
		return err
	}
	if isBuiltin != 0 {
		return errors.New("cannot delete builtin resolution")
	}
	_, err = d.db.Exec(`DELETE FROM resolutions WHERE id = ?`, id)
	return err
}
