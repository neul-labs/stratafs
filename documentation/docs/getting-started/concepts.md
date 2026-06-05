# Concepts

A short glossary of the moving parts. Each term has a longer treatment elsewhere in the docs — links inline.

## Source

A **source** is something StrataFS indexes: a local directory, an S3 bucket prefix, a GCS path, an Azure container, etc. Each source has an `id`, a backend `type`, a `path`, optional credentials, and filters. Each source gets its own SQLite database — there is no shared central store.

See [Storage Backends](../user-guide/storage-backends.md).

## Monitor

The **monitor** watches a source for changes. For local paths it uses `fsnotify` for real-time events; for cloud paths it polls on `worker.scan_interval`. When it sees a new or modified file, it enqueues a job.

## Queue

The **queue** is a SQLite-backed priority job queue. Jobs are parse / embed / index / cleanup tasks. The queue handles retries with exponential backoff (`max_retries = 3` by default), survives restarts, and lets you scale workers independently of producers.

## Parser

A **parser** extracts plain text from a file. StrataFS ships a parser registry covering Markdown, PDF, DOCX, XLSX/XLS, CSV, HTML, JSON/YAML/TOML, and many source-code formats. You can register your own parser for a new extension; see [Contributing → Development](../contributing/development.md).

See [File Types](../user-guide/file-types.md).

## Chunk

A **chunk** is a substring of a parsed file — small enough to embed cleanly, large enough to carry meaning. StrataFS ships four chunking strategies in `pkg/chunking`:

| Strategy | When it's used |
| --- | --- |
| `simple` | Default fallback. Fixed-width windows with overlap. |
| `sentence` | Plain text, PDFs. Splits on sentence boundaries. |
| `separator` | Markdown, code, CSV. Splits on natural separators (headings, blank lines, commas). |
| `token` | Strict token-budget chunking for downstream LLM cost control. |

The mapping from file type to strategy lives in the queue processor and parser layers; it is not exposed as user-tunable config today.

## Embedding

An **embedding** is a fixed-length vector representation of a chunk's meaning. StrataFS uses [FastEmbed-go](https://github.com/anush008/fastembed-go) with ONNX Runtime. The default model is BGE Base EN v1.5 (768 dimensions); smaller faster models like BGE Small (384 dimensions) are one config change away.

Embeddings are stored alongside chunks in the [sqlite-vec](https://github.com/asg017/sqlite-vec) index for that source's database.

## Hybrid search

The search engine runs a **single SQL query** that combines:

- **FTS5 BM25** ranking from SQLite's full-text search extension.
- **Cosine similarity** against the vector index.
- **Metadata scoring** (recency, filename match, file type bonus).

These are fused with configurable per-query weights. You can also call FTS-only or vector-only modes if you know what you want.

See [Search](../user-guide/search.md).

## Per-source isolation

Each source gets its own SQLite database file under `~/.stratafs/`. There is no central registry. This means:

- You can add or remove a source by editing one file and restarting.
- A corrupted source database affects only that source.
- Backups are per-source — copy one file, restore one file.

## Read-only architecture

StrataFS never writes to source files. All StrataFS state (queues, chunks, embeddings, caches) lives in `.stratafs/` directories that StrataFS owns. The implication: pointing StrataFS at a directory is a strictly additive operation.

## Soft delete

When a file disappears from a source, its chunks are marked `deleted_at` rather than removed. This keeps existing references valid, supports historical queries, and avoids races between the scanner and concurrent searchers. A maintenance job hard-deletes stale rows after `database.deleted_threshold` (default: 7 days).

## REST API

A standard HTTP/JSON API for search, document retrieval, and stats. Default port `:8080`. Spec at `/openapi.json`, Swagger UI at `/docs`, ReDoc at `/redoc`.

See [REST API](../ai-integration/rest-api.md).

## MCP server

A separate HTTP server speaking the [Model Context Protocol](https://modelcontextprotocol.io), tuned for AI agent consumption. Default port `:8081`.

See [Model Context Protocol](../ai-integration/mcp.md).
