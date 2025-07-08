package monitoring

import (
	"math"
	"sort"
	"sync"
	"time"
)

// Histogram tracks distribution of values
type Histogram struct {
	values []float64
	mu     sync.RWMutex
	buckets map[float64]int64 // bucket upper bound -> count
	count  int64
	sum    float64
}

// HistogramStats contains histogram statistics
type HistogramStats struct {
	Count  int64
	Sum    float64
	Mean   float64
	Min    float64
	Max    float64
	P50    float64
	P95    float64
	P99    float64
	P999   float64
	StdDev float64
}

// NewHistogram creates a new histogram
func NewHistogram() *Histogram {
	return &Histogram{
		values:  make([]float64, 0, 1000),
		buckets: make(map[float64]int64),
	}
}

// Record adds a value to the histogram
func (h *Histogram) Record(value float64) {
	h.mu.Lock()
	defer h.mu.Unlock()
	
	h.values = append(h.values, value)
	h.count++
	h.sum += value
	
	// Update buckets (exponential buckets)
	bucket := h.getBucket(value)
	h.buckets[bucket]++
}

// getBucket returns the appropriate bucket for a value
func (h *Histogram) getBucket(value float64) float64 {
	// Exponential buckets: 0.001, 0.01, 0.1, 1, 10, 100, 1000, 10000, +Inf
	buckets := []float64{0.001, 0.01, 0.1, 1, 10, 100, 1000, 10000, math.Inf(1)}
	
	for _, bucket := range buckets {
		if value <= bucket {
			return bucket
		}
	}
	return math.Inf(1)
}

// Stats returns current histogram statistics
func (h *Histogram) Stats() HistogramStats {
	h.mu.RLock()
	defer h.mu.RUnlock()
	
	if h.count == 0 {
		return HistogramStats{}
	}
	
	// Create a copy of values for calculations
	values := make([]float64, len(h.values))
	copy(values, h.values)
	sort.Float64s(values)
	
	stats := HistogramStats{
		Count: h.count,
		Sum:   h.sum,
		Mean:  h.sum / float64(h.count),
		Min:   values[0],
		Max:   values[len(values)-1],
	}
	
	// Calculate percentiles
	stats.P50 = h.percentile(values, 0.50)
	stats.P95 = h.percentile(values, 0.95)
	stats.P99 = h.percentile(values, 0.99)
	stats.P999 = h.percentile(values, 0.999)
	
	// Calculate standard deviation
	var variance float64
	for _, v := range values {
		diff := v - stats.Mean
		variance += diff * diff
	}
	variance /= float64(len(values))
	stats.StdDev = math.Sqrt(variance)
	
	return stats
}

// percentile calculates the given percentile from sorted values
func (h *Histogram) percentile(sortedValues []float64, p float64) float64 {
	if len(sortedValues) == 0 {
		return 0
	}
	
	index := p * float64(len(sortedValues)-1)
	lower := int(math.Floor(index))
	upper := int(math.Ceil(index))
	
	if lower == upper {
		return sortedValues[lower]
	}
	
	// Linear interpolation
	weight := index - float64(lower)
	return sortedValues[lower]*(1-weight) + sortedValues[upper]*weight
}

// Reset clears the histogram
func (h *Histogram) Reset() {
	h.mu.Lock()
	defer h.mu.Unlock()
	
	h.values = h.values[:0]
	h.buckets = make(map[float64]int64)
	h.count = 0
	h.sum = 0
}

// GetBuckets returns the current bucket counts
func (h *Histogram) GetBuckets() map[float64]int64 {
	h.mu.RLock()
	defer h.mu.RUnlock()
	
	buckets := make(map[float64]int64)
	for k, v := range h.buckets {
		buckets[k] = v
	}
	return buckets
}

// Timer tracks timing information
type Timer struct {
	durations []time.Duration
	mu        sync.RWMutex
	count     int64
	sum       time.Duration
}

// TimerStats contains timer statistics
type TimerStats struct {
	Count  int64
	Sum    time.Duration
	Mean   time.Duration
	Min    time.Duration
	Max    time.Duration
	P50    time.Duration
	P95    time.Duration
	P99    time.Duration
	P999   time.Duration
	StdDev time.Duration
}

// TimerInstance represents a running timer
type TimerInstance struct {
	startTime time.Time
	timer     *Timer
}

// NewTimer creates a new timer
func NewTimer() *Timer {
	return &Timer{
		durations: make([]time.Duration, 0, 1000),
	}
}

// Start starts a new timer instance
func (t *Timer) Start() *TimerInstance {
	return &TimerInstance{
		startTime: time.Now(),
		timer:     t,
	}
}

// Stop stops the timer instance and records the duration
func (ti *TimerInstance) Stop() time.Duration {
	duration := time.Since(ti.startTime)
	ti.timer.Record(duration)
	return duration
}

// Record adds a duration to the timer
func (t *Timer) Record(duration time.Duration) {
	t.mu.Lock()
	defer t.mu.Unlock()
	
	t.durations = append(t.durations, duration)
	t.count++
	t.sum += duration
}

// Stats returns current timer statistics
func (t *Timer) Stats() TimerStats {
	t.mu.RLock()
	defer t.mu.RUnlock()
	
	if t.count == 0 {
		return TimerStats{}
	}
	
	// Create a copy of durations for calculations
	durations := make([]time.Duration, len(t.durations))
	copy(durations, t.durations)
	sort.Slice(durations, func(i, j int) bool {
		return durations[i] < durations[j]
	})
	
	stats := TimerStats{
		Count: t.count,
		Sum:   t.sum,
		Mean:  time.Duration(int64(t.sum) / t.count),
		Min:   durations[0],
		Max:   durations[len(durations)-1],
	}
	
	// Calculate percentiles
	stats.P50 = t.percentileDuration(durations, 0.50)
	stats.P95 = t.percentileDuration(durations, 0.95)
	stats.P99 = t.percentileDuration(durations, 0.99)
	stats.P999 = t.percentileDuration(durations, 0.999)
	
	// Calculate standard deviation
	meanNanos := float64(stats.Mean.Nanoseconds())
	var variance float64
	for _, d := range durations {
		diff := float64(d.Nanoseconds()) - meanNanos
		variance += diff * diff
	}
	variance /= float64(len(durations))
	stats.StdDev = time.Duration(int64(math.Sqrt(variance)))
	
	return stats
}

// percentileDuration calculates the given percentile from sorted durations
func (t *Timer) percentileDuration(sortedDurations []time.Duration, p float64) time.Duration {
	if len(sortedDurations) == 0 {
		return 0
	}
	
	index := p * float64(len(sortedDurations)-1)
	lower := int(math.Floor(index))
	upper := int(math.Ceil(index))
	
	if lower == upper {
		return sortedDurations[lower]
	}
	
	// Linear interpolation
	weight := index - float64(lower)
	lowerNanos := float64(sortedDurations[lower].Nanoseconds())
	upperNanos := float64(sortedDurations[upper].Nanoseconds())
	interpolated := lowerNanos*(1-weight) + upperNanos*weight
	return time.Duration(int64(interpolated))
}

// Reset clears the timer
func (t *Timer) Reset() {
	t.mu.Lock()
	defer t.mu.Unlock()
	
	t.durations = t.durations[:0]
	t.count = 0
	t.sum = 0
}

// GetDurations returns a copy of all recorded durations
func (t *Timer) GetDurations() []time.Duration {
	t.mu.RLock()
	defer t.mu.RUnlock()
	
	durations := make([]time.Duration, len(t.durations))
	copy(durations, t.durations)
	return durations
}

// Rate calculates operations per second based on recorded durations
func (t *Timer) Rate() float64 {
	t.mu.RLock()
	defer t.mu.RUnlock()
	
	if t.count == 0 || t.sum == 0 {
		return 0
	}
	
	avgDuration := float64(t.sum.Nanoseconds()) / float64(t.count)
	return 1e9 / avgDuration // operations per second
}