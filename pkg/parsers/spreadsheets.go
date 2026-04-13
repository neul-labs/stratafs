package parsers

import (
	"archive/zip"
	"encoding/csv"
	"fmt"
	"io"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
)

// SpreadsheetParser handles Microsoft Excel formats (XLSX, XLS)
type SpreadsheetParser struct {
	filePath       string
	spreadsheetType string
}

// NewSpreadsheetParser creates a new spreadsheet parser
func NewSpreadsheetParser(filePath string) *SpreadsheetParser {
	ext := strings.ToLower(filepath.Ext(filePath))
	var sheetType string

	switch ext {
	case ".xlsx":
		sheetType = "Excel Workbook (XLSX)"
	case ".xls":
		sheetType = "Excel Legacy Workbook (XLS)"
	case ".ods":
		sheetType = "OpenDocument Spreadsheet"
	default:
		sheetType = "Spreadsheet"
	}

	return &SpreadsheetParser{
		filePath:       filePath,
		spreadsheetType: sheetType,
	}
}

// Parse extracts text content and data from spreadsheet files
func (sp *SpreadsheetParser) Parse(content io.Reader) (string, error) {
	ext := strings.ToLower(filepath.Ext(sp.filePath))

	switch ext {
	case ".xlsx":
		return sp.parseXLSX(content)
	case ".ods":
		return sp.parseODS(content)
	default:
		return "", fmt.Errorf("unsupported spreadsheet format: %s", ext)
	}
}

// parseXLSX extracts data from XLSX files
func (sp *SpreadsheetParser) parseXLSX(content io.Reader) (string, error) {
	// XLSX files are ZIP archives
	r, err := zip.OpenReader(sp.filePath)
	if err != nil {
		return "", fmt.Errorf("failed to open XLSX as ZIP: %w", err)
	}
	defer r.Close()

	var result strings.Builder
	result.WriteString(fmt.Sprintf("// Spreadsheet Type: %s\n", sp.spreadsheetType))

	// Parse shared strings first (for string references)
	sharedStrings, err := sp.parseSharedStrings(&r.Reader)
	if err != nil {
		// Continue without shared strings if not available
		sharedStrings = []string{}
	}

	// Find and parse worksheets
	worksheetCount := 0
	for _, f := range r.File {
		if strings.HasPrefix(f.Name, "xl/worksheets/sheet") && strings.HasSuffix(f.Name, ".xml") {
			worksheetCount++
			sheetName := sp.extractSheetName(f.Name)

			content, err := sp.parseWorksheet(f, sharedStrings)
			if err != nil {
				result.WriteString(fmt.Sprintf("\n// Sheet %s: [Error parsing] - %v\n", sheetName, err))
				continue
			}

			if len(strings.TrimSpace(content)) > 0 {
				result.WriteString(fmt.Sprintf("\n// Sheet %s:\n", sheetName))
				result.WriteString(content)
				result.WriteString("\n")
			} else {
				result.WriteString(fmt.Sprintf("\n// Sheet %s: [Empty]\n", sheetName))
			}
		}
	}

	result.WriteString(fmt.Sprintf("\n// Workbook Summary: %d worksheets\n", worksheetCount))
	return result.String(), nil
}

// parseODS extracts data from OpenDocument Spreadsheet files
func (sp *SpreadsheetParser) parseODS(content io.Reader) (string, error) {
	r, err := zip.OpenReader(sp.filePath)
	if err != nil {
		return "", fmt.Errorf("failed to open ODS as ZIP: %w", err)
	}
	defer r.Close()

	var result strings.Builder
	result.WriteString(fmt.Sprintf("// Spreadsheet Type: %s\n", sp.spreadsheetType))

	// Find content.xml in ODS
	for _, f := range r.File {
		if f.Name == "content.xml" {
			content, err := sp.extractTextFromXML(f, "text:p")
			if err != nil {
				return "", fmt.Errorf("failed to parse ODS content: %w", err)
			}

			result.WriteString("\n// Spreadsheet Content:\n")
			result.WriteString(content)
			break
		}
	}

	return result.String(), nil
}

// parseSharedStrings extracts shared strings from XLSX
func (sp *SpreadsheetParser) parseSharedStrings(r *zip.Reader) ([]string, error) {
	for _, f := range r.File {
		if f.Name == "xl/sharedStrings.xml" {
			rc, err := f.Open()
			if err != nil {
				return nil, err
			}
			defer rc.Close()

			content, err := io.ReadAll(rc)
			if err != nil {
				return nil, err
			}

			// Parse shared strings XML
			var sharedStrings []string
			pattern := `<si[^>]*>(?:<t[^>]*>([^<]*)</t>|<r>(?:<t[^>]*>([^<]*)</t>)+</r>)</si>`
			re := regexp.MustCompile(pattern)
			matches := re.FindAllStringSubmatch(string(content), -1)

			for _, match := range matches {
				text := ""
				for i := 1; i < len(match); i++ {
					if match[i] != "" {
						text += match[i]
					}
				}
				sharedStrings = append(sharedStrings, text)
			}

			return sharedStrings, nil
		}
	}
	return []string{}, nil
}

// parseWorksheet extracts data from a worksheet XML
func (sp *SpreadsheetParser) parseWorksheet(f *zip.File, sharedStrings []string) (string, error) {
	rc, err := f.Open()
	if err != nil {
		return "", err
	}
	defer rc.Close()

	content, err := io.ReadAll(rc)
	if err != nil {
		return "", err
	}

	var result strings.Builder

	// Extract cell values
	cellPattern := `<c[^>]*r="([A-Z]+\d+)"[^>]*><v>([^<]*)</v></c>`
	re := regexp.MustCompile(cellPattern)
	matches := re.FindAllStringSubmatch(string(content), -1)

	// Group cells by row
	rows := make(map[int]map[string]string)
	for _, match := range matches {
		if len(match) >= 3 {
			cellRef := match[1]
			value := match[2]

			row, col := sp.parseCellReference(cellRef)
			if row > 0 {
				if rows[row] == nil {
					rows[row] = make(map[string]string)
				}

				// Check if value is a shared string reference
				if idx, err := strconv.Atoi(value); err == nil && idx < len(sharedStrings) {
					rows[row][col] = sharedStrings[idx]
				} else {
					rows[row][col] = value
				}
			}
		}
	}

	// Output rows in order
	var rowNumbers []int
	for rowNum := range rows {
		rowNumbers = append(rowNumbers, rowNum)
	}

	// Simple sort
	for i := 0; i < len(rowNumbers); i++ {
		for j := i + 1; j < len(rowNumbers); j++ {
			if rowNumbers[i] > rowNumbers[j] {
				rowNumbers[i], rowNumbers[j] = rowNumbers[j], rowNumbers[i]
			}
		}
	}

	for _, rowNum := range rowNumbers {
		row := rows[rowNum]
		var cellValues []string

		// Get column order (A, B, C, etc.)
		cols := []string{"A", "B", "C", "D", "E", "F", "G", "H", "I", "J", "K", "L", "M", "N", "O", "P", "Q", "R", "S", "T", "U", "V", "W", "X", "Y", "Z"}
		for _, col := range cols {
			if value, exists := row[col]; exists && value != "" {
				cellValues = append(cellValues, value)
			}
		}

		if len(cellValues) > 0 {
			result.WriteString(fmt.Sprintf("Row %d: %s\n", rowNum, strings.Join(cellValues, " | ")))
		}
	}

	return result.String(), nil
}

// parseCellReference converts cell reference like "A1" to row/column
func (sp *SpreadsheetParser) parseCellReference(cellRef string) (int, string) {
	re := regexp.MustCompile(`([A-Z]+)(\d+)`)
	matches := re.FindStringSubmatch(cellRef)
	if len(matches) >= 3 {
		col := matches[1]
		if row, err := strconv.Atoi(matches[2]); err == nil {
			return row, col
		}
	}
	return 0, ""
}

// extractSheetName extracts sheet name from filename
func (sp *SpreadsheetParser) extractSheetName(filename string) string {
	re := regexp.MustCompile(`sheet(\d+)\.xml`)
	matches := re.FindStringSubmatch(filename)
	if len(matches) > 1 {
		return matches[1]
	}
	return "unknown"
}

// extractTextFromXML extracts text content from XML files using the specified tag
func (sp *SpreadsheetParser) extractTextFromXML(f *zip.File, textTag string) (string, error) {
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

// Supports checks if this parser can handle the given file extension
func (sp *SpreadsheetParser) Supports(extension string) bool {
	supported := []string{".xlsx", ".ods"}
	for _, ext := range supported {
		if ext == extension {
			return true
		}
	}
	return false
}

// GetMetadata extracts metadata from spreadsheet files
func (sp *SpreadsheetParser) GetMetadata() (map[string]string, error) {
	ext := strings.ToLower(filepath.Ext(sp.filePath))
	if ext != ".xlsx" {
		return map[string]string{"type": sp.spreadsheetType}, nil
	}

	r, err := zip.OpenReader(sp.filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to open spreadsheet: %w", err)
	}
	defer r.Close()

	metadata := make(map[string]string)
	metadata["type"] = sp.spreadsheetType

	// Count worksheets
	worksheetCount := 0
	for _, f := range r.File {
		if strings.HasPrefix(f.Name, "xl/worksheets/sheet") && strings.HasSuffix(f.Name, ".xml") {
			worksheetCount++
		}
	}
	metadata["worksheets"] = fmt.Sprintf("%d", worksheetCount)

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
			if title := sp.extractMetadataField(string(content), "dc:title"); title != "" {
				metadata["title"] = title
			}
			if creator := sp.extractMetadataField(string(content), "dc:creator"); creator != "" {
				metadata["creator"] = creator
			}
			if created := sp.extractMetadataField(string(content), "dcterms:created"); created != "" {
				metadata["created"] = created
			}
			if modified := sp.extractMetadataField(string(content), "dcterms:modified"); modified != "" {
				metadata["modified"] = modified
			}
		}
	}

	return metadata, nil
}

// extractMetadataField extracts a specific metadata field from XML content
func (sp *SpreadsheetParser) extractMetadataField(content, field string) string {
	pattern := fmt.Sprintf("<%s[^>]*>([^<]*)</%s>", field, field)
	re := regexp.MustCompile(pattern)
	matches := re.FindStringSubmatch(content)
	if len(matches) > 1 {
		return strings.TrimSpace(matches[1])
	}
	return ""
}

// CSVAdvancedParser handles CSV files with advanced parsing capabilities
type CSVAdvancedParser struct {
	delimiter rune
}

// NewCSVAdvancedParser creates a new advanced CSV parser
func NewCSVAdvancedParser(delimiter rune) *CSVAdvancedParser {
	if delimiter == 0 {
		delimiter = ','
	}
	return &CSVAdvancedParser{delimiter: delimiter}
}

// Parse extracts structured data from CSV files
func (cp *CSVAdvancedParser) Parse(content io.Reader) (string, error) {
	reader := csv.NewReader(content)
	reader.Comma = cp.delimiter
	reader.FieldsPerRecord = -1 // Allow variable number of fields

	records, err := reader.ReadAll()
	if err != nil {
		return "", fmt.Errorf("failed to parse CSV: %w", err)
	}

	var result strings.Builder
	result.WriteString("// Document Type: CSV/TSV Data\n")
	result.WriteString(fmt.Sprintf("// Total Rows: %d\n", len(records)))

	if len(records) > 0 {
		result.WriteString(fmt.Sprintf("// Columns: %d\n", len(records[0])))
		result.WriteString("\n// Data Content:\n")

		// Detect if first row is header
		hasHeader := cp.detectHeader(records)
		startRow := 0

		if hasHeader && len(records) > 0 {
			result.WriteString("// Headers: ")
			result.WriteString(strings.Join(records[0], " | "))
			result.WriteString("\n\n")
			startRow = 1
		}

		// Output sample rows (first 10)
		maxRows := 10
		if len(records)-startRow < maxRows {
			maxRows = len(records) - startRow
		}

		for i := startRow; i < startRow+maxRows; i++ {
			if i < len(records) {
				result.WriteString(fmt.Sprintf("Row %d: %s\n", i+1-startRow, strings.Join(records[i], " | ")))
			}
		}

		if len(records)-startRow > maxRows {
			result.WriteString(fmt.Sprintf("... (%d more rows)\n", len(records)-startRow-maxRows))
		}
	}

	return result.String(), nil
}

// detectHeader attempts to detect if the first row contains headers
func (cp *CSVAdvancedParser) detectHeader(records [][]string) bool {
	if len(records) < 2 {
		return false
	}

	firstRow := records[0]

	// Simple heuristic: check if first row has typical header patterns
	// and differs significantly from data rows
	textCount := 0
	for _, field := range firstRow {
		field = strings.TrimSpace(field)
		// Check if field looks like a header (no pure numbers, contains letters)
		if field != "" && strings.ContainsAny(field, "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ") {
			if _, err := strconv.ParseFloat(field, 64); err != nil {
				textCount++
			}
		}
	}

	// If most of the first row consists of text fields, likely headers
	return textCount >= len(firstRow)/2 && textCount > 0
}

// Supports checks if this parser can handle CSV/TSV files
func (cp *CSVAdvancedParser) Supports(extension string) bool {
	return extension == ".csv" || extension == ".tsv"
}

// TSVParser is a convenience wrapper for TSV files
type TSVParser struct {
	*CSVAdvancedParser
}

// NewTSVParser creates a new TSV parser
func NewTSVParser() *TSVParser {
	return &TSVParser{
		CSVAdvancedParser: NewCSVAdvancedParser('\t'),
	}
}