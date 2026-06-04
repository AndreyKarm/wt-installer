package main

import (
	"fmt"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/app"
	"fyne.io/fyne/v2/driver/desktop"
)

var AutoStartEnabled bool

func main() {
	App := app.New()
	App.SetIcon(FaviconResource)

	desk, ok := App.(desktop.App)
	if !ok {
		fmt.Println("This app is not supported on this platform")
		return
	}

	AutoStartEnabled = getAutoStartEnabled()

	var trayMenu *fyne.Menu

	autoStartItem := fyne.NewMenuItem("Auto-start", nil)
	autoStartItem.Checked = AutoStartEnabled
	autoStartItem.Action = func() {
		AutoStartEnabled = !AutoStartEnabled
		if err := setAutoStart(AutoStartEnabled); err != nil {
			fmt.Printf("Error toggling auto-start: %v\n", err)
			AutoStartEnabled = !AutoStartEnabled
		}
		autoStartItem.Checked = AutoStartEnabled
		trayMenu.Refresh()
	}

	trayMenu = fyne.NewMenu(appName, autoStartItem)
	desk.SetSystemTrayMenu(trayMenu)

	App.Lifecycle().SetOnStarted(func() {
		desk.SetSystemTrayIcon(FaviconResource)
	})

	go startServer()

	App.Run()
}
