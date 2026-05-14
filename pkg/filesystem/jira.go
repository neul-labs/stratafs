package filesystem

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// JiraFileSystem implements FileSystem for Jira issues and attachments
type JiraFileSystem struct {
	baseURL    string
	email      string
	apiToken   string
	projects   []string
	localCache string
	jqlFilter  string

	client *http.Client

	// Cache of issues
	issues map[string]*jiraIssue
}

// JiraConfig holds Jira connection configuration
type JiraConfig struct {
	BaseURL            string   // e.g., "https://company.atlassian.net"
	Email              string   // User email
	APIToken           string   // API token
	Projects           []string // Project keys to sync
	LocalCache         string   // Local cache directory
	JQLFilter          string   // Additional JQL filter
	IncludeAttachments bool     // Whether to sync attachments
}

// NewJiraFileSystem creates a new Jira filesystem
func NewJiraFileSystem(config JiraConfig) (*JiraFileSystem, error) {
	fs := &JiraFileSystem{
		baseURL:    strings.TrimSuffix(config.BaseURL, "/"),
		email:      config.Email,
		apiToken:   config.APIToken,
		projects:   config.Projects,
		localCache: config.LocalCache,
		jqlFilter:  config.JQLFilter,
		client:     &http.Client{Timeout: 30 * time.Second},
		issues:     make(map[string]*jiraIssue),
	}

	return fs, nil
}

// jiraRequest makes an authenticated request to Jira API
func (fs *JiraFileSystem) jiraRequest(method, endpoint string, body io.Reader) (*http.Response, error) {
	url := fs.baseURL + "/rest/api/3" + endpoint

	req, err := http.NewRequest(method, url, body)
	if err != nil {
		return nil, err
	}

	// Basic auth with email:apiToken
	auth := base64.StdEncoding.EncodeToString([]byte(fs.email + ":" + fs.apiToken))
	req.Header.Set("Authorization", "Basic "+auth)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")

	return fs.client.Do(req)
}

// ReadFile reads the content of a file (issue or attachment)
func (fs *JiraFileSystem) ReadFile(name string) ([]byte, error) {
	// Check local cache first
	if fs.localCache != "" {
		cachePath := filepath.Join(fs.localCache, name)
		if data, err := os.ReadFile(cachePath); err == nil {
			return data, nil
		}
	}

	// Parse path to determine if it's an issue or attachment
	parts := strings.Split(strings.Trim(name, "/"), "/")

	if len(parts) == 1 && strings.HasSuffix(parts[0], ".md") {
		// Issue file: PROJECT-123.md
		issueKey := strings.TrimSuffix(parts[0], ".md")
		return fs.getIssueContent(issueKey)
	}

	if len(parts) == 2 && parts[0] == "_attachments" {
		// Attachment: _attachments/attachment-id-filename
		return fs.getAttachmentContent(parts[1])
	}

	return nil, fmt.Errorf("invalid path: %s", name)
}

// getIssueContent fetches and formats a Jira issue as markdown
func (fs *JiraFileSystem) getIssueContent(issueKey string) ([]byte, error) {
	endpoint := fmt.Sprintf("/issue/%s?expand=renderedFields", issueKey)

	resp, err := fs.jiraRequest("GET", endpoint, nil)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("failed to get issue: %s - %s", resp.Status, string(body))
	}

	var issue jiraIssue
	if err := json.NewDecoder(resp.Body).Decode(&issue); err != nil {
		return nil, err
	}

	// Cache issue
	fs.issues[issueKey] = &issue

	// Format as markdown
	content := fs.formatIssueAsMarkdown(&issue)

	// Cache locally
	if fs.localCache != "" {
		cachePath := filepath.Join(fs.localCache, issueKey+".md")
		_ = os.MkdirAll(filepath.Dir(cachePath), 0755)
		_ = os.WriteFile(cachePath, content, 0644)
	}

	return content, nil
}

// formatIssueAsMarkdown converts a Jira issue to markdown format
func (fs *JiraFileSystem) formatIssueAsMarkdown(issue *jiraIssue) []byte {
	var buf bytes.Buffer

	// Title
	buf.WriteString(fmt.Sprintf("# %s: %s\n\n", issue.Key, issue.Fields.Summary))

	// Metadata
	buf.WriteString("## Metadata\n\n")
	buf.WriteString(fmt.Sprintf("- **Type**: %s\n", issue.Fields.IssueType.Name))
	buf.WriteString(fmt.Sprintf("- **Status**: %s\n", issue.Fields.Status.Name))
	buf.WriteString(fmt.Sprintf("- **Priority**: %s\n", issue.Fields.Priority.Name))

	if issue.Fields.Assignee.DisplayName != "" {
		buf.WriteString(fmt.Sprintf("- **Assignee**: %s\n", issue.Fields.Assignee.DisplayName))
	}
	if issue.Fields.Reporter.DisplayName != "" {
		buf.WriteString(fmt.Sprintf("- **Reporter**: %s\n", issue.Fields.Reporter.DisplayName))
	}

	buf.WriteString(fmt.Sprintf("- **Created**: %s\n", issue.Fields.Created.Format(time.RFC3339)))
	buf.WriteString(fmt.Sprintf("- **Updated**: %s\n", issue.Fields.Updated.Format(time.RFC3339)))

	// Labels
	if len(issue.Fields.Labels) > 0 {
		buf.WriteString(fmt.Sprintf("- **Labels**: %s\n", strings.Join(issue.Fields.Labels, ", ")))
	}

	// Components
	if len(issue.Fields.Components) > 0 {
		var names []string
		for _, c := range issue.Fields.Components {
			names = append(names, c.Name)
		}
		buf.WriteString(fmt.Sprintf("- **Components**: %s\n", strings.Join(names, ", ")))
	}

	buf.WriteString("\n")

	// Description
	if issue.Fields.Description != "" {
		buf.WriteString("## Description\n\n")
		buf.WriteString(fs.convertADFToMarkdown(issue.Fields.Description))
		buf.WriteString("\n\n")
	}

	// Comments
	if len(issue.Fields.Comment.Comments) > 0 {
		buf.WriteString("## Comments\n\n")
		for _, comment := range issue.Fields.Comment.Comments {
			buf.WriteString(fmt.Sprintf("### %s - %s\n\n", comment.Author.DisplayName, comment.Created.Format("2006-01-02 15:04")))
			buf.WriteString(fs.convertADFToMarkdown(comment.Body))
			buf.WriteString("\n\n")
		}
	}

	// Attachments
	if len(issue.Fields.Attachment) > 0 {
		buf.WriteString("## Attachments\n\n")
		for _, att := range issue.Fields.Attachment {
			buf.WriteString(fmt.Sprintf("- [%s](%s) (%d bytes)\n", att.Filename, att.Content, att.Size))
		}
	}

	// Links
	if len(issue.Fields.IssueLinks) > 0 {
		buf.WriteString("\n## Related Issues\n\n")
		for _, link := range issue.Fields.IssueLinks {
			if link.OutwardIssue.Key != "" {
				buf.WriteString(fmt.Sprintf("- %s: %s\n", link.Type.Outward, link.OutwardIssue.Key))
			}
			if link.InwardIssue.Key != "" {
				buf.WriteString(fmt.Sprintf("- %s: %s\n", link.Type.Inward, link.InwardIssue.Key))
			}
		}
	}

	return buf.Bytes()
}

// convertADFToMarkdown converts Atlassian Document Format to markdown
func (fs *JiraFileSystem) convertADFToMarkdown(adf interface{}) string {
	// Simplified conversion - in production would properly parse ADF
	if str, ok := adf.(string); ok {
		return str
	}

	data, _ := json.Marshal(adf)
	var doc struct {
		Content []struct {
			Type    string `json:"type"`
			Content []struct {
				Type string `json:"type"`
				Text string `json:"text"`
			} `json:"content"`
		} `json:"content"`
	}

	if err := json.Unmarshal(data, &doc); err != nil {
		return string(data)
	}

	var result strings.Builder
	for _, block := range doc.Content {
		for _, inline := range block.Content {
			if inline.Text != "" {
				result.WriteString(inline.Text)
			}
		}
		result.WriteString("\n")
	}

	return result.String()
}

// getAttachmentContent downloads an attachment
func (fs *JiraFileSystem) getAttachmentContent(name string) ([]byte, error) {
	// Parse attachment ID from name (format: id-filename)
	parts := strings.SplitN(name, "-", 2)
	if len(parts) != 2 {
		return nil, fmt.Errorf("invalid attachment path: %s", name)
	}

	attachmentID := parts[0]

	// Get attachment metadata
	endpoint := fmt.Sprintf("/attachment/%s", attachmentID)
	resp, err := fs.jiraRequest("GET", endpoint, nil)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var att jiraAttachment
	if err := json.NewDecoder(resp.Body).Decode(&att); err != nil {
		return nil, err
	}

	// Download content
	req, err := http.NewRequest("GET", att.Content, nil)
	if err != nil {
		return nil, err
	}

	auth := base64.StdEncoding.EncodeToString([]byte(fs.email + ":" + fs.apiToken))
	req.Header.Set("Authorization", "Basic "+auth)

	resp, err = fs.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	// Cache locally
	if fs.localCache != "" {
		cachePath := filepath.Join(fs.localCache, "_attachments", name)
		_ = os.MkdirAll(filepath.Dir(cachePath), 0755)
		_ = os.WriteFile(cachePath, data, 0644)
	}

	return data, nil
}

// Open opens a file for reading
func (fs *JiraFileSystem) Open(name string) (File, error) {
	data, err := fs.ReadFile(name)
	if err != nil {
		return nil, err
	}

	return &jiraFile{
		reader: bytes.NewReader(data),
		name:   filepath.Base(name),
		size:   int64(len(data)),
	}, nil
}

// Stat returns information about a file
func (fs *JiraFileSystem) Stat(name string) (FileInfo, error) {
	parts := strings.Split(strings.Trim(name, "/"), "/")

	if len(parts) == 1 && strings.HasSuffix(parts[0], ".md") {
		issueKey := strings.TrimSuffix(parts[0], ".md")

		// Check cache
		if issue, ok := fs.issues[issueKey]; ok {
			return &jiraFileInfo{
				name:    parts[0],
				size:    0, // Unknown without fetching
				modTime: issue.Fields.Updated,
				isDir:   false,
			}, nil
		}

		// Fetch issue
		_, err := fs.getIssueContent(issueKey)
		if err != nil {
			return nil, err
		}

		issue := fs.issues[issueKey]
		return &jiraFileInfo{
			name:    parts[0],
			size:    0,
			modTime: issue.Fields.Updated,
			isDir:   false,
		}, nil
	}

	return nil, fmt.Errorf("file not found: %s", name)
}

// Walk walks all issues in configured projects
func (fs *JiraFileSystem) Walk(root string, walkFn WalkFunc) error {
	for _, project := range fs.projects {
		if err := fs.walkProject(project, walkFn); err != nil {
			return err
		}
	}
	return nil
}

func (fs *JiraFileSystem) walkProject(project string, walkFn WalkFunc) error {
	startAt := 0
	maxResults := 100

	for {
		jql := fmt.Sprintf("project = %s", project)
		if fs.jqlFilter != "" {
			jql += " AND " + fs.jqlFilter
		}

		endpoint := fmt.Sprintf("/search?jql=%s&startAt=%d&maxResults=%d&fields=key,summary,updated,attachment",
			jql, startAt, maxResults)

		resp, err := fs.jiraRequest("GET", endpoint, nil)
		if err != nil {
			return err
		}

		var result struct {
			Total  int `json:"total"`
			Issues []struct {
				Key    string `json:"key"`
				Fields struct {
					Summary    string           `json:"summary"`
					Updated    time.Time        `json:"updated"`
					Attachment []jiraAttachment `json:"attachment"`
				} `json:"fields"`
			} `json:"issues"`
		}

		if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
			resp.Body.Close()
			return err
		}
		resp.Body.Close()

		for _, issue := range result.Issues {
			// Issue as markdown file
			name := issue.Key + ".md"
			info := &jiraFileInfo{
				name:    name,
				modTime: issue.Fields.Updated,
				isDir:   false,
			}

			if err := walkFn(name, info, nil); err != nil {
				return err
			}

			// Attachments
			for _, att := range issue.Fields.Attachment {
				attName := fmt.Sprintf("_attachments/%s-%s", att.ID, att.Filename)
				attInfo := &jiraFileInfo{
					name:    att.Filename,
					size:    int64(att.Size),
					modTime: att.Created,
					isDir:   false,
				}

				if err := walkFn(attName, attInfo, nil); err != nil {
					return err
				}
			}
		}

		startAt += len(result.Issues)
		if startAt >= result.Total {
			break
		}
	}

	return nil
}

// GetChanges returns issues updated since the given time
func (fs *JiraFileSystem) GetChanges(since time.Time) ([]JiraChange, error) {
	var changes []JiraChange

	for _, project := range fs.projects {
		jql := fmt.Sprintf("project = %s AND updated >= '%s'",
			project, since.Format("2006-01-02 15:04"))

		if fs.jqlFilter != "" {
			jql += " AND " + fs.jqlFilter
		}

		startAt := 0
		maxResults := 100

		for {
			endpoint := fmt.Sprintf("/search?jql=%s&startAt=%d&maxResults=%d&fields=key,updated",
				jql, startAt, maxResults)

			resp, err := fs.jiraRequest("GET", endpoint, nil)
			if err != nil {
				return nil, err
			}

			var result struct {
				Total  int `json:"total"`
				Issues []struct {
					Key    string `json:"key"`
					Fields struct {
						Updated time.Time `json:"updated"`
					} `json:"fields"`
				} `json:"issues"`
			}

			if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
				resp.Body.Close()
				return nil, err
			}
			resp.Body.Close()

			for _, issue := range result.Issues {
				changes = append(changes, JiraChange{
					IssueKey: issue.Key,
					Updated:  issue.Fields.Updated,
				})
			}

			startAt += len(result.Issues)
			if startAt >= result.Total {
				break
			}
		}
	}

	return changes, nil
}

// MkdirAll is not supported for Jira (read-only)
func (fs *JiraFileSystem) MkdirAll(path string, perm os.FileMode) error {
	return fmt.Errorf("Jira filesystem is read-only")
}

// Join joins path elements
func (fs *JiraFileSystem) Join(elem ...string) string {
	return filepath.Join(elem...)
}

// Base returns the last element of path
func (fs *JiraFileSystem) Base(path string) string {
	return filepath.Base(path)
}

// Dir returns all but the last element of path
func (fs *JiraFileSystem) Dir(path string) string {
	return filepath.Dir(path)
}

// Ext returns the file name extension
func (fs *JiraFileSystem) Ext(path string) string {
	return filepath.Ext(path)
}

// JiraChange represents a change in Jira
type JiraChange struct {
	IssueKey string
	Updated  time.Time
}

// Jira data structures
type jiraIssue struct {
	ID     string `json:"id"`
	Key    string `json:"key"`
	Fields struct {
		Summary   string `json:"summary"`
		IssueType struct {
			Name string `json:"name"`
		} `json:"issuetype"`
		Status struct {
			Name string `json:"name"`
		} `json:"status"`
		Priority struct {
			Name string `json:"name"`
		} `json:"priority"`
		Assignee struct {
			DisplayName string `json:"displayName"`
		} `json:"assignee"`
		Reporter struct {
			DisplayName string `json:"displayName"`
		} `json:"reporter"`
		Created     time.Time        `json:"created"`
		Updated     time.Time        `json:"updated"`
		Description interface{}      `json:"description"` // Can be string or ADF
		Labels      []string         `json:"labels"`
		Components  []jiraComponent  `json:"components"`
		Attachment  []jiraAttachment `json:"attachment"`
		Comment     struct {
			Comments []jiraComment `json:"comments"`
		} `json:"comment"`
		IssueLinks []jiraIssueLink `json:"issuelinks"`
	} `json:"fields"`
}

type jiraComponent struct {
	Name string `json:"name"`
}

type jiraAttachment struct {
	ID       string    `json:"id"`
	Filename string    `json:"filename"`
	Size     int       `json:"size"`
	Content  string    `json:"content"`
	Created  time.Time `json:"created"`
}

type jiraComment struct {
	Author struct {
		DisplayName string `json:"displayName"`
	} `json:"author"`
	Body    interface{} `json:"body"`
	Created time.Time   `json:"created"`
}

type jiraIssueLink struct {
	Type struct {
		Inward  string `json:"inward"`
		Outward string `json:"outward"`
	} `json:"type"`
	InwardIssue struct {
		Key string `json:"key"`
	} `json:"inwardIssue"`
	OutwardIssue struct {
		Key string `json:"key"`
	} `json:"outwardIssue"`
}

// jiraFile implements File interface
type jiraFile struct {
	reader *bytes.Reader
	name   string
	size   int64
}

func (f *jiraFile) Read(p []byte) (n int, err error) {
	return f.reader.Read(p)
}

func (f *jiraFile) ReadAt(p []byte, off int64) (n int, err error) {
	return f.reader.ReadAt(p, off)
}

func (f *jiraFile) Close() error {
	return nil
}

func (f *jiraFile) Stat() (FileInfo, error) {
	return &jiraFileInfo{
		name: f.name,
		size: f.size,
	}, nil
}

// jiraFileInfo implements FileInfo interface
type jiraFileInfo struct {
	name    string
	size    int64
	modTime time.Time
	isDir   bool
}

func (fi *jiraFileInfo) Name() string       { return fi.name }
func (fi *jiraFileInfo) Size() int64        { return fi.size }
func (fi *jiraFileInfo) Mode() os.FileMode  { return 0444 }
func (fi *jiraFileInfo) ModTime() time.Time { return fi.modTime }
func (fi *jiraFileInfo) IsDir() bool        { return fi.isDir }
func (fi *jiraFileInfo) Sys() interface{}   { return nil }
