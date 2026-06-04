package main

import (
	"archive/zip"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
)

var GameDir string

func init() {
	exe, err := os.Executable()
	if err != nil {
		panic(fmt.Errorf("failed to resolve executable path: %w", err))
	}
	GameDir = filepath.Dir(exe) + "\\UserSkins"
}

func DownloadSkin(postId int, fileUrl string) error {
	resp, err := http.Get(fileUrl)
	if err != nil {
		return fmt.Errorf("Download failed: %w", err)
	}
	defer resp.Body.Close()

	tmpFile, err := os.CreateTemp("", "wtskin-*.zip")
	if err != nil {
		return fmt.Errorf("Temp file error: %w", err)
	}
	tmpPath := tmpFile.Name()
	defer os.Remove(tmpPath)

	if _, err = io.Copy(tmpFile, resp.Body); err != nil {
		tmpFile.Close()
		return fmt.Errorf("Write error: %w", err)
	}
	tmpFile.Close()

	if err = ExtractZip(tmpPath, fmt.Sprintf("wtskin-%d", postId)); err != nil {
		return fmt.Errorf("Extract error for %d: %w", postId, err)
	}

	return nil
}

func ExtractZip(src, fallbackName string) error {
	r, err := zip.OpenReader(src)
	if err != nil {
		return err
	}
	defer r.Close()

	root := detectZipRoot(r)
	dest := GameDir
	if root == "" {
		dest = filepath.Join(GameDir, fallbackName)
	}

	cleanDest := filepath.Clean(dest)

	for _, f := range r.File {
		fpath := filepath.Join(cleanDest, filepath.FromSlash(f.Name))
		cleanFpath := filepath.Clean(fpath)

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

func DeleteSkin(name string) {
	go func() {
		path := filepath.Join(GameDir, name)
		if err := os.RemoveAll(path); err != nil {
			fmt.Printf("Delete error for %s: %v\n", name, err)
			return
		}
		fmt.Printf("Deleted skin: %s\n", name)
	}()
}
