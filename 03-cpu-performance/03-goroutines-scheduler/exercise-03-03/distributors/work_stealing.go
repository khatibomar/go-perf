package distributors

import (
	"context"
	"fmt"
	"math/rand"
	"runtime"
	"sync"
	"sync/atomic"
	"time"
	. "exercise-03-03/types"
)

// WorkStealingCoordinator implements work-stealing distribution
type WorkStealingCoordinator struct {
	workers     []*StealingWorker
	globalQueue chan Work
	metrics     *CoordinatorMetrics
	mu          sync.RWMutex
	ctx         context.Context
	cancel      context.CancelFunc
	running     int32
}

// StealingWorker represents a worker in the work-stealing system
type StealingWorker struct {
	id          int
	localQueue  chan Work
	coordinator *WorkStealingCoordinator
	load        int64
	processed   int64
	running     int32
	ctx         context.Context
	cancel      context.CancelFunc
}

// NewWorkStealingCoordinator creates a new work-stealing coordinator
func NewWorkStealingCoordinator(numWorkers int, queueSize int) *WorkStealingCoordinator {
	ctx, cancel := context.WithCancel(context.Background())

	coordinator := &WorkStealingCoordinator{
		workers:     make([]*StealingWorker, numWorkers),
		globalQueue: make(chan Work, queueSize*numWorkers),
		metrics:     &CoordinatorMetrics{},
		ctx:         ctx,
		cancel:      cancel,
	}

	// Create workers
	for i := 0; i < numWorkers; i++ {
		workerCtx, workerCancel := context.WithCancel(ctx)
		worker := &StealingWorker{
			id:          i,
			localQueue:  make(chan Work, queueSize),
			coordinator: coordinator,
			ctx:         workerCtx,
			cancel:      workerCancel,
		}
		coordinator.workers[i] = worker
	}

	return coordinator
}

// Start starts all workers
func (c *WorkStealingCoordinator) Start() error {
	if !atomic.CompareAndSwapInt32(&c.running, 0, 1) {
		return ErrDistributorStopped
	}

	for _, worker := range c.workers {
		go worker.run()
	}

	return nil
}

// Stop stops all workers
func (c *WorkStealingCoordinator) Stop() error {
	if !atomic.CompareAndSwapInt32(&c.running, 1, 0) {
		return nil
	}

	c.cancel()
	return nil
}

// Distribute submits work to the coordinator
func (c *WorkStealingCoordinator) Distribute(work Work) error {
	if atomic.LoadInt32(&c.running) == 0 {
		return ErrDistributorStopped
	}

	// Try to submit to a random worker's local queue first
	workerIndex := rand.Intn(len(c.workers))
	worker := c.workers[workerIndex]

	select {
	case worker.localQueue <- work:
		atomic.AddInt64(&worker.load, 1)
		return nil
	default:
		// Local queue full, try global queue
		select {
		case c.globalQueue <- work:
			return nil
		default:
			return ErrQueueFull
		}
	}
}

// DistributeBatch distributes multiple works
func (c *WorkStealingCoordinator) DistributeBatch(works []Work) error {
	for _, work := range works {
		if err := c.Distribute(work); err != nil {
			return err
		}
	}
	return nil
}

// AddWorker adds a new worker (not supported in work-stealing)
func (c *WorkStealingCoordinator) AddWorker(worker Worker) error {
	return ErrInvalidConfiguration
}

// RemoveWorker removes a worker (not supported in work-stealing)
func (c *WorkStealingCoordinator) RemoveWorker(workerID string) error {
	return ErrInvalidConfiguration
}

// GetMetrics returns coordinator metrics
func (c *WorkStealingCoordinator) GetMetrics() *DistributorMetrics {
	c.metrics.Mu.RLock()
	defer c.metrics.Mu.RUnlock()

	totalProcessed := int64(0)
	for _, worker := range c.workers {
		totalProcessed += atomic.LoadInt64(&worker.processed)
	}

	return &DistributorMetrics{
		TotalDistributed: totalProcessed,
		DistributionTime: c.metrics.StealLatency,
		WorkerCount:      len(c.workers),
		LoadBalance:      c.calculateLoadBalance(),
	}
}

// calculateLoadBalance calculates load balance across workers
func (c *WorkStealingCoordinator) calculateLoadBalance() float64 {
	if len(c.workers) == 0 {
		return 0.0
	}

	var totalLoad int64
	loads := make([]int64, len(c.workers))

	for i, worker := range c.workers {
		loads[i] = atomic.LoadInt64(&worker.load)
		totalLoad += loads[i]
	}

	if totalLoad == 0 {
		return 1.0
	}

	avgLoad := float64(totalLoad) / float64(len(c.workers))

	// Calculate variance
	var variance float64
	for _, load := range loads {
		diff := float64(load) - avgLoad
		variance += diff * diff
	}
	variance /= float64(len(c.workers))

	// Convert to balance score
	if variance == 0 {
		return 1.0
	}
	return 1.0 / (1.0 + variance/avgLoad)
}

// StealingWorker methods

// trySteal attempts to steal work from other workers
func (w *StealingWorker) trySteal() Work {
	start := time.Now()
	defer func() {
		w.coordinator.metrics.Mu.Lock()
		w.coordinator.metrics.StealAttempts++
		w.coordinator.metrics.StealLatency += time.Since(start)
		w.coordinator.metrics.Mu.Unlock()
	}()

	// Try to steal from random workers
	for attempts := 0; attempts < len(w.coordinator.workers); attempts++ {
		victimIndex := rand.Intn(len(w.coordinator.workers))
		victim := w.coordinator.workers[victimIndex]

		if victim.id != w.id {
			select {
			case work := <-victim.localQueue:
				atomic.AddInt64(&victim.load, -1)
				w.coordinator.metrics.Mu.Lock()
				w.coordinator.metrics.SuccessfulSteals++
				w.coordinator.metrics.Mu.Unlock()
				return work
			default:
				// Queue empty, try next worker
			}
		}
	}

	// Try global queue
	select {
	case work := <-w.coordinator.globalQueue:
		return work
	default:
		return nil
	}
}

// processWork executes a work item
func (w *StealingWorker) processWork(work Work) {
	if err := work.Execute(); err != nil {
		// Log error (in a real implementation)
		_ = err
	}
	atomic.AddInt64(&w.processed, 1)
}

// run is the main worker loop
func (w *StealingWorker) run() {
	atomic.StoreInt32(&w.running, 1)
	defer atomic.StoreInt32(&w.running, 0)

	for {
		select {
		case <-w.ctx.Done():
			return

		case work := <-w.localQueue:
			w.processWork(work)
			atomic.AddInt64(&w.load, -1)

		default:
			// Try to steal work
			if work := w.trySteal(); work != nil {
				w.processWork(work)
			} else {
				// No work available, yield
				runtime.Gosched()
				time.Sleep(time.Microsecond) // Brief pause to avoid busy waiting
			}
		}
	}
}

// GetID returns the worker ID
func (w *StealingWorker) GetID() string {
	return fmt.Sprintf("stealing-worker-%d", w.id)
}

// Submit submits work to the worker's local queue
func (w *StealingWorker) Submit(work Work) error {
	if atomic.LoadInt32(&w.running) == 0 {
		return ErrWorkerStopped
	}

	select {
	case w.localQueue <- work:
		atomic.AddInt64(&w.load, 1)
		return nil
	default:
		return ErrQueueFull
	}
}

// SubmitBatch submits multiple works
func (w *StealingWorker) SubmitBatch(works []Work) error {
	for _, work := range works {
		if err := w.Submit(work); err != nil {
			return err
		}
	}
	return nil
}

// Start starts the worker (already running)
func (w *StealingWorker) Start(ctx context.Context) error {
	return nil
}

// Stop stops the worker
func (w *StealingWorker) Stop() error {
	w.cancel()
	return nil
}

// GetLoad returns the current load
func (w *StealingWorker) GetLoad() float64 {
	return float64(atomic.LoadInt64(&w.load))
}

// GetCapacity returns the worker capacity
func (w *StealingWorker) GetCapacity() int {
	return cap(w.localQueue)
}

// GetQueueDepth returns the current queue depth
func (w *StealingWorker) GetQueueDepth() int {
	return len(w.localQueue)
}