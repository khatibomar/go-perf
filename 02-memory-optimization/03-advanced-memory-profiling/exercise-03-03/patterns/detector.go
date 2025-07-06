package patterns

import (
	"encoding/json"
	"fmt"
	"math"
	"os"
	"time"

	"github.com/go-perf/02-memory-optimization/03-advanced-memory-profiling/exercise-03-03/analyzer"
	"gonum.org/v1/gonum/stat"
)

// Helper functions for min/max since gonum/stat doesn't provide them
func minFloat64(data []float64) float64 {
	if len(data) == 0 {
		return 0
	}
	min := data[0]
	for _, v := range data[1:] {
		if v < min {
			min = v
		}
	}
	return min
}

func maxFloat64(data []float64) float64 {
	if len(data) == 0 {
		return 0
	}
	max := data[0]
	for _, v := range data[1:] {
		if v > max {
			max = v
		}
	}
	return max
}

// Config holds configuration for pattern detection
type Config struct {
	WindowSize       int
	AnomalyThreshold float64
	MinPatternLength int
	ConfidenceLevel  float64
}

// PatternType represents different types of allocation patterns
type PatternType string

const (
	PatternBurst   PatternType = "burst"
	PatternGrowth  PatternType = "growth"
	PatternCyclic  PatternType = "cyclic"
	PatternSteady  PatternType = "steady"
	PatternDecline PatternType = "decline"
	PatternSpike   PatternType = "spike"
	PatternAnomaly PatternType = "anomaly"
	PatternUnknown PatternType = "unknown"
)

// DetectedPattern represents a detected allocation pattern
type DetectedPattern struct {
	Type        PatternType    `json:"type"`
	Confidence  float64        `json:"confidence"`
	StartTime   time.Time      `json:"start_time"`
	EndTime     time.Time      `json:"end_time"`
	Duration    time.Duration  `json:"duration"`
	Description string         `json:"description"`
	Metrics     PatternMetrics `json:"metrics"`
	Severity    string         `json:"severity"`
	Impact      string         `json:"impact"`
	Suggestion  string         `json:"suggestion"`
}

// PatternMetrics contains statistical metrics for a pattern
type PatternMetrics struct {
	Mean           float64 `json:"mean"`
	StdDev         float64 `json:"std_dev"`
	Min            float64 `json:"min"`
	Max            float64 `json:"max"`
	Trend          float64 `json:"trend"`
	Periodicity    float64 `json:"periodicity"`
	Volatility     float64 `json:"volatility"`
	GrowthRate     float64 `json:"growth_rate"`
	BurstIntensity float64 `json:"burst_intensity"`
}

// Anomaly represents an allocation anomaly
type Anomaly struct {
	Type      string    `json:"type"`
	Timestamp time.Time `json:"timestamp"`
	Value     float64   `json:"value"`
	Expected  float64   `json:"expected"`
	Deviation float64   `json:"deviation"`
	Severity  float64   `json:"severity"`
	Message   string    `json:"message"`
}

// PatternDetector analyzes allocation patterns
type PatternDetector struct {
	config       *Config
	dataPoints   []analyzer.AllocationMetrics
	patterns     []DetectedPattern
	anomalies    []Anomaly
	lastAnalysis time.Time
}

// NewPatternDetector creates a new pattern detector
func NewPatternDetector(config *Config) *PatternDetector {
	if config.ConfidenceLevel == 0 {
		config.ConfidenceLevel = 0.8
	}

	return &PatternDetector{
		config:     config,
		dataPoints: make([]analyzer.AllocationMetrics, 0, config.WindowSize),
		patterns:   make([]DetectedPattern, 0),
		anomalies:  make([]Anomaly, 0),
	}
}

// AddDataPoint adds a new data point for analysis
func (pd *PatternDetector) AddDataPoint(metric analyzer.AllocationMetrics) {
	pd.dataPoints = append(pd.dataPoints, metric)

	// Keep only the last WindowSize data points
	if len(pd.dataPoints) > pd.config.WindowSize {
		pd.dataPoints = pd.dataPoints[1:]
	}
}

// DetectPatterns analyzes current data and detects patterns
func (pd *PatternDetector) DetectPatterns() []DetectedPattern {
	if len(pd.dataPoints) < pd.config.MinPatternLength {
		return nil
	}

	// Clear previous patterns
	pd.patterns = pd.patterns[:0]

	// Extract time series data
	allocationRates := pd.extractAllocationRates()
	timestamps := pd.extractTimestamps()

	// Detect different pattern types
	pd.detectBurstPatterns(allocationRates, timestamps)
	pd.detectGrowthPatterns(allocationRates, timestamps)
	pd.detectCyclicPatterns(allocationRates, timestamps)
	pd.detectSteadyPatterns(allocationRates, timestamps)
	pd.detectSpikePatterns(allocationRates, timestamps)

	pd.lastAnalysis = time.Now()
	return pd.patterns
}

// DetectAnomalies detects allocation anomalies
func (pd *PatternDetector) DetectAnomalies() []Anomaly {
	if len(pd.dataPoints) < 10 {
		return nil
	}

	// Clear previous anomalies
	pd.anomalies = pd.anomalies[:0]

	allocationRates := pd.extractAllocationRates()
	timestamps := pd.extractTimestamps()

	// Calculate statistical measures
	mean := stat.Mean(allocationRates, nil)
	stdDev := stat.StdDev(allocationRates, nil)
	threshold := pd.config.AnomalyThreshold * stdDev

	// Detect anomalies
	for i, rate := range allocationRates {
		deviation := math.Abs(rate - mean)
		if deviation > threshold {
			severity := deviation / stdDev
			anomalyType := "high"
			if rate < mean {
				anomalyType = "low"
			}

			anomaly := Anomaly{
				Type:      anomalyType,
				Timestamp: timestamps[i],
				Value:     rate,
				Expected:  mean,
				Deviation: deviation,
				Severity:  severity,
				Message:   fmt.Sprintf("%s allocation rate anomaly: %.2f (expected: %.2f)", anomalyType, rate, mean),
			}

			pd.anomalies = append(pd.anomalies, anomaly)
		}
	}

	return pd.anomalies
}

// extractAllocationRates extracts allocation rates from data points
func (pd *PatternDetector) extractAllocationRates() []float64 {
	rates := make([]float64, len(pd.dataPoints))
	for i, dp := range pd.dataPoints {
		rates[i] = float64(dp.BytesPerSecond)
	}
	return rates
}

// extractTimestamps extracts timestamps from data points
func (pd *PatternDetector) extractTimestamps() []time.Time {
	timestamps := make([]time.Time, len(pd.dataPoints))
	for i, dp := range pd.dataPoints {
		timestamps[i] = dp.Timestamp
	}
	return timestamps
}

// detectBurstPatterns detects burst allocation patterns
func (pd *PatternDetector) detectBurstPatterns(rates []float64, timestamps []time.Time) {
	if len(rates) < 5 {
		return
	}

	mean := stat.Mean(rates, nil)
	stdDev := stat.StdDev(rates, nil)
	burstThreshold := mean + 2*stdDev

	// Find burst sequences
	var burstStart, burstEnd int
	inBurst := false

	for i, rate := range rates {
		if rate > burstThreshold && !inBurst {
			// Start of burst
			burstStart = i
			inBurst = true
		} else if rate <= burstThreshold && inBurst {
			// End of burst
			burstEnd = i - 1
			inBurst = false

			// Create burst pattern if significant
			if burstEnd-burstStart >= 2 {
				pattern := pd.createBurstPattern(rates[burstStart:burstEnd+1], timestamps[burstStart], timestamps[burstEnd])
				pd.patterns = append(pd.patterns, pattern)
			}
		}
	}

	// Handle burst that continues to the end
	if inBurst {
		burstEnd = len(rates) - 1
		if burstEnd-burstStart >= 2 {
			pattern := pd.createBurstPattern(rates[burstStart:], timestamps[burstStart], timestamps[burstEnd])
			pd.patterns = append(pd.patterns, pattern)
		}
	}
}

// detectGrowthPatterns detects growth allocation patterns
func (pd *PatternDetector) detectGrowthPatterns(rates []float64, timestamps []time.Time) {
	if len(rates) < pd.config.MinPatternLength {
		return
	}

	// Calculate trend using linear regression
	x := make([]float64, len(rates))
	for i := range x {
		x[i] = float64(i)
	}

	alpha, beta := stat.LinearRegression(x, rates, nil, false)
	correlation := stat.Correlation(x, rates, nil)

	// Detect significant growth trend
	if beta > 0 && math.Abs(correlation) > 0.7 {
		pattern := DetectedPattern{
			Type:        PatternGrowth,
			Confidence:  math.Abs(correlation),
			StartTime:   timestamps[0],
			EndTime:     timestamps[len(timestamps)-1],
			Duration:    timestamps[len(timestamps)-1].Sub(timestamps[0]),
			Description: fmt.Sprintf("Growth pattern with rate %.2f bytes/s per sample", beta),
			Metrics: PatternMetrics{
				Mean:       stat.Mean(rates, nil),
				StdDev:     stat.StdDev(rates, nil),
				Min:        minFloat64(rates),
			Max:        maxFloat64(rates),
				Trend:      beta,
				GrowthRate: beta / alpha * 100, // Percentage growth rate
			},
		}

		pd.classifyPattern(&pattern)
		pd.patterns = append(pd.patterns, pattern)
	}
}

// detectCyclicPatterns detects cyclic allocation patterns
func (pd *PatternDetector) detectCyclicPatterns(rates []float64, timestamps []time.Time) {
	if len(rates) < 20 {
		return
	}

	// Simple autocorrelation-based cycle detection
	bestPeriod := 0
	bestCorrelation := 0.0

	// Test different periods
	for period := 3; period <= len(rates)/3; period++ {
		correlation := pd.calculateAutocorrelation(rates, period)
		if correlation > bestCorrelation && correlation > 0.6 {
			bestCorrelation = correlation
			bestPeriod = period
		}
	}

	if bestPeriod > 0 {
		pattern := DetectedPattern{
			Type:        PatternCyclic,
			Confidence:  bestCorrelation,
			StartTime:   timestamps[0],
			EndTime:     timestamps[len(timestamps)-1],
			Duration:    timestamps[len(timestamps)-1].Sub(timestamps[0]),
			Description: fmt.Sprintf("Cyclic pattern with period %d samples", bestPeriod),
			Metrics: PatternMetrics{
				Mean:        stat.Mean(rates, nil),
				StdDev:      stat.StdDev(rates, nil),
				Min:         minFloat64(rates),
			Max:         maxFloat64(rates),
				Periodicity: float64(bestPeriod),
				Volatility:  stat.StdDev(rates, nil) / stat.Mean(rates, nil),
			},
		}

		pd.classifyPattern(&pattern)
		pd.patterns = append(pd.patterns, pattern)
	}
}

// detectSteadyPatterns detects steady allocation patterns
func (pd *PatternDetector) detectSteadyPatterns(rates []float64, timestamps []time.Time) {
	if len(rates) < pd.config.MinPatternLength {
		return
	}

	mean := stat.Mean(rates, nil)
	stdDev := stat.StdDev(rates, nil)
	cv := stdDev / mean // Coefficient of variation

	// Steady pattern has low coefficient of variation
	if cv < 0.2 && mean > 0 {
		pattern := DetectedPattern{
			Type:        PatternSteady,
			Confidence:  1.0 - cv, // Higher confidence for lower variation
			StartTime:   timestamps[0],
			EndTime:     timestamps[len(timestamps)-1],
			Duration:    timestamps[len(timestamps)-1].Sub(timestamps[0]),
			Description: fmt.Sprintf("Steady allocation pattern at %.2f bytes/s", mean),
			Metrics: PatternMetrics{
				Mean:       mean,
				StdDev:     stdDev,
				Min:        minFloat64(rates),
			Max:        maxFloat64(rates),
				Volatility: cv,
			},
		}

		pd.classifyPattern(&pattern)
		pd.patterns = append(pd.patterns, pattern)
	}
}

// detectSpikePatterns detects spike allocation patterns
func (pd *PatternDetector) detectSpikePatterns(rates []float64, timestamps []time.Time) {
	if len(rates) < 5 {
		return
	}

	mean := stat.Mean(rates, nil)
	stdDev := stat.StdDev(rates, nil)
	spikeThreshold := mean + 3*stdDev

	for i, rate := range rates {
		if rate > spikeThreshold {
			// Check if it's an isolated spike
			isolated := true
			for j := max(0, i-2); j <= min(len(rates)-1, i+2); j++ {
				if j != i && rates[j] > mean+stdDev {
					isolated = false
					break
				}
			}

			if isolated {
				pattern := DetectedPattern{
					Type:        PatternSpike,
					Confidence:  0.9,
					StartTime:   timestamps[i],
					EndTime:     timestamps[i],
					Duration:    0,
					Description: fmt.Sprintf("Allocation spike: %.2f bytes/s (%.1fx normal)", rate, rate/mean),
					Metrics: PatternMetrics{
						Mean:           rate,
						BurstIntensity: rate / mean,
					},
				}

				pd.classifyPattern(&pattern)
				pd.patterns = append(pd.patterns, pattern)
			}
		}
	}
}

// createBurstPattern creates a burst pattern from data
func (pd *PatternDetector) createBurstPattern(burstRates []float64, startTime, endTime time.Time) DetectedPattern {
	mean := stat.Mean(burstRates, nil)
	max := maxFloat64(burstRates)
	overallMean := stat.Mean(pd.extractAllocationRates(), nil)

	pattern := DetectedPattern{
		Type:        PatternBurst,
		Confidence:  0.8,
		StartTime:   startTime,
		EndTime:     endTime,
		Duration:    endTime.Sub(startTime),
		Description: fmt.Sprintf("Burst pattern: %.2f bytes/s average (%.1fx normal)", mean, mean/overallMean),
		Metrics: PatternMetrics{
			Mean:           mean,
			StdDev:         stat.StdDev(burstRates, nil),
			Min:            minFloat64(burstRates),
			Max:            max,
			BurstIntensity: mean / overallMean,
		},
	}

	pd.classifyPattern(&pattern)
	return pattern
}

// calculateAutocorrelation calculates autocorrelation for a given lag
func (pd *PatternDetector) calculateAutocorrelation(data []float64, lag int) float64 {
	if lag >= len(data) {
		return 0
	}

	n := len(data) - lag
	if n <= 0 {
		return 0
	}

	x1 := data[:n]
	x2 := data[lag : lag+n]

	return stat.Correlation(x1, x2, nil)
}

// classifyPattern classifies pattern severity and generates suggestions
func (pd *PatternDetector) classifyPattern(pattern *DetectedPattern) {
	switch pattern.Type {
	case PatternBurst:
		if pattern.Metrics.BurstIntensity > 5 {
			pattern.Severity = "high"
			pattern.Impact = "Significant memory pressure during bursts"
			pattern.Suggestion = "Consider implementing object pooling or rate limiting"
		} else if pattern.Metrics.BurstIntensity > 2 {
			pattern.Severity = "medium"
			pattern.Impact = "Moderate memory pressure spikes"
			pattern.Suggestion = "Monitor burst frequency and consider optimization"
		} else {
			pattern.Severity = "low"
			pattern.Impact = "Minor allocation bursts"
			pattern.Suggestion = "Normal behavior, continue monitoring"
		}

	case PatternGrowth:
		if pattern.Metrics.GrowthRate > 50 {
			pattern.Severity = "critical"
			pattern.Impact = "Rapid memory growth may lead to OOM"
			pattern.Suggestion = "Immediate investigation required - check for memory leaks"
		} else if pattern.Metrics.GrowthRate > 20 {
			pattern.Severity = "high"
			pattern.Impact = "Concerning memory growth trend"
			pattern.Suggestion = "Review allocation patterns and implement cleanup"
		} else {
			pattern.Severity = "medium"
			pattern.Impact = "Gradual memory growth"
			pattern.Suggestion = "Monitor growth trend and optimize if needed"
		}

	case PatternCyclic:
		if pattern.Metrics.Volatility > 0.5 {
			pattern.Severity = "medium"
			pattern.Impact = "High allocation variability"
			pattern.Suggestion = "Consider smoothing allocation patterns"
		} else {
			pattern.Severity = "low"
			pattern.Impact = "Predictable cyclic behavior"
			pattern.Suggestion = "Normal cyclic pattern, optimize if needed"
		}

	case PatternSpike:
		if pattern.Metrics.BurstIntensity > 10 {
			pattern.Severity = "high"
			pattern.Impact = "Severe allocation spike"
			pattern.Suggestion = "Investigate cause of allocation spike"
		} else {
			pattern.Severity = "medium"
			pattern.Impact = "Allocation spike detected"
			pattern.Suggestion = "Monitor for recurring spikes"
		}

	case PatternSteady:
		pattern.Severity = "low"
		pattern.Impact = "Stable allocation behavior"
		pattern.Suggestion = "Good allocation stability"

	default:
		pattern.Severity = "unknown"
		pattern.Impact = "Unknown pattern impact"
		pattern.Suggestion = "Further analysis needed"
	}
}

// SaveToFile saves the detected pattern to a file
func (dp *DetectedPattern) SaveToFile(filename string) error {
	file, err := os.Create(filename)
	if err != nil {
		return err
	}
	defer file.Close()

	encoder := json.NewEncoder(file)
	encoder.SetIndent("", "  ")
	return encoder.Encode(dp)
}

// GetPatternSummary returns a summary of detected patterns
func (pd *PatternDetector) GetPatternSummary() map[string]interface{} {
	summary := make(map[string]interface{})
	typeCounts := make(map[PatternType]int)
	severityCounts := make(map[string]int)

	for _, pattern := range pd.patterns {
		typeCounts[pattern.Type]++
		severityCounts[pattern.Severity]++
	}

	summary["total_patterns"] = len(pd.patterns)
	summary["pattern_types"] = typeCounts
	summary["severity_distribution"] = severityCounts
	summary["last_analysis"] = pd.lastAnalysis
	summary["data_points"] = len(pd.dataPoints)

	return summary
}

// Helper functions
func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
