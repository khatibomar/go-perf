package main

import (
	"fmt"
	"runtime"
	"time"
)

func main() {
	fmt.Println("=== Memory Optimization Patterns Demo ===")
	fmt.Println()
	
	// Demonstrate memory layout analysis
	fmt.Println("1. Memory Layout Analysis:")
	AnalyzeMemoryLayout()
	fmt.Println()
	
	// Demonstrate compression analysis
	fmt.Println("2. Compression Analysis:")
	AnalyzeCompression()
	fmt.Println()
	
	// Demonstrate AoS vs SoA performance
	fmt.Println("3. AoS vs SoA Performance Comparison:")
	demonstrateAoSvsSoA()
	fmt.Println()
	
	// Demonstrate memory pooling
	fmt.Println("4. Memory Pooling Demo:")
	demonstrateMemoryPooling()
	fmt.Println()
	
	// Demonstrate compact data formats
	fmt.Println("5. Compact Data Formats Demo:")
	demonstrateCompactFormats()
	fmt.Println()
	
	fmt.Println("Demo completed! Run 'go test -bench=. -benchmem' to see detailed benchmarks.")
}

func demonstrateAoSvsSoA() {
	const numPersons = 10000
	
	// Setup AoS collection
	aosCollection := NewPersonCollectionAoS(numPersons)
	for i := 0; i < numPersons; i++ {
		aosCollection.Add(
			uint32(i),
			uint8(20+i%40),
			uint32(30000+i%100000),
			i%10 != 0, // 90% active
			fmt.Sprintf("Person%d", i),
			fmt.Sprintf("person%d@example.com", i),
		)
	}
	
	// Setup SoA collection
	soaCollection := NewPersonCollectionSoA(numPersons)
	for i := 0; i < numPersons; i++ {
		soaCollection.Add(
			uint32(i),
			uint8(20+i%40),
			uint32(30000+i%100000),
			i%10 != 0, // 90% active
			fmt.Sprintf("Person%d", i),
			fmt.Sprintf("person%d@example.com", i),
		)
	}
	
	// Benchmark salary calculation
	start := time.Now()
	for i := 0; i < 1000; i++ {
		_ = aosCollection.GetTotalSalary()
	}
	aosTime := time.Since(start)
	
	start = time.Now()
	for i := 0; i < 1000; i++ {
		_ = soaCollection.GetTotalSalary()
	}
	soaTime := time.Since(start)
	
	fmt.Printf("  AoS salary calculation (1000 iterations): %v\n", aosTime)
	fmt.Printf("  SoA salary calculation (1000 iterations): %v\n", soaTime)
	fmt.Printf("  SoA speedup: %.2fx\n", float64(aosTime)/float64(soaTime))
}

func demonstrateMemoryPooling() {
	// Buffer pool demo
	bufferPool := NewBufferPool(1024, 10)
	buf1 := bufferPool.Get()
	buf2 := bufferPool.Get()
	bufferPool.Put(buf1)
	bufferPool.Put(buf2)
	pooled, created := bufferPool.Stats()
	fmt.Printf("  Buffer pool: %d pooled, %d created\n", pooled, created)
	
	// Object pool demo
	pool := NewObjectPool(100, func() interface{} {
		return make([]byte, 1024)
	})
	
	start := time.Now()
	for i := 0; i < 10000; i++ {
		obj := pool.Get().([]byte)
		obj[0] = byte(i)
		pool.Put(obj)
	}
	poolTime := time.Since(start)
	
	start = time.Now()
	for i := 0; i < 10000; i++ {
		obj := make([]byte, 1024)
		obj[0] = byte(i)
		// Object goes out of scope
	}
	directTime := time.Since(start)
	
	fmt.Printf("  Object pool allocation (10000 iterations): %v\n", poolTime)
	fmt.Printf("  Direct allocation (10000 iterations): %v\n", directTime)
	fmt.Printf("  Pool speedup: %.2fx\n", float64(directTime)/float64(poolTime))
	
	// Memory arena demo
	arena := NewMemoryArena(64 * 1024)
	start = time.Now()
	for i := 0; i < 10000; i++ {
		ptr := arena.Allocate(64)
		*(*byte)(ptr) = byte(i)
		if i%1000 == 0 {
			arena.Reset()
		}
	}
	arenaTime := time.Since(start)
	
	fmt.Printf("  Memory arena allocation (10000 iterations): %v\n", arenaTime)
	fmt.Printf("  Arena vs direct speedup: %.2fx\n", float64(directTime)/float64(arenaTime))
}

func demonstrateCompactFormats() {
	// VarInt encoding demo
	encoder := NewVarIntEncoder()
	values := []uint32{1, 127, 128, 16383, 16384, 2097151, 2097152}
	
	fmt.Println("  VarInt Encoding Results:")
	totalVarInt := 0
	totalFixed := 0
	for _, val := range values {
		encoded := encoder.EncodeUint32(val)
		totalVarInt += len(encoded)
		totalFixed += 4 // Fixed 32-bit encoding
		fmt.Printf("    Value %d: %d bytes (vs 4 bytes fixed)\n", val, len(encoded))
	}
	fmt.Printf("  Total: %d bytes VarInt vs %d bytes fixed (%.1f%% savings)\n",
		totalVarInt, totalFixed, 100.0*(1.0-float64(totalVarInt)/float64(totalFixed)))
	
	// Sparse array demo
	csa := NewCompressedSparseArray(10000)
	// Set only 1% of values
	for i := 0; i < 100; i++ {
		csa.Set(i*100, uint32(i+1))
	}
	
	ratio := csa.CompressionRatio()
	fmt.Printf("  Sparse Array Compression Ratio: %.1fx\n", ratio)
	
	// Bit vector demo
	bv := NewBitVector(10000)
	for i := 0; i < 1000; i++ {
		bv.Set(i*10, true)
	}
	
	boolSliceSize := 10000 // 1 byte per bool in Go
	bitVectorSize := (10000 + 7) / 8 // Bits packed into bytes
	fmt.Printf("  Bit Vector: %d bytes vs %d bytes bool slice (%.1fx savings)\n",
		bitVectorSize, boolSliceSize, float64(boolSliceSize)/float64(bitVectorSize))
	
	// String interning demo
	interner := NewStringInterner()
	strings := []string{"hello", "world", "hello", "golang", "world", "hello"}
	
	for _, s := range strings {
		interner.Intern(s)
	}
	
	unique := interner.Size()
	total := len(strings)
	fmt.Printf("  String Interning: %d unique strings from %d total\n", unique, total)
}

func init() {
	// Force garbage collection to get clean memory stats
	runtime.GC()
	runtime.GC()
}