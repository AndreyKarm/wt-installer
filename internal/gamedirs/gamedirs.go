// Package gamedirs locates War Thunder installs (Gaijin Launcher and Steam)
// and the per-account sight folders.
package gamedirs

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"golang.org/x/sys/windows/registry"
)

type registryPath struct {
	key     registry.Key
	subKey  string
	valName string
}

// Install is one War Thunder install folder and whether skins go into it.
type Install struct {
	Path string `json:"path"`
	// Source is how it was found: "launcher" (Gaijin registry key),
	// "steam", "nearby" (next to the launcher's install) or "custom"
	// (added by the user).
	Source  string `json:"source"`
	Enabled bool   `json:"enabled"`
	// Found is false for a custom folder that is gone or no longer looks
	// like a game install. Such folders are listed but never used.
	Found bool `json:"found"`
}

// installs is guarded by mu: Refresh can run from HTTP handlers while
// other requests read it. settingsMu serializes read-modify-write of the
// settings file.
var (
	mu         sync.RWMutex
	installs   []Install
	settingsMu sync.Mutex
)

// Where installs are looked up. Variables so tests don't depend on this
// machine's registry.
var (
	launcherDir = func() (string, error) { return openRegistryValue(launcherRegistryPath) }
	steamDir    = func() (string, error) {
		steamPath, err := openRegistryValue(steamRegistryPath)
		if err != nil {
			return "", err
		}
		return FindSteamGamePath(steamPath)
	}
)

var launcherRegistryPath = registryPath{
	key:     registry.CURRENT_USER,
	subKey:  `SOFTWARE\Gaijin\WarThunder`,
	valName: "InstallDir",
}

var steamRegistryPath = registryPath{
	key:     registry.CURRENT_USER,
	subKey:  `SOFTWARE\Valve\Steam`,
	valName: "SteamPath",
}

const warThunderSteamAppID = "236390"

// CamoDirs returns the UserSkins directories of the installs found by the
// last Refresh that the user hasn't turned off.
func CamoDirs() []string {
	mu.RLock()
	defer mu.RUnlock()
	var dirs []string
	for _, in := range installs {
		if in.Enabled && in.Found {
			dirs = append(dirs, filepath.Join(in.Path, "UserSkins"))
		}
	}
	return dirs
}

// AllCamoDirs returns the UserSkins directories of every known install,
// including turned-off ones, so uninstall still cleans up a skin installed
// before its install was turned off.
func AllCamoDirs() []string {
	mu.RLock()
	defer mu.RUnlock()
	dirs := make([]string, 0, len(installs))
	for _, in := range installs {
		dirs = append(dirs, filepath.Join(in.Path, "UserSkins"))
	}
	return dirs
}

// Installs returns a copy of the installs found by the last Refresh.
func Installs() []Install {
	mu.RLock()
	defer mu.RUnlock()
	return append([]Install(nil), installs...)
}

// looksLikeGameDir reports whether dir holds a War Thunder install.
// aces.vromfs.bin is the main game archive, present in launcher and Steam
// installs alike.
func looksLikeGameDir(dir string) bool {
	info, err := os.Stat(filepath.Join(dir, "aces.vromfs.bin"))
	return err == nil && !info.IsDir()
}

func openRegistryValue(p registryPath) (string, error) {
	key, err := registry.OpenKey(p.key, p.subKey, registry.QUERY_VALUE)
	if err != nil {
		return "", err
	}
	defer key.Close()
	val, _, err := key.GetStringValue(p.valName)
	return val, err
}

// Refresh re-scans for War Thunder installs. Safe to call at any time
// (e.g. lazily before a download) so the tray app and its HTTP server keep
// running even if the game wasn't found yet, instead of killing the whole
// background service.
func Refresh() {
	s, err := loadSettings()
	if err != nil {
		log.Printf("Warning: could not read settings, using defaults: %v", err)
	}
	list := discover(s)
	_, sightErr := SightDirs()

	mu.Lock()
	installs = list
	mu.Unlock()

	if len(list) == 0 {
		log.Printf("Warning: no War Thunder installation found yet")
	} else {
		var parts []string
		for _, in := range list {
			state := "on"
			switch {
			case !in.Found:
				state = "missing"
			case !in.Enabled:
				state = "off"
			}
			parts = append(parts, fmt.Sprintf("%s (%s, %s)", in.Path, in.Source, state))
		}
		log.Printf("War Thunder install(s): %s", strings.Join(parts, ", "))
	}

	if sightErr != nil {
		log.Printf("Warning: could not find sight folders: %v", sightErr)
	}
}

// discover lists every install: the Gaijin Launcher's, Steam's, game
// folders next to the launcher's, and the user's custom folders. The
// launcher keeps a single InstallDir and rewrites it to whichever install
// ran last, so with a second copy (for example a dev client) the other
// one is only found by looking next to it.
func discover(s settings) []Install {
	var list []Install
	add := func(path, source string, found bool) {
		if path == "" || containsPath(pathsOf(list), path) {
			return
		}
		list = append(list, Install{
			Path:    filepath.Clean(path),
			Source:  source,
			Enabled: !containsPath(s.DisabledDirs, path),
			Found:   found,
		})
	}

	launcher, launcherErr := launcherDir()
	if launcherErr == nil {
		add(launcher, "launcher", dirExists(launcher))
	}
	if steam, err := steamDir(); err == nil {
		add(steam, "steam", dirExists(steam))
	}
	if launcherErr == nil {
		for _, dir := range nearbyGameDirs(launcher) {
			add(dir, "nearby", true)
		}
	}
	for _, dir := range s.CustomDirs {
		add(dir, "custom", looksLikeGameDir(dir))
	}
	return list
}

// nearbyGameDirs returns game installs in the same parent folder as dir
// (for example F:\Games\WarThunder next to F:\Games\WarThunderDev).
func nearbyGameDirs(dir string) []string {
	parent := filepath.Dir(filepath.Clean(dir))
	if parent == filepath.Clean(dir) {
		return nil
	}
	entries, err := os.ReadDir(parent)
	if err != nil {
		return nil
	}
	var dirs []string
	for _, e := range entries {
		if !e.IsDir() {
			continue
		}
		p := filepath.Join(parent, e.Name())
		if pathKey(p) != pathKey(dir) && looksLikeGameDir(p) {
			dirs = append(dirs, p)
		}
	}
	return dirs
}

func pathsOf(list []Install) []string {
	paths := make([]string, len(list))
	for i, in := range list {
		paths[i] = in.Path
	}
	return paths
}

func dirExists(dir string) bool {
	info, err := os.Stat(dir)
	return err == nil && info.IsDir()
}

// updateSettings applies change to the saved settings and re-scans.
func updateSettings(change func(*settings) error) error {
	settingsMu.Lock()
	defer settingsMu.Unlock()

	s, err := loadSettings()
	if err != nil {
		// A corrupt file shouldn't lock the user out of fixing it.
		log.Printf("Settings unreadable, starting over: %v", err)
		s = settings{}
	}
	if err := change(&s); err != nil {
		return err
	}
	if err := saveSettings(s); err != nil {
		log.Printf("Save settings: %v", err)
		return fmt.Errorf("save settings: %w", err)
	}
	Refresh()
	return nil
}

// SetEnabled turns skin installs into a known install on or off.
func SetEnabled(path string, enabled bool) error {
	if !containsPath(pathsOf(Installs()), path) {
		Refresh()
		if !containsPath(pathsOf(Installs()), path) {
			return fmt.Errorf("unknown install folder %s", path)
		}
	}
	return updateSettings(func(s *settings) error {
		s.DisabledDirs = withoutPath(s.DisabledDirs, path)
		if !enabled {
			s.DisabledDirs = append(s.DisabledDirs, filepath.Clean(path))
		}
		return nil
	})
}

// CleanInputPath turns a pasted folder path into an install folder: it
// drops surrounding quotes (Explorer's "Copy as path" adds them) and a
// trailing UserSkins, in case the user copied that folder instead.
func CleanInputPath(path string) string {
	path = strings.Trim(strings.TrimSpace(path), `"`)
	if path == "" {
		return ""
	}
	path = filepath.Clean(path)
	if strings.EqualFold(filepath.Base(path), "UserSkins") {
		path = filepath.Dir(path)
	}
	return path
}

// AddCustom adds an install folder by hand and turns it on. The folder must
// look like a game install, so the API can't be used to write skins into
// arbitrary folders.
func AddCustom(path string) error {
	path = CleanInputPath(path)
	if !filepath.IsAbs(path) {
		return fmt.Errorf("enter the full path of the War Thunder folder")
	}
	if !looksLikeGameDir(path) {
		return fmt.Errorf("%s does not look like a War Thunder folder (no aces.vromfs.bin)", path)
	}
	return updateSettings(func(s *settings) error {
		s.DisabledDirs = withoutPath(s.DisabledDirs, path)
		if !containsPath(s.CustomDirs, path) && !containsPath(pathsOf(Installs()), path) {
			s.CustomDirs = append(s.CustomDirs, path)
		}
		return nil
	})
}

// RemoveCustom forgets a folder added with AddCustom. Detected installs
// can only be turned off.
func RemoveCustom(path string) error {
	return updateSettings(func(s *settings) error {
		if !containsPath(s.CustomDirs, path) {
			return fmt.Errorf("%s was not added by hand", path)
		}
		s.CustomDirs = withoutPath(s.CustomDirs, path)
		s.DisabledDirs = withoutPath(s.DisabledDirs, path)
		return nil
	})
}

// ParseVDFKV parses a single VDF line of the form "key"  "value"
func ParseVDFKV(line string) (key, val string) {
	line = strings.TrimSpace(line)
	if len(line) == 0 || line[0] != '"' {
		return "", ""
	}
	end := strings.Index(line[1:], `"`)
	if end == -1 {
		return "", ""
	}
	key = line[1 : end+1]
	rest := strings.TrimSpace(line[end+2:])
	if len(rest) == 0 || rest[0] != '"' {
		return key, "" // block key, no value
	}
	end2 := strings.Index(rest[1:], `"`)
	if end2 == -1 {
		return key, ""
	}
	val = strings.ReplaceAll(rest[1:end2+1], `\\`, `\`)
	return key, val
}

// FindSteamGamePath parses libraryfolders.vdf to locate War Thunder
func FindSteamGamePath(steamPath string) (string, error) {
	vdfPath := filepath.Join(steamPath, "steamapps", "libraryfolders.vdf")
	data, err := os.ReadFile(vdfPath)
	if err != nil {
		return "", fmt.Errorf("could not read libraryfolders.vdf: %w", err)
	}

	var (
		currentLibPath string
		depth          int
		inApps         bool
		appsDepth      int
	)

	for _, rawLine := range strings.Split(string(data), "\n") {
		line := strings.TrimSpace(rawLine)
		switch line {
		case "{":
			depth++
		case "}":
			if inApps && depth == appsDepth {
				inApps = false
			}
			depth--
		default:
			key, val := ParseVDFKV(line)
			if key == "" {
				continue
			}
			switch {
			case depth == 2 && key == "path":
				currentLibPath = val
			case depth == 2 && key == "apps":
				inApps = true
				appsDepth = depth + 1
			case inApps && key == warThunderSteamAppID:
				return filepath.Join(
					currentLibPath,
					"steamapps", "common", "War Thunder",
				), nil
			}
		}
	}

	log.Printf("War Thunder not found in any Steam library")
	return "", fmt.Errorf("War Thunder not found in any Steam library")
}

// SavesDirBase returns Documents\My Games\WarThunder\Saves.
func SavesDirBase() string {
	homeDir, _ := os.UserHomeDir()
	return filepath.Join(
		homeDir, "Documents", "My Games", "WarThunder", "Saves",
	)
}

// SightDirs returns UserSights\all_tanks for every numeric (account id)
// folder under Saves. The folders may not exist yet.
func SightDirs() ([]string, error) {
	savesPath := SavesDirBase()
	entries, err := os.ReadDir(savesPath)
	if err != nil {
		log.Printf("could not read Saves directory: %v\n", err)
		return nil, fmt.Errorf("could not read Saves directory: %w", err)
	}

	var dirs []string
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		name := entry.Name()
		if isDigitsOnly(name) {
			dirs = append(dirs, filepath.Join(
				savesPath, name,
				"production", "UserSights", "all_tanks",
			))
		}
	}

	log.Printf("Found %d sight folders in %s", len(dirs), savesPath)
	return dirs, nil
}

func isDigitsOnly(s string) bool {
	for _, c := range s {
		if c < '0' || c > '9' {
			return false
		}
	}
	return true
}
