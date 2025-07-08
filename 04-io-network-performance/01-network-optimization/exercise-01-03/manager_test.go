package main

import (
	"context"
	"fmt"
	"net"
	"sync"
	"testing"
	"time"
)

// TestConnectionPoolManager tests the connection pool manager functionality
func TestConnectionPoolManager(t *testing.T) {
	// Start test backends
	backend1 := startTestBackend(t, ":0")
	backend2 := startTestBackend(t, ":0")
	defer backend1.Close()
	defer backend2.Close()

	// Create manager
	config := &PoolConfig{
		MaxConnections: 10,
		IdleTimeout:    30 * time.Second,
		HealthInterval: 5 * time.Second,
	}
	manager := NewConnectionPoolManager(config)

	// Add backends
	manager.AddBackend(backend1.Addr().String(), 1)
	manager.AddBackend(backend2.Addr().String(), 1)

	// Start manager
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	go func() {
		if err := manager.Start(ctx, ":0"); err != nil {
			t.Errorf("Failed to start manager: %v", err)
		}
	}()

	// Wait for manager to start
	time.Sleep(100 * time.Millisecond)

	// Test connection
	conn, err := net.Dial("tcp", manager.listener.Addr().String())
	if err != nil {
		t.Fatalf("Failed to connect to manager: %v", err)
	}
	defer conn.Close()

	// Send test message
	message := "test message"
	if _, err := conn.Write([]byte(message)); err != nil {
		t.Fatalf("Failed to write message: %v", err)
	}

	// Read response
	buf := make([]byte, len(message))
	if _, err := conn.Read(buf); err != nil {
		t.Fatalf("Failed to read response: %v", err)
	}

	if string(buf) != message {
		t.Errorf("Expected %s, got %s", message, string(buf))
	}
}

// TestLoadBalancingStrategies tests different load balancing strategies
func TestLoadBalancingStrategies(t *testing.T) {
	strategies := []LoadBalanceStrategy{RoundRobin, LeastConnections, Random}

	for _, strategy := range strategies {
		t.Run(fmt.Sprintf("Strategy_%s", strategy), func(t *testing.T) {
			// Start test backends
			backend1 := startTestBackend(t, ":0")
			backend2 := startTestBackend(t, ":0")
			defer backend1.Close()
			defer backend2.Close()

			// Create manager with specific strategy
			config := &PoolConfig{
				MaxConnections: 5,
				IdleTimeout:    30 * time.Second,
				HealthInterval: 5 * time.Second,
			}
			manager := NewConnectionPoolManager(config)
			manager.strategy = strategy

			// Add backends
			manager.AddBackend(backend1.Addr().String(), 1)
			manager.AddBackend(backend2.Addr().String(), 1)

			// Start manager
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()

			go func() {
				if err := manager.Start(ctx, ":0"); err != nil {
					t.Errorf("Failed to start manager: %v", err)
				}
			}()

			// Wait for manager to start
			time.Sleep(100 * time.Millisecond)

			// Test multiple connections to verify load balancing
			var wg sync.WaitGroup
			numConnections := 10

			for i := 0; i < numConnections; i++ {
				wg.Add(1)
				go func(id int) {
					defer wg.Done()

					conn, err := net.Dial("tcp", manager.listener.Addr().String())
					if err != nil {
						t.Errorf("Failed to connect: %v", err)
						return
					}
					defer conn.Close()

					message := fmt.Sprintf("test-%d", id)
					if _, err := conn.Write([]byte(message)); err != nil {
						t.Errorf("Failed to write: %v", err)
						return
					}

					buf := make([]byte, len(message))
					if _, err := conn.Read(buf); err != nil {
						t.Errorf("Failed to read: %v", err)
						return
					}

					if string(buf) != message {
						t.Errorf("Expected %s, got %s", message, string(buf))
					}
				}(i)
			}

			wg.Wait()
		})
	}
}

// TestConnectionPoolManagerHealthCheck tests health checking functionality
func TestConnectionPoolManagerHealthCheck(t *testing.T) {
	// Start test backend
	backend := startTestBackend(t, ":0")
	defer backend.Close()

	// Create manager
	config := &PoolConfig{
		MaxConnections: 5,
		IdleTimeout:    30 * time.Second,
		HealthInterval: 100 * time.Millisecond,
	}
	manager := NewConnectionPoolManager(config)

	// Add backend
	manager.AddBackend(backend.Addr().String(), 1)

	// Start manager
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	go func() {
		if err := manager.Start(ctx, ":0"); err != nil {
			t.Errorf("Failed to start manager: %v", err)
		}
	}()

	// Wait for manager to start and perform health checks
	time.Sleep(200 * time.Millisecond)

	// Verify backend is healthy
	manager.mutex.RLock()
	if len(manager.backends) != 1 {
		t.Errorf("Expected 1 backend, got %d", len(manager.backends))
	}
	backendInfo := manager.backends[0]
	manager.mutex.RUnlock()

	if !backendInfo.IsHealthy() {
		t.Error("Backend should be healthy")
	}

	// Close backend to simulate failure
	backend.Close()

	// Wait for health check to detect failure
	time.Sleep(300 * time.Millisecond)

	// Verify backend is marked as unhealthy
	manager.mutex.RLock()
	backendInfo = manager.backends[0]
	manager.mutex.RUnlock()

	if backendInfo.IsHealthy() {
		t.Error("Backend should be unhealthy after closure")
	}
}

// TestConnectionPoolManagerConcurrency tests concurrent access to the manager
func TestConnectionPoolManagerConcurrency(t *testing.T) {
	// Start multiple test backends
	backends := make([]net.Listener, 3)
	for i := 0; i < 3; i++ {
		backends[i] = startTestBackend(t, ":0")
		defer backends[i].Close()
	}

	// Create manager
	config := &PoolConfig{
		MaxConnections: 10,
		IdleTimeout:    30 * time.Second,
		HealthInterval: 5 * time.Second,
	}
	manager := NewConnectionPoolManager(config)

	// Add backends
	for _, backend := range backends {
		manager.AddBackend(backend.Addr().String(), 1)
	}

	// Start manager
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	go func() {
		if err := manager.Start(ctx, ":0"); err != nil {
			t.Errorf("Failed to start manager: %v", err)
		}
	}()

	// Wait for manager to start
	time.Sleep(100 * time.Millisecond)

	// Test concurrent connections
	var wg sync.WaitGroup
	numClients := 50
	requestsPerClient := 10

	for i := 0; i < numClients; i++ {
		wg.Add(1)
		go func(clientID int) {
			defer wg.Done()

			for j := 0; j < requestsPerClient; j++ {
				conn, err := net.Dial("tcp", manager.listener.Addr().String())
				if err != nil {
					t.Errorf("Client %d: Failed to connect: %v", clientID, err)
					return
				}

				message := fmt.Sprintf("client-%d-req-%d", clientID, j)
				if _, err := conn.Write([]byte(message)); err != nil {
					t.Errorf("Client %d: Failed to write: %v", clientID, err)
					conn.Close()
					return
				}

				buf := make([]byte, len(message))
				if _, err := conn.Read(buf); err != nil {
					t.Errorf("Client %d: Failed to read: %v", clientID, err)
					conn.Close()
					return
				}

				if string(buf) != message {
					t.Errorf("Client %d: Expected %s, got %s", clientID, message, string(buf))
				}

				conn.Close()
				time.Sleep(1 * time.Millisecond) // Small delay
			}
		}(i)
	}

	wg.Wait()
}

// TestConnectionPoolManagerMetrics tests metrics collection
func TestConnectionPoolManagerMetrics(t *testing.T) {
	// Start test backend
	backend := startTestBackend(t, ":0")
	defer backend.Close()

	// Create manager
	config := &PoolConfig{
		MaxConnections: 5,
		IdleTimeout:    30 * time.Second,
		HealthInterval: 5 * time.Second,
	}
	manager := NewConnectionPoolManager(config)

	// Add backend
	manager.AddBackend(backend.Addr().String(), 1)

	// Start manager
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	go func() {
		if err := manager.Start(ctx, ":0"); err != nil {
			t.Errorf("Failed to start manager: %v", err)
		}
	}()

	// Wait for manager to start
	time.Sleep(100 * time.Millisecond)

	// Make some connections to generate metrics
	numConnections := 5
	for i := 0; i < numConnections; i++ {
		conn, err := net.Dial("tcp", manager.listener.Addr().String())
		if err != nil {
			t.Fatalf("Failed to connect: %v", err)
		}

		message := fmt.Sprintf("test-%d", i)
		conn.Write([]byte(message))
		buf := make([]byte, len(message))
		conn.Read(buf)
		conn.Close()
	}

	// Wait a bit for metrics to be updated
	time.Sleep(100 * time.Millisecond)

	// Check metrics
	manager.mutex.RLock()
	if len(manager.backends) != 1 {
		t.Errorf("Expected 1 backend, got %d", len(manager.backends))
	}
	backendInfo := manager.backends[0]
	manager.mutex.RUnlock()

	// Verify that some connections were made
	metrics := backendInfo.Pool.GetMetrics()
	t.Logf("Pool metrics: %+v", metrics)
	if metrics["total_requests"].(uint64) == 0 {
		t.Error("Expected some requests to be made")
	}
}

// BenchmarkConnectionPoolManager benchmarks the connection pool manager
func BenchmarkConnectionPoolManager(b *testing.B) {
	// Start test backend
	backend := startTestBackend(b, ":0")
	defer backend.Close()

	// Create manager
	config := &PoolConfig{
		MaxConnections: 20,
		IdleTimeout:    30 * time.Second,
		HealthInterval: 5 * time.Second,
	}
	manager := NewConnectionPoolManager(config)
	manager.AddBackend(backend.Addr().String(), 1)

	// Start manager
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	go func() {
		if err := manager.Start(ctx, ":0"); err != nil {
			b.Errorf("Failed to start manager: %v", err)
		}
	}()

	// Wait for manager to start
	time.Sleep(100 * time.Millisecond)

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			conn, err := net.Dial("tcp", manager.listener.Addr().String())
			if err != nil {
				b.Errorf("Failed to connect: %v", err)
				continue
			}

			message := "benchmark-test"
			conn.Write([]byte(message))
			buf := make([]byte, len(message))
			conn.Read(buf)
			conn.Close()
		}
	})
}

// BenchmarkConnectionPoolManagerWithReuse benchmarks with connection reuse
func BenchmarkConnectionPoolManagerWithReuse(b *testing.B) {
	// Start test backend
	backend := startTestBackend(b, ":0")
	defer backend.Close()

	// Create manager
	config := &PoolConfig{
		MaxConnections: 20,
		IdleTimeout:    30 * time.Second,
		HealthInterval: 5 * time.Second,
	}
	manager := NewConnectionPoolManager(config)
	manager.AddBackend(backend.Addr().String(), 1)

	// Start manager
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	go func() {
		if err := manager.Start(ctx, ":0"); err != nil {
			b.Errorf("Failed to start manager: %v", err)
		}
	}()

	// Wait for manager to start
	time.Sleep(100 * time.Millisecond)

	// Pre-establish connections
	conns := make([]net.Conn, b.N)
	for i := 0; i < b.N; i++ {
		conn, err := net.Dial("tcp", manager.listener.Addr().String())
		if err != nil {
			b.Fatalf("Failed to connect: %v", err)
		}
		conns[i] = conn
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		message := "benchmark-reuse-test"
		conns[i].Write([]byte(message))
		buf := make([]byte, len(message))
		conns[i].Read(buf)
	}
	b.StopTimer()

	// Clean up connections
	for _, conn := range conns {
		conn.Close()
	}
}

// startTestBackend starts a simple echo server for testing
func startTestBackend(t testing.TB, addr string) net.Listener {
	listener, err := net.Listen("tcp", addr)
	if err != nil {
		t.Fatalf("Failed to start test backend: %v", err)
	}

	go func() {
		for {
			conn, err := listener.Accept()
			if err != nil {
				return // Listener closed
			}

			go func(c net.Conn) {
				defer c.Close()
				buf := make([]byte, 1024)
				for {
					n, err := c.Read(buf)
					if err != nil {
						return
					}
					// Echo back the data
					c.Write(buf[:n])
				}
			}(conn)
		}
	}()

	return listener
}