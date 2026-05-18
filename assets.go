package main

import (
	"bytes"
	_ "embed"
	"image"
	_ "image/png"
)

//go:embed media/favicon.png
var faviconBytes []byte

//go:embed media/fallback.png
var fallbackBytes []byte

func decodeEmbeddedRGBA(data []byte) (*image.RGBA, error) {
	img, _, err := image.Decode(bytes.NewReader(data))
	if err != nil {
		return nil, err
	}
	bounds := img.Bounds()
	rgba := image.NewRGBA(bounds)
	for y := bounds.Min.Y; y < bounds.Max.Y; y++ {
		for x := bounds.Min.X; x < bounds.Max.X; x++ {
			rgba.Set(x, y, img.At(x, y))
		}
	}
	return rgba, nil
}
