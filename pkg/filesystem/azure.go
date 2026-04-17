package filesystem

import (
	"fmt"
	"os"
	"path/filepath"
)

// AzureFileSystem implements FileSystem for Azure Blob Storage (placeholder)
type AzureFileSystem struct {
	container        string
	prefix           string
	accountName      string
	accountKey       string
	connectionString string
}

// NewAzureFileSystem creates a new Azure filesystem (placeholder implementation)
func NewAzureFileSystem(container, prefix, accountName, accountKey, connectionString string) (*AzureFileSystem, error) {
	return &AzureFileSystem{
		container:        container,
		prefix:           prefix,
		accountName:      accountName,
		accountKey:       accountKey,
		connectionString: connectionString,
	}, fmt.Errorf("Azure filesystem not yet implemented - please use S3 or local storage")
}

// ReadFile placeholder
func (afs *AzureFileSystem) ReadFile(name string) ([]byte, error) {
	return nil, fmt.Errorf("Azure filesystem not implemented")
}

// Open placeholder
func (afs *AzureFileSystem) Open(name string) (File, error) {
	return nil, fmt.Errorf("Azure filesystem not implemented")
}

// Stat placeholder
func (afs *AzureFileSystem) Stat(name string) (FileInfo, error) {
	return nil, fmt.Errorf("Azure filesystem not implemented")
}

// Walk placeholder
func (afs *AzureFileSystem) Walk(root string, walkFn WalkFunc) error {
	return fmt.Errorf("Azure filesystem not implemented")
}

// MkdirAll placeholder
func (afs *AzureFileSystem) MkdirAll(path string, perm os.FileMode) error {
	return fmt.Errorf("Azure filesystem not implemented")
}

// Join joins any number of path elements into a single path
func (afs *AzureFileSystem) Join(elem ...string) string {
	return filepath.Join(elem...)
}

// Base returns the last element of path
func (afs *AzureFileSystem) Base(path string) string {
	return filepath.Base(path)
}

// Dir returns all but the last element of path
func (afs *AzureFileSystem) Dir(path string) string {
	return filepath.Dir(path)
}

// Ext returns the file name extension used by path
func (afs *AzureFileSystem) Ext(path string) string {
	return filepath.Ext(path)
}