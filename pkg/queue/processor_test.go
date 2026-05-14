package queue

import (
	"context"
	"crypto/md5"
	"encoding/hex"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/neul-labs/stratafs/pkg/config"
	"github.com/neul-labs/stratafs/pkg/database"
)

type stubEmbedder struct{}

func (stubEmbedder) Embed(text string) ([]float32, error) {
	return []float32{float32(len(text))}, nil
}

func TestProcessParseJobCreatesChunks(t *testing.T) {
	tempDir := t.TempDir()
	cfg := &config.Config{AgentDir: ".stratafs"}

	dbPath := filepath.Join(tempDir, "stratafs.db")
	db, err := database.NewDB(dbPath)
	if err != nil {
		t.Fatalf("failed to create database: %v", err)
	}
	defer db.Close()

	databases := map[string]*database.DB{
		tempDir: db,
	}

	processor := NewStrataFSProcessor(cfg, databases, stubEmbedder{}, nil, nil)

	filePath := filepath.Join(tempDir, "sample.txt")
	content := "hello stratafs!"
	if err := os.WriteFile(filePath, []byte(content), 0o644); err != nil {
		t.Fatalf("failed to write file: %v", err)
	}

	stat, err := os.Stat(filePath)
	if err != nil {
		t.Fatalf("failed to stat file: %v", err)
	}

	sum := md5.Sum([]byte(content))
	fileInfo := FileInfo{
		Path:         filePath,
		Size:         stat.Size(),
		ModifiedTime: stat.ModTime(),
		Checksum:     hex.EncodeToString(sum[:]),
	}
	payloadBytes, _ := json.Marshal(fileInfo)

	job := &Job{
		Type:        JobTypeParse,
		FilePath:    filePath,
		DirectoryID: tempDir,
		Payload:     string(payloadBytes),
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}

	if err := processor.processParseJob(context.Background(), job); err != nil {
		t.Fatalf("processParseJob returned error: %v", err)
	}

	fileRecord, err := db.GetFileByPath(filePath)
	if err != nil {
		t.Fatalf("failed to get file: %v", err)
	}
	if fileRecord == nil {
		t.Fatal("file record not created")
	}

	chunks, err := db.GetChunksByFileID(fileRecord.ID)
	if err != nil {
		t.Fatalf("failed to get chunks: %v", err)
	}
	if len(chunks) == 0 {
		t.Fatal("expected at least one chunk to be created")
	}

	if chunks[0].Content != content {
		t.Fatalf("expected chunk content %q, got %q", content, chunks[0].Content)
	}
}
