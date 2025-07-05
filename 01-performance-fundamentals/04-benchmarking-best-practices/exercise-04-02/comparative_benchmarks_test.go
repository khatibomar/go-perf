package main

import (
	"fmt"
	"math/rand"
	"testing"
	"time"
)

// Benchmark data sizes for parametric testing
var benchmarkSizes = []int{10, 100, 1000, 10000}
var concurrencyLevels = []int{1, 2, 4, 8, 16}

// String Concatenation Benchmarks

func BenchmarkStringConcatenation(b *testing.B) {
	sc := &StringConcatenation{}
	
	for _, size := range []int{10, 50, 100, 500} {
		parts := GenerateRandomStrings(size, 10)
		
		b.Run(fmt.Sprintf("Naive/size_%d", size), func(b *testing.B) {
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				sc.NaiveConcat(parts)
			}
		})
		
		b.Run(fmt.Sprintf("Builder/size_%d", size), func(b *testing.B) {
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				sc.BuilderConcat(parts)
			}
		})
		
		b.Run(fmt.Sprintf("PreallocatedBuilder/size_%d", size), func(b *testing.B) {
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				sc.PreallocatedBuilderConcat(parts)
			}
		})
		
		b.Run(fmt.Sprintf("Join/size_%d", size), func(b *testing.B) {
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				sc.JoinConcat(parts)
			}
		})
		
		b.Run(fmt.Sprintf("Slice/size_%d", size), func(b *testing.B) {
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				sc.SliceConcat(parts)
			}
		})
	}
}

// Sorting Algorithm Benchmarks

func BenchmarkSortingAlgorithms(b *testing.B) {
	sa := &SortingAlgorithms{}
	
	for _, size := range []int{100, 500, 1000, 5000} {
		// Generate test data once
		originalData := GenerateRandomInts(size, size*10)
		
		b.Run(fmt.Sprintf("BubbleSort/size_%d", size), func(b *testing.B) {
			if size > 1000 {
				b.Skip("BubbleSort too slow for large datasets")
			}
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				b.StopTimer()
				data := CopySlice(originalData)
				b.StartTimer()
				sa.BubbleSort(data)
			}
		})
		
		b.Run(fmt.Sprintf("QuickSort/size_%d", size), func(b *testing.B) {
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				b.StopTimer()
				data := CopySlice(originalData)
				b.StartTimer()
				sa.QuickSort(data)
			}
		})
		
		b.Run(fmt.Sprintf("MergeSort/size_%d", size), func(b *testing.B) {
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				b.StopTimer()
				data := CopySlice(originalData)
				b.StartTimer()
				sa.MergeSort(data)
			}
		})
		
		b.Run(fmt.Sprintf("HeapSort/size_%d", size), func(b *testing.B) {
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				b.StopTimer()
				data := CopySlice(originalData)
				b.StartTimer()
				sa.HeapSort(data)
			}
		})
		
		b.Run(fmt.Sprintf("StandardSort/size_%d", size), func(b *testing.B) {
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				b.StopTimer()
				data := CopySlice(originalData)
				b.StartTimer()
				sa.StandardSort(data)
			}
		})
	}
}

// Map Implementation Benchmarks

func BenchmarkMapImplementations(b *testing.B) {
	mi := &MapImplementations{}
	
	for _, ops := range []int{1000, 5000, 10000} {
		for _, concurrency := range []int{1, 2, 4, 8} {
			b.Run(fmt.Sprintf("SyncMap/ops_%d/conc_%d", ops, concurrency), func(b *testing.B) {
				b.ResetTimer()
				for i := 0; i < b.N; i++ {
					mi.SyncMapAccess(ops, concurrency)
				}
			})
			
			b.Run(fmt.Sprintf("MutexMap/ops_%d/conc_%d", ops, concurrency), func(b *testing.B) {
				b.ResetTimer()
				for i := 0; i < b.N; i++ {
					mi.MutexMapAccess(ops, concurrency)
				}
			})
			
			b.Run(fmt.Sprintf("ChannelMap/ops_%d/conc_%d", ops, concurrency), func(b *testing.B) {
				b.ResetTimer()
				for i := 0; i < b.N; i++ {
					mi.ChannelMapAccess(ops, concurrency)
				}
			})
		}
	}
}

// Slice Operations Benchmarks

func BenchmarkSliceOperations(b *testing.B) {
	so := &SliceOperations{}
	
	for _, size := range []int{100, 1000, 10000, 100000} {
		b.Run(fmt.Sprintf("AppendGrowth/size_%d", size), func(b *testing.B) {
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				so.AppendGrowth(size)
			}
		})
		
		b.Run(fmt.Sprintf("PreallocatedSlice/size_%d", size), func(b *testing.B) {
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				so.PreallocatedSlice(size)
			}
		})
		
		b.Run(fmt.Sprintf("DirectIndexSlice/size_%d", size), func(b *testing.B) {
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				so.DirectIndexSlice(size)
			}
		})
	}
	
	// Slice copying benchmarks
	for _, size := range []int{100, 1000, 10000} {
		source := GenerateRandomInts(size, 1000)
		
		b.Run(fmt.Sprintf("CopySlice/size_%d", size), func(b *testing.B) {
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				so.CopySlice(source)
			}
		})
		
		b.Run(fmt.Sprintf("AppendSlice/size_%d", size), func(b *testing.B) {
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				so.AppendSlice(source)
			}
		})
	}
}

// Memory Pooling Benchmarks

func BenchmarkMemoryPooling(b *testing.B) {
	mp := NewMemoryPooling()
	
	for _, ops := range []int{100, 1000, 10000} {
		b.Run(fmt.Sprintf("WithoutPooling/ops_%d", ops), func(b *testing.B) {
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				mp.WithoutPooling(ops)
			}
		})
		
		b.Run(fmt.Sprintf("WithPooling/ops_%d", ops), func(b *testing.B) {
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				mp.WithPooling(ops)
			}
		})
	}
}

// Concurrency Pattern Benchmarks

func BenchmarkConcurrencyPatterns(b *testing.B) {
	cp := &ConcurrencyPatterns{}
	
	for _, size := range []int{50, 100, 200} {
		items := GenerateRandomInts(size, 1000)
		
		b.Run(fmt.Sprintf("Sequential/size_%d", size), func(b *testing.B) {
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				cp.SequentialProcessing(items)
			}
		})
		
		for _, workers := range []int{2, 4, 8} {
			b.Run(fmt.Sprintf("WorkerPool/size_%d/workers_%d", size, workers), func(b *testing.B) {
				b.ResetTimer()
				for i := 0; i < b.N; i++ {
					cp.WorkerPoolProcessing(items, workers)
				}
			})
		}
		
		b.Run(fmt.Sprintf("GoroutinePerItem/size_%d", size), func(b *testing.B) {
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				cp.GoroutinePerItemProcessing(items)
			}
		})
		
		for _, stages := range []int{2, 3, 4} {
			b.Run(fmt.Sprintf("Pipeline/size_%d/stages_%d", size, stages), func(b *testing.B) {
				b.ResetTimer()
				for i := 0; i < b.N; i++ {
					cp.PipelineProcessing(items, stages)
				}
			})
		}
	}
}

// Data Structure Benchmarks

func BenchmarkDataStructures(b *testing.B) {
	ds := &DataStructures{}
	
	for _, size := range []int{100, 1000, 10000} {
		// Prepare data
		sliceData := GenerateSortedInts(size)
		mapData := make(map[int]bool)
		for _, v := range sliceData {
			mapData[v] = true
		}
		target := sliceData[size/2] // Middle element
		
		b.Run(fmt.Sprintf("SliceSearch/size_%d", size), func(b *testing.B) {
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				ds.SliceSearch(sliceData, target)
			}
		})
		
		b.Run(fmt.Sprintf("MapLookup/size_%d", size), func(b *testing.B) {
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				ds.MapLookup(mapData, target)
			}
		})
		
		b.Run(fmt.Sprintf("BinarySearch/size_%d", size), func(b *testing.B) {
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				ds.BinarySearch(sliceData, target)
			}
		})
	}
}

// Comparative Analysis Benchmarks

func BenchmarkStringConcatComparative(b *testing.B) {
	sc := &StringConcatenation{}
	parts := GenerateRandomStrings(100, 20)
	
	b.Run("Naive", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			sc.NaiveConcat(parts)
		}
	})
	
	b.Run("Builder", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			sc.BuilderConcat(parts)
		}
	})
	
	b.Run("PreallocatedBuilder", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			sc.PreallocatedBuilderConcat(parts)
		}
	})
	
	b.Run("Join", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			sc.JoinConcat(parts)
		}
	})
}

// Memory allocation benchmarks

func BenchmarkMemoryAllocations(b *testing.B) {
	b.Run("StringConcat", func(b *testing.B) {
		sc := &StringConcatenation{}
		parts := GenerateRandomStrings(50, 10)
		
		b.Run("Naive", func(b *testing.B) {
			b.ReportAllocs()
			for i := 0; i < b.N; i++ {
				sc.NaiveConcat(parts)
			}
		})
		
		b.Run("Builder", func(b *testing.B) {
			b.ReportAllocs()
			for i := 0; i < b.N; i++ {
				sc.BuilderConcat(parts)
			}
		})
		
		b.Run("PreallocatedBuilder", func(b *testing.B) {
			b.ReportAllocs()
			for i := 0; i < b.N; i++ {
				sc.PreallocatedBuilderConcat(parts)
			}
		})
	})
	
	b.Run("SliceOperations", func(b *testing.B) {
		so := &SliceOperations{}
		size := 1000
		
		b.Run("AppendGrowth", func(b *testing.B) {
			b.ReportAllocs()
			for i := 0; i < b.N; i++ {
				so.AppendGrowth(size)
			}
		})
		
		b.Run("PreallocatedSlice", func(b *testing.B) {
			b.ReportAllocs()
			for i := 0; i < b.N; i++ {
				so.PreallocatedSlice(size)
			}
		})
		
		b.Run("DirectIndexSlice", func(b *testing.B) {
			b.ReportAllocs()
			for i := 0; i < b.N; i++ {
				so.DirectIndexSlice(size)
			}
		})
	})
}

// Scaling benchmarks to test performance characteristics

func BenchmarkScaling(b *testing.B) {
	// Test how algorithms scale with input size
	sizes := []int{10, 100, 1000, 10000}
	
	b.Run("StringConcatScaling", func(b *testing.B) {
		sc := &StringConcatenation{}
		
		for _, size := range sizes {
			parts := GenerateRandomStrings(size, 10)
			
			b.Run(fmt.Sprintf("Naive_%d", size), func(b *testing.B) {
				for i := 0; i < b.N; i++ {
					sc.NaiveConcat(parts)
				}
			})
			
			b.Run(fmt.Sprintf("Builder_%d", size), func(b *testing.B) {
				for i := 0; i < b.N; i++ {
					sc.BuilderConcat(parts)
				}
			})
		}
	})
	
	b.Run("SortingScaling", func(b *testing.B) {
		sa := &SortingAlgorithms{}
		
		for _, size := range []int{100, 500, 1000} {
			data := GenerateRandomInts(size, size*10)
			
			b.Run(fmt.Sprintf("QuickSort_%d", size), func(b *testing.B) {
				for i := 0; i < b.N; i++ {
					b.StopTimer()
					testData := CopySlice(data)
					b.StartTimer()
					sa.QuickSort(testData)
				}
			})
			
			b.Run(fmt.Sprintf("StandardSort_%d", size), func(b *testing.B) {
				for i := 0; i < b.N; i++ {
					b.StopTimer()
					testData := CopySlice(data)
					b.StartTimer()
					sa.StandardSort(testData)
				}
			})
		}
	})
}

// Parallel benchmarks to test concurrency scaling

func BenchmarkParallel(b *testing.B) {
	b.Run("ConcurrentMapAccess", func(b *testing.B) {
		mi := &MapImplementations{}
		operations := 1000
		
		b.RunParallel(func(pb *testing.PB) {
			for pb.Next() {
				mi.SyncMapAccess(operations, 1)
			}
		})
	})
	
	b.Run("ConcurrentStringConcat", func(b *testing.B) {
		sc := &StringConcatenation{}
		parts := GenerateRandomStrings(50, 10)
		
		b.RunParallel(func(pb *testing.PB) {
			for pb.Next() {
				sc.BuilderConcat(parts)
			}
		})
	})
	
	b.Run("ConcurrentMemoryPooling", func(b *testing.B) {
		mp := NewMemoryPooling()
		
		b.RunParallel(func(pb *testing.PB) {
			for pb.Next() {
				mp.WithPooling(100)
			}
		})
	})
}

// Benchmark setup and teardown examples

func BenchmarkWithSetup(b *testing.B) {
	// Setup phase - not measured
	rand.Seed(time.Now().UnixNano())
	sc := &StringConcatenation{}
	parts := GenerateRandomStrings(100, 20)
	
	// Reset timer to exclude setup time
	b.ResetTimer()
	
	// Measured phase
	for i := 0; i < b.N; i++ {
		sc.BuilderConcat(parts)
	}
	
	// Cleanup phase - not measured
	b.StopTimer()
	// Cleanup code here if needed
}

// Example of custom benchmark metrics

func BenchmarkCustomMetrics(b *testing.B) {
	sc := &StringConcatenation{}
	parts := GenerateRandomStrings(100, 20)
	
	b.ReportAllocs() // Report memory allocations
	
	var totalLength int64
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		result := sc.BuilderConcat(parts)
		totalLength += int64(len(result))
	}
	
	// Report custom metrics
	b.ReportMetric(float64(totalLength)/float64(b.N), "chars/op")
	b.ReportMetric(float64(len(parts)), "parts/op")
}

// Benchmark helper functions

func init() {
	// Initialize random seed for consistent test data
	rand.Seed(42)
}

// Example benchmark that demonstrates proper error handling
func BenchmarkWithErrorHandling(b *testing.B) {
	sc := &StringConcatenation{}
	
	for i := 0; i < b.N; i++ {
		parts := GenerateRandomStrings(50, 10)
		if len(parts) == 0 {
			b.Fatal("Failed to generate test data")
		}
		
		result := sc.BuilderConcat(parts)
		if len(result) == 0 {
			b.Error("Unexpected empty result")
		}
	}
}

// Benchmark that demonstrates skipping
func BenchmarkConditionalSkip(b *testing.B) {
	if testing.Short() {
		b.Skip("Skipping benchmark in short mode")
	}
	
	sa := &SortingAlgorithms{}
	data := GenerateRandomInts(10000, 100000)
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		b.StopTimer()
		testData := CopySlice(data)
		b.StartTimer()
		sa.QuickSort(testData)
	}
}