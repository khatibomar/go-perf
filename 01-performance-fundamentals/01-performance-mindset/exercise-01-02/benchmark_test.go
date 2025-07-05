package main

import (
	"fmt"
	"runtime"
	"testing"
)

var testItems []string
var lookupMap map[string]bool

// validateBenchmarkEnvironment ensures the benchmark environment is suitable
func validateBenchmarkEnvironment(b *testing.B) {
	if testing.Short() {
		b.Skip("Skipping benchmark in short mode")
	}
	
	var m runtime.MemStats
	runtime.ReadMemStats(&m)
	if m.Sys < 100*1024*1024 { // Less than 100MB
		b.Skip("Insufficient memory for reliable benchmarking")
	}
}

func init() {
	testItems = make([]string, 1000)
	for i := range testItems {
		testItems[i] = fmt.Sprintf("  Item%d  ", i)
	}
	lookupMap = createLookupMap(testItems)
}

// String building benchmarks
func BenchmarkBuildStringBad(b *testing.B) {
	validateBenchmarkEnvironment(b)
	smallItems := testItems[:100] // Use smaller set for string concat
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		buildStringBad(smallItems)
	}
}

func BenchmarkBuildStringGood(b *testing.B) {
	validateBenchmarkEnvironment(b)
	smallItems := testItems[:100]
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		buildStringGood(smallItems)
	}
}

// Processing benchmarks
func BenchmarkProcessItemsBad(b *testing.B) {
	validateBenchmarkEnvironment(b)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		processItemsBad(testItems)
	}
}

func BenchmarkProcessItemsGood(b *testing.B) {
	validateBenchmarkEnvironment(b)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		processItemsGood(testItems)
	}
}

// Lookup benchmarks
func BenchmarkFindItemBad(b *testing.B) {
	validateBenchmarkEnvironment(b)
	target := "  Item500  "
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		findItemBad(testItems, target)
	}
}

func BenchmarkFindItemGood(b *testing.B) {
	validateBenchmarkEnvironment(b)
	target := "  Item500  "
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		findItemGood(lookupMap, target)
	}
}

// Multiple lookups (more realistic scenario)
func BenchmarkMultipleLookupsBad(b *testing.B) {
	validateBenchmarkEnvironment(b)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		for j := 0; j < 10; j++ {
			target := fmt.Sprintf("  Item%d  ", j*100)
			findItemBad(testItems, target)
		}
	}
}

func BenchmarkMultipleLookupsGood(b *testing.B) {
	validateBenchmarkEnvironment(b)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		for j := 0; j < 10; j++ {
			target := fmt.Sprintf("  Item%d  ", j*100)
			findItemGood(lookupMap, target)
		}
	}
}

// Data processing benchmarks
func BenchmarkProcessDataBad(b *testing.B) {
	validateBenchmarkEnvironment(b)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		processDataBad(testItems)
	}
}

func BenchmarkProcessDataGood(b *testing.B) {
	validateBenchmarkEnvironment(b)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		processDataGood(testItems)
	}
}

func BenchmarkProcessDataBetter(b *testing.B) {
	validateBenchmarkEnvironment(b)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		processDataBetter(testItems)
	}
}

// Memory allocation benchmarks
func BenchmarkMemoryAllocationBad(b *testing.B) {
	validateBenchmarkEnvironment(b)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		// Simulate bad allocation pattern
		var result []string
		for j := 0; j < 100; j++ {
			// Growing slice without pre-allocation
			result = append(result, fmt.Sprintf("item%d", j))
		}
		_ = result
	}
}

func BenchmarkMemoryAllocationGood(b *testing.B) {
	validateBenchmarkEnvironment(b)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		// Pre-allocate with known capacity
		result := make([]string, 0, 100)
		for j := 0; j < 100; j++ {
			result = append(result, fmt.Sprintf("item%d", j))
		}
		_ = result
	}
}

// Test correctness of optimizations
func TestOptimizationsCorrectness(t *testing.T) {
	testData := []string{"  Hello  ", "  World  ", "  Go  "}
	
	// Test string building
	badResult := buildStringBad(testData)
	goodResult := buildStringGood(testData)
	if badResult != goodResult {
		t.Errorf("String building results differ: bad=%s, good=%s", badResult, goodResult)
	}
	
	// Test processing
	badProcessed := processItemsBad(testData)
	goodProcessed := processItemsGood(testData)
	if len(badProcessed) != len(goodProcessed) {
		t.Errorf("Processing results have different lengths")
	}
	for i := range badProcessed {
		if badProcessed[i] != goodProcessed[i] {
			t.Errorf("Processing results differ at index %d: bad=%s, good=%s", i, badProcessed[i], goodProcessed[i])
		}
	}
	
	// Test lookup
	lookup := createLookupMap(testData)
	for _, item := range testData {
		badFound := findItemBad(testData, item)
		goodFound := findItemGood(lookup, item)
		if badFound != goodFound {
			t.Errorf("Lookup results differ for %s: bad=%v, good=%v", item, badFound, goodFound)
		}
	}
	
	// Test data processing
	badData := processDataBad(testData)
	goodData := processDataGood(testData)
	betterData := processDataBetter(testData)
	
	for i := range testData {
		if badData[i] != goodData[i] || goodData[i] != betterData[i] {
			t.Errorf("Data processing results differ at index %d", i)
		}
	}
}

// Benchmark to show the impact of different slice growth strategies
func BenchmarkSliceGrowthAppend(b *testing.B) {
	validateBenchmarkEnvironment(b)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		var slice []int
		for j := 0; j < 1000; j++ {
			slice = append(slice, j)
		}
	}
}

func BenchmarkSliceGrowthPrealloc(b *testing.B) {
	validateBenchmarkEnvironment(b)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		slice := make([]int, 0, 1000)
		for j := 0; j < 1000; j++ {
			slice = append(slice, j)
		}
	}
}

func BenchmarkSliceGrowthDirect(b *testing.B) {
	validateBenchmarkEnvironment(b)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		slice := make([]int, 1000)
		for j := 0; j < 1000; j++ {
			slice[j] = j
		}
	}
}