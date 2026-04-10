package parsers

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/ledongthuc/pdf"
)

// Parser interface defines the contract for file parsers
type Parser interface {
	// Parse reads and parses the content of a file
	Parse(content io.Reader) (string, error)
	// Supports checks if this parser can handle the given file extension
	Supports(extension string) bool
}

// FileParser wraps a parser with file path information for PDF parsing
type FileParser struct {
	parser   Parser
	filePath string
}

// TextParser is a simple parser for plain text files
type TextParser struct{}

// Parse reads and returns the content of a text file
func (tp *TextParser) Parse(content io.Reader) (string, error) {
	data, err := io.ReadAll(content)
	if err != nil {
		return "", fmt.Errorf("failed to read content: %w", err)
	}

	// Validate that content is actually text (not binary)
	if !isTextContent(data) {
		return "", fmt.Errorf("file contains binary content")
	}

	return string(data), nil
}

// Supports checks if this parser can handle the given file extension
func (tp *TextParser) Supports(extension string) bool {
	supported := []string{".txt", ".md", ".markdown", ".rst", ".adoc", ".asciidoc", ".log", ".csv", ".tsv"}
	for _, ext := range supported {
		if ext == extension {
			return true
		}
	}
	return false
}

// PDFParser handles PDF document parsing
type PDFParser struct {
	filePath string
}

// NewPDFParser creates a PDF parser with the file path
func NewPDFParser(filePath string) *PDFParser {
	return &PDFParser{filePath: filePath}
}

// Parse extracts text content from PDF files
func (pp *PDFParser) Parse(content io.Reader) (string, error) {
	// For PDF parsing, we need direct file access
	file, err := os.Open(pp.filePath)
	if err != nil {
		return "", fmt.Errorf("failed to open PDF file: %w", err)
	}
	defer file.Close()

	fileInfo, err := file.Stat()
	if err != nil {
		return "", fmt.Errorf("failed to stat PDF file: %w", err)
	}

	reader, err := pdf.NewReader(file, fileInfo.Size())
	if err != nil {
		return "", fmt.Errorf("failed to create PDF reader: %w", err)
	}

	var result strings.Builder
	numPages := reader.NumPage()

	for i := 1; i <= numPages; i++ {
		page := reader.Page(i)
		if page.V.IsNull() {
			continue
		}

		text, err := page.GetPlainText(nil)
		if err != nil {
			// Continue with other pages if one page fails
			continue
		}

		result.WriteString(fmt.Sprintf("\n--- Page %d ---\n", i))
		result.WriteString(text)
		result.WriteString("\n")
	}

	return result.String(), nil
}

// Supports checks if this parser can handle PDF files
func (pp *PDFParser) Supports(extension string) bool {
	return extension == ".pdf"
}

// HTMLParser handles HTML document parsing
type HTMLParser struct{}

// Parse extracts text content from HTML files
func (hp *HTMLParser) Parse(content io.Reader) (string, error) {
	data, err := io.ReadAll(content)
	if err != nil {
		return "", fmt.Errorf("failed to read HTML content: %w", err)
	}

	htmlStr := string(data)

	// Remove script and style content
	scriptRegex := regexp.MustCompile(`(?s)<(script|style)[^>]*>.*?</\1>`)
	htmlStr = scriptRegex.ReplaceAllString(htmlStr, "")

	// Remove HTML tags
	tagRegex := regexp.MustCompile(`<[^>]*>`)
	htmlStr = tagRegex.ReplaceAllString(htmlStr, " ")

	// Clean up whitespace
	htmlStr = regexp.MustCompile(`\s+`).ReplaceAllString(htmlStr, " ")
	htmlStr = strings.TrimSpace(htmlStr)

	return htmlStr, nil
}

// Supports checks if this parser can handle HTML files
func (hp *HTMLParser) Supports(extension string) bool {
	return extension == ".html" || extension == ".htm"
}

// isTextContent checks if content appears to be text (not binary)
func isTextContent(data []byte) bool {
	if len(data) == 0 {
		return true
	}

	// Check for null bytes (common in binary files)
	for _, b := range data {
		if b == 0 {
			return false
		}
	}

	// Check the ratio of printable characters
	printable := 0
	for _, b := range data {
		if b >= 32 && b <= 126 || b == '\n' || b == '\r' || b == '\t' {
			printable++
		}
	}

	ratio := float64(printable) / float64(len(data))
	return ratio > 0.95 // 95% of characters should be printable
}

// GetParser returns the appropriate parser for a given file
func GetParser(filename string) Parser {
	extension := strings.ToLower(filepath.Ext(filename))

	// Check if the file type is supported first
	if !shouldParseFile(extension) {
		return nil // Return nil for unsupported files
	}

	// Use specific parsers for different file types
	switch extension {
	case ".pdf":
		return NewPDFParser(filename)
	case ".html", ".htm":
		return &HTMLParser{}
	case ".json", ".yaml", ".yml", ".xml":
		return &TextParser{} // These are structured text files
	default:
		// Use advanced parser for code files
		if isCodeFile(extension) {
			return NewAdvancedCodeParser(extension)
		}

		// Use text parser for text-based files
		return &TextParser{}
	}
}

// shouldParseFile determines if a file should be parsed based on its extension
func shouldParseFile(extension string) bool {
	// Supported text-based files
	supportedExts := []string{
		// Text files
		".txt", ".md", ".markdown", ".rst", ".adoc", ".asciidoc", ".log", ".csv", ".tsv",

		// Code files
		".go", ".py", ".js", ".ts", ".java", ".cpp", ".c", ".h", ".rs", ".swift", ".kt", ".php",
		".rb", ".sh", ".bash", ".zsh", ".ps1", ".sql",

		// Markup files
		".html", ".htm", ".xml", ".json", ".yaml", ".yml", ".toml",

		// Documents
		".pdf",
	}

	for _, ext := range supportedExts {
		if ext == extension {
			return true
		}
	}

	// Skip binary files, images, videos, archives, etc.
	unsupportedExts := []string{
		// Binary files
		".exe", ".bin", ".dll", ".so", ".dylib",

		// Images
		".jpg", ".jpeg", ".png", ".gif", ".bmp", ".webp", ".ico", ".svg",

		// Video
		".mp4", ".avi", ".mov", ".wmv", ".mkv", ".webm",

		// Audio
		".mp3", ".wav", ".flac", ".ogg", ".m4a",

		// Archives
		".zip", ".tar", ".gz", ".bz2", ".xz", ".7z", ".rar",

		// Other binary formats
		".docx", ".xlsx", ".pptx", ".pdf", // Note: PDF is supported above
	}

	for _, ext := range unsupportedExts {
		if ext == extension {
			return false
		}
	}

	// Default to parsing if we're not sure (but validate content)
	return true
}

// isCodeFile checks if the extension is for a code file
func isCodeFile(extension string) bool {
	supported := []string{".go", ".py", ".js", ".ts", ".java", ".cpp", ".c", ".h", ".rs", ".swift", ".kt", ".php"}
	for _, ext := range supported {
		if ext == extension {
			return true
		}
	}
	return false
}

// isTextFile checks if the extension is for a text file
func isTextFile(extension string) bool {
	supported := []string{".txt", ".md", ".markdown", ".rst", ".adoc", ".asciidoc", ".log"}
	for _, ext := range supported {
		if ext == extension {
			return true
		}
	}
	return false
}

// isMarkupFile checks if the extension is for a markup file
func isMarkupFile(extension string) bool {
	supported := []string{".html", ".htm", ".xml", ".json", ".yaml", ".yml"}
	for _, ext := range supported {
		if ext == extension {
			return true
		}
	}
	return false
}