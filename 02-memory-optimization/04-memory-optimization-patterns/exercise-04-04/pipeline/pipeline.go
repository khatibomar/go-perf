package pipeline

import (
	"context"
	"fmt"
	"sync"
	"time"

	"exercise-04-04/memory"
	"exercise-04-04/metrics"
)

// Pipeline represents the main data processing pipeline
type Pipeline struct {
	reader     Reader
	processor  Processor
	writer     Writer
	memManager *memory.Manager
	metrics    *metrics.Collector
	config     *Config
	mu         sync.RWMutex
	running    bool
	cancel     context.CancelFunc
}

// Config holds pipeline configuration
type Config struct {
	ChunkSize       int           `json:"chunk_size"`
	BufferSize      int           `json:"buffer_size"`
	WorkerCount     int           `json:"worker_count"`
	MemoryLimit     int64         `json:"memory_limit"`
	ProcessTimeout  time.Duration `json:"process_timeout"`
	BackpressureMax int           `json:"backpressure_max"`
}

// DefaultConfig returns a default pipeline configuration
func DefaultConfig() *Config {
	return &Config{
		ChunkSize:       8192,
		BufferSize:      1024,
		WorkerCount:     4,
		MemoryLimit:     100 * 1024 * 1024, // 100MB
		ProcessTimeout:  time.Second,
		BackpressureMax: 100,
	}
}

// NewPipeline creates a new data processing pipeline
func NewPipeline(reader Reader, processor Processor, writer Writer, config *Config) *Pipeline {
	if config == nil {
		config = DefaultConfig()
	}
	
	return &Pipeline{
		reader:     reader,
		processor:  processor,
		writer:     writer,
		memManager: memory.NewManager(config.MemoryLimit),
		metrics:    metrics.NewCollector(),
		config:     config,
	}
}

// Start begins pipeline processing
func (p *Pipeline) Start(ctx context.Context) error {
	p.mu.Lock()
	defer p.mu.Unlock()
	
	if p.running {
		return fmt.Errorf("pipeline already running")
	}
	
	ctx, cancel := context.WithCancel(ctx)
	p.cancel = cancel
	p.running = true
	
	// Start memory monitoring
	go p.memManager.Monitor(ctx)
	
	// Start metrics collection
	go p.metrics.Start(ctx)
	
	// Start processing
	go p.process(ctx)
	
	return nil
}

// Stop gracefully stops the pipeline
func (p *Pipeline) Stop() error {
	p.mu.Lock()
	defer p.mu.Unlock()
	
	if !p.running {
		return fmt.Errorf("pipeline not running")
	}
	
	if p.cancel != nil {
		p.cancel()
	}
	
	p.running = false
	return nil
}

// IsRunning returns whether the pipeline is currently running
func (p *Pipeline) IsRunning() bool {
	p.mu.RLock()
	defer p.mu.RUnlock()
	return p.running
}

// GetMetrics returns current pipeline metrics
func (p *Pipeline) GetMetrics() *metrics.Stats {
	return p.metrics.GetStats()
}

// process is the main processing loop
func (p *Pipeline) process(ctx context.Context) {
	defer func() {
		p.mu.Lock()
		p.running = false
		p.mu.Unlock()
	}()
	
	// Create worker pool
	workChan := make(chan *Chunk, p.config.BackpressureMax)
	resultChan := make(chan *Chunk, p.config.BackpressureMax)
	
	// Start workers
	var wg sync.WaitGroup
	for i := 0; i < p.config.WorkerCount; i++ {
		wg.Add(1)
		go p.worker(ctx, &wg, workChan, resultChan)
	}
	
	// Start result writer
	wg.Add(1)
	go p.writeResults(ctx, &wg, resultChan)
	
	// Read and distribute work
	go func() {
		defer close(workChan)
		p.readAndDistribute(ctx, workChan)
	}()
	
	// Wait for completion
	wg.Wait()
	close(resultChan)
}

// readAndDistribute reads data and distributes to workers
func (p *Pipeline) readAndDistribute(ctx context.Context, workChan chan<- *Chunk) {
	chunkChan := p.reader.ReadChunks(ctx, p.config.ChunkSize)
	
	for {
		select {
		case <-ctx.Done():
			return
		case chunk, ok := <-chunkChan:
			if !ok {
				return
			}
			
			// Check memory pressure
			if p.memManager.IsUnderPressure() {
				p.handleMemoryPressure()
			}
			
			select {
			case <-ctx.Done():
				return
			case workChan <- chunk:
				p.metrics.IncrementProcessed()
			}
		}
	}
}

// worker processes chunks
func (p *Pipeline) worker(ctx context.Context, wg *sync.WaitGroup, workChan <-chan *Chunk, resultChan chan<- *Chunk) {
	defer wg.Done()
	
	for {
		select {
		case <-ctx.Done():
			return
		case chunk, ok := <-workChan:
			if !ok {
				return
			}
			
			start := time.Now()
			
			// Process chunk with timeout
			processCtx, cancel := context.WithTimeout(ctx, p.config.ProcessTimeout)
			processedChunk, err := p.processor.Process(processCtx, chunk)
			cancel()
			
			processingTime := time.Since(start)
			p.metrics.RecordLatency(processingTime)
			
			if err != nil {
				p.metrics.IncrementErrors()
				continue
			}
			
			select {
			case <-ctx.Done():
				return
			case resultChan <- processedChunk:
				// Successfully sent result
			}
		}
	}
}

// writeResults writes processed results
func (p *Pipeline) writeResults(ctx context.Context, wg *sync.WaitGroup, resultChan <-chan *Chunk) {
	defer wg.Done()
	
	for {
		select {
		case <-ctx.Done():
			return
		case chunk, ok := <-resultChan:
			if !ok {
				return
			}
			
			if err := p.writer.WriteChunk(ctx, chunk); err != nil {
				p.metrics.IncrementErrors()
				continue
			}
			
			p.metrics.IncrementWritten()
		}
	}
}

// handleMemoryPressure responds to memory pressure
func (p *Pipeline) handleMemoryPressure() {
	// Trigger memory cleanup
	p.memManager.TriggerCleanup()
	
	// Record memory pressure event
	p.metrics.IncrementMemoryPressure()
	
	// Optionally slow down processing
	time.Sleep(10 * time.Millisecond)
}

// Chunk represents a unit of data being processed
type Chunk struct {
	ID       int64
	Data     []byte
	Metadata map[string]interface{}
	pool     *memory.BufferPool
}

// NewChunk creates a new chunk with pooled buffer
func NewChunk(id int64, size int, pool *memory.BufferPool) *Chunk {
	return &Chunk{
		ID:       id,
		Data:     pool.Get(size),
		Metadata: make(map[string]interface{}),
		pool:     pool,
	}
}

// Release returns the chunk's buffer to the pool
func (c *Chunk) Release() {
	if c.pool != nil && c.Data != nil {
		c.pool.Put(c.Data)
		c.Data = nil
	}
}

// Size returns the size of the chunk data
func (c *Chunk) Size() int {
	return len(c.Data)
}

// SetMetadata sets metadata for the chunk
func (c *Chunk) SetMetadata(key string, value interface{}) {
	c.Metadata[key] = value
}

// GetMetadata gets metadata from the chunk
func (c *Chunk) GetMetadata(key string) (interface{}, bool) {
	value, exists := c.Metadata[key]
	return value, exists
}