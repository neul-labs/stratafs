# AgentFS: The Agentic Filesystem

**"Where traditional filesystems end, agentic intelligence begins"**

AgentFS transforms passive file storage into an active, searchable, and semantically-aware knowledge base that AI agents can reason about and interact with naturally.

## Why AgentFS?

As we enter the age of agentic AI systems, traditional filesystems are no longer sufficient. They lack the semantic understanding and intelligent primitives that AI agents need to truly understand and interact with our digital knowledge.

### Traditional Filesystems Fall Short

- **Lack Semantic Understanding**: Files organized by names and paths, not meaning
- **No Built-in Search**: External tools with outdated indexes
- **Miss Implicit Relationships**: No discovery of connections between related files
- **Provide No Agentic Primitives**: No way for AI agents to ask questions or discover insights

### AgentFS Bridges the Gap

AgentFS introduces agentic primitives directly into the filesystem layer, making your filesystem a collaborative intelligence partner rather than just a data store.

## Key Features

🏗️ **Multi-Storage Architecture**
- Local directories with real-time monitoring
- Cloud storage (S3, GCS, Azure) with intelligent sync
- Read-only design preserves source integrity

🔍 **Semantic Intelligence**
- Streaming text chunking with multiple strategies (simple, separator, sentence, token)
- Automatic file-type optimization and vector embedding
- Hybrid search combining full-text and semantic similarity
- Cross-file relationship discovery

🤖 **AI Agent Integration**
- Model Context Protocol (MCP) server for direct agent access
- REST API for custom integrations
- Natural language query processing

⚡ **Performance & Scale**
- Memory-efficient streaming processing for large files
- Intelligent caching and compression (40-60% space savings)
- Soft delete strategy for consistent file updates
- Concurrent processing with configurable workers
- Automatic maintenance and optimization

## Use Cases

### 📚 Documentation & Knowledge Management
Transform scattered documentation into a queryable knowledge base:
- **Research Teams**: Index papers, notes, and references across multiple storage locations
- **Engineering Teams**: Search code, docs, and specifications with natural language
- **Content Teams**: Find related articles, drafts, and assets across cloud storage

### 🤖 AI Agent Development
Provide AI agents with semantic file system access:
- **Code Assistants**: Understand entire codebases for better suggestions
- **Documentation Bots**: Answer questions using your organization's knowledge
- **Research Agents**: Analyze and synthesize information from document collections

### 📂 Personal Knowledge Systems
Build your second brain with intelligent file organization:
- **Note Taking**: Connect related notes and documents automatically
- **Research**: Query your reading materials and saved articles
- **Projects**: Find relevant files across different storage services

### 🔧 Development & DevOps
Enhance development workflows with semantic search:
- **Configuration Management**: Find related configs across repositories
- **Log Analysis**: Search and correlate logs with contextual understanding
- **API Documentation**: Query API docs and implementation examples

## Quick Start

### 1. Installation
```bash
# Clone and build
git clone https://github.com/yourusername/agentfs.git
cd agentfs
go build -o build/agentfs -tags "fts5" ./cmd/agentfs

# Initialize configuration
./build/agentfs config init
```

### 2. Configure Storage Sources
```bash
# Add local directory
./build/agentfs source add

# Or edit config directly
vim ~/.agentfs/config.json
```

### 3. Start AgentFS
```bash
./build/agentfs
```

### 4. Search Your Content
```bash
# REST API
curl "http://localhost:8080/search?q=machine learning"

# MCP for AI agents
curl "http://localhost:8081/mcp/search?q=API documentation"
```

## Building & Releases

### Local builds (with ONNX Runtime)
AgentFS needs ONNX Runtime for FastEmbed. Run:
```bash
make fetch-onnx   # downloads the correct runtime for your host OS/arch
make build        # produces build/agentfs with FTS5 + FastEmbed support
```
The downloader honors `ONNX_VERSION`, `TARGET_OS`, and `TARGET_ARCH` if you need a different runtime.

### Cross-platform release artifacts
To build self-contained archives (Linux/macOS/Windows, amd64/arm64) with the matching ONNX Runtime libraries bundled alongside the binary:
```bash
make release
```
Artifacts are placed in `build/release/agentfs-<version>-<os>-<arch>.tar.gz` (or `.zip` on Windows) and include:
- `agentfs` (or `agentfs.exe`)
- `lib/` with the ONNX Runtime shared libraries for that platform (Windows gets `.dll`s next to the executable)
- A README with install instructions

The release builder also accepts `VERSION` and `ONNX_VERSION` overrides, e.g. `VERSION=0.3.0 ONNX_VERSION=1.17.0 make release`.

### Running tests with ONNX Runtime
To exercise the full suite—including the FastEmbed/ONNX-dependent search tests—run:
```bash
make test-onnx        # downloads ONNX Runtime if missing and runs go test -tags fts5 ./...
```
Use `TARGET_OS`, `TARGET_ARCH`, or `ONNX_VERSION` if you need to force a specific runtime build.

### Filesystem export (metadata view)
To build a filesystem-friendly mirror that exposes AgentFS metadata alongside the original tree:
```bash
./build/agentfs fs export --source-id default-local --output /tmp/agentfs-virtual
```
The export includes each file's metadata (`metadata.json`) and chunk contents under `_chunks/`, making it easy to inspect semantic context with standard OS tools while the REST API and MCP interface keep running.

### Desktop UI & packaging (Roadmap)
We are actively adding a native desktop experience using [Wails](https://wails.io/) while keeping the REST API/MCP server unchanged:
- **Linux (in progress)**: ship an AppImage/Deb package containing the AgentFS daemon and Wails-based control panel (start/stop service, monitor queue, run exports). A systemd user service keeps the daemon running.
- **macOS**: deliver a signed `.app` + LaunchAgent that runs the daemon in the background with a menu bar UI built in Wails.
- **Windows**: provide an installer that sets up a Windows Service for the daemon and bundles a tray application (Wails) for controls/status.

Follow `docs/roadmap.md` for updated timelines and details.

## Example Queries

### Natural Language Search
- *"Find documents about API authentication"*
- *"Show me configuration files for microservices"*
- *"What papers discuss neural network architectures?"*

### Cross-Source Discovery
- Search simultaneously across local files, S3 buckets, and Google Drive
- Find related content regardless of storage location
- Discover connections between documents in different formats

### AI Agent Integration
```python
# Using MCP protocol
agent.search("deployment strategies for kubernetes")
agent.retrieve("docs/api/authentication.md")
agent.stats("show indexing progress")
```

## Architecture

AgentFS implements a modular, multi-storage architecture:

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
    │ Database  │ │ Embedder  │ │Job Queue  │
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

## Repository Structure

```
agentfs/
├── cmd/agentfs/        # CLI entry point that wires all services together
├── internal/utils/     # Private helpers not intended for reuse
├── pkg/                # Reusable domain packages
│   ├── api/            # REST server
│   ├── chunking/       # Chunking strategies and façade
│   ├── config/         # Config loading/defaults
│   ├── database/       # SQLite schema + compression/maintenance
│   ├── embeddings/     # FastEmbed wrapper
│   ├── filesystem/     # Filesystem abstraction and implementations
│   ├── monitor/        # File watcher + remote scanner
│   ├── parsers/        # Parser registry and implementations
│   ├── queue/          # Persistent job queue and processor
│   ├── search/         # Hybrid (FTS + vector) engine
│   └── storage/        # Storage factory (local/S3/GCS/Azure)
├── docs/               # Guides, API/architecture references
├── installers/, docker/, helm/, scripts/  # Deployment tooling
└── models/, build/, etc.                 # Pre-bundled assets (optional)
```

Packages with heavy native dependencies (notably `pkg/search`) automatically skip their tests when ONNX Runtime or SQLite FTS5 support is unavailable, allowing the rest of the suite to run anywhere.

## Documentation

- **[Configuration Guide](docs/configuration.md)** - Setup and configuration options
- **[Storage Backends](docs/storage-backends.md)** - Local and cloud storage setup
- **[API Reference](docs/api.md)** - REST API and MCP server documentation
- **[Development Guide](docs/development.md)** - Contributing and development setup
- **[Architecture Overview](docs/architecture.md)** - Technical architecture details
- **[Roadmap](docs/roadmap.md)** - Short-term and long-term plans

## Prerequisites

- **Go 1.21+** for building from source
- **ONNX Runtime** for embeddings (auto-downloaded on first run)
- **SQLite with FTS5** for full-text search (usually pre-installed)

## Supported File Types

### Documents
- **Text**: Markdown, plain text, reStructuredText
- **Office**: PDF, DOCX, PPTX, RTF
- **Spreadsheets**: XLSX, XLS, ODS, CSV, TSV

### Code & Markup
- **Code**: Go, Python, JavaScript, TypeScript, Java, C++, and more
- **Markup**: HTML, XML, JSON, YAML
- **Config**: INI, TOML, environment files

### Extensible Parser System
Add support for new file types through the modular parser registry.

## Storage Backends

- **Local Filesystem**: Real-time monitoring with immediate indexing
- **Amazon S3**: S3-compatible object storage with configurable endpoints
- **Google Cloud Storage**: Native GCS integration with service accounts
- **Azure Blob Storage**: Azure containers with account key authentication

All backends follow a **read-only architecture** - AgentFS never modifies source files.

## Performance

### Optimization Features
- **Text Compression**: 40-60% space savings with automatic gzip compression
- **Intelligent Caching**: Remote files cached locally during processing
- **Concurrent Processing**: Configurable worker pools for parallel processing
- **Smart Indexing**: Only process changed files, skip duplicates

### Scalability
- **Per-Source Databases**: Isolated storage for each source
- **Batch Processing**: Efficient embedding generation
- **Automatic Maintenance**: Background cleanup and optimization

## License

MIT License - see [LICENSE](LICENSE) for details.

## Contributing

We welcome contributions! See the [Development Guide](docs/development.md) for setup instructions and contribution guidelines.

## Community

- **Issues**: [GitHub Issues](https://github.com/yourusername/agentfs/issues)
- **Discussions**: [GitHub Discussions](https://github.com/yourusername/agentfs/discussions)
- **Documentation**: [docs/](docs/)

---

**AgentFS**: Making your filesystem intelligent, searchable, and agent-ready.
