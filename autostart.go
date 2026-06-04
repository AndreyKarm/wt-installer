package main

import (
	"os"
	"path/filepath"

	"golang.org/x/sys/windows/registry"
)

const appName = "WT Installer Server"

func getExecutablePath() string {
	ex, _ := os.Executable()
	return filepath.ToSlash(ex)
}

func getAutoStartEnabled() bool {
	key, err := registry.OpenKey(
		registry.CURRENT_USER,
		`Software\Microsoft\Windows\CurrentVersion\Run`,
		registry.QUERY_VALUE,
	)
	if err != nil {
		return false
	}
	defer key.Close()

	val, _, err := key.GetStringValue(appName)
	if err != nil {
		return false
	}
	return val == getExecutablePath()
}

func setAutoStart(enable bool) error {
	key, err := registry.OpenKey(
		registry.CURRENT_USER,
		`Software\Microsoft\Windows\CurrentVersion\Run`,
		registry.WRITE,
	)
	if err != nil {
		return err
	}
	defer key.Close()

	if enable {
		return key.SetStringValue(appName, getExecutablePath())
	}
	return key.DeleteValue(appName)
}
