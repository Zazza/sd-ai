package preset

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func testDB(t *testing.T) *DB {
	t.Helper()
	db, err := Open(":memory:")
	require.NoError(t, err)
	t.Cleanup(func() { db.Close() })
	return db
}

func intPtr(v int64) *int64    { return &v }
func floatPtr(v float64) *float64 { return &v }
func boolPtr(v bool) *bool     { return &v }

func TestOpen_MemoryDB(t *testing.T) {
	t.Parallel()
	db, err := Open(":memory:")
	require.NoError(t, err)
	require.NotNil(t, db)
	assert.NoError(t, db.Close())
}

func TestPreset_CreateAndGet(t *testing.T) {
	t.Parallel()
	db := testDB(t)

	seed := int64(42)
	ds := 0.75
	cs := 2
	bs := 1
	bc := 4
	hf := true
	hu := 2.0
	hds := 0.5
	tid := int64(10)

	p := &Preset{
		Name:                   "test preset",
		PresetType:             "portrait",
		Prompt:                 "a beautiful landscape",
		NegativePrompt:         "ugly, blurry",
		Sampler:                "Euler a",
		ScheduleType:           "Karras",
		Steps:                  30,
		CfgScale:               7.5,
		ModelName:              "model_v1",
		Seed:                   &seed,
		DenoisingStrength:      &ds,
		ClipSkip:               &cs,
		BatchSize:              &bs,
		BatchCount:             &bc,
		HiresFix:               &hf,
		HiresUpscale:           &hu,
		HiresDenoisingStrength: &hds,
		HiresUpscaler:          "R-ESRGAN 4x+",
		VAE:                    "vae-ft-mse",
		TypeID:                 &tid,
		Tags:                   "landscape, nature",
		Loras:                  "detail_enhancer:0.8",
	}

	err := db.Create(p)
	require.NoError(t, err)
	assert.Greater(t, p.ID, int64(0))

	got, err := db.Get(p.ID)
	require.NoError(t, err)

	assert.Equal(t, p.ID, got.ID)
	assert.Equal(t, "test preset", got.Name)
	assert.Equal(t, "portrait", got.PresetType)
	assert.Equal(t, "a beautiful landscape", got.Prompt)
	assert.Equal(t, "ugly, blurry", got.NegativePrompt)
	assert.Equal(t, "Euler a", got.Sampler)
	assert.Equal(t, "Karras", got.ScheduleType)
	assert.Equal(t, 30, got.Steps)
	assert.Equal(t, 7.5, got.CfgScale)
	assert.Equal(t, "model_v1", got.ModelName)
	require.NotNil(t, got.Seed)
	assert.Equal(t, int64(42), *got.Seed)
	require.NotNil(t, got.DenoisingStrength)
	assert.Equal(t, 0.75, *got.DenoisingStrength)
	require.NotNil(t, got.ClipSkip)
	assert.Equal(t, 2, *got.ClipSkip)
	require.NotNil(t, got.HiresFix)
	assert.True(t, *got.HiresFix)
	assert.Equal(t, "R-ESRGAN 4x+", got.HiresUpscaler)
	assert.Equal(t, "vae-ft-mse", got.VAE)
	require.NotNil(t, got.TypeID)
	assert.Equal(t, int64(10), *got.TypeID)
	assert.Equal(t, "landscape, nature", got.Tags)
	assert.Equal(t, "detail_enhancer:0.8", got.Loras)
}

func TestPreset_CreateMinimal(t *testing.T) {
	t.Parallel()
	db := testDB(t)

	p := &Preset{Name: "minimal"}
	err := db.Create(p)
	require.NoError(t, err)
	assert.Greater(t, p.ID, int64(0))

	got, err := db.Get(p.ID)
	require.NoError(t, err)
	assert.Equal(t, "minimal", got.Name)
	assert.Nil(t, got.Seed)
	assert.Nil(t, got.DenoisingStrength)
	assert.Nil(t, got.ClipSkip)
	assert.Nil(t, got.HiresFix)
	assert.Nil(t, got.TypeID)
}

func TestPreset_GetNonexistent(t *testing.T) {
	t.Parallel()
	db := testDB(t)

	_, err := db.Get(99999)
	assert.Error(t, err)
}

func TestPreset_Update(t *testing.T) {
	t.Parallel()
	db := testDB(t)

	p := &Preset{Name: "original", Prompt: "old prompt", Steps: 20}
	require.NoError(t, db.Create(p))

	p.Name = "updated"
	p.Prompt = "new prompt"
	p.Steps = 30
	err := db.Update(p)
	require.NoError(t, err)

	got, err := db.Get(p.ID)
	require.NoError(t, err)
	assert.Equal(t, "updated", got.Name)
	assert.Equal(t, "new prompt", got.Prompt)
	assert.Equal(t, 30, got.Steps)
}

func TestPreset_UpdateNonexistent(t *testing.T) {
	t.Parallel()
	db := testDB(t)

	p := &Preset{ID: 99999, Name: "ghost"}
	err := db.Update(p)
	assert.NoError(t, err)
}

func TestPreset_Delete(t *testing.T) {
	t.Parallel()
	db := testDB(t)

	p := &Preset{Name: "to-delete"}
	require.NoError(t, db.Create(p))

	err := db.Delete(p.ID)
	require.NoError(t, err)

	_, err = db.Get(p.ID)
	assert.Error(t, err)
}

func TestPreset_DeleteNonexistent(t *testing.T) {
	t.Parallel()
	db := testDB(t)

	err := db.Delete(99999)
	assert.NoError(t, err)
}

func TestPreset_List(t *testing.T) {
	t.Parallel()
	db := testDB(t)

	items := []Preset{
		{Name: "first"},
		{Name: "second"},
		{Name: "third"},
	}
	for i := range items {
		require.NoError(t, db.Create(&items[i]))
	}

	list, err := db.List()
	require.NoError(t, err)
	assert.Len(t, list, 3)
}

func TestPreset_ListEmpty(t *testing.T) {
	t.Parallel()
	db := testDB(t)

	list, err := db.List()
	require.NoError(t, err)
	assert.Nil(t, list)
}

func TestPreset_ListByType(t *testing.T) {
	t.Parallel()
	db := testDB(t)

	require.NoError(t, db.Create(&Preset{Name: "p1", PresetType: "portrait"}))
	require.NoError(t, db.Create(&Preset{Name: "p2", PresetType: "landscape"}))
	require.NoError(t, db.Create(&Preset{Name: "p3", PresetType: "portrait"}))

	portraits, err := db.ListByType("portrait")
	require.NoError(t, err)
	assert.Len(t, portraits, 2)
	for _, p := range portraits {
		assert.Equal(t, "portrait", p.PresetType)
	}

	landscapes, err := db.ListByType("landscape")
	require.NoError(t, err)
	assert.Len(t, landscapes, 1)
	assert.Equal(t, "p2", landscapes[0].Name)

	empty, err := db.ListByType("nonexistent")
	require.NoError(t, err)
	assert.Nil(t, empty)
}

func TestPreset_GetByIDs(t *testing.T) {
	t.Parallel()
	db := testDB(t)

	p1 := &Preset{Name: "a"}
	p2 := &Preset{Name: "b"}
	p3 := &Preset{Name: "c"}
	require.NoError(t, db.Create(p1))
	require.NoError(t, db.Create(p2))
	require.NoError(t, db.Create(p3))

	result, err := db.GetByIDs([]int64{p1.ID, p3.ID})
	require.NoError(t, err)
	assert.Len(t, result, 2)

	names := map[string]bool{}
	for _, p := range result {
		names[p.Name] = true
	}
	assert.True(t, names["a"])
	assert.True(t, names["c"])
	assert.False(t, names["b"])
}

func TestPreset_GetByIDs_Empty(t *testing.T) {
	t.Parallel()
	db := testDB(t)

	result, err := db.GetByIDs(nil)
	require.NoError(t, err)
	assert.Nil(t, result)

	result, err = db.GetByIDs([]int64{})
	require.NoError(t, err)
	assert.Nil(t, result)
}

func TestPreset_CreateBatch(t *testing.T) {
	t.Parallel()
	db := testDB(t)

	items := []Preset{
		{Name: "batch1", Steps: 20},
		{Name: "batch2", Steps: 30},
	}
	created, err := db.CreateBatch(items)
	require.NoError(t, err)
	assert.Len(t, created, 2)
	assert.Greater(t, created[0].ID, int64(0))
	assert.Greater(t, created[1].ID, int64(0))
	assert.Equal(t, "batch1", created[0].Name)
	assert.Equal(t, "batch2", created[1].Name)

	all, err := db.List()
	require.NoError(t, err)
	assert.Len(t, all, 2)
}

func TestPresetType_CRUD(t *testing.T) {
	t.Parallel()
	db := testDB(t)

	pt := &PresetType{Name: "Character", Description: "Character presets", SortOrder: 1}
	err := db.CreatePresetType(pt)
	require.NoError(t, err)
	assert.Greater(t, pt.ID, int64(0))

	got, err := db.GetPresetType(pt.ID)
	require.NoError(t, err)
	assert.Equal(t, "Character", got.Name)
	assert.Equal(t, "Character presets", got.Description)
	assert.Equal(t, 1, got.SortOrder)

	pt.Name = "Portrait"
	pt.Description = "Portrait presets"
	pt.SortOrder = 2
	err = db.UpdatePresetType(pt)
	require.NoError(t, err)

	got, err = db.GetPresetType(pt.ID)
	require.NoError(t, err)
	assert.Equal(t, "Portrait", got.Name)
	assert.Equal(t, "Portrait presets", got.Description)
	assert.Equal(t, 2, got.SortOrder)

	err = db.DeletePresetType(pt.ID)
	require.NoError(t, err)

	_, err = db.GetPresetType(pt.ID)
	assert.Error(t, err)
}

func TestPresetType_GetNonexistent(t *testing.T) {
	t.Parallel()
	db := testDB(t)

	_, err := db.GetPresetType(99999)
	assert.Error(t, err)
}

func TestPresetType_List(t *testing.T) {
	t.Parallel()
	db := testDB(t)

	require.NoError(t, db.CreatePresetType(&PresetType{Name: "Zebra", SortOrder: 3}))
	require.NoError(t, db.CreatePresetType(&PresetType{Name: "Alpha", SortOrder: 1}))
	require.NoError(t, db.CreatePresetType(&PresetType{Name: "Middle", SortOrder: 2}))

	list, err := db.ListPresetTypes()
	require.NoError(t, err)
	assert.Len(t, list, 3)
	assert.Equal(t, "Alpha", list[0].Name)
	assert.Equal(t, "Middle", list[1].Name)
	assert.Equal(t, "Zebra", list[2].Name)
}

func TestPresetType_DeleteNonexistent(t *testing.T) {
	t.Parallel()
	db := testDB(t)

	err := db.DeletePresetType(99999)
	assert.NoError(t, err)
}

func TestSetting_SetAndGet(t *testing.T) {
	t.Parallel()
	db := testDB(t)

	err := db.SetSetting("test_key", "test_value")
	require.NoError(t, err)

	val, err := db.GetSetting("test_key")
	require.NoError(t, err)
	assert.Equal(t, "test_value", val)
}

func TestSetting_GetNonexistent(t *testing.T) {
	t.Parallel()
	db := testDB(t)

	val, err := db.GetSetting("nonexistent_key")
	require.NoError(t, err)
	assert.Equal(t, "", val)
}

func TestSetting_UpdateExisting(t *testing.T) {
	t.Parallel()
	db := testDB(t)

	require.NoError(t, db.SetSetting("key1", "value1"))
	require.NoError(t, db.SetSetting("key1", "value2"))

	val, err := db.GetSetting("key1")
	require.NoError(t, err)
	assert.Equal(t, "value2", val)
}

func TestSetting_GetAllSettings(t *testing.T) {
	t.Parallel()
	db := testDB(t)

	require.NoError(t, db.SetSetting("k1", "v1"))
	require.NoError(t, db.SetSetting("k2", "v2"))
	require.NoError(t, db.SetSetting("k3", "v3"))

	settings, err := db.GetAllSettings()
	require.NoError(t, err)
	assert.Len(t, settings, 3)
	assert.Equal(t, "v1", settings["k1"])
	assert.Equal(t, "v2", settings["k2"])
	assert.Equal(t, "v3", settings["k3"])
}

func TestSetting_GetAllSettingsEmpty(t *testing.T) {
	t.Parallel()
	db := testDB(t)

	settings, err := db.GetAllSettings()
	require.NoError(t, err)
	assert.Empty(t, settings)
}

func TestSavedDescription_CRUD(t *testing.T) {
	t.Parallel()
	db := testDB(t)

	created, err := db.CreateDescription("a beautiful sunset")
	require.NoError(t, err)
	assert.Greater(t, created.ID, int64(0))
	assert.Equal(t, "a beautiful sunset", created.Text)

	items, err := db.ListDescriptions()
	require.NoError(t, err)
	assert.Len(t, items, 1)
	assert.Equal(t, created.ID, items[0].ID)
	assert.Equal(t, "a beautiful sunset", items[0].Text)

	err = db.DeleteDescription(created.ID)
	require.NoError(t, err)

	items, err = db.ListDescriptions()
	require.NoError(t, err)
	assert.Nil(t, items)
}

func TestSavedDescription_CreateFull(t *testing.T) {
	t.Parallel()
	db := testDB(t)

	s := &SavedDescription{
		Text:           "full description",
		Name:           "My Desc",
		NegativePrompt: "bad quality",
		Type:           "portrait",
	}
	created, err := db.CreateDescriptionFull(s)
	require.NoError(t, err)
	assert.Greater(t, created.ID, int64(0))
	assert.Equal(t, "full description", created.Text)
	assert.Equal(t, "My Desc", created.Name)
	assert.Equal(t, "bad quality", created.NegativePrompt)
	assert.Equal(t, "portrait", created.Type)

	items, err := db.ListDescriptions()
	require.NoError(t, err)
	require.Len(t, items, 1)
	assert.Equal(t, "My Desc", items[0].Name)
	assert.Equal(t, "bad quality", items[0].NegativePrompt)
	assert.Equal(t, "portrait", items[0].Type)
}

func TestSavedDescription_Update(t *testing.T) {
	t.Parallel()
	db := testDB(t)

	s := &SavedDescription{
		Text:           "original",
		Name:           "Old Name",
		NegativePrompt: "old neg",
		Type:           "old type",
	}
	created, err := db.CreateDescriptionFull(s)
	require.NoError(t, err)

	created.Text = "updated text"
	created.Name = "New Name"
	created.NegativePrompt = "new neg"
	created.Type = "new type"
	err = db.UpdateDescription(created)
	require.NoError(t, err)

	items, err := db.ListDescriptions()
	require.NoError(t, err)
	require.Len(t, items, 1)
	assert.Equal(t, "updated text", items[0].Text)
	assert.Equal(t, "New Name", items[0].Name)
	assert.Equal(t, "new neg", items[0].NegativePrompt)
	assert.Equal(t, "new type", items[0].Type)
}

func TestSavedDescription_DeleteNonexistent(t *testing.T) {
	t.Parallel()
	db := testDB(t)

	err := db.DeleteDescription(99999)
	assert.NoError(t, err)
}

func TestSavedDescription_ListEmpty(t *testing.T) {
	t.Parallel()
	db := testDB(t)

	items, err := db.ListDescriptions()
	require.NoError(t, err)
	assert.Nil(t, items)
}

func TestSavedPrompt_CreateListDelete(t *testing.T) {
	t.Parallel()
	db := testDB(t)

	created, err := db.CreatePrompt("masterpiece, best quality")
	require.NoError(t, err)
	assert.Greater(t, created.ID, int64(0))
	assert.Equal(t, "masterpiece, best quality", created.Text)

	items, err := db.ListPrompts()
	require.NoError(t, err)
	assert.Len(t, items, 1)
	assert.Equal(t, created.ID, items[0].ID)
	assert.Equal(t, "masterpiece, best quality", items[0].Text)

	err = db.DeletePrompt(created.ID)
	require.NoError(t, err)

	items, err = db.ListPrompts()
	require.NoError(t, err)
	assert.Nil(t, items)
}

func TestSavedPrompt_DeleteNonexistent(t *testing.T) {
	t.Parallel()
	db := testDB(t)

	err := db.DeletePrompt(99999)
	assert.NoError(t, err)
}

func TestSavedPrompt_ListEmpty(t *testing.T) {
	t.Parallel()
	db := testDB(t)

	items, err := db.ListPrompts()
	require.NoError(t, err)
	assert.Nil(t, items)
}

func TestSession_CreateAndList(t *testing.T) {
	t.Parallel()
	db := testDB(t)

	activeID, err := db.GetActiveSessionID()
	require.NoError(t, err)
	assert.Greater(t, activeID, int64(0))

	sessions, err := db.ListSessions()
	require.NoError(t, err)
	assert.Len(t, sessions, 1)
	assert.Equal(t, "Default", sessions[0].Name)

	created, err := db.CreateSession("My Session")
	require.NoError(t, err)
	assert.Greater(t, created.ID, int64(0))
	assert.Equal(t, "My Session", created.Name)

	sessions, err = db.ListSessions()
	require.NoError(t, err)
	assert.Len(t, sessions, 2)
}

func TestSession_Rename(t *testing.T) {
	t.Parallel()
	db := testDB(t)

	created, err := db.CreateSession("Original")
	require.NoError(t, err)

	err = db.RenameSession(created.ID, "Renamed")
	require.NoError(t, err)

	sessions, err := db.ListSessions()
	require.NoError(t, err)
	for _, s := range sessions {
		if s.ID == created.ID {
			assert.Equal(t, "Renamed", s.Name)
		}
	}
}

func TestSession_Delete(t *testing.T) {
	t.Parallel()
	db := testDB(t)

	created, err := db.CreateSession("ToDelete")
	require.NoError(t, err)

	err = db.DeleteSession(created.ID)
	require.NoError(t, err)

	sessions, err := db.ListSessions()
	require.NoError(t, err)
	for _, s := range sessions {
		assert.NotEqual(t, created.ID, s.ID)
	}
}

func TestSession_SwitchActive(t *testing.T) {
	t.Parallel()
	db := testDB(t)

	s1, err := db.CreateSession("Session 1")
	require.NoError(t, err)

	s2, err := db.CreateSession("Session 2")
	require.NoError(t, err)

	err = db.SetActiveSession(s2.ID)
	require.NoError(t, err)

	activeID, err := db.GetActiveSessionID()
	require.NoError(t, err)
	assert.Equal(t, s2.ID, activeID)

	err = db.SetActiveSession(s1.ID)
	require.NoError(t, err)

	activeID, err = db.GetActiveSessionID()
	require.NoError(t, err)
	assert.Equal(t, s1.ID, activeID)
}

func TestSession_ItemCRUD(t *testing.T) {
	t.Parallel()
	db := testDB(t)

	sess, err := db.CreateSession("Items Test")
	require.NoError(t, err)

	seed := int64(12345)
	presetID := int64(5)
	item := &SessionItem{
		SessionID:      sess.ID,
		FileName:       "img001.png",
		ThumbName:      "thumb001.png",
		Source:         "txt2img",
		Prompt:         "a cat",
		NegativePrompt: "blurry",
		Sampler:        "Euler a",
		Steps:          20,
		CfgScale:       7.0,
		Seed:           &seed,
		Denoising:      0.0,
		Width:          512,
		Height:         512,
		IsPreview:      false,
		PresetID:       &presetID,
	}

	itemID, err := db.AddSessionItem(item)
	require.NoError(t, err)
	assert.Greater(t, itemID, int64(0))

	got, err := db.GetSessionItem(itemID)
	require.NoError(t, err)
	assert.Equal(t, sess.ID, got.SessionID)
	assert.Equal(t, "img001.png", got.FileName)
	assert.Equal(t, "thumb001.png", got.ThumbName)
	assert.Equal(t, "txt2img", got.Source)
	assert.Equal(t, "a cat", got.Prompt)
	assert.Equal(t, "blurry", got.NegativePrompt)
	assert.Equal(t, "Euler a", got.Sampler)
	assert.Equal(t, 20, got.Steps)
	assert.Equal(t, 7.0, got.CfgScale)
	require.NotNil(t, got.Seed)
	assert.Equal(t, int64(12345), *got.Seed)
	assert.Equal(t, 512, got.Width)
	assert.Equal(t, 512, got.Height)
	require.NotNil(t, got.PresetID)
	assert.Equal(t, int64(5), *got.PresetID)
	assert.True(t, got.IsActive)
}

func TestSession_GetItems(t *testing.T) {
	t.Parallel()
	db := testDB(t)

	sess, err := db.CreateSession("Multi Items")
	require.NoError(t, err)

	for i := 0; i < 3; i++ {
		_, err := db.AddSessionItem(&SessionItem{
			SessionID: sess.ID,
			FileName:  "img.png",
			Source:    "txt2img",
			Prompt:    "prompt",
		})
		require.NoError(t, err)
	}

	items, err := db.GetSessionItems(sess.ID)
	require.NoError(t, err)
	assert.Len(t, items, 3)
}

func TestSession_GetItemsEmpty(t *testing.T) {
	t.Parallel()
	db := testDB(t)

	sess, err := db.CreateSession("Empty Items")
	require.NoError(t, err)

	items, err := db.GetSessionItems(sess.ID)
	require.NoError(t, err)
	assert.Nil(t, items)
}

func TestSession_ActiveItem(t *testing.T) {
	t.Parallel()
	db := testDB(t)

	sess, err := db.CreateSession("Active Item Test")
	require.NoError(t, err)

	active, err := db.GetActiveItem(sess.ID)
	require.NoError(t, err)
	assert.Nil(t, active)

	id1, err := db.AddSessionItem(&SessionItem{SessionID: sess.ID, FileName: "first.png", Source: "txt2img", Prompt: "p1"})
	require.NoError(t, err)

	id2, err := db.AddSessionItem(&SessionItem{SessionID: sess.ID, FileName: "second.png", Source: "txt2img", Prompt: "p2"})
	require.NoError(t, err)

	active, err = db.GetActiveItem(sess.ID)
	require.NoError(t, err)
	require.NotNil(t, active)
	assert.Equal(t, id2, active.ID)

	err = db.SetActiveItem(id1, sess.ID)
	require.NoError(t, err)

	active, err = db.GetActiveItem(sess.ID)
	require.NoError(t, err)
	require.NotNil(t, active)
	assert.Equal(t, id1, active.ID)

	items, err := db.GetSessionItems(sess.ID)
	require.NoError(t, err)
	for _, item := range items {
		if item.ID == id1 {
			assert.True(t, item.IsActive)
		} else {
			assert.False(t, item.IsActive)
		}
	}
}

func TestSession_DeleteItem(t *testing.T) {
	t.Parallel()
	db := testDB(t)

	sess, err := db.CreateSession("Delete Item")
	require.NoError(t, err)

	itemID, err := db.AddSessionItem(&SessionItem{SessionID: sess.ID, FileName: "del.png", Source: "txt2img", Prompt: "p"})
	require.NoError(t, err)

	err = db.DeleteSessionItem(itemID)
	require.NoError(t, err)

	_, err = db.GetSessionItem(itemID)
	assert.Error(t, err)
}

func TestSession_ClearItems(t *testing.T) {
	t.Parallel()
	db := testDB(t)

	sess, err := db.CreateSession("Clear Items")
	require.NoError(t, err)

	for i := 0; i < 5; i++ {
		_, err := db.AddSessionItem(&SessionItem{SessionID: sess.ID, FileName: "img.png", Source: "txt2img", Prompt: "p"})
		require.NoError(t, err)
	}

	err = db.ClearSessionItems(sess.ID)
	require.NoError(t, err)

	items, err := db.GetSessionItems(sess.ID)
	require.NoError(t, err)
	assert.Nil(t, items)
}

func TestSession_HasAnyItems(t *testing.T) {
	t.Parallel()
	db := testDB(t)

	has, err := db.HasAnyItems()
	require.NoError(t, err)
	assert.False(t, has)

	sess, err := db.CreateSession("Has Items")
	require.NoError(t, err)

	_, err = db.AddSessionItem(&SessionItem{SessionID: sess.ID, FileName: "img.png", Source: "txt2img", Prompt: "p"})
	require.NoError(t, err)

	has, err = db.HasAnyItems()
	require.NoError(t, err)
	assert.True(t, has)
}

func TestSession_DeleteAllSessions(t *testing.T) {
	t.Parallel()
	db := testDB(t)

	_, err := db.CreateSession("Extra 1")
	require.NoError(t, err)
	_, err = db.CreateSession("Extra 2")
	require.NoError(t, err)

	err = db.DeleteAllSessions()
	require.NoError(t, err)

	sessions, err := db.ListSessions()
	require.NoError(t, err)
	assert.Len(t, sessions, 1)
	assert.Equal(t, "Default", sessions[0].Name)

	activeID, err := db.GetActiveSessionID()
	require.NoError(t, err)
	assert.Equal(t, sessions[0].ID, activeID)
}

func TestSession_UpdateSessionItemPaths(t *testing.T) {
	t.Parallel()
	db := testDB(t)

	sess, err := db.CreateSession("Paths")
	require.NoError(t, err)

	itemID, err := db.AddSessionItem(&SessionItem{SessionID: sess.ID, FileName: "old.png", ThumbName: "old_thumb.png", Source: "txt2img", Prompt: "p"})
	require.NoError(t, err)

	err = db.UpdateSessionItemPaths(itemID, "new.png", "new_thumb.png")
	require.NoError(t, err)

	got, err := db.GetSessionItem(itemID)
	require.NoError(t, err)
	assert.Equal(t, "new.png", got.FileName)
	assert.Equal(t, "new_thumb.png", got.ThumbName)
}

func TestSession_ItemCount(t *testing.T) {
	t.Parallel()
	db := testDB(t)

	sess, err := db.CreateSession("Count Test")
	require.NoError(t, err)

	for i := 0; i < 3; i++ {
		_, err := db.AddSessionItem(&SessionItem{SessionID: sess.ID, FileName: "img.png", Source: "txt2img", Prompt: "p"})
		require.NoError(t, err)
	}

	sessions, err := db.ListSessions()
	require.NoError(t, err)
	for _, s := range sessions {
		if s.ID == sess.ID {
			assert.Equal(t, 3, s.ItemCount)
		}
	}
}

func TestCompoundPreset_CRUD(t *testing.T) {
	t.Parallel()
	db := testDB(t)

	p1 := &Preset{Name: "bg preset"}
	require.NoError(t, db.Create(p1))
	p2 := &Preset{Name: "char preset"}
	require.NoError(t, db.Create(p2))

	cp := &CompoundPreset{
		Name:        "Multi-step",
		Description: "Background + character",
		Steps: []CompoundPresetStep{
			{PresetID: p1.ID, Width: 1024, Height: 768, DenoisingStrength: 1.0},
			{PresetID: p2.ID, Width: 1024, Height: 768, DenoisingStrength: 0.5},
		},
	}
	err := db.CreateCompoundPreset(cp)
	require.NoError(t, err)
	assert.Greater(t, cp.ID, int64(0))

	got, err := db.GetCompoundPreset(cp.ID)
	require.NoError(t, err)
	assert.Equal(t, "Multi-step", got.Name)
	assert.Equal(t, "Background + character", got.Description)
	require.Len(t, got.Steps, 2)
	assert.Equal(t, p1.ID, got.Steps[0].PresetID)
	assert.Equal(t, int(1), got.Steps[0].StepOrder)
	assert.Equal(t, 1024, got.Steps[0].Width)
	assert.Equal(t, 1.0, got.Steps[0].DenoisingStrength)
	assert.Equal(t, p2.ID, got.Steps[1].PresetID)
	assert.Equal(t, int(2), got.Steps[1].StepOrder)
	assert.Equal(t, 0.5, got.Steps[1].DenoisingStrength)

	got.Name = "Updated"
	got.Description = "Updated desc"
	got.Steps = []CompoundPresetStep{
		{PresetID: p2.ID, Width: 512, Height: 512, DenoisingStrength: 0.8},
	}
	err = db.UpdateCompoundPreset(got)
	require.NoError(t, err)

	updated, err := db.GetCompoundPreset(cp.ID)
	require.NoError(t, err)
	assert.Equal(t, "Updated", updated.Name)
	assert.Equal(t, "Updated desc", updated.Description)
	require.Len(t, updated.Steps, 1)
	assert.Equal(t, p2.ID, updated.Steps[0].PresetID)

	err = db.DeleteCompoundPreset(cp.ID)
	require.NoError(t, err)

	_, err = db.GetCompoundPreset(cp.ID)
	assert.Error(t, err)
}

func TestCompoundPreset_List(t *testing.T) {
	t.Parallel()
	db := testDB(t)

	p := &Preset{Name: "step"}
	require.NoError(t, db.Create(p))

	cp1 := &CompoundPreset{Name: "First", Steps: []CompoundPresetStep{{PresetID: p.ID, Width: 512, Height: 512, DenoisingStrength: 0.5}}}
	cp2 := &CompoundPreset{Name: "Second", Steps: []CompoundPresetStep{{PresetID: p.ID, Width: 512, Height: 512, DenoisingStrength: 0.5}}}
	require.NoError(t, db.CreateCompoundPreset(cp1))
	require.NoError(t, db.CreateCompoundPreset(cp2))

	list, err := db.ListCompoundPresets()
	require.NoError(t, err)
	assert.Len(t, list, 2)
	require.Len(t, list[0].Steps, 1)
	require.Len(t, list[1].Steps, 1)
}

func TestCompoundPreset_ListEmpty(t *testing.T) {
	t.Parallel()
	db := testDB(t)

	list, err := db.ListCompoundPresets()
	require.NoError(t, err)
	assert.Nil(t, list)
}

func TestCompoundPreset_GetNonexistent(t *testing.T) {
	t.Parallel()
	db := testDB(t)

	_, err := db.GetCompoundPreset(99999)
	assert.Error(t, err)
}

func TestCompoundPreset_DeleteNonexistent(t *testing.T) {
	t.Parallel()
	db := testDB(t)

	err := db.DeleteCompoundPreset(99999)
	assert.NoError(t, err)
}

func TestSavedScene_CRUD(t *testing.T) {
	t.Parallel()
	db := testDB(t)

	s := &SavedScene{Name: "Test Scene", SceneJSON: `{"characters": []}`}
	err := db.CreateSavedScene(s)
	require.NoError(t, err)
	assert.Greater(t, s.ID, int64(0))

	got, err := db.GetSavedScene(s.ID)
	require.NoError(t, err)
	assert.Equal(t, "Test Scene", got.Name)
	assert.Equal(t, `{"characters": []}`, got.SceneJSON)

	s.Name = "Updated Scene"
	s.SceneJSON = `{"characters": [{"name": "Alice"}]}`
	err = db.UpdateSavedScene(s)
	require.NoError(t, err)

	got, err = db.GetSavedScene(s.ID)
	require.NoError(t, err)
	assert.Equal(t, "Updated Scene", got.Name)
	assert.Equal(t, `{"characters": [{"name": "Alice"}]}`, got.SceneJSON)

	items, err := db.ListSavedScenes()
	require.NoError(t, err)
	assert.Len(t, items, 1)

	err = db.DeleteSavedScene(s.ID)
	require.NoError(t, err)

	items, err = db.ListSavedScenes()
	require.NoError(t, err)
	assert.Nil(t, items)
}

func TestSavedScene_GetNonexistent(t *testing.T) {
	t.Parallel()
	db := testDB(t)

	_, err := db.GetSavedScene(99999)
	assert.Error(t, err)
}

func TestSavedScene_ListEmpty(t *testing.T) {
	t.Parallel()
	db := testDB(t)

	items, err := db.ListSavedScenes()
	require.NoError(t, err)
	assert.Nil(t, items)
}

func TestExportPreset_CRUD(t *testing.T) {
	t.Parallel()
	db := testDB(t)

	defaults, err := db.ListExportPresets()
	require.NoError(t, err)
	assert.Len(t, defaults, 4)

	ep := &ExportPreset{
		Name:          "Custom Export",
		Format:        "webp",
		Width:         1920,
		Height:        1080,
		LockRatio:     true,
		Quality:       85,
		Interpolation: "lanczos",
	}
	err = db.CreateExportPreset(ep)
	require.NoError(t, err)
	assert.Greater(t, ep.ID, int64(0))

	got, err := db.GetExportPreset(ep.ID)
	require.NoError(t, err)
	assert.Equal(t, "Custom Export", got.Name)
	assert.Equal(t, "webp", got.Format)
	assert.Equal(t, 1920, got.Width)
	assert.Equal(t, 1080, got.Height)
	assert.True(t, got.LockRatio)
	assert.Equal(t, 85, got.Quality)
	assert.Equal(t, "lanczos", got.Interpolation)

	got.Name = "Updated Export"
	got.Format = "png"
	got.Quality = 100
	err = db.UpdateExportPreset(got)
	require.NoError(t, err)

	updated, err := db.GetExportPreset(ep.ID)
	require.NoError(t, err)
	assert.Equal(t, "Updated Export", updated.Name)
	assert.Equal(t, "png", updated.Format)
	assert.Equal(t, 100, updated.Quality)

	err = db.DeleteExportPreset(ep.ID)
	require.NoError(t, err)

	_, err = db.GetExportPreset(ep.ID)
	assert.Error(t, err)
}

func TestExportPreset_GetNonexistent(t *testing.T) {
	t.Parallel()
	db := testDB(t)

	_, err := db.GetExportPreset(99999)
	assert.Error(t, err)
}

func TestGetAllTags(t *testing.T) {
	t.Parallel()
	db := testDB(t)

	tags, err := db.GetAllTags()
	require.NoError(t, err)
	assert.Nil(t, tags)

	require.NoError(t, db.Create(&Preset{Name: "p1", Tags: "landscape, nature"}))
	require.NoError(t, db.Create(&Preset{Name: "p2", Tags: "portrait, nature"}))
	require.NoError(t, db.Create(&Preset{Name: "p3", Tags: ""}))
	require.NoError(t, db.Create(&Preset{Name: "p4", Tags: "landscape"}))

	tags, err = db.GetAllTags()
	require.NoError(t, err)
	tagSet := map[string]bool{}
	for _, tag := range tags {
		tagSet[tag] = true
	}
	assert.True(t, tagSet["landscape"])
	assert.True(t, tagSet["nature"])
	assert.True(t, tagSet["portrait"])
	assert.Len(t, tags, 3)
}

func TestResolution_BuiltinDefaults(t *testing.T) {
	t.Parallel()
	db := testDB(t)

	items, err := db.ListResolutions()
	require.NoError(t, err)
	assert.Len(t, items, 14)
	assert.Equal(t, "Square 512x512", items[0].Name)
	assert.Equal(t, 512, items[0].Width)
	assert.Equal(t, 512, items[0].Height)
	assert.True(t, items[0].IsBuiltin)
	for _, r := range items {
		assert.True(t, r.IsBuiltin)
		assert.NotEmpty(t, r.CreatedAt)
	}
}

func TestResolution_CRUD(t *testing.T) {
	t.Parallel()
	db := testDB(t)

	r := &Resolution{Name: "Custom Res", Width: 640, Height: 480}
	err := db.CreateResolution(r)
	require.NoError(t, err)
	assert.Greater(t, r.ID, int64(0))
	assert.False(t, r.IsBuiltin)

	got, err := db.GetResolution(r.ID)
	require.NoError(t, err)
	assert.Equal(t, "Custom Res", got.Name)
	assert.Equal(t, 640, got.Width)
	assert.Equal(t, 480, got.Height)
	assert.False(t, got.IsBuiltin)

	got.Name = "Updated Res"
	got.Width = 800
	got.Height = 600
	err = db.UpdateResolution(got)
	require.NoError(t, err)

	updated, err := db.GetResolution(r.ID)
	require.NoError(t, err)
	assert.Equal(t, "Updated Res", updated.Name)
	assert.Equal(t, 800, updated.Width)
	assert.Equal(t, 600, updated.Height)

	err = db.DeleteResolution(r.ID)
	require.NoError(t, err)

	_, err = db.GetResolution(r.ID)
	assert.Error(t, err)
}

func TestResolution_List(t *testing.T) {
	t.Parallel()
	db := testDB(t)

	require.NoError(t, db.CreateResolution(&Resolution{Name: "Custom A", Width: 100, Height: 100}))
	require.NoError(t, db.CreateResolution(&Resolution{Name: "Custom B", Width: 200, Height: 200}))

	items, err := db.ListResolutions()
	require.NoError(t, err)
	assert.Len(t, items, 16)

	customCount := 0
	for _, r := range items {
		if !r.IsBuiltin {
			customCount++
		}
	}
	assert.Equal(t, 2, customCount)
}

func TestResolution_GetNonexistent(t *testing.T) {
	t.Parallel()
	db := testDB(t)

	_, err := db.GetResolution(99999)
	assert.Error(t, err)
}

func TestResolution_DeleteBuiltin(t *testing.T) {
	t.Parallel()
	db := testDB(t)

	items, err := db.ListResolutions()
	require.NoError(t, err)
	require.NotEmpty(t, items)

	err = db.DeleteResolution(items[0].ID)
	assert.Error(t, err)
	assert.Equal(t, "cannot delete builtin resolution", err.Error())
}

func TestResolution_DeleteNotFound(t *testing.T) {
	t.Parallel()
	db := testDB(t)

	err := db.DeleteResolution(99999)
	assert.Error(t, err)
	assert.Equal(t, "resolution not found", err.Error())
}

func TestHiresProfile_BuiltinDefaults(t *testing.T) {
	t.Parallel()
	db := testDB(t)

	items, err := db.ListHiresProfiles()
	require.NoError(t, err)
	assert.Len(t, items, 4)

	names := []string{"Light", "Standard", "Heavy", "Max"}
	for i, name := range names {
		assert.Equal(t, name, items[i].Name)
		assert.True(t, items[i].IsBuiltin)
		assert.NotEmpty(t, items[i].CreatedAt)
	}

	assert.Equal(t, 1.5, items[0].Upscale)
	assert.Equal(t, 0.3, items[0].DenoisingStrength)
	assert.Equal(t, 2.0, items[1].Upscale)
	assert.Equal(t, 0.45, items[1].DenoisingStrength)
	assert.Equal(t, 2.5, items[2].Upscale)
	assert.Equal(t, 0.55, items[2].DenoisingStrength)
	assert.Equal(t, 4.0, items[3].Upscale)
	assert.Equal(t, 0.4, items[3].DenoisingStrength)
}

func TestHiresProfile_CRUD(t *testing.T) {
	t.Parallel()
	db := testDB(t)

	h := &HiresProfile{Name: "Custom Profile", Upscale: 1.5, DenoisingStrength: 0.5, Upscaler: "R-ESRGAN 4x+"}
	err := db.CreateHiresProfile(h)
	require.NoError(t, err)
	assert.Greater(t, h.ID, int64(0))
	assert.False(t, h.IsBuiltin)

	got, err := db.GetHiresProfile(h.ID)
	require.NoError(t, err)
	assert.Equal(t, "Custom Profile", got.Name)
	assert.Equal(t, 1.5, got.Upscale)
	assert.Equal(t, 0.5, got.DenoisingStrength)
	assert.Equal(t, "R-ESRGAN 4x+", got.Upscaler)
	assert.False(t, got.IsBuiltin)

	got.Name = "Updated Profile"
	got.Upscale = 2.0
	got.DenoisingStrength = 0.6
	got.Upscaler = "SwinIR_4x"
	err = db.UpdateHiresProfile(got)
	require.NoError(t, err)

	updated, err := db.GetHiresProfile(h.ID)
	require.NoError(t, err)
	assert.Equal(t, "Updated Profile", updated.Name)
	assert.Equal(t, 2.0, updated.Upscale)
	assert.Equal(t, 0.6, updated.DenoisingStrength)
	assert.Equal(t, "SwinIR_4x", updated.Upscaler)

	err = db.DeleteHiresProfile(h.ID)
	require.NoError(t, err)

	_, err = db.GetHiresProfile(h.ID)
	assert.Error(t, err)
}

func TestHiresProfile_List(t *testing.T) {
	t.Parallel()
	db := testDB(t)

	require.NoError(t, db.CreateHiresProfile(&HiresProfile{Name: "Custom X", Upscale: 2.0, DenoisingStrength: 0.4, Upscaler: ""}))

	items, err := db.ListHiresProfiles()
	require.NoError(t, err)
	assert.Len(t, items, 5)

	customCount := 0
	for _, h := range items {
		if !h.IsBuiltin {
			customCount++
		}
	}
	assert.Equal(t, 1, customCount)
}

func TestHiresProfile_GetNonexistent(t *testing.T) {
	t.Parallel()
	db := testDB(t)

	_, err := db.GetHiresProfile(99999)
	assert.Error(t, err)
}

func TestHiresProfile_DeleteBuiltin(t *testing.T) {
	t.Parallel()
	db := testDB(t)

	items, err := db.ListHiresProfiles()
	require.NoError(t, err)
	require.NotEmpty(t, items)

	err = db.DeleteHiresProfile(items[0].ID)
	assert.Error(t, err)
	assert.Equal(t, "cannot delete builtin hires profile", err.Error())
}

func TestHiresProfile_DeleteNotFound(t *testing.T) {
	t.Parallel()
	db := testDB(t)

	err := db.DeleteHiresProfile(99999)
	assert.Error(t, err)
	assert.Equal(t, "hires profile not found", err.Error())
}
