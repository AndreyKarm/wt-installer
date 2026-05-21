package main

import (
	"archive/zip"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	g "github.com/AllenDang/giu"
)

func DownloadSkin(post Post) {
	go func() {
		id := post.ID
		DownloadStatus[id] = "Downloading..."
		g.Update()

		cfg, err := LoadConfig()
		if err != nil {
			DownloadStatus[id] = "Config error"
			g.Update()
			return
		}

		resp, err := http.Get(post.File.Link)
		if err != nil {
			DownloadStatus[id] = "Download failed"
			g.Update()
			return
		}
		defer resp.Body.Close()

		tmpFile, err := os.CreateTemp("", "wtskin-*.zip")
		if err != nil {
			DownloadStatus[id] = "Temp file error"
			g.Update()
			return
		}
		tmpPath := tmpFile.Name()
		defer os.Remove(tmpPath)

		if _, err = io.Copy(tmpFile, resp.Body); err != nil {
			tmpFile.Close()
			DownloadStatus[id] = "Write error"
			g.Update()
			return
		}
		tmpFile.Close()

		// Fallback folder name if the zip has no root directory
		fallbackName := strings.TrimSuffix(
			post.File.Name,
			filepath.Ext(post.File.Name),
		)

		if err = ExtractZip(tmpPath, cfg.UserSkins, fallbackName); err != nil {
			DownloadStatus[id] = "Extract error"
			fmt.Printf("Extract error for %s: %v\n", post.File.Name, err)
			g.Update()
			return
		}

		DownloadStatus[id] = "Installed"
		g.Update()
	}()
}

func detectZipRoot(r *zip.ReadCloser) string {
	var root string
	for _, f := range r.File {
		parts := strings.SplitN(filepath.ToSlash(f.Name), "/", 2)
		top := parts[0]
		if root == "" {
			root = top
		} else if top != root {
			return ""
		}
	}
	return root
}

func ExtractZip(src, baseDir, fallbackName string) error {
	r, err := zip.OpenReader(src)
	if err != nil {
		return err
	}
	defer r.Close()

	dest := baseDir
	if detectZipRoot(r) == "" {
		dest = filepath.Join(baseDir, fallbackName)
	}

	cleanDest := filepath.Clean(dest)

	for _, f := range r.File {
		fpath := filepath.Join(dest, filepath.FromSlash(f.Name))
		cleanFpath := filepath.Clean(fpath)

		// Zip slip protection
		if cleanFpath != cleanDest &&
			!strings.HasPrefix(
				cleanFpath,
				cleanDest+string(os.PathSeparator),
			) {
			return fmt.Errorf("illegal path in archive: %s", f.Name)
		}

		if f.FileInfo().IsDir() {
			if err := os.MkdirAll(fpath, f.Mode()); err != nil {
				return err
			}
			continue
		}

		if err := os.MkdirAll(filepath.Dir(fpath), 0755); err != nil {
			return err
		}

		out, err := os.OpenFile(
			fpath,
			os.O_WRONLY|os.O_CREATE|os.O_TRUNC,
			f.Mode(),
		)
		if err != nil {
			return err
		}

		rc, err := f.Open()
		if err != nil {
			out.Close()
			return err
		}

		_, err = io.Copy(out, rc)
		out.Close()
		rc.Close()
		if err != nil {
			return err
		}
	}

	return nil
}

func DeleteSkin(name string) {
	go func() {
		cfg, err := LoadConfig()
		if err != nil {
			fmt.Printf("Config error: %v\n", err)
			g.Update()
			return
		}

		path := filepath.Join(cfg.UserSkins, name)
		if err := os.RemoveAll(path); err != nil {
			fmt.Printf("Delete error for %s: %v\n", name, err)
			g.Update()
			return
		}

		fmt.Printf("Deleted skin: %s\n", name)
		g.Update()
	}()
}
