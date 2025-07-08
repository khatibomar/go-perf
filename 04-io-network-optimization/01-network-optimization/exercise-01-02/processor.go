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
	"unsafe"
)

// PacketProcessor implements high-performance UDP packet processing
type PacketProcessor struct {
	conn       *net.UDPConn
	bufferPool sync.Pool
	workers    []chan *Packet
	numWorkers int
	bufferSize int

	// Metrics (atomic counters)
	packetsReceived uint64
	packetsProcessed uint64
	packetsDropped  uint64
	bytesReceived   uint64

	// Shutdown coordination
	shutdown chan struct{}
	wg       sync.WaitGroup
}

// Packet represents a UDP packet with zero-copy design
type Packet struct {
	data []byte
	addr *net.UDPAddr
	size int
}

// NewPacketProcessor creates a new high-performance UDP packet processor
func NewPacketProcessor(port string, numWorkers, bufferSize, poolSize int) (*PacketProcessor, error) {
	addr, err := net.ResolveUDPAddr("udp", ":"+port)
	if err != nil {
		return nil, fmt.Errorf("failed to resolve UDP address: %w", err)
	}

	conn, err := net.ListenUDP("udp", addr)
	if err != nil {
		return nil, fmt.Errorf("failed to listen on UDP: %w", err)
	}

	// Set socket buffer sizes for high throughput
	if err := conn.SetReadBuffer(4 * 1024 * 1024); err != nil {
		log.Printf("Warning: failed to set read buffer: %v", err)
	}
	if err := conn.SetWriteBuffer(4 * 1024 * 1024); err != nil {
		log.Printf("Warning: failed to set write buffer: %v", err)
	}

	p := &PacketProcessor{
		conn:       conn,
		numWorkers: numWorkers,
		bufferSize: bufferSize,
		workers:    make([]chan *Packet, numWorkers),
		shutdown:   make(chan struct{}),
	}

	// Initialize buffer pool with pre-allocated buffers
	p.bufferPool = sync.Pool{
		New: func() interface{} {
			return make([]byte, bufferSize)
		},
	}

	// Pre-populate the pool
	for i := 0; i < poolSize; i++ {
		p.bufferPool.Put(make([]byte, bufferSize))
	}

	// Initialize worker channels with buffering for lock-free operation
	for i := 0; i < numWorkers; i++ {
		p.workers[i] = make(chan *Packet, 1000) // Buffered to reduce blocking
	}

	return p, nil
}

// Start begins packet processing
func (p *PacketProcessor) Start(ctx context.Context) {
	// Start worker goroutines
	for i := 0; i < p.numWorkers; i++ {
		p.wg.Add(1)
		go p.worker(i, ctx)
	}

	// Start packet receiver
	p.wg.Add(1)
	go p.receiver(ctx)

	// Wait for context cancellation
	<-ctx.Done()
	close(p.shutdown)
}

// Stop gracefully shuts down the processor
func (p *PacketProcessor) Stop() {
	p.conn.Close()
	p.wg.Wait()
}

// receiver handles incoming UDP packets with zero-copy design
func (p *PacketProcessor) receiver(ctx context.Context) {
	defer p.wg.Done()

	for {
		select {
		case <-p.shutdown:
			return
		default:
		}

		// Get buffer from pool (zero allocation in steady state)
		buf := p.bufferPool.Get().([]byte)

		// Read packet with timeout to allow shutdown
		p.conn.SetReadDeadline(time.Now().Add(100 * time.Millisecond))
		n, addr, err := p.conn.ReadFromUDP(buf)
		if err != nil {
			p.bufferPool.Put(buf)
			if netErr, ok := err.(net.Error); ok && netErr.Timeout() {
				continue // Timeout is expected for shutdown handling
			}
			if ctx.Err() != nil {
				return // Context cancelled
			}
			continue
		}

		// Update metrics
		atomic.AddUint64(&p.packetsReceived, 1)
		atomic.AddUint64(&p.bytesReceived, uint64(n))

		// Create packet struct (minimal allocation)
		packet := &Packet{
			data: buf,
			addr: addr,
			size: n,
		}

		// Distribute to worker using fast hash (lock-free)
		workerIdx := p.fastHash(addr) % uint32(p.numWorkers)
		select {
		case p.workers[workerIdx] <- packet:
			// Packet queued successfully
		default:
			// Worker queue is full, drop packet
			atomic.AddUint64(&p.packetsDropped, 1)
			p.bufferPool.Put(buf)
		}
	}
}

// worker processes packets from the queue
func (p *PacketProcessor) worker(id int, ctx context.Context) {
	defer p.wg.Done()

	for {
		select {
		case <-p.shutdown:
			// Drain remaining packets
			for {
				select {
				case packet := <-p.workers[id]:
					p.processPacket(packet)
				default:
					return
				}
			}
		case packet := <-p.workers[id]:
			p.processPacket(packet)
		}
	}
}

// processPacket handles individual packet processing
func (p *PacketProcessor) processPacket(packet *Packet) {
	defer p.bufferPool.Put(packet.data)

	// Simulate packet processing (echo back)
	// In a real implementation, this would be your business logic
	data := packet.data[:packet.size]
	
	// Echo the packet back (optional - remove for pure processing)
	_, err := p.conn.WriteToUDP(data, packet.addr)
	if err != nil {
		// Log error but don't block processing
		log.Printf("Failed to echo packet: %v", err)
	}

	atomic.AddUint64(&p.packetsProcessed, 1)
}

// fastHash implements a fast hash function for load balancing
func (p *PacketProcessor) fastHash(addr *net.UDPAddr) uint32 {
	// Simple but fast hash based on IP and port
	ip := addr.IP
	port := uint32(addr.Port)
	
	if len(ip) >= 4 {
		// IPv4 or IPv6 (use first 4 bytes)
		return *(*uint32)(unsafe.Pointer(&ip[0])) ^ port
	}
	return port
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
		case <-ticker.C:
			now := time.Now()
			duration := now.Sub(lastTime).Seconds()
			lastTime = now

			received := atomic.LoadUint64(&p.packetsReceived)
			processed := atomic.LoadUint64(&p.packetsProcessed)
			dropped := atomic.LoadUint64(&p.packetsDropped)
			bytes := atomic.LoadUint64(&p.bytesReceived)

			receivedRate := float64(received-lastReceived) / duration
			processedRate := float64(processed-lastProcessed) / duration
			dropRate := float64(dropped) / float64(received) * 100

			lastReceived = received
			lastProcessed = processed

			var m runtime.MemStats
			runtime.ReadMemStats(&m)

			log.Printf("Metrics: RX=%.0f pps, Processed=%.0f pps, Dropped=%.2f%%, Bytes=%d, Allocs=%d, Sys=%dMB",
				receivedRate, processedRate, dropRate, bytes, m.TotalAlloc, m.Sys/1024/1024)
		}
	}
}

// GetMetrics returns current performance metrics
func (p *PacketProcessor) GetMetrics() (received, processed, dropped, bytes uint64) {
	return atomic.LoadUint64(&p.packetsReceived),
		atomic.LoadUint64(&p.packetsProcessed),
		atomic.LoadUint64(&p.packetsDropped),
		atomic.LoadUint64(&p.bytesReceived)
}