package profiling

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"runtime"
	"runtime/pprof"
	"sync"
	"time"
)

type ProfileType string

const (
	CPUProfile       ProfileType = "cpu"
	MemProfile       ProfileType = "mem"
	BlockProfile     ProfileType = "block"
	MutexProfile     ProfileType = "mutex"
	GoroutineProfile ProfileType = "goroutine"
)

type LiveProfiler struct {
	config         ProfilerConfig
	activeProfiles map[ProfileType]*ProfileSession
	mu             sync.RWMutex
	storage        ProfileStorage
	scheduler      *ProfileScheduler
}

type ProfilerConfig struct {
	EnableContinuous bool
	ProfileInterval  time.Duration
	ProfileDuration  time.Duration
	MaxConcurrent    int
	StoragePath      string
	RetentionDays    int
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

// isValidProfileType checks if the given profile type is supported
func isValidProfileType(profileType ProfileType) bool {
	switch profileType {
	case CPUProfile, MemProfile, BlockProfile, MutexProfile, GoroutineProfile:
		return true
	default:
		return false
	}
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
	// Validate profile type first
	if !isValidProfileType(profileType) {
		return nil, fmt.Errorf("unsupported profile type: %s", profileType)
	}

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

func (lp *LiveProfiler) handleListProfiles(w http.ResponseWriter, r *http.Request) {
	profileTypeStr := r.URL.Query().Get("type")
	sinceStr := r.URL.Query().Get("since")

	var profileType ProfileType
	if profileTypeStr != "" {
		profileType = ProfileType(profileTypeStr)
	}

	var since time.Time
	if sinceStr != "" {
		var err error
		since, err = time.Parse(time.RFC3339, sinceStr)
		if err != nil {
			http.Error(w, "Invalid since parameter", http.StatusBadRequest)
			return
		}
	}

	profiles, err := lp.storage.List(profileType, since)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(profiles)
}

func (lp *LiveProfiler) GetActiveProfiles() map[ProfileType]*ProfileSession {
	lp.mu.RLock()
	defer lp.mu.RUnlock()

	result := make(map[ProfileType]*ProfileSession)
	for k, v := range lp.activeProfiles {
		result[k] = v
	}
	return result
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