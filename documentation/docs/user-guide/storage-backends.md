# Storage Backends

StrataFS supports four storage backends through a unified interface. All sources are **read-only** — StrataFS never modifies the underlying files.

| Backend | `type` | Real-time updates | Credentials |
| --- | --- | --- | --- |
| Local filesystem | `local` | Yes (fsnotify) | None |
| Amazon S3 (and compatibles) | `s3` | Polling | Access key + secret |
| Google Cloud Storage | `gcs` | Polling | Service account JSON |
| Azure Blob Storage | `azure` | Polling | Account key |

## Local

```json
{
  "id": "my-documents",
  "name": "My Documents",
  "type": "local",
  "enabled": true,
  "path": "/home/you/Documents",
  "filters": {
    "include_patterns": ["*"],
    "exclude_patterns": [".git/**", "node_modules/**"],
    "max_file_size": 104857600,
    "ignore_hidden": true
  }
}
```

Local sources use `fsnotify` for real-time change detection. There is no caching layer — files are read directly from disk.

## Amazon S3

```json
{
  "id": "s3-documents",
  "name": "S3 Documents",
  "type": "s3",
  "enabled": true,
  "path": "my-bucket/documents/",
  "local_cache_dir": "/home/you/.stratafs/cache/s3-documents",
  "credentials": {
    "access_key": "AKIA...",
    "secret_key": "...",
    "region": "us-west-2",
    "endpoint": ""
  },
  "filters": {
    "include_patterns": ["*.pdf", "*.docx", "*.md"],
    "max_file_size": 52428800
  }
}
```

The `s3` backend works with anything that speaks the S3 API:

- AWS S3 — leave `endpoint` empty.
- MinIO, Wasabi, DigitalOcean Spaces, Cloudflare R2 — set `endpoint` to the provider's endpoint URL.

Remote files are fetched to `local_cache_dir`, parsed, embedded, then evicted. Set `worker.scan_interval` higher (e.g. `"5m"`) for large or low-traffic buckets to reduce API costs.

## Google Cloud Storage

```json
{
  "id": "gcs-documents",
  "name": "GCS Documents",
  "type": "gcs",
  "enabled": true,
  "path": "my-bucket/documents/",
  "local_cache_dir": "/home/you/.stratafs/cache/gcs-documents",
  "credentials": {
    "project_id": "my-project",
    "credentials_path": "/etc/stratafs/gcs-sa.json"
  }
}
```

The service account needs the **Storage Object Viewer** role on the target bucket. `credentials_path` may also be a relative path — StrataFS resolves it from the global config directory.

## Azure Blob Storage

```json
{
  "id": "azure-documents",
  "name": "Azure Documents",
  "type": "azure",
  "enabled": true,
  "path": "my-container/documents/",
  "local_cache_dir": "/home/you/.stratafs/cache/azure-documents",
  "credentials": {
    "account_name": "mystorageaccount",
    "account_key": "...",
    "container": "my-container"
  }
}
```

The container must already exist. StrataFS will not create it.

## Workflow differences

**Local sources** observe events in real time. A new file is indexed within seconds.

**Remote sources** scan on `worker.scan_interval`:

1. List the prefix / container.
2. Compare mtimes / etags against the last index.
3. Download changed files into `local_cache_dir`.
4. Parse, chunk, embed, write to SQLite.
5. Evict the cache entry.

## Filters

Each source has its own `filters` block. Filters apply in order: `include_patterns` first, then `exclude_patterns`, then `max_file_size`, then `ignore_hidden`. Use the `*.ext` style for simple extensions and `**` for recursive matches.

```json
{
  "filters": {
    "include_patterns": ["*.md", "*.pdf", "src/**/*.go"],
    "exclude_patterns": [
      "*.tmp", "*.log",
      "vendor/**", "node_modules/**",
      "*.zip", "*.tar.gz",
      "*.mp4", "*.jpg", "*.png"
    ],
    "max_file_size": 10485760,
    "ignore_hidden": true
  }
}
```

## Tuning for large datasets

```json
{
  "worker": {
    "count": 8,
    "scan_interval": "30s",
    "batch_size": 20
  },
  "database": {
    "compression_enabled": true,
    "maintenance_interval": "12h"
  }
}
```

- Increase `worker.count` to saturate available CPU.
- Lengthen `scan_interval` for buckets that rarely change.
- Keep `compression_enabled: true` — it's effectively free.
- Schedule `maintenance_interval` more frequently if you churn a lot of files.

## Troubleshooting

**Credential errors**
: Run `curl http://localhost:8080/sources/stats` — failed sources report their last error. Common causes: wrong region, missing IAM permission, expired key.

**Slow scans**
: Check `local_cache_dir` disk usage and free space. Raise `worker.scan_interval` and trim `include_patterns`.

**Files missing from search**
: Confirm the file passed filters by inspecting the queue stats endpoint. Hidden files require `ignore_hidden: false`. Files above `max_file_size` are silently skipped.

```bash
# Verbose logs to debug a source
STRATAFS_LOG_LEVEL=debug stratafs serve
```
