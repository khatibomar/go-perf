// main.go
package main

import (
	"context"
	"encoding/json"
	"log"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"syscall"
	"time"

	"exercise-03-03/analysis"
	"exercise-03-03/simulation"
)

type Server struct {
	analyzer  *analysis.BottleneckAnalyzer
	simulator *simulation.WorkloadSimulator
}

func main() {
	// Create bottleneck analyzer
	config := analysis.BottleneckConfig{
		SampleInterval:      time.Second * 5,
		AnalysisWindow:      time.Minute * 10,
		ThresholdMultiplier: 2.0,
		MinSamples:          5,
	}

	analyzer := analysis.NewBottleneckAnalyzer(config)

	// Create workload simulator
	simulator := simulation.NewWorkloadSimulator()

	server := &Server{
		analyzer:  analyzer,
		simulator: simulator,
	}

	// Register bottleneck callback
	analyzer.RegisterBottleneckCallback(func(bottleneck analysis.Bottleneck) {
		log.Printf("[BOTTLENECK DETECTED] Type: %s, Component: %s, Severity: %s, Description: %s",
			bottleneck.Type, bottleneck.Component, bottleneck.Severity, bottleneck.Description)
	})

	// Start analyzer
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	go analyzer.Start(ctx)

	// Setup HTTP server
	mux := http.NewServeMux()
	server.registerHandlers(mux)

	httpServer := &http.Server{
		Addr:    ":8083",
		Handler: mux,
	}

	// Start HTTP server
	go func() {
		log.Printf("Performance Bottleneck Analysis System starting on port 8083")
		log.Printf("Available endpoints:")
		log.Printf("  GET  /health                    - Health check")
		log.Printf("  GET  /metrics                   - Current performance metrics")
		log.Printf("  GET  /bottlenecks               - Recent bottlenecks")
		log.Printf("  POST /analyze                   - Trigger immediate analysis")
		log.Printf("  POST /baseline/{metric}         - Update baseline for metric")
		log.Printf("  POST /simulate/{workload}       - Start workload simulation")
		log.Printf("  POST /simulate/stop             - Stop workload simulation")
		log.Printf("  GET  /simulate/status           - Get simulation status")
		if err := httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("HTTP server failed: %v", err)
		}
	}()

	// Wait for interrupt signal
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
	<-sigChan

	log.Println("Shutting down...")
	cancel()

	// Shutdown HTTP server
	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer shutdownCancel()

	if err := httpServer.Shutdown(shutdownCtx); err != nil {
		log.Printf("HTTP server shutdown error: %v", err)
	}

	log.Println("Server stopped")
}

func (s *Server) registerHandlers(mux *http.ServeMux) {
	mux.HandleFunc("/health", s.handleHealth)
	mux.HandleFunc("/metrics", s.handleMetrics)
	mux.HandleFunc("/bottlenecks", s.handleBottlenecks)
	mux.HandleFunc("/analyze", s.handleAnalyze)
	mux.HandleFunc("/baseline/", s.handleBaseline)
	mux.HandleFunc("/simulate/", s.handleSimulate)
}

func (s *Server) handleHealth(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"status":    "healthy",
		"timestamp": time.Now(),
		"service":   "bottleneck-analyzer",
		"version":   "1.0.0",
	})
}

func (s *Server) handleMetrics(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	metrics := s.analyzer.GetCurrentMetrics()
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(metrics)
}

func (s *Server) handleBottlenecks(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	bottlenecks := s.analyzer.GetRecentBottlenecks(time.Hour)
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(bottlenecks)
}

func (s *Server) handleAnalyze(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	start := time.Now()
	bottlenecks := s.analyzer.AnalyzeNow()
	duration := time.Since(start)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"bottlenecks":      bottlenecks,
		"analysis_time":    duration,
		"timestamp":        time.Now(),
		"bottleneck_count": len(bottlenecks),
	})
}

func (s *Server) handleBaseline(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		// Return all baselines
		baselines := s.analyzer.GetBaselines()
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(baselines)

	case http.MethodPost:
		// Extract metric name from path
		path := r.URL.Path[len("/baseline/"):]
		if path == "" {
			http.Error(w, "Metric name required", http.StatusBadRequest)
			return
		}

		// Parse query parameters
		valueStr := r.URL.Query().Get("value")
		stdDevStr := r.URL.Query().Get("stddev")

		if valueStr == "" || stdDevStr == "" {
			http.Error(w, "Both value and stddev parameters required", http.StatusBadRequest)
			return
		}

		value, err := strconv.ParseFloat(valueStr, 64)
		if err != nil {
			http.Error(w, "Invalid value parameter", http.StatusBadRequest)
			return
		}

		stdDev, err := strconv.ParseFloat(stdDevStr, 64)
		if err != nil {
			http.Error(w, "Invalid stddev parameter", http.StatusBadRequest)
			return
		}

		s.analyzer.UpdateBaseline(path, value, stdDev)

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"status":    "updated",
			"metric":    path,
			"value":     value,
			"stddev":    stdDev,
			"timestamp": time.Now(),
		})

	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

func (s *Server) handleSimulate(w http.ResponseWriter, r *http.Request) {
	path := r.URL.Path[len("/simulate/"):]

	switch {
	case path == "stop" && r.Method == http.MethodPost:
		s.simulator.Stop()
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"status":    "stopped",
			"timestamp": time.Now(),
		})

	case path == "status" && r.Method == http.MethodGet:
		status := s.simulator.GetStatus()
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(status)

	case r.Method == http.MethodPost:
		// Start workload simulation
		durationStr := r.URL.Query().Get("duration")
		intensityStr := r.URL.Query().Get("intensity")

		duration := time.Minute * 5 // default
		if durationStr != "" {
			if d, err := time.ParseDuration(durationStr); err != nil {
				http.Error(w, "Invalid duration parameter", http.StatusBadRequest)
				return
			} else {
				duration = d
			}
		}

		intensity := "medium" // default
		if intensityStr != "" {
			intensity = intensityStr
		}

		err := s.simulator.StartWorkload(path, duration, intensity)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"status":    "started",
			"workload":  path,
			"duration":  duration,
			"intensity": intensity,
			"timestamp": time.Now(),
		})

	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}
