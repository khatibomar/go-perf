package distributors

import (
	"math"
	"sync"
	"time"
	. "exercise-03-03/types"
)

// LoadBasedDistributor implements load-aware work distribution
type LoadBasedDistributor struct {
	workers []LoadAwareWorker
	mu      sync.RWMutex
	metrics *DistributorMetrics
}

// NewLoadBasedDistributor creates a new load-based distributor
func NewLoadBasedDistributor() *LoadBasedDistributor {
	return &LoadBasedDistributor{
		workers: make([]LoadAwareWorker, 0),
		metrics: &DistributorMetrics{},
	}
}

// Distribute assigns work to the worker with the lowest load
func (d *LoadBasedDistributor) Distribute(work Work) error {
	start := time.Now()
	defer func() {
		d.metrics.Mu.Lock()
		d.metrics.TotalDistributed++
		d.metrics.DistributionTime += time.Since(start)
		d.metrics.Mu.Unlock()
	}()

	d.mu.RLock()
	defer d.mu.RUnlock()

	if len(d.workers) == 0 {
		return ErrNoWorkersAvailable
	}

	// Find worker with lowest load
	var bestWorker LoadAwareWorker
	minLoad := math.MaxFloat64

	for _, worker := range d.workers {
		if load := worker.GetLoad(); load < minLoad {
			minLoad = load
			bestWorker = worker
		}
	}

	if bestWorker == nil {
		return ErrNoWorkersAvailable
	}

	return bestWorker.Submit(work)
}

// DistributeBatch distributes multiple works based on load
func (d *LoadBasedDistributor) DistributeBatch(works []Work) error {
	start := time.Now()
	defer func() {
		d.metrics.Mu.Lock()
		d.metrics.TotalDistributed += int64(len(works))
		d.metrics.DistributionTime += time.Since(start)
		d.metrics.Mu.Unlock()
	}()

	d.mu.RLock()
	defer d.mu.RUnlock()

	if len(d.workers) == 0 {
		return ErrNoWorkersAvailable
	}

	// Sort works by size (larger works to less loaded workers)
	worksCopy := make([]Work, len(works))
	copy(worksCopy, works)

	// Simple bubble sort by work size (descending)
	for i := 0; i < len(worksCopy)-1; i++ {
		for j := 0; j < len(worksCopy)-i-1; j++ {
			if worksCopy[j].GetSize() < worksCopy[j+1].GetSize() {
				worksCopy[j], worksCopy[j+1] = worksCopy[j+1], worksCopy[j]
			}
		}
	}

	// Distribute each work to the least loaded worker
	for _, work := range worksCopy {
		var bestWorker LoadAwareWorker
		minLoad := math.MaxFloat64

		for _, worker := range d.workers {
			if load := worker.GetLoad(); load < minLoad {
				minLoad = load
				bestWorker = worker
			}
		}

		if bestWorker == nil {
			return ErrNoWorkersAvailable
		}

		if err := bestWorker.Submit(work); err != nil {
			return err
		}
	}

	return nil
}

// AddWorker adds a new load-aware worker to the distributor
func (d *LoadBasedDistributor) AddWorker(worker Worker) error {
	loadAwareWorker, ok := worker.(LoadAwareWorker)
	if !ok {
		return ErrInvalidConfiguration
	}

	d.mu.Lock()
	defer d.mu.Unlock()

	d.workers = append(d.workers, loadAwareWorker)
	d.metrics.WorkerCount = len(d.workers)
	return nil
}

// RemoveWorker removes a worker from the distributor
func (d *LoadBasedDistributor) RemoveWorker(workerID string) error {
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
func (d *LoadBasedDistributor) GetMetrics() *DistributorMetrics {
	d.metrics.Mu.RLock()
	defer d.metrics.Mu.RUnlock()

	return &DistributorMetrics{
		TotalDistributed: d.metrics.TotalDistributed,
		DistributionTime: d.metrics.DistributionTime,
		WorkerCount:      d.metrics.WorkerCount,
		LoadBalance:      d.calculateLoadBalance(),
	}
}

// calculateLoadBalance calculates the load balance metric
func (d *LoadBasedDistributor) calculateLoadBalance() float64 {
	d.mu.RLock()
	defer d.mu.RUnlock()

	if len(d.workers) == 0 {
		return 0.0
	}

	// Calculate load variance
	var totalLoad float64
	loads := make([]float64, len(d.workers))

	for i, worker := range d.workers {
		loads[i] = worker.GetLoad()
		totalLoad += loads[i]
	}

	avgLoad := totalLoad / float64(len(d.workers))

	// Calculate variance
	var variance float64
	for _, load := range loads {
		variance += math.Pow(load-avgLoad, 2)
	}
	variance /= float64(len(d.workers))

	// Convert variance to balance score (lower variance = higher balance)
	if variance == 0 {
		return 1.0
	}
	return 1.0 / (1.0 + variance)
}

// GetWorkerLoads returns current loads of all workers
func (d *LoadBasedDistributor) GetWorkerLoads() map[string]float64 {
	d.mu.RLock()
	defer d.mu.RUnlock()

	loads := make(map[string]float64)
	for _, worker := range d.workers {
		loads[worker.GetID()] = worker.GetLoad()
	}
	return loads
}

// GetLeastLoadedWorker returns the worker with the lowest load
func (d *LoadBasedDistributor) GetLeastLoadedWorker() LoadAwareWorker {
	d.mu.RLock()
	defer d.mu.RUnlock()

	if len(d.workers) == 0 {
		return nil
	}

	var bestWorker LoadAwareWorker
	minLoad := math.MaxFloat64

	for _, worker := range d.workers {
		if load := worker.GetLoad(); load < minLoad {
			minLoad = load
			bestWorker = worker
		}
	}

	return bestWorker
}