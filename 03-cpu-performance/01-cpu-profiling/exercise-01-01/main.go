package main

import (
	"flag"
	"fmt"
	"sync"
	"time"
)

func main() {
	// Parse command-line flags
	var (
		cpuProfile  = flag.String("cpuprofile", "", "write cpu profile to file")
		httpProfile = flag.Bool("httpprofile", false, "enable HTTP profiling")
		profilePort = flag.String("profileport", "6060", "HTTP profiling port")
		duration    = flag.Duration("duration", 30*time.Second, "run duration")
		workers     = flag.Int("workers", 4, "number of worker goroutines")
		workload    = flag.String("workload", "mixed", "workload type: primes, fibonacci, strings, hashes, sort, or mixed")
		verbose     = flag.Bool("verbose", false, "verbose output")
	)
	flag.Parse()

	// Setup profiling
	cleanup := setupProfiling(ProfilingConfig{
		CPUProfile:  *cpuProfile,
		HTTPProfile: *httpProfile,
		ProfilePort: *profilePort,
		Verbose:     *verbose,
	})
	defer cleanup()

	fmt.Printf("Starting CPU-intensive workload...\n")
	fmt.Printf("Duration: %v\n", *duration)
	fmt.Printf("Workers: %d\n", *workers)
	fmt.Printf("Workload: %s\n", *workload)
	if *httpProfile {
		fmt.Printf("HTTP profiling available at: http://localhost:%s/debug/pprof/\n", *profilePort)
	}
	fmt.Println()

	// Run CPU-intensive workload
	start := time.Now()
	runWorkload(*duration, *workers, *workload, *verbose)
	elapsed := time.Since(start)

	fmt.Printf("\nWorkload completed in %v\n", elapsed)
	if *cpuProfile != "" {
		fmt.Printf("CPU profile written to: %s\n", *cpuProfile)
		fmt.Printf("Analyze with: go tool pprof %s\n", *cpuProfile)
	}
}

func runWorkload(duration time.Duration, workers int, workloadType string, verbose bool) {
	var wg sync.WaitGroup
	done := make(chan bool)

	// Start timer
	go func() {
		time.Sleep(duration)
		close(done)
	}()

	// Start worker goroutines
	for i := 0; i < workers; i++ {
		wg.Add(1)
		go func(workerID int) {
			defer wg.Done()
			runWorker(workerID, done, workloadType, verbose)
		}(i)
	}

	// Wait for all workers to complete
	wg.Wait()
}

func runWorker(workerID int, done <-chan bool, workloadType string, verbose bool) {
	iterations := 0
	for {
		select {
		case <-done:
			if verbose {
				fmt.Printf("Worker %d completed %d iterations\n", workerID, iterations)
			}
			return
		default:
			// Perform work based on workload type
			switch workloadType {
			case "primes":
				calculatePrimes(1000)
			case "fibonacci":
				calculateFibonacci(35)
			case "strings":
				processStrings(generateStringData(100))
			case "hashes":
				calculateHashes(generateStringData(50))
			case "sort":
				sortAndSearch(generateIntData(1000))
			case "mixed":
				// Mix of different operations
				switch iterations % 5 {
				case 0:
					calculatePrimes(500)
				case 1:
					calculateFibonacci(30)
				case 2:
					processStrings(generateStringData(50))
				case 3:
					calculateHashes(generateStringData(25))
				case 4:
					sortAndSearch(generateIntData(500))
				}
			default:
				// Default to mixed workload
				calculatePrimes(500)
				processStrings(generateStringData(50))
			}
			iterations++
		}
	}
}

// Helper functions to generate test data
func generateStringData(count int) []string {
	data := make([]string, count)
	for i := 0; i < count; i++ {
		data[i] = fmt.Sprintf("test-string-data-item-%d-with-some-extra-content", i)
	}
	return data
}

func generateIntData(count int) []int {
	data := make([]int, count)
	for i := 0; i < count; i++ {
		data[i] = count - i // Reverse order to make sorting work
	}
	return data
}