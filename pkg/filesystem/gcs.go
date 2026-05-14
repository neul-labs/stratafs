package filesystem

import (
	"fmt"
	"os"
	"path/filepath"
)

// GCSFileSystem implements FileSystem for Google Cloud Storage (placeholder)
type GCSFileSystem struct {
	bucket    string
	prefix    string
	projectID string
	credPath  string
}

// NewGCSFileSystem creates a new GCS filesystem (placeholder implementation)
func NewGCSFileSystem(bucket, prefix, projectID, credentialsPath string) (*GCSFileSystem, error) {
	return &GCSFileSystem{
		bucket:    bucket,
		prefix:    prefix,
		projectID: projectID,
		credPath:  credentialsPath,
	}, fmt.Errorf("GCS filesystem not yet implemented - please use S3 or local storage")
}

// ReadFile placeholder
func (gfs *GCSFileSystem) ReadFile(name string) ([]byte, error) {
	return nil, fmt.Errorf("GCS filesystem not implemented")
}

// Open placeholder
func (gfs *GCSFileSystem) Open(name string) (File, error) {
	return nil, fmt.Errorf("GCS filesystem not implemented")
}

// Stat placeholder
func (gfs *GCSFileSystem) Stat(name string) (FileInfo, error) {
	return nil, fmt.Errorf("GCS filesystem not implemented")
}

// Walk placeholder
func (gfs *GCSFileSystem) Walk(root string, walkFn WalkFunc) error {
	return fmt.Errorf("GCS filesystem not implemented")
}

// MkdirAll placeholder
func (gfs *GCSFileSystem) MkdirAll(path string, perm os.FileMode) error {
	return fmt.Errorf("GCS filesystem not implemented")
}

// Join joins any number of path elements into a single path
func (gfs *GCSFileSystem) Join(elem ...string) string {
	return filepath.Join(elem...)
}

// Base returns the last element of path
func (gfs *GCSFileSystem) Base(path string) string {
	return filepath.Base(path)
}

// Dir returns all but the last element of path
func (gfs *GCSFileSystem) Dir(path string) string {
	return filepath.Dir(path)
}

// Ext returns the file name extension used by path
func (gfs *GCSFileSystem) Ext(path string) string {
	return filepath.Ext(path)
}
