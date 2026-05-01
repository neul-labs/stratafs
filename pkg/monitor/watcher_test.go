package monitor

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"

	"agentfs/pkg/config"
	"agentfs/pkg/queue"
)

func TestFileWatcherLifecycle(t *testing.T) {
	tempDir := t.TempDir()
	testDir := filepath.Join(tempDir, "watched")
	queueDir := filepath.Join(tempDir, "queue")

	// Create test directories
	if err := os.MkdirAll(testDir, 0755); err != nil {
		t.Fatalf("Failed to create test directory: %v", err)
	}
	if err := os.MkdirAll(queueDir, 0755); err != nil {
		t.Fatalf("Failed to create queue directory: %v", err)
	}

	// Setup config
	cfg := &config.Config{
		Sources: []config.StorageSource{{
			ID:      "test-local",
			Name:    "Test Directory",
			Type:    config.StorageTypeLocal,
			Enabled: true,
			Path:    testDir,
		}},
		AgentDir: ".agentfs",
		Worker: config.WorkerConfig{
			ScanInterval: 100 * time.Millisecond,
		},
	}

	// Setup queue
	queuePath := filepath.Join(queueDir, "test.db")
	testQueue, err := queue.NewQueue(queuePath)
	if err != nil {
		t.Fatalf("Failed to create test queue: %v", err)
	}
	defer testQueue.Stop()

	// Create file watcher
	watcher, err := NewFileWatcher(cfg, testQueue)
	if err != nil {
		t.Fatalf("Failed to create file watcher: %v", err)
	}
	defer watcher.Stop()

	// Start watching
	if err := watcher.Start(); err != nil {
		t.Fatalf("Failed to start file watcher: %v", err)
	}

	// Test file creation
	t.Run("File Creation", func(t *testing.T) {
		testFile := filepath.Join(testDir, "test.go")
		content := "package main\n\nfunc main() {\n\tprintln(\"Hello, World!\")\n}"

		err := os.WriteFile(testFile, []byte(content), 0644)
		if err != nil {
			t.Fatalf("Failed to create test file: %v", err)
		}

		// Wait for file system event processing
		time.Sleep(200 * time.Millisecond)

		// Check if job was queued
		jobs := getQueuedJobs(t, testQueue)
		if len(jobs) == 0 {
			t.Error("Expected job to be queued for file creation")
		}

		// Verify job details
		found := false
		for _, job := range jobs {
			if job.FilePath == testFile && job.Type == queue.JobTypeParse {
				found = true

				// Verify file info in payload
				var fileInfo queue.FileInfo
				if err := json.Unmarshal([]byte(job.Payload), &fileInfo); err != nil {
					t.Errorf("Failed to parse job payload: %v", err)
				} else {
					if fileInfo.Path != testFile {
						t.Errorf("Expected path %s, got %s", testFile, fileInfo.Path)
					}
					if fileInfo.Size != int64(len(content)) {
						t.Errorf("Expected size %d, got %d", len(content), fileInfo.Size)
					}
				}
				break
			}
		}

		if !found {
			t.Error("Expected parse job for created file")
		}
	})

	// Test file modification
	t.Run("File Modification", func(t *testing.T) {
		testFile := filepath.Join(testDir, "modify.go")

		// Create initial file
		initialContent := "package main\n"
		err := os.WriteFile(testFile, []byte(initialContent), 0644)
		if err != nil {
			t.Fatalf("Failed to create initial file: %v", err)
		}

		time.Sleep(200 * time.Millisecond)

		// Clear existing jobs
		clearQueue(t, testQueue)

		// Modify file
		modifiedContent := "package main\n\nfunc main() {\n\tprintln(\"Modified!\")\n}"
		err = os.WriteFile(testFile, []byte(modifiedContent), 0644)
		if err != nil {
			t.Fatalf("Failed to modify file: %v", err)
		}

		time.Sleep(200 * time.Millisecond)

		// Check if modification was detected
		jobs := getQueuedJobs(t, testQueue)
		found := false
		for _, job := range jobs {
			if job.FilePath == testFile && job.Type == queue.JobTypeParse {
				found = true
				break
			}
		}

		if !found {
			t.Error("Expected job to be queued for file modification")
		}
	})

	// Test file deletion
	t.Run("File Deletion", func(t *testing.T) {
		testFile := filepath.Join(testDir, "delete.go")

		// Create file
		content := "package main\n"
		err := os.WriteFile(testFile, []byte(content), 0644)
		if err != nil {
			t.Fatalf("Failed to create file for deletion: %v", err)
		}

		time.Sleep(200 * time.Millisecond)
		clearQueue(t, testQueue)

		// Delete file
		err = os.Remove(testFile)
		if err != nil {
			t.Fatalf("Failed to delete file: %v", err)
		}

		time.Sleep(200 * time.Millisecond)

		// Check if deletion was detected
		jobs := getQueuedJobs(t, testQueue)
		found := false
		for _, job := range jobs {
			if job.FilePath == testFile {
				// Verify it's a deletion job
				var payload map[string]interface{}
				if err := json.Unmarshal([]byte(job.Payload), &payload); err == nil {
					if action, exists := payload["action"]; exists && action == "delete" {
						found = true
						break
					}
				}
			}
		}

		if !found {
			t.Error("Expected deletion job to be queued for file removal")
		}
	})

	// Test directory creation
	t.Run("Directory Creation", func(t *testing.T) {
		newDir := filepath.Join(testDir, "newdir")

		err := os.Mkdir(newDir, 0755)
		if err != nil {
			t.Fatalf("Failed to create directory: %v", err)
		}

		time.Sleep(200 * time.Millisecond)

		// Create file in new directory
		testFile := filepath.Join(newDir, "test.go")
		content := "package main\n"
		err = os.WriteFile(testFile, []byte(content), 0644)
		if err != nil {
			t.Fatalf("Failed to create file in new directory: %v", err)
		}

		time.Sleep(200 * time.Millisecond)

		// Check if file in new directory was detected
		jobs := getQueuedJobs(t, testQueue)
		found := false
		for _, job := range jobs {
			if job.FilePath == testFile {
				found = true
				break
			}
		}

		if !found {
			t.Error("Expected file in new directory to be detected")
		}
	})
}

func TestSupportedFileTypes(t *testing.T) {
	tempDir := t.TempDir()
	queueDir := filepath.Join(tempDir, "queue")

	// Create queue directory
	if err := os.MkdirAll(queueDir, 0755); err != nil {
		t.Fatalf("Failed to create queue directory: %v", err)
	}

	cfg := &config.Config{
		Sources: []config.StorageSource{{
			ID:      "test-local",
			Name:    "Test Directory",
			Type:    config.StorageTypeLocal,
			Enabled: true,
			Path:    tempDir,
		}},
		AgentDir: ".agentfs",
		Worker: config.WorkerConfig{
			ScanInterval: 100 * time.Millisecond,
		},
	}

	queuePath := filepath.Join(queueDir, "test.db")
	testQueue, err := queue.NewQueue(queuePath)
	if err != nil {
		t.Fatalf("Failed to create test queue: %v", err)
	}
	defer testQueue.Stop()

	watcher, err := NewFileWatcher(cfg, testQueue)
	if err != nil {
		t.Fatalf("Failed to create file watcher: %v", err)
	}
	defer watcher.Stop()

	testCases := []struct {
		filename  string
		content   string
		shouldIndex bool
	}{
		{"test.go", "package main", true},
		{"test.py", "print('hello')", true},
		{"test.js", "console.log('hello')", true},
		{"test.txt", "plain text", true},
		{"test.md", "# Markdown", true},
		{"test.json", `{"key": "value"}`, true},
		{"test.yaml", "key: value", true},
		{"README", "readme content", true},
		{"test.jpg", "binary image data", false},
		{"test.exe", "binary executable", false},
		{"test.zip", "binary archive", false},
	}

	for _, tc := range testCases {
		t.Run(tc.filename, func(t *testing.T) {
			testFile := filepath.Join(tempDir, tc.filename)
			err := os.WriteFile(testFile, []byte(tc.content), 0644)
			if err != nil {
				t.Fatalf("Failed to create test file: %v", err)
			}

			supported := watcher.isSupportedFile(testFile)
			if supported != tc.shouldIndex {
				t.Errorf("File %s: expected supported=%v, got %v", tc.filename, tc.shouldIndex, supported)
			}
		})
	}
}

func TestPeriodicScan(t *testing.T) {
	tempDir := t.TempDir()
	queueDir := filepath.Join(tempDir, "queue")

	// Create queue directory
	if err := os.MkdirAll(queueDir, 0755); err != nil {
		t.Fatalf("Failed to create queue directory: %v", err)
	}

	cfg := &config.Config{
		Sources: []config.StorageSource{{
			ID:      "test-local",
			Name:    "Test Directory",
			Type:    config.StorageTypeLocal,
			Enabled: true,
			Path:    tempDir,
		}},
		AgentDir: ".agentfs",
		Worker: config.WorkerConfig{
			ScanInterval: 50 * time.Millisecond, // Very frequent for testing
		},
	}

	queuePath := filepath.Join(queueDir, "test.db")
	testQueue, err := queue.NewQueue(queuePath)
	if err != nil {
		t.Fatalf("Failed to create test queue: %v", err)
	}
	defer testQueue.Stop()

	watcher, err := NewFileWatcher(cfg, testQueue)
	if err != nil {
		t.Fatalf("Failed to create file watcher: %v", err)
	}
	defer watcher.Stop()

	// Create test file before starting watcher
	testFile := filepath.Join(tempDir, "existing.go")
	content := "package main"
	err = os.WriteFile(testFile, []byte(content), 0644)
	if err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	// Start watcher
	if err := watcher.Start(); err != nil {
		t.Fatalf("Failed to start file watcher: %v", err)
	}

	// Wait for periodic scan to detect existing file
	time.Sleep(200 * time.Millisecond)

	// Check if existing file was detected by periodic scan
	jobs := getQueuedJobs(t, testQueue)
	found := false
	for _, job := range jobs {
		if job.FilePath == testFile {
			found = true
			break
		}
	}

	if !found {
		t.Error("Expected periodic scan to detect existing file")
	}
}

func TestIgnoreAgentDirectories(t *testing.T) {
	tempDir := t.TempDir()
	queueDir := filepath.Join(tempDir, "queue")

	// Create queue directory
	if err := os.MkdirAll(queueDir, 0755); err != nil {
		t.Fatalf("Failed to create queue directory: %v", err)
	}

	cfg := &config.Config{
		Sources: []config.StorageSource{{
			ID:      "test-local",
			Name:    "Test Directory",
			Type:    config.StorageTypeLocal,
			Enabled: true,
			Path:    tempDir,
		}},
		AgentDir: ".agentfs",
		Worker: config.WorkerConfig{
			ScanInterval: 100 * time.Millisecond,
		},
	}

	queuePath := filepath.Join(queueDir, "test.db")
	testQueue, err := queue.NewQueue(queuePath)
	if err != nil {
		t.Fatalf("Failed to create test queue: %v", err)
	}
	defer testQueue.Stop()

	watcher, err := NewFileWatcher(cfg, testQueue)
	if err != nil {
		t.Fatalf("Failed to create file watcher: %v", err)
	}
	defer watcher.Stop()

	if err := watcher.Start(); err != nil {
		t.Fatalf("Failed to start file watcher: %v", err)
	}

	// Create .agentfs directory and file inside it
	agentDir := filepath.Join(tempDir, ".agentfs")
	err = os.Mkdir(agentDir, 0755)
	if err != nil {
		t.Fatalf("Failed to create agent directory: %v", err)
	}

	agentFile := filepath.Join(agentDir, "database.db")
	err = os.WriteFile(agentFile, []byte("database content"), 0644)
	if err != nil {
		t.Fatalf("Failed to create agent file: %v", err)
	}

	time.Sleep(200 * time.Millisecond)

	// Check that no jobs were created for agent directory files
	jobs := getQueuedJobs(t, testQueue)
	for _, job := range jobs {
		if filepath.Dir(job.FilePath) == agentDir {
			t.Errorf("Agent directory file should be ignored: %s", job.FilePath)
		}
	}
}

// Helper functions for tests

func getQueuedJobs(t *testing.T, q *queue.Queue) []*queue.Job {
	stats, err := q.GetQueueStats()
	if err != nil {
		t.Fatalf("Failed to get queue stats: %v", err)
	}

	var jobs []*queue.Job

	// Get pending jobs
	for i := 0; i < int(stats[queue.JobStatusPending]); i++ {
		job, err := q.GetNextJob()
		if err != nil {
			break
		}
		if job == nil {
			break
		}
		jobs = append(jobs, job)
	}

	return jobs
}

func clearQueue(t *testing.T, q *queue.Queue) {
	// Process all pending jobs to clear the queue
	for {
		job, err := q.GetNextJob()
		if err != nil || job == nil {
			break
		}
		// Mark as completed to remove from queue
		q.CompleteJob(job.ID)
	}
}