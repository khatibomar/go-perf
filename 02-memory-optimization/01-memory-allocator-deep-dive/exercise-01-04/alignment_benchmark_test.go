package main

import (
	"fmt"
	"runtime"
	"sync"
	"testing"
	"time"
	"unsafe"
)

// Test structs for benchmarking
type BadAlignment struct {
	A bool   // 1 byte
	B int64  // 8 bytes
	C bool   // 1 byte
	D int32  // 4 bytes
	E bool   // 1 byte
	// Total: 32 bytes with padding
}

type GoodAlignment struct {
	B int64  // 8 bytes
	D int32  // 4 bytes
	A bool   // 1 byte
	C bool   // 1 byte
	E bool   // 1 byte
	// Total: 24 bytes with padding
}

type CacheUnfriendly struct {
	Hot1 int64     // Frequently accessed
	Cold [56]byte  // Rarely accessed, fills cache line
	Hot2 int64     // Frequently accessed, next cache line
}

type CacheFriendly struct {
	Hot1 int64     // Frequently accessed
	Hot2 int64     // Frequently accessed, same cache line
	Cold [56]byte  // Rarely accessed
}

type ArrayElementBad struct {
	Flag  bool  // 1 byte
	Value int64 // 8 bytes
	// 7 bytes padding per element
}

type ArrayElementGood struct {
	Value int64 // 8 bytes
	Flag  bool  // 1 byte
	// 7 bytes padding per element
}

// Benchmark struct field access patterns
func BenchmarkStructAccess_BadAlignment(b *testing.B) {
	s := BadAlignment{true, 42, false, 123, true}
	b.ResetTimer()
	
	for i := 0; i < b.N; i++ {
		_ = s.A
		_ = s.B
		_ = s.C
		_ = s.D
		_ = s.E
	}
}

func BenchmarkStructAccess_GoodAlignment(b *testing.B) {
	s := GoodAlignment{42, 123, true, false, true}
	b.ResetTimer()
	
	for i := 0; i < b.N; i++ {
		_ = s.A
		_ = s.B
		_ = s.C
		_ = s.D
		_ = s.E
	}
}

// Benchmark array access with different alignments
func BenchmarkArrayAccess_BadAlignment(b *testing.B) {
	const size = 1000
	arr := make([]BadAlignment, size)
	
	// Initialize
	for i := 0; i < size; i++ {
		arr[i] = BadAlignment{true, int64(i), false, int32(i), true}
	}
	
	b.ResetTimer()
	b.ReportAllocs()
	
	for i := 0; i < b.N; i++ {
		sum := int64(0)
		for j := 0; j < size; j++ {
			if arr[j].A {
				sum += arr[j].B
			}
			if arr[j].C {
				sum += int64(arr[j].D)
			}
		}
		_ = sum
	}
}

func BenchmarkArrayAccess_GoodAlignment(b *testing.B) {
	const size = 1000
	arr := make([]GoodAlignment, size)
	
	// Initialize
	for i := 0; i < size; i++ {
		arr[i] = GoodAlignment{int64(i), int32(i), true, false, true}
	}
	
	b.ResetTimer()
	b.ReportAllocs()
	
	for i := 0; i < b.N; i++ {
		sum := int64(0)
		for j := 0; j < size; j++ {
			if arr[j].A {
				sum += arr[j].B
			}
			if arr[j].C {
				sum += int64(arr[j].D)
			}
		}
		_ = sum
	}
}

// Benchmark cache line effects
func BenchmarkCacheLineAccess_Unfriendly(b *testing.B) {
	const size = 1000
	arr := make([]CacheUnfriendly, size)
	
	// Initialize
	for i := 0; i < size; i++ {
		arr[i].Hot1 = int64(i)
		arr[i].Hot2 = int64(i * 2)
	}
	
	b.ResetTimer()
	b.ReportAllocs()
	
	for i := 0; i < b.N; i++ {
		sum := int64(0)
		for j := 0; j < size; j++ {
			sum += arr[j].Hot1 + arr[j].Hot2
		}
		_ = sum
	}
}

func BenchmarkCacheLineAccess_Friendly(b *testing.B) {
	const size = 1000
	arr := make([]CacheFriendly, size)
	
	// Initialize
	for i := 0; i < size; i++ {
		arr[i].Hot1 = int64(i)
		arr[i].Hot2 = int64(i * 2)
	}
	
	b.ResetTimer()
	b.ReportAllocs()
	
	for i := 0; i < b.N; i++ {
		sum := int64(0)
		for j := 0; j < size; j++ {
			sum += arr[j].Hot1 + arr[j].Hot2
		}
		_ = sum
	}
}

// Benchmark array element packing
func BenchmarkArrayElementPacking_Bad(b *testing.B) {
	const size = 10000
	arr := make([]ArrayElementBad, size)
	
	// Initialize
	for i := 0; i < size; i++ {
		arr[i] = ArrayElementBad{i%2 == 0, int64(i)}
	}
	
	b.ResetTimer()
	b.ReportAllocs()
	
	for i := 0; i < b.N; i++ {
		sum := int64(0)
		for j := 0; j < size; j++ {
			if arr[j].Flag {
				sum += arr[j].Value
			}
		}
		_ = sum
	}
}

func BenchmarkArrayElementPacking_Good(b *testing.B) {
	const size = 10000
	arr := make([]ArrayElementGood, size)
	
	// Initialize
	for i := 0; i < size; i++ {
		arr[i] = ArrayElementGood{int64(i), i%2 == 0}
	}
	
	b.ResetTimer()
	b.ReportAllocs()
	
	for i := 0; i < b.N; i++ {
		sum := int64(0)
		for j := 0; j < size; j++ {
			if arr[j].Flag {
				sum += arr[j].Value
			}
		}
		_ = sum
	}
}

// Benchmark separate arrays vs struct arrays
func BenchmarkSeparateArrays(b *testing.B) {
	const size = 10000
	values := make([]int64, size)
	flags := make([]bool, size)
	
	// Initialize
	for i := 0; i < size; i++ {
		values[i] = int64(i)
		flags[i] = i%2 == 0
	}
	
	b.ResetTimer()
	b.ReportAllocs()
	
	for i := 0; i < b.N; i++ {
		sum := int64(0)
		for j := 0; j < size; j++ {
			if flags[j] {
				sum += values[j]
			}
		}
		_ = sum
	}
}

// Benchmark memory allocation patterns
func BenchmarkAllocation_BadAlignment(b *testing.B) {
	b.ReportAllocs()
	
	for i := 0; i < b.N; i++ {
		_ = &BadAlignment{true, int64(i), false, int32(i), true}
	}
}

func BenchmarkAllocation_GoodAlignment(b *testing.B) {
	b.ReportAllocs()
	
	for i := 0; i < b.N; i++ {
		_ = &GoodAlignment{int64(i), int32(i), true, false, true}
	}
}

// Benchmark concurrent access patterns
func BenchmarkConcurrentAccess_BadAlignment(b *testing.B) {
	const numGoroutines = 4
	const size = 1000
	arr := make([]BadAlignment, size)
	
	// Initialize
	for i := 0; i < size; i++ {
		arr[i] = BadAlignment{true, int64(i), false, int32(i), true}
	}
	
	b.ResetTimer()
	b.ReportAllocs()
	
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			sum := int64(0)
			for j := 0; j < size; j++ {
				if arr[j].A {
					sum += arr[j].B
				}
			}
			_ = sum
		}
	})
}

func BenchmarkConcurrentAccess_GoodAlignment(b *testing.B) {
	const numGoroutines = 4
	const size = 1000
	arr := make([]GoodAlignment, size)
	
	// Initialize
	for i := 0; i < size; i++ {
		arr[i] = GoodAlignment{int64(i), int32(i), true, false, true}
	}
	
	b.ResetTimer()
	b.ReportAllocs()
	
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			sum := int64(0)
			for j := 0; j < size; j++ {
				if arr[j].A {
					sum += arr[j].B
				}
			}
			_ = sum
		}
	})
}

// Benchmark false sharing scenarios
func BenchmarkFalseSharing_Bad(b *testing.B) {
	type SharedData struct {
		Counter1 int64 // Will be on same cache line
		Counter2 int64 // Will be on same cache line
	}
	
	data := &SharedData{}
	const numGoroutines = 2
	
	b.ResetTimer()
	b.ReportAllocs()
	
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			// Simulate false sharing
			for i := 0; i < 100; i++ {
				data.Counter1++
				data.Counter2++
			}
		}
	})
}

func BenchmarkFalseSharing_Good(b *testing.B) {
	type PaddedData struct {
		Counter1 int64
		_        [56]byte // Padding to separate cache lines
		Counter2 int64
		_        [56]byte // Padding to separate cache lines
	}
	
	data := &PaddedData{}
	const numGoroutines = 2
	
	b.ResetTimer()
	b.ReportAllocs()
	
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			// No false sharing
			for i := 0; i < 100; i++ {
				data.Counter1++
				data.Counter2++
			}
		}
	})
}

// Benchmark interface overhead
func BenchmarkInterfaceOverhead_Concrete(b *testing.B) {
	type Data struct {
		Value int64
	}
	
	data := Data{42}
	
	b.ResetTimer()
	b.ReportAllocs()
	
	for i := 0; i < b.N; i++ {
		_ = data.Value
	}
}

func BenchmarkInterfaceOverhead_Interface(b *testing.B) {
	type Data struct {
		Value int64
	}
	
	var data interface{} = Data{42}
	
	b.ResetTimer()
	b.ReportAllocs()
	
	for i := 0; i < b.N; i++ {
		if d, ok := data.(Data); ok {
			_ = d.Value
		}
	}
}

// Benchmark struct copying
func BenchmarkStructCopy_Small(b *testing.B) {
	type SmallStruct struct {
		A int64
		B int64
	}
	
	src := SmallStruct{1, 2}
	
	b.ResetTimer()
	b.ReportAllocs()
	
	for i := 0; i < b.N; i++ {
		dst := src // Copy by value
		_ = dst
	}
}

func BenchmarkStructCopy_Large(b *testing.B) {
	type LargeStruct struct {
		Data [1024]byte
	}
	
	src := LargeStruct{}
	
	b.ResetTimer()
	b.ReportAllocs()
	
	for i := 0; i < b.N; i++ {
		dst := src // Copy by value
		_ = dst
	}
}

func BenchmarkStructCopy_Pointer(b *testing.B) {
	type LargeStruct struct {
		Data [1024]byte
	}
	
	src := &LargeStruct{}
	
	b.ResetTimer()
	b.ReportAllocs()
	
	for i := 0; i < b.N; i++ {
		dst := src // Copy pointer
		_ = dst
	}
}

// Test functions for correctness
func TestAlignmentAnalyzer(t *testing.T) {
	analyzer := NewAlignmentAnalyzer()
	
	// Test basic struct analysis
	info := analyzer.AnalyzeStruct(BadAlignment{}, nil)
	
	if info.Size != int(unsafe.Sizeof(BadAlignment{})) {
		t.Errorf("Expected size %d, got %d", unsafe.Sizeof(BadAlignment{}), info.Size)
	}
	
	if len(info.Fields) != 5 {
		t.Errorf("Expected 5 fields, got %d", len(info.Fields))
	}
	
	// Test that padding is calculated correctly
	if info.TotalPadding == 0 {
		t.Error("Expected some padding for BadAlignment struct")
	}
	
	// Test efficiency calculation
	if info.Efficiency <= 0 || info.Efficiency > 1 {
		t.Errorf("Invalid efficiency: %f", info.Efficiency)
	}
}

func TestStructOptimization(t *testing.T) {
	analyzer := NewAlignmentAnalyzer()
	
	// Test optimization
	originalInfo := analyzer.AnalyzeStruct(BadAlignment{}, nil)
	result := analyzer.OptimizeStruct(originalInfo, MinimizeSize)
	
	if result.OptimizedInfo.Size > originalInfo.Size {
		t.Error("Optimization should not increase size")
	}
	
	if len(result.Recommendations) == 0 {
		t.Error("Expected some recommendations")
	}
}

func TestPaddingAnalyzer(t *testing.T) {
	analyzer := NewPaddingAnalyzer()
	
	// Test that analyzer doesn't panic
	defer func() {
		if r := recover(); r != nil {
			t.Errorf("PaddingAnalyzer panicked: %v", r)
		}
	}()
	
	analyzer.AnalyzePaddingScenarios()
}

func TestPerformanceTester(t *testing.T) {
	tester := NewPerformanceTester(1000)
	
	// Test that performance tester doesn't panic
	defer func() {
		if r := recover(); r != nil {
			t.Errorf("PerformanceTester panicked: %v", r)
		}
	}()
	
	tester.TestAlignmentPerformance()
}

func TestConcurrentAccess(t *testing.T) {
	analyzer := NewAlignmentAnalyzer()
	const numGoroutines = 10
	const iterations = 100
	
	var wg sync.WaitGroup
	wg.Add(numGoroutines)
	
	for i := 0; i < numGoroutines; i++ {
		go func() {
			defer wg.Done()
			for j := 0; j < iterations; j++ {
				info := analyzer.AnalyzeStruct(BadAlignment{}, nil)
				_ = analyzer.OptimizeStruct(info, MinimizeSize)
			}
		}()
	}
	
	wg.Wait()
}

func TestMemoryLeaks(t *testing.T) {
	var m1, m2 runtime.MemStats
	runtime.GC()
	runtime.ReadMemStats(&m1)
	
	// Perform operations that might leak
	analyzer := NewAlignmentAnalyzer()
	for i := 0; i < 1000; i++ {
		info := analyzer.AnalyzeStruct(BadAlignment{}, nil)
		_ = analyzer.OptimizeStruct(info, MinimizeSize)
	}
	
	runtime.GC()
	runtime.ReadMemStats(&m2)
	
	// Check for significant memory increase
	if m2.HeapAlloc > m1.HeapAlloc*2 {
		t.Errorf("Potential memory leak: heap grew from %d to %d bytes",
			m1.HeapAlloc, m2.HeapAlloc)
	}
}

// Benchmark real-world scenarios
func BenchmarkRealWorld_WebServer(b *testing.B) {
	// Simulate web server request structure
	type Request struct {
		ID        int64     // 8 bytes - frequently accessed
		Timestamp time.Time // 24 bytes - moderately accessed
		Active    bool      // 1 byte - frequently accessed
		UserID    int64     // 8 bytes - frequently accessed
		Headers   [256]byte // 256 bytes - rarely accessed
		Body      [1024]byte // 1024 bytes - rarely accessed
	}
	
	reqs := make([]Request, 1000)
	for i := range reqs {
		reqs[i] = Request{
			ID:        int64(i),
			Timestamp: time.Now(),
			Active:    true,
			UserID:    int64(i * 2),
		}
	}
	
	b.ResetTimer()
	b.ReportAllocs()
	
	for i := 0; i < b.N; i++ {
		count := 0
		for j := range reqs {
			if reqs[j].Active && reqs[j].ID > 0 {
				count++
			}
		}
		_ = count
	}
}

func BenchmarkRealWorld_DatabaseRecord(b *testing.B) {
	// Simulate database record structure
	type Record struct {
		ID          int64     // 8 bytes - primary key, frequently accessed
		CreatedAt   time.Time // 24 bytes - timestamp
		UpdatedAt   time.Time // 24 bytes - timestamp
		Active      bool      // 1 byte - status flag
		Version     int32     // 4 bytes - version number
		Data        [512]byte // 512 bytes - payload data
		Metadata    [256]byte // 256 bytes - metadata
	}
	
	records := make([]Record, 1000)
	for i := range records {
		records[i] = Record{
			ID:        int64(i),
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
			Active:    true,
			Version:   1,
		}
	}
	
	b.ResetTimer()
	b.ReportAllocs()
	
	for i := 0; i < b.N; i++ {
		count := 0
		for j := range records {
			if records[j].Active && records[j].Version > 0 {
				count++
			}
		}
		_ = count
	}
}

// Example function to demonstrate usage
func ExampleAlignmentAnalyzer() {
	analyzer := NewAlignmentAnalyzer()
	
	type ExampleStruct struct {
		A bool
		B int64
		C bool
	}
	
	info := analyzer.AnalyzeStruct(ExampleStruct{}, nil)
	fmt.Printf("Struct size: %d bytes\n", info.Size)
	fmt.Printf("Total padding: %d bytes\n", info.TotalPadding)
	fmt.Printf("Efficiency: %.1f%%\n", info.Efficiency*100)
	
	// Output:
	// Struct size: 24 bytes
	// Total padding: 14 bytes
	// Efficiency: 41.7%
}