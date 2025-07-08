package main

import (
	"context"
	"flag"
	"log"
	"os"
	"os/signal"
	"runtime"
	"syscall"
)

func main() {
	var (
		port = flag.String("port", "8080", "UDP port to listen on")
		workers = flag.Int("workers", runtime.NumCPU(), "Number of worker goroutines")
		bufferSize = flag.Int("buffer", 1024, "Buffer size for packets")
		poolSize = flag.Int("pool", 1000, "Buffer pool size")
	)
	flag.Parse()

	// Create UDP packet processor
	processor, err := NewPacketProcessor(*port, *workers, *bufferSize, *poolSize)
	if err != nil {
		log.Fatalf("Failed to create packet processor: %v", err)
	}

	// Setup graceful shutdown
	ctx, cancel := context.WithCancel(context.Background())
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	// Start the processor
	log.Printf("Starting UDP packet processor on port %s with %d workers", *port, *workers)
	go processor.Start(ctx)

	// Start metrics reporting
	go processor.ReportMetrics(ctx)

	// Wait for shutdown signal
	<-sigChan
	log.Println("Received shutdown signal, stopping processor...")
	cancel()

	// Wait for graceful shutdown
	processor.Stop()
	log.Println("UDP packet processor stopped")
}