package main

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"log"
	"net"
	"runtime"
	"sync"
	"sync/atomic"
	"time"
)

const (
	// Server configuration
	DefaultPort        = ":8080"
	DefaultWorkerCount = 1000
	DefaultBufferSize  = 4096
	ReadTimeout        = 30 * time.Second
	WriteTimeout       = 30 * time.Second
	IdleTimeout        = 60 * time.Second
)

// Server represents a high-performance TCP server
type Server struct {
	listener   net.Listener
	bufferPool *sync.Pool
	workerPool chan struct{}
	shutdown   chan struct{}
	wg         sync.WaitGroup
	
	// Metrics
	activeConnections int64
	totalConnections  int64
	totalRequests     int64
	totalBytes        int64
	startTime         time.Time
}

// NewServer creates a new high-performance TCP server
func NewServer(workerCount int) *Server {
	if workerCount <= 0 {
		workerCount = DefaultWorkerCount
	}
	
	return &Server{
		bufferPool: &sync.Pool{
			New: func() interface{} {
				return make([]byte, DefaultBufferSize)
			},
		},
		workerPool: make(chan struct{}, workerCount),
		shutdown:   make(chan struct{}),
		startTime:  time.Now(),
	}
}

// Start starts the TCP server on the specified address
func (s *Server) Start(addr string) error {
	if addr == "" {
		addr = DefaultPort
	}
	
	listener, err := net.Listen("tcp", addr)
	if err != nil {
		return fmt.Errorf("failed to start listener: %w", err)
	}
	
	s.listener = listener
	log.Printf("TCP server started on %s with %d workers", addr, cap(s.workerPool))
	
	// Fill worker pool
	for i := 0; i < cap(s.workerPool); i++ {
		s.workerPool <- struct{}{}
	}
	
	// Start metrics reporter
	go s.reportMetrics()
	
	// Accept connections
	for {
		select {
		case <-s.shutdown:
			return nil
		default:
		}
		
		conn, err := s.listener.Accept()
		if err != nil {
			select {
			case <-s.shutdown:
				return nil
			default:
				log.Printf("Accept error: %v", err)
				continue
			}
		}
		
		// Try to get a worker
		select {
		case <-s.workerPool:
			s.wg.Add(1)
			go s.handleConnection(conn)
		default:
			// No workers available, reject connection
			conn.Close()
			log.Printf("Connection rejected: no workers available")
		}
	}
}

// handleConnection handles a single client connection
func (s *Server) handleConnection(conn net.Conn) {
	defer s.wg.Done()
	defer conn.Close()
	defer func() { s.workerPool <- struct{}{} }() // Return worker to pool
	
	// Update connection metrics
	atomic.AddInt64(&s.activeConnections, 1)
	atomic.AddInt64(&s.totalConnections, 1)
	defer atomic.AddInt64(&s.activeConnections, -1)
	
	// Set connection timeouts
	if tcpConn, ok := conn.(*net.TCPConn); ok {
		tcpConn.SetKeepAlive(true)
		tcpConn.SetKeepAlivePeriod(30 * time.Second)
	}
	
	// Get buffer from pool
	buf := s.bufferPool.Get().([]byte)
	defer s.bufferPool.Put(buf)
	
	reader := bufio.NewReader(conn)
	writer := bufio.NewWriter(conn)
	
	// Handle requests in a loop
	for {
		select {
		case <-s.shutdown:
			return
		default:
		}
		
		// Set shorter read timeout during shutdown
		readTimeout := ReadTimeout
		select {
		case <-s.shutdown:
			readTimeout = 100 * time.Millisecond
		default:
		}
		conn.SetReadDeadline(time.Now().Add(readTimeout))
		
		// Read request
		line, err := reader.ReadBytes('\n')
		if err != nil {
			if err == io.EOF {
				return // Client closed connection
			}
			if netErr, ok := err.(net.Error); ok && netErr.Timeout() {
				// Check if shutdown was signaled during timeout
				select {
				case <-s.shutdown:
					return
				default:
					return // Regular timeout
				}
			}
			log.Printf("Read error: %v", err)
			return
		}
		
		// Update metrics
		atomic.AddInt64(&s.totalRequests, 1)
		atomic.AddInt64(&s.totalBytes, int64(len(line)))
		
		// Process request (echo server)
		response := s.processRequest(line)
		
		// Set write timeout
		conn.SetWriteDeadline(time.Now().Add(WriteTimeout))
		
		// Write response
		_, err = writer.Write(response)
		if err != nil {
			log.Printf("Write error: %v", err)
			return
		}
		
		err = writer.Flush()
		if err != nil {
			log.Printf("Flush error: %v", err)
			return
		}
		
		// Update bytes sent
		atomic.AddInt64(&s.totalBytes, int64(len(response)))
	}
}

// processRequest processes a client request (simple echo server)
func (s *Server) processRequest(request []byte) []byte {
	// Simple echo with timestamp
	timestamp := time.Now().Format(time.RFC3339Nano)
	response := fmt.Sprintf("[%s] Echo: %s", timestamp, string(request))
	return []byte(response)
}

// Shutdown gracefully shuts down the server
func (s *Server) Shutdown(ctx context.Context) error {
	log.Println("Shutting down server...")
	
	// Signal shutdown
	close(s.shutdown)
	
	// Close listener
	if s.listener != nil {
		s.listener.Close()
	}
	
	// Wait for all connections to finish with timeout
	done := make(chan struct{})
	go func() {
		s.wg.Wait()
		close(done)
	}()
	
	// Give connections a short time to close naturally
	shortTimeout := time.NewTimer(100 * time.Millisecond)
	select {
	case <-done:
		shortTimeout.Stop()
		log.Println("All connections closed gracefully")
		return nil
	case <-shortTimeout.C:
		// Continue to full timeout
	}
	
	select {
	case <-done:
		log.Println("All connections closed gracefully")
	case <-ctx.Done():
		log.Println("Shutdown timeout reached")
		return ctx.Err()
	}
	
	return nil
}

// GetMetrics returns current server metrics
func (s *Server) GetMetrics() map[string]interface{} {
	uptime := time.Since(s.startTime)
	activeConns := atomic.LoadInt64(&s.activeConnections)
	totalConns := atomic.LoadInt64(&s.totalConnections)
	totalReqs := atomic.LoadInt64(&s.totalRequests)
	totalBytes := atomic.LoadInt64(&s.totalBytes)
	
	reqsPerSec := float64(totalReqs) / uptime.Seconds()
	bytesPerSec := float64(totalBytes) / uptime.Seconds()
	
	return map[string]interface{}{
		"uptime_seconds":        uptime.Seconds(),
		"active_connections":    activeConns,
		"total_connections":     totalConns,
		"total_requests":        totalReqs,
		"total_bytes":           totalBytes,
		"requests_per_second":   reqsPerSec,
		"bytes_per_second":      bytesPerSec,
		"worker_pool_size":      cap(s.workerPool),
		"goroutines":            runtime.NumGoroutine(),
	}
}

// reportMetrics periodically reports server metrics
func (s *Server) reportMetrics() {
	ticker := time.NewTicker(10 * time.Second)
	defer ticker.Stop()
	
	for {
		select {
		case <-s.shutdown:
			return
		case <-ticker.C:
			metrics := s.GetMetrics()
			log.Printf("Metrics: Active=%d, Total=%d, RPS=%.0f, Goroutines=%d",
				metrics["active_connections"],
				metrics["total_connections"],
				metrics["requests_per_second"],
				metrics["goroutines"])
		}
	}
}