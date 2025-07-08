package benchmarks

import (
	"os"
	"runtime/pprof"
	"testing"

	"exercise-02-03/algorithms"
)

// BenchmarkQuickSortProfile benchmarks QuickSort with CPU profiling
func BenchmarkQuickSortProfile(b *testing.B) {
	size := 100000
	data := GenerateRandomData(size)
	
	// Start CPU profiling
	f, err := os.Create("quicksort_cpu.prof")
	if err != nil {
		b.Fatal(err)
	}
	defer f.Close()
	
	if err := pprof.StartCPUProfile(f); err != nil {
		b.Fatal(err)
	}
	defer pprof.StopCPUProfile()
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		dataCopy := make([]int, len(data))
		copy(dataCopy, data)
		algorithms.QuickSort(dataCopy)
	}
}

// BenchmarkMergeSortProfile benchmarks MergeSort with CPU profiling
func BenchmarkMergeSortProfile(b *testing.B) {
	size := 100000
	data := GenerateRandomData(size)
	
	// Start CPU profiling
	f, err := os.Create("mergesort_cpu.prof")
	if err != nil {
		b.Fatal(err)
	}
	defer f.Close()
	
	if err := pprof.StartCPUProfile(f); err != nil {
		b.Fatal(err)
	}
	defer pprof.StopCPUProfile()
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		dataCopy := make([]int, len(data))
		copy(dataCopy, data)
		algorithms.MergeSort(dataCopy)
	}
}

// BenchmarkHeapSortProfile benchmarks HeapSort with CPU profiling
func BenchmarkHeapSortProfile(b *testing.B) {
	size := 100000
	data := GenerateRandomData(size)
	
	// Start CPU profiling
	f, err := os.Create("heapsort_cpu.prof")
	if err != nil {
		b.Fatal(err)
	}
	defer f.Close()
	
	if err := pprof.StartCPUProfile(f); err != nil {
		b.Fatal(err)
	}
	defer pprof.StopCPUProfile()
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		dataCopy := make([]int, len(data))
		copy(dataCopy, data)
		algorithms.HeapSort(dataCopy)
	}
}

// BenchmarkRadixSortProfile benchmarks RadixSort with CPU profiling
func BenchmarkRadixSortProfile(b *testing.B) {
	size := 100000
	data := GenerateRandomData(size)
	
	// Start CPU profiling
	f, err := os.Create("radixsort_cpu.prof")
	if err != nil {
		b.Fatal(err)
	}
	defer f.Close()
	
	if err := pprof.StartCPUProfile(f); err != nil {
		b.Fatal(err)
	}
	defer pprof.StopCPUProfile()
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		dataCopy := make([]int, len(data))
		copy(dataCopy, data)
		algorithms.RadixSortLSD(dataCopy)
	}
}

// BenchmarkTimSortProfile benchmarks TimSort with CPU profiling
func BenchmarkTimSortProfile(b *testing.B) {
	size := 100000
	data := GenerateRandomData(size)
	
	// Start CPU profiling
	f, err := os.Create("timsort_cpu.prof")
	if err != nil {
		b.Fatal(err)
	}
	defer f.Close()
	
	if err := pprof.StartCPUProfile(f); err != nil {
		b.Fatal(err)
	}
	defer pprof.StopCPUProfile()
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		dataCopy := make([]int, len(data))
		copy(dataCopy, data)
		algorithms.TimSort(dataCopy)
	}
}

// BenchmarkCachePerformance benchmarks cache performance patterns
func BenchmarkCachePerformance(b *testing.B) {
	sizes := []int{1000, 10000, 100000, 1000000}
	
	for _, size := range sizes {
		b.Run("Sequential_"+string(rune(size)), func(b *testing.B) {
			data := GenerateRandomData(size)
			b.ResetTimer()
			
			for i := 0; i < b.N; i++ {
				dataCopy := make([]int, len(data))
				copy(dataCopy, data)
				// Sequential access pattern (cache-friendly)
				algorithms.MergeSort(dataCopy)
			}
		})
		
		b.Run("Random_"+string(rune(size)), func(b *testing.B) {
			data := GenerateRandomData(size)
			b.ResetTimer()
			
			for i := 0; i < b.N; i++ {
				dataCopy := make([]int, len(data))
				copy(dataCopy, data)
				// Random access pattern (cache-unfriendly)
				algorithms.QuickSort(dataCopy)
			}
		})
	}
}

// BenchmarkBranchPrediction benchmarks branch prediction performance
func BenchmarkBranchPrediction(b *testing.B) {
	size := 100000
	
	b.Run("Predictable_Sorted", func(b *testing.B) {
		data := GenerateSortedData(size)
		b.ResetTimer()
		
		for i := 0; i < b.N; i++ {
			dataCopy := make([]int, len(data))
			copy(dataCopy, data)
			algorithms.QuickSort(dataCopy)
		}
	})
	
	b.Run("Unpredictable_Random", func(b *testing.B) {
		data := GenerateRandomData(size)
		b.ResetTimer()
		
		for i := 0; i < b.N; i++ {
			dataCopy := make([]int, len(data))
			copy(dataCopy, data)
			algorithms.QuickSort(dataCopy)
		}
	})
	
	b.Run("Optimized_3Way", func(b *testing.B) {
		data := GenerateManyDuplicatesData(size)
		b.ResetTimer()
		
		for i := 0; i < b.N; i++ {
			dataCopy := make([]int, len(data))
			copy(dataCopy, data)
			algorithms.QuickSort3Way(dataCopy)
		}
	})
}

// BenchmarkMemoryAccess benchmarks different memory access patterns
func BenchmarkMemoryAccess(b *testing.B) {
	size := 100000
	
	b.Run("InPlace_HeapSort", func(b *testing.B) {
		data := GenerateRandomData(size)
		b.ReportAllocs()
		b.ResetTimer()
		
		for i := 0; i < b.N; i++ {
			dataCopy := make([]int, len(data))
			copy(dataCopy, data)
			algorithms.HeapSort(dataCopy)
		}
	})
	
	b.Run("ExtraSpace_MergeSort", func(b *testing.B) {
		data := GenerateRandomData(size)
		b.ReportAllocs()
		b.ResetTimer()
		
		for i := 0; i < b.N; i++ {
			dataCopy := make([]int, len(data))
			copy(dataCopy, data)
			algorithms.MergeSort(dataCopy)
		}
	})
	
	b.Run("LinearSpace_RadixSort", func(b *testing.B) {
		data := GenerateRandomData(size)
		b.ReportAllocs()
		b.ResetTimer()
		
		for i := 0; i < b.N; i++ {
			dataCopy := make([]int, len(data))
			copy(dataCopy, data)
			algorithms.RadixSortLSD(dataCopy)
		}
	})
}

// BenchmarkWorstCase benchmarks worst-case scenarios
func BenchmarkWorstCase(b *testing.B) {
	size := 10000 // Smaller size for worst-case scenarios
	
	b.Run("QuickSort_Worst", func(b *testing.B) {
		data := GenerateSortedData(size) // Worst case for basic QuickSort
		b.ResetTimer()
		
		for i := 0; i < b.N; i++ {
			dataCopy := make([]int, len(data))
			copy(dataCopy, data)
			algorithms.QuickSort(dataCopy)
		}
	})
	
	b.Run("QuickSort_Optimized_Worst", func(b *testing.B) {
		data := GenerateSortedData(size)
		b.ResetTimer()
		
		for i := 0; i < b.N; i++ {
			dataCopy := make([]int, len(data))
			copy(dataCopy, data)
			algorithms.QuickSortOptimized(dataCopy)
		}
	})
	
	b.Run("MergeSort_Consistent", func(b *testing.B) {
		data := GenerateSortedData(size)
		b.ResetTimer()
		
		for i := 0; i < b.N; i++ {
			dataCopy := make([]int, len(data))
			copy(dataCopy, data)
			algorithms.MergeSort(dataCopy)
		}
	})
}

// BenchmarkBestCase benchmarks best-case scenarios
func BenchmarkBestCase(b *testing.B) {
	size := 100000
	
	b.Run("TimSort_Best", func(b *testing.B) {
		data := GenerateSortedData(size) // Best case for TimSort
		b.ResetTimer()
		
		for i := 0; i < b.N; i++ {
			dataCopy := make([]int, len(data))
			copy(dataCopy, data)
			algorithms.TimSort(dataCopy)
		}
	})
	
	b.Run("RadixSort_Integers", func(b *testing.B) {
		data := GenerateRandomData(size) // Good case for RadixSort
		b.ResetTimer()
		
		for i := 0; i < b.N; i++ {
			dataCopy := make([]int, len(data))
			copy(dataCopy, data)
			algorithms.RadixSortLSD(dataCopy)
		}
	})
	
	b.Run("QuickSort3Way_Duplicates", func(b *testing.B) {
		data := GenerateManyDuplicatesData(size) // Best case for 3-way QuickSort
		b.ResetTimer()
		
		for i := 0; i < b.N; i++ {
			dataCopy := make([]int, len(data))
			copy(dataCopy, data)
			algorithms.QuickSort3Way(dataCopy)
		}
	})
}