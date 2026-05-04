package fsbridge

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"agentfs/pkg/config"
	"agentfs/pkg/database"
)

// FileMetadata represents the metadata persisted for each file during export.
type FileMetadata struct {
	Path        string            `json:"path"`
	Relative    string            `json:"relative_path"`
	Checksum    string            `json:"checksum"`
	Size        int64             `json:"size"`
	CreatedAt   time.Time         `json:"created_at"`
	UpdatedAt   time.Time         `json:"updated_at"`
	ChunkCount  int               `json:"chunk_count"`
	ChunkFiles  []ChunkFileEntry  `json:"chunk_files"`
	ExtraFields map[string]string `json:"extra,omitempty"`
}

// ChunkFileEntry captures the on-disk chunk representation.
type ChunkFileEntry struct {
	Index  int    `json:"index"`
	Offset int    `json:"offset"`
	Length int    `json:"length"`
	File   string `json:"file"`
}

// ExportVirtualFS mirrors the indexed files for a source into outputDir with metadata and chunk files.
func ExportVirtualFS(db *database.DB, source config.StorageSource, outputDir string) error {
	if outputDir == "" {
		return fmt.Errorf("output directory cannot be empty")
	}

	if err := os.MkdirAll(outputDir, 0o755); err != nil {
		return fmt.Errorf("failed to create output directory: %w", err)
	}

	if err := writeSourceMetadata(outputDir, source); err != nil {
		return err
	}

	files, err := db.ListFiles(false)
	if err != nil {
		return fmt.Errorf("failed to list files: %w", err)
	}

	for _, file := range files {
		if err := exportFile(db, source, file, outputDir); err != nil {
			return err
		}
	}

	return nil
}

func writeSourceMetadata(outputDir string, source config.StorageSource) error {
	payload := map[string]interface{}{
		"id":        source.ID,
		"name":      source.Name,
		"path":      source.Path,
		"type":      source.Type,
		"filters":   source.Filters,
		"generated": time.Now().UTC(),
	}

	data, err := json.MarshalIndent(payload, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal source metadata: %w", err)
	}

	return os.WriteFile(filepath.Join(outputDir, "source.json"), data, 0o644)
}

func exportFile(db *database.DB, source config.StorageSource, file *database.File, outputDir string) error {
	relative, err := filepath.Rel(source.Path, file.Path)
	if err != nil || strings.HasPrefix(relative, "..") {
		// Skip files that are outside the scoped source path.
		return nil
	}

	destDir := filepath.Join(outputDir, relative)
	if err := os.MkdirAll(destDir, 0o755); err != nil {
		return fmt.Errorf("failed to create directory for %s: %w", relative, err)
	}

	chunks, err := db.GetChunksByFileID(file.ID)
	if err != nil {
		return fmt.Errorf("failed to get chunks for %s: %w", file.Path, err)
	}

	chunkDir := filepath.Join(destDir, "_chunks")
	if err := os.MkdirAll(chunkDir, 0o755); err != nil {
		return fmt.Errorf("failed to create chunk directory for %s: %w", file.Path, err)
	}

	var chunkEntries []ChunkFileEntry
	for idx, chunk := range chunks {
		fileName := fmt.Sprintf("chunk_%04d.txt", idx)
		chunkPath := filepath.Join(chunkDir, fileName)
		if err := os.WriteFile(chunkPath, []byte(chunk.Content), 0o644); err != nil {
			return fmt.Errorf("failed to write chunk for %s: %w", file.Path, err)
		}
		chunkEntries = append(chunkEntries, ChunkFileEntry{
			Index:  idx,
			Offset: chunk.Offset,
			Length: chunk.Length,
			File:   fileName,
		})
	}

	meta := FileMetadata{
		Path:       file.Path,
		Relative:   relative,
		Checksum:   file.Checksum,
		Size:       file.Size,
		CreatedAt:  file.CreatedAt,
		UpdatedAt:  file.UpdatedAt,
		ChunkCount: len(chunkEntries),
		ChunkFiles: chunkEntries,
	}

	data, err := json.MarshalIndent(meta, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal metadata for %s: %w", file.Path, err)
	}

	return os.WriteFile(filepath.Join(destDir, "metadata.json"), data, 0o644)
}
