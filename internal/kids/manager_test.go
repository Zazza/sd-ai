package kids

import (
	"testing"

	"go-sd/internal/preset"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func newTestManager(t *testing.T) (*Manager, *preset.DB) {
	t.Helper()
	db, err := preset.Open(":memory:")
	require.NoError(t, err)
	t.Cleanup(func() { db.Close() })
	return NewManager(db), db
}

func TestStorePin_Valid(t *testing.T) {
	m, db := newTestManager(t)
	err := m.storePin("1234")
	assert.NoError(t, err)
	hash, _ := db.GetSetting("kids_pin_hash")
	assert.NotEmpty(t, hash)
	salt, _ := db.GetSetting("kids_pin_salt")
	assert.NotEmpty(t, salt)
}

func TestStorePin_InvalidLength(t *testing.T) {
	m, _ := newTestManager(t)
	err := m.storePin("12")
	assert.Error(t, err)
}

func TestVerifyPin_Correct(t *testing.T) {
	m, _ := newTestManager(t)
	require.NoError(t, m.storePin("1234"))
	ok, err := m.verifyPin("1234")
	assert.NoError(t, err)
	assert.True(t, ok)
}

func TestVerifyPin_Incorrect(t *testing.T) {
	m, _ := newTestManager(t)
	require.NoError(t, m.storePin("1234"))
	ok, err := m.verifyPin("0000")
	assert.NoError(t, err)
	assert.False(t, ok)
}

func TestVerifyPin_NotSet(t *testing.T) {
	m, _ := newTestManager(t)
	_, err := m.verifyPin("1234")
	assert.Error(t, err)
}

func TestHashPin_UsesSalt(t *testing.T) {
	h1 := hashPin("1234", "salt1")
	h2 := hashPin("1234", "salt2")
	assert.NotEqual(t, h1, h2, "same PIN with different salts must produce different hashes")
}

func TestSetKidsMode_EnableWithPin(t *testing.T) {
	m, db := newTestManager(t)
	err := m.SetKidsMode(true, "1234")
	assert.NoError(t, err)
	v, _ := db.GetSetting("kids_mode")
	assert.Equal(t, "true", v)
}

func TestSetKidsMode_DisableWithCorrectPin(t *testing.T) {
	m, db := newTestManager(t)
	require.NoError(t, m.SetKidsMode(true, "1234"))
	err := m.SetKidsMode(false, "1234")
	assert.NoError(t, err)
	v, _ := db.GetSetting("kids_mode")
	assert.Equal(t, "false", v)
}

func TestSetKidsMode_DisableWithWrongPin(t *testing.T) {
	m, _ := newTestManager(t)
	require.NoError(t, m.SetKidsMode(true, "1234"))
	err := m.SetKidsMode(false, "0000")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "incorrect PIN")
}

func TestSetKidsMode_DisableWithoutPin(t *testing.T) {
	m, _ := newTestManager(t)
	require.NoError(t, m.SetKidsMode(true, "1234"))
	err := m.SetKidsMode(false, "")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "PIN required")
}

func TestSetKidsMode_EnableBadPin(t *testing.T) {
	m, _ := newTestManager(t)
	err := m.SetKidsMode(true, "12")
	assert.Error(t, err)
}

func TestSetKidsCategory_RejectsWrongPin(t *testing.T) {
	m, _ := newTestManager(t)
	require.NoError(t, m.SetKidsMode(true, "1234"))
	err := m.SetCategory("violence", false, "0000")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "incorrect PIN")
}
