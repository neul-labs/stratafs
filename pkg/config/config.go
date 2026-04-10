package config

import (
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

// Config holds the application configuration
type Config struct {
	Version            string
	Directories        []string
	AgentDir           string
	CentralDir         string // Central directory for global config and queue
	QueueDBPath        string // Path to the central queue database
	WorkerCount        int    // Number of worker goroutines
	ScanInterval       time.Duration // Interval for scanning directories
	FastEmbedModel     FastEmbedModel // FastEmbed model to use
	FastEmbedCacheDir  string        // FastEmbed model cache directory
	EmbeddingDimension int           // Embedding dimension (auto-detected)
}

// NewConfig creates a new configuration with default values
func NewConfig() *Config {
	// Get directories from environment variable or use current directory
	dirs := os.Getenv("AGENTFS_DIRS")
	var directories []string

	if dirs != "" {
		directories = strings.Split(dirs, ",")
	} else {
		wd, _ := os.Getwd()
		directories = []string{wd}
	}

	// Clean up directory paths
	for i, dir := range directories {
		directories[i] = filepath.Clean(dir)
	}

	// Get central directory (defaults to user's home/.agentfs)
	centralDir := os.Getenv("AGENTFS_CENTRAL_DIR")
	if centralDir == "" {
		home, _ := os.UserHomeDir()
		centralDir = filepath.Join(home, ".agentfs")
	}

	// Worker count (defaults to 4)
	workerCount := 4
	if wc := os.Getenv("AGENTFS_WORKERS"); wc != "" {
		if parsed, err := strconv.Atoi(wc); err == nil && parsed > 0 {
			workerCount = parsed
		}
	}

	// Scan interval (defaults to 10 seconds)
	scanInterval := 10 * time.Second
	if si := os.Getenv("AGENTFS_SCAN_INTERVAL"); si != "" {
		if parsed, err := time.ParseDuration(si); err == nil {
			scanInterval = parsed
		}
	}

	// FastEmbed model configuration (defaults to BGE Base EN)
	fastEmbedModel := FastEmbedBGEBaseEN
	if fem := os.Getenv("AGENTFS_MODEL"); fem != "" {
		switch strings.ToLower(fem) {
		case "bge-base-en":
			fastEmbedModel = FastEmbedBGEBaseEN
		case "bge-base-en-v1.5":
			fastEmbedModel = FastEmbedBGEBaseENV15
		case "bge-small-en":
			fastEmbedModel = FastEmbedBGESmallEN
		case "bge-small-en-v1.5":
			fastEmbedModel = FastEmbedBGESmallENV15
		case "all-minilm-l6-v2":
			fastEmbedModel = FastEmbedAllMiniLML6V2
		}
	}

	// FastEmbed cache directory
	fastEmbedCacheDir := filepath.Join(centralDir, "fastembed_cache")
	if fcd := os.Getenv("AGENTFS_FASTEMBED_CACHE"); fcd != "" {
		fastEmbedCacheDir = fcd
	}

	return &Config{
		Version:            "0.1.0",
		Directories:        directories,
		AgentDir:           ".agentfs",
		CentralDir:         centralDir,
		QueueDBPath:        filepath.Join(centralDir, "queue.db"),
		WorkerCount:        workerCount,
		ScanInterval:       scanInterval,
		FastEmbedModel:     fastEmbedModel,
		FastEmbedCacheDir:  fastEmbedCacheDir,
		EmbeddingDimension: 0, // Will be auto-detected by embedder
	}
}

// GetAgentPath returns the full path to the agent directory for a given base directory
func (c *Config) GetAgentPath(baseDir string) string {
	return filepath.Join(baseDir, c.AgentDir)
}

// GetDBPath returns the full path to the database file for a given base directory
func (c *Config) GetDBPath(baseDir string) string {
	return filepath.Join(c.GetAgentPath(baseDir), "agentfs.db")
}

