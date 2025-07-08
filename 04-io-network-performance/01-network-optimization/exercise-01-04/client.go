package main

import (
	"context"
	"fmt"
	"net"
	"sync/atomic"
	"time"
)

// ClientStats holds client statistics
type ClientStats struct {
	MessagesSent     int64
	MessagesReceived int64
	BytesSent        int64
	BytesReceived    int64
	Errors           int64
	Connections      int64
}

// Client implements a high-performance protocol client
type Client struct {
	protocol     *Protocol
	connPool     chan net.Conn
	maxConns     int
	timeout      time.Duration
	stats        *ClientStats
	msgIDCounter uint64
}

// NewClient creates a new protocol client
func NewClient(maxConns int, timeout time.Duration) *Client {
	return &Client{
		protocol: NewProtocol(),
		connPool: make(chan net.Conn, maxConns),
		maxConns: maxConns,
		timeout:  timeout,
		stats:    &ClientStats{},
	}
}

// Connect establishes connections to the server
func (c *Client) Connect(addr string) error {
	for i := 0; i < c.maxConns; i++ {
		conn, err := net.DialTimeout("tcp", addr, c.timeout)
		if err != nil {
			return fmt.Errorf("failed to connect to %s: %w", addr, err)
		}

		c.connPool <- conn
		atomic.AddInt64(&c.stats.Connections, 1)
	}

	return nil
}

// getConnection gets a connection from the pool
func (c *Client) getConnection() (net.Conn, error) {
	select {
	case conn := <-c.connPool:
		return conn, nil
	case <-time.After(c.timeout):
		return nil, fmt.Errorf("connection pool timeout")
	}
}

// putConnection returns a connection to the pool
func (c *Client) putConnection(conn net.Conn) {
	select {
	case c.connPool <- conn:
	default:
		// Pool is full, close connection
		conn.Close()
	}
}

// SendMessage sends a message and waits for response
func (c *Client) SendMessage(ctx context.Context, msgType uint8, data []byte, metadata map[string]string) (*Message, error) {
	// Generate unique message ID
	msgID := atomic.AddUint64(&c.msgIDCounter, 1)

	// Create message
	msg := &Message{
		ID:       msgID,
		Type:     msgType,
		Data:     data,
		Metadata: metadata,
	}

	// Get connection
	conn, err := c.getConnection()
	if err != nil {
		atomic.AddInt64(&c.stats.Errors, 1)
		return nil, err
	}
	defer c.putConnection(conn)

	// Serialize message
	serializedData, err := c.protocol.SerializeMessage(msg)
	if err != nil {
		atomic.AddInt64(&c.stats.Errors, 1)
		return nil, err
	}
	defer c.protocol.ReleaseBuffer(serializedData)

	// Create frame
	frame := &Frame{
		Length:  uint32(len(serializedData)),
		Type:    MsgTypeData,
		Flags:   0,
		Payload: serializedData,
	}

	// Send frame
	conn.SetWriteDeadline(time.Now().Add(c.timeout))
	err = c.protocol.WriteFrame(conn, frame)
	if err != nil {
		atomic.AddInt64(&c.stats.Errors, 1)
		return nil, err
	}

	atomic.AddInt64(&c.stats.MessagesSent, 1)
	atomic.AddInt64(&c.stats.BytesSent, int64(frame.Length+FrameHeaderSize))

	// Read response synchronously
	conn.SetReadDeadline(time.Now().Add(c.timeout))
	respFrame, err := c.protocol.ReadFrame(conn)
	if err != nil {
		atomic.AddInt64(&c.stats.Errors, 1)
		return nil, fmt.Errorf("failed to read response: %w", err)
	}
	defer c.protocol.ReleaseFrame(respFrame)

	atomic.AddInt64(&c.stats.BytesReceived, int64(respFrame.Length+FrameHeaderSize))

	// Handle response frame
	if respFrame.Type != MsgTypeData {
		atomic.AddInt64(&c.stats.Errors, 1)
		return nil, fmt.Errorf("unexpected response frame type: %d", respFrame.Type)
	}

	// Deserialize response
	resp, err := c.protocol.DeserializeMessage(respFrame.Payload)
	if err != nil {
		atomic.AddInt64(&c.stats.Errors, 1)
		return nil, fmt.Errorf("failed to deserialize response: %w", err)
	}

	atomic.AddInt64(&c.stats.MessagesReceived, 1)
	return resp, nil
}



// Ping sends a ping and waits for pong
func (c *Client) Ping(ctx context.Context) error {
	// Get connection
	conn, err := c.getConnection()
	if err != nil {
		return err
	}
	defer c.putConnection(conn)

	// Create ping frame
	frame := &Frame{
		Length:  0,
		Type:    MsgTypePing,
		Flags:   0,
		Payload: nil,
	}

	// Send ping
	conn.SetWriteDeadline(time.Now().Add(c.timeout))
	err = c.protocol.WriteFrame(conn, frame)
	if err != nil {
		return err
	}

	// Read pong response
	conn.SetReadDeadline(time.Now().Add(c.timeout))
	pongFrame, err := c.protocol.ReadFrame(conn)
	if err != nil {
		return fmt.Errorf("failed to read pong: %w", err)
	}
	defer c.protocol.ReleaseFrame(pongFrame)

	// Verify it's a pong
	if pongFrame.Type != MsgTypePong {
		return fmt.Errorf("expected pong, got frame type: %d", pongFrame.Type)
	}

	return nil
}



// GetStats returns client statistics
func (c *Client) GetStats() ClientStats {
	return ClientStats{
		MessagesSent:     atomic.LoadInt64(&c.stats.MessagesSent),
		MessagesReceived: atomic.LoadInt64(&c.stats.MessagesReceived),
		BytesSent:        atomic.LoadInt64(&c.stats.BytesSent),
		BytesReceived:    atomic.LoadInt64(&c.stats.BytesReceived),
		Errors:           atomic.LoadInt64(&c.stats.Errors),
		Connections:      atomic.LoadInt64(&c.stats.Connections),
	}
}

// Close closes all connections
func (c *Client) Close() error {
	for {
		select {
		case conn := <-c.connPool:
			conn.Close()
		default:
			return nil
		}
	}
}