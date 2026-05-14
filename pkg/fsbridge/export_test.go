package fsbridge

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/neul-labs/stratafs/pkg/config"
	"github.com/neul-labs/stratafs/pkg/database"
)

func TestExportVirtualFS(t *testing.T) {
	tempDir := t.TempDir()
	sourceRoot := filepath.Join(tempDir, "source")
	if err := os.MkdirAll(sourceRoot, 0o755); err != nil {
		t.Fatalf("failed to create source root: %v", err)
	}

	dbPath := filepath.Join(tempDir, "stratafs.db")
	db, err := database.NewDB(dbPath)
	if err != nil {
		t.Fatalf("failed to create db: %v", err)
	}
	defer db.Close()

	filePath := filepath.Join(sourceRoot, "docs", "example.txt")
	fileRecord, err := db.UpsertFile(filePath, "checksum", 42)
	if err != nil {
		t.Fatalf("failed to upsert file: %v", err)
	}

	_, err = db.UpsertChunk(fileRecord.ID, "chunk content", []float32{0.1, 0.2}, 0, 13)
	if err != nil {
		t.Fatalf("failed to upsert chunk: %v", err)
	}

	destDir := filepath.Join(tempDir, "export")
	source := config.StorageSource{
		ID:   "test-source",
		Name: "Test Source",
		Path: sourceRoot,
		Type: config.StorageTypeLocal,
	}

	if err := ExportVirtualFS(db, source, destDir); err != nil {
		t.Fatalf("export failed: %v", err)
	}

	metaPath := filepath.Join(destDir, "docs", "example.txt", "metadata.json")
	if _, err := os.Stat(metaPath); err != nil {
		t.Fatalf("expected metadata file at %s: %v", metaPath, err)
	}

	chunkPath := filepath.Join(destDir, "docs", "example.txt", "_chunks", "chunk_0000.txt")
	data, err := os.ReadFile(chunkPath)
	if err != nil {
		t.Fatalf("expected chunk file: %v", err)
	}
	if string(data) != "chunk content" {
		t.Fatalf("chunk content mismatch: %s", string(data))
	}
}
