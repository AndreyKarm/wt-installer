package main

import (
	"fmt"

	g "github.com/AllenDang/giu"
)

func SettingsPage() []g.Widget {
	return []g.Widget{
		g.Label("Configuration"),
		g.Row(
			g.Label("User Skins Path"),
			g.InputText(&SkinPathInput),
		),

		g.Row(
			g.Label("Cookies"),
			g.InputText(&Cookies),
		),

		g.Button("Save Settings").OnClick(func() {
			CurrentConfig.UserSkins = SkinPathInput
			if err := SaveConfig(CurrentConfig); err != nil {
				fmt.Println("Error saving config:", err)
			} else {
				fmt.Println("Settings saved!")
			}
		}),
	}
}
