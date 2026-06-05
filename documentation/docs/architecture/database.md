# Database

Each source gets its own SQLite database under the agent directory for that source (`<source-path>/.stratafs/stratafs.db`). There is no shared central store; adding or removing a source is one filesystem operation. A separate `queue.db` lives in the global config directory and holds the cross-source job queue.

The schema lives in `pkg/database/database.go` (`initSchema`, `enableFTS`).

## Core tables

### `files`

Tracks every file StrataFS has seen.

| Column | Type | Notes |
| --- | --- | --- |
| `id` | INTEGER | Primary key (autoincrement). |
| `path` | TEXT | Path within the source. UNIQUE. |
| `checksum` | TEXT | Content hash for change detection. |
| `size` | INTEGER | Bytes. |
| `created_at` | DATETIME | First seen. |
| `updated_at` | DATETIME | Last (re)indexed. |
| `deleted_at` | DATETIME | Soft-delete timestamp; NULL means live. |

Indexes: `idx_files_path` on `path`, `idx_files_deleted_at` on `deleted_at`.

### `file_chunks`

Where parsed content lives.

| Column | Type | Notes |
| --- | --- | --- |
| `id` | INTEGER | Primary key. |
| `file_id` | INTEGER | FK to `files.id`, ON DELETE CASCADE. |
| `content` | TEXT | Raw chunk text (used directly for small chunks and as the FTS5 source). |
| `content_compressed` | BLOB | Gzip blob when the chunk is compressed. |
| `is_compressed` | BOOLEAN | 1 when the canonical payload lives in `content_compressed`. |
| `embedding` | BLOB | Float32 vector for the chunk (nil if embeddings are disabled). |
| `offset` | INTEGER | Character offset within the parsed text. |
| `length` | INTEGER | Character length. |
| `created_at` | DATETIME | First write. |
| `updated_at` | DATETIME | Last update. |
| `deleted_at` | DATETIME | Soft-delete timestamp. |

Indexes: unique `(file_id, offset)`, plus indexes on `file_id` and `deleted_at`.

Compression kicks in for chunks larger than 512 bytes, and only when gzip yields at least 10% savings (see `compressContent` in `database.go`). Compressed chunks store the canonical payload in `content_compressed` with `is_compressed = 1`; smaller or incompressible chunks stay in `content`.

### `file_chunks_fts`

A virtual FTS5 table backed by `file_chunks.content`:

```sql
CREATE VIRTUAL TABLE file_chunks_fts USING fts5(
  content,
  content='file_chunks',
  content_rowid='id'
);
```

Three triggers (`file_chunks_ai`, `file_chunks_ad`, `file_chunks_au`) keep the FTS index in sync on insert / delete / update; application code never writes to the virtual table directly. If FTS5 isn't compiled into the host SQLite build, `enableFTS` logs a warning and falls back to simple text matching.

### Vector index

Vector search runs through the `sqlite-vec` extension (`github.com/asg017/sqlite-vec-go-bindings/cgo`), loaded by `sqlite_vec.Auto()` when the database opens. Embeddings are persisted in the `file_chunks.embedding` BLOB column and presented to `sqlite-vec` at query time.

## Hybrid query

Hybrid search runs an FTS5 BM25 match against `file_chunks_fts` and a vector lookup via `sqlite-vec`, joins both back to `file_chunks` / `files`, and combines the component scores in Go with `SearchWeights`. Per-source isolation means each query runs against a single SQLite file; there is no central index. See `pkg/search/engine.go` for the exact pipeline.

## Maintenance

`pkg/database` exposes maintenance helpers used by the daemon. The current behaviour:

- Hard-deletes are applied to soft-deleted rows older than 1 day (see the `DELETE FROM file_chunks WHERE deleted_at IS NOT NULL AND deleted_at < datetime('now', '-1 day')` query in `database.go`).
- `INSERT INTO file_chunks_fts(file_chunks_fts) VALUES('optimize')` compacts the FTS index.

`database.maintenance_interval` (default `"24h"`) controls how often the daemon runs this pass; `database.deleted_threshold` lives in the config but the hardcoded "1 day" inside the SQL takes effect today. Tightening the threshold is on the roadmap.

## Compression

`compressContent` decides per chunk:

```go
if len(content) < compressionThreshold {   // 512 bytes
    return content, false
}
gz := gzip(content)
if len(gz) < len(content)*9/10 {           // at least 10% savings
    return gz, true
}
return content, false                       // not worth compressing
```

Reads use `decompressContent`. Compression is opt-out via `database.compression_enabled: false`.

## Soft delete

Removed files are marked `deleted_at = CURRENT_TIMESTAMP` rather than deleted; their chunks get the same treatment via `UPDATE file_chunks SET deleted_at = ...`. Queries filter `WHERE deleted_at IS NULL`. Benefits:

- No race between scanner and concurrent searcher.
- Historical queries are trivially possible (drop the filter).
- Re-creating a deleted file is a single update instead of a re-insert.

## Backup

A live SQLite database can be backed up safely with `sqlite3 <db> ".backup <target>"` or with the `VACUUM INTO` statement. Per-source isolation means each source can be backed up independently:

```bash
# Each enabled source has its own DB under <source-path>/.stratafs/stratafs.db
sqlite3 /path/to/source/.stratafs/stratafs.db ".backup /backup/source.bak"
```

For volume-level backups, snapshot the source directories and the global `~/.stratafs/` together while the daemon is paused for the most consistent state.
