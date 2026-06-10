# WarThunder Live Installer

Tool to download skins from WarThunder Live's official website into the game's skins folder directly from the browser.

## Usage

1. Download the latest release from the [releases page](https://github.com/AndreyKarm/wt-installer/releases).
2. Place the executable file in any place save (e.g., `C:\Users\%username%\Documents\WT Installer.exe`).
3. Run the executable file.
4. In the tray select "Auto-start" to start the server automatically when pc starts.
5. Install extension for [firefox](https://addons.mozilla.org/en-US/firefox/addon/wt-live-downloader/)(chrome requires fee to publish extension).
6. (Optional) Pin the extension icon to the taskbar.
7. Go to [WarThunder Live's official website](https://live.warthunder.com/feed/all/) and download skins and sights in a single click.

## Features

- Supports Gaijin Launcher and Steam versions of the game.
- Sights and camouflage skins are supported.

## Planned features

- [ ] Display if skin is already downloaded
- [ ] Selecting which skin to download if multiple are found
- [ ] Search bar
- [ ] Sounds support

## Building from source

1. Install [Go](https://go.dev/doc/install).
2. Clone the repository.
3. Run `go build -o wt-installer.exe .` or run build.bat.
4. Run the executable file.
