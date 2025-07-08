package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"math"
	"math/rand"
	"net/http"
	"os"
	"os/signal"
	"runtime"
	"strconv"
	"sync"
	"syscall"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

// SLI/SLO Configuration
type SLIConfig struct {
	Name        string  `json:"name"`
	Description string  `json:"description"`
	Target      float64 `json:"target"`
	Window      string  `json:"window"`
	Query       string  `json:"query"`
}

type SLOConfig struct {
	Name        string      `json:"name"`
	Description string      `json:"description"`
	Objective   float64     `json:"objective"`
	SLIs        []SLIConfig `json:"slis"`
}

// Dashboard Configuration
type DashboardPanel struct {
	ID          int                    `json:"id"`
	Title       string                 `json:"title"`
	Type        string                 `json:"type"`
	Targets     []DashboardTarget      `json:"targets"`
	FieldConfig map[string]interface{} `json:"fieldConfig"`
	Options     map[string]interface{} `json:"options"`
	GridPos     map[string]int         `json:"gridPos"`
}

type DashboardTarget struct {
	Expr         string `json:"expr"`
	LegendFormat string `json:"legendFormat"`
	RefID        string `json:"refId"`
}

type Dashboard struct {
	ID          int               `json:"id"`
	Title       string            `json:"title"`
	Description string            `json:"description"`
	Tags        []string          `json:"tags"`
	Panels      []DashboardPanel  `json:"panels"`
	Time        map[string]string `json:"time"`
	Refresh     string            `json:"refresh"`
}

// Alert Rule Configuration
type AlertRule struct {
	Alert       string            `json:"alert"`
	Expr        string            `json:"expr"`
	For         string            `json:"for"`
	Labels      map[string]string `json:"labels"`
	Annotations map[string]string `json:"annotations"`
}

type AlertGroup struct {
	Name  string      `json:"name"`
	Rules []AlertRule `json:"rules"`
}

// Metrics Collector with SLI/SLO tracking
type MetricsCollector struct {
	// HTTP metrics
	httpRequestsTotal    *prometheus.CounterVec
	httpRequestDuration  *prometheus.HistogramVec
	httpRequestsInFlight prometheus.Gauge
	httpErrorRate        *prometheus.CounterVec

	// SLI metrics
	sliAvailability     *prometheus.GaugeVec
	sliLatency          *prometheus.GaugeVec
	sliErrorRate        *prometheus.GaugeVec
	sliThroughput       *prometheus.GaugeVec
	sloCompliance       *prometheus.GaugeVec

	// Business metrics
	businessOpsTotal    *prometheus.CounterVec
	businessOpsDuration *prometheus.HistogramVec
	businessErrors      *prometheus.CounterVec

	// System metrics
	systemCPUUsage      prometheus.Gauge
	systemMemoryUsage   prometheus.Gauge
	goroutineCount      prometheus.Gauge
	gcDuration          *prometheus.HistogramVec

	// Dashboard metrics
	dashboardLoadTime   *prometheus.HistogramVec
	dashboardRequests   *prometheus.CounterVec
	alertsFired         *prometheus.CounterVec

	registry *prometheus.Registry
	mu       sync.RWMutex
}

func NewMetricsCollector(namespace, subsystem string) *MetricsCollector {
	registry := prometheus.NewRegistry()
	factory := promauto.With(registry)

	mc := &MetricsCollector{
		httpRequestsTotal: factory.NewCounterVec(
			prometheus.CounterOpts{
				Namespace: namespace,
				Subsystem: subsystem,
				Name:      "http_requests_total",
				Help:      "Total number of HTTP requests",
			},
			[]string{"method", "endpoint", "status_code"},
		),

		httpRequestDuration: factory.NewHistogramVec(
			prometheus.HistogramOpts{
				Namespace: namespace,
				Subsystem: subsystem,
				Name:      "http_request_duration_seconds",
				Help:      "HTTP request duration in seconds",
				Buckets:   []float64{0.001, 0.005, 0.01, 0.025, 0.05, 0.1, 0.25, 0.5, 1, 2.5, 5},
			},
			[]string{"method", "endpoint"},
		),

		httpRequestsInFlight: factory.NewGauge(
			prometheus.GaugeOpts{
				Namespace: namespace,
				Subsystem: subsystem,
				Name:      "http_requests_in_flight",
				Help:      "Number of HTTP requests currently being processed",
			},
		),

		httpErrorRate: factory.NewCounterVec(
			prometheus.CounterOpts{
				Namespace: namespace,
				Subsystem: subsystem,
				Name:      "http_error_rate",
				Help:      "HTTP error rate",
			},
			[]string{"endpoint", "error_type"},
		),

		// SLI metrics
		sliAvailability: factory.NewGaugeVec(
			prometheus.GaugeOpts{
				Namespace: namespace,
				Subsystem: "sli",
				Name:      "availability",
				Help:      "Service availability SLI",
			},
			[]string{"service", "endpoint"},
		),

		sliLatency: factory.NewGaugeVec(
			prometheus.GaugeOpts{
				Namespace: namespace,
				Subsystem: "sli",
				Name:      "latency_p95",
				Help:      "95th percentile latency SLI",
			},
			[]string{"service", "endpoint"},
		),

		sliErrorRate: factory.NewGaugeVec(
			prometheus.GaugeOpts{
				Namespace: namespace,
				Subsystem: "sli",
				Name:      "error_rate",
				Help:      "Error rate SLI",
			},
			[]string{"service", "endpoint"},
		),

		sliThroughput: factory.NewGaugeVec(
			prometheus.GaugeOpts{
				Namespace: namespace,
				Subsystem: "sli",
				Name:      "throughput",
				Help:      "Throughput SLI",
			},
			[]string{"service", "endpoint"},
		),

		sloCompliance: factory.NewGaugeVec(
			prometheus.GaugeOpts{
				Namespace: namespace,
				Subsystem: "slo",
				Name:      "compliance",
				Help:      "SLO compliance percentage",
			},
			[]string{"service", "slo_name"},
		),

		// Business metrics
		businessOpsTotal: factory.NewCounterVec(
			prometheus.CounterOpts{
				Namespace: namespace,
				Subsystem: "business",
				Name:      "operations_total",
				Help:      "Total business operations",
			},
			[]string{"operation", "status"},
		),

		businessOpsDuration: factory.NewHistogramVec(
			prometheus.HistogramOpts{
				Namespace: namespace,
				Subsystem: "business",
				Name:      "operation_duration_seconds",
				Help:      "Business operation duration",
				Buckets:   []float64{0.001, 0.005, 0.01, 0.025, 0.05, 0.1, 0.25, 0.5, 1, 2.5, 5, 10},
			},
			[]string{"operation"},
		),

		businessErrors: factory.NewCounterVec(
			prometheus.CounterOpts{
				Namespace: namespace,
				Subsystem: "business",
				Name:      "errors_total",
				Help:      "Total business errors",
			},
			[]string{"operation", "error_type"},
		),

		// System metrics
		systemCPUUsage: factory.NewGauge(
			prometheus.GaugeOpts{
				Namespace: namespace,
				Subsystem: "system",
				Name:      "cpu_usage_percent",
				Help:      "System CPU usage percentage",
			},
		),

		systemMemoryUsage: factory.NewGauge(
			prometheus.GaugeOpts{
				Namespace: namespace,
				Subsystem: "system",
				Name:      "memory_usage_bytes",
				Help:      "System memory usage in bytes",
			},
		),

		goroutineCount: factory.NewGauge(
			prometheus.GaugeOpts{
				Namespace: namespace,
				Subsystem: "system",
				Name:      "goroutines_count",
				Help:      "Number of goroutines",
			},
		),

		gcDuration: factory.NewHistogramVec(
			prometheus.HistogramOpts{
				Namespace: namespace,
				Subsystem: "system",
				Name:      "gc_duration_seconds",
				Help:      "Garbage collection duration",
				Buckets:   []float64{0.0001, 0.0005, 0.001, 0.005, 0.01, 0.05, 0.1, 0.5, 1},
			},
			[]string{"gc_type"},
		),

		// Dashboard metrics
		dashboardLoadTime: factory.NewHistogramVec(
			prometheus.HistogramOpts{
				Namespace: namespace,
				Subsystem: "dashboard",
				Name:      "load_time_seconds",
				Help:      "Dashboard load time",
				Buckets:   []float64{0.1, 0.25, 0.5, 1, 2, 5, 10},
			},
			[]string{"dashboard_name"},
		),

		dashboardRequests: factory.NewCounterVec(
			prometheus.CounterOpts{
				Namespace: namespace,
				Subsystem: "dashboard",
				Name:      "requests_total",
				Help:      "Dashboard requests",
			},
			[]string{"dashboard_name", "status"},
		),

		alertsFired: factory.NewCounterVec(
			prometheus.CounterOpts{
				Namespace: namespace,
				Subsystem: "alerts",
				Name:      "fired_total",
				Help:      "Total alerts fired",
			},
			[]string{"alert_name", "severity"},
		),

		registry: registry,
	}

	// Start system metrics collection
	go mc.collectSystemMetrics()
	go mc.calculateSLIs()

	return mc
}

// HTTP Metrics
func (mc *MetricsCollector) RecordHTTPRequest(method, endpoint, statusCode string, duration time.Duration) {
	mc.httpRequestsTotal.WithLabelValues(method, endpoint, statusCode).Inc()
	mc.httpRequestDuration.WithLabelValues(method, endpoint).Observe(duration.Seconds())

	// Record error if status code >= 400
	if code, _ := strconv.Atoi(statusCode); code >= 400 {
		errorType := "client_error"
		if code >= 500 {
			errorType = "server_error"
		}
		mc.httpErrorRate.WithLabelValues(endpoint, errorType).Inc()
	}
}

func (mc *MetricsCollector) IncHTTPRequestsInFlight() {
	mc.httpRequestsInFlight.Inc()
}

func (mc *MetricsCollector) DecHTTPRequestsInFlight() {
	mc.httpRequestsInFlight.Dec()
}

// Business Metrics
func (mc *MetricsCollector) RecordBusinessOperation(operation, status string, duration time.Duration) {
	mc.businessOpsTotal.WithLabelValues(operation, status).Inc()
	mc.businessOpsDuration.WithLabelValues(operation).Observe(duration.Seconds())

	if status == "error" {
		mc.businessErrors.WithLabelValues(operation, "business_logic").Inc()
	}
}

// SLI/SLO Methods
func (mc *MetricsCollector) UpdateSLI(service, endpoint string, availability, latencyP95, errorRate, throughput float64) {
	mc.sliAvailability.WithLabelValues(service, endpoint).Set(availability)
	mc.sliLatency.WithLabelValues(service, endpoint).Set(latencyP95)
	mc.sliErrorRate.WithLabelValues(service, endpoint).Set(errorRate)
	mc.sliThroughput.WithLabelValues(service, endpoint).Set(throughput)
}

func (mc *MetricsCollector) UpdateSLOCompliance(service, sloName string, compliance float64) {
	mc.sloCompliance.WithLabelValues(service, sloName).Set(compliance)
}

// Dashboard Metrics
func (mc *MetricsCollector) RecordDashboardLoad(dashboardName string, duration time.Duration, success bool) {
	mc.dashboardLoadTime.WithLabelValues(dashboardName).Observe(duration.Seconds())
	status := "success"
	if !success {
		status = "error"
	}
	mc.dashboardRequests.WithLabelValues(dashboardName, status).Inc()
}

func (mc *MetricsCollector) RecordAlert(alertName, severity string) {
	mc.alertsFired.WithLabelValues(alertName, severity).Inc()
}

// System metrics collection
func (mc *MetricsCollector) collectSystemMetrics() {
	ticker := time.NewTicker(15 * time.Second)
	defer ticker.Stop()

	for range ticker.C {
		// Collect goroutine count
		mc.goroutineCount.Set(float64(runtime.NumGoroutine()))

		// Collect memory stats
		var m runtime.MemStats
		runtime.ReadMemStats(&m)
		mc.systemMemoryUsage.Set(float64(m.Alloc))

		// Simulate CPU usage
		cpuUsage := 20 + rand.Float64()*30 // 20-50% CPU usage
		mc.systemCPUUsage.Set(cpuUsage)
	}
}

// Calculate SLIs based on metrics
func (mc *MetricsCollector) calculateSLIs() {
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	for range ticker.C {
		// Simulate SLI calculations
		availability := 99.5 + rand.Float64()*0.5 // 99.5-100%
		latencyP95 := 0.05 + rand.Float64()*0.1   // 50-150ms
		errorRate := rand.Float64() * 0.02        // 0-2%
		throughput := 100 + rand.Float64()*50     // 100-150 RPS

		mc.UpdateSLI("api", "/api/v1", availability, latencyP95, errorRate, throughput)

		// Calculate SLO compliance
		availabilitySLO := 99.9
		latencySLO := 0.1
		errorRateSLO := 0.01

		availabilityCompliance := math.Min(100, (availability/availabilitySLO)*100)
		latencyCompliance := math.Min(100, (latencySLO/latencyP95)*100)
		errorRateCompliance := math.Min(100, (errorRateSLO/errorRate)*100)

		mc.UpdateSLOCompliance("api", "availability", availabilityCompliance)
		mc.UpdateSLOCompliance("api", "latency", latencyCompliance)
		mc.UpdateSLOCompliance("api", "error_rate", errorRateCompliance)
	}
}

// HTTP Middleware
func (mc *MetricsCollector) HTTPMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		mc.IncHTTPRequestsInFlight()
		defer mc.DecHTTPRequestsInFlight()

		wrapped := &responseWriter{ResponseWriter: w, statusCode: 200}
		next.ServeHTTP(wrapped, r)

		duration := time.Since(start)
		mc.RecordHTTPRequest(
			r.Method,
			r.URL.Path,
			fmt.Sprintf("%d", wrapped.statusCode),
			duration,
		)
	})
}

type responseWriter struct {
	http.ResponseWriter
	statusCode int
}

func (rw *responseWriter) WriteHeader(code int) {
	rw.statusCode = code
	rw.ResponseWriter.WriteHeader(code)
}

// Dashboard Manager
type DashboardManager struct {
	dashboards map[string]Dashboard
	slos       map[string]SLOConfig
	alertRules map[string][]AlertRule
	metrics    *MetricsCollector
	mu         sync.RWMutex
}

func NewDashboardManager(metrics *MetricsCollector) *DashboardManager {
	dm := &DashboardManager{
		dashboards: make(map[string]Dashboard),
		slos:       make(map[string]SLOConfig),
		alertRules: make(map[string][]AlertRule),
		metrics:    metrics,
	}

	// Initialize default dashboards and SLOs
	dm.initializeDefaults()

	return dm
}

func (dm *DashboardManager) initializeDefaults() {
	// Create main application dashboard
	mainDashboard := Dashboard{
		ID:          1,
		Title:       "Application Overview",
		Description: "Main application monitoring dashboard",
		Tags:        []string{"application", "overview"},
		Panels: []DashboardPanel{
			{
				ID:    1,
				Title: "HTTP Request Rate",
				Type:  "graph",
				Targets: []DashboardTarget{
					{
						Expr:         "rate(monitoring_api_http_requests_total[5m])",
						LegendFormat: "{{method}} {{endpoint}}",
						RefID:        "A",
					},
				},
				GridPos: map[string]int{"h": 8, "w": 12, "x": 0, "y": 0},
			},
			{
				ID:    2,
				Title: "HTTP Request Duration",
				Type:  "graph",
				Targets: []DashboardTarget{
					{
						Expr:         "histogram_quantile(0.95, rate(monitoring_api_http_request_duration_seconds_bucket[5m]))",
						LegendFormat: "95th percentile",
						RefID:        "A",
					},
					{
						Expr:         "histogram_quantile(0.50, rate(monitoring_api_http_request_duration_seconds_bucket[5m]))",
						LegendFormat: "50th percentile",
						RefID:        "B",
					},
				},
				GridPos: map[string]int{"h": 8, "w": 12, "x": 12, "y": 0},
			},
			{
				ID:    3,
				Title: "Error Rate",
				Type:  "stat",
				Targets: []DashboardTarget{
					{
						Expr:         "rate(monitoring_api_http_requests_total{status_code=~\"5..\"}[5m]) / rate(monitoring_api_http_requests_total[5m]) * 100",
						LegendFormat: "Error Rate %",
						RefID:        "A",
					},
				},
				GridPos: map[string]int{"h": 4, "w": 6, "x": 0, "y": 8},
			},
			{
				ID:    4,
				Title: "System Memory Usage",
				Type:  "gauge",
				Targets: []DashboardTarget{
					{
						Expr:         "monitoring_system_memory_usage_bytes / 1024 / 1024",
						LegendFormat: "Memory MB",
						RefID:        "A",
					},
				},
				GridPos: map[string]int{"h": 4, "w": 6, "x": 6, "y": 8},
			},
		},
		Time:    map[string]string{"from": "now-1h", "to": "now"},
		Refresh: "5s",
	}

	// Create SLI/SLO dashboard
	sliDashboard := Dashboard{
		ID:          2,
		Title:       "SLI/SLO Monitoring",
		Description: "Service Level Indicators and Objectives monitoring",
		Tags:        []string{"sli", "slo", "reliability"},
		Panels: []DashboardPanel{
			{
				ID:    1,
				Title: "Service Availability SLI",
				Type:  "stat",
				Targets: []DashboardTarget{
					{
						Expr:         "monitoring_sli_availability",
						LegendFormat: "{{service}} {{endpoint}}",
						RefID:        "A",
					},
				},
				GridPos: map[string]int{"h": 4, "w": 6, "x": 0, "y": 0},
			},
			{
				ID:    2,
				Title: "Latency P95 SLI",
				Type:  "stat",
				Targets: []DashboardTarget{
					{
						Expr:         "monitoring_sli_latency_p95",
						LegendFormat: "{{service}} {{endpoint}}",
						RefID:        "A",
					},
				},
				GridPos: map[string]int{"h": 4, "w": 6, "x": 6, "y": 0},
			},
			{
				ID:    3,
				Title: "SLO Compliance",
				Type:  "bargauge",
				Targets: []DashboardTarget{
					{
						Expr:         "monitoring_slo_compliance",
						LegendFormat: "{{service}} {{slo_name}}",
						RefID:        "A",
					},
				},
				GridPos: map[string]int{"h": 8, "w": 12, "x": 0, "y": 4},
			},
		},
		Time:    map[string]string{"from": "now-6h", "to": "now"},
		Refresh: "30s",
	}

	dm.dashboards["main"] = mainDashboard
	dm.dashboards["sli-slo"] = sliDashboard

	// Initialize SLO configurations
	availabilitySLO := SLOConfig{
		Name:        "API Availability",
		Description: "API service availability target",
		Objective:   99.9,
		SLIs: []SLIConfig{
			{
				Name:        "Availability",
				Description: "Percentage of successful requests",
				Target:      99.9,
				Window:      "30d",
				Query:       "(1 - rate(monitoring_api_http_requests_total{status_code=~\"5..\"}[30d]) / rate(monitoring_api_http_requests_total[30d])) * 100",
			},
		},
	}

	latencySLO := SLOConfig{
		Name:        "API Latency",
		Description: "API response time target",
		Objective:   95.0,
		SLIs: []SLIConfig{
			{
				Name:        "Latency P95",
				Description: "95th percentile response time under 100ms",
				Target:      0.1,
				Window:      "7d",
				Query:       "histogram_quantile(0.95, rate(monitoring_api_http_request_duration_seconds_bucket[7d]))",
			},
		},
	}

	dm.slos["availability"] = availabilitySLO
	dm.slos["latency"] = latencySLO

	// Initialize alert rules
	alertRules := []AlertRule{
		{
			Alert: "HighErrorRate",
			Expr:  "rate(monitoring_api_http_requests_total{status_code=~\"5..\"}[5m]) / rate(monitoring_api_http_requests_total[5m]) > 0.05",
			For:   "2m",
			Labels: map[string]string{
				"severity": "critical",
				"team":     "platform",
			},
			Annotations: map[string]string{
				"summary":     "High error rate detected",
				"description": "Error rate is above 5% for more than 2 minutes",
				"runbook":     "https://runbooks.example.com/high-error-rate",
			},
		},
		{
			Alert: "HighLatency",
			Expr:  "histogram_quantile(0.95, rate(monitoring_api_http_request_duration_seconds_bucket[5m])) > 0.5",
			For:   "5m",
			Labels: map[string]string{
				"severity": "warning",
				"team":     "platform",
			},
			Annotations: map[string]string{
				"summary":     "High latency detected",
				"description": "95th percentile latency is above 500ms for more than 5 minutes",
				"runbook":     "https://runbooks.example.com/high-latency",
			},
		},
		{
			Alert: "SLOViolation",
			Expr:  "monitoring_slo_compliance < 95",
			For:   "10m",
			Labels: map[string]string{
				"severity": "critical",
				"team":     "sre",
			},
			Annotations: map[string]string{
				"summary":     "SLO violation detected",
				"description": "SLO compliance is below 95% for {{$labels.service}} {{$labels.slo_name}}",
				"runbook":     "https://runbooks.example.com/slo-violation",
			},
		},
	}

	dm.alertRules["main"] = alertRules
}

// Dashboard API handlers
func (dm *DashboardManager) GetDashboard(name string) (Dashboard, bool) {
	dm.mu.RLock()
	defer dm.mu.RUnlock()
	dashboard, exists := dm.dashboards[name]
	return dashboard, exists
}

func (dm *DashboardManager) ListDashboards() []string {
	dm.mu.RLock()
	defer dm.mu.RUnlock()
	names := make([]string, 0, len(dm.dashboards))
	for name := range dm.dashboards {
		names = append(names, name)
	}
	return names
}

func (dm *DashboardManager) GetSLO(name string) (SLOConfig, bool) {
	dm.mu.RLock()
	defer dm.mu.RUnlock()
	slo, exists := dm.slos[name]
	return slo, exists
}

func (dm *DashboardManager) GetAlertRules(group string) ([]AlertRule, bool) {
	dm.mu.RLock()
	defer dm.mu.RUnlock()
	rules, exists := dm.alertRules[group]
	return rules, exists
}

// Business Service
type BusinessService struct {
	metrics *MetricsCollector
}

func NewBusinessService(metrics *MetricsCollector) *BusinessService {
	return &BusinessService{
		metrics: metrics,
	}
}

func (bs *BusinessService) ProcessOrder(orderID string) error {
	start := time.Now()
	defer func() {
		duration := time.Since(start)
		status := "success"
		if rand.Float64() < 0.05 { // 5% error rate
			status = "error"
		}
		bs.metrics.RecordBusinessOperation("process_order", status, duration)
	}()

	// Simulate processing time
	processingTime := time.Duration(50+rand.Intn(100)) * time.Millisecond
	time.Sleep(processingTime)

	// Simulate occasional errors
	if rand.Float64() < 0.05 {
		return fmt.Errorf("order processing failed for order %s", orderID)
	}

	return nil
}

func (bs *BusinessService) ValidatePayment(paymentID string) error {
	start := time.Now()
	defer func() {
		duration := time.Since(start)
		status := "success"
		if rand.Float64() < 0.03 { // 3% error rate
			status = "error"
		}
		bs.metrics.RecordBusinessOperation("validate_payment", status, duration)
	}()

	// Simulate processing time
	processingTime := time.Duration(20+rand.Intn(50)) * time.Millisecond
	time.Sleep(processingTime)

	// Simulate occasional errors
	if rand.Float64() < 0.03 {
		return fmt.Errorf("payment validation failed for payment %s", paymentID)
	}

	return nil
}

// HTTP Handlers
func healthHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]interface{}{
		"status":    "healthy",
		"timestamp": time.Now().Unix(),
		"version":   "1.0.0",
	})
}

func dashboardHandler(dm *DashboardManager) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		dashboardName := r.URL.Query().Get("name")
		if dashboardName == "" {
			dashboardName = "main"
		}

		dashboard, exists := dm.GetDashboard(dashboardName)
		if !exists {
			http.Error(w, "Dashboard not found", http.StatusNotFound)
			dm.metrics.RecordDashboardLoad(dashboardName, time.Since(start), false)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(dashboard)
		dm.metrics.RecordDashboardLoad(dashboardName, time.Since(start), true)
	}
}

func dashboardListHandler(dm *DashboardManager) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		dashboards := dm.ListDashboards()

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"dashboards": dashboards,
			"count":      len(dashboards),
		})
		dm.metrics.RecordDashboardLoad("list", time.Since(start), true)
	}
}

func sloHandler(dm *DashboardManager) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		sloName := r.URL.Query().Get("name")
		if sloName == "" {
			// Return all SLOs
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(dm.slos)
			return
		}

		slo, exists := dm.GetSLO(sloName)
		if !exists {
			http.Error(w, "SLO not found", http.StatusNotFound)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(slo)
	}
}

func alertRulesHandler(dm *DashboardManager) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		group := r.URL.Query().Get("group")
		if group == "" {
			group = "main"
		}

		rules, exists := dm.GetAlertRules(group)
		if !exists {
			http.Error(w, "Alert rules not found", http.StatusNotFound)
			return
		}

		alertGroup := AlertGroup{
			Name:  group,
			Rules: rules,
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(alertGroup)
	}
}

func orderHandler(bs *BusinessService) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		orderID := r.URL.Query().Get("id")
		if orderID == "" {
			orderID = fmt.Sprintf("order-%d", time.Now().UnixNano())
		}

		err := bs.ProcessOrder(orderID)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"order_id": orderID,
			"status":   "processed",
			"timestamp": time.Now().Unix(),
		})
	}
}

func paymentHandler(bs *BusinessService) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		paymentID := r.URL.Query().Get("id")
		if paymentID == "" {
			paymentID = fmt.Sprintf("payment-%d", time.Now().UnixNano())
		}

		err := bs.ValidatePayment(paymentID)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"payment_id": paymentID,
			"status":     "validated",
			"timestamp":  time.Now().Unix(),
		})
	}
}

func loadTestHandler(bs *BusinessService) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		operationsStr := r.URL.Query().Get("operations")
		operations := 100
		if operationsStr != "" {
			if parsed, err := strconv.Atoi(operationsStr); err == nil {
				operations = parsed
			}
		}

		start := time.Now()
		var wg sync.WaitGroup
		errorCount := 0
		var mu sync.Mutex

		for i := 0; i < operations; i++ {
			wg.Add(1)
			go func(i int) {
				defer wg.Done()
				orderID := fmt.Sprintf("load-test-order-%d", i)
				if err := bs.ProcessOrder(orderID); err != nil {
					mu.Lock()
					errorCount++
					mu.Unlock()
				}
			}(i)
		}

		wg.Wait()
		duration := time.Since(start)

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"operations":    operations,
			"duration_ms":   duration.Milliseconds(),
			"errors":        errorCount,
			"success_rate":  float64(operations-errorCount) / float64(operations) * 100,
			"ops_per_sec":   float64(operations) / duration.Seconds(),
		})
	}
}

func simulateAlerts(metrics *MetricsCollector) {
	ticker := time.NewTicker(2 * time.Minute)
	defer ticker.Stop()

	alertTypes := []struct {
		name     string
		severity string
		prob     float64
	}{
		{"HighErrorRate", "critical", 0.1},
		{"HighLatency", "warning", 0.15},
		{"HighMemoryUsage", "warning", 0.05},
		{"SLOViolation", "critical", 0.03},
	}

	for range ticker.C {
		for _, alert := range alertTypes {
			if rand.Float64() < alert.prob {
				metrics.RecordAlert(alert.name, alert.severity)
			}
		}
	}
}

func main() {
	// Initialize metrics collector
	metrics := NewMetricsCollector("monitoring", "api")

	// Initialize dashboard manager
	dashboardManager := NewDashboardManager(metrics)

	// Initialize business service
	businessService := NewBusinessService(metrics)

	// Create HTTP server
	mux := http.NewServeMux()

	// Add metrics middleware
	handler := metrics.HTTPMiddleware(mux)

	// Register handlers
	mux.HandleFunc("/health", healthHandler)
	mux.HandleFunc("/dashboard", dashboardHandler(dashboardManager))
	mux.HandleFunc("/dashboards", dashboardListHandler(dashboardManager))
	mux.HandleFunc("/slo", sloHandler(dashboardManager))
	mux.HandleFunc("/alerts", alertRulesHandler(dashboardManager))
	mux.HandleFunc("/order", orderHandler(businessService))
	mux.HandleFunc("/payment", paymentHandler(businessService))
	mux.HandleFunc("/loadtest", loadTestHandler(businessService))
	mux.Handle("/metrics", promhttp.HandlerFor(metrics.registry, promhttp.HandlerOpts{}))

	// Start alert simulation
	go simulateAlerts(metrics)

	// Create server
	server := &http.Server{
		Addr:    ":8080",
		Handler: handler,
	}

	// Start server in goroutine
	go func() {
		log.Printf("Starting server on :8080")
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("Server failed to start: %v", err)
		}
	}()

	// Wait for interrupt signal
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
	<-sigChan

	// Graceful shutdown
	log.Println("Shutting down server...")
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if err := server.Shutdown(ctx); err != nil {
		log.Printf("Server shutdown error: %v", err)
	}

	log.Println("Server stopped")
}