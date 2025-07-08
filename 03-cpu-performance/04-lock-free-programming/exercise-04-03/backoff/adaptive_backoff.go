package backoff

import (
	"math/rand"
	"runtime"
	"sync/atomic"
	"time"
)

// AdaptiveBackoff adjusts strategy based on system load
type AdaptiveBackoff struct {
	minDelay        time.Duration
	maxDelay        time.Duration
	currentDelay    time.Duration
	successCount    int64
	failureCount    int64
	lastAdjustment  time.Time
	adjustInterval  time.Duration
	successThreshold int64
	failureThreshold int64
}

func NewAdaptiveBackoff(min, max time.Duration) *AdaptiveBackoff {
	return &AdaptiveBackoff{
		minDelay:         min,
		maxDelay:         max,
		currentDelay:     min,
		lastAdjustment:   time.Now(),
		adjustInterval:   100 * time.Millisecond,
		successThreshold: 10,
		failureThreshold: 5,
	}
}

// RecordSuccess records a successful operation
func (ab *AdaptiveBackoff) RecordSuccess() {
	atomic.AddInt64(&ab.successCount, 1)
	ab.adjustDelay()
}

// RecordFailure records a failed operation
func (ab *AdaptiveBackoff) RecordFailure() {
	atomic.AddInt64(&ab.failureCount, 1)
	ab.adjustDelay()
}

// adjustDelay adjusts the delay based on success/failure ratio
func (ab *AdaptiveBackoff) adjustDelay() {
	now := time.Now()
	if now.Sub(ab.lastAdjustment) < ab.adjustInterval {
		return
	}
	
	successCount := atomic.LoadInt64(&ab.successCount)
	failureCount := atomic.LoadInt64(&ab.failureCount)
	total := successCount + failureCount
	
	if total == 0 {
		return
	}
	
	successRate := float64(successCount) / float64(total)
	
	// Adjust delay based on success rate
	if successRate > 0.8 {
		// High success rate, reduce delay
		ab.currentDelay = time.Duration(float64(ab.currentDelay) * 0.9)
		if ab.currentDelay < ab.minDelay {
			ab.currentDelay = ab.minDelay
		}
	} else if successRate < 0.3 {
		// Low success rate, increase delay
		ab.currentDelay = time.Duration(float64(ab.currentDelay) * 1.5)
		if ab.currentDelay > ab.maxDelay {
			ab.currentDelay = ab.maxDelay
		}
	}
	
	ab.lastAdjustment = now
	
	// Reset counters periodically
	if total > 1000 {
		atomic.StoreInt64(&ab.successCount, successCount/2)
		atomic.StoreInt64(&ab.failureCount, failureCount/2)
	}
}

// Backoff performs adaptive backoff
func (ab *AdaptiveBackoff) Backoff() {
	delay := ab.currentDelay
	
	// Add small random jitter
	if delay > 0 {
		jitter := time.Duration(rand.Int63n(int64(delay / 4)))
		time.Sleep(delay + jitter)
	} else {
		runtime.Gosched()
	}
}

// BackoffStats represents backoff statistics
type BackoffStats struct {
	SuccessCount int64
	FailureCount int64
	SuccessRate  float64
	CurrentDelay time.Duration
}

// GetStats returns current statistics
func (ab *AdaptiveBackoff) GetStats() BackoffStats {
	successCount := atomic.LoadInt64(&ab.successCount)
	failureCount := atomic.LoadInt64(&ab.failureCount)
	total := successCount + failureCount
	
	var successRate float64
	if total > 0 {
		successRate = float64(successCount) / float64(total)
	}
	
	return BackoffStats{
		SuccessCount: successCount,
		FailureCount: failureCount,
		SuccessRate:  successRate,
		CurrentDelay: ab.currentDelay,
	}
}

// Reset resets the adaptive backoff state
func (ab *AdaptiveBackoff) Reset() {
	ab.currentDelay = ab.minDelay
	atomic.StoreInt64(&ab.successCount, 0)
	atomic.StoreInt64(&ab.failureCount, 0)
	ab.lastAdjustment = time.Now()
}

// LoadBasedBackoff adjusts based on system load
type LoadBasedBackoff struct {
	baseDelay    time.Duration
	maxDelay     time.Duration
	loadSamples  []float64
	sampleIndex  int
	lastSample   time.Time
	sampleWindow time.Duration
}

func NewLoadBasedBackoff(base, max time.Duration, sampleSize int) *LoadBasedBackoff {
	return &LoadBasedBackoff{
		baseDelay:    base,
		maxDelay:     max,
		loadSamples:  make([]float64, sampleSize),
		lastSample:   time.Now(),
		sampleWindow: 50 * time.Millisecond,
	}
}

// sampleLoad estimates current system load
func (lb *LoadBasedBackoff) sampleLoad() float64 {
	// Simple load estimation based on goroutine scheduling
	start := time.Now()
	runtime.Gosched()
	elapsed := time.Since(start)
	
	// Convert to load factor (higher = more loaded)
	loadFactor := float64(elapsed.Nanoseconds()) / float64(time.Microsecond.Nanoseconds())
	if loadFactor > 10.0 {
		loadFactor = 10.0
	}
	
	return loadFactor
}

// updateLoadSample updates the load sample window
func (lb *LoadBasedBackoff) updateLoadSample() {
	now := time.Now()
	if now.Sub(lb.lastSample) < lb.sampleWindow {
		return
	}
	
	load := lb.sampleLoad()
	lb.loadSamples[lb.sampleIndex] = load
	lb.sampleIndex = (lb.sampleIndex + 1) % len(lb.loadSamples)
	lb.lastSample = now
}

// getAverageLoad calculates average load over the sample window
func (lb *LoadBasedBackoff) getAverageLoad() float64 {
	var sum float64
	count := 0
	
	for _, sample := range lb.loadSamples {
		if sample > 0 {
			sum += sample
			count++
		}
	}
	
	if count == 0 {
		return 1.0
	}
	
	return sum / float64(count)
}

// Backoff performs load-based backoff
func (lb *LoadBasedBackoff) Backoff() {
	lb.updateLoadSample()
	avgLoad := lb.getAverageLoad()
	
	// Scale delay based on load
	delay := time.Duration(float64(lb.baseDelay) * avgLoad)
	if delay > lb.maxDelay {
		delay = lb.maxDelay
	}
	
	if delay > 0 {
		time.Sleep(delay)
	} else {
		runtime.Gosched()
	}
}

// GetCurrentLoad returns the current estimated load
func (lb *LoadBasedBackoff) GetCurrentLoad() float64 {
	return lb.getAverageLoad()
}