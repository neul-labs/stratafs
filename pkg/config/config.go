package config

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

// Removed GGUF support - using FastEmbed only

// FastEmbedModel represents available FastEmbed models
type FastEmbedModel string

const (
	FastEmbedBGEBaseEN     FastEmbedModel = "bge-base-en"
	FastEmbedBGEBaseENV15  FastEmbedModel = "bge-base-en-v1.5"
	FastEmbedBGESmallEN    FastEmbedModel = "bge-small-en"
	FastEmbedBGESmallENV15 FastEmbedModel = "bge-small-en-v1.5"
	FastEmbedAllMiniLML6V2 FastEmbedModel = "all-minilm-l6-v2"
)

// StorageType represents different storage backend types
type StorageType string

const (
	StorageTypeLocal       StorageType = "local"
	StorageTypeS3          StorageType = "s3"
	StorageTypeGCS         StorageType = "gcs"
	StorageTypeAzure       StorageType = "azure"
	StorageTypeSharePoint  StorageType = "sharepoint"
	StorageTypeGoogleDrive StorageType = "google-drive"
	StorageTypeJira        StorageType = "jira"
)

// StorageSource represents a configured storage source
type StorageSource struct {
	ID            string                 `json:"id"`              // Unique identifier for this source
	Name          string                 `json:"name"`            // Human-readable name
	Type          StorageType            `json:"type"`            // Storage type (local, s3, gcs, azure)
	Enabled       bool                   `json:"enabled"`         // Whether this source is active
	Path          string                 `json:"path"`            // Base path (local dir, S3 prefix, etc.)
	LocalCacheDir string                 `json:"local_cache_dir"` // Local directory for remote filesystem mirrors
	Credentials   map[string]interface{} `json:"credentials"`     // Storage-specific credentials
	Filters       SourceFilters          `json:"filters"`         // File filtering options
}

// SourceFilters defines filtering options for a storage source
type SourceFilters struct {
	IncludePatterns []string `json:"include_patterns"` // Glob patterns for files to include
	ExcludePatterns []string `json:"exclude_patterns"` // Glob patterns for files to exclude
	MaxFileSize     int64    `json:"max_file_size"`    // Maximum file size in bytes (0 = no limit)
	IgnoreHidden    bool     `json:"ignore_hidden"`    // Whether to ignore hidden files/directories
}

// ServerConfig holds server configuration
type ServerConfig struct {
	APIPort int `json:"api_port"` // REST API server port
	MCPPort int `json:"mcp_port"` // Model Context Protocol server port
}

// WorkerConfig holds worker pool configuration
type WorkerConfig struct {
	Count        int           `json:"count"`         // Number of worker goroutines
	ScanInterval time.Duration `json:"scan_interval"` // Interval for scanning directories
	BatchSize    int           `json:"batch_size"`    // Number of files to process in a batch
}

// EmbeddingConfig holds embedding model configuration
type EmbeddingConfig struct {
	Model       FastEmbedModel       `json:"model"`       // FastEmbed model to use
	CacheDir    string               `json:"cache_dir"`   // FastEmbed model cache directory
	Dimension   int                  `json:"dimension"`   // Embedding dimension (auto-detected)
	ModelInfo   EmbeddingModelInfo   `json:"model_info"`  // Detailed model information
	Performance EmbeddingPerformance `json:"performance"` // Performance settings
}

// EmbeddingModelInfo provides detailed information about the embedding model
type EmbeddingModelInfo struct {
	Name        string `json:"name"`        // Human-readable model name
	Description string `json:"description"` // Model description
	MaxTokens   int    `json:"max_tokens"`  // Maximum input tokens
	FileSize    string `json:"file_size"`   // Approximate model file size
	Speed       string `json:"speed"`       // Relative speed (fast/medium/slow)
	Quality     string `json:"quality"`     // Relative quality (high/medium/low)
	Language    string `json:"language"`    // Primary language support
}

// EmbeddingPerformance holds performance-related settings
type EmbeddingPerformance struct {
	BatchSize      int  `json:"batch_size"`      // Number of texts to process in a batch
	MaxConcurrency int  `json:"max_concurrency"` // Maximum concurrent embedding requests
	CacheResults   bool `json:"cache_results"`   // Whether to cache embedding results
	EnableGPU      bool `json:"enable_gpu"`      // Enable GPU acceleration if available
}

// DatabaseConfig holds database configuration
type DatabaseConfig struct {
	CompressionEnabled   bool          `json:"compression_enabled"`   // Enable text compression
	CompressionThreshold int           `json:"compression_threshold"` // Minimum bytes before compression
	MaintenanceInterval  time.Duration `json:"maintenance_interval"`  // How often to run maintenance
	DeletedThreshold     time.Duration `json:"deleted_threshold"`     // How long to keep soft-deleted records
}

// Config holds the application configuration
type Config struct {
	Version   string `json:"version"`
	AgentDir  string `json:"agent_dir"`  // Local agent directory name
	GlobalDir string `json:"global_dir"` // Global configuration directory

	// Storage sources configuration
	Sources []StorageSource `json:"sources"`

	// Server configuration
	Server ServerConfig `json:"server"`

	// Worker configuration
	Worker WorkerConfig `json:"worker"`

	// Embedding configuration
	Embedding EmbeddingConfig `json:"embedding"`

	// Database configuration
	Database DatabaseConfig `json:"database"`

	// Runtime paths (not saved to JSON)
	QueueDBPath string `json:"-"` // Path to the central queue database
}

// DefaultConfig creates a new configuration with default values
func DefaultConfig() *Config {
	// Get global directory (defaults to user's home/.stratafs)
	globalDir := os.Getenv("STRATAFS_GLOBAL_DIR")
	if globalDir == "" {
		home, _ := os.UserHomeDir()
		globalDir = filepath.Join(home, ".stratafs")
	}

	// Create default local source for current directory
	wd, _ := os.Getwd()
	defaultSource := StorageSource{
		ID:            "default-local",
		Name:          "Current Directory",
		Type:          StorageTypeLocal,
		Enabled:       true,
		Path:          wd,
		LocalCacheDir: "", // Not needed for local sources
		Filters: SourceFilters{
			IncludePatterns: []string{"*"},
			ExcludePatterns: []string{".git/**", "node_modules/**", "*.tmp", "*.log"},
			MaxFileSize:     100 * 1024 * 1024, // 100MB
			IgnoreHidden:    true,
		},
	}

	sources := []StorageSource{defaultSource}

	// Add any environment-specified directories as additional sources
	if dirs := os.Getenv("STRATAFS_DIRS"); dirs != "" {
		additionalPaths := strings.Split(dirs, ",")
		for i, path := range additionalPaths {
			cleanPath := filepath.Clean(path)
			if cleanPath == wd {
				continue // skip duplicates
			}
			additionalSource := StorageSource{
				ID:      fmt.Sprintf("env-local-%d", i),
				Name:    fmt.Sprintf("Environment Directory %d", i+1),
				Type:    StorageTypeLocal,
				Enabled: true,
				Path:    cleanPath,
				Filters: defaultSource.Filters, // Use same filters as default
			}
			sources = append(sources, additionalSource)
		}
	}

	config := &Config{
		Version:   "0.2.0",
		AgentDir:  ".stratafs",
		GlobalDir: globalDir,
		Sources:   sources,
		Server: ServerConfig{
			APIPort: 8080,
			MCPPort: 8081,
		},
		Worker: WorkerConfig{
			Count:        4,
			ScanInterval: 10 * time.Second,
			BatchSize:    10,
		},
		Embedding: EmbeddingConfig{
			Model:     FastEmbedBGEBaseENV15,
			CacheDir:  filepath.Join(globalDir, "fastembed_cache"),
			Dimension: GetModelDimension(FastEmbedBGEBaseENV15),
			ModelInfo: GetModelInfo(FastEmbedBGEBaseENV15),
			Performance: EmbeddingPerformance{
				BatchSize:      32,
				MaxConcurrency: 4,
				CacheResults:   true,
				EnableGPU:      false, // Default to CPU
			},
		},
		Database: DatabaseConfig{
			CompressionEnabled:   true,
			CompressionThreshold: 512,
			MaintenanceInterval:  24 * time.Hour,
			DeletedThreshold:     7 * 24 * time.Hour,
		},
		QueueDBPath: filepath.Join(globalDir, "queue.db"),
	}

	// Apply environment variable overrides
	config.applyEnvironmentOverrides()
	return config
}

// applyEnvironmentOverrides applies configuration from environment variables
func (c *Config) applyEnvironmentOverrides() {
	// Worker count override
	if wc := os.Getenv("STRATAFS_WORKERS"); wc != "" {
		if parsed, err := strconv.Atoi(wc); err == nil && parsed > 0 {
			c.Worker.Count = parsed
		}
	}

	// Scan interval override
	if si := os.Getenv("STRATAFS_SCAN_INTERVAL"); si != "" {
		if parsed, err := time.ParseDuration(si); err == nil {
			c.Worker.ScanInterval = parsed
		}
	}

	// API port override
	if port := os.Getenv("STRATAFS_API_PORT"); port != "" {
		if parsed, err := strconv.Atoi(port); err == nil && parsed > 0 {
			c.Server.APIPort = parsed
		}
	}

	// MCP port override
	if port := os.Getenv("STRATAFS_MCP_PORT"); port != "" {
		if parsed, err := strconv.Atoi(port); err == nil && parsed > 0 {
			c.Server.MCPPort = parsed
		}
	}

	// FastEmbed model override
	if model := os.Getenv("STRATAFS_MODEL"); model != "" {
		var newModel FastEmbedModel
		switch strings.ToLower(model) {
		case "bge-base-en":
			newModel = FastEmbedBGEBaseEN
		case "bge-base-en-v1.5":
			newModel = FastEmbedBGEBaseENV15
		case "bge-small-en":
			newModel = FastEmbedBGESmallEN
		case "bge-small-en-v1.5":
			newModel = FastEmbedBGESmallENV15
		case "all-minilm-l6-v2":
			newModel = FastEmbedAllMiniLML6V2
		default:
			newModel = c.Embedding.Model // Keep existing if invalid
		}

		// Update model and its metadata
		if newModel != c.Embedding.Model {
			c.Embedding.Model = newModel
			c.Embedding.Dimension = GetModelDimension(newModel)
			c.Embedding.ModelInfo = GetModelInfo(newModel)
		}
	}

	// FastEmbed cache directory override
	if cacheDir := os.Getenv("STRATAFS_FASTEMBED_CACHE"); cacheDir != "" {
		c.Embedding.CacheDir = cacheDir
	}
}

// LoadConfig loads configuration from a JSON file, creating default if not found
func LoadConfig() (*Config, error) {
	// Get global directory
	globalDir := os.Getenv("STRATAFS_GLOBAL_DIR")
	if globalDir == "" {
		home, _ := os.UserHomeDir()
		globalDir = filepath.Join(home, ".stratafs")
	}

	configPath := filepath.Join(globalDir, "config.json")

	// Create global directory if it doesn't exist
	if err := os.MkdirAll(globalDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create global directory: %w", err)
	}

	// Check if config file exists
	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		// Create default config and save it
		config := DefaultConfig()
		if err := config.Save(); err != nil {
			return nil, fmt.Errorf("failed to save default config: %w", err)
		}
		return config, nil
	}

	// Load existing config
	data, err := os.ReadFile(configPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read config file: %w", err)
	}

	var config Config
	if err := json.Unmarshal(data, &config); err != nil {
		return nil, fmt.Errorf("failed to parse config JSON: %w", err)
	}

	// Set runtime paths
	config.QueueDBPath = filepath.Join(config.GlobalDir, "queue.db")

	// Apply environment overrides
	config.applyEnvironmentOverrides()

	return &config, nil
}

// Save saves the configuration to a JSON file
func (c *Config) Save() error {
	configPath := filepath.Join(c.GlobalDir, "config.json")

	// Create global directory if it doesn't exist
	if err := os.MkdirAll(c.GlobalDir, 0755); err != nil {
		return fmt.Errorf("failed to create global directory: %w", err)
	}

	// Marshal to JSON with nice formatting
	data, err := json.MarshalIndent(c, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal config to JSON: %w", err)
	}

	// Write to file
	if err := os.WriteFile(configPath, data, 0644); err != nil {
		return fmt.Errorf("failed to write config file: %w", err)
	}

	return nil
}

// NewConfig creates a new configuration (kept for backward compatibility)
func NewConfig() *Config {
	return DefaultConfig()
}

// GetAgentPath returns the full path to the agent directory for a given base directory
func (c *Config) GetAgentPath(baseDir string) string {
	return filepath.Join(baseDir, c.AgentDir)
}

// GetDBPath returns the full path to the database file for a given base directory
func (c *Config) GetDBPath(baseDir string) string {
	return filepath.Join(c.GetAgentPath(baseDir), "stratafs.db")
}

// GetEnabledSources returns all enabled storage sources
func (c *Config) GetEnabledSources() []StorageSource {
	var enabled []StorageSource
	for _, source := range c.Sources {
		if source.Enabled {
			enabled = append(enabled, source)
		}
	}
	return enabled
}

// GetSourceByID returns a storage source by its ID
func (c *Config) GetSourceByID(id string) *StorageSource {
	for i := range c.Sources {
		if c.Sources[i].ID == id {
			return &c.Sources[i]
		}
	}
	return nil
}

// AddSource adds a new storage source to the configuration
func (c *Config) AddSource(source StorageSource) error {
	// Check for duplicate IDs
	if c.GetSourceByID(source.ID) != nil {
		return fmt.Errorf("source with ID %q already exists", source.ID)
	}

	// Set up local cache directory for remote sources
	if err := c.setupLocalCacheDir(&source); err != nil {
		return fmt.Errorf("failed to setup local cache directory: %w", err)
	}

	c.Sources = append(c.Sources, source)
	return nil
}

// setupLocalCacheDir sets up local cache directory for a storage source
func (c *Config) setupLocalCacheDir(source *StorageSource) error {
	// Local sources don't need cache directories
	if source.Type == StorageTypeLocal {
		source.LocalCacheDir = ""
		return nil
	}

	// If LocalCacheDir is already set and valid, use it
	if source.LocalCacheDir != "" {
		if err := os.MkdirAll(source.LocalCacheDir, 0755); err != nil {
			return fmt.Errorf("failed to create local cache directory %s: %w", source.LocalCacheDir, err)
		}
		return nil
	}

	// Generate a cache directory based on source ID
	cacheDir := filepath.Join(c.GlobalDir, "cache", source.ID)
	if err := os.MkdirAll(cacheDir, 0755); err != nil {
		return fmt.Errorf("failed to create local cache directory %s: %w", cacheDir, err)
	}

	source.LocalCacheDir = cacheDir
	return nil
}

// GetLocalPath returns the local path for a source (either the source path for local, or cache dir for remote)
func (c *Config) GetLocalPath(source StorageSource) string {
	if source.Type == StorageTypeLocal {
		return source.Path
	}
	return source.LocalCacheDir
}

// GetAgentPath returns the agent directory path for a storage source
func (c *Config) GetAgentPathForSource(source StorageSource) string {
	localPath := c.GetLocalPath(source)
	return filepath.Join(localPath, c.AgentDir)
}

// GetDBPathForSource returns the database path for a storage source
func (c *Config) GetDBPathForSource(source StorageSource) string {
	return filepath.Join(c.GetAgentPathForSource(source), "stratafs.db")
}

// RemoveSource removes a storage source by ID
func (c *Config) RemoveSource(id string) error {
	for i, source := range c.Sources {
		if source.ID == id {
			c.Sources = append(c.Sources[:i], c.Sources[i+1:]...)
			return nil
		}
	}
	return fmt.Errorf("source with ID %q not found", id)
}

// UpdateSource updates an existing storage source
func (c *Config) UpdateSource(updated StorageSource) error {
	for i, source := range c.Sources {
		if source.ID == updated.ID {
			c.Sources[i] = updated
			return nil
		}
	}
	return fmt.Errorf("source with ID %q not found", updated.ID)
}

// ValidateSource validates a storage source configuration
func (c *Config) ValidateSource(source StorageSource) error {
	if source.ID == "" {
		return fmt.Errorf("source ID cannot be empty")
	}

	if source.Name == "" {
		return fmt.Errorf("source name cannot be empty")
	}

	if source.Path == "" {
		return fmt.Errorf("source path cannot be empty")
	}

	switch source.Type {
	case StorageTypeLocal:
		// For local sources, validate that the path exists
		if _, err := os.Stat(source.Path); os.IsNotExist(err) {
			return fmt.Errorf("local path %q does not exist", source.Path)
		}
	case StorageTypeS3:
		// Validate S3 configuration
		if source.Credentials["bucket"] == nil || source.Credentials["bucket"] == "" {
			return fmt.Errorf("S3 source must specify bucket in credentials")
		}
	case StorageTypeGCS:
		// Validate GCS configuration
		if source.Credentials["bucket"] == nil || source.Credentials["bucket"] == "" {
			return fmt.Errorf("GCS source must specify bucket in credentials")
		}
	case StorageTypeAzure:
		// Validate Azure configuration
		if source.Credentials["container"] == nil || source.Credentials["container"] == "" {
			return fmt.Errorf("Azure source must specify container in credentials")
		}
	default:
		return fmt.Errorf("unsupported storage type: %s", source.Type)
	}

	return nil
}

// GetModelInfo returns detailed information about a FastEmbed model
func GetModelInfo(model FastEmbedModel) EmbeddingModelInfo {
	switch model {
	case FastEmbedBGEBaseEN:
		return EmbeddingModelInfo{
			Name:        "BGE Base EN",
			Description: "BAAI General Embedding, balanced speed and quality",
			MaxTokens:   512,
			FileSize:    "420MB",
			Speed:       "medium",
			Quality:     "high",
			Language:    "English",
		}
	case FastEmbedBGEBaseENV15:
		return EmbeddingModelInfo{
			Name:        "BGE Base EN v1.5",
			Description: "Improved BAAI General Embedding with better performance",
			MaxTokens:   512,
			FileSize:    "420MB",
			Speed:       "medium",
			Quality:     "high",
			Language:    "English",
		}
	case FastEmbedBGESmallEN:
		return EmbeddingModelInfo{
			Name:        "BGE Small EN",
			Description: "Smaller, faster BAAI General Embedding",
			MaxTokens:   512,
			FileSize:    "130MB",
			Speed:       "fast",
			Quality:     "medium",
			Language:    "English",
		}
	case FastEmbedBGESmallENV15:
		return EmbeddingModelInfo{
			Name:        "BGE Small EN v1.5",
			Description: "Improved smaller BAAI General Embedding",
			MaxTokens:   512,
			FileSize:    "130MB",
			Speed:       "fast",
			Quality:     "medium",
			Language:    "English",
		}
	case FastEmbedAllMiniLML6V2:
		return EmbeddingModelInfo{
			Name:        "All-MiniLM-L6-v2",
			Description: "Sentence-BERT model, very fast inference",
			MaxTokens:   256,
			FileSize:    "90MB",
			Speed:       "fast",
			Quality:     "medium",
			Language:    "Multilingual",
		}
	default:
		return EmbeddingModelInfo{
			Name:        "Unknown Model",
			Description: "Unknown embedding model",
			MaxTokens:   512,
			FileSize:    "Unknown",
			Speed:       "unknown",
			Quality:     "unknown",
			Language:    "Unknown",
		}
	}
}

// GetModelDimension returns the vector dimension for a model
func GetModelDimension(model FastEmbedModel) int {
	switch model {
	case FastEmbedBGEBaseEN, FastEmbedBGEBaseENV15:
		return 768
	case FastEmbedBGESmallEN, FastEmbedBGESmallENV15, FastEmbedAllMiniLML6V2:
		return 384
	default:
		return 384 // Default to smaller dimension
	}
}
