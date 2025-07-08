package memory

import (
	"runtime"
	"sync"
	"sync/atomic"
	"time"
	"unsafe"
)

const (
	// EpochIncrement is the value added to advance epochs
	EpochIncrement = 2
	// MaxRetiredPerEpoch is the maximum number of retired objects per epoch
	MaxRetiredPerEpoch = 1000
)

// EpochRecord represents a thread's epoch participation record
type EpochRecord struct {
	epoch   uint64
	active  int32
	retired [3][]unsafe.Pointer // Retired objects for each epoch
	mu      sync.Mutex
	next    *EpochRecord
	_       [8]uintptr // Cache line padding
}

// EpochManager manages epoch-based memory reclamation
type EpochManager struct {
	globalEpoch uint64
	head       *EpochRecord
	mu          sync.RWMutex
	local       sync.Map // map[int]*EpochRecord (goroutine ID -> record)
	deleter     func(unsafe.Pointer)
	_           [8]uintptr // Cache line padding
}

// NewEpochManager creates a new epoch manager
func NewEpochManager() *EpochManager {
	return &EpochManager{
		globalEpoch: 0,
		deleter:     nil,
	}
}

// GetRecord gets or creates an epoch record for the current thread
func (em *EpochManager) GetRecord() *EpochRecord {
	gid := getGoroutineID()
	if record, ok := em.local.Load(gid); ok {
		return record.(*EpochRecord)
	}

	// Create new record
	record := &EpochRecord{}
	for i := range record.retired {
		record.retired[i] = make([]unsafe.Pointer, 0, MaxRetiredPerEpoch)
	}

	// Add to global list
	em.mu.Lock()
	record.next = em.head
	em.head = record
	em.mu.Unlock()

	em.local.Store(gid, record)
	return record
}

// EnterCriticalSection marks the beginning of a critical section
func (em *EpochManager) EnterCriticalSection() *EpochRecord {
	record := em.GetRecord()
	atomic.StoreInt32(&record.active, 1)
	
	// Load current global epoch
	globalEpoch := atomic.LoadUint64(&em.globalEpoch)
	atomic.StoreUint64(&record.epoch, globalEpoch)
	
	return record
}

// ExitCriticalSection marks the end of a critical section
func (em *EpochManager) ExitCriticalSection(record *EpochRecord) {
	atomic.StoreInt32(&record.active, 0)
}

// RetireObject marks an object for retirement with a deleter function
func (em *EpochManager) RetireObject(ptr unsafe.Pointer, deleter func(unsafe.Pointer)) {
	record := em.GetRecord()
	globalEpoch := atomic.LoadUint64(&em.globalEpoch)
	epochIndex := (globalEpoch / EpochIncrement) % 3
	
	// Store the deleter for this retirement
	if em.deleter == nil {
		em.deleter = deleter
	}
	
	record.mu.Lock()
	record.retired[epochIndex] = append(record.retired[epochIndex], ptr)
	shouldTryAdvance := len(record.retired[epochIndex]) >= MaxRetiredPerEpoch
	record.mu.Unlock()
	
	if shouldTryAdvance {
		em.TryAdvanceEpoch()
	}
}

// TryAdvanceEpoch attempts to advance the global epoch
func (em *EpochManager) TryAdvanceEpoch() bool {
	currentEpoch := atomic.LoadUint64(&em.globalEpoch)
	nextEpoch := currentEpoch + EpochIncrement
	
	// Check if all threads have moved past the epoch we want to reclaim
	canAdvance := em.canAdvanceToEpoch(nextEpoch)
	if !canAdvance {
		return false
	}
	
	// Try to advance the global epoch
	if !atomic.CompareAndSwapUint64(&em.globalEpoch, currentEpoch, nextEpoch) {
		return false
	}
	
	// Successfully advanced, now we can reclaim objects from the old epoch
	em.reclaimEpoch((currentEpoch / EpochIncrement) % 3)
	return true
}

// canAdvanceToEpoch checks if we can safely advance to the given epoch
func (em *EpochManager) canAdvanceToEpoch(targetEpoch uint64) bool {
	em.mu.RLock()
	defer em.mu.RUnlock()
	
	current := em.head
	for current != nil {
		if atomic.LoadInt32(&current.active) == 1 {
			threadEpoch := atomic.LoadUint64(&current.epoch)
			if threadEpoch < targetEpoch {
				// Thread is still in an older epoch
				return false
			}
		}
		current = current.next
	}
	
	return true
}

// reclaimEpoch reclaims objects from the specified epoch
func (em *EpochManager) reclaimEpoch(epochIndex uint64) {
	em.mu.RLock()
	current := em.head
	for current != nil {
		current.mu.Lock()
		for _, ptr := range current.retired[epochIndex] {
			if em.deleter != nil {
				em.deleter(ptr)
			}
		}
		current.retired[epochIndex] = current.retired[epochIndex][:0] // Clear slice
		current.mu.Unlock()
		current = current.next
	}
	em.mu.RUnlock()
}

// ForceReclamation forces reclamation of all retired objects
func (em *EpochManager) ForceReclamation() {
	// Advance epoch multiple times to ensure all objects are reclaimed
	for i := 0; i < 3; i++ {
		em.TryAdvanceEpoch()
		runtime.Gosched() // Allow other goroutines to exit critical sections
	}
	
	// Force reclaim all epochs
	for i := uint64(0); i < 3; i++ {
		em.reclaimEpoch(i)
	}
	
	runtime.GC() // Trigger garbage collection
}

// TryReclaim attempts to reclaim retired objects
func (em *EpochManager) TryReclaim() {
	em.TryAdvanceEpoch()
}

// Stats returns statistics about the epoch manager
func (em *EpochManager) Stats() (epoch uint64, retired int, reclaimed int) {
	epoch = atomic.LoadUint64(&em.globalEpoch)
	
	em.mu.RLock()
	defer em.mu.RUnlock()
	
	current := em.head
	for current != nil {
		current.mu.Lock()
		for i := range current.retired {
			retired += len(current.retired[i])
		}
		current.mu.Unlock()
		
		current = current.next
	}
	
	// For simplicity, we'll return 0 for reclaimed count
	// In a real implementation, you'd track this separately
	return epoch, retired, 0
}

// EpochGuard provides RAII-style epoch management
type EpochGuard struct {
	manager *EpochManager
	record  *EpochRecord
}

// NewEpochGuard creates a new epoch guard
func (em *EpochManager) NewEpochGuard() *EpochGuard {
	record := em.GetRecord()
	atomic.StoreInt32(&record.active, 1)
	
	// Load current global epoch
	globalEpoch := atomic.LoadUint64(&em.globalEpoch)
	atomic.StoreUint64(&record.epoch, globalEpoch)
	
	return &EpochGuard{
		manager: em,
		record:  record,
	}
}

// Exit releases the epoch guard
func (eg *EpochGuard) Exit() {
	if eg.record != nil {
		eg.manager.ExitCriticalSection(eg.record)
		eg.record = nil
	}
}

// Release releases the epoch guard
func (eg *EpochGuard) Release() {
	eg.Exit()
}

// WithEpochGuard executes a function within an epoch guard
func (em *EpochManager) WithEpochGuard(fn func()) {
	guard := em.NewEpochGuard()
	defer guard.Release()
	fn()
}

// PeriodicReclamation starts a background goroutine for periodic reclamation
func (em *EpochManager) PeriodicReclamation(intervalMs int) chan struct{} {
	stop := make(chan struct{})
	
	go func() {
		ticker := time.NewTimer(time.Duration(intervalMs) * time.Millisecond)
		defer ticker.Stop()
		
		for {
			select {
			case <-stop:
				return
			case <-ticker.C:
				em.TryAdvanceEpoch()
				ticker.Reset(time.Duration(intervalMs) * time.Millisecond)
			}
		}
	}()
	
	return stop
}

// PeriodicReclamation starts periodic reclamation with the given interval
func PeriodicReclamation(em *EpochManager, interval time.Duration) func() {
	stop := make(chan struct{})
	
	go func() {
		ticker := time.NewTicker(interval)
		defer ticker.Stop()
		
		for {
			select {
			case <-stop:
				return
			case <-ticker.C:
				em.TryReclaim()
			}
		}
	}()
	
	return func() {
		close(stop)
	}
}