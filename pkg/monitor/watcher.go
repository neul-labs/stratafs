package monitor

import (
	"context"
	"crypto/md5"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"time"

	"github.com/fsnotify/fsnotify"

	"github.com/neul-labs/stratafs/pkg/config"
	"github.com/neul-labs/stratafs/pkg/filesystem"
	"github.com/neul-labs/stratafs/pkg/parsers"
	"github.com/neul-labs/stratafs/pkg/queue"
)

// FileWatcher provides cross-platform file watching capabilities
type FileWatcher struct {
	config     *config.Config
	queue      *queue.Queue
	watcher    *fsnotify.Watcher
	filesystem filesystem.FileSystem
	ctx        context.Context
	cancel     context.CancelFunc
}

// NewFileWatcher creates a new file watcher for local sources
func NewFileWatcher(cfg *config.Config, jobQueue *queue.Queue) (*FileWatcher, error) {
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		return nil, fmt.Errorf("failed to create file watcher: %w", err)
	}

	ctx, cancel := context.WithCancel(context.Background())

	fw := &FileWatcher{
		config:     cfg,
		queue:      jobQueue,
		watcher:    watcher,
		filesystem: filesystem.NewLocalFileSystem(),
		ctx:        ctx,
		cancel:     cancel,
	}

	return fw, nil
}

// getLocalSources returns all enabled local storage sources
func (fw *FileWatcher) getLocalSources() []config.StorageSource {
	var localSources []config.StorageSource
	for _, source := range fw.config.GetEnabledSources() {
		if source.Type == config.StorageTypeLocal {
			localSources = append(localSources, source)
		}
	}
	return localSources
}

// StartForSource begins watching a specific local storage source
func (fw *FileWatcher) StartForSource(source config.StorageSource) error {
	if source.Type != config.StorageTypeLocal {
		return fmt.Errorf("file watcher only supports local storage sources")
	}

	if err := fw.addDirectoryRecursive(source.Path); err != nil {
		return fmt.Errorf("failed to watch directory %s: %w", source.Path, err)
	}

	go fw.processEvents()
	return nil
}

// Start begins watching all enabled local storage sources
func (fw *FileWatcher) Start() error {
	// Get all enabled local sources
	localSources := fw.getLocalSources()
	if len(localSources) == 0 {
		log.Println("No local storage sources found to watch")
		return nil
	}

	// Add all local sources to the watcher
	for _, source := range localSources {
		if err := fw.addDirectoryRecursive(source.Path); err != nil {
			log.Printf("Warning: Failed to watch directory %s: %v", source.Path, err)
		}
	}

	// Start event processing goroutine
	go fw.processEvents()

	// Start periodic scan goroutine
	go fw.periodicScan()

	return nil
}

// addDirectoryRecursive adds a directory and all its subdirectories to the watcher
func (fw *FileWatcher) addDirectoryRecursive(root string) error {
	return filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		// Skip .stratafs directories
		if info.IsDir() && info.Name() == fw.config.AgentDir {
			return filepath.SkipDir
		}

		// Add directories to watcher
		if info.IsDir() {
			if err := fw.watcher.Add(path); err != nil {
				log.Printf("Warning: Failed to watch directory %s: %v", path, err)
			}
		}

		return nil
	})
}

// processEvents handles file system events
func (fw *FileWatcher) processEvents() {
	for {
		select {
		case <-fw.ctx.Done():
			return
		case event, ok := <-fw.watcher.Events:
			if !ok {
				return
			}
			fw.handleEvent(event)
		case err, ok := <-fw.watcher.Errors:
			if !ok {
				return
			}
			log.Printf("File watcher error: %v", err)
		}
	}
}

// handleEvent processes a single file system event
func (fw *FileWatcher) handleEvent(event fsnotify.Event) {
	// Skip events for .stratafs directories
	if filepath.Base(filepath.Dir(event.Name)) == fw.config.AgentDir ||
	   filepath.Base(event.Name) == fw.config.AgentDir {
		return
	}

	// Handle different event types
	switch {
	case event.Has(fsnotify.Create):
		fw.handleCreate(event.Name)
	case event.Has(fsnotify.Write):
		fw.handleWrite(event.Name)
	case event.Has(fsnotify.Remove):
		fw.handleRemove(event.Name)
	case event.Has(fsnotify.Rename):
		fw.handleRename(event.Name)
	}
}

// handleCreate processes file/directory creation events
func (fw *FileWatcher) handleCreate(path string) {
	info, err := os.Stat(path)
	if err != nil {
		return // File might have been deleted already
	}

	if info.IsDir() {
		// Add new directory to watcher
		if err := fw.addDirectoryRecursive(path); err != nil {
			log.Printf("Warning: Failed to watch new directory %s: %v", path, err)
		}
	} else {
		// Queue file for processing
		fw.queueFileJob(path, queue.JobTypeParse, 5)
	}
}

// handleWrite processes file modification events
func (fw *FileWatcher) handleWrite(path string) {
	info, err := os.Stat(path)
	if err != nil {
		return // File might have been deleted
	}

	if !info.IsDir() && fw.isSupportedFile(path) {
		// Queue file for processing
		fw.queueFileJob(path, queue.JobTypeParse, 5)
	}
}

// handleRemove processes file/directory removal events
func (fw *FileWatcher) handleRemove(path string) {
	// Create a job to mark the file as deleted
	directoryID := fw.getDirectoryID(path)
	if directoryID == "" {
		return
	}

	payload := map[string]interface{}{
		"action": "delete",
		"path":   path,
	}

	payloadBytes, _ := json.Marshal(payload)

	_, err := fw.queue.AddJob(queue.JobTypeParse, path, directoryID, 1, string(payloadBytes))
	if err != nil {
		log.Printf("Failed to queue delete job for %s: %v", path, err)
	}
}

// handleRename processes file/directory rename events
func (fw *FileWatcher) handleRename(path string) {
	// Treat rename as a delete (old name) + create (new name)
	fw.handleRemove(path)
}

// queueFileJob creates and queues a job for file processing
func (fw *FileWatcher) queueFileJob(path string, jobType queue.JobType, priority int) {
	if !fw.isSupportedFile(path) {
		return
	}

	directoryID := fw.getDirectoryID(path)
	if directoryID == "" {
		return
	}

	// Get file info
	stat, err := os.Stat(path)
	if err != nil {
		log.Printf("Failed to stat file %s: %v", path, err)
		return
	}

	// Calculate checksum
	checksum, err := fw.calculateChecksum(path)
	if err != nil {
		log.Printf("Failed to calculate checksum for %s: %v", path, err)
		return
	}

	// Create file info payload
	fileInfo := queue.FileInfo{
		Path:         path,
		Size:         stat.Size(),
		ModifiedTime: stat.ModTime(),
		Checksum:     checksum,
	}

	payload, err := json.Marshal(fileInfo)
	if err != nil {
		log.Printf("Failed to marshal file info for %s: %v", path, err)
		return
	}

	// Queue the job
	_, err = fw.queue.AddJob(jobType, path, directoryID, priority, string(payload))
	if err != nil {
		log.Printf("Failed to queue job for %s: %v", path, err)
	} else {
		log.Printf("Queued %s job for file: %s", jobType, path)
	}
}

// periodicScan performs periodic directory scans to catch missed events
func (fw *FileWatcher) periodicScan() {
	ticker := time.NewTicker(fw.config.Worker.ScanInterval)
	defer ticker.Stop()

	for {
		select {
		case <-fw.ctx.Done():
			return
		case <-ticker.C:
			fw.performScan()
		}
	}
}

// performScan scans all directories for changes
func (fw *FileWatcher) performScan() {
	for _, source := range fw.getLocalSources() {
		dir := source.Path
		fw.scanDirectory(dir)
	}
}

// scanDirectory scans a single directory for changes
func (fw *FileWatcher) scanDirectory(dir string) {
	err := filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		// Skip .stratafs directories
		if info.IsDir() && info.Name() == fw.config.AgentDir {
			return filepath.SkipDir
		}

		// Skip directories
		if info.IsDir() {
			return nil
		}

		// Check if file should be processed
		if fw.isSupportedFile(path) {
			fw.queueFileJob(path, queue.JobTypeParse, 1) // Lower priority for scan
		}

		return nil
	})

	if err != nil {
		log.Printf("Error scanning directory %s: %v", dir, err)
	}
}

// isSupportedFile checks if a file type is supported for indexing
func (fw *FileWatcher) isSupportedFile(path string) bool {
	// Use the parser's file type detection
	parser := parsers.GetParser(path)
	return parser != nil // If parser is nil, file is not supported
}

// getDirectoryID returns the directory ID for a given file path
func (fw *FileWatcher) getDirectoryID(filePath string) string {
	// Find which configured directory contains this file
	for _, source := range fw.getLocalSources() {
		dir := source.Path
		if isSubPath(filePath, dir) {
			return dir
		}
	}
	return ""
}

// isSubPath checks if path is under root directory
func isSubPath(path, root string) bool {
	abs1, err1 := filepath.Abs(path)
	abs2, err2 := filepath.Abs(root)
	if err1 != nil || err2 != nil {
		return false
	}

	rel, err := filepath.Rel(abs2, abs1)
	if err != nil {
		return false
	}

	return !filepath.IsAbs(rel) && !filepath.HasPrefix(rel, "..")
}

// calculateChecksum calculates MD5 checksum of a file
func (fw *FileWatcher) calculateChecksum(filePath string) (string, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return "", err
	}
	defer file.Close()

	hash := md5.New()
	if _, err := io.Copy(hash, file); err != nil {
		return "", err
	}

	return fmt.Sprintf("%x", hash.Sum(nil)), nil
}

// Stop stops the file watcher
func (fw *FileWatcher) Stop() {
	fw.cancel()
	if fw.watcher != nil {
		fw.watcher.Close()
	}
}