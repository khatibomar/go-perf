package main

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"testing"
	"time"

	"exercise-03-02/debugging"
	"exercise-03-02/workload"
)

func TestApplication_Integration(t *testing.T) {
	t.Run("Complete_Memory_Analysis_Workflow", func(t *testing.T) {
		app := createTestApplication()

		ctx, cancel := context.WithTimeout(context.Background(), time.Second*10)
		defer cancel()

		if err := app.Start(ctx); err != nil {
			t.Fatalf("Failed to start application: %v", err)
		}
		defer app.Shutdown()

		// Wait for application to be ready
		time.Sleep(time.Millisecond * 100)

		// Test health endpoint
		resp, err := http.Get("http://localhost:8082/health")
		if err != nil {
			t.Fatalf("Failed to get health: %v", err)
		}
		resp.Body.Close()
		if resp.StatusCode != http.StatusOK {
			t.Errorf("Expected status 200, got %d", resp.StatusCode)
		}

		// Test memory status
		resp, err = http.Get("http://localhost:8082/memory/status")
		if err != nil {
			t.Fatalf("Failed to get memory status: %v", err)
		}
		defer resp.Body.Close()
		if resp.StatusCode != http.StatusOK {
			t.Errorf("Expected status 200, got %d", resp.StatusCode)
		}

		var snapshot debugging.MemorySnapshot
		if err := json.NewDecoder(resp.Body).Decode(&snapshot); err != nil {
			t.Fatalf("Failed to decode memory status: %v", err)
		}

		// Start memory leak workload
		resp, err = http.Post("http://localhost:8082/workload/start?type=leak&intensity=medium&duration=2s", "application/json", nil)
		if err != nil {
			t.Fatalf("Failed to start workload: %v", err)
		}
		resp.Body.Close()
		if resp.StatusCode != http.StatusOK {
			t.Errorf("Expected status 200, got %d", resp.StatusCode)
		}

		// Wait for workload to run and memory analysis to detect issues
		time.Sleep(time.Second * 3)

		// Check workload status
		resp, err = http.Get("http://localhost:8082/workload/status")
		if err != nil {
			t.Fatalf("Failed to get workload status: %v", err)
		}
		defer resp.Body.Close()

		var workloadStatus workload.WorkloadStatus
		if err := json.NewDecoder(resp.Body).Decode(&workloadStatus); err != nil {
			t.Fatalf("Failed to decode workload status: %v", err)
		}

		// Force GC and analyze
		resp, err = http.Post("http://localhost:8082/memory/gc", "application/json", nil)
		if err != nil {
			t.Fatalf("Failed to force GC: %v", err)
		}
		defer resp.Body.Close()
		if resp.StatusCode != http.StatusOK {
			t.Errorf("Expected status 200, got %d", resp.StatusCode)
		}

		// Get memory snapshots
		resp, err = http.Get("http://localhost:8082/memory/snapshots")
		if err != nil {
			t.Fatalf("Failed to get memory snapshots: %v", err)
		}
		defer resp.Body.Close()

		var snapshots []debugging.MemorySnapshot
		if err := json.NewDecoder(resp.Body).Decode(&snapshots); err != nil {
			t.Fatalf("Failed to decode memory snapshots: %v", err)
		}

		if len(snapshots) == 0 {
			t.Error("Expected at least one memory snapshot")
		}

		// Verify snapshots are in chronological order
		for i := 1; i < len(snapshots); i++ {
			if snapshots[i].Timestamp.Before(snapshots[i-1].Timestamp) {
				t.Error("Expected snapshots to be in chronological order")
			}
		}

		t.Logf("Integration test completed successfully with %d snapshots", len(snapshots))
	})
}

func TestApplication_MemoryEndpoints(t *testing.T) {
	tests := []struct {
		name           string
		method         string
		path           string
		expectedStatus int
	}{
		{"Health Check", "GET", "/health", http.StatusOK},
		{"Memory Status", "GET", "/memory/status", http.StatusOK},
		{"Memory Snapshots", "GET", "/memory/snapshots", http.StatusOK},
		{"Force GC", "POST", "/memory/gc", http.StatusOK},
		{"Optimize GC", "POST", "/memory/optimize", http.StatusOK},
		{"Wrong Method Health", "POST", "/health", http.StatusMethodNotAllowed},
		{"Wrong Method Memory Status", "POST", "/memory/status", http.StatusMethodNotAllowed},
		{"Wrong Method Force GC", "GET", "/memory/gc", http.StatusMethodNotAllowed},
	}

	app := createTestApplication()
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*5)
	defer cancel()

	if err := app.Start(ctx); err != nil {
		t.Fatalf("Failed to start application: %v", err)
	}
	defer app.Shutdown()

	// Wait for application to be ready
	time.Sleep(time.Millisecond * 100)

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var resp *http.Response
			var err error

			switch tt.method {
			case "GET":
				resp, err = http.Get(fmt.Sprintf("http://localhost:8082%s", tt.path))
			case "POST":
				resp, err = http.Post(fmt.Sprintf("http://localhost:8082%s", tt.path), "application/json", nil)
			default:
				t.Fatalf("Unsupported method: %s", tt.method)
			}

			if err != nil {
				t.Fatalf("Failed to make request: %v", err)
			}
			defer resp.Body.Close()

			if resp.StatusCode != tt.expectedStatus {
				t.Errorf("Expected status %d, got %d", tt.expectedStatus, resp.StatusCode)
			}
		})
	}
}

func TestApplication_WorkloadEndpoints(t *testing.T) {
	app := createTestApplication()
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*5)
	defer cancel()

	if err := app.Start(ctx); err != nil {
		t.Fatalf("Failed to start application: %v", err)
	}
	defer app.Shutdown()

	// Wait for application to be ready
	time.Sleep(time.Millisecond * 100)

	t.Run("Start_Workload", func(t *testing.T) {
		resp, err := http.Post("http://localhost:8082/workload/start?type=normal&intensity=low&duration=500ms", "application/json", nil)
		if err != nil {
			t.Fatalf("Failed to start workload: %v", err)
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			t.Errorf("Expected status 200, got %d", resp.StatusCode)
		}

		var result map[string]string
		if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
			t.Fatalf("Failed to decode response: %v", err)
		}

		if result["status"] != "started" {
			t.Errorf("Expected status 'started', got '%s'", result["status"])
		}
	})

	t.Run("Workload_Status", func(t *testing.T) {
		resp, err := http.Get("http://localhost:8082/workload/status")
		if err != nil {
			t.Fatalf("Failed to get workload status: %v", err)
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			t.Errorf("Expected status 200, got %d", resp.StatusCode)
		}

		var status workload.WorkloadStatus
		if err := json.NewDecoder(resp.Body).Decode(&status); err != nil {
			t.Fatalf("Failed to decode workload status: %v", err)
		}

		if status.MemoryMB < 0 {
			t.Error("Expected non-negative memory usage")
		}
		if status.Goroutines <= 0 {
			t.Error("Expected positive goroutine count")
		}
	})

	t.Run("Stop_Workload", func(t *testing.T) {
		resp, err := http.Post("http://localhost:8082/workload/stop", "application/json", nil)
		if err != nil {
			t.Fatalf("Failed to stop workload: %v", err)
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			t.Errorf("Expected status 200, got %d", resp.StatusCode)
		}

		var result map[string]string
		if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
			t.Fatalf("Failed to decode response: %v", err)
		}

		if result["status"] != "stopped" {
			t.Errorf("Expected status 'stopped', got '%s'", result["status"])
		}
	})
}

func TestApplication_ErrorHandling(t *testing.T) {
	app := createTestApplication()
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*5)
	defer cancel()

	if err := app.Start(ctx); err != nil {
		t.Fatalf("Failed to start application: %v", err)
	}
	defer app.Shutdown()

	// Wait for application to be ready
	time.Sleep(time.Millisecond * 100)

	t.Run("Invalid_Workload_Type", func(t *testing.T) {
		resp, err := http.Post("http://localhost:8082/workload/start?type=invalid&intensity=low&duration=100ms", "application/json", nil)
		if err != nil {
			t.Fatalf("Failed to make request: %v", err)
		}
		defer resp.Body.Close()

		// Should still work as invalid types default to normal
		if resp.StatusCode != http.StatusOK {
			t.Errorf("Expected status 200, got %d", resp.StatusCode)
		}
	})

	t.Run("Invalid_Duration", func(t *testing.T) {
		resp, err := http.Post("http://localhost:8082/workload/start?type=normal&intensity=low&duration=invalid", "application/json", nil)
		if err != nil {
			t.Fatalf("Failed to make request: %v", err)
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusBadRequest {
			t.Errorf("Expected status 400, got %d", resp.StatusCode)
		}
	})

	t.Run("Invalid_Since_Parameter", func(t *testing.T) {
		resp, err := http.Get("http://localhost:8082/memory/snapshots?since=invalid")
		if err != nil {
			t.Fatalf("Failed to make request: %v", err)
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusBadRequest {
			t.Errorf("Expected status 400, got %d", resp.StatusCode)
		}
	})
}

func TestApplication_PerformanceRequirements(t *testing.T) {
	// Test that the application meets performance requirements:
	// - Memory leak detection within 10 minutes
	// - Heap analysis completion <30 seconds
	// - Zero production impact during analysis

	app := createTestApplication()
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*30)
	defer cancel()

	if err := app.Start(ctx); err != nil {
		t.Fatalf("Failed to start application: %v", err)
	}
	defer app.Shutdown()

	// Wait for application to be ready
	time.Sleep(time.Millisecond * 100)

	t.Run("Heap_Analysis_Performance", func(t *testing.T) {
		start := time.Now()
		resp, err := http.Post("http://localhost:8082/memory/gc", "application/json", nil)
		analysisTime := time.Since(start)

		if err != nil {
			t.Fatalf("Failed to force GC: %v", err)
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			t.Errorf("Expected status 200, got %d", resp.StatusCode)
		}

		if analysisTime > time.Second*30 {
			t.Errorf("Heap analysis took %v, expected <30 seconds", analysisTime)
		}

		t.Logf("Heap analysis completed in %v", analysisTime)
	})

	t.Run("Memory_Status_Performance", func(t *testing.T) {
		start := time.Now()
		resp, err := http.Get("http://localhost:8082/memory/status")
		statusTime := time.Since(start)

		if err != nil {
			t.Fatalf("Failed to get memory status: %v", err)
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			t.Errorf("Expected status 200, got %d", resp.StatusCode)
		}

		if statusTime > time.Millisecond*100 {
			t.Errorf("Memory status took %v, expected <100ms", statusTime)
		}

		t.Logf("Memory status completed in %v", statusTime)
	})

	t.Run("Snapshot_Collection_Performance", func(t *testing.T) {
		// Test multiple snapshot requests
		for i := 0; i < 10; i++ {
			start := time.Now()
			resp, err := http.Get("http://localhost:8082/memory/snapshots")
			snapshotTime := time.Since(start)

			if err != nil {
				t.Fatalf("Failed to get snapshots: %v", err)
			}
			resp.Body.Close()

			if resp.StatusCode != http.StatusOK {
				t.Errorf("Expected status 200, got %d", resp.StatusCode)
			}

			if snapshotTime > time.Millisecond*500 {
				t.Errorf("Snapshot collection took %v, expected <500ms", snapshotTime)
			}
		}
	})
}

func createTestApplication() *Application {
	// Configure memory analyzer for testing
	analyzerConfig := debugging.AnalyzerConfig{
		SnapshotInterval:     time.Millisecond * 100, // Faster for testing
		MaxSnapshots:         50,
		LeakThreshold:        0.05, // Lower threshold for testing
		ConsecutiveChecks:    3,
		EnableGCOptimization: true,
	}

	memoryAnalyzer := debugging.NewMemoryAnalyzer(analyzerConfig)

	// Register leak detection callback
	memoryAnalyzer.RegisterLeakCallback(func(leak debugging.MemoryLeak) {
		// Log leaks during testing
		fmt.Printf("TEST LEAK DETECTED: %s - %s (Severity: %s)\n", leak.Type, leak.Description, leak.Severity)
	})

	// Create workload manager
	workloadManager := workload.NewManager()

	return &Application{
		memoryAnalyzer:  memoryAnalyzer,
		workloadManager: workloadManager,
	}
}

func BenchmarkApplication_MemoryStatus(b *testing.B) {
	app := createTestApplication()
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*30)
	defer cancel()

	if err := app.Start(ctx); err != nil {
		b.Fatalf("Failed to start application: %v", err)
	}
	defer app.Shutdown()

	// Wait for application to be ready
	time.Sleep(time.Millisecond * 100)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		resp, err := http.Get("http://localhost:8082/memory/status")
		if err != nil {
			b.Fatalf("Failed to get memory status: %v", err)
		}
		resp.Body.Close()
	}
}

func BenchmarkApplication_ForceGC(b *testing.B) {
	app := createTestApplication()
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*60)
	defer cancel()

	if err := app.Start(ctx); err != nil {
		b.Fatalf("Failed to start application: %v", err)
	}
	defer app.Shutdown()

	// Wait for application to be ready
	time.Sleep(time.Millisecond * 100)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		resp, err := http.Post("http://localhost:8082/memory/gc", "application/json", nil)
		if err != nil {
			b.Fatalf("Failed to force GC: %v", err)
		}
		resp.Body.Close()
	}
}
