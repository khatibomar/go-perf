package main

import (
	"fmt"
	"testing"
	"unsafe"
)

// Benchmark struct layouts
func BenchmarkBadLayoutAllocation(b *testing.B) {
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		_ = make([]BadLayout, 1000)
	}
}

func BenchmarkGoodLayoutAllocation(b *testing.B) {
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		_ = make([]GoodLayout, 1000)
	}
}

func BenchmarkOptimalLayoutAllocation(b *testing.B) {
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		_ = make([]OptimalLayout, 1000)
	}
}

// Benchmark cache-friendly vs cache-unfriendly access patterns
func BenchmarkCacheUnfriendlyAccess(b *testing.B) {
	data := make([]CacheUnfriendly, 1000)
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		sum := int64(0)
		for j := range data {
			// Access hot data scattered throughout the struct
			sum += data[j].HotData1
			sum += data[j].HotData2
			sum += data[j].HotData3
		}
		_ = sum
	}
}

func BenchmarkCacheFriendlyAccess(b *testing.B) {
	data := make([]CacheFriendly, 1000)
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		sum := int64(0)
		for j := range data {
			// Access hot data grouped together
			sum += data[j].HotData1
			sum += data[j].HotData2
			sum += data[j].HotData3
		}
		_ = sum
	}
}

// Benchmark aligned vs misaligned struct access
func BenchmarkAlignedStructAccess(b *testing.B) {
	data := make([]Aligned, 1000)
	for i := range data {
		data[i] = Aligned{a: int64(i), b: int32(i), c: int16(i), d: int8(i)}
	}
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		sum := int64(0)
		for j := range data {
			sum += data[j].a
			sum += int64(data[j].b)
			sum += int64(data[j].c)
			sum += int64(data[j].d)
		}
		_ = sum
	}
}

func BenchmarkMisalignedStructAccess(b *testing.B) {
	data := make([]Misaligned, 1000)
	for i := range data {
		data[i] = Misaligned{a: int8(i), b: int64(i), c: int8(i), d: int32(i), e: int8(i), f: int16(i)}
	}
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		sum := int64(0)
		for j := range data {
			sum += int64(data[j].a)
			sum += data[j].b
			sum += int64(data[j].c)
			sum += int64(data[j].d)
			sum += int64(data[j].e)
			sum += int64(data[j].f)
		}
		_ = sum
	}
}

// Benchmark array vs slice access
func BenchmarkSmallArrayAccess(b *testing.B) {
	var data SmallArray
	for i := range data {
		data[i] = i
	}
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		sum := 0
		for j := range data {
			sum += data[j]
		}
		_ = sum
	}
}

func BenchmarkSmallSliceAccess(b *testing.B) {
	data := make([]int, 10)
	for i := range data {
		data[i] = i
	}
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		sum := 0
		for j := range data {
			sum += data[j]
		}
		_ = sum
	}
}

// Benchmark pointer vs value access
func BenchmarkPointerChainAccess(b *testing.B) {
	// Create a chain of nodes
	var head *Node
	for i := 0; i < 1000; i++ {
		head = &Node{Value: i, Next: head}
	}
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		sum := 0
		current := head
		for current != nil {
			sum += current.Value
			current = current.Next
		}
		_ = sum
	}
}

func BenchmarkValueArrayAccess(b *testing.B) {
	data := make([]int, 1000)
	for i := range data {
		data[i] = i
	}
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		sum := 0
		for j := range data {
			sum += data[j]
		}
		_ = sum
	}
}

// Benchmark interface overhead
func BenchmarkDirectAccess(b *testing.B) {
	data := make([]int, 1000)
	for i := range data {
		data[i] = i
	}
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		sum := 0
		for j := range data {
			sum += data[j]
		}
		_ = sum
	}
}

func BenchmarkInterfaceAccess(b *testing.B) {
	data := make([]interface{}, 1000)
	for i := range data {
		data[i] = i
	}
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		sum := 0
		for j := range data {
			if v, ok := data[j].(int); ok {
				sum += v
			}
		}
		_ = sum
	}
}

// Benchmark memory allocation patterns
func BenchmarkStackAllocation(b *testing.B) {
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		var data [100]int
		for j := range data {
			data[j] = j
		}
		_ = data
	}
}

func BenchmarkHeapAllocation(b *testing.B) {
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		data := make([]int, 100)
		for j := range data {
			data[j] = j
		}
		_ = data
	}
}

// Benchmark string vs []byte operations
func BenchmarkStringConcatenation(b *testing.B) {
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		result := ""
		for j := 0; j < 100; j++ {
			result += "a"
		}
		_ = result
	}
}

func BenchmarkByteSliceAppend(b *testing.B) {
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		result := make([]byte, 0, 100)
		for j := 0; j < 100; j++ {
			result = append(result, 'a')
		}
		_ = string(result)
	}
}

// Comparative benchmarks
func BenchmarkStructLayoutComparison(b *testing.B) {
	b.Run("BadLayout", BenchmarkBadLayoutAllocation)
	b.Run("GoodLayout", BenchmarkGoodLayoutAllocation)
	b.Run("OptimalLayout", BenchmarkOptimalLayoutAllocation)
}

func BenchmarkCacheEffectsComparison(b *testing.B) {
	b.Run("CacheUnfriendly", BenchmarkCacheUnfriendlyAccess)
	b.Run("CacheFriendly", BenchmarkCacheFriendlyAccess)
}

func BenchmarkAlignmentComparison(b *testing.B) {
	b.Run("Aligned", BenchmarkAlignedStructAccess)
	b.Run("Misaligned", BenchmarkMisalignedStructAccess)
}

func BenchmarkDataStructureComparison(b *testing.B) {
	b.Run("PointerChain", BenchmarkPointerChainAccess)
	b.Run("ValueArray", BenchmarkValueArrayAccess)
}

func BenchmarkInterfaceOverheadComparison(b *testing.B) {
	b.Run("Direct", BenchmarkDirectAccess)
	b.Run("Interface", BenchmarkInterfaceAccess)
}

func BenchmarkAllocationComparison(b *testing.B) {
	b.Run("Stack", BenchmarkStackAllocation)
	b.Run("Heap", BenchmarkHeapAllocation)
}

func BenchmarkStringVsBytesComparison(b *testing.B) {
	b.Run("StringConcat", BenchmarkStringConcatenation)
	b.Run("ByteSliceAppend", BenchmarkByteSliceAppend)
}

// Benchmark different struct sizes
func BenchmarkStructSizes(b *testing.B) {
	sizes := []struct {
		name string
		size uintptr
		fn   func(*testing.B)
	}{
		{"BadLayout", unsafe.Sizeof(BadLayout{}), BenchmarkBadLayoutAllocation},
		{"GoodLayout", unsafe.Sizeof(GoodLayout{}), BenchmarkGoodLayoutAllocation},
		{"OptimalLayout", unsafe.Sizeof(OptimalLayout{}), BenchmarkOptimalLayoutAllocation},
	}
	
	for _, size := range sizes {
		b.Run(fmt.Sprintf("%s_%dbytes", size.name, size.size), size.fn)
	}
}

// Test correctness
func TestMemoryLayout(t *testing.T) {
	// Test struct sizes
	badSize := unsafe.Sizeof(BadLayout{})
	goodSize := unsafe.Sizeof(GoodLayout{})
	optimalSize := unsafe.Sizeof(OptimalLayout{})
	
	if goodSize >= badSize {
		t.Errorf("GoodLayout (%d bytes) should be smaller than BadLayout (%d bytes)", goodSize, badSize)
	}
	
	if optimalSize > goodSize {
		t.Errorf("OptimalLayout (%d bytes) should be smaller than or equal to GoodLayout (%d bytes)", optimalSize, goodSize)
	}
	
	// Test cache-friendly vs unfriendly sizes
	unfriendlySize := unsafe.Sizeof(CacheUnfriendly{})
	friendlySize := unsafe.Sizeof(CacheFriendly{})
	
	if unfriendlySize != friendlySize {
		t.Logf("Cache layouts have different sizes: unfriendly=%d, friendly=%d", unfriendlySize, friendlySize)
	}
	
	// Test alignment
	alignedSize := unsafe.Sizeof(Aligned{})
	misalignedSize := unsafe.Sizeof(Misaligned{})
	
	if alignedSize >= misalignedSize {
		t.Errorf("Aligned struct (%d bytes) should be smaller than misaligned struct (%d bytes)", alignedSize, misalignedSize)
	}
	
	// Test basic type alignments
	if unsafe.Alignof(int8(0)) != 1 {
		t.Errorf("int8 alignment should be 1, got %d", unsafe.Alignof(int8(0)))
	}
	
	if unsafe.Alignof(int16(0)) != 2 {
		t.Errorf("int16 alignment should be 2, got %d", unsafe.Alignof(int16(0)))
	}
	
	if unsafe.Alignof(int32(0)) != 4 {
		t.Errorf("int32 alignment should be 4, got %d", unsafe.Alignof(int32(0)))
	}
	
	if unsafe.Alignof(int64(0)) != 8 {
		t.Errorf("int64 alignment should be 8, got %d", unsafe.Alignof(int64(0)))
	}
	
	// Test slice and string header sizes
	sliceSize := unsafe.Sizeof([]int{})
	stringSize := unsafe.Sizeof(string(""))
	interfaceSize := unsafe.Sizeof(interface{}(nil))
	
	expectedSliceSize := uintptr(24) // 3 * 8 bytes on 64-bit
	expectedStringSize := uintptr(16) // 2 * 8 bytes on 64-bit
	expectedInterfaceSize := uintptr(16) // 2 * 8 bytes on 64-bit
	
	if sliceSize != expectedSliceSize {
		t.Errorf("Slice header size should be %d, got %d", expectedSliceSize, sliceSize)
	}
	
	if stringSize != expectedStringSize {
		t.Errorf("String header size should be %d, got %d", expectedStringSize, stringSize)
	}
	
	if interfaceSize != expectedInterfaceSize {
		t.Errorf("Interface header size should be %d, got %d", expectedInterfaceSize, interfaceSize)
	}
	
	// Test array sizes
	smallArraySize := unsafe.Sizeof(SmallArray{})
	largeArraySize := unsafe.Sizeof(LargeArray{})
	
	expectedSmallSize := uintptr(10 * 8) // 10 ints * 8 bytes each
	expectedLargeSize := uintptr(10000 * 8) // 10000 ints * 8 bytes each
	
	if smallArraySize != expectedSmallSize {
		t.Errorf("Small array size should be %d, got %d", expectedSmallSize, smallArraySize)
	}
	
	if largeArraySize != expectedLargeSize {
		t.Errorf("Large array size should be %d, got %d", expectedLargeSize, largeArraySize)
	}
}

// Benchmark memory copying
func BenchmarkMemoryCopy(b *testing.B) {
	src := make([]byte, 1024)
	dst := make([]byte, 1024)
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		copy(dst, src)
	}
}

func BenchmarkMemorySet(b *testing.B) {
	data := make([]byte, 1024)
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		for j := range data {
			data[j] = 0
		}
	}
}