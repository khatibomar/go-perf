package pools

import (
	"context"
	"sync"
	"sync/atomic"
	"time"
)

// DynamicPool implements a dynamically scaling worker pool
type DynamicPool struct {
	minWorkers    int
	maxWorkers    int
	currentWorkers int64
	workChan      chan func()
	wg            sync.WaitGroup
	ctx           context.Context
	cancel        context.CancelFunc
	metrics       *PoolMetrics
	scaler        *PoolScaler
	started       bool
	mu            sync.RWMutex
}

// PoolScaler manages dynamic scaling decisions
type PoolScaler struct {
	scaleUpThreshold   float64
	scaleDownThreshold float64
	scaleInterval      time.Duration
	lastScaleTime      time.Time
	mu                 sync.RWMutex
}

// NewDynamicPool creates a new dynamic worker pool
func NewDynamicPool(minWorkers, maxWorkers int, queueSize int) *DynamicPool {
	ctx, cancel := context.WithCancel(context.Background())
	return &DynamicPool{
		minWorkers:     minWorkers,
		maxWorkers:     maxWorkers,
		currentWorkers: int64(minWorkers),
		workChan:       make(chan func(), queueSize),
		ctx:            ctx,
		cancel:         cancel,
		metrics:        &PoolMetrics{},
		scaler: &PoolScaler{
			scaleUpThreshold:   0.8,   // Scale up when 80% utilized
			scaleDownThreshold: 0.2,   // Scale down when 20% utilized
			scaleInterval:      time.Second * 5,
		},
	}
}

// Start initializes and starts the dynamic worker pool
func (p *DynamicPool) Start() error {
	p.mu.Lock()
	defer p.mu.Unlock()
	
	if p.started {
		return nil
	}
	
	// Start minimum workers
	for i := 0; i < p.minWorkers; i++ {
		p.wg.Add(1)
		go p.worker(i)
	}
	
	// Start scaler goroutine
	go p.scaleManager()
	
	p.started = true
	return nil
}

// Submit adds a task to the pool
func (p *DynamicPool) Submit(task func()) error {
	p.metrics.mu.Lock()
	p.metrics.TasksSubmitted++
	p.metrics.mu.Unlock()
	
	select {
	case p.workChan <- task:
		return nil
	case <-p.ctx.Done():
		return p.ctx.Err()
	}
}

// worker processes tasks from the work channel
func (p *DynamicPool) worker(id int) {
	defer func() {
		atomic.AddInt64(&p.currentWorkers, -1)
		p.wg.Done()
	}()
	
	idleTimeout := time.NewTimer(time.Minute * 2) // Workers timeout after 2 minutes of inactivity
	defer idleTimeout.Stop()
	
	for {
		select {
		case task := <-p.workChan:
			if task != nil {
				idleTimeout.Reset(time.Minute * 2)
				
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
		case <-idleTimeout.C:
			// Worker timeout - exit if we have more than minimum workers
			if atomic.LoadInt64(&p.currentWorkers) > int64(p.minWorkers) {
				return
			}
			idleTimeout.Reset(time.Minute * 2)
		case <-p.ctx.Done():
			return
		}
	}
}

// scaleManager monitors pool utilization and scales workers
func (p *DynamicPool) scaleManager() {
	ticker := time.NewTicker(p.scaler.scaleInterval)
	defer ticker.Stop()
	
	for {
		select {
		case <-ticker.C:
			p.evaluateScaling()
		case <-p.ctx.Done():
			return
		}
	}
}

// evaluateScaling determines if scaling is needed
func (p *DynamicPool) evaluateScaling() {
	p.scaler.mu.Lock()
	defer p.scaler.mu.Unlock()
	
	// Prevent too frequent scaling
	if time.Since(p.scaler.lastScaleTime) < p.scaler.scaleInterval {
		return
	}
	
	currentWorkers := atomic.LoadInt64(&p.currentWorkers)
	queueDepth := len(p.workChan)
	utilization := float64(queueDepth) / float64(cap(p.workChan))
	
	// Scale up if utilization is high and we haven't reached max workers
	if utilization > p.scaler.scaleUpThreshold && currentWorkers < int64(p.maxWorkers) {
		newWorkers := min(int64(p.maxWorkers), currentWorkers+int64(currentWorkers/4)+1)
		for i := currentWorkers; i < newWorkers; i++ {
			atomic.AddInt64(&p.currentWorkers, 1)
			p.wg.Add(1)
			go p.worker(int(i))
		}
		p.scaler.lastScaleTime = time.Now()
	}
	
	// Scale down if utilization is low and we have more than minimum workers
	if utilization < p.scaler.scaleDownThreshold && currentWorkers > int64(p.minWorkers) {
		// Scaling down happens naturally through worker timeouts
		p.scaler.lastScaleTime = time.Now()
	}
}

// Shutdown gracefully shuts down the pool
func (p *DynamicPool) Shutdown(timeout time.Duration) error {
	p.mu.Lock()
	defer p.mu.Unlock()
	
	if !p.started {
		return nil
	}
	
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
func (p *DynamicPool) GetMetrics() PoolMetrics {
	p.metrics.mu.RLock()
	defer p.metrics.mu.RUnlock()
	return *p.metrics
}

// QueueDepth returns the current queue depth
func (p *DynamicPool) QueueDepth() int {
	return len(p.workChan)
}

// WorkerCount returns the current number of workers
func (p *DynamicPool) WorkerCount() int {
	return int(atomic.LoadInt64(&p.currentWorkers))
}

// SetScalingParameters updates scaling thresholds
func (p *DynamicPool) SetScalingParameters(upThreshold, downThreshold float64, interval time.Duration) {
	p.scaler.mu.Lock()
	defer p.scaler.mu.Unlock()
	
	p.scaler.scaleUpThreshold = upThreshold
	p.scaler.scaleDownThreshold = downThreshold
	p.scaler.scaleInterval = interval
}

func min(a, b int64) int64 {
	if a < b {
		return a
	}
	return b
}