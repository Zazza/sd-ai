package preset

import (
	"database/sql"
	"errors"
)

const hiresProfileColumns = `id, name, upscale, denoising_strength, upscaler, is_builtin, created_at`

func scanHiresProfile(scanner interface{ Scan(...any) error }, h *HiresProfile) error {
	var isBuiltin int
	err := scanner.Scan(&h.ID, &h.Name, &h.Upscale, &h.DenoisingStrength, &h.Upscaler, &isBuiltin, &h.CreatedAt)
	if err != nil {
		return err
	}
	h.IsBuiltin = isBuiltin != 0
	return nil
}

func (d *DB) ListHiresProfiles() ([]HiresProfile, error) {
	rows, err := d.db.Query(`SELECT `+hiresProfileColumns+` FROM hires_profiles ORDER BY created_at ASC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var items []HiresProfile
	for rows.Next() {
		var h HiresProfile
		if err := scanHiresProfile(rows, &h); err != nil {
			return nil, err
		}
		items = append(items, h)
	}
	return items, rows.Err()
}

func (d *DB) GetHiresProfile(id int64) (*HiresProfile, error) {
	var h HiresProfile
	err := scanHiresProfile(d.db.QueryRow(`SELECT `+hiresProfileColumns+` FROM hires_profiles WHERE id = ?`, id), &h)
	if err != nil {
		return nil, err
	}
	return &h, nil
}

func (d *DB) CreateHiresProfile(h *HiresProfile) error {
	result, err := d.db.Exec(
		`INSERT INTO hires_profiles (name, upscale, denoising_strength, upscaler, is_builtin) VALUES (?, ?, ?, ?, 0)`,
		h.Name, h.Upscale, h.DenoisingStrength, h.Upscaler,
	)
	if err != nil {
		return err
	}
	h.ID, _ = result.LastInsertId()
	h.IsBuiltin = false
	return nil
}

func (d *DB) UpdateHiresProfile(h *HiresProfile) error {
	_, err := d.db.Exec(
		`UPDATE hires_profiles SET name=?, upscale=?, denoising_strength=?, upscaler=? WHERE id=? AND is_builtin = 0`,
		h.Name, h.Upscale, h.DenoisingStrength, h.Upscaler, h.ID,
	)
	if err != nil {
		return err
	}
	return nil
}

func (d *DB) DeleteHiresProfile(id int64) error {
	var isBuiltin int
	err := d.db.QueryRow(`SELECT is_builtin FROM hires_profiles WHERE id = ?`, id).Scan(&isBuiltin)
	if err != nil {
		if err == sql.ErrNoRows {
			return errors.New("hires profile not found")
		}
		return err
	}
	if isBuiltin != 0 {
		return errors.New("cannot delete builtin hires profile")
	}
	_, err = d.db.Exec(`DELETE FROM hires_profiles WHERE id = ?`, id)
	return err
}
