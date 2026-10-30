# WarThunder Live Installer

Install camouflages and sights from [War Thunder Live](https://live.warthunder.com/feed/all/) straight into your game folders with one click in the browser.

It has two parts: a small tray app that does the installing, and a Firefox extension that adds the buttons to the site. The extension needs the tray app to be running.

## Usage

1. Download `WT Installer.exe` from the [releases page](https://github.com/AndreyKarm/wt-installer/releases).
2. Put it in a folder you can write to, for example `C:\Users\%username%\Documents\WT Installer.exe`. Avoid `Program Files`: the app updates itself by replacing the exe.
3. Run it. It appears in the system tray.
4. In the tray menu, tick "Auto-start" so it starts with Windows and is always ready.
5. Install the extension for [Firefox](https://addons.mozilla.org/en-US/firefox/addon/wt-live-downloader/). (Chrome isn't supported; publishing there requires a fee.)
6. (Optional) Pin the extension icon to the toolbar.
7. On [War Thunder Live](https://live.warthunder.com/feed/all/), click "Install Camo" or "Install Sight" on any post.

## Features

- **One-click install** of camouflages and sights, in the feed and in the full post view.
- **Gaijin Launcher and Steam** installs are both supported. If you have both, skins go into each one.
- **Never overwrites:** a different skin or sight that happens to have the same name is installed alongside as `name_2`, `name_3`, and so on. Installing the same one again is skipped.
- **"Installed" marks:** posts you've installed show a green check on the site. Delete the files by hand and the mark disappears.
- **Uninstall from the site:** the trash-can button next to an installed post deletes exactly what that post installed.
- **Tray menu:** open the skins, sights or logs folder, and toggle auto-start.
- **Automatic updates:** the tray app checks GitHub on start and updates itself.

Installs are remembered in `%APPDATA%\WTInstaller\installed.json`. Logs are in `%TEMP%\WTInstallerLogs` (the "Open logs folder" menu item opens them).

## Planned features

- [ ] Selecting which skin to download if multiple are found
- [ ] Search bar
- [ ] Sounds support

## Building from source

1. Install [Go](https://go.dev/doc/install).
2. Clone the repository.
3. Run `go build -o wt-installer.exe ./cmd/wt-installer`, or run `build.bat` for a release build with the icon (needs ImageMagick, `rsrc` and UPX).
4. Run `go test ./...` to run the tests.
