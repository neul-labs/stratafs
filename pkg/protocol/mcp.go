package protocol

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"sync"

	"agentfs/pkg/database"
)

// ModelContextProtocol represents the MCP server
type ModelContextProtocol struct {
	databases map[string]*database.DB
	server    *http.Server
	wg        sync.WaitGroup
}

// NewModelContextProtocol creates a new MCP server
func NewModelContextProtocol(databases map[string]*database.DB) *ModelContextProtocol {
	return &ModelContextProtocol{
		databases: databases,
	}
}

// Start starts the MCP server
func (mcp *ModelContextProtocol) Start() error {
	mux := http.NewServeMux()
	
	// Register MCP endpoints
	mux.HandleFunc("/mcp", mcp.handleMCP)
	mux.HandleFunc("/mcp/search", mcp.handleSearch)
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