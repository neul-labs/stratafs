package filesystem

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/s3"
)

// S3FileSystem implements FileSystem for Amazon S3
type S3FileSystem struct {
	bucket string
	prefix string
	client *s3.S3
}

// NewS3FileSystem creates a new S3 filesystem
func NewS3FileSystem(bucket, prefix, region, accessKey, secretKey, endpoint string) (*S3FileSystem, error) {
	config := &aws.Config{
		Region: aws.String(region),
	}

	// Set custom endpoint if provided (for S3-compatible services)
	if endpoint != "" {
		config.Endpoint = aws.String(endpoint)
		config.S3ForcePathStyle = aws.Bool(true)
	}

	// Set credentials if provided
	if accessKey != "" && secretKey != "" {
		config.Credentials = credentials.NewStaticCredentials(accessKey, secretKey, "")
	}

	sess, err := session.NewSession(config)
	if err != nil {
		return nil, fmt.Errorf("failed to create AWS session: %w", err)
	}

	return &S3FileSystem{
		bucket: bucket,
		prefix: strings.TrimPrefix(prefix, "/"),
		client: s3.New(sess),
	}, nil
}

// getS3Key converts a path to an S3 key
func (s3fs *S3FileSystem) getS3Key(path string) string {
	key := strings.TrimPrefix(path, "/")
	if s3fs.prefix != "" {
		key = s3fs.prefix + "/" + key
	}
	return key
}

// ReadFile reads the content of a file from S3
func (s3fs *S3FileSystem) ReadFile(name string) ([]byte, error) {
	key := s3fs.getS3Key(name)

	result, err := s3fs.client.GetObject(&s3.GetObjectInput{
		Bucket: aws.String(s3fs.bucket),
		Key:    aws.String(key),
	})
	if err != nil {
		return nil, fmt.Errorf("failed to get S3 object %s: %w", key, err)
	}
	defer result.Body.Close()

	return io.ReadAll(result.Body)
}

// Open opens a file for reading from S3
func (s3fs *S3FileSystem) Open(name string) (File, error) {
	key := s3fs.getS3Key(name)

	result, err := s3fs.client.GetObject(&s3.GetObjectInput{
		Bucket: aws.String(s3fs.bucket),
		Key:    aws.String(key),
	})
	if err != nil {
		return nil, fmt.Errorf("failed to get S3 object %s: %w", key, err)
	}

	return &s3File{
		name:   name,
		reader: result.Body,
		size:   *result.ContentLength,
		modTime: *result.LastModified,
	}, nil
}

// Stat returns information about a file in S3
func (s3fs *S3FileSystem) Stat(name string) (FileInfo, error) {
	key := s3fs.getS3Key(name)

	result, err := s3fs.client.HeadObject(&s3.HeadObjectInput{
		Bucket: aws.String(s3fs.bucket),
		Key:    aws.String(key),
	})
	if err != nil {
		return nil, fmt.Errorf("failed to get S3 object metadata %s: %w", key, err)
	}

	return &s3FileInfo{
		name:    filepath.Base(name),
		size:    *result.ContentLength,
		modTime: *result.LastModified,
		isDir:   false,
	}, nil
}

// Walk walks the file tree in S3
func (s3fs *S3FileSystem) Walk(root string, walkFn WalkFunc) error {
	prefix := s3fs.getS3Key(root)
	if prefix != "" && !strings.HasSuffix(prefix, "/") {
		prefix += "/"
	}

	return s3fs.client.ListObjectsV2Pages(&s3.ListObjectsV2Input{
		Bucket: aws.String(s3fs.bucket),
		Prefix: aws.String(prefix),
	}, func(page *s3.ListObjectsV2Output, lastPage bool) bool {
		for _, obj := range page.Contents {
			key := *obj.Key
			// Remove prefix to get relative path
			path := strings.TrimPrefix(key, s3fs.prefix)
			path = strings.TrimPrefix(path, "/")

			info := &s3FileInfo{
				name:    filepath.Base(path),
				size:    *obj.Size,
				modTime: *obj.LastModified,
				isDir:   false,
			}

			if err := walkFn(path, info, nil); err != nil {
				return false // Stop walking
			}
		}
		return true // Continue to next page
	})
}

// MkdirAll creates a directory path in S3 (no-op since S3 doesn't have directories)
func (s3fs *S3FileSystem) MkdirAll(path string, perm os.FileMode) error {
	// S3 doesn't have directories, so this is a no-op
	return nil
}

// Join joins any number of path elements into a single path
func (s3fs *S3FileSystem) Join(elem ...string) string {
	return filepath.Join(elem...)
}

// Base returns the last element of path
func (s3fs *S3FileSystem) Base(path string) string {
	return filepath.Base(path)
}

// Dir returns all but the last element of path
func (s3fs *S3FileSystem) Dir(path string) string {
	return filepath.Dir(path)
}

// Ext returns the file name extension used by path
func (s3fs *S3FileSystem) Ext(path string) string {
	return filepath.Ext(path)
}

// s3File implements the File interface for S3 objects
type s3File struct {
	name    string
	reader  io.ReadCloser
	size    int64
	modTime time.Time
}

// Read reads from the S3 object
func (f *s3File) Read(p []byte) (n int, err error) {
	return f.reader.Read(p)
}

// ReadAt reads from the S3 object at offset (not implemented for streaming)
func (f *s3File) ReadAt(p []byte, off int64) (n int, err error) {
	return 0, fmt.Errorf("ReadAt not supported for S3 streaming reads")
}

// Close closes the S3 object reader
func (f *s3File) Close() error {
	return f.reader.Close()
}

// Stat returns file information
func (f *s3File) Stat() (FileInfo, error) {
	return &s3FileInfo{
		name:    filepath.Base(f.name),
		size:    f.size,
		modTime: f.modTime,
		isDir:   false,
	}, nil
}

// s3FileInfo implements the FileInfo interface for S3 objects
type s3FileInfo struct {
	name    string
	size    int64
	modTime time.Time
	isDir   bool
}

// Name returns the base name of the file
func (fi *s3FileInfo) Name() string {
	return fi.name
}

// Size returns the length in bytes for regular files
func (fi *s3FileInfo) Size() int64 {
	return fi.size
}

// Mode returns the file mode bits
func (fi *s3FileInfo) Mode() os.FileMode {
	if fi.isDir {
		return os.ModeDir | 0755
	}
	return 0644
}

// ModTime returns the modification time
func (fi *s3FileInfo) ModTime() time.Time {
	return fi.modTime
}

// IsDir returns whether this is a directory
func (fi *s3FileInfo) IsDir() bool {
	return fi.isDir
}

// Sys returns the underlying data source
func (fi *s3FileInfo) Sys() interface{} {
	return nil
}