package metrics

import (
	"sync"
	"time"
)

// PoolMetrics tracks comprehensive pool performance metrics
type PoolMetrics struct {
	mu                    sync.RWMutex
	StartTime             time.Time
	TasksSubmitted        int64
	TasksCompleted        int64
	TasksFailed           int64
	TasksInQueue          int64
	TasksInProgress       int64
	ActiveWorkers         int64
	IdleWorkers           int64
	TotalWorkers          int64
	PeakWorkers           int64
	WorkerUtilization     float64
	AverageQueueTime      time.Duration
	AverageExecutionTime  time.Duration
	TotalQueueTime        time.Duration
	TotalExecutionTime    time.Duration
	Throughput            float64 // tasks per second
	LatencyP50            time.Duration
	LatencyP95            time.Duration
	LatencyP99            time.Duration
	QueueUtilization      float64
	MemoryUsage           int64
	GoroutineCount        int64
	WorkStealingAttempts  int64
	WorkStealingSuccesses int64
	ScalingEvents         int64
	LastScalingEvent      time.Time
	ErrorRate             float64
	CustomMetrics         map[string]interface{}
}

// WorkerMetrics tracks individual worker performance
type WorkerMetrics struct {
	mu                   sync.RWMutex
	WorkerID             string
	StartTime            time.Time
	TasksProcessed       int64
	TasksFailed          int64
	TotalExecutionTime   time.Duration
	AverageExecutionTime time.Duration
	IdleTime             time.Duration
	BusyTime             time.Duration
	Utilization          float64
	LastActivity         time.Time
	WorkSteals           int64
	WorkStolen           int64
	QueueSize            int64
	LocalQueueHits       int64
	GlobalQueueHits      int64
	CPUUsage             float64
	MemoryUsage          int64
	ErrorRate            float64
}

// TaskMetrics tracks individual task performance
type TaskMetrics struct {
	TaskID        string
	SubmitTime    time.Time
	StartTime     time.Time
	EndTime       time.Time
	QueueTime     time.Duration
	ExecutionTime time.Duration
	TotalTime     time.Duration
	WorkerID      string
	Priority      int
	Retries       int
	Success       bool
	Error         error
	MemoryUsed    int64
	CPUTime       time.Duration
}

// LatencyTracker tracks latency percentiles
type LatencyTracker struct {
	mu        sync.RWMutex
	latencies []time.Duration
	maxSize   int
}

// MetricsCollector aggregates and analyzes metrics
type MetricsCollector struct {
	mu              sync.RWMutex
	poolMetrics     *PoolMetrics
	workerMetrics   map[string]*WorkerMetrics
	taskMetrics     []*TaskMetrics
	latencyTracker  *LatencyTracker
	collectionStart time.Time
	updateInterval  time.Duration
	stopChan        chan struct{}
	callbacks       []MetricsCallback
}

// MetricsCallback is called when metrics are updated
type MetricsCallback func(*PoolMetrics, map[string]*WorkerMetrics)

// NewPoolMetrics creates a new pool metrics instance
func NewPoolMetrics() *PoolMetrics {
	return &PoolMetrics{
		StartTime:     time.Now(),
		CustomMetrics: make(map[string]interface{}),
	}
}

// NewWorkerMetrics creates a new worker metrics instance
func NewWorkerMetrics(workerID string) *WorkerMetrics {
	return &WorkerMetrics{
		WorkerID:     workerID,
		StartTime:    time.Now(),
		LastActivity: time.Now(),
	}
}

// NewLatencyTracker creates a new latency tracker
func NewLatencyTracker(maxSize int) *LatencyTracker {
	return &LatencyTracker{
		latencies: make([]time.Duration, 0, maxSize),
		maxSize:   maxSize,
	}
}

// NewMetricsCollector creates a new metrics collector
func NewMetricsCollector(updateInterval time.Duration) *MetricsCollector {
	return &MetricsCollector{
		poolMetrics:     NewPoolMetrics(),
		workerMetrics:   make(map[string]*WorkerMetrics),
		taskMetrics:     make([]*TaskMetrics, 0),
		latencyTracker:  NewLatencyTracker(10000),
		collectionStart: time.Now(),
		updateInterval:  updateInterval,
		stopChan:        make(chan struct{}),
		callbacks:       make([]MetricsCallback, 0),
	}
}

// Start begins metrics collection
func (mc *MetricsCollector) Start() {
	go mc.collectMetrics()
}

// Stop stops metrics collection
func (mc *MetricsCollector) Stop() {
	close(mc.stopChan)
}

// collectMetrics runs the metrics collection loop
func (mc *MetricsCollector) collectMetrics() {
	ticker := time.NewTicker(mc.updateInterval)
	defer ticker.Stop()
	
	for {
		select {
		case <-ticker.C:
			mc.updateMetrics()
		case <-mc.stopChan:
			return
		}
	}
}

// updateMetrics calculates and updates derived metrics
func (mc *MetricsCollector) updateMetrics() {
	mc.mu.Lock()
	defer mc.mu.Unlock()
	
	// Update pool metrics
	mc.updatePoolMetrics()
	
	// Update worker metrics
	for _, worker := range mc.workerMetrics {
		mc.updateWorkerMetrics(worker)
	}
	
	// Update latency percentiles
	mc.updateLatencyPercentiles()
	
	// Call callbacks
	for _, callback := range mc.callbacks {
		callback(mc.poolMetrics, mc.workerMetrics)
	}
}

// updatePoolMetrics updates pool-level metrics
func (mc *MetricsCollector) updatePoolMetrics() {
	pm := mc.poolMetrics
	
	// Calculate throughput (tasks per second)
	elapsed := time.Since(pm.StartTime).Seconds()
	if elapsed > 0 {
		pm.Throughput = float64(pm.TasksCompleted) / elapsed
	}
	
	// Calculate worker utilization
	if pm.TotalWorkers > 0 {
		pm.WorkerUtilization = float64(pm.ActiveWorkers) / float64(pm.TotalWorkers)
	}
	
	// Calculate error rate
	totalTasks := pm.TasksCompleted + pm.TasksFailed
	if totalTasks > 0 {
		pm.ErrorRate = float64(pm.TasksFailed) / float64(totalTasks)
	}
	
	// Calculate average times
	if pm.TasksCompleted > 0 {
		pm.AverageQueueTime = pm.TotalQueueTime / time.Duration(pm.TasksCompleted)
		pm.AverageExecutionTime = pm.TotalExecutionTime / time.Duration(pm.TasksCompleted)
	}
	
	// Calculate work stealing success rate
	if pm.WorkStealingAttempts > 0 {
		successRate := float64(pm.WorkStealingSuccesses) / float64(pm.WorkStealingAttempts)
		pm.CustomMetrics["work_stealing_success_rate"] = successRate
	}
}

// updateWorkerMetrics updates worker-level metrics
func (mc *MetricsCollector) updateWorkerMetrics(wm *WorkerMetrics) {
	wm.mu.Lock()
	defer wm.mu.Unlock()
	
	// Calculate utilization
	totalTime := time.Since(wm.StartTime)
	if totalTime > 0 {
		wm.Utilization = float64(wm.BusyTime) / float64(totalTime)
	}
	
	// Calculate average execution time
	if wm.TasksProcessed > 0 {
		wm.AverageExecutionTime = wm.TotalExecutionTime / time.Duration(wm.TasksProcessed)
	}
	
	// Calculate error rate
	totalTasks := wm.TasksProcessed + wm.TasksFailed
	if totalTasks > 0 {
		wm.ErrorRate = float64(wm.TasksFailed) / float64(totalTasks)
	}
}

// updateLatencyPercentiles calculates latency percentiles
func (mc *MetricsCollector) updateLatencyPercentiles() {
	mc.latencyTracker.mu.RLock()
	latencies := make([]time.Duration, len(mc.latencyTracker.latencies))
	copy(latencies, mc.latencyTracker.latencies)
	mc.latencyTracker.mu.RUnlock()
	
	if len(latencies) == 0 {
		return
	}
	
	// Sort latencies for percentile calculation
	for i := 0; i < len(latencies)-1; i++ {
		for j := i + 1; j < len(latencies); j++ {
			if latencies[i] > latencies[j] {
				latencies[i], latencies[j] = latencies[j], latencies[i]
			}
		}
	}
	
	// Calculate percentiles
	mc.poolMetrics.LatencyP50 = percentile(latencies, 0.50)
	mc.poolMetrics.LatencyP95 = percentile(latencies, 0.95)
	mc.poolMetrics.LatencyP99 = percentile(latencies, 0.99)
}

// percentile calculates the given percentile from sorted latencies
func percentile(sortedLatencies []time.Duration, p float64) time.Duration {
	if len(sortedLatencies) == 0 {
		return 0
	}
	
	index := int(float64(len(sortedLatencies)-1) * p)
	if index >= len(sortedLatencies) {
		index = len(sortedLatencies) - 1
	}
	
	return sortedLatencies[index]
}

// RecordTaskSubmitted records a task submission
func (mc *MetricsCollector) RecordTaskSubmitted() {
	mc.poolMetrics.mu.Lock()
	mc.poolMetrics.TasksSubmitted++
	mc.poolMetrics.TasksInQueue++
	mc.poolMetrics.mu.Unlock()
}

// RecordTaskStarted records a task start
func (mc *MetricsCollector) RecordTaskStarted(workerID string) {
	mc.poolMetrics.mu.Lock()
	mc.poolMetrics.TasksInQueue--
	mc.poolMetrics.TasksInProgress++
	mc.poolMetrics.mu.Unlock()
	
	if wm, exists := mc.workerMetrics[workerID]; exists {
		wm.mu.Lock()
		wm.LastActivity = time.Now()
		wm.mu.Unlock()
	}
}

// RecordTaskCompleted records a task completion
func (mc *MetricsCollector) RecordTaskCompleted(workerID string, queueTime, executionTime time.Duration) {
	mc.poolMetrics.mu.Lock()
	mc.poolMetrics.TasksCompleted++
	mc.poolMetrics.TasksInProgress--
	mc.poolMetrics.TotalQueueTime += queueTime
	mc.poolMetrics.TotalExecutionTime += executionTime
	mc.poolMetrics.mu.Unlock()
	
	// Record latency
	mc.latencyTracker.AddLatency(queueTime + executionTime)
	
	// Update worker metrics
	if wm, exists := mc.workerMetrics[workerID]; exists {
		wm.mu.Lock()
		wm.TasksProcessed++
		wm.TotalExecutionTime += executionTime
		wm.BusyTime += executionTime
		wm.LastActivity = time.Now()
		wm.mu.Unlock()
	}
}

// RecordTaskFailed records a task failure
func (mc *MetricsCollector) RecordTaskFailed(workerID string, err error) {
	mc.poolMetrics.mu.Lock()
	mc.poolMetrics.TasksFailed++
	mc.poolMetrics.TasksInProgress--
	mc.poolMetrics.mu.Unlock()
	
	if wm, exists := mc.workerMetrics[workerID]; exists {
		wm.mu.Lock()
		wm.TasksFailed++
		wm.LastActivity = time.Now()
		wm.mu.Unlock()
	}
}

// RecordWorkerStarted records a worker start
func (mc *MetricsCollector) RecordWorkerStarted(workerID string) {
	mc.mu.Lock()
	mc.workerMetrics[workerID] = NewWorkerMetrics(workerID)
	mc.mu.Unlock()
	
	mc.poolMetrics.mu.Lock()
	mc.poolMetrics.TotalWorkers++
	mc.poolMetrics.ActiveWorkers++
	if mc.poolMetrics.ActiveWorkers > mc.poolMetrics.PeakWorkers {
		mc.poolMetrics.PeakWorkers = mc.poolMetrics.ActiveWorkers
	}
	mc.poolMetrics.mu.Unlock()
}

// RecordWorkerStopped records a worker stop
func (mc *MetricsCollector) RecordWorkerStopped(workerID string) {
	mc.poolMetrics.mu.Lock()
	mc.poolMetrics.ActiveWorkers--
	mc.poolMetrics.mu.Unlock()
}

// RecordWorkSteal records a work stealing attempt
func (mc *MetricsCollector) RecordWorkSteal(fromWorkerID, toWorkerID string, success bool) {
	mc.poolMetrics.mu.Lock()
	mc.poolMetrics.WorkStealingAttempts++
	if success {
		mc.poolMetrics.WorkStealingSuccesses++
	}
	mc.poolMetrics.mu.Unlock()
	
	// Update worker metrics
	if wm, exists := mc.workerMetrics[toWorkerID]; exists {
		wm.mu.Lock()
		wm.WorkSteals++
		wm.mu.Unlock()
	}
	
	if wm, exists := mc.workerMetrics[fromWorkerID]; exists {
		wm.mu.Lock()
		wm.WorkStolen++
		wm.mu.Unlock()
	}
}

// RecordScalingEvent records a pool scaling event
func (mc *MetricsCollector) RecordScalingEvent() {
	mc.poolMetrics.mu.Lock()
	mc.poolMetrics.ScalingEvents++
	mc.poolMetrics.LastScalingEvent = time.Now()
	mc.poolMetrics.mu.Unlock()
}

// AddLatency adds a latency measurement
func (lt *LatencyTracker) AddLatency(latency time.Duration) {
	lt.mu.Lock()
	defer lt.mu.Unlock()
	
	if len(lt.latencies) >= lt.maxSize {
		// Remove oldest entry
		lt.latencies = lt.latencies[1:]
	}
	
	lt.latencies = append(lt.latencies, latency)
}

// GetPoolMetrics returns a copy of pool metrics
func (mc *MetricsCollector) GetPoolMetrics() PoolMetrics {
	mc.poolMetrics.mu.RLock()
	defer mc.poolMetrics.mu.RUnlock()
	return *mc.poolMetrics
}

// GetWorkerMetrics returns a copy of worker metrics
func (mc *MetricsCollector) GetWorkerMetrics(workerID string) (WorkerMetrics, bool) {
	mc.mu.RLock()
	wm, exists := mc.workerMetrics[workerID]
	mc.mu.RUnlock()
	
	if !exists {
		return WorkerMetrics{}, false
	}
	
	wm.mu.RLock()
	defer wm.mu.RUnlock()
	return *wm, true
}

// GetAllWorkerMetrics returns copies of all worker metrics
func (mc *MetricsCollector) GetAllWorkerMetrics() map[string]WorkerMetrics {
	mc.mu.RLock()
	defer mc.mu.RUnlock()
	
	result := make(map[string]WorkerMetrics)
	for id, wm := range mc.workerMetrics {
		wm.mu.RLock()
		result[id] = *wm
		wm.mu.RUnlock()
	}
	
	return result
}

// AddCallback adds a metrics callback
func (mc *MetricsCollector) AddCallback(callback MetricsCallback) {
	mc.mu.Lock()
	mc.callbacks = append(mc.callbacks, callback)
	mc.mu.Unlock()
}

// SetCustomMetric sets a custom metric value
func (mc *MetricsCollector) SetCustomMetric(key string, value interface{}) {
	mc.poolMetrics.mu.Lock()
	mc.poolMetrics.CustomMetrics[key] = value
	mc.poolMetrics.mu.Unlock()
}

// GetCustomMetric gets a custom metric value
func (mc *MetricsCollector) GetCustomMetric(key string) (interface{}, bool) {
	mc.poolMetrics.mu.RLock()
	value, exists := mc.poolMetrics.CustomMetrics[key]
	mc.poolMetrics.mu.RUnlock()
	return value, exists
}

// Reset resets all metrics
func (mc *MetricsCollector) Reset() {
	mc.mu.Lock()
	defer mc.mu.Unlock()
	
	mc.poolMetrics = NewPoolMetrics()
	mc.workerMetrics = make(map[string]*WorkerMetrics)
	mc.taskMetrics = make([]*TaskMetrics, 0)
	mc.latencyTracker = NewLatencyTracker(mc.latencyTracker.maxSize)
	mc.collectionStart = time.Now()
}

// GetSummary returns a summary of key metrics
func (mc *MetricsCollector) GetSummary() MetricsSummary {
	pm := mc.GetPoolMetrics()
	
	return MetricsSummary{
		Uptime:               time.Since(pm.StartTime),
		TotalTasks:           pm.TasksSubmitted,
		CompletedTasks:       pm.TasksCompleted,
		FailedTasks:          pm.TasksFailed,
		ActiveWorkers:        pm.ActiveWorkers,
		TotalWorkers:         pm.TotalWorkers,
		Throughput:           pm.Throughput,
		AverageLatency:       pm.AverageExecutionTime,
		P95Latency:           pm.LatencyP95,
		ErrorRate:            pm.ErrorRate,
		WorkerUtilization:    pm.WorkerUtilization,
		WorkStealingSuccess:  float64(pm.WorkStealingSuccesses) / float64(pm.WorkStealingAttempts),
	}
}

// MetricsSummary provides a high-level overview of pool performance
type MetricsSummary struct {
	Uptime              time.Duration
	TotalTasks          int64
	CompletedTasks      int64
	FailedTasks         int64
	ActiveWorkers       int64
	TotalWorkers        int64
	Throughput          float64
	AverageLatency      time.Duration
	P95Latency          time.Duration
	ErrorRate           float64
	WorkerUtilization   float64
	WorkStealingSuccess float64
}