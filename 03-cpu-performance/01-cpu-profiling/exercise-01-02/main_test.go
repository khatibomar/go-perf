package main

import (
	"exercise-01-02/algorithms"
	"exercise-01-02/data"
	"exercise-01-02/server"
	"exercise-01-02/service"
	"fmt"
	"math/rand"
	"runtime"
	"sync"
	"testing"
	"time"
)

// Benchmark functions for flame graph analysis

// BenchmarkRealisticWorkload benchmarks the realistic workload
func BenchmarkRealisticWorkload(b *testing.B) {
	for i := 0; i < b.N; i++ {
		runRealisticWorkload(time.Second*5, 4, 1000, "realistic", false)
	}
}

// BenchmarkCPUHeavyWorkload benchmarks CPU-intensive operations
func BenchmarkCPUHeavyWorkload(b *testing.B) {
	for i := 0; i < b.N; i++ {
		runCPUHeavyWorkload(5, 1000)
	}
}

// BenchmarkIOHeavyWorkload benchmarks I/O-intensive operations
func BenchmarkIOHeavyWorkload(b *testing.B) {
	for i := 0; i < b.N; i++ {
		runIOHeavyWorkload(5, 1000)
	}
}

// BenchmarkMixedWorkload benchmarks mixed operations
func BenchmarkMixedWorkload(b *testing.B) {
	for i := 0; i < b.N; i++ {
		runMixedWorkload(5, 1000)
	}
}

// Component-specific benchmarks

// BenchmarkServerHandler benchmarks server request handling
func BenchmarkServerHandler(b *testing.B) {
	appServer := server.New()
	request := &server.RequestData{
		ID:      1,
		Payload: generateTestPayload(1024),
		Headers: map[string]string{
			"Content-Type":   "application/json",
			"Authorization": "Bearer test-token",
			"User-Agent":    "benchmark-client",
		},
		Metadata: map[string]interface{}{
			"timestamp": time.Now(),
			"source":    "benchmark",
			"priority":  "high",
		},
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		appServer.HandleRequest(request)
	}
}

// BenchmarkServiceProcessor benchmarks service processing
func BenchmarkServiceProcessor(b *testing.B) {
	processor := service.NewProcessor()
	data := generateTestData(1000)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		processor.ProcessData(data)
	}
}

// BenchmarkServiceAnalyzer benchmarks data analysis
func BenchmarkServiceAnalyzer(b *testing.B) {
	analyzer := service.NewAnalyzer()
	data := generateTestData(500)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		analyzer.AnalyzeData(data)
	}
}

// BenchmarkDataStructures benchmarks data structure operations
func BenchmarkDataStructures(b *testing.B) {
	generator := data.NewDataGenerator()
	keys := generator.GenerateKeys(1000, "random")
	values := generator.GenerateValues(1000, "mixed")
	operations := []string{"insert", "get", "delete"}

	// Create a simple map-based data structure for benchmarking
	ds := data.NewHashMap()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		data.BenchmarkDataStructure(ds, keys, values, operations)
		ds.Clear()
	}
}

// Algorithm-specific benchmarks

// BenchmarkAlgorithmsPrimes benchmarks prime calculation
func BenchmarkAlgorithmsPrimes(b *testing.B) {
	engine := algorithms.NewComputationalEngine()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		engine.CalculatePrimes(10000)
	}
}

// BenchmarkAlgorithmsFibonacci benchmarks Fibonacci calculation
func BenchmarkAlgorithmsFibonacci(b *testing.B) {
	engine := algorithms.NewComputationalEngine()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		engine.CalculateFibonacci(40)
	}
}

// BenchmarkAlgorithmsQuickSort benchmarks quicksort
func BenchmarkAlgorithmsQuickSort(b *testing.B) {
	engine := algorithms.NewComputationalEngine()
	data := generateRandomArray(10000)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		engine.QuickSort(data)
	}
}

// BenchmarkAlgorithmsMergeSort benchmarks merge sort
func BenchmarkAlgorithmsMergeSort(b *testing.B) {
	engine := algorithms.NewComputationalEngine()
	data := generateRandomArray(10000)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		engine.MergeSort(data)
	}
}

// BenchmarkAlgorithmsMatrixMultiply benchmarks matrix multiplication
func BenchmarkAlgorithmsMatrixMultiply(b *testing.B) {
	engine := algorithms.NewComputationalEngine()
	matrixA := generateRandomMatrix(100, 100)
	matrixB := generateRandomMatrix(100, 100)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		engine.MatrixMultiply(matrixA, matrixB)
	}
}

// BenchmarkAlgorithmsStringSearch benchmarks string searching
func BenchmarkAlgorithmsStringSearch(b *testing.B) {
	engine := algorithms.NewComputationalEngine()
	text := generateRandomString(10000)
	pattern := text[1000:1010] // 10 character pattern

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		engine.KMPSearch(text, pattern)
	}
}

// BenchmarkAlgorithmsGraphDFS benchmarks depth-first search
func BenchmarkAlgorithmsGraphDFS(b *testing.B) {
	engine := algorithms.NewComputationalEngine()
	graph := generateRandomGraph(1000, 5000)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		engine.DFS(graph, 0)
	}
}

// BenchmarkAlgorithmsGraphBFS benchmarks breadth-first search
func BenchmarkAlgorithmsGraphBFS(b *testing.B) {
	engine := algorithms.NewComputationalEngine()
	graph := generateRandomGraph(1000, 5000)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		engine.BFS(graph, 0)
	}
}

// Memory allocation benchmarks

// BenchmarkMemoryAllocation benchmarks memory allocation patterns
func BenchmarkMemoryAllocation(b *testing.B) {
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		// Simulate various allocation patterns
		data := make([][]byte, 100)
		for j := range data {
			data[j] = make([]byte, 1024)
		}
		
		// Force some allocations
		for j := 0; j < 1000; j++ {
			_ = make([]int, 100)
		}
		
		// Create some maps
		m := make(map[string]interface{})
		for j := 0; j < 100; j++ {
			m[fmt.Sprintf("key_%d", j)] = j
		}
		
		runtime.KeepAlive(data)
		runtime.KeepAlive(m)
	}
}

// BenchmarkGoroutineCreation benchmarks goroutine creation and synchronization
func BenchmarkGoroutineCreation(b *testing.B) {
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		var wg sync.WaitGroup
		for j := 0; j < 100; j++ {
			wg.Add(1)
			go func() {
				defer wg.Done()
				// Simulate some work
				for k := 0; k < 1000; k++ {
					_ = k * k
				}
			}()
		}
		wg.Wait()
	}
}

// BenchmarkChannelOperations benchmarks channel operations
func BenchmarkChannelOperations(b *testing.B) {
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		ch := make(chan int, 100)
		var wg sync.WaitGroup
		
		// Producer
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := 0; j < 1000; j++ {
				ch <- j
			}
			close(ch)
		}()
		
		// Consumer
		wg.Add(1)
		go func() {
			defer wg.Done()
			for range ch {
				// Process data
			}
		}()
		
		wg.Wait()
	}
}

// Comparative benchmarks

// BenchmarkSortingComparison compares different sorting algorithms
func BenchmarkSortingComparison(b *testing.B) {
	engine := algorithms.NewComputationalEngine()
	data := generateRandomArray(5000)

	b.Run("QuickSort", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			engine.QuickSort(data)
		}
	})

	b.Run("MergeSort", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			engine.MergeSort(data)
		}
	})

	b.Run("HeapSort", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			engine.HeapSort(data)
		}
	})
}

// BenchmarkSearchComparison compares different search algorithms
func BenchmarkSearchComparison(b *testing.B) {
	engine := algorithms.NewComputationalEngine()
	data := generateSortedArray(10000)
	target := data[5000]

	b.Run("BinarySearch", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			engine.BinarySearch(data, target)
		}
	})

	b.Run("LinearSearch", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			engine.LinearSearch(data, target)
		}
	})
}

// Stress test benchmarks

// BenchmarkStressTest runs a comprehensive stress test
func BenchmarkStressTest(b *testing.B) {
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		var wg sync.WaitGroup
		
		// CPU-intensive goroutines
		for j := 0; j < 10; j++ {
			wg.Add(1)
			go func() {
				defer wg.Done()
				engine := algorithms.NewComputationalEngine()
				engine.RunMixedWorkload(100)
			}()
		}
		
		// Memory-intensive goroutines
		for j := 0; j < 5; j++ {
			wg.Add(1)
			go func() {
				defer wg.Done()
				generator := data.NewDataGenerator()
				keys := generator.GenerateKeys(1000, "random")
				values := generator.GenerateValues(1000, "mixed")
				operations := []string{"insert", "get", "delete"}
				ds := data.NewHashMap()
				data.BenchmarkDataStructure(ds, keys, values, operations)
			}()
		}
		
		// I/O simulation goroutines
		for j := 0; j < 3; j++ {
			wg.Add(1)
			go func() {
				defer wg.Done()
				appServer := server.New()
				for k := 0; k < 100; k++ {
					request := &server.RequestData{
						ID:      k,
						Payload: generateTestPayload(512),
						Headers: map[string]string{"Content-Type": "application/json"},
						Metadata: map[string]interface{}{"timestamp": time.Now()},
					}
					appServer.HandleRequest(request)
				}
			}()
		}
		
		wg.Wait()
	}
}

// Helper functions for benchmark data generation

func generateTestPayload(size int) []byte {
	payload := make([]byte, size)
	for i := range payload {
		payload[i] = byte(rand.Intn(256))
	}
	return payload
}

func generateTestData(size int) []interface{} {
	data := make([]interface{}, size)
	for i := range data {
		switch rand.Intn(4) {
		case 0:
			data[i] = rand.Intn(10000)
		case 1:
			data[i] = rand.Float64() * 1000
		case 2:
			data[i] = generateRandomString(20)
		case 3:
			data[i] = map[string]interface{}{
				"id":    rand.Intn(1000),
				"value": rand.Float64(),
				"name":  generateRandomString(10),
			}
		}
	}
	return data
}

func generateRandomArray(size int) []int {
	arr := make([]int, size)
	for i := range arr {
		arr[i] = rand.Intn(100000)
	}
	return arr
}

func generateSortedArray(size int) []int {
	arr := generateRandomArray(size)
	// Simple bubble sort for small arrays, or use built-in sort
	for i := 0; i < len(arr)-1; i++ {
		for j := 0; j < len(arr)-i-1; j++ {
			if arr[j] > arr[j+1] {
				arr[j], arr[j+1] = arr[j+1], arr[j]
			}
		}
	}
	return arr
}

func generateRandomString(length int) string {
	const charset = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
	b := make([]byte, length)
	for i := range b {
		b[i] = charset[rand.Intn(len(charset))]
	}
	return string(b)
}

func generateRandomMatrix(rows, cols int) algorithms.Matrix {
	matrix := algorithms.NewMatrix(rows, cols)
	for i := 0; i < rows; i++ {
		for j := 0; j < cols; j++ {
			matrix[i][j] = rand.Float64() * 100
		}
	}
	return matrix
}

func generateRandomGraph(vertices, edges int) *algorithms.Graph {
	graph := algorithms.NewGraph(vertices)
	for i := 0; i < edges; i++ {
		from := rand.Intn(vertices)
		to := rand.Intn(vertices)
		if from != to {
			graph.AddEdge(from, to)
		}
	}
	return graph
}

// Example test functions

func TestRealisticWorkload(t *testing.T) {
	// Test that the realistic workload runs without errors
	runRealisticWorkload(time.Second*1, 1, 10, "realistic", false)
}

func TestCPUHeavyWorkload(t *testing.T) {
	// Test that the CPU-heavy workload runs without errors
	runCPUHeavyWorkload(1, 10)
}

func TestIOHeavyWorkload(t *testing.T) {
	// Test that the I/O-heavy workload runs without errors
	runIOHeavyWorkload(1, 10)
}

func TestMixedWorkload(t *testing.T) {
	// Test that the mixed workload runs without errors
	runMixedWorkload(1, 10)
}

func TestAlgorithmsEngine(t *testing.T) {
	engine := algorithms.NewComputationalEngine()
	
	// Test prime calculation
	primes := engine.CalculatePrimes(100)
	if len(primes) == 0 {
		t.Error("Expected primes to be calculated")
	}
	
	// Test Fibonacci calculation
	fib := engine.CalculateFibonacci(10)
	if len(fib) != 10 {
		t.Errorf("Expected 10 Fibonacci numbers, got %d", len(fib))
	}
	
	// Test sorting
	data := []int{3, 1, 4, 1, 5, 9, 2, 6}
	sorted := engine.QuickSort(data)
	if len(sorted) != len(data) {
		t.Error("Sorted array length mismatch")
	}
	
	// Test matrix operations
	matrixA := algorithms.NewMatrix(2, 2)
	matrixB := algorithms.NewMatrix(2, 2)
	matrixA[0][0], matrixA[0][1] = 1, 2
	matrixA[1][0], matrixA[1][1] = 3, 4
	matrixB[0][0], matrixB[0][1] = 5, 6
	matrixB[1][0], matrixB[1][1] = 7, 8
	
	result := engine.MatrixMultiply(matrixA, matrixB)
	if result == nil {
		t.Error("Matrix multiplication failed")
	}
}

func TestDataStructures(t *testing.T) {
	generator := data.NewDataGenerator()
	
	// Test data structure operations
	keys := generator.GenerateKeys(100, "random")
	values := generator.GenerateValues(100, "mixed")
	operations := []string{"insert", "get", "delete"}
	ds := data.NewHashMap()
	data.BenchmarkDataStructure(ds, keys, values, operations)

	// Test should complete without panics
}

func TestServerComponents(t *testing.T) {
	appServer := server.New()
	
	request := &server.RequestData{
		ID:      1,
		Payload: []byte("test payload"),
		Headers: map[string]string{
			"Content-Type": "application/json",
		},
		Metadata: map[string]interface{}{
			"timestamp": time.Now(),
		},
	}
	
	result := appServer.HandleRequest(request)
	if result.Original.ID != request.ID {
		t.Error("Request ID mismatch")
	}
}

func TestServiceComponents(t *testing.T) {
	processor := service.NewProcessor()
	analyzer := service.NewAnalyzer()
	
	testData := map[string]interface{}{
		"data_type": "numeric",
		"numeric_data": []int{1, 2, 3, 4, 5},
		"string_data": []string{"test1", "test2", "test3"},
		"length": 5,
		"metadata": map[string]interface{}{
			"source": "test",
			"timestamp": time.Now(),
		},
	}
	
	// Test processor
	result := processor.ProcessData(testData)
	if result == nil {
		t.Error("Expected processor to return a result")
		return
	}
	if len(result.Data) == 0 {
		t.Error("Expected data to be processed")
	}
	
	// Test analyzer
	analysisResult := analyzer.AnalyzeData(testData)
	if analysisResult.Statistics == nil {
		t.Error("Expected data to be analyzed")
	}
}

// Benchmark memory allocation patterns
func BenchmarkMemoryPatterns(b *testing.B) {
	b.Run("SliceGrowth", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			var slice []int
			for j := 0; j < 1000; j++ {
				slice = append(slice, j)
			}
			runtime.KeepAlive(slice)
		}
	})

	b.Run("PreallocatedSlice", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			slice := make([]int, 0, 1000)
			for j := 0; j < 1000; j++ {
				slice = append(slice, j)
			}
			runtime.KeepAlive(slice)
		}
	})

	b.Run("MapOperations", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			m := make(map[int]string)
			for j := 0; j < 1000; j++ {
				m[j] = fmt.Sprintf("value_%d", j)
			}
			runtime.KeepAlive(m)
		}
	})

	b.Run("PreallocatedMap", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			m := make(map[int]string, 1000)
			for j := 0; j < 1000; j++ {
				m[j] = fmt.Sprintf("value_%d", j)
			}
			runtime.KeepAlive(m)
		}
	})
}