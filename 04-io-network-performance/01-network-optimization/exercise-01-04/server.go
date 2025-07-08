package main

import (
	"context"
	"fmt"
	"io"
	"log"
	"net"
	"sync"
	"sync/atomic"
	"time"
)

// ServerStats holds server statistics
type ServerStats struct {
	Connections     int64
	MessagesHandled int64
	BytesReceived   int64
	BytesSent       int64
	Errors          int64
}

// Server implements a high-performance protocol server
type Server struct {
	protocol   *Protocol
	listener   net.Listener
	workerPool chan struct{}
	connPool   *sync.Pool
	stats      *ServerStats
	shutdown   chan struct{}
	wg         sync.WaitGroup
	handlers   map[uint8]MessageHandler
	mu         sync.RWMutex
}

// MessageHandler defines the interface for message handlers
type MessageHandler interface {
	Handle(ctx context.Context, msg *Message) (*Message, error)
}

// MessageHandlerFunc is an adapter to allow functions to be used as MessageHandlers
type MessageHandlerFunc func(ctx context.Context, msg *Message) (*Message, error)

func (f MessageHandlerFunc) Handle(ctx context.Context, msg *Message) (*Message, error) {
	return f(ctx, msg)
}

// NewServer creates a new protocol server
func NewServer(maxWorkers int) *Server {
	return &Server{
		protocol:   NewProtocol(),
		workerPool: make(chan struct{}, maxWorkers),
		stats:      &ServerStats{},
		shutdown:   make(chan struct{}),
		handlers:   make(map[uint8]MessageHandler),
		connPool: &sync.Pool{
			New: func() interface{} {
				return &ConnWrapper{}
			},
		},
	}
}

// ConnWrapper wraps a connection with additional metadata
type ConnWrapper struct {
	conn      net.Conn
	lastUsed  time.Time
	buffer    []byte
	readBuf   []byte
}

// RegisterHandler registers a message handler for a specific message type
func (s *Server) RegisterHandler(msgType uint8, handler MessageHandler) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.handlers[msgType] = handler
}

// Listen starts listening on the specified address
func (s *Server) Listen(addr string) error {
	var err error
	s.listener, err = net.Listen("tcp", addr)
	if err != nil {
		return fmt.Errorf("failed to listen on %s: %w", addr, err)
	}
	return nil
}

// Start starts the server
func (s *Server) Start(ctx context.Context) error {
	if s.listener == nil {
		return fmt.Errorf("server not listening")
	}

	// Fill worker pool
	for i := 0; i < cap(s.workerPool); i++ {
		s.workerPool <- struct{}{}
	}

	log.Printf("Server started on %s with %d workers", s.listener.Addr(), cap(s.workerPool))

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
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
				atomic.AddInt64(&s.stats.Errors, 1)
				continue
			}
		}

		// Try to get a worker
		select {
		case <-s.workerPool:
			s.wg.Add(1)
			go s.handleConnection(conn)
		default:
			// No workers available, close connection
			conn.Close()
			atomic.AddInt64(&s.stats.Errors, 1)
		}
	}
}

// handleConnection handles a single connection
func (s *Server) handleConnection(conn net.Conn) {
	defer s.wg.Done()
	defer func() { s.workerPool <- struct{}{} }() // Return worker to pool
	defer conn.Close()

	atomic.AddInt64(&s.stats.Connections, 1)

	// Get connection wrapper from pool
	connWrapper := s.connPool.Get().(*ConnWrapper)
	connWrapper.conn = conn
	connWrapper.lastUsed = time.Now()
	if connWrapper.buffer == nil {
		connWrapper.buffer = make([]byte, 4096)
	}
	if connWrapper.readBuf == nil {
		connWrapper.readBuf = make([]byte, 4096)
	}
	defer func() {
		connWrapper.conn = nil
		s.connPool.Put(connWrapper)
	}()

	// Set connection timeouts
	conn.SetReadDeadline(time.Now().Add(30 * time.Second))
	conn.SetWriteDeadline(time.Now().Add(30 * time.Second))

	for {
		// Read frame
		frame, err := s.protocol.ReadFrame(conn)
		if err != nil {
			if err != io.EOF {
				atomic.AddInt64(&s.stats.Errors, 1)
			}
			return
		}

		atomic.AddInt64(&s.stats.BytesReceived, int64(frame.Length+FrameHeaderSize))

		// Handle different frame types
		switch frame.Type {
		case MsgTypeData:
			if err := s.handleDataFrame(conn, frame); err != nil {
				atomic.AddInt64(&s.stats.Errors, 1)
				s.protocol.ReleaseFrame(frame)
				return
			}
		case MsgTypePing:
			if err := s.handlePingFrame(conn, frame); err != nil {
				atomic.AddInt64(&s.stats.Errors, 1)
				s.protocol.ReleaseFrame(frame)
				return
			}
		case MsgTypeClose:
			s.protocol.ReleaseFrame(frame)
			return
		default:
			atomic.AddInt64(&s.stats.Errors, 1)
		}

		s.protocol.ReleaseFrame(frame)
		atomic.AddInt64(&s.stats.MessagesHandled, 1)

		// Update connection timeout
		conn.SetReadDeadline(time.Now().Add(30 * time.Second))
	}
}

// handleDataFrame handles data frames
func (s *Server) handleDataFrame(conn net.Conn, frame *Frame) error {
	// Deserialize message
	msg, err := s.protocol.DeserializeMessage(frame.Payload)
	if err != nil {
		return err
	}

	// Get handler
	s.mu.RLock()
	handler, exists := s.handlers[msg.Type]
	s.mu.RUnlock()

	if !exists {
		// Send error response
		errorResp := &Message{
			ID:   msg.ID,
			Type: MsgTypeAck,
			Data: []byte("unknown message type"),
		}
		return s.sendMessage(conn, errorResp)
	}

	// Handle message
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	resp, err := handler.Handle(ctx, msg)
	if err != nil {
		// Send error response
		errorResp := &Message{
			ID:   msg.ID,
			Type: MsgTypeAck,
			Data: []byte(err.Error()),
		}
		return s.sendMessage(conn, errorResp)
	}

	if resp != nil {
		return s.sendMessage(conn, resp)
	}

	return nil
}

// handlePingFrame handles ping frames
func (s *Server) handlePingFrame(conn net.Conn, frame *Frame) error {
	// Send pong response
	pongFrame := &Frame{
		Length:  0,
		Type:    MsgTypePong,
		Flags:   0,
		Payload: nil,
	}

	return s.protocol.WriteFrame(conn, pongFrame)
}

// sendMessage sends a message over the connection
func (s *Server) sendMessage(conn net.Conn, msg *Message) error {
	// Serialize message
	data, err := s.protocol.SerializeMessage(msg)
	if err != nil {
		return err
	}
	defer s.protocol.ReleaseBuffer(data)

	// Create frame
	frame := &Frame{
		Length:  uint32(len(data)),
		Type:    MsgTypeData,
		Flags:   0,
		Payload: data,
	}

	err = s.protocol.WriteFrame(conn, frame)
	if err == nil {
		atomic.AddInt64(&s.stats.BytesSent, int64(frame.Length+FrameHeaderSize))
	}

	return err
}

// GetStats returns server statistics
func (s *Server) GetStats() ServerStats {
	return ServerStats{
		Connections:     atomic.LoadInt64(&s.stats.Connections),
		MessagesHandled: atomic.LoadInt64(&s.stats.MessagesHandled),
		BytesReceived:   atomic.LoadInt64(&s.stats.BytesReceived),
		BytesSent:       atomic.LoadInt64(&s.stats.BytesSent),
		Errors:          atomic.LoadInt64(&s.stats.Errors),
	}
}

// Shutdown gracefully shuts down the server
func (s *Server) Shutdown() error {
	close(s.shutdown)
	if s.listener != nil {
		s.listener.Close()
	}
	s.wg.Wait()
	return nil
}