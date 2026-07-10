package main

import (
	"fmt"
	"log"
	"os"

	"fyne.io/systray"
	"golang.org/x/sys/windows"
)

var AutoStartEnabled bool

func main() {
	ensureFileLogging()

	mutex, err := windows.CreateMutex(nil, false, windows.StringToUTF16Ptr("WTInstallerServer"))
	if err != nil || windows.GetLastError() == windows.ERROR_ALREADY_EXISTS {
		os.Exit(1)
	}
	defer windows.CloseHandle(mutex)

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
	updateItem := systray.AddMenuItem("Checking for updates...", "")
	updateItem.Disable()
	systray.AddSeparator()
	quitItem := systray.AddMenuItem("Quit", "Quit "+appName)

	go startServer()

	go func() {
		hasUpdate, tag, dlURL, err := CheckForUpdate()
		if err != nil {
			log.Printf("Update check failed: %v", err)
			updateItem.SetTitle("Update check failed")
			return
		}
		if !hasUpdate {
			updateItem.SetTitle(fmt.Sprintf("Up to date (v%s)", Version))
			return
		}

		log.Printf("Update %s found, applying automatically", tag)
		updateItem.SetTitle(fmt.Sprintf("Updating to %s…", tag))
		if err := ApplyUpdate(dlURL); err != nil {
			log.Printf("Update failed: %v", err)
			updateItem.SetTitle("Update failed")
		}
	}()

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
