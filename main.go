package main

import (
	"bytes"
	"embed"
	"fmt"
	"image"
	"image/png"

	g "github.com/AllenDang/giu"
)

var (
	wnd     *g.MasterWindow
	mediaFS embed.FS
)

func main() {
	var err error
	rgba, err = decodeEmbeddedRGBA(faviconBytes)
	if err != nil {
		fmt.Println("Error decoding embedded favicon:", err)
	}

	wnd = g.NewMasterWindow(
		"WTLive Installer", 1200, 900, 0,
	)

	if rgba != nil {
		g.EnqueueNewTextureFromRgba(rgba, func(t *g.Texture) {
			tex = t
		})
		wnd.SetIcon(rgba)
	}

	fallbackRGBA, err := decodeEmbeddedRGBA(fallbackBytes)
	if err == nil {
		g.EnqueueNewTextureFromRgba(fallbackRGBA, func(t *g.Texture) {
			fallbackTex = t
		})
	}

	cfg, err := LoadConfig()
	if err != nil {
		fmt.Println("Error loading config, using defaults:", err)
		cfg = GetDefaultConfig()
	}
	currentConfig = cfg
	skinPathInput = currentConfig.UserSkins

	go func() {
		data, err := GetFiltersFromAPI(criteria)
		if err != nil {
			fmt.Println("Error fetching filters:", err)
			return
		}
		filters = *data
		fmt.Println("Filters loaded!")
		g.Update()
	}()

	go OnRequestData()

	wnd.Run(loop)
}

func loadEmbeddedImage(path string) (*image.RGBA, error) {
	data, err := mediaFS.ReadFile(path)
	if err != nil {
		return nil, err
	}
	img, err := png.Decode(bytes.NewReader(data))
	if err != nil {
		return nil, err
	}
	rgba := image.NewRGBA(img.Bounds())
	return rgba, nil
}
