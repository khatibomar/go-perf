package monitoring

import (
	"context"
	"sync"
	"sync/atomic"
	"time"
	. "exercise-03-03/types"
)

// LoadMonitor tracks system load and performance metrics
type LoadMonitor struct {
	metrics         *Metrics
	distributors    map[string]Distributor
	workers         map[string]LoadAwareWorker
	mu              sync.RWMutex
	ctx             context.Context
	cancel          context.CancelFunc
	running         int32
	monitorInterval time.Duration
	latencySamples  []time.Duration
	sampleIndex     int
	maxSamples      int
}

// NewLoadMonitor creates a new load monitor
func NewLoadMonitor(monitorInterval time.Duration) *LoadMonitor {
	maxSamples := 1000
	return &LoadMonitor{
		metrics: &Metrics{
			Distributors: make(map[string]*DistributorMetrics),
			Workers:      make(map[string]*WorkerMetrics),
			Overall:      &OverallMetrics{StartTime: time.Now()},
		},
		distributors:    make(map[string]Distributor),
		workers:         make(map[string]LoadAwareWorker),
		monitorInterval: monitorInterval,
		latencySamples:  make([]time.Duration, maxSamples),
		maxSamples:      maxSamples,
	}
}

// Start starts the monitoring process
func (m *LoadMonitor) Start(ctx context.Context) {
	if !atomic.CompareAndSwapInt32(&m.running, 0, 1) {
		return
	}

	m.ctx, m.cancel = context.WithCancel(ctx)
	m.metrics.Overall.StartTime = time.Now()

	go m.monitorLoop()
}

// Stop stops the monitoring process
func (m *LoadMonitor) Stop() {
	if !atomic.CompareAndSwapInt32(&m.running, 1, 0) {
		return
	}

	if m.cancel != nil {
		m.cancel()
	}
}

// AddDistributor adds a distributor to monitor
func (m *LoadMonitor) AddDistributor(name string, distributor Distributor) {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.distributors[name] = distributor
	m.metrics.Distributors[name] = &DistributorMetrics{}
}

// AddWorker adds a worker to monitor
func (m *LoadMonitor) AddWorker(worker LoadAwareWorker) {
	m.mu.Lock()
	defer m.mu.Unlock()

	workerID := worker.GetID()
	m.workers[workerID] = worker
	m.metrics.Workers[workerID] = &WorkerMetrics{
		WorkerID:       workerID,
		LastActivity:   time.Now(),
	}
}

// RemoveDistributor removes a distributor from monitoring
func (m *LoadMonitor) RemoveDistributor(name string) {
	m.mu.Lock()
	defer m.mu.Unlock()

	delete(m.distributors, name)
	delete(m.metrics.Distributors, name)
}

// RemoveWorker removes a worker from monitoring
func (m *LoadMonitor) RemoveWorker(workerID string) {
	m.mu.Lock()
	defer m.mu.Unlock()

	delete(m.workers, workerID)
	delete(m.metrics.Workers, workerID)
}

// RecordDistribution records a distribution event
func (m *LoadMonitor) RecordDistribution(workerID string, latency time.Duration) {
	m.mu.Lock()
	defer m.mu.Unlock()

	// Update latency samples
	m.latencySamples[m.sampleIndex] = latency
	m.sampleIndex = (m.sampleIndex + 1) % m.maxSamples

	// Update worker metrics if exists
	if workerMetrics, exists := m.metrics.Workers[workerID]; exists {
		workerMetrics.Mu.Lock()
		workerMetrics.LastActivity = time.Now()
		workerMetrics.Mu.Unlock()
	}
}

// RecordExecution records a work execution event
func (m *LoadMonitor) RecordExecution(workID string, duration time.Duration) {
	m.metrics.Overall.Mu.Lock()
	m.metrics.Overall.TotalTasks++
	m.metrics.Overall.Mu.Unlock()

	// Update latency samples
	m.mu.Lock()
	m.latencySamples[m.sampleIndex] = duration
	m.sampleIndex = (m.sampleIndex + 1) % m.maxSamples
	m.mu.Unlock()
}

// GetMetrics returns current metrics
func (m *LoadMonitor) GetMetrics() *Metrics {
	m.mu.RLock()
	defer m.mu.RUnlock()

	// Create a deep copy of metrics
	metrics := &Metrics{
		Distributors: make(map[string]*DistributorMetrics),
		Workers:      make(map[string]*WorkerMetrics),
		Overall:      &OverallMetrics{},
	}

	// Copy distributor metrics
	for name, dist := range m.distributors {
		metrics.Distributors[name] = dist.GetMetrics()
	}

	// Copy worker metrics
	for workerID, worker := range m.workers {
		metrics.Workers[workerID] = &WorkerMetrics{
			WorkerID:       workerID,
			AverageLoad:    worker.GetLoad(),
			QueueDepth:     worker.GetQueueDepth(),
			LastActivity:   time.Now(),
		}
	}

	// Copy overall metrics
	m.metrics.Overall.Mu.RLock()
	metrics.Overall = &OverallMetrics{
		TotalTasks:     m.metrics.Overall.TotalTasks,
		Throughput:     m.calculateThroughput(),
		AverageLatency: m.calculateAverageLatency(),
		P99Latency:     m.calculateP99Latency(),
		StartTime:      m.metrics.Overall.StartTime,
	}
	m.metrics.Overall.Mu.RUnlock()

	return metrics
}

// monitorLoop is the main monitoring loop
func (m *LoadMonitor) monitorLoop() {
	ticker := time.NewTicker(m.monitorInterval)
	defer ticker.Stop()

	for {
		select {
		case <-m.ctx.Done():
			return

		case <-ticker.C:
			m.updateMetrics()
		}
	}
}

// updateMetrics updates all metrics
func (m *LoadMonitor) updateMetrics() {
	m.mu.RLock()
	defer m.mu.RUnlock()

	// Update worker metrics
	for workerID, worker := range m.workers {
		if workerMetrics, exists := m.metrics.Workers[workerID]; exists {
			workerMetrics.Mu.Lock()
			workerMetrics.AverageLoad = worker.GetLoad()
			workerMetrics.QueueDepth = worker.GetQueueDepth()
			workerMetrics.Mu.Unlock()
		}
	}

	// Update overall metrics
	m.metrics.Overall.Mu.Lock()
	m.metrics.Overall.Throughput = m.calculateThroughput()
	m.metrics.Overall.AverageLatency = m.calculateAverageLatency()
	m.metrics.Overall.P99Latency = m.calculateP99Latency()
	m.metrics.Overall.Mu.Unlock()
}

// calculateThroughput calculates current throughput
func (m *LoadMonitor) calculateThroughput() float64 {
	elapsed := time.Since(m.metrics.Overall.StartTime).Seconds()
	if elapsed == 0 {
		return 0
	}
	return float64(m.metrics.Overall.TotalTasks) / elapsed
}

// calculateAverageLatency calculates average latency from samples
func (m *LoadMonitor) calculateAverageLatency() time.Duration {
	var total time.Duration
	validSamples := 0

	for _, sample := range m.latencySamples {
		if sample > 0 {
			total += sample
			validSamples++
		}
	}

	if validSamples == 0 {
		return 0
	}

	return total / time.Duration(validSamples)
}

// calculateP99Latency calculates 99th percentile latency
func (m *LoadMonitor) calculateP99Latency() time.Duration {
	// Collect valid samples
	validSamples := make([]time.Duration, 0, m.maxSamples)
	for _, sample := range m.latencySamples {
		if sample > 0 {
			validSamples = append(validSamples, sample)
		}
	}

	if len(validSamples) == 0 {
		return 0
	}

	// Simple bubble sort for small datasets
	for i := 0; i < len(validSamples)-1; i++ {
		for j := 0; j < len(validSamples)-i-1; j++ {
			if validSamples[j] > validSamples[j+1] {
				validSamples[j], validSamples[j+1] = validSamples[j+1], validSamples[j]
			}
		}
	}

	// Calculate 99th percentile index
	p99Index := int(float64(len(validSamples)) * 0.99)
	if p99Index >= len(validSamples) {
		p99Index = len(validSamples) - 1
	}

	return validSamples[p99Index]
}

// GetSystemLoad returns overall system load
func (m *LoadMonitor) GetSystemLoad() float64 {
	m.mu.RLock()
	defer m.mu.RUnlock()

	if len(m.workers) == 0 {
		return 0.0
	}

	totalLoad := 0.0
	for _, worker := range m.workers {
		totalLoad += worker.GetLoad()
	}

	return totalLoad / float64(len(m.workers))
}

// GetLoadDistribution returns load distribution across workers
func (m *LoadMonitor) GetLoadDistribution() map[string]float64 {
	m.mu.RLock()
	defer m.mu.RUnlock()

	loadDist := make(map[string]float64)
	for workerID, worker := range m.workers {
		loadDist[workerID] = worker.GetLoad()
	}

	return loadDist
}

// GetBottlenecks identifies potential bottlenecks
func (m *LoadMonitor) GetBottlenecks(threshold float64) []string {
	m.mu.RLock()
	defer m.mu.RUnlock()

	bottlenecks := make([]string, 0)
	for workerID, worker := range m.workers {
		if worker.GetLoad() > threshold {
			bottlenecks = append(bottlenecks, workerID)
		}
	}

	return bottlenecks
}

// Reset resets all metrics
func (m *LoadMonitor) Reset() {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.metrics = &Metrics{
		Distributors: make(map[string]*DistributorMetrics),
		Workers:      make(map[string]*WorkerMetrics),
		Overall:      &OverallMetrics{StartTime: time.Now()},
	}

	m.latencySamples = make([]time.Duration, m.maxSamples)
	m.sampleIndex = 0
}