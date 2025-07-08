package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"net"
	"runtime"
	"sync"
	"sync/atomic"
	"time"
)

// BenchmarkClient implements a high-performance UDP client for testing
type BenchmarkClient struct {
	serverAddr *net.UDPAddr
	connections []*net.UDPConn
	numConnections int
	packetSize int

	// Metrics
	packetsSent uint64
	packetsReceived uint64
	errorCount uint64
	bytesSent uint64
	bytesReceived uint64
}

// NewBenchmarkClient creates a new benchmark client
func NewBenchmarkClient(serverAddr string, numConnections, packetSize int) (*BenchmarkClient, error) {
	addr, err := net.ResolveUDPAddr("udp", serverAddr)
	if err != nil {
		return nil, fmt.Errorf("failed to resolve server address: %w", err)
	}

	client := &BenchmarkClient{
		serverAddr: addr,
		numConnections: numConnections,
		packetSize: packetSize,
		connections: make([]*net.UDPConn, numConnections),
	}

	// Create connections
	for i := 0; i < numConnections; i++ {
		conn, err := net.DialUDP("udp", nil, addr)
		if err != nil {
			return nil, fmt.Errorf("failed to create connection %d: %w", i, err)
		}
		client.connections[i] = conn
	}

	return client, nil
}

// Close closes all connections
func (c *BenchmarkClient) Close() {
	for _, conn := range c.connections {
		if conn != nil {
			conn.Close()
		}
	}
}

// RunBenchmark executes the benchmark test
func (c *BenchmarkClient) RunBenchmark(duration time.Duration, targetPPS int) {
	ctx, cancel := context.WithTimeout(context.Background(), duration)
	defer cancel()

	// Create test packet
	testData := make([]byte, c.packetSize)
	for i := range testData {
		testData[i] = byte(i % 256)
	}

	// Calculate packets per connection per second
	ppsPerConn := targetPPS / c.numConnections
	if ppsPerConn == 0 {
		ppsPerConn = 1
	}

	// Start sender goroutines
	var wg sync.WaitGroup
	for i := 0; i < c.numConnections; i++ {
		wg.Add(1)
		go c.sender(ctx, &wg, i, testData, ppsPerConn)
	}

	// Start metrics reporter
	go c.reportMetrics(ctx)

	// Wait for completion
	wg.Wait()

	// Final report
	c.printFinalReport(duration)
}

// sender sends packets at the specified rate
func (c *BenchmarkClient) sender(ctx context.Context, wg *sync.WaitGroup, connID int, data []byte, pps int) {
	defer wg.Done()

	conn := c.connections[connID]
	if conn == nil {
		return
	}

	// Calculate interval between packets
	interval := time.Second / time.Duration(pps)
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			// Send packet
			n, err := conn.Write(data)
			if err != nil {
				atomic.AddUint64(&c.errorCount, 1)
				continue
			}
			atomic.AddUint64(&c.packetsSent, 1)
			atomic.AddUint64(&c.bytesSent, uint64(n))

			// Try to read response (non-blocking)
			conn.SetReadDeadline(time.Now().Add(1 * time.Millisecond))
			buf := make([]byte, c.packetSize)
			n, err = conn.Read(buf)
			if err == nil {
				atomic.AddUint64(&c.packetsReceived, 1)
				atomic.AddUint64(&c.bytesReceived, uint64(n))
			}
		}
	}
}

// reportMetrics periodically reports benchmark metrics
func (c *BenchmarkClient) reportMetrics(ctx context.Context) {
	ticker := time.NewTicker(2 * time.Second)
	defer ticker.Stop()

	var lastSent, lastReceived uint64
	lastTime := time.Now()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			now := time.Now()
			duration := now.Sub(lastTime).Seconds()
			lastTime = now

			sent := atomic.LoadUint64(&c.packetsSent)
			received := atomic.LoadUint64(&c.packetsReceived)
			errors := atomic.LoadUint64(&c.errorCount)

			sentRate := float64(sent-lastSent) / duration
			receivedRate := float64(received-lastReceived) / duration
			errorRate := float64(errors) / float64(sent) * 100

			lastSent = sent
			lastReceived = received

			log.Printf("Benchmark: Sent=%.0f pps, Received=%.0f pps, Errors=%.2f%%, Total Sent=%d",
				sentRate, receivedRate, errorRate, sent)
		}
	}
}

// printFinalReport prints the final benchmark results
func (c *BenchmarkClient) printFinalReport(duration time.Duration) {
	sent := atomic.LoadUint64(&c.packetsSent)
	received := atomic.LoadUint64(&c.packetsReceived)
	errors := atomic.LoadUint64(&c.errorCount)
	bytesSent := atomic.LoadUint64(&c.bytesSent)
	bytesReceived := atomic.LoadUint64(&c.bytesReceived)

	durationSec := duration.Seconds()
	sentPPS := float64(sent) / durationSec
	receivedPPS := float64(received) / durationSec
	errorRate := float64(errors) / float64(sent) * 100
	lossRate := float64(sent-received) / float64(sent) * 100
	throughputMBps := float64(bytesSent) / durationSec / 1024 / 1024

	fmt.Println("\n=== BENCHMARK RESULTS ===")
	fmt.Printf("Duration: %.2f seconds\n", durationSec)
	fmt.Printf("Packets Sent: %d (%.0f pps)\n", sent, sentPPS)
	fmt.Printf("Packets Received: %d (%.0f pps)\n", received, receivedPPS)
	fmt.Printf("Errors: %d (%.2f%%)\n", errors, errorRate)
	fmt.Printf("Packet Loss: %.2f%%\n", lossRate)
	fmt.Printf("Throughput: %.2f MB/s\n", throughputMBps)
	fmt.Printf("Bytes Sent: %d\n", bytesSent)
	fmt.Printf("Bytes Received: %d\n", bytesReceived)

	// Performance evaluation
	fmt.Println("\n=== PERFORMANCE EVALUATION ===")
	if sentPPS >= 1000000 {
		fmt.Printf("✅ Throughput Target: PASSED (%.0f pps >= 1M pps)\n", sentPPS)
	} else {
		fmt.Printf("❌ Throughput Target: FAILED (%.0f pps < 1M pps)\n", sentPPS)
	}

	if lossRate <= 0.01 {
		fmt.Printf("✅ Packet Loss Target: PASSED (%.4f%% <= 0.01%%)\n", lossRate)
	} else {
		fmt.Printf("❌ Packet Loss Target: FAILED (%.4f%% > 0.01%%)\n", lossRate)
	}

	if errorRate <= 1.0 {
		fmt.Printf("✅ Error Rate: PASSED (%.2f%% <= 1.0%%)\n", errorRate)
	} else {
		fmt.Printf("❌ Error Rate: FAILED (%.2f%% > 1.0%%)\n", errorRate)
	}
}

// runBenchmarkCommand runs the benchmark as a standalone command
func runBenchmarkCommand() {
	var (
		serverAddr = flag.String("server", "localhost:8080", "Server address")
		duration = flag.Duration("duration", 30*time.Second, "Benchmark duration")
		targetPPS = flag.Int("pps", 100000, "Target packets per second")
		connections = flag.Int("connections", runtime.NumCPU(), "Number of connections")
		packetSize = flag.Int("size", 64, "Packet size in bytes")
	)
	flag.Parse()

	log.Printf("Starting UDP benchmark: server=%s, duration=%v, target=%d pps, connections=%d, size=%d bytes",
		*serverAddr, *duration, *targetPPS, *connections, *packetSize)

	client, err := NewBenchmarkClient(*serverAddr, *connections, *packetSize)
	if err != nil {
		log.Fatalf("Failed to create benchmark client: %v", err)
	}
	defer client.Close()

	client.RunBenchmark(*duration, *targetPPS)
}