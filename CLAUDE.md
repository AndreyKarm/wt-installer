# CLAUDE.md

WT Installer installs War Thunder Live (live.warthunder.com) camouflages and sights into the local game folders with one click from the browser. It has two parts that talk over HTTP on `127.0.0.1:4316`:

1. **Go tray app** (`cmd/` + `internal/`): a Windows-only system-tray program that runs a local HTTP server, downloads zips and extracts them into the game folders.
2. **Firefox extension** (`extension/`, Manifest V3): injects "Install Camo" / "Install Sight" buttons into War Thunder Live posts and sends requests to the tray app.

## Structure

Module `liotom/installer` (the name differs from the repo name).

| Path | Purpose |
|---|---|
| `cmd/wt-installer/main.go` | Entry point. `logging.Init()`, single-instance mutex (`WTInstallerServer`), `gamedirs.Refresh()`, tray menu (Auto-start toggle, update status, Quit), starts `server.Start()` and the update check. The generated `rsrc.syso` (exe icon) must be in this folder. |
| `internal/app` | Shared constants: `Name` (also the auto-start registry value name), `Version`, `Port`. |
| `internal/server` | HTTP server and handlers, CORS/origin allow-list (`isAllowedOrigin`, tested in `server_test.go`). |
| `internal/installer` | `DownloadSkin` / `DownloadSight`. Archives are downloaded and extracted to a temp folder first (`downloadAndExtract` → `extractArchive`), then copied into place (`copyTree`). Includes Windows path sanitizing, zip-slip check, folder dedup for skins, per-file dedup for sights, and fallback to external `7z`/`tar` for compression methods Go can't read (LZMA, PPMd, Deflate64). BZip2 is registered natively. `installer_test.go` covers install/dedup, bad downloads and path traversal. |
| `internal/gamedirs` | Finds game installs: Gaijin Launcher (`HKCU\SOFTWARE\Gaijin\WarThunder\InstallDir`) and Steam (parses `libraryfolders.vdf` for app ID `236390`). `CamoDirs()` returns `<game>\UserSkins`. `SightDirs()` returns `Documents\My Games\WarThunder\Saves\<numeric id>\production\UserSights\all_tanks`. |
| `internal/update` | Self-update from GitHub Releases (`AndreyKarm/wt-installer`). `Check()` compares the latest tag with `app.Version`. `Apply()` downloads the `.exe` asset and starts a hidden `_update.bat` that swaps it in; the caller then quits the tray. |
| `internal/logging` | Logs to `%TEMP%\WTInstallerLogs\wtinstaller-<timestamp>.log` and to an in-memory buffer (200 entries) served at `/logs`. Prunes logs older than 7 days. `OpenFolder()` opens the log folder at most once per run. |
| `internal/winutil` | `OpenInExplorer` (ShellExecute), `ExecutablePath`, auto-start via `HKCU\...\CurrentVersion\Run`. |
| `media/` | `icon.png` (source), `icon.ico` (generated, committed) and `media.go`, which embeds the icon as `media.Icon`. |
| `extension/content.js` | Content script on live.warthunder.com: adds buttons to feed items (`.feed_item[post_id]`) and the lightbox (`#clb .wrapper[post_id]`) through `MutationObserver`s. Shows toasts. |
| `extension/background.js` | Relays content-script messages (`wt_install`, `wt_server_version`, `wt_open_folder`, `wt_*_log*`) to the server and keeps extension logs in memory. |
| `extension/popup/` | Popup: extension and server versions, buttons to open folders, server and extension log viewer. |
| `build.bat`, `archive.bat` | Local build script and the script that zips the extension. |
| `.github/workflows/release.yml` | Builds and uploads `WT Installer.exe` when a GitHub release is created. |

Package dependencies flow one way: `cmd` → `server` → `installer` → `gamedirs` / `logging` → `winutil` → `app`.

### HTTP API (internal/server)

- `GET /health` returns `OK`
- `GET /version` returns `Version: x.y.z` (the extension strips the prefix)
- `POST /download` with JSON `{type: "camo"|"sight", postId, fileUrl}`. Returns a plain-text summary (for example "Installed as tiger_2"), which the extension shows in its toast.
- `POST /openfolder?target=camo|sight|installer|logs` opens one Explorer window per matching folder.
- `GET /logs` (JSON array of `{time, message}`), `POST /logs/clear`

`corsMiddleware` only accepts requests from `https://live.warthunder.com`, `moz-extension://` / `chrome-extension://` origins, or requests with no Origin header. Any other website gets a 403.

## How to run

### Go app (Windows only: it uses `golang.org/x/sys/windows` and the registry)

```bash
go build -o wt-installer.exe ./cmd/wt-installer
```

- Dev run: `go run ./cmd/wt-installer` (log output goes to stdout as well as the log file).
- Release-style build: `build.bat`. It needs ImageMagick (`magick`), `rsrc` (`go install github.com/akavel/rsrc@latest`) and `upx` on PATH. It turns `media/icon.png` into `media/icon.ico`, creates `cmd/wt-installer/rsrc.syso`, builds with `-ldflags "-s -w -H=windowsgui"` to `WT Installer.exe` in the repo root, then compresses it with UPX. It stops on the first failed step.
- Only one instance can run (named mutex). Quit the running tray app before starting a new build.
- Check changes with `go vet ./...` and `go test ./...`. The tests use `httptest` and temp folders and never touch the real game folders.
- A local build whose `app.Version` is lower than the latest GitHub release updates itself as soon as it starts. Keep `Version` in sync with the latest release when running test builds.

### Extension

- Firefox: open `about:debugging` → This Firefox → Load Temporary Add-on → pick `extension/manifest.json`. It is published on AMO as "WT Local Downloader".
- `archive.bat` packs `extension/` into `extension.zip` for upload.
- The code uses the `browser.*` namespace (Firefox). Chrome is not supported.

### Release

1. Bump `Version` in `internal/app/app.go`.
2. Create a GitHub release whose tag is that version (with or without `v`; `compareVersions` strips it). The tag must match `Version`: a release whose exe reports a lower version makes clients update in a loop.
3. CI attaches `WT Installer.exe`, and running clients update themselves on their next start. The extension version is set separately in `extension/manifest.json`.

## Conventions

- **Go**
  - The only `package main` is `cmd/wt-installer`; everything else lives in `internal/` packages with one job each. Don't add import cycles: lower packages (`winutil`, `app`) must not import higher ones.
  - Package state stays private behind functions (`gamedirs.CamoDirs()`, `logging.Dir()`). Tests swap package-level function variables (for example `listCamoDirs` in `installer`) instead of touching another package's globals.
  - Don't do work in `init()`; `main` calls `logging.Init()` first, then `gamedirs.Refresh()`. The one exception is registering the BZip2 decompressor in `installer`.
  - On a failure the code does `log.Printf(...)` and then returns `fmt.Errorf("...: %w", err)`. User-facing failures in the download path also call `logging.OpenFolder()`.
  - Use the standard `log` package so messages reach the log file and the `/logs` buffer.
  - Missing game installs must not crash the app: `gamedirs.Refresh()` only warns, and downloads call it again lazily.
  - The camo dir list is guarded by a mutex in `gamedirs`. Installs are serialized by `installMu` in `installer`.
  - Every extraction path must keep the `sanitizeWindowsPath` and zip-slip prefix checks.
  - Installs never overwrite existing content:
    - **Skins** dedup at the folder level. If `UserSkins/<name>` already holds every incoming file unchanged, the install is skipped. Otherwise the skin goes into `<name>_2`, `<name>_3`, and so on. Archives without a single root folder go into `wtskin-<postId>`.
    - **Sights** dedup per file (`copyTree(..., true)`). Identical files are skipped, and different ones get a `_2` suffix.
  - Skins install into **every** detected game dir, sights into **every** numeric Saves folder.
  - Check the HTTP status of every download, and use an `http.Client` with a timeout (`downloadClient` in `installer`, `httpClient` in `update`) instead of `http.Get`.
  - Open folders with `winutil.OpenInExplorer` (ShellExecute), not `cmd /c start`. Hide windows of external processes with `SysProcAttr{HideWindow: true}`.
  - Doc comments are short and say why the code does something (for example, why there is a fallback extractor).
- **Extension**
  - Plain JS with no build step or dependencies. Two-space indentation, double quotes, `async/await`.
  - Message actions use a `wt_` prefix. Injected DOM ids and classes use `wt-local-` / `local-dl-btn`.
  - The content script never calls the server directly; requests go through `background.js`. The popup calls the server directly.
  - Log through `wtLog()` in the content script so entries show up in the popup's Extension tab.
- **The port `4316` is hard-coded** in `internal/app/app.go`, `extension/background.js`, `extension/popup/popup.js` and `manifest.json` `host_permissions`. Change all four together.
- Generated or build outputs (`*.exe`, `rsrc.syso`, `*.zip`) are gitignored. `media/icon.ico` is committed because `media.go` embeds it.

## Gotchas

- CI reads the Go version from `go.mod` (`go-version-file`), so bumping the `go` directive there is enough.
- Files are checked out with CRLF line endings on Windows, so `gofmt -l` lists them even when the code is formatted. `.gitattributes` normalizes them to LF on commit.
- `rsrc.syso` only gets linked from the main package's folder. A stray `rsrc.syso` in the repo root is ignored now that the root has no Go package.
