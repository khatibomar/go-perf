package main

import (
	"context"
	"errors"
	"net"
	"sync"
	"sync/atomic"
	"time"
)

// Errors
var (
	ErrPoolClosed     = errors.New("connection pool is closed")
	ErrPoolTimeout    = errors.New("connection pool timeout")
	ErrPoolExhausted  = errors.New("connection pool exhausted")
	ErrConnUnhealthy  = errors.New("connection is unhealthy")
)

// PoolConfig holds configuration for the connection pool
type PoolConfig struct {
	MaxConnections int           // Maximum total connections
	MaxIdle        int           // Maximum idle connections
	IdleTimeout    time.Duration // Idle connection timeout
	HealthInterval time.Duration // Health check interval
	Workers        int           // Number of worker goroutines
}

// ConnectionWrapper wraps a connection with metadata
type ConnectionWrapper struct {
	conn        net.Conn
	lastUsed    time.Time
	createdAt   time.Time
	healthy     bool
	inUse       bool
	useCount    uint64
	mutex       sync.RWMutex
}

// NewConnectionWrapper creates a new connection wrapper
func NewConnectionWrapper(conn net.Conn) *ConnectionWrapper {
	now := time.Now()
	return &ConnectionWrapper{
		conn:      conn,
		lastUsed:  now,
		createdAt: now,
		healthy:   true,
		inUse:     false,
		useCount:  0,
	}
}

// IsHealthy checks if the connection is healthy
func (cw *ConnectionWrapper) IsHealthy() bool {
	cw.mutex.RLock()
	defer cw.mutex.RUnlock()
	return cw.healthy
}

// MarkUnhealthy marks the connection as unhealthy
func (cw *ConnectionWrapper) MarkUnhealthy() {
	cw.mutex.Lock()
	defer cw.mutex.Unlock()
	cw.healthy = false
}

// Use marks the connection as in use
func (cw *ConnectionWrapper) Use() {
	cw.mutex.Lock()
	defer cw.mutex.Unlock()
	cw.inUse = true
	cw.lastUsed = time.Now()
	cw.useCount++
}

// Release marks the connection as not in use
func (cw *ConnectionWrapper) Release() {
	cw.mutex.Lock()
	defer cw.mutex.Unlock()
	cw.inUse = false
	cw.lastUsed = time.Now()
}

// IsIdle checks if the connection is idle for too long
func (cw *ConnectionWrapper) IsIdle(timeout time.Duration) bool {
	cw.mutex.RLock()
	defer cw.mutex.RUnlock()
	return !cw.inUse && time.Since(cw.lastUsed) > timeout
}

// Close closes the underlying connection
func (cw *ConnectionWrapper) Close() error {
	cw.mutex.Lock()
	defer cw.mutex.Unlock()
	return cw.conn.Close()
}

// ConnectionPool manages a pool of connections
type ConnectionPool struct {
	config       *PoolConfig
	connections  []*ConnectionWrapper
	idle         chan *ConnectionWrapper
	mutex        sync.RWMutex
	closed       bool
	shutdownChan chan struct{}
	wg           sync.WaitGroup

	// Metrics
	totalCreated   uint64
	totalDestroyed uint64
	totalRequests  uint64
	totalHits      uint64
	totalMisses    uint64
	healthChecks   uint64
	unhealthyConns uint64
}

// NewConnectionPool creates a new connection pool
func NewConnectionPool(config *PoolConfig) *ConnectionPool {
	return &ConnectionPool{
		config:       config,
		connections:  make([]*ConnectionWrapper, 0, config.MaxConnections),
		idle:         make(chan *ConnectionWrapper, config.MaxIdle),
		shutdownChan: make(chan struct{}),
	}
}

// Start starts the connection pool background tasks
func (p *ConnectionPool) Start(ctx context.Context) {
	p.wg.Add(2)
	go p.healthChecker(ctx)
	go p.idleConnectionCleaner(ctx)
}

// Get retrieves a connection from the pool
func (p *ConnectionPool) Get(ctx context.Context, target string) (*ConnectionWrapper, error) {
	p.mutex.RLock()
	if p.closed {
		p.mutex.RUnlock()
		return nil, ErrPoolClosed
	}
	p.mutex.RUnlock()

	atomic.AddUint64(&p.totalRequests, 1)

	// Try to get an idle connection first
	select {
	case conn := <-p.idle:
		if conn.IsHealthy() {
			conn.Use()
			atomic.AddUint64(&p.totalHits, 1)
			return conn, nil
		}
		// Connection is unhealthy, destroy it
		p.destroyConnection(conn)
	default:
		// No idle connections available
	}

	// Try to create a new connection
	if p.canCreateConnection() {
		conn, err := p.createConnection(target)
		if err != nil {
			atomic.AddUint64(&p.totalMisses, 1)
			return nil, err
		}
		conn.Use()
		atomic.AddUint64(&p.totalMisses, 1)
		return conn, nil
	}

	// Pool is exhausted, wait for a connection with timeout
	select {
	case conn := <-p.idle:
		if conn.IsHealthy() {
			conn.Use()
			atomic.AddUint64(&p.totalHits, 1)
			return conn, nil
		}
		p.destroyConnection(conn)
		return nil, ErrConnUnhealthy
	case <-ctx.Done():
		atomic.AddUint64(&p.totalMisses, 1)
		return nil, ErrPoolTimeout
	}
}

// Put returns a connection to the pool
func (p *ConnectionPool) Put(conn *ConnectionWrapper) {
	p.mutex.RLock()
	if p.closed {
		p.mutex.RUnlock()
		p.destroyConnection(conn)
		return
	}
	p.mutex.RUnlock()

	conn.Release()

	// Try to put back in idle pool
	select {
	case p.idle <- conn:
		// Successfully returned to pool
	default:
		// Idle pool is full, destroy the connection
		p.destroyConnection(conn)
	}
}

// canCreateConnection checks if we can create a new connection
func (p *ConnectionPool) canCreateConnection() bool {
	p.mutex.RLock()
	defer p.mutex.RUnlock()
	return len(p.connections) < p.config.MaxConnections
}

// createConnection creates a new connection
func (p *ConnectionPool) createConnection(target string) (*ConnectionWrapper, error) {
	conn, err := net.DialTimeout("tcp", target, 5*time.Second)
	if err != nil {
		return nil, err
	}

	// Set TCP keep-alive
	if tcpConn, ok := conn.(*net.TCPConn); ok {
		tcpConn.SetKeepAlive(true)
		tcpConn.SetKeepAlivePeriod(30 * time.Second)
	}

	wrapper := NewConnectionWrapper(conn)

	p.mutex.Lock()
	p.connections = append(p.connections, wrapper)
	p.mutex.Unlock()

	atomic.AddUint64(&p.totalCreated, 1)
	return wrapper, nil
}

// destroyConnection removes and closes a connection
func (p *ConnectionPool) destroyConnection(conn *ConnectionWrapper) {
	conn.Close()

	p.mutex.Lock()
	for i, c := range p.connections {
		if c == conn {
			p.connections = append(p.connections[:i], p.connections[i+1:]...)
			break
		}
	}
	p.mutex.Unlock()

	atomic.AddUint64(&p.totalDestroyed, 1)
}

// healthChecker periodically checks connection health
func (p *ConnectionPool) healthChecker(ctx context.Context) {
	defer p.wg.Done()
	ticker := time.NewTicker(p.config.HealthInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-p.shutdownChan:
			return
		case <-ticker.C:
			p.performHealthCheck()
		}
	}
}

// performHealthCheck checks all connections for health
func (p *ConnectionPool) performHealthCheck() {
	p.mutex.RLock()
	connections := make([]*ConnectionWrapper, len(p.connections))
	copy(connections, p.connections)
	p.mutex.RUnlock()

	for _, conn := range connections {
		atomic.AddUint64(&p.healthChecks, 1)
		if !p.isConnectionHealthy(conn) {
			conn.MarkUnhealthy()
			atomic.AddUint64(&p.unhealthyConns, 1)
			// Remove from idle pool if present
			select {
			case <-p.idle:
				p.destroyConnection(conn)
			default:
				// Not in idle pool, will be destroyed when returned
			}
		}
	}
}

// isConnectionHealthy performs a health check on a connection
func (p *ConnectionPool) isConnectionHealthy(conn *ConnectionWrapper) bool {
	conn.mutex.RLock()
	if conn.inUse {
		conn.mutex.RUnlock()
		return true // Assume in-use connections are healthy
	}
	tcpConn := conn.conn
	conn.mutex.RUnlock()

	// Set a short deadline for health check
	tcpConn.SetReadDeadline(time.Now().Add(100 * time.Millisecond))
	defer tcpConn.SetReadDeadline(time.Time{})

	// Try to read one byte (should timeout immediately for healthy connections)
	buf := make([]byte, 1)
	_, err := tcpConn.Read(buf)

	// If we get a timeout, the connection is healthy
	if netErr, ok := err.(net.Error); ok && netErr.Timeout() {
		return true
	}

	// Any other error means the connection is unhealthy
	return err == nil
}

// idleConnectionCleaner removes idle connections
func (p *ConnectionPool) idleConnectionCleaner(ctx context.Context) {
	defer p.wg.Done()
	ticker := time.NewTicker(p.config.IdleTimeout / 2)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-p.shutdownChan:
			return
		case <-ticker.C:
			p.cleanIdleConnections()
		}
	}
}

// cleanIdleConnections removes connections that have been idle too long
func (p *ConnectionPool) cleanIdleConnections() {
	p.mutex.RLock()
	connections := make([]*ConnectionWrapper, len(p.connections))
	copy(connections, p.connections)
	p.mutex.RUnlock()

	for _, conn := range connections {
		if conn.IsIdle(p.config.IdleTimeout) {
			// Try to remove from idle pool
			select {
			case idleConn := <-p.idle:
				if idleConn == conn {
					p.destroyConnection(conn)
				} else {
					// Put back the wrong connection
					select {
					case p.idle <- idleConn:
					default:
						p.destroyConnection(idleConn)
					}
				}
			default:
				// Connection not in idle pool
			}
		}
	}
}

// Close closes the connection pool
func (p *ConnectionPool) Close() error {
	p.mutex.Lock()
	if p.closed {
		p.mutex.Unlock()
		return nil
	}
	p.closed = true
	close(p.shutdownChan)
	p.mutex.Unlock()

	// Wait for background goroutines to finish
	p.wg.Wait()

	// Close all connections
	p.mutex.Lock()
	for _, conn := range p.connections {
		conn.Close()
	}
	p.connections = nil
	p.mutex.Unlock()

	// Drain idle channel
	for {
		select {
		case conn := <-p.idle:
			conn.Close()
		default:
			return nil
		}
	}
}

// GetMetrics returns pool metrics
func (p *ConnectionPool) GetMetrics() map[string]interface{} {
	p.mutex.RLock()
	totalConns := len(p.connections)
	idleConns := len(p.idle)
	p.mutex.RUnlock()

	totalCreated := atomic.LoadUint64(&p.totalCreated)
	totalDestroyed := atomic.LoadUint64(&p.totalDestroyed)
	totalRequests := atomic.LoadUint64(&p.totalRequests)
	totalHits := atomic.LoadUint64(&p.totalHits)
	totalMisses := atomic.LoadUint64(&p.totalMisses)
	healthChecks := atomic.LoadUint64(&p.healthChecks)
	unhealthyConns := atomic.LoadUint64(&p.unhealthyConns)

	hitRate := float64(0)
	if totalRequests > 0 {
		hitRate = float64(totalHits) / float64(totalRequests) * 100
	}

	utilization := float64(0)
	if p.config.MaxConnections > 0 {
		utilization = float64(totalConns) / float64(p.config.MaxConnections) * 100
	}

	return map[string]interface{}{
		"total_connections":    totalConns,
		"idle_connections":     idleConns,
		"active_connections":   totalConns - idleConns,
		"total_created":        totalCreated,
		"total_destroyed":      totalDestroyed,
		"total_requests":       totalRequests,
		"total_hits":           totalHits,
		"total_misses":         totalMisses,
		"hit_rate_percent":     hitRate,
		"utilization_percent":  utilization,
		"health_checks":        healthChecks,
		"unhealthy_connections": unhealthyConns,
	}
}