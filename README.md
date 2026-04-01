# AgentFS

AgentFS is a Go application that monitors directories for file changes and maintains an updated `.agentfs` directory in all directories and subdirectories. Each `.agentfs` directory contains:

1. A SQLite database with file chunks and full-text search capabilities
2. A usearch vector search index for semantic search

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

AgentFS uses the fastembed-go library for text embeddings. This library requires the ONNX Runtime library to be installed on your system.

If the ONNX Runtime is not available, AgentFS will fall back to a mock implementation that generates deterministic embeddings based on text content. This allows the application to function for testing and development.

For full functionality, please install the ONNX Runtime by following the instructions in [ONNX_INSTALL.md](ONNX_INSTALL.md).

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