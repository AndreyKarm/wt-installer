// Package server is the local HTTP API used by the browser extension.
package server

import (
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"liotom/installer/internal/app"
	"liotom/installer/internal/gamedirs"
	"liotom/installer/internal/installed"
	"liotom/installer/internal/installer"
	"liotom/installer/internal/logging"
	"liotom/installer/internal/winutil"
)

type PostRequest struct {
	Type    string `json:"type"`
	PostId  int    `json:"postId"`
	FileUrl string `json:"fileUrl"`
}

// isAllowedOrigin limits callers to the extension and War Thunder Live.
// Without this any website open in the browser could make the app install
// arbitrary archives or read the logs. Requests with no Origin (curl, local
// tools) are allowed; browsers always send one on cross-origin POSTs.
func isAllowedOrigin(origin string) bool {
	return origin == "" ||
		origin == "https://live.warthunder.com" ||
		strings.HasPrefix(origin, "moz-extension://") ||
		strings.HasPrefix(origin, "chrome-extension://")
}

func corsMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		origin := r.Header.Get("Origin")
		if !isAllowedOrigin(origin) {
			log.Printf("Rejected request from origin %s\n", origin)
			http.Error(w, "Forbidden origin", http.StatusForbidden)
			return
		}
		if origin != "" {
			w.Header().Set("Access-Control-Allow-Origin", origin)
			w.Header().Set("Vary", "Origin")
		}
		w.Header().Set("Access-Control-Allow-Methods", "POST, GET, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type")

		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}

		next.ServeHTTP(w, r)
	})
}

func healthHandler(w http.ResponseWriter, r *http.Request) {
	fmt.Fprintf(w, "OK")
}

func versionHandler(w http.ResponseWriter, r *http.Request) {
	fmt.Fprintf(w, "Version: %s", app.Version)
}

var errInvalidTarget = errors.New("invalid target")

// OpenFolder opens the folders for target ("camo", "sight", "installer" or
// "logs") in Explorer, one window per folder: with several game installs or
// save folders each one is opened separately. Used by /openfolder and the
// tray menu.
func OpenFolder(target string) error {
	var paths []string
	switch target {
	case "camo":
		paths = gamedirs.CamoDirs()
		if len(paths) == 0 {
			gamedirs.Refresh()
			paths = gamedirs.CamoDirs()
		}
		if len(paths) == 0 {
			log.Printf("Camo folder not found\n")
			return errors.New("camo folder not found")
		}
	case "sight":
		sightDirs, err := gamedirs.SightDirs()
		if err != nil || len(sightDirs) == 0 {
			log.Printf("Sight folder not found: %v\n", err)
			return errors.New("sight folder not found")
		}
		paths = sightDirs
	case "installer":
		paths = []string{filepath.Dir(winutil.ExecutablePath())}
	case "logs":
		if logging.Dir() == "" {
			log.Printf("Log folder not found\n")
			return errors.New("log folder not found")
		}
		paths = []string{logging.Dir()}
	default:
		log.Printf("Invalid target: %s\n", target)
		return errInvalidTarget
	}

	for _, path := range paths {
		// UserSkins / all_tanks may not exist until the first install.
		if err := os.MkdirAll(path, 0755); err != nil {
			log.Printf("mkdir %s: %v\n", path, err)
		}
		log.Printf("Opening %s folder: %s\n", target, path)
		if err := winutil.OpenInExplorer(path); err != nil {
			log.Printf("Open folder %s: %v\n", path, err)
			return err
		}
	}
	return nil
}

func openFolderHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		log.Printf("Method not allowed: %s\n", r.Method)
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	if err := OpenFolder(r.URL.Query().Get("target")); err != nil {
		status := http.StatusInternalServerError
		if errors.Is(err, errInvalidTarget) {
			status = http.StatusBadRequest
		}
		http.Error(w, err.Error(), status)
		return
	}
	fmt.Fprintf(w, "OK")
}

func uninstallHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var body struct {
		PostId int `json:"postId"`
	}
	r.Body = http.MaxBytesReader(w, r.Body, 64<<10)
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil || body.PostId == 0 {
		log.Printf("Invalid uninstall body: %v\n", err)
		http.Error(w, "Invalid JSON body", http.StatusBadRequest)
		return
	}

	summary, err := installer.Uninstall(body.PostId)
	if err != nil {
		log.Printf("Uninstall %d error: %v\n", body.PostId, err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	log.Printf("Uninstalled post %d: %s", body.PostId, summary)
	fmt.Fprint(w, summary)
}

func downloadHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var body PostRequest
	r.Body = http.MaxBytesReader(w, r.Body, 64<<10)
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		log.Printf("Invalid JSON body: %v\n", err)
		http.Error(w, "Invalid JSON body", http.StatusBadRequest)
		return
	}

	log.Printf("Downloading for Post ID: %d, fileUrl: %s", body.PostId, body.FileUrl)

	var (
		summary string
		err     error
	)
	switch body.Type {
	case "camo":
		summary, err = installer.DownloadSkin(body.PostId, body.FileUrl)
	case "sight":
		summary, err = installer.DownloadSight(body.PostId, body.FileUrl)
	default:
		log.Printf("Invalid type: %s\n", body.Type)
		http.Error(w, "Invalid type", http.StatusBadRequest)
		return
	}
	if err != nil {
		log.Printf("Install %s error: %v\n", body.Type, err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	log.Printf("Post ID %d: %s", body.PostId, summary)
	fmt.Fprint(w, summary)
}

// installedHandler returns {"<postId>": "camo"|"sight"} for posts whose
// installed files are still on disk.
func installedHandler(w http.ResponseWriter, r *http.Request) {
	status, err := installed.Status()
	if err != nil {
		log.Printf("Installed list error: %v\n", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(status)
}

// Start serves the API on 127.0.0.1:app.Port. It blocks, so run it in a
// goroutine.
func Start() {
	mux := http.NewServeMux()

	mux.HandleFunc("/health", healthHandler)
	mux.HandleFunc("/version", versionHandler)
	mux.HandleFunc("/openfolder", openFolderHandler)
	mux.HandleFunc("/download", downloadHandler)
	mux.HandleFunc("/installed", installedHandler)
	mux.HandleFunc("/uninstall", uninstallHandler)
	mux.HandleFunc("/logs", func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode(logging.Entries())
	})

	mux.HandleFunc("/logs/clear", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}
		logging.Clear()
		fmt.Fprint(w, "OK")
	})

	if err := http.ListenAndServe(fmt.Sprintf("127.0.0.1:%d", app.Port), corsMiddleware(mux)); err != nil {
		log.Printf("Server error: %v", err)
		logging.OpenFolder()
	}
}
