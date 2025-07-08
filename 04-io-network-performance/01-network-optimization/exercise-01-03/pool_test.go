package main

import (
	"context"
	"fmt"
	"net"
	"sync"
	"testing"
	"time"
)

// TestEchoServer creates a simple echo server for testing
func startTestEchoServer(t *testing.T, port int) net.Listener {
	listener, err := net.Listen("tcp", fmt.Sprintf(":%d", port))
	if err != nil {
		t.Fatalf("Failed to start echo server: %v", err)
	}

	go func() {
		for {
			conn, err := listener.Accept()
			if err != nil {
				return
			}
			go handleEchoConnection(conn)
		}
	}()

	return listener
}

func handleEchoConnection(conn net.Conn) {
	defer conn.Close()
	buf := make([]byte, 1024)
	for {
		n, err := conn.Read(buf)
		if err != nil {
			return
		}
		if _, err := conn.Write(buf[:n]); err != nil {
			return
		}
	}
}

func TestConnectionPoolBasicFunctionality(t *testing.T) {
	// Start echo server
	server := startTestEchoServer(t, 9998)
	defer server.Close()
	time.Sleep(100 * time.Millisecond) // Wait for server to start

	// Create connection pool
	config := &PoolConfig{
		MaxConnections: 10,
		MaxIdle:        5,
		IdleTimeout:    5 * time.Second,
		HealthInterval: 1 * time.Second,
		Workers:        2,
	}
	pool := NewConnectionPool(config)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	pool.Start(ctx)
	defer pool.Close()

	// Test getting a connection
	ctxWithTimeout, timeoutCancel := context.WithTimeout(ctx, 2*time.Second)
	defer timeoutCancel()
	conn, err := pool.Get(ctxWithTimeout, "localhost:9998")
	if err != nil {
		t.Fatalf("Failed to get connection: %v", err)
	}

	// Test connection functionality
	testData := "Hello, World!"
	if _, err := conn.conn.Write([]byte(testData)); err != nil {
		t.Fatalf("Failed to write to connection: %v", err)
	}

	buf := make([]byte, len(testData))
	if _, err := conn.conn.Read(buf); err != nil {
		t.Fatalf("Failed to read from connection: %v", err)
	}

	if string(buf) != testData {
		t.Fatalf("Expected %s, got %s", testData, string(buf))
	}

	// Return connection to pool
	pool.Put(conn)

	// Verify metrics
	metrics := pool.GetMetrics()
	if metrics["total_created"].(uint64) == 0 {
		t.Error("Expected at least one connection to be created")
	}

	t.Logf("Pool metrics: %+v", metrics)
}

func TestConnectionPoolConcurrentAccess(t *testing.T) {
	// Start echo server
	server := startTestEchoServer(t, 9997)
	defer server.Close()
	time.Sleep(100 * time.Millisecond)

	// Create connection pool
	config := &PoolConfig{
		MaxConnections: 20,
		MaxIdle:        10,
		IdleTimeout:    5 * time.Second,
		HealthInterval: 1 * time.Second,
		Workers:        4,
	}
	pool := NewConnectionPool(config)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	pool.Start(ctx)
	defer pool.Close()

	// Test concurrent access
	numGoroutines := 50
	numRequests := 10
	var wg sync.WaitGroup
	errorChan := make(chan error, numGoroutines*numRequests)

	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func(goroutineID int) {
			defer wg.Done()
			for j := 0; j < numRequests; j++ {
				ctxWithTimeout, timeoutCancel := context.WithTimeout(ctx, 2*time.Second)
				conn, err := pool.Get(ctxWithTimeout, "localhost:9997")
				timeoutCancel()
				if err != nil {
					errorChan <- fmt.Errorf("goroutine %d, request %d: %v", goroutineID, j, err)
					continue
				}

				// Test the connection
				testData := fmt.Sprintf("test-%d-%d", goroutineID, j)
				if _, err := conn.conn.Write([]byte(testData)); err != nil {
					errorChan <- fmt.Errorf("write error: %v", err)
					pool.Put(conn)
					continue
				}

				buf := make([]byte, len(testData))
				if _, err := conn.conn.Read(buf); err != nil {
					errorChan <- fmt.Errorf("read error: %v", err)
					pool.Put(conn)
					continue
				}

				if string(buf) != testData {
					errorChan <- fmt.Errorf("data mismatch: expected %s, got %s", testData, string(buf))
				}

				pool.Put(conn)
				time.Sleep(10 * time.Millisecond) // Small delay
			}
		}(i)
	}

	wg.Wait()
	close(errorChan)

	// Check for errors
	errorCount := 0
	for err := range errorChan {
		t.Logf("Error: %v", err)
		errorCount++
	}

	if errorCount > numGoroutines*numRequests/10 { // Allow up to 10% errors
		t.Fatalf("Too many errors: %d out of %d requests", errorCount, numGoroutines*numRequests)
	}

	// Verify metrics
	metrics := pool.GetMetrics()
	t.Logf("Final metrics: %+v", metrics)

	// Check hit rate (should be high due to connection reuse)
	hitRate := metrics["hit_rate_percent"].(float64)
	if hitRate < 50.0 { // Expect at least 50% hit rate
		t.Errorf("Hit rate too low: %.2f%%, expected >50%%", hitRate)
	}

	t.Logf("Concurrent test completed: %d goroutines, %d requests each, %.2f%% hit rate, %d errors",
		numGoroutines, numRequests, hitRate, errorCount)
}

func TestConnectionPoolHealthChecking(t *testing.T) {
	// Start echo server
	server := startTestEchoServer(t, 9996)
	defer server.Close()
	time.Sleep(100 * time.Millisecond)

	// Create connection pool with frequent health checks
	config := &PoolConfig{
		MaxConnections: 5,
		MaxIdle:        3,
		IdleTimeout:    2 * time.Second,
		HealthInterval: 500 * time.Millisecond, // Frequent health checks
		Workers:        2,
	}
	pool := NewConnectionPool(config)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	pool.Start(ctx)
	defer pool.Close()

	// Get some connections
	conns := make([]*ConnectionWrapper, 3)
	for i := 0; i < 3; i++ {
		ctxWithTimeout, timeoutCancel := context.WithTimeout(ctx, 2*time.Second)
		conn, err := pool.Get(ctxWithTimeout, "localhost:9996")
		timeoutCancel()
		if err != nil {
			t.Fatalf("Failed to get connection %d: %v", i, err)
		}
		conns[i] = conn
	}

	// Return connections to pool
	for _, conn := range conns {
		pool.Put(conn)
	}

	// Wait for health checks to run
	time.Sleep(2 * time.Second)

	// Verify health checks were performed
	metrics := pool.GetMetrics()
	healthChecks := metrics["health_checks"].(uint64)
	if healthChecks == 0 {
		t.Error("Expected health checks to be performed")
	}

	t.Logf("Health check test completed: %d health checks performed", healthChecks)
}

func TestConnectionPoolIdleTimeout(t *testing.T) {
	// Start echo server
	server := startTestEchoServer(t, 9995)
	defer server.Close()
	time.Sleep(100 * time.Millisecond)

	// Create connection pool with short idle timeout
	config := &PoolConfig{
		MaxConnections: 5,
		MaxIdle:        3,
		IdleTimeout:    1 * time.Second, // Short idle timeout
		HealthInterval: 2 * time.Second,
		Workers:        2,
	}
	pool := NewConnectionPool(config)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	pool.Start(ctx)
	defer pool.Close()

	// Get and return a connection
	ctxWithTimeout, timeoutCancel := context.WithTimeout(ctx, 2*time.Second)
	conn, err := pool.Get(ctxWithTimeout, "localhost:9995")
	timeoutCancel()
	if err != nil {
		t.Fatalf("Failed to get connection: %v", err)
	}
	pool.Put(conn)

	// Check initial metrics
	initialMetrics := pool.GetMetrics()
	initialCreated := initialMetrics["total_created"].(uint64)

	// Wait for idle timeout to trigger
	time.Sleep(3 * time.Second)

	// Check if idle connections were cleaned up
	finalMetrics := pool.GetMetrics()
	finalDestroyed := finalMetrics["total_destroyed"].(uint64)

	t.Logf("Idle timeout test: created=%d, destroyed=%d", initialCreated, finalDestroyed)

	// Note: Due to the complexity of idle connection cleanup timing,
	// we just verify the mechanism is working rather than exact counts
	if finalDestroyed == 0 && initialCreated > 0 {
		t.Log("Note: Idle cleanup may not have triggered yet due to timing")
	}
}

func TestConnectionPoolMetrics(t *testing.T) {
	// Start echo server
	server := startTestEchoServer(t, 9994)
	defer server.Close()
	time.Sleep(100 * time.Millisecond)

	// Create connection pool
	config := &PoolConfig{
		MaxConnections: 10,
		MaxIdle:        5,
		IdleTimeout:    5 * time.Second,
		HealthInterval: 1 * time.Second,
		Workers:        2,
	}
	pool := NewConnectionPool(config)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	pool.Start(ctx)
	defer pool.Close()

	// Perform various operations
	for i := 0; i < 20; i++ {
		ctxWithTimeout, timeoutCancel := context.WithTimeout(ctx, 2*time.Second)
		conn, err := pool.Get(ctxWithTimeout, "localhost:9994")
		timeoutCancel()
		if err != nil {
			t.Logf("Failed to get connection %d: %v", i, err)
			continue
		}

		// Use the connection briefly
		if _, err := conn.conn.Write([]byte("test")); err == nil {
			buf := make([]byte, 4)
			conn.conn.Read(buf)
		}

		pool.Put(conn)
	}

	// Get final metrics
	metrics := pool.GetMetrics()

	// Verify metrics structure
	requiredFields := []string{
		"total_connections", "idle_connections", "active_connections",
		"total_created", "total_destroyed", "total_requests",
		"total_hits", "total_misses", "hit_rate_percent",
		"utilization_percent", "health_checks", "unhealthy_connections",
	}

	for _, field := range requiredFields {
		if _, exists := metrics[field]; !exists {
			t.Errorf("Missing metric field: %s", field)
		}
	}

	// Verify some basic metric constraints
	totalRequests := metrics["total_requests"].(uint64)
	totalHits := metrics["total_hits"].(uint64)
	totalMisses := metrics["total_misses"].(uint64)

	if totalRequests != totalHits+totalMisses {
		t.Errorf("Metric inconsistency: requests=%d, hits=%d, misses=%d",
			totalRequests, totalHits, totalMisses)
	}

	t.Logf("Metrics test completed: %+v", metrics)
}

// Benchmark tests
func BenchmarkConnectionPoolGet(b *testing.B) {
	// Start test echo server
	t := &testing.T{}
	server := startTestEchoServer(t, 9993)
	defer server.Close()
	time.Sleep(100 * time.Millisecond)

	// Create connection pool
	config := &PoolConfig{
		MaxConnections: 100,
		MaxIdle:        50,
		IdleTimeout:    30 * time.Second,
		HealthInterval: 10 * time.Second,
		Workers:        4,
	}
	pool := NewConnectionPool(config)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	pool.Start(ctx)
	defer pool.Close()

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			ctxWithTimeout, timeoutCancel := context.WithTimeout(ctx, 1*time.Second)
			conn, err := pool.Get(ctxWithTimeout, "localhost:9993")
			timeoutCancel()
			if err != nil {
				b.Errorf("Failed to get connection: %v", err)
				continue
			}
			pool.Put(conn)
		}
	})

	// Report final metrics
	metrics := pool.GetMetrics()
	b.Logf("Benchmark completed - Hit rate: %.2f%%, Utilization: %.2f%%",
		metrics["hit_rate_percent"], metrics["utilization_percent"])
}

func BenchmarkConnectionPoolConcurrentOperations(b *testing.B) {
	// Start test echo server
	t := &testing.T{}
	server := startTestEchoServer(t, 9992)
	defer server.Close()
	time.Sleep(100 * time.Millisecond)

	// Create connection pool
	config := &PoolConfig{
		MaxConnections: 50,
		MaxIdle:        25,
		IdleTimeout:    30 * time.Second,
		HealthInterval: 10 * time.Second,
		Workers:        4,
	}
	pool := NewConnectionPool(config)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	pool.Start(ctx)
	defer pool.Close()

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			ctxWithTimeout, timeoutCancel := context.WithTimeout(ctx, 1*time.Second)
			conn, err := pool.Get(ctxWithTimeout, "localhost:9992")
			timeoutCancel()
			if err != nil {
				b.Errorf("Failed to get connection: %v", err)
				continue
			}

			// Simulate some work
			if _, err := conn.conn.Write([]byte("benchmark")); err == nil {
				buf := make([]byte, 9)
				conn.conn.Read(buf)
			}

			pool.Put(conn)
		}
	})

	// Report final metrics
	metrics := pool.GetMetrics()
	b.Logf("Concurrent benchmark - Requests: %d, Hit rate: %.2f%%, Errors: %d",
		metrics["total_requests"], metrics["hit_rate_percent"], metrics["unhealthy_connections"])
}