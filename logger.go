package main

import (
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"golang.org/x/sys/windows"
)

var (
	LogDir      string
	logFile     *os.File
	fileLogOnce sync.Once
	openLogOnce sync.Once
)

// maxLogAge is how long old log files are kept before being pruned.
const maxLogAge = 7 * 24 * time.Hour

// ensureFileLogging sets up logging to a file inside a folder in the OS
// temp directory, in addition to the existing in-memory log buffer. Safe to
// call multiple times - the setup only runs once.
func ensureFileLogging() {
	fileLogOnce.Do(func() {
		log.SetFlags(0)

		LogDir = filepath.Join(os.TempDir(), "WTInstallerLogs")
		if err := os.MkdirAll(LogDir, 0755); err != nil {
			log.Printf("could not create log directory: %v", err)
		} else {
			pruneOldLogs(LogDir, maxLogAge)

			fileName := fmt.Sprintf(
				"wtinstaller-%s.log",
				time.Now().Format("20060102-150405"),
			)
			f, err := os.OpenFile(
				filepath.Join(LogDir, fileName),
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
		log.SetOutput(&logCapture{Writer: writer})
	})
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

// openLogsFolder opens the temp log directory in the OS file explorer so
// the user can inspect what went wrong. It only opens once per run to
// avoid spamming the user if multiple errors occur.
func openLogsFolder() {
	openLogOnce.Do(func() {
		if LogDir == "" {
			return
		}
		if err := openInExplorer(LogDir); err != nil {
			log.Printf("failed to open logs folder: %v", err)
		}
	})
}

// openInExplorer opens a folder in Windows Explorer via ShellExecute. This
// avoids `cmd /c start`, which misparses paths containing & or ^ and
// flashes a console window from the GUI build.
func openInExplorer(path string) error {
	verb, err := windows.UTF16PtrFromString("open")
	if err != nil {
		return err
	}
	file, err := windows.UTF16PtrFromString(path)
	if err != nil {
		return err
	}
	return windows.ShellExecute(0, verb, file, nil, nil, windows.SW_SHOWNORMAL)
}
