package pipeline

import "context"

// Reader interface defines data reading capabilities
type Reader interface {
	ReadChunks(ctx context.Context, chunkSize int) <-chan *Chunk
	Close() error
}

// Processor interface defines data processing capabilities
type Processor interface {
	Process(ctx context.Context, chunk *Chunk) (*Chunk, error)
	Close() error
}

// Writer interface defines data writing capabilities
type Writer interface {
	WriteChunk(ctx context.Context, chunk *Chunk) error
	Flush() error
	Close() error
}