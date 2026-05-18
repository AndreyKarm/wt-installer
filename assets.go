package main

import (
	_ "embed"
	"image"
	_ "image/png"

	g "github.com/AllenDang/giu"
)

var (
	rgba        *image.RGBA
	tex         *g.Texture
	FallbackTex *g.Texture
)

//go:embed media/skyquake.ttf
var skyquakeFontBytes []byte

//go:embed media/favicon.png
var faviconBytes []byte

//go:embed media/fallback.png
var fallbackBytes []byte
