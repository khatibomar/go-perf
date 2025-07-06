package main

import (
	"bytes"
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"math"
	"net/http"
	_ "net/http/pprof"
	"os"
	"os/signal"
	"runtime/pprof"
	"sync"
	"syscall"
	"time"

	"github.com/gorilla/mux"
	"gopkg.in/yaml.v3"
)

// Configuration structure
type Config struct {
	Profiling struct {
		CPU struct {
			Enabled    bool          `yaml:"enabled"`
			Duration   time.Duration `yaml:"duration"`
			OutputDir  string        `yaml:"output_dir"`
			SampleRate int           `yaml:"sample_rate"`
		} `yaml:"cpu"`
		Memory struct {
			Enabled      bool   `yaml:"enabled"`
			OutputDir    string `yaml:"output_dir"`
			GCBeforeHeap bool   `yaml:"gc_before_heap"`
		} `yaml:"memory"`
	} `yaml:"profiling"`
	Server struct {
		Port         int           `yaml:"port"`
		ReadTimeout  time.Duration `yaml:"read_timeout"`
		WriteTimeout time.Duration `yaml:"write_timeout"`
	} `yaml:"server"`
}

// ProfileStorage interface for storing profiles
type ProfileStorage interface {
	Store(profileType string, data []byte) error
	List() ([]string, error)
	Get(id string) ([]byte, error)
}

// FileStorage implements ProfileStorage
type FileStorage struct {
	baseDir string
	mu      sync.RWMutex
}

func NewFileStorage(baseDir string) *FileStorage {
	return &FileStorage{baseDir: baseDir}
}

func (fs *FileStorage) Store(profileType string, data []byte) error {
	fs.mu.Lock()
	defer fs.mu.Unlock()
	
	timestamp := time.Now().Format("2006-01-02_15-04-05")
	filename := fmt.Sprintf("%s_%s.prof", profileType, timestamp)
	filepath := fmt.Sprintf("%s/%s", fs.baseDir, filename)
	
	return os.WriteFile(filepath, data, 0644)
}

func (fs *FileStorage) List() ([]string, error) {
	fs.mu.RLock()
	defer fs.mu.RUnlock()
	
	entries, err := os.ReadDir(fs.baseDir)
	if err != nil {
		return nil, err
	}
	
	var files []string
	for _, entry := range entries {
		if !entry.IsDir() {
			files = append(files, entry.Name())
		}
	}
	return files, nil
}

func (fs *FileStorage) Get(id string) ([]byte, error) {
	fs.mu.RLock()
	defer fs.mu.RUnlock()
	
	filepath := fmt.Sprintf("%s/%s", fs.baseDir, id)
	return os.ReadFile(filepath)
}

// ContinuousProfiler handles continuous profiling
type ContinuousProfiler struct {
	interval time.Duration
	duration time.Duration
	storage  ProfileStorage
	running  bool
	mu       sync.RWMutex
}

func NewContinuousProfiler(interval, duration time.Duration, storage ProfileStorage) *ContinuousProfiler {
	return &ContinuousProfiler{
		interval: interval,
		duration: duration,
		storage:  storage,
	}
}

func (cp *ContinuousProfiler) Start(ctx context.Context) {
	cp.mu.Lock()
	cp.running = true
	cp.mu.Unlock()
	
	ticker := time.NewTicker(cp.interval)
	defer ticker.Stop()
	
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			cp.collectProfile()
		}
	}
}

func (cp *ContinuousProfiler) collectProfile() {
	var buf bytes.Buffer
	if err := pprof.StartCPUProfile(&buf); err != nil {
		log.Printf("Failed to start CPU profile: %v", err)
		return
	}
	
	time.Sleep(cp.duration)
	pprof.StopCPUProfile()
	
	if err := cp.storage.Store("cpu", buf.Bytes()); err != nil {
		log.Printf("Failed to store profile: %v", err)
	}
}

func (cp *ContinuousProfiler) Stop() {
	cp.mu.Lock()
	cp.running = false
	cp.mu.Unlock()
}

func (cp *ContinuousProfiler) IsRunning() bool {
	cp.mu.RLock()
	defer cp.mu.RUnlock()
	return cp.running
}

// Dashboard represents the monitoring dashboard
type Dashboard struct {
	storage ProfileStorage
}

func NewDashboard(storage ProfileStorage) *Dashboard {
	return &Dashboard{storage: storage}
}

func (d *Dashboard) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	switch r.URL.Path {
	case "/":
		d.handleIndex(w, r)
	case "/api/profiles":
		d.handleProfiles(w, r)
	case "/api/profile":
		d.handleProfile(w, r)
	default:
		http.NotFound(w, r)
	}
}

func (d *Dashboard) handleIndex(w http.ResponseWriter, r *http.Request) {
	html := `<!DOCTYPE html>
<html>
<head>
    <title>Production Profiling Dashboard</title>
    <style>
        body { font-family: Arial, sans-serif; margin: 20px; }
        .profile-list { margin: 20px 0; }
        .profile-item { padding: 10px; border: 1px solid #ddd; margin: 5px 0; }
        .btn { padding: 8px 16px; background: #007cba; color: white; text-decoration: none; border-radius: 4px; }
    </style>
</head>
<body>
    <h1>Production Profiling Dashboard</h1>
    <div id="profiles"></div>
    <script>
        fetch('/api/profiles')
            .then(response => response.json())
            .then(data => {
                const container = document.getElementById('profiles');
                data.forEach(profile => {
                    const div = document.createElement('div');
                    div.className = 'profile-item';
                    div.innerHTML = '<strong>' + profile + '</strong> <a href="/api/profile?id=' + profile + '" class="btn">Download</a>';
                    container.appendChild(div);
                });
            });
    </script>
</body>
</html>`
	w.Header().Set("Content-Type", "text/html")
	w.Write([]byte(html))
}

func (d *Dashboard) handleProfiles(w http.ResponseWriter, r *http.Request) {
	profiles, err := d.storage.List()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(profiles)
}

func (d *Dashboard) handleProfile(w http.ResponseWriter, r *http.Request) {
	id := r.URL.Query().Get("id")
	if id == "" {
		http.Error(w, "Missing profile id", http.StatusBadRequest)
		return
	}
	
	data, err := d.storage.Get(id)
	if err != nil {
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}
	
	w.Header().Set("Content-Type", "application/octet-stream")
	w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=%s", id))
	w.Write(data)
}

func main() {
	var (
		configFile = flag.String("config", "config.yaml", "Configuration file path")
		profileDir = flag.String("profile-dir", "profiles", "Directory to store profiles")
		port       = flag.Int("port", 8080, "Server port")
	)
	flag.Parse()

	// Create profile directory if it doesn't exist
	if err := os.MkdirAll(*profileDir, 0755); err != nil {
		log.Fatalf("Failed to create profile directory: %v", err)
	}

	// Load configuration (with defaults if file doesn't exist)
	config, err := loadConfig(*configFile)
	if err != nil {
		log.Printf("Failed to load config, using defaults: %v", err)
		config = getDefaultConfig()
	}

	// Create profile storage
	storage := NewFileStorage(*profileDir)

	// Create continuous profiler
	profiler := NewContinuousProfiler(
		5*time.Minute,  // interval
		30*time.Second, // duration
		storage,
	)

	// Create dashboard
	dash := NewDashboard(storage)

	// Setup HTTP server
	router := mux.NewRouter()
	router.PathPrefix("/debug/pprof/").Handler(http.DefaultServeMux)
	router.PathPrefix("/dashboard/").Handler(http.StripPrefix("/dashboard", dash))
	router.HandleFunc("/workload", cpuIntensiveHandler)
	router.HandleFunc("/health", healthHandler)

	server := &http.Server{
		Addr:         fmt.Sprintf(":%d", *port),
		Handler:      router,
		ReadTimeout:  config.Server.ReadTimeout,
		WriteTimeout: config.Server.WriteTimeout,
	}

	// Start continuous profiling
	ctx, cancel := context.WithCancel(context.Background())
	go profiler.Start(ctx)

	// Start server
	go func() {
		log.Printf("Server starting on port %d", *port)
		log.Printf("pprof available at http://localhost:%d/debug/pprof/", *port)
		log.Printf("Dashboard available at http://localhost:%d/dashboard/", *port)
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("Server failed: %v", err)
		}
	}()

	// Wait for interrupt signal
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
	<-sigChan

	log.Println("Shutting down server...")

	// Stop profiler
	cancel()

	// Shutdown server
	ctx, cancel = context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	if err := server.Shutdown(ctx); err != nil {
		log.Printf("Server shutdown error: %v", err)
	}

	log.Println("Server stopped")
}

// loadConfig loads configuration from file
func loadConfig(filename string) (*Config, error) {
	data, err := os.ReadFile(filename)
	if err != nil {
		return nil, err
	}
	
	var config Config
	err = yaml.Unmarshal(data, &config)
	return &config, err
}

// getDefaultConfig returns default configuration
func getDefaultConfig() *Config {
	config := &Config{}
	config.Profiling.CPU.Enabled = true
	config.Profiling.CPU.Duration = 30 * time.Second
	config.Profiling.CPU.OutputDir = "profiles"
	config.Profiling.CPU.SampleRate = 100
	config.Profiling.Memory.Enabled = true
	config.Profiling.Memory.OutputDir = "profiles"
	config.Profiling.Memory.GCBeforeHeap = true
	config.Server.Port = 8080
	config.Server.ReadTimeout = 15 * time.Second
	config.Server.WriteTimeout = 15 * time.Second
	return config
}

// cpuIntensiveHandler simulates CPU-intensive work
func cpuIntensiveHandler(w http.ResponseWriter, r *http.Request) {
	start := time.Now()
	
	// Simulate CPU-intensive work
	result := 0.0
	for i := 0; i < 1000000; i++ {
		result += math.Sin(float64(i)) * math.Cos(float64(i))
	}
	
	duration := time.Since(start)
	response := map[string]interface{}{
		"result":   result,
		"duration": duration.String(),
		"message":  "CPU-intensive work completed",
	}
	
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// healthHandler provides health check endpoint
func healthHandler(w http.ResponseWriter, r *http.Request) {
	response := map[string]string{
		"status": "healthy",
		"time":   time.Now().Format(time.RFC3339),
	}
	
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}