package main

import (
	"bytes"
	"context"
	"fmt"
	"sync"
	"testing"
	"time"
)

// BenchmarkSerialization tests serialization performance
func BenchmarkSerialization(b *testing.B) {
	protocol := NewProtocol()
	msg := &Message{
		ID:   12345,
		Type: MsgTypeData,
		Data: make([]byte, 1024), // 1KB payload
		Metadata: map[string]string{
			"key1": "value1",
			"key2": "value2",
			"key3": "value3",
		},
	}

	// Fill data with test pattern
	for i := range msg.Data {
		msg.Data[i] = byte(i % 256)
	}

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		data, err := protocol.SerializeMessage(msg)
		if err != nil {
			b.Fatal(err)
		}
		protocol.ReleaseBuffer(data)
	}
}

// BenchmarkDeserialization tests deserialization performance
func BenchmarkDeserialization(b *testing.B) {
	protocol := NewProtocol()
	msg := &Message{
		ID:   12345,
		Type: MsgTypeData,
		Data: make([]byte, 1024), // 1KB payload
		Metadata: map[string]string{
			"key1": "value1",
			"key2": "value2",
			"key3": "value3",
		},
	}

	// Fill data with test pattern
	for i := range msg.Data {
		msg.Data[i] = byte(i % 256)
	}

	// Pre-serialize the message
	data, err := protocol.SerializeMessage(msg)
	if err != nil {
		b.Fatal(err)
	}
	defer protocol.ReleaseBuffer(data)

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		_, err := protocol.DeserializeMessage(data)
		if err != nil {
			b.Fatal(err)
		}
	}
}

// BenchmarkSerializationThroughput measures serialization throughput
func BenchmarkSerializationThroughput(b *testing.B) {
	sizes := []int{64, 256, 1024, 4096, 16384} // Different payload sizes

	for _, size := range sizes {
		b.Run(fmt.Sprintf("Size%d", size), func(b *testing.B) {
			protocol := NewProtocol()
			msg := &Message{
				ID:   12345,
				Type: MsgTypeData,
				Data: make([]byte, size),
				Metadata: map[string]string{
					"timestamp": "2023-01-01T00:00:00Z",
					"source":    "benchmark",
				},
			}

			// Fill data with test pattern
			for i := range msg.Data {
				msg.Data[i] = byte(i % 256)
			}

			b.SetBytes(int64(size))
			b.ResetTimer()
			b.ReportAllocs()

			for i := 0; i < b.N; i++ {
				data, err := protocol.SerializeMessage(msg)
				if err != nil {
					b.Fatal(err)
				}
				protocol.ReleaseBuffer(data)
			}
		})
	}
}

// BenchmarkFrameProcessing tests frame read/write performance
func BenchmarkFrameProcessing(b *testing.B) {
	protocol := NewProtocol()
	payload := make([]byte, 1024)
	for i := range payload {
		payload[i] = byte(i % 256)
	}

	frame := &Frame{
		Length:  uint32(len(payload)),
		Type:    MsgTypeData,
		Flags:   0,
		Payload: payload,
	}

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		buf := &bytes.Buffer{}
		err := protocol.WriteFrame(buf, frame)
		if err != nil {
			b.Fatal(err)
		}

		readFrame, err := protocol.ReadFrame(buf)
		if err != nil {
			b.Fatal(err)
		}
		protocol.ReleaseFrame(readFrame)
	}
}

// BenchmarkConcurrentSerialization tests concurrent serialization performance
func BenchmarkConcurrentSerialization(b *testing.B) {
	protocol := NewProtocol()
	msg := &Message{
		ID:   12345,
		Type: MsgTypeData,
		Data: make([]byte, 1024),
		Metadata: map[string]string{
			"key1": "value1",
			"key2": "value2",
		},
	}

	// Fill data with test pattern
	for i := range msg.Data {
		msg.Data[i] = byte(i % 256)
	}

	b.ResetTimer()
	b.ReportAllocs()

	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			data, err := protocol.SerializeMessage(msg)
			if err != nil {
				b.Error(err)
				return
			}
			protocol.ReleaseBuffer(data)
		}
	})
}

// BenchmarkEndToEndCommunication tests full client-server communication
func BenchmarkEndToEndCommunication(b *testing.B) {
	// Start server
	server := NewServer(8)
	server.RegisterHandler(MsgTypeData, MessageHandlerFunc(echoHandler))

	err := server.Listen(":0")
	if err != nil {
		b.Fatalf("Failed to start server: %v", err)
	}

	addr := server.listener.Addr().String()
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	go func() {
		server.Start(ctx)
	}()
	defer server.Shutdown()

	time.Sleep(100 * time.Millisecond)

	// Create client
	client := NewClient(4, 10*time.Second)
	err = client.Connect(addr)
	if err != nil {
		b.Fatalf("Failed to connect client: %v", err)
	}
	defer client.Close()

	testData := make([]byte, 1024)
	for i := range testData {
		testData[i] = byte(i % 256)
	}

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		_, err := client.SendMessage(context.Background(), MsgTypeData, testData, nil)
		if err != nil {
			b.Fatal(err)
		}
	}
}

// BenchmarkConcurrentClients tests performance with multiple concurrent clients
func BenchmarkConcurrentClients(b *testing.B) {
	// Start server
	server := NewServer(16)
	server.RegisterHandler(MsgTypeData, MessageHandlerFunc(echoHandler))

	err := server.Listen(":0")
	if err != nil {
		b.Fatalf("Failed to start server: %v", err)
	}

	addr := server.listener.Addr().String()
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	go func() {
		server.Start(ctx)
	}()
	defer server.Shutdown()

	time.Sleep(100 * time.Millisecond)

	// Create multiple clients
	numClients := 8
	clients := make([]*Client, numClients)
	for i := 0; i < numClients; i++ {
		clients[i] = NewClient(2, 10*time.Second)
		err = clients[i].Connect(addr)
		if err != nil {
			b.Fatalf("Failed to connect client %d: %v", i, err)
		}
		defer clients[i].Close()
	}

	testData := make([]byte, 512)
	for i := range testData {
		testData[i] = byte(i % 256)
	}

	b.ResetTimer()
	b.ReportAllocs()

	b.RunParallel(func(pb *testing.PB) {
		clientID := 0
		var mu sync.Mutex
		mu.Lock()
		myClient := clients[clientID%numClients]
		clientID++
		mu.Unlock()

		for pb.Next() {
			_, err := myClient.SendMessage(context.Background(), MsgTypeData, testData, nil)
			if err != nil {
				b.Error(err)
				return
			}
		}
	})
}

// BenchmarkMessageHandlers tests performance of different message handlers
func BenchmarkMessageHandlers(b *testing.B) {
	handlers := map[string]MessageHandler{
		"echo":      MessageHandlerFunc(echoHandler),
		"reverse":   MessageHandlerFunc(reverseHandler),
		"uppercase": MessageHandlerFunc(uppercaseHandler),
	}

	testData := make([]byte, 1024)
	for i := range testData {
		testData[i] = byte('a' + (i % 26))
	}

	msg := &Message{
		ID:   1,
		Type: MsgTypeData,
		Data: testData,
	}

	for name, handler := range handlers {
		b.Run(name, func(b *testing.B) {
			b.ResetTimer()
			b.ReportAllocs()

			for i := 0; i < b.N; i++ {
				_, err := handler.Handle(context.Background(), msg)
				if err != nil {
					b.Fatal(err)
				}
			}
		})
	}
}

// BenchmarkBufferPooling tests buffer pool performance
func BenchmarkBufferPooling(b *testing.B) {
	protocol := NewProtocol()

	b.Run("WithPool", func(b *testing.B) {
		b.ResetTimer()
		b.ReportAllocs()

		for i := 0; i < b.N; i++ {
			buf := protocol.bufferPool.Get().([]byte)
			// Simulate usage
			_ = buf[:1024]
			protocol.bufferPool.Put(buf)
		}
	})

	b.Run("WithoutPool", func(b *testing.B) {
		b.ResetTimer()
		b.ReportAllocs()

		for i := 0; i < b.N; i++ {
			buf := make([]byte, 4096)
			// Simulate usage
			_ = buf[:1024]
			// No pooling, let GC handle it
		}
	})
}



// BenchmarkProtocolOverhead measures protocol overhead
func BenchmarkProtocolOverhead(b *testing.B) {
	protocol := NewProtocol()

	payloadSizes := []int{0, 64, 256, 1024, 4096}

	for _, size := range payloadSizes {
		b.Run(fmt.Sprintf("Payload%d", size), func(b *testing.B) {
			msg := &Message{
				ID:   12345,
				Type: MsgTypeData,
				Data: make([]byte, size),
			}

			// Fill data with test pattern
			for i := range msg.Data {
				msg.Data[i] = byte(i % 256)
			}

			b.ResetTimer()

			for i := 0; i < b.N; i++ {
				data, err := protocol.SerializeMessage(msg)
				if err != nil {
					b.Fatal(err)
				}

				// Calculate overhead
				overhead := len(data) - size
				if size > 0 {
					overheadPercent := float64(overhead) / float64(size) * 100
					if i == 0 { // Report only once
						b.Logf("Payload: %d bytes, Total: %d bytes, Overhead: %d bytes (%.2f%%)",
							size, len(data), overhead, overheadPercent)
					}
				}

				protocol.ReleaseBuffer(data)
			}
		})
	}
}

// BenchmarkZeroCopyDeserialization tests zero-copy deserialization performance
func BenchmarkZeroCopyDeserialization(b *testing.B) {
	protocol := NewProtocol()

	// Create a large message with metadata
	msg := &Message{
		ID:   12345,
		Type: MsgTypeData,
		Data: make([]byte, 8192), // 8KB payload
		Metadata: map[string]string{
			"timestamp": "2023-01-01T00:00:00Z",
			"source":    "benchmark",
			"version":   "1.0.0",
			"checksum":  "abc123def456",
		},
	}

	// Fill data with test pattern
	for i := range msg.Data {
		msg.Data[i] = byte(i % 256)
	}

	// Pre-serialize the message
	data, err := protocol.SerializeMessage(msg)
	if err != nil {
		b.Fatal(err)
	}
	defer protocol.ReleaseBuffer(data)

	b.SetBytes(int64(len(data)))
	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		deserializedMsg, err := protocol.DeserializeMessage(data)
		if err != nil {
			b.Fatal(err)
		}
		// Verify zero-copy by checking if data slice points to original buffer
		if len(deserializedMsg.Data) > 0 && &deserializedMsg.Data[0] == &data[0] {
			// This would indicate zero-copy is working (data shares memory)
			// Note: This is a simplified check and may not work in all cases
		}
	}
}