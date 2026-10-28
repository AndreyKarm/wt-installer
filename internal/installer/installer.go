// Package installer downloads skin and sight archives and installs them
// into the game folders without overwriting existing content.
package installer

import (
	"archive/zip"
	"bytes"
	"compress/bzip2"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"syscall"
	"time"

	"liotom/installer/internal/gamedirs"
	"liotom/installer/internal/installed"
	"liotom/installer/internal/logging"
)

// installMu serializes installs so two concurrent downloads can't race on
// picking the same free "_2" folder or file name.
var installMu sync.Mutex

// Where installs go. Variables so tests can point them at temp folders.
var (
	listCamoDirs    = gamedirs.CamoDirs
	refreshGameDirs = gamedirs.Refresh
	listSightDirs   = gamedirs.SightDirs
	recordInstall   = installed.Add
)

// remember records a finished install for the "already installed" marks.
// Failing to record only loses the mark, so it doesn't fail the install.
func remember(postId int, kind string, paths []string) {
	if err := recordInstall(postId, kind, paths); err != nil {
		log.Printf("Could not record install of post %d: %v", postId, err)
	}
}

var downloadClient = &http.Client{Timeout: 5 * time.Minute}

// DownloadSkin downloads a skin and installs it into every detected
// UserSkins directory. If a skin folder with the same name already exists,
// its content is compared to the incoming skin: if every incoming file is
// already there unchanged, nothing is written; otherwise the skin is
// installed under name_2, name_3, ... so an unrelated skin is never
// overwritten. Returns a short human-readable summary.
func DownloadSkin(postId int, fileUrl string) (string, error) {
	installMu.Lock()
	defer installMu.Unlock()

	camoDirs := listCamoDirs()
	if len(camoDirs) == 0 {
		refreshGameDirs()
		camoDirs = listCamoDirs()
		if len(camoDirs) == 0 {
			log.Printf("No camo directories found\n")
			logging.OpenFolder()
			return "", fmt.Errorf("no camo directories found")
		}
	}

	srcDir, root, cleanup, err := downloadAndExtract(fileUrl, "wtskin")
	if err != nil {
		logging.OpenFolder()
		return "", err
	}
	defer cleanup()

	name := root
	if name == "" {
		name = fmt.Sprintf("wtskin-%d", postId)
	}

	var notes, folders []string
	for _, dir := range camoDirs {
		if err := os.MkdirAll(dir, 0755); err != nil {
			log.Printf("mkdir %s: %v\n", dir, err)
			logging.OpenFolder()
			return "", fmt.Errorf("mkdir %s: %w", dir, err)
		}
		note, folder, err := installSkinFolder(srcDir, dir, name)
		if err != nil {
			log.Printf("Install error for %d in %s: %v\n", postId, dir, err)
			logging.OpenFolder()
			return "", fmt.Errorf("install error for %d in %s: %w", postId, dir, err)
		}
		notes = append(notes, note)
		folders = append(folders, folder)
	}
	remember(postId, "camo", folders)
	return joinUnique(notes), nil
}

// installSkinFolder copies srcDir into destBase/name, or into the first
// name_N that is either free or already holds this exact skin. It returns
// a summary and the folder that now holds the skin.
func installSkinFolder(srcDir, destBase, name string) (note, folder string, err error) {
	for i := 1; ; i++ {
		candidate := name
		if i > 1 {
			candidate = fmt.Sprintf("%s_%d", name, i)
		}
		target := filepath.Join(destBase, candidate)

		info, err := os.Stat(target)
		if os.IsNotExist(err) {
			if _, err := copyTree(srcDir, target, false); err != nil {
				return "", "", err
			}
			log.Printf("Installed skin into %s", target)
			if i > 1 {
				return fmt.Sprintf("Installed as %s (a different skin named %s already exists)", candidate, name), target, nil
			}
			return fmt.Sprintf("Installed as %s", candidate), target, nil
		}
		if err != nil {
			return "", "", err
		}
		if !info.IsDir() {
			continue
		}

		same, err := treeContains(target, srcDir)
		if err != nil {
			return "", "", err
		}
		if same {
			log.Printf("Skin already installed at %s, skipping", target)
			return fmt.Sprintf("Already installed as %s", candidate), target, nil
		}
		log.Printf("Folder %s holds a different skin, trying next name", target)
	}
}

// DownloadSight downloads a sight zip and extracts it into every
// UserSights/all_tanks directory found across all save folders. If a file
// with the same name already exists in a destination, its content is
// compared to the incoming file: identical content is left untouched,
// while differing content is written under name_2, name_3, etc.
func DownloadSight(postId int, fileUrl string) (string, error) {
	installMu.Lock()
	defer installMu.Unlock()

	sightDirs, err := listSightDirs()
	if err != nil || len(sightDirs) == 0 {
		log.Printf("could not locate sight directories: %v\n", err)
		logging.OpenFolder()
		return "", fmt.Errorf("could not locate sight directories: %v", err)
	}

	srcDir, _, cleanup, err := downloadAndExtract(fileUrl, "wtsight")
	if err != nil {
		logging.OpenFolder()
		return "", err
	}
	defer cleanup()

	var total copyStats
	for _, dir := range sightDirs {
		if err := os.MkdirAll(dir, 0755); err != nil {
			log.Printf("mkdir %s: %v\n", dir, err)
			logging.OpenFolder()
			return "", fmt.Errorf("mkdir %s: %w", dir, err)
		}
		stats, err := copyTree(srcDir, dir, true)
		if err != nil {
			log.Printf("install sight %d to %s: %v\n", postId, dir, err)
			logging.OpenFolder()
			return "", fmt.Errorf("install to %s: %w", dir, err)
		}
		total.written += stats.written
		total.skipped += stats.skipped
		total.renamed += stats.renamed
		total.paths = append(total.paths, stats.paths...)
	}
	remember(postId, "sight", total.paths)

	msg := fmt.Sprintf("Installed %d file(s)", total.written)
	if total.renamed > 0 {
		msg += fmt.Sprintf(", %d renamed to avoid overwriting a different sight", total.renamed)
	}
	if total.skipped > 0 {
		msg += fmt.Sprintf(", %d already present", total.skipped)
	}
	return msg, nil
}

// downloadAndExtract fetches fileUrl and extracts it into a temp folder.
// srcDir is the folder holding the archive contents, with a single
// top-level folder (root) stripped if the archive has one. The caller must
// call cleanup.
func downloadAndExtract(fileUrl, prefix string) (srcDir, root string, cleanup func(), err error) {
	zipPath, err := fetchArchive(fileUrl, prefix)
	if err != nil {
		return "", "", nil, err
	}
	defer os.Remove(zipPath)

	tmpDir, err := os.MkdirTemp("", prefix+"-extract-*")
	if err != nil {
		return "", "", nil, fmt.Errorf("temp dir error: %w", err)
	}
	cleanup = func() { os.RemoveAll(tmpDir) }

	if err := extractArchive(zipPath, tmpDir); err != nil {
		cleanup()
		log.Printf("extract error: %v\n", err)
		return "", "", nil, fmt.Errorf("extract error: %w", err)
	}

	srcDir = tmpDir
	entries, err := os.ReadDir(tmpDir)
	if err != nil {
		cleanup()
		return "", "", nil, err
	}
	if len(entries) == 0 {
		cleanup()
		return "", "", nil, fmt.Errorf("archive is empty")
	}
	if len(entries) == 1 && entries[0].IsDir() {
		root = entries[0].Name()
		srcDir = filepath.Join(tmpDir, root)
	}
	return srcDir, root, cleanup, nil
}

// fetchArchive downloads fileUrl into a temp file and returns its path.
func fetchArchive(fileUrl, prefix string) (string, error) {
	u, err := url.Parse(fileUrl)
	if err != nil || (u.Scheme != "https" && u.Scheme != "http") || u.Host == "" {
		log.Printf("Invalid file URL: %q\n", fileUrl)
		return "", fmt.Errorf("invalid file URL: %q", fileUrl)
	}

	resp, err := downloadClient.Get(fileUrl)
	if err != nil {
		log.Printf("Download failed: %v\n", err)
		return "", fmt.Errorf("download failed: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		log.Printf("Download failed: %s returned %s\n", fileUrl, resp.Status)
		return "", fmt.Errorf("download failed: server returned %s", resp.Status)
	}

	tmpFile, err := os.CreateTemp("", prefix+"-*.zip")
	if err != nil {
		log.Printf("Temp file error: %v\n", err)
		return "", fmt.Errorf("temp file error: %w", err)
	}
	if _, err := io.Copy(tmpFile, resp.Body); err != nil {
		tmpFile.Close()
		os.Remove(tmpFile.Name())
		log.Printf("Write error: %v\n", err)
		return "", fmt.Errorf("write error: %w", err)
	}
	tmpFile.Close()
	return tmpFile.Name(), nil
}

func init() {
	// Go's archive/zip only natively supports Store (0) and Deflate (8).
	// Some skin archives are built with 7-Zip using BZip2 (12), LZMA (14),
	// PPMd (98) or Deflate64 (9). BZip2 has a stdlib decoder we can wire
	// in directly; the others fall back to an external tool below.
	zip.RegisterDecompressor(12, func(r io.Reader) io.ReadCloser {
		return io.NopCloser(bzip2.NewReader(r))
	})
}

var nativelySupportedMethods = map[uint16]bool{
	zip.Store:   true,
	zip.Deflate: true,
	12:          true, // bzip2 (registered above)
}

// hasUnsupportedMethod reports the first compression method in the archive
// that Go can't decompress on its own (e.g. LZMA/PPMd/Deflate64 from 7-Zip).
func hasUnsupportedMethod(r *zip.ReadCloser) (uint16, bool) {
	for _, f := range r.File {
		if !nativelySupportedMethods[f.Method] {
			return f.Method, true
		}
	}
	return 0, false
}

// runExternalUnzip tries 7-Zip, then tar (bsdtar on Windows 10 1803+/macOS
// ships built-in and supports more zip compression methods than Go's
// stdlib), extracting src fully into destDir.
func runExternalUnzip(src, destDir string) error {
	if p, err := exec.LookPath("7z"); err == nil {
		cmd := exec.Command(p, "x", "-y", "-o"+destDir, src)
		cmd.SysProcAttr = &syscall.SysProcAttr{HideWindow: true}
		if err := cmd.Run(); err == nil {
			return nil
		}
	}
	if p, err := exec.LookPath("tar"); err == nil {
		cmd := exec.Command(p, "-xf", src, "-C", destDir)
		cmd.SysProcAttr = &syscall.SysProcAttr{HideWindow: true}
		if err := cmd.Run(); err == nil {
			return nil
		}
	}
	return fmt.Errorf("no working extractor found for this archive's compression method (tried 7z, tar) - please install 7-Zip")
}

// extractArchive extracts the zip at src into the empty directory dest,
// sanitizing names for Windows and dropping macOS metadata entries.
func extractArchive(src, dest string) error {
	r, err := zip.OpenReader(src)
	if err != nil {
		return err
	}
	defer r.Close()

	if method, unsupported := hasUnsupportedMethod(r); unsupported {
		log.Printf(
			"Archive uses unsupported compression method %d, falling back to external extractor\n",
			method,
		)
		rawDir, err := os.MkdirTemp("", "wtextract-raw-*")
		if err != nil {
			return err
		}
		defer os.RemoveAll(rawDir)
		if err := runExternalUnzip(src, rawDir); err != nil {
			return err
		}
		_, err = copyTree(rawDir, dest, false)
		return err
	}

	cleanDest := filepath.Clean(dest)
	for _, f := range r.File {
		name := sanitizeWindowsPath(strings.TrimSuffix(filepath.ToSlash(f.Name), "/"))
		if isJunkEntry(name) {
			continue
		}
		fpath := filepath.Clean(filepath.Join(cleanDest, filepath.FromSlash(name)))
		if !strings.HasPrefix(fpath, cleanDest+string(os.PathSeparator)) {
			log.Printf("Illegal path in archive: %s", f.Name)
			return fmt.Errorf("illegal path in archive: %s", f.Name)
		}

		if f.FileInfo().IsDir() {
			if err := os.MkdirAll(fpath, 0755); err != nil {
				return err
			}
			continue
		}
		if err := os.MkdirAll(filepath.Dir(fpath), 0755); err != nil {
			return err
		}
		if err := writeZipFile(f, fpath); err != nil {
			return err
		}
	}
	return nil
}

func writeZipFile(f *zip.File, path string) error {
	rc, err := f.Open()
	if err != nil {
		return err
	}
	defer rc.Close()

	// Fixed permissions: modes stored in the zip can be 0 or read-only,
	// which on Windows would block reinstalling over the file later.
	out, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0644)
	if err != nil {
		return err
	}
	if _, err := io.Copy(out, rc); err != nil {
		out.Close()
		return err
	}
	return out.Close()
}

// isJunkEntry reports archive entries that are OS metadata, not content.
func isJunkEntry(slashPath string) bool {
	for _, part := range strings.Split(slashPath, "/") {
		if part == "__MACOSX" || part == ".DS_Store" {
			return true
		}
	}
	return false
}

type copyStats struct {
	written, skipped, renamed int
	// paths lists every destination file now holding incoming content,
	// including identical files that were skipped.
	paths []string
}

// copyTree copies everything under srcDir into destDir, applying the same
// Windows filename sanitization used for native extraction. When
// dedupIdentical is true, files that already exist at the destination are
// compared by content: identical files are left alone, differing files are
// written under a "_2", "_3", ... suffix.
func copyTree(srcDir, destDir string, dedupIdentical bool) (copyStats, error) {
	var stats copyStats
	err := filepath.Walk(srcDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		rel, err := filepath.Rel(srcDir, path)
		if err != nil {
			return err
		}
		if rel == "." {
			return os.MkdirAll(destDir, 0755)
		}
		slashRel := sanitizeWindowsPath(filepath.ToSlash(rel))
		if isJunkEntry(slashRel) {
			if info.IsDir() {
				return filepath.SkipDir
			}
			return nil
		}
		target := filepath.Join(destDir, filepath.FromSlash(slashRel))

		if info.IsDir() {
			return os.MkdirAll(target, 0755)
		}
		if err := os.MkdirAll(filepath.Dir(target), 0755); err != nil {
			return err
		}

		data, err := os.ReadFile(path)
		if err != nil {
			return err
		}

		writeTarget := target
		if dedupIdentical {
			finalPath, skip, derr := dedupeTargetPath(target, data)
			if derr != nil {
				return derr
			}
			if skip {
				log.Printf("Skipping identical file: %s", finalPath)
				stats.skipped++
				stats.paths = append(stats.paths, finalPath)
				return nil
			}
			if finalPath != target {
				log.Printf("File differs, writing as: %s", finalPath)
				stats.renamed++
			}
			writeTarget = finalPath
		}

		if err := os.WriteFile(writeTarget, data, 0644); err != nil {
			return err
		}
		stats.written++
		stats.paths = append(stats.paths, writeTarget)
		return nil
	})
	return stats, err
}

// treeContains reports whether every file under srcDir also exists under
// dir with identical content. Extra files in dir are ignored, so a skin the
// user tweaked by adding files still counts as the same skin.
func treeContains(dir, srcDir string) (bool, error) {
	same := true
	err := filepath.Walk(srcDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.IsDir() {
			return nil
		}
		rel, err := filepath.Rel(srcDir, path)
		if err != nil {
			return err
		}
		other := filepath.Join(dir, rel)
		otherInfo, err := os.Stat(other)
		if err != nil || otherInfo.IsDir() || otherInfo.Size() != info.Size() {
			same = false
			return filepath.SkipAll
		}
		a, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		b, err := os.ReadFile(other)
		if err != nil {
			return err
		}
		if !bytes.Equal(a, b) {
			same = false
			return filepath.SkipAll
		}
		return nil
	})
	return same, err
}

// dedupeTargetPath decides where newData should actually be written given a
// desired target path. If target doesn't exist, it's used as-is. If it
// exists with identical content, skip=true is returned along with the
// matching path (caller should write nothing). If it exists with different
// content, target_2, target_3, ...
// are tried until either a free name or a byte-identical match is found.
func dedupeTargetPath(target string, newData []byte) (finalPath string, skip bool, err error) {
	ext := filepath.Ext(target)
	base := strings.TrimSuffix(target, ext)

	for i := 1; ; i++ {
		candidate := target
		if i > 1 {
			candidate = fmt.Sprintf("%s_%d%s", base, i, ext)
		}

		existing, readErr := os.ReadFile(candidate)
		if readErr != nil {
			if os.IsNotExist(readErr) {
				return candidate, false, nil
			}
			return "", false, readErr
		}

		if bytes.Equal(existing, newData) {
			return candidate, true, nil
		}
		// content differs, try the next suffix
	}
}

// DeleteSkin removes a skin folder from every detected CamoDir by name.
func DeleteSkin(name string) {
	go func() {
		for _, dir := range listCamoDirs() {
			path := filepath.Join(dir, name)
			if err := os.RemoveAll(path); err != nil {
				log.Printf("Delete error for %s in %s: %v\n", name, dir, err)
				continue
			}
			log.Printf("Deleted skin: %s from %s\n", name, dir)
		}
	}()
}

// joinUnique joins notes with "; ", dropping repeats (the same outcome in
// several game installs is reported once).
func joinUnique(notes []string) string {
	seen := make(map[string]bool)
	var out []string
	for _, n := range notes {
		if !seen[n] {
			seen[n] = true
			out = append(out, n)
		}
	}
	return strings.Join(out, "; ")
}

var invalidWinChars = strings.NewReplacer(
	"<", "_", ">", "_", ":", "_", `"`, "_", "|", "_", "?", "_", "*", "_",
)

// sanitizeWindowsPath replaces characters that are illegal in Windows
// file/folder names and strips trailing dots/spaces from each segment.
func sanitizeWindowsPath(p string) string {
	parts := strings.Split(filepath.ToSlash(p), "/")
	for i, part := range parts {
		part = invalidWinChars.Replace(part)
		part = strings.TrimRight(part, " .")
		if part == "" {
			part = "_"
		}
		parts[i] = part
	}
	return strings.Join(parts, "/")
}
