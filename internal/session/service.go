package session

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"image"
	"image/jpeg"
	"io"
	"math"
	"os"
	"path/filepath"
	"strconv"

	"go-sd/internal/preset"

	xdraw "golang.org/x/image/draw"
)

type EventEmitter interface {
	Emit(event string, data ...any)
}

type Service struct {
	db      *preset.DB
	dataDir string
	emit    EventEmitter
}

func New(db *preset.DB, dataDir string, emit EventEmitter) *Service {
	return &Service{db: db, dataDir: dataDir, emit: emit}
}

type SDInfo struct {
	Prompt         string  `json:"prompt"`
	NegativePrompt string  `json:"negative_prompt"`
	SamplerName    string  `json:"sampler_name"`
	Steps          int     `json:"steps"`
	CfgScale       float64 `json:"cfg_scale"`
	Seed           int64   `json:"seed"`
	Width          int     `json:"width"`
	Height         int     `json:"height"`
	Denoising      float64 `json:"denoising_strength"`
}

func (s *Service) AddToSession(imageBase64 string, info json.RawMessage, source string, isPreview bool, presetID *int64) int64 {
	if len(imageBase64) > 50*1024*1024 {
		return 0
	}

	sessionID, err := s.db.GetActiveSessionID()
	if err != nil || sessionID == 0 {
		return 0
	}

	imgData, err := base64.StdEncoding.DecodeString(imageBase64)
	if err != nil {
		return 0
	}

	img, _, err := image.Decode(bytes.NewReader(imgData))
	if err != nil {
		return 0
	}

	item := &preset.SessionItem{
		SessionID: sessionID,
		Source:    source,
		IsPreview: isPreview,
		PresetID:  presetID,
		Width:     img.Bounds().Dx(),
		Height:    img.Bounds().Dy(),
	}

	if info != nil {
		var sd SDInfo
		if json.Unmarshal(info, &sd) == nil {
			item.Prompt = sd.Prompt
			item.NegativePrompt = sd.NegativePrompt
			item.Sampler = sd.SamplerName
			item.Steps = sd.Steps
			item.CfgScale = sd.CfgScale
			item.Seed = &sd.Seed
			item.Denoising = sd.Denoising
			if sd.Width > 0 {
				item.Width = sd.Width
			}
			if sd.Height > 0 {
				item.Height = sd.Height
			}
		}
	}

	itemID, err := s.db.AddSessionItem(item)
	if err != nil {
		return 0
	}

	sessionDir := filepath.Join(s.dataDir, "sessions", strconv.FormatInt(sessionID, 10))
	thumbDir := filepath.Join(s.dataDir, "thumbs", strconv.FormatInt(sessionID, 10))
	os.MkdirAll(sessionDir, 0o755)
	os.MkdirAll(thumbDir, 0o755)

	fileName := strconv.FormatInt(itemID, 10) + ".jpg"
	filePath := filepath.Join(sessionDir, fileName)

	f, err := os.Create(filePath)
	if err == nil {
		jpeg.Encode(f, img, &jpeg.Options{Quality: 95})
		f.Close()
	}

	thumbName := fileName
	thumbPath := filepath.Join(thumbDir, thumbName)
	const thumbSize = 128
	origW := img.Bounds().Dx()
	origH := img.Bounds().Dy()
	if origW > thumbSize || origH > thumbSize {
		ratio := math.Min(float64(thumbSize)/float64(origW), float64(thumbSize)/float64(origH))
		tw := int(float64(origW) * ratio)
		th := int(float64(origH) * ratio)
		if tw < 1 {
			tw = 1
		}
		if th < 1 {
			th = 1
		}
		dst := image.NewRGBA(image.Rect(0, 0, tw, th))
		xdraw.CatmullRom.Scale(dst, dst.Bounds(), img, img.Bounds(), xdraw.Over, nil)
		tf, err := os.Create(thumbPath)
		if err == nil {
			jpeg.Encode(tf, dst, &jpeg.Options{Quality: 80})
			tf.Close()
		}
	} else {
		srcF, _ := os.Open(filePath)
		dstF, _ := os.Create(thumbPath)
		if srcF != nil && dstF != nil {
			io.Copy(dstF, srcF)
		}
		if srcF != nil {
			srcF.Close()
		}
		if dstF != nil {
			dstF.Close()
		}
	}

	s.db.UpdateSessionItemPaths(itemID, fileName, thumbName)
	s.db.SetActiveItem(itemID, sessionID)

	s.emit.Emit("session:added", map[string]int64{"id": itemID})
	s.emit.Emit("session:active", map[string]int64{"id": itemID})

	return itemID
}

func (s *Service) CreateSession(name string) (*preset.SessionInfo, error) {
	si, err := s.db.CreateSession(name)
	if err != nil {
		return nil, err
	}
	if err := s.db.SetActiveSession(si.ID); err != nil {
		return nil, err
	}
	s.emit.Emit("session:created", map[string]any{"session_id": si.ID, "name": si.Name})
	s.emit.Emit("session:switched", map[string]int64{"session_id": si.ID})
	return si, nil
}

func (s *Service) ListSessions() ([]preset.SessionInfo, error) {
	return s.db.ListSessions()
}

func (s *Service) SwitchSession(id int64) error {
	if err := s.db.SetActiveSession(id); err != nil {
		return err
	}
	s.emit.Emit("session:switched", map[string]int64{"session_id": id})
	return nil
}

func (s *Service) RenameSession(id int64, name string) error {
	return s.db.RenameSession(id, name)
}

func (s *Service) DeleteSession(id int64) error {
	sessions, err := s.db.ListSessions()
	if err != nil {
		return err
	}
	if len(sessions) <= 1 {
		return fmt.Errorf("cannot delete the last session")
	}

	activeID, _ := s.db.GetActiveSessionID()
	if err := s.db.DeleteSession(id); err != nil {
		return err
	}

	sessionDir := filepath.Join(s.dataDir, "sessions", strconv.FormatInt(id, 10))
	thumbDir := filepath.Join(s.dataDir, "thumbs", strconv.FormatInt(id, 10))
	os.RemoveAll(sessionDir)
	os.RemoveAll(thumbDir)

	if activeID == id {
		for _, si := range sessions {
			if si.ID != id {
				s.db.SetActiveSession(si.ID)
				break
			}
		}
	}

	s.emit.Emit("session:deleted", map[string]int64{"session_id": id})
	return nil
}

func (s *Service) GetSessionItems() ([]preset.SessionItem, error) {
	sessionID, err := s.db.GetActiveSessionID()
	if err != nil {
		return nil, err
	}
	if sessionID == 0 {
		return []preset.SessionItem{}, nil
	}
	return s.db.GetSessionItems(sessionID)
}

func (s *Service) GetActiveSessionItem() (*preset.SessionItem, error) {
	sessionID, err := s.db.GetActiveSessionID()
	if err != nil {
		return nil, err
	}
	if sessionID == 0 {
		return nil, nil
	}
	return s.db.GetActiveItem(sessionID)
}

func (s *Service) SetActiveSessionItem(id int64) error {
	sessionID, err := s.db.GetActiveSessionID()
	if err != nil {
		return err
	}
	if err := s.db.SetActiveItem(id, sessionID); err != nil {
		return err
	}
	s.emit.Emit("session:active", map[string]int64{"id": id})
	return nil
}

func (s *Service) DeleteSessionItem(id int64) error {
	item, err := s.db.GetSessionItem(id)
	if err != nil {
		return err
	}
	if item == nil {
		return nil
	}
	sessionDir := filepath.Join(s.dataDir, "sessions", strconv.FormatInt(item.SessionID, 10))
	thumbDir := filepath.Join(s.dataDir, "thumbs", strconv.FormatInt(item.SessionID, 10))
	if item.FileName != "" {
		os.Remove(filepath.Join(sessionDir, item.FileName))
	}
	if item.ThumbName != "" {
		os.Remove(filepath.Join(thumbDir, item.ThumbName))
	}
	if err := s.db.DeleteSessionItem(id); err != nil {
		return err
	}
	s.emit.Emit("session:removed", map[string]int64{"id": id})
	return nil
}

func (s *Service) ClearSession() error {
	sessionID, err := s.db.GetActiveSessionID()
	if err != nil {
		return err
	}
	if sessionID == 0 {
		return nil
	}
	sessionDir := filepath.Join(s.dataDir, "sessions", strconv.FormatInt(sessionID, 10))
	thumbDir := filepath.Join(s.dataDir, "thumbs", strconv.FormatInt(sessionID, 10))
	os.RemoveAll(sessionDir)
	os.RemoveAll(thumbDir)
	os.MkdirAll(sessionDir, 0o755)
	os.MkdirAll(thumbDir, 0o755)
	if err := s.db.ClearSessionItems(sessionID); err != nil {
		return err
	}
	s.emit.Emit("session:cleared")
	return nil
}

func (s *Service) GetSessionImage(id int64) (string, error) {
	item, err := s.db.GetSessionItem(id)
	if err != nil || item == nil {
		return "", fmt.Errorf("session item not found")
	}
	sessionDir := filepath.Join(s.dataDir, "sessions", strconv.FormatInt(item.SessionID, 10))
	data, err := os.ReadFile(filepath.Join(sessionDir, item.FileName))
	if err != nil {
		return "", fmt.Errorf("read image: %w", err)
	}
	return base64.StdEncoding.EncodeToString(data), nil
}

func (s *Service) GetSessionThumb(id int64) (string, error) {
	item, err := s.db.GetSessionItem(id)
	if err != nil || item == nil {
		return "", fmt.Errorf("session item not found")
	}
	thumbDir := filepath.Join(s.dataDir, "thumbs", strconv.FormatInt(item.SessionID, 10))
	data, err := os.ReadFile(filepath.Join(thumbDir, item.ThumbName))
	if err != nil {
		return "", fmt.Errorf("read thumbnail: %w", err)
	}
	return base64.StdEncoding.EncodeToString(data), nil
}

func (s *Service) HasSessionItems() (bool, error) {
	return s.db.HasAnyItems()
}

func (s *Service) ConfirmClose(action string) {
	if action == "discard" {
		sessions, _ := s.db.ListSessions()
		for _, si := range sessions {
			sessionDir := filepath.Join(s.dataDir, "sessions", strconv.FormatInt(si.ID, 10))
			thumbDir := filepath.Join(s.dataDir, "thumbs", strconv.FormatInt(si.ID, 10))
			os.RemoveAll(sessionDir)
			os.RemoveAll(thumbDir)
		}
		s.db.DeleteAllSessions()
	}
}
