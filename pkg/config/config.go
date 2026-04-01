package config

import (
	"os"
	"path/filepath"
	"strings"
)

// Config holds the application configuration
type Config struct {
	Version     string
	Directories []string
	AgentDir    string
	DBPath      string
	IndexPath   string
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
	
	// Default agent directory name
	agentDir := ".agentfs"
	
	return &Config{
		Version:     "0.1.0",
		Directories: directories,
		AgentDir:    agentDir,
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

// GetIndexPath returns the full path to the index file for a given base directory
func (c *Config) GetIndexPath(baseDir string) string {
	return filepath.Join(c.GetAgentPath(baseDir), "index.usearch")
}