package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strconv"
	"time"
)

// Dashboard provides a web interface for viewing profiling data
type Dashboard struct {
	profiler *ContinuousProfiler
	storage  *ProfileStorage
	server   *http.Server
}

// NewDashboard creates a new dashboard instance
func NewDashboard(profiler *ContinuousProfiler, storage *ProfileStorage) *Dashboard {
	return &Dashboard{
		profiler: profiler,
		storage:  storage,
	}
}

// Start starts the dashboard web server
func (d *Dashboard) Start(port int) error {
	mux := http.NewServeMux()
	
	// Register routes
	mux.HandleFunc("/", d.handleHome)
	mux.HandleFunc("/api/stats", d.handleStats)
	mux.HandleFunc("/api/profiles", d.handleProfiles)
	mux.HandleFunc("/api/profile/", d.handleProfile)
	mux.HandleFunc("/api/metrics", d.handleMetrics)
	
	d.server = &http.Server{
		Addr:    fmt.Sprintf(":%d", port),
		Handler: mux,
	}
	
	fmt.Printf("Dashboard starting on http://localhost:%d\n", port)
	return d.server.ListenAndServe()
}

// Stop stops the dashboard web server
func (d *Dashboard) Stop() error {
	if d.server != nil {
		return d.server.Close()
	}
	return nil
}

// handleHome serves the main dashboard page
func (d *Dashboard) handleHome(w http.ResponseWriter, r *http.Request) {
	html := `
<!DOCTYPE html>
<html>
<head>
    <title>Memory Profiling Dashboard</title>
    <style>
        body { font-family: Arial, sans-serif; margin: 20px; }
        .container { max-width: 1200px; margin: 0 auto; }
        .card { border: 1px solid #ddd; border-radius: 8px; padding: 20px; margin: 20px 0; }
        .stats { display: grid; grid-template-columns: repeat(auto-fit, minmax(200px, 1fr)); gap: 20px; }
        .stat { background: #f5f5f5; padding: 15px; border-radius: 5px; text-align: center; }
        .stat-value { font-size: 24px; font-weight: bold; color: #333; }
        .stat-label { color: #666; margin-top: 5px; }
        .profiles-list { max-height: 400px; overflow-y: auto; }
        .profile-item { padding: 10px; border-bottom: 1px solid #eee; }
        .profile-item:hover { background: #f9f9f9; }
        .btn { background: #007bff; color: white; padding: 8px 16px; border: none; border-radius: 4px; cursor: pointer; }
        .btn:hover { background: #0056b3; }
        .status { padding: 4px 8px; border-radius: 4px; font-size: 12px; }
        .status.running { background: #d4edda; color: #155724; }
        .status.stopped { background: #f8d7da; color: #721c24; }
    </style>
</head>
<body>
    <div class="container">
        <h1>Memory Profiling Dashboard</h1>
        
        <div class="card">
            <h2>Profiler Status</h2>
            <div id="profiler-status">Loading...</div>
            <button class="btn" onclick="toggleProfiler()">Toggle Profiler</button>
        </div>
        
        <div class="card">
            <h2>Statistics</h2>
            <div class="stats" id="stats">Loading...</div>
        </div>
        
        <div class="card">
            <h2>Recent Profiles</h2>
            <div class="profiles-list" id="profiles">Loading...</div>
        </div>
        
        <div class="card">
            <h2>Current Memory Metrics</h2>
            <div id="metrics">Loading...</div>
        </div>
    </div>
    
    <script>
        function loadData() {
            // Load stats
            fetch('/api/stats')
                .then(response => response.json())
                .then(data => {
                    document.getElementById('stats').innerHTML = 
                        '<div class="stat">' +
                            '<div class="stat-value">' + data.total_profiles + '</div>' +
                            '<div class="stat-label">Total Profiles</div>' +
                        '</div>' +
                        '<div class="stat">' +
                            '<div class="stat-value">' + data.formatted_size + '</div>' +
                            '<div class="stat-label">Storage Used</div>' +
                        '</div>' +
                        '<div class="stat">' +
                            '<div class="stat-value">' + data.retention_days + '</div>' +
                            '<div class="stat-label">Retention Days</div>' +
                        '</div>' +
                        '<div class="stat">' +
                            '<div class="stat-value">' + data.collection_count + '</div>' +
                            '<div class="stat-label">Collections</div>' +
                        '</div>';
                });
            
            // Load profiles
            fetch('/api/profiles')
                .then(response => response.json())
                .then(data => {
                    const profilesHtml = data.map(profile => 
                        '<div class="profile-item">' +
                            '<strong>' + profile.filename + '</strong><br>' +
                            '<small>Type: ' + profile.type + ' | Size: ' + formatBytes(profile.size) + ' | ' + new Date(profile.timestamp).toLocaleString() + '</small>' +
                        '</div>'
                    ).join('');
                    document.getElementById('profiles').innerHTML = profilesHtml || '<p>No profiles found</p>';
                });
            
            // Load metrics
            fetch('/api/metrics')
                .then(response => response.json())
                .then(data => {
                    document.getElementById('metrics').innerHTML = 
                        '<div class="stats">' +
                            '<div class="stat">' +
                                '<div class="stat-value">' + formatBytes(data.alloc) + '</div>' +
                                '<div class="stat-label">Current Alloc</div>' +
                            '</div>' +
                            '<div class="stat">' +
                                '<div class="stat-value">' + formatBytes(data.total_alloc) + '</div>' +
                                '<div class="stat-label">Total Alloc</div>' +
                            '</div>' +
                            '<div class="stat">' +
                                '<div class="stat-value">' + formatBytes(data.sys) + '</div>' +
                                '<div class="stat-label">System Memory</div>' +
                            '</div>' +
                            '<div class="stat">' +
                                '<div class="stat-value">' + data.num_gc + '</div>' +
                                '<div class="stat-label">GC Cycles</div>' +
                            '</div>' +
                        '</div>';
                });
        }
        
        function formatBytes(bytes) {
            if (bytes === 0) return '0 B';
            const k = 1024;
            const sizes = ['B', 'KB', 'MB', 'GB'];
            const i = Math.floor(Math.log(bytes) / Math.log(k));
            return parseFloat((bytes / Math.pow(k, i)).toFixed(2)) + ' ' + sizes[i];
        }
        
        function toggleProfiler() {
            // This would need to be implemented with actual API endpoints
            alert('Profiler toggle functionality would be implemented here');
        }
        
        // Load data initially and refresh every 5 seconds
        loadData();
        setInterval(loadData, 5000);
    </script>
</body>
</html>
`
	w.Header().Set("Content-Type", "text/html")
	w.Write([]byte(html))
}

// handleStats returns profiler and storage statistics
func (d *Dashboard) handleStats(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	
	storageStats := d.storage.GetStorageStats()
	profilerStats := d.profiler.GetStats()
	
	stats := map[string]interface{}{
		"total_profiles":   storageStats.TotalProfiles,
		"total_size":      storageStats.TotalSize,
		"formatted_size":  storageStats.FormatSize(),
		"retention_days":  storageStats.RetentionDays,
		"collection_count": profilerStats.ProfilesCount,
		"last_collection": profilerStats.LastCollection,
		"is_running":      profilerStats.Running,
	}
	
	json.NewEncoder(w).Encode(stats)
}

// handleProfiles returns list of stored profiles
func (d *Dashboard) handleProfiles(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	
	// Get query parameters for filtering
	limitStr := r.URL.Query().Get("limit")
	limit := 20 // Default limit
	if limitStr != "" {
		if l, err := strconv.Atoi(limitStr); err == nil && l > 0 {
			limit = l
		}
	}
	
	profiles := d.storage.GetProfiles()
	if len(profiles) > limit {
		profiles = profiles[:limit]
	}
	
	json.NewEncoder(w).Encode(profiles)
}

// handleProfile returns details of a specific profile
func (d *Dashboard) handleProfile(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	
	// Extract filename from URL path
	filename := r.URL.Path[len("/api/profile/"):]
	if filename == "" {
		http.Error(w, "Profile filename required", http.StatusBadRequest)
		return
	}
	
	profile, exists := d.storage.GetProfile(filename)
	if !exists {
		http.Error(w, "Profile not found", http.StatusNotFound)
		return
	}
	
	json.NewEncoder(w).Encode(profile)
}

// handleMetrics returns current memory metrics
func (d *Dashboard) handleMetrics(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	
	metrics := d.profiler.GetCurrentMetrics()
	json.NewEncoder(w).Encode(metrics)
}

// DashboardConfig contains dashboard configuration
type DashboardConfig struct {
	Port           int    `json:"port"`
	RefreshInterval int    `json:"refresh_interval"`
	Title          string `json:"title"`
}

// DefaultDashboardConfig returns default dashboard configuration
func DefaultDashboardConfig() *DashboardConfig {
	return &DashboardConfig{
		Port:           8080,
		RefreshInterval: 5,
		Title:          "Memory Profiling Dashboard",
	}
}

// StartDashboardExample demonstrates how to start the dashboard
func StartDashboardExample() {
	// This is an example function, not meant to be called from main
	profiler := NewContinuousProfiler(30*time.Second, "./profiles")
	storage := NewProfileStorage("./profiles")
	dashboard := NewDashboard(profiler, storage)
	
	// Start profiler
	if err := profiler.Start(); err != nil {
		log.Printf("Failed to start profiler: %v", err)
		return
	}
	
	// Start dashboard
	if err := dashboard.Start(8080); err != nil {
		log.Printf("Dashboard server error: %v", err)
	}
}