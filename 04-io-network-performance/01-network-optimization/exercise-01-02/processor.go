package main

import (
	"context"
	"fmt"
	"log"
	"net"
	"runtime"
	"sync"
	"sync/atomic"
	"time"
)

// Packet represents a UDP packet with zero-copy design
type Packet struct {
	Data []byte
	Addr *net.UDPAddr
	Size int
}

// PacketProcessor implements high-performance UDP packet processing
type PacketProcessor struct {
	conn       *net.UDPConn
	bufferPool sync.Pool
	workers    []chan *Packet
	numWorkers int
	bufferSize int

	// Metrics (atomic counters)
	packetsReceived  uint64
	packetsProcessed uint64
	packetsDropped   uint64
	bytesReceived    uint64

	// Shutdown coordination
	shutdown chan struct{}
	wg       sync.WaitGroup
}

// NewPacketProcessor creates a new UDP packet processor
func NewPacketProcessor(port string, numWorkers, bufferSize, poolSize int) (*PacketProcessor, error) {
	// Create UDP address
	addr, err := net.ResolveUDPAddr("udp", ":"+port)
	if err != nil {
		return nil, fmt.Errorf("failed to resolve UDP address: %w", err)
	}

	// Create UDP connection
	conn, err := net.ListenUDP("udp", addr)
	if err != nil {
		return nil, fmt.Errorf("failed to listen on UDP: %w", err)
	}

	// Set socket buffer sizes for high performance
	if err := conn.SetReadBuffer(4 * 1024 * 1024); err != nil {
		log.Printf("Warning: failed to set read buffer size: %v", err)
	}
	if err := conn.SetWriteBuffer(4 * 1024 * 1024); err != nil {
		log.Printf("Warning: failed to set write buffer size: %v", err)
	}

	processor := &PacketProcessor{
		conn:       conn,
		numWorkers: numWorkers,
		bufferSize: bufferSize,
		workers:    make([]chan *Packet, numWorkers),
		shutdown:   make(chan struct{}),
	}

	// Initialize buffer pool with zero-allocation design
	processor.bufferPool = sync.Pool{
		New: func() interface{} {
			return make([]byte, bufferSize)
		},
	}

	// Initialize worker channels
	for i := 0; i < numWorkers; i++ {
		processor.workers[i] = make(chan *Packet, 1000) // Buffered channels for better performance
	}

	return processor, nil
}

// Start begins packet processing
func (p *PacketProcessor) Start(ctx context.Context) {
	log.Printf("Starting packet processor with %d workers", p.numWorkers)

	// Start worker goroutines
	for i := 0; i < p.numWorkers; i++ {
		p.wg.Add(1)
		go p.worker(i, p.workers[i])
	}

	// Start packet receiver
	p.wg.Add(1)
	go p.receiver(ctx)

	log.Println("Packet processor started")
}

// Stop gracefully shuts down the processor
func (p *PacketProcessor) Stop() {
	log.Println("Stopping packet processor...")

	// Signal shutdown
	close(p.shutdown)

	// Close UDP connection
	if p.conn != nil {
		p.conn.Close()
	}

	// Close worker channels
	for _, worker := range p.workers {
		close(worker)
	}

	// Wait for all goroutines to finish
	p.wg.Wait()

	log.Println("Packet processor stopped")
}

// receiver handles incoming UDP packets with zero-copy design
func (p *PacketProcessor) receiver(ctx context.Context) {
	defer p.wg.Done()

	for {
		select {
		case <-p.shutdown:
			return
		case <-ctx.Done():
			return
		default:
			// Get buffer from pool
			buf := p.bufferPool.Get().([]byte)

			// Read packet with timeout
			p.conn.SetReadDeadline(time.Now().Add(100 * time.Millisecond))
			n, addr, err := p.conn.ReadFromUDP(buf)
			if err != nil {
				p.bufferPool.Put(buf)
				if netErr, ok := err.(net.Error); ok && netErr.Timeout() {
					continue // Timeout is expected
				}
				if netErr, ok := err.(net.Error); ok && netErr.Temporary() {
					continue // Temporary error, retry
				}
				return // Permanent error or shutdown
			}

			// Update metrics
			atomic.AddUint64(&p.packetsReceived, 1)
			atomic.AddUint64(&p.bytesReceived, uint64(n))

			// Create packet (zero-copy)
			packet := &Packet{
				Data: buf[:n],
				Addr: addr,
				Size: n,
			}

			// Distribute to worker based on address hash for load balancing
			workerID := p.hashAddr(addr) % uint32(p.numWorkers)
			select {
			case p.workers[workerID] <- packet:
				// Packet queued successfully
			default:
				// Worker queue is full, drop packet
				atomic.AddUint64(&p.packetsDropped, 1)
				p.bufferPool.Put(buf)
			}
		}
	}
}

// worker processes packets from its queue
func (p *PacketProcessor) worker(id int, packets <-chan *Packet) {
	defer p.wg.Done()

	for packet := range packets {
		// Process packet (simulate work)
		p.processPacket(packet)

		// Update metrics
		atomic.AddUint64(&p.packetsProcessed, 1)

		// Return buffer to pool
		p.bufferPool.Put(packet.Data[:cap(packet.Data)])
	}
}

// processPacket simulates packet processing work
func (p *PacketProcessor) processPacket(packet *Packet) {
	// Simulate minimal processing work
	// In a real application, this would contain actual packet processing logic
	_ = packet.Size
	_ = packet.Addr
	
	// Fast hash computation for demonstration
	_ = p.fastHash(packet.Data)
}

// hashAddr creates a hash from UDP address for worker distribution
func (p *PacketProcessor) hashAddr(addr *net.UDPAddr) uint32 {
	if addr == nil {
		return 0
	}
	
	// Simple hash based on IP and port
	ip := addr.IP
	port := uint32(addr.Port)
	
	var hash uint32
	for _, b := range ip {
		hash = hash*31 + uint32(b)
	}
	hash = hash*31 + port
	
	return hash
}

// fastHash computes a fast hash of the packet data
func (p *PacketProcessor) fastHash(data []byte) uint32 {
	if len(data) == 0 {
		return 0
	}
	
	// FNV-1a hash for speed
	const (
		offset32 = 2166136261
		prime32  = 16777619
	)
	
	hash := uint32(offset32)
	for _, b := range data {
		hash ^= uint32(b)
		hash *= prime32
	}
	
	return hash
}

// GetMetrics returns current processor metrics
func (p *PacketProcessor) GetMetrics() (received, processed, dropped, bytes uint64) {
	return atomic.LoadUint64(&p.packetsReceived),
		atomic.LoadUint64(&p.packetsProcessed),
		atomic.LoadUint64(&p.packetsDropped),
		atomic.LoadUint64(&p.bytesReceived)
}

// ReportMetrics periodically reports performance metrics
func (p *PacketProcessor) ReportMetrics(ctx context.Context) {
	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()

	var lastReceived, lastProcessed uint64
	lastTime := time.Now()

	for {
		select {
		case <-ctx.Done():
			return
		case <-p.shutdown:
			return
		case <-ticker.C:
			received, processed, dropped, bytes := p.GetMetrics()
			now := time.Now()
			duration := now.Sub(lastTime).Seconds()

			// Calculate rates
			receivedRate := float64(received-lastReceived) / duration
			processedRate := float64(processed-lastProcessed) / duration
			dropRate := float64(dropped) / float64(received) * 100

			// Memory stats
			var m runtime.MemStats
			runtime.ReadMemStats(&m)

			log.Printf("Metrics: Received=%.0f pps, Processed=%.0f pps, Dropped=%.2f%%, Bytes=%d, Alloc=%d KB, NumGC=%d",
				receivedRate, processedRate, dropRate, bytes, m.Alloc/1024, m.NumGC)

			lastReceived = received
			lastProcessed = processed
			lastTime = now
		}
	}
}