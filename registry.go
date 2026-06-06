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
	GameDir string
	CamoDir string
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

func openRegistryValue(p RegistryPath) (string, error) {
	key, err := registry.OpenKey(p.Key, p.SubKey, registry.QUERY_VALUE)
	if err != nil {
		return "", err
	}
	defer key.Close()
	val, _, err := key.GetStringValue(p.ValName)
	return val, err
}

// parseVDFKV parses a single VDF line of the form "key"  "value"
func parseVDFKV(line string) (key, val string) {
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

// findSteamGamePath parses libraryfolders.vdf to locate War Thunder
func findSteamGamePath(steamPath string) (string, error) {
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
			key, val := parseVDFKV(line)
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

	return "", fmt.Errorf("War Thunder not found in any Steam library")
}

func findGameDir() (string, error) {
	// Installed via Gaijin Launcher
	if gamePath, err := openRegistryValue(launcherRegistryPath); err == nil {
		return gamePath, nil
	}

	// Installed via Steam
	if steamPath, err := openRegistryValue(steamRegistryPath); err == nil {
		return findSteamGamePath(steamPath)
	}

	return "", fmt.Errorf("War Thunder installation not found")
}

func init() {
	dir, err := findGameDir()
	if err != nil {
		log.Fatalf("Could not find War Thunder install path: %v", err)
	}
	GameDir = dir
	CamoDir = filepath.Join(GameDir, "UserSkins")
	log.Printf("Found War Thunder at: %s", GameDir)
}
