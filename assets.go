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

//go:embed media/icon.png
var iconBytes []byte

var IconResource = fyne.NewStaticResource("icon.png", iconBytes)
