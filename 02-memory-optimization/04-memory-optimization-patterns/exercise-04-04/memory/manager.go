package memory

import (
	"context"
	"runtime"
	"sync"
	"time"
)

// Manager handles memory monitoring and management
type Manager struct {
	limit       int64
	current     int64
	pressure    bool
	mu          sync.RWMutex
	cleanupFunc func()
}

// NewManager creates a new memory manager with the specified limit
func NewManager(limit int64) *Manager {
	return &Manager{
		limit: limit,
	}
}

// Monitor starts memory monitoring in a goroutine
func (m *Manager) Monitor(ctx context.Context) {
	ticker := time.NewTicker(1 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			m.updateMemoryStats()
		}
	}
}

// updateMemoryStats updates current memory usage and pressure status
func (m *Manager) updateMemoryStats() {
	var memStats runtime.MemStats
	runtime.ReadMemStats(&memStats)

	m.mu.Lock()
	m.current = int64(memStats.Alloc)
	m.pressure = m.current > m.limit*80/100 // 80% threshold
	m.mu.Unlock()
}

// IsUnderPressure returns true if memory usage is above threshold
func (m *Manager) IsUnderPressure() bool {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.pressure
}

// GetUsage returns current memory usage
func (m *Manager) GetUsage() int64 {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.current
}

// GetLimit returns the memory limit
func (m *Manager) GetLimit() int64 {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.limit
}

// TriggerCleanup forces garbage collection and runs cleanup functions
func (m *Manager) TriggerCleanup() {
	runtime.GC()
	runtime.GC() // Double GC for better cleanup

	m.mu.RLock()
	cleanupFunc := m.cleanupFunc
	m.mu.RUnlock()

	if cleanupFunc != nil {
		cleanupFunc()
	}
}

// SetCleanupFunc sets a custom cleanup function
func (m *Manager) SetCleanupFunc(fn func()) {
	m.mu.Lock()
	m.cleanupFunc = fn
	m.mu.Unlock()
}