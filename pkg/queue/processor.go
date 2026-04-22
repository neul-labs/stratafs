package queue

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	"agentfs/pkg/chunking"
	"agentfs/pkg/config"
	"agentfs/pkg/database"
	"agentfs/pkg/embeddings"
	"agentfs/pkg/filesystem"
	"agentfs/pkg/parsers"
	"agentfs/pkg/search"
)

// FileInfo represents file metadata for job processing
type FileInfo struct {
	Path         string    `json:"path"`
	Size         int64     `json:"size"`
	ModifiedTime time.Time `json:"modified_time"`
	Checksum     string    `json:"checksum"`
}

// AgentFSProcessor processes AgentFS jobs
type AgentFSProcessor struct {
	config        *config.Config
	databases     map[string]*database.DB
	embedder      *embeddings.Embedder
	filesystem    filesystem.FileSystem
	queue         *Queue
	searchEngine  *search.Engine
	chunkingService *chunking.ChunkingService
	updateManagers  map[string]*database.FileUpdateManager
}

// NewAgentFSProcessor creates a new processor
func NewAgentFSProcessor(cfg *config.Config, databases map[string]*database.DB, embedder *embeddings.Embedder, queue *Queue, searchEngine *search.Engine) *AgentFSProcessor {
	// Initialize update managers for each database
	updateManagers := make(map[string]*database.FileUpdateManager)
	for dbID, db := range databases {
		updateManagers[dbID] = database.NewFileUpdateManager(db, database.UpdateStrategySoftDelete)
	}

	return &AgentFSProcessor{
		config:          cfg,
		databases:       databases,
		embedder:        embedder,
		filesystem:      filesystem.NewLocalFileSystem(),
		queue:           queue,
		searchEngine:    searchEngine,
		chunkingService: chunking.NewChunkingService(),
		updateManagers:  updateManagers,
	}
}

// ProcessJob processes a job based on its type
func (p *AgentFSProcessor) ProcessJob(ctx context.Context, job *Job) error {
	switch job.Type {
	case JobTypeParse:
		return p.processParseJob(ctx, job)
	case JobTypeEmbed:
		return p.processEmbedJob(ctx, job)
	case JobTypeIndex:
		return p.processIndexJob(ctx, job)
	default:
		return fmt.Errorf("unknown job type: %s", job.Type)
	}
}

// processParseJob handles file parsing
func (p *AgentFSProcessor) processParseJob(ctx context.Context, job *Job) error {
	// Try to parse as remote file payload first, fall back to FileInfo
	var payload map[string]interface{}
	var fileInfo FileInfo
	var isRemoteFile bool
	var shouldCleanup bool

	if err := json.Unmarshal([]byte(job.Payload), &payload); err == nil {
		// Check if this is a remote file job with cleanup flag
		if cleanup, exists := payload["cleanup_after_processing"]; exists {
			if cleanupBool, ok := cleanup.(bool); ok && cleanupBool {
				isRemoteFile = true
				shouldCleanup = true
			}
		}

		// For remote files, we don't have traditional FileInfo, so create minimal info
		if isRemoteFile {
			stat, err := os.Stat(job.FilePath)
			if err != nil {
				if os.IsNotExist(err) {
					return fmt.Errorf("cached file not found: %s", job.FilePath)
				}
				return fmt.Errorf("failed to stat cached file: %w", err)
			}
			fileInfo = FileInfo{
				Path:         job.FilePath,
				Size:         stat.Size(),
				ModifiedTime: stat.ModTime(),
				Checksum:     "", // Will be calculated later if needed
			}
		}
	}

	// If not a remote file, parse as traditional FileInfo
	if !isRemoteFile {
		if err := json.Unmarshal([]byte(job.Payload), &fileInfo); err != nil {
			return fmt.Errorf("failed to parse file info: %w", err)
		}
	}

	// Check if file still exists and hasn't changed (skip for remote files)
	if !isRemoteFile {
		stat, err := os.Stat(job.FilePath)
		if err != nil {
			if os.IsNotExist(err) {
				// File was deleted, mark it as deleted in database
				return p.markFileDeleted(job.DirectoryID, job.FilePath)
			}
			return fmt.Errorf("failed to stat file: %w", err)
		}

		// Check if file was modified since job was created
		if stat.ModTime().After(fileInfo.ModifiedTime) {
			return fmt.Errorf("file was modified after job creation, skipping")
		}
	}

	// Get database for this directory
	_, exists := p.databases[job.DirectoryID]
	if !exists {
		return fmt.Errorf("database not found for directory: %s", job.DirectoryID)
	}

	// Open and parse the file
	file, err := p.filesystem.Open(job.FilePath)
	if err != nil {
		return fmt.Errorf("failed to open file: %w", err)
	}
	defer file.Close()

	// Get appropriate parser
	parser := parsers.GetParser(job.FilePath)
	if parser == nil {
		return fmt.Errorf("unsupported file type for parsing: %s", job.FilePath)
	}

	// Parse content
	content, err := parser.Parse(file)
	if err != nil {
		return fmt.Errorf("failed to parse file: %w", err)
	}

	// Skip empty files
	if len(content) == 0 {
		// Clean up cached file if this was a remote file
		if shouldCleanup {
			if err := os.Remove(job.FilePath); err != nil {
				fmt.Printf("Warning: failed to clean up cached file %s: %v\n", job.FilePath, err)
			}
		}
		return nil
	}

	// Process content using streaming chunking
	err = p.processContentStreaming(job.DirectoryID, job.FilePath, fileInfo, file)
	if err != nil {
		return fmt.Errorf("failed to process content: %w", err)
	}

	// Clean up cached file if this was a remote file
	if shouldCleanup {
		if err := os.Remove(job.FilePath); err != nil {
			// Log warning but don't fail the job since processing was successful
			fmt.Printf("Warning: failed to clean up cached file %s: %v\n", job.FilePath, err)
		} else {
			fmt.Printf("Cleaned up cached file: %s\n", job.FilePath)
		}
	}

	return nil
}

// processEmbedJob handles text embedding
func (p *AgentFSProcessor) processEmbedJob(ctx context.Context, job *Job) error {
	// Parse embed payload
	var payload map[string]interface{}
	if err := json.Unmarshal([]byte(job.Payload), &payload); err != nil {
		return fmt.Errorf("failed to parse embed payload: %w", err)
	}

	content, ok := payload["content"].(string)
	if !ok {
		return fmt.Errorf("invalid content in payload")
	}

	fileID, ok := payload["file_id"].(float64) // JSON numbers are float64
	if !ok {
		return fmt.Errorf("invalid file_id in payload")
	}

	// Get database for this directory
	db, exists := p.databases[job.DirectoryID]
	if !exists {
		return fmt.Errorf("database not found for directory: %s", job.DirectoryID)
	}

	// Use new chunking service
	contentReader := strings.NewReader(content)
	fileExt := strings.TrimPrefix(filepath.Ext(job.FilePath), ".")
	if fileExt == "" {
		fileExt = "txt"
	}

	chunkCh, errCh := p.chunkingService.ChunkStreamByFileType(contentReader, fileExt, nil)
	var chunks []chunking.Chunk

	// Collect chunks
	for chunk := range chunkCh {
		chunks = append(chunks, chunk)
	}

	// Check for streaming errors
	select {
	case err := <-errCh:
		if err != nil {
			return fmt.Errorf("chunking error: %w", err)
		}
	default:
	}

	// Process each chunk
	for _, chunk := range chunks {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		// Skip empty chunks
		if len(strings.TrimSpace(chunk.Content)) == 0 {
			continue
		}

		// Generate embedding
		embedding, err := p.embedder.Embed(chunk.Content)
		if err != nil {
			return fmt.Errorf("failed to generate embedding for chunk %d: %w", chunk.Index, err)
		}

		// Store chunk with embedding
		fileChunk, err := db.UpsertChunk(int64(fileID), chunk.Content, embedding, chunk.Offset, chunk.Length)
		if err != nil {
			return fmt.Errorf("failed to store chunk %d: %w", chunk.Index, err)
		}

		// Add to vector index if search engine is available
		if p.searchEngine != nil {
			if err := p.searchEngine.AddToVectorIndex(job.DirectoryID, fileChunk.ID, embedding); err != nil {
				// Log warning but don't fail the job
				fmt.Printf("Warning: failed to add chunk %d to vector index for %s: %v\n", fileChunk.ID, job.DirectoryID, err)
			}
		}
	}

	return nil
}

// processIndexJob handles search index updates
func (p *AgentFSProcessor) processIndexJob(ctx context.Context, job *Job) error {
	// Get database for this directory
	db, exists := p.databases[job.DirectoryID]
	if !exists {
		return fmt.Errorf("database not found for directory: %s", job.DirectoryID)
	}

	// Rebuild search indexes if needed
	// This could involve optimizing FTS5 indexes, cleaning up old data, etc.
	if err := db.OptimizeIndexes(); err != nil {
		return fmt.Errorf("failed to optimize indexes: %w", err)
	}

	return nil
}

// markFileDeleted marks a file as deleted in the database
func (p *AgentFSProcessor) markFileDeleted(directoryID, filePath string) error {
	db, exists := p.databases[directoryID]
	if !exists {
		return fmt.Errorf("database not found for directory: %s", directoryID)
	}

	return db.SoftDeleteFile(filePath)
}

// processContentStreaming processes file content using streaming chunking and embeddings
func (p *AgentFSProcessor) processContentStreaming(directoryID, filePath string, fileInfo FileInfo, file io.ReadCloser) error {
	defer file.Close()

	// Get appropriate parser
	parser := parsers.GetParser(filePath)
	if parser == nil {
		return fmt.Errorf("unsupported file type for parsing: %s", filePath)
	}

	// Parse content
	content, err := parser.Parse(file)
	if err != nil {
		return fmt.Errorf("failed to parse file: %w", err)
	}

	// Skip empty files
	if len(content) == 0 {
		return nil
	}

	// Reopen file for chunking (since parser consumed it)
	file, err = p.filesystem.Open(filePath)
	if err != nil {
		return fmt.Errorf("failed to reopen file for chunking: %w", err)
	}
	defer file.Close()

	// Parse again for chunking (optimization: could cache parsed content)
	content, err = parser.Parse(file)
	if err != nil {
		return fmt.Errorf("failed to reparse file for chunking: %w", err)
	}

	// Get file extension for chunking strategy
	fileExt := strings.TrimPrefix(filepath.Ext(filePath), ".")
	if fileExt == "" {
		fileExt = "txt"
	}

	// Stream chunks and process them
	contentReader := strings.NewReader(content)
	chunkCh, errCh := p.chunkingService.ChunkStreamByFileType(contentReader, fileExt, nil)

	// Collect chunks and embeddings
	var chunks []database.ChunkData

	// Process chunks as they arrive
	for chunk := range chunkCh {
		// Generate embedding for this chunk
		embedding, err := p.embedder.Embed(chunk.Content)
		if err != nil {
			return fmt.Errorf("failed to generate embedding for chunk %d: %w", chunk.Index, err)
		}

		// Add to chunk data
		chunks = append(chunks, database.ChunkData{
			Content:   chunk.Content,
			Embedding: embedding,
			Offset:    chunk.Offset,
			Length:    chunk.Length,
		})
	}

	// Check for streaming errors
	select {
	case err := <-errCh:
		if err != nil {
			return fmt.Errorf("chunking error: %w", err)
		}
	default:
	}

	// Use update manager to handle file update with soft delete strategy
	updateManager, exists := p.updateManagers[directoryID]
	if !exists {
		return fmt.Errorf("update manager not found for directory: %s", directoryID)
	}

	// Update file and chunks atomically
	fileRecord, err := updateManager.UpdateFile(filePath, fileInfo.Checksum, fileInfo.Size, chunks)
	if err != nil {
		return fmt.Errorf("failed to update file and chunks: %w", err)
	}

	// Add chunks to vector index if search engine is available
	if p.searchEngine != nil {
		db := p.databases[directoryID]
		if db != nil {
			// Get the actual chunk records to get their IDs
			fileChunks, err := db.GetChunksByFileID(fileRecord.ID)
			if err != nil {
				// Log warning but don't fail
				fmt.Printf("Warning: failed to get chunks for vector indexing: %v\n", err)
			} else {
				for _, chunk := range fileChunks {
					if chunk.DeletedAt == nil { // Only index non-deleted chunks
						if len(chunk.Embedding) > 0 {
							if err := p.searchEngine.AddToVectorIndex(directoryID, chunk.ID, chunk.Embedding); err != nil {
								fmt.Printf("Warning: failed to add chunk %d to vector index: %v\n", chunk.ID, err)
							}
						}
					}
				}
			}
		}
	}

	return nil
}

// addEmbedJob adds an embedding job to the queue (deprecated - use streaming processing)
func (p *AgentFSProcessor) addEmbedJob(job *Job) error {
	_, err := p.queue.AddJob(job.Type, job.FilePath, job.DirectoryID, job.Priority, job.Payload)
	return err
}