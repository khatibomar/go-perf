package main

import (
	"context"
	"fmt"
	"log"
	"math/rand"
	"sync"
	"time"

	"exercise-03-04/backpressure"
	"exercise-03-04/buffer"
	"exercise-03-04/pipeline"
	"exercise-03-04/stages"
)

func main() {
	fmt.Println("High-Performance Concurrent Pipeline Demo")
	fmt.Println("=========================================")
	
	// Run different pipeline demonstrations
	runLinearPipelineDemo()
	runFanOutFanInDemo()
	runDynamicPipelineDemo()
	runBufferDemo()
	runBackpressureDemo()
}

// runLinearPipelineDemo demonstrates a simple linear pipeline
func runLinearPipelineDemo() {
	fmt.Println("\n1. Linear Pipeline Demo")
	fmt.Println("------------------------")
	
	// Create pipeline
	pipeline := pipeline.NewLinearPipeline("demo-linear", 100)
	
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
		log.Fatalf("Failed to start pipeline: %v", err)
	}
	defer pipeline.Stop()
	
	// Process data
	go func() {
		for i := 0; i < 20; i++ {
			value := float64(rand.Intn(100))
			if err := pipeline.Process(value); err != nil {
				log.Printf("Error processing %f: %v", value, err)
			}
			time.Sleep(time.Millisecond * 50)
		}
	}()
	
	// Collect results
	resultCount := 0
	timeout := time.After(time.Second * 5)
	
loop:
	for {
		select {
		case result := <-pipeline.GetOutput():
			fmt.Printf("Output: %v\n", result)
			resultCount++
			if resultCount >= 10 {
				break loop
			}
		case err := <-pipeline.GetErrors():
			fmt.Printf("Error: %v\n", err)
		case <-timeout:
			fmt.Println("Timeout reached")
			break loop
		}
	}
	
	// Print metrics
	metrics := pipeline.GetMetrics()
	fmt.Printf("Processed: %d, Errors: %d, Throughput: %.2f/s\n",
		metrics.ProcessedCount, metrics.ErrorCount, metrics.Throughput)
}

// runFanOutFanInDemo demonstrates fan-out/fan-in pipeline
func runFanOutFanInDemo() {
	fmt.Println("\n2. Fan-Out/Fan-In Pipeline Demo")
	fmt.Println("-------------------------------")
	
	// Create pipeline with 4 parallel workers
	pipeline := pipeline.NewFanOutFanInPipeline("demo-fanout", 4, 50)
	
	// Add pre-processing stage
	preStage := stages.NewTransformStage("pre-process", func(data interface{}) (interface{}, error) {
		if num, ok := data.(int); ok {
			return float64(num), nil
		}
		return data, nil
	})
	pipeline.AddPreStage(preStage)
	
	// Add parallel processing stages
	parallelStage1 := stages.NewMultiplyStage("parallel-multiply", 3.0)
	parallelStage2 := stages.NewTransformStage("parallel-transform", func(data interface{}) (interface{}, error) {
		if num, ok := data.(float64); ok {
			// Simulate some processing time
			time.Sleep(time.Millisecond * 10)
			return num + 10, nil
		}
		return data, nil
	})
	
	pipeline.AddParallelStage(parallelStage1)
	pipeline.AddParallelStage(parallelStage2)
	
	// Add post-processing stage
	postStage := stages.NewTransformStage("post-process", func(data interface{}) (interface{}, error) {
		if num, ok := data.(float64); ok {
			return fmt.Sprintf("Final: %.2f", num), nil
		}
		return data, nil
	})
	pipeline.AddPostStage(postStage)
	
	// Start pipeline
	if err := pipeline.Start(); err != nil {
		log.Fatalf("Failed to start fan-out pipeline: %v", err)
	}
	defer pipeline.Stop()
	
	// Process data
	go func() {
		for i := 0; i < 30; i++ {
			value := rand.Intn(50)
			if err := pipeline.Process(value); err != nil {
				log.Printf("Error processing %d: %v", value, err)
			}
			time.Sleep(time.Millisecond * 20)
		}
	}()
	
	// Collect results
	resultCount := 0
	timeout := time.After(time.Second * 8)
	
loop2:
	for {
		select {
		case result := <-pipeline.GetOutput():
			fmt.Printf("Fan-Out Result: %v\n", result)
			resultCount++
			if resultCount >= 15 {
				break loop2
			}
		case err := <-pipeline.GetErrors():
			fmt.Printf("Fan-Out Error: %v\n", err)
		case <-timeout:
			fmt.Println("Fan-Out timeout reached")
			break loop2
		}
	}
	
	// Print metrics
	metrics := pipeline.GetMetrics()
	fmt.Printf("Fan-Out Processed: %d, Errors: %d, Throughput: %.2f/s\n",
		metrics.ProcessedCount, metrics.ErrorCount, metrics.Throughput)
}

// runDynamicPipelineDemo demonstrates dynamic pipeline modification
func runDynamicPipelineDemo() {
	fmt.Println("\n3. Dynamic Pipeline Demo")
	fmt.Println("-------------------------")
	
	// Create dynamic pipeline
	pipeline := pipeline.NewDynamicPipeline("demo-dynamic", 50)
	
	// Add initial stages
	stage1 := stages.NewTransformStage("input", func(data interface{}) (interface{}, error) {
		if num, ok := data.(int); ok {
			return float64(num * 2), nil
		}
		return data, nil
	})
	
	stage2 := stages.NewMultiplyStage("multiply", 1.5)
	
	stage3 := stages.NewTransformStage("output", func(data interface{}) (interface{}, error) {
		if num, ok := data.(float64); ok {
			return fmt.Sprintf("Dynamic: %.2f", num), nil
		}
		return data, nil
	})
	
	pipeline.AddStage(stage1)
	pipeline.AddStage(stage2)
	pipeline.AddStage(stage3)
	
	// Connect stages
	pipeline.ConnectStages("input", "multiply")
	pipeline.ConnectStages("multiply", "output")
	
	// Start pipeline
	if err := pipeline.Start(); err != nil {
		log.Fatalf("Failed to start dynamic pipeline: %v", err)
	}
	defer pipeline.Stop()
	
	// Process some data
	go func() {
		for i := 0; i < 20; i++ {
			value := rand.Intn(25)
			if err := pipeline.Process(value); err != nil {
				log.Printf("Error processing %d: %v", value, err)
			}
			time.Sleep(time.Millisecond * 30)
			
			// Dynamically add a filter stage after processing some data
			if i == 10 {
				fmt.Println("Adding filter stage dynamically...")
				filterStage := stages.NewNumericFilterStage("filter", 5, 50)
				pipeline.AddStage(filterStage)
				
				// Reconnect pipeline
				pipeline.DisconnectStages("multiply", "output")
				pipeline.ConnectStages("multiply", "filter")
				pipeline.ConnectStages("filter", "output")
			}
		}
	}()
	
	// Collect results
	resultCount := 0
	timeout := time.After(time.Second * 10)
	
loop3:
	for {
		select {
		case result := <-pipeline.GetOutput():
			fmt.Printf("Dynamic Result: %v\n", result)
			resultCount++
			if resultCount >= 12 {
				break loop3
			}
		case err := <-pipeline.GetErrors():
			fmt.Printf("Dynamic Error: %v\n", err)
		case <-timeout:
			fmt.Println("Dynamic timeout reached")
			break loop3
		}
	}
	
	// Print final configuration
	fmt.Printf("Final stages: %v\n", pipeline.GetStageNames())
	fmt.Printf("Connections: %v\n", pipeline.GetConnections())
	
	// Print metrics
	metrics := pipeline.GetMetrics()
	fmt.Printf("Dynamic Processed: %d, Errors: %d, Throughput: %.2f/s\n",
		metrics.ProcessedCount, metrics.ErrorCount, metrics.Throughput)
}

// runBufferDemo demonstrates adaptive buffer functionality
func runBufferDemo() {
	fmt.Println("\n4. Adaptive Buffer Demo")
	fmt.Println("------------------------")
	
	// Create adaptive buffer
	buf := buffer.NewAdaptiveBuffer(10, 5, 50)
	defer buf.Close()
	
	// Create ring buffer for comparison
	ringBuf := buffer.NewRingBuffer(20)
	
	var wg sync.WaitGroup
	
	// Producer for adaptive buffer
	wg.Add(1)
	go func() {
		defer wg.Done()
		for i := 0; i < 100; i++ {
			data := fmt.Sprintf("adaptive-item-%d", i)
			if !buf.Send(data) {
				fmt.Printf("Adaptive buffer full, dropped: %s\n", data)
			}
			time.Sleep(time.Millisecond * 5)
		}
	}()
	
	// Consumer for adaptive buffer
	wg.Add(1)
	go func() {
		defer wg.Done()
		received := 0
		for received < 80 {
			if data, ok := buf.Receive(); ok {
				fmt.Printf("Adaptive received: %v\n", data)
				received++
			}
			time.Sleep(time.Millisecond * 8)
		}
	}()
	
	// Producer for ring buffer
	wg.Add(1)
	go func() {
		defer wg.Done()
		for i := 0; i < 50; i++ {
			data := fmt.Sprintf("ring-item-%d", i)
			if !ringBuf.Send(data) {
				fmt.Printf("Ring buffer full, dropped: %s\n", data)
			}
			time.Sleep(time.Millisecond * 3)
		}
	}()
	
	// Consumer for ring buffer
	wg.Add(1)
	go func() {
		defer wg.Done()
		received := 0
		for received < 40 {
			if data, ok := ringBuf.Receive(); ok {
				fmt.Printf("Ring received: %v\n", data)
				received++
			}
			time.Sleep(time.Millisecond * 4)
		}
	}()
	
	wg.Wait()
	
	// Print buffer metrics
	metrics := buf.GetMetrics()
	fmt.Printf("Adaptive Buffer - Sent: %d, Received: %d, Dropped: %d, Resizes: %d\n",
		metrics.SendCount, metrics.ReceiveCount, metrics.DropCount, metrics.ResizeCount)
	fmt.Printf("Peak Capacity: %d, Avg Load Factor: %.2f\n",
		metrics.PeakCapacity, metrics.AvgLoadFactor)
	
	fmt.Printf("Ring Buffer - Size: %d, Capacity: %d\n",
		ringBuf.Size(), ringBuf.Capacity())
}

// runBackpressureDemo demonstrates backpressure control
func runBackpressureDemo() {
	fmt.Println("\n5. Backpressure Control Demo")
	fmt.Println("-----------------------------")
	
	// Create backpressure controller
	thresholds := &backpressure.Thresholds{
		LowWaterMark:    0.3,
		MediumWaterMark: 0.6,
		HighWaterMark:   0.8,
		CriticalMark:    0.95,
	}
	
	ctrl := backpressure.NewBackpressureController(thresholds)
	defer ctrl.Close()
	
	// Create action handlers
	throttleAction := backpressure.NewThrottleAction(10.0) // 10 requests per second
	dropAction := backpressure.NewDropAction(0.1)          // 10% drop rate initially
	rejectAction := backpressure.NewRejectAction()
	
	ctrl.AddActionHandler(throttleAction)
	ctrl.AddActionHandler(dropAction)
	ctrl.AddActionHandler(rejectAction)
	
	// Simulate varying load
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*5)
	defer cancel()
	
	var wg sync.WaitGroup
	
	// Load generator
	wg.Add(1)
	go func() {
		defer wg.Done()
		load := 0.0
		increment := 0.05
		
		for {
			select {
			case <-ctx.Done():
				return
			default:
				// Simulate load variation
				load += increment
				if load >= 1.0 || load <= 0.0 {
					increment = -increment
				}
				
				action := ctrl.CheckAndApply(load)
				fmt.Printf("Load: %.2f, Action: %v\n", load, actionToString(action))
				
				// Simulate request processing based on action
				switch action {
				case backpressure.ActionThrottle:
					if throttleAction.ShouldThrottle() {
						fmt.Println("  -> Request throttled")
					} else {
						fmt.Println("  -> Request processed")
					}
				case backpressure.ActionDrop:
					if dropAction.ShouldDrop() {
						fmt.Println("  -> Request dropped")
					} else {
						fmt.Println("  -> Request processed")
					}
				case backpressure.ActionReject:
					if rejectAction.ShouldReject() {
						fmt.Println("  -> Request rejected")
					}
				default:
					fmt.Println("  -> Request processed normally")
				}
				
				time.Sleep(time.Millisecond * 200)
			}
		}
	}()
	
	wg.Wait()
	
	// Print backpressure metrics
	metrics := ctrl.GetMetrics()
	fmt.Printf("Backpressure Metrics - Throttle: %d, Drop: %d, Block: %d, Reject: %d\n",
		metrics.ThrottleCount, metrics.DropCount, metrics.BlockCount, metrics.RejectCount)
	fmt.Printf("Avg Load: %.2f, Peak Load: %.2f\n", metrics.AvgLoad, metrics.PeakLoad)
	
	// Print action-specific stats
	dropped, total, dropRate := dropAction.GetDropStats()
	fmt.Printf("Drop Action - Dropped: %d/%d (%.2f%%)\n", dropped, total, dropRate*100)
	
	rejected, totalReject, rejectRate := rejectAction.GetRejectStats()
	fmt.Printf("Reject Action - Rejected: %d/%d (%.2f%%)\n", rejected, totalReject, rejectRate*100)
}

// actionToString converts action enum to string
func actionToString(action backpressure.Action) string {
	switch action {
	case backpressure.ActionNone:
		return "None"
	case backpressure.ActionThrottle:
		return "Throttle"
	case backpressure.ActionDrop:
		return "Drop"
	case backpressure.ActionBlock:
		return "Block"
	case backpressure.ActionReject:
		return "Reject"
	default:
		return "Unknown"
	}
}