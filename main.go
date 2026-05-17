package main

import (
	"fmt"

	g "github.com/AllenDang/giu"
)

var wnd *g.MasterWindow

func main() {
	var err error
	rgba, err = g.LoadImage("./media/favicon.png")
	if err != nil {
		fmt.Println("Error loading fallback image:", err)
	}

	wnd = g.NewMasterWindow(
		"WTLive Installer", 1200, 900, 0,
	)

	if rgba != nil {
		g.EnqueueNewTextureFromRgba(rgba, func(t *g.Texture) {
			tex = t
		})
	}

	wnd.SetIcon(rgba)

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
