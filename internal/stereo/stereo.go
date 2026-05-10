package stereo

import (
	"encoding/base64"
	"fmt"
	"image"
	"image/color"
	_ "image/jpeg"
	"image/png"
	_ "image/png"
	"math"
	"strings"
)

type Format string

const (
	SideBySide Format = "side-by-side"
	AnaglyphRC Format = "anaglyph-rc"
)

type Params struct {
	ImageBase64 string
	DepthBase64 string
	Format      Format
	EyeShift    float64
}

func GenerateStereo(p Params) (string, error) {
	img, err := decodeBase64(p.ImageBase64)
	if err != nil {
		return "", fmt.Errorf("decode image: %w", err)
	}
	depth, err := decodeBase64(p.DepthBase64)
	if err != nil {
		return "", fmt.Errorf("decode depth: %w", err)
	}

	if p.EyeShift <= 0 {
		p.EyeShift = 8.0
	}

	depthGray := toGray(depth)
	normalizeDepth(depthGray)

	switch p.Format {
	case SideBySide:
		return generateSideBySide(img, depthGray, p.EyeShift)
	case AnaglyphRC:
		return generateAnaglyph(img, depthGray, p.EyeShift)
	default:
		return generateSideBySide(img, depthGray, p.EyeShift)
	}
}

func generateSideBySide(img image.Image, depth [][]float64, shift float64) (string, error) {
	bounds := img.Bounds()
	w, h := bounds.Dx(), bounds.Dy()
	sbsWidth := w * 2
	result := image.NewRGBA(image.Rect(0, 0, sbsWidth, h))

	shiftPixels := shift
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			d := 0.0
			if y < len(depth) && x < len(depth[y]) {
				d = depth[y][x]
			}
			offset := int(d * shiftPixels)

			srcColor := img.At(bounds.Min.X+x, bounds.Min.Y+y)

			// Left eye: shift left
			lx := x - offset
			if lx >= 0 && lx < w {
				result.Set(lx, y, srcColor)
			}

			// Right eye: shift right
			rx := w + x + offset
			if rx >= w && rx < sbsWidth {
				result.Set(rx, y, srcColor)
			}
		}
	}

	return encodePNG(result), nil
}

func generateAnaglyph(img image.Image, depth [][]float64, shift float64) (string, error) {
	bounds := img.Bounds()
	w, h := bounds.Dx(), bounds.Dy()
	result := image.NewRGBA(bounds)

	shiftPixels := shift
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			d := 0.0
			if y < len(depth) && x < len(depth[y]) {
				d = depth[y][x]
			}
			offset := int(d * shiftPixels)

			// Left eye (red channel): shift left
			lx := x - offset
			r := 0.0
			if lx >= 0 && lx < w {
				lr, _, _, _ := img.At(bounds.Min.X+lx, bounds.Min.Y+y).RGBA()
				r = float64(lr) / 257.0
			}

			// Right eye (cyan channels): shift right
			rx := x + offset
			var g, b float64
			if rx >= 0 && rx < w {
				_, rg, rb, _ := img.At(bounds.Min.X+rx, bounds.Min.Y+y).RGBA()
				g = float64(rg) / 257.0
				b = float64(rb) / 257.0
			}

			result.Set(bounds.Min.X+x, bounds.Min.Y+y, color.NRGBA{
				R: uint8(clamp(r)),
				G: uint8(clamp(g)),
				B: uint8(clamp(b)),
				A: 255,
			})
		}
	}

	return encodePNG(result), nil
}

func toGray(img image.Image) [][]float64 {
	bounds := img.Bounds()
	w, h := bounds.Dx(), bounds.Dy()
	gray := make([][]float64, h)
	for y := 0; y < h; y++ {
		gray[y] = make([]float64, w)
		for x := 0; x < w; x++ {
			c := color.GrayModel.Convert(img.At(bounds.Min.X+x, bounds.Min.Y+y)).(color.Gray)
			gray[y][x] = float64(c.Y) / 255.0
		}
	}
	return gray
}

func normalizeDepth(depth [][]float64) {
	min, max := 1.0, 0.0
	for _, row := range depth {
		for _, v := range row {
			if v < min {
				min = v
			}
			if v > max {
				max = v
			}
		}
	}
	if max == min {
		return
	}
	range_ := max - min
	for y, row := range depth {
		for x, v := range row {
			depth[y][x] = (v - min) / range_
		}
	}
}

func decodeBase64(b64 string) (image.Image, error) {
	b64 = strings.TrimPrefix(b64, "data:image/png;base64,")
	b64 = strings.TrimPrefix(b64, "data:image/jpeg;base64,")
	data, err := base64.StdEncoding.DecodeString(b64)
	if err != nil {
		return nil, err
	}
	img, _, err := image.Decode(strings.NewReader(string(data)))
	return img, err
}

func encodePNG(img image.Image) string {
	var buf strings.Builder
	png.Encode(&writerAdapter{&buf}, img)
	return base64.StdEncoding.EncodeToString([]byte(buf.String()))
}

type writerAdapter struct {
	*strings.Builder
}

func clamp(v float64) uint8 {
	return uint8(math.Max(0, math.Min(255, v)))
}
