package implementations

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"sync"

	"exercise-04-04/pipeline"
)

// FileWriter implements the Writer interface for file output
type FileWriter struct {
	filePath string
	file     *os.File
	writer   *bufio.Writer
	mu       sync.Mutex
	closed   bool
}

// NewFileWriter creates a new file writer
func NewFileWriter(filePath string) *FileWriter {
	return &FileWriter{
		filePath: filePath,
	}
}

// WriteChunk writes a chunk to the file
func (fw *FileWriter) WriteChunk(ctx context.Context, chunk *pipeline.Chunk) error {
	select {
	case <-ctx.Done():
		return ctx.Err()
	default:
	}

	fw.mu.Lock()
	defer fw.mu.Unlock()

	if fw.closed {
		return fmt.Errorf("writer is closed")
	}

	if err := fw.ensureFileOpen(); err != nil {
		return err
	}

	// Write chunk data
	if _, err := fw.writer.Write(chunk.Data); err != nil {
		return fmt.Errorf("failed to write chunk %d: %w", chunk.ID, err)
	}

	// Release chunk data back to pool
	chunk.Release()

	return nil
}

// ensureFileOpen opens the file if it's not already open
func (fw *FileWriter) ensureFileOpen() error {
	if fw.file != nil {
		return nil
	}

	file, err := os.Create(fw.filePath)
	if err != nil {
		return fmt.Errorf("failed to create file %s: %w", fw.filePath, err)
	}

	fw.file = file
	fw.writer = bufio.NewWriter(file)

	return nil
}

// Flush flushes any buffered data to the file
func (fw *FileWriter) Flush() error {
	fw.mu.Lock()
	defer fw.mu.Unlock()

	if fw.writer != nil {
		return fw.writer.Flush()
	}
	return nil
}

// Close closes the file writer
func (fw *FileWriter) Close() error {
	fw.mu.Lock()
	defer fw.mu.Unlock()

	if fw.closed {
		return nil
	}

	fw.closed = true

	var err error
	if fw.writer != nil {
		err = fw.writer.Flush()
		fw.writer = nil
	}

	if fw.file != nil {
		if closeErr := fw.file.Close(); closeErr != nil && err == nil {
			err = closeErr
		}
		fw.file = nil
	}

	return err
}

// ConsoleWriter implements the Writer interface for console output
type ConsoleWriter struct {
	mu sync.Mutex
}

// NewConsoleWriter creates a new console writer
func NewConsoleWriter() *ConsoleWriter {
	return &ConsoleWriter{}
}

// WriteChunk writes a chunk to the console
func (cw *ConsoleWriter) WriteChunk(ctx context.Context, chunk *pipeline.Chunk) error {
	select {
	case <-ctx.Done():
		return ctx.Err()
	default:
	}

	cw.mu.Lock()
	defer cw.mu.Unlock()

	// Write to stdout
	fmt.Printf("Chunk %d: %s", chunk.ID, string(chunk.Data))

	// Release chunk data back to pool
	chunk.Release()

	return nil
}

// Flush flushes console output (no-op)
func (cw *ConsoleWriter) Flush() error {
	return nil
}

// Close closes the console writer (no-op)
func (cw *ConsoleWriter) Close() error {
	return nil
}

// BufferedWriter implements the Writer interface with in-memory buffering
type BufferedWriter struct {
	buffer [][]byte
	mu     sync.Mutex
	closed bool
}

// NewBufferedWriter creates a new buffered writer
func NewBufferedWriter() *BufferedWriter {
	return &BufferedWriter{
		buffer: make([][]byte, 0),
	}
}

// WriteChunk writes a chunk to the buffer
func (bw *BufferedWriter) WriteChunk(ctx context.Context, chunk *pipeline.Chunk) error {
	select {
	case <-ctx.Done():
		return ctx.Err()
	default:
	}

	bw.mu.Lock()
	defer bw.mu.Unlock()

	if bw.closed {
		return fmt.Errorf("writer is closed")
	}

	// Make a copy of the data since we're storing it
	data := make([]byte, len(chunk.Data))
	copy(data, chunk.Data)
	bw.buffer = append(bw.buffer, data)

	// Release original chunk data back to pool
	chunk.Release()

	return nil
}

// GetBuffer returns a copy of the buffered data
func (bw *BufferedWriter) GetBuffer() [][]byte {
	bw.mu.Lock()
	defer bw.mu.Unlock()

	// Return a copy to avoid race conditions
	result := make([][]byte, len(bw.buffer))
	for i, data := range bw.buffer {
		result[i] = make([]byte, len(data))
		copy(result[i], data)
	}
	return result
}

// Clear clears the buffer
func (bw *BufferedWriter) Clear() {
	bw.mu.Lock()
	defer bw.mu.Unlock()
	bw.buffer = bw.buffer[:0]
}

// Flush flushes the buffer (no-op for in-memory buffer)
func (bw *BufferedWriter) Flush() error {
	return nil
}

// Close closes the buffered writer
func (bw *BufferedWriter) Close() error {
	bw.mu.Lock()
	defer bw.mu.Unlock()
	bw.closed = true
	return nil
}