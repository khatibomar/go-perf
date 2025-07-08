// main.go
package main

import (
	"context"
	"encoding/json"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"exercise-03-04/incident"
	"exercise-03-04/monitoring"
)

type Server struct {
	incidentManager *incident.Manager
	monitor         *monitoring.SystemMonitor
}

func main() {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Initialize incident management system
	incidentConfig := incident.Config{
		DetectionInterval:    time.Second * 30,
		ResponseTimeout:      time.Minute * 5,
		ResolutionTimeout:    time.Minute * 30,
		MaxConcurrentIncidents: 10,
		EnableAutoResponse:   true,
		AlertChannels:        []string{"console", "webhook"},
	}

	incidentManager := incident.NewManager(incidentConfig)

	// Initialize system monitor
	monitorConfig := monitoring.Config{
		SampleInterval:     time.Second * 5,
		MetricsRetention:   time.Hour * 24,
		AlertThresholds: monitoring.AlertThresholds{
			CPUUsage:        80.0,
			MemoryUsage:     85.0,
			DiskUsage:       90.0,
			ResponseTime:    time.Second * 2,
			ErrorRate:       5.0,
			GoroutineCount:  1000,
		},
	}

	monitor := monitoring.NewSystemMonitor(monitorConfig)

	// Connect monitor to incident manager
	monitor.RegisterAlertCallback(incidentManager.HandleAlert)

	server := &Server{
		incidentManager: incidentManager,
		monitor:         monitor,
	}

	// Start background services
	go incidentManager.Start(ctx)
	go monitor.Start(ctx)

	// Setup HTTP server
	mux := http.NewServeMux()
	server.registerHandlers(mux)

	httpServer := &http.Server{
		Addr:    ":8084",
		Handler: mux,
	}

	// Start HTTP server
	go func() {
		log.Printf("Incident Response System starting on port 8084")
		log.Printf("Available endpoints:")
		log.Printf("  GET  /health                    - Health check")
		log.Printf("  GET  /metrics                   - Current system metrics")
		log.Printf("  GET  /incidents                 - List incidents")
		log.Printf("  POST /incidents                 - Create incident")
		log.Printf("  GET  /incidents/{id}            - Get incident details")
		log.Printf("  POST /incidents/{id}/resolve    - Resolve incident")
		log.Printf("  POST /incidents/{id}/escalate   - Escalate incident")
		log.Printf("  GET  /debug/profile             - Emergency profiling")
		log.Printf("  POST /debug/analyze             - Trigger analysis")
		log.Printf("  GET  /postmortem/{id}           - Get post-mortem")
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
	mux.HandleFunc("/incidents", s.handleIncidents)
	mux.HandleFunc("/incidents/", s.handleIncidentDetails)
	mux.HandleFunc("/debug/profile", s.handleEmergencyProfile)
	mux.HandleFunc("/debug/analyze", s.handleEmergencyAnalyze)
	mux.HandleFunc("/postmortem/", s.handlePostMortem)
}

func (s *Server) handleHealth(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"status":    "healthy",
		"timestamp": time.Now(),
		"service":   "incident-response",
		"version":   "1.0.0",
	})
}

func (s *Server) handleMetrics(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	metrics := s.monitor.GetCurrentMetrics()
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(metrics)
}

func (s *Server) handleIncidents(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		incidents := s.incidentManager.ListIncidents()
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(incidents)

	case http.MethodPost:
		var req incident.CreateRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "Invalid JSON", http.StatusBadRequest)
			return
		}

		incident, err := s.incidentManager.CreateIncident(req)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(incident)

	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

func (s *Server) handleIncidentDetails(w http.ResponseWriter, r *http.Request) {
	path := r.URL.Path[len("/incidents/"):]
	if path == "" {
		http.Error(w, "Incident ID required", http.StatusBadRequest)
		return
	}

	// Parse action from path
	var incidentID, action string
	if len(path) > 36 && path[36] == '/' { // UUID length
		incidentID = path[:36]
		action = path[37:]
	} else {
		incidentID = path
	}

	switch {
	case action == "resolve" && r.Method == http.MethodPost:
		err := s.incidentManager.ResolveIncident(incidentID)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		w.WriteHeader(http.StatusOK)

	case action == "escalate" && r.Method == http.MethodPost:
		err := s.incidentManager.EscalateIncident(incidentID)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		w.WriteHeader(http.StatusOK)

	case action == "" && r.Method == http.MethodGet:
		incident, err := s.incidentManager.GetIncident(incidentID)
		if err != nil {
			http.Error(w, err.Error(), http.StatusNotFound)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(incident)

	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

func (s *Server) handleEmergencyProfile(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	profile, err := s.monitor.EmergencyProfile()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(profile)
}

func (s *Server) handleEmergencyAnalyze(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	analysis, err := s.monitor.EmergencyAnalyze()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(analysis)
}

func (s *Server) handlePostMortem(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	incidentID := r.URL.Path[len("/postmortem/"):]
	if incidentID == "" {
		http.Error(w, "Incident ID required", http.StatusBadRequest)
		return
	}

	postMortem, err := s.incidentManager.GetPostMortem(incidentID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(postMortem)
}