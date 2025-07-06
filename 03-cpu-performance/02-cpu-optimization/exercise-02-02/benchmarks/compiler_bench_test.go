package benchmarks

import (
	"fmt"
	"testing"

	"exercise-02-02/compiler"
)

// Benchmark data setup
func setupBenchmarkData(size int) []int {
	data := make([]int, size)
	for i := range data {
		data[i] = i % 1000
	}
	return data
}

// Inlining Benchmarks
func BenchmarkInlining(b *testing.B) {
	sizes := []int{100, 1000, 10000}
	
	for _, size := range sizes {
		data := setupBenchmarkData(size)
		
		b.Run(fmt.Sprintf("WithInlining_%d", size), func(b *testing.B) {
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				compiler.ProcessWithInlining(data)
			}
		})
		
		b.Run(fmt.Sprintf("WithoutInlining_%d", size), func(b *testing.B) {
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				compiler.ProcessWithoutInlining(data)
			}
		})
	}
}

// Escape Analysis Benchmarks
func BenchmarkEscapeAnalysis(b *testing.B) {
	sizes := []int{100, 1000, 10000}
	
	for _, size := range sizes {
		b.Run(fmt.Sprintf("StackAllocation_%d", size), func(b *testing.B) {
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				compiler.StackAllocation(size)
			}
		})
		
		b.Run(fmt.Sprintf("HeapAllocation_%d", size), func(b *testing.B) {
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				compiler.HeapAllocation(size)
			}
		})
	}
}

// Dead Code Elimination Benchmarks
func BenchmarkDeadCodeElimination(b *testing.B) {
	sizes := []int{100, 1000, 10000}
	
	for _, size := range sizes {
		data := setupBenchmarkData(size)
		
		b.Run(fmt.Sprintf("WithDeadCode_%d", size), func(b *testing.B) {
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				compiler.ProcessWithDeadCode(data)
			}
		})
		
		b.Run(fmt.Sprintf("WithoutDeadCode_%d", size), func(b *testing.B) {
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				compiler.ProcessWithoutDeadCode(data)
			}
		})
	}
}

// Build Flags Optimization Benchmarks
func BenchmarkBuildFlags(b *testing.B) {
	sizes := []int{100, 1000, 10000}
	
	for _, size := range sizes {
		data := setupBenchmarkData(size)
		
		b.Run(fmt.Sprintf("Optimized_%d", size), func(b *testing.B) {
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				compiler.OptimizedFunction(data)
			}
		})
		
		b.Run(fmt.Sprintf("Unoptimized_%d", size), func(b *testing.B) {
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				compiler.UnoptimizedFunction(data)
			}
		})
	}
}

// Function Call Overhead Benchmarks
func BenchmarkFunctionCallOverhead(b *testing.B) {
	b.Run("DirectCall", func(b *testing.B) {
		data := setupBenchmarkData(1000)
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			compiler.ProcessDirect(data)
		}
	})
	
	b.Run("InterfaceCall", func(b *testing.B) {
		data := setupBenchmarkData(1000)
		calc := compiler.SimpleCalculator{}
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			compiler.ProcessWithInterface(data, calc)
		}
	})
	
	b.Run("WithDefer", func(b *testing.B) {
		data := setupBenchmarkData(1000)
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			compiler.ProcessWithDefer(data)
		}
	})
	
	b.Run("WithoutDefer", func(b *testing.B) {
		data := setupBenchmarkData(1000)
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			compiler.ProcessWithoutDefer(data)
		}
	})
}

// Hot vs Cold Path Benchmarks
func BenchmarkHotColdPath(b *testing.B) {
	b.Run("HotPath", func(b *testing.B) {
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			compiler.HotPathCalculation(i, i+1)
		}
	})
	
	b.Run("ColdPath", func(b *testing.B) {
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			compiler.ColdPathCalculation(i%10, (i+1)%10)
		}
	})
}

// Memory Allocation Pattern Benchmarks
func BenchmarkMemoryPatterns(b *testing.B) {
	b.Run("StackStruct", func(b *testing.B) {
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			compiler.CreateSmallStruct()
		}
	})
	
	b.Run("HeapStruct", func(b *testing.B) {
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			compiler.CreateSmallStructPointer()
		}
	})
	
	b.Run("LargeStruct", func(b *testing.B) {
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			compiler.CreateLargeStruct()
		}
	})
	
	b.Run("LargeStructPointer", func(b *testing.B) {
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			compiler.CreateLargeStructPointer()
		}
	})
}

// Constant Folding and Dead Code Benchmarks
func BenchmarkConstantFolding(b *testing.B) {
	data := setupBenchmarkData(1000)
	
	b.Run("ConstantFolding", func(b *testing.B) {
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			compiler.ConstantFolding(data)
		}
	})
	
	b.Run("DeadMapOperations", func(b *testing.B) {
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			compiler.DeadMapOperations(data)
		}
	})
	
	b.Run("DeadSliceOperations", func(b *testing.B) {
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			compiler.DeadSliceOperations(data)
		}
	})
}

// Pure Function vs Side Effect Benchmarks
func BenchmarkPureFunctions(b *testing.B) {
	data := setupBenchmarkData(1000)
	
	b.Run("DeadPureFunctionCall", func(b *testing.B) {
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			compiler.DeadPureFunctionCall(data)
		}
	})
	
	// Note: DeadCodeWithSideEffects benchmark would produce output
	// so we skip it in automated benchmarks
}

// Recursion Optimization Benchmarks
func BenchmarkRecursion(b *testing.B) {
	b.Run("DeadRecursion", func(b *testing.B) {
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			compiler.DeadRecursion(10)
		}
	})
	
	b.Run("OptimizedRecursion", func(b *testing.B) {
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			compiler.OptimizedRecursion(10)
		}
	})
}

// Comprehensive Compiler Optimization Benchmark
func BenchmarkCompilerOptimizations(b *testing.B) {
	data := setupBenchmarkData(10000)
	
	b.Run("AllOptimizations", func(b *testing.B) {
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			// Combine multiple optimizations
			result1 := compiler.ProcessWithInlining(data)
			result2 := compiler.StackAllocation(len(data))
			result3 := compiler.ProcessWithoutDeadCode(data)
			result4 := compiler.OptimizedFunction(data)
			
			// Use results to prevent elimination
			_ = result1 + result2 + result3 + result4
		}
		
		b.ReportMetric(float64(len(data)), "elements/op")
	})
	
	b.Run("NoOptimizations", func(b *testing.B) {
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			// Use unoptimized versions
			result1 := compiler.ProcessWithoutInlining(data)
			result2 := compiler.HeapAllocation(len(data))
			result3 := compiler.ProcessWithDeadCode(data)
			result4 := compiler.UnoptimizedFunction(data)
			
			// Use results to prevent elimination
			_ = result1 + result2 + result3 + result4
		}
		
		b.ReportMetric(float64(len(data)), "elements/op")
	})
}

// Memory allocation benchmarks with detailed metrics
func BenchmarkMemoryAllocation(b *testing.B) {
	b.Run("StackVsHeap", func(b *testing.B) {
		b.ReportAllocs()
		b.ResetTimer()
		
		for i := 0; i < b.N; i++ {
			// Mix of stack and heap allocations
			stackResult := compiler.StackAllocation(100)
			heapResult := compiler.HeapAllocation(100)
			_ = stackResult + heapResult
		}
	})
}

// Scaling analysis benchmark
func BenchmarkScalingAnalysis(b *testing.B) {
	sizes := []int{10, 100, 1000, 10000, 100000}
	
	for _, size := range sizes {
		data := setupBenchmarkData(size)
		
		b.Run(fmt.Sprintf("OptimizedScaling_%d", size), func(b *testing.B) {
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				compiler.OptimizedFunction(data)
			}
			b.ReportMetric(float64(size), "elements")
			b.ReportMetric(float64(b.Elapsed().Nanoseconds())/float64(b.N*size), "ns/element")
		})
	}
}