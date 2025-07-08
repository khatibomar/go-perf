package monitoring

import (
	"context"
	"fmt"
	"sync"
	"sync/atomic"
	"time"
)

// MetricType represents different types of metrics
type MetricType int

const (
	CounterType MetricType = iota
	GaugeType
	HistogramType
	TimerType
)

// Metric represents a single metric
type Metric struct {
	Name        string
	Type        MetricType
	Value       int64
	Timestamp   time.Time
	Labels      map[string]string
	Description string
}

// MetricsCollector collects and manages performance metrics
type MetricsCollector struct {
	metrics    map[string]*Metric
	mu         sync.RWMutex
	histograms map[string]*Histogram
	timers     map[string]*Timer
	subscribers []chan Metric
	subMu      sync.RWMutex
}

// NewMetricsCollector creates a new metrics collector
func NewMetricsCollector() *MetricsCollector {
	return &MetricsCollector{
		metrics:    make(map[string]*Metric),
		histograms: make(map[string]*Histogram),
		timers:     make(map[string]*Timer),
	}
}

// Counter operations
func (mc *MetricsCollector) IncrementCounter(name string, labels ...map[string]string) {
	var labelMap map[string]string
	if len(labels) > 0 {
		labelMap = labels[0]
	}
	mc.AddToCounter(name, 1, labelMap)
}

func (mc *MetricsCollector) AddToCounter(name string, value int64, labels ...map[string]string) {
	var labelMap map[string]string
	if len(labels) > 0 {
		labelMap = labels[0]
	}
	mc.addToCounterWithLabels(name, value, labelMap)
}

func (mc *MetricsCollector) addToCounterWithLabels(name string, value int64, labels map[string]string) {
	mc.mu.Lock()
	defer mc.mu.Unlock()
	
	key := mc.buildKey(name, labels)
	if metric, exists := mc.metrics[key]; exists {
		atomic.AddInt64(&metric.Value, value)
		metric.Timestamp = time.Now()
	} else {
		mc.metrics[key] = &Metric{
			Name:      name,
			Type:      CounterType,
			Value:     value,
			Timestamp: time.Now(),
			Labels:    labels,
		}
	}
	
	mc.notifySubscribers(*mc.metrics[key])
}

// Gauge operations
func (mc *MetricsCollector) SetGauge(name string, value float64, labels ...map[string]string) {
	var labelMap map[string]string
	if len(labels) > 0 {
		labelMap = labels[0]
	}
	mc.setGaugeWithLabels(name, int64(value), labelMap)
}

func (mc *MetricsCollector) setGaugeWithLabels(name string, value int64, labels map[string]string) {
	mc.mu.Lock()
	defer mc.mu.Unlock()
	
	key := mc.buildKey(name, labels)
	mc.metrics[key] = &Metric{
		Name:      name,
		Type:      GaugeType,
		Value:     value,
		Timestamp: time.Now(),
		Labels:    labels,
	}
	
	mc.notifySubscribers(*mc.metrics[key])
}

// Histogram operations
func (mc *MetricsCollector) RecordHistogram(name string, value float64, labels ...map[string]string) {
	var labelMap map[string]string
	if len(labels) > 0 {
		labelMap = labels[0]
	}
	mc.recordHistogramWithLabels(name, value, labelMap)
}

func (mc *MetricsCollector) recordHistogramWithLabels(name string, value float64, labels map[string]string) {
	mc.mu.Lock()
	key := mc.buildKey(name, labels)
	if _, exists := mc.histograms[key]; !exists {
		mc.histograms[key] = NewHistogram()
	}
	hist := mc.histograms[key]
	mc.mu.Unlock()
	
	hist.Record(value)
	
	// Update metric with current histogram stats
	stats := hist.Stats()
	metric := Metric{
		Name:      name,
		Type:      HistogramType,
		Value:     int64(stats.Count),
		Timestamp: time.Now(),
		Labels:    labels,
	}
	
	mc.mu.Lock()
	mc.metrics[key] = &metric
	mc.mu.Unlock()
	
	mc.notifySubscribers(metric)
}

// Timer operations
func (mc *MetricsCollector) StartTimer(name string, labels ...map[string]string) *TimerInstance {
	var labelMap map[string]string
	if len(labels) > 0 {
		labelMap = labels[0]
	}
	return mc.startTimerWithLabels(name, labelMap)
}

func (mc *MetricsCollector) startTimerWithLabels(name string, labels map[string]string) *TimerInstance {
	mc.mu.Lock()
	key := mc.buildKey(name, labels)
	if _, exists := mc.timers[key]; !exists {
		mc.timers[key] = NewTimer()
	}
	timer := mc.timers[key]
	mc.mu.Unlock()
	
	return timer.Start()
}

func (mc *MetricsCollector) RecordTimer(name string, duration time.Duration, labels ...map[string]string) {
	var labelMap map[string]string
	if len(labels) > 0 {
		labelMap = labels[0]
	}
	mc.recordTimerWithLabels(name, duration, labelMap)
}

func (mc *MetricsCollector) recordTimerWithLabels(name string, duration time.Duration, labels map[string]string) {
	mc.mu.Lock()
	key := mc.buildKey(name, labels)
	if _, exists := mc.timers[key]; !exists {
		mc.timers[key] = NewTimer()
	}
	timer := mc.timers[key]
	mc.mu.Unlock()
	
	timer.Record(duration)
	
	// Update metric with current timer stats
	stats := timer.Stats()
	metric := Metric{
		Name:      name,
		Type:      TimerType,
		Value:     int64(stats.Count),
		Timestamp: time.Now(),
		Labels:    labels,
	}
	
	mc.mu.Lock()
	mc.metrics[key] = &metric
	mc.mu.Unlock()
	
	mc.notifySubscribers(metric)
}

// buildKey creates a unique key for a metric with labels
func (mc *MetricsCollector) buildKey(name string, labels map[string]string) string {
	key := name
	for k, v := range labels {
		key += fmt.Sprintf(",%s=%s", k, v)
	}
	return key
}

// notifySubscribers notifies all subscribers of a metric update
func (mc *MetricsCollector) notifySubscribers(metric Metric) {
	mc.subMu.RLock()
	defer mc.subMu.RUnlock()
	
	for _, ch := range mc.subscribers {
		select {
		case ch <- metric:
		default:
			// Channel is full, skip this subscriber
		}
	}
}

// Subscribe returns a channel that receives metric updates
func (mc *MetricsCollector) Subscribe() <-chan Metric {
	mc.subMu.Lock()
	defer mc.subMu.Unlock()
	
	ch := make(chan Metric, 100) // Buffered channel
	mc.subscribers = append(mc.subscribers, ch)
	return ch
}

// GetCounter returns the value of a counter metric
func (mc *MetricsCollector) GetCounter(name string, labels ...map[string]string) int64 {
	var labelMap map[string]string
	if len(labels) > 0 {
		labelMap = labels[0]
	}
	
	mc.mu.RLock()
	defer mc.mu.RUnlock()
	
	key := mc.buildKey(name, labelMap)
	if metric, exists := mc.metrics[key]; exists && metric.Type == CounterType {
		return metric.Value
	}
	return 0
}

// GetGauge returns the value of a gauge metric
func (mc *MetricsCollector) GetGauge(name string, labels ...map[string]string) float64 {
	var labelMap map[string]string
	if len(labels) > 0 {
		labelMap = labels[0]
	}
	
	mc.mu.RLock()
	defer mc.mu.RUnlock()
	
	key := mc.buildKey(name, labelMap)
	if metric, exists := mc.metrics[key]; exists && metric.Type == GaugeType {
		return float64(metric.Value)
	}
	return 0.0
}

// GetHistogramStats returns histogram statistics with optional labels
func (mc *MetricsCollector) GetHistogramStats(name string, labels ...map[string]string) *HistogramStats {
	var labelMap map[string]string
	if len(labels) > 0 {
		labelMap = labels[0]
	}
	
	mc.mu.RLock()
	defer mc.mu.RUnlock()
	
	key := mc.buildKey(name, labelMap)
	if hist, exists := mc.histograms[key]; exists {
		stats := hist.Stats()
		return &stats
	}
	return nil
}

// GetTimerStats returns timer statistics with optional labels
func (mc *MetricsCollector) GetTimerStats(name string, labels ...map[string]string) *TimerStats {
	var labelMap map[string]string
	if len(labels) > 0 {
		labelMap = labels[0]
	}
	
	mc.mu.RLock()
	defer mc.mu.RUnlock()
	
	key := mc.buildKey(name, labelMap)
	if timer, exists := mc.timers[key]; exists {
		stats := timer.Stats()
		return &stats
	}
	return nil
}

// GetMetrics returns all current metrics
func (mc *MetricsCollector) GetMetrics() map[string]Metric {
	mc.mu.RLock()
	defer mc.mu.RUnlock()
	
	result := make(map[string]Metric)
	for key, metric := range mc.metrics {
		result[key] = *metric
	}
	return result
}



// Reset clears all metrics
func (mc *MetricsCollector) Reset() {
	mc.mu.Lock()
	defer mc.mu.Unlock()
	
	mc.metrics = make(map[string]*Metric)
	mc.histograms = make(map[string]*Histogram)
	mc.timers = make(map[string]*Timer)
}

// QueueReport contains queue monitoring statistics
type QueueReport struct {
	TotalEnqueues      int64
	SuccessfulEnqueues int64
	FailedEnqueues     int64
	TotalDequeues      int64
	SuccessfulDequeues int64
	FailedDequeues     int64
	CurrentQueueSize   int64
	MaxQueueSize       int64
	ContentionEvents   int64
	AverageEnqueueTime time.Duration
	AverageDequeueTime time.Duration
}

// QueueMonitor monitors queue-specific metrics
type QueueMonitor struct {
	collector        *MetricsCollector
	queueName        string
	ctx              context.Context
	cancel           context.CancelFunc
	wg               sync.WaitGroup
	mu               sync.RWMutex
	totalEnqueues    int64
	successEnqueues  int64
	failedEnqueues   int64
	totalDequeues    int64
	successDequeues  int64
	failedDequeues   int64
	currentQueueSize int64
	maxQueueSize     int64
	contentionEvents int64
	enqueueTimes     []time.Duration
	dequeueTimes     []time.Duration
}

// NewQueueMonitor creates a new queue monitor
func NewQueueMonitor(queueName ...string) *QueueMonitor {
	ctx, cancel := context.WithCancel(context.Background())
	name := "default"
	if len(queueName) > 0 {
		name = queueName[0]
	}
	return &QueueMonitor{
		collector:        NewMetricsCollector(),
		queueName:        name,
		ctx:              ctx,
		cancel:           cancel,
		enqueueTimes:     make([]time.Duration, 0),
		dequeueTimes:     make([]time.Duration, 0),
	}
}

// Queue operation tracking
func (qm *QueueMonitor) RecordEnqueue(duration time.Duration, success bool) {
	qm.mu.Lock()
	defer qm.mu.Unlock()
	
	qm.totalEnqueues++
	if success {
		qm.successEnqueues++
	} else {
		qm.failedEnqueues++
	}
	qm.enqueueTimes = append(qm.enqueueTimes, duration)
	
	labels := map[string]string{
		"queue":     qm.queueName,
		"operation": "enqueue",
		"success":   fmt.Sprintf("%t", success),
	}
	
	qm.collector.IncrementCounter("queue_operations_total", labels)
	qm.collector.RecordTimer("queue_operation_duration", duration, labels)
	
	if success {
		qm.collector.IncrementCounter("queue_enqueues_total", map[string]string{"queue": qm.queueName})
	} else {
		qm.collector.IncrementCounter("queue_enqueue_failures_total", map[string]string{"queue": qm.queueName})
	}
}

func (qm *QueueMonitor) RecordDequeue(duration time.Duration, success bool) {
	qm.mu.Lock()
	defer qm.mu.Unlock()
	
	qm.totalDequeues++
	if success {
		qm.successDequeues++
	} else {
		qm.failedDequeues++
	}
	qm.dequeueTimes = append(qm.dequeueTimes, duration)
	
	labels := map[string]string{
		"queue":     qm.queueName,
		"operation": "dequeue",
		"success":   fmt.Sprintf("%t", success),
	}
	
	qm.collector.IncrementCounter("queue_operations_total", labels)
	qm.collector.RecordTimer("queue_operation_duration", duration, labels)
	
	if success {
		qm.collector.IncrementCounter("queue_dequeues_total", map[string]string{"queue": qm.queueName})
	} else {
		qm.collector.IncrementCounter("queue_dequeue_failures_total", map[string]string{"queue": qm.queueName})
	}
}

func (qm *QueueMonitor) RecordQueueSize(size int64) {
	qm.mu.Lock()
	defer qm.mu.Unlock()
	
	qm.currentQueueSize = size
	if size > qm.maxQueueSize {
		qm.maxQueueSize = size
	}
	
	qm.collector.SetGauge("queue_size", float64(size), map[string]string{"queue": qm.queueName})
}

func (qm *QueueMonitor) RecordContention(operation string, duration time.Duration) {
	qm.mu.Lock()
	defer qm.mu.Unlock()
	
	qm.contentionEvents++
	
	qm.collector.AddToCounter("queue_contention_events_total", 1, map[string]string{
		"queue":     qm.queueName,
		"operation": operation,
	})
	qm.collector.RecordHistogram("queue_contention_duration", float64(duration.Nanoseconds()), map[string]string{
		"queue":     qm.queueName,
		"operation": operation,
	})
}

// GenerateReport generates a comprehensive queue report
func (qm *QueueMonitor) GenerateReport() QueueReport {
	qm.mu.RLock()
	defer qm.mu.RUnlock()
	
	// Calculate average times
	var avgEnqueueTime, avgDequeueTime time.Duration
	if len(qm.enqueueTimes) > 0 {
		var total time.Duration
		for _, t := range qm.enqueueTimes {
			total += t
		}
		avgEnqueueTime = total / time.Duration(len(qm.enqueueTimes))
	}
	if len(qm.dequeueTimes) > 0 {
		var total time.Duration
		for _, t := range qm.dequeueTimes {
			total += t
		}
		avgDequeueTime = total / time.Duration(len(qm.dequeueTimes))
	}
	
	return QueueReport{
		TotalEnqueues:      qm.totalEnqueues,
		SuccessfulEnqueues: qm.successEnqueues,
		FailedEnqueues:     qm.failedEnqueues,
		TotalDequeues:      qm.totalDequeues,
		SuccessfulDequeues: qm.successDequeues,
		FailedDequeues:     qm.failedDequeues,
		CurrentQueueSize:   qm.currentQueueSize,
		MaxQueueSize:       qm.maxQueueSize,
		ContentionEvents:   qm.contentionEvents,
		AverageEnqueueTime: avgEnqueueTime,
		AverageDequeueTime: avgDequeueTime,
	}
}

// StartPeriodicReporting starts periodic reporting of metrics
func (qm *QueueMonitor) StartPeriodicReporting(interval time.Duration, reporter func(QueueReport)) func() {
	qm.wg.Add(1)
	go func() {
		defer qm.wg.Done()
		ticker := time.NewTicker(interval)
		defer ticker.Stop()
		
		for {
			select {
			case <-qm.ctx.Done():
				return
			case <-ticker.C:
				report := qm.GenerateReport()
				reporter(report)
			}
		}
	}()
	
	// Return stop function
	return func() {
		qm.cancel()
		qm.wg.Wait()
	}
}

// Stop stops the queue monitor
func (qm *QueueMonitor) Stop() {
	qm.cancel()
	qm.wg.Wait()
}

// GetCollector returns the underlying metrics collector
func (qm *QueueMonitor) GetCollector() *MetricsCollector {
	return qm.collector
}

// GetHistograms returns all histograms (for internal use)
func (mc *MetricsCollector) GetHistograms() map[string]*Histogram {
	mc.mu.RLock()
	defer mc.mu.RUnlock()
	
	result := make(map[string]*Histogram)
	for key, hist := range mc.histograms {
		result[key] = hist
	}
	return result
}

// GetTimers returns all timers (for internal use)
func (mc *MetricsCollector) GetTimers() map[string]*Timer {
	mc.mu.RLock()
	defer mc.mu.RUnlock()
	
	result := make(map[string]*Timer)
	for key, timer := range mc.timers {
		result[key] = timer
	}
	return result
}

// GenerateStringReport generates a comprehensive metrics report as string
func (qm *QueueMonitor) GenerateStringReport() string {
	metrics := qm.collector.GetMetrics()
	
	report := fmt.Sprintf("Queue Metrics Report for %s:\n", qm.queueName)
	report += "========================================\n"
	
	for name, metric := range metrics {
		report += fmt.Sprintf("%-30s: %d (Type: %v, Time: %v)\n",
			name, metric.Value, metric.Type, metric.Timestamp.Format("15:04:05"))
	}
	
	// Add histogram stats if available
	histograms := qm.collector.GetHistograms()
	for name, hist := range histograms {
		stats := hist.Stats()
		report += fmt.Sprintf("\nHistogram %s:\n", name)
		report += fmt.Sprintf("  Count: %d, Mean: %.2f, P95: %.2f, P99: %.2f\n",
			stats.Count, stats.Mean, stats.P95, stats.P99)
	}
	
	// Add timer stats if available
	timers := qm.collector.GetTimers()
	for name, timer := range timers {
		stats := timer.Stats()
		report += fmt.Sprintf("\nTimer %s:\n", name)
		report += fmt.Sprintf("  Count: %d, Mean: %v, P95: %v, P99: %v\n",
			stats.Count, stats.Mean, stats.P95, stats.P99)
	}
	
	return report
}