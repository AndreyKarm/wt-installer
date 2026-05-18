package main

import (
	"fmt"
	"liotom/installer/installer"
	"os"
	"os/exec"
	"path/filepath"

	g "github.com/AllenDang/giu"
)

func InstalledPage() []g.Widget {
	var widgets []g.Widget

	config, err := installer.LoadConfig()
	if err != nil {
		return append(widgets, g.Label(fmt.Sprintf("Config error: %v", err)))
	}

	entries, err := os.ReadDir(config.UserSkins)
	if err != nil {
		return append(
			widgets,
			g.Label(fmt.Sprintf("Folder read error: %v", err)),
		)
	}

	rows := []*g.TreeTableRowWidget{}
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}

		name := entry.Name()
		directory := filepath.Join(config.UserSkins, name)

		rows = append(rows, g.TreeTableRow(
			name,
			g.ContextMenu().Layout(
				g.Selectable("Open Folder").OnClick(func() {
					exec.Command("explorer", directory).Start()
				}),
				g.Selectable("Delete").OnClick(func() {
					installer.DeleteSkin(name)
				}),
			),
		))
	}

	return []g.Widget{
		g.TreeTable().
			Columns(g.TableColumn("Name")).
			Rows(
				rows...,
			),
	}
}
