# Search

StrataFS exposes search through three surfaces — CLI, REST, and MCP — all backed by the same engine.

## Modes

`pkg/search` defines five modes (`SearchMode` in `pkg/search/types.go`); the three you'll reach for day-to-day are:

| Mode | What it does | When to use it |
| --- | --- | --- |
| `hybrid` (default) | FTS5 BM25 + vector similarity + metadata score, fused with weighted scoring | General-purpose, always a safe default |
| `fulltext` | FTS5 BM25 only | Exact keyword / phrase / boolean queries |
| `vector` | Cosine similarity only | Semantic queries where wording differs from the source |

Two additional modes — `faceted` (metadata-only filtering) and `weighted` (caller-supplied component weights) — are available through the REST API for advanced callers.

## CLI

```bash
stratafs search "authentication middleware"
stratafs search "k8s deployment" --mode hybrid --limit 5
stratafs search "TODO performance" --mode fulltext
stratafs search "explain caching layer" --mode vector --content
```

Flags:

| Flag | Default | Description |
| --- | --- | --- |
| `--mode` | `hybrid` | One of `hybrid`, `fulltext`, `vector`. |
| `--limit` | `10` | Maximum results. |
| `--content` | `false` | Include full chunk content in the output. |
| `--json` | `false` | Emit JSON instead of formatted text. |

## REST API

```bash
curl "http://localhost:8080/search?q=authentication+middleware&limit=5"
```

Common query parameters (parsed by `parseSearchParams` in `pkg/api/server.go`):

| Parameter | Default | Description |
| --- | --- | --- |
| `q` | _required_ | Query string. Mapped to `SearchRequest.Query`. |
| `mode` | `hybrid` | `hybrid` \| `fulltext` \| `vector` \| `faceted` \| `weighted`. |
| `limit` | `10` | Max results. |
| `extensions` | _all_ | Comma-separated file extensions, e.g. `md,go`. |

For complex requests, `POST /search` accepts the full `SearchRequest` JSON shape:

```bash
curl -X POST http://localhost:8080/search \
  -H 'Content-Type: application/json' \
  -d '{
    "query": "authentication middleware",
    "mode": "weighted",
    "limit": 10,
    "weights": {
      "fulltext": 0.4,
      "vector": 0.3,
      "recency": 0.1,
      "filename": 0.1,
      "filetype": 0.05,
      "filesize": 0.05
    },
    "filters": {
      "file_extensions": [".go", ".md"],
      "directories": ["pkg/", "docs/"]
    },
    "include_content": true,
    "highlight_results": false
  }'
```

The `SearchFilters` block supports `file_extensions`, `file_types`, `directories`, `min_size` / `max_size`, `modified_after` / `modified_before`, `created_after` / `created_before`, `has_embeddings`, `min_length` / `max_length`, and `languages`.

## Ranking weights

The hybrid scorer fuses six components — `SearchWeights` in `pkg/search/types.go`:

| Weight | Default | What it scores |
| --- | --- | --- |
| `fulltext` | `0.4` | FTS5 BM25. |
| `vector` | `0.3` | Cosine similarity from `sqlite-vec`. |
| `recency` | `0.1` | How recently the file was modified. |
| `filename` | `0.1` | Filename token overlap with the query. |
| `filetype` | `0.05` | Bonus for code / doc extensions over noise. |
| `filesize` | `0.05` | Penalty for outlier file sizes. |

`hybrid` uses these defaults. `weighted` uses whatever you send in the request body.

## Hybrid query, under the hood

Hybrid search runs as a single statement that combines an FTS5 BM25 match over the `file_chunks_fts` virtual table with a vector lookup against the `sqlite-vec` index, joined back to `file_chunks` / `files`. Component scores are normalised in Go and combined with `SearchWeights`. Per-source isolation means each query runs against a single SQLite file — there is no application-level merge across sources beyond ranking the merged result list.

## Response shape

The handler returns a `SearchResponse` containing the results array, the resolved `mode`, the original `query`, and `search_time_ms`. Each `SearchResult` carries the file path, chunk content / snippet, the per-component scores (`fulltext_score`, `vector_score`, `recency_score`, `filename_score`, `filetype_score`, `filesize_score`), and optional `metadata` and `chunk_offset` / `chunk_length` fields. See `pkg/search/types.go` for the exact JSON tags.

## Tips

- Quote phrases for exact-match FTS: `stratafs search '"feature flag"' --mode fulltext`.
- Combine modes: use `fulltext` first; if results are thin, retry with `hybrid`.
- Filter aggressively. `extensions=go,md` over a five-source index is often 10× faster.
- For agent traffic, prefer the MCP endpoint — it returns chunks already shaped for LLM context windows.
