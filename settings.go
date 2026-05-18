package main

import (
	"fmt"
	"liotom/installer/installer"

	g "github.com/AllenDang/giu"
)

func SettingsPage() []g.Widget {
	return []g.Widget{
		g.Label("Configuration"),
		g.InputText(&installer.SkinPathInput),
		g.Button("Save Settings").OnClick(func() {
			installer.CurrentConfig.UserSkins = installer.SkinPathInput
			if err := installer.SaveConfig(installer.CurrentConfig); err != nil {
				fmt.Println("Error saving config:", err)
			} else {
				fmt.Println("Settings saved!")
			}
		}),
	}
}
