package search

import (
	"strings"
	"testing"
)

// handleEmbedderInitError skips tests gracefully when FastEmbed/ONNX dependencies
// are missing in the current environment, otherwise it fails the test.
func handleEmbedderInitError(t *testing.T, err error) {
	t.Helper()
	if err == nil {
		return
	}

	if shouldSkipEmbedder(err) {
		t.Skipf("Skipping search tests: FastEmbed/ONNX runtime unavailable (%v)", err)
	}

	t.Fatalf("Failed to create test embedder: %v", err)
}

func shouldSkipEmbedder(err error) bool {
	if err == nil {
		return false
	}

	msg := err.Error()
	skipPatterns := []string{
		"onnxruntime",
		"shared object file",
		"shared library",
		"Platform-specific initialization failed",
		"model download failed",
		"403 Forbidden",
	}

	for _, pattern := range skipPatterns {
		if strings.Contains(msg, pattern) {
			return true
		}
	}

	return false
}
