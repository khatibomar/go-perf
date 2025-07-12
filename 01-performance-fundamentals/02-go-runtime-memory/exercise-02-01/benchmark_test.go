package main

import (
	"runtime"
	"testing"
)

// validateBenchmarkEnvironment ensures the benchmark environment is suitable
// this pattern is useful when running in CI
func validateBenchmarkEnvironment(b *testing.B) {
	if testing.Short() {
		b.Skip("Skipping benchmark in short mode")
	}

	var m runtime.MemStats
	runtime.ReadMemStats(&m)
	if m.Sys < 10*1024*1024 { // Less than 10MB
		b.Skip("Insufficient memory for reliable benchmarking")
	}
}

// Benchmark stack allocation
func BenchmarkStackAllocation(b *testing.B) {
	validateBenchmarkEnvironment(b)
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		_ = stackAllocation()
	}
}

// Benchmark heap allocation via pointer
func BenchmarkHeapAllocationPointer(b *testing.B) {
	validateBenchmarkEnvironment(b)
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		_ = heapAllocationPointer()
	}
}

// Benchmark stack allocation with slice
func BenchmarkStackAllocationSlice(b *testing.B) {
	validateBenchmarkEnvironment(b)
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		_ = stackAllocationSlice()
	}
}

// Benchmark heap allocation with slice
func BenchmarkHeapAllocationSlice(b *testing.B) {
	validateBenchmarkEnvironment(b)
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		_ = heapAllocationSlice()
	}
}

// Benchmark small array (stack)
func BenchmarkSmallArrayAllocation(b *testing.B) {
	validateBenchmarkEnvironment(b)
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		_ = smallArrayAllocation()
	}
}

// Benchmark large array (heap)
func BenchmarkLargeArrayAllocation(b *testing.B) {
	validateBenchmarkEnvironment(b)
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		_ = largeArrayAllocation()
	}
}

// Benchmark struct stack allocation
func BenchmarkStructStackAllocation(b *testing.B) {
	validateBenchmarkEnvironment(b)
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		_ = structStackAllocation()
	}
}

// Benchmark struct heap allocation
func BenchmarkStructHeapAllocation(b *testing.B) {
	validateBenchmarkEnvironment(b)
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		_ = structHeapAllocation()
	}
}

// Benchmark closure without escape
func BenchmarkNoEscapeClosure(b *testing.B) {
	validateBenchmarkEnvironment(b)
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		_ = noEscapeClosure()
	}
}

// Benchmark closure with escape
func BenchmarkClosureAllocation(b *testing.B) {
	validateBenchmarkEnvironment(b)
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		_ = closureAllocation()
	}
}

// Benchmark interface allocation
func BenchmarkInterfaceAllocation(b *testing.B) {
	validateBenchmarkEnvironment(b)
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		interfaceAllocation(i)
	}
}

// Benchmark conditional escape (conservative compiler)
func BenchmarkConditionalEscape(b *testing.B) {
	validateBenchmarkEnvironment(b)
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		_ = conditionalEscape(false)
	}
}

// Benchmark slice append escape
func BenchmarkSliceAppendEscape(b *testing.B) {
	validateBenchmarkEnvironment(b)
	b.ReportAllocs()
	slice := make([]int, 0, 100)
	for i := 0; i < b.N; i++ {
		slice = sliceAppendEscape(slice[:0])
	}
}

// Comparative benchmarks
func BenchmarkStackVsHeapInt(b *testing.B) {
	validateBenchmarkEnvironment(b)
	b.Run("Stack", func(b *testing.B) {
		b.ReportAllocs()
		for i := 0; i < b.N; i++ {
			_ = stackAllocation()
		}
	})

	b.Run("Heap", func(b *testing.B) {
		b.ReportAllocs()
		for i := 0; i < b.N; i++ {
			_ = heapAllocationPointer()
		}
	})
}

func BenchmarkStackVsHeapSlice(b *testing.B) {
	validateBenchmarkEnvironment(b)
	b.Run("Stack", func(b *testing.B) {
		b.ReportAllocs()
		for i := 0; i < b.N; i++ {
			_ = stackAllocationSlice()
		}
	})

	b.Run("Heap", func(b *testing.B) {
		b.ReportAllocs()
		for i := 0; i < b.N; i++ {
			_ = heapAllocationSlice()
		}
	})
}

func BenchmarkStackVsHeapStruct(b *testing.B) {
	validateBenchmarkEnvironment(b)
	b.Run("Stack", func(b *testing.B) {
		b.ReportAllocs()
		for i := 0; i < b.N; i++ {
			_ = structStackAllocation()
		}
	})

	b.Run("Heap", func(b *testing.B) {
		b.ReportAllocs()
		for i := 0; i < b.N; i++ {
			_ = structHeapAllocation()
		}
	})
}

// Test correctness
func TestEscapeAnalysisFunctions(t *testing.T) {
	// Test stack allocation
	if result := stackAllocation(); result != 42 {
		t.Errorf("stackAllocation() = %d, want 42", result)
	}

	// Test heap allocation pointer
	if result := heapAllocationPointer(); *result != 42 {
		t.Errorf("heapAllocationPointer() = %d, want 42", *result)
	}

	// Test slice allocations
	if result := stackAllocationSlice(); result != 15 {
		t.Errorf("stackAllocationSlice() = %d, want 15", result)
	}

	if result := heapAllocationSlice(); len(result) != 5 {
		t.Errorf("heapAllocationSlice() length = %d, want 5", len(result))
	}

	// Test array allocations
	smallArr := smallArrayAllocation()
	if smallArr[0] != 0 || smallArr[9] != 9 {
		t.Errorf("smallArrayAllocation() incorrect values")
	}

	largeArr := largeArrayAllocation()
	if largeArr[0] != 0 || largeArr[99999] != 99999 {
		t.Errorf("largeArrayAllocation() incorrect values")
	}

	// Test struct allocations
	stackStruct := structStackAllocation()
	if stackStruct.Name != "John" || stackStruct.Age != 30 {
		t.Errorf("structStackAllocation() incorrect values")
	}

	heapStruct := structHeapAllocation()
	if heapStruct.Name != "John" || heapStruct.Age != 30 {
		t.Errorf("structHeapAllocation() incorrect values")
	}

	// Test closure
	if result := noEscapeClosure(); result != 42 {
		t.Errorf("noEscapeClosure() = %d, want 42", result)
	}

	closureFn := closureAllocation()
	if result := closureFn(); result != 42 {
		t.Errorf("closureAllocation()() = %d, want 42", result)
	}

	// Test conditional escape
	result := conditionalEscape(false)
	if result.(int) != 42 {
		t.Errorf("conditionalEscape(false) = %v, want 42", result)
	}

	result = conditionalEscape(true)
	if *result.(*int) != 42 {
		t.Errorf("conditionalEscape(true) = %v, want pointer to 42", result)
	}
}
