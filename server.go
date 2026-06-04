package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strconv"
)

const (
	Port    = 4316
	Version = "0.0.1"
)

type PostRequest struct {
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

func versionHandler(w http.ResponseWriter, r *http.Request) {
	fmt.Fprintf(w, "Version: %s", Version)
}

func postHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	postId, err := strconv.Atoi(r.PathValue("postId"))
	if err != nil {
		http.Error(w, "Invalid post ID", http.StatusBadRequest)
		return
	}

	var body PostRequest
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		http.Error(w, "Invalid JSON body", http.StatusBadRequest)
		return
	}
	defer r.Body.Close()

	log.Printf("Downloading skin for postId: %d, fileUrl: %s", postId, body.FileUrl)

	if err := DownloadSkin(postId, body.FileUrl); err != nil {
		log.Printf("Failed to download skin for postId %d: %v", postId, err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	log.Printf("Successfully installed skin for postId %d", postId)
	fmt.Fprintf(w, "Installed")
}

func startServer() {
	mux := http.NewServeMux()
	mux.HandleFunc("/version", versionHandler)
	mux.HandleFunc("/post/{postId}", postHandler)

	log.Printf("Server starting on localhost:%d", Port)
	if err := http.ListenAndServe(
		fmt.Sprintf(":%d", Port),
		corsMiddleware(mux),
	); err != nil {
		log.Fatalf("Server failed to start: %v", err)
	}
}
