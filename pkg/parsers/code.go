package parsers

import (
	"fmt"
	"io"
	"regexp"
	"strings"
)

// CodeParser handles programming language files with syntax awareness
type CodeParser struct {
	language string
	patterns *LanguagePatterns
}

// LanguagePatterns defines syntax patterns for different programming languages
type LanguagePatterns struct {
	CommentSingle   []string // Single-line comment prefixes
	CommentMulti    []string // Multi-line comment patterns (start, end)
	StringDelims    []string // String delimiters
	FunctionPattern *regexp.Regexp
	ClassPattern    *regexp.Regexp
	ImportPattern   *regexp.Regexp
}

// NewCodeParser creates a new code parser for a specific language
func NewCodeParser(language string) *CodeParser {
	return &CodeParser{
		language: language,
		patterns: getLanguagePatterns(language),
	}
}

// Parse extracts meaningful content from code files
func (cp *CodeParser) Parse(content io.Reader) (string, error) {
	data, err := io.ReadAll(content)
	if err != nil {
		return "", fmt.Errorf("failed to read code content: %w", err)
	}

	text := string(data)

	// Validate that content is actually text (not binary)
	if !isTextContent(data) {
		return "", fmt.Errorf("file contains binary content")
	}

	// Extract structured information
	result := cp.extractStructuredContent(text)

	return result, nil
}

// Supports checks if this parser can handle the given file extension
func (cp *CodeParser) Supports(extension string) bool {
	supportedExts := getLanguageExtensions(cp.language)
	for _, ext := range supportedExts {
		if ext == extension {
			return true
		}
	}
	return false
}

// extractStructuredContent extracts meaningful content from code
func (cp *CodeParser) extractStructuredContent(text string) string {
	var result strings.Builder

	lines := strings.Split(text, "\n")

	// Extract different types of content
	result.WriteString(fmt.Sprintf("// Language: %s\n", cp.language))

	// Extract imports/includes
	imports := cp.extractImports(lines)
	if len(imports) > 0 {
		result.WriteString("\n// Imports and Dependencies:\n")
		for _, imp := range imports {
			result.WriteString(fmt.Sprintf("// %s\n", imp))
		}
	}

	// Extract function definitions
	functions := cp.extractFunctions(lines)
	if len(functions) > 0 {
		result.WriteString("\n// Functions and Methods:\n")
		for _, fn := range functions {
			result.WriteString(fmt.Sprintf("// %s\n", fn))
		}
	}

	// Extract class definitions
	classes := cp.extractClasses(lines)
	if len(classes) > 0 {
		result.WriteString("\n// Classes and Types:\n")
		for _, class := range classes {
			result.WriteString(fmt.Sprintf("// %s\n", class))
		}
	}

	// Include meaningful comments
	comments := cp.extractComments(lines)
	if len(comments) > 0 {
		result.WriteString("\n// Comments and Documentation:\n")
		for _, comment := range comments {
			result.WriteString(fmt.Sprintf("// %s\n", comment))
		}
	}

	// Include the full source code for search
	result.WriteString("\n// Full Source Code:\n")
	result.WriteString(text)

	return result.String()
}

// extractImports finds import/include statements
func (cp *CodeParser) extractImports(lines []string) []string {
	var imports []string

	if cp.patterns.ImportPattern == nil {
		return imports
	}

	for _, line := range lines {
		line = strings.TrimSpace(line)
		if cp.patterns.ImportPattern.MatchString(line) {
			imports = append(imports, line)
		}
	}

	return imports
}

// extractFunctions finds function definitions
func (cp *CodeParser) extractFunctions(lines []string) []string {
	var functions []string

	if cp.patterns.FunctionPattern == nil {
		return functions
	}

	for _, line := range lines {
		line = strings.TrimSpace(line)
		if cp.patterns.FunctionPattern.MatchString(line) {
			functions = append(functions, line)
		}
	}

	return functions
}

// extractClasses finds class definitions
func (cp *CodeParser) extractClasses(lines []string) []string {
	var classes []string

	if cp.patterns.ClassPattern == nil {
		return classes
	}

	for _, line := range lines {
		line = strings.TrimSpace(line)
		if cp.patterns.ClassPattern.MatchString(line) {
			classes = append(classes, line)
		}
	}

	return classes
}

// extractComments finds meaningful comments
func (cp *CodeParser) extractComments(lines []string) []string {
	var comments []string

	inMultiComment := false
	var multiStart, multiEnd string
	if len(cp.patterns.CommentMulti) >= 2 {
		multiStart = cp.patterns.CommentMulti[0]
		multiEnd = cp.patterns.CommentMulti[1]
	}

	for _, line := range lines {
		line = strings.TrimSpace(line)

		// Handle multi-line comments
		if multiStart != "" {
			if strings.Contains(line, multiStart) {
				inMultiComment = true
			}
			if inMultiComment {
				cleaned := cp.cleanComment(line, []string{multiStart, multiEnd})
				if len(cleaned) > 10 { // Only meaningful comments
					comments = append(comments, cleaned)
				}
			}
			if strings.Contains(line, multiEnd) {
				inMultiComment = false
			}
			continue
		}

		// Handle single-line comments
		for _, prefix := range cp.patterns.CommentSingle {
			if strings.HasPrefix(line, prefix) {
				cleaned := cp.cleanComment(line, []string{prefix})
				if len(cleaned) > 10 { // Only meaningful comments
					comments = append(comments, cleaned)
				}
				break
			}
		}
	}

	return comments
}

// cleanComment removes comment delimiters and cleans up the text
func (cp *CodeParser) cleanComment(line string, delimiters []string) string {
	cleaned := line
	for _, delim := range delimiters {
		cleaned = strings.ReplaceAll(cleaned, delim, " ")
	}
	cleaned = strings.TrimSpace(cleaned)
	cleaned = regexp.MustCompile(`\s+`).ReplaceAllString(cleaned, " ")
	return cleaned
}

// getLanguagePatterns returns syntax patterns for different languages
func getLanguagePatterns(language string) *LanguagePatterns {
	switch strings.ToLower(language) {
	case "go":
		return &LanguagePatterns{
			CommentSingle:   []string{"//"},
			CommentMulti:    []string{"/*", "*/"},
			StringDelims:    []string{"\"", "`"},
			FunctionPattern: regexp.MustCompile(`^func\s+(\w+|\([^)]*\)\s*\w+)\s*\(`),
			ClassPattern:    regexp.MustCompile(`^type\s+\w+\s+(struct|interface)`),
			ImportPattern:   regexp.MustCompile(`^import\s+`),
		}

	case "python":
		return &LanguagePatterns{
			CommentSingle:   []string{"#"},
			CommentMulti:    []string{`"""`, `"""`},
			StringDelims:    []string{"\"", "'"},
			FunctionPattern: regexp.MustCompile(`^def\s+\w+\s*\(`),
			ClassPattern:    regexp.MustCompile(`^class\s+\w+`),
			ImportPattern:   regexp.MustCompile(`^(import|from)\s+`),
		}

	case "javascript", "typescript":
		return &LanguagePatterns{
			CommentSingle:   []string{"//"},
			CommentMulti:    []string{"/*", "*/"},
			StringDelims:    []string{"\"", "'", "`"},
			FunctionPattern: regexp.MustCompile(`^(function\s+\w+|const\s+\w+\s*=|\w+\s*:\s*function)`),
			ClassPattern:    regexp.MustCompile(`^class\s+\w+`),
			ImportPattern:   regexp.MustCompile(`^(import|export)\s+`),
		}

	case "java":
		return &LanguagePatterns{
			CommentSingle:   []string{"//"},
			CommentMulti:    []string{"/*", "*/"},
			StringDelims:    []string{"\""},
			FunctionPattern: regexp.MustCompile(`^\s*(public|private|protected)?\s*(static)?\s*\w+\s+\w+\s*\(`),
			ClassPattern:    regexp.MustCompile(`^(public\s+)?(class|interface|enum)\s+\w+`),
			ImportPattern:   regexp.MustCompile(`^import\s+`),
		}

	case "cpp", "c":
		return &LanguagePatterns{
			CommentSingle:   []string{"//"},
			CommentMulti:    []string{"/*", "*/"},
			StringDelims:    []string{"\""},
			FunctionPattern: regexp.MustCompile(`^\w+\s+\w+\s*\(`),
			ClassPattern:    regexp.MustCompile(`^(class|struct)\s+\w+`),
			ImportPattern:   regexp.MustCompile(`^#include\s+`),
		}

	case "rust":
		return &LanguagePatterns{
			CommentSingle:   []string{"//"},
			CommentMulti:    []string{"/*", "*/"},
			StringDelims:    []string{"\""},
			FunctionPattern: regexp.MustCompile(`^fn\s+\w+\s*\(`),
			ClassPattern:    regexp.MustCompile(`^(struct|enum|trait)\s+\w+`),
			ImportPattern:   regexp.MustCompile(`^use\s+`),
		}

	case "php":
		return &LanguagePatterns{
			CommentSingle:   []string{"//", "#"},
			CommentMulti:    []string{"/*", "*/"},
			StringDelims:    []string{"\"", "'"},
			FunctionPattern: regexp.MustCompile(`^function\s+\w+\s*\(`),
			ClassPattern:    regexp.MustCompile(`^class\s+\w+`),
			ImportPattern:   regexp.MustCompile(`^(require|include)\s+`),
		}

	case "ruby":
		return &LanguagePatterns{
			CommentSingle:   []string{"#"},
			StringDelims:    []string{"\"", "'"},
			FunctionPattern: regexp.MustCompile(`^def\s+\w+`),
			ClassPattern:    regexp.MustCompile(`^class\s+\w+`),
			ImportPattern:   regexp.MustCompile(`^require\s+`),
		}

	case "bash", "zsh":
		return &LanguagePatterns{
			CommentSingle:   []string{"#"},
			StringDelims:    []string{"\"", "'"},
			FunctionPattern: regexp.MustCompile(`^\w+\s*\(\s*\)\s*\{`),
			ImportPattern:   regexp.MustCompile(`^(source|\.|require)\s+`),
		}

	case "sql":
		return &LanguagePatterns{
			CommentSingle: []string{"--"},
			CommentMulti:  []string{"/*", "*/"},
			StringDelims:  []string{"'"},
		}

	default:
		// Generic patterns for unknown languages
		return &LanguagePatterns{
			CommentSingle: []string{"//", "#"},
			CommentMulti:  []string{"/*", "*/"},
			StringDelims:  []string{"\"", "'"},
		}
	}
}

// getLanguageExtensions returns file extensions for different languages
func getLanguageExtensions(language string) []string {
	switch strings.ToLower(language) {
	case "go":
		return []string{".go"}
	case "python":
		return []string{".py", ".pyw"}
	case "javascript":
		return []string{".js", ".mjs"}
	case "typescript":
		return []string{".ts", ".tsx"}
	case "java":
		return []string{".java"}
	case "cpp":
		return []string{".cpp", ".cxx", ".cc", ".C"}
	case "c":
		return []string{".c", ".h"}
	case "rust":
		return []string{".rs"}
	case "swift":
		return []string{".swift"}
	case "kotlin":
		return []string{".kt", ".kts"}
	case "php":
		return []string{".php", ".phtml"}
	case "ruby":
		return []string{".rb", ".rbw"}
	case "bash":
		return []string{".sh", ".bash"}
	case "zsh":
		return []string{".zsh"}
	case "sql":
		return []string{".sql"}
	default:
		return []string{}
	}
}