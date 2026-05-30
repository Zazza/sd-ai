package kids

import (
	"crypto/rand"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/hex"
	"fmt"
	"strconv"
	"time"

	"go-sd/internal/config"
)

const (
	maxPinAttempts = 5
	pinLockoutMins = 5
)

type SettingsStore interface {
	GetSetting(key string) (string, error)
	SetSetting(key, value string) error
}

type CategoryInfo struct {
	Name     string `json:"name"`
	Label    string `json:"label"`
	AlwaysOn bool   `json:"alwaysOn"`
	Enabled  bool   `json:"enabled"`
}

type Manager struct {
	store SettingsStore
}

func NewManager(store SettingsStore) *Manager {
	return &Manager{store: store}
}

func (m *Manager) IsActive() bool {
	v, _ := m.store.GetSetting("kids_mode")
	return v == "true"
}

func (m *Manager) SetKidsMode(enabled bool, pin string) error {
	if enabled {
		if pin != "" {
			if err := m.storePin(pin); err != nil {
				return err
			}
		}
		m.store.SetSetting("kids_pin_attempts", "0")
		m.store.SetSetting("kids_pin_lockout", "")
		return m.store.SetSetting("kids_mode", "true")
	}

	storedHash, _ := m.store.GetSetting("kids_pin_hash")
	if storedHash != "" {
		if err := m.checkPinLockout(); err != nil {
			return err
		}
		if pin == "" {
			return fmt.Errorf("PIN required")
		}
		ok, err := m.verifyPin(pin)
		if err != nil {
			return err
		}
		if !ok {
			m.recordFailedPinAttempt()
			return fmt.Errorf("incorrect PIN")
		}
		m.store.SetSetting("kids_pin_attempts", "0")
		m.store.SetSetting("kids_pin_lockout", "")
	}
	return m.store.SetSetting("kids_mode", "false")
}

func (m *Manager) GetCategories() ([]CategoryInfo, error) {
	var result []CategoryInfo
	for _, cat := range Categories {
		v, _ := m.store.GetSetting("kids_cat_" + cat.Name)
		enabled := cat.AlwaysOn || v != "false"
		result = append(result, CategoryInfo{
			Name:     cat.Name,
			Label:    cat.Label,
			AlwaysOn: cat.AlwaysOn,
			Enabled:  enabled,
		})
	}
	return result, nil
}

func (m *Manager) SetCategory(name string, enabled bool, pin string) error {
	if !m.IsActive() {
		return fmt.Errorf("Kids Mode is not active")
	}
	var found *Category
	for i := range Categories {
		if Categories[i].Name == name {
			found = &Categories[i]
			break
		}
	}
	if found == nil {
		return fmt.Errorf("unknown category: %s", name)
	}
	if found.AlwaysOn {
		return fmt.Errorf("category %s cannot be disabled", name)
	}
	storedHash, _ := m.store.GetSetting("kids_pin_hash")
	if storedHash != "" {
		if err := m.checkPinLockout(); err != nil {
			return err
		}
		if pin == "" {
			return fmt.Errorf("PIN required")
		}
		ok, err := m.verifyPin(pin)
		if err != nil {
			return err
		}
		if !ok {
			m.recordFailedPinAttempt()
			return fmt.Errorf("incorrect PIN")
		}
		m.store.SetSetting("kids_pin_attempts", "0")
		m.store.SetSetting("kids_pin_lockout", "")
	}
	val := "true"
	if !enabled {
		val = "false"
	}
	return m.store.SetSetting("kids_cat_"+name, val)
}

func (m *Manager) ApplySystemPrompt(systemPrompt string) string {
	if !m.IsActive() {
		return systemPrompt
	}
	return systemPrompt + config.KidsModePrompt
}

func (m *Manager) ApplyNegative(negativePrompt string) string {
	if !m.IsActive() {
		return negativePrompt
	}
	disabled := m.getDisabledCategories()
	kidsNeg := NegativePrompt(disabled)
	if negativePrompt != "" {
		return negativePrompt + ", " + kidsNeg
	}
	return kidsNeg
}

func (m *Manager) FilterInput(text string) (string, error) {
	if !m.IsActive() {
		return text, nil
	}
	return FilterInput(text, m.getDisabledCategories())
}

func (m *Manager) FilterOutput(text string) string {
	if !m.IsActive() {
		return text
	}
	return FilterOutput(text, m.getDisabledCategories())
}

func (m *Manager) getDisabledCategories() map[string]bool {
	disabled := make(map[string]bool)
	for _, cat := range Categories {
		if cat.AlwaysOn {
			continue
		}
		v, _ := m.store.GetSetting("kids_cat_" + cat.Name)
		if v == "false" {
			disabled[cat.Name] = true
		}
	}
	return disabled
}

func (m *Manager) verifyPin(pin string) (bool, error) {
	storedHash, _ := m.store.GetSetting("kids_pin_hash")
	salt, _ := m.store.GetSetting("kids_pin_salt")
	if storedHash == "" || salt == "" {
		return false, fmt.Errorf("PIN not set")
	}
	computed := hashPin(pin, salt)
	return subtle.ConstantTimeCompare([]byte(computed), []byte(storedHash)) == 1, nil
}

func (m *Manager) storePin(pin string) error {
	if len(pin) != 4 {
		return fmt.Errorf("PIN must be 4 digits")
	}
	for _, c := range pin {
		if c < '0' || c > '9' {
			return fmt.Errorf("PIN must be 4 digits")
		}
	}
	saltBytes := make([]byte, 16)
	if _, err := rand.Read(saltBytes); err != nil {
		return fmt.Errorf("failed to generate salt: %w", err)
	}
	salt := hex.EncodeToString(saltBytes)
	hash := hashPin(pin, salt)
	if err := m.store.SetSetting("kids_pin_salt", salt); err != nil {
		return err
	}
	return m.store.SetSetting("kids_pin_hash", hash)
}

func (m *Manager) checkPinLockout() error {
	lockoutStr, _ := m.store.GetSetting("kids_pin_lockout")
	if lockoutStr == "" {
		return nil
	}
	lockoutTime, err := time.Parse(time.RFC3339, lockoutStr)
	if err != nil {
		m.store.SetSetting("kids_pin_lockout", "")
		m.store.SetSetting("kids_pin_attempts", "0")
		return nil
	}
	if time.Now().Before(lockoutTime) {
		remaining := time.Until(lockoutTime).Truncate(time.Second)
		return fmt.Errorf("PIN locked. Try again in %s", remaining)
	}
	m.store.SetSetting("kids_pin_lockout", "")
	m.store.SetSetting("kids_pin_attempts", "0")
	return nil
}

func (m *Manager) recordFailedPinAttempt() {
	attemptsStr, _ := m.store.GetSetting("kids_pin_attempts")
	attempts := 0
	if n, err := strconv.Atoi(attemptsStr); err == nil {
		attempts = n
	}
	attempts++
	m.store.SetSetting("kids_pin_attempts", strconv.Itoa(attempts))
	if attempts >= maxPinAttempts {
		lockoutUntil := time.Now().Add(pinLockoutMins * time.Minute).Format(time.RFC3339)
		m.store.SetSetting("kids_pin_lockout", lockoutUntil)
	}
}

func hashPin(pin, salt string) string {
	h := sha256.Sum256([]byte(salt + pin))
	return hex.EncodeToString(h[:])
}
