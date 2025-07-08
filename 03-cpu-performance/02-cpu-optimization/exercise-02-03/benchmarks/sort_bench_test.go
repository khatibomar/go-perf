package benchmarks

import (
	"fmt"
	"sort"
	"testing"

	"exercise-02-03/algorithms"
)

// Benchmark data sizes
var sizes = []int{100, 1000, 10000, 100000}

// BenchmarkQuickSort benchmarks all QuickSort variants
func BenchmarkQuickSort(b *testing.B) {
	for _, size := range sizes {
		b.Run(fmt.Sprintf("Standard_%d", size), func(b *testing.B) {
			data := GenerateRandomData(size)
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				dataCopy := make([]int, len(data))
				copy(dataCopy, data)
				algorithms.QuickSort(dataCopy)
			}
		})
		
		b.Run(fmt.Sprintf("Optimized_%d", size), func(b *testing.B) {
			data := GenerateRandomData(size)
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				dataCopy := make([]int, len(data))
				copy(dataCopy, data)
				algorithms.QuickSortOptimized(dataCopy)
			}
		})
		
		b.Run(fmt.Sprintf("Hybrid_%d", size), func(b *testing.B) {
			data := GenerateRandomData(size)
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				dataCopy := make([]int, len(data))
				copy(dataCopy, data)
				algorithms.QuickSortHybrid(dataCopy)
			}
		})
		
		b.Run(fmt.Sprintf("3Way_%d", size), func(b *testing.B) {
			data := GenerateRandomData(size)
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				dataCopy := make([]int, len(data))
				copy(dataCopy, data)
				algorithms.QuickSort3Way(dataCopy)
			}
		})
		
		b.Run(fmt.Sprintf("Iterative_%d", size), func(b *testing.B) {
			data := GenerateRandomData(size)
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				dataCopy := make([]int, len(data))
				copy(dataCopy, data)
				algorithms.QuickSortIterative(dataCopy)
			}
		})
	}
}

// BenchmarkMergeSort benchmarks all MergeSort variants
func BenchmarkMergeSort(b *testing.B) {
	for _, size := range sizes {
		b.Run(fmt.Sprintf("Standard_%d", size), func(b *testing.B) {
			data := GenerateRandomData(size)
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				dataCopy := make([]int, len(data))
				copy(dataCopy, data)
				algorithms.MergeSort(dataCopy)
			}
		})
		
		b.Run(fmt.Sprintf("Iterative_%d", size), func(b *testing.B) {
			data := GenerateRandomData(size)
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				dataCopy := make([]int, len(data))
				copy(dataCopy, data)
				algorithms.MergeSortIterative(dataCopy)
			}
		})
		
		b.Run(fmt.Sprintf("InPlace_%d", size), func(b *testing.B) {
			data := GenerateRandomData(size)
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				dataCopy := make([]int, len(data))
				copy(dataCopy, data)
				algorithms.MergeSortInPlace(dataCopy)
			}
		})
		
		b.Run(fmt.Sprintf("Optimized_%d", size), func(b *testing.B) {
			data := GenerateRandomData(size)
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				dataCopy := make([]int, len(data))
				copy(dataCopy, data)
				algorithms.MergeSortOptimized(dataCopy)
			}
		})
	}
}

// BenchmarkHeapSort benchmarks all HeapSort variants
func BenchmarkHeapSort(b *testing.B) {
	for _, size := range sizes {
		b.Run(fmt.Sprintf("Standard_%d", size), func(b *testing.B) {
			data := GenerateRandomData(size)
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				dataCopy := make([]int, len(data))
				copy(dataCopy, data)
				algorithms.HeapSort(dataCopy)
			}
		})
		
		b.Run(fmt.Sprintf("Optimized_%d", size), func(b *testing.B) {
			data := GenerateRandomData(size)
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				dataCopy := make([]int, len(data))
				copy(dataCopy, data)
				algorithms.HeapSortOptimized(dataCopy)
			}
		})
		
		b.Run(fmt.Sprintf("WithInsertionSort_%d", size), func(b *testing.B) {
			data := GenerateRandomData(size)
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				dataCopy := make([]int, len(data))
				copy(dataCopy, data)
				algorithms.HeapSortWithInsertionSort(dataCopy)
			}
		})
		
		b.Run(fmt.Sprintf("BottomUp_%d", size), func(b *testing.B) {
			data := GenerateRandomData(size)
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				dataCopy := make([]int, len(data))
				copy(dataCopy, data)
				algorithms.HeapSortBottomUp(dataCopy)
			}
		})
	}
}

// BenchmarkRadixSort benchmarks all RadixSort variants
func BenchmarkRadixSort(b *testing.B) {
	for _, size := range sizes {
		b.Run(fmt.Sprintf("LSD_%d", size), func(b *testing.B) {
			data := GenerateRandomData(size)
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				dataCopy := make([]int, len(data))
				copy(dataCopy, data)
				algorithms.RadixSortLSD(dataCopy)
			}
		})
		
		b.Run(fmt.Sprintf("MSD_%d", size), func(b *testing.B) {
			data := GenerateRandomData(size)
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				dataCopy := make([]int, len(data))
				copy(dataCopy, data)
				algorithms.RadixSortMSD(dataCopy)
			}
		})
		
		b.Run(fmt.Sprintf("Optimized_%d", size), func(b *testing.B) {
			data := GenerateRandomData(size)
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				dataCopy := make([]int, len(data))
				copy(dataCopy, data)
				algorithms.RadixSortOptimized(dataCopy)
			}
		})
	}
}

// BenchmarkTimSort benchmarks TimSort
func BenchmarkTimSort(b *testing.B) {
	for _, size := range sizes {
		b.Run(fmt.Sprintf("TimSort_%d", size), func(b *testing.B) {
			data := GenerateRandomData(size)
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				dataCopy := make([]int, len(data))
				copy(dataCopy, data)
				algorithms.TimSort(dataCopy)
			}
		})
	}
}

// BenchmarkGoStandardSort benchmarks Go's standard sort for comparison
func BenchmarkGoStandardSort(b *testing.B) {
	for _, size := range sizes {
		b.Run(fmt.Sprintf("GoSort_%d", size), func(b *testing.B) {
			data := GenerateRandomData(size)
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				dataCopy := make([]int, len(data))
				copy(dataCopy, data)
				sort.Ints(dataCopy)
			}
		})
	}
}

// BenchmarkDataPatterns benchmarks algorithms on different data patterns
func BenchmarkDataPatterns(b *testing.B) {
	size := 10000
	patterns := map[string]func(int) []int{
		"Random": GenerateRandomData,
		"Sorted": GenerateSortedData,
		"Reverse": GenerateReverseSortedData,
		"Partiallysorted": GeneratePartiallySortedData,
		"ManyDuplicates": GenerateManyDuplicatesData,
		"FewUnique": GenerateFewUniqueData,
	}
	
	algorithms := map[string]func([]int){
		"QuickSort": algorithms.QuickSort,
		"QuickSortOptimized": algorithms.QuickSortOptimized,
		"QuickSort3Way": algorithms.QuickSort3Way,
		"MergeSort": algorithms.MergeSort,
		"MergeSortOptimized": algorithms.MergeSortOptimized,
		"HeapSort": algorithms.HeapSort,
		"HeapSortOptimized": algorithms.HeapSortOptimized,
		"RadixSortLSD": algorithms.RadixSortLSD,
		"TimSort": algorithms.TimSort,
	}
	
	for patternName, patternFunc := range patterns {
		for algoName, algoFunc := range algorithms {
			b.Run(fmt.Sprintf("%s_%s_%d", algoName, patternName, size), func(b *testing.B) {
				data := patternFunc(size)
				b.ResetTimer()
				for i := 0; i < b.N; i++ {
					dataCopy := make([]int, len(data))
					copy(dataCopy, data)
					algoFunc(dataCopy)
				}
			})
		}
	}
}

// BenchmarkMemoryAllocations benchmarks memory allocation patterns
func BenchmarkMemoryAllocations(b *testing.B) {
	size := 10000
	data := GenerateRandomData(size)
	
	b.Run("QuickSort_Allocs", func(b *testing.B) {
		b.ReportAllocs()
		for i := 0; i < b.N; i++ {
			dataCopy := make([]int, len(data))
			copy(dataCopy, data)
			algorithms.QuickSort(dataCopy)
		}
	})
	
	b.Run("MergeSort_Allocs", func(b *testing.B) {
		b.ReportAllocs()
		for i := 0; i < b.N; i++ {
			dataCopy := make([]int, len(data))
			copy(dataCopy, data)
			algorithms.MergeSort(dataCopy)
		}
	})
	
	b.Run("HeapSort_Allocs", func(b *testing.B) {
		b.ReportAllocs()
		for i := 0; i < b.N; i++ {
			dataCopy := make([]int, len(data))
			copy(dataCopy, data)
			algorithms.HeapSort(dataCopy)
		}
	})
	
	b.Run("RadixSort_Allocs", func(b *testing.B) {
		b.ReportAllocs()
		for i := 0; i < b.N; i++ {
			dataCopy := make([]int, len(data))
			copy(dataCopy, data)
			algorithms.RadixSortLSD(dataCopy)
		}
	})
	
	b.Run("TimSort_Allocs", func(b *testing.B) {
		b.ReportAllocs()
		for i := 0; i < b.N; i++ {
			dataCopy := make([]int, len(data))
			copy(dataCopy, data)
			algorithms.TimSort(dataCopy)
		}
	})
}

// BenchmarkSmallArrays benchmarks performance on small arrays
func BenchmarkSmallArrays(b *testing.B) {
	smallSizes := []int{5, 10, 20, 50, 100}
	
	for _, size := range smallSizes {
		b.Run(fmt.Sprintf("QuickSort_%d", size), func(b *testing.B) {
			data := GenerateRandomData(size)
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				dataCopy := make([]int, len(data))
				copy(dataCopy, data)
				algorithms.QuickSort(dataCopy)
			}
		})
		
		b.Run(fmt.Sprintf("InsertionSort_%d", size), func(b *testing.B) {
			data := GenerateRandomData(size)
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				dataCopy := make([]int, len(data))
				copy(dataCopy, data)
				insertionSort(dataCopy)
			}
		})
		
		b.Run(fmt.Sprintf("TimSort_%d", size), func(b *testing.B) {
			data := GenerateRandomData(size)
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				dataCopy := make([]int, len(data))
				copy(dataCopy, data)
				algorithms.TimSort(dataCopy)
			}
		})
	}
}

// insertionSort for comparison in small array benchmarks
func insertionSort(data []int) {
	for i := 1; i < len(data); i++ {
		key := data[i]
		j := i - 1
		for j >= 0 && data[j] > key {
			data[j+1] = data[j]
			j--
		}
		data[j+1] = key
	}
}

// BenchmarkLargeArrays benchmarks performance on large arrays
func BenchmarkLargeArrays(b *testing.B) {
	largeSizes := []int{500000, 1000000}
	
	for _, size := range largeSizes {
		b.Run(fmt.Sprintf("QuickSortOptimized_%d", size), func(b *testing.B) {
			data := GenerateRandomData(size)
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				dataCopy := make([]int, len(data))
				copy(dataCopy, data)
				algorithms.QuickSortOptimized(dataCopy)
			}
		})
		
		b.Run(fmt.Sprintf("MergeSortOptimized_%d", size), func(b *testing.B) {
			data := GenerateRandomData(size)
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				dataCopy := make([]int, len(data))
				copy(dataCopy, data)
				algorithms.MergeSortOptimized(dataCopy)
			}
		})
		
		b.Run(fmt.Sprintf("TimSort_%d", size), func(b *testing.B) {
			data := GenerateRandomData(size)
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				dataCopy := make([]int, len(data))
				copy(dataCopy, data)
				algorithms.TimSort(dataCopy)
			}
		})
		
		b.Run(fmt.Sprintf("RadixSortOptimized_%d", size), func(b *testing.B) {
			data := GenerateRandomData(size)
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				dataCopy := make([]int, len(data))
				copy(dataCopy, data)
				algorithms.RadixSortOptimized(dataCopy)
			}
		})
	}
}