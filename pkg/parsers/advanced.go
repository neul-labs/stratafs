package parsers

import (
	"bufio"
	"fmt"
	"io"
	"strings"
)

// AdvancedCodeParser is a more sophisticated parser for code files
type AdvancedCodeParser struct {
	extension string
}

// NewAdvancedCodeParser creates a new advanced code parser
func NewAdvancedCodeParser(extension string) *AdvancedCodeParser {
	return &AdvancedCodeParser{
		extension: strings.ToLower(extension),
	}
}

// Parse reads and parses the content of a code file, extracting comments and documentation
func (acp *AdvancedCodeParser) Parse(content io.Reader) (string, error) {
	scanner := bufio.NewScanner(content)

	switch acp.extension {
	case ".go":
		return acp.parseGoFile(scanner)
	case ".py":
		return acp.parsePythonFile(scanner)
	case ".js", ".ts":
		return acp.parseJavaScriptFile(scanner)
	default:
		// Fallback to simple parsing
		data, err := io.ReadAll(content)
		if err != nil {
			return "", fmt.Errorf("failed to read content: %w", err)
		}
		return string(data), nil
	}
}

// Supports checks if this parser can handle the given file extension
func (acp *AdvancedCodeParser) Supports(extension string) bool {
	supported := []string{".go", ".py", ".js", ".ts", ".java", ".cpp", ".c", ".h", ".rs"}
	for _, ext := range supported {
		if ext == extension {
			return true
		}
	}
	return false
}

// parseGoFile parses a Go file and extracts comments and documentation
func (acp *AdvancedCodeParser) parseGoFile(scanner *bufio.Scanner) (string, error) {
	var result strings.Builder
	inBlockComment := false

	for scanner.Scan() {
		line := scanner.Text()

		// Handle block comments
		if strings.Contains(line, "/*") && !inBlockComment {
			inBlockComment = true
		}

		if inBlockComment {
			result.WriteString(line)
			result.WriteString("\n")

			if strings.Contains(line, "*/") {
				inBlockComment = false
			}
			continue
		}

		// Handle single line comments
		if strings.HasPrefix(strings.TrimSpace(line), "//") {
			result.WriteString(line)
			result.WriteString("\n")
			continue
		}

		// Include package and import statements
		if strings.HasPrefix(line, "package ") || strings.HasPrefix(line, "import ") {
			result.WriteString(line)
			result.WriteString("\n")
			continue
		}

		// Include function and type declarations (without implementation)
		if strings.HasPrefix(line, "func ") || strings.HasPrefix(line, "type ") || strings.HasPrefix(line, "var ") || strings.HasPrefix(line, "const ") {
			result.WriteString(line)
			result.WriteString("\n")
			continue
		}
	}

	if err := scanner.Err(); err != nil {
		return "", fmt.Errorf("error scanning file: %w", err)
	}

	return result.String(), nil
}

// parsePythonFile parses a Python file and extracts comments and documentation
func (acp *AdvancedCodeParser) parsePythonFile(scanner *bufio.Scanner) (string, error) {
	var result strings.Builder
	inMultilineString := false
	multilineDelimiter := ""

	for scanner.Scan() {
		line := scanner.Text()

		// Handle multiline strings (docstrings)
		if !inMultilineString && (strings.HasPrefix(line, `"""`) || strings.HasPrefix(line, `'''`)) {
			inMultilineString = true
			if strings.HasPrefix(line, `"""`) {
				multilineDelimiter = `"""`
			} else {
				multilineDelimiter = `'''`
			}
			result.WriteString(line)
			result.WriteString("\n")

			// Check if it's a single line multiline string
			if strings.Count(line, multilineDelimiter) == 2 {
				inMultilineString = false
			}
			continue
		}

		if inMultilineString {
			result.WriteString(line)
			result.WriteString("\n")

			if strings.Contains(line, multilineDelimiter) {
				inMultilineString = false
			}
			continue
		}

		// Handle single line comments
		if strings.HasPrefix(strings.TrimSpace(line), "#") {
			result.WriteString(line)
			result.WriteString("\n")
			continue
		}

		// Include function and class definitions
		if strings.HasPrefix(line, "def ") || strings.HasPrefix(line, "class ") {
			result.WriteString(line)
			result.WriteString("\n")
			continue
		}
	}

	if err := scanner.Err(); err != nil {
		return "", fmt.Errorf("error scanning file: %w", err)
	}

	return result.String(), nil
}

// parseJavaScriptFile parses a JavaScript/TypeScript file and extracts comments
func (acp *AdvancedCodeParser) parseJavaScriptFile(scanner *bufio.Scanner) (string, error) {
	var result strings.Builder
	inBlockComment := false

	for scanner.Scan() {
		line := scanner.Text()

		// Handle block comments
		if strings.Contains(line, "/*") && !inBlockComment {
			inBlockComment = true
		}

		if inBlockComment {
			result.WriteString(line)
			result.WriteString("\n")

			if strings.Contains(line, "*/") {
				inBlockComment = false
			}
			continue
		}

		// Handle single line comments
		if strings.HasPrefix(strings.TrimSpace(line), "//") {
			result.WriteString(line)
			result.WriteString("\n")
			continue
		}

		// Include function and class definitions
		if strings.Contains(line, "function ") || strings.Contains(line, "class ") || strings.HasPrefix(line, "const ") || strings.HasPrefix(line, "let ") || strings.HasPrefix(line, "var ") {
			result.WriteString(line)
			result.WriteString("\n")
			continue
		}
	}

	if err := scanner.Err(); err != nil {
		return "", fmt.Errorf("error scanning file: %w", err)
	}

	return result.String(), nil
}
