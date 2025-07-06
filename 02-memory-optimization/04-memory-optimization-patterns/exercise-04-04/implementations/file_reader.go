package implementations

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"sync/atomic"

	"exercise-04-04/memory"
	"exercise-04-04/pipeline"
)

// FileReader implements the Reader interface for file input
type FileReader struct {
	filePath string
	file     *os.File
	scanner  *bufio.Scanner
	pool     *memory.BufferPool
	chunkID  int64
	closed   bool
}

// NewFileReader creates a new file reader
func NewFileReader(filePath string, pool *memory.BufferPool) *FileReader {
	return &FileReader{
		filePath: filePath,
		pool:     pool,
	}
}

// ReadChunks reads data chunks from the file
func (fr *FileReader) ReadChunks(ctx context.Context, chunkSize int) <-chan *pipeline.Chunk {
	chunkChan := make(chan *pipeline.Chunk, 10)

	go func() {
		defer close(chunkChan)

		if err := fr.openFile(); err != nil {
			fmt.Printf("Error opening file: %v\n", err)
			return
		}
		defer fr.Close()

		buffer := make([]byte, 0, chunkSize)
		for fr.scanner.Scan() {
			select {
			case <-ctx.Done():
				return
			default:
			}

			line := fr.scanner.Bytes()
			// Add newline back
			lineWithNewline := append(line, '\n')

			// Check if adding this line would exceed chunk size
			if len(buffer)+len(lineWithNewline) > chunkSize && len(buffer) > 0 {
				// Send current buffer as chunk
				chunk := fr.createChunk(buffer)
				select {
				case <-ctx.Done():
					return
				case chunkChan <- chunk:
				}
				// Reset buffer
				buffer = buffer[:0]
			}

			// Add line to buffer
			buffer = append(buffer, lineWithNewline...)
		}

		// Send remaining data as final chunk
		if len(buffer) > 0 {
			chunk := fr.createChunk(buffer)
			select {
			case <-ctx.Done():
			case chunkChan <- chunk:
			}
		}

		if err := fr.scanner.Err(); err != nil {
			fmt.Printf("Error reading file: %v\n", err)
		}
	}()

	return chunkChan
}

// openFile opens the file for reading
func (fr *FileReader) openFile() error {
	if fr.file != nil {
		return nil
	}

	file, err := os.Open(fr.filePath)
	if err != nil {
		return fmt.Errorf("failed to open file %s: %w", fr.filePath, err)
	}

	fr.file = file
	fr.scanner = bufio.NewScanner(file)
	// Set a larger buffer for the scanner to handle big lines
	buf := make([]byte, 0, 64*1024)
	fr.scanner.Buffer(buf, 1024*1024) // 1MB max token size

	return nil
}

// createChunk creates a new chunk from the buffer
func (fr *FileReader) createChunk(data []byte) *pipeline.Chunk {
	chunkID := atomic.AddInt64(&fr.chunkID, 1)
	
	// Get buffer from pool
	buf := fr.pool.Get(len(data))
	copy(buf, data)

	return &pipeline.Chunk{
		ID:       chunkID,
		Data:     buf,
		Metadata: make(map[string]interface{}),
	}
}

// Close closes the file reader
func (fr *FileReader) Close() error {
	if fr.closed {
		return nil
	}

	fr.closed = true
	if fr.file != nil {
		err := fr.file.Close()
		fr.file = nil
		fr.scanner = nil
		return err
	}
	return nil
}

// StringReader implements the Reader interface for string input
type StringReader struct {
	data    string
	pool    *memory.BufferPool
	chunkID int64
	offset  int
}

// NewStringReader creates a new string reader
func NewStringReader(data string, pool *memory.BufferPool) *StringReader {
	return &StringReader{
		data: data,
		pool: pool,
	}
}

// ReadChunks reads data chunks from the string
func (sr *StringReader) ReadChunks(ctx context.Context, chunkSize int) <-chan *pipeline.Chunk {
	chunkChan := make(chan *pipeline.Chunk, 10)

	go func() {
		defer close(chunkChan)

		for sr.offset < len(sr.data) {
			select {
			case <-ctx.Done():
				return
			default:
			}

			end := sr.offset + chunkSize
			if end > len(sr.data) {
				end = len(sr.data)
			}

			chunkData := sr.data[sr.offset:end]
			chunk := sr.createChunk([]byte(chunkData))

			select {
			case <-ctx.Done():
				return
			case chunkChan <- chunk:
			}

			sr.offset = end
		}
	}()

	return chunkChan
}

// createChunk creates a new chunk from the data
func (sr *StringReader) createChunk(data []byte) *pipeline.Chunk {
	chunkID := atomic.AddInt64(&sr.chunkID, 1)
	
	// Get buffer from pool
	buf := sr.pool.Get(len(data))
	copy(buf, data)

	return &pipeline.Chunk{
		ID:       chunkID,
		Data:     buf,
		Metadata: make(map[string]interface{}),
	}
}

// Close closes the string reader
func (sr *StringReader) Close() error {
	return nil
}