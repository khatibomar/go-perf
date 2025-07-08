package main

import (
	"bytes"
	"context"
	"fmt"
	"sync"
	"testing"
	"time"
)

func TestProtocolSerialization(t *testing.T) {
	protocol := NewProtocol()

	// Test message with data and metadata
	msg := &Message{
		ID:   12345,
		Type: MsgTypeData,
		Data: []byte("Hello, World!"),
		Metadata: map[string]string{
			"key1": "value1",
			"key2": "value2",
		},
	}

	// Serialize
	data, err := protocol.SerializeMessage(msg)
	if err != nil {
		t.Fatalf("Serialization failed: %v", err)
	}
	defer protocol.ReleaseBuffer(data)

	// Deserialize
	deserializedMsg, err := protocol.DeserializeMessage(data)
	if err != nil {
		t.Fatalf("Deserialization failed: %v", err)
	}

	// Verify
	if deserializedMsg.ID != msg.ID {
		t.Errorf("ID mismatch: expected %d, got %d", msg.ID, deserializedMsg.ID)
	}
	if deserializedMsg.Type != msg.Type {
		t.Errorf("Type mismatch: expected %d, got %d", msg.Type, deserializedMsg.Type)
	}
	if !bytes.Equal(deserializedMsg.Data, msg.Data) {
		t.Errorf("Data mismatch: expected %s, got %s", msg.Data, deserializedMsg.Data)
	}
	if len(deserializedMsg.Metadata) != len(msg.Metadata) {
		t.Errorf("Metadata length mismatch: expected %d, got %d", len(msg.Metadata), len(deserializedMsg.Metadata))
	}
	for k, v := range msg.Metadata {
		if deserializedMsg.Metadata[k] != v {
			t.Errorf("Metadata mismatch for key %s: expected %s, got %s", k, v, deserializedMsg.Metadata[k])
		}
	}
}

func TestProtocolFraming(t *testing.T) {
	protocol := NewProtocol()

	// Create test frame
	originalFrame := &Frame{
		Length:   13,
		Type:     MsgTypeData,
		Flags:    FlagCompressed,
		Reserved: 0,
		Payload:  []byte("Hello, World!"),
	}

	// Write frame to buffer
	buf := &bytes.Buffer{}
	err := protocol.WriteFrame(buf, originalFrame)
	if err != nil {
		t.Fatalf("WriteFrame failed: %v", err)
	}

	// Read frame from buffer
	readFrame, err := protocol.ReadFrame(buf)
	if err != nil {
		t.Fatalf("ReadFrame failed: %v", err)
	}
	defer protocol.ReleaseFrame(readFrame)

	// Verify
	if readFrame.Length != originalFrame.Length {
		t.Errorf("Length mismatch: expected %d, got %d", originalFrame.Length, readFrame.Length)
	}
	if readFrame.Type != originalFrame.Type {
		t.Errorf("Type mismatch: expected %d, got %d", originalFrame.Type, readFrame.Type)
	}
	if readFrame.Flags != originalFrame.Flags {
		t.Errorf("Flags mismatch: expected %d, got %d", originalFrame.Flags, readFrame.Flags)
	}
	if !bytes.Equal(readFrame.Payload, originalFrame.Payload) {
		t.Errorf("Payload mismatch: expected %s, got %s", originalFrame.Payload, readFrame.Payload)
	}
}

func TestServerClientCommunication(t *testing.T) {
	// Start server
	server := NewServer(4)
	server.RegisterHandler(MsgTypeData, MessageHandlerFunc(echoHandler))

	err := server.Listen(":0") // Use random port
	if err != nil {
		t.Fatalf("Failed to start server: %v", err)
	}

	addr := server.listener.Addr().String()
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	go func() {
		server.Start(ctx)
	}()
	defer server.Shutdown()

	// Give server time to start
	time.Sleep(100 * time.Millisecond)

	// Create client
	client := NewClient(2, 5*time.Second)
	err = client.Connect(addr)
	if err != nil {
		t.Fatalf("Failed to connect client: %v", err)
	}
	defer client.Close()

	// Test ping
	pingCtx, pingCancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer pingCancel()
	err = client.Ping(pingCtx)
	if err != nil {
		t.Fatalf("Ping failed: %v", err)
	}

	// Test message exchange
	testData := []byte("Hello, Server!")
	testMetadata := map[string]string{"test": "value"}

	resp, err := client.SendMessage(context.Background(), MsgTypeData, testData, testMetadata)
	if err != nil {
		t.Fatalf("SendMessage failed: %v", err)
	}

	if !bytes.Equal(resp.Data, testData) {
		t.Errorf("Response data mismatch: expected %s, got %s", testData, resp.Data)
	}
}

func TestConcurrentClients(t *testing.T) {
	// Start server
	server := NewServer(4)
	server.RegisterHandler(MsgTypeData, MessageHandlerFunc(echoHandler))

	err := server.Listen(":0")
	if err != nil {
		t.Fatalf("Failed to start server: %v", err)
	}

	addr := server.listener.Addr().String()
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	go func() {
		server.Start(ctx)
	}()
	defer server.Shutdown()

	time.Sleep(200 * time.Millisecond)

	// Test with multiple concurrent clients
	numClients := 3
	messagesPerClient := 10

	var wg sync.WaitGroup
	errorChan := make(chan error, numClients)

	for i := 0; i < numClients; i++ {
		wg.Add(1)
		go func(clientID int) {
			defer wg.Done()

			client := NewClient(1, 10*time.Second)
			err := client.Connect(addr)
			if err != nil {
				errorChan <- fmt.Errorf("client %d connect failed: %v", clientID, err)
				return
			}
			defer client.Close()

			for j := 0; j < messagesPerClient; j++ {
				testData := []byte(fmt.Sprintf("Message %d from client %d", j, clientID))
				resp, err := client.SendMessage(context.Background(), MsgTypeData, testData, nil)
				if err != nil {
					errorChan <- fmt.Errorf("client %d message %d failed: %v", clientID, j, err)
					return
				}

				if !bytes.Equal(resp.Data, testData) {
					errorChan <- fmt.Errorf("client %d message %d response mismatch", clientID, j)
					return
				}
			}
		}(i)
	}

	wg.Wait()
	close(errorChan)

	// Check for errors
	for err := range errorChan {
		t.Error(err)
	}

	// Verify server stats
	stats := server.GetStats()
	expectedMessages := int64(numClients * messagesPerClient)
	if stats.MessagesHandled < expectedMessages {
		t.Errorf("Expected at least %d messages handled, got %d", expectedMessages, stats.MessagesHandled)
	}
}

func TestConnectionPooling(t *testing.T) {
	// Start server
	server := NewServer(5)
	server.RegisterHandler(MsgTypeData, MessageHandlerFunc(echoHandler))

	err := server.Listen(":0")
	if err != nil {
		t.Fatalf("Failed to start server: %v", err)
	}

	addr := server.listener.Addr().String()
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	go func() {
		server.Start(ctx)
	}()
	defer server.Shutdown()

	time.Sleep(100 * time.Millisecond)

	// Create client with limited connections
	client := NewClient(3, 5*time.Second)
	err = client.Connect(addr)
	if err != nil {
		t.Fatalf("Failed to connect client: %v", err)
	}
	defer client.Close()

	// Verify connection count
	stats := client.GetStats()
	if stats.Connections != 3 {
		t.Errorf("Expected 3 connections, got %d", stats.Connections)
	}

	// Send multiple messages to test connection reuse
	for i := 0; i < 50; i++ {
		testData := []byte(fmt.Sprintf("Message %d", i))
		_, err := client.SendMessage(context.Background(), MsgTypeData, testData, nil)
		if err != nil {
			t.Errorf("Message %d failed: %v", i, err)
		}
	}

	// Verify that we still have the same number of connections
	stats = client.GetStats()
	if stats.Connections != 3 {
		t.Errorf("Connection count changed: expected 3, got %d", stats.Connections)
	}
}

func TestErrorHandling(t *testing.T) {
	protocol := NewProtocol()

	// Test invalid frame deserialization
	invalidData := []byte{0x01, 0x02, 0x03} // Too short
	_, err := protocol.DeserializeMessage(invalidData)
	if err == nil {
		t.Error("Expected error for invalid data, got nil")
	}

	// Test frame too big
	buf := &bytes.Buffer{}
	bigFrame := &Frame{
		Length:  MaxFrameSize + 1,
		Type:    MsgTypeData,
		Flags:   0,
		Payload: make([]byte, MaxFrameSize+1),
	}

	err = protocol.WriteFrame(buf, bigFrame)
	if err != nil {
		t.Fatalf("WriteFrame failed: %v", err)
	}

	_, err = protocol.ReadFrame(buf)
	if err != ErrFrameTooBig {
		t.Errorf("Expected ErrFrameTooBig, got %v", err)
	}
}

func TestMessageHandlers(t *testing.T) {
	tests := []struct {
		name     string
		handler  MessageHandler
		input    string
		expected string
	}{
		{
			name:     "echo",
			handler:  MessageHandlerFunc(echoHandler),
			input:    "Hello, World!",
			expected: "Hello, World!",
		},
		{
			name:     "reverse",
			handler:  MessageHandlerFunc(reverseHandler),
			input:    "Hello",
			expected: "olleH",
		},
		{
			name:     "uppercase",
			handler:  MessageHandlerFunc(uppercaseHandler),
			input:    "hello world",
			expected: "HELLO WORLD",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			msg := &Message{
				ID:   1,
				Type: MsgTypeData,
				Data: []byte(tt.input),
			}

			resp, err := tt.handler.Handle(context.Background(), msg)
			if err != nil {
				t.Fatalf("Handler failed: %v", err)
			}

			if string(resp.Data) != tt.expected {
				t.Errorf("Expected %s, got %s", tt.expected, string(resp.Data))
			}
		})
	}
}



// Helper function to start a test server
func startTestServer(t *testing.T) (string, func()) {
	server := NewServer(4)
	server.RegisterHandler(MsgTypeData, MessageHandlerFunc(echoHandler))

	err := server.Listen(":0")
	if err != nil {
		t.Fatalf("Failed to start server: %v", err)
	}

	addr := server.listener.Addr().String()
	ctx, cancel := context.WithCancel(context.Background())

	go func() {
		server.Start(ctx)
	}()

	time.Sleep(100 * time.Millisecond)

	cleanup := func() {
		cancel()
		server.Shutdown()
	}

	return addr, cleanup
}