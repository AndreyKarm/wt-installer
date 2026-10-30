package update

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"syscall"
	"testing"
	"time"
)

func TestCompareVersions(t *testing.T) {
	for _, c := range []struct {
		a, b string
		want int
	}{
		{"0.1.8", "0.1.7", 1},
		{"v0.1.8", "0.1.8", 0},
		{"0.1.10", "0.1.9", 1}, // numeric, not string, comparison
		{"0.2", "0.1.99", 1},
		{"1.0", "1.0.0", 0},
		{"0.1.7", "v0.1.8", -1},
		{"1.0.0-beta", "1.0.0", 0}, // suffixes are ignored
	} {
		if got := compareVersions(c.a, c.b); got != c.want {
			t.Errorf("compareVersions(%q, %q) = %d, want %d", c.a, c.b, got, c.want)
		}
	}
}

func TestRecentlyAttempted(t *testing.T) {
	old := attemptPath
	attemptPath = filepath.Join(t.TempDir(), "sub", "update-attempt.json")
	t.Cleanup(func() { attemptPath = old })

	if RecentlyAttempted("v0.1.9") {
		t.Fatal("no record yet, but reported as attempted")
	}
	recordAttempt("v0.1.9")
	if !RecentlyAttempted("v0.1.9") {
		t.Error("just-recorded attempt not reported")
	}
	if RecentlyAttempted("v0.2.0") {
		t.Error("a different tag reported as attempted")
	}

	stale, _ := json.Marshal(attempt{Tag: "v0.1.9", Time: time.Now().Add(-retryAfter - time.Minute)})
	os.WriteFile(attemptPath, stale, 0644)
	if RecentlyAttempted("v0.1.9") {
		t.Error("attempt older than retryAfter still blocks")
	}
}

// runSwap writes the swap script with a launch command that only drops a
// marker file, runs it, and reports whether the marker appeared.
func runSwap(t *testing.T, dir, newExe, exe string, tries int) bool {
	t.Helper()
	marker := filepath.Join(dir, "launched.txt")
	bat := filepath.Join(dir, "_update.bat")
	script := swapScript(newExe, exe, fmt.Sprintf(`echo launched> "%s"`, marker), tries)
	if err := os.WriteFile(bat, []byte(script), 0644); err != nil {
		t.Fatal(err)
	}
	cmd := exec.Command("cmd", "/C", bat)
	cmd.SysProcAttr = &syscall.SysProcAttr{HideWindow: true}
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("script failed: %v\n%s", err, out)
	}
	if _, err := os.Stat(bat); !os.IsNotExist(err) {
		t.Error("script did not delete itself")
	}
	_, err := os.Stat(marker)
	return err == nil
}

func TestSwapScriptRetriesUntilExeUnlocked(t *testing.T) {
	dir := t.TempDir()
	exe := filepath.Join(dir, "WT Installer.exe") // space in the name, like the real one
	newExe := filepath.Join(dir, "_update_new.exe")
	os.WriteFile(exe, []byte("old"), 0644)
	os.WriteFile(newExe, []byte("new"), 0644)

	// Hold the exe open like a process that hasn't fully exited yet.
	lock, err := os.Open(exe)
	if err != nil {
		t.Fatal(err)
	}
	go func() {
		time.Sleep(2500 * time.Millisecond)
		lock.Close()
	}()

	if !runSwap(t, dir, newExe, exe, 15) {
		t.Error("launch command did not run")
	}
	if got, _ := os.ReadFile(exe); string(got) != "new" {
		t.Errorf("exe not replaced after unlock, content %q", got)
	}
	if _, err := os.Stat(newExe); !os.IsNotExist(err) {
		t.Error("staged exe left behind")
	}
}

func TestSwapScriptGivesUpAndStartsOldExe(t *testing.T) {
	dir := t.TempDir()
	exe := filepath.Join(dir, "WT Installer.exe")
	newExe := filepath.Join(dir, "_update_new.exe")
	os.WriteFile(exe, []byte("old"), 0644)
	os.WriteFile(newExe, []byte("new"), 0644)

	lock, err := os.Open(exe)
	if err != nil {
		t.Fatal(err)
	}
	defer lock.Close()

	start := time.Now()
	if !runSwap(t, dir, newExe, exe, 2) {
		t.Error("old exe not launched after giving up")
	}
	if got, _ := os.ReadFile(exe); string(got) != "old" {
		t.Errorf("locked exe changed: %q", got)
	}
	if _, err := os.Stat(newExe); !os.IsNotExist(err) {
		t.Error("staged exe not cleaned up after giving up")
	}
	if elapsed := time.Since(start); elapsed > 15*time.Second {
		t.Errorf("gave up too slowly: %v", elapsed)
	}
}
