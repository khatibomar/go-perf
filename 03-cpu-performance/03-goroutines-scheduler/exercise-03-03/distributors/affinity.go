package distributors

import (
	"hash/fnv"
	"sync"
	"time"
	. "exercise-03-03/types"
)

// AffinityDistributor implements affinity-based work distribution
type AffinityDistributor struct {
	workers     []Worker
	affinityMap map[string]int // Data key -> worker index
	mu          sync.RWMutex
	metrics     *DistributorMetrics
	hashFunc    func(string) uint32
}

// NewAffinityDistributor creates a new affinity-based distributor
func NewAffinityDistributor() *AffinityDistributor {
	return &AffinityDistributor{
		workers:     make([]Worker, 0),
		affinityMap: make(map[string]int),
		metrics:     &DistributorMetrics{},
		hashFunc:    defaultHashFunc,
	}
}

// defaultHashFunc provides a default hash function
func defaultHashFunc(key string) uint32 {
	h := fnv.New32a()
	h.Write([]byte(key))
	return h.Sum32()
}

// Distribute assigns work based on affinity key
func (d *AffinityDistributor) Distribute(work Work) error {
	start := time.Now()
	defer func() {
		d.metrics.Mu.Lock()
		d.metrics.TotalDistributed++
		d.metrics.DistributionTime += time.Since(start)
		d.metrics.Mu.Unlock()
	}()

	key := work.GetAffinityKey()
	if key == "" {
		// No affinity key, use hash-based distribution
		return d.distributeByHash(work)
	}

	d.mu.RLock()
	workerIndex, exists := d.affinityMap[key]
	workerCount := len(d.workers)
	d.mu.RUnlock()

	if !exists {
		// Assign new affinity
		d.mu.Lock()
		if workerIndex, exists = d.affinityMap[key]; !exists {
			if workerCount == 0 {
				d.mu.Unlock()
				return ErrNoWorkersAvailable
			}
			// Use hash to determine initial assignment
			workerIndex = int(d.hashFunc(key)) % workerCount
			d.affinityMap[key] = workerIndex
		}
		d.mu.Unlock()
	}

	if workerIndex >= workerCount {
		// Worker no longer exists, reassign
		d.mu.Lock()
		if workerCount > 0 {
			workerIndex = int(d.hashFunc(key)) % workerCount
			d.affinityMap[key] = workerIndex
		} else {
			d.mu.Unlock()
			return ErrNoWorkersAvailable
		}
		d.mu.Unlock()
	}

	d.mu.RLock()
	worker := d.workers[workerIndex]
	d.mu.RUnlock()

	return worker.Submit(work)
}

// distributeByHash distributes work using hash-based assignment
func (d *AffinityDistributor) distributeByHash(work Work) error {
	d.mu.RLock()
	workerCount := len(d.workers)
	d.mu.RUnlock()

	if workerCount == 0 {
		return ErrNoWorkersAvailable
	}

	workerIndex := work.Hash() % workerCount

	d.mu.RLock()
	worker := d.workers[workerIndex]
	d.mu.RUnlock()

	return worker.Submit(work)
}

// DistributeBatch distributes multiple works based on affinity
func (d *AffinityDistributor) DistributeBatch(works []Work) error {
	start := time.Now()
	defer func() {
		d.metrics.Mu.Lock()
		d.metrics.TotalDistributed += int64(len(works))
		d.metrics.DistributionTime += time.Since(start)
		d.metrics.Mu.Unlock()
	}()

	// Group works by affinity key for batch processing
	affinityGroups := make(map[string][]Work)
	hashGroups := make(map[int][]Work)

	for _, work := range works {
		key := work.GetAffinityKey()
		if key != "" {
			affinityGroups[key] = append(affinityGroups[key], work)
		} else {
			hashIndex := work.Hash() % d.getWorkerCount()
			hashGroups[hashIndex] = append(hashGroups[hashIndex], work)
		}
	}

	// Process affinity groups
	for key, workGroup := range affinityGroups {
		workerIndex := d.getOrAssignAffinity(key)
		if workerIndex == -1 {
			return ErrNoWorkersAvailable
		}

		d.mu.RLock()
		worker := d.workers[workerIndex]
		d.mu.RUnlock()

		if err := worker.SubmitBatch(workGroup); err != nil {
			return err
		}
	}

	// Process hash groups
	for workerIndex, workGroup := range hashGroups {
		d.mu.RLock()
		if workerIndex >= len(d.workers) {
			d.mu.RUnlock()
			continue
		}
		worker := d.workers[workerIndex]
		d.mu.RUnlock()

		if err := worker.SubmitBatch(workGroup); err != nil {
			return err
		}
	}

	return nil
}

// getOrAssignAffinity gets or assigns worker index for affinity key
func (d *AffinityDistributor) getOrAssignAffinity(key string) int {
	d.mu.RLock()
	workerIndex, exists := d.affinityMap[key]
	workerCount := len(d.workers)
	d.mu.RUnlock()

	if !exists {
		d.mu.Lock()
		if workerIndex, exists = d.affinityMap[key]; !exists {
			if workerCount == 0 {
				d.mu.Unlock()
				return -1
			}
			workerIndex = int(d.hashFunc(key)) % workerCount
			d.affinityMap[key] = workerIndex
		}
		d.mu.Unlock()
	}

	if workerIndex >= workerCount {
		// Reassign if worker no longer exists
		d.mu.Lock()
		if workerCount > 0 {
			workerIndex = int(d.hashFunc(key)) % workerCount
			d.affinityMap[key] = workerIndex
		} else {
			workerIndex = -1
		}
		d.mu.Unlock()
	}

	return workerIndex
}

// getWorkerCount returns the current worker count
func (d *AffinityDistributor) getWorkerCount() int {
	d.mu.RLock()
	defer d.mu.RUnlock()
	return len(d.workers)
}

// AddWorker adds a new worker to the distributor
func (d *AffinityDistributor) AddWorker(worker Worker) error {
	d.mu.Lock()
	defer d.mu.Unlock()

	d.workers = append(d.workers, worker)
	d.metrics.WorkerCount = len(d.workers)

	// Rebalance existing affinities
	d.rebalanceAffinities()
	return nil
}

// RemoveWorker removes a worker from the distributor
func (d *AffinityDistributor) RemoveWorker(workerID string) error {
	d.mu.Lock()
	defer d.mu.Unlock()

	for i, worker := range d.workers {
		if worker.GetID() == workerID {
			// Remove worker
			d.workers = append(d.workers[:i], d.workers[i+1:]...)
			d.metrics.WorkerCount = len(d.workers)

			// Rebalance affinities
			d.rebalanceAffinities()
			return nil
		}
	}

	return ErrWorkerNotFound
}

// rebalanceAffinities reassigns affinities after worker changes
func (d *AffinityDistributor) rebalanceAffinities() {
	if len(d.workers) == 0 {
		d.affinityMap = make(map[string]int)
		return
	}

	// Reassign all affinities based on current worker count
	for key := range d.affinityMap {
		d.affinityMap[key] = int(d.hashFunc(key)) % len(d.workers)
	}
}

// GetMetrics returns the distributor metrics
func (d *AffinityDistributor) GetMetrics() *DistributorMetrics {
	d.metrics.Mu.RLock()
	defer d.metrics.Mu.RUnlock()

	return &DistributorMetrics{
		TotalDistributed: d.metrics.TotalDistributed,
		DistributionTime: d.metrics.DistributionTime,
		WorkerCount:      d.metrics.WorkerCount,
		LoadBalance:      d.calculateLoadBalance(),
	}
}

// calculateLoadBalance calculates load balance based on affinity distribution
func (d *AffinityDistributor) calculateLoadBalance() float64 {
	d.mu.RLock()
	defer d.mu.RUnlock()

	if len(d.workers) == 0 {
		return 0.0
	}

	// Count assignments per worker
	workerCounts := make([]int, len(d.workers))
	for _, workerIndex := range d.affinityMap {
		if workerIndex >= 0 && workerIndex < len(d.workers) {
			workerCounts[workerIndex]++
		}
	}

	if len(d.affinityMap) == 0 {
		return 1.0 // Perfect balance when no affinities
	}

	// Calculate variance
	avgCount := float64(len(d.affinityMap)) / float64(len(d.workers))
	var variance float64
	for _, count := range workerCounts {
		diff := float64(count) - avgCount
		variance += diff * diff
	}
	variance /= float64(len(d.workers))

	// Convert to balance score
	if variance == 0 {
		return 1.0
	}
	return 1.0 / (1.0 + variance/avgCount)
}

// GetAffinityMap returns a copy of the current affinity mappings
func (d *AffinityDistributor) GetAffinityMap() map[string]int {
	d.mu.RLock()
	defer d.mu.RUnlock()

	affinityMap := make(map[string]int)
	for key, workerIndex := range d.affinityMap {
		affinityMap[key] = workerIndex
	}
	return affinityMap
}

// ClearAffinity removes affinity for a specific key
func (d *AffinityDistributor) ClearAffinity(key string) {
	d.mu.Lock()
	defer d.mu.Unlock()
	delete(d.affinityMap, key)
}

// SetHashFunction sets a custom hash function
func (d *AffinityDistributor) SetHashFunction(hashFunc func(string) uint32) {
	d.mu.Lock()
	defer d.mu.Unlock()
	d.hashFunc = hashFunc

	// Rebalance with new hash function
	d.rebalanceAffinities()
}

// GetAffinityStats returns statistics about affinity distribution
func (d *AffinityDistributor) GetAffinityStats() map[string]interface{} {
	d.mu.RLock()
	defer d.mu.RUnlock()

	workerCounts := make([]int, len(d.workers))
	for _, workerIndex := range d.affinityMap {
		if workerIndex >= 0 && workerIndex < len(d.workers) {
			workerCounts[workerIndex]++
		}
	}

	stats := map[string]interface{}{
		"total_affinities": len(d.affinityMap),
		"worker_count":     len(d.workers),
		"worker_loads":     workerCounts,
		"load_balance":     d.calculateLoadBalance(),
	}

	return stats
}