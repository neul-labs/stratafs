package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"time"
)

// App struct
type App struct {
	ctx       context.Context
	apiURL    string
	configDir string
}

// NewApp creates a new App application struct
func NewApp() *App {
	homeDir, _ := os.UserHomeDir()
	return &App{
		apiURL:    "http://localhost:8080",
		configDir: filepath.Join(homeDir, ".agentfs"),
	}
}

// startup is called when the app starts
func (a *App) startup(ctx context.Context) {
	a.ctx = ctx
}

// Types for API responses

type HealthResponse struct {
	Status  string `json:"status"`
	Version string `json:"version"`
}

type QueueStats struct {
	Pending    int `json:"pending"`
	Processing int `json:"processing"`
	Completed  int `json:"completed"`
	Failed     int `json:"failed"`
	Total      int `json:"total"`
}

type QueueStatsResponse struct {
	QueueStats QueueStats `json:"queue_stats"`
	Timestamp  string     `json:"timestamp"`
}

type SearchResult struct {
	ID           int64             `json:"id"`
	FileID       int64             `json:"file_id"`
	FilePath     string            `json:"file_path"`
	Content      string            `json:"content"`
	Snippet      string            `json:"snippet"`
	Score        float64           `json:"score"`
	Metadata     map[string]interface{} `json:"metadata"`
}

type SearchResponse struct {
	Results   []SearchResult `json:"results"`
	Total     int            `json:"total"`
	Query     string         `json:"query"`
	Mode      string         `json:"mode"`
	TimeTaken string         `json:"time_taken"`
	Limit     int            `json:"limit"`
	Offset    int            `json:"offset"`
	HasMore   bool           `json:"has_more"`
}

type Source struct {
	Name     string `json:"name"`
	Type     string `json:"type"`
	Path     string `json:"path"`
	Enabled  bool   `json:"enabled"`
}

type Config struct {
	Version   string   `json:"version"`
	Sources   []Source `json:"sources"`
	APIPort   int      `json:"api_port"`
	MCPPort   int      `json:"mcp_port"`
}

type AppStatus struct {
	Running     bool   `json:"running"`
	APIHealthy  bool   `json:"api_healthy"`
	Version     string `json:"version"`
	PID         int    `json:"pid"`
	ConfigDir   string `json:"config_dir"`
}

// GetStatus returns the current status of AgentFS
func (a *App) GetStatus() AppStatus {
	status := AppStatus{
		ConfigDir: a.configDir,
	}

	// Check if process is running
	status.Running = a.isProcessRunning()

	// Check API health
	resp, err := http.Get(a.apiURL + "/health")
	if err == nil {
		defer resp.Body.Close()
		if resp.StatusCode == 200 {
			var health HealthResponse
			if json.NewDecoder(resp.Body).Decode(&health) == nil {
				status.APIHealthy = true
				status.Version = health.Version
			}
		}
	}

	return status
}

// isProcessRunning checks if agentfs process is running
func (a *App) isProcessRunning() bool {
	var cmd *exec.Cmd
	if runtime.GOOS == "windows" {
		cmd = exec.Command("tasklist", "/FI", "IMAGENAME eq agentfs.exe")
	} else {
		cmd = exec.Command("pgrep", "-f", "agentfs")
	}
	output, err := cmd.Output()
	if err != nil {
		return false
	}
	if runtime.GOOS == "windows" {
		return strings.Contains(string(output), "agentfs.exe")
	}
	return len(strings.TrimSpace(string(output))) > 0
}

// StartAgentFS starts the AgentFS daemon
func (a *App) StartAgentFS() error {
	if a.isProcessRunning() {
		return fmt.Errorf("AgentFS is already running")
	}

	var cmd *exec.Cmd
	if runtime.GOOS == "windows" {
		cmd = exec.Command("agentfs", "--config-dir", a.configDir)
		cmd.SysProcAttr = nil // No window
	} else {
		cmd = exec.Command("agentfs", "--config-dir", a.configDir)
	}

	cmd.Stdout = nil
	cmd.Stderr = nil

	if err := cmd.Start(); err != nil {
		return fmt.Errorf("failed to start AgentFS: %w", err)
	}

	// Wait for API to become available
	for i := 0; i < 10; i++ {
		time.Sleep(500 * time.Millisecond)
		if resp, err := http.Get(a.apiURL + "/health"); err == nil {
			resp.Body.Close()
			if resp.StatusCode == 200 {
				return nil
			}
		}
	}

	return nil
}

// StopAgentFS stops the AgentFS daemon
func (a *App) StopAgentFS() error {
	if !a.isProcessRunning() {
		return fmt.Errorf("AgentFS is not running")
	}

	var cmd *exec.Cmd
	if runtime.GOOS == "windows" {
		cmd = exec.Command("taskkill", "/F", "/IM", "agentfs.exe")
	} else {
		cmd = exec.Command("pkill", "-f", "agentfs")
	}

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to stop AgentFS: %w", err)
	}

	return nil
}

// RestartAgentFS restarts the AgentFS daemon
func (a *App) RestartAgentFS() error {
	if a.isProcessRunning() {
		if err := a.StopAgentFS(); err != nil {
			return err
		}
		time.Sleep(2 * time.Second)
	}
	return a.StartAgentFS()
}

// GetQueueStats returns job queue statistics
func (a *App) GetQueueStats() (*QueueStatsResponse, error) {
	resp, err := http.Get(a.apiURL + "/queue/stats")
	if err != nil {
		return nil, fmt.Errorf("failed to connect to API: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("API error: %s", string(body))
	}

	var stats QueueStatsResponse
	if err := json.NewDecoder(resp.Body).Decode(&stats); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return &stats, nil
}

// Search performs a search query
func (a *App) Search(query string, mode string, limit int) (*SearchResponse, error) {
	if mode == "" {
		mode = "hybrid"
	}
	if limit == 0 {
		limit = 10
	}

	url := fmt.Sprintf("%s/search?q=%s&mode=%s&limit=%d&include_content=true",
		a.apiURL, query, mode, limit)

	resp, err := http.Get(url)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to API: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("search failed: %s", string(body))
	}

	var results SearchResponse
	if err := json.NewDecoder(resp.Body).Decode(&results); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return &results, nil
}

// GetConfig returns the current configuration
func (a *App) GetConfig() (*Config, error) {
	configPath := filepath.Join(a.configDir, "config.json")

	data, err := os.ReadFile(configPath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, fmt.Errorf("config not found - please initialize AgentFS first")
		}
		return nil, fmt.Errorf("failed to read config: %w", err)
	}

	var config Config
	if err := json.Unmarshal(data, &config); err != nil {
		return nil, fmt.Errorf("failed to parse config: %w", err)
	}

	return &config, nil
}

// InitConfig initializes a new configuration
func (a *App) InitConfig() error {
	cmd := exec.Command("agentfs", "config", "init", "--config-dir", a.configDir)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("failed to initialize config: %s", string(output))
	}
	return nil
}

// AddSource adds a new storage source
func (a *App) AddSource(name, path string) error {
	// Read current config
	configPath := filepath.Join(a.configDir, "config.json")
	data, err := os.ReadFile(configPath)
	if err != nil {
		return fmt.Errorf("failed to read config: %w", err)
	}

	var config map[string]interface{}
	if err := json.Unmarshal(data, &config); err != nil {
		return fmt.Errorf("failed to parse config: %w", err)
	}

	// Add new source
	sources, ok := config["sources"].([]interface{})
	if !ok {
		sources = []interface{}{}
	}

	newSource := map[string]interface{}{
		"name":    name,
		"type":    "local",
		"path":    path,
		"enabled": true,
	}
	sources = append(sources, newSource)
	config["sources"] = sources

	// Write back
	updatedData, err := json.MarshalIndent(config, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to encode config: %w", err)
	}

	if err := os.WriteFile(configPath, updatedData, 0644); err != nil {
		return fmt.Errorf("failed to write config: %w", err)
	}

	return nil
}

// RemoveSource removes a storage source by name
func (a *App) RemoveSource(name string) error {
	configPath := filepath.Join(a.configDir, "config.json")
	data, err := os.ReadFile(configPath)
	if err != nil {
		return fmt.Errorf("failed to read config: %w", err)
	}

	var config map[string]interface{}
	if err := json.Unmarshal(data, &config); err != nil {
		return fmt.Errorf("failed to parse config: %w", err)
	}

	sources, ok := config["sources"].([]interface{})
	if !ok {
		return fmt.Errorf("no sources found")
	}

	// Filter out the source
	var newSources []interface{}
	for _, s := range sources {
		source, ok := s.(map[string]interface{})
		if !ok {
			continue
		}
		if source["name"] != name {
			newSources = append(newSources, source)
		}
	}

	config["sources"] = newSources

	updatedData, err := json.MarshalIndent(config, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to encode config: %w", err)
	}

	if err := os.WriteFile(configPath, updatedData, 0644); err != nil {
		return fmt.Errorf("failed to write config: %w", err)
	}

	return nil
}

// ExportSource exports a source's metadata view
func (a *App) ExportSource(sourceName, outputPath string) error {
	cmd := exec.Command("agentfs", "fs", "export",
		"--source", sourceName,
		"--output", outputPath,
		"--config-dir", a.configDir)

	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("export failed: %s", string(output))
	}
	return nil
}

// OpenConfigDir opens the config directory in file manager
func (a *App) OpenConfigDir() error {
	var cmd *exec.Cmd
	switch runtime.GOOS {
	case "windows":
		cmd = exec.Command("explorer", a.configDir)
	case "darwin":
		cmd = exec.Command("open", a.configDir)
	default:
		cmd = exec.Command("xdg-open", a.configDir)
	}
	return cmd.Start()
}

// SelectDirectory opens a directory picker (handled by frontend via Wails runtime)
func (a *App) GetConfigDir() string {
	return a.configDir
}

// SetAPIURL allows changing the API URL
func (a *App) SetAPIURL(url string) {
	a.apiURL = url
}
