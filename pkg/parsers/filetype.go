package parsers

import (
	"io"
	"mime"
	"net/http"
	"path/filepath"
	"strings"
)

// FileType represents different file categories
type FileType int

const (
	FileTypeUnknown FileType = iota
	FileTypeText
	FileTypeCode
	FileTypeMarkdown
	FileTypeHTML
	FileTypeJSON
	FileTypeXML
	FileTypeYAML
	FileTypePDF
	FileTypeDocx
	FileTypeBinary
	FileTypeImage
	FileTypeVideo
	FileTypeAudio
	FileTypeArchive
)

// FileTypeInfo contains metadata about a file type
type FileTypeInfo struct {
	Type        FileType
	MimeType    string
	Extension   string
	IsTextBased bool
	CanExtract  bool // Whether we can extract text content
	Description string
}

// DetectFileType determines the file type from extension and content
func DetectFileType(filename string, content io.Reader) (*FileTypeInfo, error) {
	ext := strings.ToLower(filepath.Ext(filename))

	// First try extension-based detection
	if info := getFileTypeByExtension(ext); info != nil {
		return info, nil
	}

	// If no extension match, try MIME type detection
	if content != nil {
		// Read first 512 bytes for MIME detection
		buffer := make([]byte, 512)
		n, err := content.Read(buffer)
		if err != nil && err != io.EOF {
			return getDefaultFileType(ext), nil
		}

		mimeType := http.DetectContentType(buffer[:n])
		if info := getFileTypeByMime(mimeType); info != nil {
			return info, nil
		}
	}

	return getDefaultFileType(ext), nil
}

// getFileTypeByExtension returns file type info based on extension
func getFileTypeByExtension(ext string) *FileTypeInfo {
	extensionMap := map[string]*FileTypeInfo{
		// Text files
		".txt":  {FileTypeText, "text/plain", ext, true, true, "Plain text"},
		".log":  {FileTypeText, "text/plain", ext, true, true, "Log file"},
		".csv":  {FileTypeText, "text/csv", ext, true, true, "CSV file"},
		".tsv":  {FileTypeText, "text/tab-separated-values", ext, true, true, "TSV file"},
		".ini":  {FileTypeText, "text/plain", ext, true, true, "Configuration file"},
		".conf": {FileTypeText, "text/plain", ext, true, true, "Configuration file"},
		".cfg":  {FileTypeText, "text/plain", ext, true, true, "Configuration file"},

		// Markdown
		".md":       {FileTypeMarkdown, "text/markdown", ext, true, true, "Markdown"},
		".markdown": {FileTypeMarkdown, "text/markdown", ext, true, true, "Markdown"},
		".rst":      {FileTypeMarkdown, "text/x-rst", ext, true, true, "reStructuredText"},
		".adoc":     {FileTypeMarkdown, "text/asciidoc", ext, true, true, "AsciiDoc"},
		".asciidoc": {FileTypeMarkdown, "text/asciidoc", ext, true, true, "AsciiDoc"},

		// Code files
		".go":   {FileTypeCode, "text/x-go", ext, true, true, "Go source"},
		".py":   {FileTypeCode, "text/x-python", ext, true, true, "Python source"},
		".js":   {FileTypeCode, "application/javascript", ext, true, true, "JavaScript"},
		".ts":   {FileTypeCode, "application/typescript", ext, true, true, "TypeScript"},
		".java": {FileTypeCode, "text/x-java-source", ext, true, true, "Java source"},
		".cpp":  {FileTypeCode, "text/x-c++src", ext, true, true, "C++ source"},
		".c":    {FileTypeCode, "text/x-csrc", ext, true, true, "C source"},
		".h":    {FileTypeCode, "text/x-chdr", ext, true, true, "C header"},
		".hpp":  {FileTypeCode, "text/x-c++hdr", ext, true, true, "C++ header"},
		".rs":   {FileTypeCode, "text/x-rust", ext, true, true, "Rust source"},
		".php":  {FileTypeCode, "text/x-php", ext, true, true, "PHP source"},
		".rb":   {FileTypeCode, "text/x-ruby", ext, true, true, "Ruby source"},
		".sh":   {FileTypeCode, "application/x-shellscript", ext, true, true, "Shell script"},
		".bash": {FileTypeCode, "application/x-shellscript", ext, true, true, "Bash script"},
		".zsh":  {FileTypeCode, "application/x-shellscript", ext, true, true, "Zsh script"},
		".ps1":  {FileTypeCode, "text/plain", ext, true, true, "PowerShell script"},
		".sql":  {FileTypeCode, "application/sql", ext, true, true, "SQL script"},

		// Markup files
		".html": {FileTypeHTML, "text/html", ext, true, true, "HTML"},
		".htm":  {FileTypeHTML, "text/html", ext, true, true, "HTML"},
		".xml":  {FileTypeXML, "application/xml", ext, true, true, "XML"},
		".svg":  {FileTypeXML, "image/svg+xml", ext, true, true, "SVG"},

		// Data files
		".json": {FileTypeJSON, "application/json", ext, true, true, "JSON"},
		".yaml": {FileTypeYAML, "application/x-yaml", ext, true, true, "YAML"},
		".yml":  {FileTypeYAML, "application/x-yaml", ext, true, true, "YAML"},
		".toml": {FileTypeText, "application/toml", ext, true, true, "TOML"},

		// Documents
		".pdf":  {FileTypePDF, "application/pdf", ext, false, true, "PDF document"},
		".docx": {FileTypeDocx, "application/vnd.openxmlformats-officedocument.wordprocessingml.document", ext, false, true, "Word document"},
		".doc":  {FileTypeDocx, "application/msword", ext, false, false, "Word document (old format)"},

		// Binary files we should skip
		".exe":   {FileTypeBinary, "application/octet-stream", ext, false, false, "Executable"},
		".bin":   {FileTypeBinary, "application/octet-stream", ext, false, false, "Binary file"},
		".dll":   {FileTypeBinary, "application/octet-stream", ext, false, false, "Dynamic library"},
		".so":    {FileTypeBinary, "application/octet-stream", ext, false, false, "Shared library"},
		".dylib": {FileTypeBinary, "application/octet-stream", ext, false, false, "Dynamic library"},

		// Images
		".jpg":  {FileTypeImage, "image/jpeg", ext, false, false, "JPEG image"},
		".jpeg": {FileTypeImage, "image/jpeg", ext, false, false, "JPEG image"},
		".png":  {FileTypeImage, "image/png", ext, false, false, "PNG image"},
		".gif":  {FileTypeImage, "image/gif", ext, false, false, "GIF image"},
		".bmp":  {FileTypeImage, "image/bmp", ext, false, false, "BMP image"},
		".webp": {FileTypeImage, "image/webp", ext, false, false, "WebP image"},
		".ico":  {FileTypeImage, "image/x-icon", ext, false, false, "Icon"},

		// Video
		".mp4": {FileTypeVideo, "video/mp4", ext, false, false, "MP4 video"},
		".avi": {FileTypeVideo, "video/x-msvideo", ext, false, false, "AVI video"},
		".mov": {FileTypeVideo, "video/quicktime", ext, false, false, "QuickTime video"},
		".wmv": {FileTypeVideo, "video/x-ms-wmv", ext, false, false, "WMV video"},
		".mkv": {FileTypeVideo, "video/x-matroska", ext, false, false, "Matroska video"},

		// Audio
		".mp3":  {FileTypeAudio, "audio/mpeg", ext, false, false, "MP3 audio"},
		".wav":  {FileTypeAudio, "audio/wav", ext, false, false, "WAV audio"},
		".flac": {FileTypeAudio, "audio/flac", ext, false, false, "FLAC audio"},
		".ogg":  {FileTypeAudio, "audio/ogg", ext, false, false, "OGG audio"},

		// Archives
		".zip": {FileTypeArchive, "application/zip", ext, false, false, "ZIP archive"},
		".tar": {FileTypeArchive, "application/x-tar", ext, false, false, "TAR archive"},
		".gz":  {FileTypeArchive, "application/gzip", ext, false, false, "Gzip archive"},
		".bz2": {FileTypeArchive, "application/x-bzip2", ext, false, false, "Bzip2 archive"},
		".xz":  {FileTypeArchive, "application/x-xz", ext, false, false, "XZ archive"},
		".7z":  {FileTypeArchive, "application/x-7z-compressed", ext, false, false, "7-Zip archive"},
		".rar": {FileTypeArchive, "application/vnd.rar", ext, false, false, "RAR archive"},
	}

	return extensionMap[ext]
}

// getFileTypeByMime returns file type info based on MIME type
func getFileTypeByMime(mimeType string) *FileTypeInfo {
	// Parse MIME type to get main type
	mainType, _, err := mime.ParseMediaType(mimeType)
	if err != nil {
		return nil
	}

	switch {
	case strings.HasPrefix(mainType, "text/"):
		return &FileTypeInfo{FileTypeText, mimeType, "", true, true, "Text file"}
	case strings.HasPrefix(mainType, "application/json"):
		return &FileTypeInfo{FileTypeJSON, mimeType, "", true, true, "JSON file"}
	case strings.HasPrefix(mainType, "application/xml") || strings.HasPrefix(mainType, "text/xml"):
		return &FileTypeInfo{FileTypeXML, mimeType, "", true, true, "XML file"}
	case strings.HasPrefix(mainType, "text/html"):
		return &FileTypeInfo{FileTypeHTML, mimeType, "", true, true, "HTML file"}
	case strings.HasPrefix(mainType, "application/pdf"):
		return &FileTypeInfo{FileTypePDF, mimeType, "", false, true, "PDF document"}
	case strings.HasPrefix(mainType, "image/"):
		return &FileTypeInfo{FileTypeImage, mimeType, "", false, false, "Image file"}
	case strings.HasPrefix(mainType, "video/"):
		return &FileTypeInfo{FileTypeVideo, mimeType, "", false, false, "Video file"}
	case strings.HasPrefix(mainType, "audio/"):
		return &FileTypeInfo{FileTypeAudio, mimeType, "", false, false, "Audio file"}
	default:
		return &FileTypeInfo{FileTypeBinary, mimeType, "", false, false, "Binary file"}
	}
}

// getDefaultFileType returns a default file type for unknown extensions
func getDefaultFileType(ext string) *FileTypeInfo {
	return &FileTypeInfo{
		Type:        FileTypeUnknown,
		MimeType:    "application/octet-stream",
		Extension:   ext,
		IsTextBased: false,
		CanExtract:  false,
		Description: "Unknown file type",
	}
}

// ShouldIndex returns true if the file type should be indexed
func (info *FileTypeInfo) ShouldIndex() bool {
	return info.CanExtract && (info.IsTextBased || info.Type == FileTypePDF || info.Type == FileTypeDocx)
}

// IsSupported returns true if we can process this file type
func (info *FileTypeInfo) IsSupported() bool {
	switch info.Type {
	case FileTypeText, FileTypeCode, FileTypeMarkdown, FileTypeHTML,
		FileTypeJSON, FileTypeXML, FileTypeYAML, FileTypePDF:
		return true
	default:
		return false
	}
}
