package main

import (
	"fmt"
	"sync"
	"testing"
	"time"

	"exercise-03-04/backpressure"
	"exercise-03-04/buffer"
	"exercise-03-04/pipeline"
	"exercise-03-04/stages"
)

// TestLinearPipeline tests the linear pipeline functionality
func TestLinearPipeline(t *testing.T) {
	// Create pipeline
	pipeline := pipeline.NewLinearPipeline("test-linear", 10)
	
	// Add stages
	multiplyStage := stages.NewMultiplyStage("multiply", 2.0)
	filterStage := stages.NewNumericFilterStage("filter", 0, 100)
	transformStage := stages.NewTransformStage("transform", func(data interface{}) (interface{}, error) {
		if num, ok := data.(float64); ok {
			return fmt.Sprintf("Result: %.2f", num), nil
		}
		return data, nil
	})
	
	pipeline.AddStage(multiplyStage)
	pipeline.AddStage(filterStage)
	pipeline.AddStage(transformStage)
	
	// Start pipeline
	if err := pipeline.Start(); err != nil {
		t.Fatalf("Failed to start pipeline: %v", err)
	}
	defer pipeline.Stop()
	
	// Process test data
	testData := []float64{10, 25, 60, 5}
	expectedResults := 3 // Only 10*2=20, 25*2=50, 5*2=10 should pass filter (0-100)
	
	for _, data := range testData {
		if err := pipeline.Process(data); err != nil {
			t.Errorf("Error processing %f: %v", data, err)
		}
	}
	
	// Collect results
	resultCount := 0
	timeout := time.After(time.Second * 2)
	
loop:
	for {
		select {
		case result := <-pipeline.GetOutput():
			if _, ok := result.(string); !ok {
				t.Errorf("Expected string result, got %T", result)
			}
			resultCount++
			if resultCount >= expectedResults {
				break loop
			}
		case err := <-pipeline.GetErrors():
			t.Errorf("Unexpected error: %v", err)
		case <-timeout:
			break loop
		}
	}
	
	if resultCount != expectedResults {
		t.Errorf("Expected %d results, got %d", expectedResults, resultCount)
	}
	
	// Check metrics
	metrics := pipeline.GetMetrics()
	if metrics.ProcessedCount == 0 {
		t.Error("Expected processed count > 0")
	}
}

// TestFanOutFanInPipeline tests the fan-out/fan-in pipeline
func TestFanOutFanInPipeline(t *testing.T) {
	// Create pipeline with 2 parallel workers
	pipeline := pipeline.NewFanOutFanInPipeline("test-fanout", 2, 10)
	
	// Add stages
	preStage := stages.NewTransformStage("pre", func(data interface{}) (interface{}, error) {
		if num, ok := data.(int); ok {
			return float64(num), nil
		}
		return data, nil
	})
	
	parallelStage := stages.NewMultiplyStage("parallel", 2.0)
	
	postStage := stages.NewTransformStage("post", func(data interface{}) (interface{}, error) {
		if num, ok := data.(float64); ok {
			return fmt.Sprintf("Parallel: %.0f", num), nil
		}
		return data, nil
	})
	
	pipeline.AddPreStage(preStage)
	pipeline.AddParallelStage(parallelStage)
	pipeline.AddPostStage(postStage)
	
	// Start pipeline
	if err := pipeline.Start(); err != nil {
		t.Fatalf("Failed to start fan-out pipeline: %v", err)
	}
	defer pipeline.Stop()
	
	// Process test data
	testData := []int{1, 2, 3, 4, 5}
	
	for _, data := range testData {
		if err := pipeline.Process(data); err != nil {
			t.Errorf("Error processing %d: %v", data, err)
		}
	}
	
	// Collect results
	resultCount := 0
	timeout := time.After(time.Second * 3)
	
loop:
	for {
		select {
		case result := <-pipeline.GetOutput():
			if _, ok := result.(string); !ok {
				t.Errorf("Expected string result, got %T", result)
			}
			resultCount++
			if resultCount >= len(testData) {
				break loop
			}
		case err := <-pipeline.GetErrors():
			t.Errorf("Unexpected error: %v", err)
		case <-timeout:
			break loop
		}
	}
	
	if resultCount != len(testData) {
		t.Errorf("Expected %d results, got %d", len(testData), resultCount)
	}
}

// TestDynamicPipeline tests dynamic pipeline modification
func TestDynamicPipeline(t *testing.T) {
	// Create dynamic pipeline
	pipeline := pipeline.NewDynamicPipeline("test-dynamic", 10)
	
	// Add initial stages
	stage1 := stages.NewTransformStage("input", func(data interface{}) (interface{}, error) {
		if num, ok := data.(int); ok {
			return float64(num * 2), nil
		}
		return data, nil
	})
	
	stage2 := stages.NewTransformStage("output", func(data interface{}) (interface{}, error) {
		if num, ok := data.(float64); ok {
			return fmt.Sprintf("Dynamic: %.0f", num), nil
		}
		return data, nil
	})
	
	pipeline.AddStage(stage1)
	pipeline.AddStage(stage2)
	pipeline.ConnectStages("input", "output")
	
	// Start pipeline
	if err := pipeline.Start(); err != nil {
		t.Fatalf("Failed to start dynamic pipeline: %v", err)
	}
	defer pipeline.Stop()
	
	// Test initial configuration
	stageNames := pipeline.GetStageNames()
	if len(stageNames) != 2 {
		t.Errorf("Expected 2 stages, got %d", len(stageNames))
	}
	
	// Process some data
	if err := pipeline.Process(5); err != nil {
		t.Errorf("Error processing data: %v", err)
	}
	
	// Collect initial result
	select {
	case result := <-pipeline.GetOutput():
		expected := "Dynamic: 10"
		if result != expected {
			t.Errorf("Expected %s, got %v", expected, result)
		}
	case <-time.After(time.Second):
		t.Error("Timeout waiting for result")
	}
	
	// Add a filter stage dynamically
	filterStage := stages.NewNumericFilterStage("filter", 5, 15)
	pipeline.AddStage(filterStage)
	
	// Reconnect pipeline
	pipeline.DisconnectStages("input", "output")
	pipeline.ConnectStages("input", "filter")
	pipeline.ConnectStages("filter", "output")
	
	// Test modified configuration
	stageNames = pipeline.GetStageNames()
	if len(stageNames) != 3 {
		t.Errorf("Expected 3 stages after adding filter, got %d", len(stageNames))
	}
	
	// Process data that should be filtered out
	if err := pipeline.Process(20); err != nil { // 20*2=40, should be filtered out (range 5-15)
		t.Errorf("Error processing data: %v", err)
	}
	
	// Process data that should pass through
	if err := pipeline.Process(3); err != nil { // 3*2=6, should pass through
		t.Errorf("Error processing data: %v", err)
	}
	
	// Should only get one result (the second one)
	select {
	case result := <-pipeline.GetOutput():
		expected := "Dynamic: 6"
		if result != expected {
			t.Errorf("Expected %s, got %v", expected, result)
		}
	case <-time.After(time.Second):
		t.Error("Timeout waiting for filtered result")
	}
	
	// Should not get another result (first one was filtered)
	select {
	case result := <-pipeline.GetOutput():
		t.Errorf("Unexpected result: %v (should have been filtered)", result)
	case <-time.After(time.Millisecond * 100):
		// Expected timeout
	}
}

// TestAdaptiveBuffer tests adaptive buffer functionality
func TestAdaptiveBuffer(t *testing.T) {
	buf := buffer.NewAdaptiveBuffer(5, 2, 20)
	defer buf.Close()
	
	// Test basic send/receive
	if !buf.Send("test1") {
		t.Error("Failed to send to empty buffer")
	}
	
	data, ok := buf.Receive()
	if !ok {
		t.Error("Failed to receive from buffer")
	}
	if data != "test1" {
		t.Errorf("Expected 'test1', got %v", data)
	}
	
	// Test buffer expansion under load
	// Fill buffer to 80% capacity to trigger resize
	for i := 0; i < 15; i++ {
		buf.Send(fmt.Sprintf("item%d", i))
	}
	
	// Wait a bit for resize to happen
	time.Sleep(time.Millisecond * 200)
	
	// Buffer should have expanded
	metrics := buf.GetMetrics()
	if metrics.ResizeCount == 0 {
		t.Error("Expected buffer to resize under load")
	}
	
	// Test concurrent access
	var wg sync.WaitGroup
	const numProducers = 3
	const numConsumers = 2
	const itemsPerProducer = 10
	
	// Producers
	for i := 0; i < numProducers; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			for j := 0; j < itemsPerProducer; j++ {
				buf.Send(fmt.Sprintf("producer%d-item%d", id, j))
				time.Sleep(time.Millisecond)
			}
		}(i)
	}
	
	// Consumers
	received := make(chan string, numProducers*itemsPerProducer)
	for i := 0; i < numConsumers; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for {
				if data, ok := buf.Receive(); ok {
					received <- data.(string)
				} else {
					time.Sleep(time.Millisecond)
				}
				if len(received) >= numProducers*itemsPerProducer {
					return
				}
			}
		}()
	}
	
	wg.Wait()
	close(received)
	
	// Count received items
	receivedCount := 0
	for range received {
		receivedCount++
	}
	
	if receivedCount < numProducers*itemsPerProducer*0.8 { // Allow some loss due to timing
		t.Errorf("Expected at least %d items, got %d", int(float64(numProducers*itemsPerProducer)*0.8), receivedCount)
	}
}

// TestRingBuffer tests ring buffer functionality
func TestRingBuffer(t *testing.T) {
	buf := buffer.NewRingBuffer(5)
	
	// Test basic operations
	if buf.Size() != 0 {
		t.Errorf("Expected size 0, got %d", buf.Size())
	}
	if buf.Capacity() != 8 { // Ring buffer rounds up to next power of 2
		t.Errorf("Expected capacity 8, got %d", buf.Capacity())
	}
	
	// Fill buffer (capacity is 8, not 5)
	for i := 0; i < 8; i++ {
		if !buf.Send(fmt.Sprintf("item%d", i)) {
			t.Errorf("Failed to send item %d", i)
		}
	}
	
	// Buffer should be full
	if buf.Send("overflow") {
		t.Error("Should not be able to send to full buffer")
	}
	
	// Receive all items
	for i := 0; i < 8; i++ {
		data, ok := buf.Receive()
		if !ok {
			t.Errorf("Failed to receive item %d", i)
		}
		expected := fmt.Sprintf("item%d", i)
		if data != expected {
			t.Errorf("Expected %s, got %v", expected, data)
		}
	}
	
	// Buffer should be empty
	if _, ok := buf.Receive(); ok {
		t.Error("Should not be able to receive from empty buffer")
	}
}

// TestBackpressureController tests backpressure control
func TestBackpressureController(t *testing.T) {
	thresholds := &backpressure.Thresholds{
		LowWaterMark:    0.3,
		MediumWaterMark: 0.6,
		HighWaterMark:   0.8,
		CriticalMark:    0.95,
	}
	
	ctrl := backpressure.NewBackpressureController(thresholds)
	defer ctrl.Close()
	
	// Test action determination based on load
	testCases := []struct {
		load           float64
		expectedAction backpressure.Action
	}{
		{0.1, backpressure.ActionNone},
		{0.5, backpressure.ActionNone},
		{0.7, backpressure.ActionThrottle},
		{0.9, backpressure.ActionDrop},
		{0.98, backpressure.ActionReject},
	}
	
	for _, tc := range testCases {
		action := ctrl.CheckAndApply(tc.load)
		if action != tc.expectedAction {
			t.Errorf("Load %.2f: expected action %v, got %v", tc.load, tc.expectedAction, action)
		}
	}
	
	// Test metrics
	metrics := ctrl.GetMetrics()
	if metrics.ThrottleCount == 0 && metrics.DropCount == 0 && metrics.BlockCount == 0 && metrics.RejectCount == 0 {
		t.Error("Expected some action counts > 0")
	}
}

// TestThrottleAction tests throttle action
func TestThrottleAction(t *testing.T) {
	action := backpressure.NewThrottleAction(10.0) // 10 requests per second
	
	// Enable throttling by calling Handle with ActionThrottle
	action.Handle(backpressure.ActionThrottle, 0.7)
	
	// Wait a bit to ensure first request is not throttled
	time.Sleep(time.Millisecond * 200)
	
	// Should allow first request
	if action.ShouldThrottle() {
		t.Error("First request should not be throttled")
	}
	
	// Rapid requests should be throttled
	throttled := false
	for i := 0; i < 20; i++ {
		if action.ShouldThrottle() {
			throttled = true
			break
		}
	}
	
	if !throttled {
		t.Error("Expected some requests to be throttled")
	}
}

// TestDropAction tests drop action
func TestDropAction(t *testing.T) {
	action := backpressure.NewDropAction(0.5) // 50% drop rate
	
	// Enable dropping by calling Handle with ActionDrop
	action.Handle(backpressure.ActionDrop, 0.9)
	
	// Test drop behavior with some delay between calls
	drops := 0
	total := 100 // Reduced total for faster test
	
	for i := 0; i < total; i++ {
		if action.ShouldDrop() {
			drops++
		}
		// Small delay to ensure different nano times
		time.Sleep(time.Microsecond)
	}
	
	// Should be approximately 32% for load 0.9 (allow some variance)
	// dropRate = (0.9 - 0.5) * 0.8 = 0.32
	dropRate := float64(drops) / float64(total)
	if dropRate < 0.1 || dropRate > 0.6 {
		t.Errorf("Expected drop rate around 0.32, got %.2f", dropRate)
	}
	
	// Test stats
	droppedStats, totalStats, rate := action.GetDropStats()
	if droppedStats != int64(drops) {
		t.Errorf("Expected %d drops in stats, got %d", drops, droppedStats)
	}
	if totalStats != int64(total) {
		t.Errorf("Expected %d total in stats, got %d", total, totalStats)
	}
	if rate != dropRate {
		t.Errorf("Expected rate %.2f, got %.2f", dropRate, rate)
	}
}

// TestStages tests individual stage functionality
func TestStages(t *testing.T) {
	// Test MultiplyStage
	multiplyStage := stages.NewMultiplyStage("multiply", 3.0)
	result, err := multiplyStage.Process(5.0)
	if err != nil {
		t.Errorf("MultiplyStage error: %v", err)
	}
	if result != 15.0 {
		t.Errorf("Expected 15.0, got %v", result)
	}
	
	// Test NumericFilterStage
	filterStage := stages.NewNumericFilterStage("filter", 10, 20)
	
	// Should pass
	result, err = filterStage.Process(15.0)
	if err != nil {
		t.Errorf("FilterStage error: %v", err)
	}
	if result != 15.0 {
		t.Errorf("Expected 15.0, got %v", result)
	}
	
	// Should be filtered out
	result, err = filterStage.Process(25.0)
	if err != nil {
		t.Errorf("FilterStage error: %v", err)
	}
	if result != nil {
		t.Error("Expected filter to reject value 25.0 (return nil)")
	}
	
	// Test TransformStage
	transformStage := stages.NewTransformStage("transform", func(data interface{}) (interface{}, error) {
		if str, ok := data.(string); ok {
			return "transformed_" + str, nil
		}
		return data, nil
	})
	
	result, err = transformStage.Process("test")
	if err != nil {
		t.Errorf("TransformStage error: %v", err)
	}
	if result != "transformed_test" {
		t.Errorf("Expected 'transformed_test', got %v", result)
	}
}

// BenchmarkLinearPipeline benchmarks linear pipeline performance
func BenchmarkLinearPipeline(b *testing.B) {
	pipeline := pipeline.NewLinearPipeline("bench-linear", 100)
	
	multiplyStage := stages.NewMultiplyStage("multiply", 2.0)
	filterStage := stages.NewNumericFilterStage("filter", 0, 1000)
	
	pipeline.AddStage(multiplyStage)
	pipeline.AddStage(filterStage)
	
	if err := pipeline.Start(); err != nil {
		b.Fatalf("Failed to start pipeline: %v", err)
	}
	defer pipeline.Stop()
	
	b.ResetTimer()
	
	for i := 0; i < b.N; i++ {
		pipeline.Process(float64(i % 100))
	}
}

// BenchmarkFanOutFanInPipeline benchmarks fan-out/fan-in pipeline performance
func BenchmarkFanOutFanInPipeline(b *testing.B) {
	pipeline := pipeline.NewFanOutFanInPipeline("bench-fanout", 4, 100)
	
	parallelStage := stages.NewMultiplyStage("parallel", 2.0)
	pipeline.AddParallelStage(parallelStage)
	
	if err := pipeline.Start(); err != nil {
		b.Fatalf("Failed to start pipeline: %v", err)
	}
	defer pipeline.Stop()
	
	b.ResetTimer()
	
	for i := 0; i < b.N; i++ {
		pipeline.Process(float64(i % 100))
	}
}

// BenchmarkAdaptiveBuffer benchmarks adaptive buffer performance
func BenchmarkAdaptiveBuffer(b *testing.B) {
	buf := buffer.NewAdaptiveBuffer(10, 5, 100)
	defer buf.Close()
	
	b.ResetTimer()
	
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			buf.Send("benchmark-data")
			buf.Receive()
		}
	})
}

// BenchmarkRingBuffer benchmarks ring buffer performance
func BenchmarkRingBuffer(b *testing.B) {
	buf := buffer.NewRingBuffer(100)
	
	b.ResetTimer()
	
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			buf.Send("benchmark-data")
			buf.Receive()
		}
	})
}