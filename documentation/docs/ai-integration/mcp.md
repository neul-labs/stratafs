# Model Context Protocol

StrataFS runs a [Model Context Protocol](https://modelcontextprotocol.io) server alongside the REST API. The MCP server reuses the same hybrid search engine, with response shapes trimmed for LLM context windows.

Default port: `:8081` (`pkg/protocol/mcp.go`).

## Endpoints

The MCP server registers four routes:

| Method | Path | Purpose |
| --- | --- | --- |
| `GET` | `/mcp` | Protocol version and advertised capabilities. |
| `GET` / `POST` | `/mcp/search` | Search optimized for agent consumption. |
| `GET` | `/mcp/documents/{path}` | Full-document retrieval. |
| `GET` | `/mcp/resources` | List indexed sources as MCP resources. |

### `GET /mcp`

Protocol info and capabilities:

```bash
curl http://localhost:8081/mcp
```

```json
{
  "protocol": "mcp",
  "version": "1.0.0",
  "capabilities": ["search", "resources"]
}
```

### `GET /mcp/search`

Search optimized for agent consumption. Backed by the same `SearchEngine` as the REST `/search` endpoint, but the response is shaped for LLM context windows. The richer filter / weights surface lives on the REST API — see [Search → REST API](../user-guide/search.md#rest-api).

```bash
curl "http://localhost:8081/mcp/search?q=API+authentication&limit=5"
```

| Parameter | Default | Description |
| --- | --- | --- |
| `q` | _required_ | The query. |
| `limit` | `10` | Cap on results. |

### `GET /mcp/documents/{path}`

Fetch a stored document by its path within an indexed source. The path after `/mcp/documents/` is looked up in the per-source database.

```bash
curl "http://localhost:8081/mcp/documents/docs/auth.md"
```

### `GET /mcp/resources`

Enumerate the sources StrataFS is currently indexing as MCP resources. Each entry carries a `type`, `name`, and `path` so an agent can map the resource back to a search call.

```json
{
  "resources": [
    { "type": "directory", "name": "/Users/you/Documents", "path": "/Users/you/Documents" }
  ]
}
```

## Wiring it into an agent

### Claude Desktop / Claude Code

Run `stratafs serve` (which boots both the REST and MCP servers on `:8080` and `:8081`), then point your client at the running daemon's MCP endpoint via your client's HTTP-MCP bridge.

There is no dedicated MCP-only flag today — `serve` always brings both servers up together. The two share state, so anything indexed via one is visible to the other.

### Custom Python client

```python
import requests

resp = requests.get(
    "http://localhost:8081/mcp/search",
    params={"q": "rate limiting", "limit": 5},
    timeout=10,
)
for hit in resp.json()["results"]:
    print(hit["file"], hit["score"])
```

### Custom TypeScript client

```typescript
const res = await fetch(
  `http://localhost:8081/mcp/search?q=${encodeURIComponent("rate limiting")}&limit=5`,
);
const { results } = await res.json();
results.forEach(r => console.log(r.file, r.score));
```

## When to prefer MCP over REST

| Use case | API |
| --- | --- |
| Agent tool-use loop | **MCP** — designed for it. |
| Application embedding | REST — richer filters, OpenAPI. |
| One-off scripts | Either. REST is more familiar. |
| Document retrieval inside an agent | MCP `/mcp/documents/{path}` keeps the conversation on one origin. |

The two servers share state. Anything indexed via one is visible to the other.
