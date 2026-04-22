package chunking

import (
	"bufio"
	"io"
	"regexp"
	"strings"
)

// SimpleChunker implements basic fixed-size chunking with overlap
type SimpleChunker struct{}

func (c *SimpleChunker) Name() string {
	return "simple"
}

func (c *SimpleChunker) Description() string {
	return "Fixed-size chunking with configurable overlap"
}

func (c *SimpleChunker) DefaultOptions() ChunkOptions {
	return ChunkOptions{
		ChunkSize:   1000,
		OverlapSize: 100,
		Strategy:    "simple",
	}
}

// ChunkStream streams chunks from reader with fixed size and overlap
func (c *SimpleChunker) ChunkStream(reader io.Reader, options ChunkOptions) (<-chan Chunk, <-chan error) {
	chunkCh := make(chan Chunk, 10)
	errCh := make(chan error, 1)

	go func() {
		defer close(chunkCh)
		defer close(errCh)

		buffer := make([]byte, 0, options.ChunkSize*2)
		readBuffer := make([]byte, 8192) // 8KB read buffer
		globalOffset := 0
		chunkIndex := 0

		for {
			n, err := reader.Read(readBuffer)
			if n > 0 {
				buffer = append(buffer, readBuffer[:n]...)
			}

			// Process complete chunks
			for len(buffer) >= options.ChunkSize {
				chunkSize := options.ChunkSize
				if chunkSize > len(buffer) {
					chunkSize = len(buffer)
				}

				chunk := Chunk{
					Content: string(buffer[:chunkSize]),
					Offset:  globalOffset,
					Length:  chunkSize,
					Index:   chunkIndex,
				}

				select {
				case chunkCh <- chunk:
				case <-errCh:
					return
				}

				// Move buffer with overlap
				overlapSize := options.OverlapSize
				if overlapSize > chunkSize {
					overlapSize = chunkSize
				}

				moveSize := chunkSize - overlapSize
				copy(buffer, buffer[moveSize:])
				buffer = buffer[:len(buffer)-moveSize]
				globalOffset += moveSize
				chunkIndex++
			}

			if err == io.EOF {
				// Process remaining buffer
				if len(buffer) > 0 {
					chunk := Chunk{
						Content: string(buffer),
						Offset:  globalOffset,
						Length:  len(buffer),
						Index:   chunkIndex,
					}
					chunkCh <- chunk
				}
				break
			} else if err != nil {
				errCh <- err
				return
			}
		}
	}()

	return chunkCh, errCh
}

// Chunk implements non-streaming version for small text
func (c *SimpleChunker) Chunk(text string, options ChunkOptions) ([]Chunk, error) {
	reader := strings.NewReader(text)
	chunkCh, errCh := c.ChunkStream(reader, options)

	var chunks []Chunk
	for chunk := range chunkCh {
		chunks = append(chunks, chunk)
	}

	// Check for errors
	select {
	case err := <-errCh:
		if err != nil {
			return nil, err
		}
	default:
	}

	return chunks, nil
}

// SeparatorChunker implements chunking by separator (lines, paragraphs, etc.)
type SeparatorChunker struct{}

func (c *SeparatorChunker) Name() string {
	return "separator"
}

func (c *SeparatorChunker) Description() string {
	return "Chunking by separator with size limits and overlap"
}

func (c *SeparatorChunker) DefaultOptions() ChunkOptions {
	return ChunkOptions{
		ChunkSize:   1000,
		OverlapSize: 100,
		Strategy:    "separator",
		Separator:   "\n",
	}
}

func (c *SeparatorChunker) ChunkStream(reader io.Reader, options ChunkOptions) (<-chan Chunk, <-chan error) {
	chunkCh := make(chan Chunk, 10)
	errCh := make(chan error, 1)

	go func() {
		defer close(chunkCh)
		defer close(errCh)

		scanner := bufio.NewScanner(reader)

		// Set custom split function based on separator
		if options.Separator == "\n" {
			scanner.Split(bufio.ScanLines)
		} else {
			scanner.Split(c.createSeparatorSplitter(options.Separator))
		}

		var currentChunk strings.Builder
		var overlapLines []string
		globalOffset := 0
		chunkIndex := 0

		for scanner.Scan() {
			line := scanner.Text()
			lineWithSep := line + options.Separator

			// Check if adding this line would exceed chunk size
			if currentChunk.Len() > 0 && currentChunk.Len()+len(lineWithSep) > options.ChunkSize {
				// Emit current chunk
				chunk := Chunk{
					Content: strings.TrimSuffix(currentChunk.String(), options.Separator),
					Offset:  globalOffset,
					Length:  currentChunk.Len(),
					Index:   chunkIndex,
				}

				select {
				case chunkCh <- chunk:
				case <-errCh:
					return
				}

				// Start new chunk with overlap
				currentChunk.Reset()
				for _, overlapLine := range overlapLines {
					currentChunk.WriteString(overlapLine)
					currentChunk.WriteString(options.Separator)
				}

				globalOffset += chunk.Length - len(strings.Join(overlapLines, options.Separator))
				chunkIndex++
			}

			// Add line to current chunk
			currentChunk.WriteString(lineWithSep)

			// Maintain overlap buffer
			overlapLines = append(overlapLines, line)
			if len(overlapLines) > 10 { // Limit overlap buffer
				overlapLines = overlapLines[1:]
			}
		}

		// Emit final chunk
		if currentChunk.Len() > 0 {
			chunk := Chunk{
				Content: strings.TrimSuffix(currentChunk.String(), options.Separator),
				Offset:  globalOffset,
				Length:  currentChunk.Len(),
				Index:   chunkIndex,
			}
			chunkCh <- chunk
		}

		if err := scanner.Err(); err != nil {
			errCh <- err
		}
	}()

	return chunkCh, errCh
}

func (c *SeparatorChunker) createSeparatorSplitter(separator string) bufio.SplitFunc {
	return func(data []byte, atEOF bool) (advance int, token []byte, err error) {
		if atEOF && len(data) == 0 {
			return 0, nil, nil
		}

		// Find separator
		sepBytes := []byte(separator)
		if i := strings.Index(string(data), separator); i >= 0 {
			return i + len(sepBytes), data[0:i], nil
		}

		// If at EOF, return remaining data
		if atEOF {
			return len(data), data, nil
		}

		// Request more data
		return 0, nil, nil
	}
}

func (c *SeparatorChunker) Chunk(text string, options ChunkOptions) ([]Chunk, error) {
	reader := strings.NewReader(text)
	chunkCh, errCh := c.ChunkStream(reader, options)

	var chunks []Chunk
	for chunk := range chunkCh {
		chunks = append(chunks, chunk)
	}

	select {
	case err := <-errCh:
		if err != nil {
			return nil, err
		}
	default:
	}

	return chunks, nil
}

// SentenceChunker implements sentence-aware chunking
type SentenceChunker struct {
	sentenceRegex *regexp.Regexp
}

func (c *SentenceChunker) Name() string {
	return "sentence"
}

func (c *SentenceChunker) Description() string {
	return "Sentence-aware chunking with natural boundaries"
}

func (c *SentenceChunker) DefaultOptions() ChunkOptions {
	return ChunkOptions{
		ChunkSize:   1000,
		OverlapSize: 100,
		Strategy:    "sentence",
	}
}

func (c *SentenceChunker) ChunkStream(reader io.Reader, options ChunkOptions) (<-chan Chunk, <-chan error) {
	chunkCh := make(chan Chunk, 10)
	errCh := make(chan error, 1)

	// Initialize regex if needed
	if c.sentenceRegex == nil {
		c.sentenceRegex = regexp.MustCompile(`[.!?]+\s+`)
	}

	go func() {
		defer close(chunkCh)
		defer close(errCh)

		// Read all content first (could be optimized for very large files)
		content, err := io.ReadAll(reader)
		if err != nil {
			errCh <- err
			return
		}

		text := string(content)
		sentences := c.sentenceRegex.Split(text, -1)

		var currentChunk strings.Builder
		var overlapSentences []string
		globalOffset := 0
		chunkIndex := 0

		for _, sentence := range sentences {
			sentence = strings.TrimSpace(sentence)
			if sentence == "" {
				continue
			}

			// Check if adding this sentence would exceed chunk size
			if currentChunk.Len() > 0 && currentChunk.Len()+len(sentence)+1 > options.ChunkSize {
				// Emit current chunk
				chunk := Chunk{
					Content: strings.TrimSpace(currentChunk.String()),
					Offset:  globalOffset,
					Length:  currentChunk.Len(),
					Index:   chunkIndex,
				}

				select {
				case chunkCh <- chunk:
				case <-errCh:
					return
				}

				// Start new chunk with overlap
				currentChunk.Reset()
				for _, overlapSentence := range overlapSentences {
					if currentChunk.Len() > 0 {
						currentChunk.WriteString(" ")
					}
					currentChunk.WriteString(overlapSentence)
				}

				if currentChunk.Len() > 0 {
					globalOffset += chunk.Length - currentChunk.Len()
				} else {
					globalOffset += chunk.Length
				}
				chunkIndex++
			}

			// Add sentence to current chunk
			if currentChunk.Len() > 0 {
				currentChunk.WriteString(" ")
			}
			currentChunk.WriteString(sentence)

			// Maintain overlap buffer
			overlapSentences = append(overlapSentences, sentence)
			if len(overlapSentences) > 3 { // Keep last 3 sentences for overlap
				overlapSentences = overlapSentences[1:]
			}
		}

		// Emit final chunk
		if currentChunk.Len() > 0 {
			chunk := Chunk{
				Content: strings.TrimSpace(currentChunk.String()),
				Offset:  globalOffset,
				Length:  currentChunk.Len(),
				Index:   chunkIndex,
			}
			chunkCh <- chunk
		}
	}()

	return chunkCh, errCh
}

func (c *SentenceChunker) Chunk(text string, options ChunkOptions) ([]Chunk, error) {
	reader := strings.NewReader(text)
	chunkCh, errCh := c.ChunkStream(reader, options)

	var chunks []Chunk
	for chunk := range chunkCh {
		chunks = append(chunks, chunk)
	}

	select {
	case err := <-errCh:
		if err != nil {
			return nil, err
		}
	default:
	}

	return chunks, nil
}

// TokenChunker implements token-aware chunking (placeholder for future implementation)
type TokenChunker struct{}

func (c *TokenChunker) Name() string {
	return "token"
}

func (c *TokenChunker) Description() string {
	return "Token-aware chunking respecting token boundaries"
}

func (c *TokenChunker) DefaultOptions() ChunkOptions {
	return ChunkOptions{
		ChunkSize:   1000,
		OverlapSize: 100,
		Strategy:    "token",
		MaxTokens:   256, // Default token limit
	}
}

func (c *TokenChunker) ChunkStream(reader io.Reader, options ChunkOptions) (<-chan Chunk, <-chan error) {
	// For now, fallback to simple chunking
	// TODO: Implement proper tokenization
	simple := &SimpleChunker{}
	return simple.ChunkStream(reader, options)
}

func (c *TokenChunker) Chunk(text string, options ChunkOptions) ([]Chunk, error) {
	// For now, fallback to simple chunking
	// TODO: Implement proper tokenization
	simple := &SimpleChunker{}
	return simple.Chunk(text, options)
}