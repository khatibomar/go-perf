package workers

import (
	"context"
	"fmt"
	"sync"
	"sync/atomic"
	"time"
	. "exercise-03-03/types"
)

// BasicWorker implements a basic worker with a queue
type BasicWorker struct {
	id          string
	queue       chan Work
	ctx         context.Context
	cancel      context.CancelFunc
	running     int32
	processed   int64
	lastActivity time.Time
	mu          sync.RWMutex
}

// NewBasicWorker creates a new basic worker
func NewBasicWorker(id string, queueSize int) *BasicWorker {
	return &BasicWorker{
		id:    id,
		queue: make(chan Work, queueSize),
	}
}

// GetID returns the worker ID
func (w *BasicWorker) GetID() string {
	return w.id
}

// Submit submits work to the worker
func (w *BasicWorker) Submit(work Work) error {
	if atomic.LoadInt32(&w.running) == 0 {
		return ErrWorkerStopped
	}

	select {
	case w.queue <- work:
		w.mu.Lock()
		w.lastActivity = time.Now()
		w.mu.Unlock()
		return nil
	default:
		return ErrQueueFull
	}
}

// SubmitBatch submits multiple works to the worker
func (w *BasicWorker) SubmitBatch(works []Work) error {
	for _, work := range works {
		if err := w.Submit(work); err != nil {
			return err
		}
	}
	return nil
}

// Start starts the worker
func (w *BasicWorker) Start(ctx context.Context) error {
	if !atomic.CompareAndSwapInt32(&w.running, 0, 1) {
		return fmt.Errorf("worker %s is already running", w.id)
	}

	w.ctx, w.cancel = context.WithCancel(ctx)
	w.mu.Lock()
	w.lastActivity = time.Now()
	w.mu.Unlock()

	go w.run()
	return nil
}

// Stop stops the worker
func (w *BasicWorker) Stop() error {
	if !atomic.CompareAndSwapInt32(&w.running, 1, 0) {
		return nil
	}

	if w.cancel != nil {
		w.cancel()
	}
	return nil
}

// run is the main worker loop
func (w *BasicWorker) run() {
	for {
		select {
		case <-w.ctx.Done():
			return

		case work := <-w.queue:
			w.processWork(work)
		}
	}
}

// processWork executes a work item
func (w *BasicWorker) processWork(work Work) {
	start := time.Now()
	defer func() {
		atomic.AddInt64(&w.processed, 1)
		w.mu.Lock()
		w.lastActivity = time.Now()
		w.mu.Unlock()
	}()

	if err := work.Execute(); err != nil {
		// Log error (in a real implementation)
		_ = err
	}

	// Simulate some processing time for demonstration
	_ = time.Since(start)
}

// GetProcessedCount returns the number of processed works
func (w *BasicWorker) GetProcessedCount() int64 {
	return atomic.LoadInt64(&w.processed)
}

// GetQueueDepth returns the current queue depth
func (w *BasicWorker) GetQueueDepth() int {
	return len(w.queue)
}

// GetCapacity returns the worker capacity
func (w *BasicWorker) GetCapacity() int {
	return cap(w.queue)
}

// IsRunning returns whether the worker is running
func (w *BasicWorker) IsRunning() bool {
	return atomic.LoadInt32(&w.running) == 1
}

// GetLastActivity returns the last activity time
func (w *BasicWorker) GetLastActivity() time.Time {
	w.mu.RLock()
	defer w.mu.RUnlock()
	return w.lastActivity
}



// LoadAwareWorkerImpl implements LoadAwareWorker interface
type LoadAwareWorkerImpl struct {
	*BasicWorker
	load         float64
	loadMu       sync.RWMutex
	loadSamples  []float64
	sampleIndex  int
	maxSamples   int
}

// NewLoadAwareWorker creates a new load-aware worker
func NewLoadAwareWorker(id string, queueSize int) *LoadAwareWorkerImpl {
	maxSamples := 10
	return &LoadAwareWorkerImpl{
		BasicWorker: NewBasicWorker(id, queueSize),
		loadSamples: make([]float64, maxSamples),
		maxSamples:  maxSamples,
	}
}

// GetLoad returns the current load (0.0 to 1.0)
func (w *LoadAwareWorkerImpl) GetLoad() float64 {
	w.loadMu.RLock()
	defer w.loadMu.RUnlock()

	// Calculate load based on queue depth and recent processing times
	queueLoad := float64(w.GetQueueDepth()) / float64(w.GetCapacity())

	// Average recent load samples
	var avgSample float64
	validSamples := 0
	for _, sample := range w.loadSamples {
		if sample > 0 {
			avgSample += sample
			validSamples++
		}
	}

	if validSamples > 0 {
		avgSample /= float64(validSamples)
	}

	// Combine queue load and processing load
	return (queueLoad + avgSample) / 2.0
}

// processWork overrides BasicWorker's processWork to track load
func (w *LoadAwareWorkerImpl) processWork(work Work) {
	start := time.Now()

	// Call parent's processWork
	w.BasicWorker.processWork(work)

	// Update load metrics
	processingTime := time.Since(start)
	w.updateLoadSample(processingTime)
}

// updateLoadSample updates the load sample with processing time
func (w *LoadAwareWorkerImpl) updateLoadSample(processingTime time.Duration) {
	w.loadMu.Lock()
	defer w.loadMu.Unlock()

	// Convert processing time to load metric (normalized)
	// Assume 1ms is baseline, anything above increases load
	baselineMs := 1.0
	actualMs := float64(processingTime.Nanoseconds()) / 1e6
	loadSample := actualMs / baselineMs
	if loadSample > 1.0 {
		loadSample = 1.0
	}

	w.loadSamples[w.sampleIndex] = loadSample
	w.sampleIndex = (w.sampleIndex + 1) % w.maxSamples
}

// GetQueueSize returns the current queue size (alias for GetQueueDepth)
func (w *LoadAwareWorkerImpl) GetQueueSize() int {
	return w.GetQueueDepth()
}

// WorkerPool manages a collection of workers
type WorkerPool struct {
	workers []*LoadAwareWorkerImpl
	mu      sync.RWMutex
	ctx     context.Context
	cancel  context.CancelFunc
	running int32
}

// NewWorkerPool creates a new worker pool
func NewWorkerPool(size int, queueSize int) *WorkerPool {
	workers := make([]*LoadAwareWorkerImpl, size)
	for i := 0; i < size; i++ {
		workers[i] = NewLoadAwareWorker(fmt.Sprintf("worker-%d", i), queueSize)
	}

	return &WorkerPool{
		workers: workers,
	}
}

// Start starts all workers in the pool
func (p *WorkerPool) Start(ctx context.Context) error {
	if !atomic.CompareAndSwapInt32(&p.running, 0, 1) {
		return fmt.Errorf("worker pool is already running")
	}

	p.ctx, p.cancel = context.WithCancel(ctx)

	for _, worker := range p.workers {
		if err := worker.Start(p.ctx); err != nil {
			p.Stop()
			return err
		}
	}

	return nil
}

// Stop stops all workers in the pool
func (p *WorkerPool) Stop() error {
	if !atomic.CompareAndSwapInt32(&p.running, 1, 0) {
		return nil
	}

	if p.cancel != nil {
		p.cancel()
	}

	p.mu.RLock()
	defer p.mu.RUnlock()

	for _, worker := range p.workers {
		worker.Stop()
	}

	return nil
}

// GetWorkers returns all workers
func (p *WorkerPool) GetWorkers() []*LoadAwareWorkerImpl {
	p.mu.RLock()
	defer p.mu.RUnlock()

	workers := make([]*LoadAwareWorkerImpl, len(p.workers))
	copy(workers, p.workers)
	return workers
}

// GetWorker returns a specific worker by index
func (p *WorkerPool) GetWorker(index int) *LoadAwareWorkerImpl {
	p.mu.RLock()
	defer p.mu.RUnlock()

	if index >= 0 && index < len(p.workers) {
		return p.workers[index]
	}
	return nil
}

// GetSize returns the pool size
func (p *WorkerPool) GetSize() int {
	p.mu.RLock()
	defer p.mu.RUnlock()
	return len(p.workers)
}

// GetMetrics returns pool metrics
func (p *WorkerPool) GetMetrics() map[string]interface{} {
	p.mu.RLock()
	defer p.mu.RUnlock()

	totalProcessed := int64(0)
	totalQueueDepth := 0
	totalLoad := 0.0
	runningWorkers := 0

	for _, worker := range p.workers {
		totalProcessed += worker.GetProcessedCount()
		totalQueueDepth += worker.GetQueueDepth()
		totalLoad += worker.GetLoad()
		if worker.IsRunning() {
			runningWorkers++
		}
	}

	avgLoad := 0.0
	if len(p.workers) > 0 {
		avgLoad = totalLoad / float64(len(p.workers))
	}

	return map[string]interface{}{
		"total_processed":  totalProcessed,
		"total_queue_depth": totalQueueDepth,
		"average_load":     avgLoad,
		"running_workers":  runningWorkers,
		"total_workers":    len(p.workers),
	}
}