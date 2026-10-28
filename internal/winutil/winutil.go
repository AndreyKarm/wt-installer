// Package winutil wraps the small Windows shell and registry helpers the
// app needs.
package winutil

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"golang.org/x/sys/windows"
	"golang.org/x/sys/windows/registry"

	"liotom/installer/internal/app"
)

// ExecutablePath returns the cleaned path of the running executable.
func ExecutablePath() string {
	ex, _ := os.Executable()
	return filepath.Clean(ex)
}

// OpenInExplorer opens a folder in Windows Explorer via ShellExecute. This
// avoids `cmd /c start`, which misparses paths containing & or ^ and
// flashes a console window from the GUI build.
func OpenInExplorer(path string) error {
	verb, err := windows.UTF16PtrFromString("open")
	if err != nil {
		return err
	}
	file, err := windows.UTF16PtrFromString(path)
	if err != nil {
		return err
	}
	return windows.ShellExecute(0, verb, file, nil, nil, windows.SW_SHOWNORMAL)
}

func executablePathForRegistry() string {
	path := ExecutablePath()
	if strings.Contains(path, " ") {
		return fmt.Sprintf("\"%s\"", path)
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

// AutoStartEnabled reports whether the Run key points at this executable.
func AutoStartEnabled() bool {
	key, err := openAutoStartKey(registry.QUERY_VALUE)
	if err != nil {
		return false
	}
	defer key.Close()

	val, _, err := key.GetStringValue(app.Name)
	if err != nil {
		return false
	}
	return val == executablePathForRegistry()
}

// SetAutoStart adds or removes this executable from the Run key.
func SetAutoStart(enable bool) error {
	key, err := openAutoStartKey(registry.SET_VALUE)
	if err != nil {
		return err
	}
	defer key.Close()

	if enable {
		return key.SetStringValue(app.Name, executablePathForRegistry())
	}
	return key.DeleteValue(app.Name)
}
