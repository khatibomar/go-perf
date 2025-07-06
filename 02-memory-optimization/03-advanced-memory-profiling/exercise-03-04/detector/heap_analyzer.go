package detector

import (
	"encoding/json"
	"fmt"
	"math"
	"os"
	"path/filepath"
	"runtime"
	"sync"
	"time"

	"gonum.org/v1/gonum/stat"
)

// HeapConfig holds configuration for heap analysis
type HeapConfig struct {
	Interval  time.Duration
	Threshold float64 // MB/min growth threshold
	Verbose   bool
}

// HeapMetrics represents heap metrics at a point in time
type HeapMetrics struct {
	Timestamp    time.Time `json:"timestamp"`
	HeapAlloc    uint64    `json:"heap_alloc"`
	HeapSys      uint64    `json:"heap_sys"`
	HeapIdle     uint64    `json:"heap_idle"`
	HeapInuse    uint64    `json:"heap_inuse"`
	HeapReleased uint64    `json:"heap_released"`
	HeapObjects  uint64    `json:"heap_objects"`
	StackInuse   uint64    `json:"stack_inuse"`
	StackSys     uint64    `json:"stack_sys"`
	MSpanInuse   uint64    `json:"mspan_inuse"`
	MSpanSys     uint64    `json:"mspan_sys"`
	MCacheInuse  uint64    `json:"mcache_inuse"`
	MCacheSys    uint64    `json:"mcache_sys"`
	BuckHashSys  uint64    `json:"buck_hash_sys"`
	GCSys        uint64    `json:"gc_sys"`
	OtherSys     uint64    `json:"other_sys"`
	NextGC       uint64    `json:"next_gc"`
	LastGC       uint64    `json:"last_gc"`
	NumGC        uint32    `json:"num_gc"`
	NumForcedGC  uint32    `json:"num_forced_gc"`
	GCCPUFraction float64  `json:"gc_cpu_fraction"`
	TotalAlloc   uint64    `json:"total_alloc"`
	Mallocs      uint64    `json:"mallocs"`
	Frees        uint64    `json:"frees"`
}

// HeapAnalysis represents the result of heap analysis
type HeapAnalysis struct {
	StartTime        time.Time     `json:"start_time"`
	EndTime          time.Time     `json:"end_time"`
	Duration         time.Duration `json:"duration"`
	Metrics          []HeapMetrics `json:"metrics"`
	GrowthRate       float64       `json:"growth_rate_mb_per_min"`
	TrendDirection   string        `json:"trend_direction"`
	LeakProbability  float64       `json:"leak_probability"`
	Severity         string        `json:"severity"`
	Anomalies        []Anomaly     `json:"anomalies"`
	Recommendations  []string      `json:"recommendations"`
	Statistics       HeapStats     `json:"statistics"`
}

// HeapStats contains statistical analysis of heap metrics
type HeapStats struct {
	MeanHeapSize     float64 `json:"mean_heap_size"`
	StdDevHeapSize   float64 `json:"stddev_heap_size"`
	MinHeapSize      uint64  `json:"min_heap_size"`
	MaxHeapSize      uint64  `json:"max_heap_size"`
	GrowthVariance   float64 `json:"growth_variance"`
	CorrelationGCRate float64 `json:"correlation_gc_rate"`
	PeakGrowthPeriod string  `json:"peak_growth_period"`
}

// Anomaly represents an anomalous heap behavior
type Anomaly struct {
	Timestamp   time.Time `json:"timestamp"`
	Type        string    `json:"type"`
	Severity    string    `json:"severity"`
	Description string    `json:"description"`
	Value       float64   `json:"value"`
	Threshold   float64   `json:"threshold"`
}

// HeapAnalyzer monitors and analyzes heap memory usage
type HeapAnalyzer struct {
	config   *HeapConfig
	metrics  []HeapMetrics
	running  bool
	stopChan chan struct{}
	mutex    sync.RWMutex
}

// NewHeapAnalyzer creates a new heap analyzer
func NewHeapAnalyzer(config *HeapConfig) *HeapAnalyzer {
	return &HeapAnalyzer{
		config:   config,
		metrics:  make([]HeapMetrics, 0),
		stopChan: make(chan struct{}),
	}
}

// Start begins heap monitoring
func (ha *HeapAnalyzer) Start() error {
	ha.mutex.Lock()
	defer ha.mutex.Unlock()

	if ha.running {
		return fmt.Errorf("heap analyzer already running")
	}

	ha.running = true
	go ha.monitorLoop()

	if ha.config.Verbose {
		fmt.Printf("Heap analyzer started with interval %v\n", ha.config.Interval)
	}

	return nil
}

// Stop stops heap monitoring
func (ha *HeapAnalyzer) Stop() {
	ha.mutex.Lock()
	defer ha.mutex.Unlock()

	if !ha.running {
		return
	}

	ha.running = false
	close(ha.stopChan)

	if ha.config.Verbose {
		fmt.Println("Heap analyzer stopped")
	}
}

// GetMetrics returns current metrics
func (ha *HeapAnalyzer) GetMetrics() []HeapMetrics {
	ha.mutex.RLock()
	defer ha.mutex.RUnlock()

	// Return a copy to avoid race conditions
	metrics := make([]HeapMetrics, len(ha.metrics))
	copy(metrics, ha.metrics)
	return metrics
}

// AnalyzeHeap performs comprehensive heap analysis
func (ha *HeapAnalyzer) AnalyzeHeap() (*HeapAnalysis, error) {
	ha.mutex.RLock()
	metrics := make([]HeapMetrics, len(ha.metrics))
	copy(metrics, ha.metrics)
	ha.mutex.RUnlock()

	if len(metrics) < 2 {
		return nil, fmt.Errorf("insufficient data for analysis (need at least 2 data points)")
	}

	analysis := &HeapAnalysis{
		StartTime: metrics[0].Timestamp,
		EndTime:   metrics[len(metrics)-1].Timestamp,
		Metrics:   metrics,
	}
	analysis.Duration = analysis.EndTime.Sub(analysis.StartTime)

	// Calculate growth rate
	analysis.GrowthRate = ha.calculateGrowthRate(metrics)
	analysis.TrendDirection = ha.determineTrendDirection(analysis.GrowthRate)

	// Calculate leak probability
	analysis.LeakProbability = ha.calculateLeakProbability(metrics, analysis.GrowthRate)
	analysis.Severity = ha.determineSeverity(analysis.LeakProbability, analysis.GrowthRate)

	// Detect anomalies
	analysis.Anomalies = ha.detectAnomalies(metrics)

	// Generate recommendations
	analysis.Recommendations = ha.generateRecommendations(analysis)

	// Calculate statistics
	analysis.Statistics = ha.calculateStatistics(metrics)

	return analysis, nil
}

// SaveResults saves analysis results to files
func (ha *HeapAnalyzer) SaveResults(outputDir string) error {
	analysis, err := ha.AnalyzeHeap()
	if err != nil {
		return err
	}

	if err := os.MkdirAll(outputDir, 0755); err != nil {
		return err
	}

	// Save detailed analysis
	analysisFile := filepath.Join(outputDir, "heap_analysis.json")
	file, err := os.Create(analysisFile)
	if err != nil {
		return err
	}
	defer file.Close()

	encoder := json.NewEncoder(file)
	encoder.SetIndent("", "  ")
	if err := encoder.Encode(analysis); err != nil {
		return err
	}

	// Save raw metrics
	metricsFile := filepath.Join(outputDir, "heap_metrics.json")
	file, err = os.Create(metricsFile)
	if err != nil {
		return err
	}
	defer file.Close()

	encoder = json.NewEncoder(file)
	encoder.SetIndent("", "  ")
	return encoder.Encode(ha.GetMetrics())
}

// monitorLoop runs the monitoring loop
func (ha *HeapAnalyzer) monitorLoop() {
	ticker := time.NewTicker(ha.config.Interval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			ha.collectMetrics()
		case <-ha.stopChan:
			return
		}
	}
}

// collectMetrics collects current heap metrics
func (ha *HeapAnalyzer) collectMetrics() {
	var m runtime.MemStats
	runtime.ReadMemStats(&m)

	metrics := HeapMetrics{
		Timestamp:     time.Now(),
		HeapAlloc:     m.HeapAlloc,
		HeapSys:       m.HeapSys,
		HeapIdle:      m.HeapIdle,
		HeapInuse:     m.HeapInuse,
		HeapReleased:  m.HeapReleased,
		HeapObjects:   m.HeapObjects,
		StackInuse:    m.StackInuse,
		StackSys:      m.StackSys,
		MSpanInuse:    m.MSpanInuse,
		MSpanSys:      m.MSpanSys,
		MCacheInuse:   m.MCacheInuse,
		MCacheSys:     m.MCacheSys,
		BuckHashSys:   m.BuckHashSys,
		GCSys:         m.GCSys,
		OtherSys:      m.OtherSys,
		NextGC:        m.NextGC,
		LastGC:        m.LastGC,
		NumGC:         m.NumGC,
		NumForcedGC:   m.NumForcedGC,
		GCCPUFraction: m.GCCPUFraction,
		TotalAlloc:    m.TotalAlloc,
		Mallocs:       m.Mallocs,
		Frees:         m.Frees,
	}

	ha.mutex.Lock()
	ha.metrics = append(ha.metrics, metrics)

	// Keep only recent metrics to prevent memory growth
	maxMetrics := 10000
	if len(ha.metrics) > maxMetrics {
		ha.metrics = ha.metrics[len(ha.metrics)-maxMetrics:]
	}
	ha.mutex.Unlock()

	// Check for immediate alerts
	if len(ha.metrics) >= 2 {
		ha.checkForAlerts(metrics)
	}
}

// calculateGrowthRate calculates heap growth rate in MB/min
func (ha *HeapAnalyzer) calculateGrowthRate(metrics []HeapMetrics) float64 {
	if len(metrics) < 2 {
		return 0
	}

	// Use linear regression to calculate growth rate
	x := make([]float64, len(metrics))
	y := make([]float64, len(metrics))

	startTime := metrics[0].Timestamp
	for i, metric := range metrics {
		x[i] = metric.Timestamp.Sub(startTime).Minutes()
		y[i] = float64(metric.HeapAlloc) / 1024 / 1024 // Convert to MB
	}

	// Calculate linear regression slope
	alpha, beta := stat.LinearRegression(x, y, nil, false)
	_ = alpha // Intercept not needed for growth rate

	return beta // Slope is growth rate in MB/min
}

// determineTrendDirection determines the trend direction
func (ha *HeapAnalyzer) determineTrendDirection(growthRate float64) string {
	if growthRate > 0.1 {
		return "increasing"
	} else if growthRate < -0.1 {
		return "decreasing"
	}
	return "stable"
}

// calculateLeakProbability calculates the probability of a memory leak
func (ha *HeapAnalyzer) calculateLeakProbability(metrics []HeapMetrics, growthRate float64) float64 {
	probability := 0.0

	// Factor 1: Growth rate (higher rate = higher probability)
	if growthRate > ha.config.Threshold {
		probability += 0.4 * math.Min(growthRate/ha.config.Threshold, 2.0)
	}

	// Factor 2: Consistency of growth
	consistency := ha.calculateGrowthConsistency(metrics)
	probability += 0.3 * consistency

	// Factor 3: GC effectiveness
	gcEffectiveness := ha.calculateGCEffectiveness(metrics)
	probability += 0.2 * (1.0 - gcEffectiveness)

	// Factor 4: Object count growth
	objectGrowth := ha.calculateObjectGrowthRate(metrics)
	if objectGrowth > 0 {
		probability += 0.1 * math.Min(objectGrowth/1000, 1.0) // Normalize to objects per minute
	}

	return math.Min(probability, 1.0)
}

// calculateGrowthConsistency calculates how consistent the growth pattern is
func (ha *HeapAnalyzer) calculateGrowthConsistency(metrics []HeapMetrics) float64 {
	if len(metrics) < 3 {
		return 0
	}

	// Calculate growth deltas
	deltas := make([]float64, len(metrics)-1)
	for i := 1; i < len(metrics); i++ {
		timeDiff := metrics[i].Timestamp.Sub(metrics[i-1].Timestamp).Minutes()
		heapDiff := float64(metrics[i].HeapAlloc-metrics[i-1].HeapAlloc) / 1024 / 1024
		if timeDiff > 0 {
			deltas[i-1] = heapDiff / timeDiff
		}
	}

	// Calculate coefficient of variation (lower = more consistent)
	mean := stat.Mean(deltas, nil)
	stddev := stat.StdDev(deltas, nil)

	if mean == 0 {
		return 0
	}

	cv := stddev / math.Abs(mean)
	// Convert to consistency score (higher = more consistent)
	return math.Max(0, 1.0-cv/2.0)
}

// calculateGCEffectiveness calculates how effective GC is at reclaiming memory
func (ha *HeapAnalyzer) calculateGCEffectiveness(metrics []HeapMetrics) float64 {
	if len(metrics) < 2 {
		return 1.0 // Assume effective if no data
	}

	first := metrics[0]
	last := metrics[len(metrics)-1]

	gcCount := last.NumGC - first.NumGC
	if gcCount == 0 {
		return 0.5 // No GC activity
	}

	// Calculate memory reclaimed per GC
	totalAllocated := last.TotalAlloc - first.TotalAlloc
	currentHeap := last.HeapAlloc

	if totalAllocated == 0 {
		return 1.0
	}

	// Effectiveness = (allocated - retained) / allocated
	effectiveness := float64(totalAllocated-currentHeap) / float64(totalAllocated)
	return math.Max(0, math.Min(effectiveness, 1.0))
}

// calculateObjectGrowthRate calculates object count growth rate
func (ha *HeapAnalyzer) calculateObjectGrowthRate(metrics []HeapMetrics) float64 {
	if len(metrics) < 2 {
		return 0
	}

	first := metrics[0]
	last := metrics[len(metrics)-1]

	objectDiff := float64(last.HeapObjects) - float64(first.HeapObjects)
	timeDiff := last.Timestamp.Sub(first.Timestamp).Minutes()

	if timeDiff == 0 {
		return 0
	}

	return objectDiff / timeDiff
}

// determineSeverity determines the severity level
func (ha *HeapAnalyzer) determineSeverity(leakProbability, growthRate float64) string {
	if leakProbability > 0.8 || growthRate > ha.config.Threshold*3 {
		return "critical"
	} else if leakProbability > 0.6 || growthRate > ha.config.Threshold*2 {
		return "high"
	} else if leakProbability > 0.4 || growthRate > ha.config.Threshold {
		return "medium"
	}
	return "low"
}

// detectAnomalies detects anomalous heap behavior
func (ha *HeapAnalyzer) detectAnomalies(metrics []HeapMetrics) []Anomaly {
	var anomalies []Anomaly

	if len(metrics) < 10 {
		return anomalies // Need sufficient data for anomaly detection
	}

	// Calculate moving statistics
	windowSize := 10
	for i := windowSize; i < len(metrics); i++ {
		window := metrics[i-windowSize : i]
		current := metrics[i]

		// Calculate window statistics
		heapSizes := make([]float64, len(window))
		for j, m := range window {
			heapSizes[j] = float64(m.HeapAlloc)
		}

		mean := stat.Mean(heapSizes, nil)
		stddev := stat.StdDev(heapSizes, nil)

		// Detect sudden spikes (> 3 standard deviations)
		currentHeap := float64(current.HeapAlloc)
		if stddev > 0 && math.Abs(currentHeap-mean) > 3*stddev {
			anomalyType := "spike"
			if currentHeap < mean {
				anomalyType = "drop"
			}

			anomalies = append(anomalies, Anomaly{
				Timestamp:   current.Timestamp,
				Type:        anomalyType,
				Severity:    ha.classifyAnomalySeverity(math.Abs(currentHeap-mean), stddev),
				Description: fmt.Sprintf("Heap %s detected: %.2f MB (%.1f σ from mean)", anomalyType, currentHeap/1024/1024, math.Abs(currentHeap-mean)/stddev),
				Value:       currentHeap / 1024 / 1024,
				Threshold:   (mean + 3*stddev) / 1024 / 1024,
			})
		}

		// Detect GC anomalies
		if i > 0 {
			prev := metrics[i-1]
			gcDiff := current.NumGC - prev.NumGC
			timeDiff := current.Timestamp.Sub(prev.Timestamp)

			// Detect excessive GC frequency
			if gcDiff > 0 && timeDiff < time.Second {
				anomalies = append(anomalies, Anomaly{
					Timestamp:   current.Timestamp,
					Type:        "excessive_gc",
					Severity:    "high",
					Description: fmt.Sprintf("Excessive GC frequency: %d GCs in %v", gcDiff, timeDiff),
					Value:       float64(gcDiff),
					Threshold:   1.0,
				})
			}
		}
	}

	return anomalies
}

// classifyAnomalySeverity classifies anomaly severity
func (ha *HeapAnalyzer) classifyAnomalySeverity(deviation, stddev float64) string {
	sigmas := deviation / stddev
	if sigmas > 5 {
		return "critical"
	} else if sigmas > 4 {
		return "high"
	} else if sigmas > 3 {
		return "medium"
	}
	return "low"
}

// generateRecommendations generates recommendations based on analysis
func (ha *HeapAnalyzer) generateRecommendations(analysis *HeapAnalysis) []string {
	var recommendations []string

	if analysis.LeakProbability > 0.7 {
		recommendations = append(recommendations, "High probability of memory leak detected. Investigate object lifecycle and ensure proper cleanup.")
	}

	if analysis.GrowthRate > ha.config.Threshold*2 {
		recommendations = append(recommendations, "Rapid heap growth detected. Consider implementing object pooling or reducing allocation rate.")
	}

	if len(analysis.Anomalies) > 5 {
		recommendations = append(recommendations, "Multiple heap anomalies detected. Review recent code changes and monitor GC behavior.")
	}

	if analysis.Statistics.CorrelationGCRate < 0.3 {
		recommendations = append(recommendations, "Poor correlation between heap growth and GC activity. GC may not be effectively reclaiming memory.")
	}

	if analysis.TrendDirection == "increasing" && analysis.GrowthRate > ha.config.Threshold {
		recommendations = append(recommendations, "Implement heap profiling to identify allocation hotspots and optimize memory usage.")
	}

	if len(recommendations) == 0 {
		recommendations = append(recommendations, "Heap usage appears normal. Continue monitoring for any changes in allocation patterns.")
	}

	return recommendations
}

// calculateStatistics calculates comprehensive heap statistics
func (ha *HeapAnalyzer) calculateStatistics(metrics []HeapMetrics) HeapStats {
	if len(metrics) == 0 {
		return HeapStats{}
	}

	// Extract heap sizes
	heapSizes := make([]float64, len(metrics))
	for i, m := range metrics {
		heapSizes[i] = float64(m.HeapAlloc)
	}

	stats := HeapStats{
		MeanHeapSize:   stat.Mean(heapSizes, nil),
		StdDevHeapSize: stat.StdDev(heapSizes, nil),
		MinHeapSize:    metrics[0].HeapAlloc,
		MaxHeapSize:    metrics[0].HeapAlloc,
	}

	// Find min/max
	for _, m := range metrics {
		if m.HeapAlloc < stats.MinHeapSize {
			stats.MinHeapSize = m.HeapAlloc
		}
		if m.HeapAlloc > stats.MaxHeapSize {
			stats.MaxHeapSize = m.HeapAlloc
		}
	}

	// Calculate growth variance
	if len(metrics) > 1 {
		growthRates := make([]float64, len(metrics)-1)
		for i := 1; i < len(metrics); i++ {
			timeDiff := metrics[i].Timestamp.Sub(metrics[i-1].Timestamp).Minutes()
			heapDiff := float64(metrics[i].HeapAlloc - metrics[i-1].HeapAlloc)
			if timeDiff > 0 {
				growthRates[i-1] = heapDiff / timeDiff
			}
		}
		stats.GrowthVariance = stat.Variance(growthRates, nil)
	}

	// Calculate correlation with GC rate
	if len(metrics) > 2 {
		stats.CorrelationGCRate = ha.calculateGCCorrelation(metrics)
	}

	// Find peak growth period
	stats.PeakGrowthPeriod = ha.findPeakGrowthPeriod(metrics)

	return stats
}

// calculateGCCorrelation calculates correlation between heap growth and GC activity
func (ha *HeapAnalyzer) calculateGCCorrelation(metrics []HeapMetrics) float64 {
	if len(metrics) < 3 {
		return 0
	}

	heapGrowth := make([]float64, len(metrics)-1)
	gcActivity := make([]float64, len(metrics)-1)

	for i := 1; i < len(metrics); i++ {
		heapGrowth[i-1] = float64(metrics[i].HeapAlloc - metrics[i-1].HeapAlloc)
		gcActivity[i-1] = float64(metrics[i].NumGC - metrics[i-1].NumGC)
	}

	return stat.Correlation(heapGrowth, gcActivity, nil)
}

// findPeakGrowthPeriod finds the period with highest growth rate
func (ha *HeapAnalyzer) findPeakGrowthPeriod(metrics []HeapMetrics) string {
	if len(metrics) < 2 {
		return "insufficient_data"
	}

	maxGrowthRate := 0.0
	maxGrowthTime := metrics[0].Timestamp

	for i := 1; i < len(metrics); i++ {
		timeDiff := metrics[i].Timestamp.Sub(metrics[i-1].Timestamp).Minutes()
		heapDiff := float64(metrics[i].HeapAlloc - metrics[i-1].HeapAlloc)
		if timeDiff > 0 {
			growthRate := heapDiff / timeDiff
			if growthRate > maxGrowthRate {
				maxGrowthRate = growthRate
				maxGrowthTime = metrics[i].Timestamp
			}
		}
	}

	return maxGrowthTime.Format("2006-01-02 15:04:05")
}

// checkForAlerts checks for immediate alerts
func (ha *HeapAnalyzer) checkForAlerts(current HeapMetrics) {
	ha.mutex.RLock()
	metricsLen := len(ha.metrics)
	ha.mutex.RUnlock()

	if metricsLen < 2 {
		return
	}

	ha.mutex.RLock()
	previous := ha.metrics[metricsLen-2]
	ha.mutex.RUnlock()

	// Check for rapid growth
	timeDiff := current.Timestamp.Sub(previous.Timestamp).Minutes()
	heapDiff := float64(current.HeapAlloc-previous.HeapAlloc) / 1024 / 1024

	if timeDiff > 0 {
		growthRate := heapDiff / timeDiff
		if growthRate > ha.config.Threshold*2 && ha.config.Verbose {
			fmt.Printf("ALERT: Rapid heap growth detected: %.2f MB/min\n", growthRate)
		}
	}

	// Check for excessive GC
	gcDiff := current.NumGC - previous.NumGC
	if gcDiff > 5 && timeDiff < 1 && ha.config.Verbose {
		fmt.Printf("ALERT: Excessive GC activity: %d GCs in %.1f minutes\n", gcDiff, timeDiff)
	}
}