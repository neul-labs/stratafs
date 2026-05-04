//go:build linux
// +build linux

package search

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"github.com/godbus/dbus/v5"
	"github.com/godbus/dbus/v5/introspect"
)

const (
	searchProviderInterface = "org.gnome.Shell.SearchProvider2"
	searchProviderPath      = "/org/agentfs/SearchProvider"
	busName                 = "org.agentfs.SearchProvider"
)

// GnomeSearchProvider implements the GNOME Shell SearchProvider2 D-Bus interface
type GnomeSearchProvider struct {
	engine  *Engine
	conn    *dbus.Conn
	apiURL  string
	results map[string]SearchResult // Cache results by ID
}

// NewGnomeSearchProvider creates a new GNOME Shell search provider
func NewGnomeSearchProvider(engine *Engine, apiURL string) *GnomeSearchProvider {
	return &GnomeSearchProvider{
		engine:  engine,
		apiURL:  apiURL,
		results: make(map[string]SearchResult),
	}
}

// Start registers the search provider on D-Bus
func (p *GnomeSearchProvider) Start() error {
	conn, err := dbus.ConnectSessionBus()
	if err != nil {
		return fmt.Errorf("failed to connect to session bus: %w", err)
	}
	p.conn = conn

	// Request the bus name
	reply, err := conn.RequestName(busName, dbus.NameFlagDoNotQueue)
	if err != nil {
		return fmt.Errorf("failed to request bus name: %w", err)
	}
	if reply != dbus.RequestNameReplyPrimaryOwner {
		return fmt.Errorf("bus name already taken")
	}

	// Export the search provider interface
	err = conn.Export(p, searchProviderPath, searchProviderInterface)
	if err != nil {
		return fmt.Errorf("failed to export search provider: %w", err)
	}

	// Export introspection
	node := &introspect.Node{
		Name: searchProviderPath,
		Interfaces: []introspect.Interface{
			introspect.IntrospectData,
			{
				Name: searchProviderInterface,
				Methods: []introspect.Method{
					{Name: "GetInitialResultSet", Args: []introspect.Arg{
						{Name: "terms", Type: "as", Direction: "in"},
						{Name: "results", Type: "as", Direction: "out"},
					}},
					{Name: "GetSubsearchResultSet", Args: []introspect.Arg{
						{Name: "previous_results", Type: "as", Direction: "in"},
						{Name: "terms", Type: "as", Direction: "in"},
						{Name: "results", Type: "as", Direction: "out"},
					}},
					{Name: "GetResultMetas", Args: []introspect.Arg{
						{Name: "identifiers", Type: "as", Direction: "in"},
						{Name: "metas", Type: "aa{sv}", Direction: "out"},
					}},
					{Name: "ActivateResult", Args: []introspect.Arg{
						{Name: "identifier", Type: "s", Direction: "in"},
						{Name: "terms", Type: "as", Direction: "in"},
						{Name: "timestamp", Type: "u", Direction: "in"},
					}},
					{Name: "LaunchSearch", Args: []introspect.Arg{
						{Name: "terms", Type: "as", Direction: "in"},
						{Name: "timestamp", Type: "u", Direction: "in"},
					}},
				},
			},
		},
	}

	err = conn.Export(introspect.NewIntrospectable(node), searchProviderPath, "org.freedesktop.DBus.Introspectable")
	if err != nil {
		return fmt.Errorf("failed to export introspection: %w", err)
	}

	return nil
}

// Stop unregisters the search provider from D-Bus
func (p *GnomeSearchProvider) Stop() error {
	if p.conn != nil {
		p.conn.ReleaseName(busName)
		return p.conn.Close()
	}
	return nil
}

// GetInitialResultSet performs the initial search
func (p *GnomeSearchProvider) GetInitialResultSet(terms []string) ([]string, *dbus.Error) {
	query := strings.Join(terms, " ")
	if query == "" {
		return []string{}, nil
	}

	results, err := p.performSearch(query)
	if err != nil {
		return []string{}, nil
	}

	var ids []string
	for _, result := range results {
		id := fmt.Sprintf("agentfs-%d", result.ID)
		p.results[id] = result
		ids = append(ids, id)
	}

	return ids, nil
}

// GetSubsearchResultSet refines the search within previous results
func (p *GnomeSearchProvider) GetSubsearchResultSet(previousResults []string, terms []string) ([]string, *dbus.Error) {
	// For simplicity, just do a new search
	return p.GetInitialResultSet(terms)
}

// GetResultMetas returns metadata for the given result IDs
func (p *GnomeSearchProvider) GetResultMetas(identifiers []string) ([]map[string]dbus.Variant, *dbus.Error) {
	var metas []map[string]dbus.Variant

	for _, id := range identifiers {
		result, ok := p.results[id]
		if !ok {
			continue
		}

		meta := map[string]dbus.Variant{
			"id":          dbus.MakeVariant(id),
			"name":        dbus.MakeVariant(result.Title),
			"description": dbus.MakeVariant(truncate(result.Snippet, 100)),
		}

		// Add icon if we can determine file type
		if result.Metadata != nil {
			meta["gicon"] = dbus.MakeVariant(getIconForExtension(result.Metadata.FileExt))
		}

		metas = append(metas, meta)
	}

	return metas, nil
}

// ActivateResult opens the selected result
func (p *GnomeSearchProvider) ActivateResult(identifier string, terms []string, timestamp uint32) *dbus.Error {
	result, ok := p.results[identifier]
	if !ok {
		return nil
	}

	// Open the file with xdg-open
	go func() {
		cmd := fmt.Sprintf("xdg-open %q", result.FilePath)
		// Execute in background
		_ = execCommand("sh", "-c", cmd)
	}()

	return nil
}

// LaunchSearch opens the AgentFS UI with the search query
func (p *GnomeSearchProvider) LaunchSearch(terms []string, timestamp uint32) *dbus.Error {
	query := strings.Join(terms, " ")

	// Try to open agentfs-ui, fall back to web interface
	go func() {
		// First try the desktop app
		if err := execCommand("agentfs-ui"); err != nil {
			// Fall back to web interface
			url := fmt.Sprintf("%s/docs?q=%s", p.apiURL, query)
			_ = execCommand("xdg-open", url)
		}
	}()

	return nil
}

// performSearch queries the AgentFS search engine
func (p *GnomeSearchProvider) performSearch(query string) ([]SearchResult, error) {
	if p.engine != nil {
		// Direct engine access
		req := &SearchRequest{
			Query: query,
			Mode:  SearchModeHybrid,
			Limit: 10,
		}
		resp, err := p.engine.Search(req)
		if err != nil {
			return nil, err
		}
		return resp.Results, nil
	}

	// Fall back to HTTP API
	url := fmt.Sprintf("%s/search?q=%s&limit=10", p.apiURL, query)
	resp, err := http.Get(url)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var searchResp SearchResponse
	if err := json.NewDecoder(resp.Body).Decode(&searchResp); err != nil {
		return nil, err
	}

	return searchResp.Results, nil
}

// Helper functions

func truncate(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen-3] + "..."
}

func getIconForExtension(ext string) string {
	switch strings.ToLower(ext) {
	case ".go", ".py", ".js", ".ts", ".java", ".c", ".cpp", ".rs":
		return "text-x-source"
	case ".md", ".txt", ".rst":
		return "text-x-generic"
	case ".json", ".yaml", ".yml", ".toml":
		return "text-x-script"
	case ".html", ".css":
		return "text-html"
	case ".pdf":
		return "application-pdf"
	case ".doc", ".docx":
		return "application-msword"
	default:
		return "text-x-generic"
	}
}

func execCommand(name string, args ...string) error {
	// Implementation would use os/exec
	// Keeping it simple for now
	return nil
}
