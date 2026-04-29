package compositor

import (
	"image"
	"image/color"
	"image/draw"
	"math"
)

func RemoveWhiteBackground(img image.Image) *image.RGBA {
	bounds := img.Bounds()
	w, h := bounds.Dx(), bounds.Dy()

	removed := make([][]bool, h)
	for y := 0; y < h; y++ {
		removed[y] = make([]bool, w)
	}

	visited := make([][]bool, h)
	for y := 0; y < h; y++ {
		visited[y] = make([]bool, w)
	}

	queue := make([][2]int, 0, w*2+h*2)

	for x := 0; x < w; x++ {
		if isNearWhite(img, x, 0, bounds.Min) {
			queue = append(queue, [2]int{0, x})
		}
		if isNearWhite(img, x, h-1, bounds.Min) {
			queue = append(queue, [2]int{h - 1, x})
		}
	}
	for y := 1; y < h-1; y++ {
		if isNearWhite(img, 0, y, bounds.Min) {
			queue = append(queue, [2]int{y, 0})
		}
		if isNearWhite(img, w-1, y, bounds.Min) {
			queue = append(queue, [2]int{y, w - 1})
		}
	}

	for len(queue) > 0 {
		cy, cx := queue[0][0], queue[0][1]
		queue = queue[1:]

		if visited[cy][cx] {
			continue
		}
		visited[cy][cx] = true

		if !isNearWhite(img, cx, cy, bounds.Min) {
			continue
		}

		removed[cy][cx] = true

		dirs := [4][2]int{{-1, 0}, {1, 0}, {0, -1}, {0, 1}}
		for _, d := range dirs {
			ny, nx := cy+d[0], cx+d[1]
			if ny >= 0 && ny < h && nx >= 0 && nx < w && !visited[ny][nx] {
				queue = append(queue, [2]int{ny, nx})
			}
		}
	}

	dist := computeDistanceField(removed, w, h)

	const featherRadius = 6.0
	alphaMask := make([][]float64, h)
	for y := 0; y < h; y++ {
		alphaMask[y] = make([]float64, w)
		for x := 0; x < w; x++ {
			if removed[y][x] {
				alphaMask[y][x] = 0
			} else {
				d := dist[y][x]
				if d < featherRadius {
					alphaMask[y][x] = d / featherRadius
				} else {
					alphaMask[y][x] = 1.0
				}
			}
		}
	}

	smoothed := blurMask(alphaMask, w, h, 3)

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

func isNearWhite(img image.Image, x, y int, min image.Point) bool {
	c := color.NRGBAModel.Convert(img.At(x+min.X, y+min.Y)).(color.NRGBA)
	r, g, b := float64(c.R), float64(c.G), float64(c.B)
	mx := math.Max(r, math.Max(g, b))
	mn := math.Min(r, math.Min(g, b))
	lightness := (mx + mn) / 2.0
	saturation := 0.0
	if mx > 0 {
		saturation = (mx - mn) / mx
	}
	return lightness > 200 && saturation < 0.20
}

func computeDistanceField(removed [][]bool, w, h int) [][]float64 {
	dist := make([][]float64, h)
	for y := 0; y < h; y++ {
		dist[y] = make([]float64, w)
		for x := 0; x < w; x++ {
			if removed[y][x] {
				dist[y][x] = 0
			} else {
				dist[y][x] = 1e9
			}
		}
	}

	for y := 1; y < h; y++ {
		for x := 1; x < w; x++ {
			d := math.Min(dist[y-1][x]+1, dist[y][x-1]+1)
			if d < dist[y][x] {
				dist[y][x] = d
			}
		}
	}
	for y := h - 2; y >= 0; y-- {
		for x := w - 2; x >= 0; x-- {
			d := math.Min(dist[y+1][x]+1, dist[y][x+1]+1)
			if d < dist[y][x] {
				dist[y][x] = d
			}
		}
	}

	return dist
}

func blurMask(mask [][]float64, w, h, radius int) [][]float64 {
	result := make([][]float64, h)
	for y := 0; y < h; y++ {
		result[y] = make([]float64, w)
		for x := 0; x < w; x++ {
			sum := 0.0
			weight := 0.0
			for dy := -radius; dy <= radius; dy++ {
				for dx := -radius; dx <= radius; dx++ {
					ny, nx := y+dy, x+dx
					if ny >= 0 && ny < h && nx >= 0 && nx < w {
						d := math.Sqrt(float64(dx*dx + dy*dy))
						g := math.Exp(-d * d / (2.0 * float64(radius) * float64(radius)))
						sum += mask[ny][nx] * g
						weight += g
					}
				}
			}
			if weight > 0 {
				result[y][x] = sum / weight
			}
		}
	}
	return result
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

func computeDifferenceMask(bg image.Image, fg image.Image) [][]float64 {
	bounds := bg.Bounds()
	w, h := bounds.Dx(), bounds.Dy()
	mask := make([][]float64, h)

	for y := 0; y < h; y++ {
		mask[y] = make([]float64, w)
		for x := 0; x < w; x++ {
			br, bg_, bb, _ := bg.At(x+bounds.Min.X, y+bounds.Min.Y).RGBA()
			fr, fg_, fb, _ := fg.At(x+bounds.Min.X, y+bounds.Min.Y).RGBA()

			dr := float64(int(br>>8)) - float64(int(fr>>8))
			dg := float64(int(bg_>>8)) - float64(int(fg_>>8))
			db := float64(int(bb>>8)) - float64(int(fb>>8))

			dist := math.Sqrt(dr*dr + dg*dg + db*db)

			const threshold = 20.0
			const feather = 15.0
			alpha := (dist - threshold) / feather
			if alpha < 0 {
				alpha = 0
			}
			if alpha > 1 {
				alpha = 1
			}
			mask[y][x] = alpha
		}
	}

	return blurMask(mask, w, h, 4)
}

func compositeDifference(dst *image.RGBA, src image.Image, mask [][]float64) {
	bounds := dst.Bounds()

	layer := image.NewRGBA(bounds)
	for y := bounds.Min.Y; y < bounds.Max.Y; y++ {
		for x := bounds.Min.X; x < bounds.Max.X; x++ {
			a := mask[y-bounds.Min.Y][x-bounds.Min.X]
			if a < 0.01 {
				continue
			}
			r, g, b, _ := src.At(x, y).RGBA()
			layer.SetRGBA(x, y, color.RGBA{
				R: uint8(r >> 8),
				G: uint8(g >> 8),
				B: uint8(b >> 8),
				A: uint8(a * 255),
			})
		}
	}

	draw.Draw(dst, bounds, layer, bounds.Min, draw.Over)
}
