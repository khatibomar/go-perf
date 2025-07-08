package main

import (
	"context"
	"flag"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"
)

func main() {
	// Command line flags
	addr := flag.String("addr", ":8080", "Server address")
	workers := flag.Int("workers", 1000, "Number of worker goroutines")
	flag.Parse()
	
	log.Printf("Starting TCP server on %s with %d workers", *addr, *workers)
	
	// Create and start server
	server := NewServer(*workers)
	
	// Handle graceful shutdown
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
	
	go func() {
		if err := server.Start(*addr); err != nil {
			log.Printf("Server error: %v", err)
		}
	}()
	
	// Wait for shutdown signal
	<-sigChan
	log.Println("Received shutdown signal")
	
	// Graceful shutdown with timeout
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	
	if err := server.Shutdown(ctx); err != nil {
		log.Printf("Shutdown error: %v", err)
		os.Exit(1)
	}
	
	log.Println("Server shutdown complete")
}