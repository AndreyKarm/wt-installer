# CLAUDE.md

WT Installer installs War Thunder Live (live.warthunder.com) camouflages and sights into the local game folders with one click from the browser. It has two parts that talk over HTTP on `127.0.0.1:4316`:

1. **Go tray app** (`cmd/` + `internal/`): a Windows-only system-tray program that runs a local HTTP server, downloads zips and extracts them into the game folders.
2. **Firefox extension** (`extension/`, Manifest V3): injects "Install Camo" / "Install Sight" buttons into War Thunder Live posts and sends requests to the tray app.

## Structure

Module `liotom/installer` (the name differs from the repo name).

| Path | Purpose |
|---|---|
| `cmd/wt-installer/main.go` | Entry point. `logging.Init()`, single-instance mutex (`WTInstallerServer`), `gamedirs.Refresh()`, tray menu (Auto-start toggle, update status, open skins/sights/logs folder, Quit), starts `server.Start()` and the update check. The generated `rsrc.syso` (exe icon) must be in this folder. |
| `internal/app` | Shared constants: `Name` (also the auto-start registry value name), `Version`, `Port`. |
| `internal/server` | HTTP server and handlers, CORS/origin allow-list (`isAllowedOrigin`), `OpenFolder(target)` shared by `/openfolder` and the tray menu. `server_test.go` tests the origin check and bad-request handling. |
| `internal/installer` | `DownloadSkin` / `DownloadSight`. Archives are downloaded and extracted to a temp folder first (`downloadAndExtract` → `extractArchive`), then copied into place (`copyTree`). Includes Windows path sanitizing, zip-slip check, folder dedup for skins, per-file dedup for sights, and fallback to external `7z`/`tar` for compression methods Go can't read (LZMA, PPMd, Deflate64). BZip2 is registered natively. Also `Uninstall(postId)`. `installer_test.go` covers install/dedup, recording, uninstall, host allow-list, size caps, bad downloads and path traversal. |
| `internal/installed` | Remembers installed posts in `%APPDATA%\WTInstaller\installed.json` (post id → type, time, and the skin folders or sight files it went to). `Status()` reports only posts with at least one path still on disk. `Lookup()` also returns which paths other posts share; `Remove()` forgets a post. |
| `internal/gamedirs` | (`gamedirs_test.go` covers the VDF parser.) Finds game installs: Gaijin Launcher (`HKCU\SOFTWARE\Gaijin\WarThunder\InstallDir`) and Steam (parses `libraryfolders.vdf` for app ID `236390`). `CamoDirs()` returns `<game>\UserSkins`. `SightDirs()` returns `Documents\My Games\WarThunder\Saves\<numeric id>\production\UserSights\all_tanks`. |
| `internal/update` | Self-update from GitHub Releases (`AndreyKarm/wt-installer`). `Check()` compares the latest tag with `app.Version`. `Apply()` downloads the `.exe` asset and starts a hidden `_update.bat` (`swapScript`) that swaps it in; the caller then quits the tray. The script retries the swap for about 30 s while the old exe is still locked, and if it never succeeds, deletes the download and restarts the old exe. Each attempt is recorded in `%APPDATA%\WTInstaller\update-attempt.json`, and `RecentlyAttempted()` blocks retrying the same tag for 6 hours, so a failed swap can't turn into a restart loop. `update_test.go` runs the real script against a locked file. |
| `internal/logging` | Logs to `%TEMP%\WTInstallerLogs\wtinstaller-<timestamp>.log` and to an in-memory buffer (200 entries) served at `/logs`. Prunes logs older than 7 days. `OpenFolder()` opens the log folder at most once per run. |
| `internal/winutil` | `OpenInExplorer` (ShellExecute), `ExecutablePath`, auto-start via `HKCU\...\CurrentVersion\Run`. |
| `media/` | `icon.png` (source), `icon.ico` (generated, committed) and `media.go`, which embeds the icon as `media.Icon`. |
| `extension/content.js` | Content script on live.warthunder.com: adds buttons to feed items (`.feed_item[post_id]`) and the lightbox (`#clb .wrapper[post_id]`) through `MutationObserver`s. Shows toasts. |
| `extension/background.js` | Relays content-script messages (`wt_install`, `wt_uninstall`, `wt_installed`, `wt_server_version`, `wt_open_folder`, `wt_*_log*`) to the server and keeps extension logs in memory. |
| `extension/popup/` | Popup: extension and server versions, buttons to open folders, server and extension log viewer. Warns when the tray app is older than `MIN_SERVER_VERSION` in `popup.js`; raise it when the extension starts depending on a new endpoint. |
| `build.bat`, `archive.bat` | Local build script and the script that zips the extension. |
| `.github/workflows/release.yml` | Builds and uploads `WT Installer.exe` when a GitHub release is created. |

Package dependencies flow one way: `cmd` → `server` → `installer` → `gamedirs` / `installed` / `logging` → `winutil` → `app`.

### HTTP API (internal/server)

- `GET /health` returns `OK`
- `GET /version` returns `Version: x.y.z` (the extension strips the prefix)
- `POST /download` with JSON `{type: "camo"|"sight", postId, fileUrl}`. Returns a plain-text summary (for example "Installed as tiger_2"), which the extension shows in its toast.
- `GET /installed` returns `{"<postId>": "camo"|"sight"}` for installed posts whose files still exist. The content script uses it to mark buttons as "✓ Installed".
- `POST /uninstall` with JSON `{postId}` deletes what that post installed and forgets it. Returns a plain-text summary.
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
- The CI runner has no War Thunder install, so tests must not depend on this machine's game or Saves folders. To reproduce CI locally, run the tests with an empty profile: set `$env:USERPROFILE` to an empty temp folder, then run `go test ./...`.
- If the installed WT Installer is running, a new build exits at once (single-instance mutex), and anything on port 4316 is answered by the running copy. Quit it from the tray before smoke-testing a build.
- A local build whose `app.Version` is lower than the latest GitHub release updates itself as soon as it starts. Keep `Version` in sync with the latest release when running test builds.

### Extension

- Firefox: open `about:debugging` → This Firefox → Load Temporary Add-on → pick `extension/manifest.json`. It is published on AMO as "WT Local Downloader".
- `archive.bat` packs `extension/` into `extension.zip` for upload.
- The code uses the `browser.*` namespace (Firefox). Chrome is not supported.

### Release

1. Bump `Version` in `internal/app/app.go`.
2. Create a GitHub release whose tag is that version (with or without `v`; `compareVersions` strips it). The tag must match `Version`, because a release whose exe reports a lower version makes clients update in a loop. CI enforces this and fails the build otherwise.
3. CI runs `go vet` and `go test`, builds with `CGO_ENABLED=0` (no C toolchain needed), compresses with UPX and attaches `WT Installer.exe`, and running clients update themselves on their next start. The extension version is set separately in `extension/manifest.json`.

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
  - Every successful install, including "already installed" outcomes, is recorded through `recordInstall` (`installed.Add`). A failure to record is only logged. Installer tests stub `recordInstall` in `TestMain` so they never write to the real store.
  - Check the HTTP status of every download, and use an `http.Client` with a timeout (`downloadClient` in `installer`, `httpClient` in `update`) instead of `http.Get`.
  - Downloads must be https on `warthunder.com` or a subdomain (`allowDownloadURL`), checked on every redirect hop. Site links are `https://live.warthunder.com/dl/<hash>/` and redirect to `cdn-live.warthunder.com`.
  - Size caps: 512 MiB download, 4 GiB extracted, 20,000 entries (`maxArchiveBytes`, `maxExtractedBytes`, `maxArchiveFiles`). The site caps skin/sight uploads at 300 MB. Native extraction enforces the byte cap on the stream; the external-extractor fallback checks the total afterwards.
  - The fallback extractor finds 7-Zip on PATH, through its registry key, or in Program Files (`find7z`), then falls back to `tar`.
  - `Uninstall` only deletes recorded paths that sit in a detected game folder: a skin folder directly inside `UserSkins`, or a sight file inside an `all_tanks` folder. It keeps paths another recorded post shares. It then removes sight subfolders left empty (`removeEmptyParents`), never the `all_tanks` folder itself.
  - Tests swap the hooks (`listCamoDirs`, `listSightDirs`, `recordInstall`, `lookupInstall`, `forgetInstall`, `allowDownloadURL`, the size caps). `TestMain` in `installer` stubs recording and allows the local `httptest` hosts.
  - Open folders with `winutil.OpenInExplorer` (ShellExecute), not `cmd /c start`. Hide windows of external processes with `SysProcAttr{HideWindow: true}`.
  - Doc comments are short and say why the code does something (for example, why there is a fallback extractor).
- **Extension**
  - Plain JS with no build step or dependencies. Two-space indentation, double quotes, `async/await`.
  - Message actions use a `wt_` prefix. Injected DOM ids and classes use `wt-local-` / `local-dl-btn`.
  - The content script never calls the server directly; requests go through `background.js`. The popup calls the server directly.
  - Log through `wtLog()` in the content script so entries show up in the popup's Extension tab.
  - Build DOM with `createElement` / `createElementNS` and `textContent`, never `innerHTML`. AMO review flags `innerHTML`, and log text comes from the server and the page.
  - Hide injected buttons with the `hidden` attribute, not `style.display`. Their CSS uses `display: ... !important` to beat the site's id-based rules, which also beats inline styles.
  - `createDownloadButtons()` returns the install button and its trash-can uninstall button. Callers insert both. The trash can shows only for installed posts and asks for confirmation before uninstalling.
  - Button idle state is drawn only by `renderIdle()` from `installedPosts`. The list is refreshed on load and whenever the tab becomes visible, and kept as-is when the server is offline.
- **The port `4316` is hard-coded** in `internal/app/app.go`, `extension/background.js`, `extension/popup/popup.js` and `manifest.json` `host_permissions`. Change all four together.
- Generated or build outputs (`*.exe`, `rsrc.syso`, `*.zip`) are gitignored. `media/icon.ico` is committed because `media.go` embeds it.

## Gotchas

- CI reads the Go version from `go.mod` (`go-version-file`), so bumping the `go` directive there is enough. Actions are on Node 24 majors (`checkout@v7`, `setup-go@v7`, `action-gh-release@v3`), and `rsrc` is pinned to `v0.10.2`.
- A release run builds the commit its tag pointed to when the release was created. Re-running the job, or recreating the release on an existing tag, rebuilds the same commit. To ship a fix under the same version, delete the release *and* its tag, then create both again.
- The extension talks to `http://localhost:4316`, even though the server binds `127.0.0.1`. Changing `host_permissions` to `127.0.0.1` would be a new permission, which Firefox may ask every user to approve on update.
- Files are checked out with CRLF line endings on Windows, so `gofmt -l` lists them even when the code is formatted. `.gitattributes` normalizes them to LF on commit.
- `rsrc.syso` only gets linked from the main package's folder. A stray `rsrc.syso` in the repo root is ignored now that the root has no Go package.
