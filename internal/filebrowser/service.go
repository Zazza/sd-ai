package filebrowser

import (
	"bytes"
	"encoding/base64"
	"fmt"
	"image"
	"image/jpeg"
	"image/png"
	"math"
	"os"
	"path/filepath"
	"strings"

	xdraw "golang.org/x/image/draw"
)

var imageExts = map[string]bool{
	".png":  true,
	".jpg":  true,
	".jpeg": true,
	".webp": true,
}

type FileEntry struct {
	Name    string `json:"name"`
	Path    string `json:"path"`
	IsDir   bool   `json:"is_dir"`
	Size    int64  `json:"size"`
	ModTime string `json:"mod_time"`
}

func BrowseDirectory(dirPath string) ([]FileEntry, error) {
	if dirPath == "" {
		return []FileEntry{}, nil
	}
	entries, err := os.ReadDir(dirPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read directory: %w", err)
	}
	var result []FileEntry
	for _, e := range entries {
		name := e.Name()
		ext := strings.ToLower(filepath.Ext(name))
		if e.IsDir() {
			result = append(result, FileEntry{
				Name:  name,
				Path:  filepath.Join(dirPath, name),
				IsDir: true,
			})
			continue
		}
		if !imageExts[ext] {
			continue
		}
		info, err := e.Info()
		if err != nil {
			continue
		}
		result = append(result, FileEntry{
			Name:    name,
			Path:    filepath.Join(dirPath, name),
			IsDir:   false,
			Size:    info.Size(),
			ModTime: info.ModTime().Format("2006-01-02 15:04"),
		})
	}
	return result, nil
}

func ReadFileAsBase64(filePath string) (string, error) {
	if filePath == "" {
		return "", nil
	}
	ext := strings.ToLower(filepath.Ext(filePath))
	if !imageExts[ext] {
		return "", fmt.Errorf("unsupported file type: %s", ext)
	}
	info, err := os.Stat(filePath)
	if err != nil {
		return "", fmt.Errorf("failed to read file: %w", err)
	}
	if info.Size() > 16*1024*1024 {
		return "", fmt.Errorf("image too large (max 16 MB)")
	}
	data, err := os.ReadFile(filePath)
	if err != nil {
		return "", fmt.Errorf("failed to read file: %w", err)
	}
	return base64.StdEncoding.EncodeToString(data), nil
}

func ReadThumbnail(filePath string) (string, error) {
	if filePath == "" {
		return "", nil
	}
	ext := strings.ToLower(filepath.Ext(filePath))
	if !imageExts[ext] {
		return "", fmt.Errorf("unsupported file type: %s", ext)
	}
	info, err := os.Stat(filePath)
	if err != nil {
		return "", fmt.Errorf("failed to read file: %w", err)
	}
	if info.Size() > 16*1024*1024 {
		return "", fmt.Errorf("image too large (max 16 MB)")
	}
	data, err := os.ReadFile(filePath)
	if err != nil {
		return "", fmt.Errorf("failed to read file: %w", err)
	}

	img, _, err := image.Decode(bytes.NewReader(data))
	if err != nil {
		return "", fmt.Errorf("decode image: %w", err)
	}

	const thumbSize = 256
	origW := img.Bounds().Dx()
	origH := img.Bounds().Dy()
	if origW <= thumbSize && origH <= thumbSize {
		return base64.StdEncoding.EncodeToString(data), nil
	}

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

	var buf bytes.Buffer
	switch ext {
	case ".jpg", ".jpeg":
		err = jpeg.Encode(&buf, dst, &jpeg.Options{Quality: 80})
	case ".webp":
		return "", fmt.Errorf("webp thumbnails not supported in this context")
	default:
		err = png.Encode(&buf, dst)
	}
	if err != nil {
		return "", fmt.Errorf("encode thumbnail: %w", err)
	}

	return base64.StdEncoding.EncodeToString(buf.Bytes()), nil
}

func DecodeImageSize(b64 string) (int, int) {
	data, err := base64.StdEncoding.DecodeString(b64)
	if err != nil {
		return 0, 0
	}
	cfg, _, err := image.DecodeConfig(bytes.NewReader(data))
	if err != nil {
		return 0, 0
	}
	return cfg.Width, cfg.Height
}
