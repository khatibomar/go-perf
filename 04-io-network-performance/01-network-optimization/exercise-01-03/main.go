package main

import (
	"context"
	"flag"
	"log"
	"os"
	"os/signal"
	"runtime"
	"syscall"
	"time"
)

func main() {
	// Command line flags
	var (
		port = flag.String("port", "8080", "Server port")
		maxConns = flag.Int("max-conns", 1000, "Maximum connections in pool")
		maxIdle = flag.Int("max-idle", 100, "Maximum idle connections")
		idleTimeout = flag.Duration("idle-timeout", 30*time.Second, "Idle connection timeout")
		healthInterval = flag.Duration("health-interval", 10*time.Second, "Health check interval")
		workers = flag.Int("workers", runtime.NumCPU(), "Number of worker goroutines")
	)
	flag.Parse()

	// Create connection pool manager
	poolManager := NewConnectionPoolManager(&PoolConfig{
		MaxConnections: *maxConns,
		MaxIdle:        *maxIdle,
		IdleTimeout:    *idleTimeout,
		HealthInterval: *healthInterval,
		Workers:        *workers,
	})

	// Create context for graceful shutdown
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Start the pool manager
	if err := poolManager.Start(ctx, ":"+*port); err != nil {
		log.Fatalf("Failed to start pool manager: %v", err)
	}

	log.Printf("Connection pool manager started on port %s", *port)
	log.Printf("Configuration: max_conns=%d, max_idle=%d, idle_timeout=%v, health_interval=%v, workers=%d",
		*maxConns, *maxIdle, *idleTimeout, *healthInterval, *workers)

	// Wait for interrupt signal
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
	<-sigChan

	log.Println("Shutting down connection pool manager...")
	cancel()

	// Wait for graceful shutdown
	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer shutdownCancel()

	if err := poolManager.Shutdown(shutdownCtx); err != nil {
		log.Printf("Error during shutdown: %v", err)
	} else {
		log.Println("Connection pool manager stopped gracefully")
	}

	// Print final metrics
	poolManager.PrintMetrics()
}