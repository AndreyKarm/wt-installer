package gamedirs

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// fakeGame creates a folder that looks like a War Thunder install.
func fakeGame(t *testing.T, dir string) string {
	t.Helper()
	if err := os.MkdirAll(dir, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "aces.vromfs.bin"), []byte("x"), 0644); err != nil {
		t.Fatal(err)
	}
	return dir
}

// withFakeMachine points the registry lookups at launcher (empty = none)
// and the settings file at a temp folder.
func withFakeMachine(t *testing.T, launcher string) {
	t.Helper()
	oldL, oldS, oldPath := launcherDir, steamDir, settingsPath
	launcherDir = func() (string, error) {
		if launcher == "" {
			return "", os.ErrNotExist
		}
		return launcher, nil
	}
	steamDir = func() (string, error) { return "", os.ErrNotExist }
	settingsPath = filepath.Join(t.TempDir(), "settings.json")
	t.Cleanup(func() {
		launcherDir, steamDir, settingsPath = oldL, oldS, oldPath
		Refresh()
	})
}

// The launcher's registry key only names the install that ran last; a
// second install next to it must be found too.
func TestDiscoverFindsNearbyInstall(t *testing.T) {
	games := t.TempDir()
	dev := fakeGame(t, filepath.Join(games, "WarThunderDev"))
	regular := fakeGame(t, filepath.Join(games, "WarThunder"))
	os.MkdirAll(filepath.Join(games, "OtherGame"), 0755)
	withFakeMachine(t, dev)

	Refresh()
	got := Installs()
	if len(got) != 2 || got[0].Path != dev || got[0].Source != "launcher" ||
		got[1].Path != regular || got[1].Source != "nearby" {
		t.Fatalf("installs: %+v", got)
	}
	if dirs := CamoDirs(); len(dirs) != 2 {
		t.Errorf("both installs should be on by default: %v", dirs)
	}
}

func TestEnableAddRemove(t *testing.T) {
	games := t.TempDir()
	dev := fakeGame(t, filepath.Join(games, "WarThunderDev"))
	regular := fakeGame(t, filepath.Join(games, "WarThunder"))
	custom := fakeGame(t, filepath.Join(t.TempDir(), "WT Copy"))
	withFakeMachine(t, dev)
	Refresh()

	// Paths are compared case-insensitively, like Windows does.
	if err := SetEnabled(strings.ToUpper(dev), false); err != nil {
		t.Fatal(err)
	}
	if dirs := CamoDirs(); len(dirs) != 1 || dirs[0] != filepath.Join(regular, "UserSkins") {
		t.Errorf("after turning dev off: %v", dirs)
	}
	if len(AllCamoDirs()) != 2 {
		t.Errorf("turned-off install missing from AllCamoDirs: %v", AllCamoDirs())
	}
	if err := SetEnabled(filepath.Join(games, "Nope"), true); err == nil {
		t.Errorf("enabled an unknown folder")
	}

	if err := AddCustom(`"` + filepath.Join(custom, "UserSkins") + `"`); err != nil {
		t.Fatal(err)
	}
	if dirs := CamoDirs(); len(dirs) != 2 || dirs[1] != filepath.Join(custom, "UserSkins") {
		t.Errorf("after adding custom: %v", dirs)
	}
	if err := AddCustom(t.TempDir()); err == nil || !strings.Contains(err.Error(), "does not look like") {
		t.Errorf("added a non-game folder: %v", err)
	}
	if err := RemoveCustom(dev); err == nil {
		t.Errorf("removed a detected install")
	}

	// Settings survive a restart.
	Refresh()
	if dirs := CamoDirs(); len(dirs) != 2 {
		t.Errorf("after re-scan: %v", dirs)
	}

	if err := RemoveCustom(custom); err != nil {
		t.Fatal(err)
	}
	if got := Installs(); len(got) != 2 {
		t.Errorf("custom folder still listed: %+v", got)
	}
}

func TestParseVDFKV(t *testing.T) {
	for line, want := range map[string][2]string{
		`"path"		"D:\\SteamLibrary"`:   {"path", `D:\SteamLibrary`},
		`  "236390"   "51234567890"  `: {"236390", "51234567890"},
		`"apps"`:                       {"apps", ""},
		`{`:                            {"", ""},
		``:                             {"", ""},
		`"unterminated`:                {"", ""},
	} {
		k, v := ParseVDFKV(line)
		if k != want[0] || v != want[1] {
			t.Errorf("%q: got (%q, %q), want (%q, %q)", line, k, v, want[0], want[1])
		}
	}
}

func writeVDF(t *testing.T, content string) string {
	t.Helper()
	steam := t.TempDir()
	os.MkdirAll(filepath.Join(steam, "steamapps"), 0755)
	if err := os.WriteFile(filepath.Join(steam, "steamapps", "libraryfolders.vdf"), []byte(content), 0644); err != nil {
		t.Fatal(err)
	}
	return steam
}

// Current Steam format: numbered library blocks with an "apps" block each.
const multiLibraryVDF = `"libraryfolders"
{
	"0"
	{
		"path"		"C:\\Program Files (x86)\\Steam"
		"label"		""
		"apps"
		{
			"228980"		"123"
		}
	}
	"1"
	{
		"path"		"E:\\SteamLibrary"
		"contentid"		"42"
		"apps"
		{
			"730"		"456"
			"236390"		"60000000000"
		}
	}
}
`

func TestFindSteamGamePath(t *testing.T) {
	got, err := FindSteamGamePath(writeVDF(t, multiLibraryVDF))
	if err != nil {
		t.Fatal(err)
	}
	want := filepath.Join(`E:\SteamLibrary`, "steamapps", "common", "War Thunder")
	if got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}

func TestFindSteamGamePathCRLFAndMissing(t *testing.T) {
	crlf := ""
	for _, r := range multiLibraryVDF {
		if r == '\n' {
			crlf += "\r"
		}
		crlf += string(r)
	}
	if _, err := FindSteamGamePath(writeVDF(t, crlf)); err != nil {
		t.Errorf("CRLF file: %v", err)
	}

	noWT := `"libraryfolders"
{
	"0"
	{
		"path"		"C:\\Steam"
		"apps"
		{
			"730"		"1"
		}
	}
}
`
	if _, err := FindSteamGamePath(writeVDF(t, noWT)); err == nil {
		t.Errorf("found War Thunder in a library without it")
	}
	if _, err := FindSteamGamePath(t.TempDir()); err == nil {
		t.Errorf("missing libraryfolders.vdf not reported")
	}
}

// An app id that only appears outside an "apps" block (for example as a
// library's contentid) must not match.
func TestFindSteamGamePathIgnoresIdOutsideApps(t *testing.T) {
	vdf := `"libraryfolders"
{
	"0"
	{
		"path"		"C:\\Steam"
		"236390"		"not an app entry"
		"apps"
		{
			"730"		"1"
		}
	}
}
`
	if got, err := FindSteamGamePath(writeVDF(t, vdf)); err == nil {
		t.Errorf("matched id outside apps block: %q", got)
	}
}
