package installer

import (
	"archive/zip"
	"bytes"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestMain(m *testing.M) {
	// Never write to the real %APPDATA% store from tests.
	recordInstall = func(int, string, []string) error { return nil }
	os.Exit(m.Run())
}

type recorded struct {
	postId int
	kind   string
	paths  []string
}

func captureRecords(t *testing.T) *[]recorded {
	var got []recorded
	old := recordInstall
	recordInstall = func(postId int, kind string, paths []string) error {
		got = append(got, recorded{postId, kind, paths})
		return nil
	}
	t.Cleanup(func() { recordInstall = old })
	return &got
}

func TestInstallsAreRecorded(t *testing.T) {
	camo := withCamoDir(t)
	sights := t.TempDir()
	oldSights := listSightDirs
	listSightDirs = func() ([]string, error) { return []string{sights}, nil }
	t.Cleanup(func() { listSightDirs = oldSights })
	records := captureRecords(t)

	srv := serveZips(t, map[string][]byte{
		"skin":  makeZip(t, map[string]string{"tiger/body_c.dds": "A"}),
		"skin2": makeZip(t, map[string]string{"tiger/body_c.dds": "B"}),
		"sight": makeZip(t, map[string]string{"pack/a.blk": "1", "pack/b.blk": "2"}),
	})

	DownloadSkin(1, srv.URL+"/skin")
	DownloadSkin(1, srv.URL+"/skin") // already installed: still recorded
	DownloadSkin(2, srv.URL+"/skin2")
	DownloadSight(3, srv.URL+"/sight")
	DownloadSight(3, srv.URL+"/sight") // identical files skipped: still recorded

	want := []recorded{
		{1, "camo", []string{filepath.Join(camo, "tiger")}},
		{1, "camo", []string{filepath.Join(camo, "tiger")}},
		{2, "camo", []string{filepath.Join(camo, "tiger_2")}},
		{3, "sight", []string{filepath.Join(sights, "a.blk"), filepath.Join(sights, "b.blk")}},
		{3, "sight", []string{filepath.Join(sights, "a.blk"), filepath.Join(sights, "b.blk")}},
	}
	if len(*records) != len(want) {
		t.Fatalf("got %d records, want %d: %+v", len(*records), len(want), *records)
	}
	for i, w := range want {
		g := (*records)[i]
		if g.postId != w.postId || g.kind != w.kind || strings.Join(g.paths, "|") != strings.Join(w.paths, "|") {
			t.Errorf("record %d: got %+v, want %+v", i, g, w)
		}
	}
}

func makeZip(t *testing.T, files map[string]string) []byte {
	t.Helper()
	var buf bytes.Buffer
	zw := zip.NewWriter(&buf)
	for name, content := range files {
		w, err := zw.Create(name)
		if err != nil {
			t.Fatal(err)
		}
		w.Write([]byte(content))
	}
	if err := zw.Close(); err != nil {
		t.Fatal(err)
	}
	return buf.Bytes()
}

// serveZips serves each archive at "/<key>".
func serveZips(t *testing.T, zips map[string][]byte) *httptest.Server {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		data, ok := zips[strings.TrimPrefix(r.URL.Path, "/")]
		if !ok {
			http.NotFound(w, r)
			return
		}
		w.Write(data)
	}))
	t.Cleanup(srv.Close)
	return srv
}

func withCamoDir(t *testing.T) string {
	dir := t.TempDir()
	old := listCamoDirs
	listCamoDirs = func() []string { return []string{dir} }
	t.Cleanup(func() { listCamoDirs = old })
	return dir
}

func readFile(t *testing.T, path string) string {
	t.Helper()
	b, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	return string(b)
}

func TestDownloadSkinDoesNotOverwriteDifferentSkin(t *testing.T) {
	camo := withCamoDir(t)
	srv := serveZips(t, map[string][]byte{
		"a": makeZip(t, map[string]string{"tiger/body_c.dds": "A", "tiger/skin.blk": "a"}),
		"b": makeZip(t, map[string]string{"tiger/body_c.dds": "B", "tiger/skin.blk": "b"}),
	})

	steps := []struct {
		file, want string
	}{
		{"a", "Installed as tiger"},
		{"a", "Already installed as tiger"},
		{"b", "Installed as tiger_2"},
		{"b", "Already installed as tiger_2"},
		{"a", "Already installed as tiger"},
	}
	for _, s := range steps {
		got, err := DownloadSkin(1, srv.URL+"/"+s.file)
		if err != nil {
			t.Fatalf("%s: %v", s.file, err)
		}
		if !strings.HasPrefix(got, s.want) {
			t.Fatalf("%s: got %q, want prefix %q", s.file, got, s.want)
		}
	}

	if got := readFile(t, filepath.Join(camo, "tiger", "body_c.dds")); got != "A" {
		t.Errorf("tiger overwritten: %q", got)
	}
	if got := readFile(t, filepath.Join(camo, "tiger_2", "body_c.dds")); got != "B" {
		t.Errorf("tiger_2 content: %q", got)
	}
}

func TestDownloadSkinFlatArchiveUsesFallbackFolder(t *testing.T) {
	camo := withCamoDir(t)
	srv := serveZips(t, map[string][]byte{
		// A single top-level file used to be treated as the root folder
		// and landed directly in UserSkins.
		"single": makeZip(t, map[string]string{"body_c.dds": "x"}),
		"flat":   makeZip(t, map[string]string{"a.dds": "1", "b.blk": "2", "__MACOSX/._a.dds": "junk"}),
	})

	if _, err := DownloadSkin(7, srv.URL+"/single"); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(camo, "wtskin-7", "body_c.dds")); err != nil {
		t.Errorf("single-file skin not in fallback folder: %v", err)
	}

	if _, err := DownloadSkin(8, srv.URL+"/flat"); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(camo, "wtskin-8", "b.blk")); err != nil {
		t.Errorf("flat skin not in fallback folder: %v", err)
	}
	if _, err := os.Stat(filepath.Join(camo, "wtskin-8", "__MACOSX")); !os.IsNotExist(err) {
		t.Errorf("__MACOSX was installed")
	}
}

func TestDownloadSkinRejectsBadResponses(t *testing.T) {
	withCamoDir(t)
	srv := serveZips(t, map[string][]byte{})

	if _, err := DownloadSkin(1, srv.URL+"/missing"); err == nil || !strings.Contains(err.Error(), "404") {
		t.Errorf("404 not reported: %v", err)
	}
	if _, err := DownloadSkin(1, "file:///C:/Windows/win.ini"); err == nil {
		t.Errorf("non-http URL accepted")
	}
}

func TestExtractArchiveBlocksTraversal(t *testing.T) {
	zipPath := filepath.Join(t.TempDir(), "evil.zip")
	os.WriteFile(zipPath, makeZip(t, map[string]string{"../../evil.txt": "x"}), 0644)
	dest := t.TempDir()

	if err := extractArchive(zipPath, dest); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(filepath.Dir(dest), "evil.txt")); !os.IsNotExist(err) {
		t.Fatalf("file escaped destination")
	}
}

func TestCopyTreeDedupForSights(t *testing.T) {
	dest := t.TempDir()
	write := func(content string) copyStats {
		src := t.TempDir()
		os.WriteFile(filepath.Join(src, "sight.blk"), []byte(content), 0644)
		stats, err := copyTree(src, dest, true)
		if err != nil {
			t.Fatal(err)
		}
		return stats
	}

	if s := write("one"); s.written != 1 {
		t.Errorf("first: %+v", s)
	}
	if s := write("one"); s.skipped != 1 || s.written != 0 {
		t.Errorf("identical: %+v", s)
	}
	if s := write("two"); s.renamed != 1 {
		t.Errorf("different: %+v", s)
	}
	if got := readFile(t, filepath.Join(dest, "sight.blk")); got != "one" {
		t.Errorf("original overwritten: %q", got)
	}
	if got := readFile(t, filepath.Join(dest, "sight_2.blk")); got != "two" {
		t.Errorf("sight_2: %q", got)
	}
}
