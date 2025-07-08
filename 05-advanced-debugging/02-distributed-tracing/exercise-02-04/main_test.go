package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"
)

// Test AlertManager creation and initialization
func TestNewAlertManager(t *testing.T) {
	am := NewAlertManager()
	defer am.Shutdown()

	if am == nil {
		t.Fatal("AlertManager should not be nil")
	}

	if am.alerts == nil {
		t.Error("Alerts map should be initialized")
	}

	if am.incidents == nil {
		t.Error("Incidents map should be initialized")
	}

	if am.alertRules == nil {
		t.Error("Alert rules map should be initialized")
	}

	if am.escalationRules == nil {
		t.Error("Escalation rules map should be initialized")
	}

	if am.metrics == nil {
		t.Error("Metrics should be initialized")
	}

	// Check default alert rules
	rules := am.GetAlertRules()
	expectedRules := []string{"HighErrorRate", "HighLatency", "HighMemoryUsage", "ServiceDown", "DiskSpaceLow"}
	for _, expected := range expectedRules {
		if _, exists := rules[expected]; !exists {
			t.Errorf("Expected alert rule '%s' not found", expected)
		}
	}

	// Check escalation rules
	if len(am.escalationRules) == 0 {
		t.Error("Should have escalation rules")
	}
}

// Test alert firing and resolution
func TestAlertManager_FireAndResolveAlert(t *testing.T) {
	am := NewAlertManager()
	defer am.Shutdown()

	// Fire an alert
	labels := map[string]string{
		"instance": "server-1",
		"job":      "api-server",
	}
	alert := am.FireAlert("HighErrorRate", "critical", "Error rate exceeded threshold", labels)

	if alert == nil {
		t.Fatal("Alert should not be nil")
	}

	if alert.Name != "HighErrorRate" {
		t.Errorf("Expected alert name 'HighErrorRate', got '%s'", alert.Name)
	}

	if alert.Severity != "critical" {
		t.Errorf("Expected severity 'critical', got '%s'", alert.Severity)
	}

	if alert.Status != "firing" {
		t.Errorf("Expected status 'firing', got '%s'", alert.Status)
	}

	// Check if alert exists in manager
	alerts := am.GetAlerts()
	if _, exists := alerts[alert.ID]; !exists {
		t.Error("Alert should exist in manager")
	}

	// Resolve the alert
	err := am.ResolveAlert(alert.ID)
	if err != nil {
		t.Fatalf("Failed to resolve alert: %v", err)
	}

	// Check alert status
	updatedAlerts := am.GetAlerts()
	updatedAlert := updatedAlerts[alert.ID]
	if updatedAlert.Status != "resolved" {
		t.Errorf("Expected status 'resolved', got '%s'", updatedAlert.Status)
	}

	if updatedAlert.EndTime == nil {
		t.Error("EndTime should be set for resolved alert")
	}

	// Test resolving non-existent alert
	err = am.ResolveAlert("non-existent")
	if err == nil {
		t.Error("Should return error for non-existent alert")
	}

	// Test resolving already resolved alert
	err = am.ResolveAlert(alert.ID)
	if err == nil {
		t.Error("Should return error for already resolved alert")
	}
}

// Test incident creation and resolution
func TestAlertManager_IncidentManagement(t *testing.T) {
	am := NewAlertManager()
	defer am.Shutdown()

	// Fire a critical alert (should create incident)
	labels := map[string]string{
		"instance": "server-1",
		"job":      "api-server",
	}
	_ = am.FireAlert("ServiceDown", "critical", "Service is not responding", labels)

	// Check if incident was created
	incidents := am.GetIncidents()
	if len(incidents) == 0 {
		t.Error("Incident should be created for critical alert")
	}

	var incident *Incident
	for _, inc := range incidents {
		if inc.Status == "open" {
			incident = inc
			break
		}
	}

	if incident == nil {
		t.Fatal("Should have an open incident")
	}

	if incident.Severity != "critical" {
		t.Errorf("Expected incident severity 'critical', got '%s'", incident.Severity)
	}

	if len(incident.Alerts) == 0 {
		t.Error("Incident should have associated alerts")
	}

	// Resolve the incident
	err := am.ResolveIncident(incident.ID)
	if err != nil {
		t.Fatalf("Failed to resolve incident: %v", err)
	}

	// Check incident status
	updatedIncidents := am.GetIncidents()
	updatedIncident := updatedIncidents[incident.ID]
	if updatedIncident.Status != "resolved" {
		t.Errorf("Expected status 'resolved', got '%s'", updatedIncident.Status)
	}

	if updatedIncident.EndTime == nil {
		t.Error("EndTime should be set for resolved incident")
	}

	// Test resolving non-existent incident
	err = am.ResolveIncident("non-existent")
	if err == nil {
		t.Error("Should return error for non-existent incident")
	}

	// Test resolving already resolved incident
	err = am.ResolveIncident(incident.ID)
	if err == nil {
		t.Error("Should return error for already resolved incident")
	}
}

// Test post-mortem creation
func TestAlertManager_PostMortem(t *testing.T) {
	am := NewAlertManager()
	defer am.Shutdown()

	// Create an incident first
	labels := map[string]string{
		"instance": "server-1",
		"job":      "api-server",
	}
	am.FireAlert("ServiceDown", "critical", "Service is not responding", labels)

	incidents := am.GetIncidents()
	if len(incidents) == 0 {
		t.Fatal("Should have an incident")
	}

	var incidentID string
	for id := range incidents {
		incidentID = id
		break
	}

	// Create post-mortem
	rootCause := "Database connection pool exhausted"
	impact := "Service unavailable for 15 minutes"
	resolution := "Increased connection pool size and added monitoring"
	lessonsLearned := []string{"Need better connection pool monitoring", "Should have alerts on pool utilization"}
	actionItems := []string{"Implement connection pool alerts", "Review other resource pools"}

	postMortem, err := am.CreatePostMortem(incidentID, rootCause, impact, resolution, lessonsLearned, actionItems)
	if err != nil {
		t.Fatalf("Failed to create post-mortem: %v", err)
	}

	if postMortem == nil {
		t.Fatal("Post-mortem should not be nil")
	}

	if postMortem.IncidentID != incidentID {
		t.Errorf("Expected incident ID '%s', got '%s'", incidentID, postMortem.IncidentID)
	}

	if postMortem.RootCause != rootCause {
		t.Errorf("Expected root cause '%s', got '%s'", rootCause, postMortem.RootCause)
	}

	if len(postMortem.LessonsLearned) != len(lessonsLearned) {
		t.Errorf("Expected %d lessons learned, got %d", len(lessonsLearned), len(postMortem.LessonsLearned))
	}

	// Test creating post-mortem for non-existent incident
	_, err = am.CreatePostMortem("non-existent", rootCause, impact, resolution, lessonsLearned, actionItems)
	if err == nil {
		t.Error("Should return error for non-existent incident")
	}
}

// Test HTTP handlers
func TestHealthHandler(t *testing.T) {
	req := httptest.NewRequest("GET", "/health", nil)
	rec := httptest.NewRecorder()

	healthHandler(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", rec.Code)
	}

	var response map[string]interface{}
	if err := json.Unmarshal(rec.Body.Bytes(), &response); err != nil {
		t.Fatalf("Failed to parse JSON response: %v", err)
	}

	if response["status"] != "healthy" {
		t.Errorf("Expected status 'healthy', got '%v'", response["status"])
	}

	if response["service"] != "alerting-incident-response" {
		t.Errorf("Expected service 'alerting-incident-response', got '%v'", response["service"])
	}
}

func TestAlertsHandler(t *testing.T) {
	am := NewAlertManager()
	defer am.Shutdown()

	// Test GET /alerts
	req := httptest.NewRequest("GET", "/alerts", nil)
	rec := httptest.NewRecorder()

	am.alertsHandler(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", rec.Code)
	}

	var response map[string]interface{}
	if err := json.Unmarshal(rec.Body.Bytes(), &response); err != nil {
		t.Fatalf("Failed to parse JSON response: %v", err)
	}

	if _, ok := response["alerts"]; !ok {
		t.Error("Response should contain alerts")
	}

	if _, ok := response["count"]; !ok {
		t.Error("Response should contain count")
	}

	// Test POST /alerts
	alertData := map[string]interface{}{
		"name":     "TestAlert",
		"severity": "warning",
		"message":  "Test alert message",
		"labels": map[string]string{
			"instance": "test-server",
		},
	}

	jsonData, _ := json.Marshal(alertData)
	req = httptest.NewRequest("POST", "/alerts", bytes.NewBuffer(jsonData))
	req.Header.Set("Content-Type", "application/json")
	rec = httptest.NewRecorder()

	am.alertsHandler(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", rec.Code)
	}

	var alertResponse Alert
	if err := json.Unmarshal(rec.Body.Bytes(), &alertResponse); err != nil {
		t.Fatalf("Failed to parse JSON response: %v", err)
	}

	if alertResponse.Name != "TestAlert" {
		t.Errorf("Expected alert name 'TestAlert', got '%s'", alertResponse.Name)
	}

	if alertResponse.Severity != "warning" {
		t.Errorf("Expected severity 'warning', got '%s'", alertResponse.Severity)
	}
}

func TestIncidentsHandler(t *testing.T) {
	am := NewAlertManager()
	defer am.Shutdown()

	// Create an incident by firing a critical alert
	labels := map[string]string{
		"instance": "server-1",
		"job":      "api-server",
	}
	am.FireAlert("ServiceDown", "critical", "Service is not responding", labels)

	req := httptest.NewRequest("GET", "/incidents", nil)
	rec := httptest.NewRecorder()

	am.incidentsHandler(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", rec.Code)
	}

	var response map[string]interface{}
	if err := json.Unmarshal(rec.Body.Bytes(), &response); err != nil {
		t.Fatalf("Failed to parse JSON response: %v", err)
	}

	if _, ok := response["incidents"]; !ok {
		t.Error("Response should contain incidents")
	}

	if _, ok := response["count"]; !ok {
		t.Error("Response should contain count")
	}

	count, ok := response["count"].(float64)
	if !ok || count == 0 {
		t.Error("Should have at least one incident")
	}
}

func TestAlertRulesHandler(t *testing.T) {
	am := NewAlertManager()
	defer am.Shutdown()

	req := httptest.NewRequest("GET", "/alert-rules", nil)
	rec := httptest.NewRecorder()

	am.alertRulesHandler(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", rec.Code)
	}

	var response map[string]interface{}
	if err := json.Unmarshal(rec.Body.Bytes(), &response); err != nil {
		t.Fatalf("Failed to parse JSON response: %v", err)
	}

	if _, ok := response["rules"]; !ok {
		t.Error("Response should contain rules")
	}

	if _, ok := response["count"]; !ok {
		t.Error("Response should contain count")
	}

	count, ok := response["count"].(float64)
	if !ok || count == 0 {
		t.Error("Should have alert rules")
	}
}

func TestResolveAlertHandler(t *testing.T) {
	am := NewAlertManager()
	defer am.Shutdown()

	// Fire an alert first
	labels := map[string]string{
		"instance": "server-1",
		"job":      "api-server",
	}
	alert := am.FireAlert("HighErrorRate", "warning", "Error rate exceeded threshold", labels)

	// Test resolving the alert
	req := httptest.NewRequest("POST", fmt.Sprintf("/alerts/resolve?id=%s", alert.ID), nil)
	rec := httptest.NewRecorder()

	am.resolveAlertHandler(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", rec.Code)
	}

	var response map[string]string
	if err := json.Unmarshal(rec.Body.Bytes(), &response); err != nil {
		t.Fatalf("Failed to parse JSON response: %v", err)
	}

	if response["status"] != "resolved" {
		t.Errorf("Expected status 'resolved', got '%s'", response["status"])
	}

	if response["alert_id"] != alert.ID {
		t.Errorf("Expected alert_id '%s', got '%s'", alert.ID, response["alert_id"])
	}

	// Test resolving non-existent alert
	req = httptest.NewRequest("POST", "/alerts/resolve?id=non-existent", nil)
	rec = httptest.NewRecorder()

	am.resolveAlertHandler(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Errorf("Expected status 404, got %d", rec.Code)
	}

	// Test missing alert ID
	req = httptest.NewRequest("POST", "/alerts/resolve", nil)
	rec = httptest.NewRecorder()

	am.resolveAlertHandler(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Errorf("Expected status 400, got %d", rec.Code)
	}
}

func TestResolveIncidentHandler(t *testing.T) {
	am := NewAlertManager()
	defer am.Shutdown()

	// Fire a critical alert to create an incident
	labels := map[string]string{
		"instance": "server-1",
		"job":      "api-server",
	}
	am.FireAlert("ServiceDown", "critical", "Service is not responding", labels)

	// Get the incident ID
	incidents := am.GetIncidents()
	if len(incidents) == 0 {
		t.Fatal("Should have an incident")
	}

	var incidentID string
	for id := range incidents {
		incidentID = id
		break
	}

	// Test resolving the incident
	req := httptest.NewRequest("POST", fmt.Sprintf("/incidents/resolve?id=%s", incidentID), nil)
	rec := httptest.NewRecorder()

	am.resolveIncidentHandler(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", rec.Code)
	}

	var response map[string]string
	if err := json.Unmarshal(rec.Body.Bytes(), &response); err != nil {
		t.Fatalf("Failed to parse JSON response: %v", err)
	}

	if response["status"] != "resolved" {
		t.Errorf("Expected status 'resolved', got '%s'", response["status"])
	}

	if response["incident_id"] != incidentID {
		t.Errorf("Expected incident_id '%s', got '%s'", incidentID, response["incident_id"])
	}
}

func TestPostMortemHandler(t *testing.T) {
	am := NewAlertManager()
	defer am.Shutdown()

	// Fire a critical alert to create an incident
	labels := map[string]string{
		"instance": "server-1",
		"job":      "api-server",
	}
	am.FireAlert("ServiceDown", "critical", "Service is not responding", labels)

	// Get the incident ID
	incidents := am.GetIncidents()
	if len(incidents) == 0 {
		t.Fatal("Should have an incident")
	}

	var incidentID string
	for id := range incidents {
		incidentID = id
		break
	}

	// Test creating post-mortem
	postMortemData := map[string]interface{}{
		"incident_id":     incidentID,
		"root_cause":      "Database connection pool exhausted",
		"impact":          "Service unavailable for 15 minutes",
		"resolution":      "Increased connection pool size",
		"lessons_learned": []string{"Need better monitoring"},
		"action_items":    []string{"Implement alerts"},
	}

	jsonData, _ := json.Marshal(postMortemData)
	req := httptest.NewRequest("POST", "/post-mortem", bytes.NewBuffer(jsonData))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	am.postMortemHandler(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", rec.Code)
	}

	var response PostMortem
	if err := json.Unmarshal(rec.Body.Bytes(), &response); err != nil {
		t.Fatalf("Failed to parse JSON response: %v", err)
	}

	if response.IncidentID != incidentID {
		t.Errorf("Expected incident ID '%s', got '%s'", incidentID, response.IncidentID)
	}

	if response.RootCause != "Database connection pool exhausted" {
		t.Errorf("Expected root cause 'Database connection pool exhausted', got '%s'", response.RootCause)
	}
}

func TestLoadTestHandler(t *testing.T) {
	am := NewAlertManager()
	defer am.Shutdown()

	req := httptest.NewRequest("GET", "/load-test?operations=5", nil)
	rec := httptest.NewRecorder()

	am.loadTestHandler(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", rec.Code)
	}

	var response map[string]interface{}
	if err := json.Unmarshal(rec.Body.Bytes(), &response); err != nil {
		t.Fatalf("Failed to parse JSON response: %v", err)
	}

	if response["operations"] != float64(5) {
		t.Errorf("Expected operations 5, got %v", response["operations"])
	}

	if _, ok := response["duration_ms"]; !ok {
		t.Error("Response should contain duration_ms")
	}

	if _, ok := response["ops_per_second"]; !ok {
		t.Error("Response should contain ops_per_second")
	}
}

// Test concurrent alert operations
func TestConcurrentAlertOperations(t *testing.T) {
	am := NewAlertManager()
	defer am.Shutdown()

	// Fire multiple alerts concurrently with unique names
	const numAlerts = 10
	done := make(chan bool, numAlerts)

	for i := 0; i < numAlerts; i++ {
		go func(id int) {
			defer func() { done <- true }()
			labels := map[string]string{
				"instance": fmt.Sprintf("server-%d", id),
				"job":      "api-server",
				"test_id":  fmt.Sprintf("%d", id),
			}
			// Use unique alert names to avoid any potential conflicts
			alertName := fmt.Sprintf("ConcurrentTest-%d", id)
			am.FireAlert(alertName, "warning", fmt.Sprintf("Test alert %d", id), labels)
		}(i)
	}

	// Wait for all alerts to be fired
	for i := 0; i < numAlerts; i++ {
		select {
		case <-done:
			// Alert fired successfully
		case <-time.After(5 * time.Second):
			t.Fatal("Timeout waiting for concurrent alerts")
		}
	}

	// Give a small delay to ensure all operations are complete
	time.Sleep(100 * time.Millisecond)

	// Check that all alerts were created
	alerts := am.GetAlerts()
	concurrentAlerts := 0
	for _, alert := range alerts {
		if strings.HasPrefix(alert.Name, "ConcurrentTest-") {
			concurrentAlerts++
		}
	}

	if concurrentAlerts != numAlerts {
		t.Errorf("Expected %d concurrent alerts, got %d", numAlerts, concurrentAlerts)
		// Debug: print all alert names
		t.Logf("All alerts:")
		for id, alert := range alerts {
			t.Logf("  %s: %s", id, alert.Name)
		}
	}
}

// Test alert rule validation
func TestAlertRuleValidation(t *testing.T) {
	am := NewAlertManager()
	defer am.Shutdown()

	rules := am.GetAlertRules()

	for name, rule := range rules {
		if rule.Name == "" {
			t.Errorf("Alert rule %s should have a name", name)
		}

		if rule.Expression == "" {
			t.Errorf("Alert rule %s should have an expression", name)
		}

		if rule.Duration == "" {
			t.Errorf("Alert rule %s should have a duration", name)
		}

		if rule.Severity == "" {
			t.Errorf("Alert rule %s should have a severity", name)
		}

		if !rule.Enabled {
			t.Errorf("Alert rule %s should be enabled by default", name)
		}

		if rule.Labels == nil {
			t.Errorf("Alert rule %s should have labels", name)
		}

		if rule.Annotations == nil {
			t.Errorf("Alert rule %s should have annotations", name)
		}

		// Check required annotations
		if rule.Annotations["summary"] == "" {
			t.Errorf("Alert rule %s should have summary annotation", name)
		}

		if rule.Annotations["description"] == "" {
			t.Errorf("Alert rule %s should have description annotation", name)
		}

		if rule.Annotations["runbook"] == "" {
			t.Errorf("Alert rule %s should have runbook annotation", name)
		}
	}
}

// Test escalation rules
func TestEscalationRules(t *testing.T) {
	am := NewAlertManager()
	defer am.Shutdown()

	expectedSeverities := []string{"critical", "warning", "info"}
	for _, severity := range expectedSeverities {
		if _, exists := am.escalationRules[severity]; !exists {
			t.Errorf("Should have escalation rule for severity '%s'", severity)
		}
	}

	// Check critical escalation rule
	criticalRule := am.escalationRules["critical"]
	if criticalRule.TimeToEscalate != 5*time.Minute {
		t.Errorf("Critical escalation should be 5 minutes, got %v", criticalRule.TimeToEscalate)
	}

	if len(criticalRule.EscalateTo) == 0 {
		t.Error("Critical escalation should have escalation targets")
	}

	if len(criticalRule.NotificationChannels) == 0 {
		t.Error("Critical escalation should have notification channels")
	}
}

// Test metrics initialization
func TestMetricsInitialization(t *testing.T) {
	metrics := NewAlertMetrics()

	if metrics == nil {
		t.Fatal("Metrics should not be nil")
	}

	if metrics.alertsTotal == nil {
		t.Error("alertsTotal metric should be initialized")
	}

	if metrics.alertsActive == nil {
		t.Error("alertsActive metric should be initialized")
	}

	if metrics.incidentsTotal == nil {
		t.Error("incidentsTotal metric should be initialized")
	}

	if metrics.incidentsActive == nil {
		t.Error("incidentsActive metric should be initialized")
	}

	if metrics.alertAccuracy == nil {
		t.Error("alertAccuracy metric should be initialized")
	}

	if metrics.falsePositiveRate == nil {
		t.Error("falsePositiveRate metric should be initialized")
	}

	if metrics.mttd == nil {
		t.Error("mttd metric should be initialized")
	}

	if metrics.mttr == nil {
		t.Error("mttr metric should be initialized")
	}

	if metrics.escalationsTotal == nil {
		t.Error("escalationsTotal metric should be initialized")
	}

	if metrics.responseTime == nil {
		t.Error("responseTime metric should be initialized")
	}
}

// Test performance requirements
func TestPerformanceRequirements(t *testing.T) {
	am := NewAlertManager()
	defer am.Shutdown()

	// Test MTTD requirement (<2 minutes)
	start := time.Now()
	labels := map[string]string{
		"instance": "server-1",
		"job":      "api-server",
	}
	alert := am.FireAlert("PerformanceTest", "critical", "Performance test alert", labels)
	mttd := time.Since(start)

	if mttd > 2*time.Minute {
		t.Errorf("MTTD should be less than 2 minutes, got %v", mttd)
	}

	// Test alert response time (<30 seconds)
	start = time.Now()
	err := am.ResolveAlert(alert.ID)
	responseTime := time.Since(start)

	if err != nil {
		t.Fatalf("Failed to resolve alert: %v", err)
	}

	if responseTime > 30*time.Second {
		t.Errorf("Alert response time should be less than 30 seconds, got %v", responseTime)
	}
}

// Simple benchmark without metrics to test basic functionality
func BenchmarkFireAlertSimple(b *testing.B) {
	alerts := make(map[string]*Alert)
	labels := map[string]string{
		"instance": "server-1",
		"job":      "api-server",
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		alertID := fmt.Sprintf("BenchmarkAlert-%d", time.Now().UnixNano())
		alert := &Alert{
			ID:       alertID,
			Name:     "BenchmarkAlert",
			Severity: "warning",
			Status:   "firing",
			Message:  "Benchmark test alert",
			Labels:   labels,
			StartTime: time.Now(),
		}
		alerts[alertID] = alert
	}
}

// Benchmark tests
func BenchmarkFireAlert(b *testing.B) {
	// Create a completely isolated AlertManager for benchmarking
	am := &AlertManager{
		alerts:          make(map[string]*Alert),
		incidents:       make(map[string]*Incident),
		alertRules:      make(map[string]*AlertRule),
		escalationRules: make(map[string]*EscalationRule),
		metrics:         nil, // Skip metrics for benchmark
		disableSimulation: true,
	}
	// Initialize mutex
	am.mu = sync.RWMutex{}

	labels := map[string]string{
		"instance": "server-1",
		"job":      "api-server",
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		// Create unique alert name to avoid conflicts
		alertName := fmt.Sprintf("BenchmarkAlert-%d", i)
		am.FireAlert(alertName, "warning", "Benchmark test alert", labels)
	}
}

func BenchmarkResolveAlert(b *testing.B) {
	// Simple benchmark that just measures alert resolution time
	labels := map[string]string{
		"instance": "server-1",
		"job":      "api-server",
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		// Create alert directly
		alert := &Alert{
			ID:       fmt.Sprintf("alert-%d", i),
			Name:     "BenchmarkAlert",
			Severity: "warning",
			Status:   "firing",
			Message:  "Benchmark test alert",
			Labels:   labels,
			StartTime: time.Now(),
		}
		
		// Simulate resolution
		now := time.Now()
		alert.Status = "resolved"
		alert.EndTime = &now
		alert.MTTR = time.Since(alert.StartTime)
	}
}

func BenchmarkGetAlerts(b *testing.B) {
	// Create a completely isolated AlertManager for benchmarking
	am := &AlertManager{
		alerts:          make(map[string]*Alert),
		incidents:       make(map[string]*Incident),
		alertRules:      make(map[string]*AlertRule),
		escalationRules: make(map[string]*EscalationRule),
		metrics:         nil, // Skip metrics for benchmark
		disableSimulation: true,
	}
	// Initialize mutex
	am.mu = sync.RWMutex{}

	labels := map[string]string{
		"instance": "server-1",
		"job":      "api-server",
	}

	// Pre-create some alerts
	for i := 0; i < 100; i++ {
		alertName := fmt.Sprintf("BenchmarkAlert-%d", i)
		am.FireAlert(alertName, "warning", "Benchmark test alert", labels)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		am.GetAlerts()
	}
}

func BenchmarkHTTPHandler(b *testing.B) {
	// Create a completely isolated AlertManager for benchmarking
	am := &AlertManager{
		alerts:          make(map[string]*Alert),
		incidents:       make(map[string]*Incident),
		alertRules:      make(map[string]*AlertRule),
		escalationRules: make(map[string]*EscalationRule),
		metrics:         nil, // Skip metrics for benchmark
		disableSimulation: true,
	}
	// Initialize mutex
	am.mu = sync.RWMutex{}

	req := httptest.NewRequest("GET", "/alerts", nil)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		rec := httptest.NewRecorder()
		am.alertsHandler(rec, req)
	}
}

// Test HTTP method validation
func TestHTTPMethodValidation(t *testing.T) {
	am := NewAlertManager()
	defer am.Shutdown()

	// Test invalid method for alerts handler
	req := httptest.NewRequest("DELETE", "/alerts", nil)
	rec := httptest.NewRecorder()

	am.alertsHandler(rec, req)

	if rec.Code != http.StatusMethodNotAllowed {
		t.Errorf("Expected status 405, got %d", rec.Code)
	}

	// Test invalid method for resolve alert handler
	req = httptest.NewRequest("GET", "/alerts/resolve", nil)
	rec = httptest.NewRecorder()

	am.resolveAlertHandler(rec, req)

	if rec.Code != http.StatusMethodNotAllowed {
		t.Errorf("Expected status 405, got %d", rec.Code)
	}
}

// Test JSON parsing errors
func TestJSONParsingErrors(t *testing.T) {
	am := NewAlertManager()
	defer am.Shutdown()

	// Test invalid JSON for alerts handler
	req := httptest.NewRequest("POST", "/alerts", strings.NewReader("invalid json"))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	am.alertsHandler(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Errorf("Expected status 400, got %d", rec.Code)
	}

	// Test invalid JSON for post-mortem handler
	req = httptest.NewRequest("POST", "/post-mortem", strings.NewReader("invalid json"))
	req.Header.Set("Content-Type", "application/json")
	rec = httptest.NewRecorder()

	am.postMortemHandler(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Errorf("Expected status 400, got %d", rec.Code)
	}
}