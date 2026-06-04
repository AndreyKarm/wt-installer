package main

import (
	_ "embed"
	"image"
	_ "image/png"

	"fyne.io/fyne/v2"
)

var (
	rgba *image.RGBA
	tex  *image.RGBA
)

//go:embed media/favicon.png
var faviconBytes []byte

var FaviconResource = fyne.NewStaticResource("favicon.png", faviconBytes)
