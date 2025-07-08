package main

import (
	"context"
	"fmt"
	"log"
	"net"
	"runtime"
	"sync"
	"sync/atomic"
	"time"
)

// LoadBalanceStrategy defines load balancing strategies
type LoadBalanceStrategy int

const (
	RoundRobin LoadBalanceStrategy = iota
	LeastConnections
	Random
)

// String returns the string representation of LoadBalanceStrategy
func (s LoadBalanceStrategy) String() string {
	switch s {
	case RoundRobin:
		return "RoundRobin"
	case LeastConnections:
		return "LeastConnections"
	case Random:
		return "Random"
	default:
		return "Unknown"
	}
}

// Backend represents a backend server
type Backend struct {
	Address string
	Weight  int
	Pool    *ConnectionPool
	Healthy bool
	mutex   sync.RWMutex
}

// IsHealthy returns if the backend is healthy
func (b *Backend) IsHealthy() bool {
	b.mutex.RLock()
	defer b.mutex.RUnlock()
	return b.Healthy
}

// SetHealthy sets the backend health status
func (b *Backend) SetHealthy(healthy bool) {
	b.mutex.Lock()
	defer b.mutex.Unlock()
	b.Healthy = healthy
}

// ConnectionPoolManager manages multiple connection pools
type ConnectionPoolManager struct {
	config    *PoolConfig
	backends  []*Backend
	listener  net.Listener
	strategy  LoadBalanceStrategy
	rrCounter uint64 // Round-robin counter
	mutex     sync.RWMutex
	shutdown  chan struct{}
	wg        sync.WaitGroup

	// Metrics
	totalConnections uint64
	activeConnections uint64
	totalRequests    uint64
	totalErrors      uint64
}

// NewConnectionPoolManager creates a new connection pool manager
func NewConnectionPoolManager(config *PoolConfig) *ConnectionPoolManager {
	return &ConnectionPoolManager{
		config:   config,
		backends: make([]*Backend, 0),
		strategy: RoundRobin,
		shutdown: make(chan struct{}),
	}
}

// AddBackend adds a backend server
func (m *ConnectionPoolManager) AddBackend(address string, weight int) {
	pool := NewConnectionPool(m.config)
	backend := &Backend{
		Address: address,
		Weight:  weight,
		Pool:    pool,
		Healthy: true,
	}

	m.mutex.Lock()
	m.backends = append(m.backends, backend)
	m.mutex.Unlock()

	log.Printf("Added backend: %s (weight: %d)", address, weight)
}

// Start starts the connection pool manager
func (m *ConnectionPoolManager) Start(ctx context.Context, listenAddr string) error {
	// Start all backend pools
	m.mutex.RLock()
	for _, backend := range m.backends {
		backend.Pool.Start(ctx)
	}
	m.mutex.RUnlock()

	// Start TCP listener
	listener, err := net.Listen("tcp", listenAddr)
	if err != nil {
		return fmt.Errorf("failed to listen on %s: %w", listenAddr, err)
	}
	m.listener = listener

	// Start background tasks
	m.wg.Add(3)
	go m.acceptConnections(ctx)
	go m.healthChecker(ctx)
	go m.metricsReporter(ctx)

	return nil
}

// acceptConnections accepts and handles incoming connections
func (m *ConnectionPoolManager) acceptConnections(ctx context.Context) {
	defer m.wg.Done()

	for {
		select {
		case <-ctx.Done():
			return
		case <-m.shutdown:
			return
		default:
		}

		conn, err := m.listener.Accept()
		if err != nil {
			select {
			case <-ctx.Done():
				return
			case <-m.shutdown:
				return
			default:
				log.Printf("Accept error: %v", err)
				continue
			}
		}

		atomic.AddUint64(&m.totalConnections, 1)
		atomic.AddUint64(&m.activeConnections, 1)

		// Handle connection in a goroutine
		go m.handleConnection(ctx, conn)
	}
}

// handleConnection handles a client connection
func (m *ConnectionPoolManager) handleConnection(ctx context.Context, clientConn net.Conn) {
	defer func() {
		clientConn.Close()
		atomic.AddUint64(&m.activeConnections, ^uint64(0)) // Decrement
	}()

	atomic.AddUint64(&m.totalRequests, 1)

	// Select backend using load balancing strategy
	backend := m.selectBackend()
	if backend == nil {
		log.Println("No healthy backends available")
		atomic.AddUint64(&m.totalErrors, 1)
		return
	}

	// Get connection from pool
	ctxWithTimeout, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	backendConn, err := backend.Pool.Get(ctxWithTimeout, backend.Address)
	if err != nil {
		log.Printf("Failed to get connection from pool: %v", err)
		atomic.AddUint64(&m.totalErrors, 1)
		return
	}

	// Proxy data between client and backend
	m.proxyConnection(clientConn, backendConn)

	// Return connection to pool
	backend.Pool.Put(backendConn)
}

// selectBackend selects a backend using the configured strategy
func (m *ConnectionPoolManager) selectBackend() *Backend {
	m.mutex.RLock()
	defer m.mutex.RUnlock()

	// Filter healthy backends
	healthyBackends := make([]*Backend, 0, len(m.backends))
	for _, backend := range m.backends {
		if backend.IsHealthy() {
			healthyBackends = append(healthyBackends, backend)
		}
	}

	if len(healthyBackends) == 0 {
		return nil
	}

	switch m.strategy {
	case RoundRobin:
		idx := atomic.AddUint64(&m.rrCounter, 1) % uint64(len(healthyBackends))
		return healthyBackends[idx]
	case LeastConnections:
		return m.selectLeastConnections(healthyBackends)
	case Random:
		idx := time.Now().UnixNano() % int64(len(healthyBackends))
		return healthyBackends[idx]
	default:
		return healthyBackends[0]
	}
}

// selectLeastConnections selects backend with least connections
func (m *ConnectionPoolManager) selectLeastConnections(backends []*Backend) *Backend {
	var selected *Backend
	minConnections := int(^uint(0) >> 1) // Max int

	for _, backend := range backends {
		metrics := backend.Pool.GetMetrics()
		activeConns := metrics["active_connections"].(int)
		if activeConns < minConnections {
			minConnections = activeConns
			selected = backend
		}
	}

	return selected
}

// proxyConnection proxies data between client and backend connections
func (m *ConnectionPoolManager) proxyConnection(clientConn net.Conn, backendConn *ConnectionWrapper) {
	// Set timeouts
	clientConn.SetDeadline(time.Now().Add(30 * time.Second))
	backendConn.conn.SetDeadline(time.Now().Add(30 * time.Second))

	// Create channels for bidirectional copying
	done := make(chan struct{}, 2)

	// Client to backend
	go func() {
		defer func() { done <- struct{}{} }()
		buf := make([]byte, 32*1024) // 32KB buffer
		for {
			n, err := clientConn.Read(buf)
			if err != nil {
				return
			}
			if _, err := backendConn.conn.Write(buf[:n]); err != nil {
				return
			}
		}
	}()

	// Backend to client
	go func() {
		defer func() { done <- struct{}{} }()
		buf := make([]byte, 32*1024) // 32KB buffer
		for {
			n, err := backendConn.conn.Read(buf)
			if err != nil {
				return
			}
			if _, err := clientConn.Write(buf[:n]); err != nil {
				return
			}
		}
	}()

	// Wait for either direction to complete
	<-done
}

// healthChecker periodically checks backend health
func (m *ConnectionPoolManager) healthChecker(ctx context.Context) {
	defer m.wg.Done()
	ticker := time.NewTicker(m.config.HealthInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-m.shutdown:
			return
		case <-ticker.C:
			m.performBackendHealthCheck()
		}
	}
}

// performBackendHealthCheck checks health of all backends
func (m *ConnectionPoolManager) performBackendHealthCheck() {
	m.mutex.RLock()
	backends := make([]*Backend, len(m.backends))
	copy(backends, m.backends)
	m.mutex.RUnlock()

	for _, backend := range backends {
		healthy := m.checkBackendHealth(backend.Address)
		backend.SetHealthy(healthy)
		if !healthy {
			log.Printf("Backend %s is unhealthy", backend.Address)
		}
	}
}

// checkBackendHealth performs a health check on a backend
func (m *ConnectionPoolManager) checkBackendHealth(address string) bool {
	conn, err := net.DialTimeout("tcp", address, 2*time.Second)
	if err != nil {
		return false
	}
	defer conn.Close()

	// Set a short deadline for the health check
	conn.SetDeadline(time.Now().Add(1 * time.Second))

	// Send a simple health check message
	if _, err := conn.Write([]byte("HEALTH_CHECK\n")); err != nil {
		return false
	}

	// Try to read response
	buf := make([]byte, 1024)
	_, err = conn.Read(buf)
	return err == nil
}

// metricsReporter periodically reports metrics
func (m *ConnectionPoolManager) metricsReporter(ctx context.Context) {
	defer m.wg.Done()
	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-m.shutdown:
			return
		case <-ticker.C:
			m.reportMetrics()
		}
	}
}

// reportMetrics logs current metrics
func (m *ConnectionPoolManager) reportMetrics() {
	totalConns := atomic.LoadUint64(&m.totalConnections)
	activeConns := atomic.LoadUint64(&m.activeConnections)
	totalReqs := atomic.LoadUint64(&m.totalRequests)
	totalErrs := atomic.LoadUint64(&m.totalErrors)

	errorRate := float64(0)
	if totalReqs > 0 {
		errorRate = float64(totalErrs) / float64(totalReqs) * 100
	}

	// Get memory stats
	var memStats runtime.MemStats
	runtime.ReadMemStats(&memStats)

	log.Printf("Metrics: Total=%d, Active=%d, Requests=%d, Errors=%d (%.2f%%), Memory=%d KB, Goroutines=%d",
		totalConns, activeConns, totalReqs, totalErrs, errorRate,
		memStats.Alloc/1024, runtime.NumGoroutine())

	// Report backend pool metrics
	m.mutex.RLock()
	for i, backend := range m.backends {
		metrics := backend.Pool.GetMetrics()
		log.Printf("Backend[%d] %s: Connections=%d, Idle=%d, Hit Rate=%.1f%%, Utilization=%.1f%%",
			i, backend.Address,
			metrics["total_connections"],
			metrics["idle_connections"],
			metrics["hit_rate_percent"],
			metrics["utilization_percent"])
	}
	m.mutex.RUnlock()
}

// Shutdown gracefully shuts down the manager
func (m *ConnectionPoolManager) Shutdown(ctx context.Context) error {
	log.Println("Shutting down connection pool manager...")

	// Close listener
	if m.listener != nil {
		m.listener.Close()
	}

	// Signal shutdown
	close(m.shutdown)

	// Wait for goroutines to finish with timeout
	done := make(chan struct{})
	go func() {
		m.wg.Wait()
		close(done)
	}()

	select {
	case <-done:
		log.Println("All goroutines finished")
	case <-ctx.Done():
		log.Println("Shutdown timeout reached")
	}

	// Close all backend pools
	m.mutex.RLock()
	for _, backend := range m.backends {
		backend.Pool.Close()
	}
	m.mutex.RUnlock()

	return nil
}

// PrintMetrics prints final metrics
func (m *ConnectionPoolManager) PrintMetrics() {
	totalConns := atomic.LoadUint64(&m.totalConnections)
	activeConns := atomic.LoadUint64(&m.activeConnections)
	totalReqs := atomic.LoadUint64(&m.totalRequests)
	totalErrs := atomic.LoadUint64(&m.totalErrors)

	errorRate := float64(0)
	if totalReqs > 0 {
		errorRate = float64(totalErrs) / float64(totalReqs) * 100
	}

	fmt.Printf("\n=== Final Connection Pool Metrics ===\n")
	fmt.Printf("Total Connections: %d\n", totalConns)
	fmt.Printf("Active Connections: %d\n", activeConns)
	fmt.Printf("Total Requests: %d\n", totalReqs)
	fmt.Printf("Total Errors: %d\n", totalErrs)
	fmt.Printf("Error Rate: %.2f%%\n", errorRate)

	// Print backend metrics
	m.mutex.RLock()
	for i, backend := range m.backends {
		metrics := backend.Pool.GetMetrics()
		fmt.Printf("\nBackend[%d] %s:\n", i, backend.Address)
		fmt.Printf("  Total Connections: %v\n", metrics["total_connections"])
		fmt.Printf("  Idle Connections: %v\n", metrics["idle_connections"])
		fmt.Printf("  Active Connections: %v\n", metrics["active_connections"])
		fmt.Printf("  Total Created: %v\n", metrics["total_created"])
		fmt.Printf("  Total Destroyed: %v\n", metrics["total_destroyed"])
		fmt.Printf("  Hit Rate: %.1f%%\n", metrics["hit_rate_percent"])
		fmt.Printf("  Utilization: %.1f%%\n", metrics["utilization_percent"])
		fmt.Printf("  Health Checks: %v\n", metrics["health_checks"])
		fmt.Printf("  Unhealthy Connections: %v\n", metrics["unhealthy_connections"])
	}
	m.mutex.RUnlock()

	// Performance evaluation
	fmt.Printf("\n=== Performance Evaluation ===\n")

	// Connection reuse rate (hit rate should be >95%)
	m.mutex.RLock()
	if len(m.backends) > 0 {
		metrics := m.backends[0].Pool.GetMetrics()
		hitRate := metrics["hit_rate_percent"].(float64)
		if hitRate >= 95.0 {
			fmt.Printf("✅ Connection Reuse Rate: %.1f%% (target: >95%%)\n", hitRate)
		} else {
			fmt.Printf("❌ Connection Reuse Rate: %.1f%% (target: >95%%)\n", hitRate)
		}

		// Pool utilization (should be >90%)
		utilization := metrics["utilization_percent"].(float64)
		if utilization >= 90.0 {
			fmt.Printf("✅ Pool Utilization: %.1f%% (target: >90%%)\n", utilization)
		} else {
			fmt.Printf("❌ Pool Utilization: %.1f%% (target: >90%%)\n", utilization)
		}
	}
	m.mutex.RUnlock()

	// Health check overhead (should be <1%)
	if errorRate <= 1.0 {
		fmt.Printf("✅ Error Rate: %.2f%% (target: <1%%)\n", errorRate)
	} else {
		fmt.Printf("❌ Error Rate: %.2f%% (target: <1%%)\n", errorRate)
	}

	fmt.Printf("\n")
}