package implementations

import (
	"context"
	"fmt"
	"strings"
	"time"

	"exercise-04-04/memory"
	"exercise-04-04/pipeline"
)

// TextProcessor implements the Processor interface for text processing
type TextProcessor struct {
	pool      *memory.BufferPool
	transform TransformFunc
}

// TransformFunc defines a function that transforms text data
type TransformFunc func(string) string

// NewTextProcessor creates a new text processor
func NewTextProcessor(pool *memory.BufferPool, transform TransformFunc) *TextProcessor {
	return &TextProcessor{
		pool:      pool,
		transform: transform,
	}
}

// Process processes a chunk of text data
func (tp *TextProcessor) Process(ctx context.Context, chunk *pipeline.Chunk) (*pipeline.Chunk, error) {
	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	default:
	}

	start := time.Now()

	// Convert bytes to string for processing
	input := string(chunk.Data)

	// Apply transformation
	var output string
	if tp.transform != nil {
		output = tp.transform(input)
	} else {
		// Default transformation: uppercase
		output = strings.ToUpper(input)
	}

	// Get new buffer from pool
	outputBytes := []byte(output)
	newBuf := tp.pool.Get(len(outputBytes))
	copy(newBuf, outputBytes)

	// Create new chunk
	newChunk := &pipeline.Chunk{
		ID:       chunk.ID,
		Data:     newBuf,
		Metadata: make(map[string]interface{}),
	}

	// Copy metadata from original chunk
	for k, v := range chunk.Metadata {
		newChunk.Metadata[k] = v
	}

	// Add processing metadata
	newChunk.Metadata["processed_at"] = time.Now()
	newChunk.Metadata["processing_time"] = time.Since(start)
	newChunk.Metadata["original_size"] = len(chunk.Data)
	newChunk.Metadata["processed_size"] = len(newChunk.Data)

	// Release original chunk data back to pool
	chunk.Release()

	return newChunk, nil
}

// Close closes the processor
func (tp *TextProcessor) Close() error {
	return nil
}

// Common transformation functions

// UppercaseTransform converts text to uppercase
func UppercaseTransform(text string) string {
	return strings.ToUpper(text)
}

// LowercaseTransform converts text to lowercase
func LowercaseTransform(text string) string {
	return strings.ToLower(text)
}

// TrimSpaceTransform trims whitespace from each line
func TrimSpaceTransform(text string) string {
	lines := strings.Split(text, "\n")
	for i, line := range lines {
		lines[i] = strings.TrimSpace(line)
	}
	return strings.Join(lines, "\n")
}

// WordCountTransform adds word count to each line
func WordCountTransform(text string) string {
	lines := strings.Split(text, "\n")
	for i, line := range lines {
		if strings.TrimSpace(line) != "" {
			wordCount := len(strings.Fields(line))
			lines[i] = fmt.Sprintf("%s [words: %d]", line, wordCount)
		}
	}
	return strings.Join(lines, "\n")
}

// ReverseTransform reverses each line
func ReverseTransform(text string) string {
	lines := strings.Split(text, "\n")
	for i, line := range lines {
		lines[i] = reverseString(line)
	}
	return strings.Join(lines, "\n")
}

// reverseString reverses a string
func reverseString(s string) string {
	runes := []rune(s)
	for i, j := 0, len(runes)-1; i < j; i, j = i+1, j-1 {
		runes[i], runes[j] = runes[j], runes[i]
	}
	return string(runes)
}

// ChainTransforms chains multiple transformations together
func ChainTransforms(transforms ...TransformFunc) TransformFunc {
	return func(text string) string {
		result := text
		for _, transform := range transforms {
			result = transform(result)
		}
		return result
	}
}

// JSONProcessor processes JSON data
type JSONProcessor struct {
	pool *memory.BufferPool
}

// NewJSONProcessor creates a new JSON processor
func NewJSONProcessor(pool *memory.BufferPool) *JSONProcessor {
	return &JSONProcessor{
		pool: pool,
	}
}

// Process processes JSON data (placeholder implementation)
func (jp *JSONProcessor) Process(ctx context.Context, chunk *pipeline.Chunk) (*pipeline.Chunk, error) {
	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	default:
	}

	// For now, just pass through the data
	// In a real implementation, you would parse and transform JSON
	newBuf := jp.pool.Get(len(chunk.Data))
	copy(newBuf, chunk.Data)

	newChunk := &pipeline.Chunk{
		ID:       chunk.ID,
		Data:     newBuf,
		Metadata: make(map[string]interface{}),
	}

	// Copy metadata
	for k, v := range chunk.Metadata {
		newChunk.Metadata[k] = v
	}

	newChunk.Metadata["processed_at"] = time.Now()
	newChunk.Metadata["processor_type"] = "json"

	// Release original chunk
	chunk.Release()

	return newChunk, nil
}

// Close closes the JSON processor
func (jp *JSONProcessor) Close() error {
	return nil
}