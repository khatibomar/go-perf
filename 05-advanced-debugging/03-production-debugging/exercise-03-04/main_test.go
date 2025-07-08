// main_test.go
package main

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"exercise-03-04/incident"
	"exercise-03-04/monitoring"
)

func TestServer_Health(t *testing.T) {
	server := createTestServer()

	req := httptest.NewRequest("GET", "/health", nil)
	w := httptest.NewRecorder()

	server.handleHealth(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}

	var response map[string]interface{}
	if err := json.NewDecoder(w.Body).Decode(&response); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	if response["status"] != "healthy" {
		t.Errorf("Expected status 'healthy', got %v", response["status"])
	}

	if response["service"] != "incident-response" {
		t.Errorf("Expected service 'incident-response', got %v", response["service"])
	}
}

func TestServer_Metrics(t *testing.T) {
	server := createTestServer()

	req := httptest.NewRequest("GET", "/metrics", nil)
	w := httptest.NewRecorder()

	server.handleMetrics(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}

	var metrics monitoring.SystemMetrics
	if err := json.NewDecoder(w.Body).Decode(&metrics); err != nil {
		t.Fatalf("Failed to decode metrics: %v", err)
	}

	if metrics.CustomMetrics == nil {
		t.Error("Expected custom metrics to be initialized")
	}
}

func TestServer_CreateIncident(t *testing.T) {
	server := createTestServer()

	createReq := incident.CreateRequest{
		Title:       "Test Incident",
		Description: "This is a test incident",
		Severity:    incident.SeverityHigh,
		Tags:        []string{"test", "api"},
		Metrics:     map[string]interface{}{"cpu": 85.0},
	}

	body, _ := json.Marshal(createReq)
	req := httptest.NewRequest("POST", "/incidents", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	server.handleIncidents(w, req)

	if w.Code != http.StatusCreated {
		t.Errorf("Expected status 201, got %d", w.Code)
	}

	var createdIncident incident.Incident
	if err := json.NewDecoder(w.Body).Decode(&createdIncident); err != nil {
		t.Fatalf("Failed to decode incident: %v", err)
	}

	if createdIncident.Title != createReq.Title {
		t.Errorf("Expected title '%s', got '%s'", createReq.Title, createdIncident.Title)
	}

	if createdIncident.Severity != createReq.Severity {
		t.Errorf("Expected severity '%s', got '%s'", createReq.Severity, createdIncident.Severity)
	}

	if createdIncident.Status != incident.StatusInProgress {
		t.Errorf("Expected status '%s', got '%s'", incident.StatusInProgress, createdIncident.Status)
	}
}

func TestServer_ListIncidents(t *testing.T) {
	server := createTestServer()

	// Create a test incident first
	createReq := incident.CreateRequest{
		Title:       "Test Incident",
		Description: "This is a test incident",
		Severity:    incident.SeverityMedium,
	}

	_, err := server.incidentManager.CreateIncident(createReq)
	if err != nil {
		t.Fatalf("Failed to create test incident: %v", err)
	}

	req := httptest.NewRequest("GET", "/incidents", nil)
	w := httptest.NewRecorder()

	server.handleIncidents(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}

	var incidents []*incident.Incident
	if err := json.NewDecoder(w.Body).Decode(&incidents); err != nil {
		t.Fatalf("Failed to decode incidents: %v", err)
	}

	if len(incidents) == 0 {
		t.Error("Expected at least one incident")
	}
}

func TestServer_GetIncident(t *testing.T) {
	server := createTestServer()

	// Create a test incident first
	createReq := incident.CreateRequest{
		Title:       "Test Incident",
		Description: "This is a test incident",
		Severity:    incident.SeverityLow,
	}

	createdIncident, err := server.incidentManager.CreateIncident(createReq)
	if err != nil {
		t.Fatalf("Failed to create test incident: %v", err)
	}

	req := httptest.NewRequest("GET", "/incidents/"+createdIncident.ID, nil)
	w := httptest.NewRecorder()

	server.handleIncidentDetails(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}

	var retrievedIncident incident.Incident
	if err := json.NewDecoder(w.Body).Decode(&retrievedIncident); err != nil {
		t.Fatalf("Failed to decode incident: %v", err)
	}

	if retrievedIncident.ID != createdIncident.ID {
		t.Errorf("Expected ID '%s', got '%s'", createdIncident.ID, retrievedIncident.ID)
	}
}

func TestServer_ResolveIncident(t *testing.T) {
	server := createTestServer()

	// Create a test incident first
	createReq := incident.CreateRequest{
		Title:       "Test Incident",
		Description: "This is a test incident",
		Severity:    incident.SeverityCritical,
	}

	createdIncident, err := server.incidentManager.CreateIncident(createReq)
	if err != nil {
		t.Fatalf("Failed to create test incident: %v", err)
	}

	req := httptest.NewRequest("POST", "/incidents/"+createdIncident.ID+"/resolve", nil)
	w := httptest.NewRecorder()

	server.handleIncidentDetails(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}

	// Verify the incident is resolved
	resolvedIncident, err := server.incidentManager.GetIncident(createdIncident.ID)
	if err != nil {
		t.Fatalf("Failed to get incident: %v", err)
	}

	if resolvedIncident.Status != incident.StatusResolved {
		t.Errorf("Expected status '%s', got '%s'", incident.StatusResolved, resolvedIncident.Status)
	}

	if resolvedIncident.ResolvedAt == nil {
		t.Error("Expected ResolvedAt to be set")
	}
}

func TestServer_EscalateIncident(t *testing.T) {
	server := createTestServer()

	// Create a test incident first
	createReq := incident.CreateRequest{
		Title:       "Test Incident",
		Description: "This is a test incident",
		Severity:    incident.SeverityMedium,
	}

	createdIncident, err := server.incidentManager.CreateIncident(createReq)
	if err != nil {
		t.Fatalf("Failed to create test incident: %v", err)
	}

	req := httptest.NewRequest("POST", "/incidents/"+createdIncident.ID+"/escalate", nil)
	w := httptest.NewRecorder()

	server.handleIncidentDetails(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}

	// Verify the incident is escalated
	escalatedIncident, err := server.incidentManager.GetIncident(createdIncident.ID)
	if err != nil {
		t.Fatalf("Failed to get incident: %v", err)
	}

	if escalatedIncident.Status != incident.StatusEscalated {
		t.Errorf("Expected status '%s', got '%s'", incident.StatusEscalated, escalatedIncident.Status)
	}
}

func TestServer_EmergencyProfile(t *testing.T) {
	server := createTestServer()

	req := httptest.NewRequest("GET", "/debug/profile", nil)
	w := httptest.NewRecorder()

	server.handleEmergencyProfile(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}

	var profile monitoring.EmergencyProfile
	if err := json.NewDecoder(w.Body).Decode(&profile); err != nil {
		t.Fatalf("Failed to decode profile: %v", err)
	}

	if profile.Goroutines <= 0 {
		t.Error("Expected goroutine count to be positive")
	}

	if len(profile.StackTraces) == 0 {
		t.Error("Expected stack traces to be present")
	}
}

func TestServer_EmergencyAnalyze(t *testing.T) {
	server := createTestServer()

	req := httptest.NewRequest("POST", "/debug/analyze", nil)
	w := httptest.NewRecorder()

	server.handleEmergencyAnalyze(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}

	var analysis monitoring.EmergencyAnalysis
	if err := json.NewDecoder(w.Body).Decode(&analysis); err != nil {
		t.Fatalf("Failed to decode analysis: %v", err)
	}

	if analysis.Severity == "" {
		t.Error("Expected severity to be set")
	}

	if analysis.Metrics == nil {
		t.Error("Expected metrics to be present")
	}
}

func TestServer_PostMortem(t *testing.T) {
	server := createTestServer()

	// Create and resolve a high severity incident to generate post-mortem
	createReq := incident.CreateRequest{
		Title:       "Critical Test Incident",
		Description: "This is a critical test incident",
		Severity:    incident.SeverityCritical,
	}

	createdIncident, err := server.incidentManager.CreateIncident(createReq)
	if err != nil {
		t.Fatalf("Failed to create test incident: %v", err)
	}

	// Resolve the incident to trigger post-mortem generation
	err = server.incidentManager.ResolveIncident(createdIncident.ID)
	if err != nil {
		t.Fatalf("Failed to resolve incident: %v", err)
	}

	// Small delay to ensure post-mortem is generated
	time.Sleep(100 * time.Millisecond)

	req := httptest.NewRequest("GET", "/postmortem/"+createdIncident.ID, nil)
	w := httptest.NewRecorder()

	server.handlePostMortem(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}

	var postMortem incident.PostMortem
	if err := json.NewDecoder(w.Body).Decode(&postMortem); err != nil {
		t.Fatalf("Failed to decode post-mortem: %v", err)
	}

	if postMortem.IncidentID != createdIncident.ID {
		t.Errorf("Expected incident ID '%s', got '%s'", createdIncident.ID, postMortem.IncidentID)
	}
}

func TestServer_InvalidMethods(t *testing.T) {
	server := createTestServer()

	tests := []struct {
		name           string
		method         string
		path           string
		expectedStatus int
	}{
		{"POST to metrics", "POST", "/metrics", http.StatusMethodNotAllowed},
		{"PUT to health", "PUT", "/health", http.StatusOK}, // Health accepts all methods
		{"DELETE to incidents", "DELETE", "/incidents", http.StatusMethodNotAllowed},
		{"GET to debug/analyze", "GET", "/debug/analyze", http.StatusMethodNotAllowed},
		{"POST to debug/profile", "POST", "/debug/profile", http.StatusMethodNotAllowed},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(tt.method, tt.path, nil)
			w := httptest.NewRecorder()

			mux := http.NewServeMux()
			server.registerHandlers(mux)
			mux.ServeHTTP(w, req)

			if w.Code != tt.expectedStatus {
				t.Errorf("Expected status %d, got %d", tt.expectedStatus, w.Code)
			}
		})
	}
}

func TestServer_InvalidJSON(t *testing.T) {
	server := createTestServer()

	req := httptest.NewRequest("POST", "/incidents", strings.NewReader("invalid json"))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	server.handleIncidents(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("Expected status 400, got %d", w.Code)
	}
}

func TestServer_NotFoundIncident(t *testing.T) {
	server := createTestServer()

	req := httptest.NewRequest("GET", "/incidents/nonexistent-id", nil)
	w := httptest.NewRecorder()

	server.handleIncidentDetails(w, req)

	if w.Code != http.StatusNotFound {
		t.Errorf("Expected status 404, got %d", w.Code)
	}
}

func TestMonitoringIntegration(t *testing.T) {
	server := createTestServer()

	// Start monitoring in background
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	go server.monitor.Start(ctx)

	// Wait a bit for monitoring to collect metrics
	time.Sleep(100 * time.Millisecond)

	// Check that metrics are being collected
	metrics := server.monitor.GetCurrentMetrics()
	if metrics == nil {
		t.Error("Expected metrics to be collected")
	}

	if metrics.GoroutineCount <= 0 {
		t.Error("Expected positive goroutine count")
	}
}

func createTestServer() *Server {
	// Initialize incident management system
	incidentConfig := incident.Config{
		DetectionInterval:       time.Second * 30,
		ResponseTimeout:         time.Minute * 5,
		ResolutionTimeout:       time.Minute * 30,
		MaxConcurrentIncidents:  10,
		EnableAutoResponse:      true,
		AlertChannels:           []string{"console", "webhook"},
	}

	incidentManager := incident.NewManager(incidentConfig)

	// Initialize system monitor
	monitorConfig := monitoring.Config{
		SampleInterval:   time.Second * 5,
		MetricsRetention: time.Hour * 24,
		AlertThresholds: monitoring.AlertThresholds{
			CPUUsage:       80.0,
			MemoryUsage:    85.0,
			DiskUsage:      90.0,
			ResponseTime:   time.Second * 2,
			ErrorRate:      5.0,
			GoroutineCount: 1000,
		},
	}

	monitor := monitoring.NewSystemMonitor(monitorConfig)

	// Connect monitor to incident manager
	monitor.RegisterAlertCallback(incidentManager.HandleAlert)

	return &Server{
		incidentManager: incidentManager,
		monitor:         monitor,
	}
}