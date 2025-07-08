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

// BenchmarkClient represents a UDP client for performance testing
type BenchmarkClient struct {
	serverAddr   *net.UDPAddr
	connections  []*net.UDPConn
	numConns     int
	packetSize   int
	targetPPS    int
	duration     time.Duration

	// Metrics
	packetsSent     uint64
	packetsFailed   uint64
	bytesSent       uint64
	startTime       time.Time
	endTime         time.Time
}

// NewBenchmarkClient creates a new benchmark client
func NewBenchmarkClient(serverAddr string, numConns, packetSize, targetPPS int, duration time.Duration) (*BenchmarkClient, error) {
	addr, err := net.ResolveUDPAddr("udp", serverAddr)
	if err != nil {
		return nil, fmt.Errorf("failed to resolve server address: %w", err)
	}

	client := &BenchmarkClient{
		serverAddr: addr,
		numConns:   numConns,
		packetSize: packetSize,
		targetPPS:  targetPPS,
		duration:   duration,
	}

	// Create connections
	client.connections = make([]*net.UDPConn, numConns)
	for i := 0; i < numConns; i++ {
		conn, err := net.DialUDP("udp", nil, addr)
		if err != nil {
			// Clean up already created connections
			for j := 0; j < i; j++ {
				client.connections[j].Close()
			}
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

// RunBenchmark executes the performance benchmark
func (c *BenchmarkClient) RunBenchmark() error {
	log.Printf("Starting benchmark: %d connections, %d bytes/packet, %d target PPS, %v duration",
		c.numConns, c.packetSize, c.targetPPS, c.duration)

	// Create packet data
	packetData := make([]byte, c.packetSize)
	for i := range packetData {
		packetData[i] = byte(i % 256)
	}

	// Calculate timing
	packetsPerConn := c.targetPPS / c.numConns
	if packetsPerConn == 0 {
		packetsPerConn = 1
	}
	connInterval := time.Second / time.Duration(packetsPerConn)

	ctx, cancel := context.WithTimeout(context.Background(), c.duration)
	defer cancel()

	c.startTime = time.Now()

	// Start sender goroutines
	var wg sync.WaitGroup
	for i, conn := range c.connections {
		wg.Add(1)
		go func(connID int, connection *net.UDPConn) {
			defer wg.Done()
			c.sender(ctx, connID, connection, packetData, connInterval)
		}(i, conn)
	}

	// Start metrics reporter
	wg.Add(1)
	go func() {
		defer wg.Done()
		c.reportMetrics(ctx)
	}()

	wg.Wait()
	c.endTime = time.Now()

	return nil
}

// sender sends packets at the specified rate
func (c *BenchmarkClient) sender(ctx context.Context, connID int, conn *net.UDPConn, data []byte, interval time.Duration) {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			_, err := conn.Write(data)
			if err != nil {
				atomic.AddUint64(&c.packetsFailed, 1)
			} else {
				atomic.AddUint64(&c.packetsSent, 1)
				atomic.AddUint64(&c.bytesSent, uint64(len(data)))
			}
		}
	}
}

// reportMetrics periodically reports benchmark metrics
func (c *BenchmarkClient) reportMetrics(ctx context.Context) {
	ticker := time.NewTicker(2 * time.Second)
	defer ticker.Stop()

	var lastSent uint64
	lastTime := time.Now()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			sent := atomic.LoadUint64(&c.packetsSent)
			failed := atomic.LoadUint64(&c.packetsFailed)
			bytes := atomic.LoadUint64(&c.bytesSent)

			now := time.Now()
			duration := now.Sub(lastTime).Seconds()
			rate := float64(sent-lastSent) / duration

			log.Printf("Benchmark progress: Sent=%d, Failed=%d, Rate=%.0f pps, Bytes=%d MB",
				sent, failed, rate, bytes/(1024*1024))

			lastSent = sent
			lastTime = now
		}
	}
}

// PrintResults prints the final benchmark results
func (c *BenchmarkClient) PrintResults() {
	sent := atomic.LoadUint64(&c.packetsSent)
	failed := atomic.LoadUint64(&c.packetsFailed)
	bytes := atomic.LoadUint64(&c.bytesSent)

	duration := c.endTime.Sub(c.startTime)
	rate := float64(sent) / duration.Seconds()
	throughput := float64(bytes) / duration.Seconds() / (1024 * 1024) // MB/s
	successRate := float64(sent) / float64(sent+failed) * 100

	fmt.Printf("\n=== Benchmark Results ===\n")
	fmt.Printf("Duration: %v\n", duration)
	fmt.Printf("Packets Sent: %d\n", sent)
	fmt.Printf("Packets Failed: %d\n", failed)
	fmt.Printf("Success Rate: %.2f%%\n", successRate)
	fmt.Printf("Packet Rate: %.0f pps\n", rate)
	fmt.Printf("Throughput: %.2f MB/s\n", throughput)
	fmt.Printf("Total Bytes: %d\n", bytes)

	// Performance evaluation
	c.evaluatePerformance(rate, successRate)
}

// evaluatePerformance evaluates if performance targets are met
func (c *BenchmarkClient) evaluatePerformance(actualPPS, successRate float64) {
	fmt.Printf("\n=== Performance Evaluation ===\n")

	// Target: 1M+ packets per second
	targetPPS := 1000000.0
	if actualPPS >= targetPPS {
		fmt.Printf("✅ Packet Rate: %.0f pps (target: >%.0f pps)\n", actualPPS, targetPPS)
	} else {
		fmt.Printf("❌ Packet Rate: %.0f pps (target: >%.0f pps)\n", actualPPS, targetPPS)
	}

	// Target: Success rate >99.99%
	targetSuccessRate := 99.99
	if successRate >= targetSuccessRate {
		fmt.Printf("✅ Success Rate: %.2f%% (target: >%.2f%%)\n", successRate, targetSuccessRate)
	} else {
		fmt.Printf("❌ Success Rate: %.2f%% (target: >%.2f%%)\n", successRate, targetSuccessRate)
	}

	// Target: Packet loss <0.01%
	lossRate := 100.0 - successRate
	targetLossRate := 0.01
	if lossRate <= targetLossRate {
		fmt.Printf("✅ Packet Loss: %.4f%% (target: <%.2f%%)\n", lossRate, targetLossRate)
	} else {
		fmt.Printf("❌ Packet Loss: %.4f%% (target: <%.2f%%)\n", lossRate, targetLossRate)
	}

	// CPU utilization check
	var m runtime.MemStats
	runtime.ReadMemStats(&m)
	fmt.Printf("Memory Usage: %d KB\n", m.Alloc/1024)
	fmt.Printf("GC Cycles: %d\n", m.NumGC)

	// Overall assessment
	if actualPPS >= targetPPS && successRate >= targetSuccessRate && lossRate <= targetLossRate {
		fmt.Printf("\n🎉 BENCHMARK PASSED: All performance targets met!\n")
	} else {
		fmt.Printf("\n⚠️  BENCHMARK PARTIAL: Some targets not met. Consider optimization.\n")
	}

	// Recommendations
	fmt.Printf("\n=== Recommendations ===\n")
	if actualPPS < targetPPS {
		fmt.Printf("- Increase worker pool size\n")
		fmt.Printf("- Optimize packet processing logic\n")
		fmt.Printf("- Consider lock-free data structures\n")
	}
	if successRate < targetSuccessRate {
		fmt.Printf("- Increase buffer sizes\n")
		fmt.Printf("- Optimize worker queue management\n")
		fmt.Printf("- Check for resource contention\n")
	}
	if lossRate > targetLossRate {
		fmt.Printf("- Increase worker count\n")
		fmt.Printf("- Optimize buffer pool management\n")
		fmt.Printf("- Consider batch processing\n")
	}
}

func main() {
	var (
		serverAddr = flag.String("addr", "localhost:8080", "Server address")
		duration   = flag.Duration("duration", 10*time.Second, "Benchmark duration")
		targetPPS  = flag.Int("pps", 100000, "Target packets per second")
		numConns   = flag.Int("conns", 10, "Number of connections")
		packetSize = flag.Int("size", 64, "Packet size in bytes")
	)
	flag.Parse()

	log.Printf("UDP Benchmark Client")
	log.Printf("Server: %s", *serverAddr)
	log.Printf("Duration: %v", *duration)
	log.Printf("Target PPS: %d", *targetPPS)
	log.Printf("Connections: %d", *numConns)
	log.Printf("Packet Size: %d bytes", *packetSize)

	// Create benchmark client
	client, err := NewBenchmarkClient(*serverAddr, *numConns, *packetSize, *targetPPS, *duration)
	if err != nil {
		log.Fatalf("Failed to create benchmark client: %v", err)
	}
	defer client.Close()

	// Run benchmark
	if err := client.RunBenchmark(); err != nil {
		log.Fatalf("Benchmark failed: %v", err)
	}

	// Print results
	client.PrintResults()
}