package utils

import (
	"strings"
)

// ChunkOptions represents options for chunking text
type ChunkOptions struct {
	ChunkSize    int
	OverlapSize  int
	Separator    string
}

// DefaultChunkOptions returns default chunking options
func DefaultChunkOptions() ChunkOptions {
	return ChunkOptions{
		ChunkSize:   1000,
		OverlapSize: 100,
		Separator:   "\n",
	}
}

// ChunkText splits text into overlapping chunks
func ChunkText(text string, opts ChunkOptions) []string {
	if opts.ChunkSize <= 0 {
		opts.ChunkSize = 1000
	}
	
	if opts.OverlapSize < 0 {
		opts.OverlapSize = 0
	}
	
	if opts.Separator == "" {
		opts.Separator = "\n"
	}
	
	// If text is smaller than chunk size, return as is
	if len(text) <= opts.ChunkSize {
		return []string{text}
	}
	
	var chunks []string
	runes := []rune(text)
	
	for i := 0; i < len(runes); i += (opts.ChunkSize - opts.OverlapSize) {
		// Calculate end position
		end := i + opts.ChunkSize
		if end > len(runes) {
			end = len(runes)
		}
		
		// Extract chunk
		chunk := string(runes[i:end])
		chunks = append(chunks, chunk)
		
		// If we've reached the end, break
		if end == len(runes) {
			break
		}
	}
	
	return chunks
}

// ChunkTextBySeparator splits text into chunks by separator with overlap
func ChunkTextBySeparator(text string, opts ChunkOptions) []string {
	if opts.ChunkSize <= 0 {
		opts.ChunkSize = 1000
	}
	
	if opts.OverlapSize < 0 {
		opts.OverlapSize = 0
	}
	
	if opts.Separator == "" {
		opts.Separator = "\n"
	}
	
	// Split text by separator
	parts := strings.Split(text, opts.Separator)
	
	var chunks []string
	var currentChunk strings.Builder
	var currentSize int
	var overlap []string
	
	for _, part := range parts {
		partSize := len(part)
		
		// If adding this part would exceed chunk size
		if currentSize > 0 && currentSize+partSize+len(opts.Separator) > opts.ChunkSize {
			// Save current chunk
			if currentChunk.Len() > 0 {
				chunks = append(chunks, strings.TrimSuffix(currentChunk.String(), opts.Separator))
			}
			
			// Prepare for next chunk with overlap
			currentChunk.Reset()
			currentSize = 0
			
			// Add overlap parts
			for _, overlapPart := range overlap {
				currentChunk.WriteString(overlapPart)
				currentChunk.WriteString(opts.Separator)
				currentSize += len(overlapPart) + len(opts.Separator)
			}
		}
		
		// Add part to current chunk
		currentChunk.WriteString(part)
		currentChunk.WriteString(opts.Separator)
		currentSize += partSize + len(opts.Separator)
		
		// Update overlap
		overlap = append(overlap, part)
		if len(overlap) > 10 { // Limit overlap to last 10 parts
			overlap = overlap[1:]
		}
	}
	
	// Add final chunk
	if currentChunk.Len() > 0 {
		chunk := strings.TrimSuffix(currentChunk.String(), opts.Separator)
		if len(chunk) > 0 {
			chunks = append(chunks, chunk)
		}
	}
	
	return chunks
}