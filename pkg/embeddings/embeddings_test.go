package embeddings

import (
	"testing"

	"agentfs/pkg/config"
)

func TestValidateModel(t *testing.T) {
	tests := []struct {
		model    config.FastEmbedModel
		expected int
		hasError bool
	}{
		{config.FastEmbedBGEBaseEN, 768, false},
		{config.FastEmbedBGEBaseENV15, 768, false},
		{config.FastEmbedBGESmallEN, 384, false},
		{config.FastEmbedBGESmallENV15, 384, false},
		{config.FastEmbedAllMiniLML6V2, 384, false},
		{"invalid-model", 0, true},
	}

	for _, tt := range tests {
		t.Run(string(tt.model), func(t *testing.T) {
			dimension, err := ValidateModel(tt.model)

			if tt.hasError {
				if err == nil {
					t.Error("Expected error for invalid model")
				}
			} else {
				if err != nil {
					t.Errorf("Unexpected error: %v", err)
				}
				if dimension != tt.expected {
					t.Errorf("Expected dimension %d, got %d", tt.expected, dimension)
				}
			}
		})
	}
}

func TestGetAvailableModels(t *testing.T) {
	models := GetAvailableModels()

	expectedModels := map[config.FastEmbedModel]int{
		config.FastEmbedBGEBaseEN:     768,
		config.FastEmbedBGEBaseENV15:  768,
		config.FastEmbedBGESmallEN:    384,
		config.FastEmbedBGESmallENV15: 384,
		config.FastEmbedAllMiniLML6V2: 384,
	}

	if len(models) != len(expectedModels) {
		t.Errorf("Expected %d models, got %d", len(expectedModels), len(models))
	}

	for model, expectedDim := range expectedModels {
		if dim, exists := models[model]; !exists {
			t.Errorf("Model %s not found in available models", model)
		} else if dim != expectedDim {
			t.Errorf("Model %s: expected dimension %d, got %d", model, expectedDim, dim)
		}
	}
}

func TestUnsupportedModel(t *testing.T) {
	tempDir := t.TempDir()

	cfg := &config.Config{
		Embedding: config.EmbeddingConfig{
			Model:     "unsupported-model",
			CacheDir:  tempDir,
			Dimension: 0,
		},
	}

	_, err := NewEmbedder(cfg)
	if err == nil {
		t.Error("Expected error for unsupported model")
	}
}

func TestModelDimensionMapping(t *testing.T) {
	// Test that all available models have valid dimension mappings
	models := GetAvailableModels()

	for model, expectedDim := range models {
		dimension, err := ValidateModel(model)
		if err != nil {
			t.Errorf("Model %s should be valid but got error: %v", model, err)
		}

		if dimension != expectedDim {
			t.Errorf("Model %s: dimension mismatch. Expected %d, got %d", model, expectedDim, dimension)
		}

		// Check that dimensions are reasonable
		if dimension != 384 && dimension != 768 {
			t.Errorf("Model %s has unexpected dimension %d", model, dimension)
		}
	}
}

func TestConfigConstants(t *testing.T) {
	// Test that all model constants are properly defined
	expectedModels := []config.FastEmbedModel{
		config.FastEmbedBGEBaseEN,
		config.FastEmbedBGEBaseENV15,
		config.FastEmbedBGESmallEN,
		config.FastEmbedBGESmallENV15,
		config.FastEmbedAllMiniLML6V2,
	}

	for _, model := range expectedModels {
		if string(model) == "" {
			t.Errorf("Model constant should not be empty")
		}

		// Validate each model
		_, err := ValidateModel(model)
		if err != nil {
			t.Errorf("Model constant %s should be valid: %v", model, err)
		}
	}
}