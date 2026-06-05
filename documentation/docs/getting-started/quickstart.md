# Quickstart

This guide takes you from a fresh install to your first search in under five minutes.

## 1. Initialize config

```bash
stratafs config init
```

This writes a default config to `~/.stratafs/config.json` and creates the supporting directories (cache, logs, embedding model cache).

Open the resulting `~/.stratafs/config.json` in your editor to inspect the defaults.

## 2. Add a source

A **source** is a directory or cloud bucket that StrataFS will index. The simplest case is your local Documents folder. Edit `~/.stratafs/config.json` and add an entry to the `sources` array:

```json
{
  "id": "my-docs",
  "name": "My Documents",
  "type": "local",
  "enabled": true,
  "path": "/Users/you/Documents",
  "filters": {
    "include_patterns": ["*"],
    "exclude_patterns": [".git/**", "node_modules/**"],
    "max_file_size": 104857600,
    "ignore_hidden": true
  }
}
```

For cloud buckets (S3, GCS, Azure) see [Storage Backends](../user-guide/storage-backends.md).

## 3. Start the daemon

```bash
stratafs serve
```

You should see log lines for the REST API (`:8080`), the MCP server (`:8081`), and the file watcher kicking off the initial scan. Indexing typically runs at 50–100 files/sec depending on file size and the embedding model.

Watch progress in another terminal:

```bash
curl -s http://localhost:8080/queue/stats | jq
```

## 4. Run a search

=== "CLI"

    ```bash
    stratafs search "authentication middleware"
    ```

=== "REST"

    ```bash
    curl "http://localhost:8080/search?q=authentication+middleware&limit=5"
    ```

=== "MCP"

    ```bash
    curl "http://localhost:8081/mcp/search?q=authentication+middleware"
    ```

The CLI shows ranked chunks with file paths and relevance scores. The REST and MCP endpoints return JSON suitable for piping into other tools or for an agent to consume directly.

## 5. Wire up an agent

The MCP server speaks the [Model Context Protocol](https://modelcontextprotocol.io). Point any MCP-capable client at `http://localhost:8081/mcp` to expose your indexed content as a structured resource. See [AI Integration → MCP](../ai-integration/mcp.md) for client examples.

## What just happened?

1. The **monitor** detected files under your source and pushed jobs onto the **queue**.
2. Each job ran through a **parser** (markdown, PDF, code-aware, etc.) and was split into **chunks** by a strategy chosen for the file type.
3. Each chunk got a **vector embedding** from the FastEmbed/ONNX runtime.
4. Chunks and embeddings were stored in a **per-source SQLite database** with FTS5 + `sqlite-vec`.
5. Your query ran a single SQL statement that fused FTS5 BM25 with cosine similarity and returned ranked results.

For the long version, see [Concepts](concepts.md) and the [Architecture overview](../architecture/overview.md).

## Next steps

- [Configuration reference](../user-guide/configuration.md) — every knob, with defaults.
- [Search guide](../user-guide/search.md) — hybrid / FTS-only / semantic modes, filters, ranking weights.
- [CLI reference](../user-guide/cli.md) — `stratafs serve`, `search`, `config`, `fs export`, `mount`.
- [Desktop app](../desktop/overview.md) — if you'd rather not touch a terminal.
