//go:build windows
// +build windows

package fsbridge

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"github.com/neul-labs/stratafs/pkg/config"
	"github.com/neul-labs/stratafs/pkg/database"

	"github.com/winfsp/cgofuse/fuse"
)

// WinFspMount represents a WinFsp filesystem mount
type WinFspMount struct {
	db           *database.DB
	source       config.StorageSource
	showChunks   bool
	showMetadata bool
	fs           *strataFSWin
	host         *fuse.FileSystemHost
}

// NewWinFspMount creates a new WinFsp mount
func NewWinFspMount(db *database.DB, source config.StorageSource, showChunks, showMetadata bool) *WinFspMount {
	return &WinFspMount{
		db:           db,
		source:       source,
		showChunks:   showChunks,
		showMetadata: showMetadata,
	}
}

// Mount mounts the filesystem at the specified drive letter or path
func (m *WinFspMount) Mount(mountPoint string, allowOther, debug bool) error {
	m.fs = &strataFSWin{
		db:           m.db,
		source:       m.source,
		showChunks:   m.showChunks,
		showMetadata: m.showMetadata,
		openFiles:    make(map[uint64]*openFile),
		nextHandle:   1,
	}

	m.host = fuse.NewFileSystemHost(m.fs)

	// Build mount options
	opts := []string{"-o", "volname=StrataFS"}
	if debug {
		opts = append(opts, "-d")
	}

	// Mount in background
	go func() {
		m.host.Mount(mountPoint, opts)
	}()

	return nil
}

// Unmount unmounts the filesystem
func (m *WinFspMount) Unmount() error {
	if m.host != nil {
		m.host.Unmount()
	}
	return nil
}

// strataFSWin implements the WinFsp filesystem interface
type strataFSWin struct {
	fuse.FileSystemBase
	db           *database.DB
	source       config.StorageSource
	showChunks   bool
	showMetadata bool
	mu           sync.RWMutex
	openFiles    map[uint64]*openFile
	nextHandle   uint64
}

type openFile struct {
	data []byte
	path string
}

// Statfs returns filesystem statistics
func (fs *strataFSWin) Statfs(path string, stat *fuse.Statfs_t) int {
	stat.Bsize = 4096
	stat.Frsize = 4096
	stat.Blocks = 1000000
	stat.Bfree = 500000
	stat.Bavail = 500000
	stat.Files = 1000000
	stat.Ffree = 500000
	return 0
}

// Getattr returns file attributes
func (fs *strataFSWin) Getattr(path string, stat *fuse.Stat_t, fh uint64) int {
	path = normalizePath(path)

	if path == "/" {
		stat.Mode = fuse.S_IFDIR | 0555
		stat.Nlink = 2
		return 0
	}

	// Check if it's a real file from the database
	file, err := fs.db.GetFileByPath(fs.toSourcePath(path))
	if err == nil && file != nil {
		stat.Mode = fuse.S_IFREG | 0444
		stat.Size = file.Size
		stat.Nlink = 1
		stat.Mtim = fuse.NewTimespec(file.UpdatedAt)
		stat.Ctim = fuse.NewTimespec(file.CreatedAt)
		return 0
	}

	// Check if it's a directory
	if fs.isDirectory(path) {
		stat.Mode = fuse.S_IFDIR | 0555
		stat.Nlink = 2
		return 0
	}

	// Check for _chunks directory
	if strings.HasSuffix(path, "/_chunks") && fs.showChunks {
		parentPath := strings.TrimSuffix(path, "/_chunks")
		file, err := fs.db.GetFileByPath(fs.toSourcePath(parentPath))
		if err == nil && file != nil {
			stat.Mode = fuse.S_IFDIR | 0555
			stat.Nlink = 2
			return 0
		}
	}

	// Check for metadata.json
	if strings.HasSuffix(path, "/metadata.json") && fs.showMetadata {
		parentPath := strings.TrimSuffix(path, "/metadata.json")
		file, err := fs.db.GetFileByPath(fs.toSourcePath(parentPath))
		if err == nil && file != nil {
			meta := fs.buildMetadata(file)
			data, _ := json.MarshalIndent(meta, "", "  ")
			stat.Mode = fuse.S_IFREG | 0444
			stat.Size = int64(len(data))
			stat.Nlink = 1
			return 0
		}
	}

	// Check for chunk files
	if strings.Contains(path, "/_chunks/chunk_") {
		parts := strings.Split(path, "/_chunks/")
		if len(parts) == 2 {
			parentPath := parts[0]
			file, err := fs.db.GetFileByPath(fs.toSourcePath(parentPath))
			if err == nil && file != nil {
				chunks, _ := fs.db.GetChunksByFileID(file.ID)
				for i, chunk := range chunks {
					if parts[1] == chunkFileName(i) {
						stat.Mode = fuse.S_IFREG | 0444
						stat.Size = int64(len(chunk.Content))
						stat.Nlink = 1
						return 0
					}
				}
			}
		}
	}

	return -fuse.ENOENT
}

// Readdir reads directory contents
func (fs *strataFSWin) Readdir(path string, fill func(name string, stat *fuse.Stat_t, ofst int64) bool, ofst int64, fh uint64) int {
	path = normalizePath(path)

	fill(".", nil, 0)
	fill("..", nil, 0)

	// Handle _chunks directory
	if strings.HasSuffix(path, "/_chunks") && fs.showChunks {
		parentPath := strings.TrimSuffix(path, "/_chunks")
		file, err := fs.db.GetFileByPath(fs.toSourcePath(parentPath))
		if err == nil && file != nil {
			chunks, _ := fs.db.GetChunksByFileID(file.ID)
			for i := range chunks {
				fill(chunkFileName(i), nil, 0)
			}
		}
		return 0
	}

	// Get files in this directory
	files, err := fs.db.ListFiles(false)
	if err != nil {
		return -fuse.EIO
	}

	seen := make(map[string]bool)
	sourcePath := fs.toSourcePath(path)

	for _, file := range files {
		relPath := strings.TrimPrefix(file.Path, fs.source.Path)
		relPath = strings.TrimPrefix(relPath, "/")

		if path == "/" {
			// Root directory - show top-level entries
			parts := strings.SplitN(relPath, "/", 2)
			name := parts[0]
			if name != "" && !seen[name] {
				seen[name] = true
				fill(name, nil, 0)
			}
		} else {
			// Subdirectory
			if strings.HasPrefix(file.Path, sourcePath+"/") {
				rest := strings.TrimPrefix(file.Path, sourcePath+"/")
				parts := strings.SplitN(rest, "/", 2)
				name := parts[0]
				if name != "" && !seen[name] {
					seen[name] = true
					fill(name, nil, 0)
				}
			}

			// Show metadata for files in this directory
			if file.Path == sourcePath {
				if fs.showMetadata {
					fill("metadata.json", nil, 0)
				}
				if fs.showChunks {
					fill("_chunks", nil, 0)
				}
			}
		}
	}

	return 0
}

// Open opens a file
func (fs *strataFSWin) Open(path string, flags int) (int, uint64) {
	path = normalizePath(path)

	var data []byte

	// Check for metadata.json
	if strings.HasSuffix(path, "/metadata.json") && fs.showMetadata {
		parentPath := strings.TrimSuffix(path, "/metadata.json")
		file, err := fs.db.GetFileByPath(fs.toSourcePath(parentPath))
		if err == nil && file != nil {
			meta := fs.buildMetadata(file)
			data, _ = json.MarshalIndent(meta, "", "  ")
		}
	}

	// Check for chunk files
	if strings.Contains(path, "/_chunks/chunk_") {
		parts := strings.Split(path, "/_chunks/")
		if len(parts) == 2 {
			parentPath := parts[0]
			file, err := fs.db.GetFileByPath(fs.toSourcePath(parentPath))
			if err == nil && file != nil {
				chunks, _ := fs.db.GetChunksByFileID(file.ID)
				for i, chunk := range chunks {
					if parts[1] == chunkFileName(i) {
						data = []byte(chunk.Content)
						break
					}
				}
			}
		}
	}

	// Regular file - read from source
	if data == nil {
		sourcePath := fs.toSourcePath(path)
		file, err := fs.db.GetFileByPath(sourcePath)
		if err != nil || file == nil {
			return -fuse.ENOENT, 0
		}

		// Read actual file content
		data, err = os.ReadFile(file.Path)
		if err != nil {
			return -fuse.EIO, 0
		}
	}

	fs.mu.Lock()
	handle := fs.nextHandle
	fs.nextHandle++
	fs.openFiles[handle] = &openFile{data: data, path: path}
	fs.mu.Unlock()

	return 0, handle
}

// Read reads file content
func (fs *strataFSWin) Read(path string, buff []byte, ofst int64, fh uint64) int {
	fs.mu.RLock()
	of, ok := fs.openFiles[fh]
	fs.mu.RUnlock()

	if !ok {
		return -fuse.EBADF
	}

	if ofst >= int64(len(of.data)) {
		return 0
	}

	n := copy(buff, of.data[ofst:])
	return n
}

// Release closes a file
func (fs *strataFSWin) Release(path string, fh uint64) int {
	fs.mu.Lock()
	delete(fs.openFiles, fh)
	fs.mu.Unlock()
	return 0
}

// Helper functions

func (fs *strataFSWin) toSourcePath(path string) string {
	if path == "/" {
		return fs.source.Path
	}
	return filepath.Join(fs.source.Path, path)
}

func (fs *strataFSWin) isDirectory(path string) bool {
	files, err := fs.db.ListFiles(false)
	if err != nil {
		return false
	}

	sourcePath := fs.toSourcePath(path)
	for _, file := range files {
		if strings.HasPrefix(file.Path, sourcePath+"/") {
			return true
		}
	}
	return false
}

func (fs *strataFSWin) buildMetadata(file *database.File) map[string]interface{} {
	chunks, _ := fs.db.GetChunksByFileID(file.ID)

	return map[string]interface{}{
		"id":         file.ID,
		"path":       file.Path,
		"checksum":   file.Checksum,
		"size":       file.Size,
		"created_at": file.CreatedAt,
		"updated_at": file.UpdatedAt,
		"chunks":     len(chunks),
	}
}

func normalizePath(path string) string {
	path = strings.ReplaceAll(path, "\\", "/")
	if !strings.HasPrefix(path, "/") {
		path = "/" + path
	}
	return path
}

func chunkFileName(index int) string {
	return "chunk_" + string(rune('0'+index/100)) + string(rune('0'+(index/10)%10)) + string(rune('0'+index%10)) + ".txt"
}
