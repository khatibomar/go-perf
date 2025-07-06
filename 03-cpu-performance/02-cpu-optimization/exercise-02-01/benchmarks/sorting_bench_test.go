package benchmarks

import (
	"fmt"
	"math/rand"
	"testing"
	"time"

	"exercise-02-01/algorithms"
)

// Benchmark data sizes
var benchmarkSizes = []int{100, 1000, 10000}

// generateBenchmarkData creates test data for benchmarks
func generateBenchmarkData(size int) []int {
	rand.Seed(time.Now().UnixNano())
	data := make([]int, size)
	for i := 0; i < size; i++ {
		data[i] = rand.Intn(size * 10)
	}
	return data
}

// generateSortedData creates sorted test data
func generateSortedData(size int) []int {
	data := generateBenchmarkData(size)
	algorithms.QuickSort(data)
	return data
}

// generateReverseSortedData creates reverse sorted test data
func generateReverseSortedData(size int) []int {
	data := generateSortedData(size)
	// Reverse the array
	for i, j := 0, len(data)-1; i < j; i, j = i+1, j-1 {
		data[i], data[j] = data[j], data[i]
	}
	return data
}

// BenchmarkBubbleSort benchmarks bubble sort algorithm
func BenchmarkBubbleSort(b *testing.B) {
	for _, size := range benchmarkSizes {
		b.Run(fmt.Sprintf("Size%d", size), func(b *testing.B) {
			data := generateBenchmarkData(size)
			b.ResetTimer()
			b.ReportAllocs()
			
			for i := 0; i < b.N; i++ {
				testData := make([]int, len(data))
				copy(testData, data)
				algorithms.BubbleSort(testData)
			}
		})
	}
}

// BenchmarkQuickSort benchmarks quick sort algorithm
func BenchmarkQuickSort(b *testing.B) {
	for _, size := range benchmarkSizes {
		b.Run(fmt.Sprintf("Size%d", size), func(b *testing.B) {
			data := generateBenchmarkData(size)
			b.ResetTimer()
			b.ReportAllocs()
			
			for i := 0; i < b.N; i++ {
				testData := make([]int, len(data))
				copy(testData, data)
				algorithms.QuickSort(testData)
			}
		})
	}
}

// BenchmarkMergeSort benchmarks merge sort algorithm
func BenchmarkMergeSort(b *testing.B) {
	for _, size := range benchmarkSizes {
		b.Run(fmt.Sprintf("Size%d", size), func(b *testing.B) {
			data := generateBenchmarkData(size)
			b.ResetTimer()
			b.ReportAllocs()
			
			for i := 0; i < b.N; i++ {
				testData := make([]int, len(data))
				copy(testData, data)
				algorithms.MergeSort(testData)
			}
		})
	}
}

// BenchmarkHeapSort benchmarks heap sort algorithm
func BenchmarkHeapSort(b *testing.B) {
	for _, size := range benchmarkSizes {
		b.Run(fmt.Sprintf("Size%d", size), func(b *testing.B) {
			data := generateBenchmarkData(size)
			b.ResetTimer()
			b.ReportAllocs()
			
			for i := 0; i < b.N; i++ {
				testData := make([]int, len(data))
				copy(testData, data)
				algorithms.HeapSort(testData)
			}
		})
	}
}

// BenchmarkRadixSort benchmarks radix sort algorithm
func BenchmarkRadixSort(b *testing.B) {
	for _, size := range benchmarkSizes {
		b.Run(fmt.Sprintf("Size%d", size), func(b *testing.B) {
			data := generateBenchmarkData(size)
			b.ResetTimer()
			b.ReportAllocs()
			
			for i := 0; i < b.N; i++ {
				testData := make([]int, len(data))
				copy(testData, data)
				algorithms.RadixSort(testData)
			}
		})
	}
}

// BenchmarkInsertionSort benchmarks insertion sort algorithm
func BenchmarkInsertionSort(b *testing.B) {
	for _, size := range benchmarkSizes {
		b.Run(fmt.Sprintf("Size%d", size), func(b *testing.B) {
			data := generateBenchmarkData(size)
			b.ResetTimer()
			b.ReportAllocs()
			
			for i := 0; i < b.N; i++ {
				testData := make([]int, len(data))
				copy(testData, data)
				algorithms.InsertionSort(testData)
			}
		})
	}
}

// BenchmarkSelectionSort benchmarks selection sort algorithm
func BenchmarkSelectionSort(b *testing.B) {
	for _, size := range benchmarkSizes {
		b.Run(fmt.Sprintf("Size%d", size), func(b *testing.B) {
			data := generateBenchmarkData(size)
			b.ResetTimer()
			b.ReportAllocs()
			
			for i := 0; i < b.N; i++ {
				testData := make([]int, len(data))
				copy(testData, data)
				algorithms.SelectionSort(testData)
			}
		})
	}
}

// BenchmarkTimSort benchmarks Tim sort algorithm
func BenchmarkTimSort(b *testing.B) {
	for _, size := range benchmarkSizes {
		b.Run(fmt.Sprintf("Size%d", size), func(b *testing.B) {
			data := generateBenchmarkData(size)
			b.ResetTimer()
			b.ReportAllocs()
			
			for i := 0; i < b.N; i++ {
				testData := make([]int, len(data))
				copy(testData, data)
				algorithms.TimSort(testData)
			}
		})
	}
}

// BenchmarkCountingSort benchmarks counting sort algorithm
func BenchmarkCountingSort(b *testing.B) {
	for _, size := range benchmarkSizes {
		b.Run(fmt.Sprintf("Size%d", size), func(b *testing.B) {
			data := generateBenchmarkData(size)
			maxVal := size * 10 // Known max value for counting sort
			b.ResetTimer()
			b.ReportAllocs()
			
			for i := 0; i < b.N; i++ {
				testData := make([]int, len(data))
				copy(testData, data)
				algorithms.CountingSort(testData, maxVal)
			}
		})
	}
}

// BenchmarkBucketSort benchmarks bucket sort algorithm
func BenchmarkBucketSort(b *testing.B) {
	for _, size := range benchmarkSizes {
		b.Run(fmt.Sprintf("Size%d", size), func(b *testing.B) {
			data := generateBenchmarkData(size)
			bucketCount := size / 10 // Reasonable bucket count
			if bucketCount < 1 {
				bucketCount = 1
			}
			b.ResetTimer()
			b.ReportAllocs()
			
			for i := 0; i < b.N; i++ {
				testData := make([]int, len(data))
				copy(testData, data)
				algorithms.BucketSort(testData, bucketCount)
			}
		})
	}
}

// Benchmark sorting algorithms on different data patterns

// BenchmarkQuickSortSorted benchmarks QuickSort on already sorted data
func BenchmarkQuickSortSorted(b *testing.B) {
	for _, size := range benchmarkSizes {
		b.Run(fmt.Sprintf("Size%d", size), func(b *testing.B) {
			data := generateSortedData(size)
			b.ResetTimer()
			b.ReportAllocs()
			
			for i := 0; i < b.N; i++ {
				testData := make([]int, len(data))
				copy(testData, data)
				algorithms.QuickSort(testData)
			}
		})
	}
}

// BenchmarkQuickSortReverseSorted benchmarks QuickSort on reverse sorted data
func BenchmarkQuickSortReverseSorted(b *testing.B) {
	for _, size := range benchmarkSizes {
		b.Run(fmt.Sprintf("Size%d", size), func(b *testing.B) {
			data := generateReverseSortedData(size)
			b.ResetTimer()
			b.ReportAllocs()
			
			for i := 0; i < b.N; i++ {
				testData := make([]int, len(data))
				copy(testData, data)
				algorithms.QuickSort(testData)
			}
		})
	}
}

// BenchmarkInsertionSortSorted benchmarks InsertionSort on already sorted data (best case)
func BenchmarkInsertionSortSorted(b *testing.B) {
	for _, size := range benchmarkSizes {
		b.Run(fmt.Sprintf("Size%d", size), func(b *testing.B) {
			data := generateSortedData(size)
			b.ResetTimer()
			b.ReportAllocs()
			
			for i := 0; i < b.N; i++ {
				testData := make([]int, len(data))
				copy(testData, data)
				algorithms.InsertionSort(testData)
			}
		})
	}
}

// Comparative benchmarks

// BenchmarkSortingComparison runs all sorting algorithms for comparison
func BenchmarkSortingComparison(b *testing.B) {
	size := 1000
	data := generateBenchmarkData(size)
	
	algorithms := map[string]func([]int){
		"QuickSort":     algorithms.QuickSort,
		"MergeSort":     algorithms.MergeSort,
		"HeapSort":      algorithms.HeapSort,
		"RadixSort":     algorithms.RadixSort,
		"InsertionSort": algorithms.InsertionSort,
		"TimSort":       algorithms.TimSort,
	}
	
	for name, sortFunc := range algorithms {
		b.Run(name, func(b *testing.B) {
			b.ReportAllocs()
			for i := 0; i < b.N; i++ {
				testData := make([]int, len(data))
				copy(testData, data)
				sortFunc(testData)
			}
		})
	}
}

// Memory allocation benchmarks

// BenchmarkSortingMemoryUsage measures memory allocations for sorting algorithms
func BenchmarkSortingMemoryUsage(b *testing.B) {
	size := 1000
	data := generateBenchmarkData(size)
	
	b.Run("QuickSort", func(b *testing.B) {
		b.ReportAllocs()
		for i := 0; i < b.N; i++ {
			testData := make([]int, len(data))
			copy(testData, data)
			algorithms.QuickSort(testData)
		}
	})
	
	b.Run("MergeSort", func(b *testing.B) {
		b.ReportAllocs()
		for i := 0; i < b.N; i++ {
			testData := make([]int, len(data))
			copy(testData, data)
			algorithms.MergeSort(testData)
		}
	})
	
	b.Run("RadixSort", func(b *testing.B) {
		b.ReportAllocs()
		for i := 0; i < b.N; i++ {
			testData := make([]int, len(data))
			copy(testData, data)
			algorithms.RadixSort(testData)
		}
	})
}