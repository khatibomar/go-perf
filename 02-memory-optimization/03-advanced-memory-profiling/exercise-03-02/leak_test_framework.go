package main

import (
	"context"
	"fmt"
	"runtime"
	"strings"
	"sync"
	"time"
)

// LeakTestFramework provides automated testing for memory leaks
type LeakTestFramework struct {
	detector  *LeakDetector
	simulator *LeakSimulator
	testSuite *LeakTestSuite
	results   []*LeakTestResult
	mu        sync.RWMutex
}

// LeakTestSuite contains a collection of leak tests
type LeakTestSuite struct {
	Name        string
	Description string
	Tests       []*LeakTest
	SetupFunc   func() error
	TeardownFunc func() error
}

// LeakTest represents a single leak test case
type LeakTest struct {
	Name        string
	Description string
	TestFunc    func(*LeakTestContext) *LeakTestResult
	Timeout     time.Duration
	Expected    LeakTestExpectation
}

// LeakTestContext provides context for leak tests
type LeakTestContext struct {
	Detector  *LeakDetector
	Simulator *LeakSimulator
	StartTime time.Time
	Timeout   time.Duration
	Cancel    context.CancelFunc
	Expected  LeakTestExpectation
}

// LeakTestExpectation defines what the test expects to find
type LeakTestExpectation struct {
	ShouldDetectLeak bool
	ExpectedLeakType LeakType
	MaxDetectionTime time.Duration
	MinSeverity      LeakSeverity
}

// LeakTestResult contains the results of a leak test
type LeakTestResult struct {
	TestName     string
	Passed       bool
	Duration     time.Duration
	DetectedLeaks []*LeakDetection
	Expected     LeakTestExpectation
	Actual       LeakTestActual
	ErrorMessage string
	Metrics      map[string]interface{}
}

// LeakTestActual contains what was actually detected
type LeakTestActual struct {
	LeakDetected     bool
	DetectedLeakType LeakType
	DetectionTime    time.Duration
	Severity         LeakSeverity
}

// NewLeakTestFramework creates a new leak test framework
func NewLeakTestFramework() *LeakTestFramework {
	return &LeakTestFramework{
		detector:  NewLeakDetector(DefaultLeakDetectorConfig()),
		simulator: NewLeakSimulator(),
		results:   make([]*LeakTestResult, 0),
	}
}

// CreateDefaultTestSuite creates a comprehensive test suite
func (ltf *LeakTestFramework) CreateDefaultTestSuite() *LeakTestSuite {
	return &LeakTestSuite{
		Name:        "Comprehensive Leak Detection Tests",
		Description: "Tests various types of memory leaks and detection capabilities",
		Tests: []*LeakTest{
			{
				Name:        "GoroutineLeakDetection",
				Description: "Test detection of goroutine leaks",
				TestFunc:    ltf.testGoroutineLeakDetection,
				Timeout:     2 * time.Minute,
				Expected: LeakTestExpectation{
					ShouldDetectLeak: true,
					ExpectedLeakType: LeakTypeGoroutine,
					MaxDetectionTime: 90 * time.Second,
					MinSeverity:      LeakSeverityMedium,
				},
			},
			{
				Name:        "MemoryLeakDetection",
				Description: "Test detection of memory leaks",
				TestFunc:    ltf.testMemoryLeakDetection,
				Timeout:     2 * time.Minute,
				Expected: LeakTestExpectation{
					ShouldDetectLeak: true,
					ExpectedLeakType: LeakTypeMemory,
					MaxDetectionTime: 90 * time.Second,
					MinSeverity:      LeakSeverityMedium,
				},
			},
			{
				Name:        "HeapObjectLeakDetection",
				Description: "Test detection of heap object accumulation",
				TestFunc:    ltf.testHeapObjectLeakDetection,
				Timeout:     2 * time.Minute,
				Expected: LeakTestExpectation{
					ShouldDetectLeak: true,
					ExpectedLeakType: LeakTypeHeapObject,
					MaxDetectionTime: 90 * time.Second,
					MinSeverity:      LeakSeverityMedium,
				},
			},
			{
				Name:        "GCPressureDetection",
				Description: "Test detection of GC pressure issues",
				TestFunc:    ltf.testGCPressureDetection,
				Timeout:     2 * time.Minute,
				Expected: LeakTestExpectation{
					ShouldDetectLeak: true,
					ExpectedLeakType: LeakTypeGCPressure,
					MaxDetectionTime: 90 * time.Second,
					MinSeverity:      LeakSeverityMedium,
				},
			},
			{
				Name:        "FalsePositiveTest",
				Description: "Test that normal memory usage doesn't trigger false positives",
				TestFunc:    ltf.testFalsePositives,
				Timeout:     1 * time.Minute,
				Expected: LeakTestExpectation{
					ShouldDetectLeak: false,
					MaxDetectionTime: 60 * time.Second,
				},
			},
			{
				Name:        "MultipleLeakDetection",
				Description: "Test detection of multiple simultaneous leaks",
				TestFunc:    ltf.testMultipleLeakDetection,
				Timeout:     3 * time.Minute,
				Expected: LeakTestExpectation{
					ShouldDetectLeak: true,
					MaxDetectionTime: 120 * time.Second,
					MinSeverity:      LeakSeverityMedium,
				},
			},
		},
		SetupFunc: func() error {
			// Force GC to start with clean state
			runtime.GC()
			runtime.GC()
			return nil
		},
		TeardownFunc: func() error {
			// Clean up any leaked resources
			runtime.GC()
			return nil
		},
	}
}

// testGoroutineLeakDetection tests goroutine leak detection
func (ltf *LeakTestFramework) testGoroutineLeakDetection(ctx *LeakTestContext) *LeakTestResult {
	result := &LeakTestResult{
		TestName: "GoroutineLeakDetection",
		Metrics:  make(map[string]interface{}),
	}
	
	// Start leak detector
	ctx.Detector.Start()
	defer ctx.Detector.Stop()
	
	// Start goroutine leak simulation
	config := &LeakSimulationConfig{
		Duration:    ctx.Timeout - 30*time.Second,
		Intensity:   5,
		AutoCleanup: false,
	}
	
	leakID, err := ctx.Simulator.StartGoroutineLeak(config)
	if err != nil {
		result.ErrorMessage = fmt.Sprintf("Failed to start goroutine leak: %v", err)
		return result
	}
	
	defer ctx.Simulator.StopLeak(leakID)
	
	// Wait for detection
	detectionStart := time.Now()
	detected := false
	
	for time.Since(detectionStart) < ctx.Timeout-30*time.Second {
		report := ctx.Detector.GenerateReport()
		if len(report.Detections) > 0 {
			for _, detection := range report.Detections {
				if detection.Type == LeakTypeGoroutine {
					result.DetectedLeaks = append(result.DetectedLeaks, &detection)
					result.Actual.LeakDetected = true
					result.Actual.DetectedLeakType = detection.Type
					result.Actual.DetectionTime = time.Since(detectionStart)
					result.Actual.Severity = detection.Severity
					detected = true
					break
				}
			}
			if detected {
				break
			}
		}
		time.Sleep(5 * time.Second)
	}
	
	result.Duration = time.Since(ctx.StartTime)
	result.Passed = result.Actual.LeakDetected && 
		result.Actual.DetectedLeakType == LeakTypeGoroutine &&
		result.Actual.DetectionTime <= ctx.Expected.MaxDetectionTime
	
	return result
}

// testMemoryLeakDetection tests memory leak detection
func (ltf *LeakTestFramework) testMemoryLeakDetection(ctx *LeakTestContext) *LeakTestResult {
	result := &LeakTestResult{
		TestName: "MemoryLeakDetection",
		Metrics:  make(map[string]interface{}),
	}
	
	// Start leak detector
	ctx.Detector.Start()
	defer ctx.Detector.Stop()
	
	// Start memory leak simulation
	config := &LeakSimulationConfig{
		Duration:    ctx.Timeout - 30*time.Second,
		Intensity:   7,
		AutoCleanup: false,
	}
	
	leakID, err := ctx.Simulator.StartMemoryLeak(config)
	if err != nil {
		result.ErrorMessage = fmt.Sprintf("Failed to start memory leak: %v", err)
		return result
	}
	
	defer ctx.Simulator.StopLeak(leakID)
	
	// Wait for detection
	detectionStart := time.Now()
	detected := false
	
	for time.Since(detectionStart) < ctx.Timeout-30*time.Second {
		report := ctx.Detector.GenerateReport()
		if len(report.Detections) > 0 {
			for _, detection := range report.Detections {
				if detection.Type == LeakTypeMemory {
					result.DetectedLeaks = append(result.DetectedLeaks, &detection)
					result.Actual.LeakDetected = true
					result.Actual.DetectedLeakType = detection.Type
					result.Actual.DetectionTime = time.Since(detectionStart)
					result.Actual.Severity = detection.Severity
					detected = true
					break
				}
			}
			if detected {
				break
			}
		}
		time.Sleep(5 * time.Second)
	}
	
	result.Duration = time.Since(ctx.StartTime)
	result.Passed = result.Actual.LeakDetected && 
		result.Actual.DetectedLeakType == LeakTypeMemory &&
		result.Actual.DetectionTime <= ctx.Expected.MaxDetectionTime
	
	return result
}

// testHeapObjectLeakDetection tests heap object leak detection
func (ltf *LeakTestFramework) testHeapObjectLeakDetection(ctx *LeakTestContext) *LeakTestResult {
	result := &LeakTestResult{
		TestName: "HeapObjectLeakDetection",
		Metrics:  make(map[string]interface{}),
	}
	
	// Start leak detector
	ctx.Detector.Start()
	defer ctx.Detector.Stop()
	
	// Simulate heap object accumulation
	go func() {
		var objects []interface{}
		for i := 0; i < 10000; i++ {
			objects = append(objects, make([]byte, 1024))
			time.Sleep(10 * time.Millisecond)
		}
		// Keep objects alive
		time.Sleep(ctx.Timeout)
		_ = objects
	}()
	
	// Wait for detection
	detectionStart := time.Now()
	detected := false
	
	for time.Since(detectionStart) < ctx.Timeout-30*time.Second {
		report := ctx.Detector.GenerateReport()
		if len(report.Detections) > 0 {
			for _, detection := range report.Detections {
				if detection.Type == LeakTypeHeapObject {
					result.DetectedLeaks = append(result.DetectedLeaks, &detection)
					result.Actual.LeakDetected = true
					result.Actual.DetectedLeakType = detection.Type
					result.Actual.DetectionTime = time.Since(detectionStart)
					result.Actual.Severity = detection.Severity
					detected = true
					break
				}
			}
			if detected {
				break
			}
		}
		time.Sleep(5 * time.Second)
	}
	
	result.Duration = time.Since(ctx.StartTime)
	result.Passed = result.Actual.LeakDetected && 
		result.Actual.DetectedLeakType == LeakTypeHeapObject &&
		result.Actual.DetectionTime <= ctx.Expected.MaxDetectionTime
	
	return result
}

// testGCPressureDetection tests GC pressure detection
func (ltf *LeakTestFramework) testGCPressureDetection(ctx *LeakTestContext) *LeakTestResult {
	result := &LeakTestResult{
		TestName: "GCPressureDetection",
		Metrics:  make(map[string]interface{}),
	}
	
	// Start leak detector
	ctx.Detector.Start()
	defer ctx.Detector.Stop()
	
	// Create GC pressure by rapidly allocating and releasing memory
	go func() {
		for i := 0; i < 1000; i++ {
			data := make([][]byte, 1000)
			for j := range data {
				data[j] = make([]byte, 1024)
			}
			// Force GC
			runtime.GC()
			time.Sleep(10 * time.Millisecond)
		}
	}()
	
	// Wait for detection
	detectionStart := time.Now()
	detected := false
	
	for time.Since(detectionStart) < ctx.Timeout-30*time.Second {
		report := ctx.Detector.GenerateReport()
		if len(report.Detections) > 0 {
			for _, detection := range report.Detections {
				if detection.Type == LeakTypeGCPressure {
					result.DetectedLeaks = append(result.DetectedLeaks, &detection)
					result.Actual.LeakDetected = true
					result.Actual.DetectedLeakType = detection.Type
					result.Actual.DetectionTime = time.Since(detectionStart)
					result.Actual.Severity = detection.Severity
					detected = true
					break
				}
			}
			if detected {
				break
			}
		}
		time.Sleep(5 * time.Second)
	}
	
	result.Duration = time.Since(ctx.StartTime)
	result.Passed = result.Actual.LeakDetected && 
		result.Actual.DetectedLeakType == LeakTypeGCPressure &&
		result.Actual.DetectionTime <= ctx.Expected.MaxDetectionTime
	
	return result
}

// testFalsePositives tests that normal usage doesn't trigger false positives
func (ltf *LeakTestFramework) testFalsePositives(ctx *LeakTestContext) *LeakTestResult {
	result := &LeakTestResult{
		TestName: "FalsePositiveTest",
		Metrics:  make(map[string]interface{}),
	}
	
	// Start leak detector
	ctx.Detector.Start()
	defer ctx.Detector.Stop()
	
	// Simulate normal memory usage patterns
	go func() {
		for i := 0; i < 100; i++ {
			data := make([]byte, 1024)
			_ = data
			time.Sleep(100 * time.Millisecond)
		}
	}()
	
	// Wait and check for false positives
	time.Sleep(ctx.Timeout - 10*time.Second)
	
	report := ctx.Detector.GenerateReport()
		// Convert []LeakDetection to []*LeakDetection
		for i := range report.Detections {
			result.DetectedLeaks = append(result.DetectedLeaks, &report.Detections[i])
		}
		result.Actual.LeakDetected = len(report.Detections) > 0
	
	result.Duration = time.Since(ctx.StartTime)
	result.Passed = !result.Actual.LeakDetected // Should NOT detect any leaks
	
	return result
}

// testMultipleLeakDetection tests detection of multiple simultaneous leaks
func (ltf *LeakTestFramework) testMultipleLeakDetection(ctx *LeakTestContext) *LeakTestResult {
	result := &LeakTestResult{
		TestName: "MultipleLeakDetection",
		Metrics:  make(map[string]interface{}),
	}
	
	// Start leak detector
	ctx.Detector.Start()
	defer ctx.Detector.Stop()
	
	// Start multiple types of leaks
	config := &LeakSimulationConfig{
		Duration:    ctx.Timeout - 30*time.Second,
		Intensity:   3,
		AutoCleanup: false,
	}
	
	goroutineLeakID, _ := ctx.Simulator.StartGoroutineLeak(config)
	memoryLeakID, _ := ctx.Simulator.StartMemoryLeak(config)
	
	defer func() {
		ctx.Simulator.StopLeak(goroutineLeakID)
		ctx.Simulator.StopLeak(memoryLeakID)
	}()
	
	// Wait for detection
	detectionStart := time.Now()
	detectedTypes := make(map[LeakType]bool)
	
	for time.Since(detectionStart) < ctx.Timeout-30*time.Second {
		report := ctx.Detector.GenerateReport()
		if len(report.Detections) > 0 {
			for _, detection := range report.Detections {
				if !detectedTypes[detection.Type] {
					result.DetectedLeaks = append(result.DetectedLeaks, &detection)
					detectedTypes[detection.Type] = true
					result.Actual.LeakDetected = true
					result.Actual.DetectionTime = time.Since(detectionStart)
				}
			}
			
			// Check if we detected both types
			if detectedTypes[LeakTypeGoroutine] && detectedTypes[LeakTypeMemory] {
				break
			}
		}
		time.Sleep(5 * time.Second)
	}
	
	result.Duration = time.Since(ctx.StartTime)
	result.Passed = len(detectedTypes) >= 2 && 
		result.Actual.DetectionTime <= ctx.Expected.MaxDetectionTime
	
	return result
}

// RunTestSuite executes all tests in the test suite
func (ltf *LeakTestFramework) RunTestSuite(suite *LeakTestSuite) ([]*LeakTestResult, error) {
	ltf.mu.Lock()
	defer ltf.mu.Unlock()
	
	fmt.Printf("Running test suite: %s\n", suite.Name)
	fmt.Printf("Description: %s\n", suite.Description)
	fmt.Printf("Number of tests: %d\n\n", len(suite.Tests))
	
	// Run setup
	if suite.SetupFunc != nil {
		if err := suite.SetupFunc(); err != nil {
			return nil, fmt.Errorf("setup failed: %v", err)
		}
	}
	
	// Run teardown at the end
	defer func() {
		if suite.TeardownFunc != nil {
			suite.TeardownFunc()
		}
	}()
	
	results := make([]*LeakTestResult, 0, len(suite.Tests))
	
	for i, test := range suite.Tests {
		fmt.Printf("Running test %d/%d: %s\n", i+1, len(suite.Tests), test.Name)
		
		// Create test context
		_, cancel := context.WithTimeout(context.Background(), test.Timeout)
		testCtx := &LeakTestContext{
			Detector:  ltf.detector,
			Simulator: ltf.simulator,
			StartTime: time.Now(),
			Timeout:   test.Timeout,
			Cancel:    cancel,
		}
		testCtx.Expected = test.Expected
		
		// Run the test
		result := test.TestFunc(testCtx)
		result.Expected = test.Expected
		
		cancel()
		
		// Add to results
		results = append(results, result)
		ltf.results = append(ltf.results, result)
		
		// Print result
		status := "PASS"
		if !result.Passed {
			status = "FAIL"
		}
		fmt.Printf("  Result: %s (Duration: %v)\n", status, result.Duration)
		if result.ErrorMessage != "" {
			fmt.Printf("  Error: %s\n", result.ErrorMessage)
		}
		fmt.Println()
		
		// Clean up between tests
		ltf.simulator.StopAllLeaks()
		ltf.simulator.CleanupLeakedResources()
		runtime.GC()
		time.Sleep(2 * time.Second)
	}
	
	return results, nil
}

// GenerateTestReport generates a comprehensive test report
func (ltf *LeakTestFramework) GenerateTestReport(results []*LeakTestResult) *LeakTestReport {
	report := &LeakTestReport{
		Timestamp:    time.Now(),
		TotalTests:   len(results),
		PassedTests:  0,
		FailedTests:  0,
		TestResults:  results,
		Summary:      make(map[string]interface{}),
	}
	
	totalDuration := time.Duration(0)
	for _, result := range results {
		if result.Passed {
			report.PassedTests++
		} else {
			report.FailedTests++
		}
		totalDuration += result.Duration
	}
	
	report.PassRate = float64(report.PassedTests) / float64(report.TotalTests) * 100
	report.TotalDuration = totalDuration
	
	report.Summary["pass_rate"] = report.PassRate
	report.Summary["total_duration"] = totalDuration.String()
	report.Summary["average_test_duration"] = (totalDuration / time.Duration(len(results))).String()
	
	return report
}

// LeakTestReport contains comprehensive test results
type LeakTestReport struct {
	Timestamp     time.Time
	TotalTests    int
	PassedTests   int
	FailedTests   int
	PassRate      float64
	TotalDuration time.Duration
	TestResults   []*LeakTestResult
	Summary       map[string]interface{}
}

// PrintReport prints a formatted test report
func (ltr *LeakTestReport) PrintReport() {
	fmt.Println("\n" + strings.Repeat("=", 60))
	fmt.Println("LEAK DETECTION TEST REPORT")
	fmt.Println(strings.Repeat("=", 60))
	fmt.Printf("Timestamp: %s\n", ltr.Timestamp.Format(time.RFC3339))
	fmt.Printf("Total Tests: %d\n", ltr.TotalTests)
	fmt.Printf("Passed: %d\n", ltr.PassedTests)
	fmt.Printf("Failed: %d\n", ltr.FailedTests)
	fmt.Printf("Pass Rate: %.2f%%\n", ltr.PassRate)
	fmt.Printf("Total Duration: %v\n", ltr.TotalDuration)
	fmt.Println()
	
	fmt.Println("TEST DETAILS:")
	fmt.Println(strings.Repeat("-", 60))
	
	for _, result := range ltr.TestResults {
		status := "PASS"
		if !result.Passed {
			status = "FAIL"
		}
		
		fmt.Printf("Test: %s [%s]\n", result.TestName, status)
		fmt.Printf("  Duration: %v\n", result.Duration)
		fmt.Printf("  Detected Leaks: %d\n", len(result.DetectedLeaks))
		
		if result.Actual.LeakDetected {
			fmt.Printf("  Leak Type: %s\n", result.Actual.DetectedLeakType)
			fmt.Printf("  Detection Time: %v\n", result.Actual.DetectionTime)
			fmt.Printf("  Severity: %s\n", result.Actual.Severity)
		}
		
		if result.ErrorMessage != "" {
			fmt.Printf("  Error: %s\n", result.ErrorMessage)
		}
		fmt.Println()
	}
	
	fmt.Println(strings.Repeat("=", 60))
}

// LeakTestFrameworkExample demonstrates how to use the leak test framework
func LeakTestFrameworkExample() {
	// This is an example function, not meant to be called from main
	framework := NewLeakTestFramework()
	testSuite := framework.CreateDefaultTestSuite()
	
	// Run the test suite
	results, err := framework.RunTestSuite(testSuite)
	if err != nil {
		fmt.Printf("Test suite failed: %v\n", err)
		return
	}
	
	// Generate and print report
	report := framework.GenerateTestReport(results)
	report.PrintReport()
}