---
hide:
  - navigation
  - toc
---

<div class="hero" markdown>

# StrataFS

<p class="tagline">A semantic filesystem that transforms passive file storage into an intelligent, searchable knowledge base.</p>

[Get Started](getting-started/quickstart.md){ .md-button .md-button--primary }
[View on GitHub :fontawesome-brands-github:](https://github.com/neul-labs/stratafs){ .md-button }

</div>

## Install in 30 seconds

=== "npm"

    ```bash
    npm install -g stratafs
    stratafs config init
    stratafs serve
    ```

=== "PyPI"

    ```bash
    pip install stratafs
    stratafs config init
    stratafs serve
    ```

=== "macOS / Linux"

    ```bash
    curl -fsSL https://raw.githubusercontent.com/neul-labs/stratafs/main/scripts/install.sh | bash
    stratafs config init
    stratafs serve
    ```

=== "Docker"

    ```bash
    docker run -d \
      --name stratafs \
      -p 8080:8080 -p 8081:8081 \
      -v $(pwd)/data:/app/data \
      ghcr.io/neul-labs/stratafs:latest
    ```

Then search:

```bash
stratafs search "authentication middleware"
# or via REST
curl "http://localhost:8080/search?q=machine+learning"
```

<div class="feature-grid" markdown>

<div class="feature-card" markdown>
### :material-text-search: Hybrid search
Full-text (FTS5) and semantic (vector) ranking fused in a single SQL query. Natural-language queries return chunk-level results in under 100 ms on a laptop.
</div>

<div class="feature-card" markdown>
### :material-robot: AI-native
A built-in [Model Context Protocol](ai-integration/mcp.md) server lets any MCP-aware agent query your filesystem as a structured resource — no glue code needed.
</div>

<div class="feature-card" markdown>
### :material-cloud-outline: Multi-storage
Index local directories, S3, GCS, and Azure Blob with a unified, read-only interface. Each source is isolated in its own SQLite database.
</div>

<div class="feature-card" markdown>
### :material-shield-lock-outline: Read-only by design
StrataFS never writes to your source files. All metadata lives in a parallel `.stratafs/` directory. No permission escalation, no surprise mutations.
</div>

<div class="feature-card" markdown>
### :material-zip-box: Compression-aware
Chunk content above 512 bytes is gzip-compressed at rest. Typical savings: 40–60 % disk overhead with transparent decompression at query time.
</div>

<div class="feature-card" markdown>
### :material-package-variant: Cross-platform
Native binaries for macOS, Linux, and Windows. A Wails-based desktop UI ships alongside the daemon, with system tray integration on every platform.
</div>

</div>

## Why StrataFS?

StrataFS sits between your storage backends and your AI consumers. It does not replace your filesystem — it augments it with a parallel semantic index:

- **No more `grep -r`** — ask natural-language questions across your entire knowledge base.
- **Cross-file context** — find related code, configs, and docs even when filenames don't match.
- **Zero lock-in** — standard SQLite + FTS5, plain HTTP APIs, JSON-on-disk metadata. Walk away whenever you want.
- **Built for agents** — the MCP server speaks the protocol your agent already knows.

## Next steps

- [Quickstart](getting-started/quickstart.md) — index your first source and run a search in under five minutes.
- [Concepts](getting-started/concepts.md) — sources, chunks, embeddings, hybrid search.
- [MCP integration](ai-integration/mcp.md) — wire StrataFS into Claude, ChatGPT, or your own agent.
- [Architecture](architecture/overview.md) — a tour of how the pieces fit together.
