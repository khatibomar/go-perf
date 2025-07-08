// workload/manager.go
package workload

import (
	"context"
	"fmt"
	"math/rand"
	"runtime"
	"sync"
	"time"
)

type Manager struct {
	mu              sync.RWMutex
	activeWorkload  *Workload
	cancel          context.CancelFunc
	leakyGoroutines []chan struct{}
	leakyData       [][]byte
}

type Workload struct {
	Type      string    `json:"type"`
	Intensity string    `json:"intensity"`
	Duration  time.Duration `json:"duration"`
	StartTime time.Time `json:"start_time"`
	Status    string    `json:"status"`
}

type WorkloadStatus struct {
	Active    bool      `json:"active"`
	Workload  *Workload `json:"workload,omitempty"`
	Uptime    time.Duration `json:"uptime"`
	MemoryMB  float64   `json:"memory_mb"`
	Goroutines int      `json:"goroutines"`
}

func NewManager() *Manager {
	return &Manager{
		leakyGoroutines: make([]chan struct{}, 0),
		leakyData:       make([][]byte, 0),
	}
}

func (m *Manager) Start(ctx context.Context) {
	// Manager is ready to handle workloads
	<-ctx.Done()
	m.StopWorkload()
}

func (m *Manager) StartWorkload(workloadType, intensity string, duration time.Duration) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.activeWorkload != nil {
		return fmt.Errorf("workload already running: %s", m.activeWorkload.Type)
	}

	workload := &Workload{
		Type:      workloadType,
		Intensity: intensity,
		Duration:  duration,
		StartTime: time.Now(),
		Status:    "running",
	}

	ctx, cancel := context.WithTimeout(context.Background(), duration)
	m.cancel = cancel
	m.activeWorkload = workload

	go m.runWorkload(ctx, workload)

	return nil
}

func (m *Manager) StopWorkload() {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.cancel != nil {
		m.cancel()
		m.cancel = nil
	}

	if m.activeWorkload != nil {
		m.activeWorkload.Status = "stopped"
		m.activeWorkload = nil
	}

	// Clean up leaky resources
	m.cleanupLeakyResources()
}

func (m *Manager) GetStatus() WorkloadStatus {
	m.mu.RLock()
	defer m.mu.RUnlock()

	var m_runtime runtime.MemStats
	runtime.ReadMemStats(&m_runtime)

	status := WorkloadStatus{
		Active:     m.activeWorkload != nil,
		MemoryMB:   float64(m_runtime.Alloc) / 1024 / 1024,
		Goroutines: runtime.NumGoroutine(),
	}

	if m.activeWorkload != nil {
		status.Workload = m.activeWorkload
		status.Uptime = time.Since(m.activeWorkload.StartTime)
	}

	return status
}

func (m *Manager) runWorkload(ctx context.Context, workload *Workload) {
	defer func() {
		m.mu.Lock()
		if m.activeWorkload != nil {
			m.activeWorkload.Status = "completed"
		}
		m.mu.Unlock()
	}()

	switch workload.Type {
	case "leak":
		m.runMemoryLeakWorkload(ctx, workload.Intensity)
	case "goroutine_leak":
		m.runGoroutineLeakWorkload(ctx, workload.Intensity)
	case "gc_pressure":
		m.runGCPressureWorkload(ctx, workload.Intensity)
	case "normal":
		m.runNormalWorkload(ctx, workload.Intensity)
	default:
		m.runNormalWorkload(ctx, workload.Intensity)
	}
}

func (m *Manager) runMemoryLeakWorkload(ctx context.Context, intensity string) {
	var allocSize int
	var interval time.Duration

	switch intensity {
	case "low":
		allocSize = 1024 * 100 // 100KB
		interval = time.Second
	case "medium":
		allocSize = 1024 * 500 // 500KB
		interval = time.Millisecond * 500
	case "high":
		allocSize = 1024 * 1024 // 1MB
		interval = time.Millisecond * 100
	default:
		allocSize = 1024 * 500
		interval = time.Millisecond * 500
	}

	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			// Allocate memory and keep reference to create leak
			data := make([]byte, allocSize)
			for i := range data {
				data[i] = byte(rand.Intn(256))
			}
			m.mu.Lock()
			m.leakyData = append(m.leakyData, data)
			m.mu.Unlock()
		}
	}
}

func (m *Manager) runGoroutineLeakWorkload(ctx context.Context, intensity string) {
	var goroutineCount int
	var interval time.Duration

	switch intensity {
	case "low":
		goroutineCount = 5
		interval = time.Second * 2
	case "medium":
		goroutineCount = 20
		interval = time.Second
	case "high":
		goroutineCount = 50
		interval = time.Millisecond * 500
	default:
		goroutineCount = 20
		interval = time.Second
	}

	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			// Create leaky goroutines
			for i := 0; i < goroutineCount; i++ {
				stopChan := make(chan struct{})
				m.mu.Lock()
				m.leakyGoroutines = append(m.leakyGoroutines, stopChan)
				m.mu.Unlock()

				go func(stop chan struct{}) {
					ticker := time.NewTicker(time.Millisecond * 100)
					defer ticker.Stop()

					for {
						select {
						case <-stop:
							return
						case <-ticker.C:
							// Do some work to keep goroutine busy
							_ = rand.Intn(1000)
						}
					}
				}(stopChan)
			}
		}
	}
}

func (m *Manager) runGCPressureWorkload(ctx context.Context, intensity string) {
	var allocSize int
	var allocCount int
	var interval time.Duration

	switch intensity {
	case "low":
		allocSize = 1024 * 10 // 10KB
		allocCount = 100
		interval = time.Millisecond * 100
	case "medium":
		allocSize = 1024 * 50 // 50KB
		allocCount = 200
		interval = time.Millisecond * 50
	case "high":
		allocSize = 1024 * 100 // 100KB
		allocCount = 500
		interval = time.Millisecond * 10
	default:
		allocSize = 1024 * 50
		allocCount = 200
		interval = time.Millisecond * 50
	}

	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			// Allocate and immediately discard to create GC pressure
			for i := 0; i < allocCount; i++ {
				data := make([]byte, allocSize)
				for j := range data {
					data[j] = byte(rand.Intn(256))
				}
				// Data goes out of scope and becomes eligible for GC
				_ = data
			}
		}
	}
}

func (m *Manager) runNormalWorkload(ctx context.Context, intensity string) {
	var allocSize int
	var interval time.Duration
	var holdTime time.Duration

	switch intensity {
	case "low":
		allocSize = 1024 * 10 // 10KB
		interval = time.Millisecond * 200
		holdTime = time.Millisecond * 100
	case "medium":
		allocSize = 1024 * 50 // 50KB
		interval = time.Millisecond * 100
		holdTime = time.Millisecond * 50
	case "high":
		allocSize = 1024 * 100 // 100KB
		interval = time.Millisecond * 50
		holdTime = time.Millisecond * 25
	default:
		allocSize = 1024 * 50
		interval = time.Millisecond * 100
		holdTime = time.Millisecond * 50
	}

	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			// Allocate memory, use it briefly, then release
			data := make([]byte, allocSize)
			for i := range data {
				data[i] = byte(rand.Intn(256))
			}

			// Hold the memory for a short time
			time.Sleep(holdTime)

			// Process the data (simulate work)
			sum := 0
			for _, b := range data {
				sum += int(b)
			}
			_ = sum // Use the result

			// Data goes out of scope and becomes eligible for GC
		}
	}
}

func (m *Manager) cleanupLeakyResources() {
	// Stop all leaky goroutines
	for _, stopChan := range m.leakyGoroutines {
		close(stopChan)
	}
	m.leakyGoroutines = m.leakyGoroutines[:0]

	// Clear leaky data (but keep the slice to demonstrate partial cleanup)
	if len(m.leakyData) > 0 {
		// Only clean up half of the leaked data to simulate partial cleanup
		halfway := len(m.leakyData) / 2
		m.leakyData = m.leakyData[:halfway]
	}
}