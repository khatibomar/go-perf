// Package main demonstrates live application profiling with a web server.
// This file contains a web application with various endpoints that exhibit
// different performance characteristics for profiling analysis.
package main

import (
	"encoding/json"
	"fmt"
	"log"
	"math/rand"
	"net/http"
	_ "net/http/pprof" // Import pprof HTTP handlers
	"runtime"
	"strconv"
	"strings"
	"sync"
	"time"
)

// Server represents our web application
type Server struct {
	mu    sync.RWMutex
	data  map[string]interface{}
	cache map[string]CacheEntry
	stats ServerStats
}

// ServerStats tracks server metrics
type ServerStats struct {
	mu           sync.Mutex
	Requests     int64
	Errors       int64
	TotalLatency time.Duration
}

// CacheEntry represents a cached item
type CacheEntry struct {
	Value     interface{}
	Expiry    time.Time
	HitCount  int
	CreatedAt time.Time
}

// User represents a user entity
type User struct {
	ID       int    `json:"id"`
	Name     string `json:"name"`
	Email    string `json:"email"`
	Profile  UserProfile `json:"profile"`
	Settings map[string]interface{} `json:"settings"`
}

// UserProfile contains user profile information
type UserProfile struct {
	Age      int      `json:"age"`
	Location string   `json:"location"`
	Tags     []string `json:"tags"`
	Metadata map[string]string `json:"metadata"`
}

// NewServer creates a new server instance
func NewServer() *Server {
	return &Server{
		data:  make(map[string]interface{}),
		cache: make(map[string]CacheEntry),
	}
}

// cpuIntensiveHandler simulates CPU-intensive operations
func (s *Server) cpuIntensiveHandler(w http.ResponseWriter, r *http.Request) {
	start := time.Now()
	defer s.recordRequest(start)
	
	// Parse iterations parameter
	iterationsStr := r.URL.Query().Get("iterations")
	iterations := 1000000 // default
	if iterationsStr != "" {
		if parsed, err := strconv.Atoi(iterationsStr); err == nil {
			iterations = parsed
		}
	}
	
	// CPU-intensive computation
	result := s.performCPUIntensiveTask(iterations)
	
	response := map[string]interface{}{
		"result":     result,
		"iterations": iterations,
		"duration":   time.Since(start).String(),
	}
	
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// memoryIntensiveHandler simulates memory-intensive operations
func (s *Server) memoryIntensiveHandler(w http.ResponseWriter, r *http.Request) {
	start := time.Now()
	defer s.recordRequest(start)
	
	// Parse size parameter
	sizeStr := r.URL.Query().Get("size")
	size := 1000 // default
	if sizeStr != "" {
		if parsed, err := strconv.Atoi(sizeStr); err == nil {
			size = parsed
		}
	}
	
	// Memory-intensive operations
	data := s.performMemoryIntensiveTask(size)
	
	response := map[string]interface{}{
		"allocated_objects": len(data),
		"size_parameter":   size,
		"duration":         time.Since(start).String(),
		"memory_stats":     s.getMemoryStats(),
	}
	
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// inefficientHandler demonstrates various inefficient patterns
func (s *Server) inefficientHandler(w http.ResponseWriter, r *http.Request) {
	start := time.Now()
	defer s.recordRequest(start)
	
	// Inefficient string concatenation
	result := ""
	for i := 0; i < 1000; i++ {
		result += fmt.Sprintf("item_%d_", i)
	}
	
	// Inefficient slice operations
	var data []int
	for i := 0; i < 10000; i++ {
		data = append(data, i*i)
	}
	
	// Inefficient map operations
	mapData := make(map[string]interface{})
	for i := 0; i < 1000; i++ {
		key := fmt.Sprintf("key_%d", i)
		mapData[key] = map[string]interface{}{
			"value": i,
			"data":  make([]byte, 1024),
		}
	}
	
	response := map[string]interface{}{
		"string_length": len(result),
		"slice_length":  len(data),
		"map_size":      len(mapData),
		"duration":      time.Since(start).String(),
	}
	
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// databaseSimulationHandler simulates database operations with artificial latency
func (s *Server) databaseSimulationHandler(w http.ResponseWriter, r *http.Request) {
	start := time.Now()
	defer s.recordRequest(start)
	
	// Parse query parameters
	queryType := r.URL.Query().Get("type")
	if queryType == "" {
		queryType = "select"
	}
	
	recordsStr := r.URL.Query().Get("records")
	records := 100
	if recordsStr != "" {
		if parsed, err := strconv.Atoi(recordsStr); err == nil {
			records = parsed
		}
	}
	
	// Simulate database operation
	result := s.simulateDatabaseOperation(queryType, records)
	
	response := map[string]interface{}{
		"query_type": queryType,
		"records":    records,
		"result":     result,
		"duration":   time.Since(start).String(),
	}
	
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// cacheHandler demonstrates caching with potential issues
func (s *Server) cacheHandler(w http.ResponseWriter, r *http.Request) {
	start := time.Now()
	defer s.recordRequest(start)
	
	key := r.URL.Query().Get("key")
	if key == "" {
		key = "default"
	}
	
	ttlStr := r.URL.Query().Get("ttl")
	ttl := 60 * time.Second
	if ttlStr != "" {
		if parsed, err := time.ParseDuration(ttlStr); err == nil {
			ttl = parsed
		}
	}
	
	// Check cache first
	value, hit := s.getCachedValue(key)
	if !hit {
		// Cache miss - generate expensive value
		value = s.generateExpensiveValue(key)
		s.setCachedValue(key, value, ttl)
	}
	
	response := map[string]interface{}{
		"key":       key,
		"value":     value,
		"cache_hit": hit,
		"duration":  time.Since(start).String(),
		"cache_stats": s.getCacheStats(),
	}
	
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// concurrentHandler demonstrates concurrent operations
func (s *Server) concurrentHandler(w http.ResponseWriter, r *http.Request) {
	start := time.Now()
	defer s.recordRequest(start)
	
	workersStr := r.URL.Query().Get("workers")
	workers := 10
	if workersStr != "" {
		if parsed, err := strconv.Atoi(workersStr); err == nil {
			workers = parsed
		}
	}
	
	tasksStr := r.URL.Query().Get("tasks")
	tasks := 100
	if tasksStr != "" {
		if parsed, err := strconv.Atoi(tasksStr); err == nil {
			tasks = parsed
		}
	}
	
	// Perform concurrent work
	results := s.performConcurrentWork(workers, tasks)
	
	response := map[string]interface{}{
		"workers":    workers,
		"tasks":      tasks,
		"results":    len(results),
		"duration":   time.Since(start).String(),
		"goroutines": runtime.NumGoroutine(),
	}
	
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// leakyHandler demonstrates potential memory leaks
func (s *Server) leakyHandler(w http.ResponseWriter, r *http.Request) {
	start := time.Now()
	defer s.recordRequest(start)
	
	sizeStr := r.URL.Query().Get("size")
	size := 1000
	if sizeStr != "" {
		if parsed, err := strconv.Atoi(sizeStr); err == nil {
			size = parsed
		}
	}
	
	// Create potentially leaky data structures
	s.createLeakyStructures(size)
	
	response := map[string]interface{}{
		"size":         size,
		"duration":     time.Since(start).String(),
		"memory_stats": s.getMemoryStats(),
		"data_size":    len(s.data),
	}
	
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// statsHandler returns server statistics
func (s *Server) statsHandler(w http.ResponseWriter, r *http.Request) {
	s.stats.mu.Lock()
	stats := ServerStats{
		Requests:     s.stats.Requests,
		Errors:       s.stats.Errors,
		TotalLatency: s.stats.TotalLatency,
	}
	s.stats.mu.Unlock()
	
	avgLatency := time.Duration(0)
	if stats.Requests > 0 {
		avgLatency = stats.TotalLatency / time.Duration(stats.Requests)
	}
	
	response := map[string]interface{}{
		"requests":      stats.Requests,
		"errors":        stats.Errors,
		"avg_latency":   avgLatency.String(),
		"total_latency": stats.TotalLatency.String(),
		"memory_stats":  s.getMemoryStats(),
		"goroutines":    runtime.NumGoroutine(),
		"cache_size":    len(s.cache),
		"data_size":     len(s.data),
	}
	
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// Helper methods

func (s *Server) performCPUIntensiveTask(iterations int) float64 {
	result := 0.0
	for i := 0; i < iterations; i++ {
		// Simulate CPU-intensive computation
		result += float64(i) * 1.1
		if i%1000 == 0 {
			// Add some complexity
			for j := 0; j < 100; j++ {
				result += float64(j) / float64(i+1)
			}
		}
	}
	return result
}

func (s *Server) performMemoryIntensiveTask(size int) []interface{} {
	data := make([]interface{}, 0, size)
	for i := 0; i < size; i++ {
		// Create various types of objects
		switch i % 4 {
		case 0:
			data = append(data, make([]byte, 1024))
		case 1:
			data = append(data, make(map[string]interface{}))
		case 2:
			data = append(data, &User{
				ID:   i,
				Name: fmt.Sprintf("User_%d", i),
				Profile: UserProfile{
					Tags:     make([]string, 10),
					Metadata: make(map[string]string),
				},
				Settings: make(map[string]interface{}),
			})
		case 3:
			data = append(data, strings.Repeat("data", 256))
		}
	}
	return data
}

func (s *Server) simulateDatabaseOperation(queryType string, records int) interface{} {
	// Simulate database latency
	latency := time.Duration(records) * time.Millisecond
	if queryType == "insert" || queryType == "update" {
		latency *= 2
	}
	time.Sleep(latency)
	
	// Generate mock data
	switch queryType {
	case "select":
		result := make([]User, records)
		for i := 0; i < records; i++ {
			result[i] = User{
				ID:    i,
				Name:  fmt.Sprintf("User_%d", i),
				Email: fmt.Sprintf("user%d@example.com", i),
			}
		}
		return result
	case "insert", "update":
		return map[string]interface{}{
			"affected_rows": records,
			"operation":     queryType,
		}
	default:
		return map[string]interface{}{"status": "unknown_operation"}
	}
}

func (s *Server) getCachedValue(key string) (interface{}, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	
	entry, exists := s.cache[key]
	if !exists || time.Now().After(entry.Expiry) {
		return nil, false
	}
	
	// Update hit count (this creates a race condition for demonstration)
	entry.HitCount++
	s.cache[key] = entry
	
	return entry.Value, true
}

func (s *Server) setCachedValue(key string, value interface{}, ttl time.Duration) {
	s.mu.Lock()
	defer s.mu.Unlock()
	
	s.cache[key] = CacheEntry{
		Value:     value,
		Expiry:    time.Now().Add(ttl),
		HitCount:  0,
		CreatedAt: time.Now(),
	}
}

func (s *Server) generateExpensiveValue(key string) interface{} {
	// Simulate expensive computation
	time.Sleep(100 * time.Millisecond)
	
	return map[string]interface{}{
		"key":        key,
		"generated": time.Now(),
		"data":       strings.Repeat(key, 100),
		"random":     rand.Intn(1000),
	}
}

func (s *Server) getCacheStats() map[string]interface{} {
	s.mu.RLock()
	defer s.mu.RUnlock()
	
	totalHits := 0
	expiredCount := 0
	now := time.Now()
	
	for _, entry := range s.cache {
		totalHits += entry.HitCount
		if now.After(entry.Expiry) {
			expiredCount++
		}
	}
	
	return map[string]interface{}{
		"total_entries": len(s.cache),
		"total_hits":    totalHits,
		"expired_count": expiredCount,
	}
}

func (s *Server) performConcurrentWork(workers, tasks int) []interface{} {
	results := make([]interface{}, 0, tasks)
	resultsCh := make(chan interface{}, tasks)
	workCh := make(chan int, tasks)
	
	// Start workers
	var wg sync.WaitGroup
	for i := 0; i < workers; i++ {
		wg.Add(1)
		go func(workerID int) {
			defer wg.Done()
			for taskID := range workCh {
				// Simulate work
				time.Sleep(time.Duration(rand.Intn(10)) * time.Millisecond)
				result := map[string]interface{}{
					"worker": workerID,
					"task":   taskID,
					"result": taskID * workerID,
				}
				resultsCh <- result
			}
		}(i)
	}
	
	// Send tasks
	go func() {
		defer close(workCh)
		for i := 0; i < tasks; i++ {
			workCh <- i
		}
	}()
	
	// Collect results
	go func() {
		wg.Wait()
		close(resultsCh)
	}()
	
	for result := range resultsCh {
		results = append(results, result)
	}
	
	return results
}

func (s *Server) createLeakyStructures(size int) {
	s.mu.Lock()
	defer s.mu.Unlock()
	
	// Create data that might not be properly cleaned up
	for i := 0; i < size; i++ {
		key := fmt.Sprintf("leak_%d_%d", time.Now().UnixNano(), i)
		
		// Create large objects that accumulate over time
		largeData := make([]byte, 10*1024) // 10KB per object
		for j := range largeData {
			largeData[j] = byte(i + j)
		}
		
		s.data[key] = map[string]interface{}{
			"data":      largeData,
			"timestamp": time.Now(),
			"metadata":  make(map[string]string),
		}
	}
}

func (s *Server) getMemoryStats() map[string]interface{} {
	var m runtime.MemStats
	runtime.ReadMemStats(&m)
	
	return map[string]interface{}{
		"alloc_mb":      float64(m.Alloc) / 1024 / 1024,
		"total_alloc_mb": float64(m.TotalAlloc) / 1024 / 1024,
		"sys_mb":        float64(m.Sys) / 1024 / 1024,
		"num_gc":        m.NumGC,
		"gc_pause_ns":   m.PauseNs[(m.NumGC+255)%256],
		"heap_objects":  m.HeapObjects,
		"goroutines":    runtime.NumGoroutine(),
	}
}

func (s *Server) recordRequest(start time.Time) {
	duration := time.Since(start)
	
	s.stats.mu.Lock()
	s.stats.Requests++
	s.stats.TotalLatency += duration
	s.stats.mu.Unlock()
}

// setupRoutes configures HTTP routes
func (s *Server) setupRoutes() {
	http.HandleFunc("/cpu", s.cpuIntensiveHandler)
	http.HandleFunc("/memory", s.memoryIntensiveHandler)
	http.HandleFunc("/inefficient", s.inefficientHandler)
	http.HandleFunc("/database", s.databaseSimulationHandler)
	http.HandleFunc("/cache", s.cacheHandler)
	http.HandleFunc("/concurrent", s.concurrentHandler)
	http.HandleFunc("/leaky", s.leakyHandler)
	http.HandleFunc("/stats", s.statsHandler)
	
	// Health check endpoint
	http.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]string{
			"status": "healthy",
			"time":   time.Now().Format(time.RFC3339),
		})
	})
	
	// Root endpoint with usage information
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		endpoints := map[string]string{
			"/cpu":        "CPU-intensive operations (params: iterations)",
			"/memory":     "Memory-intensive operations (params: size)",
			"/inefficient": "Inefficient operations demo",
			"/database":   "Database simulation (params: type, records)",
			"/cache":      "Cache operations (params: key, ttl)",
			"/concurrent": "Concurrent operations (params: workers, tasks)",
			"/leaky":      "Memory leak simulation (params: size)",
			"/stats":      "Server statistics",
			"/health":     "Health check",
			"/debug/pprof/": "pprof profiling endpoints",
		}
		json.NewEncoder(w).Encode(map[string]interface{}{
			"message":   "Live Application Profiling Demo Server",
			"endpoints": endpoints,
		})
	})
}

// main function starts the web server
func main() {
	server := NewServer()
	server.setupRoutes()
	
	port := ":8080"
	log.Printf("Starting server on port %s", port)
	log.Printf("pprof endpoints available at http://localhost%s/debug/pprof/", port)
	log.Printf("Server endpoints: http://localhost%s/", port)
	
	if err := http.ListenAndServe(port, nil); err != nil {
		log.Fatal("Server failed to start:", err)
	}
}