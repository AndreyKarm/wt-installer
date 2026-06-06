package main

import (
	"os"
	"path/filepath"
	"strings"

	"golang.org/x/sys/windows/registry"
)

const appName = "WT Installer Server"

func getExecutablePath() string {
	ex, _ := os.Executable()
	path := filepath.Clean(ex)
	if strings.Contains(path, " ") {
		return `"` + path + `"`
	}
	return path
}

func openAutoStartKey(access uint32) (registry.Key, error) {
	return registry.OpenKey(
		registry.CURRENT_USER,
		`Software\Microsoft\Windows\CurrentVersion\Run`,
		access,
	)
}

func getAutoStartEnabled() bool {
	key, err := openAutoStartKey(registry.QUERY_VALUE)
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
	key, err := openAutoStartKey(registry.SET_VALUE)
	if err != nil {
		return err
	}
	defer key.Close()

	if enable {
		return key.SetStringValue(appName, getExecutablePath())
	}
	return key.DeleteValue(appName)
}
