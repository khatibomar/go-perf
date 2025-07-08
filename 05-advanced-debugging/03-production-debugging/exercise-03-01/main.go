package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"math/rand"
	"net/http"
	_ "net/http/pprof" // Enable default pprof endpoints
	"os"
	"os/signal"
	"runtime"
	"sync"
	"syscall"
	"time"

	"exercise-03-01/profiling"
)

// Application represents our sample application
type Application struct {
	profiler *profiling.LiveProfiler
	server   *http.Server
	workload *Workload
	ctx      context.Context
	cancel   context.CancelFunc
	wg       sync.WaitGroup
}

// Workload simulates application work to generate profiling data
type Workload struct {
	ctx     context.Context
	cancel  context.CancelFunc
	wg      sync.WaitGroup
	running bool
	mu      sync.RWMutex
}

func main() {
	// Enable profiling
	runtime.SetBlockProfileRate(1)
	runtime.SetMutexProfileFraction(1)

	app, err := NewApplication()
	if err != nil {
		log.Fatalf("Failed to create application: %v", err)
	}

	// Start the application
	if err := app.Start(); err != nil {
		log.Fatalf("Failed to start application: %v", err)
	}

	// Wait for shutdown signal
	app.WaitForShutdown()

	// Graceful shutdown
	if err := app.Shutdown(); err != nil {
		log.Printf("Error during shutdown: %v", err)
	}

	fmt.Println("Application stopped")
}

// NewApplication creates a new application instance
func NewApplication() (*Application, error) {
	// Create storage
	storage, err := profiling.NewFileStorage("./profiles")
	if err != nil {
		return nil, fmt.Errorf("failed to create storage: %w", err)
	}

	// Configure profiler
	config := profiling.ProfilerConfig{
		EnableContinuous: true,
		ProfileInterval:  30 * time.Second, // Profile every 30 seconds
		ProfileDuration:  10 * time.Second, // Each profile lasts 10 seconds
		MaxConcurrent:    3,                // Max 3 concurrent profiles
		StoragePath:      "./profiles",
		RetentionDays:    30, // Keep profiles for 30 days
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

// Start starts the application
func (app *Application) Start() error {
	fmt.Println("Starting Live Profiling Demo Application...")

	// Create workload simulation (but don't start it automatically)
	app.workload = NewWorkload(app.ctx)

	// Setup HTTP server with profiling endpoints
	mux := http.NewServeMux()

	// Register default pprof handlers
	mux.Handle("/debug/pprof/", http.DefaultServeMux)
	mux.Handle("/debug/pprof/cmdline", http.DefaultServeMux)
	mux.Handle("/debug/pprof/profile", http.DefaultServeMux)
	mux.Handle("/debug/pprof/symbol", http.DefaultServeMux)
	mux.Handle("/debug/pprof/trace", http.DefaultServeMux)

	// Register live profiler endpoints
	app.profiler.RegisterHTTPHandlers(mux)

	// Add application endpoints
	mux.HandleFunc("/", app.handleHome)
	mux.HandleFunc("/health", app.handleHealth)
	mux.HandleFunc("/workload", app.handleWorkload)
	mux.HandleFunc("/stats", app.handleStats)

	// Create HTTP server
	app.server = &http.Server{
		Addr:    ":8081",
		Handler: mux,
	}

	// Start HTTP server
	app.wg.Add(1)
	go func() {
		defer app.wg.Done()
		fmt.Println("HTTP server starting on :8081")
		if err := app.server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Printf("HTTP server error: %v", err)
		}
	}()

	fmt.Println("Application started successfully")
	fmt.Println("Available endpoints:")
	fmt.Println("  - http://localhost:8081/ - Home page")
	fmt.Println("  - http://localhost:8081/health - Health check")
	fmt.Println("  - http://localhost:8081/workload - Workload control")
	fmt.Println("  - http://localhost:8081/stats - Runtime statistics")
	fmt.Println("  - http://localhost:8081/debug/pprof/ - Standard pprof endpoints")
	fmt.Println("  - http://localhost:8081/debug/pprof/start - Start live profiling")
	fmt.Println("  - http://localhost:8081/debug/pprof/stop - Stop live profiling")
	fmt.Println("  - http://localhost:8081/debug/pprof/status - Profiling status")
	fmt.Println("  - http://localhost:8081/debug/pprof/list - List stored profiles")

	return nil
}

// WaitForShutdown waits for shutdown signal
func (app *Application) WaitForShutdown() {
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	select {
	case sig := <-sigChan:
		fmt.Printf("\nReceived signal: %v\n", sig)
	case <-app.ctx.Done():
		fmt.Println("\nContext cancelled")
	}
}

// Shutdown gracefully shuts down the application
func (app *Application) Shutdown() error {
	fmt.Println("Shutting down application...")

	// Cancel context
	app.cancel()

	// Shutdown HTTP server
	if app.server != nil {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		if err := app.server.Shutdown(ctx); err != nil {
			log.Printf("HTTP server shutdown error: %v", err)
		}
	}

	// Shutdown profiler
	if err := app.profiler.Shutdown(); err != nil {
		log.Printf("Profiler shutdown error: %v", err)
	}

	// Shutdown workload
	if app.workload != nil {
		app.workload.Stop()
	}

	// Wait for goroutines
	app.wg.Wait()

	return nil
}

// HTTP Handlers

func (app *Application) handleHome(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/html")
	fmt.Fprintf(w, `
<!DOCTYPE html>
<html>
<head>
    <title>Live Profiling Demo</title>
    <style>
        body { font-family: Arial, sans-serif; margin: 40px; }
        .endpoint { margin: 10px 0; }
        .endpoint a { text-decoration: none; color: #0066cc; }
        .endpoint a:hover { text-decoration: underline; }
        .section { margin: 20px 0; }
    </style>
</head>
<body>
    <h1>Live Profiling Demo Application</h1>
    
    <div class="section">
        <h2>Application Endpoints</h2>
        <div class="endpoint"><a href="/health">Health Check</a> - Application health status</div>
        <div class="endpoint"><a href="/workload">Workload Control</a> - Control simulation workload</div>
        <div class="endpoint"><a href="/stats">Runtime Stats</a> - Go runtime statistics</div>
    </div>
    
    <div class="section">
        <h2>Standard pprof Endpoints</h2>
        <div class="endpoint"><a href="/debug/pprof/">Profile Index</a> - Available profiles</div>
        <div class="endpoint"><a href="/debug/pprof/goroutine">Goroutines</a> - Current goroutines</div>
        <div class="endpoint"><a href="/debug/pprof/heap">Heap</a> - Memory heap profile</div>
        <div class="endpoint"><a href="/debug/pprof/profile?seconds=10">CPU Profile</a> - 10-second CPU profile</div>
    </div>
    
    <div class="section">
        <h2>Live Profiling Endpoints</h2>
        <div class="endpoint"><a href="/debug/pprof/status">Profiling Status</a> - Current profiling status</div>
        <div class="endpoint"><a href="/debug/pprof/list">Stored Profiles</a> - List of stored profiles</div>
        <div class="endpoint">Start CPU Profile: <code>curl -X POST "http://localhost:8081/debug/pprof/start?type=cpu&duration=30s"</code></div>
        <div class="endpoint">Stop Profile: <code>curl -X POST "http://localhost:8081/debug/pprof/stop?type=cpu"</code></div>
    </div>
    
    <div class="section">
        <h2>Usage Examples</h2>
        <pre>
# Start a CPU profile for 30 seconds
curl -X POST "http://localhost:8081/debug/pprof/start?type=cpu&duration=30s"

# Take a memory snapshot
curl -X POST "http://localhost:8081/debug/pprof/start?type=mem"

# Check profiling status
curl http://localhost:8081/debug/pprof/status

# List stored profiles
curl http://localhost:8081/debug/pprof/list

# Analyze with go tool pprof
go tool pprof http://localhost:8081/debug/pprof/profile?seconds=30
        </pre>
    </div>
</body>
</html>
`)
}

func (app *Application) handleHealth(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	health := map[string]interface{}{
		"status":    "healthy",
		"timestamp": time.Now().Format(time.RFC3339),
		"uptime":    time.Since(time.Now()).String(),
		"goroutines": runtime.NumGoroutine(),
	}

	if err := json.NewEncoder(w).Encode(health); err != nil {
		http.Error(w, "Failed to encode health status", http.StatusInternalServerError)
	}
}

func (app *Application) handleWorkload(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	if r.Method == http.MethodPost {
		action := r.URL.Query().Get("action")
		switch action {
		case "start":
			if app.workload != nil {
				app.workload.Start()
			}
		case "stop":
			if app.workload != nil {
				app.workload.Stop()
			}
		}
	}

	status := map[string]interface{}{
		"running": app.workload != nil && app.workload.IsRunning(),
	}

	if err := json.NewEncoder(w).Encode(status); err != nil {
		http.Error(w, "Failed to encode workload status", http.StatusInternalServerError)
	}
}

func (app *Application) handleStats(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	var m runtime.MemStats
	runtime.ReadMemStats(&m)

	stats := map[string]interface{}{
		"timestamp":      time.Now().Format(time.RFC3339),
		"goroutines":     runtime.NumGoroutine(),
		"heap_alloc":     m.Alloc,
		"heap_sys":       m.HeapSys,
		"heap_objects":   m.HeapObjects,
		"gc_runs":        m.NumGC,
		"gc_pause_total": m.PauseTotalNs,
		"next_gc":        m.NextGC,
		"last_gc":        time.Unix(0, int64(m.LastGC)).Format(time.RFC3339),
	}

	if err := json.NewEncoder(w).Encode(stats); err != nil {
		http.Error(w, "Failed to encode stats", http.StatusInternalServerError)
	}
}

// Workload implementation

func NewWorkload(ctx context.Context) *Workload {
	ctx, cancel := context.WithCancel(ctx)
	return &Workload{
		ctx:     ctx,
		cancel:  cancel,
		running: false,
	}
}

func (w *Workload) Start() {
	w.mu.Lock()
	defer w.mu.Unlock()
	
	if w.running {
		return // Already running
	}
	
	w.running = true
	
	// CPU-intensive work
	w.wg.Add(1)
	go w.cpuWork()

	// Memory allocation work
	w.wg.Add(1)
	go w.memoryWork()

	// Goroutine creation work
	w.wg.Add(1)
	go w.goroutineWork()
}

func (w *Workload) Stop() {
	w.mu.Lock()
	defer w.mu.Unlock()
	
	if !w.running {
		return // Already stopped
	}
	
	w.running = false
	w.cancel()
	w.wg.Wait()
}

func (w *Workload) IsRunning() bool {
	w.mu.RLock()
	defer w.mu.RUnlock()
	return w.running
}

func (w *Workload) cpuWork() {
	defer w.wg.Done()

	ticker := time.NewTicker(100 * time.Millisecond)
	defer ticker.Stop()

	for {
		select {
		case <-w.ctx.Done():
			return
		case <-ticker.C:
			// Simulate CPU-intensive work
			for i := 0; i < 100000; i++ {
				_ = rand.Float64() * rand.Float64()
			}
		}
	}
}

func (w *Workload) memoryWork() {
	defer w.wg.Done()

	ticker := time.NewTicker(500 * time.Millisecond)
	defer ticker.Stop()

	var data [][]byte

	for {
		select {
		case <-w.ctx.Done():
			return
		case <-ticker.C:
			// Allocate memory
			buf := make([]byte, rand.Intn(1024*1024)) // Up to 1MB
			data = append(data, buf)

			// Occasionally free some memory
			if len(data) > 10 {
				data = data[len(data)/2:]
			}
		}
	}
}

func (w *Workload) goroutineWork() {
	defer w.wg.Done()

	ticker := time.NewTicker(2 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-w.ctx.Done():
			return
		case <-ticker.C:
			// Create short-lived goroutines
			for i := 0; i < 10; i++ {
				go func() {
					time.Sleep(time.Duration(rand.Intn(1000)) * time.Millisecond)
				}()
			}
		}
	}
}