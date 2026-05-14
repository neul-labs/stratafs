package monitor

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"path"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/neul-labs/stratafs/pkg/config"
	"github.com/neul-labs/stratafs/pkg/filesystem"
	"github.com/neul-labs/stratafs/pkg/queue"
	"github.com/neul-labs/stratafs/pkg/storage"
)

// RemoteScanner handles periodic scanning and syncing of remote storage sources
type RemoteScanner struct {
	config         *config.Config
	queue          *queue.Queue
	storageFactory *storage.StorageFactory
	scanners       map[string]*sourceScanner // Maps source ID to scanner
	ctx            context.Context
	cancel         context.CancelFunc
	wg             sync.WaitGroup
}

// sourceScanner handles scanning for a single remote source
type sourceScanner struct {
	source     config.StorageSource
	filesystem filesystem.FileSystem
	lastScan   time.Time
	scanner    *RemoteScanner
}

// NewRemoteScanner creates a new remote source scanner
func NewRemoteScanner(cfg *config.Config, jobQueue *queue.Queue) (*RemoteScanner, error) {
	ctx, cancel := context.WithCancel(context.Background())

	rs := &RemoteScanner{
		config:         cfg,
		queue:          jobQueue,
		storageFactory: storage.NewStorageFactory(),
		scanners:       make(map[string]*sourceScanner),
		ctx:            ctx,
		cancel:         cancel,
	}

	// Initialize scanners for remote sources
	for _, source := range cfg.GetEnabledSources() {
		if source.Type != config.StorageTypeLocal {
			if err := rs.addSource(source); err != nil {
				log.Printf("Warning: Failed to add remote source %s: %v", source.Name, err)
			}
		}
	}

	return rs, nil
}

// addSource adds a remote source to be scanned
func (rs *RemoteScanner) addSource(source config.StorageSource) error {
	// Validate source credentials
	if err := rs.storageFactory.ValidateSourceCredentials(source); err != nil {
		return fmt.Errorf("invalid credentials for source %s: %w", source.Name, err)
	}

	// Create filesystem for this source
	fs, err := rs.storageFactory.CreateFileSystem(source)
	if err != nil {
		return fmt.Errorf("failed to create filesystem for source %s: %w", source.Name, err)
	}

	// Create source scanner
	scanner := &sourceScanner{
		source:     source,
		filesystem: fs,
		lastScan:   time.Time{}, // Never scanned before
		scanner:    rs,
	}

	rs.scanners[source.ID] = scanner
	log.Printf("Added remote source for scanning: %s (%s)", source.Name, source.Type)
	return nil
}

// Start begins periodic scanning of all remote sources
func (rs *RemoteScanner) Start() error {
	if len(rs.scanners) == 0 {
		log.Println("No remote sources to scan")
		return nil
	}

	log.Printf("Starting remote scanner for %d sources", len(rs.scanners))

	// Start individual scanners for each source
	for sourceID, scanner := range rs.scanners {
		rs.wg.Add(1)
		go func(id string, s *sourceScanner) {
			defer rs.wg.Done()
			rs.runSourceScanner(id, s)
		}(sourceID, scanner)
	}

	log.Println("Remote scanning started")
	return nil
}

// runSourceScanner runs the scanning loop for a single source
func (rs *RemoteScanner) runSourceScanner(sourceID string, scanner *sourceScanner) {
	// Do an initial scan immediately
	if err := rs.scanSource(scanner); err != nil {
		log.Printf("Initial scan failed for source %s: %v", scanner.source.Name, err)
	}

	// Set up periodic scanning
	ticker := time.NewTicker(rs.config.Worker.ScanInterval)
	defer ticker.Stop()

	for {
		select {
		case <-rs.ctx.Done():
			log.Printf("Stopping scanner for source %s", scanner.source.Name)
			return
		case <-ticker.C:
			if err := rs.scanSource(scanner); err != nil {
				log.Printf("Scan failed for source %s: %v", scanner.source.Name, err)
			}
		}
	}
}

// scanSource performs a full scan of a remote source
func (rs *RemoteScanner) scanSource(scanner *sourceScanner) error {
	log.Printf("Scanning remote source: %s", scanner.source.Name)
	startTime := time.Now()

	// Track scan statistics
	var filesFound, filesProcessed, filesSkipped int

	// Walk the remote filesystem
	err := scanner.filesystem.Walk("", func(path string, info filesystem.FileInfo, err error) error {
		if err != nil {
			log.Printf("Walk error for %s: %v", path, err)
			return nil // Continue walking despite errors
		}

		// Skip directories
		if info.IsDir() {
			return nil
		}

		filesFound++

		// Check if file should be processed
		if !rs.shouldProcessFile(scanner.source, path, info) {
			filesSkipped++
			return nil
		}

		// Check if file has changed since last scan
		if !rs.hasFileChanged(scanner, path, info) {
			filesSkipped++
			return nil
		}

		// Queue file for processing
		if err := rs.queueFileForProcessing(scanner, path, info); err != nil {
			log.Printf("Failed to queue file %s: %v", path, err)
			return nil // Continue processing other files
		}

		filesProcessed++
		return nil
	})

	if err != nil {
		return fmt.Errorf("failed to walk remote source %s: %w", scanner.source.Name, err)
	}

	// Update last scan time
	scanner.lastScan = startTime

	duration := time.Since(startTime)
	log.Printf("Scan completed for %s: %d files found, %d processed, %d skipped (took %v)",
		scanner.source.Name, filesFound, filesProcessed, filesSkipped, duration)

	return nil
}

// shouldProcessFile determines if a file should be processed based on filters
func (rs *RemoteScanner) shouldProcessFile(source config.StorageSource, path string, info filesystem.FileInfo) bool {
	// Check file size limit
	if source.Filters.MaxFileSize > 0 && info.Size() > source.Filters.MaxFileSize {
		return false
	}

	// Check if hidden files should be ignored
	if source.Filters.IgnoreHidden && isHiddenFile(path) {
		return false
	}

	// Check include patterns
	if len(source.Filters.IncludePatterns) > 0 {
		matched := false
		for _, pattern := range source.Filters.IncludePatterns {
			if matches, _ := filepath.Match(pattern, filepath.Base(path)); matches {
				matched = true
				break
			}
		}
		if !matched {
			return false
		}
	}

	// Check exclude patterns
	for _, pattern := range source.Filters.ExcludePatterns {
		if matches, _ := filepath.Match(pattern, path); matches {
			return false
		}
	}

	return true
}

// hasFileChanged checks if a file has changed since the last scan
func (rs *RemoteScanner) hasFileChanged(scanner *sourceScanner, path string, info filesystem.FileInfo) bool {
	// If this is the first scan, process all files
	if scanner.lastScan.IsZero() {
		return true
	}

	// Check if file was modified after last scan
	return info.ModTime().After(scanner.lastScan)
}

// queueFileForProcessing downloads the file to cache and queues it for processing
func (rs *RemoteScanner) queueFileForProcessing(scanner *sourceScanner, remotePath string, info filesystem.FileInfo) error {
	// Get the local cache path for this file
	cachePath, err := rs.getCachePath(scanner.source, remotePath)
	if err != nil {
		return fmt.Errorf("invalid remote path %s: %w", remotePath, err)
	}

	// Ensure cache directory exists
	cacheDir := filepath.Dir(cachePath)
	if err := os.MkdirAll(cacheDir, 0755); err != nil {
		return fmt.Errorf("failed to create cache directory: %w", err)
	}

	// Download file to cache
	if err := rs.downloadFileToCache(scanner.filesystem, remotePath, cachePath); err != nil {
		return fmt.Errorf("failed to download file to cache: %w", err)
	}

	// Create payload with metadata including cleanup flag
	payload := map[string]interface{}{
		"cleanup_after_processing": true, // Flag to delete after processing
		"original_remote_path":     remotePath,
		"source_name":              scanner.source.Name,
		"source_type":              string(scanner.source.Type),
	}
	payloadJSON, _ := json.Marshal(payload)

	// Queue the processing job
	_, err = rs.queue.AddJob(queue.JobTypeParse, cachePath, scanner.source.ID, 3, string(payloadJSON))
	if err != nil {
		// Clean up cache file if queueing fails
		_ = os.Remove(cachePath)
		return fmt.Errorf("failed to add job to queue: %w", err)
	}

	return nil
}

// downloadFileToCache downloads a remote file to the local cache
func (rs *RemoteScanner) downloadFileToCache(fs filesystem.FileSystem, remotePath, cachePath string) error {
	// Open remote file
	remoteFile, err := fs.Open(remotePath)
	if err != nil {
		return fmt.Errorf("failed to open remote file: %w", err)
	}
	defer remoteFile.Close()

	// Create cache file
	cacheFile, err := os.Create(cachePath)
	if err != nil {
		return fmt.Errorf("failed to create cache file: %w", err)
	}
	defer cacheFile.Close()

	// Copy data
	if _, err := cacheFile.ReadFrom(remoteFile); err != nil {
		return fmt.Errorf("failed to copy file data: %w", err)
	}

	return nil
}

// getCachePath returns the local cache path for a remote file
func (rs *RemoteScanner) getCachePath(source config.StorageSource, remotePath string) (string, error) {
	sanitized, err := sanitizeRemotePath(remotePath)
	if err != nil {
		return "", err
	}
	return filepath.Join(source.LocalCacheDir, sanitized), nil
}

// isHiddenFile checks if a file/directory is hidden
func isHiddenFile(path string) bool {
	base := filepath.Base(path)
	return len(base) > 0 && base[0] == '.'
}

// sanitizeRemotePath ensures the remote path cannot escape the local cache directory
func sanitizeRemotePath(remotePath string) (string, error) {
	cleaned := path.Clean(remotePath)
	cleaned = strings.ReplaceAll(cleaned, "\\", "/")

	for strings.HasPrefix(cleaned, "./") {
		cleaned = strings.TrimPrefix(cleaned, "./")
	}

	if cleaned == "." || cleaned == "" {
		return "", fmt.Errorf("remote path cannot be empty")
	}
	if cleaned == ".." || strings.HasPrefix(cleaned, "../") {
		return "", fmt.Errorf("remote path escapes cache")
	}
	if strings.HasPrefix(cleaned, "/") {
		cleaned = strings.TrimPrefix(cleaned, "/")
		if cleaned == "" {
			return "", fmt.Errorf("remote path cannot be root")
		}
	}

	return filepath.FromSlash(cleaned), nil
}

// Stop stops all remote scanning
func (rs *RemoteScanner) Stop() {
	log.Println("Stopping remote scanner...")
	rs.cancel()
	rs.wg.Wait()
	log.Println("Remote scanner stopped")
}

// GetScanStats returns scanning statistics for all sources
func (rs *RemoteScanner) GetScanStats() map[string]SourceScanStats {
	stats := make(map[string]SourceScanStats)
	for sourceID, scanner := range rs.scanners {
		stats[sourceID] = SourceScanStats{
			SourceID:   sourceID,
			SourceName: scanner.source.Name,
			SourceType: string(scanner.source.Type),
			LastScan:   scanner.lastScan,
			NextScan:   scanner.lastScan.Add(rs.config.Worker.ScanInterval),
		}
	}
	return stats
}

// SourceScanStats represents scanning statistics for a source
type SourceScanStats struct {
	SourceID   string    `json:"source_id"`
	SourceName string    `json:"source_name"`
	SourceType string    `json:"source_type"`
	LastScan   time.Time `json:"last_scan"`
	NextScan   time.Time `json:"next_scan"`
}
