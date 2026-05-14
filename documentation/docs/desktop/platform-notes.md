# Platform Notes

Each desktop platform has its own packaging story, tray integration, and recommended install path.

## macOS

- Distribution: signed `.pkg` installer (`StrataFS-{version}.pkg`).
- App bundle: `/Applications/StrataFS.app`.
- Daemon: started by `~/Library/LaunchAgents/org.stratafs.daemon.plist`.
- Menu bar: native status icon with start/stop/quit and a search shortcut.
- Spotlight integration: optional `mdimporter` plugin ships under `installers/spotlight/` and indexes StrataFS-managed metadata into Spotlight.
- Quick Look: planned.

### Uninstall

```bash
sudo /Applications/StrataFS.app/Contents/Resources/uninstall.sh
```

The uninstaller stops the LaunchAgent, removes the app bundle, and (optionally) removes `~/.stratafs/`. It will prompt before deleting user data.

## Windows

- Distribution: NSIS installer (`StrataFS-Setup.exe`).
- Install path: `%PROGRAMFILES%\StrataFS\`.
- Data: `%APPDATA%\StrataFS\` (mirrors `~/.stratafs/` on POSIX).
- Daemon: optional Windows Service named `StrataFS`. Toggle it during install or via `sc start StrataFS` / `sc stop StrataFS`.
- Tray: dedicated `stratafs-tray.exe` with start/stop/search shortcuts.
- Explorer integration: right-click "Index with StrataFS" shell extension ships under `installers/explorer/`.
- Search integration: IFilter under `installers/ifilter/` lets Windows Search index StrataFS chunk metadata.

### Uninstall

Use **Settings → Apps → StrataFS → Uninstall**. The uninstaller will offer to remove `%APPDATA%\StrataFS\`.

## Linux

Two packaging options, depending on your distribution:

### AppImage (universal)

```bash
chmod +x StrataFS-{version}-x86_64.AppImage
./StrataFS-{version}-x86_64.AppImage
```

Self-contained, no install needed. The bundle includes the ONNX Runtime so there are no system dependencies beyond glibc.

### Debian / Ubuntu (`.deb`)

```bash
sudo apt install ./stratafs_{version}_amd64.deb
```

Installs:

- `/usr/bin/stratafs`
- `~/.config/systemd/user/stratafs.service` (enable with `systemctl --user enable --now stratafs`)
- `/usr/share/applications/stratafs.desktop` (launcher for the Wails UI)

### GNOME integration

- Nautilus extension under `installers/nautilus/` adds a context-menu "Open in StrataFS" action.
- A GNOME Shell SearchProvider at `org.stratafs.SearchProvider` surfaces results in the Activities overview. Enabled by the `.service` file in `installers/desktop/`.

### Uninstall

```bash
sudo apt remove stratafs
# Or, for the AppImage, just delete the file.
```

User data under `~/.stratafs/` is left in place unless you remove it manually.
