# Chapter 03: Production Debugging Techniques

**Focus:** Safe and effective debugging in production environments

## Chapter Overview

This chapter focuses on mastering production debugging techniques that allow you to diagnose and resolve issues in live systems without causing disruption. You'll learn to implement live profiling, debug memory leaks, analyze performance bottlenecks, and handle critical incidents effectively.

## Learning Objectives

By the end of this chapter, you will:

- ✅ Implement safe live profiling techniques
- ✅ Debug memory leaks in production systems
- ✅ Analyze and resolve performance bottlenecks
- ✅ Handle critical incidents effectively
- ✅ Use non-invasive debugging methods
- ✅ Implement automated debugging workflows
- ✅ Establish production debugging best practices

## Prerequisites

- Advanced Go programming knowledge
- Understanding of production systems
- Experience with monitoring and observability
- Knowledge of system administration

## Chapter Structure

### Exercise 03-01: Live Profiling Techniques
**Objective:** Master safe live profiling in production environments

**Key Concepts:**
- Non-invasive profiling methods
- Continuous profiling implementation
- Profile analysis and interpretation
- Performance impact minimization

**Performance Targets:**
- Profiling overhead <1%
- Profile collection latency <5 seconds
- Continuous profiling uptime >99.9%
- Profile data retention 30+ days

### Exercise 03-02: Memory Leak Detection and Resolution
**Objective:** Identify and resolve memory leaks in production

**Key Concepts:**
- Memory leak detection patterns
- Heap analysis techniques
- Garbage collection optimization
- Memory usage monitoring

**Performance Targets:**
- Memory leak detection within 10 minutes
- Heap analysis completion <30 seconds
- Memory usage accuracy >99%
- Zero production impact during analysis

### Exercise 03-03: Performance Bottleneck Analysis
**Objective:** Identify and resolve performance bottlenecks

**Key Concepts:**
- Bottleneck identification methods
- Performance regression analysis
- Resource utilization optimization
- Scalability issue resolution

**Performance Targets:**
- Bottleneck identification <5 minutes
- Performance analysis overhead <0.5%
- Resolution time <30 minutes
- Performance improvement >20%

### Exercise 03-04: Incident Response and Debugging
**Objective:** Implement effective incident response procedures

**Key Concepts:**
- Incident response workflows
- Emergency debugging techniques
- Root cause analysis methods
- Post-incident optimization

**Performance Targets:**
- Incident detection <2 minutes
- Initial response <5 minutes
- Mean time to resolution <30 minutes
- Post-incident analysis within 24 hours

## Key Technologies

### Profiling Tools
- **pprof** - Go's built-in profiler
- **Continuous Profiler** - Always-on profiling
- **Pyroscope** - Continuous profiling platform
- **Datadog Profiler** - Cloud profiling service

### Memory Analysis
- **go tool pprof** - Memory profiling
- **Heap dumps** - Memory snapshots
- **GC analysis** - Garbage collection insights
- **Memory maps** - Process memory layout

### Performance Analysis
- **Flame graphs** - Performance visualization
- **CPU profiling** - Execution analysis
- **Trace analysis** - Request flow analysis
- **Benchmark comparison** - Performance regression

### Incident Management
- **PagerDuty** - Incident alerting
- **Slack/Teams** - Communication
- **Runbooks** - Response procedures
- **Post-mortems** - Learning documentation

## Implementation Patterns

### Live Profiling System
```go
// profiling/live_profiler.go
package profiling

import (
    "context"
    "fmt"
    "net/http"
    "runtime"
    "runtime/pprof"
    "sync"
    "time"

    "github.com/pkg/profile"
)

type ProfileType string

const (
    CPUProfile    ProfileType = "cpu"
    MemProfile    ProfileType = "mem"
    BlockProfile  ProfileType = "block"
    MutexProfile  ProfileType = "mutex"
    GoroutineProfile ProfileType = "goroutine"
)

type LiveProfiler struct {
    config       ProfilerConfig
    activeProfiles map[ProfileType]*ProfileSession
    mu           sync.RWMutex
    storage      ProfileStorage
    scheduler    *ProfileScheduler
}

type ProfilerConfig struct {
    EnableContinuous bool
    ProfileInterval  time.Duration
    ProfileDuration  time.Duration
    MaxConcurrent   int
    StoragePath     string
    RetentionDays   int
}

type ProfileSession struct {
    Type      ProfileType
    StartTime time.Time
    Duration  time.Duration
    FilePath  string
    Status    string
    cancel    context.CancelFunc
}

type ProfileStorage interface {
    Store(profileType ProfileType, data []byte, metadata map[string]string) error
    List(profileType ProfileType, since time.Time) ([]ProfileMetadata, error)
    Get(id string) ([]byte, error)
    Delete(id string) error
    Cleanup(olderThan time.Time) error
}

type ProfileMetadata struct {
    ID        string
    Type      ProfileType
    Timestamp time.Time
    Duration  time.Duration
    Size      int64
    Tags      map[string]string
}

func NewLiveProfiler(config ProfilerConfig, storage ProfileStorage) *LiveProfiler {
    lp := &LiveProfiler{
        config:         config,
        activeProfiles: make(map[ProfileType]*ProfileSession),
        storage:        storage,
    }
    
    if config.EnableContinuous {
        lp.scheduler = NewProfileScheduler(lp, config)
        lp.scheduler.Start()
    }
    
    return lp
}

func (lp *LiveProfiler) StartProfile(profileType ProfileType, duration time.Duration) (*ProfileSession, error) {
    lp.mu.Lock()
    defer lp.mu.Unlock()
    
    // Check if profile type is already running
    if session, exists := lp.activeProfiles[profileType]; exists {
        return nil, fmt.Errorf("profile type %s is already running since %v", profileType, session.StartTime)
    }
    
    // Check concurrent limit
    if len(lp.activeProfiles) >= lp.config.MaxConcurrent {
        return nil, fmt.Errorf("maximum concurrent profiles (%d) reached", lp.config.MaxConcurrent)
    }
    
    ctx, cancel := context.WithTimeout(context.Background(), duration)
    
    session := &ProfileSession{
        Type:      profileType,
        StartTime: time.Now(),
        Duration:  duration,
        Status:    "running",
        cancel:    cancel,
    }
    
    lp.activeProfiles[profileType] = session
    
    go lp.runProfile(ctx, session)
    
    return session, nil
}

func (lp *LiveProfiler) StopProfile(profileType ProfileType) error {
    lp.mu.Lock()
    defer lp.mu.Unlock()
    
    session, exists := lp.activeProfiles[profileType]
    if !exists {
        return fmt.Errorf("no active profile of type %s", profileType)
    }
    
    session.cancel()
    delete(lp.activeProfiles, profileType)
    
    return nil
}

func (lp *LiveProfiler) runProfile(ctx context.Context, session *ProfileSession) {
    defer func() {
        lp.mu.Lock()
        delete(lp.activeProfiles, session.Type)
        lp.mu.Unlock()
    }()
    
    var profileData []byte
    var err error
    
    switch session.Type {
    case CPUProfile:
        profileData, err = lp.collectCPUProfile(ctx, session.Duration)
    case MemProfile:
        profileData, err = lp.collectMemProfile()
    case BlockProfile:
        profileData, err = lp.collectBlockProfile()
    case MutexProfile:
        profileData, err = lp.collectMutexProfile()
    case GoroutineProfile:
        profileData, err = lp.collectGoroutineProfile()
    default:
        err = fmt.Errorf("unsupported profile type: %s", session.Type)
    }
    
    if err != nil {
        session.Status = fmt.Sprintf("failed: %v", err)
        return
    }
    
    // Store profile data
    metadata := map[string]string{
        "timestamp": session.StartTime.Format(time.RFC3339),
        "duration":  session.Duration.String(),
        "type":      string(session.Type),
        "version":   runtime.Version(),
    }
    
    if err := lp.storage.Store(session.Type, profileData, metadata); err != nil {
        session.Status = fmt.Sprintf("storage failed: %v", err)
        return
    }
    
    session.Status = "completed"
}

func (lp *LiveProfiler) collectCPUProfile(ctx context.Context, duration time.Duration) ([]byte, error) {
    var buf bytes.Buffer
    
    if err := pprof.StartCPUProfile(&buf); err != nil {
        return nil, fmt.Errorf("failed to start CPU profile: %w", err)
    }
    
    select {
    case <-ctx.Done():
        pprof.StopCPUProfile()
        return nil, ctx.Err()
    case <-time.After(duration):
        pprof.StopCPUProfile()
        return buf.Bytes(), nil
    }
}

func (lp *LiveProfiler) collectMemProfile() ([]byte, error) {
    var buf bytes.Buffer
    
    runtime.GC() // Force GC to get accurate memory stats
    
    if err := pprof.WriteHeapProfile(&buf); err != nil {
        return nil, fmt.Errorf("failed to write heap profile: %w", err)
    }
    
    return buf.Bytes(), nil
}

func (lp *LiveProfiler) collectBlockProfile() ([]byte, error) {
    var buf bytes.Buffer
    
    profile := pprof.Lookup("block")
    if profile == nil {
        return nil, fmt.Errorf("block profile not available")
    }
    
    if err := profile.WriteTo(&buf, 0); err != nil {
        return nil, fmt.Errorf("failed to write block profile: %w", err)
    }
    
    return buf.Bytes(), nil
}

func (lp *LiveProfiler) collectMutexProfile() ([]byte, error) {
    var buf bytes.Buffer
    
    profile := pprof.Lookup("mutex")
    if profile == nil {
        return nil, fmt.Errorf("mutex profile not available")
    }
    
    if err := profile.WriteTo(&buf, 0); err != nil {
        return nil, fmt.Errorf("failed to write mutex profile: %w", err)
    }
    
    return buf.Bytes(), nil
}

func (lp *LiveProfiler) collectGoroutineProfile() ([]byte, error) {
    var buf bytes.Buffer
    
    profile := pprof.Lookup("goroutine")
    if profile == nil {
        return nil, fmt.Errorf("goroutine profile not available")
    }
    
    if err := profile.WriteTo(&buf, 0); err != nil {
        return nil, fmt.Errorf("failed to write goroutine profile: %w", err)
    }
    
    return buf.Bytes(), nil
}

// HTTP handlers for remote profiling
func (lp *LiveProfiler) RegisterHTTPHandlers(mux *http.ServeMux) {
    mux.HandleFunc("/debug/pprof/start", lp.handleStartProfile)
    mux.HandleFunc("/debug/pprof/stop", lp.handleStopProfile)
    mux.HandleFunc("/debug/pprof/status", lp.handleProfileStatus)
    mux.HandleFunc("/debug/pprof/list", lp.handleListProfiles)
}

func (lp *LiveProfiler) handleStartProfile(w http.ResponseWriter, r *http.Request) {
    if r.Method != http.MethodPost {
        http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
        return
    }
    
    profileType := ProfileType(r.URL.Query().Get("type"))
    durationStr := r.URL.Query().Get("duration")
    
    duration, err := time.ParseDuration(durationStr)
    if err != nil {
        duration = lp.config.ProfileDuration
    }
    
    session, err := lp.StartProfile(profileType, duration)
    if err != nil {
        http.Error(w, err.Error(), http.StatusBadRequest)
        return
    }
    
    w.Header().Set("Content-Type", "application/json")
    json.NewEncoder(w).Encode(session)
}

func (lp *LiveProfiler) handleStopProfile(w http.ResponseWriter, r *http.Request) {
    if r.Method != http.MethodPost {
        http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
        return
    }
    
    profileType := ProfileType(r.URL.Query().Get("type"))
    
    if err := lp.StopProfile(profileType); err != nil {
        http.Error(w, err.Error(), http.StatusBadRequest)
        return
    }
    
    w.WriteHeader(http.StatusOK)
}

func (lp *LiveProfiler) handleProfileStatus(w http.ResponseWriter, r *http.Request) {
    lp.mu.RLock()
    defer lp.mu.RUnlock()
    
    w.Header().Set("Content-Type", "application/json")
    json.NewEncoder(w).Encode(lp.activeProfiles)
}

func (lp *LiveProfiler) Shutdown() error {
    if lp.scheduler != nil {
        lp.scheduler.Stop()
    }
    
    lp.mu.Lock()
    defer lp.mu.Unlock()
    
    // Stop all active profiles
    for _, session := range lp.activeProfiles {
        session.cancel()
    }
    
    return nil
}
```

### Memory Leak Detection System
```go
// debugging/memory_analyzer.go
package debugging

import (
    "context"
    "fmt"
    "runtime"
    "runtime/debug"
    "sync"
    "time"
)

type MemoryAnalyzer struct {
    config        AnalyzerConfig
    snapshots     []MemorySnapshot
    mu            sync.RWMutex
    alertThresholds AlertThresholds
    callbacks     []LeakCallback
}

type AnalyzerConfig struct {
    SnapshotInterval    time.Duration
    MaxSnapshots       int
    LeakThreshold      float64 // Percentage increase
    ConsecutiveChecks  int
    EnableGCOptimization bool
}

type MemorySnapshot struct {
    Timestamp    time.Time
    HeapAlloc    uint64
    HeapSys      uint64
    HeapIdle     uint64
    HeapInuse    uint64
    HeapReleased uint64
    HeapObjects  uint64
    StackInuse   uint64
    StackSys     uint64
    MSpanInuse   uint64
    MSpanSys     uint64
    MCacheInuse  uint64
    MCacheSys    uint64
    GCSys        uint64
    OtherSys     uint64
    NextGC       uint64
    LastGC       time.Time
    NumGC        uint32
    NumForcedGC  uint32
    GCCPUFraction float64
    PauseTotalNs uint64
    Goroutines   int
}

type AlertThresholds struct {
    HeapGrowthRate    float64
    GoroutineGrowth   int
    GCFrequency      time.Duration
    MemoryUtilization float64
}

type LeakCallback func(leak MemoryLeak)

type MemoryLeak struct {
    Type        string
    Severity    string
    Description string
    Metrics     map[string]interface{}
    Timestamp   time.Time
    Snapshots   []MemorySnapshot
}

func NewMemoryAnalyzer(config AnalyzerConfig) *MemoryAnalyzer {
    return &MemoryAnalyzer{
        config: config,
        snapshots: make([]MemorySnapshot, 0, config.MaxSnapshots),
        alertThresholds: AlertThresholds{
            HeapGrowthRate:    0.1, // 10% growth
            GoroutineGrowth:   100,
            GCFrequency:      time.Minute,
            MemoryUtilization: 0.8, // 80% utilization
        },
    }
}

func (ma *MemoryAnalyzer) Start(ctx context.Context) {
    ticker := time.NewTicker(ma.config.SnapshotInterval)
    defer ticker.Stop()
    
    for {
        select {
        case <-ctx.Done():
            return
        case <-ticker.C:
            ma.takeSnapshot()
            ma.analyzeLeaks()
        }
    }
}

func (ma *MemoryAnalyzer) takeSnapshot() {
    var m runtime.MemStats
    runtime.ReadMemStats(&m)
    
    snapshot := MemorySnapshot{
        Timestamp:     time.Now(),
        HeapAlloc:     m.Alloc,
        HeapSys:       m.HeapSys,
        HeapIdle:      m.HeapIdle,
        HeapInuse:     m.HeapInuse,
        HeapReleased:  m.HeapReleased,
        HeapObjects:   m.HeapObjects,
        StackInuse:    m.StackInuse,
        StackSys:      m.StackSys,
        MSpanInuse:    m.MSpanInuse,
        MSpanSys:      m.MSpanSys,
        MCacheInuse:   m.MCacheInuse,
        MCacheSys:     m.MCacheSys,
        GCSys:         m.GCSys,
        OtherSys:      m.OtherSys,
        NextGC:        m.NextGC,
        LastGC:        time.Unix(0, int64(m.LastGC)),
        NumGC:         m.NumGC,
        NumForcedGC:   m.NumForcedGC,
        GCCPUFraction: m.GCCPUFraction,
        PauseTotalNs:  m.PauseTotalNs,
        Goroutines:    runtime.NumGoroutine(),
    }
    
    ma.mu.Lock()
    defer ma.mu.Unlock()
    
    ma.snapshots = append(ma.snapshots, snapshot)
    
    // Keep only the last N snapshots
    if len(ma.snapshots) > ma.config.MaxSnapshots {
        ma.snapshots = ma.snapshots[1:]
    }
}

func (ma *MemoryAnalyzer) analyzeLeaks() {
    ma.mu.RLock()
    defer ma.mu.RUnlock()
    
    if len(ma.snapshots) < ma.config.ConsecutiveChecks {
        return
    }
    
    recent := ma.snapshots[len(ma.snapshots)-ma.config.ConsecutiveChecks:]
    
    // Check for heap growth
    if leak := ma.checkHeapGrowth(recent); leak != nil {
        ma.notifyLeak(*leak)
    }
    
    // Check for goroutine leaks
    if leak := ma.checkGoroutineLeak(recent); leak != nil {
        ma.notifyLeak(*leak)
    }
    
    // Check GC efficiency
    if leak := ma.checkGCEfficiency(recent); leak != nil {
        ma.notifyLeak(*leak)
    }
}

func (ma *MemoryAnalyzer) checkHeapGrowth(snapshots []MemorySnapshot) *MemoryLeak {
    if len(snapshots) < 2 {
        return nil
    }
    
    first := snapshots[0]
    last := snapshots[len(snapshots)-1]
    
    growthRate := float64(last.HeapAlloc-first.HeapAlloc) / float64(first.HeapAlloc)
    
    if growthRate > ma.alertThresholds.HeapGrowthRate {
        return &MemoryLeak{
            Type:        "heap_growth",
            Severity:    ma.getSeverity(growthRate, ma.alertThresholds.HeapGrowthRate),
            Description: fmt.Sprintf("Heap memory grew by %.2f%% over %d snapshots", growthRate*100, len(snapshots)),
            Metrics: map[string]interface{}{
                "growth_rate":    growthRate,
                "initial_heap":   first.HeapAlloc,
                "current_heap":   last.HeapAlloc,
                "time_span":      last.Timestamp.Sub(first.Timestamp),
            },
            Timestamp: time.Now(),
            Snapshots: snapshots,
        }
    }
    
    return nil
}

func (ma *MemoryAnalyzer) checkGoroutineLeak(snapshots []MemorySnapshot) *MemoryLeak {
    if len(snapshots) < 2 {
        return nil
    }
    
    first := snapshots[0]
    last := snapshots[len(snapshots)-1]
    
    growth := last.Goroutines - first.Goroutines
    
    if growth > ma.alertThresholds.GoroutineGrowth {
        return &MemoryLeak{
            Type:        "goroutine_leak",
            Severity:    ma.getSeverity(float64(growth), float64(ma.alertThresholds.GoroutineGrowth)),
            Description: fmt.Sprintf("Goroutine count increased by %d over %d snapshots", growth, len(snapshots)),
            Metrics: map[string]interface{}{
                "growth":           growth,
                "initial_count":    first.Goroutines,
                "current_count":    last.Goroutines,
                "time_span":        last.Timestamp.Sub(first.Timestamp),
            },
            Timestamp: time.Now(),
            Snapshots: snapshots,
        }
    }
    
    return nil
}

func (ma *MemoryAnalyzer) checkGCEfficiency(snapshots []MemorySnapshot) *MemoryLeak {
    if len(snapshots) < 2 {
        return nil
    }
    
    last := snapshots[len(snapshots)-1]
    
    // Check if GC is running too frequently
    avgGCInterval := time.Since(last.LastGC) / time.Duration(last.NumGC)
    
    if avgGCInterval < ma.alertThresholds.GCFrequency {
        return &MemoryLeak{
            Type:        "gc_inefficiency",
            Severity:    "warning",
            Description: fmt.Sprintf("GC running too frequently: average interval %v", avgGCInterval),
            Metrics: map[string]interface{}{
                "avg_gc_interval": avgGCInterval,
                "gc_cpu_fraction": last.GCCPUFraction,
                "num_gc":          last.NumGC,
                "num_forced_gc":   last.NumForcedGC,
            },
            Timestamp: time.Now(),
            Snapshots: snapshots,
        }
    }
    
    return nil
}

func (ma *MemoryAnalyzer) getSeverity(actual, threshold float64) string {
    ratio := actual / threshold
    switch {
    case ratio > 3:
        return "critical"
    case ratio > 2:
        return "high"
    case ratio > 1.5:
        return "medium"
    default:
        return "low"
    }
}

func (ma *MemoryAnalyzer) notifyLeak(leak MemoryLeak) {
    for _, callback := range ma.callbacks {
        go callback(leak)
    }
}

func (ma *MemoryAnalyzer) RegisterLeakCallback(callback LeakCallback) {
    ma.callbacks = append(ma.callbacks, callback)
}

func (ma *MemoryAnalyzer) GetCurrentSnapshot() MemorySnapshot {
    ma.mu.RLock()
    defer ma.mu.RUnlock()
    
    if len(ma.snapshots) == 0 {
        return MemorySnapshot{}
    }
    
    return ma.snapshots[len(ma.snapshots)-1]
}

func (ma *MemoryAnalyzer) GetSnapshots(since time.Time) []MemorySnapshot {
    ma.mu.RLock()
    defer ma.mu.RUnlock()
    
    var result []MemorySnapshot
    for _, snapshot := range ma.snapshots {
        if snapshot.Timestamp.After(since) {
            result = append(result, snapshot)
        }
    }
    
    return result
}

// Force garbage collection and return before/after stats
func (ma *MemoryAnalyzer) ForceGCAndAnalyze() (before, after MemorySnapshot, freed uint64) {
    // Take snapshot before GC
    ma.takeSnapshot()
    before = ma.GetCurrentSnapshot()
    
    // Force garbage collection
    runtime.GC()
    runtime.GC() // Run twice to ensure cleanup
    
    // Take snapshot after GC
    time.Sleep(100 * time.Millisecond) // Allow GC to complete
    ma.takeSnapshot()
    after = ma.GetCurrentSnapshot()
    
    freed = before.HeapAlloc - after.HeapAlloc
    
    return before, after, freed
}

// Optimize GC settings based on current memory patterns
func (ma *MemoryAnalyzer) OptimizeGC() {
    if !ma.config.EnableGCOptimization {
        return
    }
    
    current := ma.GetCurrentSnapshot()
    
    // Adjust GOGC based on heap size and growth patterns
    if current.HeapAlloc > 100*1024*1024 { // > 100MB
        debug.SetGCPercent(50) // More aggressive GC for large heaps
    } else {
        debug.SetGCPercent(100) // Default GC
    }
    
    // Set memory limit if heap is growing too fast
    if len(ma.snapshots) >= 2 {
        recent := ma.snapshots[len(ma.snapshots)-2:]
        growthRate := float64(recent[1].HeapAlloc-recent[0].HeapAlloc) / float64(recent[0].HeapAlloc)
        
        if growthRate > 0.2 { // 20% growth
            // Set soft memory limit to current heap * 1.5
            limit := int64(current.HeapAlloc * 3 / 2)
            debug.SetMemoryLimit(limit)
        }
    }
}
```

### Performance Bottleneck Analyzer
```go
// debugging/bottleneck_analyzer.go
package debugging

import (
    "context"
    "fmt"
    "runtime"
    "sort"
    "sync"
    "time"
)

type BottleneckAnalyzer struct {
    config      BottleneckConfig
    metrics     []PerformanceMetric
    mu          sync.RWMutex
    callbacks   []BottleneckCallback
    baselines   map[string]Baseline
}

type BottleneckConfig struct {
    SampleInterval    time.Duration
    AnalysisWindow    time.Duration
    ThresholdMultiplier float64
    MinSamples        int
}

type PerformanceMetric struct {
    Timestamp       time.Time
    CPUUsage        float64
    MemoryUsage     uint64
    GoroutineCount  int
    GCPauseTime     time.Duration
    RequestLatency  time.Duration
    RequestRate     float64
    ErrorRate       float64
    DatabaseLatency time.Duration
    CacheHitRate    float64
}

type Baseline struct {
    Metric string
    Value  float64
    StdDev float64
}

type Bottleneck struct {
    Type        string
    Component   string
    Severity    string
    Description string
    Impact      string
    Metrics     map[string]interface{}
    Timestamp   time.Time
    Suggestions []string
}

type BottleneckCallback func(bottleneck Bottleneck)

func NewBottleneckAnalyzer(config BottleneckConfig) *BottleneckAnalyzer {
    return &BottleneckAnalyzer{
        config:    config,
        metrics:   make([]PerformanceMetric, 0),
        baselines: make(map[string]Baseline),
    }
}

func (ba *BottleneckAnalyzer) Start(ctx context.Context) {
    ticker := time.NewTicker(ba.config.SampleInterval)
    defer ticker.Stop()
    
    for {
        select {
        case <-ctx.Done():
            return
        case <-ticker.C:
            ba.collectMetrics()
            ba.analyzeBottlenecks()
        }
    }
}

func (ba *BottleneckAnalyzer) collectMetrics() {
    var m runtime.MemStats
    runtime.ReadMemStats(&m)
    
    metric := PerformanceMetric{
        Timestamp:      time.Now(),
        MemoryUsage:    m.Alloc,
        GoroutineCount: runtime.NumGoroutine(),
        GCPauseTime:    time.Duration(m.PauseNs[(m.NumGC+255)%256]),
        // Other metrics would be collected from your application
    }
    
    ba.mu.Lock()
    defer ba.mu.Unlock()
    
    ba.metrics = append(ba.metrics, metric)
    
    // Keep only metrics within the analysis window
    cutoff := time.Now().Add(-ba.config.AnalysisWindow)
    for i, m := range ba.metrics {
        if m.Timestamp.After(cutoff) {
            ba.metrics = ba.metrics[i:]
            break
        }
    }
}

func (ba *BottleneckAnalyzer) analyzeBottlenecks() {
    ba.mu.RLock()
    defer ba.mu.RUnlock()
    
    if len(ba.metrics) < ba.config.MinSamples {
        return
    }
    
    // Analyze different types of bottlenecks
    bottlenecks := []Bottleneck{}
    
    if bn := ba.analyzeCPUBottleneck(); bn != nil {
        bottlenecks = append(bottlenecks, *bn)
    }
    
    if bn := ba.analyzeMemoryBottleneck(); bn != nil {
        bottlenecks = append(bottlenecks, *bn)
    }
    
    if bn := ba.analyzeGoroutineBottleneck(); bn != nil {
        bottlenecks = append(bottlenecks, *bn)
    }
    
    if bn := ba.analyzeGCBottleneck(); bn != nil {
        bottlenecks = append(bottlenecks, *bn)
    }
    
    if bn := ba.analyzeLatencyBottleneck(); bn != nil {
        bottlenecks = append(bottlenecks, *bn)
    }
    
    // Notify about bottlenecks
    for _, bottleneck := range bottlenecks {
        ba.notifyBottleneck(bottleneck)
    }
}

func (ba *BottleneckAnalyzer) analyzeCPUBottleneck() *Bottleneck {
    recent := ba.getRecentMetrics(time.Minute * 5)
    if len(recent) == 0 {
        return nil
    }
    
    avgCPU := ba.calculateAverage(recent, func(m PerformanceMetric) float64 {
        return m.CPUUsage
    })
    
    baseline := ba.getBaseline("cpu_usage")
    threshold := baseline.Value + (baseline.StdDev * ba.config.ThresholdMultiplier)
    
    if avgCPU > threshold {
        return &Bottleneck{
            Type:        "performance",
            Component:   "cpu",
            Severity:    ba.calculateSeverity(avgCPU, threshold),
            Description: fmt.Sprintf("High CPU usage detected: %.2f%% (threshold: %.2f%%)", avgCPU, threshold),
            Impact:      "High CPU usage may cause request latency and reduced throughput",
            Metrics: map[string]interface{}{
                "current_cpu": avgCPU,
                "threshold":   threshold,
                "baseline":    baseline.Value,
            },
            Timestamp: time.Now(),
            Suggestions: []string{
                "Profile CPU usage to identify hot paths",
                "Optimize algorithms and data structures",
                "Consider horizontal scaling",
                "Review goroutine usage patterns",
            },
        }
    }
    
    return nil
}

func (ba *BottleneckAnalyzer) analyzeMemoryBottleneck() *Bottleneck {
    recent := ba.getRecentMetrics(time.Minute * 5)
    if len(recent) == 0 {
        return nil
    }
    
    avgMemory := ba.calculateAverage(recent, func(m PerformanceMetric) float64 {
        return float64(m.MemoryUsage)
    })
    
    baseline := ba.getBaseline("memory_usage")
    threshold := baseline.Value + (baseline.StdDev * ba.config.ThresholdMultiplier)
    
    if avgMemory > threshold {
        return &Bottleneck{
            Type:        "resource",
            Component:   "memory",
            Severity:    ba.calculateSeverity(avgMemory, threshold),
            Description: fmt.Sprintf("High memory usage detected: %.2fMB (threshold: %.2fMB)", avgMemory/1024/1024, threshold/1024/1024),
            Impact:      "High memory usage may trigger frequent GC and cause performance degradation",
            Metrics: map[string]interface{}{
                "current_memory": avgMemory,
                "threshold":      threshold,
                "baseline":       baseline.Value,
            },
            Timestamp: time.Now(),
            Suggestions: []string{
                "Profile memory allocation patterns",
                "Identify and fix memory leaks",
                "Optimize data structures",
                "Implement object pooling",
                "Tune garbage collection settings",
            },
        }
    }
    
    return nil
}

func (ba *BottleneckAnalyzer) analyzeGoroutineBottleneck() *Bottleneck {
    recent := ba.getRecentMetrics(time.Minute * 5)
    if len(recent) == 0 {
        return nil
    }
    
    avgGoroutines := ba.calculateAverage(recent, func(m PerformanceMetric) float64 {
        return float64(m.GoroutineCount)
    })
    
    baseline := ba.getBaseline("goroutine_count")
    threshold := baseline.Value + (baseline.StdDev * ba.config.ThresholdMultiplier)
    
    if avgGoroutines > threshold {
        return &Bottleneck{
            Type:        "concurrency",
            Component:   "goroutines",
            Severity:    ba.calculateSeverity(avgGoroutines, threshold),
            Description: fmt.Sprintf("High goroutine count detected: %.0f (threshold: %.0f)", avgGoroutines, threshold),
            Impact:      "Excessive goroutines may cause scheduler overhead and memory pressure",
            Metrics: map[string]interface{}{
                "current_goroutines": avgGoroutines,
                "threshold":          threshold,
                "baseline":           baseline.Value,
            },
            Timestamp: time.Now(),
            Suggestions: []string{
                "Profile goroutine creation patterns",
                "Implement goroutine pooling",
                "Review blocking operations",
                "Optimize channel usage",
                "Check for goroutine leaks",
            },
        }
    }
    
    return nil
}

func (ba *BottleneckAnalyzer) analyzeGCBottleneck() *Bottleneck {
    recent := ba.getRecentMetrics(time.Minute * 5)
    if len(recent) == 0 {
        return nil
    }
    
    avgGCPause := ba.calculateAverage(recent, func(m PerformanceMetric) float64 {
        return float64(m.GCPauseTime.Nanoseconds())
    })
    
    baseline := ba.getBaseline("gc_pause_time")
    threshold := baseline.Value + (baseline.StdDev * ba.config.ThresholdMultiplier)
    
    if avgGCPause > threshold {
        return &Bottleneck{
            Type:        "gc",
            Component:   "garbage_collector",
            Severity:    ba.calculateSeverity(avgGCPause, threshold),
            Description: fmt.Sprintf("High GC pause time detected: %.2fms (threshold: %.2fms)", avgGCPause/1e6, threshold/1e6),
            Impact:      "Long GC pauses cause request latency spikes and reduced throughput",
            Metrics: map[string]interface{}{
                "current_gc_pause": avgGCPause,
                "threshold":        threshold,
                "baseline":         baseline.Value,
            },
            Timestamp: time.Now(),
            Suggestions: []string{
                "Tune GOGC environment variable",
                "Reduce allocation rate",
                "Implement object pooling",
                "Optimize data structures",
                "Consider using sync.Pool",
            },
        }
    }
    
    return nil
}

func (ba *BottleneckAnalyzer) analyzeLatencyBottleneck() *Bottleneck {
    recent := ba.getRecentMetrics(time.Minute * 5)
    if len(recent) == 0 {
        return nil
    }
    
    avgLatency := ba.calculateAverage(recent, func(m PerformanceMetric) float64 {
        return float64(m.RequestLatency.Nanoseconds())
    })
    
    baseline := ba.getBaseline("request_latency")
    threshold := baseline.Value + (baseline.StdDev * ba.config.ThresholdMultiplier)
    
    if avgLatency > threshold {
        return &Bottleneck{
            Type:        "latency",
            Component:   "request_processing",
            Severity:    ba.calculateSeverity(avgLatency, threshold),
            Description: fmt.Sprintf("High request latency detected: %.2fms (threshold: %.2fms)", avgLatency/1e6, threshold/1e6),
            Impact:      "High latency affects user experience and system throughput",
            Metrics: map[string]interface{}{
                "current_latency": avgLatency,
                "threshold":       threshold,
                "baseline":        baseline.Value,
            },
            Timestamp: time.Now(),
            Suggestions: []string{
                "Profile request processing paths",
                "Optimize database queries",
                "Implement caching strategies",
                "Review external service calls",
                "Optimize serialization/deserialization",
            },
        }
    }
    
    return nil
}

func (ba *BottleneckAnalyzer) getRecentMetrics(duration time.Duration) []PerformanceMetric {
    cutoff := time.Now().Add(-duration)
    var recent []PerformanceMetric
    
    for _, metric := range ba.metrics {
        if metric.Timestamp.After(cutoff) {
            recent = append(recent, metric)
        }
    }
    
    return recent
}

func (ba *BottleneckAnalyzer) calculateAverage(metrics []PerformanceMetric, extractor func(PerformanceMetric) float64) float64 {
    if len(metrics) == 0 {
        return 0
    }
    
    sum := 0.0
    for _, metric := range metrics {
        sum += extractor(metric)
    }
    
    return sum / float64(len(metrics))
}

func (ba *BottleneckAnalyzer) getBaseline(metric string) Baseline {
    if baseline, exists := ba.baselines[metric]; exists {
        return baseline
    }
    
    // Return default baseline if not found
    return Baseline{
        Metric: metric,
        Value:  0,
        StdDev: 1,
    }
}

func (ba *BottleneckAnalyzer) calculateSeverity(current, threshold float64) string {
    ratio := current / threshold
    switch {
    case ratio > 3:
        return "critical"
    case ratio > 2:
        return "high"
    case ratio > 1.5:
        return "medium"
    default:
        return "low"
    }
}

func (ba *BottleneckAnalyzer) notifyBottleneck(bottleneck Bottleneck) {
    for _, callback := range ba.callbacks {
        go callback(bottleneck)
    }
}

func (ba *BottleneckAnalyzer) RegisterBottleneckCallback(callback BottleneckCallback) {
    ba.callbacks = append(ba.callbacks, callback)
}

func (ba *BottleneckAnalyzer) UpdateBaseline(metric string, value, stdDev float64) {
    ba.baselines[metric] = Baseline{
        Metric: metric,
        Value:  value,
        StdDev: stdDev,
    }
}
```

## Profiling and Debugging

### Production Debugging Checklist
```bash
#!/bin/bash
# production-debug-checklist.sh

echo "=== Production Debugging Checklist ==="
echo "Timestamp: $(date)"
echo "Host: $(hostname)"
echo "User: $(whoami)"
echo

# 1. System Health Check
echo "1. System Health Check"
echo "   CPU Usage: $(top -bn1 | grep 'Cpu(s)' | awk '{print $2}' | cut -d'%' -f1)%"
echo "   Memory Usage: $(free | grep Mem | awk '{printf "%.2f%%", $3/$2 * 100.0}')%"
echo "   Load Average: $(uptime | awk -F'load average:' '{print $2}')"
echo "   Disk Usage: $(df -h / | awk 'NR==2 {print $5}')"
echo

# 2. Application Status
echo "2. Application Status"
APP_PID=$(pgrep -f "your-go-app")
if [ ! -z "$APP_PID" ]; then
    echo "   Application PID: $APP_PID"
    echo "   Application CPU: $(ps -p $APP_PID -o pcpu --no-headers)%"
    echo "   Application Memory: $(ps -p $APP_PID -o pmem --no-headers)%"
    echo "   Application Uptime: $(ps -p $APP_PID -o etime --no-headers)"
    echo "   File Descriptors: $(lsof -p $APP_PID 2>/dev/null | wc -l)"
    echo "   Network Connections: $(netstat -an | grep $APP_PID | wc -l)"
else
    echo "   Application: NOT RUNNING"
fi
echo

# 3. Go Runtime Stats
echo "3. Go Runtime Stats"
if [ ! -z "$APP_PID" ]; then
    echo "   Goroutines: $(curl -s http://localhost:6060/debug/vars | jq '.cmdline' 2>/dev/null || echo 'N/A')"
    echo "   Heap Size: $(curl -s http://localhost:6060/debug/vars | jq '.memstats.Alloc' 2>/dev/null || echo 'N/A')"
    echo "   GC Count: $(curl -s http://localhost:6060/debug/vars | jq '.memstats.NumGC' 2>/dev/null || echo 'N/A')"
else
    echo "   Go Runtime: NOT AVAILABLE"
fi
echo

# 4. Recent Errors
echo "4. Recent Errors (last 100 lines)"
if [ -f "/var/log/app.log" ]; then
    tail -100 /var/log/app.log | grep -i error | tail -5
else
    echo "   Log file not found"
fi
echo

# 5. Network Connectivity
echo "5. Network Connectivity"
echo "   Listening Ports: $(netstat -tuln | grep LISTEN | wc -l)"
echo "   Active Connections: $(netstat -an | grep ESTABLISHED | wc -l)"
echo "   Failed Connections: $(netstat -s | grep 'failed connection attempts' || echo 'N/A')"
echo

# 6. Disk I/O
echo "6. Disk I/O"
echo "   Read IOPS: $(iostat -x 1 1 | awk 'NR>3 && $1!="" {print $4}' | head -1)"
echo "   Write IOPS: $(iostat -x 1 1 | awk 'NR>3 && $1!="" {print $5}' | head -1)"
echo "   I/O Wait: $(iostat -x 1 1 | awk 'NR>3 && $1!="" {print $10}' | head -1)%"
echo

echo "=== Debugging Checklist Complete ==="
```

### Emergency Response Script
```bash
#!/bin/bash
# emergency-response.sh

APP_NAME="your-go-app"
LOG_DIR="/var/log/emergency-debug"
TIMESTAMP=$(date +"%Y%m%d_%H%M%S")
DEBUG_DIR="$LOG_DIR/$TIMESTAMP"

# Create debug directory
mkdir -p $DEBUG_DIR

echo "Emergency debugging started at $(date)" | tee $DEBUG_DIR/debug.log

# 1. Capture system state
echo "Capturing system state..." | tee -a $DEBUG_DIR/debug.log
top -bn1 > $DEBUG_DIR/top.txt
ps aux > $DEBUG_DIR/processes.txt
free -h > $DEBUG_DIR/memory.txt
df -h > $DEBUG_DIR/disk.txt
netstat -tuln > $DEBUG_DIR/network.txt

# 2. Capture application state
APP_PID=$(pgrep -f $APP_NAME)
if [ ! -z "$APP_PID" ]; then
    echo "Capturing application state for PID $APP_PID..." | tee -a $DEBUG_DIR/debug.log
    
    # Process details
    ps -p $APP_PID -o pid,ppid,pcpu,pmem,vsz,rss,etime,cmd > $DEBUG_DIR/app_process.txt
    
    # Memory map
    pmap -x $APP_PID > $DEBUG_DIR/app_memory_map.txt 2>/dev/null
    
    # Open files
    lsof -p $APP_PID > $DEBUG_DIR/app_files.txt 2>/dev/null
    
    # Stack traces
    kill -QUIT $APP_PID 2>/dev/null
    sleep 2
    
    # Go profiles (if pprof is available)
    if curl -s http://localhost:6060/debug/pprof/ >/dev/null 2>&1; then
        echo "Capturing Go profiles..." | tee -a $DEBUG_DIR/debug.log
        
        # CPU profile
        curl -s "http://localhost:6060/debug/pprof/profile?seconds=30" > $DEBUG_DIR/cpu_profile.pb.gz &
        
        # Memory profile
        curl -s "http://localhost:6060/debug/pprof/heap" > $DEBUG_DIR/heap_profile.pb.gz
        
        # Goroutine profile
        curl -s "http://localhost:6060/debug/pprof/goroutine" > $DEBUG_DIR/goroutine_profile.txt
        
        # Runtime vars
        curl -s "http://localhost:6060/debug/vars" > $DEBUG_DIR/runtime_vars.json
        
        wait # Wait for CPU profile to complete
    fi
else
    echo "Application not running" | tee -a $DEBUG_DIR/debug.log
fi

# 3. Capture logs
echo "Capturing logs..." | tee -a $DEBUG_DIR/debug.log
if [ -f "/var/log/app.log" ]; then
    tail -1000 /var/log/app.log > $DEBUG_DIR/app_recent.log
fi

# System logs
journalctl -u $APP_NAME --since "1 hour ago" > $DEBUG_DIR/systemd.log 2>/dev/null
dmesg | tail -100 > $DEBUG_DIR/dmesg.log

# 4. Network diagnostics
echo "Running network diagnostics..." | tee -a $DEBUG_DIR/debug.log
ss -tuln > $DEBUG_DIR/sockets.txt
netstat -s > $DEBUG_DIR/network_stats.txt

# 5. Create summary
echo "Creating summary..." | tee -a $DEBUG_DIR/debug.log
cat > $DEBUG_DIR/summary.txt << EOF
Emergency Debug Summary
======================
Timestamp: $(date)
Host: $(hostname)
Application: $APP_NAME
PID: ${APP_PID:-"NOT RUNNING"}

System Resources:
- CPU Usage: $(top -bn1 | grep 'Cpu(s)' | awk '{print $2}' | cut -d'%' -f1)%
- Memory Usage: $(free | grep Mem | awk '{printf "%.2f%%", $3/$2 * 100.0}')%
- Load Average: $(uptime | awk -F'load average:' '{print $2}')
- Disk Usage: $(df -h / | awk 'NR==2 {print $5}')

Files Captured:
$(ls -la $DEBUG_DIR)

Next Steps:
1. Analyze profiles with 'go tool pprof'
2. Review application logs for errors
3. Check system logs for issues
4. Monitor resource usage trends
5. Consider scaling or optimization
EOF

echo "Emergency debugging complete. Files saved to: $DEBUG_DIR" | tee -a $DEBUG_DIR/debug.log
echo "Summary:" | tee -a $DEBUG_DIR/debug.log
cat $DEBUG_DIR/summary.txt | tee -a $DEBUG_DIR/debug.log
```

## Performance Optimization Techniques

### 1. Live Profiling Optimization
- Use sampling-based profiling
- Implement profile rotation
- Minimize profiling overhead
- Use continuous profiling platforms

### 2. Memory Debugging Optimization
- Implement automated leak detection
- Use memory-efficient data structures
- Optimize garbage collection
- Monitor memory patterns

### 3. Performance Analysis Optimization
- Use statistical analysis
- Implement baseline comparisons
- Automate bottleneck detection
- Use machine learning for anomaly detection

### 4. Incident Response Optimization
- Automate data collection
- Implement runbook automation
- Use chatops for coordination
- Implement post-incident learning

## Common Pitfalls

❌ **Debugging Issues:**
- Invasive debugging techniques
- High debugging overhead
- Poor data collection
- Inadequate analysis tools

❌ **Production Impact:**
- Disrupting live systems
- Causing performance degradation
- Inadequate safety measures
- Poor rollback procedures

❌ **Analysis Problems:**
- Insufficient data collection
- Poor correlation analysis
- Missing context information
- Inadequate documentation

❌ **Incident Response:**
- Slow response times
- Poor communication
- Inadequate escalation
- Missing post-incident analysis

## Best Practices

### Safe Debugging
- Use non-invasive techniques
- Implement safety limits
- Monitor debugging impact
- Have rollback procedures
- Test debugging tools

### Data Collection
- Collect comprehensive data
- Use structured formats
- Implement data retention
- Ensure data security
- Document collection procedures

### Analysis Methodology
- Use systematic approaches
- Correlate multiple data sources
- Implement automated analysis
- Document findings
- Share knowledge

### Incident Management
- Implement clear procedures
- Use effective communication
- Automate response actions
- Conduct post-incident reviews
- Continuously improve

## Success Criteria

- [ ] Safe live profiling implementation
- [ ] Effective memory leak detection
- [ ] Automated bottleneck analysis
- [ ] Robust incident response procedures
- [ ] Debugging overhead <1%
- [ ] Incident detection <2 minutes
- [ ] Mean time to resolution <30 minutes
- [ ] Team trained on debugging procedures

## Real-World Applications

- **Production Monitoring:** Continuous health monitoring
- **Performance Optimization:** Data-driven improvements
- **Incident Response:** Quick problem resolution
- **Capacity Planning:** Resource requirement analysis
- **Quality Assurance:** Performance regression detection
- **Cost Optimization:** Resource usage optimization

## Next Steps

After completing this chapter:
1. Proceed to **Chapter 04: Performance Testing and Load Testing**
2. Implement production debugging in your systems
3. Study advanced debugging techniques
4. Explore automated debugging solutions
5. Develop debugging best practices

---

**Ready to master production debugging?** Start with [Exercise 03-01: Live Profiling Techniques](./exercise-03-01/README.md)

*Chapter 03 of Module 05: Advanced Debugging and Production*