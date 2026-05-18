package main

import (
	g "github.com/AllenDang/giu"
)

func loop() {
	g.SingleWindow().Layout(
		g.TabBar().TabItems(
			g.TabItem("Download").Layout(
				g.Child().Layout(
					g.Column(DownloadPage()...),
				),
			),

			g.TabItem("Manage Skins").Layout(
				g.Child().Layout(
					g.Column(InstalledPage()...),
				),
			),

			g.TabItem("Settings").Layout(
				g.Child().Layout(SettingsPage()...),
			),
		),
	)
}
