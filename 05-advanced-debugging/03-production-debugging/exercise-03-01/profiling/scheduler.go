package profiling

import (
	"context"
	"fmt"
	"sync"
	"time"
)

// ProfileScheduler manages continuous profiling
type ProfileScheduler struct {
	profiler *LiveProfiler
	config   ProfilerConfig
	ctx      context.Context
	cancel   context.CancelFunc
	wg       sync.WaitGroup
	running  bool
	mu       sync.RWMutex
}

// NewProfileScheduler creates a new profile scheduler
func NewProfileScheduler(profiler *LiveProfiler, config ProfilerConfig) *ProfileScheduler {
	ctx, cancel := context.WithCancel(context.Background())
	return &ProfileScheduler{
		profiler: profiler,
		config:   config,
		ctx:      ctx,
		cancel:   cancel,
	}
}

// Start begins continuous profiling
func (ps *ProfileScheduler) Start() {
	ps.mu.Lock()
	defer ps.mu.Unlock()

	if ps.running {
		return
	}

	ps.running = true

	// Start cleanup routine
	ps.wg.Add(1)
	go ps.cleanupRoutine()

	// Start profiling routines for each profile type
	profileTypes := []ProfileType{CPUProfile, MemProfile, GoroutineProfile}

	for _, profileType := range profileTypes {
		ps.wg.Add(1)
		go ps.profileRoutine(profileType)
	}
}

// Stop stops continuous profiling
func (ps *ProfileScheduler) Stop() {
	ps.mu.Lock()
	defer ps.mu.Unlock()

	if !ps.running {
		return
	}

	ps.running = false
	ps.cancel()
	ps.wg.Wait()
}

// IsRunning returns whether the scheduler is currently running
func (ps *ProfileScheduler) IsRunning() bool {
	ps.mu.RLock()
	defer ps.mu.RUnlock()
	return ps.running
}

// profileRoutine runs continuous profiling for a specific profile type
func (ps *ProfileScheduler) profileRoutine(profileType ProfileType) {
	defer ps.wg.Done()

	ticker := time.NewTicker(ps.config.ProfileInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ps.ctx.Done():
			return
		case <-ticker.C:
			ps.runScheduledProfile(profileType)
		}
	}
}

// runScheduledProfile executes a single scheduled profile
func (ps *ProfileScheduler) runScheduledProfile(profileType ProfileType) {
	// Check if this profile type is already running
	activeProfiles := ps.profiler.GetActiveProfiles()
	if _, exists := activeProfiles[profileType]; exists {
		return // Skip if already running
	}

	// Determine profile duration based on type
	duration := ps.getProfileDuration(profileType)

	// Start the profile
	session, err := ps.profiler.StartProfile(profileType, duration)
	if err != nil {
		fmt.Printf("Scheduled profile failed to start: %v\n", err)
		return
	}

	fmt.Printf("Started scheduled %s profile (duration: %v)\n", profileType, session.Duration)
}

// getProfileDuration returns the appropriate duration for each profile type
func (ps *ProfileScheduler) getProfileDuration(profileType ProfileType) time.Duration {
	switch profileType {
	case CPUProfile:
		return ps.config.ProfileDuration
	case MemProfile:
		return time.Second // Memory profiles are instantaneous
	case GoroutineProfile:
		return time.Second // Goroutine profiles are instantaneous
	case BlockProfile:
		return ps.config.ProfileDuration
	case MutexProfile:
		return ps.config.ProfileDuration
	default:
		return ps.config.ProfileDuration
	}
}

// cleanupRoutine periodically cleans up old profiles
func (ps *ProfileScheduler) cleanupRoutine() {
	defer ps.wg.Done()

	// Run cleanup every hour
	cleanupInterval := time.Hour
	ticker := time.NewTicker(cleanupInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ps.ctx.Done():
			return
		case <-ticker.C:
			ps.runCleanup()
		}
	}
}

// runCleanup removes old profiles based on retention policy
func (ps *ProfileScheduler) runCleanup() {
	if ps.config.RetentionDays <= 0 {
		return // No cleanup if retention is not set
	}

	cutoff := time.Now().AddDate(0, 0, -ps.config.RetentionDays)

	if err := ps.profiler.storage.Cleanup(cutoff); err != nil {
		fmt.Printf("Profile cleanup failed: %v\n", err)
	} else {
		fmt.Printf("Profile cleanup completed (removed profiles older than %v)\n", cutoff.Format(time.RFC3339))
	}
}

// GetStatus returns the current status of the scheduler
func (ps *ProfileScheduler) GetStatus() map[string]interface{} {
	ps.mu.RLock()
	defer ps.mu.RUnlock()

	status := map[string]interface{}{
		"running":          ps.running,
		"profile_interval": ps.config.ProfileInterval.String(),
		"profile_duration": ps.config.ProfileDuration.String(),
		"retention_days":   ps.config.RetentionDays,
	}

	if ps.running {
		activeProfiles := ps.profiler.GetActiveProfiles()
		status["active_profiles"] = len(activeProfiles)
		status["active_types"] = make([]string, 0, len(activeProfiles))
		for profileType := range activeProfiles {
			status["active_types"] = append(status["active_types"].([]string), string(profileType))
		}
	}

	return status
}