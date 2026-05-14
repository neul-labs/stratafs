package parsers

import (
	"fmt"
	"io"
	"regexp"
	"strings"
)

// Parser interface defines the contract for file parsers
type Parser interface {
	// Parse reads and parses the content of a file
	Parse(content io.Reader) (string, error)
	// Supports checks if this parser can handle the given file extension
	Supports(extension string) bool
}

// FileParser wraps a parser with file path information for PDF parsing
type FileParser struct{}

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
	scriptRegex := regexp.MustCompile(`(?s)<script[^>]*>.*?</script>`)
	htmlStr = scriptRegex.ReplaceAllString(htmlStr, "")
	styleRegex := regexp.MustCompile(`(?s)<style[^>]*>.*?</style>`)
	htmlStr = styleRegex.ReplaceAllString(htmlStr, "")

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

// GetParser returns the appropriate parser for a given file (deprecated - use registry)
func GetParser(filename string) Parser {
	// Use the new registry system
	return GlobalRegistry.GetParser(filename)
}

// Note: Legacy DOC format (.doc) is not supported due to complexity of the binary format.
// Modern DOCX format is supported via ZIP-based XML parsing.
// For DOC support, external tools like 'wv' or 'antiword' would be required.
