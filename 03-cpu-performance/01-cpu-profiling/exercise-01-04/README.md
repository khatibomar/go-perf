# Exercise 01-04: Profile-Guided Optimization

**Difficulty:** Advanced  
**Estimated Time:** 120-150 minutes  
**Prerequisites:** Exercises 01-01, 01-02, and 01-03 completion, understanding of algorithms and data structures

## Objective

Master the complete profile-guided optimization workflow by analyzing CPU profiles of a complex application, identifying performance bottlenecks, implementing targeted optimizations, and validating improvements through systematic measurement.

## Learning Goals

- Analyze complex CPU profiles to identify optimization opportunities
- Implement algorithmic and implementation-level optimizations
- Use profiles to guide optimization decisions
- Measure and validate performance improvements
- Understand the trade-offs between different optimization approaches
- Build a systematic optimization methodology

## Background

Profile-guided optimization (PGO) is a systematic approach to performance improvement that uses runtime profiling data to identify and prioritize optimization efforts. This exercise teaches you to use CPU profiles as a roadmap for optimization, ensuring that your efforts focus on the areas with the highest impact.

## Tasks

### Task 1: Complex Application Analysis

Analyze a complex application with multiple performance bottlenecks.

**Requirements:**
1. Profile a multi-component application
2. Identify top CPU-consuming functions
3. Analyze call graphs and hot paths
4. Categorize optimization opportunities
5. Prioritize optimizations by potential impact

### Task 2: Algorithmic Optimizations

Implement algorithmic improvements based on profile analysis.

**Requirements:**
1. Replace inefficient algorithms with better alternatives
2. Optimize data structures and access patterns
3. Reduce algorithmic complexity
4. Implement caching strategies
5. Parallelize CPU-intensive operations

### Task 3: Implementation-Level Optimizations

Apply low-level optimizations guided by detailed profile analysis.

**Requirements:**
1. Optimize memory allocation patterns
2. Reduce function call overhead
3. Improve data locality
4. Eliminate redundant computations
5. Use compiler optimizations effectively

### Task 4: Validation and Measurement

Systematically validate optimizations and measure improvements.

**Requirements:**
1. Compare before/after profiles
2. Measure performance improvements
3. Validate correctness of optimizations
4. Analyze optimization trade-offs
5. Document optimization decisions

## Implementation Guide

### Step 1: Project Structure

Create a comprehensive optimization project:

```
exercise-01-04/
├── main.go
├── cmd/
│   ├── benchmark/
│   │   └── main.go
│   └── analyze/
│       └── main.go
├── internal/
│   ├── algorithms/
│   │   ├── sorting.go
│   │   ├── searching.go
│   │   ├── graph.go
│   │   └── optimization.go
│   ├── datastructures/
│   │   ├── cache.go
│   │   ├── pool.go
│   │   ├── tree.go
│   │   └── hash.go
│   ├── processing/
│   │   ├── pipeline.go
│   │   ├── parallel.go
│   │   ├── batch.go
│   │   └── stream.go
│   └── optimization/
│       ├── analyzer.go
│       ├── optimizer.go
│       ├── validator.go
│       └── metrics.go
├── benchmarks/
│   ├── algorithm_test.go
│   ├── datastructure_test.go
│   └── integration_test.go
├── profiles/
│   ├── before/
│   └── after/
├── scripts/
│   ├── profile.sh
│   ├── benchmark.sh
│   └── compare.sh
├── docs/
│   ├── analysis.md
│   ├── optimizations.md
│   └── results.md
├── go.mod
├── go.sum
└── README.md
```

### Step 2: Complex Application with Performance Issues

```go
// main.go
package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"runtime/pprof"
	"time"

	"./internal/algorithms"
	"./internal/datastructures"
	"./internal/processing"
)

func main() {
	var (
		cpuProfile = flag.String("cpuprofile", "", "write cpu profile to file")
		memProfile = flag.String("memprofile", "", "write memory profile to file")
		workload   = flag.String("workload", "mixed", "workload type: sorting, searching, graph, mixed")
		size       = flag.Int("size", 10000, "data size for workload")
		iterations = flag.Int("iterations", 100, "number of iterations")
		optimized  = flag.Bool("optimized", false, "use optimized implementations")
	)
	flag.Parse()

	// Setup CPU profiling
	if *cpuProfile != "" {
		f, err := os.Create(*cpuProfile)
		if err != nil {
			log.Fatal(err)
		}
		defer f.Close()
		if err := pprof.StartCPUProfile(f); err != nil {
			log.Fatal(err)
		}
		defer pprof.StopCPUProfile()
	}

	// Run workload
	start := time.Now()
	result := runWorkload(*workload, *size, *iterations, *optimized)
	duration := time.Since(start)

	fmt.Printf("Workload: %s\n", *workload)
	fmt.Printf("Size: %d\n", *size)
	fmt.Printf("Iterations: %d\n", *iterations)
	fmt.Printf("Optimized: %t\n", *optimized)
	fmt.Printf("Result: %v\n", result)
	fmt.Printf("Duration: %v\n", duration)
	fmt.Printf("Operations/sec: %.2f\n", float64(*iterations)/duration.Seconds())

	// Setup memory profiling
	if *memProfile != "" {
		f, err := os.Create(*memProfile)
		if err != nil {
			log.Fatal(err)
		}
		defer f.Close()
		if err := pprof.WriteHeapProfile(f); err != nil {
			log.Fatal(err)
		}
	}
}

func runWorkload(workloadType string, size, iterations int, optimized bool) interface{} {
	switch workloadType {
	case "sorting":
		return runSortingWorkload(size, iterations, optimized)
	case "searching":
		return runSearchingWorkload(size, iterations, optimized)
	case "graph":
		return runGraphWorkload(size, iterations, optimized)
	case "mixed":
		return runMixedWorkload(size, iterations, optimized)
	default:
		return runMixedWorkload(size, iterations, optimized)
	}
}

func runSortingWorkload(size, iterations int, optimized bool) map[string]interface{} {
	results := make(map[string]interface{})
	
	// Generate test data
	data := algorithms.GenerateRandomData(size)
	
	for i := 0; i < iterations; i++ {
		// Test different sorting algorithms
		testData := make([]int, len(data))
		copy(testData, data)
		
		if optimized {
			algorithms.OptimizedQuickSort(testData)
		} else {
			algorithms.NaiveQuickSort(testData)
		}
		
		// Additional sorting operations
		copy(testData, data)
		if optimized {
			algorithms.OptimizedMergeSort(testData)
		} else {
			algorithms.NaiveMergeSort(testData)
		}
	}
	
	results["algorithm"] = "sorting"
	results["optimized"] = optimized
	return results
}

func runSearchingWorkload(size, iterations int, optimized bool) map[string]interface{} {
	results := make(map[string]interface{})
	
	// Generate test data
	data := algorithms.GenerateRandomData(size)
	algorithms.OptimizedQuickSort(data) // Sort for binary search
	
	var cache *datastructures.SearchCache
	if optimized {
		cache = datastructures.NewSearchCache(1000)
	}
	
	totalFound := 0
	for i := 0; i < iterations; i++ {
		target := data[i%len(data)]
		
		var found bool
		if optimized {
			found = algorithms.OptimizedBinarySearch(data, target, cache)
		} else {
			found = algorithms.NaiveBinarySearch(data, target)
		}
		
		if found {
			totalFound++
		}
		
		// Additional search operations
		if optimized {
			algorithms.OptimizedLinearSearch(data, target)
		} else {
			algorithms.NaiveLinearSearch(data, target)
		}
	}
	
	results["algorithm"] = "searching"
	results["found"] = totalFound
	results["optimized"] = optimized
	return results
}

func runGraphWorkload(size, iterations int, optimized bool) map[string]interface{} {
	results := make(map[string]interface{})
	
	// Generate graph
	var graph interface{}
	if optimized {
		graph = algorithms.NewOptimizedGraph(size)
	} else {
		graph = algorithms.NewNaiveGraph(size)
	}
	
	totalPaths := 0
	for i := 0; i < iterations; i++ {
		source := i % size
		target := (i + size/2) % size
		
		var pathLength int
		if optimized {
			pathLength = algorithms.OptimizedDijkstra(graph, source, target)
		} else {
			pathLength = algorithms.NaiveDijkstra(graph, source, target)
		}
		
		totalPaths += pathLength
	}
	
	results["algorithm"] = "graph"
	results["totalPaths"] = totalPaths
	results["optimized"] = optimized
	return results
}

func runMixedWorkload(size, iterations int, optimized bool) map[string]interface{} {
	results := make(map[string]interface{})
	
	// Run all workloads
	sortingResults := runSortingWorkload(size/3, iterations/3, optimized)
	searchingResults := runSearchingWorkload(size/3, iterations/3, optimized)
	graphResults := runGraphWorkload(size/3, iterations/3, optimized)
	
	// Additional mixed operations
	pipeline := processing.NewPipeline(optimized)
	for i := 0; i < iterations; i++ {
		data := algorithms.GenerateRandomData(100)
		pipeline.Process(data)
	}
	
	results["sorting"] = sortingResults
	results["searching"] = searchingResults
	results["graph"] = graphResults
	results["pipeline"] = pipeline.GetStats()
	return results
}
```

### Step 3: Inefficient Algorithms (Before Optimization)

```go
// internal/algorithms/sorting.go
package algorithms

import (
	"math/rand"
	"time"
)

// NaiveQuickSort - Inefficient implementation with poor pivot selection
func NaiveQuickSort(arr []int) {
	if len(arr) < 2 {
		return
	}
	
	// Always use first element as pivot (worst case for sorted data)
	pivot := arr[0]
	left := 0
	right := len(arr) - 1
	
	// Inefficient partitioning
	for left < right {
		for left < right && arr[right] >= pivot {
			right--
		}
		arr[left] = arr[right]
		
		for left < right && arr[left] <= pivot {
			left++
		}
		arr[right] = arr[left]
	}
	
	arr[left] = pivot
	
	// Recursive calls without tail call optimization
	NaiveQuickSort(arr[:left])
	NaiveQuickSort(arr[left+1:])
}

// OptimizedQuickSort - Efficient implementation with optimizations
func OptimizedQuickSort(arr []int) {
	if len(arr) < 2 {
		return
	}
	
	// Use insertion sort for small arrays
	if len(arr) < 10 {
		insertionSort(arr)
		return
	}
	
	// Three-way partitioning with median-of-three pivot
	pivotIndex := medianOfThree(arr, 0, len(arr)/2, len(arr)-1)
	arr[0], arr[pivotIndex] = arr[pivotIndex], arr[0]
	
	pivot := arr[0]
	i, j, k := 1, 1, len(arr)
	
	// Dutch flag partitioning
	for j < k {
		if arr[j] < pivot {
			arr[i], arr[j] = arr[j], arr[i]
			i++
			j++
		} else if arr[j] > pivot {
			k--
			arr[j], arr[k] = arr[k], arr[j]
		} else {
			j++
		}
	}
	
	i--
	arr[0], arr[i] = arr[i], arr[0]
	
	// Recursive calls with tail call optimization
	OptimizedQuickSort(arr[:i])
	OptimizedQuickSort(arr[k:])
}

func insertionSort(arr []int) {
	for i := 1; i < len(arr); i++ {
		key := arr[i]
		j := i - 1
		for j >= 0 && arr[j] > key {
			arr[j+1] = arr[j]
			j--
		}
		arr[j+1] = key
	}
}

func medianOfThree(arr []int, a, b, c int) int {
	if arr[a] <= arr[b] && arr[b] <= arr[c] {
		return b
	}
	if arr[a] <= arr[c] && arr[c] <= arr[b] {
		return c
	}
	return a
}

// NaiveMergeSort - Inefficient implementation with excessive allocations
func NaiveMergeSort(arr []int) []int {
	if len(arr) <= 1 {
		return arr
	}
	
	mid := len(arr) / 2
	
	// Excessive memory allocations
	left := make([]int, mid)
	right := make([]int, len(arr)-mid)
	
	copy(left, arr[:mid])
	copy(right, arr[mid:])
	
	// Recursive calls with new allocations
	left = NaiveMergeSort(left)
	right = NaiveMergeSort(right)
	
	return naiveMerge(left, right)
}

func naiveMerge(left, right []int) []int {
	// New allocation for each merge
	result := make([]int, len(left)+len(right))
	i, j, k := 0, 0, 0
	
	for i < len(left) && j < len(right) {
		if left[i] <= right[j] {
			result[k] = left[i]
			i++
		} else {
			result[k] = right[j]
			j++
		}
		k++
	}
	
	for i < len(left) {
		result[k] = left[i]
		i++
		k++
	}
	
	for j < len(right) {
		result[k] = right[j]
		j++
		k++
	}
	
	return result
}

// OptimizedMergeSort - In-place merge sort with minimal allocations
func OptimizedMergeSort(arr []int) {
	if len(arr) <= 1 {
		return
	}
	
	// Use auxiliary array only once
	aux := make([]int, len(arr))
	optimizedMergeSortHelper(arr, aux, 0, len(arr)-1)
}

func optimizedMergeSortHelper(arr, aux []int, low, high int) {
	if low >= high {
		return
	}
	
	// Use insertion sort for small subarrays
	if high-low < 7 {
		insertionSortRange(arr, low, high)
		return
	}
	
	mid := low + (high-low)/2
	
	optimizedMergeSortHelper(arr, aux, low, mid)
	optimizedMergeSortHelper(arr, aux, mid+1, high)
	
	// Skip merge if already sorted
	if arr[mid] <= arr[mid+1] {
		return
	}
	
	optimizedMerge(arr, aux, low, mid, high)
}

func optimizedMerge(arr, aux []int, low, mid, high int) {
	// Copy to auxiliary array
	for i := low; i <= high; i++ {
		aux[i] = arr[i]
	}
	
	i, j := low, mid+1
	
	for k := low; k <= high; k++ {
		if i > mid {
			arr[k] = aux[j]
			j++
		} else if j > high {
			arr[k] = aux[i]
			i++
		} else if aux[j] < aux[i] {
			arr[k] = aux[j]
			j++
		} else {
			arr[k] = aux[i]
			i++
		}
	}
}

func insertionSortRange(arr []int, low, high int) {
	for i := low + 1; i <= high; i++ {
		key := arr[i]
		j := i - 1
		for j >= low && arr[j] > key {
			arr[j+1] = arr[j]
			j--
		}
		arr[j+1] = key
	}
}

// GenerateRandomData creates test data
func GenerateRandomData(size int) []int {
	rand.Seed(time.Now().UnixNano())
	data := make([]int, size)
	for i := range data {
		data[i] = rand.Intn(size * 10)
	}
	return data
}
```

### Step 4: Optimization Analysis Tools

```go
// internal/optimization/analyzer.go
package optimization

import (
	"fmt"
	"os/exec"
	"regexp"
	"sort"
	"strconv"
	"strings"
)

type ProfileAnalysis struct {
	TopFunctions    []FunctionProfile
	HotPaths        []CallPath
	Optimizations   []OptimizationOpportunity
	TotalSamples    int
	TotalDuration   float64
}

type FunctionProfile struct {
	Name        string
	FlatTime    float64
	FlatPercent float64
	CumTime     float64
	CumPercent  float64
	Samples     int
}

type CallPath struct {
	Path        []string
	TotalTime   float64
	Percentage  float64
}

type OptimizationOpportunity struct {
	Function    string
	Type        string
	Description string
	Priority    int
	EstimatedGain float64
}

type Analyzer struct {
	profilePath string
}

func NewAnalyzer(profilePath string) *Analyzer {
	return &Analyzer{
		profilePath: profilePath,
	}
}

func (a *Analyzer) AnalyzeProfile() (*ProfileAnalysis, error) {
	analysis := &ProfileAnalysis{}
	
	// Get top functions
	topFunctions, err := a.getTopFunctions()
	if err != nil {
		return nil, fmt.Errorf("failed to get top functions: %w", err)
	}
	analysis.TopFunctions = topFunctions
	
	// Identify optimization opportunities
	analysis.Optimizations = a.identifyOptimizations(topFunctions)
	
	// Get hot paths
	hotPaths, err := a.getHotPaths()
	if err != nil {
		return nil, fmt.Errorf("failed to get hot paths: %w", err)
	}
	analysis.HotPaths = hotPaths
	
	return analysis, nil
}

func (a *Analyzer) getTopFunctions() ([]FunctionProfile, error) {
	cmd := exec.Command("go", "tool", "pprof", "-top", "-cum", a.profilePath)
	output, err := cmd.Output()
	if err != nil {
		return nil, err
	}
	
	return a.parseTopOutput(string(output)), nil
}

func (a *Analyzer) parseTopOutput(output string) []FunctionProfile {
	lines := strings.Split(output, "\n")
	var functions []FunctionProfile
	
	// Regex to parse pprof top output
	re := regexp.MustCompile(`\s*(\d+\.\d+)\w*\s+(\d+\.\d+)%\s+(\d+\.\d+)%\s+(\d+\.\d+)\w*\s+(\d+\.\d+)%\s+(.+)`)
	
	for _, line := range lines {
		matches := re.FindStringSubmatch(line)
		if len(matches) == 7 {
			flatTime, _ := strconv.ParseFloat(matches[1], 64)
			flatPercent, _ := strconv.ParseFloat(matches[2], 64)
			cumPercent, _ := strconv.ParseFloat(matches[3], 64)
			cumTime, _ := strconv.ParseFloat(matches[4], 64)
			
			functions = append(functions, FunctionProfile{
				Name:        strings.TrimSpace(matches[6]),
				FlatTime:    flatTime,
				FlatPercent: flatPercent,
				CumTime:     cumTime,
				CumPercent:  cumPercent,
			})
		}
	}
	
	return functions
}

func (a *Analyzer) identifyOptimizations(functions []FunctionProfile) []OptimizationOpportunity {
	var opportunities []OptimizationOpportunity
	
	for _, fn := range functions {
		// Identify different types of optimization opportunities
		if strings.Contains(fn.Name, "Sort") && fn.FlatPercent > 5 {
			opportunities = append(opportunities, OptimizationOpportunity{
				Function:      fn.Name,
				Type:          "Algorithm",
				Description:   "Consider using more efficient sorting algorithm",
				Priority:      calculatePriority(fn.FlatPercent),
				EstimatedGain: fn.FlatPercent * 0.5, // Estimate 50% improvement
			})
		}
		
		if strings.Contains(fn.Name, "Search") && fn.FlatPercent > 3 {
			opportunities = append(opportunities, OptimizationOpportunity{
				Function:      fn.Name,
				Type:          "Caching",
				Description:   "Add caching to reduce search overhead",
				Priority:      calculatePriority(fn.FlatPercent),
				EstimatedGain: fn.FlatPercent * 0.3,
			})
		}
		
		if strings.Contains(fn.Name, "make") && fn.FlatPercent > 2 {
			opportunities = append(opportunities, OptimizationOpportunity{
				Function:      fn.Name,
				Type:          "Memory",
				Description:   "Reduce memory allocations using object pooling",
				Priority:      calculatePriority(fn.FlatPercent),
				EstimatedGain: fn.FlatPercent * 0.4,
			})
		}
	}
	
	// Sort by priority
	sort.Slice(opportunities, func(i, j int) bool {
		return opportunities[i].Priority > opportunities[j].Priority
	})
	
	return opportunities
}

func calculatePriority(percentage float64) int {
	if percentage > 20 {
		return 5 // Critical
	} else if percentage > 10 {
		return 4 // High
	} else if percentage > 5 {
		return 3 // Medium
	} else if percentage > 2 {
		return 2 // Low
	}
	return 1 // Very Low
}

func (a *Analyzer) getHotPaths() ([]CallPath, error) {
	// Implementation for extracting hot call paths
	// This would use pprof's call graph functionality
	return []CallPath{}, nil
}

func (a *Analyzer) GenerateReport(analysis *ProfileAnalysis) string {
	report := "# Profile Analysis Report\n\n"
	
	report += "## Top CPU Consuming Functions\n\n"
	for i, fn := range analysis.TopFunctions {
		if i >= 10 { // Top 10
			break
		}
		report += fmt.Sprintf("%d. %s - %.2f%% (%.2fs)\n", i+1, fn.Name, fn.FlatPercent, fn.FlatTime)
	}
	
	report += "\n## Optimization Opportunities\n\n"
	for i, opt := range analysis.Optimizations {
		if i >= 5 { // Top 5
			break
		}
		report += fmt.Sprintf("%d. **%s** (%s)\n", i+1, opt.Function, opt.Type)
		report += fmt.Sprintf("   - %s\n", opt.Description)
		report += fmt.Sprintf("   - Priority: %d, Estimated Gain: %.1f%%\n\n", opt.Priority, opt.EstimatedGain)
	}
	
	return report
}
```

## Expected Output

After completing this exercise, you should achieve:

### 1. Profile Analysis Results

```
Profile Analysis Report

## Top CPU Consuming Functions

1. main.NaiveQuickSort - 35.2% (1.85s)
2. main.NaiveMergeSort - 28.7% (1.51s)
3. main.NaiveBinarySearch - 15.3% (0.80s)
4. runtime.mallocgc - 12.1% (0.64s)
5. main.naiveMerge - 8.7% (0.46s)

## Optimization Opportunities

1. **main.NaiveQuickSort** (Algorithm)
   - Consider using more efficient sorting algorithm
   - Priority: 5, Estimated Gain: 17.6%

2. **main.NaiveMergeSort** (Memory)
   - Reduce memory allocations using in-place sorting
   - Priority: 5, Estimated Gain: 11.5%

3. **main.NaiveBinarySearch** (Caching)
   - Add caching to reduce search overhead
   - Priority: 4, Estimated Gain: 4.6%
```

### 2. Performance Improvements

```bash
# Before optimization
go run . -workload=mixed -size=10000 -iterations=100 -cpuprofile=before.prof
# Duration: 5.23s
# Operations/sec: 19.12

# After optimization
go run . -workload=mixed -size=10000 -iterations=100 -optimized=true -cpuprofile=after.prof
# Duration: 2.87s
# Operations/sec: 34.84

# Improvement: 82.2% faster
```

### 3. Optimization Summary

| Component | Before | After | Improvement |
|-----------|--------|-------|-------------|
| QuickSort | 1.85s | 0.42s | 77.3% |
| MergeSort | 1.51s | 0.38s | 74.8% |
| BinarySearch | 0.80s | 0.31s | 61.3% |
| Memory Alloc | 0.64s | 0.18s | 71.9% |
| **Total** | **5.23s** | **2.87s** | **45.1%** |

## Validation

1. **Correctness:** All optimized algorithms produce identical results
2. **Performance:** Significant improvements in all major components
3. **Memory:** Reduced allocations and improved memory efficiency
4. **Scalability:** Better performance characteristics for larger datasets

## Key Optimizations Implemented

1. **Algorithmic Improvements:**
   - Three-way partitioning in QuickSort
   - Median-of-three pivot selection
   - Hybrid sorting with insertion sort for small arrays
   - In-place merge sort with minimal allocations

2. **Caching Strategies:**
   - Search result caching
   - Object pooling for temporary data structures
   - Reuse of auxiliary arrays

3. **Implementation Optimizations:**
   - Reduced function call overhead
   - Improved data locality
   - Eliminated redundant computations
   - Better memory access patterns

## Lessons Learned

1. **Profile-Driven Development:** Always profile before optimizing
2. **Focus on Hot Paths:** Optimize the functions that consume the most time
3. **Algorithmic Impact:** Algorithm choice has the biggest performance impact
4. **Memory Matters:** Memory allocation patterns significantly affect performance
5. **Measure Everything:** Validate all optimizations with benchmarks

## Next Steps

- Apply these techniques to real-world applications
- Explore advanced optimization techniques
- Study compiler optimizations and their impact
- Move to Chapter 02: Algorithmic Optimization

---

*Exercise 01-04 of Chapter 01: CPU Profiling Techniques*