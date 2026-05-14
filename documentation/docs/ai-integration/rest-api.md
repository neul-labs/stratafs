# REST API

The REST API is the general-purpose interface to StrataFS. It is the same engine the [MCP server](mcp.md) sits on, but with richer filters, structured pagination, and an OpenAPI spec.

Default port: `:8080`.

Interactive docs are served from a running daemon:

- **Swagger UI** — <http://localhost:8080/docs>
- **ReDoc** — <http://localhost:8080/redoc>
- **OpenAPI JSON** — <http://localhost:8080/openapi.json>

## Endpoints

### `GET /health`

```json
{
  "status": "healthy",
  "version": "0.2.0",
  "uptime": "2h15m30s",
  "sources": 3,
  "indexed_files": 1250,
  "storage_used": "245MB"
}
```

### `GET /search`

The hybrid search endpoint. See the [Search](../user-guide/search.md) page for all parameters and ranking details.

```bash
curl "http://localhost:8080/search?q=machine+learning&limit=5"
```

Returns:

```json
{
  "query": "machine learning",
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

### `POST /search`

Same engine, JSON body for complex requests. See [Search → REST API](../user-guide/search.md#rest-api).

### `GET /documents/{path}`

Retrieve a full document by URL-encoded path.

```bash
curl "http://localhost:8080/documents/$(printf 'docs/ml/neural-networks.md' | jq -sRr @uri)"
```

```json
{
  "file_path": "/docs/ml/neural-networks.md",
  "source_name": "Documentation",
  "content": "# Neural Networks\n\nNeural networks are...",
  "file_size": 15420,
  "modified_time": "2026-04-22T10:30:00Z",
  "chunks": 12,
  "chunking_strategy": "sentence",
  "processing_stats": {
    "chunks_generated": 12,
    "avg_chunk_size": 284,
    "overlap_size": 50
  }
}
```

### `GET /queue/stats`

```json
{
  "pending_jobs": 5,
  "processing_jobs": 2,
  "completed_jobs": 1248,
  "failed_jobs": 3,
  "worker_count": 4,
  "average_processing_time_ms": 250
}
```

### `GET /chunking/stats`

Chunking strategy usage and compression effectiveness.

### `GET /sources/stats`

Per-source health, file counts, last-scan timestamps, and the last error string if a source is unhappy.

## Error responses

All errors share a common shape:

```json
{
  "error": "invalid_query",
  "message": "Search query cannot be empty",
  "code": 400,
  "timestamp": "2026-04-22T10:30:00Z"
}
```

| Status | Meaning |
| --- | --- |
| `200` | Success. |
| `400` | Bad request (invalid parameters). |
| `404` | Not found. |
| `429` | Rate limited. |
| `500` | Internal server error. |
| `503` | Service unavailable (e.g. embedder still loading). |

## Rate limiting

| API | Limit |
| --- | --- |
| REST | 100 requests/minute/IP |
| MCP | 200 requests/minute/IP |
| Concurrent searches | 10 |

Standard headers are returned on every response:

```http
X-RateLimit-Limit: 100
X-RateLimit-Remaining: 85
X-RateLimit-Reset: 1761133800
```

## Authentication

StrataFS does not implement authentication today — it is designed for local use. For production deployments, place a reverse proxy in front:

- nginx / Caddy / Traefik with basic auth or OAuth2 proxy.
- An API gateway (Kong, Ambassador) for finer-grained policies.
- Network isolation (firewall rules, VPN, Tailscale).

The [Production Checklist](../deployment/production-checklist.md) covers a hardened reference setup.

## Client examples

=== "cURL"

    ```bash
    curl -G "http://localhost:8080/search" \
      --data-urlencode "q=kubernetes deployment" \
      --data-urlencode "sources=devops-docs,k8s-examples" \
      --data-urlencode "limit=10"
    ```

=== "Python"

    ```python
    import requests
    r = requests.get("http://localhost:8080/search",
                     params={"q": "machine learning", "limit": 5})
    for hit in r.json()["results"]:
        print(hit["file_path"], hit["relevance_score"])
    ```

=== "JavaScript"

    ```javascript
    const r = await fetch(
      `http://localhost:8080/search?q=${encodeURIComponent("API docs")}`
    );
    const { results } = await r.json();
    results.forEach(h => console.log(h.file_path, h.relevance_score));
    ```

=== "Go"

    ```go
    resp, _ := http.Get("http://localhost:8080/search?q=auth")
    defer resp.Body.Close()
    var out struct {
        Results []map[string]any `json:"results"`
    }
    json.NewDecoder(resp.Body).Decode(&out)
    ```
