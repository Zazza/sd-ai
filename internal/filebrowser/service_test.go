package filebrowser

import (
	"bytes"
	"encoding/base64"
	"image"
	"image/jpeg"
	"image/png"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func createPNGFile(t *testing.T, path string, w, h int) {
	t.Helper()
	img := image.NewRGBA(image.Rect(0, 0, w, h))
	var buf bytes.Buffer
	err := png.Encode(&buf, img)
	require.NoError(t, err)
	require.NoError(t, os.WriteFile(path, buf.Bytes(), 0644))
}

func createJPEGFile(t *testing.T, path string, w, h int) {
	t.Helper()
	img := image.NewRGBA(image.Rect(0, 0, w, h))
	var buf bytes.Buffer
	err := jpeg.Encode(&buf, img, &jpeg.Options{Quality: 80})
	require.NoError(t, err)
	require.NoError(t, os.WriteFile(path, buf.Bytes(), 0644))
}

func TestBrowseDirectory_EmptyPath(t *testing.T) {
	t.Parallel()

	result, err := BrowseDirectory("")
	assert.NoError(t, err)
	assert.Equal(t, []FileEntry{}, result)
}

func TestBrowseDirectory_NonExistentDir(t *testing.T) {
	t.Parallel()

	_, err := BrowseDirectory("/nonexistent/dir/that/does/not/exist")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to read directory")
}

func TestBrowseDirectory_EmptyDir(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	result, err := BrowseDirectory(dir)
	assert.NoError(t, err)
	assert.Empty(t, result)
}

func TestBrowseDirectory_WithImageFiles(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	createPNGFile(t, filepath.Join(dir, "photo.png"), 10, 10)
	createJPEGFile(t, filepath.Join(dir, "photo.jpg"), 10, 10)

	result, err := BrowseDirectory(dir)
	require.NoError(t, err)
	assert.Len(t, result, 2)

	names := map[string]bool{}
	for _, entry := range result {
		names[entry.Name] = true
		assert.False(t, entry.IsDir)
		assert.Equal(t, filepath.Join(dir, entry.Name), entry.Path)
		assert.NotEmpty(t, entry.ModTime)
		assert.Greater(t, entry.Size, int64(0))
	}
	assert.True(t, names["photo.png"])
	assert.True(t, names["photo.jpg"])
}

func TestBrowseDirectory_FiltersNonImageFiles(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	createPNGFile(t, filepath.Join(dir, "image.png"), 10, 10)
	require.NoError(t, os.WriteFile(filepath.Join(dir, "document.txt"), []byte("hello"), 0644))
	require.NoError(t, os.WriteFile(filepath.Join(dir, "script.sh"), []byte("#!/bin/bash"), 0644))

	result, err := BrowseDirectory(dir)
	require.NoError(t, err)
	assert.Len(t, result, 1)
	assert.Equal(t, "image.png", result[0].Name)
}

func TestBrowseDirectory_IncludesDirs(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	require.NoError(t, os.Mkdir(filepath.Join(dir, "subdir"), 0755))
	createPNGFile(t, filepath.Join(dir, "img.png"), 10, 10)

	result, err := BrowseDirectory(dir)
	require.NoError(t, err)
	assert.Len(t, result, 2)

	dirFound := false
	imgFound := false
	for _, entry := range result {
		if entry.Name == "subdir" {
			dirFound = true
			assert.True(t, entry.IsDir)
		}
		if entry.Name == "img.png" {
			imgFound = true
			assert.False(t, entry.IsDir)
		}
	}
	assert.True(t, dirFound)
	assert.True(t, imgFound)
}

func TestBrowseDirectory_CaseInsensitiveExtension(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	createPNGFile(t, filepath.Join(dir, "upper.PNG"), 10, 10)
	createJPEGFile(t, filepath.Join(dir, "mixed.JpG"), 10, 10)

	result, err := BrowseDirectory(dir)
	require.NoError(t, err)
	assert.Len(t, result, 2)
}

func TestBrowseDirectory_SupportedExtensions(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		ext  string
	}{
		{name: "png", ext: ".png"},
		{name: "jpg", ext: ".jpg"},
		{name: "jpeg", ext: ".jpeg"},
		{name: "webp", ext: ".webp"},
	}

	dir := t.TempDir()
	for _, tt := range tests {
		if tt.ext == ".webp" {
			require.NoError(t, os.WriteFile(filepath.Join(dir, "image"+tt.ext), []byte("fake"), 0644))
		} else if tt.ext == ".png" {
			createPNGFile(t, filepath.Join(dir, "image"+tt.ext), 10, 10)
		} else {
			createJPEGFile(t, filepath.Join(dir, "image"+tt.ext), 10, 10)
		}
	}

	result, err := BrowseDirectory(dir)
	require.NoError(t, err)
	assert.Len(t, result, 4)
}

func TestReadFileAsBase64_EmptyPath(t *testing.T) {
	t.Parallel()

	result, err := ReadFileAsBase64("")
	assert.NoError(t, err)
	assert.Equal(t, "", result)
}

func TestReadFileAsBase64_UnsupportedExtension(t *testing.T) {
	t.Parallel()

	_, err := ReadFileAsBase64("/some/file.txt")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "unsupported file type")
}

func TestReadFileAsBase64_NonExistentFile(t *testing.T) {
	t.Parallel()

	_, err := ReadFileAsBase64("/nonexistent/image.png")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to read file")
}

func TestReadFileAsBase64_ValidPNG(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	path := filepath.Join(dir, "test.png")
	rawData := []byte("png file content")
	require.NoError(t, os.WriteFile(path, rawData, 0644))

	result, err := ReadFileAsBase64(path)
	require.NoError(t, err)
	assert.Equal(t, base64.StdEncoding.EncodeToString(rawData), result)
}

func TestReadFileAsBase64_ValidJPG(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	path := filepath.Join(dir, "test.jpg")
	rawData := []byte("jpeg file content")
	require.NoError(t, os.WriteFile(path, rawData, 0644))

	result, err := ReadFileAsBase64(path)
	require.NoError(t, err)
	assert.Equal(t, base64.StdEncoding.EncodeToString(rawData), result)
}

func TestReadFileAsBase64_FileTooLarge(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	path := filepath.Join(dir, "big.png")

	f, err := os.Create(path)
	require.NoError(t, err)
	require.NoError(t, f.Truncate(16*1024*1024 + 1))
	require.NoError(t, f.Close())

	_, err = ReadFileAsBase64(path)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "image too large")
}

func TestReadThumbnail_EmptyPath(t *testing.T) {
	t.Parallel()

	result, err := ReadThumbnail("")
	assert.NoError(t, err)
	assert.Equal(t, "", result)
}

func TestReadThumbnail_UnsupportedExtension(t *testing.T) {
	t.Parallel()

	_, err := ReadThumbnail("/some/file.txt")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "unsupported file type")
}

func TestReadThumbnail_NonExistentFile(t *testing.T) {
	t.Parallel()

	_, err := ReadThumbnail("/nonexistent/image.png")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to read file")
}

func TestReadThumbnail_SmallPNG_NoResize(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	path := filepath.Join(dir, "small.png")
	createPNGFile(t, path, 50, 50)

	result, err := ReadThumbnail(path)
	require.NoError(t, err)
	assert.NotEmpty(t, result)

	raw, err := os.ReadFile(path)
	require.NoError(t, err)
	assert.Equal(t, base64.StdEncoding.EncodeToString(raw), result)
}

func TestReadThumbnail_SmallJPEG_NoResize(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	path := filepath.Join(dir, "small.jpg")
	createJPEGFile(t, path, 100, 100)

	result, err := ReadThumbnail(path)
	require.NoError(t, err)
	assert.NotEmpty(t, result)

	raw, err := os.ReadFile(path)
	require.NoError(t, err)
	assert.Equal(t, base64.StdEncoding.EncodeToString(raw), result)
}

func TestReadThumbnail_LargePNG_Resized(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	path := filepath.Join(dir, "large.png")
	createPNGFile(t, path, 1024, 768)

	result, err := ReadThumbnail(path)
	require.NoError(t, err)
	assert.NotEmpty(t, result)

	data, err := base64.StdEncoding.DecodeString(result)
	require.NoError(t, err)

	cfg, _, err := image.DecodeConfig(bytes.NewReader(data))
	require.NoError(t, err)
	assert.LessOrEqual(t, cfg.Width, 256)
	assert.LessOrEqual(t, cfg.Height, 256)
}

func TestReadThumbnail_LargeJPEG_Resized(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	path := filepath.Join(dir, "large.jpeg")
	createJPEGFile(t, path, 800, 600)

	result, err := ReadThumbnail(path)
	require.NoError(t, err)
	assert.NotEmpty(t, result)

	data, err := base64.StdEncoding.DecodeString(result)
	require.NoError(t, err)

	cfg, _, err := image.DecodeConfig(bytes.NewReader(data))
	require.NoError(t, err)
	assert.LessOrEqual(t, cfg.Width, 256)
	assert.LessOrEqual(t, cfg.Height, 256)
}

func TestReadThumbnail_WebPNotSupported(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	path := filepath.Join(dir, "image.webp")
	var buf bytes.Buffer
	img := image.NewRGBA(image.Rect(0, 0, 512, 512))
	require.NoError(t, png.Encode(&buf, img))
	require.NoError(t, os.WriteFile(path, buf.Bytes(), 0644))

	_, err := ReadThumbnail(path)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "webp thumbnails not supported")
}

func TestReadThumbnail_FileTooLarge(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	path := filepath.Join(dir, "big.png")

	f, err := os.Create(path)
	require.NoError(t, err)
	require.NoError(t, f.Truncate(16*1024*1024 + 1))
	require.NoError(t, f.Close())

	_, err = ReadThumbnail(path)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "image too large")
}

func TestReadThumbnail_InvalidImageData(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	path := filepath.Join(dir, "corrupt.png")
	require.NoError(t, os.WriteFile(path, []byte("not a real image"), 0644))

	_, err := ReadThumbnail(path)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "decode image")
}

func TestDecodeImageSize_ValidPNG(t *testing.T) {
	t.Parallel()

	var buf bytes.Buffer
	img := image.NewRGBA(image.Rect(0, 0, 640, 480))
	require.NoError(t, png.Encode(&buf, img))
	b64 := base64.StdEncoding.EncodeToString(buf.Bytes())

	w, h := DecodeImageSize(b64)
	assert.Equal(t, 640, w)
	assert.Equal(t, 480, h)
}

func TestDecodeImageSize_ValidJPEG(t *testing.T) {
	t.Parallel()

	var buf bytes.Buffer
	img := image.NewRGBA(image.Rect(0, 0, 800, 600))
	require.NoError(t, jpeg.Encode(&buf, img, &jpeg.Options{Quality: 80}))
	b64 := base64.StdEncoding.EncodeToString(buf.Bytes())

	w, h := DecodeImageSize(b64)
	assert.Equal(t, 800, w)
	assert.Equal(t, 600, h)
}

func TestDecodeImageSize_EmptyString(t *testing.T) {
	t.Parallel()

	w, h := DecodeImageSize("")
	assert.Equal(t, 0, w)
	assert.Equal(t, 0, h)
}

func TestDecodeImageSize_InvalidBase64(t *testing.T) {
	t.Parallel()

	w, h := DecodeImageSize("!!!not-base64!!!")
	assert.Equal(t, 0, w)
	assert.Equal(t, 0, h)
}

func TestDecodeImageSize_InvalidImageData(t *testing.T) {
	t.Parallel()

	b64 := base64.StdEncoding.EncodeToString([]byte("random garbage data"))

	w, h := DecodeImageSize(b64)
	assert.Equal(t, 0, w)
	assert.Equal(t, 0, h)
}

func TestDecodeImageSize_SquareImage(t *testing.T) {
	t.Parallel()

	var buf bytes.Buffer
	img := image.NewRGBA(image.Rect(0, 0, 512, 512))
	require.NoError(t, png.Encode(&buf, img))
	b64 := base64.StdEncoding.EncodeToString(buf.Bytes())

	w, h := DecodeImageSize(b64)
	assert.Equal(t, 512, w)
	assert.Equal(t, 512, h)
}

func TestDecodeImageSize_OnePixelImage(t *testing.T) {
	t.Parallel()

	var buf bytes.Buffer
	img := image.NewRGBA(image.Rect(0, 0, 1, 1))
	require.NoError(t, png.Encode(&buf, img))
	b64 := base64.StdEncoding.EncodeToString(buf.Bytes())

	w, h := DecodeImageSize(b64)
	assert.Equal(t, 1, w)
	assert.Equal(t, 1, h)
}
