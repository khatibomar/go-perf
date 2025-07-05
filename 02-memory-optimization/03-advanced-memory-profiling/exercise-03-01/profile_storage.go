package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"sort"
	"sync"
	"time"
)

// ProfileStorage manages profile file storage and rotation
type ProfileStorage struct {
	storageDir     string
	retentionDays  int
	profiles       map[string]*ProfileInfo
	mu             sync.RWMutex
	metadataFile   string
	lastCollection time.Time
}

// ProfileInfo contains metadata about a stored profile
type ProfileInfo struct {
	Timestamp time.Time `json:"timestamp"`
	Filename  string    `json:"filename"`
	Filepath  string    `json:"filepath"`
	Type      string    `json:"type"`
	Size      int64     `json:"size"`
	Checksum  string    `json:"checksum,omitempty"`
}

// NewProfileStorage creates a new profile storage manager
func NewProfileStorage(storageDir string) *ProfileStorage {
	ps := &ProfileStorage{
		storageDir:    storageDir,
		retentionDays: 7, // Default 7-day retention
		profiles:      make(map[string]*ProfileInfo),
		metadataFile:  filepath.Join(storageDir, "profiles_metadata.json"),
	}
	
	// Load existing metadata
	ps.loadMetadata()
	
	// Start cleanup routine
	go ps.cleanupRoutine()
	
	return ps
}

// StoreProfile stores a profile and its metadata
func (ps *ProfileStorage) StoreProfile(profile *ProfileInfo) error {
	ps.mu.Lock()
	defer ps.mu.Unlock()
	
	// Get file size
	if stat, err := os.Stat(profile.Filepath); err == nil {
		profile.Size = stat.Size()
	}
	
	// Store in memory
	ps.profiles[profile.Filename] = profile
	ps.lastCollection = profile.Timestamp
	
	// Persist metadata
	return ps.saveMetadata()
}

// GetProfile retrieves profile metadata by filename
func (ps *ProfileStorage) GetProfile(filename string) (*ProfileInfo, bool) {
	ps.mu.RLock()
	defer ps.mu.RUnlock()
	
	profile, exists := ps.profiles[filename]
	return profile, exists
}

// GetProfiles returns all stored profiles
func (ps *ProfileStorage) GetProfiles() []*ProfileInfo {
	ps.mu.RLock()
	defer ps.mu.RUnlock()
	
	profiles := make([]*ProfileInfo, 0, len(ps.profiles))
	for _, profile := range ps.profiles {
		profiles = append(profiles, profile)
	}
	
	// Sort by timestamp (newest first)
	sort.Slice(profiles, func(i, j int) bool {
		return profiles[i].Timestamp.After(profiles[j].Timestamp)
	})
	
	return profiles
}

// GetProfilesInRange returns profiles within a time range
func (ps *ProfileStorage) GetProfilesInRange(start, end time.Time) []*ProfileInfo {
	ps.mu.RLock()
	defer ps.mu.RUnlock()
	
	var profiles []*ProfileInfo
	for _, profile := range ps.profiles {
		if profile.Timestamp.After(start) && profile.Timestamp.Before(end) {
			profiles = append(profiles, profile)
		}
	}
	
	// Sort by timestamp
	sort.Slice(profiles, func(i, j int) bool {
		return profiles[i].Timestamp.Before(profiles[j].Timestamp)
	})
	
	return profiles
}

// GetProfileCount returns the number of stored profiles
func (ps *ProfileStorage) GetProfileCount() int {
	ps.mu.RLock()
	defer ps.mu.RUnlock()
	return len(ps.profiles)
}

// GetLastCollectionTime returns the timestamp of the last profile collection
func (ps *ProfileStorage) GetLastCollectionTime() time.Time {
	ps.mu.RLock()
	defer ps.mu.RUnlock()
	return ps.lastCollection
}

// GetStorageStats returns storage statistics
func (ps *ProfileStorage) GetStorageStats() *StorageStats {
	ps.mu.RLock()
	defer ps.mu.RUnlock()
	
	var totalSize int64
	oldestTime := time.Now()
	newestTime := time.Time{}
	
	for _, profile := range ps.profiles {
		totalSize += profile.Size
		if profile.Timestamp.Before(oldestTime) {
			oldestTime = profile.Timestamp
		}
		if profile.Timestamp.After(newestTime) {
			newestTime = profile.Timestamp
		}
	}
	
	return &StorageStats{
		TotalProfiles: len(ps.profiles),
		TotalSize:     totalSize,
		OldestProfile: oldestTime,
		NewestProfile: newestTime,
		StorageDir:    ps.storageDir,
		RetentionDays: ps.retentionDays,
	}
}

// SetRetentionDays sets the retention period for profiles
func (ps *ProfileStorage) SetRetentionDays(days int) {
	ps.mu.Lock()
	defer ps.mu.Unlock()
	ps.retentionDays = days
}

// CleanupOldProfiles removes profiles older than retention period
func (ps *ProfileStorage) CleanupOldProfiles() error {
	ps.mu.Lock()
	defer ps.mu.Unlock()
	
	cutoff := time.Now().AddDate(0, 0, -ps.retentionDays)
	var toDelete []string
	
	// Find profiles to delete
	for filename, profile := range ps.profiles {
		if profile.Timestamp.Before(cutoff) {
			toDelete = append(toDelete, filename)
		}
	}
	
	// Delete old profiles
	for _, filename := range toDelete {
		profile := ps.profiles[filename]
		
		// Delete file
		if err := os.Remove(profile.Filepath); err != nil && !os.IsNotExist(err) {
			fmt.Printf("Warning: failed to delete profile file %s: %v\n", profile.Filepath, err)
		}
		
		// Remove from memory
		delete(ps.profiles, filename)
		
		fmt.Printf("Deleted old profile: %s\n", filename)
	}
	
	if len(toDelete) > 0 {
		return ps.saveMetadata()
	}
	
	return nil
}

// loadMetadata loads profile metadata from disk
func (ps *ProfileStorage) loadMetadata() error {
	data, err := ioutil.ReadFile(ps.metadataFile)
	if err != nil {
		if os.IsNotExist(err) {
			return nil // No metadata file yet
		}
		return fmt.Errorf("failed to read metadata file: %w", err)
	}
	
	var metadata struct {
		Profiles       map[string]*ProfileInfo `json:"profiles"`
		LastCollection time.Time               `json:"last_collection"`
	}
	
	if err := json.Unmarshal(data, &metadata); err != nil {
		return fmt.Errorf("failed to unmarshal metadata: %w", err)
	}
	
	ps.profiles = metadata.Profiles
	if ps.profiles == nil {
		ps.profiles = make(map[string]*ProfileInfo)
	}
	ps.lastCollection = metadata.LastCollection
	
	return nil
}

// saveMetadata saves profile metadata to disk
func (ps *ProfileStorage) saveMetadata() error {
	metadata := struct {
		Profiles       map[string]*ProfileInfo `json:"profiles"`
		LastCollection time.Time               `json:"last_collection"`
	}{
		Profiles:       ps.profiles,
		LastCollection: ps.lastCollection,
	}
	
	data, err := json.MarshalIndent(metadata, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal metadata: %w", err)
	}
	
	return ioutil.WriteFile(ps.metadataFile, data, 0644)
}

// cleanupRoutine runs periodic cleanup of old profiles
func (ps *ProfileStorage) cleanupRoutine() {
	ticker := time.NewTicker(1 * time.Hour) // Cleanup every hour
	defer ticker.Stop()
	
	for range ticker.C {
		if err := ps.CleanupOldProfiles(); err != nil {
			fmt.Printf("Error during profile cleanup: %v\n", err)
		}
	}
}

// StorageStats contains storage statistics
type StorageStats struct {
	TotalProfiles int       `json:"total_profiles"`
	TotalSize     int64     `json:"total_size"`
	OldestProfile time.Time `json:"oldest_profile"`
	NewestProfile time.Time `json:"newest_profile"`
	StorageDir    string    `json:"storage_dir"`
	RetentionDays int       `json:"retention_days"`
}

// FormatSize returns human-readable size
func (ss *StorageStats) FormatSize() string {
	const unit = 1024
	if ss.TotalSize < unit {
		return fmt.Sprintf("%d B", ss.TotalSize)
	}
	div, exp := int64(unit), 0
	for n := ss.TotalSize / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f %cB", float64(ss.TotalSize)/float64(div), "KMGTPE"[exp])
}