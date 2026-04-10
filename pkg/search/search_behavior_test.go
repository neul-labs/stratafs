package search

import (
	"path/filepath"
	"testing"
	"time"

	"agentfs/pkg/config"
	"agentfs/pkg/database"
	"agentfs/pkg/embeddings"
)

func TestSearchModes(t *testing.T) {
	engine, db, embedder := setupSearchTestEngine(t)
	defer cleanup(engine, db, embedder)

	// Create test data
	setupSearchTestData(t, db, embedder)

	testCases := []struct {
		name     string
		mode     SearchMode
		query    string
		expected bool // whether we expect results
	}{
		{"Full-text search", SearchModeFullText, "golang", true},
		{"Vector search", SearchModeVector, "programming", true},
		{"Hybrid search", SearchModeHybrid, "golang programming", true},
		{"Faceted search", SearchModeFaceted, "", false}, // No filters applied
		{"Weighted search", SearchModeWeighted, "golang", true},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			req := &SearchRequest{
				Query: tc.query,
				Mode:  tc.mode,
				Limit: 10,
			}

			if tc.mode == SearchModeWeighted {
				req.Weights = &SearchWeights{
					FullText: 0.7,
					Vector:   0.3,
				}
			}

			response, err := engine.Search(req)
			if err != nil {
				t.Fatalf("Search failed for mode %s: %v", tc.mode, err)
			}

			if response == nil {
				t.Fatalf("Response should not be nil for mode %s", tc.mode)
			}

			hasResults := len(response.Results) > 0
			if hasResults != tc.expected {
				t.Errorf("Mode %s: expected results=%v, got results=%v (count: %d)",
					tc.mode, tc.expected, hasResults, len(response.Results))
			}
		})
	}
}

func TestSearchWeights(t *testing.T) {
	engine, db, embedder := setupSearchTestEngine(t)
	defer cleanup(engine, db, embedder)

	setupSearchTestData(t, db, embedder)

	// Test default weights
	defaultWeights := DefaultWeights()
	if defaultWeights.FullText != 0.4 {
		t.Errorf("Expected default FullText weight 0.4, got %f", defaultWeights.FullText)
	}
	if defaultWeights.Vector != 0.3 {
		t.Errorf("Expected default Vector weight 0.3, got %f", defaultWeights.Vector)
	}

	// Test weighted search with custom weights
	req := &SearchRequest{
		Query: "programming",
		Mode:  SearchModeWeighted,
		Limit: 5,
		Weights: &SearchWeights{
			FullText: 0.8, // Heavy emphasis on full-text
			Vector:   0.2,
		},
	}

	response, err := engine.Search(req)
	if err != nil {
		t.Fatalf("Weighted search failed: %v", err)
	}

	if response == nil {
		t.Fatal("Response should not be nil")
	}

	// Test with vector-heavy weights
	req.Weights.FullText = 0.2
	req.Weights.Vector = 0.8

	response2, err := engine.Search(req)
	if err != nil {
		t.Fatalf("Vector-heavy search failed: %v", err)
	}

	if response2 == nil {
		t.Fatal("Vector-heavy response should not be nil")
	}
}

func TestSearchFilters(t *testing.T) {
	engine, db, embedder := setupSearchTestEngine(t)
	defer cleanup(engine, db, embedder)

	setupSearchTestData(t, db, embedder)

	testCases := []struct {
		name     string
		filters  *SearchFilters
		expected bool
	}{
		{
			"File extension filter",
			&SearchFilters{FileExtensions: []string{".go"}},
			true,
		},
		{
			"Directory filter",
			&SearchFilters{Directories: []string{"/test/dir"}},
			true,
		},
		{
			"Size filter",
			&SearchFilters{MinSize: int64Ptr(10), MaxSize: int64Ptr(10000)},
			true,
		},
		{
			"Non-matching extension",
			&SearchFilters{FileExtensions: []string{".xyz"}},
			false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			req := &SearchRequest{
				Query:   "golang",
				Mode:    SearchModeFaceted,
				Filters: tc.filters,
				Limit:   10,
			}

			response, err := engine.Search(req)
			if err != nil {
				t.Fatalf("Filtered search failed: %v", err)
			}

			hasResults := len(response.Results) > 0
			if hasResults != tc.expected {
				t.Errorf("Filter %s: expected results=%v, got results=%v",
					tc.name, tc.expected, hasResults)
			}
		})
	}
}

func TestSearchPagination(t *testing.T) {
	engine, db, embedder := setupSearchTestEngine(t)
	defer cleanup(engine, db, embedder)

	// Create multiple test documents
	for i := 0; i < 20; i++ {
		file, err := db.UpsertFile(
			filepath.Join("/test/dir", "doc"+string(rune(i+65))+".go"),
			"checksum", 1000)
		if err != nil {
			t.Fatalf("Failed to create test file %d: %v", i, err)
		}

		content := "golang programming test document"
		embedding, err := embedder.Embed(content)
		if err != nil {
			t.Fatalf("Failed to generate embedding for doc %d: %v", i, err)
		}

		_, err = db.UpsertChunk(file.ID, content, embedding, 0, len(content))
		if err != nil {
			t.Fatalf("Failed to insert chunk for doc %d: %v", i, err)
		}
	}

	// Test pagination
	req := &SearchRequest{
		Query:  "golang",
		Mode:   SearchModeFullText,
		Limit:  5,
		Offset: 0,
	}

	// First page
	response1, err := engine.Search(req)
	if err != nil {
		t.Fatalf("First page search failed: %v", err)
	}

	if len(response1.Results) > 5 {
		t.Errorf("First page: expected at most 5 results, got %d", len(response1.Results))
	}

	// Second page
	req.Offset = 5
	response2, err := engine.Search(req)
	if err != nil {
		t.Fatalf("Second page search failed: %v", err)
	}

	if len(response2.Results) > 5 {
		t.Errorf("Second page: expected at most 5 results, got %d", len(response2.Results))
	}

	// Verify total count is consistent
	if response1.Total != response2.Total {
		t.Errorf("Total count inconsistent: page1=%d, page2=%d",
			response1.Total, response2.Total)
	}
}

func TestSearchSorting(t *testing.T) {
	engine, db, embedder := setupSearchTestEngine(t)
	defer cleanup(engine, db, embedder)

	setupSearchTestData(t, db, embedder)

	testCases := []struct {
		sortBy    string
		sortOrder string
		valid     bool
	}{
		{"relevance", "desc", true},
		{"modified", "desc", true},
		{"created", "asc", true},
		{"size", "desc", true},
		{"name", "asc", true},
		{"invalid", "desc", false},
		{"relevance", "invalid", false},
	}

	for _, tc := range testCases {
		t.Run(tc.sortBy+"_"+tc.sortOrder, func(t *testing.T) {
			req := &SearchRequest{
				Query:     "golang",
				Mode:      SearchModeFullText,
				Limit:     5,
				SortBy:    tc.sortBy,
				SortOrder: tc.sortOrder,
			}

			response, err := engine.Search(req)

			if tc.valid {
				if err != nil {
					t.Errorf("Valid sort should not fail: %v", err)
				}
				if response == nil {
					t.Error("Response should not be nil for valid sort")
				}
			} else {
				if err == nil {
					t.Error("Invalid sort should return error")
				}
			}
		})
	}
}

func TestSearchResultContent(t *testing.T) {
	engine, db, embedder := setupSearchTestEngine(t)
	defer cleanup(engine, db, embedder)

	setupSearchTestData(t, db, embedder)

	// Test with content inclusion
	req := &SearchRequest{
		Query:          "golang",
		Mode:           SearchModeFullText,
		Limit:          5,
		IncludeContent: true,
	}

	response, err := engine.Search(req)
	if err != nil {
		t.Fatalf("Search with content failed: %v", err)
	}

	if len(response.Results) > 0 {
		result := response.Results[0]
		if result.Content == "" {
			t.Error("Content should be included when IncludeContent is true")
		}
		if result.FilePath == "" {
			t.Error("FilePath should always be present")
		}
		if result.Score <= 0 {
			t.Error("Score should be positive")
		}
	}

	// Test without content inclusion
	req.IncludeContent = false
	response2, err := engine.Search(req)
	if err != nil {
		t.Fatalf("Search without content failed: %v", err)
	}

	if len(response2.Results) > 0 {
		result := response2.Results[0]
		// Content might still be present depending on implementation
		if result.FilePath == "" {
			t.Error("FilePath should always be present")
		}
	}
}

func TestSearchPerformance(t *testing.T) {
	engine, db, embedder := setupSearchTestEngine(t)
	defer cleanup(engine, db, embedder)

	// Create a larger dataset
	docCount := 100
	for i := 0; i < docCount; i++ {
		file, err := db.UpsertFile(
			filepath.Join("/test/perf", "doc"+string(rune(i%26+65))+".go"),
			"checksum", int64(1000+i))
		if err != nil {
			t.Fatalf("Failed to create perf test file %d: %v", i, err)
		}

		content := "golang programming language tutorial documentation"
		embedding, err := embedder.Embed(content)
		if err != nil {
			t.Fatalf("Failed to generate embedding for perf doc %d: %v", i, err)
		}

		_, err = db.UpsertChunk(file.ID, content, embedding, 0, len(content))
		if err != nil {
			t.Fatalf("Failed to insert chunk for perf doc %d: %v", i, err)
		}
	}

	// Test search performance
	req := &SearchRequest{
		Query: "golang",
		Mode:  SearchModeHybrid,
		Limit: 20,
	}

	start := time.Now()
	response, err := engine.Search(req)
	duration := time.Since(start)

	if err != nil {
		t.Fatalf("Performance test search failed: %v", err)
	}

	if response == nil {
		t.Fatal("Performance test response should not be nil")
	}

	// Search should complete within reasonable time (adjust threshold as needed)
	if duration > 5*time.Second {
		t.Errorf("Search took too long: %v", duration)
	}

	t.Logf("Search completed in %v with %d results", duration, len(response.Results))
}

func TestEdgeCases(t *testing.T) {
	engine, db, embedder := setupSearchTestEngine(t)
	defer cleanup(engine, db, embedder)

	testCases := []struct {
		name string
		req  *SearchRequest
	}{
		{
			"Empty query",
			&SearchRequest{Query: "", Mode: SearchModeFullText, Limit: 10},
		},
		{
			"Very long query",
			&SearchRequest{
				Query: "this is a very long search query that contains many words and should test how the system handles lengthy input strings",
				Mode: SearchModeFullText,
				Limit: 10,
			},
		},
		{
			"Special characters",
			&SearchRequest{Query: "test@#$%^&*()", Mode: SearchModeFullText, Limit: 10},
		},
		{
			"Zero limit",
			&SearchRequest{Query: "test", Mode: SearchModeFullText, Limit: 0},
		},
		{
			"Large limit",
			&SearchRequest{Query: "test", Mode: SearchModeFullText, Limit: 10000},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			response, err := engine.Search(tc.req)

			// These shouldn't crash the system
			if err != nil {
				t.Logf("Edge case '%s' returned error (may be expected): %v", tc.name, err)
			}

			if response != nil && response.Total < 0 {
				t.Errorf("Edge case '%s': total should not be negative", tc.name)
			}
		})
	}
}

// Helper functions

func setupSearchTestEngine(t *testing.T) (*Engine, *database.DB, *embeddings.Embedder) {
	tempDir := t.TempDir()
	dbPath := filepath.Join(tempDir, "search_test.db")

	db, err := database.NewDB(dbPath)
	if err != nil {
		t.Fatalf("Failed to create test database: %v", err)
	}

	cfg := &config.Config{
		FastEmbedModel:     config.FastEmbedAllMiniLML6V2,
		FastEmbedCacheDir:  filepath.Join(tempDir, "fastembed_cache"),
		EmbeddingDimension: 0,
	}

	embedder, err := embeddings.NewEmbedder(cfg)
	if err != nil {
		t.Fatalf("Failed to create test embedder: %v", err)
	}

	databases := map[string]*database.DB{
		"/test/dir": db,
	}

	engine, err := NewEngine(databases, embedder)
	if err != nil {
		t.Fatalf("Failed to create test search engine: %v", err)
	}

	return engine, db, embedder
}

func setupSearchTestData(t *testing.T, db *database.DB, embedder *embeddings.Embedder) {
	testFiles := []struct {
		path    string
		content string
	}{
		{"/test/dir/main.go", "golang programming language main function"},
		{"/test/dir/utils.py", "python utility functions and helpers"},
		{"/test/dir/readme.md", "documentation for the project"},
		{"/test/dir/config.json", "configuration file with settings"},
	}

	for _, tf := range testFiles {
		file, err := db.UpsertFile(tf.path, "checksum", int64(len(tf.content)))
		if err != nil {
			t.Fatalf("Failed to create test file %s: %v", tf.path, err)
		}

		embedding, err := embedder.Embed(tf.content)
		if err != nil {
			t.Fatalf("Failed to generate embedding for %s: %v", tf.path, err)
		}

		_, err = db.UpsertChunk(file.ID, tf.content, embedding, 0, len(tf.content))
		if err != nil {
			t.Fatalf("Failed to insert chunk for %s: %v", tf.path, err)
		}
	}
}

func cleanup(engine *Engine, db *database.DB, embedder *embeddings.Embedder) {
	if engine != nil {
		engine.Close()
	}
	if db != nil {
		db.Close()
	}
	if embedder != nil {
		embedder.Close()
	}
}

func int64Ptr(v int64) *int64 {
	return &v
}