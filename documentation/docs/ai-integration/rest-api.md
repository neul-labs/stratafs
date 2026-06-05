# REST API

The REST API is the general-purpose interface to StrataFS. The [MCP server](mcp.md) sits on the same search engine, with response shapes tuned for agent consumption.

Default port: `:8080` (`pkg/api/server.go`).

Interactive docs are served from a running daemon:

- **Swagger UI** — <http://localhost:8080/docs>
- **ReDoc** — <http://localhost:8080/redoc>
- **OpenAPI JSON** — <http://localhost:8080/openapi.json>

## Endpoints

The registered routes (see `Start` in `pkg/api/server.go`):

| Method | Path | Handler |
| --- | --- | --- |
| `GET` | `/health` | Liveness probe + version. |
| `GET` / `POST` | `/search` | Hybrid / FTS / vector / faceted / weighted search. |
| `GET` | `/documents/{path}` | Full document retrieval by path. |
| `GET` | `/queue/stats` | Job queue depth and worker status. |
| `GET` | `/docs` | Swagger UI. |
| `GET` | `/redoc` | ReDoc. |
| `GET` | `/openapi.json` | OpenAPI 3.0 spec. |

### `GET /health`

```json
{
  "status": "ok",
  "version": "0.2.0"
}
```

The handler returns the StrataFS version it was built against. For deeper liveness signals (queue depth, parse errors), poll `/queue/stats`.

### `GET /search`

The hybrid search endpoint. See the [Search](../user-guide/search.md) page for all parameters and ranking details.

```bash
curl "http://localhost:8080/search?q=machine+learning&limit=5"
```

Returns a `SearchResponse`. The full shape and per-component scores are documented in [Search → Response shape](../user-guide/search.md#response-shape).

### `POST /search`

Same engine, JSON body for complex requests. Accepts the full `SearchRequest` (mode, weights, filters, pagination, content / metadata toggles). See [Search → REST API](../user-guide/search.md#rest-api) for the JSON shape.

### `GET /documents/{path}`

Retrieve a stored document by path. The path after `/documents/` is treated as the file path to look up in the database.

```bash
curl "http://localhost:8080/documents/docs/ml/neural-networks.md"
```

Returns a `DocumentResponse` containing the file metadata, the joined chunk content, and the per-chunk offsets. Missing files return `404`.

### `GET /queue/stats`

Returns the job queue snapshot from `pkg/queue`: pending / processing / completed / failed counts plus the active worker count. Useful for monitoring whether indexing is keeping up with file changes.

## Error responses

The handlers in `pkg/api/server.go` emit plain-text errors via `http.Error` with a matching status code. The most common codes:

| Status | Meaning |
| --- | --- |
| `200` | Success. |
| `400` | Bad request (e.g. missing `q`, invalid JSON body). |
| `404` | Not found (e.g. unknown document path). |
| `500` | Internal server error. |
| `503` | Service unavailable (search engine or queue not initialised). |

Structured JSON error envelopes are tracked on the [Roadmap](../contributing/roadmap.md).

## Authentication and rate limiting

StrataFS does not implement authentication or in-process rate limiting today — it is designed for local-trust use. For production deployments, place a reverse proxy in front:

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
