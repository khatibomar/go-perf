package main

import (
	"context"
	"strings"
	"testing"
	"time"

	"exercise-04-04/implementations"
	"exercise-04-04/memory"
	"exercise-04-04/metrics"
	"exercise-04-04/pipeline"
)

func TestBufferPool(t *testing.T) {
	pool := memory.NewBufferPool()

	// Test getting and putting buffers
	buf1 := pool.Get(1024)
	if len(buf1) != 1024 {
		t.Errorf("Expected buffer length 1024, got %d", len(buf1))
	}

	buf2 := pool.Get(2048)
	if len(buf2) != 2048 {
		t.Errorf("Expected buffer length 2048, got %d", len(buf2))
	}

	// Put buffers back
	pool.Put(buf1)
	pool.Put(buf2)

	// Get buffers again (should reuse)
	buf3 := pool.Get(1024)
	if len(buf3) != 1024 {
		t.Errorf("Expected reused buffer length 1024, got %d", len(buf3))
	}

	pool.Put(buf3)
}

func TestMemoryManager(t *testing.T) {
	manager := memory.NewManager(100 * 1024 * 1024) // 100MB

	// Test initial state
	if manager.GetLimit() != 100*1024*1024 {
		t.Errorf("Expected limit 100MB, got %d", manager.GetLimit())
	}

	// Test pressure detection (this is approximate since it depends on actual memory usage)
	initialPressure := manager.IsUnderPressure()
	t.Logf("Initial memory pressure: %v", initialPressure)

	// Test cleanup
	manager.TriggerCleanup() // Should not panic

	// Test custom cleanup function
	cleanupCalled := false
	manager.SetCleanupFunc(func() {
		cleanupCalled = true
	})

	manager.TriggerCleanup()
	if !cleanupCalled {
		t.Error("Custom cleanup function was not called")
	}
}

func TestTextProcessor(t *testing.T) {
	pool := memory.NewBufferPool()
	processor := implementations.NewTextProcessor(pool, implementations.UppercaseTransform)

	// Create test chunk
	testData := "hello world"
	chunk := &pipeline.Chunk{
		ID:       1,
		Data:     []byte(testData),
		Metadata: make(map[string]interface{}),
	}

	ctx := context.Background()
	result, err := processor.Process(ctx, chunk)
	if err != nil {
		t.Fatalf("Processing failed: %v", err)
	}

	if result == nil {
		t.Fatal("Result is nil")
	}

	expected := "HELLO WORLD"
	if string(result.Data) != expected {
		t.Errorf("Expected %q, got %q", expected, string(result.Data))
	}

	// Check metadata
	if _, exists := result.Metadata["processed_at"]; !exists {
		t.Error("Missing processed_at metadata")
	}

	result.Release()
	processor.Close()
}

func TestTransformFunctions(t *testing.T) {
	tests := []struct {
		name      string
		transform implementations.TransformFunc
		input     string
		expected  string
	}{
		{
			name:      "Uppercase",
			transform: implementations.UppercaseTransform,
			input:     "hello world",
			expected:  "HELLO WORLD",
		},
		{
			name:      "Lowercase",
			transform: implementations.LowercaseTransform,
			input:     "HELLO WORLD",
			expected:  "hello world",
		},
		{
			name:      "TrimSpace",
			transform: implementations.TrimSpaceTransform,
			input:     "  hello  \n  world  \n",
			expected:  "hello\nworld\n",
		},
		{
			name:      "Reverse",
			transform: implementations.ReverseTransform,
			input:     "hello\nworld\n",
			expected:  "olleh\ndlrow\n",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.transform(tt.input)
			if result != tt.expected {
				t.Errorf("Expected %q, got %q", tt.expected, result)
			}
		})
	}
}

func TestChainTransforms(t *testing.T) {
	chained := implementations.ChainTransforms(
		implementations.TrimSpaceTransform,
		implementations.UppercaseTransform,
	)

	input := "  hello world  \n"
	expected := "HELLO WORLD\n"
	result := chained(input)

	if result != expected {
		t.Errorf("Expected %q, got %q", expected, result)
	}
}

func TestFileWriter(t *testing.T) {
	// Use buffered writer for testing
	writer := implementations.NewBufferedWriter()

	// Create test chunks
	chunks := []*pipeline.Chunk{
		{ID: 1, Data: []byte("chunk 1\n"), Metadata: make(map[string]interface{})},
		{ID: 2, Data: []byte("chunk 2\n"), Metadata: make(map[string]interface{})},
	}

	ctx := context.Background()
	for _, chunk := range chunks {
		if err := writer.WriteChunk(ctx, chunk); err != nil {
			t.Fatalf("Failed to write chunk: %v", err)
		}
	}

	// Check buffered data
	buffer := writer.GetBuffer()
	if len(buffer) != 2 {
		t.Errorf("Expected 2 chunks in buffer, got %d", len(buffer))
	}

	expected := []string{"chunk 1\n", "chunk 2\n"}
	for i, data := range buffer {
		if string(data) != expected[i] {
			t.Errorf("Expected %q, got %q", expected[i], string(data))
		}
	}

	writer.Close()
}

func TestStringReader(t *testing.T) {
	pool := memory.NewBufferPool()
	testData := "line 1\nline 2\nline 3\n"
	reader := implementations.NewStringReader(testData, pool)

	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	chunkChan := reader.ReadChunks(ctx, 10) // Small chunk size

	var chunks []*pipeline.Chunk
	for chunk := range chunkChan {
		chunks = append(chunks, chunk)
	}

	if len(chunks) == 0 {
		t.Error("No chunks received")
	}

	// Reconstruct data
	var reconstructed strings.Builder
	for _, chunk := range chunks {
		reconstructed.Write(chunk.Data)
		chunk.Release()
	}

	if reconstructed.String() != testData {
		t.Errorf("Data mismatch. Expected %q, got %q", testData, reconstructed.String())
	}

	reader.Close()
}

func TestPipelineIntegration(t *testing.T) {
	pool := memory.NewBufferPool()
	testData := "hello world\nthis is a test\n"

	// Create components
	reader := implementations.NewStringReader(testData, pool)
	processor := implementations.NewTextProcessor(pool, implementations.UppercaseTransform)
	writer := implementations.NewBufferedWriter()

	// Create pipeline
	config := &pipeline.Config{
		ChunkSize:       10,
		BufferSize:      5,
		WorkerCount:     2,
		MemoryLimit:     10 * 1024 * 1024,
		ProcessTimeout:  time.Second,
		BackpressureMax: 10,
	}

	pipe := pipeline.NewPipeline(reader, processor, writer, config)

	// Run pipeline
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := pipe.Start(ctx); err != nil {
		t.Fatalf("Failed to start pipeline: %v", err)
	}

	// Wait for completion
	for pipe.IsRunning() {
		time.Sleep(10 * time.Millisecond)
	}

	pipe.Stop()

	// Check results
	stats := pipe.GetMetrics()
	if stats.Processed == 0 {
		t.Error("No chunks were processed")
	}

	if stats.Written == 0 {
		t.Error("No chunks were written")
	}

	// Check output
	buffer := writer.GetBuffer()
	if len(buffer) == 0 {
		t.Error("No output in buffer")
	}

	// Verify transformation
	var output strings.Builder
	for _, data := range buffer {
		output.Write(data)
	}

	expected := strings.ToUpper(testData)
	if output.String() != expected {
		t.Errorf("Expected %q, got %q", expected, output.String())
	}

	// Cleanup
	reader.Close()
	processor.Close()
	writer.Close()
}

func TestMetricsCollector(t *testing.T) {
	collector := metrics.NewCollector()

	// Test initial state
	stats := collector.GetStats()
	if stats.Processed != 0 {
		t.Errorf("Expected 0 processed, got %d", stats.Processed)
	}

	// Test incrementing counters
	collector.IncrementProcessed()
	collector.IncrementWritten()
	collector.IncrementErrors()
	collector.IncrementMemoryPressure()

	stats = collector.GetStats()
	if stats.Processed != 1 {
		t.Errorf("Expected 1 processed, got %d", stats.Processed)
	}
	if stats.Written != 1 {
		t.Errorf("Expected 1 written, got %d", stats.Written)
	}
	if stats.Errors != 1 {
		t.Errorf("Expected 1 error, got %d", stats.Errors)
	}
	if stats.MemoryPressure != 1 {
		t.Errorf("Expected 1 memory pressure, got %d", stats.MemoryPressure)
	}

	// Test latency recording
	collector.RecordLatency(100 * time.Millisecond)
	collector.RecordLatency(200 * time.Millisecond)

	stats = collector.GetStats()
	if stats.MaxLatency != 200*time.Millisecond {
		t.Errorf("Expected max latency 200ms, got %v", stats.MaxLatency)
	}
	if stats.MinLatency != 100*time.Millisecond {
		t.Errorf("Expected min latency 100ms, got %v", stats.MinLatency)
	}

	// Test reset
	collector.Reset()
	stats = collector.GetStats()
	if stats.Processed != 0 {
		t.Errorf("Expected 0 processed after reset, got %d", stats.Processed)
	}
}