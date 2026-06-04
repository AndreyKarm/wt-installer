package main

import (
	"fmt"
	"os"

	"fyne.io/systray"
)

var AutoStartEnabled bool

func main() {
	systray.Run(onReady, onExit)
}

func onReady() {
	systray.SetIcon(iconBytes)
	systray.SetTooltip(appName)

	AutoStartEnabled = getAutoStartEnabled()

	autoStartItem := systray.AddMenuItemCheckbox(
		"Auto-start",
		"Toggle auto-start on Windows login",
		AutoStartEnabled,
	)
	systray.AddSeparator()
	quitItem := systray.AddMenuItem("Quit", "Quit "+appName)

	go startServer()

	go func() {
		for {
			select {
			case <-autoStartItem.ClickedCh:
				AutoStartEnabled = !AutoStartEnabled
				if err := setAutoStart(AutoStartEnabled); err != nil {
					fmt.Printf("Error toggling auto-start: %v\n", err)
					AutoStartEnabled = !AutoStartEnabled
				}
				if AutoStartEnabled {
					autoStartItem.Check()
				} else {
					autoStartItem.Uncheck()
				}
			case <-quitItem.ClickedCh:
				systray.Quit()
			}
		}
	}()
}

func onExit() {
	os.Exit(0)
}
