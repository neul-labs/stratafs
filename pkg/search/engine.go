package search

import (
	"fmt"
	"math"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/neul-labs/stratafs/pkg/database"
	"github.com/neul-labs/stratafs/pkg/embeddings"
)

// Engine provides hybrid search capabilities
type Engine struct {
	databases     map[string]*database.DB
	vectorIndexes map[string]*VectorIndex // Per-directory vector indexes
	embedder      *embeddings.Embedder
	ftsAvailable  map[string]bool // Track FTS5 availability per database
}

// NewEngine creates a new search engine with per-directory vector indexes in shared databases
func NewEngine(databases map[string]*database.DB, embedder *embeddings.Embedder) (*Engine, error) {
	vectorIndexes := make(map[string]*VectorIndex)
	ftsAvailable := make(map[string]bool)

	// Get embedding dimensions from the embedder
	dimensions := embedder.GetDimension()
	fmt.Printf("Using %d-dimensional embeddings from model: %s\n", dimensions, embedder.GetModelName())

	// Create vector index for each directory using the shared database with correct dimensions
	for dirPath, db := range databases {
		vectorIndex, err := NewVectorIndex(db.GetConn(), dimensions)
		if err != nil {
			return nil, fmt.Errorf("failed to create vector index for %s: %w", dirPath, err)
		}
		vectorIndexes[dirPath] = vectorIndex

		// Check FTS5 availability for this database
		_, err = db.GetConn().Exec(`CREATE VIRTUAL TABLE IF NOT EXISTS fts_test USING fts5(test)`)
		if err != nil && strings.Contains(err.Error(), "no such module") {
			ftsAvailable[dirPath] = false
		} else {
			ftsAvailable[dirPath] = true
			// Clean up test table
			_, _ = db.GetConn().Exec(`DROP TABLE IF EXISTS fts_test`)
		}
	}

	return &Engine{
		databases:     databases,
		vectorIndexes: vectorIndexes,
		embedder:      embedder,
		ftsAvailable:  ftsAvailable,
	}, nil
}

// Search performs hybrid search based on the request
func (e *Engine) Search(req *SearchRequest) (*SearchResponse, error) {
	startTime := time.Now()

	// Validate and set defaults
	if err := e.validateRequest(req); err != nil {
		return nil, err
	}

	var results []SearchResult
	var total int64
	var err error

	// Perform search based on mode
	switch req.Mode {
	case SearchModeFullText:
		results, total, err = e.searchFullText(req)
	case SearchModeVector:
		results, total, err = e.searchVectorUnified(req)
	case SearchModeFaceted:
		results, total, err = e.searchFaceted(req)
	case SearchModeWeighted:
		results, total, err = e.searchWeightedUnified(req)
	case SearchModeHybrid:
		fallthrough
	default:
		results, total, err = e.searchHybridUnified(req)
	}

	if err != nil {
		return nil, err
	}

	// Apply sorting
	e.sortResults(results, req.SortBy, req.SortOrder)

	// Apply pagination
	paginatedResults := e.paginateResults(results, req.Offset, req.Limit)

	// Generate facets if requested
	var facets *SearchFacets
	if req.Mode == SearchModeHybrid || req.Mode == SearchModeFaceted {
		facets = e.generateFacets(results)
	}

	// Calculate pagination info
	hasMore := int64(req.Offset+len(paginatedResults)) < total
	var nextPage, prevPage *int
	if hasMore {
		next := (req.Offset / req.Limit) + 1
		nextPage = &next
	}
	if req.Offset > 0 {
		prev := (req.Offset / req.Limit) - 1
		prevPage = &prev
	}

	return &SearchResponse{
		Results:   paginatedResults,
		Total:     total,
		Query:     req.Query,
		Mode:      req.Mode,
		TimeTaken: time.Since(startTime).String(),
		Facets:    facets,
		Limit:     req.Limit,
		Offset:    req.Offset,
		HasMore:   hasMore,
		NextPage:  nextPage,
		PrevPage:  prevPage,
	}, nil
}

// searchFullText performs FTS5 full-text search
func (e *Engine) searchFullText(req *SearchRequest) ([]SearchResult, int64, error) {
	var allResults []SearchResult
	var totalCount int64

	for dirPath, db := range e.databases {
		// Apply directory filter if specified
		if req.Filters != nil && len(req.Filters.Directories) > 0 {
			found := false
			for _, filterDir := range req.Filters.Directories {
				if strings.HasPrefix(dirPath, filterDir) {
					found = true
					break
				}
			}
			if !found {
				continue
			}
		}

		// Search in this database
		chunks, err := db.SearchChunks(req.Query, req.Limit*2) // Get more for better merging
		if err != nil {
			continue // Skip databases with errors
		}

		for _, chunk := range chunks {
			result := SearchResult{
				ID:      chunk.ID,
				FileID:  chunk.FileID,
				ChunkID: &chunk.ID,
				Content: chunk.Content,
				Score:   1.0, // Will be calculated based on FTS score
			}

			// Get file metadata
			if file, err := db.GetFileByID(chunk.FileID); err == nil {
				result.FilePath = file.Path
				result.Metadata = e.buildFileMetadata(file, dirPath)
			}

			allResults = append(allResults, result)
		}

		totalCount += int64(len(chunks))
	}

	return allResults, totalCount, nil
}

// searchFaceted performs metadata-based filtering
func (e *Engine) searchFaceted(req *SearchRequest) ([]SearchResult, int64, error) {
	var allResults []SearchResult
	var totalCount int64

	for dirPath, db := range e.databases {
		// Get all files and apply filters
		files, err := db.GetAllFiles()
		if err != nil {
			continue
		}

		for _, file := range files {
			// Apply filters
			if !e.matchesFilters(file, dirPath, req.Filters) {
				continue
			}

			result := SearchResult{
				ID:       file.ID,
				FileID:   file.ID,
				FilePath: file.Path,
				Score:    1.0,
				Metadata: e.buildFileMetadata(file, dirPath),
			}

			// Get file content if requested
			if req.IncludeContent {
				chunks, err := db.GetChunksByFileID(file.ID)
				if err == nil && len(chunks) > 0 {
					// Combine chunk content
					var contentBuilder strings.Builder
					for _, chunk := range chunks {
						contentBuilder.WriteString(chunk.Content)
						contentBuilder.WriteString("\n")
					}
					result.Content = contentBuilder.String()
				}
			}

			allResults = append(allResults, result)
		}

		totalCount += int64(len(allResults))
	}

	return allResults, totalCount, nil
}

// GetDocument retrieves a complete document
func (e *Engine) GetDocument(req *DocumentRequest) (*DocumentResponse, error) {
	for _, db := range e.databases {
		var file *database.File
		var err error

		// Get file by ID or path
		if req.FileID > 0 {
			file, err = db.GetFileByID(req.FileID)
		} else if req.FilePath != "" {
			file, err = db.GetFileByPath(req.FilePath)
		} else {
			return nil, fmt.Errorf("either file_id or file_path must be specified")
		}

		if err != nil {
			continue // Try next database
		}

		// Build response
		response := &DocumentResponse{
			File: &FileDocument{
				ID:       file.ID,
				Path:     file.Path,
				Checksum: file.Checksum,
				Size:     file.Size,
			},
		}

		// Add metadata if requested
		if req.IncludeMetadata {
			response.Metadata = e.buildFileMetadata(file, "")
		}

		// Add chunks if requested
		if req.IncludeChunks {
			chunks, err := db.GetChunksByFileID(file.ID)
			if err == nil {
				response.Chunks = make([]ChunkDocument, len(chunks))
				for i, chunk := range chunks {
					response.Chunks[i] = ChunkDocument{
						ID:      chunk.ID,
						Content: chunk.Content,
						Offset:  chunk.Offset,
						Length:  chunk.Length,
					}
				}
			}

			// Combine content for file
			var contentBuilder strings.Builder
			for _, chunk := range chunks {
				contentBuilder.WriteString(chunk.Content)
			}
			response.File.Content = contentBuilder.String()
		}

		return response, nil
	}

	return nil, fmt.Errorf("document not found")
}

// Helper methods

func (e *Engine) validateRequest(req *SearchRequest) error {
	if req.Limit <= 0 {
		req.Limit = 10
	}
	if req.Limit > 100 {
		req.Limit = 100
	}
	if req.Offset < 0 {
		req.Offset = 0
	}
	if req.Mode == "" {
		req.Mode = SearchModeHybrid
	}
	if req.SortBy == "" {
		req.SortBy = "relevance"
	}
	if req.SortOrder == "" {
		req.SortOrder = "desc"
	}
	return nil
}

func (e *Engine) buildFileMetadata(file *database.File, directory string) *FileMetadata {
	ext := filepath.Ext(file.Path)
	return &FileMetadata{
		FileName:      filepath.Base(file.Path),
		FileExt:       ext,
		Directory:     directory,
		Size:          file.Size,
		Checksum:      file.Checksum,
		CreatedAt:     file.CreatedAt,
		ModifiedAt:    file.UpdatedAt,
		IndexedAt:     file.CreatedAt,
		HasEmbeddings: true, // Assume true if in database
	}
}

func (e *Engine) matchesFilters(file *database.File, directory string, filters *SearchFilters) bool {
	if filters == nil {
		return true
	}

	// File extension filter
	if len(filters.FileExtensions) > 0 {
		ext := filepath.Ext(file.Path)
		found := false
		for _, filterExt := range filters.FileExtensions {
			if ext == filterExt {
				found = true
				break
			}
		}
		if !found {
			return false
		}
	}

	// Directory filter
	if len(filters.Directories) > 0 {
		found := false
		for _, filterDir := range filters.Directories {
			if strings.HasPrefix(directory, filterDir) {
				found = true
				break
			}
		}
		if !found {
			return false
		}
	}

	// Size filters
	if filters.MinSize != nil && file.Size < *filters.MinSize {
		return false
	}
	if filters.MaxSize != nil && file.Size > *filters.MaxSize {
		return false
	}

	// Date filters
	if filters.ModifiedAfter != nil && file.UpdatedAt.Before(*filters.ModifiedAfter) {
		return false
	}
	if filters.ModifiedBefore != nil && file.UpdatedAt.After(*filters.ModifiedBefore) {
		return false
	}

	return true
}

func (e *Engine) sortResults(results []SearchResult, sortBy, sortOrder string) {
	sort.Slice(results, func(i, j int) bool {
		var less bool

		switch sortBy {
		case "relevance":
			less = results[i].Score > results[j].Score // Higher score first
		case "modified":
			if results[i].Metadata != nil && results[j].Metadata != nil {
				less = results[i].Metadata.ModifiedAt.After(results[j].Metadata.ModifiedAt)
			}
		case "size":
			if results[i].Metadata != nil && results[j].Metadata != nil {
				less = results[i].Metadata.Size > results[j].Metadata.Size
			}
		case "name":
			if results[i].Metadata != nil && results[j].Metadata != nil {
				less = results[i].Metadata.FileName < results[j].Metadata.FileName
			}
		default:
			less = results[i].Score > results[j].Score
		}

		if sortOrder == "asc" {
			return !less
		}
		return less
	})
}

func (e *Engine) paginateResults(results []SearchResult, offset, limit int) []SearchResult {
	if offset >= len(results) {
		return []SearchResult{}
	}

	end := offset + limit
	if end > len(results) {
		end = len(results)
	}

	return results[offset:end]
}

func (e *Engine) generateFacets(results []SearchResult) *SearchFacets {
	facets := &SearchFacets{
		FileExtensions: make(map[string]int64),
		FileTypes:      make(map[string]int64),
		Directories:    make(map[string]int64),
		Languages:      make(map[string]int64),
		SizeRanges:     make(map[string]int64),
		DateRanges:     make(map[string]int64),
	}

	for _, result := range results {
		if result.Metadata == nil {
			continue
		}

		// Count file extensions
		if result.Metadata.FileExt != "" {
			facets.FileExtensions[result.Metadata.FileExt]++
		}

		// Count directories
		if result.Metadata.Directory != "" {
			facets.Directories[result.Metadata.Directory]++
		}

		// Size ranges
		size := result.Metadata.Size
		var sizeRange string
		switch {
		case size < 1024:
			sizeRange = "< 1KB"
		case size < 1024*1024:
			sizeRange = "1KB - 1MB"
		case size < 1024*1024*10:
			sizeRange = "1MB - 10MB"
		default:
			sizeRange = "> 10MB"
		}
		facets.SizeRanges[sizeRange]++

		facets.TotalFiles++
	}

	return facets
}

// AddToVectorIndex adds a chunk to the appropriate directory's vector index
func (e *Engine) AddToVectorIndex(dirPath string, chunkID int64, embedding []float32) error {
	if vectorIndex, exists := e.vectorIndexes[dirPath]; exists {
		return vectorIndex.AddVector(chunkID, embedding)
	}
	return fmt.Errorf("vector index not found for directory: %s", dirPath)
}

// RemoveFromVectorIndex removes a chunk from the appropriate directory's vector index
func (e *Engine) RemoveFromVectorIndex(dirPath string, chunkID int64) error {
	if vectorIndex, exists := e.vectorIndexes[dirPath]; exists {
		return vectorIndex.RemoveVector(chunkID)
	}
	return fmt.Errorf("vector index not found for directory: %s", dirPath)
}

// SaveVectorIndex saves a specific directory's vector index to disk
func (e *Engine) SaveVectorIndex(dirPath string) error {
	if vectorIndex, exists := e.vectorIndexes[dirPath]; exists {
		return vectorIndex.Save()
	}
	return fmt.Errorf("vector index not found for directory: %s", dirPath)
}

// SaveAllVectorIndexes saves all vector indexes to disk
func (e *Engine) SaveAllVectorIndexes() error {
	for dirPath, vectorIndex := range e.vectorIndexes {
		if err := vectorIndex.Save(); err != nil {
			return fmt.Errorf("failed to save vector index for %s: %w", dirPath, err)
		}
	}
	return nil
}

// Close closes all vector indexes in the search engine
func (e *Engine) Close() error {
	for dirPath, vectorIndex := range e.vectorIndexes {
		if err := vectorIndex.Close(); err != nil {
			return fmt.Errorf("failed to close vector index for %s: %w", dirPath, err)
		}
	}
	return nil
}

// searchHybridUnified performs unified hybrid search using single SQL queries
func (e *Engine) searchHybridUnified(req *SearchRequest) ([]SearchResult, int64, error) {
	if req.Query == "" {
		return e.searchFaceted(req) // Fallback to faceted search for no query
	}

	// Generate query embedding for vector component
	queryEmbedding, err := e.embedder.Embed(req.Query)
	if err != nil {
		// Fallback to full-text only if embedding fails
		return e.searchFullText(req)
	}

	// Default hybrid weights
	weights := req.Weights
	if weights == nil {
		weights = DefaultWeights()
	}

	var allResults []SearchResult
	var totalCount int64

	// Search across all directories with unified query
	for dirPath, db := range e.databases {
		results, count, err := e.searchHybridInDatabase(db, req, queryEmbedding, weights, dirPath)
		if err != nil {
			fmt.Printf("Warning: hybrid search failed for %s: %v\n", dirPath, err)
			continue
		}
		allResults = append(allResults, results...)
		totalCount += count
	}

	// Sort by combined score and limit results
	sort.Slice(allResults, func(i, j int) bool {
		return allResults[i].Score > allResults[j].Score
	})

	if len(allResults) > req.Limit {
		allResults = allResults[:req.Limit]
	}

	return allResults, totalCount, nil
}

// searchHybridInDatabase performs hybrid search within a single database using unified SQL
func (e *Engine) searchHybridInDatabase(db *database.DB, req *SearchRequest, queryEmbedding []float32, weights *SearchWeights, dirPath string) ([]SearchResult, int64, error) {
	// Convert query vector to bytes for sqlite-vec
	queryBytes := make([]byte, len(queryEmbedding)*4)
	for i, v := range queryEmbedding {
		bits := math.Float32bits(v)
		queryBytes[i*4] = byte(bits)
		queryBytes[i*4+1] = byte(bits >> 8)
		queryBytes[i*4+2] = byte(bits >> 16)
		queryBytes[i*4+3] = byte(bits >> 24)
	}

	// Unified hybrid query combining FTS5 + vector search + metadata
	query := `
	WITH fts_results AS (
		-- Full-text search results
		SELECT
			fc.id as chunk_id,
			fc.file_id,
			fc.content,
			fc.offset,
			fc.length,
			fts.rank as fts_score,
			? as fts_weight
		FROM file_chunks fc
		JOIN file_chunks_fts fts ON fc.id = fts.rowid
		WHERE fts.file_chunks_fts MATCH ?
		AND fc.deleted_at IS NULL
		ORDER BY fts.rank
		LIMIT ?
	),
	vector_results AS (
		-- Vector similarity search results
		SELECT
			vc.rowid as chunk_id,
			fc.file_id,
			fc.content,
			fc.offset,
			fc.length,
			(1.0 / (1.0 + vc.distance)) as vector_score,
			? as vector_weight
		FROM vec_chunks vc
		JOIN file_chunks fc ON vc.rowid = fc.id
		WHERE vc.embedding MATCH vec_f32(?) AND k = ?
		AND fc.deleted_at IS NULL
		ORDER BY vc.distance
	),
	combined_results AS (
		-- Combine and score results
		SELECT
			chunk_id,
			file_id,
			content,
			offset,
			length,
			COALESCE(fts.fts_score * fts.fts_weight, 0) +
			COALESCE(vec.vector_score * vec.vector_weight, 0) as combined_score,
			fts.fts_score,
			vec.vector_score
		FROM (
			SELECT DISTINCT chunk_id FROM fts_results
			UNION
			SELECT DISTINCT chunk_id FROM vector_results
		) all_chunks
		LEFT JOIN fts_results fts ON all_chunks.chunk_id = fts.chunk_id
		LEFT JOIN vector_results vec ON all_chunks.chunk_id = vec.chunk_id
		LEFT JOIN file_chunks fc ON all_chunks.chunk_id = fc.id
	)
	SELECT
		cr.chunk_id,
		cr.file_id,
		cr.content,
		cr.combined_score,
		COALESCE(cr.fts_score, 0) as fulltext_score,
		COALESCE(cr.vector_score, 0) as vector_score,
		f.path,
		f.size,
		f.created_at,
		f.updated_at
	FROM combined_results cr
	JOIN files f ON cr.file_id = f.id
	WHERE f.deleted_at IS NULL
	ORDER BY cr.combined_score DESC
	LIMIT ?
	`

	// Execute unified hybrid query
	rows, err := db.GetConn().Query(query,
		weights.FullText, // fts_weight
		req.Query,        // FTS query
		req.Limit*2,      // FTS limit
		weights.Vector,   // vector_weight
		queryBytes,       // vector query
		req.Limit*2,      // vector k parameter
		req.Limit,        // final limit
	)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to execute hybrid query: %w", err)
	}
	defer rows.Close()

	var results []SearchResult
	for rows.Next() {
		var result SearchResult
		var filePath string
		var fileSize int64
		var createdAt, updatedAt time.Time

		err := rows.Scan(
			&result.ChunkID,
			&result.FileID,
			&result.Content,
			&result.Score,
			&result.FullTextScore,
			&result.VectorScore,
			&filePath,
			&fileSize,
			&createdAt,
			&updatedAt,
		)
		if err != nil {
			continue
		}

		result.ID = *result.ChunkID
		result.FilePath = filePath

		// Build metadata if requested
		if req.IncludeMetadata {
			result.Metadata = &FileMetadata{
				FileName:      filepath.Base(filePath),
				FileExt:       filepath.Ext(filePath),
				Directory:     dirPath,
				Size:          fileSize,
				CreatedAt:     createdAt,
				ModifiedAt:    updatedAt,
				IndexedAt:     updatedAt,
				ContentLength: len(result.Content),
			}
		}

		// Create snippet if not including full content
		if !req.IncludeContent {
			result.Snippet = e.createSnippet(result.Content, req.Query, 200)
			result.Content = ""
		}

		results = append(results, result)
	}

	return results, int64(len(results)), nil
}

// searchVectorUnified performs vector-only search using sqlite-vec
func (e *Engine) searchVectorUnified(req *SearchRequest) ([]SearchResult, int64, error) {
	if req.Query == "" {
		return nil, 0, fmt.Errorf("vector search requires a query")
	}

	// Generate query embedding
	queryEmbedding, err := e.embedder.Embed(req.Query)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to generate query embedding: %w", err)
	}

	var allResults []SearchResult

	// Search each directory's database
	for dirPath, db := range e.databases {
		results, err := e.searchVectorInDatabase(db, queryEmbedding, req, dirPath)
		if err != nil {
			fmt.Printf("Warning: vector search failed for %s: %v\n", dirPath, err)
			continue
		}
		allResults = append(allResults, results...)
	}

	// Sort by vector score and limit results
	sort.Slice(allResults, func(i, j int) bool {
		return allResults[i].VectorScore > allResults[j].VectorScore
	})

	if len(allResults) > req.Limit {
		allResults = allResults[:req.Limit]
	}

	return allResults, int64(len(allResults)), nil
}

// searchVectorInDatabase performs vector search in a single database
func (e *Engine) searchVectorInDatabase(db *database.DB, queryEmbedding []float32, req *SearchRequest, dirPath string) ([]SearchResult, error) {
	// Convert query vector to bytes
	queryBytes := make([]byte, len(queryEmbedding)*4)
	for i, v := range queryEmbedding {
		bits := math.Float32bits(v)
		queryBytes[i*4] = byte(bits)
		queryBytes[i*4+1] = byte(bits >> 8)
		queryBytes[i*4+2] = byte(bits >> 16)
		queryBytes[i*4+3] = byte(bits >> 24)
	}

	// Vector search query
	query := `
	SELECT
		vc.rowid as chunk_id,
		fc.file_id,
		fc.content,
		fc.offset,
		fc.length,
		(1.0 / (1.0 + vc.distance)) as vector_score,
		f.path,
		f.size,
		f.created_at,
		f.updated_at
	FROM vec_chunks vc
	JOIN file_chunks fc ON vc.rowid = fc.id
	JOIN files f ON fc.file_id = f.id
	WHERE vc.embedding MATCH vec_f32(?) AND k = ?
	AND fc.deleted_at IS NULL
	AND f.deleted_at IS NULL
	ORDER BY vc.distance
	LIMIT ?
	`

	rows, err := db.GetConn().Query(query, queryBytes, req.Limit, req.Limit)
	if err != nil {
		return nil, fmt.Errorf("failed to execute vector query: %w", err)
	}
	defer rows.Close()

	var results []SearchResult
	for rows.Next() {
		var result SearchResult
		var filePath string
		var fileSize int64
		var createdAt, updatedAt time.Time

		err := rows.Scan(
			&result.ChunkID,
			&result.FileID,
			&result.Content,
			&result.ChunkOffset,
			&result.ChunkLength,
			&result.VectorScore,
			&filePath,
			&fileSize,
			&createdAt,
			&updatedAt,
		)
		if err != nil {
			continue
		}

		result.ID = *result.ChunkID
		result.FilePath = filePath
		result.Score = result.VectorScore

		// Build metadata if requested
		if req.IncludeMetadata {
			result.Metadata = &FileMetadata{
				FileName:      filepath.Base(filePath),
				FileExt:       filepath.Ext(filePath),
				Directory:     dirPath,
				Size:          fileSize,
				CreatedAt:     createdAt,
				ModifiedAt:    updatedAt,
				IndexedAt:     updatedAt,
				ContentLength: len(result.Content),
			}
		}

		// Create snippet if not including full content
		if !req.IncludeContent {
			result.Snippet = e.createSnippet(result.Content, req.Query, 200)
			result.Content = ""
		}

		results = append(results, result)
	}

	return results, nil
}

// searchWeightedUnified performs weighted search with custom weights
func (e *Engine) searchWeightedUnified(req *SearchRequest) ([]SearchResult, int64, error) {
	// Use custom weights if provided, otherwise use defaults
	if req.Weights == nil {
		req.Weights = DefaultWeights()
	}

	// Weighted search is essentially hybrid search with custom weights
	return e.searchHybridUnified(req)
}

// createSnippet creates a highlighted snippet around the query terms
func (e *Engine) createSnippet(content, query string, maxLength int) string {
	if len(content) <= maxLength {
		return content
	}

	// Simple snippet creation - find first occurrence of query term
	queryLower := strings.ToLower(query)
	contentLower := strings.ToLower(content)

	index := strings.Index(contentLower, queryLower)
	if index == -1 {
		// Query not found, return beginning of content
		if len(content) > maxLength {
			return content[:maxLength] + "..."
		}
		return content
	}

	// Create snippet around the found term
	start := index - maxLength/4
	if start < 0 {
		start = 0
	}

	end := start + maxLength
	if end > len(content) {
		end = len(content)
		start = end - maxLength
		if start < 0 {
			start = 0
		}
	}

	snippet := content[start:end]
	if start > 0 {
		snippet = "..." + snippet
	}
	if end < len(content) {
		snippet = snippet + "..."
	}

	return snippet
}
