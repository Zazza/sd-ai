package mcp

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"image"
	"image/draw"
	"image/png"
	"math"
	"os"
	"path/filepath"
	"strings"

	xdraw "golang.org/x/image/draw"
	"golang.org/x/image/font"
	"golang.org/x/image/font/basicfont"
	"golang.org/x/image/math/fixed"

	"go-sd/internal/filebrowser"
)

func (s *Server) saveArtifact(name, slug, imgB64 string, info json.RawMessage) (string, error) {
	if imgB64 == "" {
		return "", fmt.Errorf("no image data")
	}
	base := slug
	if base == "" {
		base = name
	}
	base = strings.TrimSpace(filebrowser.SanitizeFilename(base))
	if base == "" {
		base = "image"
	}
	if err := os.MkdirAll(s.deps.OutDir, 0o700); err != nil {
		return "", fmt.Errorf("create out dir: %w", err)
	}

	pngPath := filepath.Join(s.deps.OutDir, base+".png")
	stem := base
	for i := 1; ; i++ {
		if _, err := os.Stat(pngPath); os.IsNotExist(err) {
			break
		}
		stem = fmt.Sprintf("%s_%d", base, i)
		pngPath = filepath.Join(s.deps.OutDir, stem+".png")
	}

	data, err := base64.StdEncoding.DecodeString(imgB64)
	if err != nil {
		return "", fmt.Errorf("decode image: %w", err)
	}
	if err := os.WriteFile(pngPath, data, 0o600); err != nil {
		return "", fmt.Errorf("write image: %w", err)
	}

	if len(info) > 0 {
		var pretty bytes.Buffer
		if err := json.Indent(&pretty, info, "", "  "); err != nil {
			pretty.Reset()
			pretty.Write(info)
		}
		if err := os.WriteFile(filepath.Join(s.deps.OutDir, stem+".json"), pretty.Bytes(), 0o600); err != nil {
			return pngPath, fmt.Errorf("write sidecar (image saved at %s): %w", pngPath, err)
		}
	}
	return pngPath, nil
}

func sidecarInfoPath(pngPath string) string {
	return strings.TrimSuffix(pngPath, filepath.Ext(pngPath)) + ".json"
}

func buildContactSheet(paths, labels []string, out string) error {
	if len(paths) == 0 {
		return fmt.Errorf("no images for contact sheet")
	}
	if len(labels) < len(paths) {
		padded := make([]string, len(paths))
		copy(padded, labels)
		labels = padded
	}

	const thumb = 384
	const labelH = 20

	cols := int(math.Ceil(math.Sqrt(float64(len(paths)))))
	if cols < 1 {
		cols = 1
	}
	rows := (len(paths) + cols - 1) / cols

	canvas := image.NewRGBA(image.Rect(0, 0, cols*thumb, rows*(thumb+labelH)))
	draw.Draw(canvas, canvas.Bounds(), image.NewUniform(image.White), image.Point{}, draw.Src)

	for idx, p := range paths {
		data, err := os.ReadFile(p)
		if err != nil {
			continue
		}
		img, _, err := image.Decode(bytes.NewReader(data))
		if err != nil {
			continue
		}
		b := img.Bounds()
		ratio := math.Min(float64(thumb)/float64(b.Dx()), float64(thumb)/float64(b.Dy()))
		tw := int(float64(b.Dx()) * ratio)
		th := int(float64(b.Dy()) * ratio)
		if tw < 1 {
			tw = 1
		}
		if th < 1 {
			th = 1
		}
		dst := image.NewRGBA(image.Rect(0, 0, tw, th))
		xdraw.CatmullRom.Scale(dst, dst.Bounds(), img, b, xdraw.Over, nil)

		cellX := (idx % cols) * thumb
		cellY := (idx / cols) * (thumb + labelH)
		offX := cellX + (thumb-tw)/2
		offY := cellY + (thumb-th)/2
		draw.Draw(canvas, image.Rect(offX, offY, offX+tw, offY+th), dst, image.Point{}, draw.Over)

		label := labels[idx]
		if len(label) > 60 {
			label = label[:60]
		}
		d := &font.Drawer{
			Dst:  canvas,
			Src:  image.Black,
			Face: basicfont.Face7x13,
			Dot:  fixed.P(cellX+4, cellY+thumb+14),
		}
		d.DrawString(label)
	}

	f, err := os.OpenFile(out, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0o600)
	if err != nil {
		return fmt.Errorf("create contact sheet: %w", err)
	}
	defer f.Close()
	if err := png.Encode(f, canvas); err != nil {
		return fmt.Errorf("encode contact sheet: %w", err)
	}
	return nil
}
