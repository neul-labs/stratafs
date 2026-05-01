# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Essential Commands

### Build and Run
```bash
# Build with required FTS5 support
make build
# OR: go build -o build/agentfs -tags "fts5" cmd/agentfs/main.go

# Run during development
make run
# OR: go run -tags "fts5" cmd/agentfs/main.go

# Install globally
make install
```

### Testing
```bash
# Run all tests with FTS5 support (REQUIRED)
make test
# OR: go test -tags "fts5" -v ./...

# Run specific package tests
go test -tags "fts5" -v ./pkg/database/...

# Run single test
go test -tags "fts5" -v ./pkg/search/... -run TestNewEngine

# For search tests requiring ONNX Runtime
LD_LIBRARY_PATH=~/local/lib:$LD_LIBRARY_PATH go test -tags "fts5" -v ./pkg/search/...
```

### Code Quality
```bash
make fmt     # Format code
make vet     # Static analysis
make deps    # Manage dependencies
```

## Architecture Overview

AgentFS is a multi-storage semantic filesystem that transforms passive file storage into an intelligent, searchable knowledge base. The architecture follows a layered approach with clear separation between storage, processing, search, and API layers.

### Core Flow
1. **Storage Sources** → **Monitor** → **Queue** → **Parse/Chunk** → **Embed** → **Database** → **Search Engine** → **APIs**

### Key Components

**Storage Layer (`pkg/storage/`)**
- Multi-backend factory pattern supporting Local, S3, GCS, Azure
- Read-only architecture preserves source file integrity
- Each storage source gets isolated database and processing

**Processing Pipeline**
- **Monitor** (`pkg/monitor/`): Watches filesystem changes, triggers processing
- **Queue** (`pkg/queue/`): SQLite-based job queue with retry logic, priority handling
- **Parsers** (`pkg/parsers/`): Registry pattern supporting 15+ file types
- **Chunking** (`pkg/chunking/`): Multiple strategies (sentence, token, separator)

**Intelligence Layer**
- **Embeddings** (`pkg/embeddings/`): FastEmbed integration with ONNX Runtime
- **Database** (`pkg/database/`): SQLite with FTS5 + vector extensions, compression support
- **Search** (`pkg/search/`): Hybrid search combining full-text and semantic similarity

**API Layer**
- **REST API** (`pkg/api/`): Standard HTTP endpoints for search/retrieval
- **MCP Server** (`pkg/protocol/`): Model Context Protocol for AI agent integration

### Configuration Structure

The config system uses nested structures matching the modular architecture:

```go
type Config struct {
    Sources []StorageSource      // Multiple storage backends
    Embedding EmbeddingConfig    // FastEmbed model configuration
    Worker WorkerConfig         // Concurrent processing settings
    API APIConfig              // REST API configuration
    MCP MCPConfig              // Model Context Protocol settings
}
```

### Database Design

**Per-Source Isolation**: Each storage source gets its own SQLite database for scalability and isolation.

**Compression-Aware Schema**: The `file_chunks` table supports both raw content and compressed storage:
- `content`: Raw text content
- `content_compressed`: Gzip-compressed content (40-60% space savings)
- `is_compressed`: Flag indicating compression status

**Soft Delete Strategy**: Files are marked as deleted rather than removed, enabling consistent updates without breaking references.

## Development Patterns

### Config Field Evolution
Tests reference nested config structures. When adding config fields, update ALL test files:
- `cfg.Embedding.Model` not `cfg.FastEmbedModel`
- `cfg.Worker.ScanInterval` not `cfg.ScanInterval`
- `cfg.Sources[0].Path` not `cfg.Directories[0]`

### Database Error Handling
Database methods return `nil` for not-found cases, not errors:
```go
// Correct pattern
func (db *DB) GetFileByPath(path string) (*File, error) {
    err := db.conn.QueryRow("SELECT ... WHERE path = ?", path).Scan(...)
    if err == sql.ErrNoRows {
        return nil, nil // Not found, not an error
    }
    return file, err
}
```

### Search Test Strategy
Search tests skip gracefully when models can't be downloaded:
```go
embedder, err := embeddings.NewEmbedder(cfg)
if err != nil {
    if strings.Contains(err.Error(), "403 Forbidden") || strings.Contains(err.Error(), "model download failed") {
        t.Skip("Skipping test - model download blocked, consider using local models for testing")
    }
    t.Fatalf("Failed to create test embedder: %v", err)
}
```

### Queue Job States and Retry Logic
The queue implements automatic retry logic:
- Jobs start with `max_retries = 3`
- `FailJob()` increments retry count and resets to "pending" if retries remain
- Only marks as "failed" when retries are exhausted
- Tests expecting immediate failure should set `max_retries = 0`

## Required Dependencies

### Build-time
- **Go 1.21+**
- **FTS5 tags**: All builds MUST use `-tags "fts5"` for SQLite full-text search

### Runtime
- **ONNX Runtime**: For embedding generation (installed in `~/local/lib/`)
- **SQLite with FTS5**: Usually pre-installed on modern systems

### Local Models
The repository contains `MiniLM-L6-v2.Q8_0.gguf` for potential local embedding use, though the current FastEmbed implementation downloads ONNX models from HuggingFace.

## Project Structure

```
cmd/agentfs/          # Main application entry point
pkg/
├── api/              # REST API server
├── chunking/         # Text chunking strategies
├── config/           # Configuration management
├── database/         # SQLite + vector database layer
├── embeddings/       # FastEmbed integration
├── filesystem/       # File system utilities
├── monitor/          # File change monitoring
├── parsers/          # File type parsing registry
├── protocol/         # MCP server implementation
├── queue/            # SQLite-based job queue
├── search/           # Hybrid search engine
└── storage/          # Multi-backend storage factory
```

## Testing Notes

- **Database tests**: Verify compression-aware chunk handling and proper soft-delete behavior
- **Queue tests**: May need `max_retries` adjustment for failure testing
- **Monitor tests**: Require proper directory setup for queue database creation
- **Search tests**: Skip gracefully when model download fails (403 Forbidden)
- **All tests**: Must use `-tags "fts5"` flag

## Version and Status

Current version: 0.2.0 (Active development)
All tests passing with graceful skipping for network-dependent features.