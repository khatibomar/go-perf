package main

import (
	"context"
	"encoding/json"
	"log"
	"net/http"
	"os"
	"os/signal"
	"runtime"
	"syscall"
	"time"

	"exercise-03-02/debugging"
	"exercise-03-02/workload"
)

type Application struct {
	memoryAnalyzer  *debugging.MemoryAnalyzer
	workloadManager *workload.Manager
	server          *http.Server
}

func main() {
	app := createApplication()

	// Start the application
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	if err := app.Start(ctx); err != nil {
		log.Fatalf("Failed to start application: %v", err)
	}

	// Wait for shutdown signal
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
	<-sigChan

	log.Println("Shutting down application...")
	cancel()

	if err := app.Shutdown(); err != nil {
		log.Printf("Error during shutdown: %v", err)
	}
}

func createApplication() *Application {
	// Configure memory analyzer
	analyzerConfig := debugging.AnalyzerConfig{
		SnapshotInterval:     time.Second * 5,
		MaxSnapshots:         100,
		LeakThreshold:        0.1, // 10% growth
		ConsecutiveChecks:    3,
		EnableGCOptimization: true,
	}

	memoryAnalyzer := debugging.NewMemoryAnalyzer(analyzerConfig)

	// Register leak detection callback
	memoryAnalyzer.RegisterLeakCallback(func(leak debugging.MemoryLeak) {
		log.Printf("MEMORY LEAK DETECTED: %s - %s (Severity: %s)", leak.Type, leak.Description, leak.Severity)
		for key, value := range leak.Metrics {
			log.Printf("  %s: %v", key, value)
		}
	})

	// Create workload manager
	workloadManager := workload.NewManager()

	return &Application{
		memoryAnalyzer:  memoryAnalyzer,
		workloadManager: workloadManager,
	}
}

func (app *Application) Start(ctx context.Context) error {
	// Start memory analyzer
	go app.memoryAnalyzer.Start(ctx)

	// Start workload manager
	go app.workloadManager.Start(ctx)

	// Setup HTTP server
	mux := http.NewServeMux()
	app.registerHTTPHandlers(mux)

	app.server = &http.Server{
		Addr:    ":8082",
		Handler: mux,
	}

	go func() {
		log.Printf("Starting HTTP server on %s", app.server.Addr)
		log.Println("Available endpoints:")
		log.Println("  GET  /health                    - Health check")
		log.Println("  GET  /memory/status             - Current memory status")
		log.Println("  GET  /memory/snapshots          - Memory snapshots")
		log.Println("  POST /memory/gc                 - Force garbage collection")
		log.Println("  POST /memory/optimize           - Optimize GC settings")
		log.Println("  POST /workload/start            - Start memory workload")
		log.Println("  POST /workload/stop             - Stop memory workload")
		log.Println("  GET  /workload/status           - Workload status")
		log.Println("")
		log.Println("Example usage:")
		log.Printf("  curl http://localhost%s/health", app.server.Addr)
		log.Printf("  curl http://localhost%s/memory/status", app.server.Addr)
		log.Printf("  curl -X POST http://localhost%s/workload/start?type=leak&intensity=medium", app.server.Addr)

		if err := app.server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Printf("HTTP server error: %v", err)
		}
	}()

	return nil
}

func (app *Application) Shutdown() error {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if app.server != nil {
		return app.server.Shutdown(ctx)
	}

	return nil
}

func (app *Application) registerHTTPHandlers(mux *http.ServeMux) {
	// Health check
	mux.HandleFunc("/health", app.handleHealth)

	// Memory analysis endpoints
	mux.HandleFunc("/memory/status", app.handleMemoryStatus)
	mux.HandleFunc("/memory/snapshots", app.handleMemorySnapshots)
	mux.HandleFunc("/memory/gc", app.handleForceGC)
	mux.HandleFunc("/memory/optimize", app.handleOptimizeGC)

	// Workload management endpoints
	mux.HandleFunc("/workload/start", app.handleStartWorkload)
	mux.HandleFunc("/workload/stop", app.handleStopWorkload)
	mux.HandleFunc("/workload/status", app.handleWorkloadStatus)
}

func (app *Application) handleHealth(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var m runtime.MemStats
	runtime.ReadMemStats(&m)

	health := map[string]interface{}{
		"status":     "healthy",
		"timestamp":  time.Now(),
		"goroutines": runtime.NumGoroutine(),
		"memory": map[string]interface{}{
			"alloc":       m.Alloc,
			"total_alloc": m.TotalAlloc,
			"sys":         m.Sys,
			"num_gc":      m.NumGC,
		},
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(health)
}

func (app *Application) handleMemoryStatus(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	snapshot := app.memoryAnalyzer.GetCurrentSnapshot()

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(snapshot)
}

func (app *Application) handleMemorySnapshots(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Parse query parameters
	sinceStr := r.URL.Query().Get("since")
	var since time.Time
	if sinceStr != "" {
		var err error
		since, err = time.Parse(time.RFC3339, sinceStr)
		if err != nil {
			http.Error(w, "Invalid since parameter", http.StatusBadRequest)
			return
		}
	} else {
		since = time.Now().Add(-time.Hour) // Default to last hour
	}

	snapshots := app.memoryAnalyzer.GetSnapshots(since)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(snapshots)
}

func (app *Application) handleForceGC(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	before, after, freed := app.memoryAnalyzer.ForceGCAndAnalyze()

	result := map[string]interface{}{
		"before": before,
		"after":  after,
		"freed":  freed,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(result)
}

func (app *Application) handleOptimizeGC(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	app.memoryAnalyzer.OptimizeGC()

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"status": "optimized"})
}

func (app *Application) handleStartWorkload(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	workloadType := r.URL.Query().Get("type")
	intensity := r.URL.Query().Get("intensity")
	durationStr := r.URL.Query().Get("duration")

	if workloadType == "" {
		workloadType = "normal"
	}
	if intensity == "" {
		intensity = "medium"
	}

	var duration time.Duration = time.Minute * 5 // Default duration
	if durationStr != "" {
		var err error
		duration, err = time.ParseDuration(durationStr)
		if err != nil {
			http.Error(w, "Invalid duration parameter", http.StatusBadRequest)
			return
		}
	}

	err := app.workloadManager.StartWorkload(workloadType, intensity, duration)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"status": "started"})
}

func (app *Application) handleStopWorkload(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	app.workloadManager.StopWorkload()

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"status": "stopped"})
}

func (app *Application) handleWorkloadStatus(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	status := app.workloadManager.GetStatus()

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(status)
}
