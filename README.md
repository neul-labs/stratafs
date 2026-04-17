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

- **Real-time File Monitoring**: Monitors directories for file changes with automatic cleanup
- **Intelligent Storage**: Automatically creates and maintains `.agentfs` directories with SQLite database
- **Hybrid Search**: Combines full-text search (FTS5) with vector similarity search
- **Smart Compression**: Automatic gzip compression for text chunks (40-60% space savings)
- **Database Maintenance**: VACUUM operations, cleanup, and index optimization
- **Cross-Platform Embeddings**: FastEmbed-go with ONNX Runtime for configurable embedding models
- **Modular Parser System**: Extensible architecture supporting multiple file formats
- **Document Support**: PDF, DOCX, PPTX, RTF parsing with content extraction
- **Spreadsheet Support**: XLSX, XLS, ODS, CSV, TSV with intelligent cell processing
- **Code Intelligence**: Advanced code parsing with syntax-aware chunking
- **REST API**: Full search capabilities via HTTP endpoints
- **MCP Integration**: Model Context Protocol server for AI agent integration
- **Filesystem Abstraction**: Ready for local and object store support

## Architecture

```
agentfs/
├── cmd/agentfs/          # Main application entry point
├── pkg/
│   ├── config/           # Configuration management (FastEmbed model selection)
│   ├── monitor/          # File system monitoring with lifecycle management
│   ├── database/         # SQLite database with compression & maintenance
│   ├── embeddings/       # FastEmbed-go with ONNX Runtime
│   ├── search/           # Hybrid search engine (FTS5 + vector)
│   ├── queue/            # File processing queue system
│   ├── api/              # REST API server
│   ├── protocol/         # Model Context Protocol server
│   ├── parsers/          # Modular parser system
│   │   ├── registry.go   # Parser registration and factory
│   │   ├── documents.go  # DOCX, PPTX, RTF parsers
│   │   ├── spreadsheets.go # XLSX, XLS, ODS, CSV parsers
│   │   └── pdf.go        # PDF content extraction
│   └── filesystem/       # Filesystem abstraction
└── internal/
    ├── utils/            # Utility functions
    └── models/           # Data models
```

## Getting Started

### Prerequisites

1. **Go 1.21 or later**
2. **ONNX Runtime** (for FastEmbed embeddings):
   - Download from [ONNX Runtime releases](https://github.com/microsoft/onnxruntime/releases)
   - Extract and set `ONNX_PATH` environment variable to the lib directory
   - Or let AgentFS download it automatically on first run

### Installation

1. Clone this repository
2. Run `go mod tidy` to download dependencies
3. Build the application: `make build` (or `go build -o build/agentfs -tags "fts5" cmd/agentfs/main.go`)
4. Run the application: `./build/agentfs`

## Configuration

AgentFS can be configured using environment variables:

- `AGENTFS_DIRS`: Comma-separated list of directories to monitor (default: current directory)
- `AGENTFS_FASTEMBED_MODEL`: Embedding model to use (default: "bge-base-en-v1.5")
  - Available models: `bge-base-en`, `bge-base-en-v1.5`, `bge-small-en`, `bge-small-en-v1.5`
  - Different models have different vector dimensions (384 or 768)
- `ONNX_PATH`: Path to ONNX Runtime library directory (auto-detected if not set)

### Embedding Models

| Model | Dimensions | Description |
|-------|------------|-------------|
| `bge-base-en` | 768 | BAAI General Embedding, good balance of speed and quality |
| `bge-base-en-v1.5` | 768 | Improved version of BGE-base-en (default) |
| `bge-small-en` | 384 | Faster, smaller model with reduced accuracy |
| `bge-small-en-v1.5` | 384 | Improved version of BGE-small-en |

## Embedding Support

AgentFS uses **FastEmbed-go** with ONNX Runtime for cross-platform embedding generation. This provides:

- **High-quality embeddings**: State-of-the-art BGE (BAAI General Embedding) models
- **Cross-platform compatibility**: Works on Linux, macOS, and Windows
- **Configurable models**: Choose between speed and accuracy based on your needs
- **Automatic model download**: Models are downloaded and cached on first use
- **Dynamic dimensions**: Vector index automatically adapts to the selected model

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

### REST API (Port 8080)

AgentFS exposes a comprehensive REST API for search and management:

- `GET /health`: Health check endpoint
- `GET /search?q={query}`: Hybrid search across all monitored directories
  - Combines full-text search with vector similarity search
  - Returns ranked results with relevance scores
  - Supports natural language queries

### Model Context Protocol (Port 8081)

Built-in MCP server for seamless AI agent integration:

- `GET /mcp`: MCP protocol information and capabilities
- `GET /mcp/search?q={query}`: AI-optimized search endpoint
- `GET /mcp/resources`: List available resources and capabilities
- `POST /mcp/tools/call`: Execute MCP tools for advanced operations

### Database Management

AgentFS provides APIs for database maintenance and monitoring:

```go
// Get database statistics
stats, err := db.GetDatabaseStats()
fmt.Printf("Total size: %d bytes, %d files indexed\n", stats.TotalSize, stats.FileCount)

// Run maintenance operations
maintenanceStats, err := db.PerformMaintenance(database.DefaultMaintenanceOptions())
fmt.Printf("Space saved: %d bytes\n", maintenanceStats.SpaceSaved())
```

## Filesystem Abstraction

AgentFS includes a filesystem abstraction that allows it to work with both local files and object stores. Currently, only local filesystem support is implemented, but the architecture is designed to easily add support for:

- Amazon S3
- Google Cloud Storage
- Azure Blob Storage
- Other object stores

## File Format Support

AgentFS includes a **modular parser system** that supports a wide range of file formats:

### Document Formats
- **PDF**: Full text extraction with metadata preservation
- **DOCX**: Microsoft Word documents with rich text and formatting
- **PPTX**: PowerPoint presentations with slide content
- **RTF**: Rich Text Format documents

### Spreadsheet Formats
- **XLSX**: Modern Excel workbooks with multi-sheet support
- **XLS**: Legacy Excel workbooks (binary format)
- **ODS**: OpenDocument spreadsheets
- **CSV/TSV**: Comma and tab-separated values with smart header detection

### Code & Text Formats
- **Code files**: Syntax-aware parsing for `.go`, `.py`, `.js`, `.ts`, etc.
- **Text files**: Markdown, plain text, reStructuredText, etc.
- **Markup files**: HTML, XML, JSON with structure-aware parsing

### Unsupported Formats
- **DOC**: Legacy Word documents (complex proprietary format)
  - *Recommendation*: Convert to DOCX format for better support

### Adding New Parsers

The modular parser system makes it easy to add support for new file types:

```go
// Register a new parser for .foo files
parsers.DefaultRegistry.RegisterParser(".foo", NewFooParserFactory())
```

## Storage Optimization

AgentFS includes intelligent storage optimization to minimize disk usage:

### Text Compression
- **Automatic compression**: Text chunks larger than 512 bytes are compressed using gzip
- **Smart threshold**: Only compresses if it achieves at least 10% size reduction
- **Transparent operation**: Decompression happens automatically during reads
- **Space savings**: 40-60% reduction for text-heavy content
- **Backward compatibility**: Seamlessly works with existing uncompressed data

### Database Maintenance
AgentFS provides comprehensive database maintenance operations:

```go
// Example: Run maintenance with custom options
opts := database.DefaultMaintenanceOptions()
opts.DeletedThreshold = 7 * 24 * time.Hour  // Clean up week-old deletions
stats, err := db.PerformMaintenance(opts)
fmt.Printf("Maintenance complete: saved %d bytes\n", stats.SpaceSaved())
```

**Maintenance Operations:**
- **VACUUM**: Reclaims space from deleted records (5-10% savings)
- **Cleanup**: Removes soft-deleted records older than threshold
- **Reindexing**: Rebuilds indexes for optimal performance
- **Statistics**: Detailed database size and efficiency metrics

### Storage Impact
For a typical large project (10,000 files, 500MB text):
- **Original size**: ~1.2GB in database (includes embeddings)
- **With compression**: ~720MB (40% reduction)
- **After maintenance**: ~650MB (additional 10% savings)

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

AgentFS includes comprehensive test coverage across all modules:

```bash
# Run all tests
make test

# Run tests with coverage
go test -cover ./...

# Run specific module tests
go test ./pkg/embeddings/
go test ./pkg/database/
go test ./pkg/search/
go test ./pkg/queue/
go test ./pkg/parsers/

# Format code
make fmt

# Run vet
make vet
```

**Test Coverage Includes:**
- **Embeddings**: FastEmbed integration, model configuration, dimension handling
- **Database**: CRUD operations, compression, maintenance, migrations
- **Search**: Hybrid search, FTS5 integration, vector similarity
- **Queue**: File processing, concurrent operations, error handling
- **Parsers**: Document parsing, format detection, content extraction
- **Integration**: End-to-end file monitoring and search workflows

### Benchmarking

Performance benchmarks are included for critical operations:

```bash
# Run benchmarks
go test -bench=. ./pkg/embeddings/
go test -bench=. ./pkg/database/
go test -bench=. ./pkg/search/
```

## License

MIT
