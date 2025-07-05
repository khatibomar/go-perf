package main

import (
	"fmt"
	"testing"
	"unsafe"
)

// Benchmark slice allocation patterns
func BenchmarkDynamicAllocation(b *testing.B) {
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		_ = dynamicAllocation(1000)
	}
}

func BenchmarkPreAllocation(b *testing.B) {
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		_ = preAllocation(1000)
	}
}

func BenchmarkDirectAllocation(b *testing.B) {
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		_ = directAllocation(1000)
	}
}

// Benchmark object pooling
func BenchmarkObjectPool(b *testing.B) {
	pool := NewObjectPool(10, 1024)
	b.ReportAllocs()
	b.ResetTimer()
	
	for i := 0; i < b.N; i++ {
		buf := pool.Get()
		buf = append(buf, byte(i%256))
		pool.Put(buf)
	}
}

func BenchmarkWithoutObjectPool(b *testing.B) {
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		buf := make([]byte, 0, 1024)
		buf = append(buf, byte(i%256))
		_ = buf
	}
}

// Benchmark string building
func BenchmarkInefficientStringBuilding(b *testing.B) {
	parts := make([]string, 100)
	for i := range parts {
		parts[i] = fmt.Sprintf("part_%d_", i)
	}
	
	b.ReportAllocs()
	b.ResetTimer()
	
	for i := 0; i < b.N; i++ {
		_ = inefficientStringBuilding(parts)
	}
}

func BenchmarkEfficientStringBuilding(b *testing.B) {
	parts := make([]string, 100)
	for i := range parts {
		parts[i] = fmt.Sprintf("part_%d_", i)
	}
	
	b.ReportAllocs()
	b.ResetTimer()
	
	for i := 0; i < b.N; i++ {
		_ = efficientStringBuilding(parts)
	}
}

// Benchmark struct allocation
func BenchmarkBadStructAllocation(b *testing.B) {
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		_ = make([]BadStruct, 1000)
	}
}

func BenchmarkGoodStructAllocation(b *testing.B) {
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		_ = make([]GoodStruct, 1000)
	}
}

// Benchmark array vs slice processing
func BenchmarkProcessArray(b *testing.B) {
	arr := [1000]int{}
	for i := range arr {
		arr[i] = i
	}
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = processArray(arr)
	}
}

func BenchmarkProcessSlice(b *testing.B) {
	slice := make([]int, 1000)
	for i := range slice {
		slice[i] = i
	}
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = processSlice(slice)
	}
}

// Benchmark map allocation
func BenchmarkDynamicMap(b *testing.B) {
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		_ = dynamicMap(1000)
	}
}

func BenchmarkPreSizedMap(b *testing.B) {
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		_ = preSizedMap(1000)
	}
}

// Benchmark interface boxing
func BenchmarkProcessInts(b *testing.B) {
	ints := make([]int, 1000)
	for i := range ints {
		ints[i] = i
	}
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = processInts(ints)
	}
}

func BenchmarkProcessInterfaces(b *testing.B) {
	interfaces := make([]interface{}, 1000)
	for i := range interfaces {
		interfaces[i] = i
	}
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = processInterfaces(interfaces)
	}
}

// Comparative benchmarks
func BenchmarkSliceAllocationComparison(b *testing.B) {
	b.Run("Dynamic", BenchmarkDynamicAllocation)
	b.Run("PreAlloc", BenchmarkPreAllocation)
	b.Run("Direct", BenchmarkDirectAllocation)
}

func BenchmarkStringBuildingComparison(b *testing.B) {
	b.Run("Inefficient", BenchmarkInefficientStringBuilding)
	b.Run("Efficient", BenchmarkEfficientStringBuilding)
}

func BenchmarkStructLayoutComparison(b *testing.B) {
	b.Run("BadStruct", BenchmarkBadStructAllocation)
	b.Run("GoodStruct", BenchmarkGoodStructAllocation)
}

func BenchmarkArrayVsSliceComparison(b *testing.B) {
	b.Run("Array", BenchmarkProcessArray)
	b.Run("Slice", BenchmarkProcessSlice)
}

func BenchmarkMapAllocationComparison(b *testing.B) {
	b.Run("Dynamic", BenchmarkDynamicMap)
	b.Run("PreSized", BenchmarkPreSizedMap)
}

func BenchmarkInterfaceBoxingComparison(b *testing.B) {
	b.Run("TypedSlice", BenchmarkProcessInts)
	b.Run("InterfaceSlice", BenchmarkProcessInterfaces)
}

func BenchmarkObjectPoolComparison(b *testing.B) {
	b.Run("WithPool", BenchmarkObjectPool)
	b.Run("WithoutPool", BenchmarkWithoutObjectPool)
}

// Memory allocation size benchmarks
func BenchmarkSliceAllocationSizes(b *testing.B) {
	sizes := []int{10, 100, 1000, 10000}
	
	for _, size := range sizes {
		b.Run(fmt.Sprintf("Dynamic_%d", size), func(b *testing.B) {
			b.ReportAllocs()
			for i := 0; i < b.N; i++ {
				_ = dynamicAllocation(size)
			}
		})
		
		b.Run(fmt.Sprintf("PreAlloc_%d", size), func(b *testing.B) {
			b.ReportAllocs()
			for i := 0; i < b.N; i++ {
				_ = preAllocation(size)
			}
		})
	}
}

// Test correctness
func TestAllocationPatterns(t *testing.T) {
	// Test slice allocations
	size := 100
	dynamic := dynamicAllocation(size)
	preAlloc := preAllocation(size)
	direct := directAllocation(size)
	
	if len(dynamic) != size || len(preAlloc) != size || len(direct) != size {
		t.Errorf("Slice allocation size mismatch")
	}
	
	for i := 0; i < size; i++ {
		if dynamic[i] != i || preAlloc[i] != i || direct[i] != i {
			t.Errorf("Slice allocation content mismatch at index %d", i)
		}
	}
	
	// Test object pool
	pool := NewObjectPool(2, 1024)
	buf1 := pool.Get()
	buf2 := pool.Get()
	buf3 := pool.Get() // Should create new buffer
	
	if cap(buf1) != 1024 || cap(buf2) != 1024 {
		t.Errorf("Object pool buffer capacity mismatch")
	}
	
	pool.Put(buf1)
	pool.Put(buf2)
	pool.Put(buf3)
	
	// Test string building
	parts := []string{"hello", " ", "world"}
	inefficient := inefficientStringBuilding(parts)
	efficient := efficientStringBuilding(parts)
	
	if inefficient != efficient || inefficient != "hello world" {
		t.Errorf("String building mismatch: %s vs %s", inefficient, efficient)
	}
	
	// Test struct sizes
	if unsafe.Sizeof(BadStruct{}) <= unsafe.Sizeof(GoodStruct{}) {
		t.Errorf("Struct size optimization failed")
	}
	
	// Test array vs slice processing
	arr := [1000]int{}
	for i := 0; i < 10; i++ {
		arr[i] = i
	}
	slice := []int{0, 1, 2, 3, 4, 5, 6, 7, 8, 9}
	
	_ = processArray(arr) // Just test that it compiles and runs
	sliceSum := processSlice(slice)
	
	if sliceSum != 45 {
		t.Errorf("Slice processing mismatch: expected 45, got %d", sliceSum)
	}
	
	// Test map allocation
	dynamicMap := dynamicMap(10)
	preSizedMap := preSizedMap(10)
	
	if len(dynamicMap) != len(preSizedMap) || len(dynamicMap) != 10 {
		t.Errorf("Map allocation size mismatch")
	}
	
	// Test interface processing
	ints := []int{1, 2, 3, 4, 5}
	interfaces := []interface{}{1, 2, 3, 4, 5}
	
	intSum := processInts(ints)
	interfaceSum := processInterfaces(interfaces)
	
	if intSum != interfaceSum || intSum != 15 {
		t.Errorf("Interface processing mismatch: %d vs %d", intSum, interfaceSum)
	}
}