package main

import (
	"context"
	"encoding/binary"
	"flag"
	"fmt"
	"log"
	"net/http"
	_ "net/http/pprof"
	"os"
	"runtime/pprof"
	"sync"
	"time"

	"exercise-01-02/algorithms"
	"exercise-01-02/data"
	"exercise-01-02/server"
	"exercise-01-02/service"
)

func main() {
	// Parse command-line flags
	var (
		cpuProfile  = flag.String("cpuprofile", "", "write cpu profile to file")
		httpProfile = flag.Bool("httpprofile", false, "enable HTTP profiling")
		profilePort = flag.String("profileport", "6060", "HTTP profiling port")
		duration    = flag.Duration("duration", 30*time.Second, "run duration")
		workers     = flag.Int("workers", 8, "number of worker goroutines")
		requests    = flag.Int("requests", 1000, "number of requests to process")
		verbose     = flag.Bool("verbose", false, "verbose output")
		workload    = flag.String("workload", "realistic", "workload type: realistic, cpu-heavy, io-heavy, mixed")
	)
	flag.Parse()

	// Setup profiling
	cleanup := setupProfiling(*cpuProfile, *httpProfile, *profilePort, *verbose)
	defer cleanup()

	fmt.Printf("Starting realistic workload for flame graph analysis...\n")
	fmt.Printf("Duration: %v\n", *duration)
	fmt.Printf("Workers: %d\n", *workers)
	fmt.Printf("Requests: %d\n", *requests)
	fmt.Printf("Workload: %s\n", *workload)
	if *httpProfile {
		fmt.Printf("HTTP profiling available at: http://localhost:%s/debug/pprof/\n", *profilePort)
	}
	fmt.Println()

	// Run realistic workload
	start := time.Now()
	runRealisticWorkload(*duration, *workers, *requests, *workload, *verbose)
	elapsed := time.Since(start)

	fmt.Printf("\nWorkload completed in %v\n", elapsed)
	if *cpuProfile != "" {
		fmt.Printf("CPU profile written to: %s\n", *cpuProfile)
		fmt.Printf("Generate flame graph with: go tool pprof -http=:8080 %s\n", *cpuProfile)
		fmt.Printf("Generate SVG with: go tool pprof -svg %s > flamegraph.svg\n", *cpuProfile)
	}
}

func runRealisticWorkload(duration time.Duration, workers, requests int, workloadType string, verbose bool) {
	// Create application components
	appServer := server.New()
	processor := service.NewProcessor()
	dataGen := data.NewDataGenerator()
	// Create a single computational engine to avoid race conditions
	compEngine := algorithms.NewComputationalEngine()

	// Setup context with timeout
	ctx, cancel := context.WithTimeout(context.Background(), duration)
	defer cancel()

	// Channel for work distribution
	requestChan := make(chan int, requests)
	var wg sync.WaitGroup

	// Start worker goroutines
	for i := 0; i < workers; i++ {
		wg.Add(1)
		go func(workerID int) {
			defer wg.Done()
			processRequests(ctx, workerID, requestChan, appServer, processor, dataGen, compEngine, workloadType, verbose)
		}(i)
	}

	// Generate work
	go func() {
		for i := 0; i < requests; i++ {
			select {
			case requestChan <- i:
			case <-ctx.Done():
				return
			}
		}
		close(requestChan)
	}()

	// Wait for completion or timeout
	wg.Wait()
}

func processRequests(ctx context.Context, workerID int, requestChan <-chan int, 
	appServer *server.Server, processor *service.Processor, dataGen *data.DataGenerator, 
	compEngine *algorithms.ComputationalEngine, workloadType string, verbose bool) {
	
	processed := 0
	for {
		select {
		case requestID, ok := <-requestChan:
			if !ok {
				if verbose {
					fmt.Printf("Worker %d processed %d requests\n", workerID, processed)
				}
				return
			}
			
			// Process request based on workload type
			handleRequest(requestID, appServer, processor, dataGen, compEngine, workloadType)
			processed++
			
		case <-ctx.Done():
			if verbose {
				fmt.Printf("Worker %d processed %d requests (timeout)\n", workerID, processed)
			}
			return
		}
	}
}

func handleRequest(requestID int, appServer *server.Server, processor *service.Processor, 
	dataGen *data.DataGenerator, compEngine *algorithms.ComputationalEngine, workloadType string) {
	
	// Generate request data
	intValues := dataGen.GenerateValues(100, "integer")
	payloadData := make([]int, len(intValues))
	for i, v := range intValues {
		payloadData[i] = v.(int)
	}
	
	// Convert to bytes for payload
	payloadBytes := make([]byte, len(payloadData)*4) // 4 bytes per int
	for i, val := range payloadData {
		binary.LittleEndian.PutUint32(payloadBytes[i*4:], uint32(val))
	}
	
	// Create proper RequestData
	requestData := &server.RequestData{
		ID:      requestID,
		Payload: payloadBytes,
		Headers: map[string]string{
			"Content-Type": "application/octet-stream",
			"User-Agent":   "load-tester",
		},
		Metadata: map[string]interface{}{
			"timestamp": time.Now(),
			"worker_id": requestID % 10,
		},
	}
	
	// Process through server middleware
	processedRequest := appServer.HandleRequest(requestData)
	
	// Business logic processing
	// Extract data from processed request for business processing
	var businessData interface{}
	if processedRequest != nil && processedRequest.Original != nil {
		businessData = map[string]interface{}{
			"data_type": "request",
			"payload": processedRequest.Original.Payload,
			"headers": processedRequest.Original.Headers,
			"metadata": processedRequest.Original.Metadata,
		}
	} else {
		businessData = map[string]interface{}{
			"data_type": "fallback",
			"request_id": requestID,
		}
	}
	result := processor.ProcessData(businessData)
	
	// Additional processing based on workload type
	switch workloadType {
	case "cpu-heavy":
		// CPU-intensive operations
		compEngine.RunMixedWorkload(10)
	case "io-heavy":
		// Simulate I/O operations
		time.Sleep(time.Microsecond * 100) // Simulate I/O delay
	case "mixed":
		// Mix of CPU and I/O
		if requestID%2 == 0 {
			compEngine.RunMixedWorkload(5)
		} else {
			time.Sleep(time.Microsecond * 50)
		}
	default: // "realistic"
		// Realistic mix of operations
		performRealisticProcessing(result, requestID, compEngine)
	}
}

func performRealisticProcessing(result *service.ProcessResult, requestID int, compEngine *algorithms.ComputationalEngine) {
	// Check if result is nil
	if result == nil {
		// Fallback to basic computational work
		compEngine.CalculatePrimes(requestID % 100 + 50)
		return
	}
	
	// Simulate realistic application behavior
	switch requestID % 10 {
	case 0, 1, 2: // 30% - Light processing
		if result.Data != nil && len(result.Data) > 0 {
			compEngine.QuickSort(result.Data[:min(len(result.Data), 100)])
		}
	case 3, 4: // 20% - Medium processing
		if result.Data != nil && len(result.Data) > 0 {
			compEngine.BinarySearch(result.Data, requestID)
			compEngine.MergeSort(result.Data[:min(len(result.Data), 500)])
		}
	case 5, 6: // 20% - Heavy processing
		compEngine.CalculatePrimes(requestID % 1000 + 100)
	case 7: // 10% - Very heavy processing
		compEngine.CalculateFactorial(requestID % 20 + 10)
	case 8: // 10% - String processing
		if result.StringData != nil && len(result.StringData) > 0 {
			for _, str := range result.StringData {
				compEngine.SimpleHash(str)
			}
		}
	case 9: // 10% - Data transformation
		if result.Data != nil && len(result.Data) > 0 {
			matrixA := algorithms.Matrix{{1, 2}, {3, 4}}
			matrixB := algorithms.Matrix{{5, 6}, {7, 8}}
			compEngine.MatrixMultiply(matrixA, matrixB)
		}
	}
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

// Wrapper functions for different workload types (used by tests)
func runCPUHeavyWorkload(duration int, requests int) {
	runRealisticWorkload(time.Duration(duration)*time.Second, 1, requests, "cpu-heavy", false)
}

func runIOHeavyWorkload(duration int, requests int) {
	runRealisticWorkload(time.Duration(duration)*time.Second, 1, requests, "io-heavy", false)
}

func runMixedWorkload(duration int, requests int) {
	runRealisticWorkload(time.Duration(duration)*time.Second, 1, requests, "mixed", false)
}

func setupProfiling(cpuProfile string, httpProfile bool, profilePort string, verbose bool) func() {
	var cleanupFuncs []func()

	// Setup CPU profiling to file if requested
	if cpuProfile != "" {
		f, err := os.Create(cpuProfile)
		if err != nil {
			log.Printf("Error creating CPU profile file %s: %v", cpuProfile, err)
		} else {
			if err := pprof.StartCPUProfile(f); err != nil {
				log.Printf("Error starting CPU profile: %v", err)
				f.Close()
			} else {
				if verbose {
					fmt.Printf("CPU profiling started, writing to: %s\n", cpuProfile)
				}
				cleanupFuncs = append(cleanupFuncs, func() {
					pprof.StopCPUProfile()
					f.Close()
					if verbose {
						fmt.Printf("CPU profiling stopped\n")
					}
				})
			}
		}
	}

	// Setup HTTP profiling if requested
	if httpProfile {
		go func() {
			addr := ":" + profilePort
			if verbose {
				fmt.Printf("Starting HTTP profiling server on %s\n", addr)
			}
			if err := http.ListenAndServe(addr, nil); err != nil {
				log.Printf("HTTP profiling server error: %v", err)
			}
		}()
	}

	// Return cleanup function
	return func() {
		for _, cleanup := range cleanupFuncs {
			cleanup()
		}
	}
}