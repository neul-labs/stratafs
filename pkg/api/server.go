package api

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"agentfs/pkg/config"
	"agentfs/pkg/database"
)

// Server represents the HTTP API server
type Server struct {
	config    *config.Config
	databases map[string]*database.DB
	server    *http.Server
}

// SearchResult represents a search result
type SearchResult struct {
	File  string `json:"file"`
	Chunk string `json:"chunk"`
	Score float64 `json:"score"`
}

// NewServer creates a new API server
func NewServer(cfg *config.Config, databases map[string]*database.DB) *Server {
	return &Server{
		config:    cfg,
		databases: databases,
	}
}

// Start starts the HTTP server
func (s *Server) Start() error {
	mux := http.NewServeMux()
	
	// Register handlers
	mux.HandleFunc("/search", s.handleSearch)
	mux.HandleFunc("/health", s.handleHealth)
	
	// Create server
	s.server = &http.Server{
		Addr:    ":8080",
		Handler: mux,
	}
	
	// Start server in a goroutine
	go func() {
		fmt.Println("Starting API server on :8080")
		if err := s.server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			fmt.Printf("API server error: %v\n", err)
		}
	}()
	
	return nil
}

// Stop gracefully stops the server
func (s *Server) Stop() error {
	if s.server == nil {
		return nil
	}
	
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	
	return s.server.Shutdown(ctx)
}

// handleHealth responds to health checks
func (s *Server) handleHealth(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{
		"status": "ok",
		"version": s.config.Version,
	})
}

// handleSearch performs a search across all databases
func (s *Server) handleSearch(w http.ResponseWriter, r *http.Request) {
	query := r.URL.Query().Get("q")
	if query == "" {
		http.Error(w, "Missing query parameter 'q'", http.StatusBadRequest)
		return
	}
	
	limit := 10
	if limitStr := r.URL.Query().Get("limit"); limitStr != "" {
		// In a real implementation, we would parse the limit
		// For now, we'll just use the default
	}
	
	var results []SearchResult
	
	// Search across all databases
	for dir, db := range s.databases {
		chunks, err := db.SearchChunks(query, limit)
		if err != nil {
			fmt.Printf("Error searching in %s: %v\n", dir, err)
			continue
		}
		
		// Convert chunks to results
		for _, chunk := range chunks {
			results = append(results, SearchResult{
				File:  dir,
				Chunk: chunk.Content,
				Score: 1.0, // Simplified score
			})
		}
	}
	
	// Return results
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(results)
}