# Roadmap

This page tracks the near- and longer-term direction. The REST API, MCP server, and hybrid search remain first-class — everything below augments those surfaces.

## Done

These have shipped and are in `main`.

- **Virtual filesystem export** — mirrored tree with `metadata.json` and chunk files so traditional tools can inspect semantic data.
- **`stratafs fs export` CLI** — exports a source's metadata view into any directory.
- **FS bridge package** (`pkg/fsbridge`) — reusable code shared between `fs export` and the FUSE/WinFsp mount.
- **Wails desktop UI** — control panel for managing sources, watching the queue, triggering exports. Packaged as AppImage / `.deb` (Linux), `.pkg` (macOS), NSIS installer (Windows). See [Desktop App](../desktop/overview.md).
- **Native mounts** — `stratafs fs mount` exposes a read-only filesystem view via FUSE on Linux/macOS and WinFsp on Windows.
- **OS search integration** — GNOME Shell SearchProvider2, macOS Spotlight importer, Windows Search IFilter.
- **Contextual actions** — Nautilus extension, macOS Finder Sync, Windows Explorer shell extension.
- **Enterprise connectors** — SharePoint/OneDrive via Microsoft Graph, Google Drive with Workspace export, Jira issues and attachments. Incremental sync via delta APIs.
- **Desktop packaging** — Linux (AppImage + `.deb` + systemd user service), macOS (signed `.app` + DMG + LaunchAgent), Windows (NSIS installer + tray + Windows Service).
- **Release automation** — scripts that bundle the ONNX Runtime, binaries, and desktop UI into platform archives. See `scripts/release-bundle.sh` and `.github/workflows/release.yml`.

## In flight

- **Enterprise security** — RBAC for authentication and source-level permissions. Tracking issue: [#TBD](https://github.com/neul-labs/stratafs/issues).

## Planned

- **Streaming search results** — `Transfer-Encoding: chunked` on `/search` for very large result sets.
- **Custom embedding models** — first-class support for loading any ONNX-compatible model from disk, not just the BGE family.
- **Cross-source ranking signals** — per-source weight, recency boost, "trusted source" pinning.
- **Quality benchmarks in CI** — every PR runs `research/benchmarks/` and posts a delta against `main`.
- **Encrypted source databases** — SQLCipher-backed source DBs for at-rest encryption.

## Stretch / under discussion

- **Distributed read replicas** — read-only mirrors of a source's DB for higher search QPS. SQLite WAL streaming is the candidate mechanism.
- **Live retrievers** — keep an open connection to long-running agents and stream new chunks as they're indexed.
- **OCR pipeline** — image and scanned-PDF support, behind an opt-in flag for the additional model weight.
- **Multimodal embeddings** — image + text in one vector space for searches like "find me the architecture diagram in the docs".

## How we prioritize

Roughly: agent-facing > local-first > server / cloud. Anything that makes StrataFS a better backbone for a long-running coding or research assistant tends to jump the queue.

To propose something: open a GitHub issue with a one-paragraph problem statement and a concrete-enough scope that we can argue about it. Strong opinions weakly held.
