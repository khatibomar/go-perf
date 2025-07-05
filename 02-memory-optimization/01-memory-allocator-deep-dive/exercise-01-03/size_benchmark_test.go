package main

import (
	"fmt"
	"runtime"
	"sync"
	"testing"
)

// BenchmarkSizeClassEfficiency benchmarks different size class scenarios
func BenchmarkSizeClassEfficiency(b *testing.B) {
	sizes := []int{24, 48, 96, 128, 256, 384, 512, 1024}
	
	for _, size := range sizes {
		b.Run(fmt.Sprintf("Size_%d", size), func(b *testing.B) {
			b.ResetTimer()
			b.ReportAllocs()
			
			for i := 0; i < b.N; i++ {
				_ = make([]byte, size)
			}
		})
	}
}

// BenchmarkStructAlignment benchmarks struct alignment impact
func BenchmarkStructAlignment(b *testing.B) {
	type BadStruct struct {
		A bool   // 1 byte
		B int64  // 8 bytes
		C bool   // 1 byte
		D int32  // 4 bytes
		E bool   // 1 byte
	}
	
	type GoodStruct struct {
		B int64  // 8 bytes
		D int32  // 4 bytes
		A bool   // 1 byte
		C bool   // 1 byte
		E bool   // 1 byte
	}
	
	b.Run("BadAlignment", func(b *testing.B) {
		b.ResetTimer()
		b.ReportAllocs()
		
		for i := 0; i < b.N; i++ {
			_ = BadStruct{
				A: true,
				B: int64(i),
				C: false,
				D: int32(i),
				E: true,
			}
		}
	})
	
	b.Run("GoodAlignment", func(b *testing.B) {
		b.ResetTimer()
		b.ReportAllocs()
		
		for i := 0; i < b.N; i++ {
			_ = GoodStruct{
				B: int64(i),
				D: int32(i),
				A: true,
				C: false,
				E: true,
			}
		}
	})
	
	b.Run("ArrayBadAlignment", func(b *testing.B) {
		b.ResetTimer()
		b.ReportAllocs()
		
		for i := 0; i < b.N; i++ {
			arr := make([]BadStruct, 100)
			for j := range arr {
				arr[j] = BadStruct{
					A: true,
					B: int64(j),
					C: false,
					D: int32(j),
					E: true,
				}
			}
			_ = arr
		}
	})
	
	b.Run("ArrayGoodAlignment", func(b *testing.B) {
		b.ResetTimer()
		b.ReportAllocs()
		
		for i := 0; i < b.N; i++ {
			arr := make([]GoodStruct, 100)
			for j := range arr {
				arr[j] = GoodStruct{
					B: int64(j),
					D: int32(j),
					A: true,
					C: false,
					E: true,
				}
			}
			_ = arr
		}
	})
}

// BenchmarkAllocationPatterns benchmarks different allocation patterns
func BenchmarkAllocationPatterns(b *testing.B) {
	b.Run("SmallFrequent", func(b *testing.B) {
		b.ResetTimer()
		b.ReportAllocs()
		
		for i := 0; i < b.N; i++ {
			// Simulate frequent small allocations
			for j := 0; j < 10; j++ {
				_ = make([]byte, 24)
				_ = make([]byte, 48)
				_ = make([]byte, 96)
			}
		}
	})
	
	b.Run("MediumMixed", func(b *testing.B) {
		b.ResetTimer()
		b.ReportAllocs()
		
		for i := 0; i < b.N; i++ {
			// Simulate mixed medium allocations
			_ = make([]byte, 256)
			_ = make([]byte, 384)
			_ = make([]byte, 512)
			_ = make([]byte, 768)
			_ = make([]byte, 1024)
		}
	})
	
	b.Run("LargeInfrequent", func(b *testing.B) {
		b.ResetTimer()
		b.ReportAllocs()
		
		for i := 0; i < b.N; i++ {
			// Simulate infrequent large allocations
			_ = make([]byte, 4096)
			_ = make([]byte, 8192)
			_ = make([]byte, 16384)
		}
	})
	
	b.Run("PowerOfTwo", func(b *testing.B) {
		b.ResetTimer()
		b.ReportAllocs()
		
		for i := 0; i < b.N; i++ {
			// Power of 2 sizes (optimal for many allocators)
			_ = make([]byte, 64)
			_ = make([]byte, 128)
			_ = make([]byte, 256)
			_ = make([]byte, 512)
			_ = make([]byte, 1024)
		}
	})
	
	b.Run("NonPowerOfTwo", func(b *testing.B) {
		b.ResetTimer()
		b.ReportAllocs()
		
		for i := 0; i < b.N; i++ {
			// Non-power of 2 sizes (may cause more waste)
			_ = make([]byte, 96)
			_ = make([]byte, 192)
			_ = make([]byte, 384)
			_ = make([]byte, 768)
			_ = make([]byte, 1536)
		}
	})
}

// BenchmarkSizeClassOptimizer benchmarks the optimizer itself
func BenchmarkSizeClassOptimizer(b *testing.B) {
	optimizer := NewSizeClassOptimizer()
	
	b.Run("RecordAllocation", func(b *testing.B) {
		b.ResetTimer()
		b.ReportAllocs()
		
		for i := 0; i < b.N; i++ {
			optimizer.RecordAllocation(256)
		}
	})
	
	b.Run("AnalyzeWastage", func(b *testing.B) {
		// Pre-populate with data
		for i := 0; i < 10000; i++ {
			optimizer.RecordAllocation(24 + (i%10)*32)
		}
		
		b.ResetTimer()
		b.ReportAllocs()
		
		for i := 0; i < b.N; i++ {
			_ = optimizer.AnalyzeWastage()
		}
	})
	
	b.Run("ConcurrentRecording", func(b *testing.B) {
		b.ResetTimer()
		b.ReportAllocs()
		
		b.RunParallel(func(pb *testing.PB) {
			size := 256
			for pb.Next() {
				optimizer.RecordAllocation(size)
				size = (size % 1024) + 64
			}
		})
	})
}

// BenchmarkAllocationTracker benchmarks the allocation tracker
func BenchmarkAllocationTracker(b *testing.B) {
	tracker := NewAllocationTracker()
	tracker.Start()
	
	b.Run("Track", func(b *testing.B) {
		b.ResetTimer()
		b.ReportAllocs()
		
		for i := 0; i < b.N; i++ {
			tracker.Track(256)
		}
	})
	
	b.Run("ConcurrentTrack", func(b *testing.B) {
		b.ResetTimer()
		b.ReportAllocs()
		
		b.RunParallel(func(pb *testing.PB) {
			size := 128
			for pb.Next() {
				tracker.Track(size)
				size = (size % 2048) + 64
			}
		})
	})
	
	b.Run("GetPatterns", func(b *testing.B) {
		// Pre-populate with data
		for i := 0; i < 10000; i++ {
			tracker.Track(64 + (i%32)*16)
		}
		
		b.ResetTimer()
		b.ReportAllocs()
		
		for i := 0; i < b.N; i++ {
			_ = tracker.GetPatterns()
		}
	})
}

// BenchmarkMemoryAlignmentAnalyzer benchmarks struct analysis
func BenchmarkMemoryAlignmentAnalyzer(b *testing.B) {
	analyzer := &MemoryAlignmentAnalyzer{}
	
	type TestStruct struct {
		A bool
		B int64
		C bool
		D int32
		E bool
		F string
		G []byte
		H map[string]int
	}
	
	b.Run("AnalyzeStruct", func(b *testing.B) {
		b.ResetTimer()
		b.ReportAllocs()
		
		for i := 0; i < b.N; i++ {
			_ = analyzer.AnalyzeStruct(TestStruct{})
		}
	})
	
	b.Run("SuggestOptimization", func(b *testing.B) {
		analysis := analyzer.AnalyzeStruct(TestStruct{})
		
		b.ResetTimer()
		b.ReportAllocs()
		
		for i := 0; i < b.N; i++ {
			_ = analyzer.SuggestOptimization(analysis)
		}
	})
}

// BenchmarkOptimizedSizeClassGenerator benchmarks size class generation
func BenchmarkOptimizedSizeClassGenerator(b *testing.B) {
	generator := NewOptimizedSizeClassGenerator(0.25, 0.75)
	
	// Add patterns
	patterns := map[int]int64{
		24: 2000, 48: 1500, 96: 1000, 128: 1800, 200: 800,
		256: 600, 384: 400, 512: 300, 1024: 200,
	}
	
	for size, freq := range patterns {
		generator.AddAllocationPattern(size, freq)
	}
	
	b.Run("GenerateOptimizedSizeClasses", func(b *testing.B) {
		b.ResetTimer()
		b.ReportAllocs()
		
		for i := 0; i < b.N; i++ {
			_ = generator.GenerateOptimizedSizeClasses()
		}
	})
	
	b.Run("AddAllocationPattern", func(b *testing.B) {
		b.ResetTimer()
		b.ReportAllocs()
		
		for i := 0; i < b.N; i++ {
			generator.AddAllocationPattern(128+(i%512), int64(100+i%1000))
		}
	})
}

// BenchmarkRealWorldScenarios benchmarks realistic allocation scenarios
func BenchmarkRealWorldScenarios(b *testing.B) {
	b.Run("WebServerSimulation", func(b *testing.B) {
		b.ResetTimer()
		b.ReportAllocs()
		
		for i := 0; i < b.N; i++ {
			// Simulate HTTP request processing
			headers := make([]byte, 512)   // Request headers
			body := make([]byte, 1024)     // Request body
			response := make([]byte, 4096) // Response
			cookies := make([]byte, 256)   // Cookies
			
			_ = headers
			_ = body
			_ = response
			_ = cookies
		}
	})
	
	b.Run("JSONProcessingSimulation", func(b *testing.B) {
		b.ResetTimer()
		b.ReportAllocs()
		
		for i := 0; i < b.N; i++ {
			// Simulate JSON processing
			parseBuffer := make([]byte, 2048) // JSON parse buffer
			stringIntern := make([]byte, 1024) // String interning
			numberBuf := make([]byte, 512)     // Number parsing
			
			_ = parseBuffer
			_ = stringIntern
			_ = numberBuf
		}
	})
	
	b.Run("DatabaseSimulation", func(b *testing.B) {
		b.ResetTimer()
		b.ReportAllocs()
		
		for i := 0; i < b.N; i++ {
			// Simulate database operations
			rowBuffer := make([]byte, 256)   // Small rows
			mediumRow := make([]byte, 1024)  // Medium rows
			largeRow := make([]byte, 4096)   // Large rows with BLOBs
			resultSet := make([]byte, 8192)  // Result set buffer
			
			_ = rowBuffer
			_ = mediumRow
			_ = largeRow
			_ = resultSet
		}
	})
	
	b.Run("MixedWorkloadSimulation", func(b *testing.B) {
		b.ResetTimer()
		b.ReportAllocs()
		
		b.RunParallel(func(pb *testing.PB) {
			i := 0
			for pb.Next() {
				// Mixed allocation pattern
				switch i % 4 {
				case 0:
					_ = make([]byte, 64)   // Small
				case 1:
					_ = make([]byte, 256)  // Medium
				case 2:
					_ = make([]byte, 1024) // Large
				case 3:
					_ = make([]byte, 4096) // Very large
				}
				i++
			}
		})
	})
}

// BenchmarkMemoryWaste benchmarks memory waste scenarios
func BenchmarkMemoryWaste(b *testing.B) {
	b.Run("HighWasteScenario", func(b *testing.B) {
		b.ResetTimer()
		b.ReportAllocs()
		
		for i := 0; i < b.N; i++ {
			// Allocate sizes that cause high waste in Go's size classes
			_ = make([]byte, 25)  // Wastes ~39 bytes (64-25)
			_ = make([]byte, 65)  // Wastes ~31 bytes (96-65)
			_ = make([]byte, 129) // Wastes ~31 bytes (160-129)
			_ = make([]byte, 257) // Wastes ~31 bytes (288-257)
		}
	})
	
	b.Run("LowWasteScenario", func(b *testing.B) {
		b.ResetTimer()
		b.ReportAllocs()
		
		for i := 0; i < b.N; i++ {
			// Allocate sizes that align well with Go's size classes
			_ = make([]byte, 64)   // No waste
			_ = make([]byte, 96)   // No waste
			_ = make([]byte, 128)  // No waste
			_ = make([]byte, 256)  // No waste
		}
	})
	
	b.Run("OptimizedSizes", func(b *testing.B) {
		b.ResetTimer()
		b.ReportAllocs()
		
		for i := 0; i < b.N; i++ {
			// Use sizes optimized for Go's allocator
			_ = make([]byte, 32)   // Perfect fit
			_ = make([]byte, 48)   // Perfect fit
			_ = make([]byte, 64)   // Perfect fit
			_ = make([]byte, 80)   // Perfect fit
			_ = make([]byte, 96)   // Perfect fit
			_ = make([]byte, 112)  // Perfect fit
			_ = make([]byte, 128)  // Perfect fit
		}
	})
}

// BenchmarkCacheEffects benchmarks cache-related effects
func BenchmarkCacheEffects(b *testing.B) {
	type SmallStruct struct {
		A, B, C, D int64 // 32 bytes total
	}
	
	type LargeStruct struct {
		Data [128]int64 // 1024 bytes total
	}
	
	b.Run("SmallStructArray", func(b *testing.B) {
		arr := make([]SmallStruct, 1000)
		b.ResetTimer()
		b.ReportAllocs()
		
		for i := 0; i < b.N; i++ {
			for j := range arr {
				arr[j].A = int64(j)
			}
		}
	})
	
	b.Run("LargeStructArray", func(b *testing.B) {
		arr := make([]LargeStruct, 100)
		b.ResetTimer()
		b.ReportAllocs()
		
		for i := 0; i < b.N; i++ {
			for j := range arr {
				arr[j].Data[0] = int64(j)
			}
		}
	})
	
	b.Run("PointerChasing", func(b *testing.B) {
		type Node struct {
			Value int64
			Next  *Node
		}
		
		// Create linked list
		var head *Node
		for i := 0; i < 1000; i++ {
			node := &Node{Value: int64(i), Next: head}
			head = node
		}
		
		b.ResetTimer()
		b.ReportAllocs()
		
		for i := 0; i < b.N; i++ {
			sum := int64(0)
			current := head
			for current != nil {
				sum += current.Value
				current = current.Next
			}
			_ = sum
		}
	})
}

// TestSizeClassOptimizer tests the optimizer functionality
func TestSizeClassOptimizer(t *testing.T) {
	optimizer := NewSizeClassOptimizer()
	
	// Record some allocations
	optimizer.RecordAllocation(24)
	optimizer.RecordAllocation(48)
	optimizer.RecordAllocation(96)
	optimizer.RecordAllocation(24) // Duplicate
	
	report := optimizer.AnalyzeWastage()
	
	if report.TotalAllocations != 4 {
		t.Errorf("Expected 4 allocations, got %d", report.TotalAllocations)
	}
	
	if len(report.SizeClassStats) == 0 {
		t.Error("Expected size class statistics")
	}
	
	// Check that 24-byte allocations are tracked correctly
	if stat, exists := report.SizeClassStats[32]; exists { // 24 -> 32 size class
		if stat.Allocations != 2 {
			t.Errorf("Expected 2 allocations for size class 32, got %d", stat.Allocations)
		}
	}
}

// TestAllocationTracker tests the allocation tracker
func TestAllocationTracker(t *testing.T) {
	tracker := NewAllocationTracker()
	
	// Should not track when inactive
	tracker.Track(100)
	patterns := tracker.GetPatterns()
	if len(patterns) != 0 {
		t.Error("Should not track when inactive")
	}
	
	// Should track when active
	tracker.Start()
	tracker.Track(100)
	tracker.Track(200)
	tracker.Track(100) // Duplicate
	
	patterns = tracker.GetPatterns()
	if patterns[100] != 2 {
		t.Errorf("Expected 2 allocations of size 100, got %d", patterns[100])
	}
	if patterns[200] != 1 {
		t.Errorf("Expected 1 allocation of size 200, got %d", patterns[200])
	}
	
	// Should stop tracking when stopped
	tracker.Stop()
	tracker.Track(300)
	patterns = tracker.GetPatterns()
	if _, exists := patterns[300]; exists {
		t.Error("Should not track after stop")
	}
	
	// Test reset
	tracker.Reset()
	patterns = tracker.GetPatterns()
	if len(patterns) != 0 {
		t.Error("Patterns should be empty after reset")
	}
}

// TestMemoryAlignmentAnalyzer tests struct analysis
func TestMemoryAlignmentAnalyzer(t *testing.T) {
	analyzer := &MemoryAlignmentAnalyzer{}
	
	type TestStruct struct {
		A bool
		B int64
		C bool
	}
	
	analysis := analyzer.AnalyzeStruct(TestStruct{})
	
	if analysis.Name != "TestStruct" {
		t.Errorf("Expected name 'TestStruct', got '%s'", analysis.Name)
	}
	
	if len(analysis.Fields) != 3 {
		t.Errorf("Expected 3 fields, got %d", len(analysis.Fields))
	}
	
	if analysis.Size <= 0 {
		t.Errorf("Expected positive size, got %d", analysis.Size)
	}
	
	if analysis.Efficiency <= 0 || analysis.Efficiency > 1 {
		t.Errorf("Expected efficiency between 0 and 1, got %f", analysis.Efficiency)
	}
	
	// Test optimization suggestion
	suggestion := analyzer.SuggestOptimization(analysis)
	if suggestion.OriginalSize != analysis.Size {
		t.Errorf("Original size mismatch: %d vs %d", suggestion.OriginalSize, analysis.Size)
	}
}

// TestOptimizedSizeClassGenerator tests size class generation
func TestOptimizedSizeClassGenerator(t *testing.T) {
	generator := NewOptimizedSizeClassGenerator(0.25, 0.75)
	
	// Add some patterns
	generator.AddAllocationPattern(24, 1000)
	generator.AddAllocationPattern(48, 800)
	generator.AddAllocationPattern(96, 600)
	
	classes := generator.GenerateOptimizedSizeClasses()
	
	if len(classes) == 0 {
		t.Error("Expected at least one size class")
	}
	
	// Check that classes are sorted
	for i := 1; i < len(classes); i++ {
		if classes[i].Size <= classes[i-1].Size {
			t.Error("Size classes should be sorted by size")
		}
	}
	
	// Check that waste is within limits
	for _, class := range classes {
		if class.Waste > 0.25 {
			t.Errorf("Size class %d has waste %.2f%% > 25%%", class.Size, class.Waste*100)
		}
	}
}

// TestConcurrentAccess tests concurrent access to components
func TestConcurrentAccess(t *testing.T) {
	optimizer := NewSizeClassOptimizer()
	tracker := NewAllocationTracker()
	tracker.Start()
	
	const numGoroutines = 10
	const operationsPerGoroutine = 1000
	
	var wg sync.WaitGroup
	wg.Add(numGoroutines * 2) // For both optimizer and tracker
	
	// Test concurrent optimizer access
	for i := 0; i < numGoroutines; i++ {
		go func(id int) {
			defer wg.Done()
			for j := 0; j < operationsPerGoroutine; j++ {
				size := 64 + (id*j)%512
				optimizer.RecordAllocation(size)
			}
		}(i)
	}
	
	// Test concurrent tracker access
	for i := 0; i < numGoroutines; i++ {
		go func(id int) {
			defer wg.Done()
			for j := 0; j < operationsPerGoroutine; j++ {
				size := 32 + (id*j)%256
				tracker.Track(size)
			}
		}(i)
	}
	
	wg.Wait()
	
	// Verify data integrity
	report := optimizer.AnalyzeWastage()
	if report.TotalAllocations != int64(numGoroutines*operationsPerGoroutine) {
		t.Errorf("Expected %d allocations, got %d", 
			numGoroutines*operationsPerGoroutine, report.TotalAllocations)
	}
	
	patterns := tracker.GetPatterns()
	totalTracked := int64(0)
	for _, count := range patterns {
		totalTracked += count
	}
	if totalTracked != int64(numGoroutines*operationsPerGoroutine) {
		t.Errorf("Expected %d tracked allocations, got %d", 
			numGoroutines*operationsPerGoroutine, totalTracked)
	}
}

// TestMemoryLeaks tests for memory leaks
func TestMemoryLeaks(t *testing.T) {
	var m1, m2 runtime.MemStats
	
	// Measure initial memory
	runtime.GC()
	runtime.ReadMemStats(&m1)
	
	// Create and use components extensively
	optimizer := NewSizeClassOptimizer()
	tracker := NewAllocationTracker()
	analyzer := &MemoryAlignmentAnalyzer{}
	generator := NewOptimizedSizeClassGenerator(0.25, 0.75)
	
	tracker.Start()
	
	// Perform many operations
	for i := 0; i < 100000; i++ {
		size := 64 + (i%32)*16
		optimizer.RecordAllocation(size)
		tracker.Track(size)
		generator.AddAllocationPattern(size, int64(i%1000))
		
		if i%1000 == 0 {
			_ = optimizer.AnalyzeWastage()
			_ = tracker.GetPatterns()
			_ = generator.GenerateOptimizedSizeClasses()
		}
	}
	
	// Analyze some structs
	type TestStruct struct {
		A bool
		B int64
		C string
	}
	
	for i := 0; i < 1000; i++ {
		analysis := analyzer.AnalyzeStruct(TestStruct{})
		_ = analyzer.SuggestOptimization(analysis)
	}
	
	tracker.Stop()
	
	// Force GC and measure again
	runtime.GC()
	runtime.GC() // Double GC to ensure cleanup
	runtime.ReadMemStats(&m2)
	
	// Check for excessive memory growth
	memoryGrowth := int64(m2.HeapAlloc) - int64(m1.HeapAlloc)
	if memoryGrowth > 10*1024*1024 { // 10MB threshold
		t.Errorf("Excessive memory growth: %d bytes", memoryGrowth)
	}
	
	t.Logf("Memory growth: %d bytes", memoryGrowth)
	t.Logf("GC cycles: %d", m2.NumGC-m1.NumGC)
}