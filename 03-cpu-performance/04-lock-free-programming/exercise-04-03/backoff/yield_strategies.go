package backoff

import (
	"runtime"
	"sync/atomic"
	"time"
)

// YieldStrategy defines different CPU yield strategies
type YieldStrategy int

const (
	YieldGosched YieldStrategy = iota
	YieldSleep
	YieldSpinWait
	YieldHybrid
)

// String returns the string representation of YieldStrategy
func (ys YieldStrategy) String() string {
	switch ys {
	case YieldGosched:
		return "Gosched"
	case YieldSleep:
		return "Sleep"
	case YieldSpinWait:
		return "SpinWait"
	case YieldHybrid:
		return "Hybrid"
	default:
		return "Unknown"
	}
}

// YieldManager manages different yield strategies
type YieldManager struct {
	strategy     YieldStrategy
	spinCount    int
	sleepDuration time.Duration
	yieldCount   int64
}

func NewYieldManager(strategy YieldStrategy) *YieldManager {
	return &YieldManager{
		strategy:      strategy,
		spinCount:     100,
		sleepDuration: time.Nanosecond,
	}
}

// Yield performs the configured yield strategy
func (ym *YieldManager) Yield() {
	atomic.AddInt64(&ym.yieldCount, 1)
	
	switch ym.strategy {
	case YieldGosched:
		runtime.Gosched()
	case YieldSleep:
		time.Sleep(ym.sleepDuration)
	case YieldSpinWait:
		ym.spinWait()
	case YieldHybrid:
		ym.hybridYield()
	default:
		runtime.Gosched()
	}
}

// spinWait performs a busy-wait spin
func (ym *YieldManager) spinWait() {
	for i := 0; i < ym.spinCount; i++ {
		runtime.Gosched() // Minimal yield within spin
	}
}

// hybridYield combines different strategies based on yield count
func (ym *YieldManager) hybridYield() {
	yieldCount := atomic.LoadInt64(&ym.yieldCount)
	
	switch {
	case yieldCount < 10:
		// Start with spin for low contention
		ym.spinWait()
	case yieldCount < 100:
		// Move to gosched for medium contention
		runtime.Gosched()
	default:
		// Use sleep for high contention
		time.Sleep(ym.sleepDuration)
	}
}

// SetSpinCount sets the number of spins for spin-wait strategy
func (ym *YieldManager) SetSpinCount(count int) {
	ym.spinCount = count
}

// SetSleepDuration sets the sleep duration for sleep strategy
func (ym *YieldManager) SetSleepDuration(duration time.Duration) {
	ym.sleepDuration = duration
}

// GetYieldCount returns the total number of yields performed
func (ym *YieldManager) GetYieldCount() int64 {
	return atomic.LoadInt64(&ym.yieldCount)
}

// Reset resets the yield count
func (ym *YieldManager) Reset() {
	atomic.StoreInt64(&ym.yieldCount, 0)
}

// AdaptiveYieldManager adapts yield strategy based on performance
type AdaptiveYieldManager struct {
	currentStrategy YieldStrategy
	strategies      []YieldStrategy
	performance     []int64 // Success count for each strategy
	currentIndex    int
	evaluationCount int64
	evalInterval    int64
}

func NewAdaptiveYieldManager() *AdaptiveYieldManager {
	strategies := []YieldStrategy{YieldGosched, YieldSleep, YieldSpinWait, YieldHybrid}
	return &AdaptiveYieldManager{
		currentStrategy: YieldGosched,
		strategies:      strategies,
		performance:     make([]int64, len(strategies)),
		evalInterval:    1000,
	}
}

// Yield performs adaptive yielding
func (aym *AdaptiveYieldManager) Yield() {
	atomic.AddInt64(&aym.evaluationCount, 1)
	
	// Evaluate and potentially switch strategies
	if atomic.LoadInt64(&aym.evaluationCount)%aym.evalInterval == 0 {
		aym.evaluateAndSwitch()
	}
	
	// Perform yield with current strategy
	switch aym.currentStrategy {
	case YieldGosched:
		runtime.Gosched()
	case YieldSleep:
		time.Sleep(time.Nanosecond)
	case YieldSpinWait:
		for i := 0; i < 100; i++ {
			runtime.Gosched()
		}
	case YieldHybrid:
		count := atomic.LoadInt64(&aym.evaluationCount)
		if count%10 < 5 {
			runtime.Gosched()
		} else {
			time.Sleep(time.Nanosecond)
		}
	}
}

// RecordSuccess records a successful operation for the current strategy
func (aym *AdaptiveYieldManager) RecordSuccess() {
	atomic.AddInt64(&aym.performance[aym.currentIndex], 1)
}

// RecordPerformance records performance data for the current strategy
func (aym *AdaptiveYieldManager) RecordPerformance(duration time.Duration) {
	// Convert duration to a performance score (lower duration = better performance)
	score := int64(time.Millisecond / duration)
	if score <= 0 {
		score = 1
	}
	atomic.AddInt64(&aym.performance[aym.currentIndex], score)
}

// evaluateAndSwitch evaluates performance and switches to best strategy
func (aym *AdaptiveYieldManager) evaluateAndSwitch() {
	bestIndex := 0
	bestPerformance := atomic.LoadInt64(&aym.performance[0])
	
	for i := 1; i < len(aym.performance); i++ {
		perf := atomic.LoadInt64(&aym.performance[i])
		if perf > bestPerformance {
			bestPerformance = perf
			bestIndex = i
		}
	}
	
	// Switch to best performing strategy
	aym.currentIndex = bestIndex
	aym.currentStrategy = aym.strategies[bestIndex]
	
	// Reset performance counters
	for i := range aym.performance {
		atomic.StoreInt64(&aym.performance[i], 0)
	}
}

// GetCurrentStrategy returns the current yield strategy
func (aym *AdaptiveYieldManager) GetCurrentStrategy() YieldStrategy {
	return aym.currentStrategy
}

// GetPerformanceStats returns performance statistics for all strategies
func (aym *AdaptiveYieldManager) GetPerformanceStats() []int64 {
	stats := make([]int64, len(aym.performance))
	for i, perf := range aym.performance {
		stats[i] = atomic.LoadInt64(&perf)
	}
	return stats
}

// PauseYield implements a pause-based yield for x86 processors
func PauseYield() {
	// On x86, this would be a PAUSE instruction
	// In Go, we simulate with a very short sleep
	time.Sleep(time.Nanosecond)
}

// CPURelaxYield performs CPU relaxation
func CPURelaxYield() {
	// Combination of pause and gosched
	PauseYield()
	runtime.Gosched()
}

// ProgressiveYield implements progressive yielding strategy
func ProgressiveYield(attempt int) {
	switch {
	case attempt < 10:
		// Spin for very short time
		for i := 0; i < attempt; i++ {
			PauseYield()
		}
	case attempt < 50:
		// Use gosched
		runtime.Gosched()
	case attempt < 100:
		// Short sleep
		time.Sleep(time.Nanosecond * time.Duration(attempt-50))
	default:
		// Longer sleep with cap
		sleepTime := time.Microsecond * time.Duration(attempt-100)
		if sleepTime > time.Millisecond {
			sleepTime = time.Millisecond
		}
		time.Sleep(sleepTime)
	}
}

// CASWithYieldStrategy performs CAS with specified yield strategy
func CASWithYieldStrategy(addr *int64, old, new int64, ym *YieldManager, maxAttempts int) bool {
	for i := 0; i < maxAttempts; i++ {
		if atomic.CompareAndSwapInt64(addr, old, new) {
			return true
		}
		
		// Check if value changed
		current := atomic.LoadInt64(addr)
		if current != old {
			return false
		}
		
		ym.Yield()
	}
	
	return false
}