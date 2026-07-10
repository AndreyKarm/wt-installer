package main

import (
	"fmt"
	"io"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	goruntime "runtime"
	"sync"
	"time"
)

var (
	LogDir      string
	logFile     *os.File
	fileLogOnce sync.Once
	openLogOnce sync.Once
)

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

// openLogsFolder opens the temp log directory in the OS file explorer so
// the user can inspect what went wrong. It only opens once per run to
// avoid spamming the user if multiple errors occur.
func openLogsFolder() {
	openLogOnce.Do(func() {
		if LogDir == "" {
			return
		}
		var cmd *exec.Cmd
		switch goruntime.GOOS {
		case "windows":
			cmd = exec.Command("explorer", LogDir)
		case "darwin":
			cmd = exec.Command("open", LogDir)
		default:
			cmd = exec.Command("xdg-open", LogDir)
		}
		if err := cmd.Start(); err != nil {
			log.Printf("failed to open logs folder: %v", err)
		}
	})
}
