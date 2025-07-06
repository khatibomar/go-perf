package benchmarks

import (
	"fmt"
	"math/rand"
	"testing"

	"exercise-02-01/algorithms"
)

// BenchmarkLinearSearch benchmarks linear search algorithm
func BenchmarkLinearSearch(b *testing.B) {
	for _, size := range benchmarkSizes {
		b.Run(fmt.Sprintf("Size%d", size), func(b *testing.B) {
			data := generateBenchmarkData(size)
			target := data[size/2] // Middle element
			b.ResetTimer()
			b.ReportAllocs()
			
			for i := 0; i < b.N; i++ {
				algorithms.LinearSearch(data, target)
			}
		})
	}
}

// BenchmarkBinarySearch benchmarks binary search algorithm
func BenchmarkBinarySearch(b *testing.B) {
	for _, size := range benchmarkSizes {
		b.Run(fmt.Sprintf("Size%d", size), func(b *testing.B) {
			data := generateSortedData(size)
			target := data[size/2] // Middle element
			b.ResetTimer()
			b.ReportAllocs()
			
			for i := 0; i < b.N; i++ {
				algorithms.BinarySearch(data, target)
			}
		})
	}
}

// BenchmarkInterpolationSearch benchmarks interpolation search algorithm
func BenchmarkInterpolationSearch(b *testing.B) {
	for _, size := range benchmarkSizes {
		b.Run(fmt.Sprintf("Size%d", size), func(b *testing.B) {
			data := generateSortedData(size)
			target := data[size/2] // Middle element
			b.ResetTimer()
			b.ReportAllocs()
			
			for i := 0; i < b.N; i++ {
				algorithms.InterpolationSearch(data, target)
			}
		})
	}
}

// BenchmarkJumpSearch benchmarks jump search algorithm
func BenchmarkJumpSearch(b *testing.B) {
	for _, size := range benchmarkSizes {
		b.Run(fmt.Sprintf("Size%d", size), func(b *testing.B) {
			data := generateSortedData(size)
			target := data[size/2] // Middle element
			b.ResetTimer()
			b.ReportAllocs()
			
			for i := 0; i < b.N; i++ {
				algorithms.JumpSearch(data, target)
			}
		})
	}
}

// BenchmarkExponentialSearch benchmarks exponential search algorithm
func BenchmarkExponentialSearch(b *testing.B) {
	for _, size := range benchmarkSizes {
		b.Run(fmt.Sprintf("Size%d", size), func(b *testing.B) {
			data := generateSortedData(size)
			target := data[size/2] // Middle element
			b.ResetTimer()
			b.ReportAllocs()
			
			for i := 0; i < b.N; i++ {
				algorithms.ExponentialSearch(data, target)
			}
		})
	}
}

// BenchmarkTernarySearch benchmarks ternary search algorithm
func BenchmarkTernarySearch(b *testing.B) {
	for _, size := range benchmarkSizes {
		b.Run(fmt.Sprintf("Size%d", size), func(b *testing.B) {
			data := generateSortedData(size)
			target := data[size/2] // Middle element
			b.ResetTimer()
			b.ReportAllocs()
			
			for i := 0; i < b.N; i++ {
				algorithms.TernarySearch(data, target)
			}
		})
	}
}

// BenchmarkFibonacciSearch benchmarks Fibonacci search algorithm
func BenchmarkFibonacciSearch(b *testing.B) {
	for _, size := range benchmarkSizes {
		b.Run(fmt.Sprintf("Size%d", size), func(b *testing.B) {
			data := generateSortedData(size)
			target := data[size/2] // Middle element
			b.ResetTimer()
			b.ReportAllocs()
			
			for i := 0; i < b.N; i++ {
				algorithms.FibonacciSearch(data, target)
			}
		})
	}
}

// BenchmarkHashSearch benchmarks hash-based search
func BenchmarkHashSearch(b *testing.B) {
	for _, size := range benchmarkSizes {
		b.Run(fmt.Sprintf("Size%d", size), func(b *testing.B) {
			data := generateBenchmarkData(size)
			hashSearch := algorithms.NewHashSearch(data)
			target := data[size/2] // Middle element
			b.ResetTimer()
			b.ReportAllocs()
			
			for i := 0; i < b.N; i++ {
				hashSearch.Search(target)
			}
		})
	}
}

// BenchmarkSearchCache benchmarks cached search
func BenchmarkSearchCache(b *testing.B) {
	for _, size := range benchmarkSizes {
		b.Run(fmt.Sprintf("Size%d", size), func(b *testing.B) {
			data := generateSortedData(size)
			cache := algorithms.NewSearchCache()
			targets := make([]int, 100)
			for i := range targets {
				targets[i] = data[rand.Intn(size)]
			}
			b.ResetTimer()
			b.ReportAllocs()
			
			for i := 0; i < b.N; i++ {
				target := targets[i%len(targets)]
				cache.Search(data, target)
			}
		})
	}
}

// BenchmarkBloomFilter benchmarks Bloom filter operations
func BenchmarkBloomFilter(b *testing.B) {
	for _, size := range benchmarkSizes {
		b.Run(fmt.Sprintf("Size%d", size), func(b *testing.B) {
			data := generateBenchmarkData(size)
			bloomFilter := algorithms.NewBloomFilter(size*2, 3) // 3 hash functions
			
			// Add elements to bloom filter
			for _, item := range data {
				bloomFilter.Add(item)
			}
			
			target := data[size/2]
			b.ResetTimer()
			b.ReportAllocs()
			
			for i := 0; i < b.N; i++ {
				bloomFilter.Contains(target)
			}
		})
	}
}

// Search pattern benchmarks

// BenchmarkSearchBestCase benchmarks search algorithms in best case scenarios
func BenchmarkSearchBestCase(b *testing.B) {
	size := 1000
	data := generateSortedData(size)
	target := data[0] // First element (best case for some algorithms)
	
	b.Run("LinearSearch", func(b *testing.B) {
		b.ReportAllocs()
		for i := 0; i < b.N; i++ {
			algorithms.LinearSearch(data, target)
		}
	})
	
	b.Run("BinarySearch", func(b *testing.B) {
		target := data[size/2] // Middle element (best case for binary search)
		b.ReportAllocs()
		for i := 0; i < b.N; i++ {
			algorithms.BinarySearch(data, target)
		}
	})
}

// BenchmarkSearchWorstCase benchmarks search algorithms in worst case scenarios
func BenchmarkSearchWorstCase(b *testing.B) {
	size := 1000
	data := generateSortedData(size)
	target := data[size-1] // Last element (worst case for linear search)
	
	b.Run("LinearSearch", func(b *testing.B) {
		b.ReportAllocs()
		for i := 0; i < b.N; i++ {
			algorithms.LinearSearch(data, target)
		}
	})
	
	b.Run("BinarySearch", func(b *testing.B) {
		target := -1 // Element not in array (worst case for binary search)
		b.ReportAllocs()
		for i := 0; i < b.N; i++ {
			algorithms.BinarySearch(data, target)
		}
	})
}

// BenchmarkSearchComparison runs all search algorithms for comparison
func BenchmarkSearchComparison(b *testing.B) {
	size := 1000
	data := generateSortedData(size)
	target := data[size/2] // Middle element
	
	searchAlgorithms := map[string]func([]int, int) bool{
		"LinearSearch":        algorithms.LinearSearch,
		"BinarySearch":        algorithms.BinarySearch,
		"InterpolationSearch": algorithms.InterpolationSearch,
		"JumpSearch":          algorithms.JumpSearch,
		"ExponentialSearch":   algorithms.ExponentialSearch,
		"TernarySearch":       algorithms.TernarySearch,
		"FibonacciSearch":     algorithms.FibonacciSearch,
	}
	
	for name, searchFunc := range searchAlgorithms {
		b.Run(name, func(b *testing.B) {
			b.ReportAllocs()
			for i := 0; i < b.N; i++ {
				searchFunc(data, target)
			}
		})
	}
}

// BenchmarkSearchRange benchmarks range search operations
func BenchmarkSearchRange(b *testing.B) {
	for _, size := range benchmarkSizes {
		b.Run(fmt.Sprintf("Size%d", size), func(b *testing.B) {
			data := generateSortedData(size)
			// Add duplicates for range testing
			for i := 0; i < size/10; i++ {
				data[i] = data[size/2]
			}
			algorithms.QuickSort(data) // Re-sort after adding duplicates
			target := data[size/2]
			b.ResetTimer()
			b.ReportAllocs()
			
			for i := 0; i < b.N; i++ {
				algorithms.SearchRange(data, target)
			}
		})
	}
}

// Cache performance benchmarks

// BenchmarkCacheHitRate measures cache hit rates
func BenchmarkCacheHitRate(b *testing.B) {
	size := 1000
	data := generateSortedData(size)
	cache := algorithms.NewSearchCache()
	
	// Warm up cache with some searches
	for i := 0; i < 100; i++ {
		target := data[rand.Intn(size)]
		cache.Search(data, target)
	}
	
	b.ResetTimer()
	b.ReportAllocs()
	
	for i := 0; i < b.N; i++ {
		// Search for elements that are likely in cache
		target := data[rand.Intn(100)] // Limited range for higher hit rate
		cache.Search(data, target)
	}
	
	hits, misses := cache.GetStats()
	hitRate := float64(hits) / float64(hits+misses) * 100
	b.ReportMetric(hitRate, "hit_rate_%")
}

// Memory usage benchmarks

// BenchmarkSearchMemoryUsage measures memory allocations for search algorithms
func BenchmarkSearchMemoryUsage(b *testing.B) {
	size := 1000
	data := generateSortedData(size)
	target := data[size/2]
	
	b.Run("LinearSearch", func(b *testing.B) {
		b.ReportAllocs()
		for i := 0; i < b.N; i++ {
			algorithms.LinearSearch(data, target)
		}
	})
	
	b.Run("BinarySearch", func(b *testing.B) {
		b.ReportAllocs()
		for i := 0; i < b.N; i++ {
			algorithms.BinarySearch(data, target)
		}
	})
	
	b.Run("HashSearch", func(b *testing.B) {
		b.ReportAllocs()
		for i := 0; i < b.N; i++ {
			// Note: This includes hash table creation overhead
			hashSearch := algorithms.NewHashSearch(data)
			hashSearch.Search(target)
		}
	})
}