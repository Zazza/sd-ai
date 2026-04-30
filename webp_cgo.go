//go:build cgo

package main

import (
	"bytes"
	"image"

	"github.com/chai2010/webp"
)

func encodeWebp(img image.Image, quality int) ([]byte, error) {
	var buf bytes.Buffer
	err := webp.Encode(&buf, img, &webp.Options{Quality: float32(quality)})
	if err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}
