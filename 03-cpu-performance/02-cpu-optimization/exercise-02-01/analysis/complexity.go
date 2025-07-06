package analysis

import (
	"fmt"
	"log"
	"math"
	"math/rand"
	"time"

	"exercise-02-01/algorithms"
)

// ComplexityResult represents the result of a complexity analysis
type ComplexityResult struct {
	Algorithm    string
	InputSizes   []int
	ExecutionTimes []time.Duration
	ComplexityClass string
	RSquared     float64
}

// AnalyzeSortingComplexity analyzes the time complexity of sorting algorithms
func AnalyzeSortingComplexity(maxSize int) {
	log.Println("Analyzing sorting algorithm complexity...")
	
	sizes := generateInputSizes(maxSize)
	
	// Analyze different sorting algorithms
	analyzeAlgorithm("BubbleSort", sizes, func(data []int) {
		algorithms.BubbleSort(data)
	})
	
	analyzeAlgorithm("QuickSort", sizes, func(data []int) {
		algorithms.QuickSort(data)
	})
	
	analyzeAlgorithm("MergeSort", sizes, func(data []int) {
		algorithms.MergeSort(data)
	})
	
	analyzeAlgorithm("HeapSort", sizes, func(data []int) {
		algorithms.HeapSort(data)
	})
	
	analyzeAlgorithm("RadixSort", sizes, func(data []int) {
		algorithms.RadixSort(data)
	})
}

// AnalyzeSearchingComplexity analyzes the time complexity of searching algorithms
func AnalyzeSearchingComplexity(maxSize int) {
	log.Println("Analyzing searching algorithm complexity...")
	
	sizes := generateInputSizes(maxSize)
	
	// Analyze searching algorithms
	for _, size := range sizes {
		data := generateSortedData(size)
		target := data[size/2] // Middle element
		
		// Linear Search
		start := time.Now()
		algorithms.LinearSearch(data, target)
		linearTime := time.Since(start)
		
		// Binary Search
		start = time.Now()
		algorithms.BinarySearch(data, target)
		binaryTime := time.Since(start)
		
		// Interpolation Search
		start = time.Now()
		algorithms.InterpolationSearch(data, target)
		interpolationTime := time.Since(start)
		
		log.Printf("Size: %d, Linear: %v, Binary: %v, Interpolation: %v",
			size, linearTime, binaryTime, interpolationTime)
	}
}

// GeneratePerformancePredictions generates performance predictions for different input sizes
func GeneratePerformancePredictions(maxSize int) {
	log.Println("Generating performance predictions...")
	
	// Predict performance for various complexity classes
	predictComplexity("O(1)", maxSize, func(n int) float64 { return 1 })
	predictComplexity("O(log n)", maxSize, func(n int) float64 { return math.Log2(float64(n)) })
	predictComplexity("O(n)", maxSize, func(n int) float64 { return float64(n) })
	predictComplexity("O(n log n)", maxSize, func(n int) float64 { return float64(n) * math.Log2(float64(n)) })
	predictComplexity("O(n²)", maxSize, func(n int) float64 { return float64(n * n) })
	predictComplexity("O(n³)", maxSize, func(n int) float64 { return float64(n * n * n) })
	predictComplexity("O(2^n)", maxSize, func(n int) float64 { return math.Pow(2, float64(algorithms.Min(n, 20))) }) // Limit to prevent overflow
}

// analyzeAlgorithm performs complexity analysis on a given algorithm
func analyzeAlgorithm(name string, sizes []int, algorithm func([]int)) {
	log.Printf("Analyzing %s...", name)
	
	times := make([]time.Duration, len(sizes))
	
	for i, size := range sizes {
		// Generate test data
		data := generateRandomData(size)
		
		// Measure execution time
		start := time.Now()
		algorithm(data)
		times[i] = time.Since(start)
		
		log.Printf("%s - Size: %d, Time: %v", name, size, times[i])
	}
	
	// Analyze complexity class
	complexityClass := determineComplexityClass(sizes, times)
	log.Printf("%s appears to have %s complexity", name, complexityClass)
}

// generateInputSizes creates a range of input sizes for testing
func generateInputSizes(maxSize int) []int {
	sizes := make([]int, 0)
	
	// Start with small sizes
	for i := 100; i <= 1000; i += 200 {
		if i <= maxSize {
			sizes = append(sizes, i)
		}
	}
	
	// Add larger sizes
	for i := 2000; i <= maxSize; i += 2000 {
		sizes = append(sizes, i)
	}
	
	return sizes
}

// generateRandomData creates random test data
func generateRandomData(size int) []int {
	data := make([]int, size)
	for i := 0; i < size; i++ {
		data[i] = rand.Intn(size * 10)
	}
	return data
}

// generateSortedData creates sorted test data
func generateSortedData(size int) []int {
	data := generateRandomData(size)
	algorithms.QuickSort(data)
	return data
}

// determineComplexityClass analyzes execution times to determine complexity class
func determineComplexityClass(sizes []int, times []time.Duration) string {
	if len(sizes) < 3 {
		return "Unknown (insufficient data)"
	}
	
	// Convert times to nanoseconds for analysis
	timeValues := make([]float64, len(times))
	for i, t := range times {
		timeValues[i] = float64(t.Nanoseconds())
	}
	
	// Test different complexity classes
	complexities := map[string]func(int) float64{
		"O(1)":       func(n int) float64 { return 1 },
		"O(log n)":   func(n int) float64 { return math.Log2(float64(n)) },
		"O(n)":       func(n int) float64 { return float64(n) },
		"O(n log n)": func(n int) float64 { return float64(n) * math.Log2(float64(n)) },
		"O(n²)":      func(n int) float64 { return float64(n * n) },
		"O(n³)":      func(n int) float64 { return float64(n * n * n) },
	}
	
	bestFit := "Unknown"
	bestRSquared := -1.0
	
	for name, complexityFunc := range complexities {
		rSquared := calculateRSquared(sizes, timeValues, complexityFunc)
		if rSquared > bestRSquared {
			bestRSquared = rSquared
			bestFit = name
		}
	}
	
	return fmt.Sprintf("%s (R² = %.3f)", bestFit, bestRSquared)
}

// calculateRSquared calculates the coefficient of determination (R²)
func calculateRSquared(sizes []int, times []float64, complexityFunc func(int) float64) float64 {
	if len(sizes) != len(times) {
		return -1
	}
	
	n := len(sizes)
	if n < 2 {
		return -1
	}
	
	// Calculate predicted values
	predicted := make([]float64, n)
	for i, size := range sizes {
		predicted[i] = complexityFunc(size)
	}
	
	// Normalize predicted values to match the scale of actual times
	scaleFactor := calculateScaleFactor(times, predicted)
	for i := range predicted {
		predicted[i] *= scaleFactor
	}
	
	// Calculate R²
	meanTime := mean(times)
	ssTotal := 0.0
	ssResidual := 0.0
	
	for i := 0; i < n; i++ {
		ssTotal += (times[i] - meanTime) * (times[i] - meanTime)
		ssResidual += (times[i] - predicted[i]) * (times[i] - predicted[i])
	}
	
	if ssTotal == 0 {
		return 0
	}
	
	return 1 - (ssResidual / ssTotal)
}

// calculateScaleFactor calculates the scale factor to normalize predicted values
func calculateScaleFactor(actual, predicted []float64) float64 {
	if len(actual) == 0 || len(predicted) == 0 {
		return 1
	}
	
	actualSum := 0.0
	predictedSum := 0.0
	
	for i := 0; i < len(actual); i++ {
		actualSum += actual[i]
		predictedSum += predicted[i]
	}
	
	if predictedSum == 0 {
		return 1
	}
	
	return actualSum / predictedSum
}

// mean calculates the arithmetic mean of a slice of float64
func mean(values []float64) float64 {
	if len(values) == 0 {
		return 0
	}
	
	sum := 0.0
	for _, v := range values {
		sum += v
	}
	
	return sum / float64(len(values))
}

// predictComplexity generates performance predictions for a given complexity class
func predictComplexity(name string, maxSize int, complexityFunc func(int) float64) {
	log.Printf("Performance prediction for %s:", name)
	
	sizes := []int{1000, 10000, 100000, 1000000}
	baseTime := 1.0 // Base time in microseconds
	
	for _, size := range sizes {
		if size <= maxSize {
			predictedTime := baseTime * complexityFunc(size) / complexityFunc(1000)
			log.Printf("  Size %d: ~%.2f μs", size, predictedTime)
		}
	}
}

// MemoryComplexityAnalysis analyzes memory usage patterns
type MemoryComplexityAnalysis struct {
	Algorithm   string
	InputSize   int
	MemoryUsage int64
	Complexity  string
}

// AnalyzeMemoryComplexity performs memory complexity analysis
func AnalyzeMemoryComplexity(maxSize int) {
	log.Println("Analyzing memory complexity...")
	
	sizes := generateInputSizes(maxSize)
	
	for _, size := range sizes {
		// Analyze in-place vs out-of-place algorithms
		log.Printf("Memory analysis for size %d:", size)
		
		// In-place sorting (QuickSort, HeapSort)
		log.Printf("  In-place algorithms: O(1) additional space")
		
		// Out-of-place sorting (MergeSort)
		log.Printf("  MergeSort: O(n) additional space = %d integers", size)
		
		// RadixSort
		log.Printf("  RadixSort: O(n + k) additional space = %d integers", size+10)
		
		// Dynamic Programming algorithms
		log.Printf("  DP algorithms: O(n) to O(n²) space depending on problem")
	}
}

// CacheAnalysis analyzes cache performance characteristics
type CacheAnalysis struct {
	Algorithm        string
	CacheHitRate     float64
	CacheMissRate    float64
	LocalityOfReference string
}

// AnalyzeCachePerformance analyzes cache performance patterns
func AnalyzeCachePerformance() {
	log.Println("Analyzing cache performance patterns...")
	
	// Sequential access patterns
	log.Println("Sequential access (good cache locality):")
	log.Println("  - Linear search")
	log.Println("  - Merge sort (during merge phase)")
	log.Println("  - Radix sort")
	
	// Random access patterns
	log.Println("Random access (poor cache locality):")
	log.Println("  - Quick sort (pivot partitioning)")
	log.Println("  - Heap sort (heap operations)")
	log.Println("  - Hash table operations")
	
	// Recommendations
	log.Println("Cache optimization recommendations:")
	log.Println("  - Prefer algorithms with sequential access patterns")
	log.Println("  - Use cache-oblivious algorithms when possible")
	log.Println("  - Consider data structure layout (AoS vs SoA)")
	log.Println("  - Implement cache-friendly data structures")
}

// BranchPredictionAnalysis analyzes branch prediction patterns
func BranchPredictionAnalysis() {
	log.Println("Analyzing branch prediction patterns...")
	
	log.Println("Algorithms with predictable branches:")
	log.Println("  - Linear search on sorted data")
	log.Println("  - Binary search")
	log.Println("  - Merge sort")
	
	log.Println("Algorithms with unpredictable branches:")
	log.Println("  - Quick sort (random pivot)")
	log.Println("  - Linear search on random data")
	log.Println("  - Hash table collision resolution")
	
	log.Println("Branch prediction optimization tips:")
	log.Println("  - Minimize conditional branches in inner loops")
	log.Println("  - Use branchless programming techniques")
	log.Println("  - Sort data when possible to improve predictability")
	log.Println("  - Use lookup tables instead of complex conditionals")
}