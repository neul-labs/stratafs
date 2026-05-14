# Search

StrataFS exposes search through three surfaces — CLI, REST, and MCP — all backed by the same engine.

## Modes

| Mode | What it does | When to use it |
| --- | --- | --- |
| `hybrid` (default) | FTS5 BM25 + vector similarity + metadata score, fused with configurable weights | General-purpose, always a safe default |
| `fulltext` | FTS5 BM25 only | Exact keyword / phrase / boolean queries |
| `vector` | Cosine similarity only | Semantic queries where wording differs from the source |

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

| Parameter | Default | Description |
| --- | --- | --- |
| `q` | _required_ | Query string. |
| `mode` | `hybrid` | `hybrid` \| `fulltext` \| `vector`. |
| `limit` | `10` | Max results (cap: 100). |
| `offset` | `0` | Pagination offset. |
| `sources` | _all_ | Comma-separated source IDs. |
| `extensions` | _all_ | Comma-separated file extensions, e.g. `md,go`. |
| `types` | _all_ | High-level types: `code`, `docs`, `data`. |
| `directories` | _all_ | Comma-separated path prefixes. |
| `min_size`, `max_size` | _none_ | Filter by file size (bytes). |
| `include_content` | `true` | Return chunk content. |
| `include_metadata` | `true` | Return chunk + file metadata. |
| `highlight` | `false` | Wrap matched terms in `<mark>` tags. |
| `sort_by` | `relevance` | `relevance` \| `modified` \| `path` \| `size`. |
| `sort_order` | `desc` | `asc` \| `desc`. |

For complex requests prefer `POST /search` with a JSON body:

```bash
curl -X POST http://localhost:8080/search \
  -H 'Content-Type: application/json' \
  -d '{
    "q": "authentication middleware",
    "mode": "hybrid",
    "limit": 10,
    "weights": {
      "fts": 0.5,
      "vector": 0.4,
      "metadata": 0.1
    },
    "filters": {
      "extensions": ["go", "md"],
      "directories": ["pkg/", "docs/"]
    }
  }'
```

## Ranking weights

In `hybrid` mode the final score for a chunk is:

```
score = (w_fts × bm25_norm) + (w_vector × cosine) + (w_metadata × metadata_score)
```

Defaults: `w_fts = 0.5`, `w_vector = 0.4`, `w_metadata = 0.1`. Tune them per-request via the `weights` block on the POST body, or globally in `config.json`.

`metadata_score` rewards:

- Filename matches against the query terms.
- Recently modified files.
- Files in directories whose names match the query.
- Common code/doc extensions over binary blobs.

## Hybrid query, under the hood

StrataFS runs a single SQL statement using CTEs:

```sql
WITH fts_results AS (
  SELECT file_id, chunk_id, bm25(file_chunks_fts) AS score
  FROM file_chunks_fts WHERE file_chunks_fts MATCH ?
  ORDER BY score LIMIT ?
),
vec_results AS (
  SELECT file_id, chunk_id, distance AS score
  FROM file_chunks_vec WHERE embedding MATCH ? AND k = ?
)
SELECT ... FROM file_chunks
  LEFT JOIN fts_results USING (chunk_id)
  LEFT JOIN vec_results USING (chunk_id)
  ORDER BY weighted_score DESC LIMIT ?;
```

This means there is no application-level result merging — the database does the work, and execution stays inside a single SQLite transaction.

## Response shape

```json
{
  "query": "machine learning algorithms",
  "total_results": 42,
  "results": [
    {
      "file_path": "/docs/ml/neural-networks.md",
      "source_name": "Documentation",
      "chunk_content": "Neural networks are a class of machine learning algorithms...",
      "relevance_score": 0.95,
      "chunk_offset": 1024,
      "chunk_length": 256,
      "chunk_strategy": "sentence",
      "file_size": 15420,
      "modified_time": "2026-04-22T10:30:00Z"
    }
  ],
  "search_time_ms": 45
}
```

## Tips

- Quote phrases for exact-match FTS: `stratafs search '"feature flag"' --mode fulltext`.
- Combine modes: use `fulltext` first; if results are thin, retry with `hybrid`.
- Filter aggressively. `extensions=go,md` over a five-source index is often 10× faster.
- For agent traffic, prefer the MCP endpoint — it returns chunks already shaped for LLM context windows.
