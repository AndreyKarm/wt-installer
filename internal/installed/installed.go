// Package installed remembers which War Thunder Live posts were installed,
// so the extension can mark them on the site. Records live in
// %APPDATA%\WTInstaller\installed.json; temp would be wiped by cleanup tools.
package installed

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strconv"
	"sync"
	"time"
)

type Record struct {
	Type        string    `json:"type"`
	InstalledAt time.Time `json:"installedAt"`
	// Paths are the skin folders or sight files the post ended up in,
	// including ones that were already present with identical content.
	Paths []string `json:"paths"`
}

type file struct {
	Version int               `json:"version"`
	Posts   map[string]Record `json:"posts"`
}

var (
	mu sync.Mutex
	// storePath is a variable so tests can use a temp file.
	storePath = defaultPath()
)

func defaultPath() string {
	dir, err := os.UserConfigDir()
	if err != nil {
		dir = os.TempDir()
	}
	return filepath.Join(dir, "WTInstaller", "installed.json")
}

func load() (file, error) {
	f := file{Version: 1, Posts: map[string]Record{}}
	data, err := os.ReadFile(storePath)
	if os.IsNotExist(err) {
		return f, nil
	}
	if err != nil {
		return f, err
	}
	if err := json.Unmarshal(data, &f); err != nil {
		return f, fmt.Errorf("parse %s: %w", storePath, err)
	}
	if f.Posts == nil {
		f.Posts = map[string]Record{}
	}
	return f, nil
}

// save writes through a temp file and rename so a crash mid-write can't
// leave a truncated store behind.
func save(f file) error {
	if err := os.MkdirAll(filepath.Dir(storePath), 0755); err != nil {
		return err
	}
	data, err := json.MarshalIndent(f, "", "  ")
	if err != nil {
		return err
	}
	tmp := storePath + ".tmp"
	if err := os.WriteFile(tmp, data, 0644); err != nil {
		return err
	}
	return os.Rename(tmp, storePath)
}

// Add records that postId was installed into paths, replacing any earlier
// record for the same post.
func Add(postId int, kind string, paths []string) error {
	mu.Lock()
	defer mu.Unlock()

	f, err := load()
	if err != nil {
		// A corrupt store shouldn't block installs; start over.
		log.Printf("Installed list unreadable, starting a new one: %v", err)
		f = file{Version: 1, Posts: map[string]Record{}}
	}
	f.Posts[strconv.Itoa(postId)] = Record{
		Type:        kind,
		InstalledAt: time.Now().UTC(),
		Paths:       paths,
	}
	return save(f)
}

// Lookup returns the record for postId and the set of its paths that other
// recorded posts also point at (sights deduplicated against an identical
// file), which an uninstall must keep.
func Lookup(postId int) (rec Record, shared map[string]bool, ok bool, err error) {
	mu.Lock()
	defer mu.Unlock()

	f, err := load()
	if err != nil {
		return Record{}, nil, false, err
	}
	id := strconv.Itoa(postId)
	rec, ok = f.Posts[id]
	if !ok {
		return Record{}, nil, false, nil
	}
	mine := make(map[string]bool, len(rec.Paths))
	for _, p := range rec.Paths {
		mine[p] = true
	}
	shared = map[string]bool{}
	for other, r := range f.Posts {
		if other == id {
			continue
		}
		for _, p := range r.Paths {
			if mine[p] {
				shared[p] = true
			}
		}
	}
	return rec, shared, true, nil
}

// Remove deletes the record for postId, if any.
func Remove(postId int) error {
	mu.Lock()
	defer mu.Unlock()

	f, err := load()
	if err != nil {
		return err
	}
	id := strconv.Itoa(postId)
	if _, ok := f.Posts[id]; !ok {
		return nil
	}
	delete(f.Posts, id)
	return save(f)
}

// Status returns post id → type for every recorded post that still has at
// least one of its paths on disk. Records whose files are gone (deleted by
// the user, or on a drive that isn't connected) are left in the store but
// not reported.
func Status() (map[string]string, error) {
	mu.Lock()
	defer mu.Unlock()

	f, err := load()
	if err != nil {
		return nil, err
	}
	out := make(map[string]string, len(f.Posts))
	for id, rec := range f.Posts {
		for _, p := range rec.Paths {
			if _, err := os.Stat(p); err == nil {
				out[id] = rec.Type
				break
			}
		}
	}
	return out, nil
}
