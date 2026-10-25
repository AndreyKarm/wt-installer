# CLAUDE.md

WT Installer installs War Thunder Live (live.warthunder.com) camouflages and sights into the local game folders with one click from the browser. It has two parts that talk over HTTP on `127.0.0.1:4316`:

1. **Go tray app** (repo root, `package main`): a Windows-only system-tray program that runs a local HTTP server, downloads zips and extracts them into the game folders.
2. **Firefox extension** (`extension/`, Manifest V3): injects "Install Camo" / "Install Sight" buttons into War Thunder Live posts and sends requests to the tray app.

## Structure

| Path | Purpose |
|---|---|
| `main.go` | Entry point. Single-instance mutex (`WTInstallerServer`), tray menu (Auto-start toggle, update status, Quit), starts the server and the update check. |
| `server.go` | HTTP server and handlers, CORS middleware, in-memory log ring buffer (200 entries). Holds `Port` and `Version`. |
| `download.go` | `DownloadSkin` / `DownloadSight`. Archives are downloaded and extracted to a temp folder first (`downloadAndExtract` → `extractArchive`), then copied into place (`copyTree`). Includes Windows path sanitizing, zip-slip check, folder dedup for skins, per-file dedup for sights, and fallback to external `7z`/`tar` for compression methods Go can't read (LZMA, PPMd, Deflate64). BZip2 is registered natively. |
| `download_test.go` | Tests for install/dedup behavior, bad downloads, path traversal and the origin allow-list. |
| `registry.go` | Finds game installs: Gaijin Launcher (`HKCU\SOFTWARE\Gaijin\WarThunder\InstallDir`) and Steam (parses `libraryfolders.vdf` for app ID `236390`). Fills `GameDirs`, `CamoDirs` (`<game>\UserSkins`), `SightDirs` (`Documents\My Games\WarThunder\Saves\<numeric id>\production\UserSights\all_tanks`). |
| `autostart.go` | `appName`, executable path helpers, auto-start via `HKCU\...\CurrentVersion\Run`. |
| `update.go` | Self-update from GitHub Releases (`AndreyKarm/wt-installer`): downloads the `.exe` asset, swaps it in through a hidden `_update.bat`, restarts. |
| `logger.go` | Logs to `%TEMP%\WTInstallerLogs\wtinstaller-<timestamp>.log`, prunes logs older than 7 days, `openLogsFolder()` (opens at most once per run). |
| `assets.go` | Embeds `media/icon.ico` for the tray icon. |
| `extension/content.js` | Content script on live.warthunder.com: adds buttons to feed items (`.feed_item[post_id]`) and the lightbox (`#clb .wrapper[post_id]`) through `MutationObserver`s. Shows toasts. |
| `extension/background.js` | Relays content-script messages (`wt_install`, `wt_server_version`, `wt_open_folder`, `wt_*_log*`) to the server and keeps extension logs in memory. |
| `extension/popup/` | Popup: extension and server versions, buttons to open folders, server and extension log viewer. |
| `build.bat`, `archive.bat` | Local build script and the script that zips the extension. |
| `.github/workflows/release.yml` | Builds and uploads `WT Installer.exe` when a GitHub release is created. |

### HTTP API (server.go)

- `GET /health` returns `OK`
- `GET /version` returns `Version: x.y.z` (the extension strips the prefix)
- `POST /download` with JSON `{type: "camo"|"sight", postId, fileUrl}`. Returns a plain-text summary (for example "Installed as tiger_2"), which the extension shows in its toast.
- `POST /openfolder?target=camo|sight|installer|logs` opens one Explorer window per matching folder.
- `GET /logs` (JSON array of `{time, message}`), `POST /logs/clear`

`corsMiddleware` only accepts requests from `https://live.warthunder.com`, `moz-extension://` / `chrome-extension://` origins, or requests with no Origin header. Any other website gets a 403.

## How to run

### Go app (Windows only: it uses `golang.org/x/sys/windows` and the registry)

```bash
go build -o wt-installer.exe .
```

- Dev build with a console: `go run .` (log output goes to stdout as well as the log file).
- Release-style build: `build.bat`. It needs ImageMagick (`magick`), `rsrc` (`go install github.com/akavel/rsrc@latest`) and `upx` on PATH. It turns `media/icon.png` into `media/icon.ico`, creates `rsrc.syso`, builds with `-ldflags "-s -w -H=windowsgui"` to `WT Installer.exe`, then compresses it with UPX.
- Only one instance can run (named mutex). Quit the running tray app before starting a new build.
- Check changes with `go vet ./...` and `go test ./...`. The tests use `httptest` and temp folders and never touch the real game folders.

### Extension

- Firefox: open `about:debugging` → This Firefox → Load Temporary Add-on → pick `extension/manifest.json`. It is published on AMO as "WT Local Downloader".
- `archive.bat` packs `extension/` into `extension.zip` for upload.
- The code uses the `browser.*` namespace (Firefox). Chrome is not supported.

### Release

1. Bump `Version` in `server.go`.
2. Create a GitHub release whose tag is that version (with or without `v`; `compareVersions` strips it).
3. CI attaches `WT Installer.exe`, and running clients update themselves on their next start. The extension version is set separately in `extension/manifest.json`.

## Conventions

- **Go**
  - Everything lives in one flat `package main`; there are no subpackages.
  - Exported names (`DownloadSkin`, `RefreshGameDirs`) are for the main features. Lowercase names are internal helpers.
  - On a failure the code does `log.Printf(...)` and then returns `fmt.Errorf("...: %w", err)`. User-facing failures in the download path also call `openLogsFolder()`.
  - Use the `log` package so messages reach the log file and the `/logs` buffer. `ensureFileLogging()` runs from several `init()` functions and is guarded by `sync.Once`.
  - Missing game installs must not crash the app: `RefreshGameDirs()` only warns, and downloads call it again lazily.
  - `CamoDirs` and the other dir lists are guarded by `dirsMu`. Read them with `getCamoDirs()`. Installs are serialized by `installMu`.
  - Every extraction path must keep the `sanitizeWindowsPath` and zip-slip prefix checks.
  - Installs never overwrite existing content:
    - **Skins** dedup at the folder level. If `UserSkins/<name>` already holds every incoming file unchanged, the install is skipped. Otherwise the skin goes into `<name>_2`, `<name>_3`, and so on. Archives without a single root folder go into `wtskin-<postId>`.
    - **Sights** dedup per file (`copyTree(..., true)`). Identical files are skipped, and different ones get a `_2` suffix.
  - Skins install into **every** detected game dir, sights into **every** numeric Saves folder.
  - Check the HTTP status of every download (`fetchArchive`, `ApplyUpdate`). Use `downloadClient`, which has a timeout, instead of `http.Get`.
  - Open folders with `openInExplorer` (ShellExecute), not `cmd /c start`. Hide windows of external processes with `SysProcAttr{HideWindow: true}`.
  - Doc comments are short and say why the code does something (for example, why there is a fallback extractor).
- **Extension**
  - Plain JS with no build step or dependencies. Two-space indentation, double quotes, `async/await`.
  - Message actions use a `wt_` prefix. Injected DOM ids and classes use `wt-local-` / `local-dl-btn`.
  - The content script never calls the server directly; requests go through `background.js`. The popup calls the server directly.
  - Log through `wtLog()` in the content script so entries show up in the popup's Extension tab.
- **The port `4316` is hard-coded** in `server.go`, `extension/background.js`, `extension/popup/popup.js` and `manifest.json` `host_permissions`. Change all four together.
- Generated or build outputs (`*.exe`, `rsrc.syso`, `*.zip`) are gitignored. `media/icon.ico` is committed because `assets.go` embeds it.

## Gotchas

- CI reads the Go version from `go.mod` (`go-version-file`), so bumping the `go` directive there is enough.
- Files are checked out with CRLF line endings on Windows, so `gofmt -l` lists them even when the code is formatted. `.gitattributes` normalizes them to LF on commit.
- `go.mod` names the module `liotom/installer`, which differs from the repo name. That is harmless because everything is in `main`.
