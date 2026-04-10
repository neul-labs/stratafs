package search

import (
	"database/sql"
	"fmt"
	"math"
	"sync"
)

// VectorIndex manages vector similarity search using sqlite-vec in the main database
type VectorIndex struct {
	db         *sql.DB // Shared database with main AgentFS data
	dimensions int
	mutex      sync.RWMutex
}

// VectorSearchResult represents a vector similarity result
type VectorSearchResult struct {
	ChunkID int64   `json:"chunk_id"`
	Score   float64 `json:"score"`
	Vector  []float32 `json:"vector,omitempty"`
}

// NewVectorIndex creates a new vector index using sqlite-vec in an existing database
func NewVectorIndex(db *sql.DB, dimensions int) (*VectorIndex, error) {
	vi := &VectorIndex{
		db:         db,
		dimensions: dimensions,
	}

	// Create vector table in the existing database
	if err := vi.createVectorTable(); err != nil {
		return nil, fmt.Errorf("failed to create vector table: %w", err)
	}

	return vi, nil
}

// createVectorTable creates the vector table using sqlite-vec
func (vi *VectorIndex) createVectorTable() error {
	createSQL := fmt.Sprintf(`
		CREATE TABLE IF NOT EXISTS chunk_vectors (
			chunk_id INTEGER PRIMARY KEY,
			embedding BLOB
		);

		CREATE VIRTUAL TABLE IF NOT EXISTS vec_chunks USING vec0(
			embedding float[%d]
		);
	`, vi.dimensions)

	_, err := vi.db.Exec(createSQL)
	return err
}

// AddVector adds a vector to the index
func (vi *VectorIndex) AddVector(chunkID int64, vector []float32) error {
	vi.mutex.Lock()
	defer vi.mutex.Unlock()

	if len(vector) != vi.dimensions {
		return fmt.Errorf("vector dimension mismatch: expected %d, got %d", vi.dimensions, len(vector))
	}

	// Convert vector to bytes for storage
	vectorBytes := make([]byte, len(vector)*4)
	for i, v := range vector {
		bits := math.Float32bits(v)
		vectorBytes[i*4] = byte(bits)
		vectorBytes[i*4+1] = byte(bits >> 8)
		vectorBytes[i*4+2] = byte(bits >> 16)
		vectorBytes[i*4+3] = byte(bits >> 24)
	}

	// Insert into both tables
	tx, err := vi.db.Begin()
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback()

	// Insert into chunk_vectors table
	_, err = tx.Exec("INSERT OR REPLACE INTO chunk_vectors (chunk_id, embedding) VALUES (?, ?)", chunkID, vectorBytes)
	if err != nil {
		return fmt.Errorf("failed to insert vector: %w", err)
	}

	// Insert into vec_chunks virtual table
	_, err = tx.Exec("INSERT OR REPLACE INTO vec_chunks (rowid, embedding) VALUES (?, vec_f32(?))", chunkID, vectorBytes)
	if err != nil {
		return fmt.Errorf("failed to insert into vector index: %w", err)
	}

	return tx.Commit()
}

// RemoveVector removes a vector from the index
func (vi *VectorIndex) RemoveVector(chunkID int64) error {
	vi.mutex.Lock()
	defer vi.mutex.Unlock()

	tx, err := vi.db.Begin()
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback()

	// Remove from both tables
	_, err = tx.Exec("DELETE FROM chunk_vectors WHERE chunk_id = ?", chunkID)
	if err != nil {
		return fmt.Errorf("failed to remove from chunk_vectors: %w", err)
	}

	_, err = tx.Exec("DELETE FROM vec_chunks WHERE rowid = ?", chunkID)
	if err != nil {
		return fmt.Errorf("failed to remove from vector index: %w", err)
	}

	return tx.Commit()
}

// SearchSimilar finds vectors similar to the query vector using sqlite-vec
func (vi *VectorIndex) SearchSimilar(queryVector []float32, limit int) ([]VectorSearchResult, error) {
	vi.mutex.RLock()
	defer vi.mutex.RUnlock()

	if len(queryVector) != vi.dimensions {
		return nil, fmt.Errorf("query vector dimension mismatch: expected %d, got %d", vi.dimensions, len(queryVector))
	}

	if limit <= 0 {
		limit = 10
	}

	// Convert query vector to bytes
	queryBytes := make([]byte, len(queryVector)*4)
	for i, v := range queryVector {
		bits := math.Float32bits(v)
		queryBytes[i*4] = byte(bits)
		queryBytes[i*4+1] = byte(bits >> 8)
		queryBytes[i*4+2] = byte(bits >> 16)
		queryBytes[i*4+3] = byte(bits >> 24)
	}

	// Perform vector similarity search
	query := `
		SELECT rowid, distance
		FROM vec_chunks
		WHERE embedding MATCH vec_f32(?)
		ORDER BY distance
		LIMIT ?
	`

	rows, err := vi.db.Query(query, queryBytes, limit)
	if err != nil {
		return nil, fmt.Errorf("failed to search vectors: %w", err)
	}
	defer rows.Close()

	var results []VectorSearchResult
	for rows.Next() {
		var chunkID int64
		var distance float64

		if err := rows.Scan(&chunkID, &distance); err != nil {
			return nil, fmt.Errorf("failed to scan result: %w", err)
		}

		// Convert distance to similarity score (lower distance = higher similarity)
		score := 1.0 / (1.0 + distance)

		results = append(results, VectorSearchResult{
			ChunkID: chunkID,
			Score:   score,
		})
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating results: %w", err)
	}

	return results, nil
}

// Save persists the index to disk (SQLite handles this automatically)
func (vi *VectorIndex) Save() error {
	vi.mutex.RLock()
	defer vi.mutex.RUnlock()

	// SQLite automatically persists data, but we can force a checkpoint
	_, err := vi.db.Exec("PRAGMA wal_checkpoint(FULL)")
	return err
}

// GetSize returns the number of vectors in the index
func (vi *VectorIndex) GetSize() uint64 {
	vi.mutex.RLock()
	defer vi.mutex.RUnlock()

	var count int64
	err := vi.db.QueryRow("SELECT COUNT(*) FROM chunk_vectors").Scan(&count)
	if err != nil {
		return 0
	}
	return uint64(count)
}

// GetCapacity returns the capacity of the index (unlimited for SQLite)
func (vi *VectorIndex) GetCapacity() uint64 {
	return ^uint64(0) // Unlimited
}

// GetDimensions returns the vector dimensions
func (vi *VectorIndex) GetDimensions() int {
	return vi.dimensions
}

// Compact optimizes the index structure
func (vi *VectorIndex) Compact() error {
	vi.mutex.Lock()
	defer vi.mutex.Unlock()

	// Run VACUUM to optimize the database
	_, err := vi.db.Exec("VACUUM")
	return err
}

// GetStats returns index statistics
func (vi *VectorIndex) GetStats() map[string]interface{} {
	vi.mutex.RLock()
	defer vi.mutex.RUnlock()

	stats := map[string]interface{}{
		"size":       vi.GetSize(),
		"capacity":   vi.GetCapacity(),
		"dimensions": vi.dimensions,
	}

	// Get database file size
	var pageCount, pageSize int64
	vi.db.QueryRow("PRAGMA page_count").Scan(&pageCount)
	vi.db.QueryRow("PRAGMA page_size").Scan(&pageSize)
	stats["db_size_bytes"] = pageCount * pageSize

	return stats
}

// Close saves the index (database is managed externally)
func (vi *VectorIndex) Close() error {
	// Don't close the database as it's shared with main AgentFS database
	// Just ensure data is persisted
	return vi.Save()
}

// BatchAddVectors adds multiple vectors efficiently
func (vi *VectorIndex) BatchAddVectors(vectors map[int64][]float32) error {
	vi.mutex.Lock()
	defer vi.mutex.Unlock()

	tx, err := vi.db.Begin()
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback()

	for chunkID, vector := range vectors {
		if len(vector) != vi.dimensions {
			return fmt.Errorf("vector dimension mismatch for chunk %d: expected %d, got %d",
				chunkID, vi.dimensions, len(vector))
		}

		// Convert vector to bytes
		vectorBytes := make([]byte, len(vector)*4)
		for i, v := range vector {
			bits := math.Float32bits(v)
			vectorBytes[i*4] = byte(bits)
			vectorBytes[i*4+1] = byte(bits >> 8)
			vectorBytes[i*4+2] = byte(bits >> 16)
			vectorBytes[i*4+3] = byte(bits >> 24)
		}

		// Insert into both tables
		_, err = tx.Exec("INSERT OR REPLACE INTO chunk_vectors (chunk_id, embedding) VALUES (?, ?)", chunkID, vectorBytes)
		if err != nil {
			return fmt.Errorf("failed to insert vector for chunk %d: %w", chunkID, err)
		}

		_, err = tx.Exec("INSERT OR REPLACE INTO vec_chunks (rowid, embedding) VALUES (?, vec_f32(?))", chunkID, vectorBytes)
		if err != nil {
			return fmt.Errorf("failed to insert into vector index for chunk %d: %w", chunkID, err)
		}
	}

	return tx.Commit()
}

// ContainsVector checks if a vector exists in the index
func (vi *VectorIndex) ContainsVector(chunkID int64) bool {
	vi.mutex.RLock()
	defer vi.mutex.RUnlock()

	var exists bool
	err := vi.db.QueryRow("SELECT EXISTS(SELECT 1 FROM chunk_vectors WHERE chunk_id = ?)", chunkID).Scan(&exists)
	return err == nil && exists
}