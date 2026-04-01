package parsers

import (
	"fmt"
	"io"
	"path/filepath"
	"strings"
)

// Parser interface defines the contract for file parsers
type Parser interface {
	// Parse reads and parses the content of a file
	Parse(content io.Reader) (string, error)
	// Supports checks if this parser can handle the given file extension
	Supports(extension string) bool
}

// TextParser is a simple parser for plain text files
type TextParser struct{}

// Parse reads and returns the content of a text file
func (tp *TextParser) Parse(content io.Reader) (string, error) {
	data, err := io.ReadAll(content)
	if err != nil {
		return "", fmt.Errorf("failed to read content: %w", err)
	}
	return string(data), nil
}

// Supports checks if this parser can handle the given file extension
func (tp *TextParser) Supports(extension string) bool {
	supported := []string{".txt", ".md", ".markdown", ".rst", ".adoc", ".asciidoc", ".log"}
	for _, ext := range supported {
		if ext == extension {
			return true
		}
	}
	return false
}

// GetParser returns the appropriate parser for a given file extension
func GetParser(filename string) Parser {
	extension := strings.ToLower(filepath.Ext(filename))
	
	// Use advanced parser for code files
	if isCodeFile(extension) {
		return NewAdvancedCodeParser(extension)
	}
	
	// Use text parser for text-based files
	if isTextFile(extension) {
		return &TextParser{}
	}
	
	// Use markup parser for markup files
	if isMarkupFile(extension) {
		return &TextParser{} // Using text parser for now
	}
	
	// Default to text parser for unknown file types
	return &TextParser{}
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