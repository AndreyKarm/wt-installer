package installed

import (
	"os"
	"path/filepath"
	"testing"
)

func useTempStore(t *testing.T) {
	old := storePath
	storePath = filepath.Join(t.TempDir(), "sub", "installed.json")
	t.Cleanup(func() { storePath = old })
}

func TestStatusReportsOnlyPostsWithFilesOnDisk(t *testing.T) {
	useTempStore(t)
	dir := t.TempDir()
	kept := filepath.Join(dir, "tiger")
	gone := filepath.Join(dir, "panther")
	os.Mkdir(kept, 0755)
	os.Mkdir(gone, 0755)

	if err := Add(1, "camo", []string{kept}); err != nil {
		t.Fatal(err)
	}
	if err := Add(2, "camo", []string{gone}); err != nil {
		t.Fatal(err)
	}
	if err := Add(3, "sight", []string{filepath.Join(dir, "missing.blk"), kept}); err != nil {
		t.Fatal(err)
	}
	os.Remove(gone)

	got, err := Status()
	if err != nil {
		t.Fatal(err)
	}
	want := map[string]string{"1": "camo", "3": "sight"}
	if len(got) != len(want) {
		t.Fatalf("got %v, want %v", got, want)
	}
	for id, kind := range want {
		if got[id] != kind {
			t.Errorf("post %s: got %q, want %q", id, got[id], kind)
		}
	}
}

func TestEmptyAndCorruptStore(t *testing.T) {
	useTempStore(t)

	got, err := Status()
	if err != nil || len(got) != 0 {
		t.Fatalf("missing store: %v, %v", got, err)
	}

	os.MkdirAll(filepath.Dir(storePath), 0755)
	os.WriteFile(storePath, []byte("{not json"), 0644)
	if _, err := Status(); err == nil {
		t.Errorf("corrupt store not reported")
	}

	dir := t.TempDir()
	if err := Add(5, "camo", []string{dir}); err != nil {
		t.Fatalf("Add over corrupt store: %v", err)
	}
	if got, _ := Status(); got["5"] != "camo" {
		t.Errorf("after recovery: %v", got)
	}
}
