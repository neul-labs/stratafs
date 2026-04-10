package queue

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"time"

	"agentfs/internal/utils"
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
	config       *config.Config
	databases    map[string]*database.DB
	embedder     *embeddings.Embedder
	filesystem   filesystem.FileSystem
	queue        *Queue
	searchEngine *search.Engine
}

// NewAgentFSProcessor creates a new processor
func NewAgentFSProcessor(cfg *config.Config, databases map[string]*database.DB, embedder *embeddings.Embedder, queue *Queue, searchEngine *search.Engine) *AgentFSProcessor {
	return &AgentFSProcessor{
		config:       cfg,
		databases:    databases,
		embedder:     embedder,
		filesystem:   filesystem.NewLocalFileSystem(),
		queue:        queue,
		searchEngine: searchEngine,
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
	// Parse file info from payload
	var fileInfo FileInfo
	if err := json.Unmarshal([]byte(job.Payload), &fileInfo); err != nil {
		return fmt.Errorf("failed to parse file info: %w", err)
	}

	// Check if file still exists and hasn't changed
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

	// Get database for this directory
	db, exists := p.databases[job.DirectoryID]
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
		return nil
	}

	// Upsert file record in database
	fileRecord, err := db.UpsertFile(job.FilePath, fileInfo.Checksum, fileInfo.Size)
	if err != nil {
		return fmt.Errorf("failed to upsert file record: %w", err)
	}

	// Store parsed content for embedding job
	embedPayload := map[string]interface{}{
		"file_id": fileRecord.ID,
		"content": content,
		"file_path": job.FilePath,
	}

	payloadBytes, err := json.Marshal(embedPayload)
	if err != nil {
		return fmt.Errorf("failed to marshal embed payload: %w", err)
	}

	// Create embedding job
	embedJob := &Job{
		Type:        JobTypeEmbed,
		FilePath:    job.FilePath,
		DirectoryID: job.DirectoryID,
		Priority:    job.Priority,
		Payload:     string(payloadBytes),
	}

	return p.addEmbedJob(embedJob)
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

	// Chunk the content
	chunkOptions := utils.DefaultChunkOptions()
	chunks := utils.ChunkText(content, chunkOptions)

	// Process each chunk
	for i, chunk := range chunks {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		// Skip empty chunks
		if len(chunk) == 0 {
			continue
		}

		// Generate embedding
		embedding, err := p.embedder.Embed(chunk)
		if err != nil {
			return fmt.Errorf("failed to generate embedding for chunk %d: %w", i, err)
		}

		// Store chunk with embedding
		fileChunk, err := db.UpsertChunk(int64(fileID), chunk, embedding, i*chunkOptions.ChunkSize, len(chunk))
		if err != nil {
			return fmt.Errorf("failed to store chunk %d: %w", i, err)
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

// addEmbedJob adds an embedding job to the queue
func (p *AgentFSProcessor) addEmbedJob(job *Job) error {
	_, err := p.queue.AddJob(job.Type, job.FilePath, job.DirectoryID, job.Priority, job.Payload)
	return err
}