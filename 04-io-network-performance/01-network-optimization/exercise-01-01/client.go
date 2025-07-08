package main

import (
	"bufio"
	"fmt"
	"net"
	"sync"
	"sync/atomic"
	"time"
)

// Client represents a TCP client for testing the server
type Client struct {
	addr           string
	connectionPool chan net.Conn
	maxConnections int
	
	// Metrics
	totalRequests   int64
	successRequests int64
	failedRequests  int64
	totalLatency    int64
	startTime       time.Time
}

// NewClient creates a new TCP client
func NewClient(addr string, maxConnections int) *Client {
	return &Client{
		addr:           addr,
		connectionPool: make(chan net.Conn, maxConnections),
		maxConnections: maxConnections,
		startTime:      time.Now(),
	}
}

// getConnection gets a connection from the pool or creates a new one
func (c *Client) getConnection() (net.Conn, error) {
	select {
	case conn := <-c.connectionPool:
		// Test if connection is still alive
		conn.SetReadDeadline(time.Now().Add(1 * time.Millisecond))
		buf := make([]byte, 1)
		_, err := conn.Read(buf)
		conn.SetReadDeadline(time.Time{}) // Clear deadline
		
		if err == nil {
			// Connection has data, put it back and create new one
			select {
			case c.connectionPool <- conn:
			default:
				conn.Close()
			}
			return net.Dial("tcp", c.addr)
		}
		
		if netErr, ok := err.(net.Error); ok && netErr.Timeout() {
			// Connection is alive (timeout on read)
			return conn, nil
		}
		
		// Connection is dead, close it and create new one
		conn.Close()
		return net.Dial("tcp", c.addr)
	default:
		// No connections in pool, create new one
		return net.Dial("tcp", c.addr)
	}
}

// putConnection returns a connection to the pool
func (c *Client) putConnection(conn net.Conn) {
	select {
	case c.connectionPool <- conn:
		// Successfully returned to pool
	default:
		// Pool is full, close connection
		conn.Close()
	}
}

// SendRequest sends a single request to the server
func (c *Client) SendRequest(message string) error {
	start := time.Now()
	defer func() {
		latency := time.Since(start).Nanoseconds()
		atomic.AddInt64(&c.totalLatency, latency)
		atomic.AddInt64(&c.totalRequests, 1)
	}()
	
	conn, err := c.getConnection()
	if err != nil {
		atomic.AddInt64(&c.failedRequests, 1)
		return fmt.Errorf("failed to get connection: %w", err)
	}
	
	// Set timeouts
	conn.SetWriteDeadline(time.Now().Add(5 * time.Second))
	conn.SetReadDeadline(time.Now().Add(5 * time.Second))
	
	// Send request
	_, err = fmt.Fprintf(conn, "%s\n", message)
	if err != nil {
		conn.Close()
		atomic.AddInt64(&c.failedRequests, 1)
		return fmt.Errorf("failed to send request: %w", err)
	}
	
	// Read response
	reader := bufio.NewReader(conn)
	_, err = reader.ReadBytes('\n')
	if err != nil {
		conn.Close()
		atomic.AddInt64(&c.failedRequests, 1)
		return fmt.Errorf("failed to read response: %w", err)
	}
	
	atomic.AddInt64(&c.successRequests, 1)
	
	// Return connection to pool
	c.putConnection(conn)
	
	return nil
}

// LoadTest performs a load test with specified parameters
func (c *Client) LoadTest(numClients, requestsPerClient int, message string) {
	var wg sync.WaitGroup
	startTime := time.Now()
	
	for i := 0; i < numClients; i++ {
		wg.Add(1)
		go func(clientID int) {
			defer wg.Done()
			
			for j := 0; j < requestsPerClient; j++ {
				msg := fmt.Sprintf("%s (client:%d, req:%d)", message, clientID, j)
				err := c.SendRequest(msg)
				if err != nil {
					// Don't log every error to avoid spam
					if j%1000 == 0 {
						fmt.Printf("Client %d request %d error: %v\n", clientID, j, err)
					}
				}
			}
		}(i)
	}
	
	wg.Wait()
	duration := time.Since(startTime)
	
	// Print results
	c.PrintStats(duration)
}

// PrintStats prints client statistics
func (c *Client) PrintStats(duration time.Duration) {
	totalReqs := atomic.LoadInt64(&c.totalRequests)
	successReqs := atomic.LoadInt64(&c.successRequests)
	failedReqs := atomic.LoadInt64(&c.failedRequests)
	totalLatency := atomic.LoadInt64(&c.totalLatency)
	
	var avgLatency float64
	if successReqs > 0 {
		avgLatency = float64(totalLatency) / float64(successReqs) / 1e6 // Convert to milliseconds
	}
	
	rps := float64(totalReqs) / duration.Seconds()
	successRate := float64(successReqs) / float64(totalReqs) * 100
	
	fmt.Printf("\n=== Load Test Results ===\n")
	fmt.Printf("Duration: %v\n", duration)
	fmt.Printf("Total Requests: %d\n", totalReqs)
	fmt.Printf("Successful Requests: %d\n", successReqs)
	fmt.Printf("Failed Requests: %d\n", failedReqs)
	fmt.Printf("Success Rate: %.2f%%\n", successRate)
	fmt.Printf("Requests/Second: %.2f\n", rps)
	fmt.Printf("Average Latency: %.2f ms\n", avgLatency)
	fmt.Printf("Connection Pool Size: %d\n", c.maxConnections)
}

// Close closes all connections in the pool
func (c *Client) Close() {
	for {
		select {
		case conn := <-c.connectionPool:
			conn.Close()
		default:
			return
		}
	}
}