// Package logging sends the standard log output to a file in the temp
// directory and to an in-memory buffer served at /logs.
package logging

import (
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"liotom/installer/internal/winutil"
)

type Entry struct {
	Time    string `json:"time"`
	Message string `json:"message"`
}

const (
	// maxAge is how long old log files are kept before being pruned.
	maxAge     = 7 * 24 * time.Hour
	maxEntries = 200
)

var (
	dir      string
	logFile  *os.File
	initOnce sync.Once
	openOnce sync.Once

	bufMu  sync.Mutex
	buffer []Entry
)

type capture struct{ io.Writer }

func (c *capture) Write(p []byte) (int, error) {
	bufMu.Lock()
	buffer = append(buffer, Entry{
		Time:    time.Now().Format("15:04:05"),
		Message: string(p),
	})
	if len(buffer) > maxEntries {
		buffer = buffer[1:]
	}
	bufMu.Unlock()
	return c.Writer.Write(p)
}

// Init sets up logging to a file inside a folder in the OS temp directory,
// in addition to the in-memory buffer. Safe to call multiple times - the
// setup only runs once.
func Init() {
	initOnce.Do(func() {
		log.SetFlags(0)

		dir = filepath.Join(os.TempDir(), "WTInstallerLogs")
		if err := os.MkdirAll(dir, 0755); err != nil {
			log.Printf("could not create log directory: %v", err)
		} else {
			pruneOldLogs(dir, maxAge)

			fileName := fmt.Sprintf(
				"wtinstaller-%s.log",
				time.Now().Format("20060102-150405"),
			)
			f, err := os.OpenFile(
				filepath.Join(dir, fileName),
				os.O_CREATE|os.O_WRONLY|os.O_APPEND,
				0644,
			)
			if err != nil {
				log.Printf("could not open log file: %v", err)
			} else {
				logFile = f
			}
		}

		writer := log.Writer()
		if logFile != nil {
			writer = io.MultiWriter(log.Writer(), logFile)
		}
		log.SetOutput(&capture{Writer: writer})
	})
}

// Dir returns the log folder, or "" if Init hasn't run or failed.
func Dir() string {
	return dir
}

// Entries returns a copy of the buffered log lines (never nil, so it
// encodes as [] for the popup).
func Entries() []Entry {
	bufMu.Lock()
	defer bufMu.Unlock()
	return append([]Entry{}, buffer...)
}

// Clear empties the in-memory buffer; the log file is kept.
func Clear() {
	bufMu.Lock()
	buffer = nil
	bufMu.Unlock()
}

// pruneOldLogs removes files in dir whose name matches the
// "wtinstaller-*.log" pattern and are older than maxAge. Errors are
// swallowed (logged) since a stale log file cluttering temp is not fatal.
func pruneOldLogs(dir string, maxAge time.Duration) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		log.Printf("could not read log directory for pruning: %v", err)
		return
	}

	cutoff := time.Now().Add(-maxAge)
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		name := entry.Name()
		if !strings.HasPrefix(name, "wtinstaller-") || !strings.HasSuffix(name, ".log") {
			continue
		}

		info, err := entry.Info()
		if err != nil {
			continue
		}
		if info.ModTime().Before(cutoff) {
			path := filepath.Join(dir, name)
			if err := os.Remove(path); err != nil {
				log.Printf("could not remove old log file %s: %v", path, err)
			}
		}
	}
}

// OpenFolder opens the log directory in Explorer so the user can inspect
// what went wrong. It only opens once per run to avoid spamming the user
// if multiple errors occur.
func OpenFolder() {
	openOnce.Do(func() {
		if dir == "" {
			return
		}
		if err := winutil.OpenInExplorer(dir); err != nil {
			log.Printf("failed to open logs folder: %v", err)
		}
	})
}
