# Configuration

StrataFS reads its configuration from `~/.stratafs/config.json` (or `%USERPROFILE%\.stratafs\config.json` on Windows). Initialize it with sensible defaults:

```bash
stratafs config init
```

`config init` writes the default `config.json` and creates the supporting directories (FastEmbed model cache, per-source cache directories, queue database).

## Full config example

```json
{
  "version": "0.2.0",
  "agent_dir": ".stratafs",
  "global_dir": "/home/you/.stratafs",
  "sources": [
    {
      "id": "default-local",
      "name": "Current Directory",
      "type": "local",
      "enabled": true,
      "path": "/home/you/Documents",
      "local_cache_dir": "",
      "credentials": null,
      "filters": {
        "include_patterns": ["*"],
        "exclude_patterns": [".git/**", "node_modules/**", "*.tmp", "*.log"],
        "max_file_size": 104857600,
        "ignore_hidden": true
      }
    }
  ],
  "server": {
    "api_port": 8080,
    "mcp_port": 8081
  },
  "worker": {
    "count": 4,
    "scan_interval": "10s",
    "batch_size": 10
  },
  "embedding": {
    "model": "bge-base-en-v1.5",
    "cache_dir": "/home/you/.stratafs/fastembed_cache",
    "dimension": 768,
    "performance": {
      "batch_size": 32,
      "max_concurrency": 4,
      "cache_results": true,
      "enable_gpu": false
    }
  },
  "database": {
    "compression_enabled": true,
    "compression_threshold": 512,
    "maintenance_interval": "24h",
    "deleted_threshold": "168h"
  }
}
```

Chunking is wired into the queue processor with built-in defaults; it is not currently exposed as a top-level `chunking` config block. See [File Types](file-types.md) for which strategy each parser applies.

## Sections

### `sources`

An array of source definitions. See [Storage Backends](storage-backends.md) for every supported `type` and its credential shape.

### `server`

| Key | Default | Description |
| --- | --- | --- |
| `api_port` | `8080` | REST API port. |
| `mcp_port` | `8081` | MCP server port. |

### `worker`

| Key | Default | Description |
| --- | --- | --- |
| `count` | `4` | Concurrent worker goroutines. |
| `scan_interval` | `"10s"` | Polling interval for remote sources (Go duration). |
| `batch_size` | `10` | Files processed per worker tick. |

### `embedding`

| Key | Default | Description |
| --- | --- | --- |
| `model` | `"bge-base-en-v1.5"` | FastEmbed model identifier. |
| `dimension` | `768` | Vector dimension. Must match the model. |
| `cache_dir` | `~/.stratafs/fastembed_cache` | Where ONNX model files are cached on disk. |
| `performance.batch_size` | `32` | Embeddings generated per call to the model. |
| `performance.max_concurrency` | `4` | Concurrent embedding calls. |
| `performance.enable_gpu` | `false` | Use a GPU runtime if available. |

Available models:

| Model | Dim | Speed | Notes |
| --- | --- | --- | --- |
| `bge-base-en-v1.5` | 768 | medium | Recommended default. |
| `bge-small-en-v1.5` | 384 | fast | Smaller index, lower quality. |
| `bge-base-en` | 768 | medium | Original BGE base. |
| `bge-small-en` | 384 | fast | Original BGE small. |
| `all-minilm-l6-v2` | 384 | fast | Multilingual Sentence-BERT, very fast. |

### `database`

| Key | Default | Description |
| --- | --- | --- |
| `compression_enabled` | `true` | Gzip chunks above the threshold. |
| `compression_threshold` | `512` | Bytes. Chunks smaller than this are stored raw. |
| `maintenance_interval` | `"24h"` | How often to run vacuum / FTS optimize. |
| `deleted_threshold` | `"168h"` | Soft-deleted rows are hard-deleted after this. |

### `filters`

Each source has its own `filters` block.

| Key | Default | Description |
| --- | --- | --- |
| `include_patterns` | `["*"]` | Glob whitelist applied first. |
| `exclude_patterns` | `[".git/**", "node_modules/**"]` | Glob blacklist applied after include. |
| `max_file_size` | `104857600` (100 MiB) | Skip files larger than this. |
| `ignore_hidden` | `true` | Skip dotfiles and dot-directories. |

## Environment variables

Environment variables override config at runtime — useful for Docker and CI. The set below matches the variables `pkg/config/config.go` reads on boot.

| Variable | Effect |
| --- | --- |
| `STRATAFS_GLOBAL_DIR` | Override `global_dir` (where `config.json`, the queue DB, and caches live). |
| `STRATAFS_API_PORT` | Override `server.api_port`. |
| `STRATAFS_MCP_PORT` | Override `server.mcp_port`. |
| `STRATAFS_WORKERS` | Override `worker.count`. |
| `STRATAFS_SCAN_INTERVAL` | Override `worker.scan_interval` (Go duration string). |
| `STRATAFS_MODEL` | Override `embedding.model` (one of the IDs in the table above). |
| `STRATAFS_FASTEMBED_CACHE` | Override `embedding.cache_dir`. |
| `STRATAFS_DIRS` | Comma-separated list of extra local paths to attach as additional sources on first init. |

Example:

```bash
STRATAFS_API_PORT=9000 STRATAFS_WORKERS=8 stratafs serve
```

## First-run behaviour

On first start, `LoadConfig` creates `~/.stratafs/` if it doesn't exist, writes a default `config.json`, and seeds the FastEmbed cache directory. Subsequent starts read the file and re-apply the environment overrides above. The queue database (`queue.db`) is created next to `config.json`.

For local sources, `ValidateSource` checks that `path` exists on disk; for S3 / GCS / Azure sources it checks that `bucket` (or `container`) is set in `credentials`. Anything else — port conflicts, broken credentials, missing model weights — surfaces the first time the affected subsystem starts, not at config load.
