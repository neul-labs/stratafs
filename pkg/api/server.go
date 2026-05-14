package api

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/neul-labs/stratafs/pkg/config"
	"github.com/neul-labs/stratafs/pkg/database"
	"github.com/neul-labs/stratafs/pkg/queue"
	"github.com/neul-labs/stratafs/pkg/search"
)

// Server represents the HTTP API server
type Server struct {
	config       *config.Config
	databases    map[string]*database.DB
	queue        *queue.Queue
	searchEngine *search.Engine
	server       *http.Server
}

// SearchResult represents a search result
type SearchResult struct {
	File  string `json:"file"`
	Chunk string `json:"chunk"`
	Score float64 `json:"score"`
}

// NewServer creates a new API server
func NewServer(cfg *config.Config, databases map[string]*database.DB, jobQueue *queue.Queue, searchEngine *search.Engine) *Server {
	return &Server{
		config:       cfg,
		databases:    databases,
		queue:        jobQueue,
		searchEngine: searchEngine,
	}
}

// Start starts the HTTP server
func (s *Server) Start() error {
	mux := http.NewServeMux()
	
	// Register handlers
	mux.HandleFunc("/search", s.handleUnifiedSearch)
	mux.HandleFunc("/documents/", s.handleDocument)
	mux.HandleFunc("/health", s.handleHealth)
	mux.HandleFunc("/queue/stats", s.handleQueueStats)
	mux.HandleFunc("/docs", s.handleSwagger)
	mux.HandleFunc("/redoc", s.handleRedoc)
	mux.HandleFunc("/openapi.json", s.handleOpenAPI)
	
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

// handleUnifiedSearch performs unified search with multiple modes and comprehensive filtering
func (s *Server) handleUnifiedSearch(w http.ResponseWriter, r *http.Request) {
	if s.searchEngine == nil {
		http.Error(w, "Search engine not available", http.StatusServiceUnavailable)
		return
	}

	// Parse search request
	var req search.SearchRequest

	if r.Method == "POST" {
		// Parse JSON body for complex requests
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, fmt.Sprintf("Invalid JSON: %v", err), http.StatusBadRequest)
			return
		}
	} else {
		// Parse query parameters for simple requests
		req = s.parseSearchParams(r)
	}

	// Validate required parameters
	if req.Query == "" && req.Mode != search.SearchModeFaceted {
		http.Error(w, "Missing required parameter 'q' (query)", http.StatusBadRequest)
		return
	}

	// Set default mode if not specified
	if req.Mode == "" {
		req.Mode = search.SearchModeHybrid
	}

	// Perform search
	response, err := s.searchEngine.Search(&req)
	if err != nil {
		http.Error(w, fmt.Sprintf("Search failed: %v", err), http.StatusInternalServerError)
		return
	}

	// Return response
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// handleQueueStats returns job queue statistics
func (s *Server) handleQueueStats(w http.ResponseWriter, r *http.Request) {
	if s.queue == nil {
		http.Error(w, "Queue not available", http.StatusServiceUnavailable)
		return
	}

	stats, err := s.queue.GetQueueStats()
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to get queue stats: %v", err), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"queue_stats": stats,
		"timestamp":   time.Now(),
	})
}


// handleDocument handles document retrieval requests
func (s *Server) handleDocument(w http.ResponseWriter, r *http.Request) {
	if s.searchEngine == nil {
		http.Error(w, "Search engine not available", http.StatusServiceUnavailable)
		return
	}

	// Parse document request
	var req search.DocumentRequest

	// Extract file ID from URL path
	pathParts := strings.Split(strings.TrimPrefix(r.URL.Path, "/documents/"), "/")
	if len(pathParts) > 0 && pathParts[0] != "" {
		if fileID, err := strconv.ParseInt(pathParts[0], 10, 64); err == nil {
			req.FileID = fileID
		} else {
			// Try as file path
			req.FilePath = strings.Join(pathParts, "/")
		}
	}

	// Parse query parameters
	if r.URL.Query().Get("include_chunks") == "true" {
		req.IncludeChunks = true
	}
	if r.URL.Query().Get("include_metadata") == "true" {
		req.IncludeMetadata = true
	}
	req.Format = r.URL.Query().Get("format")
	if req.Format == "" {
		req.Format = "json"
	}

	// Get document
	response, err := s.searchEngine.GetDocument(&req)
	if err != nil {
		http.Error(w, fmt.Sprintf("Document not found: %v", err), http.StatusNotFound)
		return
	}

	// Return response based on format
	w.Header().Set("Content-Type", "application/json")
	if req.Format == "text" && response.File != nil {
		w.Header().Set("Content-Type", "text/plain")
		w.Write([]byte(response.File.Content))
		return
	}

	json.NewEncoder(w).Encode(response)
}

// parseSearchParams parses query parameters into a SearchRequest
func (s *Server) parseSearchParams(r *http.Request) search.SearchRequest {
	req := search.SearchRequest{
		Query:            r.URL.Query().Get("q"),
		Mode:             search.SearchMode(r.URL.Query().Get("mode")),
		Limit:            10,
		Offset:           0,
		SortBy:           r.URL.Query().Get("sort_by"),
		SortOrder:        r.URL.Query().Get("sort_order"),
		IncludeContent:   r.URL.Query().Get("include_content") == "true",
		IncludeMetadata:  r.URL.Query().Get("include_metadata") == "true",
		HighlightResults: r.URL.Query().Get("highlight") == "true",
	}

	// Parse limit and offset
	if limitStr := r.URL.Query().Get("limit"); limitStr != "" {
		if limit, err := strconv.Atoi(limitStr); err == nil && limit > 0 {
			req.Limit = limit
		}
	}
	if offsetStr := r.URL.Query().Get("offset"); offsetStr != "" {
		if offset, err := strconv.Atoi(offsetStr); err == nil && offset >= 0 {
			req.Offset = offset
		}
	}

	// Parse filters
	req.Filters = &search.SearchFilters{}

	// File extensions
	if exts := r.URL.Query().Get("extensions"); exts != "" {
		req.Filters.FileExtensions = strings.Split(exts, ",")
	}

	// File types
	if types := r.URL.Query().Get("types"); types != "" {
		req.Filters.FileTypes = strings.Split(types, ",")
	}

	// Directories
	if dirs := r.URL.Query().Get("directories"); dirs != "" {
		req.Filters.Directories = strings.Split(dirs, ",")
	}

	// Size filters
	if minSizeStr := r.URL.Query().Get("min_size"); minSizeStr != "" {
		if minSize, err := strconv.ParseInt(minSizeStr, 10, 64); err == nil {
			req.Filters.MinSize = &minSize
		}
	}
	if maxSizeStr := r.URL.Query().Get("max_size"); maxSizeStr != "" {
		if maxSize, err := strconv.ParseInt(maxSizeStr, 10, 64); err == nil {
			req.Filters.MaxSize = &maxSize
		}
	}

	// Parse weights for weighted search
	if req.Mode == search.SearchModeWeighted {
		req.Weights = &search.SearchWeights{}

		if ftWeight := r.URL.Query().Get("weight_fulltext"); ftWeight != "" {
			if w, err := strconv.ParseFloat(ftWeight, 64); err == nil {
				req.Weights.FullText = w
			}
		}
		if vecWeight := r.URL.Query().Get("weight_vector"); vecWeight != "" {
			if w, err := strconv.ParseFloat(vecWeight, 64); err == nil {
				req.Weights.Vector = w
			}
		}
		if recWeight := r.URL.Query().Get("weight_recency"); recWeight != "" {
			if w, err := strconv.ParseFloat(recWeight, 64); err == nil {
				req.Weights.Recency = w
			}
		}
	}

	return req
}

// handleOpenAPI serves the OpenAPI specification
func (s *Server) handleOpenAPI(w http.ResponseWriter, r *http.Request) {
	spec := s.getOpenAPISpec()
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(spec)
}

// handleSwagger serves the Swagger UI
func (s *Server) handleSwagger(w http.ResponseWriter, r *http.Request) {
	html := `<!DOCTYPE html>
<html>
<head>
	<title>StrataFS API Documentation</title>
	<link rel="stylesheet" type="text/css" href="https://unpkg.com/swagger-ui-dist@5.11.0/swagger-ui.css" />
	<style>
		html { box-sizing: border-box; overflow: -moz-scrollbars-vertical; overflow-y: scroll; }
		*, *:before, *:after { box-sizing: inherit; }
		body { margin:0; background: #fafafa; }
	</style>
</head>
<body>
	<div id="swagger-ui"></div>
	<script src="https://unpkg.com/swagger-ui-dist@5.11.0/swagger-ui-bundle.js"></script>
	<script src="https://unpkg.com/swagger-ui-dist@5.11.0/swagger-ui-standalone-preset.js"></script>
	<script>
		window.onload = function() {
			const ui = SwaggerUIBundle({
				url: '/openapi.json',
				dom_id: '#swagger-ui',
				deepLinking: true,
				presets: [
					SwaggerUIBundle.presets.apis,
					SwaggerUIStandalonePreset
				],
				plugins: [
					SwaggerUIBundle.plugins.DownloadUrl
				],
				layout: "StandaloneLayout"
			});
		};
	</script>
</body>
</html>`
	w.Header().Set("Content-Type", "text/html")
	w.Write([]byte(html))
}

// handleRedoc serves the ReDoc documentation
func (s *Server) handleRedoc(w http.ResponseWriter, r *http.Request) {
	html := `<!DOCTYPE html>
<html>
<head>
	<title>StrataFS API Documentation</title>
	<meta charset="utf-8"/>
	<meta name="viewport" content="width=device-width, initial-scale=1">
	<link href="https://fonts.googleapis.com/css?family=Montserrat:300,400,700|Roboto:300,400,700" rel="stylesheet">
	<style>
		body { margin: 0; padding: 0; }
	</style>
</head>
<body>
	<redoc spec-url='/openapi.json'></redoc>
	<script src="https://cdn.jsdelivr.net/npm/redoc@2.1.3/bundles/redoc.standalone.js"></script>
</body>
</html>`
	w.Header().Set("Content-Type", "text/html")
	w.Write([]byte(html))
}

// getOpenAPISpec returns the comprehensive OpenAPI specification
func (s *Server) getOpenAPISpec() map[string]interface{} {
	return map[string]interface{}{
		"openapi": "3.0.3",
		"info": map[string]interface{}{
			"title":       "StrataFS API",
			"description": "StrataFS - The Agentic Filesystem with hybrid search capabilities",
			"version":     s.config.Version,
			"contact": map[string]interface{}{
				"name": "StrataFS",
				"url":  "https://github.com/neul-labs/stratafs",
			},
		},
		"servers": []map[string]interface{}{
			{
				"url":         "http://localhost:8080",
				"description": "Local development server",
			},
		},
		"paths": map[string]interface{}{
			"/search": map[string]interface{}{
				"get": map[string]interface{}{
					"summary":     "Search files and content",
					"description": "Perform comprehensive search across monitored directories with multiple search modes and faceted filtering",
					"parameters": []map[string]interface{}{
						{
							"name":        "q",
							"in":          "query",
							"description": "Search query text",
							"required":    false,
							"schema":      map[string]interface{}{"type": "string"},
							"example":     "function authentication",
						},
						{
							"name":        "mode",
							"in":          "query",
							"description": "Search mode",
							"required":    false,
							"schema": map[string]interface{}{
								"type": "string",
								"enum": []string{"hybrid", "fulltext", "vector", "faceted", "weighted"},
								"default": "hybrid",
							},
						},
						{
							"name":        "limit",
							"in":          "query",
							"description": "Maximum number of results to return",
							"required":    false,
							"schema":      map[string]interface{}{"type": "integer", "default": 10, "minimum": 1, "maximum": 100},
						},
						{
							"name":        "offset",
							"in":          "query",
							"description": "Number of results to skip (for pagination)",
							"required":    false,
							"schema":      map[string]interface{}{"type": "integer", "default": 0, "minimum": 0},
						},
						{
							"name":        "sort_by",
							"in":          "query",
							"description": "Sort results by field",
							"required":    false,
							"schema": map[string]interface{}{
								"type": "string",
								"enum": []string{"relevance", "modified", "created", "size", "name"},
								"default": "relevance",
							},
						},
						{
							"name":        "sort_order",
							"in":          "query",
							"description": "Sort order",
							"required":    false,
							"schema": map[string]interface{}{
								"type": "string",
								"enum": []string{"asc", "desc"},
								"default": "desc",
							},
						},
						{
							"name":        "extensions",
							"in":          "query",
							"description": "Filter by file extensions (comma-separated)",
							"required":    false,
							"schema":      map[string]interface{}{"type": "string"},
							"example":     ".go,.py,.js",
						},
						{
							"name":        "types",
							"in":          "query",
							"description": "Filter by file types (comma-separated)",
							"required":    false,
							"schema":      map[string]interface{}{"type": "string"},
							"example":     "code,document",
						},
						{
							"name":        "directories",
							"in":          "query",
							"description": "Filter by directories (comma-separated)",
							"required":    false,
							"schema":      map[string]interface{}{"type": "string"},
							"example":     "/src,/docs",
						},
						{
							"name":        "min_size",
							"in":          "query",
							"description": "Minimum file size in bytes",
							"required":    false,
							"schema":      map[string]interface{}{"type": "integer", "minimum": 0},
						},
						{
							"name":        "max_size",
							"in":          "query",
							"description": "Maximum file size in bytes",
							"required":    false,
							"schema":      map[string]interface{}{"type": "integer", "minimum": 0},
						},
						{
							"name":        "include_content",
							"in":          "query",
							"description": "Include full content in results",
							"required":    false,
							"schema":      map[string]interface{}{"type": "boolean", "default": false},
						},
						{
							"name":        "include_metadata",
							"in":          "query",
							"description": "Include file metadata in results",
							"required":    false,
							"schema":      map[string]interface{}{"type": "boolean", "default": true},
						},
						{
							"name":        "highlight",
							"in":          "query",
							"description": "Enable result highlighting",
							"required":    false,
							"schema":      map[string]interface{}{"type": "boolean", "default": false},
						},
						{
							"name":        "weight_fulltext",
							"in":          "query",
							"description": "Weight for full-text search (weighted mode only)",
							"required":    false,
							"schema":      map[string]interface{}{"type": "number", "minimum": 0.0, "maximum": 1.0},
						},
						{
							"name":        "weight_vector",
							"in":          "query",
							"description": "Weight for vector search (weighted mode only)",
							"required":    false,
							"schema":      map[string]interface{}{"type": "number", "minimum": 0.0, "maximum": 1.0},
						},
						{
							"name":        "weight_recency",
							"in":          "query",
							"description": "Weight for file recency (weighted mode only)",
							"required":    false,
							"schema":      map[string]interface{}{"type": "number", "minimum": 0.0, "maximum": 1.0},
						},
					},
					"responses": map[string]interface{}{
						"200": map[string]interface{}{
							"description": "Search results",
							"content": map[string]interface{}{
								"application/json": map[string]interface{}{
									"schema": map[string]interface{}{"$ref": "#/components/schemas/SearchResponse"},
								},
							},
						},
						"400": map[string]interface{}{
							"description": "Bad request - invalid parameters",
							"content": map[string]interface{}{
								"application/json": map[string]interface{}{
									"schema": map[string]interface{}{"$ref": "#/components/schemas/Error"},
								},
							},
						},
						"503": map[string]interface{}{
							"description": "Service unavailable - search engine not ready",
							"content": map[string]interface{}{
								"application/json": map[string]interface{}{
									"schema": map[string]interface{}{"$ref": "#/components/schemas/Error"},
								},
							},
						},
					},
				},
				"post": map[string]interface{}{
					"summary":     "Advanced search with JSON payload",
					"description": "Perform advanced search with complex filters and options using JSON request body",
					"requestBody": map[string]interface{}{
						"required": true,
						"content": map[string]interface{}{
							"application/json": map[string]interface{}{
								"schema": map[string]interface{}{"$ref": "#/components/schemas/SearchRequest"},
							},
						},
					},
					"responses": map[string]interface{}{
						"200": map[string]interface{}{
							"description": "Search results",
							"content": map[string]interface{}{
								"application/json": map[string]interface{}{
									"schema": map[string]interface{}{"$ref": "#/components/schemas/SearchResponse"},
								},
							},
						},
						"400": map[string]interface{}{
							"description": "Bad request - invalid JSON or parameters",
							"content": map[string]interface{}{
								"application/json": map[string]interface{}{
									"schema": map[string]interface{}{"$ref": "#/components/schemas/Error"},
								},
							},
						},
						"503": map[string]interface{}{
							"description": "Service unavailable - search engine not ready",
							"content": map[string]interface{}{
								"application/json": map[string]interface{}{
									"schema": map[string]interface{}{"$ref": "#/components/schemas/Error"},
								},
							},
						},
					},
				},
			},
			"/documents/{path}": map[string]interface{}{
				"get": map[string]interface{}{
					"summary":     "Retrieve full document",
					"description": "Get complete document content with optional chunks and metadata",
					"parameters": []map[string]interface{}{
						{
							"name":        "path",
							"in":          "path",
							"description": "File path or ID",
							"required":    true,
							"schema":      map[string]interface{}{"type": "string"},
						},
						{
							"name":        "include_chunks",
							"in":          "query",
							"description": "Include text chunks in response",
							"required":    false,
							"schema":      map[string]interface{}{"type": "boolean", "default": false},
						},
						{
							"name":        "include_metadata",
							"in":          "query",
							"description": "Include file metadata in response",
							"required":    false,
							"schema":      map[string]interface{}{"type": "boolean", "default": true},
						},
						{
							"name":        "format",
							"in":          "query",
							"description": "Response format",
							"required":    false,
							"schema": map[string]interface{}{
								"type": "string",
								"enum": []string{"json", "text"},
								"default": "json",
							},
						},
					},
					"responses": map[string]interface{}{
						"200": map[string]interface{}{
							"description": "Document content",
							"content": map[string]interface{}{
								"application/json": map[string]interface{}{
									"schema": map[string]interface{}{"$ref": "#/components/schemas/DocumentResponse"},
								},
								"text/plain": map[string]interface{}{
									"schema": map[string]interface{}{"type": "string"},
								},
							},
						},
						"404": map[string]interface{}{
							"description": "Document not found",
							"content": map[string]interface{}{
								"application/json": map[string]interface{}{
									"schema": map[string]interface{}{"$ref": "#/components/schemas/Error"},
								},
							},
						},
					},
				},
			},
			"/health": map[string]interface{}{
				"get": map[string]interface{}{
					"summary":     "Health check",
					"description": "Check API server health and version",
					"responses": map[string]interface{}{
						"200": map[string]interface{}{
							"description": "Server is healthy",
							"content": map[string]interface{}{
								"application/json": map[string]interface{}{
									"schema": map[string]interface{}{
										"type": "object",
										"properties": map[string]interface{}{
											"status":  map[string]interface{}{"type": "string", "example": "ok"},
											"version": map[string]interface{}{"type": "string", "example": "0.1.0"},
										},
									},
								},
							},
						},
					},
				},
			},
			"/queue/stats": map[string]interface{}{
				"get": map[string]interface{}{
					"summary":     "Queue statistics",
					"description": "Get job queue statistics and status",
					"responses": map[string]interface{}{
						"200": map[string]interface{}{
							"description": "Queue statistics",
							"content": map[string]interface{}{
								"application/json": map[string]interface{}{
									"schema": map[string]interface{}{
										"type": "object",
										"properties": map[string]interface{}{
											"queue_stats": map[string]interface{}{"type": "object"},
											"timestamp":   map[string]interface{}{"type": "string", "format": "date-time"},
										},
									},
								},
							},
						},
						"503": map[string]interface{}{
							"description": "Queue not available",
							"content": map[string]interface{}{
								"application/json": map[string]interface{}{
									"schema": map[string]interface{}{"$ref": "#/components/schemas/Error"},
								},
							},
						},
					},
				},
			},
		},
		"components": map[string]interface{}{
			"schemas": s.getOpenAPISchemas(),
		},
	}
}

// getOpenAPISchemas returns all schema definitions for the API
func (s *Server) getOpenAPISchemas() map[string]interface{} {
	return map[string]interface{}{
		"SearchRequest": map[string]interface{}{
			"type":        "object",
			"description": "Search request with comprehensive filtering options",
			"properties": map[string]interface{}{
				"query": map[string]interface{}{
					"type":        "string",
					"description": "Search query text",
					"example":     "function authentication",
				},
				"mode": map[string]interface{}{
					"type":        "string",
					"description": "Search mode",
					"enum":        []string{"hybrid", "fulltext", "vector", "faceted", "weighted"},
					"default":     "hybrid",
				},
				"weights": map[string]interface{}{
					"$ref": "#/components/schemas/SearchWeights",
				},
				"filters": map[string]interface{}{
					"$ref": "#/components/schemas/SearchFilters",
				},
				"limit": map[string]interface{}{
					"type":    "integer",
					"default": 10,
					"minimum": 1,
					"maximum": 100,
				},
				"offset": map[string]interface{}{
					"type":    "integer",
					"default": 0,
					"minimum": 0,
				},
				"sort_by": map[string]interface{}{
					"type":    "string",
					"enum":    []string{"relevance", "modified", "created", "size", "name"},
					"default": "relevance",
				},
				"sort_order": map[string]interface{}{
					"type":    "string",
					"enum":    []string{"asc", "desc"},
					"default": "desc",
				},
				"include_content": map[string]interface{}{
					"type":    "boolean",
					"default": false,
				},
				"include_metadata": map[string]interface{}{
					"type":    "boolean",
					"default": true,
				},
				"highlight_results": map[string]interface{}{
					"type":    "boolean",
					"default": false,
				},
			},
		},
		"SearchWeights": map[string]interface{}{
			"type":        "object",
			"description": "Weights for different search components (weighted mode only)",
			"properties": map[string]interface{}{
				"fulltext": map[string]interface{}{
					"type":    "number",
					"minimum": 0.0,
					"maximum": 1.0,
					"example": 0.4,
				},
				"vector": map[string]interface{}{
					"type":    "number",
					"minimum": 0.0,
					"maximum": 1.0,
					"example": 0.3,
				},
				"recency": map[string]interface{}{
					"type":    "number",
					"minimum": 0.0,
					"maximum": 1.0,
					"example": 0.1,
				},
				"filename": map[string]interface{}{
					"type":    "number",
					"minimum": 0.0,
					"maximum": 1.0,
					"example": 0.1,
				},
				"filetype": map[string]interface{}{
					"type":    "number",
					"minimum": 0.0,
					"maximum": 1.0,
					"example": 0.05,
				},
				"filesize": map[string]interface{}{
					"type":    "number",
					"minimum": 0.0,
					"maximum": 1.0,
					"example": 0.05,
				},
			},
		},
		"SearchFilters": map[string]interface{}{
			"type":        "object",
			"description": "Faceted search filters",
			"properties": map[string]interface{}{
				"file_extensions": map[string]interface{}{
					"type":        "array",
					"description": "Filter by file extensions",
					"items":       map[string]interface{}{"type": "string"},
					"example":     []string{".go", ".py", ".js"},
				},
				"file_types": map[string]interface{}{
					"type":        "array",
					"description": "Filter by file types",
					"items":       map[string]interface{}{"type": "string"},
					"example":     []string{"code", "document", "text"},
				},
				"directories": map[string]interface{}{
					"type":        "array",
					"description": "Filter by directories",
					"items":       map[string]interface{}{"type": "string"},
					"example":     []string{"/src", "/docs"},
				},
				"min_size": map[string]interface{}{
					"type":        "integer",
					"description": "Minimum file size in bytes",
					"minimum":     0,
				},
				"max_size": map[string]interface{}{
					"type":        "integer",
					"description": "Maximum file size in bytes",
					"minimum":     0,
				},
				"modified_after": map[string]interface{}{
					"type":        "string",
					"format":      "date-time",
					"description": "Filter files modified after this date",
				},
				"modified_before": map[string]interface{}{
					"type":        "string",
					"format":      "date-time",
					"description": "Filter files modified before this date",
				},
				"has_embeddings": map[string]interface{}{
					"type":        "boolean",
					"description": "Only include files with vector embeddings",
				},
				"languages": map[string]interface{}{
					"type":        "array",
					"description": "Filter by programming languages",
					"items":       map[string]interface{}{"type": "string"},
					"example":     []string{"go", "python", "javascript"},
				},
			},
		},
		"SearchResponse": map[string]interface{}{
			"type":        "object",
			"description": "Search response with results and metadata",
			"properties": map[string]interface{}{
				"results": map[string]interface{}{
					"type":  "array",
					"items": map[string]interface{}{"$ref": "#/components/schemas/SearchResult"},
				},
				"total": map[string]interface{}{
					"type":        "integer",
					"description": "Total number of matching results",
				},
				"query": map[string]interface{}{
					"type":        "string",
					"description": "Original search query",
				},
				"mode": map[string]interface{}{
					"type":        "string",
					"description": "Search mode used",
				},
				"time_taken": map[string]interface{}{
					"type":        "string",
					"description": "Time taken to execute search",
				},
				"facets": map[string]interface{}{
					"$ref": "#/components/schemas/SearchFacets",
				},
				"limit": map[string]interface{}{
					"type": "integer",
				},
				"offset": map[string]interface{}{
					"type": "integer",
				},
				"has_more": map[string]interface{}{
					"type":        "boolean",
					"description": "Whether more results are available",
				},
			},
		},
		"SearchResult": map[string]interface{}{
			"type":        "object",
			"description": "Individual search result",
			"properties": map[string]interface{}{
				"id": map[string]interface{}{
					"type": "integer",
				},
				"file_id": map[string]interface{}{
					"type": "integer",
				},
				"file_path": map[string]interface{}{
					"type":        "string",
					"description": "Full path to the file",
				},
				"chunk_id": map[string]interface{}{
					"type":        "integer",
					"description": "Chunk ID (null for file-level results)",
					"nullable":    true,
				},
				"content": map[string]interface{}{
					"type":        "string",
					"description": "Content snippet or full content",
				},
				"snippet": map[string]interface{}{
					"type":        "string",
					"description": "Highlighted snippet",
				},
				"title": map[string]interface{}{
					"type":        "string",
					"description": "File title or first line",
				},
				"score": map[string]interface{}{
					"type":        "number",
					"description": "Overall relevance score",
				},
				"fulltext_score": map[string]interface{}{
					"type":        "number",
					"description": "Full-text search score component",
				},
				"vector_score": map[string]interface{}{
					"type":        "number",
					"description": "Vector similarity score component",
				},
				"recency_score": map[string]interface{}{
					"type":        "number",
					"description": "File recency score component",
				},
				"metadata": map[string]interface{}{
					"$ref": "#/components/schemas/FileMetadata",
				},
			},
		},
		"SearchFacets": map[string]interface{}{
			"type":        "object",
			"description": "Aggregated information for filtering",
			"properties": map[string]interface{}{
				"file_extensions": map[string]interface{}{
					"type":                 "object",
					"description":          "File extension distribution",
					"additionalProperties": map[string]interface{}{"type": "integer"},
				},
				"file_types": map[string]interface{}{
					"type":                 "object",
					"description":          "File type distribution",
					"additionalProperties": map[string]interface{}{"type": "integer"},
				},
				"directories": map[string]interface{}{
					"type":                 "object",
					"description":          "Directory distribution",
					"additionalProperties": map[string]interface{}{"type": "integer"},
				},
				"languages": map[string]interface{}{
					"type":                 "object",
					"description":          "Programming language distribution",
					"additionalProperties": map[string]interface{}{"type": "integer"},
				},
				"size_ranges": map[string]interface{}{
					"type":                 "object",
					"description":          "File size distribution",
					"additionalProperties": map[string]interface{}{"type": "integer"},
				},
				"total_files": map[string]interface{}{
					"type": "integer",
				},
				"total_chunks": map[string]interface{}{
					"type": "integer",
				},
			},
		},
		"FileMetadata": map[string]interface{}{
			"type":        "object",
			"description": "Comprehensive file metadata",
			"properties": map[string]interface{}{
				"file_name": map[string]interface{}{
					"type": "string",
				},
				"file_extension": map[string]interface{}{
					"type": "string",
				},
				"file_type": map[string]interface{}{
					"type": "string",
				},
				"directory": map[string]interface{}{
					"type": "string",
				},
				"size": map[string]interface{}{
					"type": "integer",
				},
				"checksum": map[string]interface{}{
					"type": "string",
				},
				"created_at": map[string]interface{}{
					"type":   "string",
					"format": "date-time",
				},
				"modified_at": map[string]interface{}{
					"type":   "string",
					"format": "date-time",
				},
				"indexed_at": map[string]interface{}{
					"type":   "string",
					"format": "date-time",
				},
				"content_length": map[string]interface{}{
					"type": "integer",
				},
				"chunk_count": map[string]interface{}{
					"type": "integer",
				},
				"language": map[string]interface{}{
					"type": "string",
				},
				"has_embeddings": map[string]interface{}{
					"type": "boolean",
				},
			},
		},
		"DocumentResponse": map[string]interface{}{
			"type":        "object",
			"description": "Full document response",
			"properties": map[string]interface{}{
				"file": map[string]interface{}{
					"$ref": "#/components/schemas/FileDocument",
				},
				"chunks": map[string]interface{}{
					"type":  "array",
					"items": map[string]interface{}{"$ref": "#/components/schemas/ChunkDocument"},
				},
				"metadata": map[string]interface{}{
					"$ref": "#/components/schemas/FileMetadata",
				},
			},
		},
		"FileDocument": map[string]interface{}{
			"type":        "object",
			"description": "Complete file document",
			"properties": map[string]interface{}{
				"id": map[string]interface{}{
					"type": "integer",
				},
				"path": map[string]interface{}{
					"type": "string",
				},
				"content": map[string]interface{}{
					"type": "string",
				},
				"checksum": map[string]interface{}{
					"type": "string",
				},
				"size": map[string]interface{}{
					"type": "integer",
				},
			},
		},
		"ChunkDocument": map[string]interface{}{
			"type":        "object",
			"description": "Text chunk from a document",
			"properties": map[string]interface{}{
				"id": map[string]interface{}{
					"type": "integer",
				},
				"content": map[string]interface{}{
					"type": "string",
				},
				"offset": map[string]interface{}{
					"type": "integer",
				},
				"length": map[string]interface{}{
					"type": "integer",
				},
				"embedding": map[string]interface{}{
					"type":  "array",
					"items": map[string]interface{}{"type": "number"},
				},
			},
		},
		"Error": map[string]interface{}{
			"type":        "object",
			"description": "Error response",
			"properties": map[string]interface{}{
				"error": map[string]interface{}{
					"type":        "string",
					"description": "Error message",
				},
				"code": map[string]interface{}{
					"type":        "integer",
					"description": "HTTP status code",
				},
			},
		},
	}
}