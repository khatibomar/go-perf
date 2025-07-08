package profiling

import (
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// FileStorage implements ProfileStorage interface using filesystem
type FileStorage struct {
	basePath string
}

// NewFileStorage creates a new file-based profile storage
func NewFileStorage(basePath string) (*FileStorage, error) {
	if err := os.MkdirAll(basePath, 0755); err != nil {
		return nil, fmt.Errorf("failed to create storage directory: %w", err)
	}

	return &FileStorage{
		basePath: basePath,
	}, nil
}

// Store saves profile data to filesystem
func (fs *FileStorage) Store(profileType ProfileType, data []byte, metadata map[string]string) error {
	// Generate unique ID based on timestamp and profile type
	timestamp := time.Now()
	hash := sha256.Sum256(data)
	id := fmt.Sprintf("%s_%s_%x", profileType, timestamp.Format("20060102_150405"), hash[:8])

	// Create profile directory
	profileDir := filepath.Join(fs.basePath, string(profileType))
	if err := os.MkdirAll(profileDir, 0755); err != nil {
		return fmt.Errorf("failed to create profile directory: %w", err)
	}

	// Save profile data
	profileFile := filepath.Join(profileDir, id+".pb")
	if err := os.WriteFile(profileFile, data, 0644); err != nil {
		return fmt.Errorf("failed to write profile data: %w", err)
	}

	// Save metadata
	metadata["id"] = id
	metadata["size"] = fmt.Sprintf("%d", len(data))
	metadata["file"] = profileFile

	metadataFile := filepath.Join(profileDir, id+".json")
	metadataBytes, err := json.MarshalIndent(metadata, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal metadata: %w", err)
	}

	if err := os.WriteFile(metadataFile, metadataBytes, 0644); err != nil {
		return fmt.Errorf("failed to write metadata: %w", err)
	}

	return nil
}

// List returns profile metadata for the specified type since the given time
func (fs *FileStorage) List(profileType ProfileType, since time.Time) ([]ProfileMetadata, error) {
	var profiles []ProfileMetadata

	profileDir := filepath.Join(fs.basePath, string(profileType))
	if _, err := os.Stat(profileDir); os.IsNotExist(err) {
		return profiles, nil // Return empty list if directory doesn't exist
	}

	err := filepath.WalkDir(profileDir, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}

		if d.IsDir() || !strings.HasSuffix(path, ".json") {
			return nil
		}

		// Read metadata file
		metadataBytes, err := os.ReadFile(path)
		if err != nil {
			return fmt.Errorf("failed to read metadata file %s: %w", path, err)
		}

		var metadata map[string]string
		if err := json.Unmarshal(metadataBytes, &metadata); err != nil {
			return fmt.Errorf("failed to unmarshal metadata: %w", err)
		}

		// Parse timestamp
		timestampStr, ok := metadata["timestamp"]
		if !ok {
			return nil // Skip if no timestamp
		}

		timestamp, err := time.Parse(time.RFC3339, timestampStr)
		if err != nil {
			return nil // Skip if invalid timestamp
		}

		// Filter by since time
		if timestamp.Before(since) {
			return nil
		}

		// Parse duration
		var duration time.Duration
		if durationStr, ok := metadata["duration"]; ok {
			duration, _ = time.ParseDuration(durationStr)
		}

		// Get file size
		var size int64
		if sizeStr, ok := metadata["size"]; ok {
			fmt.Sscanf(sizeStr, "%d", &size)
		}

		profile := ProfileMetadata{
			ID:        metadata["id"],
			Type:      profileType,
			Timestamp: timestamp,
			Duration:  duration,
			Size:      size,
			Tags:      metadata,
		}

		profiles = append(profiles, profile)
		return nil
	})

	if err != nil {
		return nil, fmt.Errorf("failed to walk profile directory: %w", err)
	}

	return profiles, nil
}

// Get retrieves profile data by ID
func (fs *FileStorage) Get(id string) ([]byte, error) {
	// Search for the profile file across all profile type directories
	entries, err := os.ReadDir(fs.basePath)
	if err != nil {
		return nil, fmt.Errorf("failed to read storage directory: %w", err)
	}

	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}

		profileFile := filepath.Join(fs.basePath, entry.Name(), id+".pb")
		if _, err := os.Stat(profileFile); err == nil {
			data, err := os.ReadFile(profileFile)
			if err != nil {
				return nil, fmt.Errorf("failed to read profile file: %w", err)
			}
			return data, nil
		}
	}

	return nil, fmt.Errorf("profile with ID %s not found", id)
}

// Delete removes a profile by ID
func (fs *FileStorage) Delete(id string) error {
	// Search for the profile files across all profile type directories
	entries, err := os.ReadDir(fs.basePath)
	if err != nil {
		return fmt.Errorf("failed to read storage directory: %w", err)
	}

	var deleted bool
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}

		profileDir := filepath.Join(fs.basePath, entry.Name())
		profileFile := filepath.Join(profileDir, id+".pb")
		metadataFile := filepath.Join(profileDir, id+".json")

		if _, err := os.Stat(profileFile); err == nil {
			// Delete profile data file
			if err := os.Remove(profileFile); err != nil {
				return fmt.Errorf("failed to delete profile file: %w", err)
			}

			// Delete metadata file
			if err := os.Remove(metadataFile); err != nil {
				return fmt.Errorf("failed to delete metadata file: %w", err)
			}

			deleted = true
			break
		}
	}

	if !deleted {
		return fmt.Errorf("profile with ID %s not found", id)
	}

	return nil
}

// Cleanup removes profiles older than the specified time
func (fs *FileStorage) Cleanup(olderThan time.Time) error {
	entries, err := os.ReadDir(fs.basePath)
	if err != nil {
		return fmt.Errorf("failed to read storage directory: %w", err)
	}

	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}

		profileType := ProfileType(entry.Name())

		// Get all profiles for this type
		profiles, err := fs.List(profileType, time.Time{})
		if err != nil {
			continue // Skip on error
		}

		// Delete old profiles
		for _, profile := range profiles {
			if profile.Timestamp.Before(olderThan) {
				if err := fs.Delete(profile.ID); err != nil {
					// Log error but continue cleanup
					fmt.Printf("Warning: failed to delete profile %s: %v\n", profile.ID, err)
				}
			}
		}
	}

	return nil
}