//go:build linux
// +build linux

package fsbridge

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"syscall"

	"github.com/neul-labs/stratafs/pkg/config"
	"github.com/neul-labs/stratafs/pkg/database"

	"bazil.org/fuse"
	"bazil.org/fuse/fs"
)

// MountOptions configures the FUSE mount behavior.
type MountOptions struct {
	MountPoint   string
	ReadOnly     bool
	AllowOther   bool
	Debug        bool
	ShowChunks   bool // Whether to expose _chunks directories
	ShowMetadata bool // Whether to expose metadata.json files
}

// FuseMount represents a mounted StrataFS filesystem.
type FuseMount struct {
	db      *database.DB
	source  config.StorageSource
	opts    MountOptions
	conn    *fuse.Conn
	mounted bool
	mu      sync.RWMutex
	stopCh  chan struct{}
}

// NewFuseMount creates a new FUSE mount for the given source.
func NewFuseMount(db *database.DB, source config.StorageSource, opts MountOptions) *FuseMount {
	return &FuseMount{
		db:     db,
		source: source,
		opts:   opts,
		stopCh: make(chan struct{}),
	}
}

// Mount mounts the filesystem at the specified mount point.
func (m *FuseMount) Mount() error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.mounted {
		return fmt.Errorf("filesystem already mounted")
	}

	// Ensure mount point exists
	if err := os.MkdirAll(m.opts.MountPoint, 0755); err != nil {
		return fmt.Errorf("failed to create mount point: %w", err)
	}

	// Build mount options
	mountOpts := []fuse.MountOption{
		fuse.FSName("stratafs"),
		fuse.Subtype("stratafs"),
	}

	if m.opts.ReadOnly {
		mountOpts = append(mountOpts, fuse.ReadOnly())
	}

	if m.opts.AllowOther {
		mountOpts = append(mountOpts, fuse.AllowOther())
	}

	// Mount the filesystem
	conn, err := fuse.Mount(m.opts.MountPoint, mountOpts...)
	if err != nil {
		return fmt.Errorf("failed to mount: %w", err)
	}
	m.conn = conn

	// Create the filesystem
	filesystem := &strataFS{
		db:           m.db,
		source:       m.source,
		showChunks:   m.opts.ShowChunks,
		showMetadata: m.opts.ShowMetadata,
	}

	m.mounted = true

	// Serve in a goroutine
	go func() {
		if err := fs.Serve(conn, filesystem); err != nil {
			fmt.Printf("FUSE serve error: %v\n", err)
		}
	}()

	return nil
}

// Unmount unmounts the filesystem.
func (m *FuseMount) Unmount() error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if !m.mounted {
		return nil
	}

	close(m.stopCh)

	if err := fuse.Unmount(m.opts.MountPoint); err != nil {
		return fmt.Errorf("failed to unmount: %w", err)
	}

	if m.conn != nil {
		m.conn.Close()
	}

	m.mounted = false
	return nil
}

// IsMounted returns whether the filesystem is currently mounted.
func (m *FuseMount) IsMounted() bool {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.mounted
}

// strataFS implements the FUSE filesystem interface.
type strataFS struct {
	db           *database.DB
	source       config.StorageSource
	showChunks   bool
	showMetadata bool
}

func (f *strataFS) Root() (fs.Node, error) {
	return &Dir{
		fs:   f,
		path: "",
	}, nil
}

// Dir represents a directory in the filesystem.
type Dir struct {
	fs   *strataFS
	path string // Relative path from source root
}

func (d *Dir) Attr(ctx context.Context, a *fuse.Attr) error {
	a.Mode = os.ModeDir | 0555
	a.Uid = uint32(os.Getuid())
	a.Gid = uint32(os.Getgid())
	return nil
}

func (d *Dir) Lookup(ctx context.Context, name string) (fs.Node, error) {
	childPath := filepath.Join(d.path, name)

	// Check for special nodes
	if name == "_chunks" && d.fs.showChunks {
		// Find the file for this directory
		fullPath := filepath.Join(d.fs.source.Path, d.path)
		file, err := d.fs.db.GetFileByPath(fullPath)
		if err == nil && file != nil {
			return &ChunksDir{
				fs:     d.fs,
				fileID: file.ID,
			}, nil
		}
	}

	if name == "metadata.json" && d.fs.showMetadata {
		fullPath := filepath.Join(d.fs.source.Path, d.path)
		file, err := d.fs.db.GetFileByPath(fullPath)
		if err == nil && file != nil {
			return &MetadataFile{
				fs:   d.fs,
				file: file,
			}, nil
		}
	}

	// Check if it's a directory containing indexed files
	fullPath := filepath.Join(d.fs.source.Path, childPath)

	// First check if it's an indexed file
	file, err := d.fs.db.GetFileByPath(fullPath)
	if err == nil && file != nil {
		return &FileNode{
			fs:   d.fs,
			file: file,
		}, nil
	}

	// Check if it's a directory by looking for files with this prefix
	files, err := d.fs.db.ListFiles(false)
	if err != nil {
		return nil, syscall.ENOENT
	}

	prefix := fullPath + "/"
	for _, f := range files {
		if strings.HasPrefix(f.Path, prefix) {
			return &Dir{
				fs:   d.fs,
				path: childPath,
			}, nil
		}
	}

	return nil, syscall.ENOENT
}

func (d *Dir) ReadDirAll(ctx context.Context) ([]fuse.Dirent, error) {
	var entries []fuse.Dirent
	seen := make(map[string]bool)

	files, err := d.fs.db.ListFiles(false)
	if err != nil {
		return nil, err
	}

	fullDirPath := filepath.Join(d.fs.source.Path, d.path)
	prefix := fullDirPath
	if prefix != "" && !strings.HasSuffix(prefix, "/") {
		prefix += "/"
	}

	for _, file := range files {
		if !strings.HasPrefix(file.Path, prefix) {
			continue
		}

		// Get the relative part after the prefix
		relative := strings.TrimPrefix(file.Path, prefix)
		if relative == "" {
			continue
		}

		// Get the first component
		parts := strings.SplitN(relative, "/", 2)
		name := parts[0]

		if seen[name] {
			continue
		}
		seen[name] = true

		var entryType fuse.DirentType
		if len(parts) > 1 {
			// It's a directory
			entryType = fuse.DT_Dir
		} else {
			// It's a file
			entryType = fuse.DT_File
		}

		entries = append(entries, fuse.Dirent{
			Name: name,
			Type: entryType,
		})
	}

	// Add special entries if this directory contains a file
	if d.path != "" {
		fullPath := filepath.Join(d.fs.source.Path, d.path)
		file, err := d.fs.db.GetFileByPath(fullPath)
		if err == nil && file != nil {
			if d.fs.showMetadata {
				entries = append(entries, fuse.Dirent{
					Name: "metadata.json",
					Type: fuse.DT_File,
				})
			}
			if d.fs.showChunks {
				entries = append(entries, fuse.Dirent{
					Name: "_chunks",
					Type: fuse.DT_Dir,
				})
			}
		}
	}

	return entries, nil
}

// FileNode represents an indexed file.
type FileNode struct {
	fs   *strataFS
	file *database.File
}

func (f *FileNode) Attr(ctx context.Context, a *fuse.Attr) error {
	a.Mode = 0444
	a.Size = uint64(f.file.Size)
	a.Mtime = f.file.UpdatedAt
	a.Ctime = f.file.CreatedAt
	a.Uid = uint32(os.Getuid())
	a.Gid = uint32(os.Getgid())
	return nil
}

func (f *FileNode) ReadAll(ctx context.Context) ([]byte, error) {
	// Read from the original file
	return os.ReadFile(f.file.Path)
}

// ChunksDir represents the _chunks directory for a file.
type ChunksDir struct {
	fs     *strataFS
	fileID int64
}

func (d *ChunksDir) Attr(ctx context.Context, a *fuse.Attr) error {
	a.Mode = os.ModeDir | 0555
	a.Uid = uint32(os.Getuid())
	a.Gid = uint32(os.Getgid())
	return nil
}

func (d *ChunksDir) ReadDirAll(ctx context.Context) ([]fuse.Dirent, error) {
	chunks, err := d.fs.db.GetChunksByFileID(d.fileID)
	if err != nil {
		return nil, err
	}

	var entries []fuse.Dirent
	for i := range chunks {
		entries = append(entries, fuse.Dirent{
			Name: fmt.Sprintf("chunk_%04d.txt", i),
			Type: fuse.DT_File,
		})
	}

	return entries, nil
}

func (d *ChunksDir) Lookup(ctx context.Context, name string) (fs.Node, error) {
	// Parse chunk index from name
	var index int
	if _, err := fmt.Sscanf(name, "chunk_%d.txt", &index); err != nil {
		return nil, syscall.ENOENT
	}

	chunks, err := d.fs.db.GetChunksByFileID(d.fileID)
	if err != nil || index >= len(chunks) {
		return nil, syscall.ENOENT
	}

	return &ChunkFile{
		chunk: chunks[index],
	}, nil
}

// ChunkFile represents a single chunk file.
type ChunkFile struct {
	chunk *database.FileChunk
}

func (f *ChunkFile) Attr(ctx context.Context, a *fuse.Attr) error {
	a.Mode = 0444
	a.Size = uint64(len(f.chunk.Content))
	a.Uid = uint32(os.Getuid())
	a.Gid = uint32(os.Getgid())
	return nil
}

func (f *ChunkFile) ReadAll(ctx context.Context) ([]byte, error) {
	return []byte(f.chunk.Content), nil
}

// MetadataFile represents the metadata.json virtual file.
type MetadataFile struct {
	fs   *strataFS
	file *database.File
}

func (f *MetadataFile) Attr(ctx context.Context, a *fuse.Attr) error {
	content, _ := f.generateContent()
	a.Mode = 0444
	a.Size = uint64(len(content))
	a.Mtime = f.file.UpdatedAt
	a.Uid = uint32(os.Getuid())
	a.Gid = uint32(os.Getgid())
	return nil
}

func (f *MetadataFile) ReadAll(ctx context.Context) ([]byte, error) {
	return f.generateContent()
}

func (f *MetadataFile) generateContent() ([]byte, error) {
	chunks, err := f.fs.db.GetChunksByFileID(f.file.ID)
	if err != nil {
		return nil, err
	}

	relative, _ := filepath.Rel(f.fs.source.Path, f.file.Path)

	meta := FileMetadata{
		Path:       f.file.Path,
		Relative:   relative,
		Checksum:   f.file.Checksum,
		Size:       f.file.Size,
		CreatedAt:  f.file.CreatedAt,
		UpdatedAt:  f.file.UpdatedAt,
		ChunkCount: len(chunks),
	}

	return json.MarshalIndent(meta, "", "  ")
}

// Ensure interfaces are implemented
var (
	_ fs.FS                 = (*strataFS)(nil)
	_ fs.Node               = (*Dir)(nil)
	_ fs.NodeStringLookuper = (*Dir)(nil)
	_ fs.HandleReadDirAller = (*Dir)(nil)
	_ fs.Node               = (*FileNode)(nil)
	_ fs.HandleReadAller    = (*FileNode)(nil)
	_ fs.Node               = (*ChunksDir)(nil)
	_ fs.NodeStringLookuper = (*ChunksDir)(nil)
	_ fs.HandleReadDirAller = (*ChunksDir)(nil)
	_ fs.Node               = (*ChunkFile)(nil)
	_ fs.HandleReadAller    = (*ChunkFile)(nil)
	_ fs.Node               = (*MetadataFile)(nil)
	_ fs.HandleReadAller    = (*MetadataFile)(nil)
)
