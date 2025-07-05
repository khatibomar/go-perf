package main

import (
	"context"
	"fmt"
	"math/rand"
	"net"
	"os"
	"sync"
	"time"
)

// LeakSimulator provides a framework for simulating different types of memory leaks
type LeakSimulator struct {
	runningLeaks map[string]*RunningLeak
	mu           sync.RWMutex
	stopChan     chan struct{}
	isRunning    bool
}

// RunningLeak represents an active leak simulation
type RunningLeak struct {
	ID          string
	Type        LeakType
	Description string
	StartedAt   time.Time
	StopFunc    func()
	Metrics     map[string]interface{}
}

// LeakSimulationConfig contains configuration for leak simulation
type LeakSimulationConfig struct {
	Duration     time.Duration
	Intensity    int // 1-10 scale
	RandomSeed   int64
	AutoCleanup  bool
}

// NewLeakSimulator creates a new leak simulator
func NewLeakSimulator() *LeakSimulator {
	return &LeakSimulator{
		runningLeaks: make(map[string]*RunningLeak),
		stopChan:     make(chan struct{}),
	}
}

// StartGoroutineLeak simulates a goroutine leak
func (ls *LeakSimulator) StartGoroutineLeak(config *LeakSimulationConfig) (string, error) {
	ls.mu.Lock()
	defer ls.mu.Unlock()
	
	leakID := fmt.Sprintf("goroutine_leak_%d", time.Now().Unix())
	ctx, cancel := context.WithCancel(context.Background())
	
	// Start leaking goroutines
	go ls.simulateGoroutineLeak(ctx, config)
	
	runningLeak := &RunningLeak{
		ID:          leakID,
		Type:        LeakTypeGoroutine,
		Description: fmt.Sprintf("Goroutine leak simulation (intensity: %d)", config.Intensity),
		StartedAt:   time.Now(),
		StopFunc:    cancel,
		Metrics: map[string]interface{}{
			"intensity": config.Intensity,
			"duration":  config.Duration,
		},
	}
	
	ls.runningLeaks[leakID] = runningLeak
	
	// Auto-cleanup if configured
	if config.AutoCleanup && config.Duration > 0 {
		go func() {
			time.Sleep(config.Duration)
			ls.StopLeak(leakID)
		}()
	}
	
	return leakID, nil
}

// simulateGoroutineLeak creates goroutines that will leak
func (ls *LeakSimulator) simulateGoroutineLeak(ctx context.Context, config *LeakSimulationConfig) {
	ticker := time.NewTicker(time.Duration(1000/config.Intensity) * time.Millisecond)
	defer ticker.Stop()
	
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			// Create a goroutine that will block indefinitely
			go func() {
				// Simulate different types of goroutine leaks
				switch rand.Intn(4) {
				case 0:
					// Blocked on channel receive
					ch := make(chan struct{})
					<-ch
				case 1:
					// Infinite loop
					for {
						time.Sleep(100 * time.Millisecond)
					}
				case 2:
					// Blocked on mutex
					var mu sync.Mutex
					mu.Lock()
					mu.Lock() // This will block forever
				case 3:
					// Blocked on select with no default
					select {
					case <-time.After(24 * time.Hour):
						// Will block for a very long time
					}
				}
			}()
		}
	}
}

// StartMemoryLeak simulates a memory leak
func (ls *LeakSimulator) StartMemoryLeak(config *LeakSimulationConfig) (string, error) {
	ls.mu.Lock()
	defer ls.mu.Unlock()
	
	leakID := fmt.Sprintf("memory_leak_%d", time.Now().Unix())
	ctx, cancel := context.WithCancel(context.Background())
	
	// Start memory leak
	go ls.simulateMemoryLeak(ctx, config)
	
	runningLeak := &RunningLeak{
		ID:          leakID,
		Type:        LeakTypeMemory,
		Description: fmt.Sprintf("Memory leak simulation (intensity: %d)", config.Intensity),
		StartedAt:   time.Now(),
		StopFunc:    cancel,
		Metrics: map[string]interface{}{
			"intensity": config.Intensity,
			"duration":  config.Duration,
		},
	}
	
	ls.runningLeaks[leakID] = runningLeak
	
	// Auto-cleanup if configured
	if config.AutoCleanup && config.Duration > 0 {
		go func() {
			time.Sleep(config.Duration)
			ls.StopLeak(leakID)
		}()
	}
	
	return leakID, nil
}

// Global variables to simulate memory leaks
var (
	leakedSlices [][]byte
	leakedMaps   []map[string]interface{}
	leakedData   []interface{}
	leakMutex    sync.Mutex
)

// simulateMemoryLeak creates memory that won't be garbage collected
func (ls *LeakSimulator) simulateMemoryLeak(ctx context.Context, config *LeakSimulationConfig) {
	ticker := time.NewTicker(time.Duration(1000/config.Intensity) * time.Millisecond)
	defer ticker.Stop()
	
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			leakMutex.Lock()
			
			// Simulate different types of memory leaks
			switch rand.Intn(4) {
			case 0:
				// Growing slice that's never cleaned
				data := make([]byte, 1024*config.Intensity)
				for i := range data {
					data[i] = byte(rand.Intn(256))
				}
				leakedSlices = append(leakedSlices, data)
				
			case 1:
				// Growing map that's never cleaned
				m := make(map[string]interface{})
				for i := 0; i < 100*config.Intensity; i++ {
					m[fmt.Sprintf("key_%d_%d", time.Now().UnixNano(), i)] = make([]byte, 64)
				}
				leakedMaps = append(leakedMaps, m)
				
			case 2:
				// Circular references
				type Node struct {
					Data []byte
					Next *Node
				}
				
				node1 := &Node{Data: make([]byte, 512*config.Intensity)}
				node2 := &Node{Data: make([]byte, 512*config.Intensity)}
				node1.Next = node2
				node2.Next = node1 // Circular reference
				
				leakedData = append(leakedData, node1)
				
			case 3:
				// Large string accumulation
				var builder string
				for i := 0; i < 1000*config.Intensity; i++ {
					builder += fmt.Sprintf("data_%d_", i)
				}
				leakedData = append(leakedData, builder)
			}
			
			leakMutex.Unlock()
		}
	}
}

// StartFileDescriptorLeak simulates a file descriptor leak
func (ls *LeakSimulator) StartFileDescriptorLeak(config *LeakSimulationConfig) (string, error) {
	ls.mu.Lock()
	defer ls.mu.Unlock()
	
	leakID := fmt.Sprintf("fd_leak_%d", time.Now().Unix())
	ctx, cancel := context.WithCancel(context.Background())
	
	// Start file descriptor leak
	go ls.simulateFileDescriptorLeak(ctx, config)
	
	runningLeak := &RunningLeak{
		ID:          leakID,
		Type:        LeakTypeFileDescriptor,
		Description: fmt.Sprintf("File descriptor leak simulation (intensity: %d)", config.Intensity),
		StartedAt:   time.Now(),
		StopFunc:    cancel,
		Metrics: map[string]interface{}{
			"intensity": config.Intensity,
			"duration":  config.Duration,
		},
	}
	
	ls.runningLeaks[leakID] = runningLeak
	
	// Auto-cleanup if configured
	if config.AutoCleanup && config.Duration > 0 {
		go func() {
			time.Sleep(config.Duration)
			ls.StopLeak(leakID)
		}()
	}
	
	return leakID, nil
}

// Global slice to hold leaked file handles
var leakedFiles []*os.File

// simulateFileDescriptorLeak opens files without closing them
func (ls *LeakSimulator) simulateFileDescriptorLeak(ctx context.Context, config *LeakSimulationConfig) {
	ticker := time.NewTicker(time.Duration(2000/config.Intensity) * time.Millisecond)
	defer ticker.Stop()
	
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			// Create temporary files and don't close them
			for i := 0; i < config.Intensity; i++ {
				file, err := os.CreateTemp("", "leak_test_*.tmp")
				if err != nil {
					continue
				}
				// Intentionally not closing the file to simulate leak
				leakedFiles = append(leakedFiles, file)
			}
		}
	}
}

// StartNetworkConnectionLeak simulates a network connection leak
func (ls *LeakSimulator) StartNetworkConnectionLeak(config *LeakSimulationConfig) (string, error) {
	ls.mu.Lock()
	defer ls.mu.Unlock()
	
	leakID := fmt.Sprintf("network_leak_%d", time.Now().Unix())
	ctx, cancel := context.WithCancel(context.Background())
	
	// Start network connection leak
	go ls.simulateNetworkConnectionLeak(ctx, config)
	
	runningLeak := &RunningLeak{
		ID:          leakID,
		Type:        LeakTypeNetworkConnection,
		Description: fmt.Sprintf("Network connection leak simulation (intensity: %d)", config.Intensity),
		StartedAt:   time.Now(),
		StopFunc:    cancel,
		Metrics: map[string]interface{}{
			"intensity": config.Intensity,
			"duration":  config.Duration,
		},
	}
	
	ls.runningLeaks[leakID] = runningLeak
	
	// Auto-cleanup if configured
	if config.AutoCleanup && config.Duration > 0 {
		go func() {
			time.Sleep(config.Duration)
			ls.StopLeak(leakID)
		}()
	}
	
	return leakID, nil
}

// Global slice to hold leaked connections
var leakedConnections []net.Conn

// simulateNetworkConnectionLeak creates network connections without closing them
func (ls *LeakSimulator) simulateNetworkConnectionLeak(ctx context.Context, config *LeakSimulationConfig) {
	ticker := time.NewTicker(time.Duration(3000/config.Intensity) * time.Millisecond)
	defer ticker.Stop()
	
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			// Create network connections and don't close them
			for i := 0; i < config.Intensity; i++ {
				// Try to connect to a common service (this may fail, which is fine for simulation)
				conn, err := net.DialTimeout("tcp", "8.8.8.8:53", 5*time.Second)
				if err != nil {
					// If external connection fails, create a local connection
					listener, err := net.Listen("tcp", "127.0.0.1:0")
					if err != nil {
						continue
					}
					go func() {
						conn, _ := listener.Accept()
						if conn != nil {
							leakedConnections = append(leakedConnections, conn)
						}
					}()
					
					conn, err = net.Dial("tcp", listener.Addr().String())
					if err != nil {
						listener.Close()
						continue
					}
				}
				// Intentionally not closing the connection to simulate leak
				leakedConnections = append(leakedConnections, conn)
			}
		}
	}
}

// StopLeak stops a specific leak simulation
func (ls *LeakSimulator) StopLeak(leakID string) error {
	ls.mu.Lock()
	defer ls.mu.Unlock()
	
	leak, exists := ls.runningLeaks[leakID]
	if !exists {
		return fmt.Errorf("leak with ID %s not found", leakID)
	}
	
	leak.StopFunc()
	delete(ls.runningLeaks, leakID)
	
	fmt.Printf("Stopped leak simulation: %s\n", leakID)
	return nil
}

// StopAllLeaks stops all running leak simulations
func (ls *LeakSimulator) StopAllLeaks() {
	ls.mu.Lock()
	defer ls.mu.Unlock()
	
	for leakID, leak := range ls.runningLeaks {
		leak.StopFunc()
		delete(ls.runningLeaks, leakID)
	}
	
	fmt.Println("Stopped all leak simulations")
}

// GetRunningLeaks returns information about currently running leaks
func (ls *LeakSimulator) GetRunningLeaks() map[string]*RunningLeak {
	ls.mu.RLock()
	defer ls.mu.RUnlock()
	
	leaks := make(map[string]*RunningLeak)
	for id, leak := range ls.runningLeaks {
		leaks[id] = leak
	}
	
	return leaks
}

// CleanupLeakedResources attempts to clean up leaked resources
func (ls *LeakSimulator) CleanupLeakedResources() {
	leakMutex.Lock()
	defer leakMutex.Unlock()
	
	// Clean up leaked slices
	leakedSlices = nil
	
	// Clean up leaked maps
	leakedMaps = nil
	
	// Clean up leaked data
	leakedData = nil
	
	// Close leaked files
	for _, file := range leakedFiles {
		if file != nil {
			file.Close()
			os.Remove(file.Name()) // Remove temp files
		}
	}
	leakedFiles = nil
	
	// Close leaked connections
	for _, conn := range leakedConnections {
		if conn != nil {
			conn.Close()
		}
	}
	leakedConnections = nil
	
	fmt.Println("Cleaned up leaked resources")
}

// GetLeakStats returns statistics about leaked resources
func (ls *LeakSimulator) GetLeakStats() map[string]interface{} {
	leakMutex.Lock()
	defer leakMutex.Unlock()
	
	stats := map[string]interface{}{
		"leaked_slices":     len(leakedSlices),
		"leaked_maps":       len(leakedMaps),
		"leaked_data":       len(leakedData),
		"leaked_files":      len(leakedFiles),
		"leaked_connections": len(leakedConnections),
		"running_leaks":     len(ls.runningLeaks),
	}
	
	return stats
}

// DefaultLeakSimulationConfig returns default configuration for leak simulation
func DefaultLeakSimulationConfig() *LeakSimulationConfig {
	return &LeakSimulationConfig{
		Duration:    5 * time.Minute,
		Intensity:   3, // Medium intensity
		RandomSeed:  time.Now().UnixNano(),
		AutoCleanup: true,
	}
}

// SimulateRandomLeak starts a random type of leak simulation
func (ls *LeakSimulator) SimulateRandomLeak(config *LeakSimulationConfig) (string, error) {
	if config.RandomSeed != 0 {
		rand.Seed(config.RandomSeed)
	}
	
	leakTypes := []func(*LeakSimulationConfig) (string, error){
		ls.StartGoroutineLeak,
		ls.StartMemoryLeak,
		ls.StartFileDescriptorLeak,
		ls.StartNetworkConnectionLeak,
	}
	
	selectedLeak := leakTypes[rand.Intn(len(leakTypes))]
	return selectedLeak(config)
}

// LeakSimulationExample demonstrates how to use the leak simulator
func LeakSimulationExample() {
	// This is an example function, not meant to be called from main
	simulator := NewLeakSimulator()
	config := DefaultLeakSimulationConfig()
	config.Duration = 30 * time.Second
	config.Intensity = 5
	
	// Start different types of leaks
	goroutineLeakID, _ := simulator.StartGoroutineLeak(config)
	memoryLeakID, _ := simulator.StartMemoryLeak(config)
	
	fmt.Printf("Started leaks: %s, %s\n", goroutineLeakID, memoryLeakID)
	
	// Let them run for a while
	time.Sleep(10 * time.Second)
	
	// Check stats
	stats := simulator.GetLeakStats()
	fmt.Printf("Leak stats: %+v\n", stats)
	
	// Stop specific leak
	simulator.StopLeak(goroutineLeakID)
	
	// Stop all leaks and cleanup
	simulator.StopAllLeaks()
	simulator.CleanupLeakedResources()
}