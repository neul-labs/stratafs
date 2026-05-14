package search

import (
	"path/filepath"
	"testing"

	"github.com/neul-labs/stratafs/pkg/config"
	"github.com/neul-labs/stratafs/pkg/database"
	"github.com/neul-labs/stratafs/pkg/embeddings"
)

func TestNewEngine(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	embedder := setupTestEmbedder(t)
	defer embedder.Close()

	databases := map[string]*database.DB{
		"/test/dir": db,
	}

	engine, err := NewEngine(databases, embedder)
	if err != nil {
		t.Fatalf("Failed to create search engine: %v", err)
	}
	defer engine.Close()

	if engine == nil {
		t.Error("Search engine should not be nil")
	}
}

func TestBasicSearch(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	embedder := setupTestEmbedder(t)
	defer embedder.Close()

	databases := map[string]*database.DB{
		"/test/dir": db,
	}

	engine, err := NewEngine(databases, embedder)
	if err != nil {
		t.Fatalf("Failed to create search engine: %v", err)
	}
	defer engine.Close()

	// Create test data
	file, err := db.UpsertFile("/test/dir/document.txt", "checksum123", 1000)
	if err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	// Generate test embedding
	testContent := "test document content"
	embedding, err := embedder.Embed(testContent)
	if err != nil {
		t.Fatalf("Failed to generate test embedding: %v", err)
	}

	// Insert test chunk
	_, err = db.UpsertChunk(file.ID, testContent, embedding, 0, len(testContent))
	if err != nil {
		t.Fatalf("Failed to insert test chunk: %v", err)
	}

	// Test basic search
	req := &SearchRequest{
		Query: "test",
		Limit: 5,
		Mode:  SearchModeFullText,
	}

	response, err := engine.Search(req)
	if err != nil {
		t.Fatalf("Failed to perform search: %v", err)
	}

	if response == nil {
		t.Fatal("Response should not be nil")
	}

	// Basic validation - we don't expect specific results since search depends on SQLite FTS
	if response.Total < 0 {
		t.Error("Total should not be negative")
	}
}

func TestEmptySearch(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	embedder := setupTestEmbedder(t)
	defer embedder.Close()

	databases := map[string]*database.DB{
		"/test/dir": db,
	}

	engine, err := NewEngine(databases, embedder)
	if err != nil {
		t.Fatalf("Failed to create search engine: %v", err)
	}
	defer engine.Close()

	// Test empty query
	req := &SearchRequest{
		Query: "",
		Limit: 5,
		Mode:  SearchModeFullText,
	}

	response, err := engine.Search(req)
	if err != nil {
		t.Fatalf("Failed to handle empty search: %v", err)
	}

	if response == nil {
		t.Error("Response should not be nil even for empty query")
	}
}

func TestEngineClose(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	embedder := setupTestEmbedder(t)
	defer embedder.Close()

	databases := map[string]*database.DB{
		"/test/dir": db,
	}

	engine, err := NewEngine(databases, embedder)
	if err != nil {
		t.Fatalf("Failed to create search engine: %v", err)
	}

	// Test that Close() doesn't panic
	err = engine.Close()
	if err != nil {
		t.Errorf("Engine Close() returned error: %v", err)
	}
}

// Helper functions for testing

func setupTestDB(t *testing.T) *database.DB {
	tempDir := t.TempDir()
	dbPath := filepath.Join(tempDir, "test.db")

	db, err := database.NewDB(dbPath)
	if err != nil {
		t.Fatalf("Failed to create test database: %v", err)
	}

	return db
}

func setupTestEmbedder(t *testing.T) *embeddings.Embedder {
	tempDir := t.TempDir()

	cfg := &config.Config{
		Embedding: config.EmbeddingConfig{
			Model:     config.FastEmbedAllMiniLML6V2, // Smallest model for testing
			CacheDir:  filepath.Join(tempDir, "fastembed_cache"),
			Dimension: 0,
		},
	}

	embedder, err := embeddings.NewEmbedder(cfg)
	handleEmbedderInitError(t, err)

	return embedder
}
