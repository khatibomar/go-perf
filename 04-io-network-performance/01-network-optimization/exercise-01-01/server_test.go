package main

import (
	"bufio"
	"context"
	"fmt"
	"net"
	"runtime"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

// TestServerBasicFunctionality tests basic server operations
func TestServerBasicFunctionality(t *testing.T) {
	server := NewServer(10)
	
	// Start server in background
	go func() {
		if err := server.Start(":0"); err != nil {
			t.Errorf("Server failed to start: %v", err)
		}
	}()
	
	// Wait for server to start
	time.Sleep(100 * time.Millisecond)
	
	if server.listener == nil {
		t.Fatal("Server listener not initialized")
	}
	
	addr := server.listener.Addr().String()
	
	// Test single connection
	conn, err := net.Dial("tcp", addr)
	if err != nil {
		t.Fatalf("Failed to connect to server: %v", err)
	}
	defer conn.Close()
	
	// Send a message
	message := "Hello, Server!"
	_, err = fmt.Fprintf(conn, "%s\n", message)
	if err != nil {
		t.Fatalf("Failed to send message: %v", err)
	}
	
	// Read response
	reader := bufio.NewReader(conn)
	response, err := reader.ReadString('\n')
	if err != nil {
		t.Fatalf("Failed to read response: %v", err)
	}
	
	if response == "" {
		t.Error("Empty response received")
	}
	
	// Shutdown server
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	
	if err := server.Shutdown(ctx); err != nil {
		t.Errorf("Server shutdown failed: %v", err)
	}
}

// TestServerConcurrentConnections tests handling multiple concurrent connections
func TestServerConcurrentConnections(t *testing.T) {
	server := NewServer(100)
	
	// Start server
	go func() {
		if err := server.Start(":0"); err != nil {
			t.Errorf("Server failed to start: %v", err)
		}
	}()
	
	time.Sleep(100 * time.Millisecond)
	addr := server.listener.Addr().String()
	
	// Test concurrent connections
	numConnections := 50
	var wg sync.WaitGroup
	var successCount int64
	
	for i := 0; i < numConnections; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			
			conn, err := net.Dial("tcp", addr)
			if err != nil {
				t.Errorf("Connection %d failed: %v", id, err)
				return
			}
			defer conn.Close()
			
			message := fmt.Sprintf("Message from connection %d", id)
			_, err = fmt.Fprintf(conn, "%s\n", message)
			if err != nil {
				t.Errorf("Send failed for connection %d: %v", id, err)
				return
			}
			
			reader := bufio.NewReader(conn)
			_, err = reader.ReadString('\n')
			if err != nil {
				t.Errorf("Read failed for connection %d: %v", id, err)
				return
			}
			
			atomic.AddInt64(&successCount, 1)
		}(i)
	}
	
	wg.Wait()
	
	if successCount != int64(numConnections) {
		t.Errorf("Expected %d successful connections, got %d", numConnections, successCount)
	}
	
	// Shutdown
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	server.Shutdown(ctx)
}

// TestServerMetrics tests server metrics collection
func TestServerMetrics(t *testing.T) {
	server := NewServer(10)
	
	go func() {
		server.Start(":0")
	}()
	
	time.Sleep(100 * time.Millisecond)
	addr := server.listener.Addr().String()
	
	// Make some requests
	for i := 0; i < 5; i++ {
		conn, err := net.Dial("tcp", addr)
		if err != nil {
			t.Fatalf("Failed to connect: %v", err)
		}
		
		fmt.Fprintf(conn, "test message %d\n", i)
		reader := bufio.NewReader(conn)
		reader.ReadString('\n')
		conn.Close()
	}
	
	// Check metrics
	metrics := server.GetMetrics()
	
	if metrics["total_connections"].(int64) < 5 {
		t.Error("Total connections metric not updated correctly")
	}
	
	if metrics["total_requests"].(int64) < 5 {
		t.Error("Total requests metric not updated correctly")
	}
	
	if metrics["worker_pool_size"].(int) != 10 {
		t.Error("Worker pool size metric incorrect")
	}
	
	// Shutdown
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	server.Shutdown(ctx)
}

// TestServerGracefulShutdown tests graceful shutdown functionality
func TestServerGracefulShutdown(t *testing.T) {
	server := NewServer(10)
	
	go func() {
		server.Start(":0")
	}()
	
	time.Sleep(100 * time.Millisecond)
	addr := server.listener.Addr().String()
	
	// Send some requests to ensure server is working
	conn, err := net.Dial("tcp", addr)
	if err != nil {
		t.Fatalf("Failed to connect: %v", err)
	}
	
	reader := bufio.NewReader(conn)
	fmt.Fprintf(conn, "test message\n")
	_, err = reader.ReadString('\n')
	if err != nil {
		t.Fatalf("Failed to read response: %v", err)
	}
	conn.Close()
	
	// Start shutdown
	shutdownStart := time.Now()
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	
	shutdownErr := server.Shutdown(ctx)
	shutdownDuration := time.Since(shutdownStart)
	
	if shutdownErr != nil {
		t.Errorf("Shutdown failed: %v", shutdownErr)
	}
	
	if shutdownDuration > 3*time.Second {
		t.Errorf("Shutdown took too long: %v", shutdownDuration)
	}
	
	// Verify server is actually shut down
	_, err = net.Dial("tcp", addr)
	if err == nil {
		t.Error("Server should be shut down but still accepting connections")
	}
}

// BenchmarkServerThroughput benchmarks server throughput
func BenchmarkServerThroughput(b *testing.B) {
	server := NewServer(1000)
	
	go func() {
		server.Start(":0")
	}()
	
	time.Sleep(100 * time.Millisecond)
	addr := server.listener.Addr().String()
	
	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		conn, err := net.Dial("tcp", addr)
		if err != nil {
			b.Errorf("Failed to connect: %v", err)
			return
		}
		defer conn.Close()
		
		reader := bufio.NewReader(conn)
		
		for pb.Next() {
			_, err := fmt.Fprintf(conn, "benchmark message\n")
			if err != nil {
				b.Errorf("Write failed: %v", err)
				return
			}
			
			_, err = reader.ReadString('\n')
			if err != nil {
				b.Errorf("Read failed: %v", err)
				return
			}
		}
	})
	
	b.StopTimer()
	
	// Shutdown
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	server.Shutdown(ctx)
}

// BenchmarkServerMemoryUsage benchmarks memory usage under load
func BenchmarkServerMemoryUsage(b *testing.B) {
	server := NewServer(1000)
	
	go func() {
		server.Start(":0")
	}()
	
	time.Sleep(100 * time.Millisecond)
	addr := server.listener.Addr().String()
	
	// Force GC and get baseline
	runtime.GC()
	var m1 runtime.MemStats
	runtime.ReadMemStats(&m1)
	
	b.ResetTimer()
	
	// Create many concurrent connections
	numConnections := 1000
	var wg sync.WaitGroup
	
	for i := 0; i < numConnections; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			
			conn, err := net.Dial("tcp", addr)
			if err != nil {
				return
			}
			defer conn.Close()
			
			// Send a few messages
			reader := bufio.NewReader(conn)
			for j := 0; j < 10; j++ {
				fmt.Fprintf(conn, "message %d\n", j)
				reader.ReadString('\n')
			}
		}()
	}
	
	wg.Wait()
	
	// Force GC and measure memory
	runtime.GC()
	var m2 runtime.MemStats
	runtime.ReadMemStats(&m2)
	
	b.StopTimer()
	
	memoryUsed := m2.Alloc - m1.Alloc
	memoryPerConnection := float64(memoryUsed) / float64(numConnections)
	
	b.Logf("Memory used: %d bytes (%.2f KB per connection)", memoryUsed, memoryPerConnection/1024)
	
	// Target: <100MB for 10K connections = ~10KB per connection
	if memoryPerConnection > 10*1024 {
		b.Errorf("Memory usage too high: %.2f KB per connection (target: <10KB)", memoryPerConnection/1024)
	}
	
	// Shutdown
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	server.Shutdown(ctx)
}

// TestServerWorkerPoolExhaustion tests behavior when worker pool is exhausted
func TestServerWorkerPoolExhaustion(t *testing.T) {
	// Create server with very small worker pool
	server := NewServer(2)
	
	go func() {
		server.Start(":0")
	}()
	
	time.Sleep(100 * time.Millisecond)
	addr := server.listener.Addr().String()
	
	// Create connections that will exhaust the worker pool
	var connections []net.Conn
	defer func() {
		for _, conn := range connections {
			conn.Close()
		}
	}()
	
	// Create more connections than workers
	for i := 0; i < 5; i++ {
		conn, err := net.Dial("tcp", addr)
		if err != nil {
			t.Logf("Connection %d failed (expected for worker exhaustion): %v", i, err)
			continue
		}
		connections = append(connections, conn)
		
		// Don't send anything to keep workers busy
		time.Sleep(10 * time.Millisecond)
	}
	
	// At least some connections should succeed
	if len(connections) == 0 {
		t.Error("No connections succeeded")
	}
	
	// Shutdown
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	server.Shutdown(ctx)
}