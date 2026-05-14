package embeddings

import (
	"fmt"
	"os"

	"github.com/neul-labs/stratafs/pkg/config"
	"github.com/anush008/fastembed-go"
)

// Embedder provides text embedding functionality using FastEmbed
type Embedder struct {
	model     *fastembed.FlagEmbedding
	dimension int
	config    *config.Config
}

// NewEmbedder creates a new embedder instance with the given configuration
func NewEmbedder(cfg *config.Config) (*Embedder, error) {
	// Map our config model to fastembed model
	var fastembedModel fastembed.EmbeddingModel
	var expectedDimension int

	switch cfg.Embedding.Model {
	case config.FastEmbedBGEBaseEN:
		fastembedModel = fastembed.BGEBaseEN
		expectedDimension = 768
	case config.FastEmbedBGEBaseENV15:
		fastembedModel = fastembed.BGEBaseENV15
		expectedDimension = 768
	case config.FastEmbedBGESmallEN:
		fastembedModel = fastembed.BGESmallEN
		expectedDimension = 384
	case config.FastEmbedBGESmallENV15:
		fastembedModel = fastembed.BGESmallENV15
		expectedDimension = 384
	case config.FastEmbedAllMiniLML6V2:
		fastembedModel = fastembed.AllMiniLML6V2
		expectedDimension = 384
	default:
		return nil, fmt.Errorf("unsupported FastEmbed model: %s", cfg.Embedding.Model)
	}

	// Create cache directory if it doesn't exist
	if err := os.MkdirAll(cfg.Embedding.CacheDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create cache directory: %w", err)
	}

	// Initialize FastEmbed model
	options := &fastembed.InitOptions{
		Model:     fastembedModel,
		CacheDir:  cfg.Embedding.CacheDir,
		MaxLength: cfg.Embedding.ModelInfo.MaxTokens,
	}

	model, err := fastembed.NewFlagEmbedding(options)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize FastEmbed model %s: %w", cfg.Embedding.Model, err)
	}

	// Update config with detected dimension
	cfg.Embedding.Dimension = expectedDimension

	fmt.Printf("FastEmbed model %s initialized successfully (dimension: %d)\n", cfg.Embedding.ModelInfo.Name, expectedDimension)

	return &Embedder{
		model:     model,
		dimension: expectedDimension,
		config:    cfg,
	}, nil
}

// Removed backward compatibility function - use NewEmbedder(cfg) directly

// Embed generates embeddings for a single text
func (e *Embedder) Embed(text string) ([]float32, error) {
	if text == "" {
		// Return zero vector for empty text
		return make([]float32, e.dimension), nil
	}

	// Use PassageEmbed for content embedding (better for documents)
	embeddings, err := e.model.PassageEmbed([]string{text}, 1)
	if err != nil {
		return nil, fmt.Errorf("failed to generate embedding: %w", err)
	}

	if len(embeddings) == 0 {
		return nil, fmt.Errorf("no embeddings generated")
	}

	embedding := embeddings[0]
	if len(embedding) != e.dimension {
		return nil, fmt.Errorf("unexpected embedding dimension: got %d, expected %d", len(embedding), e.dimension)
	}

	return embedding, nil
}

// EmbedBatch generates embeddings for multiple texts efficiently
func (e *Embedder) EmbedBatch(texts []string, batchSize int) ([][]float32, error) {
	if len(texts) == 0 {
		return nil, nil
	}

	if batchSize <= 0 {
		batchSize = 25 // Default batch size
	}

	// Use PassageEmbed for batch processing
	embeddings, err := e.model.PassageEmbed(texts, batchSize)
	if err != nil {
		return nil, fmt.Errorf("failed to generate batch embeddings: %w", err)
	}

	if len(embeddings) != len(texts) {
		return nil, fmt.Errorf("embedding count mismatch: got %d, expected %d", len(embeddings), len(texts))
	}

	// Validate dimensions
	for i, embedding := range embeddings {
		if len(embedding) != e.dimension {
			return nil, fmt.Errorf("unexpected embedding dimension for text %d: got %d, expected %d",
				i, len(embedding), e.dimension)
		}
	}

	return embeddings, nil
}

// EmbedQuery generates embeddings specifically for query text (better for search queries)
func (e *Embedder) EmbedQuery(query string) ([]float32, error) {
	if query == "" {
		return make([]float32, e.dimension), nil
	}

	embedding, err := e.model.QueryEmbed(query)
	if err != nil {
		return nil, fmt.Errorf("failed to generate query embedding: %w", err)
	}

	if len(embedding) != e.dimension {
		return nil, fmt.Errorf("unexpected query embedding dimension: got %d, expected %d", len(embedding), e.dimension)
	}

	return embedding, nil
}

// GetDimension returns the embedding dimension
func (e *Embedder) GetDimension() int {
	return e.dimension
}

// GetModelName returns the current model name
func (e *Embedder) GetModelName() string {
	return e.config.Embedding.ModelInfo.Name
}

// Close releases resources used by the embedder
func (e *Embedder) Close() error {
	if e.model != nil {
		e.model.Destroy()
		e.model = nil
	}
	return nil
}

// GetAvailableModels returns a list of available FastEmbed models with their dimensions
func GetAvailableModels() map[config.FastEmbedModel]int {
	return map[config.FastEmbedModel]int{
		config.FastEmbedBGEBaseEN:     768,
		config.FastEmbedBGEBaseENV15:  768,
		config.FastEmbedBGESmallEN:    384,
		config.FastEmbedBGESmallENV15: 384,
		config.FastEmbedAllMiniLML6V2: 384,
	}
}

// ValidateModel checks if a model name is valid and returns its dimension
func ValidateModel(model config.FastEmbedModel) (int, error) {
	models := GetAvailableModels()
	if dimension, exists := models[model]; exists {
		return dimension, nil
	}
	return 0, fmt.Errorf("unsupported model: %s", model)
}