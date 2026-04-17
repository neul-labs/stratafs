package filesystem

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"time"
)

// HybridFileSystem wraps a remote filesystem with local caching for directory structure compatibility
type HybridFileSystem struct {
	remote    FileSystem // The actual remote filesystem (S3, GCS, etc.)
	local     FileSystem // Local filesystem for cache directory
	cacheDir  string     // Local cache directory path
	sourceID  string     // Source ID for cache organization
}

// NewHybridFileSystem creates a new hybrid filesystem that mirrors remote content locally
func NewHybridFileSystem(remote FileSystem, cacheDir, sourceID string) *HybridFileSystem {
	return &HybridFileSystem{
		remote:   remote,
		local:    NewLocalFileSystem(),
		cacheDir: cacheDir,
		sourceID: sourceID,
	}
}

// getCachePath converts a remote path to a local cache path
func (hfs *HybridFileSystem) getCachePath(remotePath string) string {
	// Create a safe cache path by replacing problematic characters
	safePath := filepath.Clean(remotePath)
	return filepath.Join(hfs.cacheDir, safePath)
}

// isCached checks if a file exists in the local cache and is up-to-date
func (hfs *HybridFileSystem) isCached(remotePath string) (bool, error) {
	cachePath := hfs.getCachePath(remotePath)

	// Check if cache file exists
	cacheInfo, err := hfs.local.Stat(cachePath)
	if err != nil {
		return false, nil // Cache miss
	}

	// Check if remote file exists and get its modification time
	remoteInfo, err := hfs.remote.Stat(remotePath)
	if err != nil {
		return false, err // Remote file doesn't exist or error
	}

	// Compare modification times
	return !remoteInfo.ModTime().After(cacheInfo.ModTime()), nil
}

// cacheFile downloads a file from remote to local cache
func (hfs *HybridFileSystem) cacheFile(remotePath string) error {
	cachePath := hfs.getCachePath(remotePath)

	// Ensure cache directory exists
	if err := hfs.local.MkdirAll(hfs.local.Dir(cachePath), 0755); err != nil {
		return fmt.Errorf("failed to create cache directory: %w", err)
	}

	// Open remote file
	remoteFile, err := hfs.remote.Open(remotePath)
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

	// Copy content
	if _, err := io.Copy(cacheFile, remoteFile); err != nil {
		return fmt.Errorf("failed to copy file content to cache: %w", err)
	}

	// Set modification time to match remote file
	remoteInfo, err := hfs.remote.Stat(remotePath)
	if err == nil {
		os.Chtimes(cachePath, time.Now(), remoteInfo.ModTime())
	}

	return nil
}

// ReadFile reads the content of a file, caching it locally if needed
func (hfs *HybridFileSystem) ReadFile(name string) ([]byte, error) {
	// Check if file is cached and up-to-date
	cached, err := hfs.isCached(name)
	if err != nil {
		return nil, err
	}

	if !cached {
		// Cache the file
		if err := hfs.cacheFile(name); err != nil {
			return nil, fmt.Errorf("failed to cache file: %w", err)
		}
	}

	// Read from cache
	cachePath := hfs.getCachePath(name)
	return hfs.local.ReadFile(cachePath)
}

// Open opens a file for reading, caching it locally if needed
func (hfs *HybridFileSystem) Open(name string) (File, error) {
	// Check if file is cached and up-to-date
	cached, err := hfs.isCached(name)
	if err != nil {
		return nil, err
	}

	if !cached {
		// Cache the file
		if err := hfs.cacheFile(name); err != nil {
			return nil, fmt.Errorf("failed to cache file: %w", err)
		}
	}

	// Open from cache
	cachePath := hfs.getCachePath(name)
	return hfs.local.Open(cachePath)
}

// Stat returns information about a file (from remote source)
func (hfs *HybridFileSystem) Stat(name string) (FileInfo, error) {
	return hfs.remote.Stat(name)
}

// Walk walks the file tree, caching files as needed
func (hfs *HybridFileSystem) Walk(root string, walkFn WalkFunc) error {
	return hfs.remote.Walk(root, func(path string, info FileInfo, err error) error {
		if err != nil {
			return walkFn(path, info, err)
		}

		// For regular files, ensure they're cached
		if !info.IsDir() {
			cached, cacheErr := hfs.isCached(path)
			if cacheErr == nil && !cached {
				// Asynchronously cache the file (don't block walking)
				go func(p string) {
					hfs.cacheFile(p)
				}(path)
			}
		}

		return walkFn(path, info, err)
	})
}

// MkdirAll creates a directory path in the local cache
func (hfs *HybridFileSystem) MkdirAll(path string, perm os.FileMode) error {
	cachePath := hfs.getCachePath(path)
	return hfs.local.MkdirAll(cachePath, perm)
}

// Join joins any number of path elements into a single path
func (hfs *HybridFileSystem) Join(elem ...string) string {
	return hfs.remote.Join(elem...)
}

// Base returns the last element of path
func (hfs *HybridFileSystem) Base(path string) string {
	return hfs.remote.Base(path)
}

// Dir returns all but the last element of path
func (hfs *HybridFileSystem) Dir(path string) string {
	return hfs.remote.Dir(path)
}

// Ext returns the file name extension used by path
func (hfs *HybridFileSystem) Ext(path string) string {
	return hfs.remote.Ext(path)
}

// GetCacheDir returns the local cache directory path
func (hfs *HybridFileSystem) GetCacheDir() string {
	return hfs.cacheDir
}

// SyncCache ensures all remote files are cached locally
func (hfs *HybridFileSystem) SyncCache(root string) error {
	return hfs.remote.Walk(root, func(path string, info FileInfo, err error) error {
		if err != nil {
			return err
		}

		// Skip directories
		if info.IsDir() {
			return nil
		}

		// Check if file needs caching
		cached, err := hfs.isCached(path)
		if err != nil {
			return err
		}

		if !cached {
			if err := hfs.cacheFile(path); err != nil {
				return fmt.Errorf("failed to cache file %s: %w", path, err)
			}
		}

		return nil
	})
}

// PurgeCache removes all cached files for this source
func (hfs *HybridFileSystem) PurgeCache() error {
	return os.RemoveAll(hfs.cacheDir)
}

// GetCacheStats returns statistics about the local cache
func (hfs *HybridFileSystem) GetCacheStats() (CacheStats, error) {
	stats := CacheStats{
		SourceID: hfs.sourceID,
		CacheDir: hfs.cacheDir,
	}

	// Walk the cache directory to collect stats
	err := filepath.Walk(hfs.cacheDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		if !info.IsDir() {
			stats.FileCount++
			stats.TotalSize += info.Size()
		}

		return nil
	})

	return stats, err
}

// CacheStats represents cache statistics
type CacheStats struct {
	SourceID  string `json:"source_id"`
	CacheDir  string `json:"cache_dir"`
	FileCount int64  `json:"file_count"`
	TotalSize int64  `json:"total_size"`
}