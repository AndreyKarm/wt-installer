// Command wt-installer is the tray app: it runs the local HTTP server used
// by the browser extension and keeps itself up to date.
//
// build.bat and CI generate rsrc.syso (the exe icon) in this folder; Go only
// links .syso files from the main package directory.
package main

import (
	"fmt"
	"log"
	"os"

	"fyne.io/systray"
	"golang.org/x/sys/windows"

	"liotom/installer/internal/app"
	"liotom/installer/internal/gamedirs"
	"liotom/installer/internal/logging"
	"liotom/installer/internal/server"
	"liotom/installer/internal/update"
	"liotom/installer/internal/winutil"
	"liotom/installer/media"
)

func main() {
	logging.Init()

	mutex, err := windows.CreateMutex(nil, false, windows.StringToUTF16Ptr("WTInstallerServer"))
	if err != nil || windows.GetLastError() == windows.ERROR_ALREADY_EXISTS {
		os.Exit(1)
	}
	defer windows.CloseHandle(mutex)

	gamedirs.Refresh()
	systray.Run(onReady, onExit)
}

func onReady() {
	systray.SetIcon(media.Icon)
	systray.SetTooltip(app.Name)

	autoStartEnabled := winutil.AutoStartEnabled()

	autoStartItem := systray.AddMenuItemCheckbox(
		"Auto-start",
		"Toggle auto-start on Windows login",
		autoStartEnabled,
	)
	updateItem := systray.AddMenuItem("Checking for updates...", "")
	updateItem.Disable()
	systray.AddSeparator()
	skinsItem := systray.AddMenuItem("Open skins folder", "Open UserSkins in every game install")
	sightsItem := systray.AddMenuItem("Open sights folder", "Open the UserSights folders")
	logsItem := systray.AddMenuItem("Open logs folder", "Open the log files folder")
	systray.AddSeparator()
	quitItem := systray.AddMenuItem("Quit", "Quit "+app.Name)

	go server.Start()

	go func() {
		hasUpdate, tag, dlURL, err := update.Check()
		if err != nil {
			log.Printf("Update check failed: %v", err)
			updateItem.SetTitle("Update check failed")
			return
		}
		if !hasUpdate {
			updateItem.SetTitle(fmt.Sprintf("Up to date (v%s)", app.Version))
			return
		}

		log.Printf("Update %s found, applying automatically", tag)
		updateItem.SetTitle(fmt.Sprintf("Updating to %s…", tag))
		if err := update.Apply(dlURL); err != nil {
			log.Printf("Update failed: %v", err)
			updateItem.SetTitle("Update failed")
			return
		}
		systray.Quit()
	}()

	go func() {
		for {
			select {
			case <-autoStartItem.ClickedCh:
				autoStartEnabled = !autoStartEnabled
				if err := winutil.SetAutoStart(autoStartEnabled); err != nil {
					log.Printf("Error toggling auto-start: %v\n", err)
					autoStartEnabled = !autoStartEnabled
				}
				if autoStartEnabled {
					autoStartItem.Check()
				} else {
					autoStartItem.Uncheck()
				}
			case <-skinsItem.ClickedCh:
				go openFolder("camo")
			case <-sightsItem.ClickedCh:
				go openFolder("sight")
			case <-logsItem.ClickedCh:
				go openFolder("logs")
			case <-quitItem.ClickedCh:
				systray.Quit()
			}
		}
	}()
}

func openFolder(target string) {
	if err := server.OpenFolder(target); err != nil {
		log.Printf("Tray: open %s folder: %v", target, err)
	}
}

func onExit() {
	os.Exit(0)
}
