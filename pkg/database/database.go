package database

import (
	"database/sql"
	"fmt"
	"strings"
	"time"
	"unsafe"

	_ "github.com/mattn/go-sqlite3"
	sqlite_vec "github.com/asg017/sqlite-vec-go-bindings/cgo"
)

// DB represents a connection to the agentfs SQLite database
type DB struct {
	conn *sql.DB
}

// FileChunk represents a chunk of a file with its metadata
type FileChunk struct {
	ID        int64      `json:"id"`
	FileID    int64      `json:"file_id"`
	Content   string     `json:"content"`
	Embedding []float32  `json:"embedding"`
	Offset    int        `json:"offset"`
	Length    int        `json:"length"`
	CreatedAt time.Time  `json:"created_at"`
	UpdatedAt time.Time  `json:"updated_at"`
	DeletedAt *time.Time `json:"deleted_at,omitempty"`
}

// File represents a file with its metadata
type File struct {
	ID        int64      `json:"id"`
	Path      string     `json:"path"`
	Checksum  string     `json:"checksum"`
	Size      int64      `json:"size"`
	CreatedAt time.Time  `json:"created_at"`
	UpdatedAt time.Time  `json:"updated_at"`
	DeletedAt *time.Time `json:"deleted_at,omitempty"`
}

// NewDB creates a new database connection and initializes the schema
func NewDB(path string) (*DB, error) {
	// Load sqlite-vec extension
	sqlite_vec.Auto()

	conn, err := sql.Open("sqlite3", fmt.Sprintf("file:%s?_journal_mode=WAL&_busy_timeout=10000", path))
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}

	// Set connection pool settings
	conn.SetMaxOpenConns(25)
	conn.SetMaxIdleConns(25)
	conn.SetConnMaxLifetime(5 * time.Minute)

	db := &DB{conn: conn}

	// Initialize schema
	if err := db.initSchema(); err != nil {
		return nil, fmt.Errorf("failed to initialize schema: %w", err)
	}

	return db, nil
}

// initSchema creates the database schema if it doesn't exist
func (db *DB) initSchema() error {
	queries := []string{
		`CREATE TABLE IF NOT EXISTS files (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			path TEXT UNIQUE NOT NULL,
			checksum TEXT NOT NULL,
			size INTEGER NOT NULL,
			created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
			updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
			deleted_at DATETIME NULL
		)`,
		
		`CREATE TABLE IF NOT EXISTS file_chunks (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			file_id INTEGER NOT NULL,
			content TEXT NOT NULL,
			embedding BLOB,
			offset INTEGER NOT NULL,
			length INTEGER NOT NULL,
			created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
			updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
			deleted_at DATETIME NULL,
			FOREIGN KEY (file_id) REFERENCES files (id) ON DELETE CASCADE
		)`,
		
		`CREATE UNIQUE INDEX IF NOT EXISTS idx_chunks_file_offset ON file_chunks (file_id, offset)`,
		`CREATE INDEX IF NOT EXISTS idx_files_path ON files (path)`,
		`CREATE INDEX IF NOT EXISTS idx_files_deleted_at ON files (deleted_at)`,
		`CREATE INDEX IF NOT EXISTS idx_chunks_file_id ON file_chunks (file_id)`,
		`CREATE INDEX IF NOT EXISTS idx_chunks_deleted_at ON file_chunks (deleted_at)`,
	}
	
	for _, query := range queries {
		if _, err := db.conn.Exec(query); err != nil {
			return fmt.Errorf("failed to execute query: %s, error: %w", query, err)
		}
	}
	
	// Try to enable full-text search
	if err := db.enableFTS(); err != nil {
		fmt.Printf("Warning: Failed to enable FTS: %v\n", err)
		fmt.Println("Falling back to simple text search")
	}
	
	return nil
}

// enableFTS tries to enable full-text search
func (db *DB) enableFTS() error {
	// Try to create an FTS5 table to check if it's available
	_, err := db.conn.Exec(`CREATE VIRTUAL TABLE IF NOT EXISTS fts_test USING fts5(test)`)
	if err != nil {
		// Check if the error is because FTS5 is not available
		if strings.Contains(err.Error(), "no such module") || strings.Contains(err.Error(), "unknown tokenizer") {
			return fmt.Errorf("FTS5 not available: %w", err)
		}
		// If it's a different error, return it
		return fmt.Errorf("failed to create FTS test table: %w", err)
	}
	
	// If we successfully created the test table, drop it and proceed
	_, err = db.conn.Exec(`DROP TABLE IF EXISTS fts_test`)
	if err != nil {
		return fmt.Errorf("failed to drop FTS test table: %w", err)
	}
	
	// Enable full-text search
	ftsQueries := []string{
		`CREATE VIRTUAL TABLE IF NOT EXISTS file_chunks_fts USING fts5(
			content,
			content='file_chunks',
			content_rowid='id'
		)`,
		
		`CREATE TRIGGER IF NOT EXISTS file_chunks_ai AFTER INSERT ON file_chunks BEGIN
			INSERT INTO file_chunks_fts(rowid, content) VALUES (new.id, new.content);
		END`,
		
		`CREATE TRIGGER IF NOT EXISTS file_chunks_ad AFTER DELETE ON file_chunks BEGIN
			INSERT INTO file_chunks_fts(file_chunks_fts, rowid, content) 
			VALUES('delete', old.id, old.content);
		END`,
		
		`CREATE TRIGGER IF NOT EXISTS file_chunks_au AFTER UPDATE ON file_chunks BEGIN
			INSERT INTO file_chunks_fts(file_chunks_fts, rowid, content) 
			VALUES('delete', old.id, old.content);
			INSERT INTO file_chunks_fts(rowid, content) VALUES (new.id, new.content);
		END`,
	}
	
	for _, query := range ftsQueries {
		if _, err := db.conn.Exec(query); err != nil {
			return fmt.Errorf("failed to create FTS table/triggers: %w", err)
		}
	}
	
	fmt.Println("FTS5 enabled successfully")
	return nil
}

// UpsertFile inserts or updates a file record
func (db *DB) UpsertFile(path, checksum string, size int64) (*File, error) {
	query := `
		INSERT INTO files (path, checksum, size, updated_at)
		VALUES (?, ?, ?, CURRENT_TIMESTAMP)
		ON CONFLICT(path) DO UPDATE SET
			checksum=excluded.checksum,
			size=excluded.size,
			updated_at=CURRENT_TIMESTAMP,
			deleted_at=NULL
		RETURNING id, path, checksum, size, created_at, updated_at, deleted_at
	`
	
	var file File
	var deletedAt *string
	
	err := db.conn.QueryRow(query, path, checksum, size).Scan(
		&file.ID, &file.Path, &file.Checksum, &file.Size,
		&file.CreatedAt, &file.UpdatedAt, &deletedAt,
	)
	
	if err != nil {
		return nil, fmt.Errorf("failed to upsert file: %w", err)
	}
	
	if deletedAt != nil {
		parsedTime, _ := time.Parse("2006-01-02 15:04:05", *deletedAt)
		file.DeletedAt = &parsedTime
	}
	
	return &file, nil
}

// SoftDeleteFile marks a file as deleted
func (db *DB) SoftDeleteFile(path string) error {
	query := `UPDATE files SET deleted_at=CURRENT_TIMESTAMP WHERE path=?`
	
	_, err := db.conn.Exec(query, path)
	if err != nil {
		return fmt.Errorf("failed to soft delete file: %w", err)
	}
	
	return nil
}

// UpsertChunk inserts or updates a file chunk
func (db *DB) UpsertChunk(fileID int64, content string, embedding []float32, offset, length int) (*FileChunk, error) {
	// Convert embedding to bytes
	var embeddingBytes []byte
	if len(embedding) > 0 {
		embeddingBytes = make([]byte, len(embedding)*4)
		for i, f := range embedding {
			b := *(*[4]byte)(unsafe.Pointer(&f))
			copy(embeddingBytes[i*4:(i+1)*4], b[:])
		}
	}
	
	// First try to update existing chunk
	result, err := db.conn.Exec(`
		UPDATE file_chunks 
		SET content=?, embedding=?, length=?, updated_at=CURRENT_TIMESTAMP, deleted_at=NULL
		WHERE file_id=? AND offset=?`,
		content, embeddingBytes, length, fileID, offset)
	
	if err != nil {
		return nil, fmt.Errorf("failed to update chunk: %w", err)
	}
	
	// Check if we updated a row
	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return nil, fmt.Errorf("failed to get rows affected: %w", err)
	}
	
	// If we didn't update a row, insert a new one
	if rowsAffected == 0 {
		_, err := db.conn.Exec(`
			INSERT INTO file_chunks (file_id, content, embedding, offset, length, updated_at)
			VALUES (?, ?, ?, ?, ?, CURRENT_TIMESTAMP)`,
			fileID, content, embeddingBytes, offset, length)
		
		if err != nil {
			return nil, fmt.Errorf("failed to insert chunk: %w", err)
		}
	}
	
	// Retrieve the chunk
	var chunk FileChunk
	var deletedAt *string
	var embeddingBytesResult []byte
	
	err = db.conn.QueryRow(`
		SELECT id, file_id, content, embedding, offset, length, created_at, updated_at, deleted_at
		FROM file_chunks
		WHERE file_id=? AND offset=?`,
		fileID, offset).Scan(
		&chunk.ID, &chunk.FileID, &chunk.Content, &embeddingBytesResult,
		&chunk.Offset, &chunk.Length, &chunk.CreatedAt, &chunk.UpdatedAt, &deletedAt)
	
	if err != nil {
		return nil, fmt.Errorf("failed to retrieve chunk: %w", err)
	}
	
	// Convert embedding bytes back to float32 slice
	if len(embeddingBytesResult) > 0 {
		chunk.Embedding = make([]float32, len(embeddingBytesResult)/4)
		for i := 0; i < len(chunk.Embedding); i++ {
			b := (*[4]byte)(unsafe.Pointer(&embeddingBytesResult[i*4]))
			chunk.Embedding[i] = *(*float32)(unsafe.Pointer(&b[0]))
		}
	}
	
	if deletedAt != nil {
		parsedTime, _ := time.Parse("2006-01-02 15:04:05", *deletedAt)
		chunk.DeletedAt = &parsedTime
	}
	
	return &chunk, nil
}

// SoftDeleteChunksByFileID marks all chunks for a file as deleted
func (db *DB) SoftDeleteChunksByFileID(fileID int64) error {
	query := `UPDATE file_chunks SET deleted_at=CURRENT_TIMESTAMP WHERE file_id=?`
	
	_, err := db.conn.Exec(query, fileID)
	if err != nil {
		return fmt.Errorf("failed to soft delete chunks: %w", err)
	}
	
	return nil
}

// SearchChunks performs a search on file chunks
func (db *DB) SearchChunks(query string, limit int) ([]FileChunk, error) {
	// Try to use FTS search first
	chunks, err := db.searchChunksFTS(query, limit)
	if err != nil {
		// If FTS fails, fall back to simple LIKE search
		fmt.Printf("FTS search failed, falling back to LIKE search: %v\n", err)
		return db.searchChunksLike(query, limit)
	}
	
	return chunks, nil
}

// searchChunksFTS performs a full-text search on file chunks
func (db *DB) searchChunksFTS(query string, limit int) ([]FileChunk, error) {
	// Check if FTS table exists
	var count int
	err := db.conn.QueryRow(`SELECT count(*) FROM sqlite_master WHERE type='table' AND name='file_chunks_fts'`).Scan(&count)
	if err != nil || count == 0 {
		return nil, fmt.Errorf("FTS table not available")
	}
	
	sqlQuery := `
		SELECT fc.id, fc.file_id, fc.content, fc.embedding, fc.offset, fc.length, 
		       fc.created_at, fc.updated_at, fc.deleted_at
		FROM file_chunks_fts fts
		JOIN file_chunks fc ON fc.id = fts.rowid
		WHERE fts.content MATCH ? AND fc.deleted_at IS NULL
		ORDER BY rank
		LIMIT ?
	`
	
	rows, err := db.conn.Query(sqlQuery, query, limit)
	if err != nil {
		return nil, fmt.Errorf("failed to search chunks: %w", err)
	}
	defer rows.Close()
	
	var chunks []FileChunk
	for rows.Next() {
		var chunk FileChunk
		var deletedAt *string
		var embeddingBytes []byte
		
		err := rows.Scan(
			&chunk.ID, &chunk.FileID, &chunk.Content, &embeddingBytes,
			&chunk.Offset, &chunk.Length, &chunk.CreatedAt, &chunk.UpdatedAt, &deletedAt,
		)
		
		if err != nil {
			return nil, fmt.Errorf("failed to scan chunk: %w", err)
		}
		
		// Convert embedding bytes back to float32 slice
		if len(embeddingBytes) > 0 {
			chunk.Embedding = make([]float32, len(embeddingBytes)/4)
			for i := 0; i < len(chunk.Embedding); i++ {
				b := (*[4]byte)(unsafe.Pointer(&embeddingBytes[i*4]))
				chunk.Embedding[i] = *(*float32)(unsafe.Pointer(&b[0]))
			}
		}
		
		if deletedAt != nil {
			parsedTime, _ := time.Parse("2006-01-02 15:04:05", *deletedAt)
			chunk.DeletedAt = &parsedTime
		}
		
		chunks = append(chunks, chunk)
	}
	
	return chunks, nil
}

// searchChunksLike performs a simple LIKE search on file chunks
func (db *DB) searchChunksLike(query string, limit int) ([]FileChunk, error) {
	sqlQuery := `
		SELECT id, file_id, content, embedding, offset, length, 
		       created_at, updated_at, deleted_at
		FROM file_chunks
		WHERE content LIKE ? AND deleted_at IS NULL
		LIMIT ?
	`
	
	// Add wildcards for LIKE search
	likeQuery := "%" + query + "%"
	
	rows, err := db.conn.Query(sqlQuery, likeQuery, limit)
	if err != nil {
		return nil, fmt.Errorf("failed to search chunks: %w", err)
	}
	defer rows.Close()
	
	var chunks []FileChunk
	for rows.Next() {
		var chunk FileChunk
		var deletedAt *string
		var embeddingBytes []byte
		
		err := rows.Scan(
			&chunk.ID, &chunk.FileID, &chunk.Content, &embeddingBytes,
			&chunk.Offset, &chunk.Length, &chunk.CreatedAt, &chunk.UpdatedAt, &deletedAt,
		)
		
		if err != nil {
			return nil, fmt.Errorf("failed to scan chunk: %w", err)
		}
		
		// Convert embedding bytes back to float32 slice
		if len(embeddingBytes) > 0 {
			chunk.Embedding = make([]float32, len(embeddingBytes)/4)
			for i := 0; i < len(chunk.Embedding); i++ {
				b := (*[4]byte)(unsafe.Pointer(&embeddingBytes[i*4]))
				chunk.Embedding[i] = *(*float32)(unsafe.Pointer(&b[0]))
			}
		}
		
		if deletedAt != nil {
			parsedTime, _ := time.Parse("2006-01-02 15:04:05", *deletedAt)
			chunk.DeletedAt = &parsedTime
		}
		
		chunks = append(chunks, chunk)
	}
	
	return chunks, nil
}

// Compact removes soft-deleted entries
func (db *DB) Compact() error {
	queries := []string{
		`DELETE FROM file_chunks WHERE deleted_at IS NOT NULL AND deleted_at < datetime('now', '-1 day')`,
		`DELETE FROM files WHERE deleted_at IS NOT NULL AND deleted_at < datetime('now', '-1 day')`,
	}
	
	for _, query := range queries {
		if _, err := db.conn.Exec(query); err != nil {
			return fmt.Errorf("failed to compact database: %w", err)
		}
	}
	
	return nil
}

// OptimizeIndexes optimizes database indexes
func (db *DB) OptimizeIndexes() error {
	// Optimize FTS5 index if available
	_, err := db.conn.Exec("INSERT INTO chunks_fts(chunks_fts) VALUES('optimize')")
	if err != nil {
		// FTS5 might not be available, ignore the error
	}

	// Analyze all tables to update statistics
	_, err = db.conn.Exec("ANALYZE")
	if err != nil {
		return fmt.Errorf("failed to analyze database: %w", err)
	}

	// Vacuum the database to reclaim space
	_, err = db.conn.Exec("VACUUM")
	if err != nil {
		return fmt.Errorf("failed to vacuum database: %w", err)
	}

	return nil
}

// GetFileByID retrieves a file by its ID
func (db *DB) GetFileByID(fileID int64) (*File, error) {
	file := &File{}
	var deletedAt sql.NullTime

	err := db.conn.QueryRow(`
		SELECT id, path, checksum, size, created_at, updated_at, deleted_at
		FROM files WHERE id = ? AND deleted_at IS NULL
	`, fileID).Scan(
		&file.ID, &file.Path, &file.Checksum, &file.Size,
		&file.CreatedAt, &file.UpdatedAt, &deletedAt,
	)

	if err != nil {
		return nil, err
	}

	if deletedAt.Valid {
		file.DeletedAt = &deletedAt.Time
	}

	return file, nil
}

// GetFileByPath retrieves a file by its path
func (db *DB) GetFileByPath(path string) (*File, error) {
	file := &File{}
	var deletedAt sql.NullTime

	err := db.conn.QueryRow(`
		SELECT id, path, checksum, size, created_at, updated_at, deleted_at
		FROM files WHERE path = ? AND deleted_at IS NULL
	`, path).Scan(
		&file.ID, &file.Path, &file.Checksum, &file.Size,
		&file.CreatedAt, &file.UpdatedAt, &deletedAt,
	)

	if err != nil {
		return nil, err
	}

	if deletedAt.Valid {
		file.DeletedAt = &deletedAt.Time
	}

	return file, nil
}

// GetAllFiles retrieves all non-deleted files
func (db *DB) GetAllFiles() ([]*File, error) {
	rows, err := db.conn.Query(`
		SELECT id, path, checksum, size, created_at, updated_at, deleted_at
		FROM files WHERE deleted_at IS NULL
		ORDER BY updated_at DESC
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var files []*File
	for rows.Next() {
		file := &File{}
		var deletedAt sql.NullTime

		err := rows.Scan(
			&file.ID, &file.Path, &file.Checksum, &file.Size,
			&file.CreatedAt, &file.UpdatedAt, &deletedAt,
		)
		if err != nil {
			continue
		}

		if deletedAt.Valid {
			file.DeletedAt = &deletedAt.Time
		}

		files = append(files, file)
	}

	return files, nil
}

// GetChunkByID retrieves a chunk by its ID
func (db *DB) GetChunkByID(chunkID int64) (*FileChunk, error) {
	chunk := &FileChunk{}
	var deletedAt sql.NullTime

	err := db.conn.QueryRow(`
		SELECT id, file_id, content, embedding, offset, length, created_at, updated_at, deleted_at
		FROM chunks WHERE id = ? AND deleted_at IS NULL
	`, chunkID).Scan(
		&chunk.ID, &chunk.FileID, &chunk.Content, &chunk.Embedding,
		&chunk.Offset, &chunk.Length, &chunk.CreatedAt, &chunk.UpdatedAt, &deletedAt,
	)

	if err != nil {
		return nil, err
	}

	if deletedAt.Valid {
		chunk.DeletedAt = &deletedAt.Time
	}

	return chunk, nil
}

// GetChunksByFileID retrieves all chunks for a file
func (db *DB) GetChunksByFileID(fileID int64) ([]*FileChunk, error) {
	rows, err := db.conn.Query(`
		SELECT id, file_id, content, embedding, offset, length, created_at, updated_at, deleted_at
		FROM chunks WHERE file_id = ? AND deleted_at IS NULL
		ORDER BY offset ASC
	`, fileID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var chunks []*FileChunk
	for rows.Next() {
		chunk := &FileChunk{}
		var deletedAt sql.NullTime

		err := rows.Scan(
			&chunk.ID, &chunk.FileID, &chunk.Content, &chunk.Embedding,
			&chunk.Offset, &chunk.Length, &chunk.CreatedAt, &chunk.UpdatedAt, &deletedAt,
		)
		if err != nil {
			continue
		}

		if deletedAt.Valid {
			chunk.DeletedAt = &deletedAt.Time
		}

		chunks = append(chunks, chunk)
	}

	return chunks, nil
}

// GetConn returns the underlying SQL database connection
func (db *DB) GetConn() *sql.DB {
	return db.conn
}

// Close closes the database connection
func (db *DB) Close() error {
	return db.conn.Close()
}