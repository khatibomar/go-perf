package main

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"runtime"
	"strings"
	"testing"
	"time"

	"exercise-03-01/profiling"
)

func TestApplication_HTTPEndpoints(t *testing.T) {
	// Enable profiling
	runtime.SetBlockProfileRate(1)
	runtime.SetMutexProfileFraction(1)

	// Create test application
	app, err := createTestApplication()
	if err != nil {
		t.Fatalf("Failed to create test application: %v", err)
	}
	defer app.Shutdown()

	// Create test server
	mux := http.NewServeMux()
	app.profiler.RegisterHTTPHandlers(mux)
	mux.HandleFunc("/", app.handleHome)
	mux.HandleFunc("/health", app.handleHealth)
	mux.HandleFunc("/workload", app.handleWorkload)
	mux.HandleFunc("/stats", app.handleStats)

	server := httptest.NewServer(mux)
	defer server.Close()

	t.Run("Home Page", func(t *testing.T) {
		resp, err := http.Get(server.URL + "/")
		if err != nil {
			t.Fatalf("Failed to get home page: %v", err)
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			t.Errorf("Expected status 200, got %d", resp.StatusCode)
		}

		if !strings.Contains(resp.Header.Get("Content-Type"), "text/html") {
			t.Error("Expected HTML content type")
		}
	})

	t.Run("Health Check", func(t *testing.T) {
		resp, err := http.Get(server.URL + "/health")
		if err != nil {
			t.Fatalf("Failed to get health: %v", err)
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			t.Errorf("Expected status 200, got %d", resp.StatusCode)
		}

		var health map[string]interface{}
		if err := json.NewDecoder(resp.Body).Decode(&health); err != nil {
			t.Errorf("Failed to decode health response: %v", err)
		}

		if health["status"] != "healthy" {
			t.Errorf("Expected healthy status, got %v", health["status"])
		}
	})

	t.Run("Runtime Stats", func(t *testing.T) {
		resp, err := http.Get(server.URL + "/stats")
		if err != nil {
			t.Fatalf("Failed to get stats: %v", err)
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			t.Errorf("Expected status 200, got %d", resp.StatusCode)
		}

		var stats map[string]interface{}
		if err := json.NewDecoder(resp.Body).Decode(&stats); err != nil {
			t.Errorf("Failed to decode stats response: %v", err)
		}

		// Check required fields
		requiredFields := []string{"timestamp", "goroutines", "heap_alloc", "gc_runs"}
		for _, field := range requiredFields {
			if _, exists := stats[field]; !exists {
				t.Errorf("Missing required field: %s", field)
			}
		}
	})

	t.Run("Workload Control", func(t *testing.T) {
		// Test GET request
		resp, err := http.Get(server.URL + "/workload")
		if err != nil {
			t.Fatalf("Failed to get workload status: %v", err)
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			t.Errorf("Expected status 200, got %d", resp.StatusCode)
		}

		var status map[string]interface{}
		if err := json.NewDecoder(resp.Body).Decode(&status); err != nil {
			t.Errorf("Failed to decode workload response: %v", err)
		}

		if _, exists := status["running"]; !exists {
			t.Error("Missing 'running' field in workload status")
		}
	})
}

func TestApplication_ProfilingEndpoints(t *testing.T) {
	// Create test application
	app, err := createTestApplication()
	if err != nil {
		t.Fatalf("Failed to create test application: %v", err)
	}
	defer app.Shutdown()

	// Create test server
	mux := http.NewServeMux()
	app.profiler.RegisterHTTPHandlers(mux)

	server := httptest.NewServer(mux)
	defer server.Close()

	t.Run("Start Profile", func(t *testing.T) {
		// Start CPU profile
		resp, err := http.Post(server.URL+"/debug/pprof/start?type=cpu&duration=2s", "application/json", nil)
		if err != nil {
			t.Fatalf("Failed to start profile: %v", err)
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			t.Errorf("Expected status 200, got %d", resp.StatusCode)
		}

		var session map[string]interface{}
		if err := json.NewDecoder(resp.Body).Decode(&session); err != nil {
			t.Errorf("Failed to decode session response: %v", err)
		}

		if session["Type"] != "cpu" {
			t.Errorf("Expected profile type 'cpu', got %v", session["Type"])
		}

		if session["Status"] != "running" {
			t.Errorf("Expected status 'running', got %v", session["Status"])
		}
	})

	t.Run("Profile Status", func(t *testing.T) {
		resp, err := http.Get(server.URL + "/debug/pprof/status")
		if err != nil {
			t.Fatalf("Failed to get profile status: %v", err)
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			t.Errorf("Expected status 200, got %d", resp.StatusCode)
		}

		var profiles map[string]interface{}
		if err := json.NewDecoder(resp.Body).Decode(&profiles); err != nil {
			t.Errorf("Failed to decode profiles response: %v", err)
		}

		// Should have the CPU profile we started
		if _, exists := profiles["cpu"]; !exists {
			t.Error("Expected CPU profile to be active")
		}
	})

	t.Run("Stop Profile", func(t *testing.T) {
		// Stop CPU profile
		resp, err := http.Post(server.URL+"/debug/pprof/stop?type=cpu", "application/json", nil)
		if err != nil {
			t.Fatalf("Failed to stop profile: %v", err)
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			t.Errorf("Expected status 200, got %d", resp.StatusCode)
		}

		// Verify profile is no longer active
		resp, err = http.Get(server.URL + "/debug/pprof/status")
		if err != nil {
			t.Fatalf("Failed to get profile status: %v", err)
		}
		defer resp.Body.Close()

		var profiles map[string]interface{}
		if err := json.NewDecoder(resp.Body).Decode(&profiles); err != nil {
			t.Errorf("Failed to decode profiles response: %v", err)
		}

		if _, exists := profiles["cpu"]; exists {
			t.Error("CPU profile should no longer be active")
		}
	})

	t.Run("List Profiles", func(t *testing.T) {
		// Start and complete a memory profile
		resp, err := http.Post(server.URL+"/debug/pprof/start?type=mem&duration=1s", "application/json", nil)
		if err != nil {
			t.Fatalf("Failed to start memory profile: %v", err)
		}
		resp.Body.Close()

		// Wait for completion
		time.Sleep(2 * time.Second)

		// List profiles
		resp, err = http.Get(server.URL + "/debug/pprof/list?type=mem")
		if err != nil {
			t.Fatalf("Failed to list profiles: %v", err)
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			t.Errorf("Expected status 200, got %d", resp.StatusCode)
		}

		var profiles []map[string]interface{}
		if err := json.NewDecoder(resp.Body).Decode(&profiles); err != nil {
			t.Errorf("Failed to decode profiles list: %v", err)
		}

		if len(profiles) == 0 {
			t.Error("Expected at least one memory profile")
		}
	})
}

func TestApplication_ErrorHandling(t *testing.T) {
	// Create test application
	app, err := createTestApplication()
	if err != nil {
		t.Fatalf("Failed to create test application: %v", err)
	}
	defer app.Shutdown()

	// Create test server
	mux := http.NewServeMux()
	app.profiler.RegisterHTTPHandlers(mux)

	server := httptest.NewServer(mux)
	defer server.Close()

	t.Run("Invalid Profile Type", func(t *testing.T) {
		resp, err := http.Post(server.URL+"/debug/pprof/start?type=invalid&duration=1s", "application/json", nil)
		if err != nil {
			t.Fatalf("Failed to make request: %v", err)
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusBadRequest {
			t.Errorf("Expected status 400, got %d", resp.StatusCode)
		}
	})

	t.Run("Wrong HTTP Method", func(t *testing.T) {
		resp, err := http.Get(server.URL + "/debug/pprof/start")
		if err != nil {
			t.Fatalf("Failed to make request: %v", err)
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusMethodNotAllowed {
			t.Errorf("Expected status 405, got %d", resp.StatusCode)
		}
	})

	t.Run("Stop Non-existent Profile", func(t *testing.T) {
		resp, err := http.Post(server.URL+"/debug/pprof/stop?type=cpu", "application/json", nil)
		if err != nil {
			t.Fatalf("Failed to make request: %v", err)
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusBadRequest {
			t.Errorf("Expected status 400, got %d", resp.StatusCode)
		}
	})
}

func TestWorkload(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	workload := NewWorkload(ctx)

	// Test initial state
	if workload.IsRunning() {
		t.Error("Workload should not be running initially")
	}

	// Start workload
	workload.Start()

	if !workload.IsRunning() {
		t.Error("Workload should be running after start")
	}

	// Let it run for a bit
	time.Sleep(1 * time.Second)

	// Stop workload
	workload.Stop()

	if workload.IsRunning() {
		t.Error("Workload should not be running after stop")
	}
}

func TestApplication_Integration(t *testing.T) {
	// This test verifies the complete application flow
	app, err := createTestApplication()
	if err != nil {
		t.Fatalf("Failed to create test application: %v", err)
	}
	defer app.Shutdown()

	// Start workload to generate some activity
	app.workload = NewWorkload(app.ctx)
	app.workload.Start()
	defer app.workload.Stop()

	// Create test server
	mux := http.NewServeMux()
	app.profiler.RegisterHTTPHandlers(mux)
	mux.HandleFunc("/health", app.handleHealth)
	mux.HandleFunc("/stats", app.handleStats)

	server := httptest.NewServer(mux)
	defer server.Close()

	// Test complete profiling workflow
	t.Run("Complete Profiling Workflow", func(t *testing.T) {
		// 1. Check initial health
		resp, err := http.Get(server.URL + "/health")
		if err != nil {
			t.Fatalf("Health check failed: %v", err)
		}
		resp.Body.Close()

		// 2. Start CPU profile
		resp, err = http.Post(server.URL+"/debug/pprof/start?type=cpu&duration=3s", "application/json", nil)
		if err != nil {
			t.Fatalf("Failed to start CPU profile: %v", err)
		}
		resp.Body.Close()

		// 3. Start memory profile
		resp, err = http.Post(server.URL+"/debug/pprof/start?type=mem&duration=3s", "application/json", nil)
		if err != nil {
			t.Fatalf("Failed to start memory profile: %v", err)
		}
		resp.Body.Close()

		// 4. Check status
		resp, err = http.Get(server.URL + "/debug/pprof/status")
		if err != nil {
			t.Fatalf("Failed to get status: %v", err)
		}
		defer resp.Body.Close()

		var status map[string]interface{}
		if err := json.NewDecoder(resp.Body).Decode(&status); err != nil {
			t.Errorf("Failed to decode status: %v", err)
		}

		// Should have both profiles
		if len(status) < 1 {
			t.Error("Expected at least one active profile")
		}

		// 5. Wait for profiles to complete
		time.Sleep(4 * time.Second)

		// 6. Check that profiles are stored
		resp, err = http.Get(server.URL + "/debug/pprof/list")
		if err != nil {
			t.Fatalf("Failed to list profiles: %v", err)
		}
		defer resp.Body.Close()

		var profiles []map[string]interface{}
		if err := json.NewDecoder(resp.Body).Decode(&profiles); err != nil {
			t.Errorf("Failed to decode profiles: %v", err)
		}

		if len(profiles) == 0 {
			t.Error("Expected stored profiles")
		}

		// 7. Check runtime stats
		resp, err = http.Get(server.URL + "/stats")
		if err != nil {
			t.Fatalf("Failed to get stats: %v", err)
		}
		defer resp.Body.Close()

		var stats map[string]interface{}
		if err := json.NewDecoder(resp.Body).Decode(&stats); err != nil {
			t.Errorf("Failed to decode stats: %v", err)
		}

		// Should have reasonable goroutine count
		if goroutines, ok := stats["goroutines"].(float64); ok {
			if goroutines < 1 || goroutines > 1000 {
				t.Errorf("Unexpected goroutine count: %v", goroutines)
			}
		}
	})
}

// Helper function to create test application
func createTestApplication() (*Application, error) {
	// Create temporary storage
	tempDir := fmt.Sprintf("/tmp/test-profiles-%d", time.Now().UnixNano())
	storage, err := profiling.NewFileStorage(tempDir)
	if err != nil {
		return nil, fmt.Errorf("failed to create storage: %w", err)
	}

	// Configure profiler for testing
	config := profiling.ProfilerConfig{
		EnableContinuous: false, // Disable for testing
		ProfileInterval:  10 * time.Second,
		ProfileDuration:  2 * time.Second,
		MaxConcurrent:    5,
		StoragePath:      tempDir,
		RetentionDays:    1,
	}

	// Create profiler
	profiler := profiling.NewLiveProfiler(config, storage)

	ctx, cancel := context.WithCancel(context.Background())

	return &Application{
		profiler: profiler,
		ctx:      ctx,
		cancel:   cancel,
	}, nil
}

// Benchmark tests for HTTP endpoints

func BenchmarkHTTPEndpoints(b *testing.B) {
	app, err := createTestApplication()
	if err != nil {
		b.Fatalf("Failed to create test application: %v", err)
	}
	defer app.Shutdown()

	mux := http.NewServeMux()
	mux.HandleFunc("/health", app.handleHealth)
	mux.HandleFunc("/stats", app.handleStats)

	server := httptest.NewServer(mux)
	defer server.Close()

	b.Run("Health", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			resp, err := http.Get(server.URL + "/health")
			if err != nil {
				b.Errorf("Health check failed: %v", err)
				continue
			}
			resp.Body.Close()
		}
	})

	b.Run("Stats", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			resp, err := http.Get(server.URL + "/stats")
			if err != nil {
				b.Errorf("Stats request failed: %v", err)
				continue
			}
			resp.Body.Close()
		}
	})
}