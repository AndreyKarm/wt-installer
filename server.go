package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os/exec"
	"path/filepath"
	goruntime "runtime"
)

const (
	Port    = 4316
	Version = "0.0.1"
)

type PostRequest struct {
	Type    string `json:"type"`
	PostId  int    `json:"postId"`
	FileUrl string `json:"fileUrl"`
}

func corsMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
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
	fmt.Fprintf(w, "Version: %s", Version)
}

func openFolderHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	target := r.URL.Query().Get("target")
	var path string
	switch target {
	case "camo":
		if CamoDir == "" {
			http.Error(w, "Camo folder not found", http.StatusInternalServerError)
			return
		}
		path = CamoDir
	case "sight":
		//TODO
	case "installer":
		path = filepath.Dir(getExecutablePath())
	default:
		http.Error(w, "Invalid target", http.StatusBadRequest)
		return
	}

	var cmd *exec.Cmd
	switch goruntime.GOOS {
	case "windows":
		cmd = exec.Command("explorer", path)
	case "darwin":
		cmd = exec.Command("open", path)
	default:
		cmd = exec.Command("xdg-open", path)
	}

	if err := cmd.Start(); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	fmt.Fprintf(w, "OK")
}

func downloadHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var body PostRequest
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		fmt.Fprintf(w, "Invalid JSON body: %v", err)
		http.Error(w, "Invalid JSON body", http.StatusBadRequest)
		return
	}
	defer r.Body.Close()

	log.Printf("Downloading skin for postId: %d, fileUrl: %s", body.PostId, body.FileUrl)

	switch body.Type {
	case "camo":
		if err := DownloadSkin(body.PostId, body.FileUrl); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		log.Printf("Successfully installed skin for postId %d", body.PostId)
		fmt.Fprintf(w, "Installed")

	default:
		http.Error(w, "Invalid type", http.StatusBadRequest)
		return
	}
}

func startServer() {
	mux := http.NewServeMux()
	mux.HandleFunc("/health", healthHandler)
	mux.HandleFunc("/version", versionHandler)
	mux.HandleFunc("/openfolder", openFolderHandler)
	mux.HandleFunc("/download", downloadHandler)

	log.Printf("Server starting on localhost:%d", Port)
	if err := http.ListenAndServe(
		fmt.Sprintf(":%d", Port),
		corsMiddleware(mux),
	); err != nil {
		log.Fatalf("Server failed to start: %v", err)
	}
}
