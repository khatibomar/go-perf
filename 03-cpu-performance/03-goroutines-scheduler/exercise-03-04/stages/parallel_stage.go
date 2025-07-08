package stages

import (
	"context"
	"sync"
	"time"
)

// ParallelStage executes multiple stages in parallel
type ParallelStage struct {
	*BaseStage
	stages      []Stage
	parallelism int
	workChan    chan workItem
	resultChan  chan workResult
	wg          sync.WaitGroup
	workerPool  []*worker
}

type workItem struct {
	id   int64
	data interface{}
}

type workResult struct {
	id     int64
	result interface{}
	err    error
}

type worker struct {
	id     int
	stages []Stage
	ctx    context.Context
}

// NewParallelStage creates a new parallel stage
func NewParallelStage(name string, stages []Stage, parallelism int) *ParallelStage {
	if parallelism <= 0 {
		parallelism = 1
	}
	
	return &ParallelStage{
		BaseStage:   NewBaseStage(name),
		stages:      stages,
		parallelism: parallelism,
		workChan:    make(chan workItem, parallelism*2),
		resultChan:  make(chan workResult, parallelism*2),
		workerPool:  make([]*worker, parallelism),
	}
}

// Start initializes the parallel stage workers
func (s *ParallelStage) Start(ctx context.Context) error {
	s.BaseStage.Start(ctx)
	
	// Start workers
	for i := 0; i < s.parallelism; i++ {
		s.workerPool[i] = &worker{
			id:     i,
			stages: s.stages,
			ctx:    ctx,
		}
		
		s.wg.Add(1)
		go s.runWorker(s.workerPool[i])
	}
	
	return nil
}

// Stop gracefully shuts down the parallel stage
func (s *ParallelStage) Stop() error {
	close(s.workChan)
	s.wg.Wait()
	close(s.resultChan)
	return s.BaseStage.Stop()
}

// Process distributes work to parallel workers
func (s *ParallelStage) Process(data interface{}) (interface{}, error) {
	start := time.Now()
	defer func() {
		duration := time.Since(start)
		s.UpdateMetrics(duration, true)
	}()
	
	// Generate unique work ID
	workID := time.Now().UnixNano()
	
	// Send work to worker
	select {
	case s.workChan <- workItem{id: workID, data: data}:
	case <-s.ctx.Done():
		return nil, s.ctx.Err()
	}
	
	// Wait for result
	select {
	case result := <-s.resultChan:
		if result.id == workID {
			return result.result, result.err
		}
		// Handle out-of-order results (simplified)
		return result.result, result.err
	case <-s.ctx.Done():
		return nil, s.ctx.Err()
	}
}

// runWorker executes work items in a worker goroutine
func (s *ParallelStage) runWorker(w *worker) {
	defer s.wg.Done()
	
	for {
		select {
		case work, ok := <-s.workChan:
			if !ok {
				return
			}
			
			result, err := s.processInWorker(work.data, w)
			
			select {
			case s.resultChan <- workResult{id: work.id, result: result, err: err}:
			case <-w.ctx.Done():
				return
			}
			
		case <-w.ctx.Done():
			return
		}
	}
}

// processInWorker processes data through all stages in the worker
func (s *ParallelStage) processInWorker(data interface{}, w *worker) (interface{}, error) {
	current := data
	
	for _, stage := range w.stages {
		result, err := stage.Process(current)
		if err != nil {
			return nil, err
		}
		current = result
	}
	
	return current, nil
}

// GetWorkerCount returns the number of workers
func (s *ParallelStage) GetWorkerCount() int {
	return s.parallelism
}

// GetQueueLength returns the current work queue length
func (s *ParallelStage) GetQueueLength() int {
	return len(s.workChan)
}

// BatchParallelStage processes data in batches across parallel workers
type BatchParallelStage struct {
	*ParallelStage
	batchSize int
	buffer    []interface{}
	mu        sync.Mutex
}

// NewBatchParallelStage creates a new batch parallel stage
func NewBatchParallelStage(name string, stages []Stage, parallelism, batchSize int) *BatchParallelStage {
	return &BatchParallelStage{
		ParallelStage: NewParallelStage(name, stages, parallelism),
		batchSize:     batchSize,
		buffer:        make([]interface{}, 0, batchSize),
	}
}

// Process batches data before sending to parallel workers
func (s *BatchParallelStage) Process(data interface{}) (interface{}, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	
	s.buffer = append(s.buffer, data)
	
	if len(s.buffer) >= s.batchSize {
		batch := make([]interface{}, len(s.buffer))
		copy(batch, s.buffer)
		s.buffer = s.buffer[:0]
		
		// Process batch in parallel
		return s.ParallelStage.Process(batch)
	}
	
	return nil, nil // No output until batch is full
}

// Flush processes any remaining items in the buffer
func (s *BatchParallelStage) Flush() (interface{}, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	
	if len(s.buffer) == 0 {
		return nil, nil
	}
	
	batch := make([]interface{}, len(s.buffer))
	copy(batch, s.buffer)
	s.buffer = s.buffer[:0]
	
	return s.ParallelStage.Process(batch)
}