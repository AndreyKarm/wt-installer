# WarThunder Installer

Tool to download skins from WarThunder's official website into the game's skins folder directly from the browser.

## Features

- Support Gaijin Launcher and Steam versions of the game.

## Planned features

- [ ] Display if skin is already downloaded
- [ ] Selecting which skin to download if multiple are found
- [ ] Search bar
- [ ] Sights and sounds support

## Usage

1. Download the latest release from the [releases page](https://github.com/AndreyKarm/wt-installer/releases).
2. Place the executable file in any place save (e.g., `C:\Users\%username%\Documents\WT Installer.exe`).
3. Run the executable file.
4. In the tray select "Auto-start" to start the server automatically when pc starts.

## Building from source

1. Install [Go](https://go.dev/doc/install).
2. Clone the repository.
3. Run `go build -o wt-installer.exe .` or run build.bat.
4. Run the executable file.
