package filesystem

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
)

// GoogleDriveFileSystem implements FileSystem for Google Drive
type GoogleDriveFileSystem struct {
	client     *http.Client
	folderID   string // Root folder ID to sync
	localCache string

	// Export formats for Google Workspace files
	exportFormats map[string]string

	// Change tracking
	pageToken string
	tokenMu   sync.RWMutex
}

// GoogleDriveConfig holds Google Drive connection configuration
type GoogleDriveConfig struct {
	CredentialsFile string            // Path to OAuth credentials JSON
	TokenFile       string            // Path to store OAuth token
	FolderID        string            // Root folder ID (empty for root)
	LocalCache      string            // Local cache directory
	ExportFormats   map[string]string // MIME type mappings for export
}

// NewGoogleDriveFileSystem creates a new Google Drive filesystem
func NewGoogleDriveFileSystem(config GoogleDriveConfig) (*GoogleDriveFileSystem, error) {
	// Read credentials
	credBytes, err := os.ReadFile(config.CredentialsFile)
	if err != nil {
		return nil, fmt.Errorf("failed to read credentials: %w", err)
	}

	// Parse OAuth config
	oauthConfig, err := google.ConfigFromJSON(credBytes,
		"https://www.googleapis.com/auth/drive.readonly",
	)
	if err != nil {
		return nil, fmt.Errorf("failed to parse credentials: %w", err)
	}

	// Load or create token
	token, err := loadOrCreateToken(oauthConfig, config.TokenFile)
	if err != nil {
		return nil, fmt.Errorf("failed to get token: %w", err)
	}

	// Create HTTP client
	client := oauthConfig.Client(context.Background(), token)

	// Default export formats
	exportFormats := map[string]string{
		"application/vnd.google-apps.document":     "text/plain",
		"application/vnd.google-apps.spreadsheet":  "text/csv",
		"application/vnd.google-apps.presentation": "text/plain",
		"application/vnd.google-apps.drawing":      "image/png",
	}

	// Override with custom formats
	for k, v := range config.ExportFormats {
		exportFormats[k] = v
	}

	fs := &GoogleDriveFileSystem{
		client:        client,
		folderID:      config.FolderID,
		localCache:    config.LocalCache,
		exportFormats: exportFormats,
	}

	return fs, nil
}

// loadOrCreateToken loads token from file or initiates OAuth flow
func loadOrCreateToken(config *oauth2.Config, tokenFile string) (*oauth2.Token, error) {
	// Try to load existing token
	if tokenFile != "" {
		if data, err := os.ReadFile(tokenFile); err == nil {
			var token oauth2.Token
			if err := json.Unmarshal(data, &token); err == nil {
				return &token, nil
			}
		}
	}

	// Need to get new token via OAuth flow
	// In production, this would redirect to browser
	authURL := config.AuthCodeURL("state-token", oauth2.AccessTypeOffline)
	fmt.Printf("Visit this URL to authorize: %s\n", authURL)
	fmt.Print("Enter authorization code: ")

	var code string
	if _, err := fmt.Scan(&code); err != nil {
		return nil, err
	}

	token, err := config.Exchange(context.Background(), code)
	if err != nil {
		return nil, err
	}

	// Save token
	if tokenFile != "" {
		data, _ := json.Marshal(token)
		os.WriteFile(tokenFile, data, 0600)
	}

	return token, nil
}

// driveRequest makes a request to Google Drive API
func (fs *GoogleDriveFileSystem) driveRequest(method, endpoint string, body io.Reader) (*http.Response, error) {
	url := "https://www.googleapis.com/drive/v3" + endpoint

	req, err := http.NewRequest(method, url, body)
	if err != nil {
		return nil, err
	}

	return fs.client.Do(req)
}

// ReadFile reads the content of a file from Google Drive
func (fs *GoogleDriveFileSystem) ReadFile(name string) ([]byte, error) {
	// Check local cache first
	if fs.localCache != "" {
		cachePath := filepath.Join(fs.localCache, name)
		if data, err := os.ReadFile(cachePath); err == nil {
			return data, nil
		}
	}

	// Get file ID from path
	fileID, mimeType, err := fs.resolvePathToID(name)
	if err != nil {
		return nil, err
	}

	// Check if it's a Google Workspace file that needs export
	var endpoint string
	if exportMime, ok := fs.exportFormats[mimeType]; ok {
		endpoint = fmt.Sprintf("/files/%s/export?mimeType=%s", fileID, exportMime)
	} else {
		endpoint = fmt.Sprintf("/files/%s?alt=media", fileID)
	}

	resp, err := fs.driveRequest("GET", endpoint, nil)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("failed to read file: %s - %s", resp.Status, string(body))
	}

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	// Cache locally
	if fs.localCache != "" {
		cachePath := filepath.Join(fs.localCache, name)
		cacheDir := filepath.Dir(cachePath)
		os.MkdirAll(cacheDir, 0755)
		os.WriteFile(cachePath, data, 0644)
	}

	return data, nil
}

// resolvePathToID converts a path to a file ID
func (fs *GoogleDriveFileSystem) resolvePathToID(path string) (string, string, error) {
	parts := strings.Split(strings.Trim(path, "/"), "/")

	currentID := fs.folderID
	if currentID == "" {
		currentID = "root"
	}

	var mimeType string

	for _, part := range parts {
		if part == "" {
			continue
		}

		// Search for file in current folder
		query := fmt.Sprintf("'%s' in parents and name = '%s' and trashed = false", currentID, part)
		endpoint := fmt.Sprintf("/files?q=%s&fields=files(id,mimeType)", query)

		resp, err := fs.driveRequest("GET", endpoint, nil)
		if err != nil {
			return "", "", err
		}

		var result struct {
			Files []struct {
				ID       string `json:"id"`
				MimeType string `json:"mimeType"`
			} `json:"files"`
		}

		if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
			resp.Body.Close()
			return "", "", err
		}
		resp.Body.Close()

		if len(result.Files) == 0 {
			return "", "", fmt.Errorf("file not found: %s", part)
		}

		currentID = result.Files[0].ID
		mimeType = result.Files[0].MimeType
	}

	return currentID, mimeType, nil
}

// Open opens a file for reading from Google Drive
func (fs *GoogleDriveFileSystem) Open(name string) (File, error) {
	data, err := fs.ReadFile(name)
	if err != nil {
		return nil, err
	}

	return &googleDriveFile{
		reader: bytes.NewReader(data),
		name:   filepath.Base(name),
		size:   int64(len(data)),
	}, nil
}

// Stat returns information about a file in Google Drive
func (fs *GoogleDriveFileSystem) Stat(name string) (FileInfo, error) {
	fileID, _, err := fs.resolvePathToID(name)
	if err != nil {
		return nil, err
	}

	endpoint := fmt.Sprintf("/files/%s?fields=id,name,size,mimeType,modifiedTime,createdTime", fileID)

	resp, err := fs.driveRequest("GET", endpoint, nil)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("failed to stat file: %s - %s", resp.Status, string(body))
	}

	var file googleDriveItem
	if err := json.NewDecoder(resp.Body).Decode(&file); err != nil {
		return nil, err
	}

	return &googleDriveFileInfo{item: file}, nil
}

// Walk walks the file tree in Google Drive
func (fs *GoogleDriveFileSystem) Walk(root string, walkFn WalkFunc) error {
	rootID := fs.folderID
	if rootID == "" {
		rootID = "root"
	}

	// If root path specified, resolve it
	if root != "" && root != "/" {
		id, _, err := fs.resolvePathToID(root)
		if err != nil {
			return err
		}
		rootID = id
	}

	return fs.walkFolder(rootID, root, walkFn)
}

func (fs *GoogleDriveFileSystem) walkFolder(folderID, path string, walkFn WalkFunc) error {
	pageToken := ""

	for {
		query := fmt.Sprintf("'%s' in parents and trashed = false", folderID)
		endpoint := fmt.Sprintf("/files?q=%s&fields=nextPageToken,files(id,name,size,mimeType,modifiedTime,createdTime)", query)
		if pageToken != "" {
			endpoint += "&pageToken=" + pageToken
		}

		resp, err := fs.driveRequest("GET", endpoint, nil)
		if err != nil {
			return err
		}

		var result struct {
			NextPageToken string            `json:"nextPageToken"`
			Files         []googleDriveItem `json:"files"`
		}

		if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
			resp.Body.Close()
			return err
		}
		resp.Body.Close()

		for _, file := range result.Files {
			itemPath := path
			if itemPath != "" {
				itemPath += "/"
			}
			itemPath += file.Name

			info := &googleDriveFileInfo{item: file}

			if err := walkFn(itemPath, info, nil); err != nil {
				return err
			}

			// Recurse into folders
			if file.MimeType == "application/vnd.google-apps.folder" {
				if err := fs.walkFolder(file.ID, itemPath, walkFn); err != nil {
					return err
				}
			}
		}

		if result.NextPageToken == "" {
			break
		}
		pageToken = result.NextPageToken
	}

	return nil
}

// GetChanges returns changes since last sync using Drive changes API
func (fs *GoogleDriveFileSystem) GetChanges(ctx context.Context) ([]GoogleDriveChange, error) {
	fs.tokenMu.Lock()
	pageToken := fs.pageToken
	fs.tokenMu.Unlock()

	// Get start page token if we don't have one
	if pageToken == "" {
		resp, err := fs.driveRequest("GET", "/changes/startPageToken", nil)
		if err != nil {
			return nil, err
		}

		var result struct {
			StartPageToken string `json:"startPageToken"`
		}
		if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
			resp.Body.Close()
			return nil, fmt.Errorf("failed to decode start page token: %w", err)
		}
		resp.Body.Close()

		pageToken = result.StartPageToken
	}

	var changes []GoogleDriveChange

	for {
		endpoint := fmt.Sprintf("/changes?pageToken=%s&fields=nextPageToken,newStartPageToken,changes(fileId,removed,file(id,name,mimeType,modifiedTime,parents))", pageToken)

		resp, err := fs.driveRequest("GET", endpoint, nil)
		if err != nil {
			return nil, err
		}

		var result struct {
			NextPageToken     string `json:"nextPageToken"`
			NewStartPageToken string `json:"newStartPageToken"`
			Changes           []struct {
				FileID  string `json:"fileId"`
				Removed bool   `json:"removed"`
				File    *struct {
					ID           string    `json:"id"`
					Name         string    `json:"name"`
					MimeType     string    `json:"mimeType"`
					ModifiedTime time.Time `json:"modifiedTime"`
					Parents      []string  `json:"parents"`
				} `json:"file"`
			} `json:"changes"`
		}

		if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
			resp.Body.Close()
			return nil, err
		}
		resp.Body.Close()

		for _, change := range result.Changes {
			c := GoogleDriveChange{
				FileID:  change.FileID,
				Removed: change.Removed,
			}
			if change.File != nil {
				c.Name = change.File.Name
				c.MimeType = change.File.MimeType
				c.ModifiedTime = change.File.ModifiedTime
				c.IsFolder = change.File.MimeType == "application/vnd.google-apps.folder"
			}
			changes = append(changes, c)
		}

		if result.NextPageToken != "" {
			pageToken = result.NextPageToken
		} else {
			fs.tokenMu.Lock()
			fs.pageToken = result.NewStartPageToken
			fs.tokenMu.Unlock()
			break
		}
	}

	return changes, nil
}

// MkdirAll is not supported for Google Drive (read-only)
func (fs *GoogleDriveFileSystem) MkdirAll(path string, perm os.FileMode) error {
	return fmt.Errorf("Google Drive filesystem is read-only")
}

// Join joins path elements
func (fs *GoogleDriveFileSystem) Join(elem ...string) string {
	return filepath.Join(elem...)
}

// Base returns the last element of path
func (fs *GoogleDriveFileSystem) Base(path string) string {
	return filepath.Base(path)
}

// Dir returns all but the last element of path
func (fs *GoogleDriveFileSystem) Dir(path string) string {
	return filepath.Dir(path)
}

// Ext returns the file name extension
func (fs *GoogleDriveFileSystem) Ext(path string) string {
	return filepath.Ext(path)
}

// GoogleDriveChange represents a change in Google Drive
type GoogleDriveChange struct {
	FileID       string
	Name         string
	MimeType     string
	ModifiedTime time.Time
	IsFolder     bool
	Removed      bool
}

// googleDriveItem represents a Google Drive file/folder
type googleDriveItem struct {
	ID           string    `json:"id"`
	Name         string    `json:"name"`
	Size         int64     `json:"size"`
	MimeType     string    `json:"mimeType"`
	ModifiedTime time.Time `json:"modifiedTime"`
	CreatedTime  time.Time `json:"createdTime"`
}

// googleDriveFile implements File interface
type googleDriveFile struct {
	reader *bytes.Reader
	name   string
	size   int64
}

func (f *googleDriveFile) Read(p []byte) (n int, err error) {
	return f.reader.Read(p)
}

func (f *googleDriveFile) ReadAt(p []byte, off int64) (n int, err error) {
	return f.reader.ReadAt(p, off)
}

func (f *googleDriveFile) Close() error {
	return nil
}

func (f *googleDriveFile) Stat() (FileInfo, error) {
	return &googleDriveFileInfo{
		item: googleDriveItem{
			Name: f.name,
			Size: f.size,
		},
	}, nil
}

// googleDriveFileInfo implements FileInfo interface
type googleDriveFileInfo struct {
	item googleDriveItem
}

func (fi *googleDriveFileInfo) Name() string       { return fi.item.Name }
func (fi *googleDriveFileInfo) Size() int64        { return fi.item.Size }
func (fi *googleDriveFileInfo) Mode() os.FileMode  { return 0444 }
func (fi *googleDriveFileInfo) ModTime() time.Time { return fi.item.ModifiedTime }
func (fi *googleDriveFileInfo) IsDir() bool {
	return fi.item.MimeType == "application/vnd.google-apps.folder"
}
func (fi *googleDriveFileInfo) Sys() interface{} { return fi.item }
