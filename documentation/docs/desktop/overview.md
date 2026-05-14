# Desktop App

StrataFS ships with a [Wails](https://wails.io/)-powered desktop UI for users who'd rather not live in a terminal. It talks to the same REST API the daemon exposes — there is no separate codepath.

## What you get

- **Dashboard** — daemon status, queue depth, error count, embedding model in use. Start / stop / restart from the menu bar.
- **Search** — hybrid / FTS / vector modes with a results pane that lets you preview chunks in-place.
- **Sources** — add, remove, enable, disable local and cloud sources without editing JSON. Credential entry stays in the OS keychain where supported.
- **Export** — kick off `stratafs fs export` from the UI; results land in a directory of your choice.
- **Settings** — embedding model, worker count, ports, log level. Changes apply on next daemon restart.

## Architecture

The UI is a Vue 3 single-page app embedded in a Wails wrapper. It boots a local `stratafs serve` process if one isn't already running and talks to it via `http://localhost:8080`. The Wails Go bridge is intentionally thin — it only handles process supervision, OS-level keychain reads, and "show config in Finder/Explorer"-style affordances.

```
┌────────────────────────────┐    ┌────────────────────────────┐
│         Wails UI           │    │      stratafs daemon       │
│  ┌──────────────────────┐  │    │  ┌────────────────────┐   │
│  │  Vue 3 frontend      │──┼────┼──│  REST API :8080    │   │
│  └──────────────────────┘  │    │  └────────────────────┘   │
│  ┌──────────────────────┐  │    │  ┌────────────────────┐   │
│  │  Go bridge (app.go)  │──┼────┼──│  Process supervisor│   │
│  └──────────────────────┘  │    │  └────────────────────┘   │
└────────────────────────────┘    └────────────────────────────┘
```

If the daemon is already running (e.g., as a system service), the UI attaches to it rather than spawning a new one.

## Building from source

```bash
cd desktop/stratafs-ui
wails build
```

Output binaries land in `desktop/stratafs-ui/build/bin/`. Wails handles platform-specific packaging — see [Platform Notes](platform-notes.md) for OS-specific install paths and tray integration.

## Customizing

The UI reads the same `~/.stratafs/config.json` as the CLI. Changes made through the UI write to that file using atomic rewrites — no surprise reformatting, no in-app config silo.
