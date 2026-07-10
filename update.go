package main

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"syscall"

	"fyne.io/systray"
)

const (
	githubOwner = "AndreyKarm"
	githubRepo  = "wt-installer"
)

type githubRelease struct {
	TagName string        `json:"tag_name"`
	Assets  []githubAsset `json:"assets"`
}

type githubAsset struct {
	Name               string `json:"name"`
	BrowserDownloadURL string `json:"browser_download_url"`
}

func getLatestRelease() (*githubRelease, error) {
	url := fmt.Sprintf(
		"https://api.github.com/repos/%s/%s/releases/latest",
		githubOwner, githubRepo,
	)
	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", appName+"/"+Version)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("GitHub API returned %d", resp.StatusCode)
	}

	var release githubRelease
	if err := json.NewDecoder(resp.Body).Decode(&release); err != nil {
		return nil, err
	}
	return &release, nil
}

// compareVersions returns 1 if a > b, -1 if a < b, 0 if equal.
func compareVersions(a, b string) int {
	a = strings.TrimPrefix(a, "v")
	b = strings.TrimPrefix(b, "v")
	pa := strings.Split(a, ".")
	pb := strings.Split(b, ".")
	n := len(pa)
	if len(pb) > n {
		n = len(pb)
	}
	for i := 0; i < n; i++ {
		var x, y int
		if i < len(pa) {
			fmt.Sscan(pa[i], &x)
		}
		if i < len(pb) {
			fmt.Sscan(pb[i], &y)
		}
		if x > y {
			return 1
		}
		if x < y {
			return -1
		}
	}
	return 0
}

// CheckForUpdate queries GitHub for the latest release.
// Returns (hasUpdate, tagName, downloadURL, error).
func CheckForUpdate() (bool, string, string, error) {
	release, err := getLatestRelease()
	if err != nil {
		return false, "", "", err
	}
	if compareVersions(release.TagName, Version) <= 0 {
		return false, "", "", nil
	}
	for _, asset := range release.Assets {
		if strings.HasSuffix(strings.ToLower(asset.Name), ".exe") {
			return true, release.TagName, asset.BrowserDownloadURL, nil
		}
	}
	return false, "", "", fmt.Errorf(
		"no .exe asset found in release %s", release.TagName,
	)
}

// ApplyUpdate downloads the new exe, then uses a hidden batch script to swap
// it in and restart after the current process exits.
func ApplyUpdate(downloadURL string) error {
	exePath := getExecutablePath()
	dir := filepath.Dir(exePath)
	newExePath := filepath.Join(dir, "_update_new.exe")
	batPath := filepath.Join(dir, "_update.bat")

	log.Printf("Downloading update from %s", downloadURL)
	resp, err := http.Get(downloadURL)
	if err != nil {
		return fmt.Errorf("download failed: %w", err)
	}
	defer resp.Body.Close()

	f, err := os.Create(newExePath)
	if err != nil {
		return fmt.Errorf("create staging exe: %w", err)
	}
	if _, err := io.Copy(f, resp.Body); err != nil {
		f.Close()
		os.Remove(newExePath)
		return fmt.Errorf("write staging exe: %w", err)
	}
	f.Close()

	bat := fmt.Sprintf(
		"@echo off\r\ntimeout /t 2 /nobreak >nul\r\nmove /Y \"%s\" \"%s\"\r\nstart \"\" \"%s\"\r\ndel \"%%~f0\"\r\n",
		newExePath, exePath, exePath,
	)
	if err := os.WriteFile(batPath, []byte(bat), 0644); err != nil {
		os.Remove(newExePath)
		return fmt.Errorf("write update script: %w", err)
	}

	cmd := exec.Command("cmd", "/C", batPath)
	cmd.SysProcAttr = &syscall.SysProcAttr{HideWindow: true}
	if err := cmd.Start(); err != nil {
		os.Remove(newExePath)
		os.Remove(batPath)
		return fmt.Errorf("launch update script: %w", err)
	}

	log.Printf("Update pending - restarting")
	systray.Quit()
	return nil
}
