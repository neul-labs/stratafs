package protocol

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"sync"

	"agentfs/pkg/database"
	"agentfs/pkg/search"
)

// ModelContextProtocol represents the MCP server
type ModelContextProtocol struct {
	databases    map[string]*database.DB
	searchEngine *search.Engine
	server       *http.Server
	wg           sync.WaitGroup
}

// NewModelContextProtocol creates a new MCP server
func NewModelContextProtocol(databases map[string]*database.DB, searchEngine *search.Engine) *ModelContextProtocol {
	return &ModelContextProtocol{
		databases:    databases,
		searchEngine: searchEngine,
	}
}

// Start starts the MCP server
func (mcp *ModelContextProtocol) Start() error {
	mux := http.NewServeMux()
	
	// Register MCP endpoints
	mux.HandleFunc("/mcp", mcp.handleMCP)
	mux.HandleFunc("/mcp/search", mcp.handleUnifiedSearch)
	mux.HandleFunc("/mcp/documents/", mcp.handleDocuments)
	mux.HandleFunc("/mcp/resources", mcp.handleResources)
	
	// Create server
	mcp.server = &http.Server{
		Addr:    ":8081",
		Handler: mux,
	}
	
	// Start server in a goroutine
	mcp.wg.Add(1)
	go func() {
		defer mcp.wg.Done()
		fmt.Println("Starting Model Context Protocol server on :8081")
		if err := mcp.server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			fmt.Printf("MCP server error: %v\n", err)
		}
	}()
	
	return nil
}

// Stop stops the MCP server
func (mcp *ModelContextProtocol) Stop() error {
	if mcp.server == nil {
		return nil
	}
	
	// Shutdown server
	if err := mcp.server.Close(); err != nil {
		return err
	}
	
	// Wait for goroutines to finish
	mcp.wg.Wait()
	return nil
}

// handleMCP handles the main MCP endpoint
func (mcp *ModelContextProtocol) handleMCP(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	
	response := map[string]interface{}{
		"protocol": "mcp",
		"version":  "1.0.0",
		"capabilities": []string{
			"search",
			"resources",
		},
	}
	
	json.NewEncoder(w).Encode(response)
}

// handleSearch handles search requests from LLMs
func (mcp *ModelContextProtocol) handleSearch(w http.ResponseWriter, r *http.Request) {
	query := r.URL.Query().Get("q")
	if query == "" {
		http.Error(w, "Missing query parameter 'q'", http.StatusBadRequest)
		return
	}
	
	limit := 10
	if limitStr := r.URL.Query().Get("limit"); limitStr != "" {
		// In a real implementation, we would parse the limit
	}
	
	var results []map[string]interface{}
	
	// Search across all databases
	for dir, db := range mcp.databases {
		chunks, err := db.SearchChunks(query, limit)
		if err != nil {
			fmt.Printf("Error searching in %s: %v\n", dir, err)
			continue
		}
		
		// Convert chunks to results
		for _, chunk := range chunks {
			results = append(results, map[string]interface{}{
				"file":    dir,
				"content": chunk.Content,
				"score":   1.0, // Simplified score
			})
		}
	}
	
	// Return results
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"results": results,
		"query":   query,
	})
}

// handleResources handles resource listing requests
func (mcp *ModelContextProtocol) handleResources(w http.ResponseWriter, r *http.Request) {
	var resources []map[string]interface{}
	
	// List resources from all databases
	for dir := range mcp.databases {
		resources = append(resources, map[string]interface{}{
			"type": "directory",
			"name": dir,
			"path": dir,
		})
	}
	
	// Return resources
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"resources": resources,
	})
}

// Search performs a search across all databases
func (mcp *ModelContextProtocol) Search(ctx context.Context, query string, limit int) ([]SearchResult, error) {
	var results []SearchResult
	
	// Search across all databases
	for dir, db := range mcp.databases {
		chunks, err := db.SearchChunks(query, limit)
		if err != nil {
			fmt.Printf("Error searching in %s: %v\n", dir, err)
			continue
		}
		
		// Convert chunks to results
		for _, chunk := range chunks {
			results = append(results, SearchResult{
				File:    dir,
				Content: chunk.Content,
				Score:   1.0, // Simplified score
			})
		}
	}
	
	return results, nil
}

// SearchResult represents a search result
type SearchResult struct {
	File    string  `json:"file"`
	Content string  `json:"content"`
	Score   float64 `json:"score"`
}

// handleUnifiedSearch handles unified search requests via MCP (replacing both basic and hybrid search)
func (mcp *ModelContextProtocol) handleUnifiedSearch(w http.ResponseWriter, r *http.Request) {
	if mcp.searchEngine == nil {
		// Fallback to basic search if search engine not available
		mcp.handleSearch(w, r)
		return
	}

	// Parse search request
	var req search.SearchRequest
	if r.Method == "POST" {
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, fmt.Sprintf("Invalid JSON: %v", err), http.StatusBadRequest)
			return
		}
	} else {
		// Parse query parameters for simple requests
		req = search.SearchRequest{
			Query:           r.URL.Query().Get("q"),
			Mode:            search.SearchMode(r.URL.Query().Get("mode")),
			Limit:           10,
			Offset:          0,
			IncludeContent:  true,
			IncludeMetadata: true,
		}
		if req.Mode == "" {
			req.Mode = search.SearchModeHybrid
		}
	}

	// Perform unified search
	response, err := mcp.searchEngine.Search(&req)
	if err != nil {
		http.Error(w, fmt.Sprintf("Search failed: %v", err), http.StatusInternalServerError)
		return
	}

	// Return MCP-compatible response
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"type":    "search_response",
		"results": response.Results,
		"total":   response.Total,
		"query":   response.Query,
		"mode":    response.Mode,
		"facets":  response.Facets,
	})
}

// handleDocuments handles document retrieval requests via MCP
func (mcp *ModelContextProtocol) handleDocuments(w http.ResponseWriter, r *http.Request) {
	if mcp.searchEngine == nil {
		http.Error(w, "Search engine not available", http.StatusServiceUnavailable)
		return
	}

	// Parse document request from query parameters
	req := search.DocumentRequest{
		IncludeChunks:   r.URL.Query().Get("include_chunks") == "true",
		IncludeMetadata: r.URL.Query().Get("include_metadata") == "true",
		Format:          r.URL.Query().Get("format"),
	}

	if req.Format == "" {
		req.Format = "json"
	}

	// Extract file path from URL path
	path := r.URL.Path
	if len(path) > len("/mcp/documents/") {
		req.FilePath = path[len("/mcp/documents/"):]
	}

	if req.FilePath == "" {
		http.Error(w, "Missing file path", http.StatusBadRequest)
		return
	}

	// Get document
	response, err := mcp.searchEngine.GetDocument(&req)
	if err != nil {
		http.Error(w, fmt.Sprintf("Document not found: %v", err), http.StatusNotFound)
		return
	}

	// Return MCP-compatible response
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"type":     "document_response",
		"file":     response.File,
		"chunks":   response.Chunks,
		"metadata": response.Metadata,
	})
}