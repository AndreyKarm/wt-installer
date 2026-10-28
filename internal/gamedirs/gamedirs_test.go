package gamedirs

import (
	"os"
	"path/filepath"
	"testing"
)

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
