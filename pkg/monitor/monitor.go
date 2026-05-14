package monitor

import (
	"context"
	"crypto/md5"
	"fmt"
	"io"
	"strings"
	"sync"
	"time"

	"github.com/neul-labs/stratafs/internal/utils"
	"github.com/neul-labs/stratafs/pkg/api"
	"github.com/neul-labs/stratafs/pkg/config"
	"github.com/neul-labs/stratafs/pkg/database"
	"github.com/neul-labs/stratafs/pkg/embeddings"
	"github.com/neul-labs/stratafs/pkg/filesystem"
	"github.com/neul-labs/stratafs/pkg/parsers"
	"github.com/neul-labs/stratafs/pkg/protocol"
	"github.com/neul-labs/stratafs/pkg/queue"
	"github.com/neul-labs/stratafs/pkg/search"

	"github.com/sourcegraph/conc/pool"
	"golang.org/x/exp/slices"
)

// Monitor watches directories for file changes
type Monitor struct {
	config       *config.Config
	databases    map[string]*database.DB
	embedder     *embeddings.Embedder
	filesystem   filesystem.FileSystem
	fileWatcher  *FileWatcher
	jobQueue     *queue.Queue
	searchEngine *search.Engine
	apiServer    *api.Server
	mcpServer    *protocol.ModelContextProtocol
	wg           sync.WaitGroup
	ctx          context.Context
	cancel       context.CancelFunc
	done         chan struct{}
}

// NewMonitor creates a new file system monitor
func NewMonitor(cfg *config.Config) (*Monitor, error) {
	ctx, cancel := context.WithCancel(context.Background())

	m := &Monitor{
		config:     cfg,
		databases:  make(map[string]*database.DB),
		filesystem: filesystem.NewLocalFileSystem(),
		ctx:        ctx,
		cancel:     cancel,
		done:       make(chan struct{}),
	}

	// Initialize embedder
	embedder, err := embeddings.NewEmbedder(cfg)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize embedder: %w", err)
	}
	m.embedder = embedder

	// Initialize databases for each enabled local source
	for _, source := range cfg.GetEnabledSources() {
		if source.Type == config.StorageTypeLocal {
			if err := m.initializeDirectory(source.Path); err != nil {
				return nil, fmt.Errorf("failed to initialize directory %s: %w", source.Path, err)
			}
		}
	}

	// Initialize job queue (using first database path for queue storage)
	if len(m.databases) > 0 {
		var queuePath string
		for dir := range m.databases {
			queuePath = cfg.GetAgentPath(dir) + "/queue.db"
			break
		}
		jobQueue, err := queue.NewQueue(queuePath)
		if err != nil {
			return nil, fmt.Errorf("failed to initialize job queue: %w", err)
		}
		m.jobQueue = jobQueue

		// Initialize file watcher with fsnotify
		fileWatcher, err := NewFileWatcher(cfg, jobQueue)
		if err != nil {
			return nil, fmt.Errorf("failed to initialize file watcher: %w", err)
		}
		m.fileWatcher = fileWatcher

		// Initialize search engine
		searchEngine, err := search.NewEngine(m.databases, embedder)
		if err != nil {
			// Search engine is optional - log warning but continue
			fmt.Printf("Warning: Failed to initialize search engine: %v\n", err)
		} else {
			m.searchEngine = searchEngine
		}

		// Initialize API server
		m.apiServer = api.NewServer(cfg, m.databases, jobQueue, m.searchEngine)

		// Initialize MCP server
		m.mcpServer = protocol.NewModelContextProtocol(m.databases, m.searchEngine)
	}

	return m, nil
}

// initializeDirectory sets up the .stratafs directory and database for a given directory
func (m *Monitor) initializeDirectory(dir string) error {
	// Create .stratafs directory if it doesn't exist
	agentPath := m.config.GetAgentPath(dir)
	if err := m.filesystem.MkdirAll(agentPath, 0755); err != nil {
		return fmt.Errorf("failed to create agent directory: %w", err)
	}
	
	// Initialize database
	dbPath := m.config.GetDBPath(dir)
	db, err := database.NewDB(dbPath)
	if err != nil {
		return fmt.Errorf("failed to initialize database: %w", err)
	}
	
	m.databases[dir] = db
	return nil
}

// Start begins monitoring file system changes
func (m *Monitor) Start() error {
	// Start file watcher with fsnotify for real-time monitoring
	if m.fileWatcher != nil {
		if err := m.fileWatcher.Start(); err != nil {
			return fmt.Errorf("failed to start file watcher: %w", err)
		}
		fmt.Println("Started file watcher with fsnotify")
	}

	// Start compaction service
	m.wg.Add(1)
	go func() {
		defer m.wg.Done()
		m.runCompactionService()
	}()

	// Start API server
	if m.apiServer != nil {
		m.wg.Add(1)
		go func() {
			defer m.wg.Done()
			if err := m.apiServer.Start(); err != nil {
				fmt.Printf("API server error: %v\n", err)
			}
		}()
		fmt.Printf("Started API server on :%d\n", m.config.Server.APIPort)
	}

	// Start Model Context Protocol server
	if m.mcpServer != nil {
		m.wg.Add(1)
		go func() {
			defer m.wg.Done()
			if err := m.mcpServer.Start(); err != nil {
				fmt.Printf("MCP server error: %v\n", err)
			}
		}()
		fmt.Printf("Started MCP server on :%d\n", m.config.Server.MCPPort)
	}

	// Start graceful shutdown goroutine
	go func() {
		defer close(m.done)
		m.wg.Wait()
	}()

	return nil
}

// watchDirectory monitors a single directory for changes
// Deprecated: Use FileWatcher.Start() instead for fsnotify-based watching
func (m *Monitor) watchDirectory(dir string) {
	// Legacy polling-based watching - kept for fallback compatibility
	// Primary watching is now handled by FileWatcher using fsnotify
	ticker := time.NewTicker(10 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-m.ctx.Done():
			return
		case <-ticker.C:
			m.scanDirectory(dir)
		}
	}
}

// scanDirectory scans a directory for file changes
func (m *Monitor) scanDirectory(dir string) {
	db, ok := m.databases[dir]
	if !ok {
		fmt.Printf("No database found for directory: %s\n", dir)
		return
	}
	
	err := m.filesystem.Walk(dir, func(path string, info filesystem.FileInfo, err error) error {
		if err != nil {
			return err
		}
		
		// Skip the agent directory itself
		if info.IsDir() && info.Name() == m.config.AgentDir {
			return fmt.Errorf("skip directory")
		}
		
		// Skip directories
		if info.IsDir() {
			return nil
		}
		
		// Skip unsupported file types
		if !m.isSupportedFileType(path) {
			return nil
		}
		
		// Process files
		m.processFile(db, path, info)
		
		return nil
	})
	
	if err != nil && err.Error() != "skip directory" {
		fmt.Printf("Error scanning directory %s: %v\n", dir, err)
	}
}

// isSupportedFileType checks if a file type is supported for indexing
func (m *Monitor) isSupportedFileType(path string) bool {
	// For now, support common text-based file types
	// In a production implementation, this should be configurable
	supportedExts := []string{
		".txt", ".md", ".markdown", ".rst", ".adoc", ".asciidoc",
		".go", ".py", ".js", ".ts", ".java", ".cpp", ".c", ".h",
		".html", ".htm", ".xml", ".json", ".yaml", ".yml",
		".csv", ".log",
	}
	
	ext := m.filesystem.Ext(path)
	return slices.Contains(supportedExts, ext)
}

// processFile handles a file change
func (m *Monitor) processFile(db *database.DB, path string, info filesystem.FileInfo) {
	// Calculate file checksum
	checksum, err := m.calculateChecksum(path)
	if err != nil {
		fmt.Printf("Error calculating checksum for %s: %v\n", path, err)
		return
	}
	
	// Upsert file record
	file, err := db.UpsertFile(path, checksum, info.Size())
	if err != nil {
		fmt.Printf("Error upserting file record for %s: %v\n", path, err)
		return
	}
	
	// Parse file content using appropriate parser
	content, err := m.parseFileContent(path)
	if err != nil {
		fmt.Printf("Error parsing file %s: %v\n", path, err)
		return
	}
	
	// Skip empty files
	if len(content) == 0 {
		return
	}
	
	// Chunk the content
	chunkOptions := utils.DefaultChunkOptions()
	chunks := utils.ChunkText(content, chunkOptions)
	
	// Process chunks
	for i, chunk := range chunks {
		// Skip empty chunks
		if len(strings.TrimSpace(chunk)) == 0 {
			continue
		}
		
		// Generate embedding
		embedding, err := m.embedder.Embed(chunk)
		if err != nil {
			fmt.Printf("Error generating embedding for chunk %d of %s: %v\n", i, path, err)
			continue
		}
		
		// Upsert chunk
		_, err = db.UpsertChunk(file.ID, chunk, embedding, i*chunkOptions.ChunkSize, len(chunk))
		if err != nil {
			fmt.Printf("Error upserting chunk %d of %s: %v\n", i, path, err)
			continue
		}
	}
	
	fmt.Printf("Processed file: %s (%d chunks)\n", path, len(chunks))
}

// parseFileContent reads and parses the content of a file using the appropriate parser
func (m *Monitor) parseFileContent(path string) (string, error) {
	// Open the file
	file, err := m.filesystem.Open(path)
	if err != nil {
		return "", fmt.Errorf("failed to open file: %w", err)
	}
	defer file.Close()
	
	// Get the appropriate parser for this file type
	parser := parsers.GetParser(path)
	
	// Parse the content
	content, err := parser.Parse(file)
	if err != nil {
		return "", fmt.Errorf("failed to parse file: %w", err)
	}
	
	return content, nil
}

// calculateChecksum calculates the MD5 checksum of a file
func (m *Monitor) calculateChecksum(filePath string) (string, error) {
	file, err := m.filesystem.Open(filePath)
	if err != nil {
		return "", err
	}
	defer file.Close()
	
	hash := md5.New()
	if _, err := io.Copy(hash, file); err != nil {
		return "", err
	}
	
	return fmt.Sprintf("%x", hash.Sum(nil)), nil
}

// runCompactionService periodically removes soft-deleted entries
func (m *Monitor) runCompactionService() {
	ticker := time.NewTicker(1 * time.Hour)
	defer ticker.Stop()
	
	for {
		select {
		case <-m.ctx.Done():
			return
		case <-ticker.C:
			m.performCompaction()
		}
	}
}

// performCompaction removes soft-deleted entries from databases
func (m *Monitor) performCompaction() {
	p := pool.New().WithMaxGoroutines(10)
	
	for dir, db := range m.databases {
		dir := dir
		db := db
		
		p.Go(func() {
			fmt.Printf("Running compaction for directory: %s\n", dir)
			if err := db.Compact(); err != nil {
				fmt.Printf("Error during compaction for %s: %v\n", dir, err)
			}
		})
	}
	
	p.Wait()
}

// Databases returns the map of databases
func (m *Monitor) Databases() map[string]*database.DB {
	return m.databases
}

// Done returns a channel that is closed when the monitor stops
func (m *Monitor) Done() <-chan struct{} {
	return m.done
}

// Stop gracefully stops the monitor
func (m *Monitor) Stop() {
	// Stop file watcher
	if m.fileWatcher != nil {
		m.fileWatcher.Stop()
	}

	// Stop API server
	if m.apiServer != nil {
		m.apiServer.Stop()
	}

	// Stop MCP server
	if m.mcpServer != nil {
		m.mcpServer.Stop()
	}

	// Stop job queue
	if m.jobQueue != nil {
		m.jobQueue.Stop()
	}

	// Cancel context to stop all goroutines
	m.cancel()
}