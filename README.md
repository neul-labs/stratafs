# AgentFS

[![npm](https://img.shields.io/npm/v/agentfs.svg)](https://www.npmjs.com/package/agentfs)
[![PyPI](https://img.shields.io/pypi/v/agentfs.svg)](https://pypi.org/project/agentfs/)
[![Go Version](https://img.shields.io/badge/go-1.24-blue.svg)](https://golang.org)
[![License](https://img.shields.io/badge/license-MIT-blue.svg)](LICENSE)
[![CI](https://github.com/dipankarsarkar/agentfs/actions/workflows/ci.yml/badge.svg)](https://github.com/dipankarsarkar/agentfs/actions/workflows/ci.yml)

> **A semantic filesystem that transforms passive file storage into an intelligent, searchable knowledge base.**

---

## Install in 30 seconds

**npm**
```bash
npm install -g agentfs
agentfs config init
agentfs serve
```

**PyPI**
```bash
pip install agentfs
agentfs config init
agentfs serve
```

**Homebrew / direct**
```bash
curl -fsSL https://raw.githubusercontent.com/dipankarsarkar/agentfs/main/scripts/install.sh | bash
agentfs config init
agentfs serve
```

Search your files immediately:
```bash
agentfs search "authentication middleware"
# or via REST API
curl "http://localhost:8080/search?q=machine+learning"
```

---

## For Developers

### What it does
AgentFS watches your directories (local or cloud), parses files into semantic chunks, generates vector embeddings, and exposes everything through a hybrid search engine (full-text + semantic similarity). You get a REST API and an MCP server that any AI agent can query.

### Why you care
- **No more `grep -r`** — ask natural language questions across your entire codebase
- **Cross-file context** — find related code, configs, and docs even when filenames don't match
- **AI-native** — Model Context Protocol server means your agent can query your filesystem directly
- **Zero lock-in** — read-only architecture, standard SQLite + FTS5, plain HTTP APIs

### Quick integration
```bash
# Start the daemon
agentfs serve

# Search from your scripts
curl "http://localhost:8080/search?q=deployment+strategies"

# Or use the MCP endpoint for agent integration
curl "http://localhost:8081/mcp/search?q=API+documentation"
```

### Build from source
```bash
git clone https://github.com/dipankarsarkar/agentfs.git
cd agentfs
make build          # requires Go 1.24 + CGO
make test           # runs the full suite with FTS5
```

See [docs/development.md](docs/development.md) for contributing guidelines.

---

## For Architects

### System design
AgentFS is a multi-storage semantic indexing layer that sits between your storage backends and AI consumers. It does not replace your filesystem — it augments it with a parallel semantic index.

```
┌─────────────┐  ┌─────────────┐  ┌─────────────┐
│  REST API   │  │ MCP Server  │  │ CLI Tools   │
└──────┬──────┘  └──────┬──────┘  └──────┬──────┘
       │                │                │
       └────────────────┼────────────────┘
                        │
              ┌─────────▼─────────┐
              │  Hybrid Search    │
              │ (FTS5 + Vector)   │
              └─────────┬─────────┘
                        │
          ┌─────────────┼─────────────┐
          │             │             │
    ┌─────▼─────┐ ┌─────▼─────┐ ┌─────▼─────┐
    │ Database  │ │ Embedder  │ │ Job Queue │
    │(SQLite +  │ │(FastEmbed │ │Processing │
    │ Vector)   │ │+ ONNX)    │ │           │
    └───────────┘ └───────────┘ └─────┬─────┘
                                      │
                            ┌─────────▼─────────┐
                            │     Monitor       │
                            │(Local + Remote)   │
                            └─────────┬─────────┘
                                      │
                            ┌─────────▼─────────┐
                            │ Storage Factory   │
                            │(Multi-Backend)    │
                            └─────────┬─────────┘
                                      │
            ┌─────────────────────────┼─────────────────────────┐
            │                         │                         │
    ┌───────▼───────┐       ┌─────────▼─────────┐       ┌───────▼───────┐
    │ Local Files   │       │  Cloud Storage    │       │ Future Stores │
    │(Real-time)    │       │ (S3,GCS,Azure)    │       │               │
    └───────────────┘       └───────────────────┘       └───────────────┘
```

### Key architectural decisions

**Per-source database isolation**
Each storage backend gets its own SQLite database (with FTS5 + `sqlite-vec`). This means:
- No central bottleneck — add sources without impacting existing ones
- Easy backup/restore per source
- Drop a source by deleting one file

**Read-only source architecture**
AgentFS never writes to your original files. It maintains a parallel `.agentfs/` metadata directory per source. This eliminates:
- Risk of corrupting user data
- Permission escalation requirements
- Surprise modifications in version-controlled directories

**Compression-aware schema**
The `file_chunks` table stores both raw and gzip-compressed content (above 512 bytes). Typical savings: 40-60% disk space with transparent decompression at query time.

**Soft-delete strategy**
Files are marked deleted rather than removed. This enables:
- Consistent updates without breaking references
- Historical queries ("what did this file say last week?")
- Safe concurrent scanning

**Hybrid search — unified SQL CTEs**
Rather than querying FTS5 and vector indexes separately, AgentFS uses a single SQL query with CTEs that combines:
- FTS5 BM25 ranking
- `sqlite-vec` cosine similarity
- Metadata scoring (recency, filename match, file type)
- Configurable per-query weights

### Deployment patterns

**Single-node desktop**
```bash
agentfs serve          # REST on :8080, MCP on :8081
```

**Docker (stateless front, persistent volume)**
```bash
docker run -p 8080:8080 -p 8081:8081 -v ./data:/app/.agentfs ghcr.io/dipankarsarkar/agentfs:latest
```

**Systemd service**
The release bundles include systemd user service files. The daemon starts on login, the Wails UI (Linux/macOS) or tray app (Windows) provides controls.

**FUSE export**
For systems that need a traditional filesystem view:
```bash
agentfs mount --mount-point /mnt/agentfs
```
Exposes the semantic index as a read-only filesystem with `metadata.json` per file.

### Performance characteristics

| Metric | Typical Value |
|--------|--------------|
| Indexing throughput | ~50-100 files/sec (depends on size + embedder) |
| Search latency | <100ms for 10k files on consumer hardware |
| Embedding model | BGE Base EN v1.5 (768-dim) or All-MiniLM-L6-v2 (384-dim) |
| Disk overhead | ~1.5-2x original text size (with compression) |
| Memory footprint | ~200MB base + embedding model cache |

### Security & hardening
- No long-lived tokens in CI — releases use OIDC trusted publishing (PyPI, npm)
- Multi-stage Docker build with non-root user
- Gosec + Trivy scanning in CI
- Read-only source access by design

---

## Feature Overview

**Multi-Storage**
- Local directories with real-time `fsnotify` monitoring
- S3, GCS, Azure Blob with intelligent sync
- Read-only design preserves source integrity

**Semantic Intelligence**
- Streaming text chunking (sentence, token, separator strategies)
- Automatic file-type optimization and vector embedding
- Hybrid search combining full-text and semantic similarity
- Cross-file relationship discovery

**AI Agent Integration**
- Model Context Protocol (MCP) server for direct agent access
- REST API for custom integrations
- Natural language query processing

**Performance**
- Memory-efficient streaming processing for large files
- Gzip compression with 40-60% space savings
- Concurrent processing with configurable worker pools
- Automatic maintenance and optimization

---

## Supported File Types

**Documents**
- Markdown, plain text, reStructuredText, PDF, DOCX, PPTX, RTF

**Spreadsheets**
- XLSX, XLS, ODS, CSV, TSV

**Code & Markup**
- Go, Python, JavaScript, TypeScript, Java, C++, and more
- HTML, XML, JSON, YAML, INI, TOML

Add new file types through the modular parser registry in `pkg/parsers/`.

---

## Documentation

- [Configuration Guide](docs/configuration.md) — Setup and config options
- [Storage Backends](docs/storage-backends.md) — Local and cloud setup
- [API Reference](docs/api.md) — REST API and MCP server docs
- [Development Guide](docs/development.md) — Contributing and dev setup
- [Architecture Overview](docs/architecture.md) — Deep-dive technical details
- [Roadmap](docs/roadmap.md) — Short-term and long-term plans

---

## License

MIT License — see [LICENSE](LICENSE) for details.
