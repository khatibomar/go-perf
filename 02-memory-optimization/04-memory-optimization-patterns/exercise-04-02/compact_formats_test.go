package main

import (
	"fmt"
	"math/rand"
	"testing"
)

// TestDeltaEncoder tests delta encoding for integer sequences
func TestDeltaEncoder(t *testing.T) {
	encoder := NewDeltaEncoder()
	
	// Test ascending sequence
	sequence := []uint32{100, 105, 110, 115, 120, 125}
	encoded := encoder.Encode(sequence)
	decoded := DecodeDelta(encoded)
	
	if len(decoded) != len(sequence) {
		t.Errorf("Expected decoded length %d, got %d", len(sequence), len(decoded))
	}
	
	for i, val := range sequence {
		if decoded[i] != val {
			t.Errorf("Mismatch at index %d: expected %d, got %d", i, val, decoded[i])
		}
	}
	
	// Test compression ratio
	ratio := encoder.CompressionRatio(sequence, encoded)
	if ratio <= 1.0 {
		t.Errorf("Expected compression ratio > 1.0 for regular sequence, got %.2f", ratio)
	}
	
	// Test random sequence (should have lower compression)
	randomSeq := []uint32{100, 50, 200, 75, 300}
	encodedRandom := encoder.Encode(randomSeq)
	randomRatio := encoder.CompressionRatio(randomSeq, encodedRandom)
	
	if randomRatio >= ratio {
		t.Errorf("Random sequence should have lower compression ratio than regular sequence")
	}
}

// TestSharedStringTable tests string deduplication
func TestSharedStringTable(t *testing.T) {
	table := NewSharedStringTable()
	
	// Test basic functionality
	id1 := table.Add("hello")
	id2 := table.Add("world")
	id3 := table.Add("hello") // Duplicate
	
	if id1 != id3 {
		t.Errorf("Duplicate strings should have same ID: %d != %d", id1, id3)
	}
	
	if id1 == id2 {
		t.Errorf("Different strings should have different IDs: %d == %d", id1, id2)
	}
	
	// Test retrieval
	str1, ok1 := table.Get(id1)
	if !ok1 || str1 != "hello" {
		t.Errorf("Expected 'hello', got '%s', ok=%v", str1, ok1)
	}
	
	str2, ok2 := table.Get(id2)
	if !ok2 || str2 != "world" {
		t.Errorf("Expected 'world', got '%s', ok=%v", str2, ok2)
	}
	
	// Test non-existent ID
	_, ok3 := table.Get(999)
	if ok3 {
		t.Error("Should not find non-existent ID")
	}
	
	// Test stats
	unique, total, savings := table.Stats()
	if unique != 2 {
		t.Errorf("Expected 2 unique strings, got %d", unique)
	}
	if total != 3 {
		t.Errorf("Expected 3 total additions, got %d", total)
	}
	if savings <= 0 {
		t.Errorf("Expected positive memory savings, got %d", savings)
	}
}

// TestCompactRecord tests field packing
func TestCompactRecord(t *testing.T) {
	// Test basic packing and unpacking
	record := NewCompactRecord()
	record.SetField(0, 255)    // 8-bit field
	record.SetField(1, 65535)  // 16-bit field
	record.SetField(2, 1048575) // 20-bit field
	
	if record.GetField(0) != 255 {
		t.Errorf("Expected field 0 to be 255, got %d", record.GetField(0))
	}
	
	if record.GetField(1) != 65535 {
		t.Errorf("Expected field 1 to be 65535, got %d", record.GetField(1))
	}
	
	if record.GetField(2) != 1048575 {
		t.Errorf("Expected field 2 to be 1048575, got %d", record.GetField(2))
	}
	
	// Test overflow handling
	record.SetField(0, 256) // Should be truncated to 8 bits
	if record.GetField(0) != 0 {
		t.Errorf("Expected field 0 to be 0 (truncated), got %d", record.GetField(0))
	}
	
	// Test serialization
	data := record.Serialize()
	newRecord := DeserializeCompactRecord(data)
	
	for i := 0; i < 3; i++ {
		if newRecord.GetField(i) != record.GetField(i) {
			t.Errorf("Serialization mismatch for field %d: expected %d, got %d",
				i, record.GetField(i), newRecord.GetField(i))
		}
	}
}

// TestVarIntEncodingEdgeCases tests edge cases for VarInt encoding
func TestVarIntEncodingEdgeCases(t *testing.T) {
	encoder := NewVarIntEncoder()
	
	testCases := []struct {
		value    uint32
		expected int // Expected number of bytes
	}{
		{0, 1},
		{127, 1},
		{128, 2},
		{16383, 2},
		{16384, 3},
		{2097151, 3},
		{2097152, 4},
		{268435455, 4},
		{268435456, 5},
		{4294967295, 5}, // Max uint32
	}
	
	for _, tc := range testCases {
		encoded := encoder.EncodeUint32(tc.value)
		if len(encoded) != tc.expected {
			t.Errorf("Value %d: expected %d bytes, got %d", tc.value, tc.expected, len(encoded))
		}
		
		decoded, bytesRead := DecodeUint32(encoded)
		if decoded != tc.value {
			t.Errorf("Value %d: decode mismatch, got %d", tc.value, decoded)
		}
		if bytesRead != len(encoded) {
			t.Errorf("Value %d: bytes read mismatch, expected %d, got %d", tc.value, len(encoded), bytesRead)
		}
	}
}

// TestCompressedSparseArrayEdgeCases tests edge cases for sparse array
func TestCompressedSparseArrayEdgeCases(t *testing.T) {
	csa := NewCompressedSparseArray(1000)
	
	// Test setting the same value multiple times
	csa.Set(10, 100)
	csa.Set(10, 100) // Should not create duplicate runs
	csa.Set(10, 200) // Should update existing
	
	if csa.Get(10) != 200 {
		t.Errorf("Expected value 200 at index 10, got %d", csa.Get(10))
	}
	
	// Test adjacent values with same data
	csa.Set(20, 300)
	csa.Set(21, 300)
	csa.Set(22, 300)
	csa.Set(23, 300)
	
	// All should return the same value
	for i := 20; i <= 23; i++ {
		if csa.Get(i) != 300 {
			t.Errorf("Expected value 300 at index %d, got %d", i, csa.Get(i))
		}
	}
	
	// Test setting value to zero (should remove from sparse representation)
	csa.Set(20, 0)
	if csa.Get(20) != 0 {
		t.Errorf("Expected value 0 at index 20, got %d", csa.Get(20))
	}
	
	// Test out of bounds
	csa.Set(1000, 400) // Should be ignored
	if csa.Get(1000) != 0 {
		t.Errorf("Out of bounds access should return 0, got %d", csa.Get(1000))
	}
}

// TestBitVectorOperations tests bit vector operations
func TestBitVectorOperations(t *testing.T) {
	bv1 := NewBitVector(64)
	bv2 := NewBitVector(64)
	
	// Set some bits in both vectors
	bv1.Set(0, true)
	bv1.Set(5, true)
	bv1.Set(10, true)
	
	bv2.Set(0, true)
	bv2.Set(3, true)
	bv2.Set(10, true)
	
	// Test AND operation
	resultAnd := bv1.And(bv2)
	if !resultAnd.Get(0) || !resultAnd.Get(10) {
		t.Error("AND operation failed for bits 0 and 10")
	}
	if resultAnd.Get(3) || resultAnd.Get(5) {
		t.Error("AND operation should not set bits 3 and 5")
	}
	
	// Test OR operation
	resultOr := bv1.Or(bv2)
	if !resultOr.Get(0) || !resultOr.Get(3) || !resultOr.Get(5) || !resultOr.Get(10) {
		t.Error("OR operation failed")
	}
	
	// Test XOR operation
	resultXor := bv1.Xor(bv2)
	if resultXor.Get(0) || resultXor.Get(10) {
		t.Error("XOR operation should not set bits 0 and 10")
	}
	if !resultXor.Get(3) || !resultXor.Get(5) {
		t.Error("XOR operation should set bits 3 and 5")
	}
	
	// Test population count
	if bv1.PopCount() != 3 {
		t.Errorf("Expected population count 3 for bv1, got %d", bv1.PopCount())
	}
	if bv2.PopCount() != 3 {
		t.Errorf("Expected population count 3 for bv2, got %d", bv2.PopCount())
	}
	if resultXor.PopCount() != 2 {
		t.Errorf("Expected population count 2 for XOR result, got %d", resultXor.PopCount())
	}
}

// Benchmark VarInt vs Fixed encoding for different value ranges
func BenchmarkVarIntVsFixed(b *testing.B) {
	encoder := NewVarIntEncoder()
	
	valueRanges := []struct {
		name   string
		values []uint32
	}{
		{"Small", generateValues(1000, 0, 127)},
		{"Medium", generateValues(1000, 128, 16383)},
		{"Large", generateValues(1000, 16384, 2097151)},
		{"Mixed", generateMixedValues(1000)},
	}
	
	for _, vr := range valueRanges {
		b.Run(fmt.Sprintf("VarInt_%s", vr.name), func(b *testing.B) {
			b.ResetTimer()
			b.ReportAllocs()
			for i := 0; i < b.N; i++ {
				value := vr.values[i%len(vr.values)]
				encoded := encoder.EncodeUint32(value)
				decoded, _ := DecodeUint32(encoded)
				SinkUint32 = decoded
			}
		})
		
		b.Run(fmt.Sprintf("Fixed_%s", vr.name), func(b *testing.B) {
			b.ResetTimer()
			b.ReportAllocs()
			for i := 0; i < b.N; i++ {
				value := vr.values[i%len(vr.values)]
			// Simulate fixed 4-byte encoding
			SinkUint32 = value
			}
		})
	}
}

// Benchmark delta encoding vs raw encoding
func BenchmarkDeltaEncoding(b *testing.B) {
	encoder := NewDeltaEncoder()
	
	sequenceTypes := []struct {
		name     string
		sequence []uint32
	}{
		{"Ascending", generateAscendingSequence(1000, 100, 5)},
		{"Descending", generateDescendingSequence(1000, 10000, 3)},
		{"Random", generateRandomSequence(1000, 0, 100000)},
		{"Constant", generateConstantSequence(1000, 42)},
	}
	
	for _, st := range sequenceTypes {
		b.Run(fmt.Sprintf("Delta_%s", st.name), func(b *testing.B) {
			b.ResetTimer()
			b.ReportAllocs()
			for i := 0; i < b.N; i++ {
				encoded := encoder.Encode(st.sequence)
				decoded := DecodeDelta(encoded)
				SinkUint32 = uint32(len(decoded))
			}
		})
		
		b.Run(fmt.Sprintf("Raw_%s", st.name), func(b *testing.B) {
			b.ResetTimer()
			b.ReportAllocs()
			for i := 0; i < b.N; i++ {
				// Simulate raw encoding (just copying)
				raw := make([]uint32, len(st.sequence))
				copy(raw, st.sequence)
				SinkUint32 = uint32(len(raw))
			}
		})
	}
}

// Benchmark string table vs direct string storage
func BenchmarkStringTable(b *testing.B) {
	table := NewSharedStringTable()
	strings := generateTestStrings(100)
	
	b.Run("StringTable", func(b *testing.B) {
		b.ResetTimer()
		b.ReportAllocs()
		for i := 0; i < b.N; i++ {
			str := strings[i%len(strings)]
			id := table.Add(str)
			retrieved, _ := table.Get(id)
			SinkString = retrieved
		}
	})
	
	b.Run("DirectString", func(b *testing.B) {
		stringMap := make(map[string]int)
		nextID := 0
		
		b.ResetTimer()
		b.ReportAllocs()
		for i := 0; i < b.N; i++ {
			str := strings[i%len(strings)]
			id, exists := stringMap[str]
			if !exists {
				id = nextID
				stringMap[str] = id
				nextID++
			}
			SinkString = str
		}
	})
}

// Benchmark compact record vs struct
func BenchmarkCompactRecord(b *testing.B) {
	type RegularRecord struct {
		Field0 uint32
		Field1 uint32
		Field2 uint32
	}
	
	b.Run("CompactRecord", func(b *testing.B) {
		b.ResetTimer()
		b.ReportAllocs()
		for i := 0; i < b.N; i++ {
			record := NewCompactRecord()
			record.SetField(0, uint32(i%256))
			record.SetField(1, uint32(i%65536))
			record.SetField(2, uint32(i%1048576))
			
			val := record.GetField(0) + record.GetField(1) + record.GetField(2)
			SinkUint32 = val
		}
	})
	
	b.Run("RegularStruct", func(b *testing.B) {
		b.ResetTimer()
		b.ReportAllocs()
		for i := 0; i < b.N; i++ {
			record := RegularRecord{
				Field0: uint32(i % 256),
				Field1: uint32(i % 65536),
				Field2: uint32(i % 1048576),
			}
			
			val := record.Field0 + record.Field1 + record.Field2
			SinkUint32 = val
		}
	})
}

// Benchmark sparse array vs dense array
func BenchmarkSparseArray(b *testing.B) {
	const arraySize = 100000
	const sparsity = 100 // Only 1% of elements are non-zero
	
	csa := NewCompressedSparseArray(arraySize)
	denseArray := make([]uint32, arraySize)
	
	// Set sparse values
	for i := 0; i < arraySize/sparsity; i++ {
		idx := i * sparsity
		value := uint32(i + 1)
		csa.Set(idx, value)
		denseArray[idx] = value
	}
	
	b.Run("SparseArray", func(b *testing.B) {
		b.ResetTimer()
		b.ReportAllocs()
		for i := 0; i < b.N; i++ {
			idx := i % arraySize
			val := csa.Get(idx)
			SinkUint32 = val
		}
	})
	
	b.Run("DenseArray", func(b *testing.B) {
		b.ResetTimer()
		b.ReportAllocs()
		for i := 0; i < b.N; i++ {
			idx := i % arraySize
			val := denseArray[idx]
			SinkUint32 = val
		}
	})
}

// Helper functions for generating test data
func generateValues(count int, min, max uint32) []uint32 {
	values := make([]uint32, count)
	for i := range values {
		values[i] = min + uint32(rand.Intn(int(max-min+1)))
	}
	return values
}

func generateMixedValues(count int) []uint32 {
	values := make([]uint32, count)
	for i := range values {
		switch rand.Intn(4) {
		case 0:
			values[i] = uint32(rand.Intn(128)) // Small values
		case 1:
			values[i] = 128 + uint32(rand.Intn(16256)) // Medium values
		case 2:
			values[i] = 16384 + uint32(rand.Intn(2080768)) // Large values
		default:
			values[i] = uint32(rand.Uint32()) // Very large values
		}
	}
	return values
}

func generateAscendingSequence(count int, start uint32, step uint32) []uint32 {
	sequence := make([]uint32, count)
	for i := range sequence {
		sequence[i] = start + uint32(i)*step
	}
	return sequence
}

func generateDescendingSequence(count int, start uint32, step uint32) []uint32 {
	sequence := make([]uint32, count)
	for i := range sequence {
		sequence[i] = start - uint32(i)*step
	}
	return sequence
}

func generateRandomSequence(count int, min, max uint32) []uint32 {
	sequence := make([]uint32, count)
	for i := range sequence {
		sequence[i] = min + uint32(rand.Intn(int(max-min+1)))
	}
	return sequence
}

func generateConstantSequence(count int, value uint32) []uint32 {
	sequence := make([]uint32, count)
	for i := range sequence {
		sequence[i] = value
	}
	return sequence
}

func generateTestStrings(count int) []string {
	strings := make([]string, count)
	prefixes := []string{"user", "item", "order", "product", "category"}
	suffixes := []string{"data", "info", "details", "summary", "report"}
	
	for i := range strings {
		prefix := prefixes[rand.Intn(len(prefixes))]
		suffix := suffixes[rand.Intn(len(suffixes))]
		strings[i] = fmt.Sprintf("%s_%d_%s", prefix, rand.Intn(1000), suffix)
	}
	return strings
}



// Test compression analysis function
func TestCompressionAnalysisFunction(t *testing.T) {
	// This test just ensures the analysis function runs without errors
	AnalyzeCompression()
	
	// Test specific compression scenarios
	encoder := NewVarIntEncoder()
	
	// Test compression ratios for different value ranges
	smallValues := generateValues(1000, 0, 127)
	largeValues := generateValues(1000, 1000000, 2000000)
	
	smallTotal := 0
	for _, val := range smallValues {
		encoded := encoder.EncodeUint32(val)
		smallTotal += len(encoded)
	}
	
	largeTotal := 0
	for _, val := range largeValues {
		encoded := encoder.EncodeUint32(val)
		largeTotal += len(encoded)
	}
	
	// Small values should compress better
	smallAvg := float64(smallTotal) / float64(len(smallValues))
	largeAvg := float64(largeTotal) / float64(len(largeValues))
	
	if smallAvg >= largeAvg {
		t.Errorf("Small values should compress better: %.2f >= %.2f", smallAvg, largeAvg)
	}
	
	t.Logf("Average bytes per small value: %.2f", smallAvg)
	t.Logf("Average bytes per large value: %.2f", largeAvg)
}

// Test edge cases for all compact formats
func TestCompactFormatsEdgeCases(t *testing.T) {
	// Test empty sequences
	encoder := NewDeltaEncoder()
	emptyEncoded := encoder.Encode([]uint32{})
	emptyDecoded := DecodeDelta(emptyEncoded)
	if len(emptyDecoded) != 0 {
		t.Errorf("Empty sequence should decode to empty, got length %d", len(emptyDecoded))
	}
	
	// Test single element sequences
	singleEncoded := encoder.Encode([]uint32{42})
	singleDecoded := DecodeDelta(singleEncoded)
	if len(singleDecoded) != 1 || singleDecoded[0] != 42 {
		t.Errorf("Single element sequence failed: expected [42], got %v", singleDecoded)
	}
	
	// Test bit vector with size 0
	bv := NewBitVector(0)
	if bv.PopCount() != 0 {
		t.Errorf("Empty bit vector should have population count 0, got %d", bv.PopCount())
	}
	
	// Test sparse array with size 0
	csa := NewCompressedSparseArray(0)
	if csa.Get(0) != 0 {
		t.Errorf("Empty sparse array should return 0 for any index, got %d", csa.Get(0))
	}
}

// Benchmark memory usage of different formats
func BenchmarkMemoryUsage(b *testing.B) {
	const dataSize = 10000
	
	b.Run("BitVector", func(b *testing.B) {
		b.ReportAllocs()
		for i := 0; i < b.N; i++ {
			bv := NewBitVector(dataSize)
			// Set some random bits
			for j := 0; j < dataSize/10; j++ {
				bv.Set(rand.Intn(dataSize), true)
			}
			SinkInt = bv.PopCount()
		}
	})
	
	b.Run("BoolSlice", func(b *testing.B) {
		b.ReportAllocs()
		for i := 0; i < b.N; i++ {
			bools := make([]bool, dataSize)
			// Set some random values
			for j := 0; j < dataSize/10; j++ {
				bools[rand.Intn(dataSize)] = true
			}
			count := 0
			for _, b := range bools {
				if b {
					count++
				}
			}
			SinkInt = count
		}
	})
}

// Test concurrent access to shared string table
func TestSharedStringTableConcurrent(t *testing.T) {
	table := NewSharedStringTable()
	strings := generateTestStrings(100)
	
	const numGoroutines = 10
	const operationsPerGoroutine = 1000
	
	done := make(chan bool, numGoroutines)
	
	for i := 0; i < numGoroutines; i++ {
		go func(id int) {
			for j := 0; j < operationsPerGoroutine; j++ {
				str := strings[rand.Intn(len(strings))]
				id := table.Add(str)
				retrieved, ok := table.Get(id)
				if !ok || retrieved != str {
					t.Errorf("Concurrent access failed: expected %s, got %s, ok=%v", str, retrieved, ok)
					return
				}
			}
			done <- true
		}(i)
	}
	
	// Wait for all goroutines to complete
	for i := 0; i < numGoroutines; i++ {
		<-done
	}
	
	// Verify final state
	unique, total, _ := table.Stats()
	if unique == 0 || total == 0 {
		t.Errorf("String table should have entries after concurrent access: unique=%d, total=%d", unique, total)
	}
}