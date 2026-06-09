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

// DownloadSkin downloads a skin from the given URL and extracts it to CamoDir
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

	if err = ExtractZip(tmpPath, CamoDir, fmt.Sprintf("wtskin-%d", postId), false); err != nil {
		return fmt.Errorf("Extract error for %d: %w", postId, err)
	}
	return nil
}

// DownloadSight downloads a sight zip and extracts it into every
// UserSights/all_tanks directory found across all save folders.
func DownloadSight(postId int, fileUrl string) error {
	resp, err := http.Get(fileUrl)
	if err != nil {
		return fmt.Errorf("download failed: %w", err)
	}
	defer resp.Body.Close()

	tmpFile, err := os.CreateTemp("", "wtsight-*.zip")
	if err != nil {
		return fmt.Errorf("temp file error: %w", err)
	}
	tmpPath := tmpFile.Name()
	defer os.Remove(tmpPath)

	if _, err = io.Copy(tmpFile, resp.Body); err != nil {
		tmpFile.Close()
		return fmt.Errorf("write error: %w", err)
	}
	tmpFile.Close()

	sightDirs, err := GetSightDirs()
	if err != nil {
		return fmt.Errorf("could not locate sight directories: %w", err)
	}

	fallback := fmt.Sprintf("wtsight-%d", postId)
	for _, dir := range sightDirs {
		if err := os.MkdirAll(dir, 0755); err != nil {
			return fmt.Errorf("mkdir %s: %w", dir, err)
		}
		if err := ExtractZip(tmpPath, dir, fallback, true); err != nil {
			return fmt.Errorf("extract to %s: %w", dir, err)
		}
	}
	return nil
}

// ExtractZip extracts a zip file to the CamoDir directory
func ExtractZip(src, destBase, fallbackName string, stripRoot bool) error {
	r, err := zip.OpenReader(src)
	if err != nil {
		return err
	}
	defer r.Close()

	root := detectZipRoot(r)
	dest := destBase
	if root == "" && !stripRoot {
		dest = filepath.Join(destBase, fallbackName)
	}

	cleanDest := filepath.Clean(dest)

	for _, f := range r.File {
		name := filepath.FromSlash(f.Name)
		if stripRoot && root != "" {
			rel := strings.TrimPrefix(filepath.ToSlash(f.Name), root+"/")
			if rel == "" {
				continue // skip the root dir entry itself
			}
			name = filepath.FromSlash(rel)
		}
		fpath := filepath.Join(cleanDest, name)
		cleanFpath := filepath.Clean(fpath)

		fmt.Printf("fpath: %s, cleanFpath: %s\n", fpath, cleanFpath)

		if cleanFpath != cleanDest &&
			!strings.HasPrefix(
				cleanFpath,
				cleanDest+string(os.PathSeparator),
			) {
			return fmt.Errorf("Illegal path in archive: %s", f.Name)
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

// detectZipRoot returns the single top-level directory in a zip, or "" if
// there are multiple (meaning files sit directly at the root).
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

// DeleteSkin removes a skin folder from CamoDir by name.
func DeleteSkin(name string) {
	go func() {
		path := filepath.Join(CamoDir, name)
		if err := os.RemoveAll(path); err != nil {
			fmt.Printf("Delete error for %s: %v\n", name, err)
			return
		}
		fmt.Printf("Deleted skin: %s\n", name)
	}()
}
