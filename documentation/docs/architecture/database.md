# Database

Each source gets its own SQLite database under `~/.stratafs/`. There is no shared central store — adding or removing a source is one filesystem operation.

## Schema

The core tables:

### `files`

Tracks every file StrataFS has seen.

| Column | Type | Notes |
| --- | --- | --- |
| `id` | INTEGER | Primary key. |
| `path` | TEXT | Path within the source. UNIQUE. |
| `size` | INTEGER | Bytes. |
| `mtime` | INTEGER | UNIX ms. |
| `hash` | TEXT | Content hash for change detection. |
| `parser` | TEXT | The parser that handled this file. |
| `created_at` | INTEGER | First seen. |
| `updated_at` | INTEGER | Last (re)indexed. |
| `deleted_at` | INTEGER | Soft-delete timestamp; NULL means live. |

### `file_chunks`

Where parsed content lives.

| Column | Type | Notes |
| --- | --- | --- |
| `id` | INTEGER | Primary key. |
| `file_id` | INTEGER | FK to `files.id`. |
| `offset` | INTEGER | Char offset in the parsed text. |
| `length` | INTEGER | Char length. |
| `strategy` | TEXT | `simple` / `sentence` / `separator` / `token`. |
| `content` | TEXT | Raw chunk text (if not compressed). |
| `content_compressed` | BLOB | Gzip blob (if compressed). |
| `is_compressed` | INTEGER | 0/1. |
| `embedding_dim` | INTEGER | Vector dimension. |
| `deleted_at` | INTEGER | Soft-delete timestamp. |

Compression kicks in when `length > database.compression_threshold` (default 512 bytes). Typical savings: 40–60% disk.

### `file_chunks_fts`

A virtual FTS5 table over `file_chunks.content`. Auto-maintained by triggers — application code never writes to it directly.

```sql
CREATE VIRTUAL TABLE file_chunks_fts USING fts5(
  content,
  content=file_chunks,
  content_rowid=id,
  tokenize='porter unicode61'
);
```

### `file_chunks_vec`

A virtual `sqlite-vec` table for cosine similarity:

```sql
CREATE VIRTUAL TABLE file_chunks_vec USING vec0(
  embedding float[768]
);
```

The dimension matches `embedding.dimension`. Switching models means rebuilding this table.

## Hybrid query

A single SQL statement uses CTEs to fuse FTS, vector, and metadata scores:

```sql
WITH fts AS (
  SELECT rowid AS chunk_id, bm25(file_chunks_fts) AS bm25
  FROM file_chunks_fts WHERE file_chunks_fts MATCH ? LIMIT ?
),
vec AS (
  SELECT rowid AS chunk_id, distance
  FROM file_chunks_vec
  WHERE embedding MATCH ? AND k = ?
)
SELECT
  c.*,
  f.path,
  COALESCE(fts.bm25, 0) AS fts_score,
  COALESCE(vec.distance, 1) AS vec_dist,
  -- weighted final score
  (:w_fts * normalize(fts.bm25) +
   :w_vec * (1 - vec.distance) +
   :w_meta * metadata_score(f.path, f.mtime, ?)) AS score
FROM file_chunks c
JOIN files f ON f.id = c.file_id
LEFT JOIN fts ON fts.chunk_id = c.id
LEFT JOIN vec ON vec.chunk_id = c.id
WHERE c.deleted_at IS NULL AND f.deleted_at IS NULL
ORDER BY score DESC
LIMIT ?;
```

Doing the work in one query keeps everything inside a single SQLite transaction. There is no application-level merge step.

## Maintenance

A background task runs on `database.maintenance_interval` (default 24 h):

- `VACUUM` to reclaim space from soft-deleted rows.
- `INSERT INTO file_chunks_fts(file_chunks_fts) VALUES('optimize')` to compact the FTS5 index.
- Hard-delete rows where `deleted_at < now - database.deleted_threshold`.

## Compression

The trigger that writes a chunk picks raw vs. compressed at write time:

```python
if len(content) > threshold:
    store_compressed(gzip(content))
    is_compressed = 1
else:
    store_raw(content)
    is_compressed = 0
```

Reads transparently decompress. Compression is opt-out via `database.compression_enabled: false`.

## Soft delete

Removed files are marked `deleted_at = now()` rather than deleted. Queries always filter `WHERE deleted_at IS NULL`. Benefits:

- No race between scanner and concurrent searcher.
- Historical queries are trivially possible (just remove the filter).
- Re-creating a deleted file is a single update instead of a re-insert.

The maintenance task hard-deletes after `database.deleted_threshold` (default 7 days).

## Backup

A live SQLite database can be backed up safely with `sqlite3 <db> ".backup <target>"` or with the `VACUUM INTO` statement. Per-source isolation means each source can be backed up independently:

```bash
for db in ~/.stratafs/sources/*/db.sqlite; do
  sqlite3 "$db" ".backup ${db}.bak"
done
```

For volume-level backups, snapshot `~/.stratafs/` while the daemon is paused (`systemctl --user stop stratafs`) for the most consistent state.
