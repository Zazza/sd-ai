package compositor

import (
	"image"
	"image/color"
	"image/draw"
	"math"
)

func RemoveWhiteBackground(img image.Image, threshold uint8) *image.RGBA {
	bounds := img.Bounds()
	w, h := bounds.Dx(), bounds.Dy()

	alphaMask := make([][]float64, h)
	for y := 0; y < h; y++ {
		alphaMask[y] = make([]float64, w)
		for x := 0; x < w; x++ {
			c := color.NRGBAModel.Convert(img.At(x+bounds.Min.X, y+bounds.Min.Y)).(color.NRGBA)
			lum := 0.299*float64(c.R) + 0.587*float64(c.G) + 0.114*float64(c.B)

			t := float64(threshold)
			feather := 20.0
			alpha := (t - lum) / feather
			if alpha < 0 {
				alpha = 0
			}
			if alpha > 1 {
				alpha = 1
			}
			if c.A < 255 {
				alpha *= float64(c.A) / 255.0
			}
			alphaMask[y][x] = alpha
		}
	}

	const blurRadius = 2
	smoothed := make([][]float64, h)
	for y := 0; y < h; y++ {
		smoothed[y] = make([]float64, w)
		for x := 0; x < w; x++ {
			sum := 0.0
			count := 0.0
			for dy := -blurRadius; dy <= blurRadius; dy++ {
				for dx := -blurRadius; dx <= blurRadius; dx++ {
					ny, nx := y+dy, x+dx
					if ny >= 0 && ny < h && nx >= 0 && nx < w {
						sum += alphaMask[ny][nx]
						count++
					}
				}
			}
			smoothed[y][x] = sum / count
		}
	}

	result := image.NewRGBA(bounds)
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			c := color.NRGBAModel.Convert(img.At(x+bounds.Min.X, y+bounds.Min.Y)).(color.NRGBA)
			a := smoothed[y][x]
			if a < 0.01 {
				continue
			}
			result.SetRGBA(x+bounds.Min.X, y+bounds.Min.Y, color.RGBA{
				R: c.R,
				G: c.G,
				B: c.B,
				A: uint8(a * 255),
			})
		}
	}

	return cropToBoundingBox(result)
}

func cropToBoundingBox(img *image.RGBA) *image.RGBA {
	bounds := img.Bounds()
	minX, minY := bounds.Max.X, bounds.Max.Y
	maxX, maxY := bounds.Min.X, bounds.Min.Y

	found := false
	for y := bounds.Min.Y; y < bounds.Max.Y; y++ {
		for x := bounds.Min.X; x < bounds.Max.X; x++ {
			if img.RGBAAt(x, y).A > 0 {
				found = true
				if x < minX {
					minX = x
				}
				if x > maxX {
					maxX = x
				}
				if y < minY {
					minY = y
				}
				if y > maxY {
					maxY = y
				}
			}
		}
	}

	if !found {
		return img
	}

	cropRect := image.Rect(minX, minY, maxX+1, maxY+1)
	cropped := image.NewRGBA(cropRect)
	draw.Draw(cropped, cropRect, img, cropRect.Min, draw.Src)
	return cropped
}

func CompositeOver(background image.Image, character *image.RGBA, pos Position, scale float64) *image.RGBA {
	bgBounds := background.Bounds()
	result := image.NewRGBA(bgBounds)
	draw.Draw(result, bgBounds, background, bgBounds.Min, draw.Src)

	charBounds := character.Bounds()
	charW := charBounds.Dx()
	charH := charBounds.Dy()

	targetW := int(float64(bgBounds.Dx()) * scale)
	if targetW <= 0 {
		targetW = charW
	}
	targetH := int(float64(targetW) * float64(charH) / float64(charW))
	if targetH <= 0 {
		targetH = charH
	}

	scaled := scaleImage(character, targetW, targetH)

	offsetX := int(pos.X * float64(bgBounds.Dx()))
	offsetY := int(pos.Y * float64(bgBounds.Dy()))
	offsetX -= targetW / 2
	offsetY -= targetH / 2

	sp := image.Pt(
		clamp(-offsetX, 0, targetW),
		clamp(-offsetY, 0, targetH),
	)
	dp := image.Pt(
		clamp(offsetX, 0, bgBounds.Dx()),
		clamp(offsetY, 0, bgBounds.Dy()),
	)
	endX := min(offsetX+targetW, bgBounds.Dx())
	endY := min(offsetY+targetH, bgBounds.Dy())

	drawRect := image.Rect(dp.X, dp.Y, endX, endY)
	if drawRect.Dx() > 0 && drawRect.Dy() > 0 {
		draw.Draw(result, drawRect, scaled, sp, draw.Over)
	}

	return result
}

func scaleImage(src *image.RGBA, w, h int) *image.RGBA {
	dst := image.NewRGBA(image.Rect(0, 0, w, h))
	srcBounds := src.Bounds()
	sx := float64(srcBounds.Dx()) / float64(w)
	sy := float64(srcBounds.Dy()) / float64(h)

	for dy := 0; dy < h; dy++ {
		for dx := 0; dx < w; dx++ {
			srcX := int(math.Round(float64(dx)*sx)) + srcBounds.Min.X
			srcY := int(math.Round(float64(dy)*sy)) + srcBounds.Min.Y
			if srcX < srcBounds.Max.X && srcY < srcBounds.Max.Y {
				dst.SetRGBA(dx, dy, src.RGBAAt(srcX, srcY))
			}
		}
	}
	return dst
}

func clamp(v, lo, hi int) int {
	if v < lo {
		return lo
	}
	if v > hi {
		return hi
	}
	return v
}

func createCharacterMask(bounds image.Rectangle, pos Position, scale float64) *image.RGBA {
	mask := image.NewRGBA(bounds)
	w := bounds.Dx()
	h := bounds.Dy()

	maskW := int(float64(w) * scale)
	maskH := int(float64(maskW) * 1.5)
	if maskH > int(float64(h)*0.9) {
		maskH = int(float64(h) * 0.9)
		maskW = int(float64(maskH) / 1.5)
	}
	if maskW < 64 {
		maskW = 64
	}
	if maskH < 64 {
		maskH = 64
	}

	cx := int(pos.X * float64(w))
	cy := int(pos.Y * float64(h))

	x1 := clamp(cx-maskW/2, 0, w)
	y1 := clamp(cy-maskH/2, 0, h)
	x2 := clamp(cx+maskW/2, 0, w)
	y2 := clamp(cy+maskH/2, 0, h)

	draw.Draw(mask, image.Rect(x1, y1, x2, y2), &image.Uniform{color.White}, image.Point{}, draw.Src)

	return mask
}

func applyMaskedRegion(dst *image.RGBA, src image.Image, mask *image.RGBA) {
	bounds := dst.Bounds()
	for y := bounds.Min.Y; y < bounds.Max.Y; y++ {
		for x := bounds.Min.X; x < bounds.Max.X; x++ {
			if mask.RGBAAt(x, y).A > 128 {
				r, g, b, a := src.At(x, y).RGBA()
				dst.SetRGBA(x, y, color.RGBA{
					R: uint8(r >> 8),
					G: uint8(g >> 8),
					B: uint8(b >> 8),
					A: uint8(a >> 8),
				})
			}
		}
	}
}
