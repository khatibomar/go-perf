package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"os"
	"os/signal"
	"runtime"
	"syscall"
	"time"
)

func main() {
	var (
		mode     = flag.String("mode", "server", "Mode: server or client")
		addr     = flag.String("addr", ":8080", "Server address")
		workers  = flag.Int("workers", runtime.NumCPU()*2, "Number of worker goroutines")
		conns    = flag.Int("conns", 10, "Number of client connections")
		messages = flag.Int("messages", 1000, "Number of messages to send (client mode)")
		timeout  = flag.Duration("timeout", 30*time.Second, "Connection timeout")
	)
	flag.Parse()

	switch *mode {
	case "server":
		runServer(*addr, *workers)
	case "client":
		runClient(*addr, *conns, *messages, *timeout)
	case "benchmark":
		runBenchmark(*addr, *conns, *messages, *timeout)
	default:
		log.Fatalf("Invalid mode: %s. Use 'server', 'client', or 'benchmark'", *mode)
	}
}

func runServer(addr string, workers int) {
	server := NewServer(workers)

	// Register message handlers
	server.RegisterHandler(0x10, MessageHandlerFunc(echoHandler))
	server.RegisterHandler(0x20, MessageHandlerFunc(reverseHandler))
	server.RegisterHandler(0x30, MessageHandlerFunc(uppercaseHandler))
	server.RegisterHandler(0x40, MessageHandlerFunc(statsHandler(server)))

	if err := server.Listen(addr); err != nil {
		log.Fatalf("Failed to listen: %v", err)
	}

	// Handle graceful shutdown
	ctx, cancel := context.WithCancel(context.Background())
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		<-sigChan
		log.Println("Shutting down server...")
		cancel()
		server.Shutdown()
	}()

	// Start stats reporting
	go func() {
		ticker := time.NewTicker(5 * time.Second)
		defer ticker.Stop()

		for {
			select {
			case <-ticker.C:
				stats := server.GetStats()
				log.Printf("Stats: Connections=%d, Messages=%d, BytesRx=%d, BytesTx=%d, Errors=%d",
					stats.Connections, stats.MessagesHandled, stats.BytesReceived, stats.BytesSent, stats.Errors)
			case <-ctx.Done():
				return
			}
		}
	}()

	log.Printf("Server starting on %s with %d workers", addr, workers)
	if err := server.Start(ctx); err != nil && err != context.Canceled {
		log.Fatalf("Server error: %v", err)
	}

	log.Println("Server stopped")
}

func runClient(addr string, conns, messages int, timeout time.Duration) {
	client := NewClient(conns, timeout)

	if err := client.Connect(addr); err != nil {
		log.Fatalf("Failed to connect: %v", err)
	}
	defer client.Close()

	log.Printf("Connected to %s with %d connections", addr, conns)

	// Test ping
	pingCtx, pingCancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer pingCancel()
	if err := client.Ping(pingCtx); err != nil {
		log.Printf("Ping failed: %v", err)
	} else {
		log.Println("Ping successful")
	}

	// Send test messages
	ctx := context.Background()
	for i := 0; i < messages; i++ {
		data := fmt.Sprintf("Hello, World! Message %d", i)
		metadata := map[string]string{
			"timestamp": time.Now().Format(time.RFC3339),
			"sequence":  fmt.Sprintf("%d", i),
		}

		resp, err := client.SendMessage(ctx, 0x10, []byte(data), metadata)
		if err != nil {
			log.Printf("Failed to send message %d: %v", i, err)
			continue
		}

		log.Printf("Response %d: %s", i, string(resp.Data))

		if i%100 == 0 {
			stats := client.GetStats()
			log.Printf("Client Stats: Sent=%d, Received=%d, BytesTx=%d, BytesRx=%d, Errors=%d",
				stats.MessagesSent, stats.MessagesReceived, stats.BytesSent, stats.BytesReceived, stats.Errors)
		}
	}

	finalStats := client.GetStats()
	log.Printf("Final Stats: Sent=%d, Received=%d, BytesTx=%d, BytesRx=%d, Errors=%d",
		finalStats.MessagesSent, finalStats.MessagesReceived, finalStats.BytesSent, finalStats.BytesReceived, finalStats.Errors)
}

func runBenchmark(addr string, conns, messages int, timeout time.Duration) {
	client := NewClient(conns, timeout)

	if err := client.Connect(addr); err != nil {
		log.Fatalf("Failed to connect: %v", err)
	}
	defer client.Close()

	log.Printf("Starting benchmark: %d connections, %d messages", conns, messages)

	start := time.Now()
	ctx := context.Background()

	// Send messages concurrently
	done := make(chan bool, conns)
	messagesPerConn := messages / conns

	for i := 0; i < conns; i++ {
		go func(connID int) {
			for j := 0; j < messagesPerConn; j++ {
				data := fmt.Sprintf("Benchmark message %d-%d", connID, j)
				_, err := client.SendMessage(ctx, 0x10, []byte(data), nil)
				if err != nil {
					log.Printf("Error sending message: %v", err)
				}
			}
			done <- true
		}(i)
	}

	// Wait for all goroutines to complete
	for i := 0; i < conns; i++ {
		<-done
	}

	duration := time.Since(start)
	stats := client.GetStats()

	log.Printf("Benchmark completed in %v", duration)
	log.Printf("Messages sent: %d", stats.MessagesSent)
	log.Printf("Messages received: %d", stats.MessagesReceived)
	log.Printf("Bytes sent: %d", stats.BytesSent)
	log.Printf("Bytes received: %d", stats.BytesReceived)
	log.Printf("Errors: %d", stats.Errors)
	log.Printf("Messages per second: %.2f", float64(stats.MessagesSent)/duration.Seconds())
	log.Printf("Throughput: %.2f MB/s", float64(stats.BytesSent+stats.BytesReceived)/duration.Seconds()/1024/1024)
}

// Message handlers

func echoHandler(ctx context.Context, msg *Message) (*Message, error) {
	return &Message{
		ID:       msg.ID,
		Type:     msg.Type,
		Data:     msg.Data,
		Metadata: msg.Metadata,
	}, nil
}

func reverseHandler(ctx context.Context, msg *Message) (*Message, error) {
	data := make([]byte, len(msg.Data))
	for i, b := range msg.Data {
		data[len(msg.Data)-1-i] = b
	}

	return &Message{
		ID:   msg.ID,
		Type: msg.Type,
		Data: data,
	}, nil
}

func uppercaseHandler(ctx context.Context, msg *Message) (*Message, error) {
	data := make([]byte, len(msg.Data))
	for i, b := range msg.Data {
		if b >= 'a' && b <= 'z' {
			data[i] = b - 32
		} else {
			data[i] = b
		}
	}

	return &Message{
		ID:   msg.ID,
		Type: msg.Type,
		Data: data,
	}, nil
}

func statsHandler(server *Server) func(ctx context.Context, msg *Message) (*Message, error) {
	return func(ctx context.Context, msg *Message) (*Message, error) {
		stats := server.GetStats()
		statsData := fmt.Sprintf("Connections=%d,Messages=%d,BytesRx=%d,BytesTx=%d,Errors=%d",
			stats.Connections, stats.MessagesHandled, stats.BytesReceived, stats.BytesSent, stats.Errors)

		return &Message{
			ID:   msg.ID,
			Type: msg.Type,
			Data: []byte(statsData),
		}, nil
	}
}