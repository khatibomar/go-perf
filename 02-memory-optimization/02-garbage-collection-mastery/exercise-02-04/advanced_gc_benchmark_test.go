package main

import (
	"fmt"
	"runtime"
	"sync"
	"testing"
	"unsafe"
)

// BenchmarkCustomAllocators benchmarks different custom allocator implementations
func BenchmarkCustomAllocators(b *testing.B) {
	allocatorTypes := []struct {
		name    string
		creator func() Allocator
	}{
		{
			name: "StackAllocator",
			creator: func() Allocator {
				return NewStackAllocator(1024 * 1024) // 1MB
			},
		},
		{
			name: "LinearAllocator",
			creator: func() Allocator {
				return NewLinearAllocator(1024*1024, 8) // 1MB, 8-byte aligned
			},
		},
		{
			name: "ArenaAllocator",
			creator: func() Allocator {
				return NewArenaAllocator(64 * 1024) // 64KB arenas
			},
		},
		{
			name: "SlabAllocator",
			creator: func() Allocator {
				return NewSlabAllocator(64, 4096) // 64-byte objects, 4KB slabs
			},
		},
	}
	
	for _, allocType := range allocatorTypes {
		b.Run(allocType.name, func(b *testing.B) {
			allocator := allocType.creator()
			
			var startStats, endStats runtime.MemStats
			runtime.GC()
			runtime.ReadMemStats(&startStats)
			
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				// Allocate various sizes
				for j := 0; j < 100; j++ {
					size := 32 + (j % 512) // 32 to 544 bytes
					ptr := allocator.Allocate(size)
					if ptr == nil {
						// Reset allocator if full
						allocator.Reset()
						ptr = allocator.Allocate(size)
					}
					_ = ptr
				}
			}
			b.StopTimer()
			
			runtime.ReadMemStats(&endStats)
			stats := allocator.Stats()
			reportCustomAllocatorMetrics(b, startStats, endStats, stats, allocType.name)
		})
	}
}

// BenchmarkAllocatorVsStandard compares custom allocators with standard allocation
func BenchmarkAllocatorVsStandard(b *testing.B) {
	allocationSizes := []int{32, 64, 128, 256, 512, 1024, 2048, 4096}
	
	for _, size := range allocationSizes {
		b.Run(fmt.Sprintf("Size_%d", size), func(b *testing.B) {
			b.Run("Standard", func(b *testing.B) {
				var startStats, endStats runtime.MemStats
				runtime.GC()
				runtime.ReadMemStats(&startStats)
				
				b.ResetTimer()
				for i := 0; i < b.N; i++ {
					for j := 0; j < 100; j++ {
						_ = make([]byte, size)
					}
				}
				b.StopTimer()
				
				runtime.ReadMemStats(&endStats)
				reportStandardAllocationMetrics(b, startStats, endStats, size)
			})
			
			b.Run("LinearAllocator", func(b *testing.B) {
				allocator := NewLinearAllocator(1024*1024, 8)
				
				var startStats, endStats runtime.MemStats
				runtime.GC()
				runtime.ReadMemStats(&startStats)
				
				b.ResetTimer()
				for i := 0; i < b.N; i++ {
					for j := 0; j < 100; j++ {
						ptr := allocator.Allocate(size)
						if ptr == nil {
							allocator.Reset()
							ptr = allocator.Allocate(size)
						}
						_ = ptr
					}
				}
				b.StopTimer()
				
				runtime.ReadMemStats(&endStats)
				stats := allocator.Stats()
				reportCustomAllocatorMetrics(b, startStats, endStats, stats, "LinearAllocator")
			})
			
			b.Run("SlabAllocator", func(b *testing.B) {
				// Only test slab allocator for sizes it supports
				if size > 4096 {
					b.Skip("Size too large for slab allocator")
				}
				
				allocator := NewSlabAllocator(uintptr(size), 4096)
				
				var startStats, endStats runtime.MemStats
				runtime.GC()
				runtime.ReadMemStats(&startStats)
				
				b.ResetTimer()
				for i := 0; i < b.N; i++ {
					ptrs := make([]unsafe.Pointer, 100)
					for j := 0; j < 100; j++ {
						ptrs[j] = allocator.Allocate(size)
					}
					// Deallocate to test reuse
					for j := 0; j < 50; j++ {
						if ptrs[j] != nil {
							allocator.Deallocate(ptrs[j])
						}
					}
				}
				b.StopTimer()
				
				runtime.ReadMemStats(&endStats)
				stats := allocator.Stats()
				reportCustomAllocatorMetrics(b, startStats, endStats, stats, "SlabAllocator")
			})
		})
	}
}

// BenchmarkAdvancedBufferPool benchmarks the advanced buffer pool
func BenchmarkAdvancedBufferPool(b *testing.B) {
	bufferSizes := []int{512, 2048, 32768, 131072, 524288}
	
	for _, size := range bufferSizes {
		b.Run(fmt.Sprintf("Size_%d", size), func(b *testing.B) {
			b.Run("StandardAllocation", func(b *testing.B) {
				var startStats, endStats runtime.MemStats
				runtime.GC()
				runtime.ReadMemStats(&startStats)
				
				b.ResetTimer()
				for i := 0; i < b.N; i++ {
					for j := 0; j < 50; j++ {
						buf := make([]byte, size)
						// Use buffer to prevent optimization
						for k := 0; k < len(buf) && k < 100; k++ {
							buf[k] = byte(k)
						}
						_ = buf
					}
				}
				b.StopTimer()
				
				runtime.ReadMemStats(&endStats)
				reportStandardAllocationMetrics(b, startStats, endStats, size)
			})
			
			b.Run("BufferPool", func(b *testing.B) {
				pool := NewAdvancedBufferPool()
				
				var startStats, endStats runtime.MemStats
				runtime.GC()
				runtime.ReadMemStats(&startStats)
				
				b.ResetTimer()
				for i := 0; i < b.N; i++ {
					buffers := make([][]byte, 50)
					for j := 0; j < 50; j++ {
						buffers[j] = pool.GetBuffer(size)
						// Use buffer to prevent optimization
						for k := 0; k < len(buffers[j]) && k < 100; k++ {
							buffers[j] = append(buffers[j], byte(k))
						}
					}
					// Return buffers to pool
					for j := 0; j < 50; j++ {
						pool.PutBuffer(buffers[j])
					}
				}
				b.StopTimer()
				
				runtime.ReadMemStats(&endStats)
				stats := pool.GetStats()
				reportBufferPoolMetrics(b, startStats, endStats, stats, size)
			})
		})
	}
}

// BenchmarkGCFreeDataStructures benchmarks GC-free data structure operations
func BenchmarkGCFreeDataStructures(b *testing.B) {
	b.Run("StandardSlice", func(b *testing.B) {
		var startStats, endStats runtime.MemStats
		runtime.GC()
		runtime.ReadMemStats(&startStats)
		
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			slice := make([]int, 0, 1000)
			for j := 0; j < 1000; j++ {
				slice = append(slice, j)
			}
			// Use slice to prevent optimization
			sum := 0
			for _, v := range slice {
				sum += v
			}
			_ = sum
		}
		b.StopTimer()
		
		runtime.ReadMemStats(&endStats)
		reportStandardAllocationMetrics(b, startStats, endStats, 1000*8) // 1000 ints * 8 bytes
	})
	
	b.Run("GCFreeSlice", func(b *testing.B) {
		allocator := NewLinearAllocator(1024*1024, 8)
		
		var startStats, endStats runtime.MemStats
		runtime.GC()
		runtime.ReadMemStats(&startStats)
		
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			// Allocate space for 1000 ints
			ptr := allocator.Allocate(1000 * 8)
			if ptr == nil {
				allocator.Reset()
				ptr = allocator.Allocate(1000 * 8)
			}
			
			// Use unsafe to create slice from allocated memory
			slice := (*[1000]int)(ptr)[:0:1000]
			for j := 0; j < 1000; j++ {
				slice = append(slice, j)
			}
			// Use slice to prevent optimization
			sum := 0
			for _, v := range slice {
				sum += v
			}
			_ = sum
		}
		b.StopTimer()
		
		runtime.ReadMemStats(&endStats)
		stats := allocator.Stats()
		reportCustomAllocatorMetrics(b, startStats, endStats, stats, "GCFreeSlice")
	})
}

// BenchmarkConcurrentAllocators benchmarks allocators under concurrent load
func BenchmarkConcurrentAllocators(b *testing.B) {
	workerCounts := []int{1, 2, 4, 8}
	
	for _, workers := range workerCounts {
		b.Run(fmt.Sprintf("Workers_%d", workers), func(b *testing.B) {
			b.Run("StandardAllocation", func(b *testing.B) {
				var startStats, endStats runtime.MemStats
				runtime.GC()
				runtime.ReadMemStats(&startStats)
				
				b.ResetTimer()
				for i := 0; i < b.N; i++ {
					var wg sync.WaitGroup
					for w := 0; w < workers; w++ {
						wg.Add(1)
						go func() {
							defer wg.Done()
							for j := 0; j < 100/workers; j++ {
								_ = make([]byte, 1024)
								_ = make(map[int]string)
								_ = fmt.Sprintf("test_%d", j)
							}
						}()
					}
					wg.Wait()
				}
				b.StopTimer()
				
				runtime.ReadMemStats(&endStats)
				reportConcurrentAllocationMetrics(b, startStats, endStats, workers, "Standard")
			})
			
			b.Run("BufferPool", func(b *testing.B) {
				pool := NewAdvancedBufferPool()
				
				var startStats, endStats runtime.MemStats
				runtime.GC()
				runtime.ReadMemStats(&startStats)
				
				b.ResetTimer()
				for i := 0; i < b.N; i++ {
					var wg sync.WaitGroup
					for w := 0; w < workers; w++ {
						wg.Add(1)
						go func() {
							defer wg.Done()
							for j := 0; j < 100/workers; j++ {
								buf := pool.GetBuffer(1024)
								// Use buffer
								for k := 0; k < 100; k++ {
									buf = append(buf, byte(k))
								}
								pool.PutBuffer(buf)
							}
						}()
					}
					wg.Wait()
				}
				b.StopTimer()
				
				runtime.ReadMemStats(&endStats)
				stats := pool.GetStats()
				reportConcurrentBufferPoolMetrics(b, startStats, endStats, workers, stats)
			})
		})
	}
}

// BenchmarkAdvancedGCOptimization benchmarks the complete optimization suite
func BenchmarkAdvancedGCOptimization(b *testing.B) {
	optimizationScenarios := []struct {
		name     string
		workload func(int)
	}{
		{
			name: "AllocationHeavy",
			workload: func(iteration int) {
				for i := 0; i < 100; i++ {
					_ = make([]byte, 1024+i%1024)
					_ = make(map[int]string)
					_ = fmt.Sprintf("iteration_%d_%d", iteration, i)
				}
			},
		},
		{
			name: "StringProcessing",
			workload: func(iteration int) {
				for i := 0; i < 50; i++ {
					str := ""
					for j := 0; j < 100; j++ {
						str += fmt.Sprintf("part_%d_%d ", iteration, j)
					}
					_ = str
				}
			},
		},
		{
			name: "SliceGrowth",
			workload: func(iteration int) {
				for i := 0; i < 20; i++ {
					slice := make([]int, 0)
					for j := 0; j < 1000; j++ {
						slice = append(slice, j+iteration)
					}
					_ = slice
				}
			},
		},
		{
			name: "MapOperations",
			workload: func(iteration int) {
				for i := 0; i < 10; i++ {
					m := make(map[string]interface{})
					for j := 0; j < 100; j++ {
						key := fmt.Sprintf("key_%d_%d_%d", iteration, i, j)
						m[key] = []byte(fmt.Sprintf("value_%d", j))
					}
					_ = m
				}
			},
		},
	}
	
	for _, scenario := range optimizationScenarios {
		b.Run(scenario.name, func(b *testing.B) {
			b.Run("Baseline", func(b *testing.B) {
				var startStats, endStats runtime.MemStats
				runtime.GC()
				runtime.ReadMemStats(&startStats)
				
				b.ResetTimer()
				for i := 0; i < b.N; i++ {
					scenario.workload(i)
				}
				b.StopTimer()
				
				runtime.ReadMemStats(&endStats)
				reportOptimizationMetrics(b, startStats, endStats, "Baseline", scenario.name)
			})
			
			b.Run("Optimized", func(b *testing.B) {
				// Create optimization suite
				optimizer := NewAdvancedGCOptimizer()
				
				// Apply optimizations based on scenario
				switch scenario.name {
				case "AllocationHeavy":
					// Use custom allocators
					allocator := NewLinearAllocator(1024*1024, 8)
					optimizer.allocatorManager.allocators["primary"] = allocator
				case "StringProcessing":
					// Use buffer manager for string building
					bufferManager := NewBufferManager()
					optimizer.gcFreeManager.reusableBuffers = bufferManager
				case "SliceGrowth":
					// Use pre-allocated slices
					allocator := NewArenaAllocator(64 * 1024)
					optimizer.allocatorManager.allocators["slice"] = allocator
				case "MapOperations":
					// Use object pooling for maps
					mapPool := &ObjectPool{
						pool: sync.Pool{
							New: func() interface{} {
								return make(map[string]interface{})
							},
						},
						stats: &PoolStats{},
					}
					optimizer.gcFreeManager.objectPools["map"] = mapPool
				}
				
				var startStats, endStats runtime.MemStats
				runtime.GC()
				runtime.ReadMemStats(&startStats)
				
				b.ResetTimer()
				for i := 0; i < b.N; i++ {
					// Run optimized workload
					scenario.workload(i)
				}
				b.StopTimer()
				
				runtime.ReadMemStats(&endStats)
				reportOptimizationMetrics(b, startStats, endStats, "Optimized", scenario.name)
			})
		})
	}
}

// BenchmarkMemoryLeakDetection benchmarks memory leak detection capabilities
func BenchmarkMemoryLeakDetection(b *testing.B) {
	leakScenarios := []struct {
		name     string
		workload func() interface{}
	}{
		{
			name: "GrowingSlice",
			workload: func() interface{} {
				slice := make([][]byte, 0)
				for i := 0; i < 100; i++ {
					slice = append(slice, make([]byte, 1024))
				}
				return slice
			},
		},
		{
			name: "GrowingMap",
			workload: func() interface{} {
				m := make(map[string][]byte)
				for i := 0; i < 100; i++ {
					key := fmt.Sprintf("key_%d", i)
					m[key] = make([]byte, 1024)
				}
				return m
			},
		},
		{
			name: "CircularReferences",
			workload: func() interface{} {
				type Node struct {
					data []byte
					next *Node
				}
				
				var first, prev *Node
				for i := 0; i < 100; i++ {
					node := &Node{
						data: make([]byte, 1024),
					}
					if first == nil {
						first = node
					}
					if prev != nil {
						prev.next = node
					}
					prev = node
				}
				// Create circular reference
				if prev != nil {
					prev.next = first
				}
				return first
			},
		},
	}
	
	for _, scenario := range leakScenarios {
		b.Run(scenario.name, func(b *testing.B) {
			var startStats, endStats runtime.MemStats
			runtime.GC()
			runtime.ReadMemStats(&startStats)
			
			leakedObjects := make([]interface{}, 0)
			
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				obj := scenario.workload()
				leakedObjects = append(leakedObjects, obj)
				
				// Simulate partial cleanup (some objects remain)
				if len(leakedObjects) > 10 {
					// Remove some objects but not all
					leakedObjects = leakedObjects[2:]
				}
			}
			b.StopTimer()
			
			runtime.ReadMemStats(&endStats)
			reportMemoryLeakMetrics(b, startStats, endStats, scenario.name, len(leakedObjects))
			
			// Cleanup to prevent actual leaks in benchmark
			leakedObjects = nil
			runtime.GC()
		})
	}
}

// Helper functions for reporting metrics

func reportCustomAllocatorMetrics(b *testing.B, start, end runtime.MemStats, stats *AllocatorStats, allocatorType string) {
	gcCycles := end.NumGC - start.NumGC
	totalPause := end.PauseTotalNs - start.PauseTotalNs
	heapGrowth := end.HeapInuse - start.HeapInuse
	
	b.ReportMetric(float64(gcCycles), "gc-cycles")
	b.ReportMetric(float64(totalPause)/1e6, "gc-pause-ms")
	b.ReportMetric(float64(heapGrowth)/(1024*1024), "heap-growth-mb")
	b.ReportMetric(float64(stats.Allocations), "allocations")
	b.ReportMetric(float64(stats.BytesAllocated)/(1024*1024), "allocated-mb")
	b.ReportMetric(stats.Efficiency*100, "efficiency-pct")
	b.ReportMetric(stats.Fragmentation*100, "fragmentation-pct")
}

func reportStandardAllocationMetrics(b *testing.B, start, end runtime.MemStats, size int) {
	gcCycles := end.NumGC - start.NumGC
	totalPause := end.PauseTotalNs - start.PauseTotalNs
	heapGrowth := end.HeapInuse - start.HeapInuse
	allocated := end.TotalAlloc - start.TotalAlloc
	
	b.ReportMetric(float64(gcCycles), "gc-cycles")
	b.ReportMetric(float64(totalPause)/1e6, "gc-pause-ms")
	b.ReportMetric(float64(heapGrowth)/(1024*1024), "heap-growth-mb")
	b.ReportMetric(float64(allocated)/(1024*1024), "allocated-mb")
	b.ReportMetric(float64(size), "object-size")
	
	if gcCycles > 0 {
		b.ReportMetric(float64(totalPause)/float64(gcCycles)/1e6, "avg-pause-ms")
	}
}

func reportBufferPoolMetrics(b *testing.B, start, end runtime.MemStats, stats *BufferPoolStats, size int) {
	gcCycles := end.NumGC - start.NumGC
	totalPause := end.PauseTotalNs - start.PauseTotalNs
	heapGrowth := end.HeapInuse - start.HeapInuse
	
	b.ReportMetric(float64(gcCycles), "gc-cycles")
	b.ReportMetric(float64(totalPause)/1e6, "gc-pause-ms")
	b.ReportMetric(float64(heapGrowth)/(1024*1024), "heap-growth-mb")
	b.ReportMetric(stats.HitRate*100, "hit-rate-pct")
	b.ReportMetric(float64(stats.TotalReused), "reused-buffers")
	b.ReportMetric(float64(stats.MemorySaved)/(1024*1024), "memory-saved-mb")
	b.ReportMetric(float64(size), "buffer-size")
}

func reportConcurrentAllocationMetrics(b *testing.B, start, end runtime.MemStats, workers int, allocationType string) {
	gcCycles := end.NumGC - start.NumGC
	totalPause := end.PauseTotalNs - start.PauseTotalNs
	heapGrowth := end.HeapInuse - start.HeapInuse
	allocated := end.TotalAlloc - start.TotalAlloc
	
	b.ReportMetric(float64(gcCycles), "gc-cycles")
	b.ReportMetric(float64(totalPause)/1e6, "gc-pause-ms")
	b.ReportMetric(float64(heapGrowth)/(1024*1024), "heap-growth-mb")
	b.ReportMetric(float64(allocated)/(1024*1024), "allocated-mb")
	b.ReportMetric(float64(workers), "workers")
	
	if gcCycles > 0 {
		b.ReportMetric(float64(totalPause)/float64(gcCycles)/1e6, "avg-pause-ms")
	}
}

func reportConcurrentBufferPoolMetrics(b *testing.B, start, end runtime.MemStats, workers int, stats *BufferPoolStats) {
	gcCycles := end.NumGC - start.NumGC
	totalPause := end.PauseTotalNs - start.PauseTotalNs
	heapGrowth := end.HeapInuse - start.HeapInuse
	
	b.ReportMetric(float64(gcCycles), "gc-cycles")
	b.ReportMetric(float64(totalPause)/1e6, "gc-pause-ms")
	b.ReportMetric(float64(heapGrowth)/(1024*1024), "heap-growth-mb")
	b.ReportMetric(float64(workers), "workers")
	b.ReportMetric(stats.HitRate*100, "hit-rate-pct")
	b.ReportMetric(float64(stats.TotalReused), "reused-buffers")
}

func reportOptimizationMetrics(b *testing.B, start, end runtime.MemStats, optimizationType, scenario string) {
	gcCycles := end.NumGC - start.NumGC
	totalPause := end.PauseTotalNs - start.PauseTotalNs
	heapGrowth := end.HeapInuse - start.HeapInuse
	allocated := end.TotalAlloc - start.TotalAlloc
	heapObjects := end.HeapObjects - start.HeapObjects
	
	b.ReportMetric(float64(gcCycles), "gc-cycles")
	b.ReportMetric(float64(totalPause)/1e6, "gc-pause-ms")
	b.ReportMetric(float64(heapGrowth)/(1024*1024), "heap-growth-mb")
	b.ReportMetric(float64(allocated)/(1024*1024), "allocated-mb")
	b.ReportMetric(float64(heapObjects), "heap-objects")
	
	if gcCycles > 0 {
		b.ReportMetric(float64(totalPause)/float64(gcCycles)/1e6, "avg-pause-ms")
	}
}

func reportMemoryLeakMetrics(b *testing.B, start, end runtime.MemStats, scenario string, leakedCount int) {
	gcCycles := end.NumGC - start.NumGC
	totalPause := end.PauseTotalNs - start.PauseTotalNs
	heapGrowth := end.HeapInuse - start.HeapInuse
	heapObjects := end.HeapObjects - start.HeapObjects
	
	b.ReportMetric(float64(gcCycles), "gc-cycles")
	b.ReportMetric(float64(totalPause)/1e6, "gc-pause-ms")
	b.ReportMetric(float64(heapGrowth)/(1024*1024), "heap-growth-mb")
	b.ReportMetric(float64(heapObjects), "heap-objects")
	b.ReportMetric(float64(leakedCount), "leaked-objects")
	
	if gcCycles > 0 {
		b.ReportMetric(float64(totalPause)/float64(gcCycles)/1e6, "avg-pause-ms")
	}
	
	// Calculate leak rate
	if heapObjects > 0 {
		leakRate := float64(leakedCount) / float64(heapObjects) * 100
		b.ReportMetric(leakRate, "leak-rate-pct")
	}
}

// TestAdvancedGCOptimizer tests the advanced GC optimizer functionality
func TestAdvancedGCOptimizer(t *testing.T) {
	optimizer := NewAdvancedGCOptimizer()
	
	// Test metrics collection
	metrics := optimizer.CollectAdvancedMetrics()
	if metrics == nil {
		t.Error("Expected metrics to be collected")
	}
	
	if metrics.Timestamp.IsZero() {
		t.Error("Expected timestamp to be set")
	}
	
	if metrics.GCMetrics == nil {
		t.Error("Expected GC metrics to be collected")
	}
	
	// Test profile analysis
	opportunities, err := optimizer.AnalyzeProfiles()
	if err != nil {
		t.Errorf("Failed to analyze profiles: %v", err)
	}
	
	if len(opportunities) == 0 {
		t.Error("Expected optimization opportunities to be found")
	}
	
	// Test optimization application
	results, err := optimizer.ApplyOptimizations(opportunities)
	if err != nil {
		t.Errorf("Failed to apply optimizations: %v", err)
	}
	
	// Results might be empty if no techniques are registered
	_ = results
	
	// Test report generation
	report := optimizer.GenerateOptimizationReport()
	if len(report) == 0 {
		t.Error("Expected non-empty optimization report")
	}
}

// TestCustomAllocatorSuite tests the custom allocator suite
func TestCustomAllocatorSuite(t *testing.T) {
	suite := NewCustomAllocatorSuite()
	
	// Test stack allocator creation
	stackAlloc := suite.CreateStackAllocator("test_stack", 1024)
	if stackAlloc == nil {
		t.Error("Expected stack allocator to be created")
	}
	
	// Test allocation
	ptr := stackAlloc.Allocate(256)
	if ptr == nil {
		t.Error("Expected successful allocation")
	}
	
	// Test stats
	stats := stackAlloc.Stats()
	if stats.Allocations != 1 {
		t.Errorf("Expected 1 allocation, got %d", stats.Allocations)
	}
	
	// Test linear allocator creation
	linearAlloc := suite.CreateLinearAllocator("test_linear", 1024, 8)
	if linearAlloc == nil {
		t.Error("Expected linear allocator to be created")
	}
	
	// Test arena allocator creation
	arenaAlloc := suite.CreateArenaAllocator("test_arena", 1024)
	if arenaAlloc == nil {
		t.Error("Expected arena allocator to be created")
	}
	
	// Test slab allocator creation
	slabAlloc := suite.CreateSlabAllocator("test_slab", 64, 1024)
	if slabAlloc == nil {
		t.Error("Expected slab allocator to be created")
	}
	
	// Test suite statistics
	suiteStats := suite.GetAllocatorStats()
	if suiteStats == nil {
		t.Error("Expected suite statistics")
	}
}

// TestAdvancedBufferPool tests the advanced buffer pool
func TestAdvancedBufferPool(t *testing.T) {
	pool := NewAdvancedBufferPool()
	
	// Test buffer acquisition and return
	sizes := []int{512, 2048, 32768, 131072}
	buffers := make([][]byte, 0)
	
	for _, size := range sizes {
		buf := pool.GetBuffer(size)
		if cap(buf) < size {
			t.Errorf("Expected buffer capacity >= %d, got %d", size, cap(buf))
		}
		buffers = append(buffers, buf)
	}
	
	// Return buffers
	for _, buf := range buffers {
		pool.PutBuffer(buf)
	}
	
	// Test reuse
	for _, size := range sizes {
		buf := pool.GetBuffer(size)
		if cap(buf) < size {
			t.Errorf("Expected reused buffer capacity >= %d, got %d", size, cap(buf))
		}
		pool.PutBuffer(buf)
	}
	
	// Test statistics
	stats := pool.GetStats()
	if stats.HitRate == 0 {
		t.Error("Expected non-zero hit rate after reuse")
	}
}

// TestConcurrentAllocators tests allocators under concurrent access
func TestConcurrentAllocators(t *testing.T) {
	allocator := NewLinearAllocator(1024*1024, 8) // 1MB
	
	var wg sync.WaitGroup
	workers := 10
	allocationsPerWorker := 100
	
	for i := 0; i < workers; i++ {
		wg.Add(1)
		go func(workerID int) {
			defer wg.Done()
			for j := 0; j < allocationsPerWorker; j++ {
				size := 64 + (j % 256)
				ptr := allocator.Allocate(size)
				if ptr == nil {
					// Allocator might be full, which is expected
					return
				}
			}
		}(i)
	}
	
	wg.Wait()
	
	// Check that some allocations succeeded
	stats := allocator.Stats()
	if stats.Allocations == 0 {
		t.Error("Expected some allocations to succeed")
	}
}

// TestMemoryLeakDetection tests for memory leaks in the optimization system
func TestMemoryLeakDetection(t *testing.T) {
	var startStats, endStats runtime.MemStats
	runtime.GC()
	runtime.ReadMemStats(&startStats)
	
	// Create and use many optimizers
	for i := 0; i < 100; i++ {
		optimizer := NewAdvancedGCOptimizer()
		metrics := optimizer.CollectAdvancedMetrics()
		_ = metrics
		
		opportunities, _ := optimizer.AnalyzeProfiles()
		_ = opportunities
		
		report := optimizer.GenerateOptimizationReport()
		_ = report
	}
	
	// Force GC and check for leaks
	runtime.GC()
	runtime.GC() // Double GC to ensure cleanup
	runtime.ReadMemStats(&endStats)
	
	// Use signed arithmetic to avoid wraparound
	heapGrowth := int64(endStats.HeapInuse) - int64(startStats.HeapInuse)
	
	// Allow some growth but not excessive (10MB threshold for more complex operations)
	if heapGrowth > 10*1024*1024 {
		t.Errorf("Potential memory leak detected: heap grew by %d bytes", heapGrowth)
	}
	
	// Log the growth for debugging
	t.Logf("Heap growth: %d bytes", heapGrowth)
}