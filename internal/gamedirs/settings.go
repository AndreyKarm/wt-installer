package gamedirs

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// settings are the user's choices about game installs, kept in
// %APPDATA%\WTInstaller\settings.json next to installed.json.
type settings struct {
	// CustomDirs are install folders the user added by hand, for installs
	// nothing in the registry points at.
	CustomDirs []string `json:"customGameDirs,omitempty"`
	// DisabledDirs are installs skins should not go into. A list of
	// exclusions (not of selections) so a newly found install is used by
	// default, like before this setting existed.
	DisabledDirs []string `json:"disabledGameDirs,omitempty"`
}

// settingsPath is a variable so tests can use a temp file.
var settingsPath = defaultSettingsPath()

func defaultSettingsPath() string {
	dir, err := os.UserConfigDir()
	if err != nil {
		dir = os.TempDir()
	}
	return filepath.Join(dir, "WTInstaller", "settings.json")
}

func loadSettings() (settings, error) {
	var s settings
	data, err := os.ReadFile(settingsPath)
	if os.IsNotExist(err) {
		return s, nil
	}
	if err != nil {
		return s, err
	}
	if err := json.Unmarshal(data, &s); err != nil {
		return s, fmt.Errorf("parse %s: %w", settingsPath, err)
	}
	return s, nil
}

// saveSettings writes through a temp file and rename so a crash mid-write
// can't leave a truncated file behind.
func saveSettings(s settings) error {
	if err := os.MkdirAll(filepath.Dir(settingsPath), 0755); err != nil {
		return err
	}
	data, err := json.MarshalIndent(s, "", "  ")
	if err != nil {
		return err
	}
	tmp := settingsPath + ".tmp"
	if err := os.WriteFile(tmp, data, 0644); err != nil {
		return err
	}
	return os.Rename(tmp, settingsPath)
}

// pathKey compares Windows paths: case-insensitive, ignoring trailing
// slashes and "." segments.
func pathKey(p string) string {
	return strings.ToLower(filepath.Clean(p))
}

func containsPath(list []string, p string) bool {
	k := pathKey(p)
	for _, q := range list {
		if pathKey(q) == k {
			return true
		}
	}
	return false
}

func withoutPath(list []string, p string) []string {
	k := pathKey(p)
	out := list[:0:0]
	for _, q := range list {
		if pathKey(q) != k {
			out = append(out, q)
		}
	}
	return out
}
