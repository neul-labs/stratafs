package monitor

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/neul-labs/stratafs/pkg/api"
	"github.com/neul-labs/stratafs/pkg/config"
	"github.com/neul-labs/stratafs/pkg/database"
	"github.com/neul-labs/stratafs/pkg/embeddings"
	"github.com/neul-labs/stratafs/pkg/filesystem"
	"github.com/neul-labs/stratafs/pkg/protocol"
	"github.com/neul-labs/stratafs/pkg/queue"
	"github.com/neul-labs/stratafs/pkg/search"

	"github.com/sourcegraph/conc/pool"
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