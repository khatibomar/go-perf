// Package advanced_benchmarks demonstrates advanced benchmarking techniques in Go
// including custom metrics, conditional benchmarks, setup/teardown, and integration testing
package main

import (
	"context"
	"crypto/rand"
	"database/sql"
	"fmt"
	"io"
	"log"
	"math"
	"net/http"
	"net/http/httptest"
	"os"
	"runtime"
	"sync"
	"sync/atomic"
	"time"

	_ "github.com/mattn/go-sqlite3"
)

// AdvancedBenchmarkSuite provides a framework for complex benchmarking scenarios
type AdvancedBenchmarkSuite struct {
	db     *sql.DB
	server *httptest.Server
	cache  *LRUCache
	metrics *MetricsCollector
}

// MetricsCollector tracks custom performance metrics
type MetricsCollector struct {
	mu              sync.RWMutex
	operations      int64
	totalLatency    time.Duration
	errorCount      int64
	throughputStart time.Time
	customMetrics   map[string]float64
}

// NewMetricsCollector creates a new metrics collector
func NewMetricsCollector() *MetricsCollector {
	return &MetricsCollector{
		customMetrics:   make(map[string]float64),
		throughputStart: time.Now(),
	}
}

// RecordOperation records a completed operation
func (mc *MetricsCollector) RecordOperation(latency time.Duration, success bool) {
	mc.mu.Lock()
	defer mc.mu.Unlock()
	
	atomic.AddInt64(&mc.operations, 1)
	mc.totalLatency += latency
	
	if !success {
		atomic.AddInt64(&mc.errorCount, 1)
	}
}

// RecordCustomMetric records a custom metric
func (mc *MetricsCollector) RecordCustomMetric(name string, value float64) {
	mc.mu.Lock()
	defer mc.mu.Unlock()
	mc.customMetrics[name] = value
}

// GetMetrics returns current metrics
func (mc *MetricsCollector) GetMetrics() (ops int64, avgLatency time.Duration, errorRate float64, throughput float64) {
	mc.mu.RLock()
	defer mc.mu.RUnlock()
	
	ops = atomic.LoadInt64(&mc.operations)
	errors := atomic.LoadInt64(&mc.errorCount)
	
	if ops > 0 {
		avgLatency = mc.totalLatency / time.Duration(ops)
		errorRate = float64(errors) / float64(ops)
		throughput = float64(ops) / time.Since(mc.throughputStart).Seconds()
	}
	
	return
}

// LRUCache implements a simple LRU cache for benchmarking
type LRUCache struct {
	mu       sync.RWMutex
	capacity int
	cache    map[string]*CacheNode
	head     *CacheNode
	tail     *CacheNode
	hits     int64
	misses   int64
}

// CacheNode represents a cache entry
type CacheNode struct {
	key   string
	value interface{}
	prev  *CacheNode
	next  *CacheNode
}

// NewLRUCache creates a new LRU cache
func NewLRUCache(capacity int) *LRUCache {
	cache := &LRUCache{
		capacity: capacity,
		cache:    make(map[string]*CacheNode),
	}
	
	// Initialize dummy head and tail
	cache.head = &CacheNode{}
	cache.tail = &CacheNode{}
	cache.head.next = cache.tail
	cache.tail.prev = cache.head
	
	return cache
}

// Get retrieves a value from the cache
func (lru *LRUCache) Get(key string) (interface{}, bool) {
	lru.mu.Lock()
	defer lru.mu.Unlock()
	
	if node, exists := lru.cache[key]; exists {
		lru.moveToHead(node)
		atomic.AddInt64(&lru.hits, 1)
		return node.value, true
	}
	
	atomic.AddInt64(&lru.misses, 1)
	return nil, false
}

// Put stores a value in the cache
func (lru *LRUCache) Put(key string, value interface{}) {
	lru.mu.Lock()
	defer lru.mu.Unlock()
	
	if node, exists := lru.cache[key]; exists {
		node.value = value
		lru.moveToHead(node)
		return
	}
	
	node := &CacheNode{key: key, value: value}
	lru.cache[key] = node
	lru.addToHead(node)
	
	if len(lru.cache) > lru.capacity {
		tail := lru.removeTail()
		delete(lru.cache, tail.key)
	}
}

// GetHitRate returns the cache hit rate
func (lru *LRUCache) GetHitRate() float64 {
	hits := atomic.LoadInt64(&lru.hits)
	misses := atomic.LoadInt64(&lru.misses)
	total := hits + misses
	
	if total == 0 {
		return 0
	}
	
	return float64(hits) / float64(total)
}

// Helper methods for LRU cache
func (lru *LRUCache) moveToHead(node *CacheNode) {
	lru.removeNode(node)
	lru.addToHead(node)
}

func (lru *LRUCache) removeNode(node *CacheNode) {
	node.prev.next = node.next
	node.next.prev = node.prev
}

func (lru *LRUCache) addToHead(node *CacheNode) {
	node.prev = lru.head
	node.next = lru.head.next
	lru.head.next.prev = node
	lru.head.next = node
}

func (lru *LRUCache) removeTail() *CacheNode {
	last := lru.tail.prev
	lru.removeNode(last)
	return last
}

// DatabaseWorkload simulates database operations for benchmarking
type DatabaseWorkload struct {
	db *sql.DB
}

// NewDatabaseWorkload creates a new database workload
func NewDatabaseWorkload() (*DatabaseWorkload, error) {
	db, err := sql.Open("sqlite3", ":memory:")
	if err != nil {
		return nil, err
	}
	
	// Create test table
	_, err = db.Exec(`
		CREATE TABLE users (
			id INTEGER PRIMARY KEY,
			name TEXT NOT NULL,
			email TEXT UNIQUE NOT NULL,
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP
		)
	`)
	if err != nil {
		return nil, err
	}
	
	// Insert test data
	for i := 0; i < 1000; i++ {
		_, err = db.Exec("INSERT INTO users (name, email) VALUES (?, ?)",
			fmt.Sprintf("User%d", i),
			fmt.Sprintf("user%d@example.com", i))
		if err != nil {
			return nil, err
		}
	}
	
	return &DatabaseWorkload{db: db}, nil
}

// QueryUsers performs a database query
func (dw *DatabaseWorkload) QueryUsers(limit int) ([]User, error) {
	rows, err := dw.db.Query("SELECT id, name, email FROM users LIMIT ?", limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	
	var users []User
	for rows.Next() {
		var user User
		err := rows.Scan(&user.ID, &user.Name, &user.Email)
		if err != nil {
			return nil, err
		}
		users = append(users, user)
	}
	
	return users, nil
}

// InsertUser inserts a new user
func (dw *DatabaseWorkload) InsertUser(user User) error {
	_, err := dw.db.Exec("INSERT INTO users (name, email) VALUES (?, ?)",
		user.Name, user.Email)
	return err
}

// UpdateUser updates an existing user
func (dw *DatabaseWorkload) UpdateUser(user User) error {
	_, err := dw.db.Exec("UPDATE users SET name = ?, email = ? WHERE id = ?",
		user.Name, user.Email, user.ID)
	return err
}

// DeleteUser deletes a user
func (dw *DatabaseWorkload) DeleteUser(id int) error {
	_, err := dw.db.Exec("DELETE FROM users WHERE id = ?", id)
	return err
}

// Close closes the database connection
func (dw *DatabaseWorkload) Close() error {
	return dw.db.Close()
}

// User represents a user entity
type User struct {
	ID    int    `json:"id"`
	Name  string `json:"name"`
	Email string `json:"email"`
}

// HTTPWorkload simulates HTTP operations for benchmarking
type HTTPWorkload struct {
	server *httptest.Server
	client *http.Client
}

// NewHTTPWorkload creates a new HTTP workload
func NewHTTPWorkload() *HTTPWorkload {
	mux := http.NewServeMux()
	
	// Simple GET endpoint
	mux.HandleFunc("/api/users", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}
		
		// Simulate some processing time
		time.Sleep(time.Millisecond * 10)
		
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`[{"id":1,"name":"John","email":"john@example.com"}]`))
	})
	
	// POST endpoint
	mux.HandleFunc("/api/users/create", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}
		
		// Simulate processing
		time.Sleep(time.Millisecond * 20)
		
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		w.Write([]byte(`{"id":2,"name":"Jane","email":"jane@example.com"}`))
	})
	
	// Slow endpoint for timeout testing
	mux.HandleFunc("/api/slow", func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(time.Second * 2)
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("slow response"))
	})
	
	server := httptest.NewServer(mux)
	client := &http.Client{
		Timeout: time.Second * 5,
	}
	
	return &HTTPWorkload{
		server: server,
		client: client,
	}
}

// GetUsers makes a GET request to fetch users
func (hw *HTTPWorkload) GetUsers() (*http.Response, error) {
	return hw.client.Get(hw.server.URL + "/api/users")
}

// CreateUser makes a POST request to create a user
func (hw *HTTPWorkload) CreateUser() (*http.Response, error) {
	return hw.client.Post(hw.server.URL+"/api/users/create", "application/json", nil)
}

// SlowRequest makes a request to the slow endpoint
func (hw *HTTPWorkload) SlowRequest() (*http.Response, error) {
	return hw.client.Get(hw.server.URL + "/api/slow")
}

// Close closes the HTTP workload
func (hw *HTTPWorkload) Close() {
	hw.server.Close()
}

// ComputeWorkload provides CPU-intensive operations for benchmarking
type ComputeWorkload struct{}

// PrimeCalculation calculates primes up to n using Sieve of Eratosthenes
func (cw *ComputeWorkload) PrimeCalculation(n int) []int {
	if n < 2 {
		return []int{}
	}
	
	sieve := make([]bool, n+1)
	for i := 2; i <= n; i++ {
		sieve[i] = true
	}
	
	for i := 2; i*i <= n; i++ {
		if sieve[i] {
			for j := i * i; j <= n; j += i {
				sieve[j] = false
			}
		}
	}
	
	var primes []int
	for i := 2; i <= n; i++ {
		if sieve[i] {
			primes = append(primes, i)
		}
	}
	
	return primes
}

// MatrixMultiplication performs matrix multiplication
func (cw *ComputeWorkload) MatrixMultiplication(a, b [][]float64) [][]float64 {
	rows := len(a)
	cols := len(b[0])
	common := len(b)
	
	result := make([][]float64, rows)
	for i := range result {
		result[i] = make([]float64, cols)
	}
	
	for i := 0; i < rows; i++ {
		for j := 0; j < cols; j++ {
			for k := 0; k < common; k++ {
				result[i][j] += a[i][k] * b[k][j]
			}
		}
	}
	
	return result
}

// FibonacciRecursive calculates Fibonacci number recursively (inefficient)
func (cw *ComputeWorkload) FibonacciRecursive(n int) int {
	if n <= 1 {
		return n
	}
	return cw.FibonacciRecursive(n-1) + cw.FibonacciRecursive(n-2)
}

// FibonacciIterative calculates Fibonacci number iteratively (efficient)
func (cw *ComputeWorkload) FibonacciIterative(n int) int {
	if n <= 1 {
		return n
	}
	
	a, b := 0, 1
	for i := 2; i <= n; i++ {
		a, b = b, a+b
	}
	return b
}

// CryptoWorkload provides cryptographic operations for benchmarking
type CryptoWorkload struct{}

// GenerateRandomData generates random data of specified size
func (cw *CryptoWorkload) GenerateRandomData(size int) ([]byte, error) {
	data := make([]byte, size)
	_, err := rand.Read(data)
	return data, err
}

// HashData performs repeated hashing (CPU intensive)
func (cw *CryptoWorkload) HashData(data []byte, iterations int) []byte {
	result := data
	for i := 0; i < iterations; i++ {
		// Simple hash function for demonstration
		hash := make([]byte, len(result))
		for j, b := range result {
			hash[j] = b ^ byte(i)
		}
		result = hash
	}
	return result
}

// MemoryWorkload provides memory-intensive operations for benchmarking
type MemoryWorkload struct{}

// AllocateSlices allocates multiple slices of varying sizes
func (mw *MemoryWorkload) AllocateSlices(count, maxSize int) [][]byte {
	slices := make([][]byte, count)
	for i := 0; i < count; i++ {
		size := (i % maxSize) + 1
		slices[i] = make([]byte, size)
		// Fill with data to prevent optimization
		for j := range slices[i] {
			slices[i][j] = byte(i + j)
		}
	}
	return slices
}

// AllocateMap creates a map with specified number of entries
func (mw *MemoryWorkload) AllocateMap(size int) map[string][]byte {
	m := make(map[string][]byte, size)
	for i := 0; i < size; i++ {
		key := fmt.Sprintf("key_%d", i)
		value := make([]byte, 100) // 100 bytes per entry
		for j := range value {
			value[j] = byte(i)
		}
		m[key] = value
	}
	return m
}

// ConcurrencyWorkload provides concurrent operations for benchmarking
type ConcurrencyWorkload struct{}

// ParallelWork executes work across multiple goroutines
func (cw *ConcurrencyWorkload) ParallelWork(workers int, workItems int, workFunc func(int) int) []int {
	workChan := make(chan int, workItems)
	resultChan := make(chan int, workItems)
	
	// Start workers
	var wg sync.WaitGroup
	for i := 0; i < workers; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for item := range workChan {
				result := workFunc(item)
				resultChan <- result
			}
		}()
	}
	
	// Send work
	go func() {
		for i := 0; i < workItems; i++ {
			workChan <- i
		}
		close(workChan)
	}()
	
	// Collect results
	go func() {
		wg.Wait()
		close(resultChan)
	}()
	
	var results []int
	for result := range resultChan {
		results = append(results, result)
	}
	
	return results
}

// ChannelThroughput measures channel throughput
func (cw *ConcurrencyWorkload) ChannelThroughput(bufferSize int, messages int) time.Duration {
	ch := make(chan int, bufferSize)
	start := time.Now()
	
	// Producer
	go func() {
		for i := 0; i < messages; i++ {
			ch <- i
		}
		close(ch)
	}()
	
	// Consumer
	count := 0
	for range ch {
		count++
	}
	
	return time.Since(start)
}

// BenchmarkEnvironment provides environment setup and teardown
type BenchmarkEnvironment struct {
	TempDir    string
	LogFile    *os.File
	OrigStdout *os.File
	OrigStderr *os.File
}

// SetupEnvironment prepares the benchmark environment
func SetupEnvironment() (*BenchmarkEnvironment, error) {
	// Create temporary directory
	tempDir, err := os.MkdirTemp("", "benchmark_*")
	if err != nil {
		return nil, err
	}
	
	// Create log file
	logFile, err := os.Create(tempDir + "/benchmark.log")
	if err != nil {
		os.RemoveAll(tempDir)
		return nil, err
	}
	
	// Redirect stdout/stderr for clean benchmark output
	origStdout := os.Stdout
	origStderr := os.Stderr
	
	return &BenchmarkEnvironment{
		TempDir:    tempDir,
		LogFile:    logFile,
		OrigStdout: origStdout,
		OrigStderr: origStderr,
	}, nil
}

// TeardownEnvironment cleans up the benchmark environment
func (be *BenchmarkEnvironment) TeardownEnvironment() error {
	// Restore stdout/stderr
	os.Stdout = be.OrigStdout
	os.Stderr = be.OrigStderr
	
	// Close log file
	if be.LogFile != nil {
		be.LogFile.Close()
	}
	
	// Remove temporary directory
	return os.RemoveAll(be.TempDir)
}

// ConditionalBenchmark provides conditional benchmark execution
type ConditionalBenchmark struct {
	SkipShort    bool
	SkipLong     bool
	SkipMemory   bool
	SkipNetwork  bool
	SkipDatabase bool
	MinCPUs      int
	MinMemory    int64 // in bytes
}

// ShouldRun determines if a benchmark should run based on conditions
func (cb *ConditionalBenchmark) ShouldRun(benchType string, isShort bool) bool {
	if isShort && cb.SkipShort {
		return false
	}
	
	if !isShort && cb.SkipLong {
		return false
	}
	
	switch benchType {
	case "memory":
		return !cb.SkipMemory
	case "network":
		return !cb.SkipNetwork
	case "database":
		return !cb.SkipDatabase
	default:
		return true
	}
}

// CheckSystemRequirements verifies system meets benchmark requirements
func (cb *ConditionalBenchmark) CheckSystemRequirements() error {
	if runtime.NumCPU() < cb.MinCPUs {
		return fmt.Errorf("insufficient CPUs: need %d, have %d", cb.MinCPUs, runtime.NumCPU())
	}
	
	var m runtime.MemStats
	runtime.ReadMemStats(&m)
	if int64(m.Sys) < cb.MinMemory {
		return fmt.Errorf("insufficient memory: need %d bytes, have %d bytes", cb.MinMemory, m.Sys)
	}
	
	return nil
}

// PerformanceRegression detects performance regressions
type PerformanceRegression struct {
	Baseline map[string]BenchmarkResult
	Current  map[string]BenchmarkResult
}

// BenchmarkResult stores benchmark results
type BenchmarkResult struct {
	Name         string
	Iterations   int
	NsPerOp      int64
	BytesPerOp   int64
	AllocsPerOp  int64
	MBPerSec     float64
	CustomMetrics map[string]float64
}

// DetectRegressions compares current results with baseline
func (pr *PerformanceRegression) DetectRegressions(threshold float64) []string {
	var regressions []string
	
	for name, current := range pr.Current {
		baseline, exists := pr.Baseline[name]
		if !exists {
			continue
		}
		
		// Check time regression
		timeRegression := float64(current.NsPerOp-baseline.NsPerOp) / float64(baseline.NsPerOp)
		if timeRegression > threshold {
			regressions = append(regressions, fmt.Sprintf("%s: time regression %.2f%%", name, timeRegression*100))
		}
		
		// Check memory regression
		memRegression := float64(current.BytesPerOp-baseline.BytesPerOp) / float64(baseline.BytesPerOp)
		if memRegression > threshold {
			regressions = append(regressions, fmt.Sprintf("%s: memory regression %.2f%%", name, memRegression*100))
		}
		
		// Check allocation regression
		allocRegression := float64(current.AllocsPerOp-baseline.AllocsPerOp) / float64(baseline.AllocsPerOp)
		if allocRegression > threshold {
			regressions = append(regressions, fmt.Sprintf("%s: allocation regression %.2f%%", name, allocRegression*100))
		}
	}
	
	return regressions
}

// BenchmarkProfiler integrates profiling with benchmarks
type BenchmarkProfiler struct {
	CPUProfile    io.Writer
	MemProfile    io.Writer
	BlockProfile  io.Writer
	MutexProfile  io.Writer
	TraceProfile  io.Writer
}

// StartProfiling begins profiling collection
func (bp *BenchmarkProfiler) StartProfiling() error {
	// Implementation would start various profilers
	// This is a simplified version
	log.Println("Starting profiling...")
	return nil
}

// StopProfiling ends profiling collection
func (bp *BenchmarkProfiler) StopProfiling() error {
	// Implementation would stop profilers and write data
	log.Println("Stopping profiling...")
	return nil
}

// ContextualBenchmark provides context-aware benchmarking
type ContextualBenchmark struct {
	ctx     context.Context
	cancel  context.CancelFunc
	timeout time.Duration
}

// NewContextualBenchmark creates a new contextual benchmark
func NewContextualBenchmark(timeout time.Duration) *ContextualBenchmark {
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	return &ContextualBenchmark{
		ctx:     ctx,
		cancel:  cancel,
		timeout: timeout,
	}
}

// RunWithContext executes a function with context
func (cb *ContextualBenchmark) RunWithContext(fn func(context.Context) error) error {
	return fn(cb.ctx)
}

// Cancel cancels the benchmark context
func (cb *ContextualBenchmark) Cancel() {
	cb.cancel()
}

// IsExpired checks if the context has expired
func (cb *ContextualBenchmark) IsExpired() bool {
	select {
	case <-cb.ctx.Done():
		return true
	default:
		return false
	}
}

// ParametricBenchmark provides parameterized benchmark execution
type ParametricBenchmark struct {
	Parameters map[string][]interface{}
	Results    map[string][]BenchmarkResult
}

// NewParametricBenchmark creates a new parametric benchmark
func NewParametricBenchmark() *ParametricBenchmark {
	return &ParametricBenchmark{
		Parameters: make(map[string][]interface{}),
		Results:    make(map[string][]BenchmarkResult),
	}
}

// AddParameter adds a parameter set for benchmarking
func (pb *ParametricBenchmark) AddParameter(name string, values []interface{}) {
	pb.Parameters[name] = values
}

// GenerateParameterCombinations generates all parameter combinations
func (pb *ParametricBenchmark) GenerateParameterCombinations() []map[string]interface{} {
	if len(pb.Parameters) == 0 {
		return []map[string]interface{}{{}}
	}
	
	var combinations []map[string]interface{}
	var keys []string
	var values [][]interface{}
	
	for k, v := range pb.Parameters {
		keys = append(keys, k)
		values = append(values, v)
	}
	
	// Generate cartesian product
	indices := make([]int, len(keys))
	for {
		combination := make(map[string]interface{})
		for i, key := range keys {
			combination[key] = values[i][indices[i]]
		}
		combinations = append(combinations, combination)
		
		// Increment indices
		carry := 1
		for i := len(indices) - 1; i >= 0 && carry > 0; i-- {
			indices[i] += carry
			if indices[i] >= len(values[i]) {
				indices[i] = 0
				carry = 1
			} else {
				carry = 0
			}
		}
		
		if carry > 0 {
			break
		}
	}
	
	return combinations
}

// Utility functions for benchmark data generation

// GenerateRandomMatrix creates a random matrix for testing
func GenerateRandomMatrix(rows, cols int) [][]float64 {
	if rows <= 0 || cols <= 0 {
		return nil
	}
	if rows*cols > 1000000 {
		fmt.Printf("Warning: Large matrix %dx%d may cause memory issues\n", rows, cols)
	}
	
	matrix := make([][]float64, rows)
	for i := range matrix {
		matrix[i] = make([]float64, cols)
		for j := range matrix[i] {
			matrix[i][j] = math.Sqrt(float64(i*cols + j))
		}
	}
	return matrix
}

// GenerateTestData creates test data of specified size
func GenerateTestData(size int) []int {
	if size <= 0 {
		return []int{}
	}
	if size > 10000000 {
		fmt.Printf("Warning: Large data size %d may cause memory issues\n", size)
	}
	
	data := make([]int, size)
	for i := range data {
		data[i] = i
	}
	return data
}

// GenerateRandomStrings creates random strings for testing
func GenerateRandomStrings(count, length int) []string {
	strings := make([]string, count)
	for i := range strings {
		data := make([]byte, length)
		for j := range data {
			data[j] = byte('a' + (i+j)%26)
		}
		strings[i] = string(data)
	}
	return strings
}

// main function for demonstration
func main() {
	// Validate runtime environment
	var m runtime.MemStats
	runtime.ReadMemStats(&m)
	if m.Sys < 100*1024*1024 { // Less than 100MB
		fmt.Println("Warning: Low system memory may affect benchmark results")
	}
	
	fmt.Println("Advanced Benchmarking Framework")
	fmt.Println("This package provides comprehensive benchmarking utilities for Go applications.")
	fmt.Println("Run 'go test -bench=.' to execute the benchmarks.")
	
	// Example usage
	metrics := NewMetricsCollector()
	start := time.Now()
	
	// Simulate some work
	time.Sleep(time.Millisecond * 100)
	
	metrics.RecordOperation(time.Since(start), true)
	ops, avgLatency, errorRate, throughput := metrics.GetMetrics()
	
	fmt.Printf("Operations: %d\n", ops)
	fmt.Printf("Average Latency: %v\n", avgLatency)
	fmt.Printf("Error Rate: %.2f%%\n", errorRate*100)
	fmt.Printf("Throughput: %.2f ops/sec\n", throughput)
}