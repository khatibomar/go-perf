package benchmarks

import (
	"fmt"
	"runtime"
	"testing"
	"time"

	"exercise-02-02/compiler"
)

// TestBuildFlagsInfo tests the build flags information display
func TestBuildFlagsInfo(t *testing.T) {
	// Test that build flags info can be displayed without panic
	compiler.ShowBuildFlagsInfo()
	compiler.CompilerOptimizationLevels()
	compiler.MemoryOptimizationFlags()
	compiler.RuntimeOptimizationInfo()
	compiler.ConditionalCompilation()
	compiler.ShowPerformanceFlags()
	compiler.CompilerDirectives()
	compiler.EnvironmentVariables()
	compiler.BuildCommandExamples()
	compiler.PrintRecommendedFlags()
}

// TestOptimizationScenarios tests different optimization scenarios
func TestOptimizationScenarios(t *testing.T) {
	scenarios := compiler.GetOptimizationScenarios()
	
	if len(scenarios) == 0 {
		t.Error("Expected optimization scenarios, got none")
	}
	
	for _, scenario := range scenarios {
		if scenario.Name == "" {
			t.Error("Scenario name should not be empty")
		}
		if scenario.Description == "" {
			t.Error("Scenario description should not be empty")
		}
	}
}

// TestPerformanceFlags tests performance flags functionality
func TestPerformanceFlags(t *testing.T) {
	flags := compiler.GetPerformanceFlags()
	
	if len(flags) == 0 {
		t.Error("Expected performance flags, got none")
	}
	
	for _, flag := range flags {
		if flag.Flag == "" {
			t.Error("Flag should not be empty")
		}
		if flag.Description == "" {
			t.Error("Flag description should not be empty")
		}
	}
}

// TestRecommendedFlags tests recommended flags functionality
func TestRecommendedFlags(t *testing.T) {
	flags := compiler.GetRecommendedFlags()
	
	expectedScenarios := []string{"development", "testing", "production", "debugging", "profiling"}
	
	for _, scenario := range expectedScenarios {
		if _, exists := flags[scenario]; !exists {
			t.Errorf("Expected scenario %s not found in recommended flags", scenario)
		}
	}
}

// TestOptimizationImpact tests the impact of optimizations
func TestOptimizationImpact(t *testing.T) {
	data := make([]int, 1000)
	for i := range data {
		data[i] = i
	}
	
	// Test optimized vs unoptimized
	start := time.Now()
	optimizedResult := compiler.OptimizedFunction(data)
	optimizedTime := time.Since(start)
	
	start = time.Now()
	unoptimizedResult := compiler.UnoptimizedFunction(data)
	unoptimizedTime := time.Since(start)
	
	// Results should be similar (same algorithm)
	if optimizedResult == unoptimizedResult {
		// This is expected for simple cases
		t.Logf("Optimized and unoptimized results are identical: %d", optimizedResult)
	}
	
	t.Logf("Optimized time: %v, Unoptimized time: %v", optimizedTime, unoptimizedTime)
	
	// Test build flags analysis
	results := compiler.AnalyzeBuildFlags(data)
	if len(results) == 0 {
		t.Error("Expected build flags analysis results")
	}
}

// BenchmarkBuildFlagsComparison benchmarks different build flag scenarios
func BenchmarkBuildFlagsComparison(b *testing.B) {
	data := make([]int, 10000)
	for i := range data {
		data[i] = i % 1000
	}
	
	b.Run("OptimizedBuild", func(b *testing.B) {
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			compiler.OptimizedFunction(data)
		}
	})
	
	b.Run("UnoptimizedBuild", func(b *testing.B) {
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			compiler.UnoptimizedFunction(data)
		}
	})
}

// BenchmarkCompilerDirectiveImpact benchmarks the impact of compiler directives
func BenchmarkCompilerDirectiveImpact(b *testing.B) {
	data := make([]int, 1000)
	for i := range data {
		data[i] = i
	}
	
	b.Run("InlinedFunctions", func(b *testing.B) {
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			compiler.ProcessWithInlining(data)
		}
	})
	
	b.Run("NonInlinedFunctions", func(b *testing.B) {
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			compiler.ProcessWithoutInlining(data)
		}
	})
}

// BenchmarkMemoryAllocationFlags benchmarks memory allocation with different flags
func BenchmarkMemoryAllocationFlags(b *testing.B) {
	sizes := []int{100, 1000, 10000}
	
	for _, size := range sizes {
		b.Run(fmt.Sprintf("StackAllocation_%d", size), func(b *testing.B) {
			b.ReportAllocs()
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				compiler.StackAllocation(size)
			}
		})
		
		b.Run(fmt.Sprintf("HeapAllocation_%d", size), func(b *testing.B) {
			b.ReportAllocs()
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				compiler.HeapAllocation(size)
			}
		})
	}
}

// BenchmarkRuntimeOptimizations benchmarks runtime optimization features
func BenchmarkRuntimeOptimizations(b *testing.B) {
	b.Run("GoroutineCreation", func(b *testing.B) {
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			done := make(chan bool)
			go func() {
				done <- true
			}()
			<-done
		}
	})
	
	b.Run("ChannelOperations", func(b *testing.B) {
		ch := make(chan int, 1)
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			ch <- i
			<-ch
		}
	})
}

// BenchmarkGCImpact benchmarks garbage collection impact
func BenchmarkGCImpact(b *testing.B) {
	b.Run("WithGC", func(b *testing.B) {
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			// Allocate memory that will need GC
			data := make([]int, 1000)
			for j := range data {
				data[j] = j
			}
			_ = data
			
			// Force GC occasionally
			if i%100 == 0 {
				runtime.GC()
			}
		}
	})
	
	b.Run("StackOnly", func(b *testing.B) {
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			// Use stack allocation only
			result := compiler.StackAllocation(100)
			_ = result
		}
	})
}

// BenchmarkBuildFlagScenarios benchmarks different build flag scenarios
func BenchmarkBuildFlagScenarios(b *testing.B) {
	data := make([]int, 5000)
	for i := range data {
		data[i] = i % 100
	}
	
	// Simulate different build scenarios
	scenarios := map[string]func([]int) int{
		"Production":  compiler.OptimizedFunction,
		"Development": compiler.UnoptimizedFunction,
		"WithInlining": compiler.ProcessWithInlining,
		"NoInlining":   compiler.ProcessWithoutInlining,
	}
	
	for name, fn := range scenarios {
		b.Run(name, func(b *testing.B) {
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				fn(data)
			}
		})
	}
}

// BenchmarkCompilerOptimizationLevels benchmarks different optimization levels
func BenchmarkCompilerOptimizationLevels(b *testing.B) {
	data := make([]int, 1000)
	for i := range data {
		data[i] = i
	}
	
	// Test different optimization approaches
	b.Run("Level0_NoOptimization", func(b *testing.B) {
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			// Simulate -gcflags="-N -l" (no optimization, no inlining)
			result := compiler.ProcessWithoutInlining(data)
			result += compiler.HeapAllocation(len(data))
			result += compiler.ProcessWithDeadCode(data)
			_ = result
		}
	})
	
	b.Run("Level1_BasicOptimization", func(b *testing.B) {
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			// Simulate basic optimizations
			result := compiler.ProcessWithInlining(data)
			result += compiler.StackAllocation(len(data))
			result += compiler.ProcessWithoutDeadCode(data)
			_ = result
		}
	})
	
	b.Run("Level2_AggressiveOptimization", func(b *testing.B) {
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			// Simulate -gcflags="-l=4" (aggressive inlining)
			result := compiler.OptimizedFunction(data)
			result += compiler.StackAllocation(len(data))
			result += compiler.ProcessWithInlining(data)
			_ = result
		}
	})
}

// TestBuildEnvironment tests build environment detection
func TestBuildEnvironment(t *testing.T) {
	// Test runtime information
	if runtime.GOOS == "" {
		t.Error("GOOS should not be empty")
	}
	if runtime.GOARCH == "" {
		t.Error("GOARCH should not be empty")
	}
	if runtime.Version() == "" {
		t.Error("Go version should not be empty")
	}
	
	// Test that we can get CPU count
	if runtime.NumCPU() <= 0 {
		t.Error("CPU count should be positive")
	}
	
	t.Logf("Build environment: %s/%s, Go %s, %d CPUs", 
		runtime.GOOS, runtime.GOARCH, runtime.Version(), runtime.NumCPU())
}

// BenchmarkCrossCompilationImpact simulates cross-compilation scenarios
func BenchmarkCrossCompilationImpact(b *testing.B) {
	data := make([]int, 1000)
	for i := range data {
		data[i] = i
	}
	
	// Test native compilation performance
	b.Run("NativeCompilation", func(b *testing.B) {
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			compiler.OptimizedFunction(data)
		}
	})
	
	// Simulate different target architectures by using different algorithms
	b.Run("SimulatedCrossCompile_AMD64", func(b *testing.B) {
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			// Simulate AMD64 optimizations
			compiler.ProcessWithInlining(data)
		}
	})
	
	b.Run("SimulatedCrossCompile_ARM64", func(b *testing.B) {
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			// Simulate ARM64 characteristics (different optimization profile)
			compiler.ProcessWithoutInlining(data)
		}
	})
}