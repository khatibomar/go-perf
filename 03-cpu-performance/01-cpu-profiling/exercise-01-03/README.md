# Exercise 01-03: Production Profiling Integration

**Difficulty:** Advanced  
**Estimated Time:** 90-120 minutes  
**Prerequisites:** Exercise 01-01 and 01-02 completion, understanding of HTTP servers and monitoring

## Objective

Implement production-ready CPU profiling infrastructure that can be safely deployed in live environments. You'll build a comprehensive profiling system with HTTP endpoints, continuous profiling, profile storage, and monitoring capabilities.

## Learning Goals

- Implement HTTP profiling endpoints for production use
- Build continuous profiling with configurable intervals
- Create profile storage and retention systems
- Develop monitoring dashboards for profile data
- Understand production profiling best practices
- Implement safe profiling with minimal overhead

## Background

Production profiling requires careful consideration of overhead, storage, security, and operational concerns. Unlike development profiling, production systems must balance observability with performance impact. This exercise teaches you to build robust profiling infrastructure suitable for live environments.

## Tasks

### Task 1: HTTP Profiling Endpoints

Implement secure and configurable HTTP profiling endpoints.

**Requirements:**
1. Create custom profiling HTTP handlers
2. Add authentication and authorization
3. Implement rate limiting for profile requests
4. Add configurable profiling parameters
5. Include health checks and status endpoints

### Task 2: Continuous Profiling System

Build a continuous profiling system that automatically collects profiles.

**Requirements:**
1. Implement scheduled profile collection
2. Add configurable sampling intervals
3. Create profile rotation and cleanup
4. Implement graceful shutdown handling
5. Add error handling and recovery

### Task 3: Profile Storage and Management

Create a comprehensive profile storage system.

**Requirements:**
1. Implement multiple storage backends (file, S3, database)
2. Add profile compression and decompression
3. Create profile metadata tracking
4. Implement retention policies
5. Add profile search and retrieval APIs

### Task 4: Monitoring Dashboard

Build a web-based monitoring dashboard for profile analysis.

**Requirements:**
1. Create a web interface for profile browsing
2. Implement profile comparison tools
3. Add trend analysis and alerting
4. Create performance metrics visualization
5. Implement real-time profiling status

## Implementation Guide

### Step 1: Project Structure

Create a comprehensive production profiling system:

```
exercise-01-03/
├── main.go
├── cmd/
│   ├── server/
│   │   └── main.go
│   └── dashboard/
│       └── main.go
├── internal/
│   ├── profiling/
│   │   ├── collector.go
│   │   ├── continuous.go
│   │   ├── http.go
│   │   └── config.go
│   ├── storage/
│   │   ├── interface.go
│   │   ├── file.go
│   │   ├── s3.go
│   │   └── memory.go
│   ├── dashboard/
│   │   ├── server.go
│   │   ├── handlers.go
│   │   └── templates.go
│   └── monitoring/
│       ├── metrics.go
│       └── alerts.go
├── web/
│   ├── static/
│   │   ├── css/
│   │   └── js/
│   └── templates/
│       ├── dashboard.html
│       ├── profiles.html
│       └── compare.html
├── configs/
│   ├── development.yaml
│   └── production.yaml
├── scripts/
│   ├── setup.sh
│   └── deploy.sh
├── go.mod
├── go.sum
└── README.md
```

### Step 2: HTTP Profiling Infrastructure

```go
// internal/profiling/http.go
package profiling

import (
    "context"
    "fmt"
    "net/http"
    "runtime/pprof"
    "strconv"
    "time"
)

type HTTPProfiler struct {
    config     *Config
    storage    storage.Interface
    rateLimiter *RateLimiter
    auth       AuthProvider
}

func NewHTTPProfiler(config *Config, storage storage.Interface) *HTTPProfiler {
    return &HTTPProfiler{
        config:      config,
        storage:     storage,
        rateLimiter: NewRateLimiter(config.RateLimit),
        auth:        NewAuthProvider(config.Auth),
    }
}

func (h *HTTPProfiler) RegisterHandlers(mux *http.ServeMux) {
    // Custom profiling endpoints
    mux.HandleFunc("/debug/pprof/profile", h.authMiddleware(h.cpuProfileHandler))
    mux.HandleFunc("/debug/pprof/heap", h.authMiddleware(h.heapProfileHandler))
    mux.HandleFunc("/debug/pprof/goroutine", h.authMiddleware(h.goroutineProfileHandler))
    
    // Profile management endpoints
    mux.HandleFunc("/profiles/list", h.authMiddleware(h.listProfilesHandler))
    mux.HandleFunc("/profiles/download", h.authMiddleware(h.downloadProfileHandler))
    mux.HandleFunc("/profiles/compare", h.authMiddleware(h.compareProfilesHandler))
    
    // Status and health endpoints
    mux.HandleFunc("/profiling/status", h.statusHandler)
    mux.HandleFunc("/profiling/health", h.healthHandler)
}

func (h *HTTPProfiler) cpuProfileHandler(w http.ResponseWriter, r *http.Request) {
    if !h.rateLimiter.Allow() {
        http.Error(w, "Rate limit exceeded", http.StatusTooManyRequests)
        return
    }
    
    // Parse duration parameter
    duration := 30 * time.Second
    if d := r.URL.Query().Get("seconds"); d != "" {
        if parsed, err := strconv.Atoi(d); err == nil {
            duration = time.Duration(parsed) * time.Second
        }
    }
    
    // Validate duration limits
    if duration > h.config.MaxProfileDuration {
        duration = h.config.MaxProfileDuration
    }
    
    // Collect profile
    profile, err := h.collectCPUProfile(r.Context(), duration)
    if err != nil {
        http.Error(w, fmt.Sprintf("Failed to collect profile: %v", err), http.StatusInternalServerError)
        return
    }
    
    // Store profile
    profileID, err := h.storage.Store(profile)
    if err != nil {
        http.Error(w, fmt.Sprintf("Failed to store profile: %v", err), http.StatusInternalServerError)
        return
    }
    
    // Return profile
    w.Header().Set("Content-Type", "application/octet-stream")
    w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=cpu-%s.prof", profileID))
    w.Write(profile.Data)
}

func (h *HTTPProfiler) authMiddleware(next http.HandlerFunc) http.HandlerFunc {
    return func(w http.ResponseWriter, r *http.Request) {
        if !h.auth.Authenticate(r) {
            http.Error(w, "Unauthorized", http.StatusUnauthorized)
            return
        }
        next(w, r)
    }
}
```

### Step 3: Continuous Profiling System

```go
// internal/profiling/continuous.go
package profiling

import (
    "context"
    "log"
    "sync"
    "time"
)

type ContinuousProfiler struct {
    config    *Config
    collector *Collector
    storage   storage.Interface
    
    mu       sync.RWMutex
    running  bool
    stopCh   chan struct{}
    doneCh   chan struct{}
}

func NewContinuousProfiler(config *Config, storage storage.Interface) *ContinuousProfiler {
    return &ContinuousProfiler{
        config:    config,
        collector: NewCollector(config),
        storage:   storage,
        stopCh:    make(chan struct{}),
        doneCh:    make(chan struct{}),
    }
}

func (cp *ContinuousProfiler) Start(ctx context.Context) error {
    cp.mu.Lock()
    if cp.running {
        cp.mu.Unlock()
        return fmt.Errorf("continuous profiler already running")
    }
    cp.running = true
    cp.mu.Unlock()
    
    go cp.run(ctx)
    return nil
}

func (cp *ContinuousProfiler) Stop() error {
    cp.mu.Lock()
    if !cp.running {
        cp.mu.Unlock()
        return fmt.Errorf("continuous profiler not running")
    }
    cp.running = false
    cp.mu.Unlock()
    
    close(cp.stopCh)
    <-cp.doneCh
    return nil
}

func (cp *ContinuousProfiler) run(ctx context.Context) {
    defer close(cp.doneCh)
    
    ticker := time.NewTicker(cp.config.CollectionInterval)
    defer ticker.Stop()
    
    // Collect initial profile
    cp.collectAndStore(ctx)
    
    for {
        select {
        case <-ctx.Done():
            return
        case <-cp.stopCh:
            return
        case <-ticker.C:
            cp.collectAndStore(ctx)
        }
    }
}

func (cp *ContinuousProfiler) collectAndStore(ctx context.Context) {
    profiles, err := cp.collector.CollectAll(ctx, cp.config.ProfileDuration)
    if err != nil {
        log.Printf("Failed to collect profiles: %v", err)
        return
    }
    
    for profileType, profile := range profiles {
        profileID, err := cp.storage.Store(profile)
        if err != nil {
            log.Printf("Failed to store %s profile: %v", profileType, err)
            continue
        }
        
        log.Printf("Stored %s profile: %s", profileType, profileID)
    }
    
    // Cleanup old profiles
    if err := cp.storage.Cleanup(cp.config.RetentionPeriod); err != nil {
        log.Printf("Failed to cleanup old profiles: %v", err)
    }
}
```

### Step 4: Profile Storage System

```go
// internal/storage/interface.go
package storage

import (
    "context"
    "time"
)

type Profile struct {
    ID          string
    Type        string
    Data        []byte
    Metadata    map[string]string
    Timestamp   time.Time
    Size        int64
    Compressed  bool
}

type Interface interface {
    Store(profile *Profile) (string, error)
    Get(id string) (*Profile, error)
    List(filter ListFilter) ([]*Profile, error)
    Delete(id string) error
    Cleanup(retentionPeriod time.Duration) error
    Compare(id1, id2 string) (*ComparisonResult, error)
}

type ListFilter struct {
    Type      string
    StartTime time.Time
    EndTime   time.Time
    Limit     int
    Offset    int
}

type ComparisonResult struct {
    Profile1    *Profile
    Profile2    *Profile
    Differences map[string]interface{}
    Summary     string
}
```

```go
// internal/storage/file.go
package storage

import (
    "compress/gzip"
    "encoding/json"
    "fmt"
    "io/ioutil"
    "os"
    "path/filepath"
    "time"
)

type FileStorage struct {
    basePath   string
    compressed bool
}

func NewFileStorage(basePath string, compressed bool) *FileStorage {
    return &FileStorage{
        basePath:   basePath,
        compressed: compressed,
    }
}

func (fs *FileStorage) Store(profile *Profile) (string, error) {
    if profile.ID == "" {
        profile.ID = generateProfileID(profile.Type, profile.Timestamp)
    }
    
    // Create directory structure
    dir := filepath.Join(fs.basePath, profile.Type, profile.Timestamp.Format("2006/01/02"))
    if err := os.MkdirAll(dir, 0755); err != nil {
        return "", fmt.Errorf("failed to create directory: %w", err)
    }
    
    // Store profile data
    dataPath := filepath.Join(dir, profile.ID+".prof")
    if fs.compressed {
        dataPath += ".gz"
        if err := fs.storeCompressed(dataPath, profile.Data); err != nil {
            return "", err
        }
        profile.Compressed = true
    } else {
        if err := ioutil.WriteFile(dataPath, profile.Data, 0644); err != nil {
            return "", fmt.Errorf("failed to write profile data: %w", err)
        }
    }
    
    // Store metadata
    metadataPath := filepath.Join(dir, profile.ID+".json")
    metadata, err := json.Marshal(profile)
    if err != nil {
        return "", fmt.Errorf("failed to marshal metadata: %w", err)
    }
    
    if err := ioutil.WriteFile(metadataPath, metadata, 0644); err != nil {
        return "", fmt.Errorf("failed to write metadata: %w", err)
    }
    
    return profile.ID, nil
}

func (fs *FileStorage) storeCompressed(path string, data []byte) error {
    file, err := os.Create(path)
    if err != nil {
        return fmt.Errorf("failed to create file: %w", err)
    }
    defer file.Close()
    
    gzWriter := gzip.NewWriter(file)
    defer gzWriter.Close()
    
    if _, err := gzWriter.Write(data); err != nil {
        return fmt.Errorf("failed to write compressed data: %w", err)
    }
    
    return nil
}
```

### Step 5: Monitoring Dashboard

```go
// internal/dashboard/server.go
package dashboard

import (
    "html/template"
    "net/http"
    "path/filepath"
    
    "../storage"
)

type Server struct {
    storage   storage.Interface
    templates *template.Template
}

func NewServer(storage storage.Interface, templatesDir string) (*Server, error) {
    templates, err := template.ParseGlob(filepath.Join(templatesDir, "*.html"))
    if err != nil {
        return nil, err
    }
    
    return &Server{
        storage:   storage,
        templates: templates,
    }, nil
}

func (s *Server) RegisterHandlers(mux *http.ServeMux) {
    // Static files
    mux.Handle("/static/", http.StripPrefix("/static/", http.FileServer(http.Dir("web/static"))))
    
    // Dashboard pages
    mux.HandleFunc("/", s.dashboardHandler)
    mux.HandleFunc("/profiles", s.profilesHandler)
    mux.HandleFunc("/profile/", s.profileDetailHandler)
    mux.HandleFunc("/compare", s.compareHandler)
    
    // API endpoints
    mux.HandleFunc("/api/profiles", s.apiProfilesHandler)
    mux.HandleFunc("/api/profile/", s.apiProfileHandler)
    mux.HandleFunc("/api/compare", s.apiCompareHandler)
    mux.HandleFunc("/api/metrics", s.apiMetricsHandler)
}

func (s *Server) dashboardHandler(w http.ResponseWriter, r *http.Request) {
    data := struct {
        Title   string
        Metrics map[string]interface{}
    }{
        Title:   "Profiling Dashboard",
        Metrics: s.getMetrics(),
    }
    
    if err := s.templates.ExecuteTemplate(w, "dashboard.html", data); err != nil {
        http.Error(w, err.Error(), http.StatusInternalServerError)
    }
}
```

## Expected Output

After completing this exercise, you should have:

### 1. Production-Ready Profiling System

```bash
# Start the profiling server
go run cmd/server/main.go -config=configs/production.yaml

# Access profiling endpoints
curl -H "Authorization: Bearer token" http://localhost:8080/debug/pprof/profile?seconds=30

# View dashboard
open http://localhost:8080/dashboard
```

### 2. Continuous Profiling

```
Profile Collection Status:
├── CPU Profiles: Collected every 5 minutes
├── Memory Profiles: Collected every 10 minutes
├── Goroutine Profiles: Collected every 15 minutes
└── Storage: 1.2GB (last 7 days)

Retention Policy:
├── Keep profiles for 30 days
├── Compress profiles older than 1 day
└── Archive profiles older than 7 days
```

### 3. Monitoring Dashboard

- Real-time profiling status
- Profile browsing and search
- Profile comparison tools
- Performance trend analysis
- Alert configuration

### 4. Storage Management

```
Profile Storage Summary:
├── Total Profiles: 2,847
├── Storage Used: 1.2GB
├── Compression Ratio: 85%
└── Average Profile Size: 450KB

Storage Backends:
├── File System: /var/profiles/
├── S3 Bucket: profiling-data
└── Database: PostgreSQL
```

## Validation

1. **HTTP Endpoints:** Verify all profiling endpoints work correctly
2. **Authentication:** Test authentication and authorization
3. **Rate Limiting:** Confirm rate limiting prevents abuse
4. **Continuous Collection:** Verify profiles are collected automatically
5. **Storage:** Test profile storage and retrieval
6. **Dashboard:** Confirm dashboard displays data correctly
7. **Performance:** Measure profiling overhead (<2%)

## Best Practices Learned

1. **Security:** Always authenticate profiling endpoints
2. **Rate Limiting:** Prevent profiling abuse
3. **Storage Management:** Implement retention policies
4. **Monitoring:** Track profiling system health
5. **Graceful Degradation:** Handle profiling failures gracefully
6. **Documentation:** Document profiling procedures

## Next Steps

- Complete Exercise 01-04: Profile-Guided Optimization
- Deploy profiling system to staging environment
- Integrate with existing monitoring infrastructure
- Set up alerting for performance regressions

---

*Exercise 01-03 of Chapter 01: CPU Profiling Techniques*