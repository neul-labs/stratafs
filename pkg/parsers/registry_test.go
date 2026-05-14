package parsers

import (
	"strings"
	"testing"
)

func TestParserRegistry(t *testing.T) {
	registry := NewParserRegistry()

	// Test that registry has parsers registered
	extensions := registry.GetSupportedExtensions()
	if len(extensions) == 0 {
		t.Error("Registry should have default parsers registered")
	}

	// Test specific parsers
	testCases := []struct {
		extension     string
		filename      string
		shouldSupport bool
	}{
		{".go", "main.go", true},
		{".py", "script.py", true},
		{".js", "app.js", true},
		{".txt", "readme.txt", true},
		{".md", "README.md", true},
		{".json", "config.json", true},
		{".yaml", "docker-compose.yaml", true},
		{".pdf", "document.pdf", true},
		{".html", "index.html", true},
		{".xyz", "unknown.xyz", false},
		{".exe", "program.exe", false},
		{".jpg", "image.jpg", false},
	}

	for _, tc := range testCases {
		t.Run(tc.extension, func(t *testing.T) {
			supported := registry.IsSupported(tc.extension)
			if supported != tc.shouldSupport {
				t.Errorf("Extension %s: expected supported=%v, got %v", tc.extension, tc.shouldSupport, supported)
			}

			parser := registry.GetParser(tc.filename)
			hasParser := parser != nil
			if hasParser != tc.shouldSupport {
				t.Errorf("File %s: expected parser=%v, got parser=%v", tc.filename, tc.shouldSupport, hasParser)
			}
		})
	}
}

func TestCustomParserRegistration(t *testing.T) {
	registry := NewParserRegistry()

	// Register custom parser
	customExtension := ".custom"
	registry.RegisterParser(customExtension, func(path string) Parser {
		return &TextParser{}
	})

	// Test that custom parser is registered
	if !registry.IsSupported(customExtension) {
		t.Error("Custom parser should be supported")
	}

	parser := registry.GetParser("test.custom")
	if parser == nil {
		t.Error("Should return parser for custom extension")
	}

	// Test case insensitivity
	parser = registry.GetParser("test.CUSTOM")
	if parser == nil {
		t.Error("Should handle case insensitive extensions")
	}
}

func TestCodeParserLanguages(t *testing.T) {
	registry := NewParserRegistry()

	languageTests := []struct {
		filename string
		language string
	}{
		{"main.go", "go"},
		{"script.py", "python"},
		{"app.js", "javascript"},
		{"app.ts", "typescript"},
		{"Main.java", "java"},
		{"program.cpp", "cpp"},
		{"program.c", "c"},
		{"lib.rs", "rust"},
		{"script.php", "php"},
		{"script.rb", "ruby"},
		{"script.sh", "bash"},
		{"query.sql", "sql"},
	}

	for _, test := range languageTests {
		t.Run(test.filename, func(t *testing.T) {
			parser := registry.GetParser(test.filename)
			if parser == nil {
				t.Fatalf("Should return parser for %s", test.filename)
			}

			codeParser, ok := parser.(*CodeParser)
			if !ok {
				t.Fatalf("Should return CodeParser for %s", test.filename)
			}

			if codeParser.language != test.language {
				t.Errorf("Expected language %s, got %s", test.language, codeParser.language)
			}
		})
	}
}

func TestParserCreation(t *testing.T) {
	testCases := []struct {
		filename    string
		content     string
		expectError bool
	}{
		{"test.go", "package main\n\nfunc main() {\n\tprintln(\"Hello\")\n}", false},
		{"test.py", "print('Hello, World!')", false},
		{"test.txt", "Plain text content", false},
		{"test.json", `{"key": "value"}`, false},
		{"binary.dat", string([]byte{0x00, 0x01, 0x02, 0xFF}), true}, // Binary content
	}

	for _, tc := range testCases {
		t.Run(tc.filename, func(t *testing.T) {
			parser := GetParser(tc.filename)
			if parser == nil {
				if !tc.expectError {
					t.Errorf("Expected parser for %s", tc.filename)
				}
				return
			}

			content := strings.NewReader(tc.content)
			result, err := parser.Parse(content)

			if tc.expectError {
				if err == nil {
					t.Error("Expected error for binary content")
				}
			} else {
				if err != nil {
					t.Errorf("Unexpected error: %v", err)
				}
				if result == "" {
					t.Error("Expected non-empty result")
				}
			}
		})
	}
}

func TestCodeParserExtraction(t *testing.T) {
	goCode := `package main

import (
	"fmt"
	"os"
)

// Main function entry point
func main() {
	fmt.Println("Hello, World!")
}

// Helper function for processing
func process(data string) string {
	return strings.ToUpper(data)
}

type Config struct {
	Name string
	Port int
}`

	parser := NewCodeParser("go")
	content := strings.NewReader(goCode)
	result, err := parser.Parse(content)

	if err != nil {
		t.Fatalf("Failed to parse Go code: %v", err)
	}

	// Check that result contains structured information
	if !strings.Contains(result, "Language: go") {
		t.Error("Result should contain language information")
	}

	if !strings.Contains(result, "Imports and Dependencies") {
		t.Error("Result should contain import information")
	}

	if !strings.Contains(result, "Functions and Methods") {
		t.Error("Result should contain function information")
	}

	if !strings.Contains(result, "Classes and Types") {
		t.Error("Result should contain type information")
	}

	if !strings.Contains(result, "Full Source Code") {
		t.Error("Result should contain full source code")
	}

	// Verify original code is included
	if !strings.Contains(result, goCode) {
		t.Error("Result should contain original source code")
	}
}

func TestMultipleParserTypes(t *testing.T) {
	registry := NewParserRegistry()

	// Test different parser types
	parsers := map[string]string{
		"test.go":   "CodeParser",
		"test.txt":  "TextParser",
		"test.html": "HTMLParser",
		"test.pdf":  "PDFParser",
		"test.md":   "MarkdownParser",
		"test.log":  "LogParser",
		"test.csv":  "CSVAdvancedParser",
		"test.json": "JSONParser",
		"test.yaml": "YAMLParser",
		"test.xml":  "XMLParser",
	}

	for filename, expectedType := range parsers {
		t.Run(filename, func(t *testing.T) {
			parser := registry.GetParser(filename)
			if parser == nil {
				t.Fatalf("Should return parser for %s", filename)
			}

			// Check parser type
			parserType := getParserTypeName(parser)
			if parserType != expectedType {
				t.Errorf("Expected %s for %s, got %s", expectedType, filename, parserType)
			}
		})
	}
}

func TestParserSupportsMethod(t *testing.T) {
	parsers := []Parser{
		&TextParser{},
		&HTMLParser{},
		NewCodeParser("go"),
		&MarkdownParser{},
		&LogParser{},
		&CSVParser{},
		&JSONParser{},
		&YAMLParser{},
		&XMLParser{},
		&ConfigParser{},
	}

	testExtensions := []string{".txt", ".html", ".go", ".md", ".log", ".csv", ".json", ".yaml", ".xml", ".ini"}

	for _, parser := range parsers {
		parserType := getParserTypeName(parser)
		t.Run(parserType, func(t *testing.T) {
			foundSupported := false
			for _, ext := range testExtensions {
				if parser.Supports(ext) {
					foundSupported = true
					break
				}
			}
			if !foundSupported {
				t.Errorf("Parser %s should support at least one extension", parserType)
			}
		})
	}
}

func TestGlobalRegistryFunctions(t *testing.T) {
	// Test global convenience functions
	parser := GetParser("test.go")
	if parser == nil {
		t.Error("GetParser should return parser for .go files")
	}

	if !IsSupported(".go") {
		t.Error("IsSupported should return true for .go files")
	}

	if IsSupported(".unknown") {
		t.Error("IsSupported should return false for unknown extensions")
	}

	// Test registering a custom parser globally
	RegisterParser(".test", func(path string) Parser {
		return &TextParser{}
	})

	if !IsSupported(".test") {
		t.Error("Should support custom registered extension")
	}

	customParser := GetParser("file.test")
	if customParser == nil {
		t.Error("Should return parser for custom extension")
	}
}

// Helper function to get parser type name for testing
func getParserTypeName(parser Parser) string {
	switch parser.(type) {
	case *CodeParser:
		return "CodeParser"
	case *TextParser:
		return "TextParser"
	case *HTMLParser:
		return "HTMLParser"
	case *PDFParser:
		return "PDFParser"
	case *MarkdownParser:
		return "MarkdownParser"
	case *LogParser:
		return "LogParser"
	case *CSVParser:
		return "CSVParser"
	case *CSVAdvancedParser:
		return "CSVAdvancedParser"
	case *JSONParser:
		return "JSONParser"
	case *YAMLParser:
		return "YAMLParser"
	case *XMLParser:
		return "XMLParser"
	case *TOMLParser:
		return "TOMLParser"
	case *ConfigParser:
		return "ConfigParser"
	default:
		return "UnknownParser"
	}
}
