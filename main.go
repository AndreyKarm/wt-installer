package main

import (
	"fmt"
	"os"

	"fyne.io/systray"
	"golang.org/x/sys/windows"
)

var AutoStartEnabled bool

func main() {
	mutex, err := windows.CreateMutex(nil, false, windows.StringToUTF16Ptr("WTInstallerServer"))
	if err != nil || windows.GetLastError() == windows.ERROR_ALREADY_EXISTS {
		showFatalDialog("Already running", "WT Installer Server is already running.")
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
