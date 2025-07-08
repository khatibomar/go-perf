package monitoring

import (
	"sync"
	"testing"
	"time"
)

// TestMetricsCollector tests the basic metrics collection functionality
func TestMetricsCollector(t *testing.T) {
	collector := NewMetricsCollector()
	
	// Test counter operations
	t.Run("Counter", func(t *testing.T) {
		collector.IncrementCounter("test_counter")
		collector.IncrementCounter("test_counter")
		collector.AddToCounter("test_counter", 3)
		
		value := collector.GetCounter("test_counter")
		if value != 5 {
			t.Errorf("Expected counter value 5, got %d", value)
		}
	})
	
	// Test gauge operations
	t.Run("Gauge", func(t *testing.T) {
		collector.SetGauge("test_gauge", 42.5)
		collector.SetGauge("test_gauge", 100.0)
		
		value := collector.GetGauge("test_gauge")
		if value != 100.0 {
			t.Errorf("Expected gauge value 100.0, got %f", value)
		}
	})
	
	// Test histogram operations
	t.Run("Histogram", func(t *testing.T) {
		collector.RecordHistogram("test_histogram", 10.0)
		collector.RecordHistogram("test_histogram", 20.0)
		collector.RecordHistogram("test_histogram", 30.0)
		
		stats := collector.GetHistogramStats("test_histogram")
		if stats.Count != 3 {
			t.Errorf("Expected histogram count 3, got %d", stats.Count)
		}
		
		if stats.Mean != 20.0 {
			t.Errorf("Expected histogram mean 20.0, got %f", stats.Mean)
		}
	})
	
	// Test timer operations
	t.Run("Timer", func(t *testing.T) {
		timer := collector.StartTimer("test_timer")
		time.Sleep(10 * time.Millisecond)
		timer.Stop()
		
		stats := collector.GetTimerStats("test_timer")
		if stats.Count != 1 {
			t.Errorf("Expected timer count 1, got %d", stats.Count)
		}
		
		if stats.Mean < 5*time.Millisecond {
			t.Errorf("Expected timer mean >= 5ms, got %v", stats.Mean)
		}
	})
}

// TestMetricsCollectorConcurrency tests concurrent access to metrics
func TestMetricsCollectorConcurrency(t *testing.T) {
	collector := NewMetricsCollector()
	
	const (
		numGoroutines = 10
		operationsPerGoroutine = 100
	)
	
	var wg sync.WaitGroup
	
	// Test concurrent counter operations
	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := 0; j < operationsPerGoroutine; j++ {
				collector.IncrementCounter("concurrent_counter")
			}
		}()
	}
	
	// Test concurrent gauge operations
	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			for j := 0; j < operationsPerGoroutine; j++ {
				collector.SetGauge("concurrent_gauge", float64(id*operationsPerGoroutine+j))
			}
		}(i)
	}
	
	// Test concurrent histogram operations
	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			for j := 0; j < operationsPerGoroutine; j++ {
				collector.RecordHistogram("concurrent_histogram", float64(id*operationsPerGoroutine+j))
			}
		}(i)
	}
	
	wg.Wait()
	
	// Verify results
	counterValue := collector.GetCounter("concurrent_counter")
	expectedCounter := int64(numGoroutines * operationsPerGoroutine)
	if counterValue != expectedCounter {
		t.Errorf("Expected counter value %d, got %d", expectedCounter, counterValue)
	}
	
	histogramStats := collector.GetHistogramStats("concurrent_histogram")
	expectedHistogramCount := int64(numGoroutines * operationsPerGoroutine)
	if histogramStats.Count != expectedHistogramCount {
		t.Errorf("Expected histogram count %d, got %d", expectedHistogramCount, histogramStats.Count)
	}
	
	t.Logf("Concurrent test results - Counter: %d, Histogram count: %d, Gauge: %f",
		counterValue, histogramStats.Count, collector.GetGauge("concurrent_gauge"))
}

// TestQueueMonitor tests the queue-specific monitoring functionality
func TestQueueMonitor(t *testing.T) {
	monitor := NewQueueMonitor()
	
	// Test enqueue recording
	t.Run("EnqueueOperations", func(t *testing.T) {
		monitor.RecordEnqueue(10 * time.Millisecond, true)
		monitor.RecordEnqueue(20 * time.Millisecond, true)
		monitor.RecordEnqueue(15 * time.Millisecond, false) // Failed enqueue
		
		report := monitor.GenerateReport()
		if report.TotalEnqueues != 3 {
			t.Errorf("Expected 3 total enqueues, got %d", report.TotalEnqueues)
		}
		
		if report.SuccessfulEnqueues != 2 {
			t.Errorf("Expected 2 successful enqueues, got %d", report.SuccessfulEnqueues)
		}
		
		if report.FailedEnqueues != 1 {
			t.Errorf("Expected 1 failed enqueue, got %d", report.FailedEnqueues)
		}
	})
	
	// Test dequeue recording
	t.Run("DequeueOperations", func(t *testing.T) {
		monitor.RecordDequeue(5 * time.Millisecond, true)
		monitor.RecordDequeue(8 * time.Millisecond, true)
		monitor.RecordDequeue(3 * time.Millisecond, false) // Failed dequeue
		
		report := monitor.GenerateReport()
		if report.TotalDequeues != 3 {
			t.Errorf("Expected 3 total dequeues, got %d", report.TotalDequeues)
		}
		
		if report.SuccessfulDequeues != 2 {
			t.Errorf("Expected 2 successful dequeues, got %d", report.SuccessfulDequeues)
		}
		
		if report.FailedDequeues != 1 {
			t.Errorf("Expected 1 failed dequeue, got %d", report.FailedDequeues)
		}
	})
	
	// Test queue size recording
	t.Run("QueueSize", func(t *testing.T) {
		monitor.RecordQueueSize(10)
		monitor.RecordQueueSize(20)
		monitor.RecordQueueSize(5)
		
		report := monitor.GenerateReport()
		if report.CurrentQueueSize != 5 {
			t.Errorf("Expected current queue size 5, got %d", report.CurrentQueueSize)
		}
		
		if report.MaxQueueSize != 20 {
			t.Errorf("Expected max queue size 20, got %d", report.MaxQueueSize)
		}
	})
	
	// Test contention recording
	t.Run("Contention", func(t *testing.T) {
		monitor.RecordContention("enqueue", 3*time.Millisecond)
		monitor.RecordContention("dequeue", 5*time.Millisecond)
		monitor.RecordContention("enqueue", 2*time.Millisecond)
		
		report := monitor.GenerateReport()
		if report.ContentionEvents != 3 {
			t.Errorf("Expected 3 contention events, got %d", report.ContentionEvents)
		}
	})
}

// TestQueueMonitorConcurrency tests concurrent monitoring
func TestQueueMonitorConcurrency(t *testing.T) {
	monitor := NewQueueMonitor()
	
	const (
		numGoroutines = 5
		operationsPerGoroutine = 200
	)
	
	var wg sync.WaitGroup
	
	// Concurrent enqueue recording
	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			for j := 0; j < operationsPerGoroutine; j++ {
				duration := time.Duration(j%10+1) * time.Millisecond
				success := j%10 != 0 // 90% success rate
				monitor.RecordEnqueue(duration, success)
			}
		}(i)
	}
	
	// Concurrent dequeue recording
	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			for j := 0; j < operationsPerGoroutine; j++ {
				duration := time.Duration(j%8+1) * time.Millisecond
				success := j%8 != 0 // ~87.5% success rate
				monitor.RecordDequeue(duration, success)
			}
		}(i)
	}
	
	// Concurrent queue size updates
	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			for j := 0; j < operationsPerGoroutine/10; j++ {
				size := (id*operationsPerGoroutine + j) % 100
				monitor.RecordQueueSize(int64(size))
			}
		}(i)
	}
	
	wg.Wait()
	
	report := monitor.GenerateReport()
	
	expectedEnqueues := int64(numGoroutines * operationsPerGoroutine)
	expectedDequeues := int64(numGoroutines * operationsPerGoroutine)
	
	if report.TotalEnqueues != expectedEnqueues {
		t.Errorf("Expected %d total enqueues, got %d", expectedEnqueues, report.TotalEnqueues)
	}
	
	if report.TotalDequeues != expectedDequeues {
		t.Errorf("Expected %d total dequeues, got %d", expectedDequeues, report.TotalDequeues)
	}
	
	t.Logf("Concurrent monitoring results: Enqueues=%d, Dequeues=%d, Max queue size=%d",
		report.TotalEnqueues, report.TotalDequeues, report.MaxQueueSize)
}

// TestPeriodicReporting tests the periodic reporting functionality
func TestPeriodicReporting(t *testing.T) {
	monitor := NewQueueMonitor()
	
	// Start periodic reporting
	reportChan := make(chan QueueReport, 5)
	stop := monitor.StartPeriodicReporting(50*time.Millisecond, func(report QueueReport) {
		select {
		case reportChan <- report:
		default:
			// Channel full, skip
		}
	})
	defer stop()
	
	// Generate some activity
	for i := 0; i < 10; i++ {
		monitor.RecordEnqueue(time.Millisecond, true)
		monitor.RecordDequeue(time.Millisecond, true)
		monitor.RecordQueueSize(int64(i))
		time.Sleep(10 * time.Millisecond)
	}
	
	// Wait for at least one report
	select {
	case report := <-reportChan:
		if report.TotalEnqueues == 0 {
			t.Error("Expected some enqueues in periodic report")
		}
		t.Logf("Periodic report: %+v", report)
	case <-time.After(200 * time.Millisecond):
		t.Error("Expected to receive at least one periodic report")
	}
}

// TestHistogram tests the histogram implementation
func TestHistogram(t *testing.T) {
	hist := NewHistogram()
	
	// Record some values
	values := []float64{1.0, 2.0, 3.0, 4.0, 5.0, 10.0, 15.0, 20.0}
	for _, v := range values {
		hist.Record(v)
	}
	
	stats := hist.Stats()
	
	if stats.Count != int64(len(values)) {
		t.Errorf("Expected count %d, got %d", len(values), stats.Count)
	}
	
	if stats.Min != 1.0 {
		t.Errorf("Expected min 1.0, got %f", stats.Min)
	}
	
	if stats.Max != 20.0 {
		t.Errorf("Expected max 20.0, got %f", stats.Max)
	}
	
	expectedMean := 7.5 // (1+2+3+4+5+10+15+20)/8
	if stats.Mean != expectedMean {
		t.Errorf("Expected mean %f, got %f", expectedMean, stats.Mean)
	}
	
	// Test percentiles
	if stats.P50 == 0 {
		t.Error("P50 should be greater than 0")
	}
	
	if stats.P95 == 0 {
		t.Error("P95 should be greater than 0")
	}
	
	if stats.P99 == 0 {
		t.Error("P99 should be greater than 0")
	}
	
	t.Logf("Histogram stats: %+v", stats)
}

// TestTimer tests the timer implementation
func TestTimer(t *testing.T) {
	timer := NewTimer()
	
	// Record some durations
	durations := []time.Duration{
		1 * time.Millisecond,
		2 * time.Millisecond,
		5 * time.Millisecond,
		10 * time.Millisecond,
	}
	
	for _, d := range durations {
		timer.Record(d)
	}
	
	stats := timer.Stats()
	
	if stats.Count != int64(len(durations)) {
		t.Errorf("Expected count %d, got %d", len(durations), stats.Count)
	}
	
	if stats.Min != 1*time.Millisecond {
		t.Errorf("Expected min 1ms, got %v", stats.Min)
	}
	
	if stats.Max != 10*time.Millisecond {
		t.Errorf("Expected max 10ms, got %v", stats.Max)
	}
	
	expectedMean := 4500 * time.Microsecond // (1+2+5+10)/4 = 4.5ms
	if stats.Mean != expectedMean {
		t.Errorf("Expected mean %v, got %v", expectedMean, stats.Mean)
	}
	
	t.Logf("Timer stats: %+v", stats)
}

// TestTimerConcurrency tests concurrent timer usage
func TestTimerConcurrency(t *testing.T) {
	timer := NewTimer()
	
	const (
		numGoroutines = 10
		recordsPerGoroutine = 100
	)
	
	var wg sync.WaitGroup
	
	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			for j := 0; j < recordsPerGoroutine; j++ {
				duration := time.Duration(id*recordsPerGoroutine+j) * time.Microsecond
				timer.Record(duration)
			}
		}(i)
	}
	
	wg.Wait()
	
	stats := timer.Stats()
	expectedCount := int64(numGoroutines * recordsPerGoroutine)
	
	if stats.Count != expectedCount {
		t.Errorf("Expected count %d, got %d", expectedCount, stats.Count)
	}
	
	t.Logf("Concurrent timer stats: Count=%d, Mean=%v, Min=%v, Max=%v",
		stats.Count, stats.Mean, stats.Min, stats.Max)
}

// BenchmarkMetricsCollector benchmarks the metrics collector performance
func BenchmarkMetricsCollector(b *testing.B) {
	collector := NewMetricsCollector()
	
	b.Run("Counter", func(b *testing.B) {
		b.RunParallel(func(pb *testing.PB) {
			for pb.Next() {
				collector.IncrementCounter("bench_counter")
			}
		})
	})
	
	b.Run("Gauge", func(b *testing.B) {
		b.RunParallel(func(pb *testing.PB) {
			for pb.Next() {
				collector.SetGauge("bench_gauge", 42.0)
			}
		})
	})
	
	b.Run("Histogram", func(b *testing.B) {
		b.RunParallel(func(pb *testing.PB) {
			for pb.Next() {
				collector.RecordHistogram("bench_histogram", 42.0)
			}
		})
	})
	
	b.Run("Timer", func(b *testing.B) {
		b.RunParallel(func(pb *testing.PB) {
			for pb.Next() {
				timer := collector.StartTimer("bench_timer")
				timer.Stop()
			}
		})
	})
}

// BenchmarkQueueMonitor benchmarks the queue monitor performance
func BenchmarkQueueMonitor(b *testing.B) {
	monitor := NewQueueMonitor()
	
	b.Run("RecordEnqueue", func(b *testing.B) {
		b.RunParallel(func(pb *testing.PB) {
			for pb.Next() {
				monitor.RecordEnqueue(time.Millisecond, true)
			}
		})
	})
	
	b.Run("RecordDequeue", func(b *testing.B) {
		b.RunParallel(func(pb *testing.PB) {
			for pb.Next() {
				monitor.RecordDequeue(time.Millisecond, true)
			}
		})
	})
	
	b.Run("RecordQueueSize", func(b *testing.B) {
		b.RunParallel(func(pb *testing.PB) {
			for pb.Next() {
				monitor.RecordQueueSize(42)
			}
		})
	})
	
	b.Run("GenerateReport", func(b *testing.B) {
		// Add some data first
		for i := 0; i < 1000; i++ {
			monitor.RecordEnqueue(time.Millisecond, true)
			monitor.RecordDequeue(time.Millisecond, true)
		}
		
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			monitor.GenerateReport()
		}
	})
}