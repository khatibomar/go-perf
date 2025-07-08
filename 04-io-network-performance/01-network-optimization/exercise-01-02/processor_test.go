package main

import (
	"context"
	"net"
	"runtime"
	"sync"
	"testing"
	"time"
)

// TestPacketProcessorBasicFunctionality tests basic processor operations
func TestPacketProcessorBasicFunctionality(t *testing.T) {
	processor, err := NewPacketProcessor("0", 4, 1024, 100)
	if err != nil {
		t.Fatalf("Failed to create processor: %v", err)
	}
	defer processor.Stop()

	// Start processor
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	go processor.Start(ctx)
	time.Sleep(100 * time.Millisecond) // Let it start

	// Get the actual listening address
	addr := processor.conn.LocalAddr().(*net.UDPAddr)

	// Send test packets
	client, err := net.DialUDP("udp", nil, addr)
	if err != nil {
		t.Fatalf("Failed to create client: %v", err)
	}
	defer client.Close()

	// Send some packets
	for i := 0; i < 10; i++ {
		message := []byte("test packet")
		_, err := client.Write(message)
		if err != nil {
			t.Errorf("Failed to send packet %d: %v", i, err)
		}
	}

	// Wait for processing
	time.Sleep(200 * time.Millisecond)

	// Check metrics
	received, processed, dropped, _ := processor.GetMetrics()
	if received == 0 {
		t.Error("No packets received")
	}
	if processed == 0 {
		t.Error("No packets processed")
	}

	t.Logf("Received: %d, Processed: %d, Dropped: %d", received, processed, dropped)
}

// TestPacketProcessorConcurrentLoad tests concurrent packet processing
func TestPacketProcessorConcurrentLoad(t *testing.T) {
	processor, err := NewPacketProcessor("0", 8, 1024, 1000)
	if err != nil {
		t.Fatalf("Failed to create processor: %v", err)
	}
	defer processor.Stop()

	// Start processor
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	go processor.Start(ctx)
	time.Sleep(100 * time.Millisecond)

	addr := processor.conn.LocalAddr().(*net.UDPAddr)

	// Create multiple clients
	numClients := 5
	packetsPerClient := 1000
	var wg sync.WaitGroup

	for i := 0; i < numClients; i++ {
		wg.Add(1)
		go func(clientID int) {
			defer wg.Done()

			client, err := net.DialUDP("udp", nil, addr)
			if err != nil {
				t.Errorf("Client %d failed to connect: %v", clientID, err)
				return
			}
			defer client.Close()

			for j := 0; j < packetsPerClient; j++ {
				message := []byte("concurrent test packet")
				_, err := client.Write(message)
				if err != nil {
					t.Errorf("Client %d failed to send packet %d: %v", clientID, j, err)
					return
				}
				// Small delay to avoid overwhelming
				time.Sleep(10 * time.Microsecond)
			}
		}(i)
	}

	wg.Wait()

	// Wait for processing to complete
	time.Sleep(500 * time.Millisecond)

	// Check results
	received, processed, dropped, _ := processor.GetMetrics()
	totalSent := uint64(numClients * packetsPerClient)

	t.Logf("Sent: %d, Received: %d, Processed: %d, Dropped: %d", totalSent, received, processed, dropped)

	if received == 0 {
		t.Error("No packets received")
	}

	// Allow for some packet loss but not too much
	dropRate := float64(dropped) / float64(received) * 100
	if dropRate > 5.0 { // Allow up to 5% drop rate for test stability
		t.Errorf("Drop rate too high: %.2f%% (dropped: %d, received: %d)", dropRate, dropped, received)
	}
}

// TestPacketProcessorGracefulShutdown tests graceful shutdown
func TestPacketProcessorGracefulShutdown(t *testing.T) {
	processor, err := NewPacketProcessor("0", 4, 1024, 100)
	if err != nil {
		t.Fatalf("Failed to create processor: %v", err)
	}

	// Start processor
	ctx, cancel := context.WithCancel(context.Background())
	go processor.Start(ctx)
	time.Sleep(100 * time.Millisecond)

	// Send some packets
	addr := processor.conn.LocalAddr().(*net.UDPAddr)
	client, err := net.DialUDP("udp", nil, addr)
	if err != nil {
		t.Fatalf("Failed to create client: %v", err)
	}
	defer client.Close()

	for i := 0; i < 5; i++ {
		client.Write([]byte("shutdown test"))
	}

	// Shutdown
	shutdownStart := time.Now()
	cancel()
	processor.Stop()
	shutdownDuration := time.Since(shutdownStart)

	if shutdownDuration > 2*time.Second {
		t.Errorf("Shutdown took too long: %v", shutdownDuration)
	}

	// Verify metrics were collected
	received, _, _, _ := processor.GetMetrics()
	if received == 0 {
		t.Error("No packets received before shutdown")
	}
}

// TestPacketProcessorMetrics tests metrics collection
func TestPacketProcessorMetrics(t *testing.T) {
	processor, err := NewPacketProcessor("0", 4, 1024, 100)
	if err != nil {
		t.Fatalf("Failed to create processor: %v", err)
	}
	defer processor.Stop()

	// Start processor
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	go processor.Start(ctx)
	time.Sleep(100 * time.Millisecond)

	addr := processor.conn.LocalAddr().(*net.UDPAddr)
	client, err := net.DialUDP("udp", nil, addr)
	if err != nil {
		t.Fatalf("Failed to create client: %v", err)
	}
	defer client.Close()

	// Send known number of packets
	numPackets := 20
	for i := 0; i < numPackets; i++ {
		message := []byte("metrics test packet")
		_, err := client.Write(message)
		if err != nil {
			t.Errorf("Failed to send packet %d: %v", i, err)
		}
	}

	// Wait for processing
	time.Sleep(300 * time.Millisecond)

	// Check metrics
	received, processed, _, bytes := processor.GetMetrics()

	if received == 0 {
		t.Error("No packets received")
	}
	if processed == 0 {
		t.Error("No packets processed")
	}
	if bytes == 0 {
		t.Error("No bytes received")
	}

	t.Logf("Metrics - Received: %d, Processed: %d, Bytes: %d", received, processed, bytes)
}

// BenchmarkPacketProcessorThroughput benchmarks packet processing throughput
func BenchmarkPacketProcessorThroughput(b *testing.B) {
	processor, err := NewPacketProcessor("0", runtime.NumCPU(), 1024, 10000)
	if err != nil {
		b.Fatalf("Failed to create processor: %v", err)
	}
	defer processor.Stop()

	// Start processor
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	go processor.Start(ctx)
	time.Sleep(100 * time.Millisecond)

	addr := processor.conn.LocalAddr().(*net.UDPAddr)
	client, err := net.DialUDP("udp", nil, addr)
	if err != nil {
		b.Fatalf("Failed to create client: %v", err)
	}
	defer client.Close()

	message := []byte("benchmark packet data for throughput testing")

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			_, err := client.Write(message)
			if err != nil {
				b.Errorf("Failed to send packet: %v", err)
				return
			}
		}
	})
	b.StopTimer()

	// Wait for processing to complete
	time.Sleep(100 * time.Millisecond)

	// Report final metrics
	received, processed, dropped, _ := processor.GetMetrics()
	b.Logf("Final metrics - Received: %d, Processed: %d, Dropped: %d", received, processed, dropped)
}

// BenchmarkPacketProcessorMemoryAllocation benchmarks memory allocations
func BenchmarkPacketProcessorMemoryAllocation(b *testing.B) {
	processor, err := NewPacketProcessor("0", 4, 1024, 1000)
	if err != nil {
		b.Fatalf("Failed to create processor: %v", err)
	}
	defer processor.Stop()

	// Start processor
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	go processor.Start(ctx)
	time.Sleep(100 * time.Millisecond)

	addr := processor.conn.LocalAddr().(*net.UDPAddr)
	client, err := net.DialUDP("udp", nil, addr)
	if err != nil {
		b.Fatalf("Failed to create client: %v", err)
	}
	defer client.Close()

	message := []byte("memory allocation test packet")

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := client.Write(message)
		if err != nil {
			b.Errorf("Failed to send packet: %v", err)
			return
		}
	}
	b.StopTimer()

	// Wait for processing
	time.Sleep(100 * time.Millisecond)
}

// BenchmarkFastHash benchmarks the fast hash function
func BenchmarkFastHash(b *testing.B) {
	processor := &PacketProcessor{}
	data := make([]byte, 1024)
	for i := range data {
		data[i] = byte(i % 256)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = processor.fastHash(data)
	}
}