package main

import (
	"runtime"
	"sync"
	"testing"
	"unsafe"
)

// BenchmarkAllocationPatterns benchmarks different allocation patterns
func BenchmarkAllocationPatterns(b *testing.B) {
	b.Run("SmallObjects", BenchmarkSmallObjectAllocation)
	b.Run("MediumObjects", BenchmarkMediumObjectAllocation)
	b.Run("LargeObjects", BenchmarkLargeObjectAllocation)
	b.Run("MixedObjects", BenchmarkMixedObjectAllocation)
}

// BenchmarkSmallObjectAllocation benchmarks small object allocations
func BenchmarkSmallObjectAllocation(b *testing.B) {
	b.ResetTimer()
	b.ReportAllocs()
	
	for i := 0; i < b.N; i++ {
		// Allocate small objects (8-64 bytes)
		_ = make([]byte, 32)
	}
}

// BenchmarkMediumObjectAllocation benchmarks medium object allocations
func BenchmarkMediumObjectAllocation(b *testing.B) {
	b.ResetTimer()
	b.ReportAllocs()
	
	for i := 0; i < b.N; i++ {
		// Allocate medium objects (64-1024 bytes)
		_ = make([]byte, 512)
	}
}

// BenchmarkLargeObjectAllocation benchmarks large object allocations
func BenchmarkLargeObjectAllocation(b *testing.B) {
	b.ResetTimer()
	b.ReportAllocs()
	
	for i := 0; i < b.N; i++ {
		// Allocate large objects (>32KB)
		_ = make([]byte, 40960)
	}
}

// BenchmarkMixedObjectAllocation benchmarks mixed allocation patterns
func BenchmarkMixedObjectAllocation(b *testing.B) {
	b.ResetTimer()
	b.ReportAllocs()
	
	for i := 0; i < b.N; i++ {
		switch i % 4 {
		case 0:
			_ = make([]byte, 32)   // Small
		case 1:
			_ = make([]byte, 512)  // Medium
		case 2:
			_ = make([]byte, 1024) // Large-ish
		case 3:
			_ = make([]byte, 40960) // Large
		}
	}
}

// BenchmarkSizeClassEfficiency benchmarks allocation efficiency for different sizes
func BenchmarkSizeClassEfficiency(b *testing.B) {
	sizes := []struct {
		name string
		size int
	}{
		{"8B", 8},
		{"16B", 16},
		{"24B", 24},
		{"32B", 32},
		{"33B_Inefficient", 33}, // Crosses size class boundary
		{"48B", 48},
		{"64B", 64},
		{"65B_Inefficient", 65}, // Crosses size class boundary
		{"80B", 80},
		{"96B", 96},
		{"128B", 128},
		{"129B_Inefficient", 129}, // Crosses size class boundary
		{"256B", 256},
		{"512B", 512},
		{"1024B", 1024},
	}
	
	for _, size := range sizes {
		b.Run(size.name, func(b *testing.B) {
			b.ResetTimer()
			b.ReportAllocs()
			
			for i := 0; i < b.N; i++ {
				_ = make([]byte, size.size)
			}
		})
	}
}

// BenchmarkStructAlignment benchmarks different struct alignments
func BenchmarkStructAlignment(b *testing.B) {
	b.Run("PoorlyAligned", func(b *testing.B) {
		b.ResetTimer()
		b.ReportAllocs()
		
		for i := 0; i < b.N; i++ {
			_ = &PoorlyAlignedStruct{}
		}
	})
	
	b.Run("WellAligned", func(b *testing.B) {
		b.ResetTimer()
		b.ReportAllocs()
		
		for i := 0; i < b.N; i++ {
			_ = &WellAlignedStruct{}
		}
	})
	
	b.Run("Packed", func(b *testing.B) {
		b.ResetTimer()
		b.ReportAllocs()
		
		for i := 0; i < b.N; i++ {
			_ = &PackedStruct{}
		}
	})
}

// BenchmarkSliceAllocation benchmarks different slice allocation patterns
func BenchmarkSliceAllocation(b *testing.B) {
	b.Run("NoPreallocation", func(b *testing.B) {
		b.ResetTimer()
		b.ReportAllocs()
		
		for i := 0; i < b.N; i++ {
			var slice []int
			for j := 0; j < 100; j++ {
				slice = append(slice, j)
			}
			_ = slice
		}
	})
	
	b.Run("WithPreallocation", func(b *testing.B) {
		b.ResetTimer()
		b.ReportAllocs()
		
		for i := 0; i < b.N; i++ {
			slice := make([]int, 0, 100)
			for j := 0; j < 100; j++ {
				slice = append(slice, j)
			}
			_ = slice
		}
	})
	
	b.Run("ExactSize", func(b *testing.B) {
		b.ResetTimer()
		b.ReportAllocs()
		
		for i := 0; i < b.N; i++ {
			slice := make([]int, 100)
			for j := 0; j < 100; j++ {
				slice[j] = j
			}
			_ = slice
		}
	})
}

// BenchmarkMapAllocation benchmarks different map allocation patterns
func BenchmarkMapAllocation(b *testing.B) {
	b.Run("NoPreallocation", func(b *testing.B) {
		b.ResetTimer()
		b.ReportAllocs()
		
		for i := 0; i < b.N; i++ {
			m := make(map[int]int)
			for j := 0; j < 100; j++ {
				m[j] = j * 2
			}
			_ = m
		}
	})
	
	b.Run("WithPreallocation", func(b *testing.B) {
		b.ResetTimer()
		b.ReportAllocs()
		
		for i := 0; i < b.N; i++ {
			m := make(map[int]int, 100)
			for j := 0; j < 100; j++ {
				m[j] = j * 2
			}
			_ = m
		}
	})
}

// BenchmarkStringAllocation benchmarks string allocation patterns
func BenchmarkStringAllocation(b *testing.B) {
	b.Run("StringConcatenation", func(b *testing.B) {
		b.ResetTimer()
		b.ReportAllocs()
		
		for i := 0; i < b.N; i++ {
			var result string
			for j := 0; j < 10; j++ {
				result += "hello"
			}
			_ = result
		}
	})
	
	b.Run("ByteSliceConversion", func(b *testing.B) {
		b.ResetTimer()
		b.ReportAllocs()
		
		for i := 0; i < b.N; i++ {
			buf := make([]byte, 0, 50)
			for j := 0; j < 10; j++ {
				buf = append(buf, "hello"...)
			}
			result := string(buf)
			_ = result
		}
	})
}

// BenchmarkMemoryPooling benchmarks object pooling effectiveness
func BenchmarkMemoryPooling(b *testing.B) {
	// Buffer for pooling
	type Buffer struct {
		data []byte
	}
	
	// Pool for buffers
	var bufferPool = sync.Pool{
		New: func() interface{} {
			return &Buffer{
				data: make([]byte, 0, 1024),
			}
		},
	}
	
	b.Run("WithoutPooling", func(b *testing.B) {
		b.ResetTimer()
		b.ReportAllocs()
		
		for i := 0; i < b.N; i++ {
			buf := &Buffer{
				data: make([]byte, 0, 1024),
			}
			// Simulate usage
			buf.data = append(buf.data, make([]byte, 512)...)
			_ = buf
		}
	})
	
	b.Run("WithPooling", func(b *testing.B) {
		b.ResetTimer()
		b.ReportAllocs()
		
		for i := 0; i < b.N; i++ {
			buf := bufferPool.Get().(*Buffer)
			buf.data = buf.data[:0] // Reset
			
			// Simulate usage
			buf.data = append(buf.data, make([]byte, 512)...)
			
			bufferPool.Put(buf)
		}
	})
}

// Processor interface for benchmark
type Processor interface {
	Process() int
}

// ConcreteProcessor implements Processor
type ConcreteProcessor struct {
	value int
}

// Process implements the Processor interface
func (cp *ConcreteProcessor) Process() int {
	return cp.value * 2
}

// BenchmarkInterfaceAllocation benchmarks interface allocation overhead
func BenchmarkInterfaceAllocation(b *testing.B) {
	
	b.Run("DirectCall", func(b *testing.B) {
		b.ResetTimer()
		b.ReportAllocs()
		
		for i := 0; i < b.N; i++ {
			cp := &ConcreteProcessor{value: i}
			result := cp.Process()
			_ = result
		}
	})
	
	b.Run("InterfaceCall", func(b *testing.B) {
		b.ResetTimer()
		b.ReportAllocs()
		
		for i := 0; i < b.N; i++ {
			var processor Processor = &ConcreteProcessor{value: i}
			result := processor.Process()
			_ = result
		}
	})
}

// BenchmarkEscapeAnalysis benchmarks escape analysis scenarios
func BenchmarkEscapeAnalysis(b *testing.B) {
	b.Run("StackAllocation", func(b *testing.B) {
		b.ResetTimer()
		b.ReportAllocs()
		
		for i := 0; i < b.N; i++ {
			// This should stay on stack
			var x int = i
			_ = x
		}
	})
	
	b.Run("HeapAllocation", func(b *testing.B) {
		b.ResetTimer()
		b.ReportAllocs()
		
		for i := 0; i < b.N; i++ {
			// This will escape to heap
			x := new(int)
			*x = i
			_ = x
		}
	})
	
	b.Run("SliceEscape", func(b *testing.B) {
		b.ResetTimer()
		b.ReportAllocs()
		
		for i := 0; i < b.N; i++ {
			// Large slice will escape to heap
			slice := make([]int, 10000)
			slice[0] = i
			_ = slice
		}
	})
}

// BenchmarkMemoryAlignment benchmarks memory alignment impact
func BenchmarkMemoryAlignment(b *testing.B) {
	// Poorly aligned struct
	type BadAlignment struct {
		a bool   // 1 byte
		b int64  // 8 bytes (7 bytes padding)
		c bool   // 1 byte
		d int32  // 4 bytes (3 bytes padding)
		e bool   // 1 byte (3 bytes padding)
	}
	
	// Well aligned struct
	type GoodAlignment struct {
		b int64  // 8 bytes
		d int32  // 4 bytes
		a bool   // 1 byte
		c bool   // 1 byte
		e bool   // 1 byte (1 byte padding)
	}
	
	b.Run("BadAlignment", func(b *testing.B) {
		b.ResetTimer()
		b.ReportAllocs()
		
		for i := 0; i < b.N; i++ {
			_ = &BadAlignment{}
		}
	})
	
	b.Run("GoodAlignment", func(b *testing.B) {
		b.ResetTimer()
		b.ReportAllocs()
		
		for i := 0; i < b.N; i++ {
			_ = &GoodAlignment{}
		}
	})
	
	// Report size differences
	b.Logf("BadAlignment size: %d bytes", unsafe.Sizeof(BadAlignment{}))
	b.Logf("GoodAlignment size: %d bytes", unsafe.Sizeof(GoodAlignment{}))
}

// BenchmarkGCPressure benchmarks GC pressure from different allocation patterns
func BenchmarkGCPressure(b *testing.B) {
	b.Run("HighGCPressure", func(b *testing.B) {
		b.ResetTimer()
		b.ReportAllocs()
		
		for i := 0; i < b.N; i++ {
			// Many small allocations create GC pressure
			for j := 0; j < 100; j++ {
				_ = make([]byte, 32)
			}
		}
	})
	
	b.Run("LowGCPressure", func(b *testing.B) {
		b.ResetTimer()
		b.ReportAllocs()
		
		for i := 0; i < b.N; i++ {
			// Fewer, larger allocations
			_ = make([]byte, 3200) // 100 * 32
		}
	})
}

// TestAllocationAnalysis runs allocation analysis tests
func TestAllocationAnalysis(t *testing.T) {
	t.Run("SizeClassMapping", func(t *testing.T) {
		testCases := []struct {
			size     uintptr
			expected int
		}{
			{0, 0},
			{1, 8},
			{8, 8},
			{9, 16},
			{16, 16},
			{17, 24},
			{32, 32},
			{33, 48},
			{64, 64},
			{65, 80},
			{32768, 32768},
			{32769, -1}, // Large object
		}
		
		for _, tc := range testCases {
			result := GetSizeClass(tc.size)
			if result != tc.expected {
				t.Errorf("GetSizeClass(%d) = %d, expected %d", tc.size, result, tc.expected)
			}
		}
	})
	
	t.Run("StructSizes", func(t *testing.T) {
		// Verify struct sizes are as expected
		poorSize := unsafe.Sizeof(PoorlyAlignedStruct{})
		goodSize := unsafe.Sizeof(WellAlignedStruct{})
		packedSize := unsafe.Sizeof(PackedStruct{})
		
		t.Logf("PoorlyAlignedStruct: %d bytes", poorSize)
		t.Logf("WellAlignedStruct: %d bytes", goodSize)
		t.Logf("PackedStruct: %d bytes", packedSize)
		
		if goodSize >= poorSize {
			t.Errorf("WellAlignedStruct (%d) should be smaller than PoorlyAlignedStruct (%d)", goodSize, poorSize)
		}
	})
	
	t.Run("MemoryStats", func(t *testing.T) {
		var m1, m2 runtime.MemStats
		
		// Measure before allocation
		runtime.GC()
		runtime.ReadMemStats(&m1)
		
		// Allocate some memory
		var ptrs []interface{}
		for i := 0; i < 1000; i++ {
			ptrs = append(ptrs, make([]byte, 64))
		}
		
		// Measure after allocation
		runtime.ReadMemStats(&m2)
		
		// Verify allocations occurred
		if m2.TotalAlloc <= m1.TotalAlloc {
			t.Error("Expected allocations to increase total allocated memory")
		}
		
		if m2.Mallocs <= m1.Mallocs {
			t.Error("Expected malloc count to increase")
		}
		
		t.Logf("Allocated %d bytes in %d mallocs", 
			m2.TotalAlloc-m1.TotalAlloc, m2.Mallocs-m1.Mallocs)
		
		// Keep references to prevent GC
		_ = ptrs
	})
}

// Example of running benchmarks with memory profiling:
// go test -bench=. -benchmem -memprofile=mem.prof
// go tool pprof mem.prof