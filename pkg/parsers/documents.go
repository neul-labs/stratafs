package parsers

import (
	"archive/zip"
	"fmt"
	"io"
	"path/filepath"
	"regexp"
	"strings"
)

// DocumentParser handles Microsoft Office document formats (DOCX, PPTX)
type DocumentParser struct {
	filePath     string
	documentType string
}

// NewDocumentParser creates a new document parser
func NewDocumentParser(filePath string) *DocumentParser {
	ext := strings.ToLower(filepath.Ext(filePath))
	var docType string

	switch ext {
	case ".docx":
		docType = "Word Document"
	case ".pptx":
		docType = "PowerPoint Presentation"
	default:
		docType = "Office Document"
	}

	return &DocumentParser{
		filePath:     filePath,
		documentType: docType,
	}
}

// Parse extracts text content from Office documents
func (dp *DocumentParser) Parse(content io.Reader) (string, error) {
	// Office documents are ZIP archives, open as such
	r, err := zip.OpenReader(dp.filePath)
	if err != nil {
		return "", fmt.Errorf("failed to open document as ZIP: %w", err)
	}
	defer r.Close()

	var result strings.Builder
	result.WriteString(fmt.Sprintf("// Document Type: %s\n", dp.documentType))

	ext := strings.ToLower(filepath.Ext(dp.filePath))
	switch ext {
	case ".docx":
		content, err := dp.parseWordDocument(&r.Reader)
		if err != nil {
			return "", fmt.Errorf("failed to parse Word document: %w", err)
		}
		result.WriteString(content)

	case ".pptx":
		content, err := dp.parsePowerPointDocument(&r.Reader)
		if err != nil {
			return "", fmt.Errorf("failed to parse PowerPoint document: %w", err)
		}
		result.WriteString(content)

	default:
		return "", fmt.Errorf("unsupported document type: %s", ext)
	}

	return result.String(), nil
}

// parseWordDocument extracts text from DOCX files
func (dp *DocumentParser) parseWordDocument(r *zip.Reader) (string, error) {
	var result strings.Builder

	// Find and parse the main document
	for _, f := range r.File {
		if f.Name == "word/document.xml" {
			content, err := dp.extractTextFromXML(f, "w:t")
			if err != nil {
				return "", fmt.Errorf("failed to extract text from document.xml: %w", err)
			}
			result.WriteString("\n// Main Document Content:\n")
			result.WriteString(content)
			break
		}
	}

	// Extract headers and footers
	for _, f := range r.File {
		if strings.HasPrefix(f.Name, "word/header") || strings.HasPrefix(f.Name, "word/footer") {
			content, err := dp.extractTextFromXML(f, "w:t")
			if err != nil {
				continue // Skip if we can't parse headers/footers
			}
			if len(strings.TrimSpace(content)) > 0 {
				result.WriteString(fmt.Sprintf("\n// %s Content:\n", f.Name))
				result.WriteString(content)
			}
		}
	}

	// Extract comments if present
	for _, f := range r.File {
		if f.Name == "word/comments.xml" {
			content, err := dp.extractTextFromXML(f, "w:t")
			if err != nil {
				continue
			}
			if len(strings.TrimSpace(content)) > 0 {
				result.WriteString("\n// Comments:\n")
				result.WriteString(content)
			}
		}
	}

	return result.String(), nil
}

// parsePowerPointDocument extracts text from PPTX files
func (dp *DocumentParser) parsePowerPointDocument(r *zip.Reader) (string, error) {
	var result strings.Builder

	// Extract slide content
	slideCount := 0
	for _, f := range r.File {
		if strings.HasPrefix(f.Name, "ppt/slides/slide") && strings.HasSuffix(f.Name, ".xml") {
			slideCount++
			content, err := dp.extractTextFromXML(f, "a:t")
			if err != nil {
				result.WriteString(fmt.Sprintf("\n--- Slide %d: [Error extracting content] ---\n", slideCount))
				continue
			}

			if len(strings.TrimSpace(content)) > 0 {
				result.WriteString(fmt.Sprintf("\n--- Slide %d ---\n", slideCount))
				result.WriteString(content)
				result.WriteString("\n")
			} else {
				result.WriteString(fmt.Sprintf("\n--- Slide %d: [No text content] ---\n", slideCount))
			}
		}
	}

	// Extract slide notes
	for _, f := range r.File {
		if strings.HasPrefix(f.Name, "ppt/notesSlides/notesSlide") && strings.HasSuffix(f.Name, ".xml") {
			content, err := dp.extractTextFromXML(f, "a:t")
			if err != nil {
				continue
			}
			if len(strings.TrimSpace(content)) > 0 {
				slideNum := dp.extractSlideNumber(f.Name)
				result.WriteString(fmt.Sprintf("\n// Slide %s Notes:\n", slideNum))
				result.WriteString(content)
			}
		}
	}

	result.WriteString(fmt.Sprintf("\n// Presentation Summary: %d slides\n", slideCount))
	return result.String(), nil
}

// extractTextFromXML extracts text content from XML files using the specified tag
func (dp *DocumentParser) extractTextFromXML(f *zip.File, textTag string) (string, error) {
	rc, err := f.Open()
	if err != nil {
		return "", err
	}
	defer rc.Close()

	content, err := io.ReadAll(rc)
	if err != nil {
		return "", err
	}

	// Use regex to extract text from XML tags
	pattern := fmt.Sprintf("<%s[^>]*>([^<]*)</%s>", textTag, textTag)
	re := regexp.MustCompile(pattern)
	matches := re.FindAllStringSubmatch(string(content), -1)

	var texts []string
	for _, match := range matches {
		if len(match) > 1 {
			text := strings.TrimSpace(match[1])
			if len(text) > 0 {
				texts = append(texts, text)
			}
		}
	}

	return strings.Join(texts, " "), nil
}

// extractSlideNumber extracts slide number from filename
func (dp *DocumentParser) extractSlideNumber(filename string) string {
	re := regexp.MustCompile(`notesSlide(\d+)\.xml`)
	matches := re.FindStringSubmatch(filename)
	if len(matches) > 1 {
		return matches[1]
	}
	return "unknown"
}

// Supports checks if this parser can handle the given file extension
func (dp *DocumentParser) Supports(extension string) bool {
	supported := []string{".docx", ".pptx"}
	for _, ext := range supported {
		if ext == extension {
			return true
		}
	}
	return false
}

// GetMetadata extracts metadata from Office documents
func (dp *DocumentParser) GetMetadata() (map[string]string, error) {
	r, err := zip.OpenReader(dp.filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to open document: %w", err)
	}
	defer r.Close()

	metadata := make(map[string]string)
	metadata["type"] = dp.documentType

	// Extract core properties if available
	for _, f := range r.File {
		if f.Name == "docProps/core.xml" {
			rc, err := f.Open()
			if err != nil {
				continue
			}
			defer rc.Close()

			content, err := io.ReadAll(rc)
			if err != nil {
				continue
			}

			// Parse basic metadata
			if title := dp.extractMetadataField(string(content), "dc:title"); title != "" {
				metadata["title"] = title
			}
			if creator := dp.extractMetadataField(string(content), "dc:creator"); creator != "" {
				metadata["creator"] = creator
			}
			if created := dp.extractMetadataField(string(content), "dcterms:created"); created != "" {
				metadata["created"] = created
			}
			if modified := dp.extractMetadataField(string(content), "dcterms:modified"); modified != "" {
				metadata["modified"] = modified
			}
		}
	}

	return metadata, nil
}

// extractMetadataField extracts a specific metadata field from XML content
func (dp *DocumentParser) extractMetadataField(content, field string) string {
	pattern := fmt.Sprintf("<%s[^>]*>([^<]*)</%s>", field, field)
	re := regexp.MustCompile(pattern)
	matches := re.FindStringSubmatch(content)
	if len(matches) > 1 {
		return strings.TrimSpace(matches[1])
	}
	return ""
}

// RTFParser handles Rich Text Format documents
type RTFParser struct{}

// Parse extracts text from RTF files
func (rp *RTFParser) Parse(content io.Reader) (string, error) {
	data, err := io.ReadAll(content)
	if err != nil {
		return "", fmt.Errorf("failed to read RTF content: %w", err)
	}

	text := string(data)

	// Basic RTF text extraction (strips RTF control codes)
	result := rp.extractTextFromRTF(text)

	var output strings.Builder
	output.WriteString("// Document Type: Rich Text Format\n")
	output.WriteString("\n// Document Content:\n")
	output.WriteString(result)

	return output.String(), nil
}

// extractTextFromRTF performs basic RTF text extraction
func (rp *RTFParser) extractTextFromRTF(rtf string) string {
	// Remove RTF control words and groups
	re := regexp.MustCompile(`\\[a-z]+[0-9]*[ ]?`)
	text := re.ReplaceAllString(rtf, "")

	// Remove remaining control characters
	re = regexp.MustCompile(`[{}\\]`)
	text = re.ReplaceAllString(text, "")

	// Clean up whitespace
	text = regexp.MustCompile(`\s+`).ReplaceAllString(text, " ")
	text = strings.TrimSpace(text)

	return text
}

// Supports checks if this parser can handle RTF files
func (rp *RTFParser) Supports(extension string) bool {
	return extension == ".rtf"
}

// PlainTextDocumentParser handles plain text document formats
type PlainTextDocumentParser struct{}

// Parse processes plain text documents
func (ptdp *PlainTextDocumentParser) Parse(content io.Reader) (string, error) {
	data, err := io.ReadAll(content)
	if err != nil {
		return "", fmt.Errorf("failed to read document content: %w", err)
	}

	text := string(data)

	// Validate that content is actually text
	if !isTextContent(data) {
		return "", fmt.Errorf("file contains binary content")
	}

	var result strings.Builder
	result.WriteString("// Document Type: Plain Text Document\n")
	result.WriteString("\n// Document Content:\n")
	result.WriteString(text)

	return result.String(), nil
}

// Supports checks if this parser can handle plain text documents
func (ptdp *PlainTextDocumentParser) Supports(extension string) bool {
	supported := []string{".txt", ".text", ".doc"} // .doc here refers to plain text .doc files
	for _, ext := range supported {
		if ext == extension {
			return true
		}
	}
	return false
}
