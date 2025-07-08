package backoff

import (
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

// TestExponentialBackoff tests exponential backoff functionality
func TestExponentialBackoff(t *testing.T) {
	backoff := NewExponentialBackoff(
		time.Microsecond,
		time.Millisecond,
		2.0,
		false,
	)
	
	// Test initial state
	if backoff.GetAttempts() != 0 {
		t.Errorf("Expected 0 attempts, got %d", backoff.GetAttempts())
	}
	
	if backoff.GetCurrentDelay() != time.Microsecond {
		t.Errorf("Expected initial delay %v, got %v", time.Microsecond, backoff.GetCurrentDelay())
	}
	
	// Test backoff progression
	initialDelay := backoff.GetCurrentDelay()
	backoff.Backoff()
	
	if backoff.GetAttempts() != 1 {
		t.Errorf("Expected 1 attempt, got %d", backoff.GetAttempts())
	}
	
	newDelay := backoff.GetCurrentDelay()
	if newDelay <= initialDelay {
		t.Errorf("Expected delay to increase, got %v -> %v", initialDelay, newDelay)
	}
	
	// Test reset
	backoff.Reset()
	if backoff.GetAttempts() != 0 {
		t.Errorf("Expected 0 attempts after reset, got %d", backoff.GetAttempts())
	}
	
	if backoff.GetCurrentDelay() != time.Microsecond {
		t.Errorf("Expected initial delay after reset, got %v", backoff.GetCurrentDelay())
	}
}

// TestExponentialBackoffWithLimit tests backoff with retry limit
func TestExponentialBackoffWithLimit(t *testing.T) {
	backoff := NewExponentialBackoff(
		time.Microsecond,
		time.Millisecond,
		2.0,
		false,
	)
	
	// Test backoff with limit
	for i := 0; i < 5; i++ {
		if !backoff.BackoffWithLimit(3) {
			if i < 3 {
				t.Errorf("Backoff should succeed for attempt %d", i)
			}
			break
		}
	}
	
	if backoff.GetAttempts() > 3 {
		t.Errorf("Expected at most 3 attempts, got %d", backoff.GetAttempts())
	}
}

// TestExponentialBackoffWithJitter tests jitter functionality
func TestExponentialBackoffWithJitter(t *testing.T) {
	backoff := NewExponentialBackoff(
		time.Microsecond,
		time.Millisecond,
		2.0,
		true, // with jitter
	)
	
	// Test that jitter produces different delays
	delays := make([]time.Duration, 10)
	for i := 0; i < 10; i++ {
		backoff.Reset()
		backoff.Backoff()
		delays[i] = backoff.GetCurrentDelay()
	}
	
	// Check that not all delays are the same (jitter should introduce variation)
	allSame := true
	for i := 1; i < len(delays); i++ {
		if delays[i] != delays[0] {
			allSame = false
			break
		}
	}
	
	if allSame {
		t.Error("Jitter should introduce variation in delays")
	}
}

// TestSimpleExponentialBackoff tests stateless exponential backoff
func TestSimpleExponentialBackoff(t *testing.T) {
	delay1 := SimpleExponentialBackoff(time.Microsecond, 2.0, 0)
	delay2 := SimpleExponentialBackoff(time.Microsecond, 2.0, 1)
	delay3 := SimpleExponentialBackoff(time.Microsecond, 2.0, 2)
	
	if delay1 != time.Microsecond {
		t.Errorf("Expected %v for attempt 0, got %v", time.Microsecond, delay1)
	}
	
	if delay2 <= delay1 {
		t.Errorf("Expected delay to increase, got %v -> %v", delay1, delay2)
	}
	
	if delay3 <= delay2 {
		t.Errorf("Expected delay to increase, got %v -> %v", delay2, delay3)
	}
}

// TestAdaptiveBackoff tests adaptive backoff functionality
func TestAdaptiveBackoff(t *testing.T) {
	backoff := NewAdaptiveBackoff(
		time.Microsecond,
		time.Millisecond,
	)
	
	// Test initial state
	stats := backoff.GetStats()
	if stats.SuccessCount != 0 || stats.FailureCount != 0 {
		t.Error("Expected zero initial stats")
	}
	
	// Test failure recording
	backoff.RecordFailure()
	backoff.Backoff()
	
	stats = backoff.GetStats()
	if stats.FailureCount != 1 {
		t.Errorf("Expected 1 failure, got %d", stats.FailureCount)
	}
	
	// Test success recording
	backoff.RecordSuccess()
	stats = backoff.GetStats()
	if stats.SuccessCount != 1 {
		t.Errorf("Expected 1 success, got %d", stats.SuccessCount)
	}
	
	// Test reset
	backoff.Reset()
	stats = backoff.GetStats()
	if stats.SuccessCount != 0 || stats.FailureCount != 0 {
		t.Error("Expected zero stats after reset")
	}
}

// TestLoadBasedBackoff tests load-based backoff functionality
func TestLoadBasedBackoff(t *testing.T) {
	backoff := NewLoadBasedBackoff(
		time.Microsecond,
		time.Millisecond,
		10, // sample size
	)
	
	// Test initial load
	load := backoff.GetCurrentLoad()
	if load < 0 || load > 1 {
		t.Errorf("Expected load between 0 and 1, got %f", load)
	}
	
	// Test backoff
	backoff.Backoff()
	
	// Load should be updated after backoff
	newLoad := backoff.GetCurrentLoad()
	if newLoad < 0 || newLoad > 1 {
		t.Errorf("Expected load between 0 and 1, got %f", newLoad)
	}
}

// TestContentionManager tests contention management functionality
func TestContentionManager(t *testing.T) {
	cm := NewContentionManager(time.Microsecond, 2.0)
	
	// Test initial state
	stats := cm.GetStats()
	if stats.SuccessCount != 0 || stats.FailureCount != 0 {
		t.Error("Expected zero initial stats")
	}
	
	// Test failure recording and backoff
	cm.RecordFailure()
	cm.Backoff()
	
	stats = cm.GetStats()
	if stats.FailureCount != 1 {
		t.Errorf("Expected 1 failure, got %d", stats.FailureCount)
	}
	
	// Test success recording
	cm.RecordSuccess()
	stats = cm.GetStats()
	if stats.SuccessCount != 1 {
		t.Errorf("Expected 1 success, got %d", stats.SuccessCount)
	}
}

// TestThreadLocalContentionManager tests thread-local contention management
func TestThreadLocalContentionManager(t *testing.T) {
	tlcm := NewThreadLocalContentionManager(time.Microsecond, 2.0)
	
	// Test that different goroutines get different managers
	var wg sync.WaitGroup
	managers := make([]interface{}, 5)
	
	for i := 0; i < 5; i++ {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()
			managers[idx] = tlcm.GetManager()
		}(i)
	}
	
	wg.Wait()
	
	// All managers should be non-nil
	for i, mgr := range managers {
		if mgr == nil {
			t.Errorf("Manager %d should not be nil", i)
		}
	}
}

// TestContentionAwareBackoff tests contention-aware backoff
func TestContentionAwareBackoff(t *testing.T) {
	cab := NewContentionAwareBackoff(
		time.Microsecond,
		time.Millisecond,
		2.0,
		10, // contention threshold
	)
	
	// Test backoff under low contention
	cab.Backoff()
	
	// Test backoff under high contention
	for i := 0; i < 15; i++ {
		cab.RecordContention()
	}
	cab.Backoff()
	
	// Should still function normally
	if cab.GetContentionLevel() < 10 {
		t.Error("Expected high contention level")
	}
}

// TestYieldManager tests yield manager functionality
func TestYieldManager(t *testing.T) {
	strategies := []YieldStrategy{
		YieldGosched,
		YieldSleep,
		YieldSpinWait,
		YieldHybrid,
	}
	
	for _, strategy := range strategies {
		t.Run(strategy.String(), func(t *testing.T) {
			ym := NewYieldManager(strategy)
			
			// Test yield (should not panic)
			ym.Yield()
			
			// Test multiple yields
			for i := 0; i < 5; i++ {
				ym.Yield()
			}
		})
	}
}

// TestAdaptiveYieldManager tests adaptive yield manager
func TestAdaptiveYieldManager(t *testing.T) {
	aym := NewAdaptiveYieldManager()
	
	// Test initial strategy
	initialStrategy := aym.GetCurrentStrategy()
	if initialStrategy != YieldGosched {
		t.Errorf("Expected initial strategy %v, got %v", YieldGosched, initialStrategy)
	}
	
	// Test yield
	aym.Yield()
	
	// Test performance recording
	aym.RecordPerformance(time.Millisecond)
	
	// Strategy might change based on performance
	newStrategy := aym.GetCurrentStrategy()
	if newStrategy < YieldGosched || newStrategy > YieldHybrid {
		t.Errorf("Invalid strategy %v", newStrategy)
	}
}

// TestCASWithBackoff tests CAS operations with backoff
func TestCASWithBackoff(t *testing.T) {
	var counter int64
	var attempts int64
	
	// Test CAS with exponential backoff
	success := CASWithExponentialBackoff(&counter, 0, 1, time.Nanosecond, time.Microsecond, 2.0, 10)
	if !success {
		t.Error("CAS should succeed")
	}
	
	if atomic.LoadInt64(&counter) != 1 {
		t.Errorf("Expected counter to be 1, got %d", atomic.LoadInt64(&counter))
	}
	
	// Test CAS with contention management
	var wg sync.WaitGroup
	goroutines := 10
	
	for i := 0; i < goroutines; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := 0; j < 100; j++ {
				for {
					old := atomic.LoadInt64(&counter)
					atomic.AddInt64(&attempts, 1)
					if CASWithContentionManagement(&counter, old, old+1, time.Nanosecond, 2.0) {
						break
					}
				}
			}
		}()
	}
	
	wg.Wait()
	
	expected := int64(1 + goroutines*100)
	if atomic.LoadInt64(&counter) != expected {
		t.Errorf("Expected counter to be %d, got %d", expected, atomic.LoadInt64(&counter))
	}
}

// TestYieldStrategies tests different yield strategies
func TestYieldStrategies(t *testing.T) {
	// Test pause yield
	PauseYield()
	
	// Test CPU relax yield
	CPURelaxYield()
	
	// Test progressive yield
	for i := 0; i < 5; i++ {
		ProgressiveYield(i)
	}
}

// TestCASWithYieldStrategy tests CAS with yield strategies
func TestCASWithYieldStrategy(t *testing.T) {
	var counter int64
	ym := NewYieldManager(YieldGosched)
	
	success := CASWithYieldStrategy(&counter, 0, 1, ym, 10)
	if !success {
		t.Error("CAS should succeed")
	}
	
	if atomic.LoadInt64(&counter) != 1 {
		t.Errorf("Expected counter to be 1, got %d", atomic.LoadInt64(&counter))
	}
}

// TestBackoffConcurrency tests backoff mechanisms under concurrency
func TestBackoffConcurrency(t *testing.T) {
	var counter int64
	var wg sync.WaitGroup
	goroutines := 20
	operationsPerGoroutine := 100
	
	for i := 0; i < goroutines; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			backoff := NewExponentialBackoff(
				time.Nanosecond,
				time.Microsecond,
				2.0,
				true,
			)
			
			for j := 0; j < operationsPerGoroutine; j++ {
				for {
					old := atomic.LoadInt64(&counter)
					if atomic.CompareAndSwapInt64(&counter, old, old+1) {
						backoff.Reset()
						break
					}
					backoff.Backoff()
				}
			}
		}()
	}
	
	wg.Wait()
	
	expected := int64(goroutines * operationsPerGoroutine)
	if atomic.LoadInt64(&counter) != expected {
		t.Errorf("Expected %d, got %d", expected, atomic.LoadInt64(&counter))
	}
}

// TestBackoffPerformance tests backoff performance characteristics
func TestBackoffPerformance(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping performance test in short mode")
	}
	
	var counter int64
	var wg sync.WaitGroup
	goroutines := 10
	operationsPerGoroutine := 1000
	
	// Test with different backoff strategies
	strategies := []struct {
		name string
		fn   func()
	}{
		{"NoBackoff", func() {
			// Busy wait without backoff
		}},
		{"SimpleBackoff", func() {
			time.Sleep(time.Nanosecond)
		}},
		{"ExponentialBackoff", func() {
			backoff := NewExponentialBackoff(
				time.Nanosecond,
				time.Microsecond,
				2.0,
				false,
			)
			backoff.Backoff()
		}},
	}
	
	for _, strategy := range strategies {
		t.Run(strategy.name, func(t *testing.T) {
			atomic.StoreInt64(&counter, 0)
			start := time.Now()
			
			for i := 0; i < goroutines; i++ {
				wg.Add(1)
				go func() {
					defer wg.Done()
					for j := 0; j < operationsPerGoroutine; j++ {
						for {
							old := atomic.LoadInt64(&counter)
							if atomic.CompareAndSwapInt64(&counter, old, old+1) {
								break
							}
							strategy.fn()
						}
					}
				}()
			}
			
			wg.Wait()
			duration := time.Since(start)
			
			expected := int64(goroutines * operationsPerGoroutine)
			if atomic.LoadInt64(&counter) != expected {
				t.Errorf("Expected %d, got %d", expected, atomic.LoadInt64(&counter))
			}
			
			t.Logf("%s completed in %v", strategy.name, duration)
		})
	}
}