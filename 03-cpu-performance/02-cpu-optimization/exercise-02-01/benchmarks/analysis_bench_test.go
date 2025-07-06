package benchmarks

import (
	"fmt"
	"math/rand"
	"testing"
	"time"

	"exercise-02-01/analysis"
	"exercise-02-01/algorithms"
)

// Analysis benchmark sizes
var analysisSizes = []int{100, 500, 1000, 5000, 10000}

// BenchmarkComplexityAnalysis benchmarks complexity analysis functions
func BenchmarkComplexityAnalysis(b *testing.B) {
	for _, size := range analysisSizes {
		b.Run(fmt.Sprintf("Size%d", size), func(b *testing.B) {
			b.ResetTimer()
			b.ReportAllocs()
			
			for i := 0; i < b.N; i++ {
				analysis.AnalyzeSortingComplexity(size)
			}
		})
	}
}

// BenchmarkSearchComplexityAnalysis benchmarks search complexity analysis
func BenchmarkSearchComplexityAnalysis(b *testing.B) {
	for _, size := range analysisSizes {
		b.Run(fmt.Sprintf("Size%d", size), func(b *testing.B) {
			b.ResetTimer()
			b.ReportAllocs()
			
			for i := 0; i < b.N; i++ {
				analysis.AnalyzeSearchingComplexity(size)
			}
		})
	}
}

// BenchmarkPerformancePrediction benchmarks performance prediction
func BenchmarkPerformancePrediction(b *testing.B) {
	inputSizes := []int{100, 1000, 10000}
	
	for _, size := range inputSizes {
		b.Run(fmt.Sprintf("Size%d", size), func(b *testing.B) {
			b.ReportAllocs()
			
			for i := 0; i < b.N; i++ {
				analysis.GeneratePerformancePredictions(size)
			}
		})
	}
}

// BenchmarkMemoryAnalysis benchmarks memory usage analysis
func BenchmarkMemoryAnalysis(b *testing.B) {
	for _, size := range analysisSizes {
		b.Run(fmt.Sprintf("Size%d", size), func(b *testing.B) {
			b.ResetTimer()
			b.ReportAllocs()
			
			for i := 0; i < b.N; i++ {
				analysis.AnalyzeMemoryComplexity(size)
			}
		})
	}
}

// BenchmarkCacheAnalysis benchmarks cache performance analysis
func BenchmarkCacheAnalysis(b *testing.B) {
	b.ReportAllocs()
	
	for i := 0; i < b.N; i++ {
		analysis.AnalyzeCachePerformance()
	}
}

// BenchmarkBranchPredictionAnalysis benchmarks branch prediction analysis
func BenchmarkBranchPredictionAnalysis(b *testing.B) {
	b.ReportAllocs()
	
	for i := 0; i < b.N; i++ {
		analysis.BranchPredictionAnalysis()
	}
}

// Algorithm-specific analysis benchmarks

// BenchmarkSortingAlgorithmAnalysis benchmarks sorting algorithm analysis
func BenchmarkSortingAlgorithmAnalysis(b *testing.B) {
	size := 1000
	data := generateRandomData(size)
	
	b.Run("BubbleSort", func(b *testing.B) {
		b.ReportAllocs()
		for i := 0; i < b.N; i++ {
			dataCopy := make([]int, len(data))
			copy(dataCopy, data)
			analysis.AnalyzeSortingComplexity(size)
			algorithms.BubbleSort(dataCopy)
		}
	})
	
	b.Run("QuickSort", func(b *testing.B) {
		b.ReportAllocs()
		for i := 0; i < b.N; i++ {
			dataCopy := make([]int, len(data))
			copy(dataCopy, data)
			analysis.AnalyzeSortingComplexity(size)
			algorithms.QuickSort(dataCopy)
		}
	})
	
	b.Run("MergeSort", func(b *testing.B) {
		b.ReportAllocs()
		for i := 0; i < b.N; i++ {
			dataCopy := make([]int, len(data))
			copy(dataCopy, data)
			analysis.AnalyzeSortingComplexity(size)
			algorithms.MergeSort(dataCopy)
		}
	})
	
	b.Run("HeapSort", func(b *testing.B) {
		b.ReportAllocs()
		for i := 0; i < b.N; i++ {
			dataCopy := make([]int, len(data))
			copy(dataCopy, data)
			analysis.AnalyzeSortingComplexity(size)
			algorithms.HeapSort(dataCopy)
		}
	})
}

// BenchmarkSearchAlgorithmAnalysis benchmarks search algorithm analysis
func BenchmarkSearchAlgorithmAnalysis(b *testing.B) {
	size := 10000
	data := generateSortedData(size)
	target := data[size/2]
	
	b.Run("LinearSearch", func(b *testing.B) {
		b.ReportAllocs()
		for i := 0; i < b.N; i++ {
			analysis.AnalyzeSearchingComplexity(size)
			algorithms.LinearSearch(data, target)
		}
	})
	
	b.Run("BinarySearch", func(b *testing.B) {
		b.ReportAllocs()
		for i := 0; i < b.N; i++ {
			analysis.AnalyzeSearchingComplexity(size)
			algorithms.BinarySearch(data, target)
		}
	})
	
	b.Run("InterpolationSearch", func(b *testing.B) {
		b.ReportAllocs()
		for i := 0; i < b.N; i++ {
			analysis.AnalyzeSearchingComplexity(size)
			algorithms.InterpolationSearch(data, target)
		}
	})
	
	b.Run("JumpSearch", func(b *testing.B) {
		b.ReportAllocs()
		for i := 0; i < b.N; i++ {
			analysis.AnalyzeSearchingComplexity(size)
			algorithms.JumpSearch(data, target)
		}
	})
}

// Data pattern analysis benchmarks

// BenchmarkDataPatternAnalysis benchmarks analysis of different data patterns
func BenchmarkDataPatternAnalysis(b *testing.B) {
	size := 5000
	
	b.Run("RandomData", func(b *testing.B) {
		b.ReportAllocs()
		for i := 0; i < b.N; i++ {
			analysis.AnalyzeSortingComplexity(size)
		}
	})
	
	b.Run("SortedData", func(b *testing.B) {
		b.ReportAllocs()
		for i := 0; i < b.N; i++ {
			analysis.AnalyzeSortingComplexity(size)
		}
	})
	
	b.Run("ReverseSortedData", func(b *testing.B) {
		b.ReportAllocs()
		for i := 0; i < b.N; i++ {
			analysis.AnalyzeSortingComplexity(size)
		}
	})
	
	b.Run("NearlySortedData", func(b *testing.B) {
		b.ReportAllocs()
		for i := 0; i < b.N; i++ {
			analysis.AnalyzeSortingComplexity(size)
		}
	})
}

// Performance prediction benchmarks

// BenchmarkComplexityPrediction benchmarks different complexity predictions
func BenchmarkComplexityPrediction(b *testing.B) {
	size := 1000
	
	b.Run("GeneratePerformancePredictions", func(b *testing.B) {
		b.ReportAllocs()
		for i := 0; i < b.N; i++ {
			analysis.GeneratePerformancePredictions(size)
		}
	})
}

// Memory usage pattern benchmarks

// BenchmarkMemoryPatternAnalysis benchmarks memory pattern analysis
func BenchmarkMemoryPatternAnalysis(b *testing.B) {
	sizes := []int{100, 1000, 10000}
	
	for _, size := range sizes {
		b.Run(fmt.Sprintf("Sequential_Size%d", size), func(b *testing.B) {
			b.ResetTimer()
			b.ReportAllocs()
			
			for i := 0; i < b.N; i++ {
				analysis.AnalyzeMemoryComplexity(size)
			}
		})
		
		b.Run(fmt.Sprintf("Random_Size%d", size), func(b *testing.B) {
			b.ResetTimer()
			b.ReportAllocs()
			
			for i := 0; i < b.N; i++ {
				analysis.AnalyzeMemoryComplexity(size)
			}
		})
	}
}

// Cache performance benchmarks

// BenchmarkCachePatternAnalysis benchmarks cache pattern analysis
func BenchmarkCachePatternAnalysis(b *testing.B) {
	b.Run("CachePerformance", func(b *testing.B) {
		b.ReportAllocs()
		
		for i := 0; i < b.N; i++ {
			analysis.AnalyzeCachePerformance()
		}
	})
}

// Branch prediction benchmarks

// BenchmarkBranchPatternAnalysis benchmarks branch prediction pattern analysis
func BenchmarkBranchPatternAnalysis(b *testing.B) {
	b.Run("BranchPrediction", func(b *testing.B) {
		b.ReportAllocs()
		
		for i := 0; i < b.N; i++ {
			analysis.BranchPredictionAnalysis()
		}
	})
}

// Comprehensive analysis benchmarks

// BenchmarkComprehensiveAnalysis benchmarks full analysis suite
func BenchmarkComprehensiveAnalysis(b *testing.B) {
	size := 1000
	
	b.Run("FullAnalysis", func(b *testing.B) {
		b.ReportAllocs()
		for i := 0; i < b.N; i++ {
			// Perform comprehensive analysis
			analysis.AnalyzeSortingComplexity(size)
			analysis.AnalyzeSearchingComplexity(size)
			analysis.AnalyzeMemoryComplexity(size)
			analysis.AnalyzeCachePerformance()
			analysis.BranchPredictionAnalysis()
			
			// Performance predictions
			analysis.GeneratePerformancePredictions(size)
		}
	})
}

// Scalability analysis benchmarks

// BenchmarkAnalysisScalability tests analysis scalability
func BenchmarkAnalysisScalability(b *testing.B) {
	largeSizes := []int{10000, 50000, 100000}
	
	for _, size := range largeSizes {
		b.Run(fmt.Sprintf("Complexity_Size%d", size), func(b *testing.B) {
			b.ResetTimer()
			b.ReportAllocs()
			
			for i := 0; i < b.N; i++ {
				analysis.AnalyzeSortingComplexity(size)
			}
		})
		
		b.Run(fmt.Sprintf("Memory_Size%d", size), func(b *testing.B) {
			b.ResetTimer()
			b.ReportAllocs()
			
			for i := 0; i < b.N; i++ {
				analysis.AnalyzeMemoryComplexity(size)
			}
		})
	}
}

// Helper functions for data generation

// generateSequentialData creates sequential data for cache-friendly access
func generateSequentialData(size int) []int {
	data := make([]int, size)
	for i := 0; i < size; i++ {
		data[i] = i
	}
	return data
}

// generateNearlySortedData creates nearly sorted data with some disorder
func generateNearlySortedData(size int) []int {
	data := generateSortedData(size)
	
	// Introduce some disorder (swap 5% of elements)
	swaps := size / 20
	for i := 0; i < swaps; i++ {
		idx1 := i * 20 % size
		idx2 := (i*20 + 10) % size
		data[idx1], data[idx2] = data[idx2], data[idx1]
	}
	
	return data
}

// generateRandomData creates random data
func generateRandomData(size int) []int {
	rand.Seed(time.Now().UnixNano())
	data := make([]int, size)
	for i := 0; i < size; i++ {
		data[i] = rand.Intn(size * 10)
	}
	return data
}