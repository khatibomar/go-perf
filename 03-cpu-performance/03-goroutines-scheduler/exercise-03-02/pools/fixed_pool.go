package pools

import (
	"context"
	"sync"
	"time"
)

// FixedPool implements a fixed-size worker pool
type FixedPool struct {
	workers   int
	workChan  chan func()
	wg        sync.WaitGroup
	ctx       context.Context
	cancel    context.CancelFunc
	metrics   *PoolMetrics
	shutdown  chan struct{}
	started   bool
	mu        sync.RWMutex
}

// PoolMetrics tracks pool performance metrics
type PoolMetrics struct {
	TasksSubmitted   int64
	TasksCompleted   int64
	TasksInProgress  int64
	WorkerUtilization float64
	AverageLatency   time.Duration
	mu               sync.RWMutex
}

// NewFixedPool creates a new fixed-size worker pool
func NewFixedPool(workers int, queueSize int) *FixedPool {
	ctx, cancel := context.WithCancel(context.Background())
	return &FixedPool{
		workers:  workers,
		workChan: make(chan func(), queueSize),
		ctx:      ctx,
		cancel:   cancel,
		metrics:  &PoolMetrics{},
		shutdown: make(chan struct{}),
	}
}

// Start initializes and starts the worker pool
func (p *FixedPool) Start() error {
	p.mu.Lock()
	defer p.mu.Unlock()
	
	if p.started {
		return nil
	}
	
	for i := 0; i < p.workers; i++ {
		p.wg.Add(1)
		go p.worker(i)
	}
	
	p.started = true
	return nil
}

// Submit adds a task to the pool
func (p *FixedPool) Submit(task func()) error {
	p.mu.RLock()
	if !p.started {
		p.mu.RUnlock()
		return ErrPoolShutdown
	}
	p.mu.RUnlock()
	
	p.metrics.mu.Lock()
	p.metrics.TasksSubmitted++
	p.metrics.mu.Unlock()
	
	select {
	case p.workChan <- task:
		return nil
	case <-p.ctx.Done():
		return ErrPoolShutdown
	}
}

// worker processes tasks from the work channel
func (p *FixedPool) worker(id int) {
	defer p.wg.Done()
	
	for {
		select {
		case task := <-p.workChan:
			if task != nil {
				start := time.Now()
				
				p.metrics.mu.Lock()
				p.metrics.TasksInProgress++
				p.metrics.mu.Unlock()
				
				task()
				
				duration := time.Since(start)
				p.metrics.mu.Lock()
				p.metrics.TasksCompleted++
				p.metrics.TasksInProgress--
				p.metrics.AverageLatency = (p.metrics.AverageLatency + duration) / 2
				p.metrics.mu.Unlock()
			}
		case <-p.ctx.Done():
			return
		}
	}
}

// Shutdown gracefully shuts down the pool
func (p *FixedPool) Shutdown(timeout time.Duration) error {
	p.mu.Lock()
	defer p.mu.Unlock()
	
	if !p.started {
		return nil
	}
	
	p.started = false
	p.cancel()
	close(p.workChan)
	
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
func (p *FixedPool) GetMetrics() PoolMetrics {
	p.metrics.mu.RLock()
	defer p.metrics.mu.RUnlock()
	return *p.metrics
}

// QueueDepth returns the current queue depth
func (p *FixedPool) QueueDepth() int {
	return len(p.workChan)
}

// WorkerCount returns the number of workers
func (p *FixedPool) WorkerCount() int {
	return p.workers
}