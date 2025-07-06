package algorithms

import (
	"fmt"
	"math"
	"math/big"
	"math/rand"
	"sort"
	"sync"
	"time"
)

// ComputationalEngine performs various computational algorithms
type ComputationalEngine struct {
	rand       *rand.Rand
	cache      map[string]interface{}
	stats      ComputationStats
	cacheMutex sync.RWMutex
	statsMutex sync.Mutex
	randMutex  sync.Mutex
}

// ComputationStats tracks computation statistics
type ComputationStats struct {
	TotalComputations int64
	TotalTime         time.Duration
	AlgorithmCounts   map[string]int64
	AlgorithmTimes    map[string]time.Duration
	CacheHits         int64
	CacheMisses       int64
}

// NewComputationalEngine creates a new computational engine
func NewComputationalEngine() *ComputationalEngine {
	return &ComputationalEngine{
		rand:  rand.New(rand.NewSource(time.Now().UnixNano())),
		cache: make(map[string]interface{}),
		stats: ComputationStats{
			AlgorithmCounts: make(map[string]int64),
			AlgorithmTimes:  make(map[string]time.Duration),
		},
	}
}

// Mathematical Algorithms

// CalculatePrimes calculates prime numbers using Sieve of Eratosthenes
func (ce *ComputationalEngine) CalculatePrimes(limit int) []int {
	start := time.Now()
	defer ce.updateStats("primes", start)

	cacheKey := fmt.Sprintf("primes_%d", limit)
	ce.cacheMutex.RLock()
	if cached, exists := ce.cache[cacheKey]; exists {
		ce.cacheMutex.RUnlock()
		ce.statsMutex.Lock()
		ce.stats.CacheHits++
		ce.statsMutex.Unlock()
		return cached.([]int)
	}
	ce.cacheMutex.RUnlock()
	ce.statsMutex.Lock()
	ce.stats.CacheMisses++
	ce.statsMutex.Unlock()

	if limit < 2 {
		return []int{}
	}

	// Sieve of Eratosthenes
	sieve := make([]bool, limit+1)
	for i := 2; i <= limit; i++ {
		sieve[i] = true
	}

	for i := 2; i*i <= limit; i++ {
		if sieve[i] {
			for j := i * i; j <= limit; j += i {
				sieve[j] = false
			}
		}
	}

	primes := make([]int, 0)
	for i := 2; i <= limit; i++ {
		if sieve[i] {
			primes = append(primes, i)
		}
	}

	ce.cacheMutex.Lock()
	ce.cache[cacheKey] = primes
	ce.cacheMutex.Unlock()
	return primes
}

// CalculateFibonacci calculates Fibonacci numbers using dynamic programming
func (ce *ComputationalEngine) CalculateFibonacci(n int) []int {
	start := time.Now()
	defer ce.updateStats("fibonacci", start)

	cacheKey := fmt.Sprintf("fib_%d", n)
	ce.cacheMutex.RLock()
	if cached, exists := ce.cache[cacheKey]; exists {
		ce.cacheMutex.RUnlock()
		ce.statsMutex.Lock()
		ce.stats.CacheHits++
		ce.statsMutex.Unlock()
		return cached.([]int)
	}
	ce.cacheMutex.RUnlock()
	ce.statsMutex.Lock()
	ce.stats.CacheMisses++
	ce.statsMutex.Unlock()

	if n <= 0 {
		return []int{}
	}
	if n == 1 {
		return []int{0}
	}
	if n == 2 {
		return []int{0, 1}
	}

	fib := make([]int, n)
	fib[0] = 0
	fib[1] = 1

	for i := 2; i < n; i++ {
		fib[i] = fib[i-1] + fib[i-2]
	}

	ce.cacheMutex.Lock()
	ce.cache[cacheKey] = fib
	ce.cacheMutex.Unlock()
	return fib
}

// CalculateFactorial calculates factorial using iterative approach
func (ce *ComputationalEngine) CalculateFactorial(n int) *big.Int {
	start := time.Now()
	defer ce.updateStats("factorial", start)

	if n < 0 {
		return big.NewInt(0)
	}
	if n == 0 || n == 1 {
		return big.NewInt(1)
	}

	result := big.NewInt(1)
	for i := 2; i <= n; i++ {
		result.Mul(result, big.NewInt(int64(i)))
	}

	return result
}

// CalculateGCD calculates Greatest Common Divisor using Euclidean algorithm
func (ce *ComputationalEngine) CalculateGCD(a, b int) int {
	start := time.Now()
	defer ce.updateStats("gcd", start)

	for b != 0 {
		a, b = b, a%b
	}
	return a
}

// CalculateLCM calculates Least Common Multiple
func (ce *ComputationalEngine) CalculateLCM(a, b int) int {
	start := time.Now()
	defer ce.updateStats("lcm", start)

	return (a * b) / ce.CalculateGCD(a, b)
}

// Sorting Algorithms

// QuickSort implements quicksort algorithm
func (ce *ComputationalEngine) QuickSort(arr []int) []int {
	start := time.Now()
	defer ce.updateStats("quicksort", start)

	if len(arr) <= 1 {
		return arr
	}

	// Create a copy to avoid modifying original
	result := make([]int, len(arr))
	copy(result, arr)

	ce.quickSortRecursive(result, 0, len(result)-1)
	return result
}

func (ce *ComputationalEngine) quickSortRecursive(arr []int, low, high int) {
	if low < high {
		pi := ce.partition(arr, low, high)
		ce.quickSortRecursive(arr, low, pi-1)
		ce.quickSortRecursive(arr, pi+1, high)
	}
}

func (ce *ComputationalEngine) partition(arr []int, low, high int) int {
	pivot := arr[high]
	i := low - 1

	for j := low; j < high; j++ {
		if arr[j] < pivot {
			i++
			arr[i], arr[j] = arr[j], arr[i]
		}
	}
	arr[i+1], arr[high] = arr[high], arr[i+1]
	return i + 1
}

// MergeSort implements merge sort algorithm
func (ce *ComputationalEngine) MergeSort(arr []int) []int {
	start := time.Now()
	defer ce.updateStats("mergesort", start)

	if len(arr) <= 1 {
		return arr
	}

	// Create a copy to avoid modifying original
	result := make([]int, len(arr))
	copy(result, arr)

	return ce.mergeSortRecursive(result)
}

func (ce *ComputationalEngine) mergeSortRecursive(arr []int) []int {
	if len(arr) <= 1 {
		return arr
	}

	mid := len(arr) / 2
	left := ce.mergeSortRecursive(arr[:mid])
	right := ce.mergeSortRecursive(arr[mid:])

	return ce.merge(left, right)
}

func (ce *ComputationalEngine) merge(left, right []int) []int {
	result := make([]int, 0, len(left)+len(right))
	i, j := 0, 0

	for i < len(left) && j < len(right) {
		if left[i] <= right[j] {
			result = append(result, left[i])
			i++
		} else {
			result = append(result, right[j])
			j++
		}
	}

	// Append remaining elements
	result = append(result, left[i:]...)
	result = append(result, right[j:]...)

	return result
}

// HeapSort implements heap sort algorithm
func (ce *ComputationalEngine) HeapSort(arr []int) []int {
	start := time.Now()
	defer ce.updateStats("heapsort", start)

	if len(arr) <= 1 {
		return arr
	}

	// Create a copy to avoid modifying original
	result := make([]int, len(arr))
	copy(result, arr)

	n := len(result)

	// Build max heap
	for i := n/2 - 1; i >= 0; i-- {
		ce.heapify(result, n, i)
	}

	// Extract elements from heap one by one
	for i := n - 1; i > 0; i-- {
		result[0], result[i] = result[i], result[0]
		ce.heapify(result, i, 0)
	}

	return result
}

func (ce *ComputationalEngine) heapify(arr []int, n, i int) {
	largest := i
	left := 2*i + 1
	right := 2*i + 2

	if left < n && arr[left] > arr[largest] {
		largest = left
	}

	if right < n && arr[right] > arr[largest] {
		largest = right
	}

	if largest != i {
		arr[i], arr[largest] = arr[largest], arr[i]
		ce.heapify(arr, n, largest)
	}
}

// Search Algorithms

// BinarySearch implements binary search algorithm
func (ce *ComputationalEngine) BinarySearch(arr []int, target int) int {
	start := time.Now()
	defer ce.updateStats("binary_search", start)

	left, right := 0, len(arr)-1

	for left <= right {
		mid := left + (right-left)/2

		if arr[mid] == target {
			return mid
		} else if arr[mid] < target {
			left = mid + 1
		} else {
			right = mid - 1
		}
	}

	return -1 // Not found
}

// LinearSearch implements linear search algorithm
func (ce *ComputationalEngine) LinearSearch(arr []int, target int) int {
	start := time.Now()
	defer ce.updateStats("linear_search", start)

	for i, val := range arr {
		if val == target {
			return i
		}
	}
	return -1 // Not found
}

// Graph Algorithms

// Graph represents a simple graph structure
type Graph struct {
	Vertices int
	Edges    [][]int
}

// NewGraph creates a new graph
func NewGraph(vertices int) *Graph {
	return &Graph{
		Vertices: vertices,
		Edges:    make([][]int, vertices),
	}
}

// AddEdge adds an edge to the graph
func (g *Graph) AddEdge(from, to int) {
	if from >= 0 && from < g.Vertices && to >= 0 && to < g.Vertices {
		g.Edges[from] = append(g.Edges[from], to)
	}
}

// DFS performs Depth-First Search
func (ce *ComputationalEngine) DFS(graph *Graph, start int) []int {
	start_time := time.Now()
	defer ce.updateStats("dfs", start_time)

	// Bounds checking
	if start < 0 || start >= graph.Vertices {
		return []int{}
	}

	visited := make([]bool, graph.Vertices)
	result := make([]int, 0)

	ce.dfsRecursive(graph, start, visited, &result)
	return result
}

func (ce *ComputationalEngine) dfsRecursive(graph *Graph, vertex int, visited []bool, result *[]int) {
	visited[vertex] = true
	*result = append(*result, vertex)

	for _, neighbor := range graph.Edges[vertex] {
		if neighbor >= 0 && neighbor < graph.Vertices && !visited[neighbor] {
			ce.dfsRecursive(graph, neighbor, visited, result)
		}
	}
}

// BFS performs Breadth-First Search
func (ce *ComputationalEngine) BFS(graph *Graph, start int) []int {
	start_time := time.Now()
	defer ce.updateStats("bfs", start_time)

	// Bounds checking
	if start < 0 || start >= graph.Vertices {
		return []int{}
	}

	visited := make([]bool, graph.Vertices)
	queue := []int{start}
	result := make([]int, 0)

	visited[start] = true

	for len(queue) > 0 {
		vertex := queue[0]
		queue = queue[1:]
		result = append(result, vertex)

		for _, neighbor := range graph.Edges[vertex] {
			if neighbor >= 0 && neighbor < graph.Vertices && !visited[neighbor] {
				visited[neighbor] = true
				queue = append(queue, neighbor)
			}
		}
	}

	return result
}

// String Processing Algorithms

// KMPSearch implements Knuth-Morris-Pratt string searching algorithm
func (ce *ComputationalEngine) KMPSearch(text, pattern string) []int {
	start := time.Now()
	defer ce.updateStats("kmp_search", start)

	if len(pattern) == 0 {
		return []int{}
	}

	// Build failure function
	failure := ce.buildKMPFailureFunction(pattern)
	matches := make([]int, 0)

	i, j := 0, 0
	for i < len(text) {
		if text[i] == pattern[j] {
			i++
			j++
		}

		if j == len(pattern) {
			matches = append(matches, i-j)
			j = failure[j-1]
		} else if i < len(text) && text[i] != pattern[j] {
			if j != 0 {
				j = failure[j-1]
			} else {
				i++
			}
		}
	}

	return matches
}

func (ce *ComputationalEngine) buildKMPFailureFunction(pattern string) []int {
	failure := make([]int, len(pattern))
	j := 0

	for i := 1; i < len(pattern); i++ {
		for j > 0 && pattern[i] != pattern[j] {
			j = failure[j-1]
		}

		if pattern[i] == pattern[j] {
			j++
		}

		failure[i] = j
	}

	return failure
}

// LevenshteinDistance calculates edit distance between two strings
func (ce *ComputationalEngine) LevenshteinDistance(s1, s2 string) int {
	start := time.Now()
	defer ce.updateStats("levenshtein", start)

	m, n := len(s1), len(s2)
	dp := make([][]int, m+1)
	for i := range dp {
		dp[i] = make([]int, n+1)
	}

	// Initialize base cases
	for i := 0; i <= m; i++ {
		dp[i][0] = i
	}
	for j := 0; j <= n; j++ {
		dp[0][j] = j
	}

	// Fill the DP table
	for i := 1; i <= m; i++ {
		for j := 1; j <= n; j++ {
			if s1[i-1] == s2[j-1] {
				dp[i][j] = dp[i-1][j-1]
			} else {
				dp[i][j] = 1 + min(dp[i-1][j], dp[i][j-1], dp[i-1][j-1])
			}
		}
	}

	return dp[m][n]
}

// Numerical Algorithms

// MonteCarloPI estimates PI using Monte Carlo method
func (ce *ComputationalEngine) MonteCarloPI(iterations int) float64 {
	start := time.Now()
	defer ce.updateStats("monte_carlo_pi", start)

	insideCircle := 0

	for i := 0; i < iterations; i++ {
		ce.randMutex.Lock()
		x := ce.rand.Float64()*2 - 1 // Random number between -1 and 1
		y := ce.rand.Float64()*2 - 1 // Random number between -1 and 1
		ce.randMutex.Unlock()

		if x*x+y*y <= 1 {
			insideCircle++
		}
	}

	return 4.0 * float64(insideCircle) / float64(iterations)
}

// NewtonRaphson finds root using Newton-Raphson method
func (ce *ComputationalEngine) NewtonRaphson(f, df func(float64) float64, x0 float64, tolerance float64, maxIterations int) float64 {
	start := time.Now()
	defer ce.updateStats("newton_raphson", start)

	x := x0
	for i := 0; i < maxIterations; i++ {
		fx := f(x)
		if math.Abs(fx) < tolerance {
			return x
		}

		dfx := df(x)
		if math.Abs(dfx) < 1e-12 {
			break // Avoid division by zero
		}

		x = x - fx/dfx
	}

	return x
}

// Matrix Operations

// Matrix represents a 2D matrix
type Matrix [][]float64

// NewMatrix creates a new matrix
func NewMatrix(rows, cols int) Matrix {
	matrix := make(Matrix, rows)
	for i := range matrix {
		matrix[i] = make([]float64, cols)
	}
	return matrix
}

// MatrixMultiply multiplies two matrices
func (ce *ComputationalEngine) MatrixMultiply(a, b Matrix) Matrix {
	start := time.Now()
	defer ce.updateStats("matrix_multiply", start)

	if len(a[0]) != len(b) {
		return nil // Invalid dimensions
	}

	rows, cols := len(a), len(b[0])
	result := NewMatrix(rows, cols)

	for i := 0; i < rows; i++ {
		for j := 0; j < cols; j++ {
			for k := 0; k < len(b); k++ {
				result[i][j] += a[i][k] * b[k][j]
			}
		}
	}

	return result
}

// MatrixTranspose transposes a matrix
func (ce *ComputationalEngine) MatrixTranspose(matrix Matrix) Matrix {
	start := time.Now()
	defer ce.updateStats("matrix_transpose", start)

	rows, cols := len(matrix), len(matrix[0])
	result := NewMatrix(cols, rows)

	for i := 0; i < rows; i++ {
		for j := 0; j < cols; j++ {
			result[j][i] = matrix[i][j]
		}
	}

	return result
}

// Cryptographic Algorithms

// SimpleHash implements a simple hash function
func (ce *ComputationalEngine) SimpleHash(data string) uint32 {
	start := time.Now()
	defer ce.updateStats("simple_hash", start)

	hash := uint32(5381)
	for _, char := range data {
		hash = ((hash << 5) + hash) + uint32(char)
	}
	return hash
}

// CaesarCipher implements Caesar cipher encryption/decryption
func (ce *ComputationalEngine) CaesarCipher(text string, shift int, encrypt bool) string {
	start := time.Now()
	defer ce.updateStats("caesar_cipher", start)

	if !encrypt {
		shift = -shift
	}

	result := make([]rune, len(text))
	for i, char := range text {
		if char >= 'A' && char <= 'Z' {
			result[i] = 'A' + (char-'A'+rune(shift)+26)%26
		} else if char >= 'a' && char <= 'z' {
			result[i] = 'a' + (char-'a'+rune(shift)+26)%26
		} else {
			result[i] = char
		}
	}

	return string(result)
}

// Complex Algorithms

// TravelingSalesman solves TSP using brute force (for small instances)
func (ce *ComputationalEngine) TravelingSalesman(distances [][]float64) ([]int, float64) {
	start := time.Now()
	defer ce.updateStats("tsp", start)

	n := len(distances)
	if n <= 1 {
		return []int{0}, 0
	}

	// Generate all permutations of cities (excluding starting city)
	cities := make([]int, n-1)
	for i := 1; i < n; i++ {
		cities[i-1] = i
	}

	bestPath := make([]int, n+1)
	bestDistance := math.Inf(1)

	// Try all permutations
	ce.tspPermutations(cities, 0, distances, &bestPath, &bestDistance)

	return bestPath, bestDistance
}

func (ce *ComputationalEngine) tspPermutations(cities []int, index int, distances [][]float64, bestPath *[]int, bestDistance *float64) {
	if index == len(cities) {
		// Calculate total distance for this permutation
		distance := distances[0][cities[0]] // Start to first city
		for i := 0; i < len(cities)-1; i++ {
			distance += distances[cities[i]][cities[i+1]]
		}
		distance += distances[cities[len(cities)-1]][0] // Last city back to start

		if distance < *bestDistance {
			*bestDistance = distance
			(*bestPath)[0] = 0
			copy((*bestPath)[1:], cities)
			(*bestPath)[len(*bestPath)-1] = 0
		}
		return
	}

	for i := index; i < len(cities); i++ {
		cities[index], cities[i] = cities[i], cities[index]
		ce.tspPermutations(cities, index+1, distances, bestPath, bestDistance)
		cities[index], cities[i] = cities[i], cities[index] // backtrack
	}
}

// KnapsackDP solves 0/1 Knapsack problem using dynamic programming
func (ce *ComputationalEngine) KnapsackDP(weights []int, values []int, capacity int) (int, []int) {
	start := time.Now()
	defer ce.updateStats("knapsack", start)

	n := len(weights)
	dp := make([][]int, n+1)
	for i := range dp {
		dp[i] = make([]int, capacity+1)
	}

	// Fill the DP table
	for i := 1; i <= n; i++ {
		for w := 1; w <= capacity; w++ {
			if weights[i-1] <= w {
				dp[i][w] = max(dp[i-1][w], dp[i-1][w-weights[i-1]]+values[i-1])
			} else {
				dp[i][w] = dp[i-1][w]
			}
		}
	}

	// Backtrack to find selected items
	selected := make([]int, 0)
	w := capacity
	for i := n; i > 0 && w > 0; i-- {
		if dp[i][w] != dp[i-1][w] {
			selected = append(selected, i-1)
			w -= weights[i-1]
		}
	}

	return dp[n][capacity], selected
}

// Utility Functions

func min(a, b, c int) int {
	if a < b {
		if a < c {
			return a
		}
		return c
	}
	if b < c {
		return b
	}
	return c
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}

// updateStats updates algorithm execution statistics
func (ce *ComputationalEngine) updateStats(algorithm string, start time.Time) {
	duration := time.Since(start)
	ce.statsMutex.Lock()
	ce.stats.TotalComputations++
	ce.stats.TotalTime += duration
	ce.stats.AlgorithmCounts[algorithm]++
	ce.stats.AlgorithmTimes[algorithm] += duration
	ce.statsMutex.Unlock()
}

// GetStats returns computation statistics
func (ce *ComputationalEngine) GetStats() ComputationStats {
	ce.statsMutex.Lock()
	defer ce.statsMutex.Unlock()
	// Return a copy to prevent race conditions
	stats := ComputationStats{
		TotalComputations: ce.stats.TotalComputations,
		TotalTime:         ce.stats.TotalTime,
		CacheHits:         ce.stats.CacheHits,
		CacheMisses:       ce.stats.CacheMisses,
		AlgorithmCounts:   make(map[string]int64),
		AlgorithmTimes:    make(map[string]time.Duration),
	}
	for k, v := range ce.stats.AlgorithmCounts {
		stats.AlgorithmCounts[k] = v
	}
	for k, v := range ce.stats.AlgorithmTimes {
		stats.AlgorithmTimes[k] = v
	}
	return stats
}

// ClearCache clears the computation cache
func (ce *ComputationalEngine) ClearCache() {
	ce.cache = make(map[string]interface{})
	ce.stats.CacheHits = 0
	ce.stats.CacheMisses = 0
}

// Complex Workload Functions

// RunMixedWorkload executes a mixed computational workload
func (ce *ComputationalEngine) RunMixedWorkload(iterations int) {
	for i := 0; i < iterations; i++ {
		// Mathematical computations
		ce.CalculatePrimes(1000 + i%500)
		ce.CalculateFibonacci(30 + i%20)

		// Sorting operations
		data := ce.generateRandomArray(1000 + i%1000)
		ce.QuickSort(data)
		ce.MergeSort(data)

		// Search operations
		sortedData := make([]int, len(data))
		copy(sortedData, data)
		sort.Ints(sortedData)
		ce.randMutex.Lock()
		target := data[ce.rand.Intn(len(data))]
		ce.randMutex.Unlock()
		ce.BinarySearch(sortedData, target)

		// String processing
		text := ce.generateRandomString(1000)
		ce.randMutex.Lock()
		start := ce.rand.Intn(len(text)-10)
		patternLen := 5 + ce.rand.Intn(6) // Pattern length between 5-10
		ce.randMutex.Unlock()
		if start+patternLen > len(text) {
			start = len(text) - patternLen
		}
		pattern := text[start:start+patternLen]
		ce.KMPSearch(text, pattern)

		// Numerical computations
		if i%10 == 0 {
			ce.MonteCarloPI(10000)
		}

		// Matrix operations
		if i%20 == 0 {
			matrixA := ce.generateRandomMatrix(50, 50)
			matrixB := ce.generateRandomMatrix(50, 50)
			ce.MatrixMultiply(matrixA, matrixB)
		}

		// Graph algorithms
		if i%30 == 0 {
			graph := ce.generateRandomGraph(20, 40)
			ce.DFS(graph, 0)
			ce.BFS(graph, 0)
		}
	}
}

// Helper functions for workload generation

func (ce *ComputationalEngine) generateRandomArray(size int) []int {
	arr := make([]int, size)
	for i := range arr {
		ce.randMutex.Lock()
		arr[i] = ce.rand.Intn(10000)
		ce.randMutex.Unlock()
	}
	return arr
}

func (ce *ComputationalEngine) generateRandomString(length int) string {
	const charset = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
	b := make([]byte, length)
	for i := range b {
		ce.randMutex.Lock()
		b[i] = charset[ce.rand.Intn(len(charset))]
		ce.randMutex.Unlock()
	}
	return string(b)
}

func (ce *ComputationalEngine) generateRandomMatrix(rows, cols int) Matrix {
	matrix := NewMatrix(rows, cols)
	for i := 0; i < rows; i++ {
		for j := 0; j < cols; j++ {
			ce.randMutex.Lock()
			matrix[i][j] = ce.rand.Float64() * 100
			ce.randMutex.Unlock()
		}
	}
	return matrix
}

func (ce *ComputationalEngine) generateRandomGraph(vertices, edges int) *Graph {
	graph := NewGraph(vertices)
	for i := 0; i < edges; i++ {
		ce.randMutex.Lock()
		from := ce.rand.Intn(vertices)
		to := ce.rand.Intn(vertices)
		ce.randMutex.Unlock()
		if from != to {
			graph.AddEdge(from, to)
		}
	}
	return graph
}

// Benchmark functions for performance testing

// BenchmarkAlgorithm benchmarks a specific algorithm
func (ce *ComputationalEngine) BenchmarkAlgorithm(algorithmName string, iterations int) time.Duration {
	start := time.Now()

	switch algorithmName {
	case "primes":
		for i := 0; i < iterations; i++ {
			ce.CalculatePrimes(1000)
		}
	case "fibonacci":
		for i := 0; i < iterations; i++ {
			ce.CalculateFibonacci(30)
		}
	case "quicksort":
		for i := 0; i < iterations; i++ {
			data := ce.generateRandomArray(1000)
			ce.QuickSort(data)
		}
	case "mergesort":
		for i := 0; i < iterations; i++ {
			data := ce.generateRandomArray(1000)
			ce.MergeSort(data)
		}
	case "matrix_multiply":
		for i := 0; i < iterations; i++ {
			matrixA := ce.generateRandomMatrix(50, 50)
			matrixB := ce.generateRandomMatrix(50, 50)
			ce.MatrixMultiply(matrixA, matrixB)
		}
	default:
		// Run mixed workload
		ce.RunMixedWorkload(iterations)
	}

	return time.Since(start)
}

// PrintStats prints detailed statistics
func (ce *ComputationalEngine) PrintStats() {
	fmt.Printf("Computational Engine Statistics:\n")
	fmt.Printf("  Total Computations: %d\n", ce.stats.TotalComputations)
	fmt.Printf("  Total Time: %v\n", ce.stats.TotalTime)
	fmt.Printf("  Cache Hits: %d\n", ce.stats.CacheHits)
	fmt.Printf("  Cache Misses: %d\n", ce.stats.CacheMisses)

	if ce.stats.CacheHits+ce.stats.CacheMisses > 0 {
		hitRate := float64(ce.stats.CacheHits) / float64(ce.stats.CacheHits+ce.stats.CacheMisses) * 100
		fmt.Printf("  Cache Hit Rate: %.2f%%\n", hitRate)
	}

	fmt.Printf("\n  Algorithm Breakdown:\n")
	for algorithm, count := range ce.stats.AlgorithmCounts {
		avgTime := ce.stats.AlgorithmTimes[algorithm] / time.Duration(count)
		fmt.Printf("    %s: %d executions, avg time: %v\n", algorithm, count, avgTime)
	}
}

// ResetStats resets all statistics
func (ce *ComputationalEngine) ResetStats() {
	ce.stats = ComputationStats{
		AlgorithmCounts: make(map[string]int64),
		AlgorithmTimes:  make(map[string]time.Duration),
	}
}
