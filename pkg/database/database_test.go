package database

import (
	"os"
	"path/filepath"
	"testing"
)

func TestNewDB(t *testing.T) {
	tempDir := t.TempDir()
	dbPath := filepath.Join(tempDir, "test.db")

	db, err := NewDB(dbPath)
	if err != nil {
		t.Fatalf("Failed to create database: %v", err)
	}
	defer db.Close()

	// Test that database file was created
	if _, err := os.Stat(dbPath); os.IsNotExist(err) {
		t.Error("Database file was not created")
	}

	// Test that we can query the database
	conn := db.GetConn()
	var version string
	err = conn.QueryRow("SELECT sqlite_version()").Scan(&version)
	if err != nil {
		t.Fatalf("Failed to query database: %v", err)
	}

	if version == "" {
		t.Error("SQLite version should not be empty")
	}
}

func TestUpsertFile(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	// Test inserting a new file
	file, err := db.UpsertFile("/test/path.txt", "checksum123", 1024)
	if err != nil {
		t.Fatalf("Failed to upsert file: %v", err)
	}

	if file.ID == 0 {
		t.Error("File ID should not be zero")
	}
	if file.Path != "/test/path.txt" {
		t.Errorf("Expected path '/test/path.txt', got '%s'", file.Path)
	}
	if file.Checksum != "checksum123" {
		t.Errorf("Expected checksum 'checksum123', got '%s'", file.Checksum)
	}
	if file.Size != 1024 {
		t.Errorf("Expected size 1024, got %d", file.Size)
	}

	// Test updating the same file with different checksum
	updatedFile, err := db.UpsertFile("/test/path.txt", "newchecksum456", 2048)
	if err != nil {
		t.Fatalf("Failed to update file: %v", err)
	}

	if updatedFile.ID != file.ID {
		t.Errorf("Expected same file ID %d, got %d", file.ID, updatedFile.ID)
	}
	if updatedFile.Checksum != "newchecksum456" {
		t.Errorf("Expected updated checksum 'newchecksum456', got '%s'", updatedFile.Checksum)
	}
	if updatedFile.Size != 2048 {
		t.Errorf("Expected updated size 2048, got %d", updatedFile.Size)
	}
}

func TestUpsertChunk(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	// First create a file
	file, err := db.UpsertFile("/test/chunk.txt", "checksum", 100)
	if err != nil {
		t.Fatalf("Failed to create file: %v", err)
	}

	// Test upserting a chunk
	embedding := []float32{0.1, 0.2, 0.3, 0.4}
	chunk, err := db.UpsertChunk(file.ID, "test content", embedding, 0, 12)
	if err != nil {
		t.Fatalf("Failed to upsert chunk: %v", err)
	}

	if chunk.ID == 0 {
		t.Error("Chunk ID should not be zero")
	}
	if chunk.FileID != file.ID {
		t.Errorf("Expected file ID %d, got %d", file.ID, chunk.FileID)
	}
	if chunk.Content != "test content" {
		t.Errorf("Expected content 'test content', got '%s'", chunk.Content)
	}
	if len(chunk.Embedding) != len(embedding) {
		t.Errorf("Expected embedding length %d, got %d", len(embedding), len(chunk.Embedding))
	}
	if chunk.Offset != 0 {
		t.Errorf("Expected offset 0, got %d", chunk.Offset)
	}
	if chunk.Length != 12 {
		t.Errorf("Expected length 12, got %d", chunk.Length)
	}

	// Test updating the same chunk
	newEmbedding := []float32{0.5, 0.6, 0.7, 0.8}
	updatedChunk, err := db.UpsertChunk(file.ID, "updated content", newEmbedding, 0, 15)
	if err != nil {
		t.Fatalf("Failed to update chunk: %v", err)
	}

	if updatedChunk.ID != chunk.ID {
		t.Errorf("Expected same chunk ID %d, got %d", chunk.ID, updatedChunk.ID)
	}
	if updatedChunk.Content != "updated content" {
		t.Errorf("Expected updated content 'updated content', got '%s'", updatedChunk.Content)
	}
}

func TestFindFileByPath(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	// Create a file
	originalFile, err := db.UpsertFile("/test/find.txt", "checksum", 100)
	if err != nil {
		t.Fatalf("Failed to create file: %v", err)
	}

	// Test finding the file
	foundFile, err := db.GetFileByPath("/test/find.txt")
	if err != nil {
		t.Fatalf("Failed to find file: %v", err)
	}

	if foundFile == nil {
		t.Fatal("File should not be nil")
	}
	if foundFile.ID != originalFile.ID {
		t.Errorf("Expected file ID %d, got %d", originalFile.ID, foundFile.ID)
	}

	// Test finding non-existent file
	notFound, err := db.GetFileByPath("/nonexistent/path.txt")
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}
	if notFound != nil {
		t.Error("Should return nil for non-existent file")
	}
}

func TestFindChunksByFileID(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	// Create a file
	file, err := db.UpsertFile("/test/chunks.txt", "checksum", 100)
	if err != nil {
		t.Fatalf("Failed to create file: %v", err)
	}

	// Create multiple chunks
	embedding1 := []float32{0.1, 0.2}
	embedding2 := []float32{0.3, 0.4}

	_, err = db.UpsertChunk(file.ID, "chunk 1", embedding1, 0, 7)
	if err != nil {
		t.Fatalf("Failed to create chunk 1: %v", err)
	}

	_, err = db.UpsertChunk(file.ID, "chunk 2", embedding2, 7, 7)
	if err != nil {
		t.Fatalf("Failed to create chunk 2: %v", err)
	}

	// Find chunks
	chunks, err := db.GetChunksByFileID(file.ID)
	if err != nil {
		t.Fatalf("Failed to find chunks: %v", err)
	}

	if len(chunks) != 2 {
		t.Errorf("Expected 2 chunks, got %d", len(chunks))
	}

	// Verify chunk order (should be ordered by offset)
	if chunks[0].Offset > chunks[1].Offset {
		t.Error("Chunks should be ordered by offset")
	}
}

func TestListFiles(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	_, err := db.UpsertFile("/test/one.txt", "c1", 10)
	if err != nil {
		t.Fatalf("failed to create file 1: %v", err)
	}
	_, err = db.UpsertFile("/test/two.txt", "c2", 20)
	if err != nil {
		t.Fatalf("failed to create file 2: %v", err)
	}

	files, err := db.ListFiles(false)
	if err != nil {
		t.Fatalf("ListFiles failed: %v", err)
	}
	if len(files) != 2 {
		t.Fatalf("expected 2 files, got %d", len(files))
	}
}

func TestSoftDelete(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	// Create a file and chunk
	file, err := db.UpsertFile("/test/delete.txt", "checksum", 100)
	if err != nil {
		t.Fatalf("Failed to create file: %v", err)
	}

	embedding := []float32{0.1, 0.2}
	_, err = db.UpsertChunk(file.ID, "content", embedding, 0, 7)
	if err != nil {
		t.Fatalf("Failed to create chunk: %v", err)
	}

	// Test soft deleting file
	err = db.SoftDeleteFile(file.Path)
	if err != nil {
		t.Fatalf("Failed to soft delete file: %v", err)
	}

	// File should still exist but have deleted_at set
	foundFile, err := db.GetFileByPathWithDeleted(file.Path)
	if err != nil {
		t.Fatalf("Failed to find soft deleted file: %v", err)
	}
	if foundFile == nil {
		t.Fatal("Soft deleted file should still exist")
	}
	if foundFile.DeletedAt == nil {
		t.Error("File should have deleted_at timestamp")
	}

	// Test soft deleting chunks by file ID
	err = db.SoftDeleteChunksByFileID(file.ID)
	if err != nil {
		t.Fatalf("Failed to soft delete chunks: %v", err)
	}

	// Chunks should still exist but have deleted_at set
	chunks, err := db.GetChunksByFileID(file.ID)
	if err != nil {
		t.Fatalf("Failed to find chunks: %v", err)
	}
	if len(chunks) > 0 && chunks[0].DeletedAt == nil {
		t.Error("Chunks should have deleted_at timestamp")
	}
}

func TestSearchChunks(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	// Create test data
	file, err := db.UpsertFile("/test/search.txt", "checksum", 100)
	if err != nil {
		t.Fatalf("Failed to create file: %v", err)
	}

	embedding := []float32{0.1, 0.2, 0.3}
	_, err = db.UpsertChunk(file.ID, "hello world test", embedding, 0, 16)
	if err != nil {
		t.Fatalf("Failed to create chunk: %v", err)
	}

	// Test FTS search
	results, err := db.SearchChunks("hello", 10)
	if err != nil {
		t.Fatalf("Failed to search chunks: %v", err)
	}

	if len(results) == 0 {
		t.Error("Should find at least one result for 'hello'")
	}

	if len(results) > 0 && !contains(results[0].Content, "hello") {
		t.Error("Search result should contain 'hello'")
	}
}

func TestCompact(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	// Create and soft delete a file
	file, err := db.UpsertFile("/test/compact.txt", "checksum", 100)
	if err != nil {
		t.Fatalf("Failed to create file: %v", err)
	}

	err = db.SoftDeleteFile(file.Path)
	if err != nil {
		t.Fatalf("Failed to soft delete file: %v", err)
	}

	// Run compaction
	err = db.Compact()
	if err != nil {
		t.Fatalf("Failed to compact database: %v", err)
	}

	// Soft deleted file should be gone
	foundFile, err := db.GetFileByPath(file.Path)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}
	if foundFile != nil {
		t.Error("Soft deleted file should be removed after compaction")
	}
}

func TestConcurrentAccess(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	// Test concurrent file creation
	done := make(chan bool, 2)

	go func() {
		for i := 0; i < 10; i++ {
			_, err := db.UpsertFile("/test/concurrent1.txt", "checksum1", 100)
			if err != nil {
				t.Errorf("Concurrent upsert 1 failed: %v", err)
			}
		}
		done <- true
	}()

	go func() {
		for i := 0; i < 10; i++ {
			_, err := db.UpsertFile("/test/concurrent2.txt", "checksum2", 200)
			if err != nil {
				t.Errorf("Concurrent upsert 2 failed: %v", err)
			}
		}
		done <- true
	}()

	// Wait for both goroutines
	<-done
	<-done
}

// setupTestDB creates a test database in a temporary location
func setupTestDB(t *testing.T) *DB {
	tempDir := t.TempDir()
	dbPath := filepath.Join(tempDir, "test.db")

	db, err := NewDB(dbPath)
	if err != nil {
		t.Fatalf("Failed to create test database: %v", err)
	}

	return db
}

// contains checks if a string contains a substring (case insensitive)
func contains(s, substr string) bool {
	return len(s) >= len(substr) &&
		(s == substr ||
			len(s) > len(substr) &&
				(s[:len(substr)] == substr ||
					s[len(s)-len(substr):] == substr ||
					containsHelper(s, substr)))
}

func containsHelper(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
