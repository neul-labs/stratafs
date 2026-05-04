# AgentFS Roadmap

This document tracks near-term and longer-term goals for filesystem-level integration while keeping the REST API, MCP server, and other existing surfaces supported.

## Short-Term (current cycle)

- **Virtual Filesystem Export** ✅: mirrored tree with `metadata.json` and chunk files so traditional tools can inspect semantic data.
- **`agentfs fs export` CLI** ✅: new command to export a source’s metadata view into any directory.
- **FS Bridge Package** ✅: reusable code (`pkg/fsbridge`) for future FUSE/WinFsp work.
- **Wails Desktop UI (Linux first)** ✅: add a Wails-based control panel that talks to the REST API (manage sources, view queue, trigger exports). Package alongside the daemon as an AppImage/Deb with a systemd user service to keep AgentFS running in the background. See `desktop/agentfs-ui/`.
- **Release automation** ✅: scripts to bundle ONNX runtime + binaries + desktop UI. See `scripts/release-bundle.sh` and `.github/workflows/release.yml`.

## Long-Term

- **Native Mounts (FUSE/WinFsp)** ✅: `agentfs fs mount` command exposes a read-only filesystem view with original files plus `metadata.json` and `_chunks` directories. See `pkg/fsbridge/fuse.go` (Linux/macOS) and `pkg/fsbridge/winfsp.go` (Windows).
- **OS Search Integration** ✅:
  - GNOME Shell SearchProvider2 D-Bus integration. See `pkg/search/gnome_provider.go`
  - macOS Spotlight importer. See `installers/spotlight/`
  - Windows Search IFilter. See `installers/ifilter/`
- **Contextual Actions** ✅:
  - GNOME Nautilus extension. See `installers/nautilus/`
  - macOS Finder Sync extension. See `installers/finder/`
  - Windows Explorer shell extension. See `installers/explorer/`
  - CLI: `agentfs file info/chunks/reindex`
- **Enterprise Connectors** ✅:
  - SharePoint/OneDrive via Microsoft Graph API. See `pkg/filesystem/sharepoint.go`
  - Google Drive with Workspace file export. See `pkg/filesystem/googledrive.go`
  - Jira issues and attachments. See `pkg/filesystem/jira.go`
  - All connectors support incremental sync via delta/changes APIs
- **Enterprise Security**: RBAC layer for authentication and source-level permissions (planned).
- **Desktop Packaging** ✅:
  - *Linux*: AppImage/Deb + systemd user service. See `desktop/agentfs-ui/build/linux/`
  - *macOS*: Signed `.app` bundle + DMG + LaunchAgent. See `installers/macos/`
  - *Windows*: NSIS installer + tray app + Windows Service. See `installers/windows/`

APIs, the MCP server, and hybrid search remain first-class citizens. The filesystem bridge augments those interfaces so both traditional apps and agentic workflows can consume AgentFS data.
