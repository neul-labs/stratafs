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
)

// SharePointFileSystem implements FileSystem for Microsoft SharePoint/OneDrive
type SharePointFileSystem struct {
	tenantID     string
	clientID     string
	clientSecret string
	siteID       string
	driveID      string
	localCache   string

	// Token management
	accessToken string
	tokenExpiry time.Time
	tokenMu     sync.RWMutex

	// HTTP client
	client *http.Client

	// Delta tracking for incremental sync
	deltaLink string
}

// SharePointConfig holds SharePoint connection configuration
type SharePointConfig struct {
	TenantID     string
	ClientID     string
	ClientSecret string
	SiteURL      string // e.g., "https://company.sharepoint.com/sites/docs"
	DriveID      string // optional, defaults to default document library
	LocalCache   string
}

// NewSharePointFileSystem creates a new SharePoint filesystem
func NewSharePointFileSystem(config SharePointConfig) (*SharePointFileSystem, error) {
	fs := &SharePointFileSystem{
		tenantID:     config.TenantID,
		clientID:     config.ClientID,
		clientSecret: config.ClientSecret,
		localCache:   config.LocalCache,
		client:       &http.Client{Timeout: 30 * time.Second},
	}

	// Get access token
	if err := fs.refreshToken(); err != nil {
		return nil, fmt.Errorf("failed to authenticate: %w", err)
	}

	// Resolve site ID from URL
	if config.SiteURL != "" {
		siteID, err := fs.resolveSiteID(config.SiteURL)
		if err != nil {
			return nil, fmt.Errorf("failed to resolve site: %w", err)
		}
		fs.siteID = siteID
	}

	// Get default drive if not specified
	if config.DriveID != "" {
		fs.driveID = config.DriveID
	} else if fs.siteID != "" {
		driveID, err := fs.getDefaultDrive()
		if err != nil {
			return nil, fmt.Errorf("failed to get default drive: %w", err)
		}
		fs.driveID = driveID
	}

	return fs, nil
}

// refreshToken gets a new access token from Azure AD
func (fs *SharePointFileSystem) refreshToken() error {
	fs.tokenMu.Lock()
	defer fs.tokenMu.Unlock()

	// Check if current token is still valid
	if fs.accessToken != "" && time.Now().Before(fs.tokenExpiry.Add(-5*time.Minute)) {
		return nil
	}

	tokenURL := fmt.Sprintf("https://login.microsoftonline.com/%s/oauth2/v2.0/token", fs.tenantID)

	data := fmt.Sprintf(
		"client_id=%s&client_secret=%s&scope=https://graph.microsoft.com/.default&grant_type=client_credentials",
		fs.clientID, fs.clientSecret,
	)

	resp, err := fs.client.Post(tokenURL, "application/x-www-form-urlencoded", strings.NewReader(data))
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("token request failed: %s - %s", resp.Status, string(body))
	}

	var tokenResp struct {
		AccessToken string `json:"access_token"`
		ExpiresIn   int    `json:"expires_in"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&tokenResp); err != nil {
		return err
	}

	fs.accessToken = tokenResp.AccessToken
	fs.tokenExpiry = time.Now().Add(time.Duration(tokenResp.ExpiresIn) * time.Second)

	return nil
}

// graphRequest makes an authenticated request to Microsoft Graph API
func (fs *SharePointFileSystem) graphRequest(method, endpoint string, body io.Reader) (*http.Response, error) {
	if err := fs.refreshToken(); err != nil {
		return nil, err
	}

	url := "https://graph.microsoft.com/v1.0" + endpoint

	req, err := http.NewRequest(method, url, body)
	if err != nil {
		return nil, err
	}

	fs.tokenMu.RLock()
	req.Header.Set("Authorization", "Bearer "+fs.accessToken)
	fs.tokenMu.RUnlock()
	req.Header.Set("Content-Type", "application/json")

	return fs.client.Do(req)
}

// resolveSiteID gets the site ID from a SharePoint URL
func (fs *SharePointFileSystem) resolveSiteID(siteURL string) (string, error) {
	// Parse URL to get hostname and path
	// e.g., "https://company.sharepoint.com/sites/docs" -> "company.sharepoint.com:/sites/docs"
	siteURL = strings.TrimPrefix(siteURL, "https://")
	siteURL = strings.TrimPrefix(siteURL, "http://")
	parts := strings.SplitN(siteURL, "/", 2)

	var endpoint string
	if len(parts) == 2 {
		endpoint = fmt.Sprintf("/sites/%s:/%s", parts[0], parts[1])
	} else {
		endpoint = fmt.Sprintf("/sites/%s", parts[0])
	}

	resp, err := fs.graphRequest("GET", endpoint, nil)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("failed to get site: %s - %s", resp.Status, string(body))
	}

	var site struct {
		ID string `json:"id"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&site); err != nil {
		return "", err
	}

	return site.ID, nil
}

// getDefaultDrive gets the default document library drive ID
func (fs *SharePointFileSystem) getDefaultDrive() (string, error) {
	endpoint := fmt.Sprintf("/sites/%s/drive", fs.siteID)

	resp, err := fs.graphRequest("GET", endpoint, nil)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("failed to get drive: %s - %s", resp.Status, string(body))
	}

	var drive struct {
		ID string `json:"id"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&drive); err != nil {
		return "", err
	}

	return drive.ID, nil
}

// ReadFile reads the content of a file from SharePoint
func (fs *SharePointFileSystem) ReadFile(name string) ([]byte, error) {
	// Check local cache first
	cachePath := filepath.Join(fs.localCache, name)
	if data, err := os.ReadFile(cachePath); err == nil {
		return data, nil
	}

	// Download from SharePoint
	endpoint := fmt.Sprintf("/drives/%s/root:/%s:/content", fs.driveID, name)

	resp, err := fs.graphRequest("GET", endpoint, nil)
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
		cacheDir := filepath.Dir(cachePath)
		os.MkdirAll(cacheDir, 0755)
		os.WriteFile(cachePath, data, 0644)
	}

	return data, nil
}

// Open opens a file for reading from SharePoint
func (fs *SharePointFileSystem) Open(name string) (File, error) {
	data, err := fs.ReadFile(name)
	if err != nil {
		return nil, err
	}

	return &sharePointFile{
		reader: bytes.NewReader(data),
		name:   filepath.Base(name),
		size:   int64(len(data)),
	}, nil
}

// Stat returns information about a file in SharePoint
func (fs *SharePointFileSystem) Stat(name string) (FileInfo, error) {
	endpoint := fmt.Sprintf("/drives/%s/root:/%s", fs.driveID, name)

	resp, err := fs.graphRequest("GET", endpoint, nil)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("failed to stat file: %s - %s", resp.Status, string(body))
	}

	var item driveItem
	if err := json.NewDecoder(resp.Body).Decode(&item); err != nil {
		return nil, err
	}

	return &sharePointFileInfo{item: item}, nil
}

// Walk walks the file tree in SharePoint
func (fs *SharePointFileSystem) Walk(root string, walkFn WalkFunc) error {
	return fs.walkChildren(root, walkFn)
}

func (fs *SharePointFileSystem) walkChildren(path string, walkFn WalkFunc) error {
	var endpoint string
	if path == "" || path == "/" {
		endpoint = fmt.Sprintf("/drives/%s/root/children", fs.driveID)
	} else {
		endpoint = fmt.Sprintf("/drives/%s/root:/%s:/children", fs.driveID, path)
	}

	for endpoint != "" {
		resp, err := fs.graphRequest("GET", endpoint, nil)
		if err != nil {
			return err
		}

		var result struct {
			Value    []driveItem `json:"value"`
			NextLink string      `json:"@odata.nextLink"`
		}

		if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
			resp.Body.Close()
			return err
		}
		resp.Body.Close()

		for _, item := range result.Value {
			itemPath := path
			if itemPath != "" {
				itemPath += "/"
			}
			itemPath += item.Name

			info := &sharePointFileInfo{item: item}

			if err := walkFn(itemPath, info, nil); err != nil {
				return err
			}

			// Recurse into folders
			if item.Folder != nil {
				if err := fs.walkChildren(itemPath, walkFn); err != nil {
					return err
				}
			}
		}

		// Handle pagination
		if result.NextLink != "" {
			endpoint = strings.TrimPrefix(result.NextLink, "https://graph.microsoft.com/v1.0")
		} else {
			endpoint = ""
		}
	}

	return nil
}

// GetDelta returns changes since last sync
func (fs *SharePointFileSystem) GetDelta(ctx context.Context) ([]DriveChange, error) {
	var endpoint string
	if fs.deltaLink != "" {
		endpoint = strings.TrimPrefix(fs.deltaLink, "https://graph.microsoft.com/v1.0")
	} else {
		endpoint = fmt.Sprintf("/drives/%s/root/delta", fs.driveID)
	}

	var changes []DriveChange

	for endpoint != "" {
		resp, err := fs.graphRequest("GET", endpoint, nil)
		if err != nil {
			return nil, err
		}

		var result struct {
			Value     []driveItem `json:"value"`
			NextLink  string      `json:"@odata.nextLink"`
			DeltaLink string      `json:"@odata.deltaLink"`
		}

		if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
			resp.Body.Close()
			return nil, err
		}
		resp.Body.Close()

		for _, item := range result.Value {
			change := DriveChange{
				ID:       item.ID,
				Name:     item.Name,
				Path:     item.getPath(),
				IsFolder: item.Folder != nil,
				Deleted:  item.Deleted != nil,
				Modified: item.LastModifiedDateTime,
			}
			changes = append(changes, change)
		}

		if result.NextLink != "" {
			endpoint = strings.TrimPrefix(result.NextLink, "https://graph.microsoft.com/v1.0")
		} else {
			endpoint = ""
		}

		if result.DeltaLink != "" {
			fs.deltaLink = result.DeltaLink
		}
	}

	return changes, nil
}

// MkdirAll is not supported for SharePoint (read-only)
func (fs *SharePointFileSystem) MkdirAll(path string, perm os.FileMode) error {
	return fmt.Errorf("SharePoint filesystem is read-only")
}

// Join joins path elements
func (fs *SharePointFileSystem) Join(elem ...string) string {
	return filepath.Join(elem...)
}

// Base returns the last element of path
func (fs *SharePointFileSystem) Base(path string) string {
	return filepath.Base(path)
}

// Dir returns all but the last element of path
func (fs *SharePointFileSystem) Dir(path string) string {
	return filepath.Dir(path)
}

// Ext returns the file name extension
func (fs *SharePointFileSystem) Ext(path string) string {
	return filepath.Ext(path)
}

// DriveChange represents a change in SharePoint/OneDrive
type DriveChange struct {
	ID       string
	Name     string
	Path     string
	IsFolder bool
	Deleted  bool
	Modified time.Time
}

// driveItem represents a SharePoint/OneDrive item
type driveItem struct {
	ID                   string       `json:"id"`
	Name                 string       `json:"name"`
	Size                 int64        `json:"size"`
	LastModifiedDateTime time.Time    `json:"lastModifiedDateTime"`
	CreatedDateTime      time.Time    `json:"createdDateTime"`
	Folder               *folderFacet `json:"folder,omitempty"`
	File                 *fileFacet   `json:"file,omitempty"`
	Deleted              *struct{}    `json:"deleted,omitempty"`
	ParentReference      *struct {
		Path string `json:"path"`
	} `json:"parentReference,omitempty"`
}

func (d *driveItem) getPath() string {
	if d.ParentReference != nil && d.ParentReference.Path != "" {
		// Path format: /drives/{driveId}/root:/path
		parts := strings.SplitN(d.ParentReference.Path, ":/", 2)
		if len(parts) == 2 {
			return parts[1] + "/" + d.Name
		}
	}
	return d.Name
}

type folderFacet struct {
	ChildCount int `json:"childCount"`
}

type fileFacet struct {
	MimeType string `json:"mimeType"`
}

// sharePointFile implements File interface
type sharePointFile struct {
	reader *bytes.Reader
	name   string
	size   int64
}

func (f *sharePointFile) Read(p []byte) (n int, err error) {
	return f.reader.Read(p)
}

func (f *sharePointFile) ReadAt(p []byte, off int64) (n int, err error) {
	return f.reader.ReadAt(p, off)
}

func (f *sharePointFile) Close() error {
	return nil
}

func (f *sharePointFile) Stat() (FileInfo, error) {
	return &sharePointFileInfo{
		item: driveItem{
			Name: f.name,
			Size: f.size,
		},
	}, nil
}

// sharePointFileInfo implements FileInfo interface
type sharePointFileInfo struct {
	item driveItem
}

func (fi *sharePointFileInfo) Name() string       { return fi.item.Name }
func (fi *sharePointFileInfo) Size() int64        { return fi.item.Size }
func (fi *sharePointFileInfo) Mode() os.FileMode  { return 0444 }
func (fi *sharePointFileInfo) ModTime() time.Time { return fi.item.LastModifiedDateTime }
func (fi *sharePointFileInfo) IsDir() bool        { return fi.item.Folder != nil }
func (fi *sharePointFileInfo) Sys() interface{}   { return fi.item }
