package main

import (
	"encoding/binary"
	"errors"
	"io"
	"sync"
	"unsafe"
)

// Protocol constants
const (
	// Frame header size: 4 bytes length + 1 byte type + 1 byte flags + 2 bytes reserved
	FrameHeaderSize = 8
	MaxFrameSize    = 1024 * 1024 // 1MB max frame size
	MagicNumber     = 0x12345678
)

// Message types
const (
	MsgTypeData     uint8 = 0x01
	MsgTypePing     uint8 = 0x02
	MsgTypePong     uint8 = 0x03
	MsgTypeClose    uint8 = 0x04
	MsgTypeAck      uint8 = 0x05
)

// Frame flags
const (
	FlagCompressed uint8 = 0x01
	FlagEncrypted  uint8 = 0x02
	FlagFragmented uint8 = 0x04
)

var (
	ErrInvalidFrame = errors.New("invalid frame")
	ErrFrameTooBig  = errors.New("frame too big")
	ErrInvalidType  = errors.New("invalid message type")
)

// Frame represents a protocol frame
type Frame struct {
	Length   uint32
	Type     uint8
	Flags    uint8
	Reserved uint16
	Payload  []byte
}

// Message represents a protocol message
type Message struct {
	ID       uint64
	Type     uint8
	Data     []byte
	Metadata map[string]string
}

// Protocol implements the custom binary protocol
type Protocol struct {
	bufferPool sync.Pool
	framePool  sync.Pool
}

// NewProtocol creates a new protocol instance
func NewProtocol() *Protocol {
	p := &Protocol{}
	
	// Initialize buffer pool
	p.bufferPool = sync.Pool{
		New: func() interface{} {
			return make([]byte, 4096)
		},
	}
	
	// Initialize frame pool
	p.framePool = sync.Pool{
		New: func() interface{} {
			return &Frame{}
		},
	}
	
	return p
}

// SerializeMessage serializes a message to bytes using zero-copy techniques
func (p *Protocol) SerializeMessage(msg *Message) ([]byte, error) {
	// Calculate total size needed
	metadataSize := 0
	for k, v := range msg.Metadata {
		metadataSize += 2 + len(k) + 2 + len(v) // 2 bytes for length prefix each
	}
	
	totalSize := 8 + 1 + 4 + len(msg.Data) + 2 + metadataSize // ID + Type + DataLen + Data + MetadataCount + Metadata
	
	// Get buffer from pool
	buf := p.bufferPool.Get().([]byte)
	if cap(buf) < totalSize {
		p.bufferPool.Put(buf)
		buf = make([]byte, totalSize)
	} else {
		buf = buf[:totalSize]
	}
	
	offset := 0
	
	// Serialize using unsafe for maximum performance
	*(*uint64)(unsafe.Pointer(&buf[offset])) = msg.ID
	offset += 8
	
	buf[offset] = msg.Type
	offset++
	
	*(*uint32)(unsafe.Pointer(&buf[offset])) = uint32(len(msg.Data))
	offset += 4
	
	// Copy data
	copy(buf[offset:], msg.Data)
	offset += len(msg.Data)
	
	// Serialize metadata
	*(*uint16)(unsafe.Pointer(&buf[offset])) = uint16(len(msg.Metadata))
	offset += 2
	
	for k, v := range msg.Metadata {
		*(*uint16)(unsafe.Pointer(&buf[offset])) = uint16(len(k))
		offset += 2
		copy(buf[offset:], k)
		offset += len(k)
		
		*(*uint16)(unsafe.Pointer(&buf[offset])) = uint16(len(v))
		offset += 2
		copy(buf[offset:], v)
		offset += len(v)
	}
	
	return buf, nil
}

// DeserializeMessage deserializes bytes to a message using zero-copy techniques
func (p *Protocol) DeserializeMessage(data []byte) (*Message, error) {
	if len(data) < 13 { // Minimum size: ID + Type + DataLen + MetadataCount
		return nil, ErrInvalidFrame
	}
	
	msg := &Message{}
	offset := 0
	
	// Deserialize using unsafe for maximum performance
	msg.ID = *(*uint64)(unsafe.Pointer(&data[offset]))
	offset += 8
	
	msg.Type = data[offset]
	offset++
	
	dataLen := *(*uint32)(unsafe.Pointer(&data[offset]))
	offset += 4
	
	if offset+int(dataLen) > len(data) {
		return nil, ErrInvalidFrame
	}
	
	// Zero-copy slice reference
	msg.Data = data[offset : offset+int(dataLen)]
	offset += int(dataLen)
	
	if offset+2 > len(data) {
		return nil, ErrInvalidFrame
	}
	
	metadataCount := *(*uint16)(unsafe.Pointer(&data[offset]))
	offset += 2
	
	if metadataCount > 0 {
		msg.Metadata = make(map[string]string, metadataCount)
		
		for i := uint16(0); i < metadataCount; i++ {
			if offset+2 > len(data) {
				return nil, ErrInvalidFrame
			}
			
			keyLen := *(*uint16)(unsafe.Pointer(&data[offset]))
			offset += 2
			
			if offset+int(keyLen) > len(data) {
				return nil, ErrInvalidFrame
			}
			
			// Zero-copy string conversion
			key := *(*string)(unsafe.Pointer(&struct {
				data uintptr
				len  int
			}{uintptr(unsafe.Pointer(&data[offset])), int(keyLen)}))
			offset += int(keyLen)
			
			if offset+2 > len(data) {
				return nil, ErrInvalidFrame
			}
			
			valueLen := *(*uint16)(unsafe.Pointer(&data[offset]))
			offset += 2
			
			if offset+int(valueLen) > len(data) {
				return nil, ErrInvalidFrame
			}
			
			// Zero-copy string conversion
			value := *(*string)(unsafe.Pointer(&struct {
				data uintptr
				len  int
			}{uintptr(unsafe.Pointer(&data[offset])), int(valueLen)}))
			offset += int(valueLen)
			
			msg.Metadata[key] = value
		}
	}
	
	return msg, nil
}

// WriteFrame writes a frame to the writer
func (p *Protocol) WriteFrame(w io.Writer, frame *Frame) error {
	header := make([]byte, FrameHeaderSize)
	
	binary.LittleEndian.PutUint32(header[0:4], frame.Length)
	header[4] = frame.Type
	header[5] = frame.Flags
	binary.LittleEndian.PutUint16(header[6:8], frame.Reserved)
	
	if _, err := w.Write(header); err != nil {
		return err
	}
	
	if frame.Length > 0 {
		if _, err := w.Write(frame.Payload); err != nil {
			return err
		}
	}
	
	return nil
}

// ReadFrame reads a frame from the reader
func (p *Protocol) ReadFrame(r io.Reader) (*Frame, error) {
	header := make([]byte, FrameHeaderSize)
	if _, err := io.ReadFull(r, header); err != nil {
		return nil, err
	}
	
	frame := p.framePool.Get().(*Frame)
	frame.Length = binary.LittleEndian.Uint32(header[0:4])
	frame.Type = header[4]
	frame.Flags = header[5]
	frame.Reserved = binary.LittleEndian.Uint16(header[6:8])
	
	if frame.Length > MaxFrameSize {
		p.framePool.Put(frame)
		return nil, ErrFrameTooBig
	}
	
	if frame.Length > 0 {
		frame.Payload = make([]byte, frame.Length)
		if _, err := io.ReadFull(r, frame.Payload); err != nil {
			p.framePool.Put(frame)
			return nil, err
		}
	}
	
	return frame, nil
}

// ReleaseFrame returns a frame to the pool
func (p *Protocol) ReleaseFrame(frame *Frame) {
	frame.Payload = nil
	p.framePool.Put(frame)
}

// ReleaseBuffer returns a buffer to the pool
func (p *Protocol) ReleaseBuffer(buf []byte) {
	if cap(buf) == 4096 {
		p.bufferPool.Put(buf[:cap(buf)])
	}
}