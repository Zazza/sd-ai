package session

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"image"
	"image/jpeg"
	"os"
	"path/filepath"
	"sync"
	"testing"

	"go-sd/internal/preset"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type mockEmitter struct {
	mu     sync.Mutex
	events []emittedEvent
}

type emittedEvent struct {
	name string
	data []any
}

func (m *mockEmitter) Emit(event string, data ...any) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.events = append(m.events, emittedEvent{name: event, data: data})
}

func (m *mockEmitter) hasEvent(name string) bool {
	m.mu.Lock()
	defer m.mu.Unlock()
	for _, e := range m.events {
		if e.name == name {
			return true
		}
	}
	return false
}

func (m *mockEmitter) eventCount(name string) int {
	m.mu.Lock()
	defer m.mu.Unlock()
	count := 0
	for _, e := range m.events {
		if e.name == name {
			count++
		}
	}
	return count
}

func testService(t *testing.T) (*Service, *mockEmitter, string) {
	t.Helper()
	db, err := preset.Open(":memory:")
	require.NoError(t, err)
	t.Cleanup(func() { db.Close() })

	dataDir := t.TempDir()
	emitter := &mockEmitter{}
	svc := New(db, dataDir, emitter)
	return svc, emitter, dataDir
}

func makeImageJPEG(t *testing.T, w, h int) string {
	t.Helper()
	img := image.NewRGBA(image.Rect(0, 0, w, h))
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			img.Set(x, y, image.Black)
		}
	}
	var buf bytes.Buffer
	err := jpeg.Encode(&buf, img, &jpeg.Options{Quality: 90})
	require.NoError(t, err)
	return base64.StdEncoding.EncodeToString(buf.Bytes())
}

func TestNew_ServiceCreation(t *testing.T) {
	t.Parallel()
	db, err := preset.Open(":memory:")
	require.NoError(t, err)
	defer db.Close()

	svc := New(db, "/tmp/test", &mockEmitter{})
	assert.NotNil(t, svc)
}

func TestCreateSession_Success(t *testing.T) {
	t.Parallel()
	svc, emitter, _ := testService(t)

	si, err := svc.CreateSession("Test Session")
	require.NoError(t, err)
	assert.NotNil(t, si)
	assert.Equal(t, "Test Session", si.Name)
	assert.True(t, si.ID > 0)
	assert.True(t, emitter.hasEvent("session:created"))
	assert.True(t, emitter.hasEvent("session:switched"))
}

func TestCreateSession_EmptyName(t *testing.T) {
	t.Parallel()
	svc, _, _ := testService(t)

	si, err := svc.CreateSession("")
	require.NoError(t, err)
	assert.NotNil(t, si)
	assert.Equal(t, "", si.Name)
}

func TestListSessions_DefaultSession(t *testing.T) {
	t.Parallel()
	svc, _, _ := testService(t)

	sessions, err := svc.ListSessions()
	require.NoError(t, err)
	assert.Len(t, sessions, 1)
	assert.Equal(t, "Default", sessions[0].Name)
}

func TestListSessions_MultipleSessions(t *testing.T) {
	t.Parallel()
	svc, _, _ := testService(t)

	_, err := svc.CreateSession("Second")
	require.NoError(t, err)

	sessions, err := svc.ListSessions()
	require.NoError(t, err)
	assert.Len(t, sessions, 2)
}

func TestSwitchSession_Success(t *testing.T) {
	t.Parallel()
	svc, emitter, _ := testService(t)

	si, err := svc.CreateSession("Target")
	require.NoError(t, err)

	err = svc.SwitchSession(si.ID)
	require.NoError(t, err)
	assert.True(t, emitter.hasEvent("session:switched"))
}

func TestSwitchSession_InvalidID(t *testing.T) {
	t.Parallel()
	svc, _, _ := testService(t)

	err := svc.SwitchSession(99999)
	assert.NoError(t, err)
}

func TestRenameSession_Success(t *testing.T) {
	t.Parallel()
	svc, _, _ := testService(t)

	si, err := svc.CreateSession("Original")
	require.NoError(t, err)

	err = svc.RenameSession(si.ID, "Renamed")
	require.NoError(t, err)

	sessions, err := svc.ListSessions()
	require.NoError(t, err)
	found := false
	for _, s := range sessions {
		if s.ID == si.ID {
			assert.Equal(t, "Renamed", s.Name)
			found = true
		}
	}
	assert.True(t, found)
}

func TestRenameSession_NonExistent(t *testing.T) {
	t.Parallel()
	svc, _, _ := testService(t)

	err := svc.RenameSession(99999, "Ghost")
	assert.NoError(t, err)
}

func TestDeleteSession_Success(t *testing.T) {
	t.Parallel()
	svc, emitter, dataDir := testService(t)

	si, err := svc.CreateSession("ToDelete")
	require.NoError(t, err)

	sessionDir := filepath.Join(dataDir, "sessions", fmt.Sprintf("%d", si.ID))
	os.MkdirAll(sessionDir, 0o755)
	os.WriteFile(filepath.Join(sessionDir, "test.jpg"), []byte("data"), 0o644)

	err = svc.DeleteSession(si.ID)
	require.NoError(t, err)
	assert.True(t, emitter.hasEvent("session:deleted"))

	sessions, err := svc.ListSessions()
	require.NoError(t, err)
	assert.Len(t, sessions, 1)
	assert.NotEqual(t, si.ID, sessions[0].ID)
}

func TestDeleteSession_LastSession(t *testing.T) {
	t.Parallel()
	svc, _, _ := testService(t)

	sessions, err := svc.ListSessions()
	require.NoError(t, err)
	assert.Len(t, sessions, 1)

	err = svc.DeleteSession(sessions[0].ID)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "cannot delete the last session")
}

func TestDeleteSession_ActiveSessionSwitch(t *testing.T) {
	t.Parallel()
	svc, _, _ := testService(t)

	_, err := svc.CreateSession("WillBeActive")
	require.NoError(t, err)

	sessions, _ := svc.ListSessions()
	defaultID := sessions[0].ID

	err = svc.SwitchSession(defaultID)
	require.NoError(t, err)

	err = svc.DeleteSession(defaultID)
	require.NoError(t, err)

	items, err := svc.GetSessionItems()
	require.NoError(t, err)
	assert.Empty(t, items)
}

func TestGetSessionItems_EmptySession(t *testing.T) {
	t.Parallel()
	svc, _, _ := testService(t)

	items, err := svc.GetSessionItems()
	require.NoError(t, err)
	assert.Empty(t, items)
}

func TestGetActiveSessionItem_NoActive(t *testing.T) {
	t.Parallel()
	svc, _, _ := testService(t)

	item, err := svc.GetActiveSessionItem()
	require.NoError(t, err)
	assert.Nil(t, item)
}

func TestSetActiveSessionItem_Success(t *testing.T) {
	t.Parallel()
	svc, emitter, _ := testService(t)

	img := makeImageJPEG(t, 64, 64)
	itemID := svc.AddToSession(img, nil, "sd", false, nil)
	require.True(t, itemID > 0)

	err := svc.SetActiveSessionItem(itemID)
	require.NoError(t, err)
	assert.True(t, emitter.hasEvent("session:active"))
}

func TestDeleteSessionItem_Success(t *testing.T) {
	t.Parallel()
	svc, emitter, _ := testService(t)

	img := makeImageJPEG(t, 64, 64)
	itemID := svc.AddToSession(img, nil, "sd", false, nil)
	require.True(t, itemID > 0)

	err := svc.DeleteSessionItem(itemID)
	require.NoError(t, err)
	assert.True(t, emitter.hasEvent("session:removed"))
}

func TestDeleteSessionItem_NonExistent(t *testing.T) {
	t.Parallel()
	svc, _, _ := testService(t)

	err := svc.DeleteSessionItem(99999)
	assert.Error(t, err)
}

func TestClearSession_Success(t *testing.T) {
	t.Parallel()
	svc, emitter, _ := testService(t)

	img := makeImageJPEG(t, 64, 64)
	itemID := svc.AddToSession(img, nil, "sd", false, nil)
	require.True(t, itemID > 0)

	err := svc.ClearSession()
	require.NoError(t, err)
	assert.True(t, emitter.hasEvent("session:cleared"))

	items, err := svc.GetSessionItems()
	require.NoError(t, err)
	assert.Empty(t, items)
}

func TestClearSession_EmptySession(t *testing.T) {
	t.Parallel()
	svc, _, _ := testService(t)

	err := svc.ClearSession()
	assert.NoError(t, err)
}

func TestAddToSession_Success(t *testing.T) {
	t.Parallel()
	svc, emitter, dataDir := testService(t)

	img := makeImageJPEG(t, 64, 64)
	itemID := svc.AddToSession(img, nil, "sd", false, nil)
	assert.True(t, itemID > 0)
	assert.True(t, emitter.hasEvent("session:added"))
	assert.True(t, emitter.hasEvent("session:active"))

	sessions, _ := svc.ListSessions()
	sessionDir := filepath.Join(dataDir, "sessions", fmt.Sprintf("%d", sessions[0].ID))
	fileName := fmt.Sprintf("%d.png", itemID)
	_, err := os.Stat(filepath.Join(sessionDir, fileName))
	assert.NoError(t, err)
}

func TestAddToSession_WithInfo(t *testing.T) {
	t.Parallel()
	svc, _, _ := testService(t)

	img := makeImageJPEG(t, 64, 64)
	info := SDInfo{
		Prompt:         "test prompt",
		NegativePrompt: "bad quality",
		SamplerName:    "Euler a",
		Steps:          20,
		CfgScale:       7.5,
		Seed:           12345,
		Width:          512,
		Height:         512,
		Denoising:      0.5,
	}
	infoJSON, _ := json.Marshal(info)

	itemID := svc.AddToSession(img, infoJSON, "sd", false, nil)
	assert.True(t, itemID > 0)

	items, err := svc.GetSessionItems()
	require.NoError(t, err)
	require.Len(t, items, 1)
	assert.Equal(t, "test prompt", items[0].Prompt)
	assert.Equal(t, "bad quality", items[0].NegativePrompt)
	assert.Equal(t, "Euler a", items[0].Sampler)
	assert.Equal(t, 20, items[0].Steps)
	assert.Equal(t, 7.5, items[0].CfgScale)
	assert.Equal(t, 512, items[0].Width)
	assert.Equal(t, 512, items[0].Height)
	require.NotNil(t, items[0].Seed)
	assert.Equal(t, int64(12345), *items[0].Seed)
}

func TestAddToSession_TooLarge(t *testing.T) {
	t.Parallel()
	svc, _, _ := testService(t)

	largeBase64 := make([]byte, 51*1024*1024)
	for i := range largeBase64 {
		largeBase64[i] = 'A'
	}
	itemID := svc.AddToSession(string(largeBase64), nil, "sd", false, nil)
	assert.Equal(t, int64(0), itemID)
}

func TestAddToSession_InvalidBase64(t *testing.T) {
	t.Parallel()
	svc, _, _ := testService(t)

	itemID := svc.AddToSession("!!!not-valid-base64!!!", nil, "sd", false, nil)
	assert.Equal(t, int64(0), itemID)
}

func TestAddToSession_InvalidImageData(t *testing.T) {
	t.Parallel()
	svc, _, _ := testService(t)

	itemID := svc.AddToSession(base64.StdEncoding.EncodeToString([]byte("not an image")), nil, "sd", false, nil)
	assert.Equal(t, int64(0), itemID)
}

func TestAddToSession_Preview(t *testing.T) {
	t.Parallel()
	svc, _, _ := testService(t)

	img := makeImageJPEG(t, 64, 64)
	itemID := svc.AddToSession(img, nil, "sd", true, nil)
	assert.True(t, itemID > 0)

	items, err := svc.GetSessionItems()
	require.NoError(t, err)
	require.Len(t, items, 1)
	assert.True(t, items[0].IsPreview)
}

func TestAddToSession_WithPresetID(t *testing.T) {
	t.Parallel()
	svc, _, _ := testService(t)

	img := makeImageJPEG(t, 64, 64)
	presetID := int64(42)
	itemID := svc.AddToSession(img, nil, "sd", false, &presetID)
	assert.True(t, itemID > 0)

	items, err := svc.GetSessionItems()
	require.NoError(t, err)
	require.Len(t, items, 1)
	require.NotNil(t, items[0].PresetID)
	assert.Equal(t, int64(42), *items[0].PresetID)
}

func TestGetSessionImage_Success(t *testing.T) {
	t.Parallel()
	svc, _, _ := testService(t)

	img := makeImageJPEG(t, 64, 64)
	itemID := svc.AddToSession(img, nil, "sd", false, nil)
	require.True(t, itemID > 0)

	imgBase64, err := svc.GetSessionImage(itemID)
	require.NoError(t, err)
	assert.NotEmpty(t, imgBase64)

	decoded, err := base64.StdEncoding.DecodeString(imgBase64)
	require.NoError(t, err)
	assert.True(t, len(decoded) > 0)
}

func TestGetSessionImage_NonExistent(t *testing.T) {
	t.Parallel()
	svc, _, _ := testService(t)

	_, err := svc.GetSessionImage(99999)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "session item not found")
}

func TestGetSessionThumb_Success(t *testing.T) {
	t.Parallel()
	svc, _, _ := testService(t)

	img := makeImageJPEG(t, 256, 256)
	itemID := svc.AddToSession(img, nil, "sd", false, nil)
	require.True(t, itemID > 0)

	thumbBase64, err := svc.GetSessionThumb(itemID)
	require.NoError(t, err)
	assert.NotEmpty(t, thumbBase64)

	decoded, err := base64.StdEncoding.DecodeString(thumbBase64)
	require.NoError(t, err)
	assert.True(t, len(decoded) > 0)
}

func TestGetSessionThumb_NonExistent(t *testing.T) {
	t.Parallel()
	svc, _, _ := testService(t)

	_, err := svc.GetSessionThumb(99999)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "session item not found")
}

func TestGetSessionThumb_SmallImage(t *testing.T) {
	t.Parallel()
	svc, _, _ := testService(t)

	img := makeImageJPEG(t, 64, 64)
	itemID := svc.AddToSession(img, nil, "sd", false, nil)
	require.True(t, itemID > 0)

	thumbBase64, err := svc.GetSessionThumb(itemID)
	require.NoError(t, err)
	assert.NotEmpty(t, thumbBase64)
}

func TestHasSessionItems_Empty(t *testing.T) {
	t.Parallel()
	svc, _, _ := testService(t)

	has, err := svc.HasSessionItems()
	require.NoError(t, err)
	assert.False(t, has)
}

func TestHasSessionItems_WithItems(t *testing.T) {
	t.Parallel()
	svc, _, _ := testService(t)

	img := makeImageJPEG(t, 64, 64)
	itemID := svc.AddToSession(img, nil, "sd", false, nil)
	require.True(t, itemID > 0)

	has, err := svc.HasSessionItems()
	require.NoError(t, err)
	assert.True(t, has)
}

func TestConfirmClose_Discard(t *testing.T) {
	t.Parallel()
	svc, _, _ := testService(t)

	img := makeImageJPEG(t, 64, 64)
	itemID := svc.AddToSession(img, nil, "sd", false, nil)
	require.True(t, itemID > 0)

	_, err := svc.CreateSession("Extra")
	require.NoError(t, err)

	svc.ConfirmClose("discard")

	has, err := svc.HasSessionItems()
	require.NoError(t, err)
	assert.False(t, has)

	sessions, err := svc.ListSessions()
	require.NoError(t, err)
	assert.Len(t, sessions, 1)
	assert.Equal(t, "Default", sessions[0].Name)
}

func TestConfirmClose_OtherAction(t *testing.T) {
	t.Parallel()
	svc, _, _ := testService(t)

	img := makeImageJPEG(t, 64, 64)
	itemID := svc.AddToSession(img, nil, "sd", false, nil)
	require.True(t, itemID > 0)

	svc.ConfirmClose("save")

	has, err := svc.HasSessionItems()
	require.NoError(t, err)
	assert.True(t, has)
}

func TestConfirmClose_EmptyAction(t *testing.T) {
	t.Parallel()
	svc, _, _ := testService(t)

	img := makeImageJPEG(t, 64, 64)
	itemID := svc.AddToSession(img, nil, "sd", false, nil)
	require.True(t, itemID > 0)

	svc.ConfirmClose("")

	has, err := svc.HasSessionItems()
	require.NoError(t, err)
	assert.True(t, has)
}

func TestGetSessionItems_AfterAdding(t *testing.T) {
	t.Parallel()
	svc, _, _ := testService(t)

	img1 := makeImageJPEG(t, 64, 64)
	img2 := makeImageJPEG(t, 128, 128)

	id1 := svc.AddToSession(img1, nil, "sd", false, nil)
	require.True(t, id1 > 0)
	id2 := svc.AddToSession(img2, nil, "txt2img", false, nil)
	require.True(t, id2 > 0)

	items, err := svc.GetSessionItems()
	require.NoError(t, err)
	assert.Len(t, items, 2)

	var found1, found2 bool
	for _, item := range items {
		if item.ID == id1 {
			found1 = true
			assert.Equal(t, "sd", item.Source)
		}
		if item.ID == id2 {
			found2 = true
			assert.Equal(t, "txt2img", item.Source)
		}
	}
	assert.True(t, found1)
	assert.True(t, found2)
}

func TestGetActiveSessionItem_AfterAdd(t *testing.T) {
	t.Parallel()
	svc, _, _ := testService(t)

	img := makeImageJPEG(t, 64, 64)
	itemID := svc.AddToSession(img, nil, "sd", false, nil)
	require.True(t, itemID > 0)

	active, err := svc.GetActiveSessionItem()
	require.NoError(t, err)
	require.NotNil(t, active)
	assert.Equal(t, itemID, active.ID)
}

func TestSetActiveSessionItem_SwitchesActive(t *testing.T) {
	t.Parallel()
	svc, _, _ := testService(t)

	img1 := makeImageJPEG(t, 64, 64)
	img2 := makeImageJPEG(t, 64, 64)
	id1 := svc.AddToSession(img1, nil, "sd", false, nil)
	require.True(t, id1 > 0)
	id2 := svc.AddToSession(img2, nil, "sd", false, nil)
	require.True(t, id2 > 0)

	active, err := svc.GetActiveSessionItem()
	require.NoError(t, err)
	require.NotNil(t, active)
	assert.Equal(t, id2, active.ID)

	err = svc.SetActiveSessionItem(id1)
	require.NoError(t, err)

	active, err = svc.GetActiveSessionItem()
	require.NoError(t, err)
	require.NotNil(t, active)
	assert.Equal(t, id1, active.ID)
}

func TestAddToSession_MultipleSessions(t *testing.T) {
	t.Parallel()
	svc, _, _ := testService(t)

	img := makeImageJPEG(t, 64, 64)
	id1 := svc.AddToSession(img, nil, "sd", false, nil)
	require.True(t, id1 > 0)

	_, err := svc.CreateSession("Second Session")
	require.NoError(t, err)

	id2 := svc.AddToSession(img, nil, "sd", false, nil)
	require.True(t, id2 > 0)

	items, err := svc.GetSessionItems()
	require.NoError(t, err)
	assert.Len(t, items, 1)
	assert.Equal(t, id2, items[0].ID)
}
