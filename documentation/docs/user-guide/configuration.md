# Configuration

StrataFS reads its configuration from `~/.stratafs/config.json` (or `%USERPROFILE%\.stratafs\config.json` on Windows). Initialize it with sensible defaults:

```bash
stratafs config init
stratafs config show
```

## Full config example

```json
{
  "version": "0.2.0",
  "agent_dir": ".stratafs",
  "global_dir": "/home/you/.stratafs",
  "sources": [
    {
      "id": "default-local",
      "name": "Home Directory",
      "type": "local",
      "enabled": true,
      "path": "/home/you/Documents",
      "local_cache_dir": "",
      "credentials": null,
      "filters": {
        "include_patterns": ["*"],
        "exclude_patterns": [".git/**", "node_modules/**"],
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
  },
  "chunking": {
    "default_strategy": "simple",
    "chunk_size": 1000,
    "overlap_size": 100,
    "min_chunk_size": 50,
    "file_type_strategies": {
      "markdown": "separator",
      "code": "separator",
      "pdf": "sentence",
      "txt": "sentence",
      "csv": "separator"
    }
  }
}
```

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

### `database`

| Key | Default | Description |
| --- | --- | --- |
| `compression_enabled` | `true` | Gzip chunks above the threshold. |
| `compression_threshold` | `512` | Bytes. Chunks smaller than this are stored raw. |
| `maintenance_interval` | `"24h"` | How often to run vacuum / FTS optimize. |
| `deleted_threshold` | `"168h"` | Soft-deleted rows are hard-deleted after this. |

### `chunking`

| Key | Default | Description |
| --- | --- | --- |
| `default_strategy` | `"simple"` | Fallback strategy when an extension isn't mapped. |
| `chunk_size` | `1000` | Target characters per chunk. |
| `overlap_size` | `100` | Overlap between adjacent chunks. |
| `min_chunk_size` | `50` | Drop chunks smaller than this. |
| `file_type_strategies` | _see above_ | Per-extension strategy mapping. |

### `filters`

Each source has its own `filters` block.

| Key | Default | Description |
| --- | --- | --- |
| `include_patterns` | `["*"]` | Glob whitelist applied first. |
| `exclude_patterns` | `[".git/**", "node_modules/**"]` | Glob blacklist applied after include. |
| `max_file_size` | `104857600` (100 MiB) | Skip files larger than this. |
| `ignore_hidden` | `true` | Skip dotfiles and dot-directories. |

## Environment variables

Environment variables override config at runtime — useful for Docker and CI.

| Variable | Effect |
| --- | --- |
| `STRATAFS_GLOBAL_DIR` | Override `global_dir`. |
| `STRATAFS_API_PORT` | Override `server.api_port`. |
| `STRATAFS_MCP_PORT` | Override `server.mcp_port`. |
| `STRATAFS_WORKER_COUNT` | Override `worker.count`. |
| `STRATAFS_EMBEDDING_MODEL` | Override `embedding.model`. |
| `STRATAFS_CHUNK_SIZE` | Override `chunking.chunk_size`. |
| `STRATAFS_CHUNK_STRATEGY` | Override `chunking.default_strategy`. |
| `STRATAFS_LOG_LEVEL` | `debug` / `info` / `warn` / `error`. |

Example:

```bash
STRATAFS_API_PORT=9000 STRATAFS_WORKER_COUNT=8 stratafs serve
```

## Startup validation

StrataFS validates the full config on boot. Failures are reported with a hint:

- Storage credentials are exercised against the backend.
- Directory paths are checked for read access.
- Embedding models are loaded and dimension-checked.
- Ports are probed for conflicts.
- Glob patterns are compiled.

If any check fails, `stratafs serve` exits non-zero with a structured error.
