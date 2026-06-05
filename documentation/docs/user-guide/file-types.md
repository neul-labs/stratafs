# File Types

StrataFS parses a file into plain text before chunking and embedding. The parser registry is modular — adding a new format means writing a parser, registering an extension, and rebuilding.

## Supported out of the box

### Documents

| Format | Notes |
| --- | --- |
| Markdown (`.md`, `.markdown`) | Heading-aware splitting via the `separator` chunking strategy. |
| Plain text (`.txt`) | Sentence-aware chunking by default. |
| reStructuredText (`.rst`) | Treated as text with section heuristics. |
| PDF (`.pdf`) | Text extraction via [`ledongthuc/pdf`](https://github.com/ledongthuc/pdf). Page-level breaks preserved. |
| DOCX (`.docx`) | Word XML → text; runs preserved. |
| PPTX (`.pptx`) | Slide-by-slide extraction. |
| RTF (`.rtf`) | Rich-text strip-to-plain. |

### Spreadsheets and tabular data

| Format | Notes |
| --- | --- |
| XLSX (`.xlsx`) | Sheet, row, and cell extraction. |
| XLS (`.xls`) | Legacy Excel via [`extrame/xls`](https://github.com/extrame/xls). |
| ODS (`.ods`) | OpenDocument spreadsheets. |
| CSV (`.csv`), TSV (`.tsv`) | `separator` chunking strategy by default. |

### Code and markup

| Format | Notes |
| --- | --- |
| Go (`.go`) | Whitespace-aware chunking; identifiers preserved. |
| Python (`.py`) | Function/class-level boundaries when possible. |
| JavaScript / TypeScript (`.js`, `.ts`) | — |
| Java (`.java`) | — |
| C / C++ (`.c`, `.cpp`, `.h`, `.hpp`) | — |
| Rust (`.rs`), Ruby (`.rb`), PHP (`.php`) | — |
| Shell (`.sh`, `.bash`, `.zsh`) | — |
| SQL (`.sql`) | Treated as code. |
| HTML (`.html`, `.htm`), XML (`.xml`), SVG (`.svg`) | Tag-stripped to text. |
| JSON (`.json`), YAML (`.yml`, `.yaml`), TOML (`.toml`), INI (`.ini`, `.conf`, `.cfg`) | Stringified key/value structure. |

## How the right strategy is chosen

`pkg/chunking` ships four strategies (`simple`, `separator`, `sentence`, `token`). The queue processor picks one based on the parser's classification of the file's content — Markdown, code, CSV, and similar structured formats route to `separator`; PDFs and plain text route to `sentence`; everything else falls back to `simple`. There is no top-level `chunking` config block today; the mapping lives in the processor and parser layers and is not user-tunable from `config.json`.

## Filtering by extension

Both source filters and search-time filters accept extensions:

```bash
curl "http://localhost:8080/search?q=tls+config&extensions=go,md,yml"
```

This is the fastest way to keep noisy formats (logs, generated JSON) out of your results.

## Adding a new format

Implement the `parsers.Parser` interface, register it, rebuild:

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

See [Contributing → Development](../contributing/development.md) for the full extension guide.

## Files StrataFS deliberately skips

- Anything matched by `filters.exclude_patterns` (defaults exclude `.git/**` and `node_modules/**`).
- Files above `filters.max_file_size` (default 100 MiB).
- Hidden files when `filters.ignore_hidden` is `true` (the default).
- Binary blobs with no registered parser. They are not partially indexed — they're skipped entirely.
