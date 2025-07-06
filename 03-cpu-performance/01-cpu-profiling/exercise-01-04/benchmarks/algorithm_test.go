package benchmarks

import (
	"fmt"
	"testing"
	"exercise-01-04/internal/algorithms"
	"exercise-01-04/internal/datastructures"
)

// Benchmark sorting algorithms
func BenchmarkNaiveQuickSort(b *testing.B) {
	data := algorithms.GenerateRandomData(1000)
	b.ResetTimer()
	
	for i := 0; i < b.N; i++ {
		testData := make([]int, len(data))
		copy(testData, data)
		algorithms.NaiveQuickSort(testData)
	}
}

func BenchmarkOptimizedQuickSort(b *testing.B) {
	data := algorithms.GenerateRandomData(1000)
	b.ResetTimer()
	
	for i := 0; i < b.N; i++ {
		testData := make([]int, len(data))
		copy(testData, data)
		algorithms.OptimizedQuickSort(testData)
	}
}

func BenchmarkNaiveMergeSort(b *testing.B) {
	data := algorithms.GenerateRandomData(1000)
	b.ResetTimer()
	
	for i := 0; i < b.N; i++ {
		testData := make([]int, len(data))
		copy(testData, data)
		algorithms.NaiveMergeSort(testData)
	}
}

func BenchmarkOptimizedMergeSort(b *testing.B) {
	data := algorithms.GenerateRandomData(1000)
	b.ResetTimer()
	
	for i := 0; i < b.N; i++ {
		testData := make([]int, len(data))
		copy(testData, data)
		algorithms.OptimizedMergeSort(testData)
	}
}

// Benchmark searching algorithms
func BenchmarkNaiveBinarySearch(b *testing.B) {
	data := algorithms.GenerateRandomData(10000)
	algorithms.OptimizedQuickSort(data) // Sort for binary search
	target := data[len(data)/2]
	b.ResetTimer()
	
	for i := 0; i < b.N; i++ {
		algorithms.NaiveBinarySearch(data, target)
	}
}

func BenchmarkOptimizedBinarySearch(b *testing.B) {
	data := algorithms.GenerateRandomData(10000)
	algorithms.OptimizedQuickSort(data) // Sort for binary search
	target := data[len(data)/2]
	cache := datastructures.NewSearchCache(1000)
	b.ResetTimer()
	
	for i := 0; i < b.N; i++ {
		algorithms.OptimizedBinarySearch(data, target, cache)
	}
}

func BenchmarkNaiveLinearSearch(b *testing.B) {
	data := algorithms.GenerateRandomData(1000)
	target := data[len(data)/2]
	b.ResetTimer()
	
	for i := 0; i < b.N; i++ {
		algorithms.NaiveLinearSearch(data, target)
	}
}

func BenchmarkOptimizedLinearSearch(b *testing.B) {
	data := algorithms.GenerateRandomData(1000)
	target := data[len(data)/2]
	b.ResetTimer()
	
	for i := 0; i < b.N; i++ {
		algorithms.OptimizedLinearSearch(data, target)
	}
}

// Benchmark cache performance
func BenchmarkSearchCache(b *testing.B) {
	cache := datastructures.NewSearchCache(1000)
	b.ResetTimer()
	
	for i := 0; i < b.N; i++ {
		cache.Set(i%100, i%2 == 0)
		cache.Get(i % 100)
	}
}

// Comparative benchmarks
func BenchmarkSortingComparison(b *testing.B) {
	sizes := []int{100, 1000, 10000}
	
	for _, size := range sizes {
		data := algorithms.GenerateRandomData(size)
		
		b.Run(fmt.Sprintf("Naive_QuickSort_%d", size), func(b *testing.B) {
			for i := 0; i < b.N; i++ {
				testData := make([]int, len(data))
				copy(testData, data)
				algorithms.NaiveQuickSort(testData)
			}
		})
		
		b.Run(fmt.Sprintf("Optimized_QuickSort_%d", size), func(b *testing.B) {
			for i := 0; i < b.N; i++ {
				testData := make([]int, len(data))
				copy(testData, data)
				algorithms.OptimizedQuickSort(testData)
			}
		})
	}
}