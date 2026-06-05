# CLI Reference

The `stratafs` binary is the main entry point. Subcommands and flags are listed below — for the live, version-pinned reference run `stratafs help <command>`.

## `stratafs`

Global flags accepted by every subcommand:

| Flag | Description |
| --- | --- |
| `--config-dir <path>` | Override the StrataFS data directory (defaults to `~/.stratafs`). |
| `-h`, `--help` | Show help for the command. |
| `--version` | Print version and exit. |

## `stratafs serve`

Start the daemon. Runs the file monitor, the queue workers, the REST API, and the MCP server.

```bash
stratafs serve
```

There are no required flags. The daemon listens on `server.api_port` (default `8080`) and `server.mcp_port` (default `8081`).

## `stratafs search`

Query the index from the command line.

```bash
stratafs search "kubernetes deployment" --mode hybrid --limit 5
```

| Flag | Default | Description |
| --- | --- | --- |
| `--mode` | `hybrid` | `hybrid` \| `fulltext` \| `vector`. |
| `--limit` | `10` | Maximum results to return. |
| `--content` | `false` | Include the chunk content in the output. |
| `--json` | `false` | Emit JSON instead of formatted text. |

See [Search](search.md) for ranking details and tuning.

## `stratafs config`

Configuration helpers.

### `stratafs config init`

Create the default configuration file and supporting directories under `~/.stratafs/` (or the path given by `--config-dir`).

## `stratafs fs`

Filesystem bridge subcommands.

### `stratafs fs export`

Export the indexed virtual filesystem into a local directory with metadata and chunk files. Useful for offline review or packaging.

```bash
stratafs fs export --output ./export
```

| Flag | Default | Description |
| --- | --- | --- |
| `-o`, `--output` | `""` | Output directory for export. |

## `stratafs mount`

Mount the StrataFS virtual filesystem at a FUSE/WinFsp mount point.

```bash
stratafs mount --mount-point /mnt/stratafs
```

| Flag | Default | Description |
| --- | --- | --- |
| `-m`, `--mount-point` | _required_ | Mount point directory. |
| `--read-only` | `true` | Mount as read-only. |
| `--show-chunks` | `false` | Expose `_chunks/` directories alongside each file. |
| `--show-metadata` | `false` | Expose `metadata.json` files alongside each file. |

macFUSE / FUSE / WinFsp must be installed on the host.

## `stratafs version`

Print the current StrataFS version and build time, then exit.

## Exit codes

| Code | Meaning |
| --- | --- |
| `0` | Success. |
| `1` | Generic error (config invalid, source unreachable, etc.). |
| `2` | Usage error (bad flags). |
| `130` | Interrupted (`Ctrl-C`). |
