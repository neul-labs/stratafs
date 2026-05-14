package search

import (
	"time"
)

// SearchMode defines different search approaches
type SearchMode string

const (
	SearchModeHybrid   SearchMode = "hybrid"   // Balanced combination of all search types
	SearchModeFullText SearchMode = "fulltext" // FTS5 full-text search only
	SearchModeVector   SearchMode = "vector"   // Vector similarity search only
	SearchModeFaceted  SearchMode = "faceted"  // Metadata-based filtering only
	SearchModeWeighted SearchMode = "weighted" // Custom weighted combination
)

// SearchRequest represents a comprehensive search query
type SearchRequest struct {
	// Query text
	Query string `json:"query"`

	// Search mode and weights
	Mode    SearchMode     `json:"mode"`
	Weights *SearchWeights `json:"weights,omitempty"`

	// Faceted filters
	Filters *SearchFilters `json:"filters,omitempty"`

	// Pagination
	Limit  int `json:"limit"`
	Offset int `json:"offset"`

	// Sorting
	SortBy    string `json:"sort_by"`    // "relevance", "modified", "created", "size", "name"
	SortOrder string `json:"sort_order"` // "asc", "desc"

	// Result options
	IncludeContent   bool `json:"include_content"`
	IncludeMetadata  bool `json:"include_metadata"`
	HighlightResults bool `json:"highlight_results"`
}

// SearchWeights defines weights for different search components
type SearchWeights struct {
	FullText float64 `json:"fulltext"` // Weight for full-text search results
	Vector   float64 `json:"vector"`   // Weight for vector similarity results
	Recency  float64 `json:"recency"`  // Weight for file recency
	Filename float64 `json:"filename"` // Weight for filename matches
	FileType float64 `json:"filetype"` // Weight for file type relevance
	FileSize float64 `json:"filesize"` // Weight for file size relevance
}

// DefaultWeights returns balanced weights for hybrid search
func DefaultWeights() *SearchWeights {
	return &SearchWeights{
		FullText: 0.4,
		Vector:   0.3,
		Recency:  0.1,
		Filename: 0.1,
		FileType: 0.05,
		FileSize: 0.05,
	}
}

// SearchFilters defines faceted search filters
type SearchFilters struct {
	// File metadata filters
	FileExtensions []string `json:"file_extensions,omitempty"` // e.g., [".go", ".py"]
	FileTypes      []string `json:"file_types,omitempty"`      // e.g., ["code", "document", "text"]
	Directories    []string `json:"directories,omitempty"`     // Directory paths to search within

	// Size filters (in bytes)
	MinSize *int64 `json:"min_size,omitempty"`
	MaxSize *int64 `json:"max_size,omitempty"`

	// Date filters
	ModifiedAfter  *time.Time `json:"modified_after,omitempty"`
	ModifiedBefore *time.Time `json:"modified_before,omitempty"`
	CreatedAfter   *time.Time `json:"created_after,omitempty"`
	CreatedBefore  *time.Time `json:"created_before,omitempty"`

	// Content filters
	HasEmbeddings bool `json:"has_embeddings,omitempty"` // Only files with vector embeddings
	MinLength     *int `json:"min_length,omitempty"`     // Minimum content length
	MaxLength     *int `json:"max_length,omitempty"`     // Maximum content length

	// Language/parser filters
	Languages []string `json:"languages,omitempty"` // e.g., ["go", "python", "javascript"]
}

// SearchResult represents a single search result
type SearchResult struct {
	// Core identification
	ID       int64  `json:"id"`
	FileID   int64  `json:"file_id"`
	FilePath string `json:"file_path"`
	ChunkID  *int64 `json:"chunk_id,omitempty"` // Null for file-level results

	// Content
	Content     string `json:"content,omitempty"`
	Snippet     string `json:"snippet,omitempty"`     // Highlighted snippet
	Title       string `json:"title,omitempty"`       // File title or first line
	Description string `json:"description,omitempty"` // Brief description

	// Scoring
	Score         float64 `json:"score"`                    // Overall relevance score
	FullTextScore float64 `json:"fulltext_score,omitempty"` // FTS score component
	VectorScore   float64 `json:"vector_score,omitempty"`   // Vector similarity score
	RecencyScore  float64 `json:"recency_score,omitempty"`  // Recency score component
	FilenameScore float64 `json:"filename_score,omitempty"` // Filename match score
	FileTypeScore float64 `json:"filetype_score,omitempty"` // File type relevance
	FileSizeScore float64 `json:"filesize_score,omitempty"` // File size relevance

	// Metadata
	Metadata *FileMetadata `json:"metadata,omitempty"`

	// Position information (for chunks)
	ChunkOffset *int `json:"chunk_offset,omitempty"`
	ChunkLength *int `json:"chunk_length,omitempty"`
}

// FileMetadata contains comprehensive file information
type FileMetadata struct {
	// Basic file info
	FileName  string `json:"file_name"`
	FileExt   string `json:"file_extension"`
	FileType  string `json:"file_type"` // "code", "document", "text", etc.
	Directory string `json:"directory"`
	Size      int64  `json:"size"`
	Checksum  string `json:"checksum"`

	// Timestamps
	CreatedAt  time.Time `json:"created_at"`
	ModifiedAt time.Time `json:"modified_at"`
	IndexedAt  time.Time `json:"indexed_at"`

	// Content info
	ContentLength int    `json:"content_length"`
	ChunkCount    int    `json:"chunk_count"`
	Language      string `json:"language,omitempty"`    // Programming language
	ParserType    string `json:"parser_type,omitempty"` // Parser used
	HasEmbeddings bool   `json:"has_embeddings"`

	// Statistics
	WordCount    int     `json:"word_count,omitempty"`
	LineCount    int     `json:"line_count,omitempty"`
	CommentRatio float64 `json:"comment_ratio,omitempty"` // For code files
}

// SearchResponse contains search results and metadata
type SearchResponse struct {
	// Results
	Results []SearchResult `json:"results"`
	Total   int64          `json:"total"`

	// Search metadata
	Query     string     `json:"query"`
	Mode      SearchMode `json:"mode"`
	TimeTaken string     `json:"time_taken"`

	// Facets for refining search
	Facets *SearchFacets `json:"facets,omitempty"`

	// Pagination
	Limit    int  `json:"limit"`
	Offset   int  `json:"offset"`
	HasMore  bool `json:"has_more"`
	NextPage *int `json:"next_page,omitempty"`
	PrevPage *int `json:"prev_page,omitempty"`
}

// SearchFacets provides aggregated information for filtering
type SearchFacets struct {
	// File type distribution
	FileExtensions map[string]int64 `json:"file_extensions"` // Extension -> count
	FileTypes      map[string]int64 `json:"file_types"`      // Type -> count
	Directories    map[string]int64 `json:"directories"`     // Directory -> count
	Languages      map[string]int64 `json:"languages"`       // Language -> count

	// Size distribution
	SizeRanges map[string]int64 `json:"size_ranges"` // Range -> count

	// Date distribution
	DateRanges map[string]int64 `json:"date_ranges"` // Range -> count

	// Content statistics
	AvgFileSize   float64 `json:"avg_file_size"`
	AvgContentLen float64 `json:"avg_content_length"`
	TotalFiles    int64   `json:"total_files"`
	TotalChunks   int64   `json:"total_chunks"`
}

// DocumentRequest represents a request to retrieve a full document
type DocumentRequest struct {
	FileID          int64  `json:"file_id,omitempty"`
	FilePath        string `json:"file_path,omitempty"`
	IncludeChunks   bool   `json:"include_chunks"`
	IncludeMetadata bool   `json:"include_metadata"`
	Format          string `json:"format"` // "json", "text", "markdown"
}

// DocumentResponse contains a full document
type DocumentResponse struct {
	File     *FileDocument   `json:"file"`
	Chunks   []ChunkDocument `json:"chunks,omitempty"`
	Metadata *FileMetadata   `json:"metadata,omitempty"`
}

// FileDocument represents a complete file
type FileDocument struct {
	ID       int64  `json:"id"`
	Path     string `json:"path"`
	Content  string `json:"content"`
	Checksum string `json:"checksum"`
	Size     int64  `json:"size"`
}

// ChunkDocument represents a file chunk
type ChunkDocument struct {
	ID        int64     `json:"id"`
	Content   string    `json:"content"`
	Offset    int       `json:"offset"`
	Length    int       `json:"length"`
	Embedding []float32 `json:"embedding,omitempty"`
}
