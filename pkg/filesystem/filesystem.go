package filesystem

import (
	"io"
	"os"
	"path/filepath"
	"time"
)

// FileSystem is an abstraction for file system operations
type FileSystem interface {
	// ReadFile reads the content of a file
	ReadFile(name string) ([]byte, error)
	
	// Open opens a file for reading
	Open(name string) (File, error)
	
	// Stat returns information about a file
	Stat(name string) (FileInfo, error)
	
	// Walk walks the file tree rooted at root
	Walk(root string, walkFn WalkFunc) error
	
	// MkdirAll creates a directory path and all parents that does not exist
	MkdirAll(path string, perm os.FileMode) error
	
	// Join joins any number of path elements into a single path
	Join(elem ...string) string
	
	// Base returns the last element of path
	Base(path string) string
	
	// Dir returns all but the last element of path
	Dir(path string) string
	
	// Ext returns the file name extension used by path
	Ext(path string) string
}

// File represents a file in the filesystem
type File interface {
	io.Closer
	io.Reader
	io.ReaderAt
	Stat() (FileInfo, error)
}

// FileInfo describes a file and is returned by Stat
type FileInfo interface {
	Name() string       // base name of the file
	Size() int64        // length in bytes for regular files
	Mode() os.FileMode  // file mode bits
	ModTime() time.Time // modification time
	IsDir() bool        // abbreviation for Mode().IsDir()
	Sys() interface{}   // underlying data source (can return nil)
}

// WalkFunc is the type of the function called for each file or directory
type WalkFunc func(path string, info FileInfo, err error) error

// LocalFileSystem is an implementation of FileSystem for local files
type LocalFileSystem struct{}

// NewLocalFileSystem creates a new local file system
func NewLocalFileSystem() *LocalFileSystem {
	return &LocalFileSystem{}
}

// ReadFile reads the content of a file
func (lfs *LocalFileSystem) ReadFile(name string) ([]byte, error) {
	return os.ReadFile(name)
}

// Open opens a file for reading
func (lfs *LocalFileSystem) Open(name string) (File, error) {
	file, err := os.Open(name)
	if err != nil {
		return nil, err
	}
	return &localFile{File: file}, nil
}

// Stat returns information about a file
func (lfs *LocalFileSystem) Stat(name string) (FileInfo, error) {
	info, err := os.Stat(name)
	if err != nil {
		return nil, err
	}
	return &localFileInfo{FileInfo: info}, nil
}

// Walk walks the file tree rooted at root
func (lfs *LocalFileSystem) Walk(root string, walkFn WalkFunc) error {
	return filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		return walkFn(path, &localFileInfo{FileInfo: info}, err)
	})
}

// MkdirAll creates a directory path and all parents that does not exist
func (lfs *LocalFileSystem) MkdirAll(path string, perm os.FileMode) error {
	return os.MkdirAll(path, perm)
}

// Join joins any number of path elements into a single path
func (lfs *LocalFileSystem) Join(elem ...string) string {
	return filepath.Join(elem...)
}

// Base returns the last element of path
func (lfs *LocalFileSystem) Base(path string) string {
	return filepath.Base(path)
}

// Dir returns all but the last element of path
func (lfs *LocalFileSystem) Dir(path string) string {
	return filepath.Dir(path)
}

// Ext returns the file name extension used by path
func (lfs *LocalFileSystem) Ext(path string) string {
	return filepath.Ext(path)
}

// localFile wraps os.File to implement the File interface
type localFile struct {
	*os.File
}

// Stat returns information about the file
func (lf *localFile) Stat() (FileInfo, error) {
	info, err := lf.File.Stat()
	if err != nil {
		return nil, err
	}
	return &localFileInfo{FileInfo: info}, nil
}

// localFileInfo wraps os.FileInfo to implement the FileInfo interface
type localFileInfo struct {
	os.FileInfo
}

// ObjectStoreFileSystem is a placeholder for an object store implementation
type ObjectStoreFileSystem struct {
	// In a real implementation, this would contain connection details
	// to an object store like S3, GCS, etc.
	bucket string
	prefix string
}

// NewObjectStoreFileSystem creates a new object store file system
func NewObjectStoreFileSystem(bucket, prefix string) *ObjectStoreFileSystem {
	return &ObjectStoreFileSystem{
		bucket: bucket,
		prefix: prefix,
	}
}

// ReadFile reads the content of a file from object store
func (osfs *ObjectStoreFileSystem) ReadFile(name string) ([]byte, error) {
	// TODO: Implement object store reading
	// This is a placeholder implementation
	return nil, nil
}

// Open opens a file for reading from object store
func (osfs *ObjectStoreFileSystem) Open(name string) (File, error) {
	// TODO: Implement object store opening
	// This is a placeholder implementation
	return nil, nil
}

// Stat returns information about a file in object store
func (osfs *ObjectStoreFileSystem) Stat(name string) (FileInfo, error) {
	// TODO: Implement object store stat
	// This is a placeholder implementation
	return nil, nil
}

// Walk walks the file tree in object store
func (osfs *ObjectStoreFileSystem) Walk(root string, walkFn WalkFunc) error {
	// TODO: Implement object store walking
	// This is a placeholder implementation
	return nil
}

// MkdirAll creates a directory path in object store
func (osfs *ObjectStoreFileSystem) MkdirAll(path string, perm os.FileMode) error {
	// TODO: Implement object store directory creation
	// This is a placeholder implementation
	return nil
}

// Join joins any number of path elements into a single path
func (osfs *ObjectStoreFileSystem) Join(elem ...string) string {
	return filepath.Join(elem...)
}

// Base returns the last element of path
func (osfs *ObjectStoreFileSystem) Base(path string) string {
	return filepath.Base(path)
}

// Dir returns all but the last element of path
func (osfs *ObjectStoreFileSystem) Dir(path string) string {
	return filepath.Dir(path)
}

// Ext returns the file name extension used by path
func (osfs *ObjectStoreFileSystem) Ext(path string) string {
	return filepath.Ext(path)
}