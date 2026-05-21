package main

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	g "github.com/AllenDang/giu"
)

var (
	skinToDelete    string
	openDeletePopup bool
	searchInput     string
)

func InstalledPage() []g.Widget {
	var widgets []g.Widget

	config, err := LoadConfig()
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

	searchTerm := strings.ToLower(searchInput)

	rows := []*g.TreeTableRowWidget{}
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}

		name := entry.Name()

		if searchTerm != "" && !strings.Contains(strings.ToLower(name), searchTerm) {
			continue
		}

		directory := filepath.Join(config.UserSkins, name)

		rows = append(rows, g.TreeTableRow(
			name,
			g.ContextMenu().Layout(
				g.Selectable("Open Folder").OnClick(func() {
					exec.Command("explorer", directory).Start()
				}),
				g.Selectable("Delete").OnClick(func() {
					skinToDelete = name
					openDeletePopup = true
				}),
			),
		))
	}

	return []g.Widget{
		g.Row(
			g.Label("Search"),
			g.InputText(&searchInput).
				Size(400),
		),

		g.TreeTable().
			Columns(g.TableColumn("Name")).
			Rows(
				rows...,
			),

		g.Custom(func() {
			if openDeletePopup {
				g.OpenPopup("Confirm Delete")
				openDeletePopup = false
			}
		}),

		g.PopupModal("Confirm Delete").Layout(
			g.Label("Are you sure you want to delete?"),
			g.Row(
				g.Button("Yes").OnClick(func() {
					DeleteSkin(skinToDelete)
					g.CloseCurrentPopup()
				}),
				g.Button("No").OnClick(func() { g.CloseCurrentPopup() }),
			),
		),
	}
}
