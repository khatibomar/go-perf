package workers

import (
	"context"
	"sync"
	"time"
	
	"exercise-03-02/pools"
)

// Worker interface defines the basic worker contract
type Worker interface {
	// Start begins the worker's execution
	Start(ctx context.Context) error
	
	// Stop gracefully stops the worker
	Stop(timeout time.Duration) error
	
	// Submit adds work to the worker
	Submit(work WorkItem) error
	
	// GetStats returns worker statistics
	GetStats() WorkerStats
	
	// ID returns the worker's unique identifier
	ID() string
}

// WorkItem represents a unit of work
type WorkItem interface {
	// Execute performs the work
	Execute() error
	
	// Priority returns the work item's priority
	Priority() int
	
	// EstimatedDuration returns expected execution time
	EstimatedDuration() time.Duration
	
	// Context returns the work item's context
	Context() context.Context
}

// WorkerStats contains worker performance statistics
type WorkerStats struct {
	WorkerID        string
	TasksProcessed  int64
	TasksInProgress int64
	TotalExecutionTime time.Duration
	AverageExecutionTime time.Duration
	ErrorCount      int64
	LastActivity    time.Time
	Utilization     float64
}

// BasicWorker implements a simple worker
type BasicWorker struct {
	id          string
	workChan    chan WorkItem
	ctx         context.Context
	cancel      context.CancelFunc
	stats       WorkerStats
	statsMu     sync.RWMutex
	wg          sync.WaitGroup
	running     bool
	mu          sync.RWMutex
}

// NewBasicWorker creates a new basic worker
func NewBasicWorker(id string, queueSize int) *BasicWorker {
	return &BasicWorker{
		id:       id,
		workChan: make(chan WorkItem, queueSize),
		stats: WorkerStats{
			WorkerID:     id,
			LastActivity: time.Now(),
		},
	}
}

// Start begins the worker's execution
func (w *BasicWorker) Start(ctx context.Context) error {
	w.mu.Lock()
	defer w.mu.Unlock()
	
	if w.running {
		return pools.ErrWorkerAlreadyRunning
	}
	
	w.ctx, w.cancel = context.WithCancel(ctx)
	w.running = true
	
	w.wg.Add(1)
	go w.run()
	
	return nil
}

// Stop gracefully stops the worker
func (w *BasicWorker) Stop(timeout time.Duration) error {
	w.mu.Lock()
	if !w.running {
		w.mu.Unlock()
		return nil
	}
	w.mu.Unlock()
	
	w.cancel()
	close(w.workChan)
	
	done := make(chan struct{})
	go func() {
		w.wg.Wait()
		close(done)
	}()
	
	select {
	case <-done:
		w.mu.Lock()
		w.running = false
		w.mu.Unlock()
		return nil
	case <-time.After(timeout):
		return pools.ErrWorkerStopTimeout
	}
}

// Submit adds work to the worker
func (w *BasicWorker) Submit(work WorkItem) error {
	w.mu.RLock()
	running := w.running
	w.mu.RUnlock()
	
	if !running {
		return pools.ErrWorkerNotRunning
	}
	
	select {
	case w.workChan <- work:
		return nil
	case <-w.ctx.Done():
		return w.ctx.Err()
	default:
		return pools.ErrWorkerQueueFull
	}
}

// run is the main worker loop
func (w *BasicWorker) run() {
	defer w.wg.Done()
	
	for {
		select {
		case work, ok := <-w.workChan:
			if !ok {
				return // Channel closed
			}
			w.processWork(work)
		case <-w.ctx.Done():
			return
		}
	}
}

// processWork executes a work item and updates statistics
func (w *BasicWorker) processWork(work WorkItem) {
	start := time.Now()
	
	w.statsMu.Lock()
	w.stats.TasksInProgress++
	w.statsMu.Unlock()
	
	err := work.Execute()
	
	duration := time.Since(start)
	
	w.statsMu.Lock()
	w.stats.TasksInProgress--
	w.stats.TasksProcessed++
	w.stats.TotalExecutionTime += duration
	w.stats.AverageExecutionTime = w.stats.TotalExecutionTime / time.Duration(w.stats.TasksProcessed)
	w.stats.LastActivity = time.Now()
	
	if err != nil {
		w.stats.ErrorCount++
	}
	
	// Calculate utilization (simplified)
	if w.stats.TasksProcessed > 0 {
		totalTime := time.Since(start)
		w.stats.Utilization = float64(w.stats.TotalExecutionTime) / float64(totalTime)
	}
	
	w.statsMu.Unlock()
}

// GetStats returns worker statistics
func (w *BasicWorker) GetStats() WorkerStats {
	w.statsMu.RLock()
	defer w.statsMu.RUnlock()
	return w.stats
}

// ID returns the worker's unique identifier
func (w *BasicWorker) ID() string {
	return w.id
}

// QueueDepth returns the current queue depth
func (w *BasicWorker) QueueDepth() int {
	return len(w.workChan)
}

// IsRunning returns whether the worker is currently running
func (w *BasicWorker) IsRunning() bool {
	w.mu.RLock()
	defer w.mu.RUnlock()
	return w.running
}