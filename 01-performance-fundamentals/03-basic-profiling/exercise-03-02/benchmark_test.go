package main

import (
	"fmt"
	"runtime"
	"testing"
)

// BenchmarkStringConcatenationBad benchmarks inefficient string concatenation
func BenchmarkStringConcatenationBad(b *testing.B) {
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		_ = stringConcatenationBad(1000)
	}
}

// BenchmarkSliceGrowthBad benchmarks inefficient slice growth
func BenchmarkSliceGrowthBad(b *testing.B) {
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		_ = sliceGrowthBad(10000)
	}
}

// BenchmarkMapAllocationsBad benchmarks inefficient map usage
func BenchmarkMapAllocationsBad(b *testing.B) {
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		_ = mapAllocationsBad(1000)
	}
}

// BenchmarkInterfaceAllocationsBad benchmarks interface allocation overhead
func BenchmarkInterfaceAllocationsBad(b *testing.B) {
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		_ = interfaceAllocationsBad(5000)
	}
}

// BenchmarkClosureAllocationsBad benchmarks closure allocation issues
func BenchmarkClosureAllocationsBad(b *testing.B) {
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		_ = closureAllocationsBad(100)
	}
}

// BenchmarkChannelAllocationsBad benchmarks inefficient channel usage
func BenchmarkChannelAllocationsBad(b *testing.B) {
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		channelAllocationsBad(10)
	}
}

// BenchmarkCreateDeepTreeBad benchmarks recursive structure creation
func BenchmarkCreateDeepTreeBad(b *testing.B) {
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		_ = createDeepTreeBad(8, 3)
	}
}

// BenchmarkTemporaryAllocationsBad benchmarks excessive temporary allocations
func BenchmarkTemporaryAllocationsBad(b *testing.B) {
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		_ = temporaryAllocationsBad(100)
	}
}

// BenchmarkStringProcessingBad benchmarks inefficient string processing
func BenchmarkStringProcessingBad(b *testing.B) {
	b.ReportAllocs()
	testData := generateTestData(100)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = stringProcessingBad(testData)
	}
}

// BenchmarkMemoryLeaker benchmarks memory leak patterns
func BenchmarkMemoryLeaker(b *testing.B) {
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		leaker := NewMemoryLeaker()
		leaker.LeakMemory(10)
	}
}

// BenchmarkMemoryIntensiveWorkload benchmarks combined memory operations
func BenchmarkMemoryIntensiveWorkload(b *testing.B) {
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		memoryIntensiveWorkload(5)
	}
}

// Parametric benchmarks for different scales
func BenchmarkStringConcatenation(b *testing.B) {
	sizes := []int{100, 500, 1000, 2000}
	for _, size := range sizes {
		b.Run(fmt.Sprintf("Size%d", size), func(b *testing.B) {
			b.ReportAllocs()
			for i := 0; i < b.N; i++ {
				_ = stringConcatenationBad(size)
			}
		})
	}
}

func BenchmarkSliceGrowth(b *testing.B) {
	sizes := []int{1000, 5000, 10000, 20000}
	for _, size := range sizes {
		b.Run(fmt.Sprintf("Size%d", size), func(b *testing.B) {
			b.ReportAllocs()
			for i := 0; i < b.N; i++ {
				_ = sliceGrowthBad(size)
			}
		})
	}
}

func BenchmarkMapAllocations(b *testing.B) {
	sizes := []int{100, 500, 1000, 2000}
	for _, size := range sizes {
		b.Run(fmt.Sprintf("Size%d", size), func(b *testing.B) {
			b.ReportAllocs()
			for i := 0; i < b.N; i++ {
				_ = mapAllocationsBad(size)
			}
		})
	}
}

func BenchmarkTreeDepth(b *testing.B) {
	depths := []int{5, 7, 8, 10}
	for _, depth := range depths {
		b.Run(fmt.Sprintf("Depth%d", depth), func(b *testing.B) {
			b.ReportAllocs()
			for i := 0; i < b.N; i++ {
				_ = createDeepTreeBad(depth, 3)
			}
		})
	}
}

// Memory usage benchmarks with runtime stats
func BenchmarkMemoryUsageTracking(b *testing.B) {
	b.ReportAllocs()
	
	var m1, m2 runtime.MemStats
	runtime.GC()
	runtime.ReadMemStats(&m1)
	
	for i := 0; i < b.N; i++ {
		memoryIntensiveWorkload(3)
	}
	
	runtime.GC()
	runtime.ReadMemStats(&m2)
	
	b.ReportMetric(float64(m2.Alloc-m1.Alloc)/float64(b.N), "bytes/op")
	b.ReportMetric(float64(m2.TotalAlloc-m1.TotalAlloc)/float64(b.N), "total-bytes/op")
	b.ReportMetric(float64(m2.Mallocs-m1.Mallocs)/float64(b.N), "allocs/op")
}

// Garbage collection pressure benchmark
func BenchmarkGCPressure(b *testing.B) {
	b.ReportAllocs()
	
	var gcBefore, gcAfter runtime.MemStats
	runtime.ReadMemStats(&gcBefore)
	
	for i := 0; i < b.N; i++ {
		// Create lots of short-lived objects
		for j := 0; j < 1000; j++ {
			_ = make([]byte, 1024)
			_ = make(map[int]string)
			_ = fmt.Sprintf("temp_%d_%d", i, j)
		}
	}
	
	runtime.ReadMemStats(&gcAfter)
	b.ReportMetric(float64(gcAfter.NumGC-gcBefore.NumGC)/float64(b.N), "gc-cycles/op")
}

// Heap growth benchmark
func BenchmarkHeapGrowth(b *testing.B) {
	b.ReportAllocs()
	
	data := make([]*MemoryLeaker, b.N)
	
	for i := 0; i < b.N; i++ {
		data[i] = NewMemoryLeaker()
		data[i].LeakMemory(5)
	}
	
	// Keep reference to prevent GC
	runtime.KeepAlive(data)
}

// Memory allocation rate benchmark
func BenchmarkAllocationRate(b *testing.B) {
	b.ReportAllocs()
	b.SetBytes(1024) // Assume 1KB per operation
	
	for i := 0; i < b.N; i++ {
		// Rapid allocations
		_ = make([]byte, 1024)
		_ = make([]int, 256)
		_ = make(map[string]int)
		_ = &TreeNode{
			Value: i,
			Data:  make([]byte, 512),
		}
	}
}