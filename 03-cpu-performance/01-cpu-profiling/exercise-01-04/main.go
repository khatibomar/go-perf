package main

import (
	"flag"
	"log"
	"math/rand"
	"os"
	"runtime/pprof"
	"time"

	"exercise-01-04/internal/algorithms"
	"exercise-01-04/internal/datastructures"

	"github.com/pkg/profile"
)

func main() {
	var (
		size       = flag.Int("size", 10000, "Data size for testing")
		iterations = flag.Int("iterations", 100, "Number of iterations")
		profileType = flag.String("profile", "cpu", "Profile type: cpu, mem, trace")
		optimized  = flag.Bool("optimized", false, "Use optimized algorithms")
	)
	flag.Parse()

	// Start profiling based on type
	var stop func()
	switch *profileType {
	case "cpu":
		stop = profile.Start(profile.CPUProfile, profile.ProfilePath(".")).Stop
	case "mem":
		stop = profile.Start(profile.MemProfile, profile.ProfilePath(".")).Stop
	case "trace":
		stop = profile.Start(profile.TraceProfile, profile.ProfilePath(".")).Stop
	default:
		log.Fatalf("Unknown profile type: %s", *profileType)
	}
	defer stop()

	log.Printf("Starting profile-guided optimization demo")
	log.Printf("Data size: %d, Iterations: %d, Optimized: %v", *size, *iterations, *optimized)

	// Generate test data
	data := algorithms.GenerateRandomData(*size)
	cache := datastructures.NewSearchCache(1000)

	start := time.Now()

	// Run sorting benchmarks
	runSortingBenchmarks(data, *iterations, *optimized)

	// Run searching benchmarks
	runSearchingBenchmarks(data, *iterations, *optimized, cache)

	duration := time.Since(start)
	log.Printf("Total execution time: %v", duration)

	// Print cache statistics
	hits, misses, hitRate := cache.Stats()
	log.Printf("Cache stats - Hits: %d, Misses: %d, Hit Rate: %.2f%%",
		hits, misses, hitRate*100)
}

func runSortingBenchmarks(data []int, iterations int, optimized bool) {
	log.Println("Running sorting benchmarks...")

	for i := 0; i < iterations; i++ {
		// QuickSort
		testData := make([]int, len(data))
		copy(testData, data)
		if optimized {
			algorithms.OptimizedQuickSort(testData)
		} else {
			algorithms.NaiveQuickSort(testData)
		}

		// MergeSort
		testData = make([]int, len(data))
		copy(testData, data)
		if optimized {
			algorithms.OptimizedMergeSort(testData)
		} else {
			algorithms.NaiveMergeSort(testData)
		}
	}
}

func runSearchingBenchmarks(data []int, iterations int, optimized bool, cache *datastructures.SearchCache) {
	log.Println("Running searching benchmarks...")

	// Sort data for binary search
	if optimized {
		algorithms.OptimizedQuickSort(data)
	} else {
		algorithms.NaiveQuickSort(data)
	}

	for i := 0; i < iterations; i++ {
		target := data[rand.Intn(len(data))]

		// Binary Search
		if optimized {
			algorithms.OptimizedBinarySearch(data, target, cache)
		} else {
			algorithms.NaiveBinarySearch(data, target)
		}

		// Linear Search
		if optimized {
			algorithms.OptimizedLinearSearch(data, target)
		} else {
			algorithms.NaiveLinearSearch(data, target)
		}
	}
}

// Manual profiling functions for demonstration
func startCPUProfile(filename string) *os.File {
	f, err := os.Create(filename)
	if err != nil {
		log.Fatal(err)
	}
	pprof.StartCPUProfile(f)
	return f
}

func stopCPUProfile(f *os.File) {
	pprof.StopCPUProfile()
	f.Close()
}

func writeMemProfile(filename string) {
	f, err := os.Create(filename)
	if err != nil {
		log.Fatal(err)
	}
	defer f.Close()
	pprof.WriteHeapProfile(f)
}