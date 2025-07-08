package distributors

import (
	"sync"
	"time"
	. "exercise-03-03/types"
)

// HierarchicalDistributor implements hierarchical work distribution
type HierarchicalDistributor struct {
	levels  [][]Distributor
	fanout  int
	mu      sync.RWMutex
	metrics *DistributorMetrics
}

// NewHierarchicalDistributor creates a new hierarchical distributor
func NewHierarchicalDistributor(fanout int) *HierarchicalDistributor {
	return &HierarchicalDistributor{
		levels:  make([][]Distributor, 0),
		fanout:  fanout,
		metrics: &DistributorMetrics{},
	}
}

// AddLevel adds a new level to the hierarchy
func (d *HierarchicalDistributor) AddLevel(distributors []Distributor) {
	d.mu.Lock()
	defer d.mu.Unlock()

	d.levels = append(d.levels, distributors)
	d.updateMetrics()
}

// Distribute distributes work through the hierarchy
func (d *HierarchicalDistributor) Distribute(work Work) error {
	start := time.Now()
	defer func() {
		d.metrics.Mu.Lock()
		d.metrics.TotalDistributed++
		d.metrics.DistributionTime += time.Since(start)
		d.metrics.Mu.Unlock()
	}()

	d.mu.RLock()
	defer d.mu.RUnlock()

	if len(d.levels) == 0 {
		return ErrNoWorkersAvailable
	}

	// Start from top level
	currentLevel := 0
	currentIndex := 0

	// Navigate through hierarchy
	for currentLevel < len(d.levels)-1 {
		// Calculate next level indices
		nextLevelStart := currentIndex * d.fanout
		nextLevelEnd := min(nextLevelStart+d.fanout, len(d.levels[currentLevel+1]))

		if nextLevelStart >= len(d.levels[currentLevel+1]) {
			// No distributors available at next level
			return ErrNoWorkersAvailable
		}

		// Choose distributor in next level based on work hash
		targetIndex := nextLevelStart + (work.Hash() % (nextLevelEnd - nextLevelStart))

		currentLevel++
		currentIndex = targetIndex
	}

	// Submit to final level distributor
	if currentIndex >= len(d.levels[currentLevel]) {
		return ErrNoWorkersAvailable
	}

	return d.levels[currentLevel][currentIndex].Distribute(work)
}

// DistributeBatch distributes multiple works through the hierarchy
func (d *HierarchicalDistributor) DistributeBatch(works []Work) error {
	start := time.Now()
	defer func() {
		d.metrics.Mu.Lock()
		d.metrics.TotalDistributed += int64(len(works))
		d.metrics.DistributionTime += time.Since(start)
		d.metrics.Mu.Unlock()
	}()

	// Group works by their target distributors to optimize batch processing
	workGroups := make(map[Distributor][]Work)

	for _, work := range works {
		targetDistributor, err := d.findTargetDistributor(work)
		if err != nil {
			return err
		}

		workGroups[targetDistributor] = append(workGroups[targetDistributor], work)
	}

	// Submit batches to each target distributor
	for distributor, workBatch := range workGroups {
		if err := distributor.DistributeBatch(workBatch); err != nil {
			return err
		}
	}

	return nil
}

// findTargetDistributor finds the target distributor for a work item
func (d *HierarchicalDistributor) findTargetDistributor(work Work) (Distributor, error) {
	d.mu.RLock()
	defer d.mu.RUnlock()

	if len(d.levels) == 0 {
		return nil, ErrNoWorkersAvailable
	}

	currentLevel := 0
	currentIndex := 0

	for currentLevel < len(d.levels)-1 {
		nextLevelStart := currentIndex * d.fanout
		nextLevelEnd := min(nextLevelStart+d.fanout, len(d.levels[currentLevel+1]))

		if nextLevelStart >= len(d.levels[currentLevel+1]) {
			return nil, ErrNoWorkersAvailable
		}

		targetIndex := nextLevelStart + (work.Hash() % (nextLevelEnd - nextLevelStart))
		currentLevel++
		currentIndex = targetIndex
	}

	if currentIndex >= len(d.levels[currentLevel]) {
		return nil, ErrNoWorkersAvailable
	}

	return d.levels[currentLevel][currentIndex], nil
}

// AddWorker adds a worker to the bottom level
func (d *HierarchicalDistributor) AddWorker(worker Worker) error {
	d.mu.Lock()
	defer d.mu.Unlock()

	if len(d.levels) == 0 {
		// Create first level with a round-robin distributor
		roundRobin := NewRoundRobinDistributor()
		roundRobin.AddWorker(worker)
		d.levels = append(d.levels, []Distributor{roundRobin})
	} else {
		// Add to the last distributor in the bottom level
		bottomLevel := len(d.levels) - 1
		if len(d.levels[bottomLevel]) > 0 {
			lastDistributor := d.levels[bottomLevel][len(d.levels[bottomLevel])-1]
			if err := lastDistributor.AddWorker(worker); err != nil {
				return err
			}
		}
	}

	d.updateMetrics()
	return nil
}

// RemoveWorker removes a worker from the hierarchy
func (d *HierarchicalDistributor) RemoveWorker(workerID string) error {
	d.mu.Lock()
	defer d.mu.Unlock()

	// Try to remove from all distributors in bottom level
	if len(d.levels) == 0 {
		return ErrWorkerNotFound
	}

	bottomLevel := len(d.levels) - 1
	for _, distributor := range d.levels[bottomLevel] {
		if err := distributor.RemoveWorker(workerID); err == nil {
			d.updateMetrics()
			return nil
		}
	}

	return ErrWorkerNotFound
}

// GetMetrics returns the distributor metrics
func (d *HierarchicalDistributor) GetMetrics() *DistributorMetrics {
	d.metrics.Mu.RLock()
	defer d.metrics.Mu.RUnlock()

	return &DistributorMetrics{
		TotalDistributed: d.metrics.TotalDistributed,
		DistributionTime: d.metrics.DistributionTime,
		WorkerCount:      d.metrics.WorkerCount,
		LoadBalance:      d.calculateLoadBalance(),
	}
}

// calculateLoadBalance calculates load balance across the hierarchy
func (d *HierarchicalDistributor) calculateLoadBalance() float64 {
	d.mu.RLock()
	defer d.mu.RUnlock()

	if len(d.levels) == 0 {
		return 0.0
	}

	// Calculate average load balance across bottom level distributors
	bottomLevel := len(d.levels) - 1
	if len(d.levels[bottomLevel]) == 0 {
		return 0.0
	}

	var totalBalance float64
	for _, distributor := range d.levels[bottomLevel] {
		metrics := distributor.GetMetrics()
		totalBalance += metrics.LoadBalance
	}

	return totalBalance / float64(len(d.levels[bottomLevel]))
}

// updateMetrics updates the internal metrics
func (d *HierarchicalDistributor) updateMetrics() {
	workerCount := 0
	if len(d.levels) > 0 {
		bottomLevel := len(d.levels) - 1
		for _, distributor := range d.levels[bottomLevel] {
			metrics := distributor.GetMetrics()
			workerCount += metrics.WorkerCount
		}
	}

	d.metrics.Mu.Lock()
	d.metrics.WorkerCount = workerCount
	d.metrics.Mu.Unlock()
}

// GetLevelCount returns the number of levels in the hierarchy
func (d *HierarchicalDistributor) GetLevelCount() int {
	d.mu.RLock()
	defer d.mu.RUnlock()
	return len(d.levels)
}

// GetDistributorsAtLevel returns distributors at a specific level
func (d *HierarchicalDistributor) GetDistributorsAtLevel(level int) []Distributor {
	d.mu.RLock()
	defer d.mu.RUnlock()

	if level < 0 || level >= len(d.levels) {
		return nil
	}

	// Return a copy to avoid race conditions
	distributors := make([]Distributor, len(d.levels[level]))
	copy(distributors, d.levels[level])
	return distributors
}

// min returns the minimum of two integers
func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}