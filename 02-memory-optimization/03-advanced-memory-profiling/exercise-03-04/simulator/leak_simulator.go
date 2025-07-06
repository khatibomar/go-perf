package simulator

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"math/rand"
	"net"
	"net/http"
	"os"
	"runtime"
	"sync"
	"time"

	"golang.org/x/sync/errgroup"
)

// LeakType represents different types of leaks to simulate
type LeakType string

const (
	LeakTypeHeap      LeakType = "heap"
	LeakTypeGoroutine LeakType = "goroutine"
	LeakTypeFile      LeakType = "file"
	LeakTypeNetwork   LeakType = "network"
	LeakTypeReference LeakType = "reference"
	LeakTypeMixed     LeakType = "mixed"
)

// LeakConfig holds configuration for leak simulation
type LeakConfig struct {
	Type           LeakType      `json:"type"`
	Duration       time.Duration `json:"duration"`
	Intensity      int           `json:"intensity"`      // 1-10 scale
	GrowthRate     float64       `json:"growth_rate"`    // Resources per second
	BurstInterval  time.Duration `json:"burst_interval"`  // For burst patterns
	BurstSize      int           `json:"burst_size"`      // Resources per burst
	Randomness     float64       `json:"randomness"`      // 0-1 scale for random variation
	CleanupChance  float64       `json:"cleanup_chance"`  // 0-1 probability of cleanup
	Verbose        bool          `json:"verbose"`
	OutputDir      string        `json:"output_dir"`
}

// LeakStats represents statistics about a leak simulation
type LeakStats struct {
	StartTime         time.Time `json:"start_time"`
	EndTime           time.Time `json:"end_time"`
	Duration          time.Duration `json:"duration"`
	ResourcesCreated  int       `json:"resources_created"`
	ResourcesCleaned  int       `json:"resources_cleaned"`
	ResourcesLeaked   int       `json:"resources_leaked"`
	PeakResources     int       `json:"peak_resources"`
	AverageGrowthRate float64   `json:"average_growth_rate"`
	MemoryAllocated   int64     `json:"memory_allocated"`
	GoroutinesCreated int       `json:"goroutines_created"`
	FilesOpened       int       `json:"files_opened"`
	ConnectionsOpened int       `json:"connections_opened"`
}

// LeakSimulator simulates various types of memory and resource leaks
type LeakSimulator struct {
	config    *LeakConfig
	stats     *LeakStats
	running   bool
	stopChan  chan struct{}
	mutex     sync.RWMutex
	resources []interface{}
	rand      *rand.Rand
}

// NewLeakSimulator creates a new leak simulator
func NewLeakSimulator(config *LeakConfig) *LeakSimulator {
	return &LeakSimulator{
		config:    config,
		stats:     &LeakStats{},
		stopChan:  make(chan struct{}),
		resources: make([]interface{}, 0),
		rand:      rand.New(rand.NewSource(time.Now().UnixNano())),
	}
}

// Start begins the leak simulation
func (ls *LeakSimulator) Start() error {
	ls.mutex.Lock()
	defer ls.mutex.Unlock()

	if ls.running {
		return fmt.Errorf("leak simulator already running")
	}

	ls.running = true
	ls.stats.StartTime = time.Now()

	if ls.config.Verbose {
		fmt.Printf("Starting %s leak simulation (intensity: %d, duration: %v)\n",
			ls.config.Type, ls.config.Intensity, ls.config.Duration)
	}

	go ls.simulationLoop()

	return nil
}

// Stop stops the leak simulation
func (ls *LeakSimulator) Stop() {
	ls.mutex.Lock()
	defer ls.mutex.Unlock()

	if !ls.running {
		return
	}

	ls.running = false
	ls.stats.EndTime = time.Now()
	ls.stats.Duration = ls.stats.EndTime.Sub(ls.stats.StartTime)
	close(ls.stopChan)

	if ls.config.Verbose {
		fmt.Printf("Stopped %s leak simulation. Resources leaked: %d\n",
			ls.config.Type, ls.stats.ResourcesLeaked)
	}
}

// GetStats returns current simulation statistics
func (ls *LeakSimulator) GetStats() *LeakStats {
	ls.mutex.RLock()
	defer ls.mutex.RUnlock()

	// Update current stats
	ls.stats.ResourcesLeaked = ls.stats.ResourcesCreated - ls.stats.ResourcesCleaned
	ls.stats.PeakResources = len(ls.resources)

	if ls.stats.Duration > 0 {
		ls.stats.AverageGrowthRate = float64(ls.stats.ResourcesCreated) / ls.stats.Duration.Seconds()
	}

	// Return a copy
	stats := *ls.stats
	return &stats
}

// SaveStats saves simulation statistics to file
func (ls *LeakSimulator) SaveStats(filename string) error {
	stats := ls.GetStats()

	file, err := os.Create(filename)
	if err != nil {
		return err
	}
	defer file.Close()

	encoder := json.NewEncoder(file)
	encoder.SetIndent("", "  ")
	return encoder.Encode(stats)
}

// simulationLoop runs the main simulation loop
func (ls *LeakSimulator) simulationLoop() {
	ctx, cancel := context.WithTimeout(context.Background(), ls.config.Duration)
	defer cancel()

	ticker := time.NewTicker(time.Duration(1000/ls.config.GrowthRate) * time.Millisecond)
	defer ticker.Stop()

	burstTicker := time.NewTicker(ls.config.BurstInterval)
	defer burstTicker.Stop()

	for {
		select {
		case <-ctx.Done():
			ls.Stop()
			return
		case <-ls.stopChan:
			return
		case <-ticker.C:
			ls.createLeak()
		case <-burstTicker.C:
			ls.createBurst()
		}

		// Randomly cleanup some resources
		if ls.rand.Float64() < ls.config.CleanupChance {
			ls.cleanupSomeResources()
		}
	}
}

// createLeak creates a leak based on the configured type
func (ls *LeakSimulator) createLeak() {
	switch ls.config.Type {
	case LeakTypeHeap:
		ls.createHeapLeak()
	case LeakTypeGoroutine:
		ls.createGoroutineLeak()
	case LeakTypeFile:
		ls.createFileLeak()
	case LeakTypeNetwork:
		ls.createNetworkLeak()
	case LeakTypeReference:
		ls.createReferenceLeak()
	case LeakTypeMixed:
		ls.createMixedLeak()
	}
}

// createBurst creates a burst of leaks
func (ls *LeakSimulator) createBurst() {
	for i := 0; i < ls.config.BurstSize; i++ {
		ls.createLeak()
		if ls.config.Randomness > 0 {
			// Add some randomness to burst timing
			delay := time.Duration(ls.rand.Float64() * ls.config.Randomness * float64(time.Millisecond))
			time.Sleep(delay)
		}
	}
}

// createHeapLeak creates a heap memory leak
func (ls *LeakSimulator) createHeapLeak() {
	size := ls.calculateLeakSize()
	data := make([]byte, size)

	// Fill with random data to prevent optimization
	for i := range data {
		data[i] = byte(ls.rand.Intn(256))
	}

	ls.mutex.Lock()
	ls.resources = append(ls.resources, data)
	ls.stats.ResourcesCreated++
	ls.stats.MemoryAllocated += int64(size)
	ls.mutex.Unlock()

	if ls.config.Verbose && ls.stats.ResourcesCreated%100 == 0 {
		fmt.Printf("Created %d heap allocations (total: %d MB)\n",
			ls.stats.ResourcesCreated, ls.stats.MemoryAllocated/(1024*1024))
	}
}

// createGoroutineLeak creates a goroutine leak
func (ls *LeakSimulator) createGoroutineLeak() {
	// Create different types of goroutine leaks
	leakType := ls.rand.Intn(4)

	switch leakType {
	case 0:
		// Infinite loop goroutine
		go func() {
			for {
				time.Sleep(time.Millisecond * 100)
				runtime.Gosched()
			}
		}()
	case 1:
		// Channel blocking goroutine
		ch := make(chan struct{})
		go func() {
			<-ch // Will block forever
		}()
		ls.resources = append(ls.resources, ch)
	case 2:
		// Timer goroutine that never stops
		timer := time.NewTimer(time.Hour * 24) // Very long timer
		go func() {
			<-timer.C
		}()
		ls.resources = append(ls.resources, timer)
	case 3:
		// HTTP client with long timeout
		client := &http.Client{
			Timeout: time.Hour,
		}
		go func() {
			_, _ = client.Get("http://httpbin.org/delay/3600")
		}()
		ls.resources = append(ls.resources, client)
	}

	ls.mutex.Lock()
	ls.stats.ResourcesCreated++
	ls.stats.GoroutinesCreated++
	ls.mutex.Unlock()

	if ls.config.Verbose && ls.stats.GoroutinesCreated%50 == 0 {
		fmt.Printf("Created %d goroutines\n", ls.stats.GoroutinesCreated)
	}
}

// createFileLeak creates a file handle leak
func (ls *LeakSimulator) createFileLeak() {
	// Create different types of file leaks
	leakType := ls.rand.Intn(3)

	switch leakType {
	case 0:
		// Temporary file that's never closed
		tempFile, err := ioutil.TempFile("", "leak_*.tmp")
		if err != nil {
			return
		}
		// Write some data
		tempFile.WriteString(fmt.Sprintf("Leak data %d\n", ls.stats.FilesOpened))
		ls.resources = append(ls.resources, tempFile)
	case 1:
		// Open existing file for reading
		file, err := os.Open(os.Args[0]) // Open the executable
		if err != nil {
			return
		}
		ls.resources = append(ls.resources, file)
	case 2:
		// Create a pipe that's never closed
		r, w, err := os.Pipe()
		if err != nil {
			return
		}
		ls.resources = append(ls.resources, r)
		ls.resources = append(ls.resources, w)
	}

	ls.mutex.Lock()
	ls.stats.ResourcesCreated++
	ls.stats.FilesOpened++
	ls.mutex.Unlock()

	if ls.config.Verbose && ls.stats.FilesOpened%20 == 0 {
		fmt.Printf("Opened %d files\n", ls.stats.FilesOpened)
	}
}

// createNetworkLeak creates a network connection leak
func (ls *LeakSimulator) createNetworkLeak() {
	// Create different types of network leaks
	leakType := ls.rand.Intn(3)

	switch leakType {
	case 0:
		// TCP connection that's never closed
		conn, err := net.DialTimeout("tcp", "httpbin.org:80", time.Second*5)
		if err != nil {
			return
		}
		ls.resources = append(ls.resources, conn)
	case 1:
		// UDP connection that's never closed
		conn, err := net.Dial("udp", "8.8.8.8:53")
		if err != nil {
			return
		}
		ls.resources = append(ls.resources, conn)
	case 2:
		// HTTP client with keep-alive that accumulates connections
		client := &http.Client{
			Timeout: time.Second * 30,
		}
		go func() {
			resp, err := client.Get("http://httpbin.org/get")
			if err == nil {
				// Read but don't close response body
				io.Copy(ioutil.Discard, resp.Body)
				// resp.Body.Close() // Intentionally not called
			}
		}()
		ls.resources = append(ls.resources, client)
	}

	ls.mutex.Lock()
	ls.stats.ResourcesCreated++
	ls.stats.ConnectionsOpened++
	ls.mutex.Unlock()

	if ls.config.Verbose && ls.stats.ConnectionsOpened%10 == 0 {
		fmt.Printf("Opened %d network connections\n", ls.stats.ConnectionsOpened)
	}
}

// createReferenceLeak creates a reference cycle leak
func (ls *LeakSimulator) createReferenceLeak() {
	// Create circular references that prevent garbage collection
	type Node struct {
		ID       int
		Data     []byte
		Parent   *Node
		Children []*Node
		Sibling  *Node
	}

	// Create a cycle of nodes
	cycleSize := ls.rand.Intn(5) + 3 // 3-7 nodes
	nodes := make([]*Node, cycleSize)

	for i := 0; i < cycleSize; i++ {
		nodes[i] = &Node{
			ID:       i,
			Data:     make([]byte, ls.calculateLeakSize()/cycleSize),
			Children: make([]*Node, 0),
		}
	}

	// Create circular references
	for i := 0; i < cycleSize; i++ {
		next := (i + 1) % cycleSize
		prev := (i - 1 + cycleSize) % cycleSize

		nodes[i].Parent = nodes[prev]
		nodes[i].Children = append(nodes[i].Children, nodes[next])
		nodes[i].Sibling = nodes[next]
	}

	ls.mutex.Lock()
	ls.resources = append(ls.resources, nodes)
	ls.stats.ResourcesCreated++
	ls.mutex.Unlock()

	if ls.config.Verbose && ls.stats.ResourcesCreated%10 == 0 {
		fmt.Printf("Created %d reference cycles\n", ls.stats.ResourcesCreated)
	}
}

// createMixedLeak creates a random type of leak
func (ls *LeakSimulator) createMixedLeak() {
	leakTypes := []LeakType{
		LeakTypeHeap,
		LeakTypeGoroutine,
		LeakTypeFile,
		LeakTypeNetwork,
		LeakTypeReference,
	}

	// Temporarily change type and create leak
	originalType := ls.config.Type
	ls.config.Type = leakTypes[ls.rand.Intn(len(leakTypes))]
	ls.createLeak()
	ls.config.Type = originalType
}

// calculateLeakSize calculates the size of a leak based on intensity
func (ls *LeakSimulator) calculateLeakSize() int {
	baseSize := 1024 * ls.config.Intensity // Base size in bytes

	// Add randomness
	if ls.config.Randomness > 0 {
		randomFactor := 1.0 + (ls.rand.Float64()-0.5)*ls.config.Randomness
		baseSize = int(float64(baseSize) * randomFactor)
	}

	return baseSize
}

// cleanupSomeResources randomly cleans up some resources
func (ls *LeakSimulator) cleanupSomeResources() {
	ls.mutex.Lock()
	defer ls.mutex.Unlock()

	if len(ls.resources) == 0 {
		return
	}

	// Clean up 1-3 resources randomly
	cleanupCount := ls.rand.Intn(3) + 1
	for i := 0; i < cleanupCount && len(ls.resources) > 0; i++ {
		index := ls.rand.Intn(len(ls.resources))
		resource := ls.resources[index]

		// Try to clean up the resource
		ls.cleanupResource(resource)

		// Remove from slice
		ls.resources = append(ls.resources[:index], ls.resources[index+1:]...)
		ls.stats.ResourcesCleaned++
	}
}

// cleanupResource attempts to clean up a specific resource
func (ls *LeakSimulator) cleanupResource(resource interface{}) {
	switch r := resource.(type) {
	case *os.File:
		r.Close()
	case net.Conn:
		r.Close()
	case *http.Client:
		// Can't really clean up HTTP client
	case *time.Timer:
		r.Stop()
	case chan struct{}:
		close(r)
	default:
		// For other types (like slices), just let GC handle them
	}
}

// CreateComplexLeak creates a complex leak scenario
func (ls *LeakSimulator) CreateComplexLeak() {
	// Create a complex scenario with multiple leak types
	eg := errgroup.Group{}

	// Heap leak component
	eg.Go(func() error {
		for i := 0; i < 100; i++ {
			data := make([]byte, 1024*1024) // 1MB
			ls.resources = append(ls.resources, data)
			time.Sleep(time.Millisecond * 10)
		}
		return nil
	})

	// Goroutine leak component
	eg.Go(func() error {
		for i := 0; i < 50; i++ {
			go func(id int) {
				ch := make(chan struct{})
				select {
				case <-ch:
					// Will never receive
				case <-time.After(time.Hour):
					// Very long timeout
				}
			}(i)
			time.Sleep(time.Millisecond * 20)
		}
		return nil
	})

	// File leak component
	eg.Go(func() error {
		for i := 0; i < 20; i++ {
			tempFile, err := ioutil.TempFile("", "complex_leak_*.tmp")
			if err == nil {
				tempFile.WriteString(fmt.Sprintf("Complex leak data %d\n", i))
				ls.resources = append(ls.resources, tempFile)
			}
			time.Sleep(time.Millisecond * 50)
		}
		return nil
	})

	eg.Wait()

	ls.mutex.Lock()
	ls.stats.ResourcesCreated += 170 // 100 + 50 + 20
	ls.mutex.Unlock()

	if ls.config.Verbose {
		fmt.Println("Created complex leak scenario")
	}
}

// SimulateMemoryPressure simulates memory pressure scenarios
func (ls *LeakSimulator) SimulateMemoryPressure() {
	// Create large allocations that stress the garbage collector
	for i := 0; i < 10; i++ {
		go func(id int) {
			for j := 0; j < 1000; j++ {
				// Allocate and immediately release
				data := make([]byte, 1024*1024) // 1MB
				_ = data
				runtime.GC()
				time.Sleep(time.Millisecond)
			}
		}(i)
	}

	if ls.config.Verbose {
		fmt.Println("Started memory pressure simulation")
	}
}

// CreateLeakPattern creates a specific leak pattern
func (ls *LeakSimulator) CreateLeakPattern(pattern string) {
	switch pattern {
	case "gradual":
		// Gradual increase in leaks
		for i := 0; i < 100; i++ {
			ls.createHeapLeak()
			time.Sleep(time.Duration(i) * time.Millisecond)
		}
	case "burst":
		// Sudden burst of leaks
		for i := 0; i < 1000; i++ {
			ls.createHeapLeak()
		}
	case "periodic":
		// Periodic leak pattern
		for i := 0; i < 10; i++ {
			for j := 0; j < 50; j++ {
				ls.createHeapLeak()
			}
			time.Sleep(time.Second)
		}
	case "random":
		// Random leak pattern
		for i := 0; i < 500; i++ {
			ls.createMixedLeak()
			time.Sleep(time.Duration(ls.rand.Intn(100)) * time.Millisecond)
		}
	}

	if ls.config.Verbose {
		fmt.Printf("Created %s leak pattern\n", pattern)
	}
}

// GenerateReport generates a detailed report of the simulation
func (ls *LeakSimulator) GenerateReport() map[string]interface{} {
	stats := ls.GetStats()

	report := map[string]interface{}{
		"simulation_config": ls.config,
		"statistics":        stats,
		"runtime_info": map[string]interface{}{
			"goroutines":    runtime.NumGoroutine(),
			"memory_stats":  getMemoryStats(),
			"gc_stats":      getGCStats(),
		},
		"resource_summary": map[string]interface{}{
			"total_resources":   len(ls.resources),
			"leak_efficiency":   float64(stats.ResourcesLeaked) / float64(stats.ResourcesCreated),
			"cleanup_rate":      float64(stats.ResourcesCleaned) / float64(stats.ResourcesCreated),
			"memory_per_leak":   float64(stats.MemoryAllocated) / float64(stats.ResourcesCreated),
		},
		"recommendations": ls.generateRecommendations(stats),
	}

	return report
}

// getMemoryStats gets current memory statistics
func getMemoryStats() map[string]interface{} {
	var m runtime.MemStats
	runtime.ReadMemStats(&m)

	return map[string]interface{}{
		"alloc":         m.Alloc,
		"total_alloc":   m.TotalAlloc,
		"sys":           m.Sys,
		"heap_alloc":    m.HeapAlloc,
		"heap_sys":      m.HeapSys,
		"heap_objects":  m.HeapObjects,
		"stack_inuse":   m.StackInuse,
		"stack_sys":     m.StackSys,
	}
}

// getGCStats gets garbage collection statistics
func getGCStats() map[string]interface{} {
	var m runtime.MemStats
	runtime.ReadMemStats(&m)

	return map[string]interface{}{
		"num_gc":        m.NumGC,
		"pause_total":   m.PauseTotalNs,
		"pause_ns":      m.PauseNs,
		"gc_cpu_fraction": m.GCCPUFraction,
	}
}

// generateRecommendations generates recommendations based on simulation results
func (ls *LeakSimulator) generateRecommendations(stats *LeakStats) []string {
	var recommendations []string

	if stats.ResourcesLeaked > 100 {
		recommendations = append(recommendations, "High number of leaked resources detected. Implement proper resource cleanup.")
	}

	if stats.MemoryAllocated > 100*1024*1024 { // 100MB
		recommendations = append(recommendations, "High memory allocation detected. Monitor for memory leaks.")
	}

	if stats.GoroutinesCreated > 1000 {
		recommendations = append(recommendations, "High number of goroutines created. Implement goroutine pooling.")
	}

	if stats.FilesOpened > 100 {
		recommendations = append(recommendations, "High number of files opened. Ensure proper file handle management.")
	}

	if stats.ConnectionsOpened > 50 {
		recommendations = append(recommendations, "High number of network connections. Implement connection pooling.")
	}

	leakRate := float64(stats.ResourcesLeaked) / float64(stats.ResourcesCreated)
	if leakRate > 0.8 {
		recommendations = append(recommendations, "High leak rate detected. Review resource management patterns.")
	}

	if len(recommendations) == 0 {
		recommendations = append(recommendations, "Simulation completed successfully. Monitor for any unusual patterns.")
	}

	return recommendations
}

// CleanupAll attempts to clean up all remaining resources
func (ls *LeakSimulator) CleanupAll() {
	ls.mutex.Lock()
	defer ls.mutex.Unlock()

	for _, resource := range ls.resources {
		ls.cleanupResource(resource)
		ls.stats.ResourcesCleaned++
	}

	ls.resources = ls.resources[:0] // Clear the slice

	if ls.config.Verbose {
		fmt.Printf("Cleaned up all remaining resources. Total cleaned: %d\n", ls.stats.ResourcesCleaned)
	}
}

// SaveReport saves the simulation report to a file
func (ls *LeakSimulator) SaveReport(filename string) error {
	report := ls.GenerateReport()

	file, err := os.Create(filename)
	if err != nil {
		return err
	}
	defer file.Close()

	encoder := json.NewEncoder(file)
	encoder.SetIndent("", "  ")
	return encoder.Encode(report)
}

// CreateLeakScenario creates a predefined leak scenario
func CreateLeakScenario(scenario string) *LeakConfig {
	switch scenario {
	case "mild":
		return &LeakConfig{
			Type:          LeakTypeHeap,
			Duration:      time.Minute * 5,
			Intensity:     3,
			GrowthRate:    1.0,
			BurstInterval: time.Second * 30,
			BurstSize:     5,
			Randomness:    0.2,
			CleanupChance: 0.3,
			Verbose:       true,
		}
	case "moderate":
		return &LeakConfig{
			Type:          LeakTypeMixed,
			Duration:      time.Minute * 10,
			Intensity:     5,
			GrowthRate:    2.0,
			BurstInterval: time.Second * 15,
			BurstSize:     10,
			Randomness:    0.3,
			CleanupChance: 0.2,
			Verbose:       true,
		}
	case "severe":
		return &LeakConfig{
			Type:          LeakTypeMixed,
			Duration:      time.Minute * 15,
			Intensity:     8,
			GrowthRate:    5.0,
			BurstInterval: time.Second * 5,
			BurstSize:     20,
			Randomness:    0.5,
			CleanupChance: 0.1,
			Verbose:       true,
		}
	default:
		return &LeakConfig{
			Type:          LeakTypeHeap,
			Duration:      time.Minute * 2,
			Intensity:     2,
			GrowthRate:    0.5,
			BurstInterval: time.Minute,
			BurstSize:     3,
			Randomness:    0.1,
			CleanupChance: 0.5,
			Verbose:       true,
		}
	}
}