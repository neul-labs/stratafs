package parsers

import (
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/ledongthuc/pdf"
)

// PDFParser handles PDF document parsing with advanced text extraction
type PDFParser struct {
	filePath string
}

// NewPDFParser creates a PDF parser with the file path
func NewPDFParser(filePath string) *PDFParser {
	return &PDFParser{filePath: filePath}
}

// Parse extracts text content from PDF files with metadata
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

	// Add PDF metadata
	result.WriteString("// Document Type: PDF\n")
	result.WriteString(fmt.Sprintf("// Total Pages: %d\n", numPages))
	result.WriteString(fmt.Sprintf("// File Size: %d bytes\n", fileInfo.Size()))

	// Extract document info if available
	if info := reader.Trailer().Key("Info"); !info.IsNull() {
		result.WriteString("// Document Metadata:\n")
		// Add basic metadata extraction here
	}

	result.WriteString("\n// Document Content:\n")

	// Extract text from each page
	for i := 1; i <= numPages; i++ {
		page := reader.Page(i)
		if page.V.IsNull() {
			result.WriteString(fmt.Sprintf("\n--- Page %d: [Empty or unreadable] ---\n", i))
			continue
		}

		text, err := page.GetPlainText(nil)
		if err != nil {
			// Continue with other pages if one page fails
			result.WriteString(fmt.Sprintf("\n--- Page %d: [Error extracting text: %v] ---\n", i, err))
			continue
		}

		// Clean and format the extracted text
		cleanText := pp.cleanExtractedText(text)
		if len(strings.TrimSpace(cleanText)) > 0 {
			result.WriteString(fmt.Sprintf("\n--- Page %d ---\n", i))
			result.WriteString(cleanText)
			result.WriteString("\n")
		} else {
			result.WriteString(fmt.Sprintf("\n--- Page %d: [No readable text] ---\n", i))
		}
	}

	// Add document summary
	fullText := result.String()
	wordCount := len(strings.Fields(fullText))
	result.WriteString(fmt.Sprintf("\n// Document Summary: %d pages, ~%d words\n", numPages, wordCount))

	return result.String(), nil
}

// cleanExtractedText cleans and formats extracted PDF text
func (pp *PDFParser) cleanExtractedText(text string) string {
	// Remove excessive whitespace
	text = strings.TrimSpace(text)

	// Replace multiple spaces with single space
	for strings.Contains(text, "  ") {
		text = strings.ReplaceAll(text, "  ", " ")
	}

	// Replace multiple newlines with double newlines
	lines := strings.Split(text, "\n")
	var cleanLines []string

	for _, line := range lines {
		line = strings.TrimSpace(line)
		if len(line) > 0 {
			cleanLines = append(cleanLines, line)
		}
	}

	return strings.Join(cleanLines, "\n")
}

// Supports checks if this parser can handle PDF files
func (pp *PDFParser) Supports(extension string) bool {
	return extension == ".pdf"
}

// GetMetadata extracts metadata from PDF if available
func (pp *PDFParser) GetMetadata() (map[string]string, error) {
	file, err := os.Open(pp.filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to open PDF file: %w", err)
	}
	defer file.Close()

	fileInfo, err := file.Stat()
	if err != nil {
		return nil, fmt.Errorf("failed to stat PDF file: %w", err)
	}

	reader, err := pdf.NewReader(file, fileInfo.Size())
	if err != nil {
		return nil, fmt.Errorf("failed to create PDF reader: %w", err)
	}

	metadata := make(map[string]string)
	metadata["pages"] = fmt.Sprintf("%d", reader.NumPage())
	metadata["size"] = fmt.Sprintf("%d", fileInfo.Size())
	metadata["type"] = "PDF"

	// Try to extract document info
	if info := reader.Trailer().Key("Info"); !info.IsNull() {
		// Add more sophisticated metadata extraction here if needed
		metadata["has_metadata"] = "true"
	}

	return metadata, nil
}