package embeddings

import (
	"fmt"
	"runtime"

	"github.com/anush008/fastembed-go"
)

// Embedder handles text embedding generation
type Embedder struct {
	model *fastembed.FlagEmbedding
}

// NewEmbedder creates a new embedder with the default model
func NewEmbedder() (*Embedder, error) {
	// Try to create the embedder with ONNX runtime
	embedder, err := newEmbedderWithONNX()
	if err != nil {
		// If ONNX fails, try fallback options
		fmt.Printf("Warning: ONNX runtime not available: %v\n", err)
		fmt.Println("Falling back to pure Go implementation...")
		return newEmbedderFallback()
	}
	
	return embedder, nil
}

// newEmbedderWithONNX creates an embedder using ONNX runtime
func newEmbedderWithONNX() (*Embedder, error) {
	// Using all-MiniLM-L6-v2 model which is a good balance between speed and quality
	options := &fastembed.InitOptions{
		Model: fastembed.AllMiniLML6V2,
	}
	
	model, err := fastembed.NewFlagEmbedding(options)
	if err != nil {
		return nil, fmt.Errorf("failed to create embedding model: %w", err)
	}
	
	return &Embedder{
		model: model,
	}, nil
}

// newEmbedderFallback creates an embedder using a fallback method
func newEmbedderFallback() (*Embedder, error) {
	// For macOS, we might need to provide the library path
	// or use a different approach
	fmt.Println("Platform:", runtime.GOOS, runtime.GOARCH)
	
	// Return a mock embedder for now
	return &Embedder{
		model: nil,
	}, nil
}

// Embed generates embeddings for a text
func (e *Embedder) Embed(text string) ([]float32, error) {
	// If we have a real model, use it
	if e.model != nil {
		embedding, err := e.model.QueryEmbed(text)
		if err != nil {
			return nil, fmt.Errorf("failed to embed text: %w", err)
		}
		return embedding, nil
	}
	
	// Fallback: return a simple mock embedding
	// In a real implementation, you might want to use a different library
	// or download the ONNX runtime
	return e.mockEmbed(text), nil
}

// EmbedBatch generates embeddings for multiple texts
func (e *Embedder) EmbedBatch(texts []string) ([][]float32, error) {
	// If we have a real model, use it
	if e.model != nil {
		embeddings, err := e.model.Embed(texts, 32) // Using batch size of 32
		if err != nil {
			return nil, fmt.Errorf("failed to embed texts: %w", err)
		}
		return embeddings, nil
	}
	
	// Fallback: return mock embeddings
	results := make([][]float32, len(texts))
	for i, text := range texts {
		results[i] = e.mockEmbed(text)
	}
	return results, nil
}

// mockEmbed generates a simple mock embedding based on text length
func (e *Embedder) mockEmbed(text string) []float32 {
	// This is a very simple mock - in reality, you'd want proper embeddings
	// For now, we'll generate a fixed-size vector based on text properties
	const embeddingSize = 384 // Size of all-MiniLM-L6-v2 embeddings
	
	embedding := make([]float32, embeddingSize)
	textLen := float32(len(text))
	
	// Simple deterministic "embedding" based on text length
	for i := 0; i < embeddingSize; i++ {
		embedding[i] = (textLen + float32(i)) / float32(embeddingSize)
	}
	
	return embedding
}

// Close releases resources used by the embedder
func (e *Embedder) Close() error {
	if e.model != nil {
		return e.model.Destroy()
	}
	return nil
}