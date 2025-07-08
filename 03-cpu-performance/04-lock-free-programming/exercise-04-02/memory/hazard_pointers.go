package memory

import (
	"runtime"
	"sync"
	"sync/atomic"
	"unsafe"
)

const (
	// MaxHazardPointers is the maximum number of hazard pointers per thread
	MaxHazardPointers = 8
	// RetiredListThreshold is the threshold for triggering cleanup
	RetiredListThreshold = 100
)

// HazardPointer represents a single hazard pointer
type HazardPointer struct {
	ptr    unsafe.Pointer
	active int32
	_      [14]uintptr // Cache line padding
}

// HazardPointerRecord represents a thread's hazard pointer record
type HazardPointerRecord struct {
	pointers [MaxHazardPointers]*HazardPointer
	retired  []unsafe.Pointer
	mu       sync.Mutex
	next     *HazardPointerRecord
	_        [8]uintptr // Cache line padding
}

// HazardPointerManager manages hazard pointers for safe memory reclamation
type HazardPointerManager struct {
	head   *HazardPointerRecord
	mu     sync.RWMutex
	local  sync.Map // map[int]*HazardPointerRecord (goroutine ID -> record)
	deleter func(unsafe.Pointer)
}

// NewHazardPointerManager creates a new hazard pointer manager
func NewHazardPointerManager(maxPointers int) *HazardPointerManager {
	return &HazardPointerManager{
		deleter: nil,
	}
}

// getGoroutineID returns the current goroutine ID (simplified)
func getGoroutineID() int {
	// This is a simplified approach - in production, you might use
	// a more sophisticated method to get goroutine ID
	return int(uintptr(unsafe.Pointer(&struct{}{})) >> 3)
}

// GetRecord gets or creates a hazard pointer record for the current thread
func (hpm *HazardPointerManager) GetRecord() *HazardPointerRecord {
	gid := getGoroutineID()
	if record, ok := hpm.local.Load(gid); ok {
		return record.(*HazardPointerRecord)
	}

	// Create new record
	record := &HazardPointerRecord{}
	for i := range record.pointers {
		record.pointers[i] = &HazardPointer{}
	}

	// Add to global list
	hpm.mu.Lock()
	record.next = hpm.head
	hpm.head = record
	hpm.mu.Unlock()

	hpm.local.Store(gid, record)
	return record
}

// Acquire acquires a hazard pointer
func (hpm *HazardPointerManager) Acquire() *HazardPointer {
	record := hpm.GetRecord()
	
	// Find an available hazard pointer
	for _, hp := range record.pointers {
		if atomic.CompareAndSwapInt32(&hp.active, 0, 1) {
			return hp
		}
	}
	
	// No available hazard pointer - this shouldn't happen in well-designed code
	panic("No available hazard pointers")
}

// AcquireHazardPointer acquires a hazard pointer for the given pointer
func (hpm *HazardPointerManager) AcquireHazardPointer(ptr unsafe.Pointer) *HazardPointer {
	hp := hpm.Acquire()
	hp.Set(ptr)
	return hp
}

// Release releases a hazard pointer
func (hpm *HazardPointerManager) Release(hp *HazardPointer) {
	atomic.StorePointer(&hp.ptr, nil)
	atomic.StoreInt32(&hp.active, 0)
}

// ReleaseHazardPointer releases a hazard pointer
func (hpm *HazardPointerManager) ReleaseHazardPointer(hp *HazardPointer) {
	hpm.Release(hp)
}

// Set sets the pointer value in a hazard pointer
func (hp *HazardPointer) Set(ptr unsafe.Pointer) {
	atomic.StorePointer(&hp.ptr, ptr)
}

// Get returns the pointer value from a hazard pointer
func (hp *HazardPointer) Get() unsafe.Pointer {
	return atomic.LoadPointer(&hp.ptr)
}

// Clear clears the hazard pointer
func (hp *HazardPointer) Clear() {
	atomic.StorePointer(&hp.ptr, nil)
}

// GetPointer returns the pointer value from a hazard pointer
func (hp *HazardPointer) GetPointer() unsafe.Pointer {
	return hp.Get()
}

// SetPointer sets the pointer value in a hazard pointer
func (hp *HazardPointer) SetPointer(ptr unsafe.Pointer) {
	hp.Set(ptr)
}

// Retire marks a pointer for retirement (deferred deletion)
func (hpm *HazardPointerManager) Retire(ptr unsafe.Pointer, deleter func(unsafe.Pointer)) {
	record := hpm.GetRecord()
	record.mu.Lock()
	record.retired = append(record.retired, ptr)
	should_cleanup := len(record.retired) >= RetiredListThreshold
	record.mu.Unlock()

	// Store the deleter for this retirement
	if hpm.deleter == nil {
		hpm.deleter = deleter
	}

	if should_cleanup {
		hpm.cleanup(record)
	}
}

// RetirePointer marks a pointer for retirement (deferred deletion)
func (hpm *HazardPointerManager) RetirePointer(ptr unsafe.Pointer) {
	hpm.Retire(ptr, hpm.deleter)
}

// cleanup performs cleanup of retired pointers
func (hpm *HazardPointerManager) cleanup(record *HazardPointerRecord) {
	// Collect all active hazard pointers
	activePointers := make(map[unsafe.Pointer]bool)
	
	hpm.mu.RLock()
	current := hpm.head
	for current != nil {
		for _, hp := range current.pointers {
			if atomic.LoadInt32(&hp.active) == 1 {
				ptr := atomic.LoadPointer(&hp.ptr)
				if ptr != nil {
					activePointers[ptr] = true
				}
			}
		}
		current = current.next
	}
	hpm.mu.RUnlock()

	// Clean up retired pointers that are not hazardous
	record.mu.Lock()
	var remaining []unsafe.Pointer
	for _, ptr := range record.retired {
		if !activePointers[ptr] {
			// Safe to delete
			if hpm.deleter != nil {
				hpm.deleter(ptr)
			}
		} else {
			// Still hazardous, keep for later
			remaining = append(remaining, ptr)
		}
	}
	record.retired = remaining
	record.mu.Unlock()
}

// Cleanup forces cleanup of all retired pointers
func (hpm *HazardPointerManager) Cleanup() {
	hpm.mu.RLock()
	current := hpm.head
	for current != nil {
		hpm.cleanup(current)
		current = current.next
	}
	hpm.mu.RUnlock()
	runtime.GC() // Trigger garbage collection
}

// ForceCleanup forces cleanup of all retired pointers
func (hpm *HazardPointerManager) ForceCleanup() {
	hpm.Cleanup()
}

// Stats returns statistics about the hazard pointer manager
func (hpm *HazardPointerManager) Stats() (activePointers, retiredPointers int) {
	hpm.mu.RLock()
	defer hpm.mu.RUnlock()
	
	current := hpm.head
	for current != nil {
		for _, hp := range current.pointers {
			if atomic.LoadInt32(&hp.active) == 1 {
				activePointers++
			}
		}
		current.mu.Lock()
		retiredPointers += len(current.retired)
		current.mu.Unlock()
		current = current.next
	}
	
	return activePointers, retiredPointers
}