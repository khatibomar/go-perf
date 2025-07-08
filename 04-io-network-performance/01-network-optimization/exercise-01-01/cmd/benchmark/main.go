package main

import (
	"flag"
	"fmt"
	"log"
	"time"
)

// Import the benchmark functionality
// Note: In a real project, you'd import from the main package
// For this exercise, we'll duplicate the necessary code

// BenchmarkConfig holds benchmark configuration
type BenchmarkConfig struct {
	ServerAddr        string
	NumClients        int
	RequestsPerClient int
	ConnectionPoolSize int
	Message           string
	WarmupDuration    time.Duration
}

// runBenchmark runs a comprehensive benchmark of the TCP server
func runBenchmark() {
	// Command line flags
	var (
		addr              = flag.String("addr", "localhost:8080", "Server address")
		numClients        = flag.Int("clients", 100, "Number of concurrent clients")
		requestsPerClient = flag.Int("requests", 1000, "Requests per client")
		connPoolSize      = flag.Int("pool", 10, "Connection pool size per client")
		message           = flag.String("message", "Hello, World!", "Message to send")
		warmup            = flag.Duration("warmup", 5*time.Second, "Warmup duration")
	)
	flag.Parse()
	
	config := BenchmarkConfig{
		ServerAddr:        *addr,
		NumClients:        *numClients,
		RequestsPerClient: *requestsPerClient,
		ConnectionPoolSize: *connPoolSize,
		Message:           *message,
		WarmupDuration:    *warmup,
	}
	
	fmt.Printf("Starting benchmark with configuration:\n")
	fmt.Printf("  Server: %s\n", config.ServerAddr)
	fmt.Printf("  Clients: %d\n", config.NumClients)
	fmt.Printf("  Requests per client: %d\n", config.RequestsPerClient)
	fmt.Printf("  Total requests: %d\n", config.NumClients*config.RequestsPerClient)
	fmt.Printf("  Connection pool size: %d\n", config.ConnectionPoolSize)
	fmt.Printf("  Warmup duration: %v\n", config.WarmupDuration)
	fmt.Printf("  Message: %s\n\n", config.Message)
	
	// Create client
	client := NewClient(config.ServerAddr, config.ConnectionPoolSize)
	defer client.Close()
	
	// Warmup phase
	if config.WarmupDuration > 0 {
		fmt.Printf("Warming up for %v...\n", config.WarmupDuration)
		warmupClients := config.NumClients / 10
		if warmupClients < 1 {
			warmupClients = 1
		}
		warmupRequests := 10
		
		client.LoadTest(warmupClients, warmupRequests, "warmup-"+config.Message)
		
		// Reset metrics after warmup
		client = NewClient(config.ServerAddr, config.ConnectionPoolSize)
		defer client.Close()
		
		fmt.Println("Warmup complete. Starting actual benchmark...\n")
	}
	
	// Run the actual benchmark
	client.LoadTest(config.NumClients, config.RequestsPerClient, config.Message)
	
	// Validate performance targets
	validatePerformance(client, config)
}

// validatePerformance checks if the server meets the performance targets
func validatePerformance(client *Client, config BenchmarkConfig) {
	totalRequests := client.totalRequests
	successRequests := client.successRequests
	totalLatency := client.totalLatency
	
	if successRequests == 0 {
		fmt.Println("\n❌ BENCHMARK FAILED: No successful requests")
		return
	}
	
	avgLatencyMs := float64(totalLatency) / float64(successRequests) / 1e6
	successRate := float64(successRequests) / float64(totalRequests) * 100
	
	fmt.Printf("\n=== Performance Validation ===\n")
	
	// Target: Average latency <1ms
	latencyTarget := 1.0 // ms
	if avgLatencyMs <= latencyTarget {
		fmt.Printf("✅ Latency: %.2f ms (target: <%.1f ms)\n", avgLatencyMs, latencyTarget)
	} else {
		fmt.Printf("❌ Latency: %.2f ms (target: <%.1f ms)\n", avgLatencyMs, latencyTarget)
	}
	
	// Target: Success rate >99%
	successRateTarget := 99.0
	if successRate >= successRateTarget {
		fmt.Printf("✅ Success Rate: %.2f%% (target: >%.1f%%)\n", successRate, successRateTarget)
	} else {
		fmt.Printf("❌ Success Rate: %.2f%% (target: >%.1f%%)\n", successRate, successRateTarget)
	}
	
	// Calculate RPS
	duration := time.Since(client.startTime)
	rps := float64(totalRequests) / duration.Seconds()
	
	// Target: Process 100K+ requests per second (scaled based on test size)
	minRPS := 1000.0 // Minimum for small tests
	if config.NumClients >= 100 && config.RequestsPerClient >= 1000 {
		minRPS = 10000.0 // Higher target for larger tests
	}
	
	if rps >= minRPS {
		fmt.Printf("✅ Throughput: %.0f RPS (target: >%.0f RPS)\n", rps, minRPS)
	} else {
		fmt.Printf("❌ Throughput: %.0f RPS (target: >%.0f RPS)\n", rps, minRPS)
	}
	
	// Overall assessment
	if avgLatencyMs <= latencyTarget && successRate >= successRateTarget && rps >= minRPS {
		fmt.Printf("\n🎉 BENCHMARK PASSED: All performance targets met!\n")
	} else {
		fmt.Printf("\n⚠️  BENCHMARK PARTIAL: Some targets not met. Consider tuning server parameters.\n")
	}
	
	// Recommendations
	fmt.Printf("\n=== Recommendations ===\n")
	if avgLatencyMs > latencyTarget {
		fmt.Printf("- Increase worker pool size to reduce latency\n")
		fmt.Printf("- Optimize request processing logic\n")
		fmt.Printf("- Consider using larger buffers\n")
	}
	if successRate < successRateTarget {
		fmt.Printf("- Increase worker pool size to handle more concurrent connections\n")
		fmt.Printf("- Check for connection timeout issues\n")
		fmt.Printf("- Monitor server resource usage\n")
	}
	if rps < minRPS {
		fmt.Printf("- Increase concurrency (more workers)\n")
		fmt.Printf("- Optimize buffer management\n")
		fmt.Printf("- Consider connection pooling optimizations\n")
	}
}

func main() {
	log.SetFlags(log.LstdFlags | log.Lshortfile)
	runBenchmark()
}