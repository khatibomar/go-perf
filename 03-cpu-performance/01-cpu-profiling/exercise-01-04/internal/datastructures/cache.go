package datastructures

import (
	"sync"
)

// SearchCache provides caching for search operations
type SearchCache struct {
	mu       sync.RWMutex
	cache    map[int]bool
	maxSize  int
	hits     int
	misses   int
}

// NewSearchCache creates a new search cache with specified maximum size
func NewSearchCache(maxSize int) *SearchCache {
	return &SearchCache{
		cache:   make(map[int]bool),
		maxSize: maxSize,
	}
}

// Get retrieves a value from the cache
func (sc *SearchCache) Get(key int) (bool, bool) {
	sc.mu.RLock()
	defer sc.mu.RUnlock()
	
	value, found := sc.cache[key]
	if found {
		sc.hits++
	} else {
		sc.misses++
	}
	return value, found
}

// Set stores a value in the cache
func (sc *SearchCache) Set(key int, value bool) {
	sc.mu.Lock()
	defer sc.mu.Unlock()
	
	// Simple eviction policy: clear cache when full
	if len(sc.cache) >= sc.maxSize {
		sc.cache = make(map[int]bool)
	}
	
	sc.cache[key] = value
}

// Stats returns cache statistics
func (sc *SearchCache) Stats() (hits, misses int, hitRate float64) {
	sc.mu.RLock()
	defer sc.mu.RUnlock()
	
	total := sc.hits + sc.misses
	if total == 0 {
		return sc.hits, sc.misses, 0.0
	}
	return sc.hits, sc.misses, float64(sc.hits) / float64(total)
}

// Clear empties the cache
func (sc *SearchCache) Clear() {
	sc.mu.Lock()
	defer sc.mu.Unlock()
	
	sc.cache = make(map[int]bool)
	sc.hits = 0
	sc.misses = 0
}

// Size returns the current cache size
func (sc *SearchCache) Size() int {
	sc.mu.RLock()
	defer sc.mu.RUnlock()
	return len(sc.cache)
}