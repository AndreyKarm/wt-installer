// Package update implements self-update from GitHub Releases.
package update

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
	"time"

	"liotom/installer/internal/app"
	"liotom/installer/internal/winutil"
)

const (
	githubOwner = "AndreyKarm"
	githubRepo  = "wt-installer"
)

var httpClient = &http.Client{Timeout: 5 * time.Minute}

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
	req.Header.Set("User-Agent", app.Name+"/"+app.Version)

	resp, err := httpClient.Do(req)
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

// Check queries GitHub for the latest release.
// Returns (hasUpdate, tagName, downloadURL, error).
func Check() (bool, string, string, error) {
	release, err := getLatestRelease()
	if err != nil {
		return false, "", "", err
	}
	if compareVersions(release.TagName, app.Version) <= 0 {
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

// Apply downloads the release tag's exe and starts a hidden batch script
// that swaps it in and restarts once the current process exits. On success
// the caller must quit the app.
func Apply(tag, downloadURL string) error {
	exePath := winutil.ExecutablePath()
	dir := filepath.Dir(exePath)
	newExePath := filepath.Join(dir, "_update_new.exe")
	batPath := filepath.Join(dir, "_update.bat")

	log.Printf("Downloading update from %s", downloadURL)
	resp, err := httpClient.Get(downloadURL)
	if err != nil {
		return fmt.Errorf("download failed: %w", err)
	}
	defer resp.Body.Close()
	// Without this an error page would be saved as the new exe and the
	// app would replace itself with a broken file.
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("download failed: server returned %s", resp.Status)
	}

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

	bat := swapScript(newExePath, exePath, fmt.Sprintf(`start "" "%s"`, exePath), swapTries)
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

	recordAttempt(tag)
	log.Printf("Update pending - restarting")
	return nil
}

// swapTries is how many times (about a second apart) the update script
// tries to replace the exe.
const swapTries = 30

// swapScript returns a batch script that replaces exe with newExe and then
// runs launch. The old exe can stay locked for a while after the process
// quits (slow shutdown, antivirus scans), so the move is retried. If it
// never succeeds the staged file is deleted and launch starts the old exe.
// The caller's attempt record then stops the app from retrying right away.
func swapScript(newExe, exe, launch string, tries int) string {
	return strings.Join([]string{
		"@echo off",
		"set tries=0",
		":retry",
		// ping is the classic portable sleep; timeout fails without a
		// console input.
		"ping -n 2 127.0.0.1 >nul",
		fmt.Sprintf(`move /Y "%s" "%s" >nul 2>&1`, newExe, exe),
		"if not errorlevel 1 goto launch",
		"set /a tries+=1",
		fmt.Sprintf("if %%tries%% lss %d goto retry", tries),
		fmt.Sprintf(`del "%s" >nul 2>&1`, newExe),
		":launch",
		launch,
		// "(goto)" ends the script before del removes it, so cmd doesn't
		// fail trying to read the next line of a deleted file.
		`(goto) 2>nul & del "%~f0"`,
	}, "\r\n") + "\r\n"
}

// retryAfter is how long the app waits before trying the same update again
// after an attempt that didn't take effect.
const retryAfter = 6 * time.Hour

type attempt struct {
	Tag  string    `json:"tag"`
	Time time.Time `json:"time"`
}

// attemptPath is a variable so tests can use a temp file.
var attemptPath = defaultAttemptPath()

func defaultAttemptPath() string {
	dir, err := os.UserConfigDir()
	if err != nil {
		dir = os.TempDir()
	}
	return filepath.Join(dir, "WTInstaller", "update-attempt.json")
}

func recordAttempt(tag string) {
	data, _ := json.Marshal(attempt{Tag: tag, Time: time.Now().UTC()})
	err := os.MkdirAll(filepath.Dir(attemptPath), 0755)
	if err == nil {
		err = os.WriteFile(attemptPath, data, 0644)
	}
	if err != nil {
		log.Printf("Could not record update attempt: %v", err)
	}
}

// RecentlyAttempted reports whether updating to tag was already tried in
// the last retryAfter. Called with a tag that is still newer than the
// running version, it means that attempt failed (the swap never happened),
// so applying it again would just restart into the old exe in a loop.
func RecentlyAttempted(tag string) bool {
	data, err := os.ReadFile(attemptPath)
	if err != nil {
		return false
	}
	var a attempt
	if json.Unmarshal(data, &a) != nil {
		return false
	}
	return a.Tag == tag && time.Since(a.Time) < retryAfter
}
