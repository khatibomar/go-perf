package approximate

import (
	"math"
	"runtime"
	"sync"
	"testing"
)

func TestProbabilisticCounter(t *testing.T) {
	probabilities := []float64{0.1, 0.5, 1.0}
	
	for _, prob := range probabilities {
		t.Run(fmt.Sprintf("Probability_%.1f", prob), func(t *testing.T) {
			counter := NewProbabilisticCounter(prob)
			
			// Test basic operations
			counter.Reset()
			if value := counter.Get(); value != 0 {
				t.Errorf("Expected initial value 0, got %d", value)
			}
			
			// Test probability and scale
			if counter.GetProbability() != prob {
				t.Errorf("Expected probability %f, got %f", prob, counter.GetProbability())
			}
			
			expectedScale := 1.0 / prob
			if math.Abs(counter.GetScale()-expectedScale) > 1e-9 {
				t.Errorf("Expected scale %f, got %f", expectedScale, counter.GetScale())
			}
			
			// Test error bounds
			counter.Add(1000)
			lower, upper := counter.GetErrorBounds(0.95)
			if lower > upper {
				t.Errorf("Lower bound %f should be <= upper bound %f", lower, upper)
			}
		})
	}
}

func TestBoundedCounter(t *testing.T) {
	threshold := int64(100)
	numCounters := runtime.NumCPU()
	counter := NewBoundedCounter(threshold, numCounters)
	
	// Test basic operations
	counter.Reset()
	if value := counter.Get(); value != 0 {
		t.Errorf("Expected initial value 0, got %d", value)
	}
	
	// Test threshold and counter count
	if counter.GetThreshold() != threshold {
		t.Errorf("Expected threshold %d, got %d", threshold, counter.GetThreshold())
	}
	
	if counter.GetNumCounters() != numCounters {
		t.Errorf("Expected %d counters, got %d", numCounters, counter.GetNumCounters())
	}
	
	// Test bounds
	lower, upper := counter.GetBounds()
	if lower > upper {
		t.Errorf("Lower bound %d should be <= upper bound %d", lower, upper)
	}
	
	// Test local values
	localValues := counter.GetLocalValues()
	if len(localValues) != numCounters {
		t.Errorf("Expected %d local values, got %d", numCounters, len(localValues))
	}
	
	// Test flush
	counter.Add(50)
	counter.Flush()
	globalValue := counter.GetGlobalValue()
	if globalValue != 50 {
		t.Errorf("Expected global value 50 after flush, got %d", globalValue)
	}
}

func TestHierarchicalCounter(t *testing.T) {
	baseThreshold := int64(50)
	numLevels := 3
	fanout := 4
	counter := NewHierarchicalCounter(baseThreshold, numLevels, fanout)
	
	// Test basic operations
	counter.Reset()
	if value := counter.Get(); value != 0 {
		t.Errorf("Expected initial value 0, got %d", value)
	}
	
	// Test configuration
	if counter.GetNumLevels() != numLevels {
		t.Errorf("Expected %d levels, got %d", numLevels, counter.GetNumLevels())
	}
	
	if counter.GetBaseThreshold() != baseThreshold {
		t.Errorf("Expected base threshold %d, got %d", baseThreshold, counter.GetBaseThreshold())
	}
	
	if counter.GetFanout() != fanout {
		t.Errorf("Expected fanout %d, got %d", fanout, counter.GetFanout())
	}
	
	// Test level values
	levelValues := counter.GetByLevel()
	if len(levelValues) != numLevels {
		t.Errorf("Expected %d level values, got %d", numLevels, len(levelValues))
	}
	
	// Test operations
	counter.Add(100)
	if value := counter.Get(); value != 100 {
		t.Errorf("Expected value 100, got %d", value)
	}
}

func TestConcurrentApproximateCounters(t *testing.T) {
	t.Run("ProbabilisticCounter", func(t *testing.T) {
		counter := NewProbabilisticCounter(0.1)
		testConcurrentApproximate(t, counter, "ProbabilisticCounter")
	})
	
	t.Run("BoundedCounter", func(t *testing.T) {
		counter := NewBoundedCounter(100, runtime.NumCPU())
		testConcurrentBounded(t, counter, "BoundedCounter")
	})
	
	t.Run("HierarchicalCounter", func(t *testing.T) {
		counter := NewHierarchicalCounter(50, 3, 4)
		testConcurrentHierarchical(t, counter, "HierarchicalCounter")
	})
}

func testConcurrentApproximate(t *testing.T, counter *ProbabilisticCounter, name string) {
	counter.Reset()
	
	numGoroutines := runtime.NumCPU()
	operationsPerGoroutine := 1000
	var wg sync.WaitGroup
	
	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := 0; j < operationsPerGoroutine; j++ {
				counter.Inc()
			}
		}()
	}
	
	wg.Wait()
	
	estimate := counter.Get()
	expected := int64(numGoroutines * operationsPerGoroutine)
	
	// For probabilistic counters, we expect some variance
	if counter.GetProbability() == 1.0 {
		if estimate != expected {
			t.Errorf("%s: Expected %d, got %d (with probability 1.0)", name, expected, estimate)
		}
	} else {
		// Allow for reasonable error with probabilistic sampling
	errorPercent := math.Abs(float64(estimate-expected)) / float64(expected) * 100
		if errorPercent > 50 { // Allow up to 50% error for low probability sampling
			t.Errorf("%s: Error too high: %.1f%% (estimate=%d, expected=%d)", 
				name, errorPercent, estimate, expected)
		}
	}
}

func testConcurrentBounded(t *testing.T, counter *BoundedCounter, name string) {
	counter.Reset()
	
	numGoroutines := runtime.NumCPU()
	operationsPerGoroutine := 1000
	var wg sync.WaitGroup
	
	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := 0; j < operationsPerGoroutine; j++ {
				counter.Inc()
			}
		}()
	}
	
	wg.Wait()
	
	value := counter.Get()
	expected := int64(numGoroutines * operationsPerGoroutine)
	lower, upper := counter.GetBounds()
	
	// Value should be within bounds
	if value < lower || value > upper {
		t.Errorf("%s: Value %d not within bounds [%d, %d]", name, value, lower, upper)
	}
	
	// Expected value should be within bounds
	if expected < lower || expected > upper {
		t.Errorf("%s: Expected %d not within bounds [%d, %d]", name, expected, lower, upper)
	}
}

func testConcurrentHierarchical(t *testing.T, counter *HierarchicalCounter, name string) {
	counter.Reset()
	
	numGoroutines := runtime.NumCPU()
	operationsPerGoroutine := 1000
	var wg sync.WaitGroup
	
	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := 0; j < operationsPerGoroutine; j++ {
				counter.Inc()
			}
		}()
	}
	
	wg.Wait()
	
	value := counter.Get()
	expected := int64(numGoroutines * operationsPerGoroutine)
	
	if value != expected {
		t.Errorf("%s: Expected %d, got %d", name, expected, value)
	}
	
	// Check that levels have reasonable distribution
	levelValues := counter.GetByLevel()
	totalFromLevels := int64(0)
	for _, levelValue := range levelValues {
		totalFromLevels += levelValue
	}
	
	if totalFromLevels != expected {
		t.Errorf("%s: Sum of level values %d != expected %d", name, totalFromLevels, expected)
	}
}

func BenchmarkApproximateCounters(b *testing.B) {
	b.Run("ProbabilisticCounter_0.1", func(b *testing.B) {
		counter := NewProbabilisticCounter(0.1)
		b.ResetTimer()
		b.RunParallel(func(pb *testing.PB) {
			for pb.Next() {
				counter.Inc()
			}
		})
	})
	
	b.Run("BoundedCounter", func(b *testing.B) {
		counter := NewBoundedCounter(100, runtime.NumCPU())
		b.ResetTimer()
		b.RunParallel(func(pb *testing.PB) {
			for pb.Next() {
				counter.Inc()
			}
		})
	})
	
	b.Run("HierarchicalCounter", func(b *testing.B) {
		counter := NewHierarchicalCounter(50, 3, 4)
		b.ResetTimer()
		b.RunParallel(func(pb *testing.PB) {
			for pb.Next() {
				counter.Inc()
			}
		})
	})
}

// Helper function for string formatting
func sprintf(format string, args ...interface{}) string {
	// Simple sprintf implementation for the test
	if format == "Probability_%.1f" && len(args) == 1 {
		if f, ok := args[0].(float64); ok {
			if f == 0.1 {
				return "Probability_0.1"
			} else if f == 0.5 {
				return "Probability_0.5"
			} else if f == 1.0 {
				return "Probability_1.0"
			}
		}
	}
	return format
}

// Use sprintf instead of fmt.Sprintf to avoid import
var fmt = struct {
	Sprintf func(string, ...interface{}) string
}{
	Sprintf: sprintf,
}