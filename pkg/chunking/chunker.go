package chunking

import (
	"fmt"
	"io"
)

// ChunkOptions represents configuration options for text chunking
type ChunkOptions struct {
	ChunkSize    int               `json:"chunk_size"`
	OverlapSize  int               `json:"overlap_size"`
	Strategy     string            `json:"strategy"`
	Separator    string            `json:"separator,omitempty"`
	MaxTokens    int               `json:"max_tokens,omitempty"`
	MinChunkSize int               `json:"min_chunk_size,omitempty"`
	Metadata     map[string]string `json:"metadata,omitempty"`
}

// Chunk represents a text chunk with metadata
type Chunk struct {
	Content   string            `json:"content"`
	Offset    int               `json:"offset"`
	Length    int               `json:"length"`
	Index     int               `json:"index"`
	Metadata  map[string]string `json:"metadata,omitempty"`
	Embedding []float32         `json:"embedding,omitempty"`
}

// Chunker interface for text chunking strategies
type Chunker interface {
	// ChunkStream streams chunks from a reader (primary method for large files)
	ChunkStream(reader io.Reader, options ChunkOptions) (<-chan Chunk, <-chan error)

	// Chunk splits text into chunks (convenience method for small text)
	Chunk(text string, options ChunkOptions) ([]Chunk, error)

	// Name returns the name of the chunking strategy
	Name() string

	// Description returns a description of the chunking strategy
	Description() string

	// DefaultOptions returns default options for this chunker
	DefaultOptions() ChunkOptions
}

// ChunkerFactory creates chunkers by strategy name
type ChunkerFactory struct {
	chunkers map[string]Chunker
}

// NewChunkerFactory creates a new chunker factory
func NewChunkerFactory() *ChunkerFactory {
	factory := &ChunkerFactory{
		chunkers: make(map[string]Chunker),
	}

	// Register default chunkers
	factory.Register(&SimpleChunker{})
	factory.Register(&SeparatorChunker{})
	factory.Register(&SentenceChunker{})
	factory.Register(&TokenChunker{})

	return factory
}

// Register registers a chunker with the factory
func (f *ChunkerFactory) Register(chunker Chunker) {
	f.chunkers[chunker.Name()] = chunker
}

// CreateChunker creates a chunker by strategy name
func (f *ChunkerFactory) CreateChunker(strategy string) (Chunker, error) {
	chunker, exists := f.chunkers[strategy]
	if !exists {
		return nil, fmt.Errorf("unknown chunking strategy: %s", strategy)
	}
	return chunker, nil
}

// AvailableStrategies returns list of available chunking strategies
func (f *ChunkerFactory) AvailableStrategies() []string {
	var strategies []string
	for name := range f.chunkers {
		strategies = append(strategies, name)
	}
	return strategies
}

// ChunkingService provides high-level chunking operations
type ChunkingService struct {
	factory     *ChunkerFactory
	defaultOpts ChunkOptions
}

// NewChunkingService creates a new chunking service
func NewChunkingService() *ChunkingService {
	return &ChunkingService{
		factory: NewChunkerFactory(),
		defaultOpts: ChunkOptions{
			ChunkSize:    1000,
			OverlapSize:  100,
			Strategy:     "simple",
			MinChunkSize: 50,
		},
	}
}

// ChunkStream streams chunks from a reader (recommended for large files)
func (s *ChunkingService) ChunkStream(reader io.Reader, options *ChunkOptions) (<-chan Chunk, <-chan error) {
	if options == nil {
		options = &s.defaultOpts
	}

	// Merge with defaults
	opts := s.mergeOptions(*options)

	chunker, err := s.factory.CreateChunker(opts.Strategy)
	if err != nil {
		errCh := make(chan error, 1)
		errCh <- fmt.Errorf("failed to create chunker: %w", err)
		close(errCh)
		return nil, errCh
	}

	return chunker.ChunkStream(reader, opts)
}

// ChunkText chunks text using the specified strategy (for small text)
func (s *ChunkingService) ChunkText(text string, options *ChunkOptions) ([]Chunk, error) {
	if options == nil {
		options = &s.defaultOpts
	}

	// Merge with defaults
	opts := s.mergeOptions(*options)

	chunker, err := s.factory.CreateChunker(opts.Strategy)
	if err != nil {
		return nil, fmt.Errorf("failed to create chunker: %w", err)
	}

	return chunker.Chunk(text, opts)
}

// ChunkTextByFileType chunks text using file-type specific strategy
func (s *ChunkingService) ChunkTextByFileType(text string, fileType string, options *ChunkOptions) ([]Chunk, error) {
	if options == nil {
		options = &s.defaultOpts
	}

	// Override strategy based on file type
	opts := s.mergeOptions(*options)
	opts.Strategy = s.selectStrategyForFileType(fileType)

	return s.ChunkText(text, &opts)
}

// mergeOptions merges provided options with defaults
func (s *ChunkingService) mergeOptions(options ChunkOptions) ChunkOptions {
	opts := s.defaultOpts

	if options.ChunkSize > 0 {
		opts.ChunkSize = options.ChunkSize
	}
	if options.OverlapSize >= 0 {
		opts.OverlapSize = options.OverlapSize
	}
	if options.Strategy != "" {
		opts.Strategy = options.Strategy
	}
	if options.Separator != "" {
		opts.Separator = options.Separator
	}
	if options.MaxTokens > 0 {
		opts.MaxTokens = options.MaxTokens
	}
	if options.MinChunkSize > 0 {
		opts.MinChunkSize = options.MinChunkSize
	}
	if options.Metadata != nil {
		opts.Metadata = options.Metadata
	}

	return opts
}

// selectStrategyForFileType returns optimal chunking strategy for file type
func (s *ChunkingService) selectStrategyForFileType(fileType string) string {
	switch fileType {
	case "markdown", "md":
		return "separator"
	case "code", "go", "py", "js", "ts", "java", "cpp", "c":
		return "separator"
	case "pdf", "docx", "txt":
		return "sentence"
	case "csv", "xlsx", "json":
		return "separator"
	default:
		return "simple"
	}
}

// ChunkStreamByFileType streams chunks using file-type specific strategy
func (s *ChunkingService) ChunkStreamByFileType(reader io.Reader, fileType string, options *ChunkOptions) (<-chan Chunk, <-chan error) {
	if options == nil {
		options = &s.defaultOpts
	}

	// Override strategy based on file type
	opts := s.mergeOptions(*options)
	opts.Strategy = s.selectStrategyForFileType(fileType)

	return s.ChunkStream(reader, &opts)

}