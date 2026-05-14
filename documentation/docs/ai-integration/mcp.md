# Model Context Protocol

StrataFS runs a [Model Context Protocol](https://modelcontextprotocol.io) server alongside the REST API. The MCP server is purpose-built for AI agents: it returns results pre-shaped for LLM context windows, with metadata an agent can act on without extra round-trips.

Default port: `:8081`.

## Endpoints

### `GET /mcp`

Protocol info and capabilities.

```bash
curl http://localhost:8081/mcp
```

```json
{
  "protocol_version": "1.0",
  "server_name": "stratafs-mcp",
  "capabilities": {
    "search": true,
    "resources": true,
    "tools": ["search", "retrieve", "stats"]
  },
  "description": "StrataFS Model Context Protocol Server"
}
```

### `GET /mcp/search`

Search optimized for agent consumption. Same engine as `/search` on the REST API, but the response is trimmed to the fields an agent typically needs.

```bash
curl "http://localhost:8081/mcp/search?q=API+authentication&context=web+development"
```

| Parameter | Default | Description |
| --- | --- | --- |
| `q` | _required_ | The query. |
| `context` | _none_ | Additional terms used to boost relevance (e.g. the conversation topic). |
| `max_results` | `5` | Cap on results (max: 20). |

Response:

```json
{
  "query": "API authentication",
  "context": "web development",
  "results": [
    {
      "resource_id": "docs/api/auth.md#jwt-tokens",
      "title": "JWT Token Authentication",
      "content": "JWT tokens provide stateless authentication...",
      "relevance": 0.92,
      "metadata": {
        "file_type": "markdown",
        "section": "JWT Tokens",
        "source": "API Documentation"
      }
    }
  ],
  "suggested_actions": [
    "Show JWT implementation example",
    "Explain token validation process"
  ]
}
```

`suggested_actions` is a model-generated next-step prompt list. Agents can surface it as quick replies.

### `GET /mcp/resources`

Resource discovery. Lists tools an agent can call.

```json
{
  "resources": [
    { "id": "search",   "name": "Semantic Search",      "type": "tool" },
    { "id": "retrieve", "name": "Document Retrieval",   "type": "tool" }
  ]
}
```

### `POST /mcp/tools/call`

Structured tool execution.

```bash
curl -X POST http://localhost:8081/mcp/tools/call \
  -H 'Content-Type: application/json' \
  -d '{
    "tool": "search",
    "parameters": {
      "query": "error handling patterns",
      "max_results": 3,
      "include_code_examples": true
    }
  }'
```

Response:

```json
{
  "tool": "search",
  "status": "success",
  "result": {
    "matches": 3,
    "resources": [ /* ... */ ]
  },
  "execution_time_ms": 120
}
```

## Wiring it into an agent

### Claude Desktop / Claude Code

Add an entry under `mcpServers` in your client config:

```json
{
  "mcpServers": {
    "stratafs": {
      "command": "stratafs",
      "args": ["serve", "--mcp-only"],
      "env": {
        "STRATAFS_LOG_LEVEL": "warn"
      }
    }
  }
}
```

`--mcp-only` runs without the REST API and without re-spawning a daemon. For a long-running shared daemon, point the client at the existing endpoint via your client's HTTP-MCP bridge.

### Custom Python client

```python
import requests

resp = requests.post(
    "http://localhost:8081/mcp/tools/call",
    json={"tool": "search", "parameters": {"query": "rate limiting"}},
    timeout=10,
)
for hit in resp.json()["result"]["resources"]:
    print(hit["resource_id"], hit["relevance"])
```

### Custom TypeScript client

```typescript
const res = await fetch("http://localhost:8081/mcp/tools/call", {
  method: "POST",
  headers: { "Content-Type": "application/json" },
  body: JSON.stringify({
    tool: "search",
    parameters: { query: "rate limiting", max_results: 5 },
  }),
});
const { result } = await res.json();
console.log(result.resources);
```

## When to prefer MCP over REST

| Use case | API |
| --- | --- |
| Agent tool-use loop | **MCP** — designed for it. |
| Application embedding | REST — richer filters, OpenAPI. |
| One-off scripts | Either. REST is more familiar. |
| Streaming long results | REST (with `Transfer-Encoding: chunked`). |

The two servers share state. Anything indexed via one is visible to the other.
