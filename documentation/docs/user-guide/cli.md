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

Write a default `config.json` to `~/.stratafs/` (or the path given by `--config-dir`). Idempotent — refuses to overwrite an existing config.

### `stratafs config show`

Print the resolved config (file values + environment overrides) as JSON.

## `stratafs fs`

Filesystem bridge subcommands.

### `stratafs fs export`

Materialize a semantic snapshot of a source as a directory tree of `metadata.json` plus chunk files. Useful for offline review, packaging, or seeding another StrataFS install.

```bash
stratafs fs export --source my-docs --output ./export
```

| Flag | Default | Description |
| --- | --- | --- |
| `--source` | _required_ | Source ID to export. |
| `--output` | `./stratafs-export` | Destination directory. |

### `stratafs fs mount`

Expose the semantic index as a read-only FUSE/WinFsp filesystem.

```bash
stratafs fs mount --source my-docs --mount-point /mnt/stratafs
```

Each file appears alongside a `metadata.json` and a `_chunks/` directory. macFUSE / FUSE / WinFsp must be installed on the host.

## Exit codes

| Code | Meaning |
| --- | --- |
| `0` | Success. |
| `1` | Generic error (config invalid, source unreachable, etc.). |
| `2` | Usage error (bad flags). |
| `130` | Interrupted (`Ctrl-C`). |
