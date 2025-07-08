package memory

import (
	"runtime"
	"sync"
	"testing"
	"time"
	"unsafe"
)

// TestHazardPointerManager tests the hazard pointer system
func TestHazardPointerManager(t *testing.T) {
	hpm := NewHazardPointerManager(4)
	
	// Test acquiring and releasing hazard pointers
	ptr1 := unsafe.Pointer(&struct{ value int }{42})
	hp1 := hpm.Acquire()
	hp1.Set(ptr1)
	
	if hp1.Get() != ptr1 {
		t.Error("Hazard pointer should return the set pointer")
	}
	
	hp1.Clear()
	if hp1.Get() != nil {
		t.Error("Cleared hazard pointer should be nil")
	}
	
	hpm.Release(hp1)
	
	// Test retiring pointers
	ptr2 := unsafe.Pointer(&struct{ value int }{100})
	hpm.Retire(ptr2, func(p unsafe.Pointer) {
		// Cleanup function - in real usage this would free memory
		t.Logf("Cleaning up pointer %p", p)
	})
	
	// Force cleanup
	hpm.Cleanup()
	
	// Test stats
	activeCount, retiredCount := hpm.Stats()
	t.Logf("HazardPointerManager stats - Active: %d, Retired: %d", activeCount, retiredCount)
}

// TestHazardPointerConcurrency tests hazard pointers under concurrent access
func TestHazardPointerConcurrency(t *testing.T) {
	hpm := NewHazardPointerManager(8)
	
	const (
		numGoroutines = 4
		operationsPerGoroutine = 1000
	)
	
	var wg sync.WaitGroup
	
	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			
			for j := 0; j < operationsPerGoroutine; j++ {
				// Acquire hazard pointer
				hp := hpm.Acquire()
				if hp == nil {
					t.Errorf("Failed to acquire hazard pointer in goroutine %d", id)
					continue
				}
				
				// Set and use the pointer
				value := &struct{ data int }{id*operationsPerGoroutine + j}
				ptr := unsafe.Pointer(value)
				hp.Set(ptr)
				
				// Verify the pointer
				if hp.Get() != ptr {
					t.Errorf("Hazard pointer mismatch in goroutine %d", id)
				}
				
				// Clear and release
				hp.Clear()
				hpm.Release(hp)
				
				// Retire some pointers
				if j%10 == 0 {
					retirePtr := unsafe.Pointer(&struct{ retired int }{j})
					hpm.Retire(retirePtr, func(p unsafe.Pointer) {
						// Cleanup function
					})
				}
				
				runtime.Gosched()
			}
		}(i)
	}
	
	wg.Wait()
	
	// Final cleanup
	hpm.Cleanup()
	
	activeCount, retiredCount := hpm.Stats()
	t.Logf("Concurrent test stats - Active: %d, Retired: %d", activeCount, retiredCount)
}

// TestEpochManager tests the epoch-based memory reclamation system
func TestEpochManager(t *testing.T) {
	em := NewEpochManager()
	
	// Test entering and exiting critical sections
	guard := em.EnterCriticalSection()
	if guard == nil {
		t.Error("EnterCriticalSection should return a valid guard")
	}
	
	// Test retiring objects
	obj := &struct{ value int }{42}
	ptr := unsafe.Pointer(obj)
	em.RetireObject(ptr, func(p unsafe.Pointer) {
		t.Logf("Cleaning up object %p", p)
	})
	
	// Exit critical section
	em.ExitCriticalSection(guard)
	
	// Advance epoch and try reclamation
	em.TryReclaim()
	
	// Test stats
	epoch, retired, reclaimed := em.Stats()
	t.Logf("EpochManager stats - Epoch: %d, Retired: %d, Reclaimed: %d", epoch, retired, reclaimed)
}

// TestEpochManagerConcurrency tests epoch manager under concurrent access
func TestEpochManagerConcurrency(t *testing.T) {
	em := NewEpochManager()
	
	const (
		numGoroutines = 4
		operationsPerGoroutine = 500
	)
	
	var wg sync.WaitGroup
	
	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			
			for j := 0; j < operationsPerGoroutine; j++ {
				// Enter critical section
				guard := em.EnterCriticalSection()
				if guard == nil {
					t.Errorf("Failed to enter critical section in goroutine %d", id)
					continue
				}
				
				// Retire some objects
				if j%5 == 0 {
					obj := &struct{ data int }{id*operationsPerGoroutine + j}
					ptr := unsafe.Pointer(obj)
					em.RetireObject(ptr, func(p unsafe.Pointer) {
						// Cleanup function
					})
				}
				
				// Try reclamation occasionally
				if j%20 == 0 {
					em.TryReclaim()
				}
				
				// Exit critical section
				em.ExitCriticalSection(guard)
				
				runtime.Gosched()
			}
		}(i)
	}
	
	wg.Wait()
	
	// Final reclamation
	em.TryReclaim()
	
	epoch, retired, reclaimed := em.Stats()
	t.Logf("Concurrent epoch test stats - Epoch: %d, Retired: %d, Reclaimed: %d", epoch, retired, reclaimed)
}

// TestPeriodicReclamation tests the periodic reclamation functionality
func TestPeriodicReclamation(t *testing.T) {
	em := NewEpochManager()
	
	// Start periodic reclamation
	stop := PeriodicReclamation(em, 100*time.Millisecond)
	defer stop()
	
	// Retire some objects
	for i := 0; i < 10; i++ {
		obj := &struct{ value int }{i}
		ptr := unsafe.Pointer(obj)
		em.RetireObject(ptr, func(p unsafe.Pointer) {
			t.Logf("Periodic cleanup of object %p", p)
		})
	}
	
	// Wait for periodic reclamation to run
	time.Sleep(300 * time.Millisecond)
	
	epoch, retired, reclaimed := em.Stats()
	t.Logf("Periodic reclamation stats - Epoch: %d, Retired: %d, Reclaimed: %d", epoch, retired, reclaimed)
}

// BenchmarkHazardPointers benchmarks hazard pointer operations
func BenchmarkHazardPointers(b *testing.B) {
	hpm := NewHazardPointerManager(runtime.NumCPU())
	
	b.Run("AcquireRelease", func(b *testing.B) {
		b.RunParallel(func(pb *testing.PB) {
			for pb.Next() {
				hp := hpm.Acquire()
				if hp != nil {
					hpm.Release(hp)
				}
			}
		})
	})
	
	b.Run("SetClear", func(b *testing.B) {
		hp := hpm.Acquire()
		if hp == nil {
			b.Fatal("Failed to acquire hazard pointer")
		}
		defer hpm.Release(hp)
		
		ptr := unsafe.Pointer(&struct{ value int }{42})
		
		b.ResetTimer()
		b.RunParallel(func(pb *testing.PB) {
			for pb.Next() {
				hp.Set(ptr)
				hp.Clear()
			}
		})
	})
	
	b.Run("Retire", func(b *testing.B) {
		b.RunParallel(func(pb *testing.PB) {
			for pb.Next() {
				ptr := unsafe.Pointer(&struct{ value int }{42})
				hpm.Retire(ptr, func(p unsafe.Pointer) {})
			}
		})
	})
}

// BenchmarkEpochManager benchmarks epoch-based reclamation operations
func BenchmarkEpochManager(b *testing.B) {
	em := NewEpochManager()
	
	b.Run("EnterExit", func(b *testing.B) {
		b.RunParallel(func(pb *testing.PB) {
			for pb.Next() {
				guard := em.EnterCriticalSection()
				em.ExitCriticalSection(guard)
			}
		})
	})
	
	b.Run("RetireObject", func(b *testing.B) {
		b.RunParallel(func(pb *testing.PB) {
			for pb.Next() {
				ptr := unsafe.Pointer(&struct{ value int }{42})
				em.RetireObject(ptr, func(p unsafe.Pointer) {})
			}
		})
	})
	
	b.Run("TryReclaim", func(b *testing.B) {
		// Retire some objects first
		for i := 0; i < 100; i++ {
			ptr := unsafe.Pointer(&struct{ value int }{i})
			em.RetireObject(ptr, func(p unsafe.Pointer) {})
		}
		
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			em.TryReclaim()
		}
	})
}

// TestMemoryReclamationIntegration tests integration between different memory management approaches
func TestMemoryReclamationIntegration(t *testing.T) {
	t.Run("HazardPointer_vs_Epoch", func(t *testing.T) {
		// Compare hazard pointers vs epoch-based reclamation
		hpm := NewHazardPointerManager(4)
		em := NewEpochManager()
		
		const numObjects = 100
		
		// Test hazard pointer approach
		start := time.Now()
		for i := 0; i < numObjects; i++ {
			hp := hpm.Acquire()
			ptr := unsafe.Pointer(&struct{ value int }{i})
			hp.Set(ptr)
			hpm.Retire(ptr, func(p unsafe.Pointer) {})
			hp.Clear()
			hpm.Release(hp)
		}
		hpm.Cleanup()
		hpDuration := time.Since(start)
		
		// Test epoch-based approach
		start = time.Now()
		for i := 0; i < numObjects; i++ {
			guard := em.EnterCriticalSection()
			ptr := unsafe.Pointer(&struct{ value int }{i})
			em.RetireObject(ptr, func(p unsafe.Pointer) {})
			em.ExitCriticalSection(guard)
		}
		em.TryReclaim()
		epochDuration := time.Since(start)
		
		t.Logf("Performance comparison - HazardPointer: %v, Epoch: %v", hpDuration, epochDuration)
		
		// Get final stats
		hpActive, hpRetired := hpm.Stats()
		epoch, emRetired, emReclaimed := em.Stats()
		
		t.Logf("Final stats - HP(Active: %d, Retired: %d), Epoch(Epoch: %d, Retired: %d, Reclaimed: %d)",
			hpActive, hpRetired, epoch, emRetired, emReclaimed)
	})
}