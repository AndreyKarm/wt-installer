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
	wnd = g.NewMasterWindow("WTLive Installer", 1200, 900, 0)
	loadMedia()
	setupWindow()
	loadConfig()
	fetchData()
	wnd.Run(loop)
}

func loadMedia() {
	if img, _, err := image.Decode(bytes.NewReader(faviconBytes)); err == nil {
		rgba = g.ImageToRgba(img)
	} else {
		fmt.Println("Error decoding embedded favicon:", err)
	}

	if img, _, err := image.Decode(bytes.NewReader(fallbackBytes)); err == nil {
		g.EnqueueNewTextureFromRgba(g.ImageToRgba(img), func(t *g.Texture) {
			FallbackTex = t
		})
	}
}

func setupWindow() {
	g.Context.FontAtlas.SetDefaultFontSize(16)
	g.Context.FontAtlas.SetDefaultFontFromBytes(skyquakeFontBytes, 16)

	if rgba != nil {
		g.EnqueueNewTextureFromRgba(rgba, func(t *g.Texture) {
			tex = t
		})
		wnd.SetIcon(rgba)
	}
}

func loadConfig() {
	cfg, err := installer.LoadConfig()
	if err != nil {
		fmt.Println("Error loading config, using defaults:", err)
		cfg = installer.GetDefaultConfig()
	}
	installer.CurrentConfig = cfg
	installer.SkinPathInput = installer.CurrentConfig.UserSkins
}

func fetchData() {
	go func() {
		data, err := wtlive.GetFiltersFromAPI(wtlive.Criteria)
		if err != nil {
			fmt.Println("Error fetching filters:", err)
			return
		}
		wtlive.Filters = *data
		g.Update()
	}()

	go wtlive.OnRequestData()
}
