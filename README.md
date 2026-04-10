# AgentFS: The Agentic Filesystem

## The Evolution of Filesystems for the AI Era

"Where traditional filesystems end, agentic intelligence begins"

As we enter the age of agentic AI systems, traditional filesystems are no longer sufficient. While they've served us well for organizing and storing data in hierarchical structures, they lack the semantic understanding and intelligent primitives that AI agents need to truly understand and interact with our digital knowledge.

AgentFS represents the next evolution in filesystem design - one that bridges the gap between raw data storage and intelligent information retrieval. It transforms passive file storage into an active, searchable, and semantically-aware knowledge base that AI agents can reason about and interact with naturally.

### Why Traditional Filesystems Fall Short

Traditional filesystems are fundamentally limited because they:

1. **Lack Semantic Understanding**: Files are organized by names and paths, not by meaning or context
2. **Have No Built-in Search**: Searching requires external tools and indexes that are often outdated
3. **Miss Implicit Relationships**: Connections between related files across directories are not discoverable
4. **Provide No Agentic Primitives**: There's no built-in way for AI agents to ask questions or discover insights

### AgentFS: Bridging the Gap

AgentFS introduces agentic primitives directly into the filesystem layer:

- **Semantic Indexing**: Every file is automatically parsed, chunked, and indexed with vector embeddings
- **Intelligent Search**: Natural language queries that understand context and intent
- **Cross-File Relationships**: Automatic discovery of connections between related content
- **Agentic APIs**: Model Context Protocol (MCP) server for direct AI agent integration
- **File-Type Intelligence**: Specialized parsers that understand the structure and meaning of different file types

With AgentFS, your filesystem becomes a collaborative intelligence partner rather than just a data store.

## Features

- Monitors directories for file changes
- Automatically creates and maintains `.agentfs` directories
- Stores file chunks with embeddings in SQLite database
- Provides full-text search and vector search capabilities
- Soft-delete mechanism with periodic compaction
- REST API for searching content
- Model Context Protocol (MCP) server for AI integration
- File-type specific parsers for better content extraction
- Filesystem abstraction for local and object store support

## Architecture

```
agentfs/
├── cmd/agentfs/          # Main application entry point
├── pkg/
│   ├── config/           # Configuration management
│   ├── monitor/          # File system monitoring
│   ├── database/         # SQLite database operations
│   ├── embeddings/       # Text embedding generation
│   ├── api/              # REST API server
│   ├── protocol/         # Model Context Protocol server
│   ├── parsers/          # File-type specific parsers
│   └── filesystem/       # Filesystem abstraction
└── internal/
    ├── utils/            # Utility functions
    └── models/           # Data models
```

## Getting Started

1. Install Go 1.21 or later
2. Clone this repository
3. Run `go mod tidy` to download dependencies
4. Build the application: `make build` (or `go build -o build/agentfs -tags "fts5" cmd/agentfs/main.go`)
5. Run the application: `./build/agentfs`

## Configuration

AgentFS can be configured using environment variables:

- `AGENTFS_DIRS`: Comma-separated list of directories to monitor (default: current directory)

## Embedding Support

AgentFS uses the kelindar/search library for text embeddings, which provides GGUF BERT models without requiring external dependencies like ONNX Runtime. This ensures AgentFS is a fully self-contained binary that works out of the box.

## Full-Text Search

AgentFS supports full-text search using SQLite's FTS5 extension. To enable FTS5 support, build the application with the `fts5` build tag:

```bash
# Using make
make build

# Or directly
go build -o build/agentfs -tags "fts5" cmd/agentfs/main.go
```

If FTS5 is not available, the application will fall back to simple LIKE-based search.

## API

AgentFS exposes a REST API on port 8080:

- `GET /health`: Health check endpoint
- `GET /search?q={query}`: Search for content across all monitored directories

AgentFS also exposes a Model Context Protocol server on port 8081:

- `GET /mcp`: MCP protocol information
- `GET /mcp/search?q={query}`: Search for content (for LLM integration)
- `GET /mcp/resources`: List available resources

## Filesystem Abstraction

AgentFS includes a filesystem abstraction that allows it to work with both local files and object stores. Currently, only local filesystem support is implemented, but the architecture is designed to easily add support for:

- Amazon S3
- Google Cloud Storage
- Azure Blob Storage
- Other object stores

## File-Type Specific Parsers

AgentFS includes specialized parsers for different file types:

- Text files (`.txt`, `.md`, `.rst`, etc.)
- Code files (`.go`, `.py`, `.js`, `.ts`, etc.) with advanced parsing that extracts comments and documentation
- Markup files (`.html`, `.xml`, `.json`, etc.)

## Development

### Building

```bash
# Build with FTS5 support
make build

# Run directly
make run

# Clean build artifacts
make clean
```

### Testing

```bash
# Run tests
make test

# Format code
make fmt

# Run vet
make vet
```

## License

MIT
