package backoff

import (
	"math/rand"
	"runtime"
	"sync/atomic"
	"time"
)

// ContentionManager manages contention and backoff strategies
type ContentionManager struct {
	successes   int64
	failures    int64
	currentWait time.Duration
	maxWait     time.Duration
	factor      float64
	lastUpdate  time.Time
	updateInterval time.Duration
}

func NewContentionManager(maxWait time.Duration, factor float64) *ContentionManager {
	return &ContentionManager{
		maxWait:        maxWait,
		factor:         factor,
		lastUpdate:     time.Now(),
		updateInterval: 100 * time.Millisecond,
	}
}

// RecordSuccess records a successful operation
func (cm *ContentionManager) RecordSuccess() {
	atomic.AddInt64(&cm.successes, 1)
	
	// Decrease wait time on success
	if cm.currentWait > 0 {
		cm.currentWait = time.Duration(float64(cm.currentWait) / cm.factor)
		if cm.currentWait < time.Nanosecond {
			cm.currentWait = 0
		}
	}
}

// RecordFailure records a failed operation
func (cm *ContentionManager) RecordFailure() {
	atomic.AddInt64(&cm.failures, 1)
	
	// Increase wait time on failure
	if cm.currentWait < cm.maxWait {
		cm.currentWait = time.Duration(float64(cm.currentWait) * cm.factor)
		if cm.currentWait > cm.maxWait {
			cm.currentWait = cm.maxWait
		}
	}
}

// Backoff performs the backoff delay
func (cm *ContentionManager) Backoff() {
	if cm.currentWait > 0 {
		// Add jitter to prevent thundering herd
		jitter := time.Duration(rand.Int63n(int64(cm.currentWait / 2)))
		time.Sleep(cm.currentWait + jitter)
	} else {
		runtime.Gosched()
	}
}

// GetStats returns success/failure statistics
func (cm *ContentionManager) GetStats() BackoffStats {
	successes := atomic.LoadInt64(&cm.successes)
	failures := atomic.LoadInt64(&cm.failures)
	total := successes + failures
	
	var successRate float64
	if total > 0 {
		successRate = float64(successes) / float64(total)
	}
	
	return BackoffStats{
		SuccessCount: successes,
		FailureCount: failures,
		SuccessRate:  successRate,
		CurrentDelay: cm.currentWait,
	}
}

// GetCurrentWait returns the current wait duration
func (cm *ContentionManager) GetCurrentWait() time.Duration {
	return cm.currentWait
}

// Reset resets the contention manager state
func (cm *ContentionManager) Reset() {
	atomic.StoreInt64(&cm.successes, 0)
	atomic.StoreInt64(&cm.failures, 0)
	cm.currentWait = 0
	cm.lastUpdate = time.Now()
}

// ThreadLocalContentionManager manages per-thread contention
type ThreadLocalContentionManager struct {
	managers map[int]*ContentionManager
	global   *ContentionManager
}

func NewThreadLocalContentionManager(maxWait time.Duration, factor float64) *ThreadLocalContentionManager {
	return &ThreadLocalContentionManager{
		managers: make(map[int]*ContentionManager),
		global:   NewContentionManager(maxWait, factor),
	}
}

// GetManager returns the contention manager for the current thread
func (tlcm *ThreadLocalContentionManager) GetManager() *ContentionManager {
	// Use goroutine ID as thread identifier
	threadID := runtime.NumGoroutine() // Simple approximation
	
	if manager, exists := tlcm.managers[threadID]; exists {
		return manager
	}
	
	// Create new manager for this thread
	manager := NewContentionManager(tlcm.global.maxWait, tlcm.global.factor)
	tlcm.managers[threadID] = manager
	return manager
}

// GetGlobalStats returns aggregated statistics
func (tlcm *ThreadLocalContentionManager) GetGlobalStats() BackoffStats {
	var totalSuccesses, totalFailures int64
	
	for _, manager := range tlcm.managers {
		stats := manager.GetStats()
		totalSuccesses += stats.SuccessCount
		totalFailures += stats.FailureCount
	}
	
	total := totalSuccesses + totalFailures
	var successRate float64
	if total > 0 {
		successRate = float64(totalSuccesses) / float64(total)
	}
	
	return BackoffStats{
		SuccessCount: totalSuccesses,
		FailureCount: totalFailures,
		SuccessRate:  successRate,
		CurrentDelay: 0, // No single current delay for aggregated stats
	}
}

// ContentionAwareBackoff implements contention-aware backoff
type ContentionAwareBackoff struct {
	baseDelay       time.Duration
	maxDelay        time.Duration
	contentionLevel int64
	maxContention   int64
}

func NewContentionAwareBackoff(base, max time.Duration, factor float64, maxContention int64) *ContentionAwareBackoff {
	return &ContentionAwareBackoff{
		baseDelay:     base,
		maxDelay:      max,
		maxContention: maxContention,
	}
}

// EnterContention increments the contention level
func (cab *ContentionAwareBackoff) EnterContention() {
	atomic.AddInt64(&cab.contentionLevel, 1)
}

// ExitContention decrements the contention level
func (cab *ContentionAwareBackoff) ExitContention() {
	atomic.AddInt64(&cab.contentionLevel, -1)
}

// Backoff performs contention-aware backoff
func (cab *ContentionAwareBackoff) Backoff() {
	contention := atomic.LoadInt64(&cab.contentionLevel)
	if contention <= 0 {
		runtime.Gosched()
		return
	}
	
	// Scale delay based on contention level
	contentionRatio := float64(contention) / float64(cab.maxContention)
	if contentionRatio > 1.0 {
		contentionRatio = 1.0
	}
	
	delay := time.Duration(float64(cab.baseDelay) * (1.0 + contentionRatio*10.0))
	if delay > cab.maxDelay {
		delay = cab.maxDelay
	}
	
	// Add jitter
	jitter := time.Duration(rand.Int63n(int64(delay / 4)))
	time.Sleep(delay + jitter)
}

// RecordContention records a contention event
func (cab *ContentionAwareBackoff) RecordContention() {
	atomic.AddInt64(&cab.contentionLevel, 1)
}

// GetContentionLevel returns the current contention level
func (cab *ContentionAwareBackoff) GetContentionLevel() int64 {
	return atomic.LoadInt64(&cab.contentionLevel)
}

// CASWithContentionManagement performs CAS with contention management
func CASWithContentionManagement(addr *int64, old, new int64, maxWait time.Duration, factor float64) bool {
	cm := NewContentionManager(maxWait, factor)
	maxAttempts := 100 // Default max attempts
	
	for i := 0; i < maxAttempts; i++ {
		if atomic.CompareAndSwapInt64(addr, old, new) {
			cm.RecordSuccess()
			return true
		}
		
		// Check if value changed
		current := atomic.LoadInt64(addr)
		if current != old {
			cm.RecordFailure()
			return false
		}
		
		cm.RecordFailure()
		cm.Backoff()
	}
	
	return false
}