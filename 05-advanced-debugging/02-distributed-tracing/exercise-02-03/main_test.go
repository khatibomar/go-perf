package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"
)

// Test MetricsCollector initialization
func TestNewMetricsCollector(t *testing.T) {
	mc := NewMetricsCollector("test", "api")
	if mc == nil {
		t.Fatal("MetricsCollector should not be nil")
	}

	if mc.registry == nil {
		t.Error("Registry should not be nil")
	}

	if mc.httpRequestsTotal == nil {
		t.Error("HTTP requests total metric should not be nil")
	}

	if mc.sliAvailability == nil {
		t.Error("SLI availability metric should not be nil")
	}

	if mc.sloCompliance == nil {
		t.Error("SLO compliance metric should not be nil")
	}
}

// Test HTTP metrics recording
func TestMetricsCollector_HTTPMetrics(t *testing.T) {
	mc := NewMetricsCollector("test", "api")

	// Test recording HTTP request
	mc.RecordHTTPRequest("GET", "/api/test", "200", 100*time.Millisecond)
	mc.RecordHTTPRequest("POST", "/api/test", "500", 200*time.Millisecond)

	// Test in-flight requests
	mc.IncHTTPRequestsInFlight()
	mc.IncHTTPRequestsInFlight()
	mc.DecHTTPRequestsInFlight()

	// No panics should occur
}

// Test SLI/SLO metrics
func TestMetricsCollector_SLIMetrics(t *testing.T) {
	mc := NewMetricsCollector("test", "api")

	// Test SLI updates
	mc.UpdateSLI("api", "/test", 99.5, 0.05, 0.01, 100.0)
	mc.UpdateSLI("api", "/health", 99.9, 0.02, 0.005, 150.0)

	// Test SLO compliance updates
	mc.UpdateSLOCompliance("api", "availability", 98.5)
	mc.UpdateSLOCompliance("api", "latency", 95.0)
	mc.UpdateSLOCompliance("api", "error_rate", 99.0)

	// No panics should occur
}

// Test business metrics
func TestMetricsCollector_BusinessMetrics(t *testing.T) {
	mc := NewMetricsCollector("test", "api")

	// Test business operation recording
	mc.RecordBusinessOperation("process_order", "success", 50*time.Millisecond)
	mc.RecordBusinessOperation("validate_payment", "error", 30*time.Millisecond)
	mc.RecordBusinessOperation("update_inventory", "success", 20*time.Millisecond)

	// No panics should occur
}

// Test dashboard metrics
func TestMetricsCollector_DashboardMetrics(t *testing.T) {
	mc := NewMetricsCollector("test", "api")

	// Test dashboard load recording
	mc.RecordDashboardLoad("main", 500*time.Millisecond, true)
	mc.RecordDashboardLoad("sli-slo", 1*time.Second, false)

	// Test alert recording
	mc.RecordAlert("HighErrorRate", "critical")
	mc.RecordAlert("HighLatency", "warning")

	// No panics should occur
}

// Test HTTP middleware
func TestMetricsCollector_HTTPMiddleware(t *testing.T) {
	mc := NewMetricsCollector("test", "api")

	// Create a test handler
	testHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("OK"))
	})

	// Wrap with metrics middleware
	middleware := mc.HTTPMiddleware(testHandler)

	// Create test request
	req := httptest.NewRequest("GET", "/test", nil)
	rec := httptest.NewRecorder()

	// Execute request
	middleware.ServeHTTP(rec, req)

	// Verify response
	if rec.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", rec.Code)
	}

	if rec.Body.String() != "OK" {
		t.Errorf("Expected body 'OK', got '%s'", rec.Body.String())
	}
}

// Test DashboardManager initialization
func TestNewDashboardManager(t *testing.T) {
	mc := NewMetricsCollector("test", "api")
	dm := NewDashboardManager(mc)

	if dm == nil {
		t.Fatal("DashboardManager should not be nil")
	}

	if dm.metrics != mc {
		t.Error("DashboardManager should reference the provided metrics collector")
	}

	// Check that default dashboards are initialized
	dashboards := dm.ListDashboards()
	if len(dashboards) == 0 {
		t.Error("Default dashboards should be initialized")
	}

	// Check for expected dashboards
	expectedDashboards := []string{"main", "sli-slo"}
	for _, expected := range expectedDashboards {
		found := false
		for _, dashboard := range dashboards {
			if dashboard == expected {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("Expected dashboard '%s' not found", expected)
		}
	}
}

// Test dashboard retrieval
func TestDashboardManager_GetDashboard(t *testing.T) {
	mc := NewMetricsCollector("test", "api")
	dm := NewDashboardManager(mc)

	// Test getting existing dashboard
	dashboard, exists := dm.GetDashboard("main")
	if !exists {
		t.Error("Main dashboard should exist")
	}

	if dashboard.Title != "Application Overview" {
		t.Errorf("Expected title 'Application Overview', got '%s'", dashboard.Title)
	}

	if len(dashboard.Panels) == 0 {
		t.Error("Dashboard should have panels")
	}

	// Test getting non-existing dashboard
	_, exists = dm.GetDashboard("nonexistent")
	if exists {
		t.Error("Non-existent dashboard should not exist")
	}
}

// Test SLO configuration
func TestDashboardManager_GetSLO(t *testing.T) {
	mc := NewMetricsCollector("test", "api")
	dm := NewDashboardManager(mc)

	// Test getting existing SLO
	slo, exists := dm.GetSLO("availability")
	if !exists {
		t.Error("Availability SLO should exist")
	}

	if slo.Name != "API Availability" {
		t.Errorf("Expected SLO name 'API Availability', got '%s'", slo.Name)
	}

	if slo.Objective != 99.9 {
		t.Errorf("Expected objective 99.9, got %f", slo.Objective)
	}

	if len(slo.SLIs) == 0 {
		t.Error("SLO should have SLIs")
	}

	// Test getting non-existing SLO
	_, exists = dm.GetSLO("nonexistent")
	if exists {
		t.Error("Non-existent SLO should not exist")
	}
}

// Test alert rules
func TestDashboardManager_GetAlertRules(t *testing.T) {
	mc := NewMetricsCollector("test", "api")
	dm := NewDashboardManager(mc)

	// Test getting existing alert rules
	rules, exists := dm.GetAlertRules("main")
	if !exists {
		t.Error("Main alert rules should exist")
	}

	if len(rules) == 0 {
		t.Error("Should have alert rules")
	}

	// Check for expected alerts
	expectedAlerts := []string{"HighErrorRate", "HighLatency", "SLOViolation"}
	for _, expected := range expectedAlerts {
		found := false
		for _, rule := range rules {
			if rule.Alert == expected {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("Expected alert rule '%s' not found", expected)
		}
	}

	// Test getting non-existing alert rules
	_, exists = dm.GetAlertRules("nonexistent")
	if exists {
		t.Error("Non-existent alert rules should not exist")
	}
}

// Test BusinessService
func TestNewBusinessService(t *testing.T) {
	mc := NewMetricsCollector("test", "api")
	bs := NewBusinessService(mc)

	if bs == nil {
		t.Fatal("BusinessService should not be nil")
	}

	if bs.metrics != mc {
		t.Error("BusinessService should reference the provided metrics collector")
	}
}

func TestBusinessService_ProcessOrder(t *testing.T) {
	mc := NewMetricsCollector("test", "api")
	bs := NewBusinessService(mc)

	// Test successful order processing (most of the time)
	for i := 0; i < 10; i++ {
		err := bs.ProcessOrder(fmt.Sprintf("order-%d", i))
		// Error rate is 5%, so most should succeed
		if err != nil {
			t.Logf("Order processing failed (expected occasionally): %v", err)
		}
	}
}

func TestBusinessService_ValidatePayment(t *testing.T) {
	mc := NewMetricsCollector("test", "api")
	bs := NewBusinessService(mc)

	// Test payment validation (most of the time)
	for i := 0; i < 10; i++ {
		err := bs.ValidatePayment(fmt.Sprintf("payment-%d", i))
		// Error rate is 3%, so most should succeed
		if err != nil {
			t.Logf("Payment validation failed (expected occasionally): %v", err)
		}
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

	if response["version"] != "1.0.0" {
		t.Errorf("Expected version '1.0.0', got '%v'", response["version"])
	}
}

func TestDashboardHandler(t *testing.T) {
	mc := NewMetricsCollector("test", "api")
	dm := NewDashboardManager(mc)
	handler := dashboardHandler(dm)

	// Test getting main dashboard
	req := httptest.NewRequest("GET", "/dashboard?name=main", nil)
	rec := httptest.NewRecorder()

	handler(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", rec.Code)
	}

	var dashboard Dashboard
	if err := json.Unmarshal(rec.Body.Bytes(), &dashboard); err != nil {
		t.Fatalf("Failed to parse JSON response: %v", err)
	}

	if dashboard.Title != "Application Overview" {
		t.Errorf("Expected title 'Application Overview', got '%s'", dashboard.Title)
	}

	// Test getting non-existent dashboard
	req = httptest.NewRequest("GET", "/dashboard?name=nonexistent", nil)
	rec = httptest.NewRecorder()

	handler(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Errorf("Expected status 404, got %d", rec.Code)
	}
}

func TestDashboardListHandler(t *testing.T) {
	mc := NewMetricsCollector("test", "api")
	dm := NewDashboardManager(mc)
	handler := dashboardListHandler(dm)

	req := httptest.NewRequest("GET", "/dashboards", nil)
	rec := httptest.NewRecorder()

	handler(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", rec.Code)
	}

	var response map[string]interface{}
	if err := json.Unmarshal(rec.Body.Bytes(), &response); err != nil {
		t.Fatalf("Failed to parse JSON response: %v", err)
	}

	dashboards, ok := response["dashboards"].([]interface{})
	if !ok {
		t.Error("Response should contain dashboards array")
	}

	if len(dashboards) == 0 {
		t.Error("Should have at least one dashboard")
	}

	count, ok := response["count"].(float64)
	if !ok {
		t.Error("Response should contain count")
	}

	if int(count) != len(dashboards) {
		t.Errorf("Count should match dashboards length: %d != %d", int(count), len(dashboards))
	}
}

func TestSLOHandler(t *testing.T) {
	mc := NewMetricsCollector("test", "api")
	dm := NewDashboardManager(mc)
	handler := sloHandler(dm)

	// Test getting specific SLO
	req := httptest.NewRequest("GET", "/slo?name=availability", nil)
	rec := httptest.NewRecorder()

	handler(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", rec.Code)
	}

	var slo SLOConfig
	if err := json.Unmarshal(rec.Body.Bytes(), &slo); err != nil {
		t.Fatalf("Failed to parse JSON response: %v", err)
	}

	if slo.Name != "API Availability" {
		t.Errorf("Expected SLO name 'API Availability', got '%s'", slo.Name)
	}

	// Test getting all SLOs
	req = httptest.NewRequest("GET", "/slo", nil)
	rec = httptest.NewRecorder()

	handler(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", rec.Code)
	}

	var slos map[string]SLOConfig
	if err := json.Unmarshal(rec.Body.Bytes(), &slos); err != nil {
		t.Fatalf("Failed to parse JSON response: %v", err)
	}

	if len(slos) == 0 {
		t.Error("Should have at least one SLO")
	}
}

func TestAlertRulesHandler(t *testing.T) {
	mc := NewMetricsCollector("test", "api")
	dm := NewDashboardManager(mc)
	handler := alertRulesHandler(dm)

	req := httptest.NewRequest("GET", "/alerts?group=main", nil)
	rec := httptest.NewRecorder()

	handler(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", rec.Code)
	}

	var alertGroup AlertGroup
	if err := json.Unmarshal(rec.Body.Bytes(), &alertGroup); err != nil {
		t.Fatalf("Failed to parse JSON response: %v", err)
	}

	if alertGroup.Name != "main" {
		t.Errorf("Expected group name 'main', got '%s'", alertGroup.Name)
	}

	if len(alertGroup.Rules) == 0 {
		t.Error("Should have at least one alert rule")
	}
}

func TestOrderHandler(t *testing.T) {
	mc := NewMetricsCollector("test", "api")
	bs := NewBusinessService(mc)
	handler := orderHandler(bs)

	// Test with specific order ID
	req := httptest.NewRequest("GET", "/order?id=test-order-123", nil)
	rec := httptest.NewRecorder()

	handler(rec, req)

	// Should succeed most of the time (95% success rate)
	if rec.Code != http.StatusOK && rec.Code != http.StatusInternalServerError {
		t.Errorf("Expected status 200 or 500, got %d", rec.Code)
	}

	if rec.Code == http.StatusOK {
		var response map[string]interface{}
		if err := json.Unmarshal(rec.Body.Bytes(), &response); err != nil {
			t.Fatalf("Failed to parse JSON response: %v", err)
		}

		if response["order_id"] != "test-order-123" {
			t.Errorf("Expected order_id 'test-order-123', got '%v'", response["order_id"])
		}

		if response["status"] != "processed" {
			t.Errorf("Expected status 'processed', got '%v'", response["status"])
		}
	}
}

func TestPaymentHandler(t *testing.T) {
	mc := NewMetricsCollector("test", "api")
	bs := NewBusinessService(mc)
	handler := paymentHandler(bs)

	// Test with specific payment ID
	req := httptest.NewRequest("GET", "/payment?id=test-payment-456", nil)
	rec := httptest.NewRecorder()

	handler(rec, req)

	// Should succeed most of the time (97% success rate)
	if rec.Code != http.StatusOK && rec.Code != http.StatusInternalServerError {
		t.Errorf("Expected status 200 or 500, got %d", rec.Code)
	}

	if rec.Code == http.StatusOK {
		var response map[string]interface{}
		if err := json.Unmarshal(rec.Body.Bytes(), &response); err != nil {
			t.Fatalf("Failed to parse JSON response: %v", err)
		}

		if response["payment_id"] != "test-payment-456" {
			t.Errorf("Expected payment_id 'test-payment-456', got '%v'", response["payment_id"])
		}

		if response["status"] != "validated" {
			t.Errorf("Expected status 'validated', got '%v'", response["status"])
		}
	}
}

func TestLoadTestHandler(t *testing.T) {
	mc := NewMetricsCollector("test", "api")
	bs := NewBusinessService(mc)
	handler := loadTestHandler(bs)

	// Test with small number of operations
	req := httptest.NewRequest("GET", "/loadtest?operations=10", nil)
	rec := httptest.NewRecorder()

	handler(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", rec.Code)
	}

	var response map[string]interface{}
	if err := json.Unmarshal(rec.Body.Bytes(), &response); err != nil {
		t.Fatalf("Failed to parse JSON response: %v", err)
	}

	if response["operations"] != float64(10) {
		t.Errorf("Expected operations 10, got %v", response["operations"])
	}

	if _, ok := response["duration_ms"]; !ok {
		t.Error("Response should contain duration_ms")
	}

	if _, ok := response["success_rate"]; !ok {
		t.Error("Response should contain success_rate")
	}

	if _, ok := response["ops_per_sec"]; !ok {
		t.Error("Response should contain ops_per_sec")
	}
}

// Test concurrent access to metrics
func TestConcurrentMetrics(t *testing.T) {
	mc := NewMetricsCollector("test", "api")
	var wg sync.WaitGroup

	// Start multiple goroutines recording metrics
	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			for j := 0; j < 100; j++ {
				mc.RecordHTTPRequest("GET", fmt.Sprintf("/test-%d", id), "200", time.Millisecond)
				mc.RecordBusinessOperation(fmt.Sprintf("op-%d", id), "success", time.Millisecond)
				mc.UpdateSLI("api", fmt.Sprintf("/test-%d", id), 99.0, 0.05, 0.01, 100.0)
			}
		}(i)
	}

	wg.Wait()
	// Should complete without panics or race conditions
}

// Test concurrent dashboard access
func TestConcurrentDashboardAccess(t *testing.T) {
	mc := NewMetricsCollector("test", "api")
	dm := NewDashboardManager(mc)
	var wg sync.WaitGroup

	// Start multiple goroutines accessing dashboards
	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := 0; j < 50; j++ {
				_, _ = dm.GetDashboard("main")
				_ = dm.ListDashboards()
				_, _ = dm.GetSLO("availability")
				_, _ = dm.GetAlertRules("main")
			}
		}()
	}

	wg.Wait()
	// Should complete without panics or race conditions
}

// Performance test for metrics recording
func BenchmarkMetricsRecording(b *testing.B) {
	mc := NewMetricsCollector("test", "api")

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			mc.RecordHTTPRequest("GET", "/test", "200", time.Millisecond)
		}
	})
}

// Performance test for dashboard access
func BenchmarkDashboardAccess(b *testing.B) {
	mc := NewMetricsCollector("test", "api")
	dm := NewDashboardManager(mc)

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			_, _ = dm.GetDashboard("main")
		}
	})
}

// Test utility functions
func TestUtilityFunctions(t *testing.T) {
	// Test responseWriter
	rec := httptest.NewRecorder()
	rw := &responseWriter{ResponseWriter: rec, statusCode: 200}

	rw.WriteHeader(404)
	if rw.statusCode != 404 {
		t.Errorf("Expected status code 404, got %d", rw.statusCode)
	}

	rw.Write([]byte("test"))
	if rec.Body.String() != "test" {
		t.Errorf("Expected body 'test', got '%s'", rec.Body.String())
	}
}

// Test dashboard configuration validation
func TestDashboardConfiguration(t *testing.T) {
	mc := NewMetricsCollector("test", "api")
	dm := NewDashboardManager(mc)

	// Test main dashboard configuration
	mainDashboard, exists := dm.GetDashboard("main")
	if !exists {
		t.Fatal("Main dashboard should exist")
	}

	// Validate dashboard structure
	if mainDashboard.ID != 1 {
		t.Errorf("Expected dashboard ID 1, got %d", mainDashboard.ID)
	}

	if len(mainDashboard.Panels) == 0 {
		t.Error("Dashboard should have panels")
	}

	if mainDashboard.Refresh != "5s" {
		t.Errorf("Expected refresh '5s', got '%s'", mainDashboard.Refresh)
	}

	// Validate panel structure
	for _, panel := range mainDashboard.Panels {
		if panel.Title == "" {
			t.Error("Panel should have a title")
		}

		if panel.Type == "" {
			t.Error("Panel should have a type")
		}

		if len(panel.Targets) == 0 {
			t.Error("Panel should have targets")
		}

		for _, target := range panel.Targets {
			if target.Expr == "" {
				t.Error("Target should have an expression")
			}

			if target.RefID == "" {
				t.Error("Target should have a RefID")
			}
		}
	}
}

// Test SLO configuration validation
func TestSLOConfiguration(t *testing.T) {
	mc := NewMetricsCollector("test", "api")
	dm := NewDashboardManager(mc)

	// Test availability SLO
	availabilitySLO, exists := dm.GetSLO("availability")
	if !exists {
		t.Fatal("Availability SLO should exist")
	}

	if availabilitySLO.Objective != 99.9 {
		t.Errorf("Expected objective 99.9, got %f", availabilitySLO.Objective)
	}

	if len(availabilitySLO.SLIs) == 0 {
		t.Error("SLO should have SLIs")
	}

	for _, sli := range availabilitySLO.SLIs {
		if sli.Name == "" {
			t.Error("SLI should have a name")
		}

		if sli.Query == "" {
			t.Error("SLI should have a query")
		}

		if sli.Window == "" {
			t.Error("SLI should have a window")
		}
	}
}

// Test alert rule validation
func TestAlertRuleValidation(t *testing.T) {
	mc := NewMetricsCollector("test", "api")
	dm := NewDashboardManager(mc)

	rules, exists := dm.GetAlertRules("main")
	if !exists {
		t.Fatal("Main alert rules should exist")
	}

	for _, rule := range rules {
		if rule.Alert == "" {
			t.Error("Alert rule should have a name")
		}

		if rule.Expr == "" {
			t.Error("Alert rule should have an expression")
		}

		if rule.For == "" {
			t.Error("Alert rule should have a duration")
		}

		if len(rule.Labels) == 0 {
			t.Error("Alert rule should have labels")
		}

		if len(rule.Annotations) == 0 {
			t.Error("Alert rule should have annotations")
		}

		// Validate required labels
		if _, ok := rule.Labels["severity"]; !ok {
			t.Error("Alert rule should have severity label")
		}

		// Validate required annotations
		if _, ok := rule.Annotations["summary"]; !ok {
			t.Error("Alert rule should have summary annotation")
		}

		if _, ok := rule.Annotations["description"]; !ok {
			t.Error("Alert rule should have description annotation")
		}
	}
}

// Test metrics endpoint
func TestMetricsEndpoint(t *testing.T) {
	mc := NewMetricsCollector("test", "api")

	// Record some metrics
	mc.RecordHTTPRequest("GET", "/test", "200", 100*time.Millisecond)
	mc.UpdateSLI("api", "/test", 99.5, 0.05, 0.01, 100.0)
	mc.UpdateSLOCompliance("api", "availability", 98.5)

	// Create metrics handler
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		promHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Simulate metrics endpoint
			w.Header().Set("Content-Type", "text/plain")
			w.WriteHeader(http.StatusOK)
			w.Write([]byte("# HELP test_api_http_requests_total Total number of HTTP requests\n"))
			w.Write([]byte("# TYPE test_api_http_requests_total counter\n"))
			w.Write([]byte("test_api_http_requests_total{endpoint=\"/test\",method=\"GET\",status_code=\"200\"} 1\n"))
		})
		promHandler.ServeHTTP(w, r)
	})

	req := httptest.NewRequest("GET", "/metrics", nil)
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", rec.Code)
	}

	if !strings.Contains(rec.Body.String(), "test_api_http_requests_total") {
		t.Error("Metrics output should contain HTTP requests metric")
	}
}