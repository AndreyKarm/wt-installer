package main

import (
	"bytes"
	"fmt"
	"image"
	"liotom/installer/installer"
	"liotom/installer/wtlive"

	g "github.com/AllenDang/giu"
)

var wnd *g.MasterWindow

func main() {
	// Set favicon
	if img, _, err := image.Decode(bytes.NewReader(faviconBytes)); err == nil {
		rgba = g.ImageToRgba(img)
	} else {
		fmt.Println("Error decoding embedded favicon:", err)
	}

	wnd = g.NewMasterWindow("WTLive Installer", 1200, 900, 0)

	// Set font
	g.Context.FontAtlas.SetDefaultFontSize(16)
	g.Context.FontAtlas.SetDefaultFontFromBytes(skyquakeFontBytes, 16)

	// Set icon
	if rgba != nil {
		g.EnqueueNewTextureFromRgba(rgba, func(t *g.Texture) {
			tex = t
		})
		wnd.SetIcon(rgba)
	}

	// Set fallback texture
	if img, _, err := image.Decode(bytes.NewReader(fallbackBytes)); err == nil {
		g.EnqueueNewTextureFromRgba(g.ImageToRgba(img), func(t *g.Texture) {
			FallbackTex = t
		})
	}

	// Load config
	cfg, err := installer.LoadConfig()
	if err != nil {
		fmt.Println("Error loading config, using defaults:", err)
		cfg = installer.GetDefaultConfig()
	}
	installer.CurrentConfig = cfg
	installer.SkinPathInput = installer.CurrentConfig.UserSkins

	// Fetch filters
	go func() {
		data, err := wtlive.GetFiltersFromAPI(wtlive.Criteria)
		if err != nil {
			fmt.Println("Error fetching filters:", err)
			return
		}
		wtlive.Filters = *data
		g.Update()
	}()

	// Fetch data
	go wtlive.OnRequestData()

	wnd.Run(loop)
}
