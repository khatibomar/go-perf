package distributors

import (
	"sync"
	"sync/atomic"
	"time"
	. "exercise-03-03/types"
)



// RoundRobinDistributor implements round-robin work distribution
type RoundRobinDistributor struct {
	workers []Worker
	current int64
	mu      sync.RWMutex
	metrics *DistributorMetrics
}

// NewRoundRobinDistributor creates a new round-robin distributor
func NewRoundRobinDistributor() *RoundRobinDistributor {
	return &RoundRobinDistributor{
		workers: make([]Worker, 0),
		current: 0,
		metrics: &DistributorMetrics{},
	}
}

// Distribute assigns work to the next worker in round-robin fashion
func (d *RoundRobinDistributor) Distribute(work Work) error {
	start := time.Now()
	defer func() {
		d.metrics.Mu.Lock()
		d.metrics.TotalDistributed++
		d.metrics.DistributionTime += time.Since(start)
		d.metrics.Mu.Unlock()
	}()

	d.mu.RLock()
	workerCount := len(d.workers)
	d.mu.RUnlock()

	if workerCount == 0 {
		return ErrNoWorkersAvailable
	}

	// Atomic increment for thread safety
	index := atomic.AddInt64(&d.current, 1) - 1
	workerIndex := index % int64(workerCount)

	d.mu.RLock()
	worker := d.workers[workerIndex]
	d.mu.RUnlock()

	return worker.Submit(work)
}

// DistributeBatch distributes multiple works in round-robin fashion
func (d *RoundRobinDistributor) DistributeBatch(works []Work) error {
	start := time.Now()
	defer func() {
		d.metrics.Mu.Lock()
		d.metrics.TotalDistributed += int64(len(works))
		d.metrics.DistributionTime += time.Since(start)
		d.metrics.Mu.Unlock()
	}()

	d.mu.RLock()
	workerCount := len(d.workers)
	d.mu.RUnlock()

	if workerCount == 0 {
		return ErrNoWorkersAvailable
	}

	// Distribute works in round-robin fashion
	for i, work := range works {
		workerIndex := (atomic.LoadInt64(&d.current) + int64(i)) % int64(workerCount)
		d.mu.RLock()
		worker := d.workers[workerIndex]
		d.mu.RUnlock()

		if err := worker.Submit(work); err != nil {
			return err
		}
	}

	// Update current counter
	atomic.AddInt64(&d.current, int64(len(works)))
	return nil
}

// AddWorker adds a new worker to the distributor
func (d *RoundRobinDistributor) AddWorker(worker Worker) error {
	d.mu.Lock()
	defer d.mu.Unlock()

	d.workers = append(d.workers, worker)
	d.metrics.WorkerCount = len(d.workers)
	return nil
}

// RemoveWorker removes a worker from the distributor
func (d *RoundRobinDistributor) RemoveWorker(workerID string) error {
	d.mu.Lock()
	defer d.mu.Unlock()

	for i, worker := range d.workers {
		if worker.GetID() == workerID {
			// Remove worker by swapping with last element
			d.workers[i] = d.workers[len(d.workers)-1]
			d.workers = d.workers[:len(d.workers)-1]
			d.metrics.WorkerCount = len(d.workers)
			return nil
		}
	}

	return ErrWorkerNotFound
}

// GetMetrics returns the distributor metrics
func (d *RoundRobinDistributor) GetMetrics() *DistributorMetrics {
	d.metrics.Mu.RLock()
	defer d.metrics.Mu.RUnlock()

	// Create a copy to avoid race conditions
	return &DistributorMetrics{
		TotalDistributed: d.metrics.TotalDistributed,
		DistributionTime: d.metrics.DistributionTime,
		WorkerCount:      d.metrics.WorkerCount,
		LoadBalance:      d.calculateLoadBalance(),
	}
}

// calculateLoadBalance calculates the load balance metric
func (d *RoundRobinDistributor) calculateLoadBalance() float64 {
	d.mu.RLock()
	defer d.mu.RUnlock()

	if len(d.workers) == 0 {
		return 0.0
	}

	// For round-robin, load balance is theoretically perfect (1.0)
	// In practice, it depends on work execution times
	return 1.0
}

// GetWorkerCount returns the number of workers
func (d *RoundRobinDistributor) GetWorkerCount() int {
	d.mu.RLock()
	defer d.mu.RUnlock()
	return len(d.workers)
}

// Reset resets the distributor state
func (d *RoundRobinDistributor) Reset() {
	d.mu.Lock()
	defer d.mu.Unlock()

	atomic.StoreInt64(&d.current, 0)
	d.metrics = &DistributorMetrics{}
}