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

// camoDirs is guarded by mu: Refresh can run from HTTP handlers while
// other requests read it.
var (
	mu       sync.RWMutex
	camoDirs []string
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

// CamoDirs returns a copy of the UserSkins directories found by the last
// Refresh.
func CamoDirs() []string {
	mu.RLock()
	defer mu.RUnlock()
	return append([]string(nil), camoDirs...)
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
	dirs, err := FindAll()
	_, sightErr := SightDirs()

	mu.Lock()
	defer mu.Unlock()

	if err != nil {
		log.Printf("Warning: no War Thunder installation found yet: %v", err)
		camoDirs = nil
	} else {
		camoDirs = make([]string, len(dirs))
		for i, d := range dirs {
			camoDirs[i] = filepath.Join(d, "UserSkins")
		}
		log.Printf("Found War Thunder install(s): %s", strings.Join(dirs, ", "))
	}

	if sightErr != nil {
		log.Printf("Warning: could not find sight folders: %v", sightErr)
	}
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

// FindAll checks for both the Gaijin Launcher and Steam installations and
// returns every one that is actually present, instead of stopping at the
// first match.
func FindAll() ([]string, error) {
	var dirs []string
	seen := make(map[string]bool)

	if gamePath, err := openRegistryValue(launcherRegistryPath); err == nil {
		if !seen[gamePath] {
			seen[gamePath] = true
			dirs = append(dirs, gamePath)
		}
	}

	if steamPath, err := openRegistryValue(steamRegistryPath); err == nil {
		if steamGamePath, err := FindSteamGamePath(steamPath); err == nil {
			if !seen[steamGamePath] {
				seen[steamGamePath] = true
				dirs = append(dirs, steamGamePath)
			}
		}
	}

	if len(dirs) == 0 {
		log.Printf("War Thunder installation not found")
		return nil, fmt.Errorf("War Thunder installation not found")
	}

	return dirs, nil
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
