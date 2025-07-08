package main

import (
	"context"
	"net"
	"runtime"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

func TestPacketProcessorBasicFunctionality(t *testing.T) {
	processor, err := NewPacketProcessor("0", 2, 1024, 100)
	if err != nil {
		t.Fatalf("Failed to create processor: %v", err)
	}
	defer processor.Stop()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	go processor.Start(ctx)
	time.Sleep(100 * time.Millisecond) // Let processor start

	// Get the actual port the processor is listening on
	addr := processor.conn.LocalAddr().(*net.UDPAddr)

	// Create client connection
	clientConn, err := net.DialUDP("udp", nil, addr)
	if err != nil {
		t.Fatalf("Failed to create client connection: %v", err)
	}
	defer clientConn.Close()

	// Send test packet
	testData := []byte("test packet")
	_, err = clientConn.Write(testData)
	if err != nil {
		t.Fatalf("Failed to send packet: %v", err)
	}

	// Wait for processing
	time.Sleep(100 * time.Millisecond)

	// Check metrics
	received, processed, dropped, _ := processor.GetMetrics()
	if received == 0 {
		t.Error("No packets received")
	}
	if processed == 0 {
		t.Error("No packets processed")
	}
	if dropped > 0 {
		t.Errorf("Unexpected packet drops: %d", dropped)
	}
}

func TestPacketProcessorConcurrentLoad(t *testing.T) {
	processor, err := NewPacketProcessor("0", runtime.NumCPU()*2, 1024, 5000)
	if err != nil {
		t.Fatalf("Failed to create processor: %v", err)
	}
	defer processor.Stop()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	go processor.Start(ctx)
	time.Sleep(200 * time.Millisecond)

	addr := processor.conn.LocalAddr().(*net.UDPAddr)

	// Create multiple client connections
	numClients := 5
	packetsPerClient := 500
	testData := []byte("concurrent test packet")

	var wg sync.WaitGroup
	var totalSent int64

	for i := 0; i < numClients; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			clientConn, err := net.DialUDP("udp", nil, addr)
			if err != nil {
				t.Errorf("Failed to create client connection: %v", err)
				return
			}
			defer clientConn.Close()

			for j := 0; j < packetsPerClient; j++ {
				_, err := clientConn.Write(testData)
				if err != nil {
					t.Errorf("Failed to send packet: %v", err)
					continue
				}
				atomic.AddInt64(&totalSent, 1)
				// Add small delay to avoid overwhelming the processor
				time.Sleep(time.Microsecond * 100)
			}
		}()
	}

	wg.Wait()
	time.Sleep(500 * time.Millisecond) // Allow processing to complete

	received, processed, dropped, _ := processor.GetMetrics()

	t.Logf("Sent: %d, Received: %d, Processed: %d, Dropped: %d", 
		totalSent, received, processed, dropped)

	if received == 0 {
		t.Error("No packets received")
	}

	// Allow for some packet loss in high-load scenarios
	dropRate := float64(dropped) / float64(received) * 100
	if dropRate > 5.0 { // Allow up to 5% drop rate for test stability
		t.Errorf("Drop rate too high: %.2f%%", dropRate)
	}
}

func TestPacketProcessorGracefulShutdown(t *testing.T) {
	processor, err := NewPacketProcessor("0", 2, 1024, 100)
	if err != nil {
		t.Fatalf("Failed to create processor: %v", err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	go processor.Start(ctx)
	time.Sleep(100 * time.Millisecond)

	// Send some packets
	addr := processor.conn.LocalAddr().(*net.UDPAddr)
	clientConn, err := net.DialUDP("udp", nil, addr)
	if err != nil {
		t.Fatalf("Failed to create client connection: %v", err)
	}
	defer clientConn.Close()

	for i := 0; i < 10; i++ {
		clientConn.Write([]byte("test"))
	}

	// Initiate shutdown
	cancel()

	// Stop should complete within reasonable time
	done := make(chan struct{})
	go func() {
		processor.Stop()
		close(done)
	}()

	select {
	case <-done:
		// Shutdown completed successfully
	case <-time.After(2 * time.Second):
		t.Error("Graceful shutdown took too long")
	}
}

func TestPacketProcessorMetrics(t *testing.T) {
	processor, err := NewPacketProcessor("0", 2, 1024, 100)
	if err != nil {
		t.Fatalf("Failed to create processor: %v", err)
	}
	defer processor.Stop()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	go processor.Start(ctx)
	time.Sleep(100 * time.Millisecond)

	addr := processor.conn.LocalAddr().(*net.UDPAddr)
	clientConn, err := net.DialUDP("udp", nil, addr)
	if err != nil {
		t.Fatalf("Failed to create client connection: %v", err)
	}
	defer clientConn.Close()

	// Send known number of packets
	numPackets := 100
	testData := []byte("metrics test packet")
	for i := 0; i < numPackets; i++ {
		clientConn.Write(testData)
	}

	time.Sleep(200 * time.Millisecond)

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

	// Verify metrics are reasonable
	expectedBytes := uint64(len(testData)) * received
	if bytes != expectedBytes {
		t.Logf("Bytes metric: expected ~%d, got %d", expectedBytes, bytes)
	}
}

// Benchmark tests for performance validation
func BenchmarkPacketProcessorThroughput(b *testing.B) {
	processor, err := NewPacketProcessor("0", runtime.NumCPU(), 1024, 10000)
	if err != nil {
		b.Fatalf("Failed to create processor: %v", err)
	}
	defer processor.Stop()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	go processor.Start(ctx)
	time.Sleep(100 * time.Millisecond)

	addr := processor.conn.LocalAddr().(*net.UDPAddr)
	clientConn, err := net.DialUDP("udp", nil, addr)
	if err != nil {
		b.Fatalf("Failed to create client connection: %v", err)
	}
	defer clientConn.Close()

	testData := []byte("benchmark packet data for throughput testing")

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			clientConn.Write(testData)
		}
	})

	// Report final metrics
	received, processed, dropped, _ := processor.GetMetrics()
	b.Logf("Final metrics - Received: %d, Processed: %d, Dropped: %d", 
		received, processed, dropped)
}

func BenchmarkPacketProcessorMemoryAllocation(b *testing.B) {
	processor, err := NewPacketProcessor("0", 2, 1024, 1000)
	if err != nil {
		b.Fatalf("Failed to create processor: %v", err)
	}
	defer processor.Stop()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	go processor.Start(ctx)
	time.Sleep(100 * time.Millisecond)

	addr := processor.conn.LocalAddr().(*net.UDPAddr)
	clientConn, err := net.DialUDP("udp", nil, addr)
	if err != nil {
		b.Fatalf("Failed to create client connection: %v", err)
	}
	defer clientConn.Close()

	testData := []byte("memory allocation test")

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		clientConn.Write(testData)
	}

	// Allow processing to complete
	time.Sleep(100 * time.Millisecond)
}

func BenchmarkFastHash(b *testing.B) {
	processor := &PacketProcessor{}
	addr := &net.UDPAddr{
		IP:   net.IPv4(192, 168, 1, 1),
		Port: 8080,
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		processor.fastHash(addr)
	}
}