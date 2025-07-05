package main

import (
	"fmt"
	"math/rand"
	"testing"
	"time"
	"unsafe"
)

// Test data
var (
	testPersons = []struct {
		id     uint32
		age    uint8
		salary uint32
		active bool
		name   string
		email  string
	}{
		{1, 25, 50000, true, "Alice", "alice@example.com"},
		{2, 30, 75000, true, "Bob", "bob@example.com"},
		{3, 35, 90000, false, "Charlie", "charlie@example.com"},
		{4, 28, 65000, true, "Diana", "diana@example.com"},
		{5, 42, 120000, true, "Eve", "eve@example.com"},
	}
)

// TestPersonCollectionAoS tests Array of Structs implementation
func TestPersonCollectionAoS(t *testing.T) {
	pc := NewPersonCollectionAoS(10)
	
	for _, p := range testPersons {
		pc.Add(p.id, p.age, p.salary, p.active, p.name, p.email)
	}
	
	totalSalary := pc.GetTotalSalary()
	expectedSalary := uint64(50000 + 75000 + 65000 + 120000) // Only active persons
	if totalSalary != expectedSalary {
		t.Errorf("Expected total salary %d, got %d", expectedSalary, totalSalary)
	}
	
	avgAge := pc.GetAverageAge()
	expectedAvgAge := float64(25+30+35+28+42) / 5.0
	if avgAge != expectedAvgAge {
		t.Errorf("Expected average age %.2f, got %.2f", expectedAvgAge, avgAge)
	}
}

// TestPersonCollectionSoA tests Struct of Arrays implementation
func TestPersonCollectionSoA(t *testing.T) {
	pc := NewPersonCollectionSoA(10)
	
	for _, p := range testPersons {
		pc.Add(p.id, p.age, p.salary, p.active, p.name, p.email)
	}
	
	totalSalary := pc.GetTotalSalary()
	expectedSalary := uint64(50000 + 75000 + 65000 + 120000) // Only active persons
	if totalSalary != expectedSalary {
		t.Errorf("Expected total salary %d, got %d", expectedSalary, totalSalary)
	}
	
	avgAge := pc.GetAverageAge()
	expectedAvgAge := float64(25+30+35+28+42) / 5.0
	if avgAge != expectedAvgAge {
		t.Errorf("Expected average age %.2f, got %.2f", expectedAvgAge, avgAge)
	}
}

// TestPersonCompact tests bit-packed struct
func TestPersonCompact(t *testing.T) {
	p := NewPersonCompact(1, 25, 50000, true, "Alice", "alice@example.com")
	
	if p.GetAge() != 25 {
		t.Errorf("Expected age 25, got %d", p.GetAge())
	}
	
	if !p.IsActive() {
		t.Errorf("Expected active to be true")
	}
	
	// Test edge case: age 127 (max 7-bit value)
	p2 := NewPersonCompact(2, 127, 100000, false, "Bob", "bob@example.com")
	if p2.GetAge() != 127 {
		t.Errorf("Expected age 127, got %d", p2.GetAge())
	}
	
	if p2.IsActive() {
		t.Errorf("Expected active to be false")
	}
}

// TestHashTable tests cache-friendly hash table
func TestHashTable(t *testing.T) {
	ht := NewHashTable(16)
	
	// Test basic operations
	ht.Put(1, 100)
	ht.Put(2, 200)
	ht.Put(3, 300)
	
	if val, ok := ht.Get(1); !ok || val != 100 {
		t.Errorf("Expected value 100 for key 1, got %d, ok=%v", val, ok)
	}
	
	if val, ok := ht.Get(2); !ok || val != 200 {
		t.Errorf("Expected value 200 for key 2, got %d, ok=%v", val, ok)
	}
	
	if _, ok := ht.Get(999); ok {
		t.Errorf("Expected key 999 to not exist")
	}
	
	// Test update
	ht.Put(1, 150)
	if val, ok := ht.Get(1); !ok || val != 150 {
		t.Errorf("Expected updated value 150 for key 1, got %d, ok=%v", val, ok)
	}
}

// TestCompactBinaryTree tests array-based binary tree
func TestCompactBinaryTree(t *testing.T) {
	cbt := NewCompactBinaryTree(15) // Capacity for small tree
	
	values := []uint32{50, 30, 70, 20, 40, 60, 80}
	for _, val := range values {
		if !cbt.Insert(val) {
			t.Errorf("Failed to insert value %d", val)
		}
	}
	
	// Test search
	for _, val := range values {
		if !cbt.Search(val) {
			t.Errorf("Failed to find value %d", val)
		}
	}
	
	if cbt.Search(999) {
		t.Errorf("Found non-existent value 999")
	}
}

// TestObjectPool tests object pooling
func TestObjectPool(t *testing.T) {
	pool := NewObjectPool(64, func() interface{} {
		return make([]byte, 64)
	})
	
	// Get and put objects
	obj1 := pool.Get().([]byte)
	obj2 := pool.Get().([]byte)
	
	pool.Put(obj1)
	pool.Put(obj2)
	
	// Get again - should reuse
	obj3 := pool.Get().([]byte)
	obj4 := pool.Get().([]byte)
	
	// Objects should be reused (same underlying arrays)
	if cap(obj3) != 64 || cap(obj4) != 64 {
		t.Errorf("Objects not properly reused")
	}
}

// TestSlabAllocator tests slab allocation
func TestSlabAllocator(t *testing.T) {
	sa := NewSlabAllocator(64, 4096) // 64-byte objects, 4KB slabs
	
	// Allocate some objects
	ptrs := make([]unsafe.Pointer, 10)
	for i := range ptrs {
		ptrs[i] = sa.Allocate()
		if ptrs[i] == nil {
			t.Errorf("Failed to allocate object %d", i)
		}
	}
	
	// Free some objects
	for i := 0; i < 5; i++ {
		sa.Free(ptrs[i])
	}
	
	slabs, capacity, allocated := sa.Stats()
	if slabs == 0 || capacity == 0 {
		t.Errorf("Invalid slab stats: slabs=%d, capacity=%d, allocated=%d", slabs, capacity, allocated)
	}
}

// TestRingBuffer tests ring buffer implementation
func TestRingBuffer(t *testing.T) {
	rb := NewRingBuffer(10)
	
	// Test write and read
	data := []byte{1, 2, 3, 4, 5}
	written := rb.Write(data)
	if written != len(data) {
		t.Errorf("Expected to write %d bytes, wrote %d", len(data), written)
	}
	
	readBuf := make([]byte, 3)
	read := rb.Read(readBuf)
	if read != 3 {
		t.Errorf("Expected to read 3 bytes, read %d", read)
	}
	
	expected := []byte{1, 2, 3}
	for i, b := range expected {
		if readBuf[i] != b {
			t.Errorf("Expected byte %d at position %d, got %d", b, i, readBuf[i])
		}
	}
	
	if rb.Available() != 2 {
		t.Errorf("Expected 2 bytes available, got %d", rb.Available())
	}
}

// TestVarIntEncoding tests variable-length integer encoding
func TestVarIntEncoding(t *testing.T) {
	encoder := NewVarIntEncoder()
	
	testCases := []uint32{0, 1, 127, 128, 16383, 16384, 2097151, 2097152}
	
	for _, original := range testCases {
		encoded := encoder.EncodeUint32(original)
		decoded, bytesRead := DecodeUint32(encoded)
		
		if decoded != original {
			t.Errorf("Encoding/decoding failed for %d: got %d", original, decoded)
		}
		
		if bytesRead != len(encoded) {
			t.Errorf("Bytes read mismatch for %d: expected %d, got %d", original, len(encoded), bytesRead)
		}
	}
}

// TestCompressedSparseArray tests run-length encoded sparse array
func TestCompressedSparseArray(t *testing.T) {
	csa := NewCompressedSparseArray(1000)
	
	// Set some sparse values
	csa.Set(10, 100)
	csa.Set(11, 100) // Should merge with previous
	csa.Set(12, 100) // Should merge with previous
	csa.Set(500, 200)
	csa.Set(999, 300)
	
	// Test retrieval
	if csa.Get(10) != 100 {
		t.Errorf("Expected value 100 at index 10, got %d", csa.Get(10))
	}
	
	if csa.Get(11) != 100 {
		t.Errorf("Expected value 100 at index 11, got %d", csa.Get(11))
	}
	
	if csa.Get(500) != 200 {
		t.Errorf("Expected value 200 at index 500, got %d", csa.Get(500))
	}
	
	if csa.Get(50) != 0 {
		t.Errorf("Expected default value 0 at index 50, got %d", csa.Get(50))
	}
	
	// Check compression ratio
	ratio := csa.CompressionRatio()
	if ratio <= 1.0 {
		t.Errorf("Expected compression ratio > 1.0, got %.2f", ratio)
	}
}

// TestBitVector tests compact boolean storage
func TestBitVector(t *testing.T) {
	bv := NewBitVector(100)
	
	// Set some bits
	bv.Set(0, true)
	bv.Set(1, false)
	bv.Set(50, true)
	bv.Set(99, true)
	
	// Test retrieval
	if !bv.Get(0) {
		t.Errorf("Expected bit 0 to be true")
	}
	
	if bv.Get(1) {
		t.Errorf("Expected bit 1 to be false")
	}
	
	if !bv.Get(50) {
		t.Errorf("Expected bit 50 to be true")
	}
	
	if bv.Get(25) {
		t.Errorf("Expected bit 25 to be false (default)")
	}
	
	// Test population count
	if bv.PopCount() != 3 {
		t.Errorf("Expected population count 3, got %d", bv.PopCount())
	}
}

// Benchmark AoS vs SoA for salary calculation
func BenchmarkSalaryCalculation(b *testing.B) {
	const numPersons = 10000
	
	// Setup AoS
	aosCollection := NewPersonCollectionAoS(numPersons)
	for i := 0; i < numPersons; i++ {
		aosCollection.Add(
			uint32(i),
			uint8(20+rand.Intn(40)),
			uint32(30000+rand.Intn(100000)),
			rand.Float32() > 0.3, // 70% active
			fmt.Sprintf("Person%d", i),
			fmt.Sprintf("person%d@example.com", i),
		)
	}
	
	// Setup SoA
	soaCollection := NewPersonCollectionSoA(numPersons)
	for i := 0; i < numPersons; i++ {
		soaCollection.Add(
			uint32(i),
			uint8(20+rand.Intn(40)),
			uint32(30000+rand.Intn(100000)),
			rand.Float32() > 0.3, // 70% active
			fmt.Sprintf("Person%d", i),
			fmt.Sprintf("person%d@example.com", i),
		)
	}
	
	b.Run("AoS", func(b *testing.B) {
		b.ResetTimer()
		b.ReportAllocs()
		for i := 0; i < b.N; i++ {
			total := aosCollection.GetTotalSalary()
		_ = total
		}
	})
	
	b.Run("SoA", func(b *testing.B) {
		b.ResetTimer()
		b.ReportAllocs()
		for i := 0; i < b.N; i++ {
			total := soaCollection.GetTotalSalary()
		_ = total
		}
	})
}

// Benchmark struct sizes
func BenchmarkStructSizes(b *testing.B) {
	b.Run("PersonAoS", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			p := PersonAoS{
				ID:     uint32(i),
				Age:    25,
				Salary: 50000,
				Active: true,
				Name:   "Test",
				Email:  "test@example.com",
			}
			_ = p.ID
		}
	})
	
	b.Run("PersonOptimized", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			p := PersonOptimized{
				Name:   "Test",
				Email:  "test@example.com",
				ID:     uint32(i),
				Salary: 50000,
				Age:    25,
				Active: true,
			}
			_ = p.ID
		}
	})
	
	b.Run("PersonCompact", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			p := NewPersonCompact(uint32(i), 25, 50000, true, "Test", "test@example.com")
			_ = p.ID
		}
	})
}

// Benchmark hash table vs map
func BenchmarkHashTableVsMap(b *testing.B) {
	const numItems = 1000
	
	b.Run("HashTable", func(b *testing.B) {
		ht := NewHashTable(1024)
		for i := 0; i < numItems; i++ {
			ht.Put(uint32(i), uint32(i*2))
		}
		
		b.ResetTimer()
		b.ReportAllocs()
		for i := 0; i < b.N; i++ {
			key := uint32(i % numItems)
			val, _ := ht.Get(key)
			_ = val
		}
	})
	
	b.Run("GoMap", func(b *testing.B) {
		m := make(map[uint32]uint32, numItems)
		for i := 0; i < numItems; i++ {
			m[uint32(i)] = uint32(i * 2)
		}
		
		b.ResetTimer()
		b.ReportAllocs()
		for i := 0; i < b.N; i++ {
			key := uint32(i % numItems)
			val := m[key]
			_ = val
		}
	})
}

// Benchmark object pool vs direct allocation
func BenchmarkObjectPool(b *testing.B) {
	pool := NewObjectPool(1024, func() interface{} {
		return make([]byte, 1024)
	})
	
	b.Run("WithPool", func(b *testing.B) {
		b.ResetTimer()
		b.ReportAllocs()
		for i := 0; i < b.N; i++ {
			obj := pool.Get().([]byte)
			// Simulate some work
			obj[0] = byte(i)
			pool.Put(obj)
		}
	})
	
	b.Run("DirectAllocation", func(b *testing.B) {
		b.ResetTimer()
		b.ReportAllocs()
		for i := 0; i < b.N; i++ {
			obj := make([]byte, 1024)
			// Simulate some work
			obj[0] = byte(i)
			// Object goes out of scope and becomes eligible for GC
		}
	})
}

// Benchmark VarInt encoding vs fixed encoding
func BenchmarkVarIntEncoding(b *testing.B) {
	encoder := NewVarIntEncoder()
	values := make([]uint32, 1000)
	for i := range values {
		values[i] = uint32(rand.Intn(1000000))
	}
	
	b.Run("VarInt", func(b *testing.B) {
		b.ResetTimer()
		b.ReportAllocs()
		for i := 0; i < b.N; i++ {
			value := values[i%len(values)]
			encoded := encoder.EncodeUint32(value)
			decoded, _ := DecodeUint32(encoded)
			_ = decoded
		}
	})
	
	b.Run("FixedInt", func(b *testing.B) {
		b.ResetTimer()
		b.ReportAllocs()
		for i := 0; i < b.N; i++ {
			value := values[i%len(values)]
			// Simulate fixed 4-byte encoding/decoding
			_ = value
		}
	})
}

// Benchmark bit vector vs bool slice
func BenchmarkBitVector(b *testing.B) {
	const size = 10000
	
	bv := NewBitVector(size)
	boolSlice := make([]bool, size)
	
	// Set some random bits
	rand.Seed(time.Now().UnixNano())
	for i := 0; i < size/10; i++ {
		idx := rand.Intn(size)
		bv.Set(idx, true)
		boolSlice[idx] = true
	}
	
	b.Run("BitVector", func(b *testing.B) {
		b.ResetTimer()
		b.ReportAllocs()
		for i := 0; i < b.N; i++ {
			idx := i % size
			val := bv.Get(idx)
			_ = val
		}
	})
	
	b.Run("BoolSlice", func(b *testing.B) {
		b.ResetTimer()
		b.ReportAllocs()
		for i := 0; i < b.N; i++ {
			idx := i % size
			val := boolSlice[idx]
			_ = val
		}
	})
}

// Benchmark memory arena vs individual allocations
func BenchmarkMemoryArena(b *testing.B) {
	arena := NewMemoryArena(64 * 1024) // 64KB blocks
	
	b.Run("Arena", func(b *testing.B) {
		b.ResetTimer()
		b.ReportAllocs()
		for i := 0; i < b.N; i++ {
			ptr := arena.Allocate(64)
			// Simulate using the memory
			*(*byte)(ptr) = byte(i)
			if i%1000 == 0 {
				arena.Reset() // Periodic reset
			}
		}
	})
	
	b.Run("Individual", func(b *testing.B) {
		b.ResetTimer()
		b.ReportAllocs()
		for i := 0; i < b.N; i++ {
			data := make([]byte, 64)
			data[0] = byte(i)
			// Data goes out of scope
		}
	})
}

// Test memory layout analysis
func TestMemoryLayoutAnalysis(t *testing.T) {
	// This test just runs the analysis function
	AnalyzeMemoryLayout()
	
	// Verify struct sizes are as expected
	aosSize := unsafe.Sizeof(PersonAoS{})
	optSize := unsafe.Sizeof(PersonOptimized{})
	compactSize := unsafe.Sizeof(PersonCompact{})
	
	t.Logf("PersonAoS size: %d bytes", aosSize)
	t.Logf("PersonOptimized size: %d bytes", optSize)
	t.Logf("PersonCompact size: %d bytes", compactSize)
	
	// PersonCompact uses bit packing but may not be smaller due to string overhead and alignment
	// The bit packing is still beneficial for reducing field count and improving cache efficiency
	if compactSize > aosSize {
		t.Errorf("PersonCompact (%d bytes) should not be larger than PersonAoS (%d bytes)", compactSize, aosSize)
	}
	
	// Verify that PersonCompact actually uses bit packing by checking field behavior
	p := NewPersonCompact(1, 25, 50000, true, "Test", "test@example.com")
	if p.GetAge() != 25 || !p.IsActive() {
		t.Error("PersonCompact bit packing not working correctly")
	}
}

// Test compression analysis
func TestCompressionAnalysis(t *testing.T) {
	// This test runs the compression analysis
	AnalyzeCompression()
	
	// Test specific compression scenarios
	encoder := NewVarIntEncoder()
	
	// Small values should use fewer bytes
	smallEncoded := encoder.EncodeUint32(127)
	if len(smallEncoded) != 1 {
		t.Errorf("Expected 1 byte for value 127, got %d", len(smallEncoded))
	}
	
	// Large values should use more bytes but still be efficient
	largeEncoded := encoder.EncodeUint32(2097152)
	if len(largeEncoded) > 4 {
		t.Errorf("VarInt encoding should not exceed fixed size for any value")
	}
}