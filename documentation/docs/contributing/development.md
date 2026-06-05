# Development

This page covers the dev loop: setup, tests, code style, and where to drop new pieces of functionality.

## Prerequisites

- **Go 1.24+** — [Download Go](https://golang.org/dl/)
- **ONNX Runtime** — required for the embedding pipeline. See below.
- **SQLite with FTS5** — already shipped on modern systems. Build with `-tags fts5`.
- **A C compiler** — for the SQLite extensions.

## Setup

```bash
git clone https://github.com/neul-labs/stratafs.git
cd stratafs
go mod tidy
make fetch-onnx      # downloads ONNX Runtime to build/onnx/
make build
./build/stratafs config init
./build/stratafs serve
```

## Build

The Makefile targets that ship with the repo:

| Target | What it does |
| --- | --- |
| `make build` | Build with `-tags fts5` into `build/stratafs`. |
| `make run` | `go run -tags fts5 cmd/stratafs/main.go`. |
| `make install` | `go install -tags fts5 cmd/stratafs/main.go`. |
| `make test` | `go test -tags fts5 -v ./...`. |
| `make fetch-onnx` | Download the matching ONNX Runtime via `scripts/get-onnx-runtime.sh`. |
| `make test-onnx` | Run the full test suite with the ONNX Runtime on `LD_LIBRARY_PATH`. |
| `make release` | Cross-platform release archives via `scripts/build-release.sh`. |
| `make fmt` | `go fmt ./...`. |
| `make vet` | `go vet ./...`. |
| `make clean` | Remove `build/`. |
| `make deps` / `make update` | `go mod tidy` (with optional `go get -u ./...` first). |

## Build tags

| Tag | Effect |
| --- | --- |
| `fts5` | Enables SQLite FTS5. **Required.** All Makefile targets pass it for you. |

## Tests

```bash
go test -tags fts5 -v ./...
```

The `pkg/search` suite depends on FastEmbed + ONNX Runtime. Those tests skip cleanly when the native libraries are unavailable, so CI sandboxes without the runtime can still exercise the rest of the project.

To run the full search suite against the bundled runtime:

```bash
make test-onnx
```

### Writing tests

Patterns to follow:

```go
func TestFileParser(t *testing.T) {
    parser := parsers.NewTextParser()
    out, err := parser.Parse(strings.NewReader("hello"))
    require.NoError(t, err)
    assert.Equal(t, "hello", out)
}
```

For tests that need a database, open a SQLite DB into a `t.TempDir()` via `database.NewDB`; the schema is initialised on first open.

For tests that need a live embedder, skip gracefully when the model can't be downloaded:

```go
embedder, err := embeddings.NewEmbedder(cfg)
if err != nil {
    if strings.Contains(err.Error(), "403 Forbidden") ||
       strings.Contains(err.Error(), "model download failed") {
        t.Skip("model download blocked")
    }
    t.Fatalf("failed to create embedder: %v", err)
}
```

## Project structure

```
stratafs/
├── cmd/stratafs/        # CLI entry point
├── internal/utils/      # Private helpers
├── pkg/                 # Public libraries
│   ├── api/             # REST server
│   ├── chunking/        # Chunking strategies
│   ├── config/          # Config management
│   ├── database/        # SQLite schema helpers
│   ├── embeddings/      # FastEmbed integration
│   ├── filesystem/      # Filesystem abstraction + backends
│   ├── fsbridge/        # FUSE / WinFsp export
│   ├── monitor/         # File watcher + remote scanner
│   ├── parsers/         # Parser registry + implementations
│   ├── protocol/        # MCP server
│   ├── queue/           # Job queue + processor
│   ├── search/          # Hybrid search engine
│   └── storage/         # Storage factory + backends
├── desktop/stratafs-ui/ # Wails desktop app
├── installers/          # Native installer scripts
├── packages/            # npm / PyPI wrappers
├── docker/, scripts/    # Deployment tooling
└── research/            # Benchmarks + paper
```

## Adding things

### A new file parser

```go
// pkg/parsers/asciidoc.go
package parsers

type AsciidocParser struct{}

func (p *AsciidocParser) Parse(r io.Reader) (string, error) { /* ... */ }
func (p *AsciidocParser) SupportedExtensions() []string {
    return []string{".adoc", ".asciidoc"}
}

func init() {
    DefaultRegistry.Register(NewAsciidocParserFactory())
}
```

Register tests under `pkg/parsers/parser_test.go`.

### A new storage backend

1. Implement `filesystem.FileSystem` in `pkg/filesystem/<backend>.go`.
2. Add a credential struct to `pkg/config/`.
3. Add a factory branch in `pkg/storage/factory.go`.
4. Write integration tests against a local fake (MinIO for S3-compatible, etc.).

### A new chunking strategy

1. Add `pkg/chunking/<strategy>.go` implementing `Chunker`.
2. Register in `pkg/chunking/registry.go`.
3. Add a mapping in the default `file_type_strategies` if appropriate.

### A new API endpoint

1. Add a handler in `pkg/api/`.
2. Register the route.
3. Update `pkg/api/openapi.go` so `/openapi.json` stays accurate.
4. Add a corresponding MCP tool if the endpoint is useful for agents.

## Code style

- Follow [Effective Go](https://golang.org/doc/effective_go.html).
- `gofmt` on save.
- Interface names end in `-er` (`Parser`, `Chunker`, `Embedder`).
- Errors: always check, always wrap with context (`fmt.Errorf("scan failed: %w", err)`).
- Logging: structured logging with `slog`. Use the package logger; don't `fmt.Println` from library code.
- Database methods return `nil, nil` for not-found cases, not errors:

```go
func (db *DB) GetFileByPath(path string) (*File, error) {
    err := db.conn.QueryRow("SELECT ... WHERE path = ?", path).Scan(...)
    if err == sql.ErrNoRows {
        return nil, nil
    }
    return file, err
}
```

## Sending a PR

1. Fork and branch from `main`.
2. `make fmt && make vet && make test`.
3. Open a PR with a description of the **why** — the **what** is in the diff.
4. CI runs the full suite plus a Docker build. Both must be green.
5. One reviewer approval. We try to respond within 48 hours.

For larger changes, open an issue first to align on the approach.
