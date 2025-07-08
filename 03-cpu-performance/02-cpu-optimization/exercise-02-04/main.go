package main

import (
	"flag"
	"fmt"
	"log"
	"math/rand"
	"os"
	"runtime"
	"strconv"
	"strings"
	"time"

	"exercise-02-04/analysis"
	"exercise-02-04/search"
)

func main() {
	// Command line flags
	analyze := flag.Bool("analyze", false, "Run performance analysis")
	report := flag.Bool("report", false, "Generate performance report")
	test := flag.Bool("test", false, "Run correctness tests")
	demo := flag.Bool("demo", false, "Run demonstration")
	format := flag.String("format", "text", "Report format (text, markdown, json, csv)")
	output := flag.String("output", "", "Output file (default: stdout)")
	verbose := flag.Bool("verbose", false, "Verbose output")
	flag.Parse()

	if *test {
		runCorrectnessTests()
	} else if *demo {
		runDemonstration()
	} else if *analyze || *report {
		runPerformanceAnalysis(*report, *format, *output, *verbose)
	} else {
		fmt.Println("Search Algorithm Optimization Exercise")
		fmt.Println("Usage:")
		fmt.Println("  -test              Run correctness tests")
		fmt.Println("  -demo              Run demonstration")
		fmt.Println("  -analyze           Run performance analysis")
		fmt.Println("  -report            Generate detailed report")
		fmt.Println("  -format <format>   Report format (text, markdown, json, csv)")
		fmt.Println("  -output <file>     Output file (default: stdout)")
		fmt.Println("  -verbose           Verbose output")
	}
}

func runCorrectnessTests() {
	fmt.Println("Running correctness tests...")
	fmt.Println("Testing linear search algorithms...")
	testLinearSearchCorrectness()
	fmt.Println("Testing binary search algorithms...")
	testBinarySearchCorrectness()
	fmt.Println("Testing hash table algorithms...")
	testHashTableCorrectness()
	fmt.Println("Testing tree algorithms...")
	testTreeCorrectness()
	fmt.Println("Testing string search algorithms...")
	testStringSearchCorrectness()
	fmt.Println("All correctness tests passed!")
}

func runDemonstration() {
	fmt.Println("Search Algorithm Demonstration")
	fmt.Println(strings.Repeat("=", 50))
	
	// Demonstrate different search algorithms
	data := []int{1, 3, 5, 7, 9, 11, 13, 15, 17, 19}
	target := 7
	
	fmt.Printf("\nSearching for %d in array: %v\n\n", target, data)
	
	// Linear Search
	start := time.Now()
	idx := search.LinearSearch(data, target)
	found := idx != -1
	duration := time.Since(start)
	fmt.Printf("Linear Search: found=%v, index=%d, time=%v\n", found, idx, duration)

	// Binary Search
	start = time.Now()
	idx = search.BinarySearchIterative(data, target)
	found = idx != -1
	duration = time.Since(start)
	fmt.Printf("Binary Search: found=%v, index=%d, time=%v\n", found, idx, duration)
	
	// Hash Table
	ht := search.NewHashTable()
	for i, val := range data {
		ht.Insert(strconv.Itoa(val), i)
	}
	start = time.Now()
	value, found := ht.Search(strconv.Itoa(target))
	duration = time.Since(start)
	fmt.Printf("Hash Table: found=%v, value=%v, time=%v\n", found, value, duration)
	
	// BST
	bst := search.NewBST()
	for _, val := range data {
		bst.Insert(val)
	}
	start = time.Now()
	found = bst.Search(target)
	duration = time.Since(start)
	fmt.Printf("BST: found=%v, time=%v\n", found, duration)
	
	// String search demonstration
	fmt.Printf("\nString Search Demonstration\n")
	fmt.Println(strings.Repeat("-", 30))
	
	words := []string{"apple", "banana", "cherry", "date", "elderberry"}
	targetWord := "cherry"
	
	fmt.Printf("Searching for '%s' in: %v\n\n", targetWord, words)
	
	// Linear string search
	start = time.Now()
	idx = search.LinearSearchString(words, targetWord)
	found = idx != -1
	duration = time.Since(start)
	fmt.Printf("Linear String Search: found=%v, index=%d, time=%v\n", found, idx, duration)
	
	// Trie
	trie := search.NewTrie()
	for i, word := range words {
		trie.Insert(word, i)
	}
	start = time.Now()
	value, found = trie.Search(targetWord)
	duration = time.Since(start)
	fmt.Printf("Trie: found=%v, value=%v, time=%v\n", found, value, duration)
	
	// Demonstrate scalability
	fmt.Printf("\nScalability Demonstration\n")
	fmt.Println(strings.Repeat("-", 30))
	
	sizes := []int{100, 1000, 10000}
	for _, size := range sizes {
		largeData := make([]int, size)
		for i := 0; i < size; i++ {
			largeData[i] = i * 2
		}
		target := size - 10
		
		// Linear search
		start = time.Now()
		search.LinearSearch(largeData, target)
		linearTime := time.Since(start)
		
		// Binary search
		start = time.Now()
		search.BinarySearchIterative(largeData, target)
		binaryTime := time.Since(start)
		
		fmt.Printf("Size %d: Linear=%v, Binary=%v, Speedup=%.2fx\n",
			size, linearTime, binaryTime, float64(linearTime)/float64(binaryTime))
	}
}

func runPerformanceAnalysis(generateReport bool, format, outputFile string, verbose bool) {
	fmt.Println("Running performance analysis...")
	
	// Create analyzer
	config := analysis.AnalysisConfig{
		EnableDetailedMetrics: true,
		EnableCacheAnalysis:   true,
		EnableBranchAnalysis:  true,
		ConfidenceLevel:       0.95,
		SignificanceThreshold: 0.05,
	}
	analyzer := analysis.NewAnalyzer(config)
	
	// Run benchmarks and collect metrics
	runBenchmarksAndAnalyze(analyzer, verbose)
	
	if generateReport {
		// Generate comprehensive report
		report := analyzer.GenerateReport()
		
		// Setup output
		var output *os.File
		if outputFile != "" {
			var err error
			output, err = os.Create(outputFile)
			if err != nil {
				log.Fatalf("Failed to create output file: %v", err)
			}
			defer output.Close()
		} else {
			output = os.Stdout
		}
		
		// Create reporter and generate report
		reportConfig := analysis.ReportConfig{
			Format:         format,
			IncludeCharts:  true,
			IncludeDetails: verbose,
			SortBy:         "performance",
			DecimalPlaces:  2,
		}
		reporter := analysis.NewReporter(output, reportConfig)
		
		if err := reporter.GenerateReport(report); err != nil {
			log.Fatalf("Failed to generate report: %v", err)
		}
		
		if outputFile != "" {
			fmt.Printf("Report generated: %s\n", outputFile)
		}
	} else {
		// Simple analysis output
		insights := analyzer.GenerateInsights()
		recommendations := analyzer.GenerateRecommendations()
		
		fmt.Println("\nKey Insights:")
		for i, insight := range insights {
			fmt.Printf("%d. %s\n", i+1, insight)
		}
		
		fmt.Println("\nRecommendations:")
		for i, rec := range recommendations {
			fmt.Printf("%d. %s\n", i+1, rec)
		}
	}
}

func init() {
	// Set GOMAXPROCS to use all available CPU cores
	runtime.GOMAXPROCS(runtime.NumCPU())
	// Seed random number generator
	rand.Seed(time.Now().UnixNano())
}

// Helper functions for correctness testing
func testLinearSearchCorrectness() {
	data := []int{1, 3, 5, 7, 9}
	target := 5
	
	// Test basic linear search
	idx := search.LinearSearch(data, target)
	if idx != 2 {
		log.Fatalf("LinearSearch failed: expected idx=2, got idx=%d", idx)
	}
	
	// Test early termination variant
	idx = search.LinearSearchEarlyTermination(data, target)
	if idx != 2 {
		log.Fatalf("LinearSearchEarlyTermination failed: expected idx=2, got idx=%d", idx)
	}
	
	fmt.Println("✅ Linear search tests passed")
}

func testBinarySearchCorrectness() {
	data := []int{1, 3, 5, 7, 9}
	target := 5
	
	// Test recursive binary search
	idx := search.BinarySearch(data, target)
	if idx != 2 {
		log.Fatalf("BinarySearch failed: expected idx=2, got idx=%d", idx)
	}
	
	// Test iterative binary search
	idx = search.BinarySearchIterative(data, target)
	if idx != 2 {
		log.Fatalf("BinarySearchIterative failed: expected idx=2, got idx=%d", idx)
	}
	
	fmt.Println("✅ Binary search tests passed")
}

func testHashTableCorrectness() {
	ht := search.NewHashTable()
	data := []int{1, 3, 5, 7, 9}
	
	for i, val := range data {
		ht.Insert(strconv.Itoa(val), i)
	}
	
	value, found := ht.Search(strconv.Itoa(5))
	if !found || value != 2 {
		log.Fatalf("HashTable failed: expected found=true, value=2, got found=%v, value=%v", found, value)
	}
	
	fmt.Println("✅ Hash table tests passed")
}

func testTreeCorrectness() {
	bst := search.NewBST()
	data := []int{5, 3, 7, 1, 9}
	
	for _, val := range data {
		bst.Insert(val)
	}
	
	found := bst.Search(7)
	if !found {
		log.Fatalf("BST failed: expected found=true, got found=%v", found)
	}
	
	fmt.Println("✅ Tree tests passed")
}

func testStringSearchCorrectness() {
	words := []string{"apple", "banana", "cherry"}
	target := "banana"
	
	idx := search.LinearSearchString(words, target)
	if idx != 1 {
		log.Fatalf("LinearSearchString failed: expected idx=1, got idx=%d", idx)
	}
	
	trie := search.NewTrie()
	for i, word := range words {
		trie.Insert(word, i)
	}
		
	value, found := trie.Search(target)
	if !found || value != 1 {
		log.Fatalf("Trie failed: expected found=true, value=1, got found=%v, value=%v", found, value)
	}
	
	fmt.Println("✅ String search tests passed")
}

func runBenchmarksAndAnalyze(analyzer *analysis.Analyzer, verbose bool) {
	if verbose {
		fmt.Println("Running comprehensive benchmarks...")
	}
	
	// Test different data sizes
	sizes := []int{1000, 10000, 100000}
	
	for _, size := range sizes {
		if verbose {
			fmt.Printf("Testing with data size: %d\n", size)
		}
		
		// Generate test data
		data := make([]int, size)
		for i := 0; i < size; i++ {
			data[i] = i
		}
		
		// Shuffle for unsorted tests
		unsortedData := make([]int, size)
		copy(unsortedData, data)
		rand.Shuffle(len(unsortedData), func(i, j int) {
			unsortedData[i], unsortedData[j] = unsortedData[j], unsortedData[i]
		})
		
		// Test targets
		targets := []int{size / 4, size / 2, size * 3 / 4}
		
		// Benchmark linear search algorithms
		benchmarkAlgorithm(analyzer, "LinearSearch", size, func() {
			for _, target := range targets {
				search.LinearSearch(unsortedData, target)
			}
		})
		
		benchmarkAlgorithm(analyzer, "LinearSearchUnrolled", size, func() {
			for _, target := range targets {
				search.LinearSearchUnrolled(unsortedData, target)
			}
		})
		
		// Benchmark binary search algorithms
		benchmarkAlgorithm(analyzer, "BinarySearchIterative", size, func() {
			for _, target := range targets {
				search.BinarySearchIterative(data, target)
			}
		})
		
		benchmarkAlgorithm(analyzer, "BinarySearchBranchless", size, func() {
			for _, target := range targets {
				search.BinarySearchBranchless(data, target)
			}
		})
		
		benchmarkAlgorithm(analyzer, "InterpolationSearch", size, func() {
			for _, target := range targets {
				search.InterpolationSearch(data, target)
			}
		})
		
		// Benchmark hash table
		ht := search.NewHashTable()
		for _, val := range data {
			ht.Insert(strconv.Itoa(val), val)
		}
		benchmarkAlgorithm(analyzer, "HashTable", size, func() {
			for _, target := range targets {
				ht.Search(strconv.Itoa(target))
			}
		})
	}
	
	if verbose {
		fmt.Println("Benchmarks completed")
	}
}

func benchmarkAlgorithm(analyzer *analysis.Analyzer, algorithm string, dataSize int, benchFunc func()) {
	iterations := 1000
	timings := make([]time.Duration, iterations)
	
	// Warmup
	for i := 0; i < 10; i++ {
		benchFunc()
	}
	
	// Actual benchmark
	for i := 0; i < iterations; i++ {
		start := time.Now()
		benchFunc()
		timings[i] = time.Since(start)
	}
	
	// Calculate and add metrics
	metrics := analyzer.CalculateStatistics(algorithm, dataSize, timings)
	analyzer.AddMetrics(metrics)
}