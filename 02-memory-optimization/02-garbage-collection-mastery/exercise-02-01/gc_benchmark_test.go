package main

import (
	"fmt"
	"runtime"
	"runtime/debug"
	"strings"
	"sync"
	"testing"
	"time"
)

// validateBenchmarkEnvironment ensures the benchmark environment is suitable
func validateBenchmarkEnvironment(b *testing.B) {
	if testing.Short() {
		b.Skip("Skipping GC benchmark in short mode")
	}
	
	// Ensure we have enough memory
	var m runtime.MemStats
	runtime.ReadMemStats(&m)
	if m.Sys < 64*1024*1024 { // Less than 64MB
		b.Skip("Insufficient memory for GC benchmarks")
	}
}

// BenchmarkGCPausePatterns benchmarks different allocation patterns and their GC impact
func BenchmarkGCPausePatterns(b *testing.B) {
	validateBenchmarkEnvironment(b)
	tests := []struct {
		name    string
		pattern func(int)
	}{
		{
			name: "SmallObjectsHighFreq",
			pattern: func(n int) {
				for i := 0; i < n; i++ {
					_ = make([]byte, 64)
				}
			},
		},
		{
			name: "MediumObjectsModerateFreq",
			pattern: func(n int) {
				for i := 0; i < n; i++ {
					_ = make([]byte, 8192)
				}
			},
		},
		{
			name: "LargeObjectsLowFreq",
			pattern: func(n int) {
				for i := 0; i < n; i++ {
					_ = make([]byte, 1024*1024)
				}
			},
		},
		{
			name: "MixedSizeObjects",
			pattern: func(n int) {
				for i := 0; i < n; i++ {
					size := 64 + (i%1000)*32
					_ = make([]byte, size)
				}
			},
		},
		{
			name: "StringConcatenation",
			pattern: func(n int) {
				for i := 0; i < n; i++ {
					result := ""
					for j := 0; j < 10; j++ {
						result += fmt.Sprintf("item_%d_", j)
					}
					_ = result
				}
			},
		},
		{
			name: "SliceGrowth",
			pattern: func(n int) {
				for i := 0; i < n; i++ {
					slice := make([]int, 0)
					for j := 0; j < 100; j++ {
						slice = append(slice, j)
					}
					_ = slice
				}
			},
		},
		{
			name: "MapAllocations",
			pattern: func(n int) {
				for i := 0; i < n; i++ {
					m := make(map[string]int)
					for j := 0; j < 50; j++ {
						key := fmt.Sprintf("key_%d", j)
						m[key] = j
					}
					_ = m
				}
			},
		},
		{
			name: "InterfaceBoxing",
			pattern: func(n int) {
				for i := 0; i < n; i++ {
					var interfaces []interface{}
					for j := 0; j < 50; j++ {
						interfaces = append(interfaces, j)
					}
					_ = interfaces
				}
			},
		},
	}
	
	for _, tt := range tests {
		b.Run(tt.name, func(b *testing.B) {
			// Clear GC stats before benchmark
			runtime.GC()
			var startStats, endStats runtime.MemStats
			runtime.ReadMemStats(&startStats)
			
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				tt.pattern(100) // Run pattern with 100 iterations
			}
			b.StopTimer()
			
			runtime.ReadMemStats(&endStats)
			
			// Report GC statistics
			gcCycles := endStats.NumGC - startStats.NumGC
			totalPause := endStats.PauseTotalNs - startStats.PauseTotalNs
			allocated := endStats.TotalAlloc - startStats.TotalAlloc
			
			b.ReportMetric(float64(gcCycles), "gc-cycles")
			b.ReportMetric(float64(totalPause)/1e6, "gc-pause-ms")
			b.ReportMetric(float64(allocated)/(1024*1024), "allocated-mb")
			if gcCycles > 0 {
				b.ReportMetric(float64(totalPause)/float64(gcCycles)/1e6, "avg-pause-ms")
			}
		})
	}
}

// BenchmarkGCTuning benchmarks different GC tuning parameters
func BenchmarkGCTuning(b *testing.B) {
	validateBenchmarkEnvironment(b)
	gcPercents := []int{50, 100, 200, 400, 800}
	
	for _, gcPercent := range gcPercents {
		b.Run(fmt.Sprintf("GOGC_%d", gcPercent), func(b *testing.B) {
			// Save original GOGC
			originalGOGC := debug.SetGCPercent(gcPercent)
			defer debug.SetGCPercent(originalGOGC)
			
			// Clear GC stats
			runtime.GC()
			var startStats, endStats runtime.MemStats
			runtime.ReadMemStats(&startStats)
			
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				// Mixed allocation pattern
				for j := 0; j < 1000; j++ {
					if j%10 == 0 {
						_ = make([]byte, 8192) // Medium objects
					} else {
						_ = make([]byte, 64) // Small objects
					}
				}
			}
			b.StopTimer()
			
			runtime.ReadMemStats(&endStats)
			
			// Report metrics
			gcCycles := endStats.NumGC - startStats.NumGC
			totalPause := endStats.PauseTotalNs - startStats.PauseTotalNs
			allocated := endStats.TotalAlloc - startStats.TotalAlloc
			heapSize := endStats.HeapSys
			
			b.ReportMetric(float64(gcCycles), "gc-cycles")
			b.ReportMetric(float64(totalPause)/1e6, "gc-pause-ms")
			b.ReportMetric(float64(allocated)/(1024*1024), "allocated-mb")
			b.ReportMetric(float64(heapSize)/(1024*1024), "heap-size-mb")
			if gcCycles > 0 {
				b.ReportMetric(float64(totalPause)/float64(gcCycles)/1e6, "avg-pause-ms")
			}
		})
	}
}

// BenchmarkGCConcurrency benchmarks GC behavior under concurrent load
func BenchmarkGCConcurrency(b *testing.B) {
	validateBenchmarkEnvironment(b)
	workerCounts := []int{1, 2, 4, 8, 16}
	
	for _, workers := range workerCounts {
		b.Run(fmt.Sprintf("Workers_%d", workers), func(b *testing.B) {
			// Clear GC stats
			runtime.GC()
			var startStats, endStats runtime.MemStats
			runtime.ReadMemStats(&startStats)
			
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				var wg sync.WaitGroup
				for w := 0; w < workers; w++ {
					wg.Add(1)
					go func() {
						defer wg.Done()
						for j := 0; j < 1000/workers; j++ {
							_ = make([]byte, 1024)
							if j%10 == 0 {
								runtime.Gosched() // Yield to scheduler
							}
						}
					}()
				}
				wg.Wait()
			}
			b.StopTimer()
			
			runtime.ReadMemStats(&endStats)
			
			// Report metrics
			gcCycles := endStats.NumGC - startStats.NumGC
			totalPause := endStats.PauseTotalNs - startStats.PauseTotalNs
			allocated := endStats.TotalAlloc - startStats.TotalAlloc
			
			b.ReportMetric(float64(gcCycles), "gc-cycles")
			b.ReportMetric(float64(totalPause)/1e6, "gc-pause-ms")
			b.ReportMetric(float64(allocated)/(1024*1024), "allocated-mb")
			if gcCycles > 0 {
				b.ReportMetric(float64(totalPause)/float64(gcCycles)/1e6, "avg-pause-ms")
			}
		})
	}
}

// BenchmarkGCLargeObjects benchmarks GC behavior with large objects
func BenchmarkGCLargeObjects(b *testing.B) {
	sizes := []int{32 * 1024, 64 * 1024, 128 * 1024, 256 * 1024, 512 * 1024, 1024 * 1024}
	
	for _, size := range sizes {
		b.Run(fmt.Sprintf("Size_%dKB", size/1024), func(b *testing.B) {
			// Clear GC stats
			runtime.GC()
			var startStats, endStats runtime.MemStats
			runtime.ReadMemStats(&startStats)
			
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				// Allocate and immediately discard large objects
				for j := 0; j < 10; j++ {
					_ = make([]byte, size)
				}
			}
			b.StopTimer()
			
			runtime.ReadMemStats(&endStats)
			
			// Report metrics
			gcCycles := endStats.NumGC - startStats.NumGC
			totalPause := endStats.PauseTotalNs - startStats.PauseTotalNs
			allocated := endStats.TotalAlloc - startStats.TotalAlloc
			heapSize := endStats.HeapSys
			
			b.ReportMetric(float64(gcCycles), "gc-cycles")
			b.ReportMetric(float64(totalPause)/1e6, "gc-pause-ms")
			b.ReportMetric(float64(allocated)/(1024*1024), "allocated-mb")
			b.ReportMetric(float64(heapSize)/(1024*1024), "heap-size-mb")
			if gcCycles > 0 {
				b.ReportMetric(float64(totalPause)/float64(gcCycles)/1e6, "avg-pause-ms")
			}
		})
	}
}

// BenchmarkGCRetention benchmarks GC behavior with different object retention patterns
func BenchmarkGCRetention(b *testing.B) {
	tests := []struct {
		name     string
		retainer func(int) []interface{}
	}{
		{
			name: "NoRetention",
			retainer: func(n int) []interface{} {
				for i := 0; i < n; i++ {
					_ = make([]byte, 1024)
				}
				return nil
			},
		},
		{
			name: "ShortRetention",
			retainer: func(n int) []interface{} {
				var retained []interface{}
				for i := 0; i < n; i++ {
					obj := make([]byte, 1024)
					if i%10 == 0 { // Retain 10%
						retained = append(retained, obj)
					}
				}
				return retained
			},
		},
		{
			name: "LongRetention",
			retainer: func(n int) []interface{} {
				var retained []interface{}
				for i := 0; i < n; i++ {
					obj := make([]byte, 1024)
					if i%2 == 0 { // Retain 50%
						retained = append(retained, obj)
					}
				}
				return retained
			},
		},
		{
			name: "FullRetention",
			retainer: func(n int) []interface{} {
				var retained []interface{}
				for i := 0; i < n; i++ {
					obj := make([]byte, 1024)
					retained = append(retained, obj)
				}
				return retained
			},
		},
	}
	
	for _, tt := range tests {
		b.Run(tt.name, func(b *testing.B) {
			// Clear GC stats
			runtime.GC()
			var startStats, endStats runtime.MemStats
			runtime.ReadMemStats(&startStats)
			
			b.ResetTimer()
			var retained []interface{}
			for i := 0; i < b.N; i++ {
				retained = tt.retainer(1000)
			}
			b.StopTimer()
			
			// Keep retained objects alive
			_ = retained
			
			runtime.ReadMemStats(&endStats)
			
			// Report metrics
			gcCycles := endStats.NumGC - startStats.NumGC
			totalPause := endStats.PauseTotalNs - startStats.PauseTotalNs
			allocated := endStats.TotalAlloc - startStats.TotalAlloc
			heapInUse := endStats.HeapInuse
			
			b.ReportMetric(float64(gcCycles), "gc-cycles")
			b.ReportMetric(float64(totalPause)/1e6, "gc-pause-ms")
			b.ReportMetric(float64(allocated)/(1024*1024), "allocated-mb")
			b.ReportMetric(float64(heapInUse)/(1024*1024), "heap-inuse-mb")
			if gcCycles > 0 {
				b.ReportMetric(float64(totalPause)/float64(gcCycles)/1e6, "avg-pause-ms")
			}
		})
	}
}

// BenchmarkGCRealWorldScenarios benchmarks realistic application scenarios
func BenchmarkGCRealWorldScenarios(b *testing.B) {
	tests := []struct {
		name     string
		scenario func(int)
	}{
		{
			name: "WebServerRequests",
			scenario: func(n int) {
				for i := 0; i < n; i++ {
					// Simulate HTTP request processing
					request := make([]byte, 2048)  // Request body
					headers := make(map[string]string)
					for j := 0; j < 10; j++ {
						headers[fmt.Sprintf("header-%d", j)] = fmt.Sprintf("value-%d", j)
					}
					response := make([]byte, 4096) // Response body
					_ = request
					_ = headers
					_ = response
				}
			},
		},
		{
			name: "JSONProcessing",
			scenario: func(n int) {
				for i := 0; i < n; i++ {
					// Simulate JSON parsing and processing
					data := make(map[string]interface{})
					for j := 0; j < 20; j++ {
						key := fmt.Sprintf("field_%d", j)
						if j%3 == 0 {
							data[key] = fmt.Sprintf("string_value_%d", j)
						} else if j%3 == 1 {
							data[key] = j
						} else {
							data[key] = []int{j, j + 1, j + 2}
						}
					}
					_ = data
				}
			},
		},
		{
			name: "DatabaseOperations",
			scenario: func(n int) {
				for i := 0; i < n; i++ {
					// Simulate database query results
					rows := make([]map[string]interface{}, 100)
					for j := range rows {
						rows[j] = map[string]interface{}{
							"id":   j,
							"name": fmt.Sprintf("record_%d", j),
							"data": make([]byte, 512),
						}
					}
					_ = rows
				}
			},
		},
		{
			name: "CacheOperations",
			scenario: func(n int) {
				cache := make(map[string][]byte)
				for i := 0; i < n; i++ {
					// Simulate cache operations
					key := fmt.Sprintf("cache_key_%d", i%1000)
					value := make([]byte, 1024)
					cache[key] = value
					
					// Occasionally clear old entries
					if i%500 == 0 {
						for k := range cache {
							if len(cache) > 500 {
								delete(cache, k)
								break
							}
						}
					}
				}
				_ = cache
			},
		},
	}
	
	for _, tt := range tests {
		b.Run(tt.name, func(b *testing.B) {
			// Clear GC stats
			runtime.GC()
			var startStats, endStats runtime.MemStats
			runtime.ReadMemStats(&startStats)
			
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				tt.scenario(100)
			}
			b.StopTimer()
			
			runtime.ReadMemStats(&endStats)
			
			// Report comprehensive metrics
			gcCycles := endStats.NumGC - startStats.NumGC
			totalPause := endStats.PauseTotalNs - startStats.PauseTotalNs
			allocated := endStats.TotalAlloc - startStats.TotalAlloc
			heapInUse := endStats.HeapInuse
			heapSys := endStats.HeapSys
			nextGC := endStats.NextGC
			
			b.ReportMetric(float64(gcCycles), "gc-cycles")
			b.ReportMetric(float64(totalPause)/1e6, "gc-pause-ms")
			b.ReportMetric(float64(allocated)/(1024*1024), "allocated-mb")
			b.ReportMetric(float64(heapInUse)/(1024*1024), "heap-inuse-mb")
			b.ReportMetric(float64(heapSys)/(1024*1024), "heap-sys-mb")
			b.ReportMetric(float64(nextGC)/(1024*1024), "next-gc-mb")
			if gcCycles > 0 {
				b.ReportMetric(float64(totalPause)/float64(gcCycles)/1e6, "avg-pause-ms")
			}
		})
	}
}

// TestGCAnalyzer tests the GC analyzer functionality
func TestGCAnalyzer(t *testing.T) {
	analyzer := NewGCAnalyzer()
	
	// Test basic functionality
	analyzer.StartMonitoring()
	defer analyzer.StopMonitoring()
	
	// Generate some GC activity
	for i := 0; i < 1000; i++ {
		_ = make([]byte, 1024)
	}
	
	// Force GC
	runtime.GC()
	time.Sleep(100 * time.Millisecond)
	
	// Get statistics
	stats := analyzer.GetStatistics()
	if stats.TotalCycles == 0 {
		t.Error("Expected GC cycles to be recorded")
	}
	
	// Test report generation
	report := analyzer.GenerateReport()
	if len(report) == 0 {
		t.Error("Expected non-empty report")
	}
	
	if !strings.Contains(report, "GC Statistics Report") {
		t.Error("Expected report to contain title")
	}
}

// TestGCTraceParser tests the GC trace parser
func TestGCTraceParser(t *testing.T) {
	parser := NewGCTraceParser()
	
	// Test parsing sample GC trace line
	sampleTrace := "gc 1 @0.123s 0%: 0.018+1.2+0.034 ms clock, 0.072+0.12/1.0/0.96+0.13 ms cpu, 4->4->2 MB, 5 MB goal, 4 P"
	
	event, err := parser.ParseTraceLine(sampleTrace)
	if err != nil {
		t.Fatalf("Failed to parse trace line: %v", err)
	}
	
	if event.CycleNumber != 1 {
		t.Errorf("Expected cycle number 1, got %d", event.CycleNumber)
	}
	
	if event.HeapSizeBefore != 4 {
		t.Errorf("Expected heap size before 4MB, got %d", event.HeapSizeBefore)
	}
	
	if event.HeapSizeAfter != 2 {
		t.Errorf("Expected heap size after 2MB, got %d", event.HeapSizeAfter)
	}
}

// TestAllocationTracker tests the allocation tracker
func TestAllocationTracker(t *testing.T) {
	tracker := NewAllocationTracker()
	
	// Test basic tracking
	tracker.TrackAllocation(1024, "test_location_1")
	tracker.TrackAllocation(2048, "test_location_2")
	tracker.TrackAllocation(512, "test_location_1")
	
	summary := tracker.GetSummary()
	if summary.TotalAllocations != 3 {
		t.Errorf("Expected 3 allocations, got %d", summary.TotalAllocations)
	}
	
	if summary.TotalBytes != 3584 {
		t.Errorf("Expected 3584 bytes, got %d", summary.TotalBytes)
	}
	
	if summary.UniqueSites != 2 {
		t.Errorf("Expected 2 unique sites, got %d", summary.UniqueSites)
	}
	
	// Test top allocators
	topAllocators := tracker.GetTopAllocators(5)
	if len(topAllocators.ByCount) == 0 {
		t.Error("Expected top allocators by count")
	}
	
	if len(topAllocators.ByBytes) == 0 {
		t.Error("Expected top allocators by bytes")
	}
}

// TestAllocationPatternSuite tests the allocation pattern suite
func TestAllocationPatternSuite(t *testing.T) {
	suite := NewAllocationPatternSuite()
	
	// Test pattern execution
	err := suite.RunPattern("SmallObjects", 100)
	if err != nil {
		t.Fatalf("Failed to run SmallObjects pattern: %v", err)
	}
	
	// Test invalid pattern
	err = suite.RunPattern("NonExistentPattern", 100)
	if err == nil {
		t.Error("Expected error for non-existent pattern")
	}
	
	// Test pattern names
	names := suite.GetPatternNames()
	if len(names) == 0 {
		t.Error("Expected pattern names")
	}
	
	// Test report generation
	report := suite.GenerateAllocationReport()
	if len(report) == 0 {
		t.Error("Expected non-empty report")
	}
}

// TestConcurrentGCAnalysis tests GC analysis under concurrent load
func TestConcurrentGCAnalysis(t *testing.T) {
	analyzer := NewGCAnalyzer()
	analyzer.StartMonitoring()
	defer analyzer.StopMonitoring()
	
	var wg sync.WaitGroup
	workers := 4
	
	for i := 0; i < workers; i++ {
		wg.Add(1)
		go func(workerID int) {
			defer wg.Done()
			for j := 0; j < 1000; j++ {
				_ = make([]byte, 1024)
				if j%100 == 0 {
					runtime.Gosched()
				}
			}
		}(i)
	}
	
	wg.Wait()
	
	// Force GC and wait
	runtime.GC()
	time.Sleep(200 * time.Millisecond)
	
	stats := analyzer.GetStatistics()
	if stats.TotalCycles == 0 {
		t.Error("Expected GC cycles under concurrent load")
	}
}

// TestMemoryLeakDetection tests memory leak detection capabilities
func TestMemoryLeakDetection(t *testing.T) {
	var startStats, endStats runtime.MemStats
	runtime.GC()
	runtime.ReadMemStats(&startStats)
	
	// Simulate potential memory leak
	var leaked [][]byte
	for i := 0; i < 1000; i++ {
		data := make([]byte, 1024)
		leaked = append(leaked, data)
	}
	
	runtime.GC()
	runtime.ReadMemStats(&endStats)
	
	heapGrowth := endStats.HeapInuse - startStats.HeapInuse
	if heapGrowth < 1024*1024 { // Should be at least 1MB
		t.Errorf("Expected significant heap growth, got %d bytes", heapGrowth)
	}
	
	// Keep leaked data alive
	_ = leaked
	
	// Test that memory is actually retained
	runtime.GC()
	var finalStats runtime.MemStats
	runtime.ReadMemStats(&finalStats)
	
	if finalStats.HeapInuse < endStats.HeapInuse {
		t.Error("Expected memory to remain allocated (simulated leak)")
	}
}