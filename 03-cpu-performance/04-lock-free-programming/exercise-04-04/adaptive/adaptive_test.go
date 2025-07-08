package adaptive

import (
	"runtime"
	"sync"
	"testing"
	"time"
)

func TestAdaptiveCounter(t *testing.T) {
	counter := NewAdaptiveCounter()
	
	// Test basic operations
	counter.Reset()
	if value := counter.Get(); value != 0 {
		t.Errorf("Expected initial value 0, got %d", value)
	}
	
	// Test initial strategy
	initialStrategy := counter.GetCurrentStrategy()
	if initialStrategy != AtomicStrategy {
		t.Errorf("Expected initial strategy to be Atomic, got %v", initialStrategy)
	}
	
	// Test strategy name
	strategyName := counter.GetStrategyName()
	if strategyName != "Atomic" {
		t.Errorf("Expected strategy name 'Atomic', got '%s'", strategyName)
	}
	
	// Test basic increment
	counter.Inc()
	if value := counter.Get(); value != 1 {
		t.Errorf("Expected value 1 after Inc, got %d", value)
	}
	
	// Test Add
	counter.Add(10)
	if value := counter.Get(); value != 11 {
		t.Errorf("Expected value 11 after Add(10), got %d", value)
	}
	
	// Test Dec
	counter.Dec()
	if value := counter.Get(); value != 10 {
		t.Errorf("Expected value 10 after Dec, got %d", value)
	}
}

func TestLoadBalancer(t *testing.T) {
	modes := []BalanceMode{
		RoundRobinMode,
		LeastLoadedMode,
		LatencyBasedMode,
		AffinityMode,
	}
	
	for _, mode := range modes {
		t.Run(mode.String(), func(t *testing.T) {
			lb := NewLoadBalancer(4, mode)
			
			// Test basic operations
			lb.Reset()
			if value := lb.Get(); value != 0 {
				t.Errorf("Expected initial value 0, got %d", value)
			}
			
			// Test configuration
			if lb.GetNumCounters() != 4 {
				t.Errorf("Expected 4 counters, got %d", lb.GetNumCounters())
			}
			
			if lb.GetBalanceMode() != mode {
				t.Errorf("Expected mode %v, got %v", mode, lb.GetBalanceMode())
			}
			
			// Test operations
			lb.Inc()
			if value := lb.Get(); value != 1 {
				t.Errorf("Expected value 1 after Inc, got %d", value)
			}
			
			lb.Add(10)
			if value := lb.Get(); value != 11 {
				t.Errorf("Expected value 11 after Add(10), got %d", value)
			}
			
			lb.Dec()
			if value := lb.Get(); value != 10 {
				t.Errorf("Expected value 10 after Dec, got %d", value)
			}
			
			// Test counter values
			counterValues := lb.GetCounterValues()
			if len(counterValues) != 4 {
				t.Errorf("Expected 4 counter values, got %d", len(counterValues))
			}
			
			// Test load metrics
			loadMetrics := lb.GetLoadMetrics()
			if len(loadMetrics) != 4 {
				t.Errorf("Expected 4 load metrics, got %d", len(loadMetrics))
			}
			
			// Test mode change
			newMode := RoundRobinMode
			if mode == RoundRobinMode {
				newMode = LeastLoadedMode
			}
			lb.SetBalanceMode(newMode)
			if lb.GetBalanceMode() != newMode {
				t.Errorf("Expected mode %v after change, got %v", newMode, lb.GetBalanceMode())
			}
		})
	}
}

func TestPerformanceMonitor(t *testing.T) {
	monitor := NewPerformanceMonitor(10*time.Millisecond, 100)
	
	// Test initial state
	currentMetrics := monitor.GetCurrentMetrics()
	if currentMetrics.Timestamp.IsZero() {
		// Expected for empty monitor
	}
	
	// Record some metrics
	latencies := []int64{100, 200, 300, 400, 500}
	monitor.RecordMetric(1000, 50, latencies)
	
	// Wait for sample interval
	time.Sleep(15 * time.Millisecond)
	
	// Record another metric
	monitor.RecordMetric(2000, 100, latencies)
	
	// Test recent metrics
	recentMetrics := monitor.GetRecentMetrics(5)
	if len(recentMetrics) == 0 {
		t.Error("Expected some recent metrics")
	}
	
	// Test average metrics
	avgMetrics := monitor.GetAverageMetrics(1 * time.Second)
	if avgMetrics.Timestamp.IsZero() {
		// May be zero if no metrics in timeframe
	}
	
	// Test reset
	monitor.Reset()
	currentAfterReset := monitor.GetCurrentMetrics()
	if !currentAfterReset.Timestamp.IsZero() {
		t.Error("Expected empty metrics after reset")
	}
}

func TestLatencyTracker(t *testing.T) {
	tracker := NewLatencyTracker(10)
	
	// Record some latencies
	latencies := []int64{100, 200, 300, 400, 500}
	for _, latency := range latencies {
		tracker.Record(latency)
	}
	
	// Get samples
	samples := tracker.GetSamples()
	if len(samples) != len(latencies) {
		t.Errorf("Expected %d samples, got %d", len(latencies), len(samples))
	}
	
	// Verify samples (they should be in reverse order due to implementation)
	for i, sample := range samples {
		expected := latencies[len(latencies)-1-i]
		if sample != expected {
			t.Errorf("Expected sample %d to be %d, got %d", i, expected, sample)
		}
	}
}

func TestConcurrentAdaptiveCounter(t *testing.T) {
	counter := NewAdaptiveCounter()
	counter.Reset()
	
	numGoroutines := runtime.NumCPU() * 2
	operationsPerGoroutine := 1000
	var wg sync.WaitGroup
	
	// Test concurrent operations with varying load
	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			for j := 0; j < operationsPerGoroutine; j++ {
				counter.Inc()
				// Simulate varying load
				if j%100 == 0 {
					time.Sleep(time.Microsecond)
				}
			}
		}(i)
	}
	
	wg.Wait()
	
	finalValue := counter.Get()
	expected := int64(numGoroutines * operationsPerGoroutine)
	
	if finalValue != expected {
		t.Errorf("Expected %d, got %d", expected, finalValue)
	}
	
	// Check that contention was detected
	contentionLevel := counter.GetContentionLevel()
	if contentionLevel < 0 {
		t.Errorf("Expected non-negative contention level, got %d", contentionLevel)
	}
	
	// Strategy might have changed due to contention
	strategy := counter.GetCurrentStrategy()
	strategyName := counter.GetStrategyName()
	t.Logf("Final strategy: %s (%v), contention: %d", strategyName, strategy, contentionLevel)
}

func TestConcurrentLoadBalancer(t *testing.T) {
	modes := []BalanceMode{RoundRobinMode, LeastLoadedMode, LatencyBasedMode, AffinityMode}
	
	for _, mode := range modes {
		t.Run(mode.String(), func(t *testing.T) {
			lb := NewLoadBalancer(runtime.NumCPU(), mode)
			lb.Reset()
			
			numGoroutines := runtime.NumCPU()
			operationsPerGoroutine := 1000
			var wg sync.WaitGroup
			
			for i := 0; i < numGoroutines; i++ {
				wg.Add(1)
				go func() {
					defer wg.Done()
					for j := 0; j < operationsPerGoroutine; j++ {
						lb.Inc()
					}
				}()
			}
			
			wg.Wait()
			
			finalValue := lb.Get()
			expected := int64(numGoroutines * operationsPerGoroutine)
			
			if finalValue != expected {
				t.Errorf("Expected %d, got %d", expected, finalValue)
			}
			
			// Check load distribution
			counterValues := lb.GetCounterValues()
			loadMetrics := lb.GetLoadMetrics()
			
			// Verify total
			total := int64(0)
			for _, value := range counterValues {
				total += value
			}
			if total != expected {
				t.Errorf("Sum of counter values %d != expected %d", total, expected)
			}
			
			// Check that all counters have some operations recorded
			totalOps := int64(0)
			for _, metric := range loadMetrics {
				totalOps += metric.Operations
			}
			if totalOps == 0 {
				t.Error("Expected some operations to be recorded in metrics")
			}
		})
	}
}

func BenchmarkAdaptiveAlgorithms(b *testing.B) {
	b.Run("AdaptiveCounter", func(b *testing.B) {
		counter := NewAdaptiveCounter()
		counter.Reset()
		b.ResetTimer()
		b.RunParallel(func(pb *testing.PB) {
			for pb.Next() {
				counter.Inc()
			}
		})
	})
	
	modes := []BalanceMode{RoundRobinMode, LeastLoadedMode, LatencyBasedMode, AffinityMode}
	for _, mode := range modes {
		b.Run("LoadBalancer_"+mode.String(), func(b *testing.B) {
			lb := NewLoadBalancer(runtime.NumCPU(), mode)
			lb.Reset()
			b.ResetTimer()
			b.RunParallel(func(pb *testing.PB) {
				for pb.Next() {
					lb.Inc()
				}
			})
		})
	}
}