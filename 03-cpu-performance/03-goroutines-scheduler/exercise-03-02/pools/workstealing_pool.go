package pools

import (
	"context"
	"math/rand"
	"sync"
	"sync/atomic"
	"time"
)

// WorkStealingPool implements a work-stealing worker pool
type WorkStealingPool struct {
	workers     []*WorkStealingWorker
	globalQueue chan func()
	ctx         context.Context
	cancel      context.CancelFunc
	metrics     *PoolMetrics
	wg          sync.WaitGroup
	started     bool
	mu          sync.RWMutex
}

// WorkStealingWorker represents a worker with its own local queue
type WorkStealingWorker struct {
	id         int
	localQueue chan func()
	pool       *WorkStealingPool
	steals     int64
	tasks      int64
}

// NewWorkStealingPool creates a new work-stealing pool
func NewWorkStealingPool(workers int, localQueueSize, globalQueueSize int) *WorkStealingPool {
	ctx, cancel := context.WithCancel(context.Background())
	pool := &WorkStealingPool{
		workers:     make([]*WorkStealingWorker, workers),
		globalQueue: make(chan func(), globalQueueSize),
		ctx:         ctx,
		cancel:      cancel,
		metrics:     &PoolMetrics{},
	}
	
	// Initialize workers
	for i := 0; i < workers; i++ {
		pool.workers[i] = &WorkStealingWorker{
			id:         i,
			localQueue: make(chan func(), localQueueSize),
			pool:       pool,
		}
	}
	
	return pool
}

// Start initializes and starts the work-stealing pool
func (p *WorkStealingPool) Start() error {
	p.mu.Lock()
	defer p.mu.Unlock()
	
	if p.started {
		return nil
	}
	
	// Start all workers
	for _, worker := range p.workers {
		p.wg.Add(1)
		go worker.run()
	}
	
	p.started = true
	return nil
}

// Submit adds a task to the pool
func (p *WorkStealingPool) Submit(task func()) error {
	p.metrics.mu.Lock()
	p.metrics.TasksSubmitted++
	p.metrics.mu.Unlock()
	
	// Try to submit to a random worker's local queue first
	workerIdx := rand.Intn(len(p.workers))
	worker := p.workers[workerIdx]
	
	select {
	case worker.localQueue <- task:
		return nil
	default:
		// Local queue full, try global queue
		select {
		case p.globalQueue <- task:
			return nil
		case <-p.ctx.Done():
			return p.ctx.Err()
		}
	}
}

// SubmitToWorker submits a task to a specific worker's local queue
func (p *WorkStealingPool) SubmitToWorker(workerID int, task func()) error {
	if workerID < 0 || workerID >= len(p.workers) {
		return ErrInvalidWorker
	}
	
	p.metrics.mu.Lock()
	p.metrics.TasksSubmitted++
	p.metrics.mu.Unlock()
	
	worker := p.workers[workerID]
	select {
	case worker.localQueue <- task:
		return nil
	default:
		return ErrPoolFull
	}
}

// run executes the worker's main loop
func (w *WorkStealingWorker) run() {
	defer w.pool.wg.Done()
	
	for {
		task := w.getTask()
		if task == nil {
			select {
			case <-w.pool.ctx.Done():
				return
			default:
				// No task available, brief pause before trying again
				time.Sleep(time.Microsecond * 100)
				continue
			}
		}
		
		w.executeTask(task)
	}
}

// getTask attempts to get a task using work-stealing strategy
func (w *WorkStealingWorker) getTask() func() {
	// 1. Try local queue first (LIFO for better cache locality)
	select {
	case task := <-w.localQueue:
		atomic.AddInt64(&w.tasks, 1)
		return task
	default:
	}
	
	// 2. Try global queue
	select {
	case task := <-w.pool.globalQueue:
		atomic.AddInt64(&w.tasks, 1)
		return task
	default:
	}
	
	// 3. Try to steal from other workers
	return w.stealWork()
}

// stealWork attempts to steal work from other workers
func (w *WorkStealingWorker) stealWork() func() {
	// Randomize the order of workers to steal from
	workers := make([]*WorkStealingWorker, len(w.pool.workers))
	copy(workers, w.pool.workers)
	
	// Shuffle workers array
	for i := len(workers) - 1; i > 0; i-- {
		j := rand.Intn(i + 1)
		workers[i], workers[j] = workers[j], workers[i]
	}
	
	// Try to steal from each worker
	for _, victim := range workers {
		if victim.id != w.id {
			select {
			case task := <-victim.localQueue:
				atomic.AddInt64(&w.steals, 1)
				atomic.AddInt64(&w.tasks, 1)
				return task
			default:
				// Continue to next victim
			}
		}
	}
	
	return nil
}

// executeTask executes a task and updates metrics
func (w *WorkStealingWorker) executeTask(task func()) {
	start := time.Now()
	
	w.pool.metrics.mu.Lock()
	w.pool.metrics.TasksInProgress++
	w.pool.metrics.mu.Unlock()
	
	task()
	
	duration := time.Since(start)
	w.pool.metrics.mu.Lock()
	w.pool.metrics.TasksCompleted++
	w.pool.metrics.TasksInProgress--
	w.pool.metrics.AverageLatency = (w.pool.metrics.AverageLatency + duration) / 2
	w.pool.metrics.mu.Unlock()
}

// Shutdown gracefully shuts down the pool
func (p *WorkStealingPool) Shutdown(timeout time.Duration) error {
	p.mu.Lock()
	defer p.mu.Unlock()
	
	if !p.started {
		return nil
	}
	
	p.cancel()
	
	// Close all local queues
	for _, worker := range p.workers {
		close(worker.localQueue)
	}
	close(p.globalQueue)
	
	done := make(chan struct{})
	go func() {
		p.wg.Wait()
		close(done)
	}()
	
	select {
	case <-done:
		return nil
	case <-time.After(timeout):
		return ErrShutdownTimeout
	}
}

// GetMetrics returns current pool metrics
func (p *WorkStealingPool) GetMetrics() PoolMetrics {
	p.metrics.mu.RLock()
	defer p.metrics.mu.RUnlock()
	return *p.metrics
}

// GetWorkerStats returns statistics for all workers
func (p *WorkStealingPool) GetWorkerStats() []WorkerStats {
	stats := make([]WorkerStats, len(p.workers))
	for i, worker := range p.workers {
		stats[i] = WorkerStats{
			WorkerID:    worker.id,
			TasksProcessed: atomic.LoadInt64(&worker.tasks),
			StealsPerformed: atomic.LoadInt64(&worker.steals),
			QueueDepth:  len(worker.localQueue),
		}
	}
	return stats
}

// QueueDepth returns the current global queue depth
func (p *WorkStealingPool) QueueDepth() int {
	return len(p.globalQueue)
}

// WorkerCount returns the number of workers
func (p *WorkStealingPool) WorkerCount() int {
	return len(p.workers)
}

// GetWorkerQueueDepth returns the queue depth for a specific worker
func (p *WorkStealingPool) GetWorkerQueueDepth(workerID int) int {
	if workerID < 0 || workerID >= len(p.workers) {
		return -1
	}
	return len(p.workers[workerID].localQueue)
}

// WorkerStats contains statistics for a single worker
type WorkerStats struct {
	WorkerID        int
	TasksProcessed  int64
	StealsPerformed int64
	QueueDepth      int
}