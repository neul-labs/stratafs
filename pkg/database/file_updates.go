package database

import (
	"database/sql"
	"fmt"
	"time"
	"unsafe"
)

// FileUpdateStrategy defines how to handle file updates
type FileUpdateStrategy string

const (
	// UpdateStrategyReplace replaces all chunks immediately (current behavior)
	UpdateStrategyReplace FileUpdateStrategy = "replace"

	// UpdateStrategySoftDelete soft deletes old chunks, then adds new ones (recommended)
	UpdateStrategySoftDelete FileUpdateStrategy = "soft_delete"

	// UpdateStrategyVersioned keeps multiple versions of chunks (future feature)
	UpdateStrategyVersioned FileUpdateStrategy = "versioned"
)

// FileUpdateManager handles file update operations
type FileUpdateManager struct {
	db       *DB
	strategy FileUpdateStrategy
}

// NewFileUpdateManager creates a new file update manager
func NewFileUpdateManager(db *DB, strategy FileUpdateStrategy) *FileUpdateManager {
	if strategy == "" {
		strategy = UpdateStrategySoftDelete
	}

	return &FileUpdateManager{
		db:       db,
		strategy: strategy,
	}
}

// UpdateFile updates a file and its chunks using the configured strategy
func (m *FileUpdateManager) UpdateFile(path, checksum string, size int64, chunks []ChunkData) (*File, error) {
	// First, upsert the file record
	file, err := m.db.UpsertFile(path, checksum, size)
	if err != nil {
		return nil, fmt.Errorf("failed to upsert file: %w", err)
	}

	// Handle chunks based on strategy
	switch m.strategy {
	case UpdateStrategyReplace:
		return file, m.replaceChunks(file.ID, chunks)
	case UpdateStrategySoftDelete:
		return file, m.softDeleteAndReplaceChunks(file.ID, chunks)
	case UpdateStrategyVersioned:
		return file, m.versionedUpdateChunks(file.ID, chunks)
	default:
		return file, fmt.Errorf("unknown update strategy: %s", m.strategy)
	}
}

// ChunkData represents chunk data for updates
type ChunkData struct {
	Content   string
	Embedding []float32
	Offset    int
	Length    int
}

// replaceChunks immediately deletes and replaces all chunks
func (m *FileUpdateManager) replaceChunks(fileID int64, chunks []ChunkData) error {
	// Start transaction
	tx, err := m.db.conn.Begin()
	if err != nil {
		return fmt.Errorf("failed to start transaction: %w", err)
	}
	defer tx.Rollback()

	// Delete existing chunks
	_, err = tx.Exec("DELETE FROM file_chunks WHERE file_id = ?", fileID)
	if err != nil {
		return fmt.Errorf("failed to delete existing chunks: %w", err)
	}

	// Insert new chunks
	for _, chunk := range chunks {
		err = m.insertChunk(tx, fileID, chunk)
		if err != nil {
			return fmt.Errorf("failed to insert chunk: %w", err)
		}
	}

	return tx.Commit()
}

// softDeleteAndReplaceChunks soft deletes old chunks and adds new ones
func (m *FileUpdateManager) softDeleteAndReplaceChunks(fileID int64, chunks []ChunkData) error {
	// Start transaction
	tx, err := m.db.conn.Begin()
	if err != nil {
		return fmt.Errorf("failed to start transaction: %w", err)
	}
	defer tx.Rollback()

	// Soft delete existing chunks
	_, err = tx.Exec(`
		UPDATE file_chunks
		SET deleted_at = CURRENT_TIMESTAMP
		WHERE file_id = ? AND deleted_at IS NULL
	`, fileID)
	if err != nil {
		return fmt.Errorf("failed to soft delete existing chunks: %w", err)
	}

	// Insert new chunks
	for _, chunk := range chunks {
		err = m.insertChunk(tx, fileID, chunk)
		if err != nil {
			return fmt.Errorf("failed to insert chunk: %w", err)
		}
	}

	return tx.Commit()
}

// versionedUpdateChunks keeps old versions and only updates changed chunks
func (m *FileUpdateManager) versionedUpdateChunks(fileID int64, chunks []ChunkData) error {
	// Start transaction
	tx, err := m.db.conn.Begin()
	if err != nil {
		return fmt.Errorf("failed to start transaction: %w", err)
	}
	defer tx.Rollback()

	// Get existing chunks for comparison
	existingChunks, err := m.getExistingChunks(tx, fileID)
	if err != nil {
		return fmt.Errorf("failed to get existing chunks: %w", err)
	}

	// Build content hash map of existing chunks for comparison
	existingByHash := make(map[string]int64) // content hash -> chunk ID
	for _, chunk := range existingChunks {
		hash := m.hashContent(chunk.Content)
		existingByHash[hash] = chunk.ID
	}

	// Track which existing chunks are still in use
	usedChunkIDs := make(map[int64]bool)
	var newChunks []ChunkData

	// Compare new chunks against existing
	for _, chunk := range chunks {
		hash := m.hashContent(chunk.Content)
		if existingID, found := existingByHash[hash]; found {
			// Chunk content unchanged - keep it
			usedChunkIDs[existingID] = true
		} else {
			// New or modified chunk - needs to be inserted
			newChunks = append(newChunks, chunk)
		}
	}

	// Soft delete chunks that are no longer in the file
	for _, existing := range existingChunks {
		if !usedChunkIDs[existing.ID] {
			_, err = tx.Exec(`
				UPDATE file_chunks
				SET deleted_at = CURRENT_TIMESTAMP
				WHERE id = ? AND deleted_at IS NULL
			`, existing.ID)
			if err != nil {
				return fmt.Errorf("failed to soft delete old chunk: %w", err)
			}
		}
	}

	// Insert new chunks
	for _, chunk := range newChunks {
		err = m.insertChunk(tx, fileID, chunk)
		if err != nil {
			return fmt.Errorf("failed to insert new chunk: %w", err)
		}
	}

	return tx.Commit()
}

// existingChunk represents an existing chunk from the database
type existingChunk struct {
	ID      int64
	Content string
}

// getExistingChunks retrieves all non-deleted chunks for a file
func (m *FileUpdateManager) getExistingChunks(tx *sql.Tx, fileID int64) ([]existingChunk, error) {
	query := `
		SELECT id, COALESCE(content, '') as content, content_compressed, is_compressed
		FROM file_chunks
		WHERE file_id = ? AND deleted_at IS NULL
		ORDER BY offset
	`

	rows, err := tx.Query(query, fileID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var chunks []existingChunk
	for rows.Next() {
		var chunk existingChunk
		var contentCompressed []byte
		var isCompressed bool

		if err := rows.Scan(&chunk.ID, &chunk.Content, &contentCompressed, &isCompressed); err != nil {
			return nil, err
		}

		// Decompress if needed
		if isCompressed && len(contentCompressed) > 0 {
			decompressed, err := decompressContent(contentCompressed)
			if err != nil {
				return nil, fmt.Errorf("failed to decompress chunk: %w", err)
			}
			chunk.Content = decompressed
		}

		chunks = append(chunks, chunk)
	}

	return chunks, rows.Err()
}

// hashContent creates a simple hash of content for comparison
func (m *FileUpdateManager) hashContent(content string) string {
	// Use a simple FNV-1a hash for fast comparison
	h := uint64(14695981039346656037)
	for i := 0; i < len(content); i++ {
		h ^= uint64(content[i])
		h *= 1099511628211
	}
	return fmt.Sprintf("%x", h)
}

// insertChunk inserts a single chunk
func (m *FileUpdateManager) insertChunk(tx interface{}, fileID int64, chunk ChunkData) error {
	// Convert embedding to bytes
	var embeddingBytes []byte
	if len(chunk.Embedding) > 0 {
		embeddingBytes = make([]byte, len(chunk.Embedding)*4)
		for i, val := range chunk.Embedding {
			bits := *(*uint32)(unsafe.Pointer(&val))
			embeddingBytes[i*4] = byte(bits)
			embeddingBytes[i*4+1] = byte(bits >> 8)
			embeddingBytes[i*4+2] = byte(bits >> 16)
			embeddingBytes[i*4+3] = byte(bits >> 24)
		}
	}

	// Check if content should be compressed
	contentCompressed, isCompressed, err := compressContent(chunk.Content)
	if err != nil {
		return fmt.Errorf("failed to compress content: %w", err)
	}

	content := chunk.Content
	if isCompressed {
		content = "" // Clear original content when compressed
	}

	// Insert chunk
	query := `
		INSERT INTO file_chunks (file_id, content, content_compressed, is_compressed, embedding, offset, length, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP)
	`

	// Execute based on transaction type
	switch t := tx.(type) {
	case *sql.Tx:
		_, err := t.Exec(query, fileID, content, contentCompressed, isCompressed,
			embeddingBytes, chunk.Offset, chunk.Length)
		return err
	default:
		return fmt.Errorf("invalid transaction type")
	}
}

// FileUpdateStats tracks update operation statistics
type FileUpdateStats struct {
	FilesUpdated     int           `json:"files_updated"`
	ChunksDeleted    int           `json:"chunks_deleted"`
	ChunksAdded      int           `json:"chunks_added"`
	Duration         time.Duration `json:"duration"`
	Strategy         string        `json:"strategy"`
	BytesProcessed   int64         `json:"bytes_processed"`
	CompressionSaved int64         `json:"compression_saved"`
}

// GetUpdateStats returns statistics about recent update operations
func (m *FileUpdateManager) GetUpdateStats() (*FileUpdateStats, error) {
	// This would track statistics over time
	// For now, return basic stats
	stats := &FileUpdateStats{
		Strategy: string(m.strategy),
	}

	// Query for recent update statistics
	query := `
		SELECT COUNT(*) as file_count,
		       SUM(size) as total_size
		FROM files
		WHERE updated_at > datetime('now', '-1 hour')
	`

	err := m.db.conn.QueryRow(query).Scan(&stats.FilesUpdated, &stats.BytesProcessed)
	if err != nil {
		return nil, fmt.Errorf("failed to get update stats: %w", err)
	}

	return stats, nil
}

// CleanupOldVersions removes soft-deleted chunks older than threshold
func (m *FileUpdateManager) CleanupOldVersions(olderThan time.Duration) error {
	threshold := time.Now().Add(-olderThan)

	query := `
		DELETE FROM file_chunks
		WHERE deleted_at IS NOT NULL
		AND deleted_at < ?
	`

	result, err := m.db.conn.Exec(query, threshold.Format("2006-01-02 15:04:05"))
	if err != nil {
		return fmt.Errorf("failed to cleanup old versions: %w", err)
	}

	// Get number of rows affected for potential future statistics
	_, _ = result.RowsAffected()

	return nil
}

// SetStrategy changes the update strategy
func (m *FileUpdateManager) SetStrategy(strategy FileUpdateStrategy) {
	m.strategy = strategy
}

// GetStrategy returns the current update strategy
func (m *FileUpdateManager) GetStrategy() FileUpdateStrategy {
	return m.strategy
}