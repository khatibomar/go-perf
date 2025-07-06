package monitor

import (
	"context"
	"encoding/json"
	"fmt"
	"html/template"
	"log"
	"net/http"
	"runtime"
	"sync"
	"time"

	"github.com/gorilla/websocket"

	"exercise-03-04/detector"
)

// DashboardConfig holds configuration for the monitoring dashboard
type DashboardConfig struct {
	Port            int           `json:"port"`
	UpdateInterval  time.Duration `json:"update_interval"`
	MaxDataPoints   int           `json:"max_data_points"`
	EnableWebSocket bool          `json:"enable_websocket"`
	StaticDir       string        `json:"static_dir"`
	Verbose         bool          `json:"verbose"`
}

// DashboardData represents real-time dashboard data
type DashboardData struct {
	Timestamp       time.Time                    `json:"timestamp"`
	HeapAnalysis    *detector.HeapAnalysis      `json:"heap_analysis,omitempty"`
	GoroutineAnalysis *detector.GoroutineAnalysis `json:"goroutine_analysis,omitempty"`
	ReferenceAnalysis *detector.ReferenceAnalysis `json:"reference_analysis,omitempty"`
	ResourceAnalysis  *detector.ResourceAnalysis  `json:"resource_analysis,omitempty"`
	SystemMetrics   SystemMetrics                `json:"system_metrics"`
	Alerts          []Alert                      `json:"alerts"`
	Summary         DashboardSummary             `json:"summary"`
}

// SystemMetrics represents system-level metrics
type SystemMetrics struct {
	CPUUsage        float64 `json:"cpu_usage"`
	MemoryUsage     float64 `json:"memory_usage"`
	GoroutineCount  int     `json:"goroutine_count"`
	HeapSize        uint64  `json:"heap_size"`
	GCCount         uint32  `json:"gc_count"`
	GCPauseTime     uint64  `json:"gc_pause_time"`
	OpenFiles       int     `json:"open_files"`
	NetworkConns    int     `json:"network_connections"`
}

// Alert represents a monitoring alert
type Alert struct {
	ID          string    `json:"id"`
	Type        string    `json:"type"`
	Severity    string    `json:"severity"`
	Message     string    `json:"message"`
	Timestamp   time.Time `json:"timestamp"`
	Source      string    `json:"source"`
	Resolved    bool      `json:"resolved"`
	ResolvedAt  *time.Time `json:"resolved_at,omitempty"`
}

// DashboardSummary provides a high-level summary
type DashboardSummary struct {
	TotalLeaks      int     `json:"total_leaks"`
	CriticalLeaks   int     `json:"critical_leaks"`
	ActiveAlerts    int     `json:"active_alerts"`
	MemoryGrowthRate float64 `json:"memory_growth_rate"`
	GoroutineGrowthRate float64 `json:"goroutine_growth_rate"`
	OverallHealth   string  `json:"overall_health"`
	LastUpdate      time.Time `json:"last_update"`
}

// Dashboard provides real-time monitoring capabilities
type Dashboard struct {
	config          *DashboardConfig
	heapAnalyzer    *detector.HeapAnalyzer
	goroutineDetector *detector.GoroutineDetector
	referenceDetector *detector.ReferenceDetector
	resourceDetector  *detector.ResourceDetector
	server          *http.Server
	websocketClients map[*websocket.Conn]bool
	clientsMutex    sync.RWMutex
	dataHistory     []DashboardData
	alerts          []Alert
	mutex           sync.RWMutex
	running         bool
	stopChan        chan struct{}
	upgrader        websocket.Upgrader
}

// NewDashboard creates a new monitoring dashboard
func NewDashboard(config *DashboardConfig) *Dashboard {
	return &Dashboard{
		config:           config,
		websocketClients: make(map[*websocket.Conn]bool),
		dataHistory:      make([]DashboardData, 0),
		alerts:           make([]Alert, 0),
		stopChan:         make(chan struct{}),
		upgrader: websocket.Upgrader{
			CheckOrigin: func(r *http.Request) bool {
				return true // Allow all origins for development
			},
		},
	}
}

// SetDetectors sets the leak detectors
func (d *Dashboard) SetDetectors(
	heapAnalyzer *detector.HeapAnalyzer,
	goroutineDetector *detector.GoroutineDetector,
	referenceDetector *detector.ReferenceDetector,
	resourceDetector *detector.ResourceDetector,
) {
	d.heapAnalyzer = heapAnalyzer
	d.goroutineDetector = goroutineDetector
	d.referenceDetector = referenceDetector
	d.resourceDetector = resourceDetector
}

// Start starts the dashboard server
func (d *Dashboard) Start() error {
	d.mutex.Lock()
	defer d.mutex.Unlock()

	if d.running {
		return fmt.Errorf("dashboard already running")
	}

	d.running = true

	// Setup HTTP routes
	d.setupRoutes()

	// Start the HTTP server
	d.server = &http.Server{
		Addr:    fmt.Sprintf(":%d", d.config.Port),
		Handler: nil, // Use default mux
	}

	// Start data collection
	go d.dataCollectionLoop()

	// Start server in goroutine
	go func() {
		if err := d.server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Printf("Dashboard server error: %v", err)
		}
	}()

	if d.config.Verbose {
		fmt.Printf("Dashboard started on http://localhost:%d\n", d.config.Port)
	}

	return nil
}

// Stop stops the dashboard server
func (d *Dashboard) Stop() error {
	d.mutex.Lock()
	defer d.mutex.Unlock()

	if !d.running {
		return nil
	}

	d.running = false
	close(d.stopChan)

	// Close all websocket connections
	d.clientsMutex.Lock()
	for client := range d.websocketClients {
		client.Close()
	}
	d.websocketClients = make(map[*websocket.Conn]bool)
	d.clientsMutex.Unlock()

	// Shutdown HTTP server
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if d.server != nil {
		return d.server.Shutdown(ctx)
	}

	return nil
}

// setupRoutes sets up HTTP routes
func (d *Dashboard) setupRoutes() {
	http.HandleFunc("/", d.handleIndex)
	http.HandleFunc("/api/data", d.handleAPIData)
	http.HandleFunc("/api/alerts", d.handleAPIAlerts)
	http.HandleFunc("/api/summary", d.handleAPISummary)
	http.HandleFunc("/ws", d.handleWebSocket)
	http.HandleFunc("/health", d.handleHealth)

	// Serve static files if directory is specified
	if d.config.StaticDir != "" {
		http.Handle("/static/", http.StripPrefix("/static/", http.FileServer(http.Dir(d.config.StaticDir))))
	}
}

// handleIndex serves the main dashboard page
func (d *Dashboard) handleIndex(w http.ResponseWriter, r *http.Request) {
	tmpl := `
<!DOCTYPE html>
<html>
<head>
    <title>Memory Leak Detection Dashboard</title>
    <meta charset="utf-8">
    <meta name="viewport" content="width=device-width, initial-scale=1">
    <style>
        body {
            font-family: 'Segoe UI', Tahoma, Geneva, Verdana, sans-serif;
            margin: 0;
            padding: 20px;
            background-color: #f5f5f5;
        }
        .header {
            background: linear-gradient(135deg, #667eea 0%, #764ba2 100%);
            color: white;
            padding: 20px;
            border-radius: 10px;
            margin-bottom: 20px;
            text-align: center;
        }
        .dashboard-grid {
            display: grid;
            grid-template-columns: repeat(auto-fit, minmax(300px, 1fr));
            gap: 20px;
            margin-bottom: 20px;
        }
        .card {
            background: white;
            border-radius: 10px;
            padding: 20px;
            box-shadow: 0 2px 10px rgba(0,0,0,0.1);
            border-left: 4px solid #667eea;
        }
        .metric {
            display: flex;
            justify-content: space-between;
            align-items: center;
            margin: 10px 0;
            padding: 10px;
            background: #f8f9fa;
            border-radius: 5px;
        }
        .metric-value {
            font-weight: bold;
            font-size: 1.2em;
        }
        .alert {
            padding: 10px;
            margin: 5px 0;
            border-radius: 5px;
            border-left: 4px solid;
        }
        .alert-critical { border-left-color: #dc3545; background: #f8d7da; }
        .alert-high { border-left-color: #fd7e14; background: #fff3cd; }
        .alert-medium { border-left-color: #ffc107; background: #fff3cd; }
        .alert-low { border-left-color: #28a745; background: #d4edda; }
        .status-indicator {
            display: inline-block;
            width: 12px;
            height: 12px;
            border-radius: 50%;
            margin-right: 8px;
        }
        .status-healthy { background-color: #28a745; }
        .status-warning { background-color: #ffc107; }
        .status-critical { background-color: #dc3545; }
        .chart-container {
            height: 300px;
            margin: 20px 0;
        }
        .refresh-btn {
            background: #667eea;
            color: white;
            border: none;
            padding: 10px 20px;
            border-radius: 5px;
            cursor: pointer;
            margin: 10px 5px;
        }
        .refresh-btn:hover {
            background: #5a6fd8;
        }
        #connection-status {
            position: fixed;
            top: 10px;
            right: 10px;
            padding: 5px 10px;
            border-radius: 5px;
            color: white;
            font-size: 12px;
        }
        .connected { background: #28a745; }
        .disconnected { background: #dc3545; }
    </style>
    <script src="https://cdn.jsdelivr.net/npm/chart.js"></script>
</head>
<body>
    <div id="connection-status" class="disconnected">Disconnected</div>
    
    <div class="header">
        <h1>Memory Leak Detection Dashboard</h1>
        <p>Real-time monitoring of memory leaks and resource usage</p>
        <button class="refresh-btn" onclick="refreshData()">Refresh Data</button>
        <button class="refresh-btn" onclick="toggleAutoRefresh()">Toggle Auto-refresh</button>
    </div>

    <div class="dashboard-grid">
        <div class="card">
            <h3>System Overview</h3>
            <div id="system-metrics">
                <div class="metric">
                    <span>Overall Health:</span>
                    <span id="overall-health" class="metric-value">Loading...</span>
                </div>
                <div class="metric">
                    <span>Memory Usage:</span>
                    <span id="memory-usage" class="metric-value">-</span>
                </div>
                <div class="metric">
                    <span>Goroutines:</span>
                    <span id="goroutine-count" class="metric-value">-</span>
                </div>
                <div class="metric">
                    <span>Open Files:</span>
                    <span id="open-files" class="metric-value">-</span>
                </div>
            </div>
        </div>

        <div class="card">
            <h3>Leak Summary</h3>
            <div id="leak-summary">
                <div class="metric">
                    <span>Total Leaks:</span>
                    <span id="total-leaks" class="metric-value">-</span>
                </div>
                <div class="metric">
                    <span>Critical Leaks:</span>
                    <span id="critical-leaks" class="metric-value">-</span>
                </div>
                <div class="metric">
                    <span>Active Alerts:</span>
                    <span id="active-alerts" class="metric-value">-</span>
                </div>
                <div class="metric">
                    <span>Memory Growth:</span>
                    <span id="memory-growth" class="metric-value">-</span>
                </div>
            </div>
        </div>

        <div class="card">
            <h3>Recent Alerts</h3>
            <div id="alerts-container">
                <p>No alerts</p>
            </div>
        </div>

        <div class="card">
            <h3>Memory Trend</h3>
            <div class="chart-container">
                <canvas id="memory-chart"></canvas>
            </div>
        </div>
    </div>

    <div class="dashboard-grid">
        <div class="card">
            <h3>Goroutine Trend</h3>
            <div class="chart-container">
                <canvas id="goroutine-chart"></canvas>
            </div>
        </div>

        <div class="card">
            <h3>Resource Usage</h3>
            <div class="chart-container">
                <canvas id="resource-chart"></canvas>
            </div>
        </div>
    </div>

    <script>
        let ws = null;
        let autoRefresh = true;
        let charts = {};
        let dataHistory = [];

        // Initialize charts
        function initCharts() {
            const chartConfig = {
                type: 'line',
                options: {
                    responsive: true,
                    maintainAspectRatio: false,
                    scales: {
                        x: {
                            type: 'time',
                            time: {
                                unit: 'minute'
                            }
                        },
                        y: {
                            beginAtZero: true
                        }
                    },
                    plugins: {
                        legend: {
                            display: true
                        }
                    }
                }
            };

            charts.memory = new Chart(document.getElementById('memory-chart'), {
                ...chartConfig,
                data: {
                    datasets: [{
                        label: 'Heap Size (MB)',
                        data: [],
                        borderColor: '#667eea',
                        backgroundColor: 'rgba(102, 126, 234, 0.1)',
                        tension: 0.4
                    }]
                }
            });

            charts.goroutine = new Chart(document.getElementById('goroutine-chart'), {
                ...chartConfig,
                data: {
                    datasets: [{
                        label: 'Goroutine Count',
                        data: [],
                        borderColor: '#fd7e14',
                        backgroundColor: 'rgba(253, 126, 20, 0.1)',
                        tension: 0.4
                    }]
                }
            });

            charts.resource = new Chart(document.getElementById('resource-chart'), {
                ...chartConfig,
                data: {
                    datasets: [
                        {
                            label: 'Open Files',
                            data: [],
                            borderColor: '#28a745',
                            backgroundColor: 'rgba(40, 167, 69, 0.1)',
                            tension: 0.4
                        },
                        {
                            label: 'Network Connections',
                            data: [],
                            borderColor: '#dc3545',
                            backgroundColor: 'rgba(220, 53, 69, 0.1)',
                            tension: 0.4
                        }
                    ]
                }
            });
        }

        // Connect to WebSocket
        function connectWebSocket() {
            const protocol = window.location.protocol === 'https:' ? 'wss:' : 'ws:';
            const wsUrl = protocol + '//' + window.location.host + '/ws';
            
            ws = new WebSocket(wsUrl);
            
            ws.onopen = function() {
                document.getElementById('connection-status').textContent = 'Connected';
                document.getElementById('connection-status').className = 'connected';
            };
            
            ws.onmessage = function(event) {
                const data = JSON.parse(event.data);
                updateDashboard(data);
            };
            
            ws.onclose = function() {
                document.getElementById('connection-status').textContent = 'Disconnected';
                document.getElementById('connection-status').className = 'disconnected';
                // Reconnect after 5 seconds
                setTimeout(connectWebSocket, 5000);
            };
            
            ws.onerror = function(error) {
                console.error('WebSocket error:', error);
            };
        }

        // Update dashboard with new data
        function updateDashboard(data) {
            // Update system metrics
            document.getElementById('overall-health').textContent = data.summary.overall_health;
            document.getElementById('memory-usage').textContent = 
                (data.system_metrics.heap_size / 1024 / 1024).toFixed(2) + ' MB';
            document.getElementById('goroutine-count').textContent = data.system_metrics.goroutine_count;
            document.getElementById('open-files').textContent = data.system_metrics.open_files;

            // Update leak summary
            document.getElementById('total-leaks').textContent = data.summary.total_leaks;
            document.getElementById('critical-leaks').textContent = data.summary.critical_leaks;
            document.getElementById('active-alerts').textContent = data.summary.active_alerts;
            document.getElementById('memory-growth').textContent = 
                data.summary.memory_growth_rate.toFixed(2) + ' MB/min';

            // Update health indicator
            const healthElement = document.getElementById('overall-health');
            healthElement.className = 'metric-value';
            const indicator = document.createElement('span');
            indicator.className = 'status-indicator ';
            
            switch(data.summary.overall_health) {
                case 'healthy':
                    indicator.className += 'status-healthy';
                    break;
                case 'warning':
                    indicator.className += 'status-warning';
                    break;
                case 'critical':
                    indicator.className += 'status-critical';
                    break;
            }
            
            healthElement.innerHTML = indicator.outerHTML + data.summary.overall_health;

            // Update alerts
            updateAlerts(data.alerts);

            // Update charts
            updateCharts(data);

            // Store data history
            dataHistory.push(data);
            if (dataHistory.length > 100) {
                dataHistory.shift();
            }
        }

        // Update alerts display
        function updateAlerts(alerts) {
            const container = document.getElementById('alerts-container');
            
            if (!alerts || alerts.length === 0) {
                container.innerHTML = '<p>No alerts</p>';
                return;
            }

            const recentAlerts = alerts.slice(-5); // Show last 5 alerts
            container.innerHTML = recentAlerts.map(alert => 
                '<div class="alert alert-' + alert.severity + '">' +
                    '<strong>' + alert.type + '</strong>: ' + alert.message +
                    '<br><small>' + new Date(alert.timestamp).toLocaleString() + '</small>' +
                '</div>'
            ).join('');
        }

        // Update charts with new data
        function updateCharts(data) {
            const timestamp = new Date(data.timestamp);
            
            // Update memory chart
            charts.memory.data.datasets[0].data.push({
                x: timestamp,
                y: data.system_metrics.heap_size / 1024 / 1024
            });
            
            // Update goroutine chart
            charts.goroutine.data.datasets[0].data.push({
                x: timestamp,
                y: data.system_metrics.goroutine_count
            });
            
            // Update resource chart
            charts.resource.data.datasets[0].data.push({
                x: timestamp,
                y: data.system_metrics.open_files
            });
            charts.resource.data.datasets[1].data.push({
                x: timestamp,
                y: data.system_metrics.network_connections
            });
            
            // Limit data points
            Object.values(charts).forEach(chart => {
                chart.data.datasets.forEach(dataset => {
                    if (dataset.data.length > 50) {
                        dataset.data.shift();
                    }
                });
                chart.update('none');
            });
        }

        // Refresh data manually
        function refreshData() {
            fetch('/api/data')
                .then(response => response.json())
                .then(data => updateDashboard(data))
                .catch(error => console.error('Error fetching data:', error));
        }

        // Toggle auto-refresh
        function toggleAutoRefresh() {
            autoRefresh = !autoRefresh;
            if (autoRefresh) {
                connectWebSocket();
            } else if (ws) {
                ws.close();
            }
        }

        // Initialize dashboard
        document.addEventListener('DOMContentLoaded', function() {
            initCharts();
            connectWebSocket();
            
            // Initial data load
            refreshData();
            
            // Fallback polling if WebSocket fails
            setInterval(function() {
                if (!ws || ws.readyState !== WebSocket.OPEN) {
                    refreshData();
                }
            }, 5000);
        });
    </script>
</body>
</html>
`

	t, err := template.New("dashboard").Parse(tmpl)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "text/html")
	t.Execute(w, nil)
}

// handleAPIData serves current dashboard data as JSON
func (d *Dashboard) handleAPIData(w http.ResponseWriter, r *http.Request) {
	d.mutex.RLock()
	data := d.getCurrentData()
	d.mutex.RUnlock()

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(data)
}

// handleAPIAlerts serves current alerts as JSON
func (d *Dashboard) handleAPIAlerts(w http.ResponseWriter, r *http.Request) {
	d.mutex.RLock()
	alerts := make([]Alert, len(d.alerts))
	copy(alerts, d.alerts)
	d.mutex.RUnlock()

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(alerts)
}

// handleAPISummary serves dashboard summary as JSON
func (d *Dashboard) handleAPISummary(w http.ResponseWriter, r *http.Request) {
	d.mutex.RLock()
	data := d.getCurrentData()
	d.mutex.RUnlock()

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(data.Summary)
}

// handleWebSocket handles WebSocket connections
func (d *Dashboard) handleWebSocket(w http.ResponseWriter, r *http.Request) {
	conn, err := d.upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Printf("WebSocket upgrade error: %v", err)
		return
	}

	d.clientsMutex.Lock()
	d.websocketClients[conn] = true
	d.clientsMutex.Unlock()

	defer func() {
		d.clientsMutex.Lock()
		delete(d.websocketClients, conn)
		d.clientsMutex.Unlock()
		conn.Close()
	}()

	// Send initial data
	d.mutex.RLock()
	initialData := d.getCurrentData()
	d.mutex.RUnlock()

	if err := conn.WriteJSON(initialData); err != nil {
		log.Printf("WebSocket write error: %v", err)
		return
	}

	// Keep connection alive
	for {
		select {
		case <-d.stopChan:
			return
		default:
			// Read messages to detect disconnection
			if _, _, err := conn.ReadMessage(); err != nil {
				return
			}
		}
	}
}

// handleHealth serves health check endpoint
func (d *Dashboard) handleHealth(w http.ResponseWriter, r *http.Request) {
	health := map[string]interface{}{
		"status":    "healthy",
		"timestamp": time.Now(),
		"uptime":    time.Since(time.Now()), // This would be calculated properly
		"version":   "1.0.0",
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(health)
}

// dataCollectionLoop collects data from detectors periodically
func (d *Dashboard) dataCollectionLoop() {
	ticker := time.NewTicker(d.config.UpdateInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			d.collectAndBroadcastData()
		case <-d.stopChan:
			return
		}
	}
}

// collectAndBroadcastData collects current data and broadcasts to clients
func (d *Dashboard) collectAndBroadcastData() {
	d.mutex.Lock()
	data := d.getCurrentData()

	// Add to history
	d.dataHistory = append(d.dataHistory, data)
	if len(d.dataHistory) > d.config.MaxDataPoints {
		d.dataHistory = d.dataHistory[1:]
	}

	// Check for new alerts
	d.checkForAlerts(data)
	d.mutex.Unlock()

	// Broadcast to WebSocket clients
	d.broadcastData(data)
}

// getCurrentData collects current data from all detectors
func (d *Dashboard) getCurrentData() DashboardData {
	data := DashboardData{
		Timestamp:     time.Now(),
		SystemMetrics: d.getSystemMetrics(),
		Alerts:        d.alerts,
	}

	// Collect from heap analyzer
	if d.heapAnalyzer != nil {
		if analysis, err := d.heapAnalyzer.AnalyzeHeap(); err == nil {
			data.HeapAnalysis = analysis
		}
	}

	// Collect from goroutine detector
	if d.goroutineDetector != nil {
		if analysis, err := d.goroutineDetector.AnalyzeGoroutines(); err == nil {
			data.GoroutineAnalysis = analysis
		}
	}

	// Collect from reference detector
	if d.referenceDetector != nil {
		if analysis, err := d.referenceDetector.AnalyzeReferences(); err == nil {
			data.ReferenceAnalysis = analysis
		}
	}

	// Collect from resource detector
	if d.resourceDetector != nil {
		if analysis, err := d.resourceDetector.AnalyzeResources(); err == nil {
			data.ResourceAnalysis = analysis
		}
	}

	// Generate summary
	data.Summary = d.generateSummary(data)

	return data
}

// getSystemMetrics collects system-level metrics
func (d *Dashboard) getSystemMetrics() SystemMetrics {
	var m runtime.MemStats
	runtime.ReadMemStats(&m)

	return SystemMetrics{
		CPUUsage:       0.0, // Would need additional library for CPU usage
		MemoryUsage:    float64(m.Alloc) / float64(m.Sys) * 100,
		GoroutineCount: runtime.NumGoroutine(),
		HeapSize:       m.HeapAlloc,
		GCCount:        m.NumGC,
		GCPauseTime:    m.PauseTotalNs,
		OpenFiles:      d.getOpenFileCount(),
		NetworkConns:   d.getNetworkConnectionCount(),
	}
}

// getOpenFileCount gets the number of open file descriptors
func (d *Dashboard) getOpenFileCount() int {
	if d.resourceDetector != nil {
		metrics := d.resourceDetector.GetMetrics()
		if len(metrics) > 0 {
			return metrics[len(metrics)-1].FileHandles
		}
	}
	return 0
}

// getNetworkConnectionCount gets the number of network connections
func (d *Dashboard) getNetworkConnectionCount() int {
	if d.resourceDetector != nil {
		metrics := d.resourceDetector.GetMetrics()
		if len(metrics) > 0 {
			return metrics[len(metrics)-1].NetworkConnections
		}
	}
	return 0
}

// generateSummary generates a dashboard summary
func (d *Dashboard) generateSummary(data DashboardData) DashboardSummary {
	summary := DashboardSummary{
		LastUpdate: data.Timestamp,
	}

	// Count leaks from all sources
	if data.HeapAnalysis != nil {
		summary.TotalLeaks += len(data.HeapAnalysis.Anomalies)
		for _, anomaly := range data.HeapAnalysis.Anomalies {
			if anomaly.Severity == "critical" {
				summary.CriticalLeaks++
			}
		}
		summary.MemoryGrowthRate = data.HeapAnalysis.Statistics.GrowthVariance
	}

	if data.GoroutineAnalysis != nil {
		summary.TotalLeaks += len(data.GoroutineAnalysis.LeakCandidates)
		for _, candidate := range data.GoroutineAnalysis.LeakCandidates {
			// Derive severity from confidence
			severity := "medium"
			if candidate.Confidence > 0.8 {
				severity = "critical"
			} else if candidate.Confidence < 0.4 {
				severity = "low"
			}
			if severity == "critical" {
				summary.CriticalLeaks++
			}
		}
		summary.GoroutineGrowthRate = data.GoroutineAnalysis.Statistics.GrowthVariance
	}

	if data.ResourceAnalysis != nil {
		summary.TotalLeaks += len(data.ResourceAnalysis.DetectedLeaks)
		for _, leak := range data.ResourceAnalysis.DetectedLeaks {
			if leak.Severity == "critical" {
				summary.CriticalLeaks++
			}
		}
	}

	// Count active alerts
	for _, alert := range data.Alerts {
		if !alert.Resolved {
			summary.ActiveAlerts++
		}
	}

	// Determine overall health
	if summary.CriticalLeaks > 0 || summary.ActiveAlerts > 5 {
		summary.OverallHealth = "critical"
	} else if summary.TotalLeaks > 10 || summary.ActiveAlerts > 0 {
		summary.OverallHealth = "warning"
	} else {
		summary.OverallHealth = "healthy"
	}

	return summary
}

// checkForAlerts checks for new alerts based on current data
func (d *Dashboard) checkForAlerts(data DashboardData) {
	// Check memory growth rate
	if data.Summary.MemoryGrowthRate > 10.0 { // 10 MB/min
		d.addAlert(Alert{
			ID:        fmt.Sprintf("memory_growth_%d", time.Now().Unix()),
			Type:      "memory_growth",
			Severity:  "high",
			Message:   fmt.Sprintf("High memory growth rate: %.2f MB/min", data.Summary.MemoryGrowthRate),
			Timestamp: time.Now(),
			Source:    "heap_analyzer",
		})
	}

	// Check goroutine growth rate
	if data.Summary.GoroutineGrowthRate > 5.0 { // 5 goroutines/min
		d.addAlert(Alert{
			ID:        fmt.Sprintf("goroutine_growth_%d", time.Now().Unix()),
			Type:      "goroutine_growth",
			Severity:  "medium",
			Message:   fmt.Sprintf("High goroutine growth rate: %.2f/min", data.Summary.GoroutineGrowthRate),
			Timestamp: time.Now(),
			Source:    "goroutine_detector",
		})
	}

	// Check critical leaks
	if data.Summary.CriticalLeaks > 0 {
		d.addAlert(Alert{
			ID:        fmt.Sprintf("critical_leaks_%d", time.Now().Unix()),
			Type:      "critical_leaks",
			Severity:  "critical",
			Message:   fmt.Sprintf("%d critical leaks detected", data.Summary.CriticalLeaks),
			Timestamp: time.Now(),
			Source:    "dashboard",
		})
	}

	// Check system resources
	if data.SystemMetrics.OpenFiles > 1000 {
		d.addAlert(Alert{
			ID:        fmt.Sprintf("high_file_usage_%d", time.Now().Unix()),
			Type:      "resource_usage",
			Severity:  "medium",
			Message:   fmt.Sprintf("High file descriptor usage: %d", data.SystemMetrics.OpenFiles),
			Timestamp: time.Now(),
			Source:    "resource_detector",
		})
	}
}

// addAlert adds a new alert if it doesn't already exist
func (d *Dashboard) addAlert(alert Alert) {
	// Check if similar alert already exists
	for _, existing := range d.alerts {
		if existing.Type == alert.Type && !existing.Resolved {
			return // Don't add duplicate alerts
		}
	}

	d.alerts = append(d.alerts, alert)

	// Keep only recent alerts
	if len(d.alerts) > 100 {
		d.alerts = d.alerts[len(d.alerts)-100:]
	}

	if d.config.Verbose {
		fmt.Printf("New alert: %s - %s\n", alert.Type, alert.Message)
	}
}

// broadcastData broadcasts data to all connected WebSocket clients
func (d *Dashboard) broadcastData(data DashboardData) {
	d.clientsMutex.RLock()
	clients := make([]*websocket.Conn, 0, len(d.websocketClients))
	for client := range d.websocketClients {
		clients = append(clients, client)
	}
	d.clientsMutex.RUnlock()

	for _, client := range clients {
		if err := client.WriteJSON(data); err != nil {
			// Remove disconnected client
			d.clientsMutex.Lock()
			delete(d.websocketClients, client)
			d.clientsMutex.Unlock()
			client.Close()
		}
	}
}

// GetURL returns the dashboard URL
func (d *Dashboard) GetURL() string {
	return fmt.Sprintf("http://localhost:%d", d.config.Port)
}

// GetDataHistory returns the data history
func (d *Dashboard) GetDataHistory() []DashboardData {
	d.mutex.RLock()
	defer d.mutex.RUnlock()

	history := make([]DashboardData, len(d.dataHistory))
	copy(history, d.dataHistory)
	return history
}

// GetAlerts returns current alerts
func (d *Dashboard) GetAlerts() []Alert {
	d.mutex.RLock()
	defer d.mutex.RUnlock()

	alerts := make([]Alert, len(d.alerts))
	copy(alerts, d.alerts)
	return alerts
}

// ResolveAlert resolves an alert by ID
func (d *Dashboard) ResolveAlert(alertID string) {
	d.mutex.Lock()
	defer d.mutex.Unlock()

	for i, alert := range d.alerts {
		if alert.ID == alertID {
			now := time.Now()
			d.alerts[i].Resolved = true
			d.alerts[i].ResolvedAt = &now
			break
		}
	}
}

// ClearResolvedAlerts removes resolved alerts older than the specified duration
func (d *Dashboard) ClearResolvedAlerts(maxAge time.Duration) {
	d.mutex.Lock()
	defer d.mutex.Unlock()

	now := time.Now()
	var activeAlerts []Alert

	for _, alert := range d.alerts {
		if !alert.Resolved {
			activeAlerts = append(activeAlerts, alert)
		} else if alert.ResolvedAt != nil && now.Sub(*alert.ResolvedAt) < maxAge {
			activeAlerts = append(activeAlerts, alert)
		}
	}

	d.alerts = activeAlerts
}