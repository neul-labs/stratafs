# Development Guide

This guide covers development setup, testing, and contribution guidelines for AgentFS.

## Prerequisites

- **Go 1.21+** - [Download Go](https://golang.org/dl/)
- **ONNX Runtime** - [Download ONNX Runtime](https://github.com/microsoft/onnxruntime/releases)
- **SQLite** with FTS5 support (usually pre-installed on modern systems)
- **Git** for version control

## Development Setup

### 1. Clone Repository
```bash
git clone https://github.com/yourusername/agentfs.git
cd agentfs
```

### 2. Install Dependencies
```bash
go mod tidy
```

### 3. Set Up ONNX Runtime
```bash
# Option 1: Let AgentFS download automatically
export ONNX_PATH=""

# Option 2: Manual installation
wget https://github.com/microsoft/onnxruntime/releases/download/v1.16.3/onnxruntime-linux-x64-1.16.3.tgz
tar -xzf onnxruntime-linux-x64-1.16.3.tgz
export ONNX_PATH=$(pwd)/onnxruntime-linux-x64-1.16.3/lib
```

### 4. Build Application
```bash
# Using Makefile (recommended)
make build

# Or directly with Go
go build -o build/agentfs -tags "fts5" ./cmd/agentfs
```

### 5. Initialize Configuration
```bash
./build/agentfs config init
```

## Project Structure

```
agentfs/
├── cmd/agentfs/          # Main application entry point
├── pkg/                  # Public packages
│   ├── config/           # Configuration management
│   ├── monitor/          # File monitoring and remote scanning
│   ├── storage/          # Storage factory and backends
│   ├── filesystem/       # Filesystem abstraction
│   ├── database/         # SQLite database operations
│   ├── embeddings/       # FastEmbed integration
│   ├── search/           # Hybrid search engine
│   ├── queue/            # Job processing system
│   ├── api/              # REST API server
│   ├── protocol/         # Model Context Protocol
│   ├── parsers/          # File parsing system
│   └── chunking/         # Text chunking strategies
├── internal/             # Private packages
│   ├── utils/            # Utility functions
│   └── models/           # Data models
├── docs/                 # Documentation
├── examples/             # Example configurations
├── scripts/              # Build and utility scripts
└── tests/                # Integration tests
```

## Build System

### Makefile Targets
```bash
# Build application with FTS5 support
make build

# Run application directly
make run

# Run all tests
make test

# Run tests with coverage
make test-coverage

# Format code
make fmt

# Run go vet
make vet

# Clean build artifacts
make clean

# Build for multiple platforms
make build-all
```

### Build Tags
- `fts5` - Enable SQLite FTS5 full-text search
- `debug` - Enable debug logging and profiling

```bash
# Build with all features
go build -tags "fts5,debug" -o build/agentfs ./cmd/agentfs
```

## Testing

### Running Tests
```bash
# All tests
go test ./...

# Specific package
go test ./pkg/embeddings/

# With coverage
go test -cover ./...

# With race detection
go test -race ./...

# Verbose output
go test -v ./pkg/search/
```

### Test Categories

#### Unit Tests
```bash
# Database operations
go test ./pkg/database/ -v

# Search functionality
go test ./pkg/search/ -v

# File parsing
go test ./pkg/parsers/ -v

# Chunking strategies
go test ./pkg/chunking/ -v
```

#### Integration Tests
```bash
# End-to-end workflow
go test ./tests/integration/ -v

# Storage backends
go test ./pkg/storage/ -v

# API endpoints
go test ./pkg/api/ -v
```

#### Benchmark Tests
```bash
# Embedding performance
go test -bench=. ./pkg/embeddings/

# Database operations
go test -bench=. ./pkg/database/

# Search performance
go test -bench=. ./pkg/search/
```

### Writing Tests

#### Unit Test Example
```go
func TestFileParser(t *testing.T) {
    parser := parsers.NewTextParser()

    content := "This is test content"
    reader := strings.NewReader(content)

    result, err := parser.Parse(reader)
    assert.NoError(t, err)
    assert.Equal(t, content, result)
}
```

#### Chunking Strategy Test Example
```go
func TestNewStrategyChunker(t *testing.T) {
    chunker := &NewStrategyChunker{}

    options := ChunkOptions{
        ChunkSize:   100,
        OverlapSize: 20,
        Strategy:    "newstrategy",
    }

    // Test streaming
    text := "This is a long text that should be chunked..."
    reader := strings.NewReader(text)
    chunkCh, errCh := chunker.ChunkStream(reader, options)

    var chunks []Chunk
    for chunk := range chunkCh {
        chunks = append(chunks, chunk)
    }

    // Check for errors
    select {
    case err := <-errCh:
        require.NoError(t, err)
    default:
    }

    // Verify chunks
    assert.Greater(t, len(chunks), 0)
    assert.LessOrEqual(t, len(chunks[0].Content), options.ChunkSize)
}
```

#### Integration Test Example
```go
func TestSearchWorkflow(t *testing.T) {
    // Setup test database
    db, cleanup := setupTestDB(t)
    defer cleanup()

    // Index test content
    err := indexTestFile(db, "test.txt", "test content")
    require.NoError(t, err)

    // Search and verify
    results, err := db.Search("test")
    require.NoError(t, err)
    assert.Len(t, results, 1)
}
```

## Code Style

### Go Conventions
- Follow [Effective Go](https://golang.org/doc/effective_go.html)
- Use `gofmt` for formatting
- Use `golint` for style checking
- Write meaningful comments for exported functions

### Project Conventions
- Package names: lowercase, single word
- Interface names: end with -er (e.g., `Parser`, `Scanner`)
- Error handling: always check errors, use descriptive messages
- Logging: use structured logging with levels

### Example Code Style
```go
// Package documentation
package embeddings

import (
    "context"
    "fmt"
    "log"
)

// Embedder generates vector embeddings from text content.
type Embedder struct {
    model  string
    client *fastEmbed.Client
}

// NewEmbedder creates a new embedder with the specified model.
func NewEmbedder(cfg *config.Config) (*Embedder, error) {
    if cfg.Embedding.Model == "" {
        return nil, fmt.Errorf("embedding model not specified")
    }

    // Implementation...
    return &Embedder{
        model: cfg.Embedding.Model,
    }, nil
}

// Embed generates embeddings for the given text.
func (e *Embedder) Embed(ctx context.Context, text string) ([]float32, error) {
    if text == "" {
        return nil, fmt.Errorf("text cannot be empty")
    }

    // Implementation...
    log.Printf("Generated embedding for %d characters", len(text))
    return embeddings, nil
}
```

## Adding New Features

### 1. File Parsers
To add support for new file types:

```go
// pkg/parsers/newformat.go
package parsers

type NewFormatParser struct{}

func (p *NewFormatParser) Parse(reader io.Reader) (string, error) {
    // Implementation
}

func (p *NewFormatParser) SupportedExtensions() []string {
    return []string{".new", ".format"}
}

// Register in pkg/parsers/registry.go
func init() {
    DefaultRegistry.RegisterParser(".new", NewNewFormatParserFactory())
}
```

### 2. Storage Backends
To add new storage backends:

```go
// pkg/filesystem/newstorage.go
package filesystem

type NewStorageFileSystem struct {
    // Configuration
}

func (fs *NewStorageFileSystem) Open(path string) (io.ReadCloser, error) {
    // Implementation
}

func (fs *NewStorageFileSystem) Walk(root string, walkFn WalkFunc) error {
    // Implementation
}

// Register in pkg/storage/factory.go
func (f *StorageFactory) CreateFileSystem(source config.StorageSource) (filesystem.FileSystem, error) {
    switch source.Type {
    case config.StorageTypeNewStorage:
        return f.createNewStorageFileSystem(source)
    }
}
```

### 3. Chunking Strategies
To add new text chunking strategies:

```go
// pkg/chunking/newstrategy.go
package chunking

import (
    "io"
)

type NewStrategyChunker struct{}

func (c *NewStrategyChunker) Name() string {
    return "newstrategy"
}

func (c *NewStrategyChunker) Description() string {
    return "Description of new chunking strategy"
}

func (c *NewStrategyChunker) DefaultOptions() ChunkOptions {
    return ChunkOptions{
        ChunkSize:   1000,
        OverlapSize: 100,
        Strategy:    "newstrategy",
    }
}

// Primary streaming method - implement this first
func (c *NewStrategyChunker) ChunkStream(reader io.Reader, options ChunkOptions) (<-chan Chunk, <-chan error) {
    chunkCh := make(chan Chunk, 10)
    errCh := make(chan error, 1)

    go func() {
        defer close(chunkCh)
        defer close(errCh)

        // Your chunking logic here
        // Emit chunks via chunkCh
        // Report errors via errCh
    }()

    return chunkCh, errCh
}

// Convenience method for small text
func (c *NewStrategyChunker) Chunk(text string, options ChunkOptions) ([]Chunk, error) {
    reader := strings.NewReader(text)
    chunkCh, errCh := c.ChunkStream(reader, options)

    var chunks []Chunk
    for chunk := range chunkCh {
        chunks = append(chunks, chunk)
    }

    select {
    case err := <-errCh:
        if err != nil {
            return nil, err
        }
    default:
    }

    return chunks, nil
}

// Register in pkg/chunking/chunker.go NewChunkerFactory()
func NewChunkerFactory() *ChunkerFactory {
    factory := &ChunkerFactory{
        chunkers: make(map[string]Chunker),
    }

    factory.Register(&SimpleChunker{})
    factory.Register(&SeparatorChunker{})
    factory.Register(&SentenceChunker{})
    factory.Register(&TokenChunker{})
    factory.Register(&NewStrategyChunker{}) // Add your chunker

    return factory
}
```

### 4. API Endpoints
To add new API endpoints:

```go
// pkg/api/newendpoint.go
package api

func (s *Server) handleNewEndpoint(w http.ResponseWriter, r *http.Request) {
    // Implementation

    response := map[string]interface{}{
        "data": result,
    }

    w.Header().Set("Content-Type", "application/json")
    json.NewEncoder(w).Encode(response)
}

// Register in pkg/api/server.go
func (s *Server) setupRoutes() {
    s.router.HandleFunc("/new-endpoint", s.handleNewEndpoint).Methods("GET")
}
```

## Debugging

### Debug Mode
```bash
# Enable debug logging
AGENTFS_LOG_LEVEL=debug ./build/agentfs

# Enable pprof profiling
go build -tags "fts5,debug" -o build/agentfs ./cmd/agentfs
./build/agentfs &
go tool pprof http://localhost:8080/debug/pprof/profile
```

### Common Issues

#### ONNX Runtime Not Found
```bash
# Check ONNX_PATH
echo $ONNX_PATH

# Verify library exists
ls -la $ONNX_PATH/libonnxruntime.so

# Install dependencies
sudo apt-get install libonnxruntime-dev  # Ubuntu/Debian
brew install onnxruntime                 # macOS
```

#### SQLite FTS5 Not Available
```bash
# Test FTS5 support
sqlite3 ":memory:" "CREATE VIRTUAL TABLE test USING fts5(content);"

# Rebuild SQLite with FTS5
sudo apt-get install libsqlite3-dev  # Ubuntu/Debian
```

#### File Permission Issues
```bash
# Check AgentFS directory permissions
ls -la ~/.agentfs/

# Fix permissions
chmod 755 ~/.agentfs/
chmod 644 ~/.agentfs/config.json
```

## Contributing

### Development Workflow
1. Fork the repository
2. Create a feature branch: `git checkout -b feature/new-feature`
3. Make changes and add tests
4. Run tests: `make test`
5. Format code: `make fmt`
6. Commit changes: `git commit -m "Add new feature"`
7. Push branch: `git push origin feature/new-feature`
8. Create pull request

### Pull Request Guidelines
- Include tests for new functionality
- Update documentation for user-facing changes
- Ensure all tests pass
- Follow existing code style
- Include clear commit messages

### Release Process
1. Update version in `cmd/agentfs/main.go`
2. Update CHANGELOG.md
3. Create git tag: `git tag v0.3.0`
4. Push tag: `git push origin v0.3.0`
5. GitHub Actions will build and create release

## Performance Considerations

### Profiling
```bash
# CPU profiling
go test -cpuprofile=cpu.prof -bench=. ./pkg/search/
go tool pprof cpu.prof

# Memory profiling
go test -memprofile=mem.prof -bench=. ./pkg/embeddings/
go tool pprof mem.prof

# Block profiling
go test -blockprofile=block.prof -bench=. ./pkg/database/
go tool pprof block.prof
```

### Optimization Tips
- Use connection pooling for databases
- Implement caching for expensive operations
- Batch operations when possible
- Use context for cancellation and timeouts
- Monitor memory usage with large datasets

### Benchmarking
```bash
# Run specific benchmarks
go test -bench=BenchmarkEmbed -benchmem ./pkg/embeddings/
go test -bench=BenchmarkSearch -benchtime=10s ./pkg/search/

# Compare performance
go test -bench=. -count=5 ./pkg/database/ | tee results.txt
benchstat results.txt
```