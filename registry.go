package main

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"

	"golang.org/x/sys/windows/registry"
)

type RegistryPath struct {
	Key     registry.Key
	SubKey  string
	ValName string
}

var (
	GameDirs  []string
	CamoDirs  []string
	SightDirs []string
)

var launcherRegistryPath = RegistryPath{
	Key:     registry.CURRENT_USER,
	SubKey:  `SOFTWARE\Gaijin\WarThunder`,
	ValName: "InstallDir",
}

var steamRegistryPath = RegistryPath{
	Key:     registry.CURRENT_USER,
	SubKey:  `SOFTWARE\Valve\Steam`,
	ValName: "SteamPath",
}

const warThunderSteamAppID = "236390"

func init() {
	ensureFileLogging()

	dirs, err := FindAllGameDirs()
	if err != nil {
		openLogsFolder()
		os.Exit(1)
	}
	GameDirs = dirs
	CamoDirs = make([]string, len(dirs))
	for i, d := range dirs {
		CamoDirs[i] = filepath.Join(d, "UserSkins")
	}

	SightDirs, err = GetSightDirs()
	if err != nil {
		log.Printf("Warning: could not find sight folders: %v", err)
	}

	log.Printf("Found War Thunder install(s): %s", strings.Join(dirs, ", "))
}

func OpenRegistryValue(p RegistryPath) (string, error) {
	key, err := registry.OpenKey(p.Key, p.SubKey, registry.QUERY_VALUE)
	if err != nil {
		return "", err
	}
	defer key.Close()
	val, _, err := key.GetStringValue(p.ValName)
	return val, err
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

// FindAllGameDirs checks for both the Gaijin Launcher and Steam
// installations and returns every one that is actually present, instead
// of stopping at the first match.
func FindAllGameDirs() ([]string, error) {
	var dirs []string
	seen := make(map[string]bool)

	if gamePath, err := OpenRegistryValue(launcherRegistryPath); err == nil {
		if !seen[gamePath] {
			seen[gamePath] = true
			dirs = append(dirs, gamePath)
		}
	}

	if steamPath, err := OpenRegistryValue(steamRegistryPath); err == nil {
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

// Sights
func GetSavesDirBase() string {
	homeDir, _ := os.UserHomeDir()
	return filepath.Join(
		homeDir, "Documents", "My Games", "WarThunder", "Saves",
	)
}

func GetSightDirs() ([]string, error) {
	savesPath := GetSavesDirBase()
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
