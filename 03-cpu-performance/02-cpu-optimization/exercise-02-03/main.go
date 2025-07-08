package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"runtime"
	"time"

	"exercise-02-03/algorithms"
	"exercise-02-03/analysis"
	"exercise-02-03/benchmarks"
)

func main() {
	// Command line flags
	var (
		runBenchmarks = flag.Bool("bench", false, "Run comprehensive benchmarks")
		runProfile    = flag.Bool("profile", false, "Run profiling tests")
		generateReport = flag.Bool("report", false, "Generate performance report")
		compareAlgs   = flag.String("compare", "", "Compare specific algorithms (comma-separated)")
		testCorrectness = flag.Bool("test", false, "Run correctness tests")
		outputFormat  = flag.String("format", "text", "Output format: text, csv, markdown")
		outputFile    = flag.String("output", "", "Output file (default: stdout)")
		verbose       = flag.Bool("verbose", false, "Verbose output")
		quick         = flag.Bool("quick", false, "Run quick tests only")
	)
	flag.Parse()

	if *verbose {
		fmt.Printf("Go version: %s\n", runtime.Version())
		fmt.Printf("GOMAXPROCS: %d\n", runtime.GOMAXPROCS(0))
		fmt.Printf("NumCPU: %d\n", runtime.NumCPU())
		fmt.Println()
	}

	// Create performance analyzer
	analyzer := analysis.NewPerformanceAnalyzer()

	// Run correctness tests first
	if *testCorrectness {
		fmt.Println("Running correctness tests...")
		if err := runCorrectnessTests(); err != nil {
			log.Fatalf("Correctness tests failed: %v", err)
		}
		fmt.Println("✓ All correctness tests passed")
		fmt.Println()
	}

	// Run benchmarks
	if *runBenchmarks {
		fmt.Println("Running performance benchmarks...")
		results := runPerformanceBenchmarks(*quick, *verbose)
		for _, result := range results {
			analyzer.AddResult(result)
		}
		fmt.Printf("✓ Completed %d benchmark tests\n\n", len(results))
	}

	// Run profiling
	if *runProfile {
		fmt.Println("Running profiling tests...")
		if err := runProfilingTests(*verbose); err != nil {
			log.Printf("Profiling tests encountered errors: %v", err)
		} else {
			fmt.Println("✓ Profiling tests completed")
		}
		fmt.Println()
	}

	// Generate reports
	if *generateReport && len(analyzer.Results) > 0 {
		fmt.Println("Generating performance report...")
		if err := generatePerformanceReport(analyzer, *outputFormat, *outputFile); err != nil {
			log.Fatalf("Failed to generate report: %v", err)
		}
		fmt.Println("✓ Report generated successfully")
	}

	// Compare specific algorithms
	if *compareAlgs != "" && len(analyzer.Results) > 0 {
		fmt.Println("Generating algorithm comparison...")
		if err := generateComparisonReport(analyzer, *compareAlgs, *outputFile); err != nil {
			log.Fatalf("Failed to generate comparison: %v", err)
		}
	}

	// If no specific action requested, show usage
	if !*runBenchmarks && !*runProfile && !*generateReport && *compareAlgs == "" && !*testCorrectness {
		fmt.Println("Sorting Algorithm Performance Analysis Tool")
		fmt.Println("==========================================")
		fmt.Println()
		fmt.Println("Usage examples:")
		fmt.Println("  go run main.go -test                    # Run correctness tests")
		fmt.Println("  go run main.go -bench -quick             # Run quick benchmarks")
		fmt.Println("  go run main.go -bench -report            # Run benchmarks and generate report")
		fmt.Println("  go run main.go -profile                  # Run profiling tests")
		fmt.Println("  go run main.go -bench -compare=quicksort,mergesort")
		fmt.Println("  go run main.go -bench -report -format=csv -output=results.csv")
		fmt.Println()
		fmt.Println("Available algorithms: quicksort, mergesort, heapsort, radixsort, timsort")
		fmt.Println("Available formats: text, csv, markdown")
	}
}

// runCorrectnessTests runs correctness tests for all algorithms
func runCorrectnessTests() error {
	testSizes := []int{0, 1, 2, 10, 100, 1000}
	algorithmTests := map[string]func([]int){
		"quicksort": algorithms.QuickSort,
		"mergesort": algorithms.MergeSort,
		"heapsort":  algorithms.HeapSort,
		"timsort":   algorithms.TimSort,
	}

	for name, sortFunc := range algorithmTests {
		for _, size := range testSizes {
			// Test with different patterns
			patterns := []struct {
				name string
				data []int
			}{
				{"random", benchmarks.GenerateRandomData(size)},
				{"sorted", benchmarks.GenerateSortedData(size)},
				{"reverse", benchmarks.GenerateReverseSortedData(size)},
				{"duplicates", benchmarks.GenerateManyDuplicatesData(size)},
			}

			for _, pattern := range patterns {
				data := make([]int, len(pattern.data))
				copy(data, pattern.data)

				sortFunc(data)

				if !isSorted(data) {
					return fmt.Errorf("%s failed on %s data of size %d", name, pattern.name, size)
				}
			}
		}
	}

	// Test RadixSort separately (integers only)
	for _, size := range testSizes {
		data := benchmarks.GenerateRandomData(size)
		original := make([]int, len(data))
		copy(original, data)

		algorithms.RadixSortLSD(data)

		if !isSorted(data) {
			return fmt.Errorf("radixsort failed on random data of size %d", size)
		}
	}

	return nil
}

// isSorted checks if a slice is sorted
func isSorted(data []int) bool {
	for i := 1; i < len(data); i++ {
		if data[i] < data[i-1] {
			return false
		}
	}
	return true
}

// runPerformanceBenchmarks runs comprehensive performance benchmarks
func runPerformanceBenchmarks(quick, verbose bool) []analysis.AlgorithmResult {
	var results []analysis.AlgorithmResult

	// Define test parameters
	sizes := []int{1000, 10000, 100000}
	if !quick {
		sizes = append(sizes, 500000, 1000000)
	}

	patterns := []string{"random", "sorted", "reverse", "partially_sorted", "many_duplicates"}
	if !quick {
		patterns = append(patterns, "few_unique", "nearly_sorted")
	}

	algorithms := map[string]func([]int){
		"quicksort": algorithms.QuickSort,
		"mergesort": algorithms.MergeSort,
		"heapsort":  algorithms.HeapSort,
		"timsort":   algorithms.TimSort,
	}

	// Run benchmarks for each combination
	for _, size := range sizes {
		for _, pattern := range patterns {
			for algName, algFunc := range algorithms {
				if verbose {
					fmt.Printf("Testing %s on %s data (size: %d)...\n", algName, pattern, size)
				}

				result := benchmarkAlgorithm(algName, pattern, size, algFunc)
				results = append(results, result)
			}

			// Test RadixSort for integer data
			if verbose {
				fmt.Printf("Testing radixsort on %s data (size: %d)...\n", pattern, size)
			}
			result := benchmarkRadixSort(pattern, size)
			results = append(results, result)
		}
	}

	return results
}

// benchmarkAlgorithm benchmarks a single algorithm
func benchmarkAlgorithm(name, pattern string, size int, sortFunc func([]int)) analysis.AlgorithmResult {
	// Generate test data
	var data []int
	switch pattern {
	case "random":
		data = benchmarks.GenerateRandomData(size)
	case "sorted":
		data = benchmarks.GenerateSortedData(size)
	case "reverse":
		data = benchmarks.GenerateReverseSortedData(size)
	case "partially_sorted":
		data = benchmarks.GeneratePartiallySortedData(size)
	case "many_duplicates":
		data = benchmarks.GenerateManyDuplicatesData(size)
	case "few_unique":
		data = benchmarks.GenerateFewUniqueData(size)
	case "nearly_sorted":
		data = benchmarks.GenerateNearlySortedData(size)
	default:
		data = benchmarks.GenerateRandomData(size)
	}

	// Warm up
	warmupData := make([]int, len(data))
	copy(warmupData, data)
	sortFunc(warmupData)

	// Benchmark
	testData := make([]int, len(data))
	copy(testData, data)

	start := time.Now()
	sortFunc(testData)
	duration := time.Since(start)

	// Calculate throughput
	throughput := float64(size) / duration.Seconds()

	return analysis.AlgorithmResult{
		Name:          name,
		DataPattern:   pattern,
		DataSize:      size,
		ExecutionTime: duration,
		Throughput:    throughput,
		// Note: Comparisons, Swaps, etc. would require instrumented versions
		// of the algorithms to track these metrics accurately
	}
}

// benchmarkRadixSort benchmarks radix sort specifically
func benchmarkRadixSort(pattern string, size int) analysis.AlgorithmResult {
	// Generate test data
	var data []int
	switch pattern {
	case "random":
		data = benchmarks.GenerateRandomData(size)
	case "sorted":
		data = benchmarks.GenerateSortedData(size)
	case "reverse":
		data = benchmarks.GenerateReverseSortedData(size)
	case "partially_sorted":
		data = benchmarks.GeneratePartiallySortedData(size)
	case "many_duplicates":
		data = benchmarks.GenerateManyDuplicatesData(size)
	case "few_unique":
		data = benchmarks.GenerateFewUniqueData(size)
	case "nearly_sorted":
		data = benchmarks.GenerateNearlySortedData(size)
	default:
		data = benchmarks.GenerateRandomData(size)
	}

	// Warm up
	warmupData := make([]int, len(data))
	copy(warmupData, data)
	algorithms.RadixSortLSD(warmupData)

	// Benchmark
	testData := make([]int, len(data))
	copy(testData, data)

	start := time.Now()
	algorithms.RadixSortLSD(testData)
	duration := time.Since(start)

	// Calculate throughput
	throughput := float64(size) / duration.Seconds()

	return analysis.AlgorithmResult{
		Name:          "radixsort",
		DataPattern:   pattern,
		DataSize:      size,
		ExecutionTime: duration,
		Throughput:    throughput,
	}
}

// runProfilingTests runs profiling tests
func runProfilingTests(verbose bool) error {
	if verbose {
		fmt.Println("Note: For detailed profiling, run:")
		fmt.Println("  go test -bench=. -cpuprofile=cpu.prof ./benchmarks")
		fmt.Println("  go tool pprof cpu.prof")
		fmt.Println()
	}

	// Run a subset of profiling-focused tests
	size := 100000
	data := benchmarks.GenerateRandomData(size)

	algorithms := map[string]func([]int){
		"quicksort": algorithms.QuickSort,
		"mergesort": algorithms.MergeSort,
		"heapsort":  algorithms.HeapSort,
		"timsort":   algorithms.TimSort,
	}

	for name, sortFunc := range algorithms {
		if verbose {
			fmt.Printf("Profiling %s...\n", name)
		}

		testData := make([]int, len(data))
		copy(testData, data)

		// Force garbage collection before test
		runtime.GC()

		start := time.Now()
		sortFunc(testData)
		duration := time.Since(start)

		if verbose {
			fmt.Printf("  %s completed in %v\n", name, duration)
		}
	}

	return nil
}

// generatePerformanceReport generates a performance report in the specified format
func generatePerformanceReport(analyzer *analysis.PerformanceAnalyzer, format, outputFile string) error {
	reporter := analysis.NewReportGenerator(analyzer)

	var writer *os.File
	if outputFile != "" {
		var err error
		writer, err = os.Create(outputFile)
		if err != nil {
			return fmt.Errorf("failed to create output file: %v", err)
		}
		defer writer.Close()
	} else {
		writer = os.Stdout
	}

	switch format {
	case "csv":
		return reporter.GenerateCSVReport(writer)
	case "markdown":
		return reporter.GenerateMarkdownReport(writer)
	default:
		return reporter.GenerateTextReport(writer)
	}
}

// generateComparisonReport generates a comparison report for specific algorithms
func generateComparisonReport(analyzer *analysis.PerformanceAnalyzer, compareAlgs, outputFile string) error {
	reporter := analysis.NewReportGenerator(analyzer)

	var writer *os.File
	if outputFile != "" {
		var err error
		writer, err = os.Create(outputFile + "_comparison.txt")
		if err != nil {
			return fmt.Errorf("failed to create comparison file: %v", err)
		}
		defer writer.Close()
	} else {
		writer = os.Stdout
	}

	// Parse algorithm names
	algorithms := []string{}
	for _, alg := range []string{"quicksort", "mergesort", "heapsort", "radixsort", "timsort"} {
		if contains(compareAlgs, alg) {
			algorithms = append(algorithms, alg)
		}
	}

	if len(algorithms) < 2 {
		return fmt.Errorf("need at least 2 algorithms to compare")
	}

	return reporter.GenerateComparisonReport(writer, algorithms)
}

// contains checks if a string contains a substring
func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || 
		(len(s) > len(substr) && (s[:len(substr)+1] == substr+"," || 
			s[len(s)-len(substr)-1:] == ","+substr ||
			containsMiddle(s, substr))))
}

// containsMiddle checks if substr is in the middle of s (surrounded by commas)
func containsMiddle(s, substr string) bool {
	target := "," + substr + ","
	for i := 0; i <= len(s)-len(target); i++ {
		if s[i:i+len(target)] == target {
			return true
		}
	}
	return false
}