<div align="center">

# StrataFS

**A semantic filesystem that turns passive file storage into an intelligent, searchable knowledge base — built for the AI era.**

[![npm](https://img.shields.io/npm/v/stratafs.svg?logo=npm&label=npm)](https://www.npmjs.com/package/stratafs)
[![PyPI](https://img.shields.io/pypi/v/stratafs.svg?logo=pypi&label=pypi)](https://pypi.org/project/stratafs/)
[![Go Version](https://img.shields.io/badge/go-1.24-00ADD8?logo=go)](https://golang.org)
[![License](https://img.shields.io/badge/license-MIT-blue.svg)](LICENSE)
[![CI](https://github.com/neul-labs/stratafs/actions/workflows/ci.yml/badge.svg)](https://github.com/neul-labs/stratafs/actions/workflows/ci.yml)
[![Docker](https://img.shields.io/badge/docker-ghcr.io-blue?logo=docker)](https://github.com/neul-labs/stratafs/pkgs/container/stratafs)

[Documentation](documentation/) · [Quickstart](documentation/docs/getting-started/quickstart.md) · [Architecture](documentation/docs/architecture/overview.md) · [MCP for agents](documentation/docs/ai-integration/mcp.md) · [Roadmap](documentation/docs/contributing/roadmap.md)

</div>

---

StrataFS watches your directories — local **or** cloud — parses files into semantic chunks, generates vector embeddings, and exposes everything through a hybrid search engine that combines full-text and semantic similarity. It speaks the [Model Context Protocol](https://modelcontextprotocol.io), so any MCP-aware agent can use your filesystem as a structured knowledge resource. No SaaS. No lock-in. Read-only by design.

```bash
# 30 seconds to your first semantic search:
npm install -g stratafs && stratafs config init && stratafs serve &
stratafs search "where do we handle JWT refresh?"
```

---

## Install

<table>
<tr>
<td>

**npm**

```bash
npm install -g stratafs
```

</td>
<td>

**PyPI**

```bash
pip install stratafs
```

</td>
<td>

**Homebrew**

```bash
brew tap neul-labs/stratafs
brew install stratafs
```

</td>
</tr>
<tr>
<td>

**macOS / Linux**

```bash
curl -fsSL https://raw.githubusercontent.com/neul-labs/stratafs/main/scripts/install.sh | bash
```

</td>
<td>

**Docker**

```bash
docker run -d -p 8080:8080 -p 8081:8081 \
  ghcr.io/neul-labs/stratafs:latest
```

</td>
<td>

**From source**

```bash
git clone https://github.com/neul-labs/stratafs.git
cd stratafs && make build
```

</td>
</tr>
</table>

Then:

```bash
stratafs config init       # writes ~/.stratafs/config.json
stratafs serve             # REST on :8080, MCP on :8081
stratafs search "any natural language query"
```

Native installers (NSIS for Windows, signed `.pkg` for macOS, `.deb` / AppImage for Linux) are on the [releases page](https://github.com/neul-labs/stratafs/releases).

---

## See it work

```bash
$ stratafs search "rate limit middleware"

  pkg/api/middleware/ratelimit.go   ★ 0.94
  ────────────────────────────────────────
  func RateLimit(rps int) gin.HandlerFunc {
      bucket := tokenbucket.New(rps, rps*2)
      return func(c *gin.Context) {
          if !bucket.Take(1) { c.AbortWithStatus(429) ...

  docs/api/rate-limiting.md         ★ 0.88
  ────────────────────────────────────────
  Per-IP rate limits default to 100 requests/min...

  internal/gateway/policy.yaml      ★ 0.71
  ────────────────────────────────────────
  policies:
    - name: api-default
      rate: 100/m
      burst: 200
```

The same query over the REST API:

```bash
curl "http://localhost:8080/search?q=rate+limit+middleware&limit=5" | jq
```

Or from an MCP-aware agent — no glue code required:

```json
{
  "mcpServers": {
    "stratafs": { "command": "stratafs", "args": ["serve", "--mcp-only"] }
  }
}
```

---

## Why StrataFS

<table>
<tr>
<td width="33%" valign="top">

### For developers

Stop `grep`-ing. Ask **natural-language questions** across your entire codebase, configs, and docs at the same time. StrataFS finds related code even when filenames don't match.

- One command from install to first search
- Works offline — no API keys, no telemetry
- Plain HTTP API, plain SQLite on disk
- Drops into any agent loop via MCP

</td>
<td width="33%" valign="top">

### For architects

A clean, layered design built around three first-class invariants: **read-only sources**, **per-source isolation**, and **hybrid scoring in a single SQL query**.

- SQLite + FTS5 + `sqlite-vec` — zero ops dependencies
- Per-source DB → backup, drop, migrate one source at a time
- Pluggable parsers, chunkers, storage backends, embedders
- Streaming pipeline — constant memory regardless of file size

</td>
<td width="33%" valign="top">

### For builders

Every layer is an extension point. Add a parser, a backend, a chunker, or a ranking signal in a single Go file.

- Modular package layout (`pkg/parsers`, `pkg/chunking`, `pkg/storage`, …)
- MCP server speaks the same protocol your agent already knows
- FUSE / WinFsp export — mount the semantic index as a real filesystem
- Wails-based desktop UI for non-CLI users

</td>
</tr>
</table>

---

## Try it in five steps

```bash
# 1. Install
pip install stratafs

# 2. Initialize
stratafs config init

# 3. Add a source (edit ~/.stratafs/config.json)
#    {"id":"docs","type":"local","path":"/path/to/anything","enabled":true}

# 4. Start the daemon
stratafs serve &

# 5. Search — CLI, REST, or MCP
stratafs search "the thing I half-remember writing"
curl "http://localhost:8080/search?q=onboarding+flow"
```

The first scan runs at 50–100 files/sec. Searches return in under 100 ms once the index is warm. Everything lives under `~/.stratafs/` — one directory, one filesystem, one source of truth.

---

## Architecture at a glance

```
┌──────────────┐  ┌──────────────┐  ┌──────────────┐
│  REST API    │  │  MCP Server  │  │   CLI / UI   │
│    :8080     │  │     :8081    │  │              │
└──────┬───────┘  └──────┬───────┘  └──────┬───────┘
       └─────────────────┼─────────────────┘
                         │
              ┌──────────▼──────────┐
              │   Hybrid Search     │
              │  FTS5  +  Vector    │
              │   (single SQL CTE)  │
              └──────────┬──────────┘
                         │
        ┌────────────────┼────────────────┐
        │                │                │
  ┌─────▼─────┐    ┌─────▼─────┐    ┌─────▼─────┐
  │ SQLite +  │    │ FastEmbed │    │ Job Queue │
  │ sqlite-vec│    │  + ONNX   │    │ (SQLite)  │
  └───────────┘    └───────────┘    └─────┬─────┘
                                          │
                                ┌─────────▼─────────┐
                                │  Monitor (local + │
                                │   remote scanner) │
                                └─────────┬─────────┘
                                          │
                              ┌───────────▼───────────┐
                              │   Storage Factory     │
                              └───────────┬───────────┘
                                          │
              ┌───────────────────────────┼──────────────────────────┐
              │                           │                          │
       ┌──────▼──────┐            ┌───────▼───────┐          ┌───────▼───────┐
       │  Local FS   │            │ S3 / GCS /    │          │   Future      │
       │ (fsnotify)  │            │ Azure Blob    │          │   backends    │
       └─────────────┘            └───────────────┘          └───────────────┘
```

Four invariants do most of the work:

1. **Read-only sources** — StrataFS never writes back. All state lives in `.stratafs/`.
2. **Per-source SQLite** — no central registry, no shared bottleneck.
3. **Compression-aware schema** — gzip above 512 bytes, transparent at query time. 40–60% disk savings.
4. **Soft delete** — files disappear consistently, historical queries are free.

Long version: [Architecture overview](documentation/docs/architecture/overview.md) · [Database internals](documentation/docs/architecture/database.md).

---

## Integrate it

### REST

```python
import requests
r = requests.get("http://localhost:8080/search",
                 params={"q": "feature flag rollout"})
for hit in r.json()["results"]:
    print(hit["file_path"], hit["relevance_score"])
```

### MCP (any agent that speaks the protocol)

```typescript
const res = await fetch("http://localhost:8081/mcp/tools/call", {
  method: "POST",
  headers: { "Content-Type": "application/json" },
  body: JSON.stringify({
    tool: "search",
    parameters: { query: "rate limiting", max_results: 5 }
  }),
});
```

### CLI

```bash
stratafs search "deployment strategy" --mode hybrid --limit 5 --json
```

### As a Go library

```go
import "github.com/neul-labs/stratafs/pkg/search"

eng, _ := search.NewEngine(cfg)
results, _ := eng.Hybrid(ctx, "circuit breaker pattern", search.Opts{Limit: 10})
```

---

## Performance

Measured on consumer hardware (M-series Mac, NVMe SSD, BGE Base EN v1.5).

| Metric | Typical value |
| --- | --- |
| Indexing throughput | 50 – 100 files/sec |
| Search latency (10 k files) | < 100 ms |
| Disk overhead | ~1.5–2× original text (with compression) |
| Memory baseline | ~200 MB + model (~500 MB for BGE Base) |
| Cold start | < 1 s |

Performance tuning, model swaps, and benchmark methodology: [Performance guide](documentation/docs/architecture/performance.md).

---

## Extend it

Every moving part is a registry plus an interface. Adding things is intentionally boring.

<details>
<summary><strong>Add a new file parser</strong></summary>

```go
// pkg/parsers/asciidoc.go
type AsciidocParser struct{}

func (p *AsciidocParser) Parse(r io.Reader) (string, error) { /* ... */ }
func (p *AsciidocParser) SupportedExtensions() []string {
    return []string{".adoc", ".asciidoc"}
}

func init() { DefaultRegistry.Register(NewAsciidocParserFactory()) }
```

</details>

<details>
<summary><strong>Add a new storage backend</strong></summary>

```go
// pkg/filesystem/dropbox.go
type DropboxFS struct{ /* ... */ }

func (fs *DropboxFS) Open(path string) (io.ReadCloser, error) { /* ... */ }
func (fs *DropboxFS) Walk(root string, fn WalkFunc) error    { /* ... */ }

// pkg/storage/factory.go
case config.StorageTypeDropbox:
    return f.createDropboxFS(source)
```

</details>

<details>
<summary><strong>Add a new chunking strategy</strong></summary>

```go
// pkg/chunking/ast.go
type ASTChunker struct{}
func (c *ASTChunker) Name() string { return "ast" }
func (c *ASTChunker) ChunkStream(r io.Reader, o ChunkOptions) (<-chan Chunk, <-chan error) {
    // Yield one chunk per top-level AST node.
}
```

</details>

<details>
<summary><strong>Swap the embedding model</strong></summary>

```json
{
  "embedding": {
    "model": "bge-small-en-v1.5",
    "dimension": 384
  }
}
```

Any ONNX-compatible model works. Drop the weights in `~/.stratafs/fastembed_cache/` and point `embedding.model` at it.

</details>

<details>
<summary><strong>Add a new ranking signal</strong></summary>

Hybrid scoring is a single SQL query with weighted CTEs. Add a CTE, expose a weight, ship a PR. Full walkthrough in the [development guide](documentation/docs/contributing/development.md#adding-things).

</details>

---

## What's next on the roadmap

- **Enterprise security** — RBAC for authentication and source-level permissions
- **Streaming search results** — chunked HTTP for very large result sets
- **Custom embeddings** — first-class support for any ONNX-compatible model on disk
- **Cross-source ranking signals** — per-source weight, recency boost, trusted-source pinning
- **Encrypted source databases** — SQLCipher-backed at-rest encryption

Already shipped: virtual FS export, FUSE/WinFsp mount, GNOME / Spotlight / Windows Search integration, Wails desktop UI, native installers for every desktop OS, enterprise connectors for SharePoint / Google Drive / Jira.

Full list: [Roadmap](documentation/docs/contributing/roadmap.md).

---

## Documentation

The full docs live in [`documentation/`](documentation/) and are built with MkDocs Material.

| Topic | Where |
| --- | --- |
| Getting started | [`documentation/docs/getting-started/`](documentation/docs/getting-started/quickstart.md) |
| User guide (config, search, CLI, file types) | [`documentation/docs/user-guide/`](documentation/docs/user-guide/configuration.md) |
| REST + MCP integration | [`documentation/docs/ai-integration/`](documentation/docs/ai-integration/mcp.md) |
| Storage backends | [`documentation/docs/user-guide/storage-backends.md`](documentation/docs/user-guide/storage-backends.md) |
| Deployment (Docker / systemd / launchd / K8s) | [`documentation/docs/deployment/`](documentation/docs/deployment/docker.md) |
| Architecture | [`documentation/docs/architecture/`](documentation/docs/architecture/overview.md) |
| Contributing & dev setup | [`documentation/docs/contributing/`](documentation/docs/contributing/development.md) |

Preview the docs locally:

```bash
cd documentation
pip install -r requirements.txt
mkdocs serve
```

---

## Community & contributing

- **Issues** — [github.com/neul-labs/stratafs/issues](https://github.com/neul-labs/stratafs/issues)
- **Discussions** — [github.com/neul-labs/stratafs/discussions](https://github.com/neul-labs/stratafs/discussions)
- **Contributing guide** — [documentation/docs/contributing/development.md](documentation/docs/contributing/development.md)

Pull requests welcome. For larger changes, open an issue first to align on the approach. Every PR runs the full test suite plus a Docker build in CI.

---

## License

[MIT](LICENSE). Do whatever you want with it. If StrataFS ends up powering something interesting, [we'd love to hear about it](https://github.com/neul-labs/stratafs/discussions).
