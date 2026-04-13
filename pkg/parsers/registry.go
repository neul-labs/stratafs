package parsers

import (
	"io"
	"path/filepath"
	"strings"
	"sync"
)

// ParserRegistry manages parser registration and selection
type ParserRegistry struct {
	mu      sync.RWMutex
	parsers map[string]ParserFactory
}

// ParserFactory creates a new parser instance
type ParserFactory func(filePath string) Parser

// GlobalRegistry is the default parser registry
var GlobalRegistry = NewParserRegistry()

// NewParserRegistry creates a new parser registry
func NewParserRegistry() *ParserRegistry {
	registry := &ParserRegistry{
		parsers: make(map[string]ParserFactory),
	}

	// Register built-in parsers
	registry.RegisterDefaults()

	return registry
}

// RegisterDefaults registers all built-in parsers
func (r *ParserRegistry) RegisterDefaults() {
	// Text files
	r.RegisterParser(".txt", func(path string) Parser { return &TextParser{} })
	r.RegisterParser(".md", func(path string) Parser { return &MarkdownParser{} })
	r.RegisterParser(".markdown", func(path string) Parser { return &MarkdownParser{} })
	r.RegisterParser(".rst", func(path string) Parser { return &TextParser{} })
	r.RegisterParser(".log", func(path string) Parser { return &LogParser{} })
	r.RegisterParser(".csv", func(path string) Parser { return &CSVParser{} })
	r.RegisterParser(".tsv", func(path string) Parser { return &CSVParser{} })

	// Code files
	r.RegisterParser(".go", func(path string) Parser { return NewCodeParser("go") })
	r.RegisterParser(".py", func(path string) Parser { return NewCodeParser("python") })
	r.RegisterParser(".js", func(path string) Parser { return NewCodeParser("javascript") })
	r.RegisterParser(".ts", func(path string) Parser { return NewCodeParser("typescript") })
	r.RegisterParser(".java", func(path string) Parser { return NewCodeParser("java") })
	r.RegisterParser(".cpp", func(path string) Parser { return NewCodeParser("cpp") })
	r.RegisterParser(".c", func(path string) Parser { return NewCodeParser("c") })
	r.RegisterParser(".h", func(path string) Parser { return NewCodeParser("c") })
	r.RegisterParser(".rs", func(path string) Parser { return NewCodeParser("rust") })
	r.RegisterParser(".swift", func(path string) Parser { return NewCodeParser("swift") })
	r.RegisterParser(".kt", func(path string) Parser { return NewCodeParser("kotlin") })
	r.RegisterParser(".php", func(path string) Parser { return NewCodeParser("php") })
	r.RegisterParser(".rb", func(path string) Parser { return NewCodeParser("ruby") })
	r.RegisterParser(".sh", func(path string) Parser { return NewCodeParser("bash") })
	r.RegisterParser(".bash", func(path string) Parser { return NewCodeParser("bash") })
	r.RegisterParser(".zsh", func(path string) Parser { return NewCodeParser("zsh") })
	r.RegisterParser(".sql", func(path string) Parser { return NewCodeParser("sql") })

	// Markup and config files
	r.RegisterParser(".html", func(path string) Parser { return &HTMLParser{} })
	r.RegisterParser(".htm", func(path string) Parser { return &HTMLParser{} })
	r.RegisterParser(".xml", func(path string) Parser { return &XMLParser{} })
	r.RegisterParser(".json", func(path string) Parser { return &JSONParser{} })
	r.RegisterParser(".yaml", func(path string) Parser { return &YAMLParser{} })
	r.RegisterParser(".yml", func(path string) Parser { return &YAMLParser{} })
	r.RegisterParser(".toml", func(path string) Parser { return &TOMLParser{} })

	// Documents
	r.RegisterParser(".pdf", func(path string) Parser { return NewPDFParser(path) })
	r.RegisterParser(".docx", func(path string) Parser { return NewDocumentParser(path) })
	r.RegisterParser(".pptx", func(path string) Parser { return NewDocumentParser(path) })
	r.RegisterParser(".rtf", func(path string) Parser { return &RTFParser{} })

	// Spreadsheets
	r.RegisterParser(".xlsx", func(path string) Parser { return NewSpreadsheetParser(path) })
	r.RegisterParser(".ods", func(path string) Parser { return NewSpreadsheetParser(path) })
	r.RegisterParser(".csv", func(path string) Parser { return NewCSVAdvancedParser(',') })
	r.RegisterParser(".tsv", func(path string) Parser { return NewTSVParser() })

	// Other formats
	r.RegisterParser(".ini", func(path string) Parser { return &ConfigParser{} })
	r.RegisterParser(".conf", func(path string) Parser { return &ConfigParser{} })
	r.RegisterParser(".cfg", func(path string) Parser { return &ConfigParser{} })
	r.RegisterParser(".env", func(path string) Parser { return &ConfigParser{} })
}

// RegisterParser registers a parser factory for a file extension
func (r *ParserRegistry) RegisterParser(extension string, factory ParserFactory) {
	r.mu.Lock()
	defer r.mu.Unlock()

	extension = strings.ToLower(extension)
	if !strings.HasPrefix(extension, ".") {
		extension = "." + extension
	}

	r.parsers[extension] = factory
}

// GetParser returns a parser for the given file path
func (r *ParserRegistry) GetParser(filePath string) Parser {
	r.mu.RLock()
	defer r.mu.RUnlock()

	extension := strings.ToLower(filepath.Ext(filePath))

	if factory, exists := r.parsers[extension]; exists {
		return factory(filePath)
	}

	// Fall back to text parser for unknown extensions that might be text
	if r.mightBeTextFile(extension) {
		return &TextParser{}
	}

	return nil // Unsupported file type
}

// IsSupported checks if a file extension is supported
func (r *ParserRegistry) IsSupported(extension string) bool {
	r.mu.RLock()
	defer r.mu.RUnlock()

	extension = strings.ToLower(extension)
	if !strings.HasPrefix(extension, ".") {
		extension = "." + extension
	}

	_, exists := r.parsers[extension]
	if exists {
		return true
	}

	return r.mightBeTextFile(extension)
}

// GetSupportedExtensions returns all supported file extensions
func (r *ParserRegistry) GetSupportedExtensions() []string {
	r.mu.RLock()
	defer r.mu.RUnlock()

	extensions := make([]string, 0, len(r.parsers))
	for ext := range r.parsers {
		extensions = append(extensions, ext)
	}

	return extensions
}

// mightBeTextFile checks if an unknown extension might be a text file
func (r *ParserRegistry) mightBeTextFile(extension string) bool {
	// Some extensions that are likely text but not explicitly registered
	likelyText := []string{
		".txt", ".text", ".doc", ".rtf", ".readme", ".license", ".changelog",
		".makefile", ".dockerfile", ".gitignore", ".gitattributes",
	}

	for _, ext := range likelyText {
		if extension == ext {
			return true
		}
	}

	// Files without extensions might be text (like README, Makefile, etc.)
	return extension == ""
}

// Enhanced parser types for better content extraction

// MarkdownParser handles Markdown files with metadata extraction
type MarkdownParser struct{}

func (mp *MarkdownParser) Parse(content io.Reader) (string, error) {
	// For now, treat as text but could extract headers, links, etc.
	return (&TextParser{}).Parse(content)
}

func (mp *MarkdownParser) Supports(extension string) bool {
	return extension == ".md" || extension == ".markdown"
}

// LogParser handles log files with timestamp awareness
type LogParser struct{}

func (lp *LogParser) Parse(content io.Reader) (string, error) {
	// For now, treat as text but could extract structured log data
	return (&TextParser{}).Parse(content)
}

func (lp *LogParser) Supports(extension string) bool {
	return extension == ".log"
}

// CSVParser handles CSV/TSV files
type CSVParser struct{}

func (cp *CSVParser) Parse(content io.Reader) (string, error) {
	// For now, treat as text but could extract structured data
	return (&TextParser{}).Parse(content)
}

func (cp *CSVParser) Supports(extension string) bool {
	return extension == ".csv" || extension == ".tsv"
}

// XMLParser handles XML files
type XMLParser struct{}

func (xp *XMLParser) Parse(content io.Reader) (string, error) {
	// For now, treat as text but could extract structured data
	return (&TextParser{}).Parse(content)
}

func (xp *XMLParser) Supports(extension string) bool {
	return extension == ".xml"
}

// JSONParser handles JSON files
type JSONParser struct{}

func (jp *JSONParser) Parse(content io.Reader) (string, error) {
	// For now, treat as text but could extract structured data
	return (&TextParser{}).Parse(content)
}

func (jp *JSONParser) Supports(extension string) bool {
	return extension == ".json"
}

// YAMLParser handles YAML files
type YAMLParser struct{}

func (yp *YAMLParser) Parse(content io.Reader) (string, error) {
	// For now, treat as text but could extract structured data
	return (&TextParser{}).Parse(content)
}

func (yp *YAMLParser) Supports(extension string) bool {
	return extension == ".yaml" || extension == ".yml"
}

// TOMLParser handles TOML files
type TOMLParser struct{}

func (tp *TOMLParser) Parse(content io.Reader) (string, error) {
	// For now, treat as text but could extract structured data
	return (&TextParser{}).Parse(content)
}

func (tp *TOMLParser) Supports(extension string) bool {
	return extension == ".toml"
}

// ConfigParser handles configuration files
type ConfigParser struct{}

func (cp *ConfigParser) Parse(content io.Reader) (string, error) {
	// For now, treat as text but could extract key-value pairs
	return (&TextParser{}).Parse(content)
}

func (cp *ConfigParser) Supports(extension string) bool {
	supported := []string{".ini", ".conf", ".cfg", ".env"}
	for _, ext := range supported {
		if ext == extension {
			return true
		}
	}
	return false
}

// Convenience functions for backward compatibility

// IsSupported checks if a file extension is supported (using global registry)
func IsSupported(extension string) bool {
	return GlobalRegistry.IsSupported(extension)
}

// RegisterParser registers a parser in the global registry
func RegisterParser(extension string, factory ParserFactory) {
	GlobalRegistry.RegisterParser(extension, factory)
}