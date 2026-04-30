//go:build !cgo

package main

import (
	"bytes"
	"image"
	"image/png"
)

func encodeWebp(img image.Image, quality int) ([]byte, error) {
	var buf bytes.Buffer
	err := png.Encode(&buf, img)
	if err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}
