package main

import (
	"flag"
	"log"
	"math/rand"
	"time"

	"exercise-02-01/algorithms"
	"exercise-02-01/analysis"

	"github.com/pkg/profile"
)

func main() {
	var (
		size       = flag.Int("size", 10000, "Data size for testing")
		algorithm  = flag.String("algorithm", "all", "Algorithm to test: sorting, searching, dp, graph, all")
		profileType = flag.String("profile", "cpu", "Profile type: cpu, mem, trace")
		analysis   = flag.Bool("analysis", false, "Run complexity analysis")
		verbose    = flag.Bool("verbose", false, "Verbose output")
	)
	flag.Parse()

	// Start profiling
	var profiler interface{ Stop() }
	switch *profileType {
	case "cpu":
		profiler = profile.Start(profile.CPUProfile, profile.ProfilePath("."))
	case "mem":
			profiler = profile.Start(profile.MemProfile, profile.ProfilePath("."))
	case "trace":
		profiler = profile.Start(profile.TraceProfile, profile.ProfilePath("."))
	default:
		log.Fatalf("Unknown profile type: %s", *profileType)
	}
	defer profiler.Stop()

	log.Printf("Starting algorithmic optimization demo")
	log.Printf("Data size: %d, Algorithm: %s, Profile: %s", *size, *algorithm, *profileType)

	// Generate test data
	rand.Seed(time.Now().UnixNano())
	data := generateTestData(*size)

	start := time.Now()

	// Run selected algorithms
	switch *algorithm {
	case "sorting":
		runSortingAlgorithms(data, *verbose)
	case "searching":
		runSearchingAlgorithms(data, *verbose)
	case "dp":
		runDynamicProgramming(*size, *verbose)
	case "graph":
		runGraphAlgorithms(*size, *verbose)
	case "all":
		runSortingAlgorithms(data, *verbose)
		runSearchingAlgorithms(data, *verbose)
		runDynamicProgramming(*size/10, *verbose) // Smaller size for DP
		runGraphAlgorithms(*size/100, *verbose)   // Much smaller for graph
	default:
		log.Fatalf("Unknown algorithm: %s", *algorithm)
	}

	elapsed := time.Since(start)
	log.Printf("Total execution time: %v", elapsed)

	// Run complexity analysis if requested
	if *analysis {
		log.Println("Running complexity analysis...")
		runComplexityAnalysis(*size)
	}
}

func generateTestData(size int) []int {
	data := make([]int, size)
	for i := 0; i < size; i++ {
		data[i] = rand.Intn(size * 10)
	}
	return data
}

func runSortingAlgorithms(data []int, verbose bool) {
	if verbose {
		log.Println("Testing sorting algorithms...")
	}

	// Test different sorting algorithms
	testData := make([]int, len(data))

	// Bubble Sort (O(n²))
	copy(testData, data)
	start := time.Now()
	algorithms.BubbleSort(testData)
	bubbleTime := time.Since(start)
	if verbose {
		log.Printf("Bubble Sort: %v", bubbleTime)
	}

	// Quick Sort (O(n log n) average)
	copy(testData, data)
	start = time.Now()
	algorithms.QuickSort(testData)
	quickTime := time.Since(start)
	if verbose {
		log.Printf("Quick Sort: %v", quickTime)
	}

	// Merge Sort (O(n log n) guaranteed)
	copy(testData, data)
	start = time.Now()
	algorithms.MergeSort(testData)
	mergeTime := time.Since(start)
	if verbose {
		log.Printf("Merge Sort: %v", mergeTime)
	}

	// Heap Sort (O(n log n))
	copy(testData, data)
	start = time.Now()
	algorithms.HeapSort(testData)
	heapTime := time.Since(start)
	if verbose {
		log.Printf("Heap Sort: %v", heapTime)
	}

	// Radix Sort (O(d*n) for integers)
	copy(testData, data)
	start = time.Now()
	algorithms.RadixSort(testData)
	radixTime := time.Since(start)
	if verbose {
		log.Printf("Radix Sort: %v", radixTime)
	}

	log.Printf("Sorting performance - Bubble: %v, Quick: %v, Merge: %v, Heap: %v, Radix: %v",
		bubbleTime, quickTime, mergeTime, heapTime, radixTime)
}

func runSearchingAlgorithms(data []int, verbose bool) {
	if verbose {
		log.Println("Testing searching algorithms...")
	}

	// Sort data for binary search
	sortedData := make([]int, len(data))
	copy(sortedData, data)
	algorithms.QuickSort(sortedData)

	target := sortedData[len(sortedData)/2] // Middle element

	// Linear Search (O(n))
	start := time.Now()
	linearResult := algorithms.LinearSearch(data, target)
	linearTime := time.Since(start)
	if verbose {
		log.Printf("Linear Search: %v, found: %t", linearTime, linearResult)
	}

	// Binary Search (O(log n))
	start = time.Now()
	binaryResult := algorithms.BinarySearch(sortedData, target)
	binaryTime := time.Since(start)
	if verbose {
		log.Printf("Binary Search: %v, found: %t", binaryTime, binaryResult)
	}

	// Interpolation Search (O(log log n) for uniform distribution)
	start = time.Now()
	interpolationResult := algorithms.InterpolationSearch(sortedData, target)
	interpolationTime := time.Since(start)
	if verbose {
		log.Printf("Interpolation Search: %v, found: %t", interpolationTime, interpolationResult)
	}

	// Jump Search (O(√n))
	start = time.Now()
	jumpResult := algorithms.JumpSearch(sortedData, target)
	jumpTime := time.Since(start)
	if verbose {
		log.Printf("Jump Search: %v, found: %t", jumpTime, jumpResult)
	}

	log.Printf("Search performance - Linear: %v, Binary: %v, Interpolation: %v, Jump: %v",
		linearTime, binaryTime, interpolationTime, jumpTime)
}

func runDynamicProgramming(size int, verbose bool) {
	if verbose {
		log.Println("Testing dynamic programming algorithms...")
	}

	// Fibonacci (naive vs memoized vs bottom-up)
	n := algorithms.Min(size/100, 40) // Limit to prevent excessive computation

	// Naive Fibonacci (O(2^n))
	start := time.Now()
	fibNaive := algorithms.FibonacciNaive(n)
	naiveTime := time.Since(start)
	if verbose {
		log.Printf("Fibonacci Naive(%d): %d, time: %v", n, fibNaive, naiveTime)
	}

	// Memoized Fibonacci (O(n))
	start = time.Now()
	fibMemo := algorithms.FibonacciMemoized(n)
	memoTime := time.Since(start)
	if verbose {
		log.Printf("Fibonacci Memoized(%d): %d, time: %v", n, fibMemo, memoTime)
	}

	// Bottom-up Fibonacci (O(n))
	start = time.Now()
	fibBottomUp := algorithms.FibonacciBottomUp(n)
	bottomUpTime := time.Since(start)
	if verbose {
		log.Printf("Fibonacci Bottom-up(%d): %d, time: %v", n, fibBottomUp, bottomUpTime)
	}

	// Longest Common Subsequence
	str1 := generateRandomString(algorithms.Min(size/10, 100))
	str2 := generateRandomString(algorithms.Min(size/10, 100))

	start = time.Now()
	lcsLength := algorithms.LongestCommonSubsequence(str1, str2)
	lcsTime := time.Since(start)
	if verbose {
		log.Printf("LCS length: %d, time: %v", lcsLength, lcsTime)
	}

	log.Printf("DP performance - Fib Naive: %v, Fib Memo: %v, Fib Bottom-up: %v, LCS: %v",
		naiveTime, memoTime, bottomUpTime, lcsTime)
}

func runGraphAlgorithms(size int, verbose bool) {
	if verbose {
		log.Println("Testing graph algorithms...")
	}

	// Create a random graph
	nodes := algorithms.Min(size, 1000) // Limit graph size
	graph := algorithms.GenerateRandomGraph(nodes, nodes*2)

	// Dijkstra's Algorithm (O((V + E) log V))
	start := time.Now()
	distances := algorithms.Dijkstra(graph, 0)
	dijkstraTime := time.Since(start)
	if verbose {
		log.Printf("Dijkstra completed, time: %v", dijkstraTime)
	}

	// Floyd-Warshall Algorithm (O(V³))
	smallGraph := algorithms.GenerateRandomGraph(algorithms.Min(nodes, 100), algorithms.Min(nodes, 100)*2)
	start = time.Now()
	allPairsDistances := algorithms.FloydWarshall(smallGraph)
	floydTime := time.Since(start)
	if verbose {
		log.Printf("Floyd-Warshall completed, time: %v", floydTime)
	}

	// DFS and BFS
	start = time.Now()
	dfsResult := algorithms.DepthFirstSearch(graph, 0)
	dfsTime := time.Since(start)
	if verbose {
		log.Printf("DFS visited %d nodes, time: %v", len(dfsResult), dfsTime)
	}

	start = time.Now()
	bfsResult := algorithms.BreadthFirstSearch(graph, 0)
	bfsTime := time.Since(start)
	if verbose {
		log.Printf("BFS visited %d nodes, time: %v", len(bfsResult), bfsTime)
	}

	log.Printf("Graph performance - Dijkstra: %v, Floyd-Warshall: %v, DFS: %v, BFS: %v",
		dijkstraTime, floydTime, dfsTime, bfsTime)

	_ = distances
	_ = allPairsDistances
}

func runComplexityAnalysis(maxSize int) {
	log.Println("Running complexity analysis...")

	// Analyze sorting algorithms
	analysis.AnalyzeSortingComplexity(maxSize)

	// Analyze searching algorithms
	analysis.AnalyzeSearchingComplexity(maxSize)

	// Generate performance predictions
	analysis.GeneratePerformancePredictions(maxSize)
}

func generateRandomString(length int) string {
	const charset = "abcdefghijklmnopqrstuvwxyz"
	b := make([]byte, length)
	for i := range b {
		b[i] = charset[rand.Intn(len(charset))]
	}
	return string(b)
}