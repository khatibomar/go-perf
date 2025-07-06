package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"runtime"
	"time"

	"exercise-02-02/compiler"
)

func main() {
	var (
		optimization = flag.String("optimization", "inlining", "Optimization type: inlining, escape, deadcode, flags")
		size         = flag.Int("size", 10000, "Data size for benchmarks")
		iterations   = flag.Int("iterations", 1000, "Number of iterations")
		verbose      = flag.Bool("verbose", false, "Enable verbose output")
		analysis     = flag.Bool("analysis", false, "Show compiler analysis")
	)
	flag.Parse()

	fmt.Printf("=== Go Compiler Optimization Demo ===\n")
	fmt.Printf("Go Version: %s\n", runtime.Version())
	fmt.Printf("GOOS: %s, GOARCH: %s\n", runtime.GOOS, runtime.GOARCH)
	fmt.Printf("Optimization: %s\n", *optimization)
	fmt.Printf("Data Size: %d\n", *size)
	fmt.Printf("Iterations: %d\n\n", *iterations)

	switch *optimization {
	case "inlining":
		runInliningDemo(*size, *iterations, *verbose, *analysis)
	case "escape":
		runEscapeAnalysisDemo(*size, *iterations, *verbose, *analysis)
	case "deadcode":
		runDeadCodeDemo(*size, *iterations, *verbose, *analysis)
	case "flags":
		runBuildFlagsDemo(*size, *iterations, *verbose, *analysis)
	default:
		log.Fatalf("Unknown optimization type: %s", *optimization)
	}
}

func runInliningDemo(size, iterations int, verbose, analysis bool) {
	fmt.Println("=== Function Inlining Optimization ===")
	
	if analysis {
		fmt.Println("\n--- Compiler Analysis ---")
		compiler.ShowInliningAnalysis()
	}

	// Test inlined vs non-inlined functions
	data := make([]int, size)
	for i := range data {
		data[i] = i
	}

	// Inlined function test
	start := time.Now()
	for i := 0; i < iterations; i++ {
		result := compiler.ProcessWithInlining(data)
		if verbose && i == 0 {
			fmt.Printf("Inlined result sample: %d\n", result)
		}
	}
	inlinedTime := time.Since(start)

	// Non-inlined function test
	start = time.Now()
	for i := 0; i < iterations; i++ {
		result := compiler.ProcessWithoutInlining(data)
		if verbose && i == 0 {
			fmt.Printf("Non-inlined result sample: %d\n", result)
		}
	}
	nonInlinedTime := time.Since(start)

	fmt.Printf("\nResults:\n")
	fmt.Printf("Inlined function time: %v\n", inlinedTime)
	fmt.Printf("Non-inlined function time: %v\n", nonInlinedTime)
	fmt.Printf("Performance improvement: %.2fx\n", float64(nonInlinedTime)/float64(inlinedTime))
}

func runEscapeAnalysisDemo(size, iterations int, verbose, analysis bool) {
	fmt.Println("=== Escape Analysis Optimization ===")
	
	if analysis {
		fmt.Println("\n--- Compiler Analysis ---")
		compiler.ShowEscapeAnalysis()
	}

	// Test stack vs heap allocation
	start := time.Now()
	var stackResult int64
	for i := 0; i < iterations; i++ {
		result := compiler.StackAllocation(size)
		stackResult += int64(result)
	}
	stackTime := time.Since(start)

	start = time.Now()
	var heapResult int64
	for i := 0; i < iterations; i++ {
		result := compiler.HeapAllocation(size)
		heapResult += int64(result)
	}
	heapTime := time.Since(start)

	if verbose {
		fmt.Printf("Stack result: %d\n", stackResult)
		fmt.Printf("Heap result: %d\n", heapResult)
	}

	fmt.Printf("\nResults:\n")
	fmt.Printf("Stack allocation time: %v\n", stackTime)
	fmt.Printf("Heap allocation time: %v\n", heapTime)
	fmt.Printf("Performance improvement: %.2fx\n", float64(heapTime)/float64(stackTime))

	// Show memory stats
	var m1, m2 runtime.MemStats
	runtime.GC()
	runtime.ReadMemStats(&m1)
	
	for i := 0; i < 1000; i++ {
		compiler.HeapAllocation(size)
	}
	
	runtime.ReadMemStats(&m2)
	fmt.Printf("\nMemory Impact:\n")
	fmt.Printf("Heap allocations: %d\n", m2.Mallocs-m1.Mallocs)
	fmt.Printf("Heap bytes allocated: %d\n", m2.TotalAlloc-m1.TotalAlloc)
}

func runDeadCodeDemo(size, iterations int, verbose, analysis bool) {
	fmt.Println("=== Dead Code Elimination Optimization ===")
	
	if analysis {
		fmt.Println("\n--- Compiler Analysis ---")
		compiler.ShowDeadCodeAnalysis()
	}

	data := make([]int, size)
	for i := range data {
		data[i] = i
	}

	// Test with dead code
	start := time.Now()
	for i := 0; i < iterations; i++ {
		result := compiler.ProcessWithDeadCode(data)
		if verbose && i == 0 {
			fmt.Printf("With dead code result: %d\n", result)
		}
	}
	withDeadCodeTime := time.Since(start)

	// Test without dead code
	start = time.Now()
	for i := 0; i < iterations; i++ {
		result := compiler.ProcessWithoutDeadCode(data)
		if verbose && i == 0 {
			fmt.Printf("Without dead code result: %d\n", result)
		}
	}
	withoutDeadCodeTime := time.Since(start)

	fmt.Printf("\nResults:\n")
	fmt.Printf("With dead code time: %v\n", withDeadCodeTime)
	fmt.Printf("Without dead code time: %v\n", withoutDeadCodeTime)
	fmt.Printf("Performance improvement: %.2fx\n", float64(withDeadCodeTime)/float64(withoutDeadCodeTime))
}

func runBuildFlagsDemo(size, iterations int, verbose, analysis bool) {
	fmt.Println("=== Build Flags Optimization ===")
	
	if analysis {
		fmt.Println("\n--- Build Flags Analysis ---")
		compiler.ShowBuildFlagsInfo()
	}

	data := make([]int, size)
	for i := range data {
		data[i] = i
	}

	// Simulate different optimization levels
	start := time.Now()
	for i := 0; i < iterations; i++ {
		result := compiler.OptimizedFunction(data)
		if verbose && i == 0 {
			fmt.Printf("Optimized result: %d\n", result)
		}
	}
	optimizedTime := time.Since(start)

	start = time.Now()
	for i := 0; i < iterations; i++ {
		result := compiler.UnoptimizedFunction(data)
		if verbose && i == 0 {
			fmt.Printf("Unoptimized result: %d\n", result)
		}
	}
	unoptimizedTime := time.Since(start)

	fmt.Printf("\nResults:\n")
	fmt.Printf("Optimized build time: %v\n", optimizedTime)
	fmt.Printf("Unoptimized build time: %v\n", unoptimizedTime)
	fmt.Printf("Performance improvement: %.2fx\n", float64(unoptimizedTime)/float64(optimizedTime))

	// Show build information
	fmt.Printf("\nBuild Information:\n")
	fmt.Printf("Current build flags: %s\n", os.Getenv("GOFLAGS"))
	fmt.Printf("Recommended flags for production: -ldflags='-s -w' -gcflags='-l=4'\n")
	fmt.Printf("Debug flags: -gcflags='-N -l'\n")
}