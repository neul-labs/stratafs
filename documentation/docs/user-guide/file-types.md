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
| JavaScript / TypeScript (`.js`, `.jsx`, `.ts`, `.tsx`) | — |
| Java (`.java`), Kotlin (`.kt`) | — |
| C / C++ / Objective-C (`.c`, `.cc`, `.cpp`, `.h`, `.hpp`, `.m`, `.mm`) | — |
| Rust (`.rs`), Swift (`.swift`), Ruby (`.rb`), PHP (`.php`) | — |
| Shell (`.sh`, `.bash`, `.zsh`, `.fish`) | — |
| HTML (`.html`, `.htm`), XML (`.xml`) | Tag-stripped to text. |
| JSON (`.json`), YAML (`.yml`, `.yaml`), TOML (`.toml`), INI (`.ini`) | Stringified key/value structure. |

## How the right strategy is chosen

The chunking strategy is resolved in this order:

1. Explicit per-file override (advanced — not exposed via CLI).
2. `chunking.file_type_strategies` mapping for the file extension.
3. `chunking.default_strategy`.

Defaults from `stratafs config init`:

```json
"file_type_strategies": {
  "markdown": "separator",
  "code": "separator",
  "pdf": "sentence",
  "txt": "sentence",
  "csv": "separator"
}
```

The keys are **logical categories**, not raw extensions. Internally StrataFS maps each extension to a category before consulting this map. Override the global default with `chunking.default_strategy`.

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
