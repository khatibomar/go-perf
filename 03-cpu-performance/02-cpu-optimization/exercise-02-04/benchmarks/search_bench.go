package benchmarks

import (
	"fmt"
	"math/rand"
	"runtime"
	"sort"
	"strconv"
	"testing"
	"time"

	"exercise-02-04/search"
)

// BenchmarkConfig holds configuration for benchmarks
type BenchmarkConfig struct {
	DataSizes    []int
	Iterations   int
	WarmupRuns   int
	RandomSeed   int64
	SearchRatio  float64 // Ratio of successful searches
}

// DefaultBenchmarkConfig returns a default benchmark configuration
func DefaultBenchmarkConfig() *BenchmarkConfig {
	return &BenchmarkConfig{
		DataSizes:   []int{1000, 10000, 100000, 1000000},
		Iterations:  1000,
		WarmupRuns:  100,
		RandomSeed:  42,
		SearchRatio: 0.8,
	}
}

// SearchResult holds the result of a search operation
type SearchResult struct {
	Found    bool
	Index    int
	Duration time.Duration
}

// BenchmarkResult holds benchmark statistics
type BenchmarkResult struct {
	Algorithm     string
	DataSize      int
	TotalTime     time.Duration
	AverageTime   time.Duration
	MinTime       time.Duration
	MaxTime       time.Duration
	SuccessRate   float64
	MemoryUsage   int64
	Allocations   int64
}

// generateTestData creates test data for benchmarks
func generateTestData(size int, sorted bool) []int {
	data := make([]int, size)
	for i := 0; i < size; i++ {
		data[i] = rand.Intn(size * 10)
	}
	if sorted {
		sort.Ints(data)
	}
	return data
}

// generateSearchTargets creates search targets
func generateSearchTargets(data []int, count int, successRatio float64) []int {
	targets := make([]int, count)
	successCount := int(float64(count) * successRatio)
	
	// Add existing values for successful searches
	for i := 0; i < successCount; i++ {
		targets[i] = data[rand.Intn(len(data))]
	}
	
	// Add non-existing values for unsuccessful searches
	for i := successCount; i < count; i++ {
		targets[i] = len(data)*10 + rand.Intn(1000)
	}
	
	// Shuffle targets
	for i := len(targets) - 1; i > 0; i-- {
		j := rand.Intn(i + 1)
		targets[i], targets[j] = targets[j], targets[i]
	}
	
	return targets
}

// BenchmarkLinearSearch benchmarks linear search algorithms
func BenchmarkLinearSearch(b *testing.B) {
	config := DefaultBenchmarkConfig()
	
	for _, size := range config.DataSizes {
		data := generateTestData(size, false)
		targets := generateSearchTargets(data, config.Iterations, config.SearchRatio)
		
		b.Run(fmt.Sprintf("Basic_%d", size), func(b *testing.B) {
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				target := targets[i%len(targets)]
				search.LinearSearch(data, target)
			}
		})
		
		b.Run(fmt.Sprintf("EarlyTermination_%d", size), func(b *testing.B) {
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				target := targets[i%len(targets)]
				search.LinearSearchEarlyTermination(data, target)
			}
		})
		
		b.Run(fmt.Sprintf("Unrolled_%d", size), func(b *testing.B) {
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				target := targets[i%len(targets)]
				search.LinearSearchUnrolled(data, target)
			}
		})
		
		b.Run(fmt.Sprintf("Branchless_%d", size), func(b *testing.B) {
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				target := targets[i%len(targets)]
				search.LinearSearchBranchless(data, target)
			}
		})
	}
}

// BenchmarkBinarySearch benchmarks binary search algorithms
func BenchmarkBinarySearch(b *testing.B) {
	config := DefaultBenchmarkConfig()
	
	for _, size := range config.DataSizes {
		data := generateTestData(size, true) // Binary search requires sorted data
		targets := generateSearchTargets(data, config.Iterations, config.SearchRatio)
		
		b.Run(fmt.Sprintf("Recursive_%d", size), func(b *testing.B) {
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				target := targets[i%len(targets)]
				search.BinarySearch(data, target)
			}
		})
		
		b.Run(fmt.Sprintf("Iterative_%d", size), func(b *testing.B) {
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				target := targets[i%len(targets)]
				search.BinarySearchIterative(data, target)
			}
		})
		
		b.Run(fmt.Sprintf("Branchless_%d", size), func(b *testing.B) {
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				target := targets[i%len(targets)]
				search.BinarySearchBranchless(data, target)
			}
		})
		
		b.Run(fmt.Sprintf("CacheFriendly_%d", size), func(b *testing.B) {
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				target := targets[i%len(targets)]
				search.BinarySearchCacheFriendly(data, target)
			}
		})
		
		b.Run(fmt.Sprintf("Interpolation_%d", size), func(b *testing.B) {
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				target := targets[i%len(targets)]
				search.InterpolationSearch(data, target)
			}
		})
		
		b.Run(fmt.Sprintf("Exponential_%d", size), func(b *testing.B) {
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				target := targets[i%len(targets)]
				search.ExponentialSearch(data, target)
			}
		})
		
		b.Run(fmt.Sprintf("Fibonacci_%d", size), func(b *testing.B) {
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				target := targets[i%len(targets)]
				search.FibonacciSearch(data, target)
			}
		})
		
		b.Run(fmt.Sprintf("Ternary_%d", size), func(b *testing.B) {
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				target := targets[i%len(targets)]
				search.TernarySearch(data, target)
			}
		})
	}
}

// BenchmarkHashSearch benchmarks hash-based search algorithms
func BenchmarkHashSearch(b *testing.B) {
	config := DefaultBenchmarkConfig()
	
	for _, size := range config.DataSizes {
		data := generateTestData(size, false)
		targets := generateSearchTargets(data, config.Iterations, config.SearchRatio)
		
		// Setup hash tables
		hashTable := search.NewHashTable()
		optHashTable := search.NewOptimizedHashTable()
		robinHoodTable := search.NewRobinHoodHashTable()
		cuckooTable := search.NewCuckooHashTable()
		
		for _, val := range data {
			key := strconv.Itoa(val)
			hashTable.Insert(key, val)
			optHashTable.Insert(key, val)
			robinHoodTable.Insert(key, val)
			cuckooTable.Insert(key, val)
		}
		
		b.Run(fmt.Sprintf("Basic_%d", size), func(b *testing.B) {
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				target := targets[i%len(targets)]
				hashTable.Search(strconv.Itoa(target))
			}
		})
		
		b.Run(fmt.Sprintf("Optimized_%d", size), func(b *testing.B) {
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				target := targets[i%len(targets)]
				optHashTable.Search(strconv.Itoa(target))
			}
		})
		
		b.Run(fmt.Sprintf("RobinHood_%d", size), func(b *testing.B) {
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				target := targets[i%len(targets)]
				robinHoodTable.Search(strconv.Itoa(target))
			}
		})
		
		b.Run(fmt.Sprintf("Cuckoo_%d", size), func(b *testing.B) {
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				target := targets[i%len(targets)]
				cuckooTable.Search(strconv.Itoa(target))
			}
		})
	}
}

// BenchmarkTreeSearch benchmarks tree-based search algorithms
func BenchmarkTreeSearch(b *testing.B) {
	config := DefaultBenchmarkConfig()
	
	for _, size := range config.DataSizes {
		data := generateTestData(size, false)
		targets := generateSearchTargets(data, config.Iterations, config.SearchRatio)
		
		// Setup trees
		bst := search.NewBST()
		btree := search.NewBTree(100) // Degree 100
		
		for _, val := range data {
			bst.Insert(val)
			btree.Insert(val)
		}
		
		b.Run(fmt.Sprintf("BST_%d", size), func(b *testing.B) {
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				target := targets[i%len(targets)]
				bst.Search(target)
			}
		})
		
		b.Run(fmt.Sprintf("BTree_%d", size), func(b *testing.B) {
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				target := targets[i%len(targets)]
				btree.Search(target)
			}
		})
	}
}

// BenchmarkStringSearch benchmarks string search algorithms
func BenchmarkStringSearch(b *testing.B) {
	config := DefaultBenchmarkConfig()
	words := generateStringData(10000)
	
	for _, size := range []int{1000, 5000, 10000} {
		if size > len(words) {
			continue
		}
		
		data := words[:size]
		targets := make([]string, config.Iterations)
		for i := 0; i < config.Iterations; i++ {
			if rand.Float64() < config.SearchRatio {
				targets[i] = data[rand.Intn(len(data))]
			} else {
				targets[i] = "nonexistent_" + strconv.Itoa(rand.Intn(1000))
			}
		}
		
		// Setup string search structures
		trie := search.NewTrie()
		compTrie := search.NewCompressedTrie()
		
		for i, word := range data {
			trie.Insert(word, i)
			compTrie.Insert(word, i)
		}
		
		b.Run(fmt.Sprintf("Linear_%d", size), func(b *testing.B) {
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				target := targets[i%len(targets)]
				search.LinearSearchString(data, target)
			}
		})
		
		b.Run(fmt.Sprintf("Trie_%d", size), func(b *testing.B) {
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				target := targets[i%len(targets)]
				trie.Search(target)
			}
		})
		
		b.Run(fmt.Sprintf("CompressedTrie_%d", size), func(b *testing.B) {
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				target := targets[i%len(targets)]
				compTrie.Search(target)
			}
		})
	}
}

// generateStringData creates test string data
func generateStringData(count int) []string {
	words := make([]string, count)
	for i := 0; i < count; i++ {
		length := 5 + rand.Intn(10)
		word := make([]byte, length)
		for j := 0; j < length; j++ {
			word[j] = byte('a' + rand.Intn(26))
		}
		words[i] = string(word)
	}
	return words
}

// BenchmarkMemoryAccess benchmarks memory access patterns
func BenchmarkMemoryAccess(b *testing.B) {
	sizes := []int{1000, 10000, 100000}
	
	for _, size := range sizes {
		data := generateTestData(size, true)
		targets := generateSearchTargets(data, 1000, 0.8)
		
		b.Run(fmt.Sprintf("Sequential_%d", size), func(b *testing.B) {
			runtime.GC()
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				// Sequential access pattern
				for j := 0; j < len(data); j += 64 { // Cache line size
					_ = data[j]
				}
			}
		})
		
		b.Run(fmt.Sprintf("Random_%d", size), func(b *testing.B) {
			runtime.GC()
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				// Random access pattern
				for j := 0; j < 1000; j++ {
					idx := rand.Intn(len(data))
					_ = data[idx]
				}
			}
		})
		
		b.Run(fmt.Sprintf("BinarySearch_%d", size), func(b *testing.B) {
			runtime.GC()
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				target := targets[i%len(targets)]
				search.BinarySearchIterative(data, target)
			}
		})
	}
}

// BenchmarkCacheEfficiency tests cache efficiency of different algorithms
func BenchmarkCacheEfficiency(b *testing.B) {
	sizes := []int{1000, 10000, 100000, 1000000}
	
	for _, size := range sizes {
		data := generateTestData(size, true)
		targets := generateSearchTargets(data, 1000, 0.8)
		
		b.Run(fmt.Sprintf("LinearCache_%d", size), func(b *testing.B) {
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				target := targets[i%len(targets)]
				search.LinearSearchPrefetch(data, target)
			}
		})
		
		b.Run(fmt.Sprintf("BinaryCache_%d", size), func(b *testing.B) {
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				target := targets[i%len(targets)]
				search.BinarySearchCacheFriendly(data, target)
			}
		})
	}
}

// RunCustomBenchmark runs a custom benchmark with detailed metrics
func RunCustomBenchmark(algorithm string, data []int, targets []int, iterations int) *BenchmarkResult {
	result := &BenchmarkResult{
		Algorithm: algorithm,
		DataSize:  len(data),
		MinTime:   time.Hour, // Initialize with large value
	}
	
	var totalTime time.Duration
	successCount := 0
	
	// Warmup
	for i := 0; i < 10; i++ {
		target := targets[i%len(targets)]
		switch algorithm {
		case "linear":
			search.LinearSearch(data, target)
		case "binary":
			search.BinarySearchIterative(data, target)
		}
	}
	
	// Actual benchmark
	for i := 0; i < iterations; i++ {
		target := targets[i%len(targets)]
		
		start := time.Now()
		var found bool
		
		switch algorithm {
		case "linear":
			idx := search.LinearSearch(data, target)
			found = idx != -1
		case "binary":
			idx := search.BinarySearchIterative(data, target)
			found = idx != -1
		}
		
		duration := time.Since(start)
		totalTime += duration
		
		if found {
			successCount++
		}
		
		if duration < result.MinTime {
			result.MinTime = duration
		}
		if duration > result.MaxTime {
			result.MaxTime = duration
		}
	}
	
	result.TotalTime = totalTime
	result.AverageTime = totalTime / time.Duration(iterations)
	result.SuccessRate = float64(successCount) / float64(iterations)
	
	return result
}