package list

import (
	"sync"
	"testing"
)

// TestLockFreeList tests the basic lock-free list
func TestLockFreeList(t *testing.T) {
	l := NewLockFreeList()
	
	// Test empty list
	if !l.IsEmpty() {
		t.Error("New list should be empty")
	}
	
	if l.Size() != 0 {
		t.Errorf("Expected size 0, got %d", l.Size())
	}
	
	// Test insert
	if !l.Insert(1, "one") {
		t.Error("Insert should succeed")
	}
	
	if !l.Insert(3, "three") {
		t.Error("Insert should succeed")
	}
	
	if !l.Insert(2, "two") {
		t.Error("Insert should succeed")
	}
	
	if l.Size() != 3 {
		t.Errorf("Expected size 3, got %d", l.Size())
	}
	
	// Test duplicate insert
	if l.Insert(2, "two_duplicate") {
		t.Error("Duplicate insert should fail")
	}
	
	// Test contains
	if !l.Contains(1) {
		t.Error("List should contain key 1")
	}
	
	if !l.Contains(2) {
		t.Error("List should contain key 2")
	}
	
	if !l.Contains(3) {
		t.Error("List should contain key 3")
	}
	
	if l.Contains(4) {
		t.Error("List should not contain key 4")
	}
	
	// Test get
	value, exists := l.Get(2)
	if !exists {
		t.Error("Get should find existing key")
	}
	if value != "two" {
		t.Errorf("Expected 'two', got %v", value)
	}
	
	_, exists = l.Get(4)
	if exists {
		t.Error("Get should not find non-existing key")
	}
	
	// Test update
	if !l.Update(2, "two_updated") {
		t.Error("Update should succeed for existing key")
	}
	
	value, exists = l.Get(2)
	if !exists || value != "two_updated" {
		t.Errorf("Expected 'two_updated', got %v, exists=%v", value, exists)
	}
	
	if l.Update(4, "four") {
		t.Error("Update should fail for non-existing key")
	}
	
	// Test delete
	if !l.Delete(2) {
		t.Error("Delete should succeed for existing key")
	}
	
	if l.Contains(2) {
		t.Error("List should not contain deleted key")
	}
	
	if l.Delete(2) {
		t.Error("Delete should fail for already deleted key")
	}
	
	if l.Size() != 2 {
		t.Errorf("Expected size 2 after deletion, got %d", l.Size())
	}
}

// TestOptimisticList tests the optimistic list
func TestOptimisticList(t *testing.T) {
	l := NewOptimisticList()
	
	// Test basic operations
	if !l.Insert(1, "one") {
		t.Error("Insert should succeed")
	}
	
	if !l.Insert(2, "two") {
		t.Error("Insert should succeed")
	}
	
	if !l.Contains(1) {
		t.Error("List should contain key 1")
	}
	
	value, exists := l.Get(1)
	if !exists || value != "one" {
		t.Errorf("Expected 'one', got %v, exists=%v", value, exists)
	}
	
	if !l.Delete(1) {
		t.Error("Delete should succeed")
	}
	
	if l.Contains(1) {
		t.Error("List should not contain deleted key")
	}
}

// TestLazyList tests the lazy list
func TestLazyList(t *testing.T) {
	l := NewLazyList()
	
	// Test basic operations
	if !l.Insert(1, "one") {
		t.Error("Insert should succeed")
	}
	
	if !l.Insert(2, "two") {
		t.Error("Insert should succeed")
	}
	
	if !l.Contains(1) {
		t.Error("List should contain key 1")
	}
	
	value, exists := l.Get(1)
	if !exists || value != "one" {
		t.Errorf("Expected 'one', got %v, exists=%v", value, exists)
	}
	
	if !l.Delete(1) {
		t.Error("Delete should succeed")
	}
	
	if l.Contains(1) {
		t.Error("List should not contain deleted key")
	}
}

// TestHarrisLockFreeList tests Harris's lock-free list
func TestHarrisLockFreeList(t *testing.T) {
	l := NewHarrisLockFreeList()
	
	// Test basic operations
	if !l.Insert(1, "one") {
		t.Error("Insert should succeed")
	}
	
	if !l.Insert(2, "two") {
		t.Error("Insert should succeed")
	}
	
	if !l.Contains(1) {
		t.Error("List should contain key 1")
	}
	
	value, exists := l.Get(1)
	if !exists || value != "one" {
		t.Errorf("Expected 'one', got %v, exists=%v", value, exists)
	}
	
	if !l.Delete(1) {
		t.Error("Delete should succeed")
	}
	
	if l.Contains(1) {
		t.Error("List should not contain deleted key")
	}
	
	// Test range operation
	l.Insert(3, "three")
	l.Insert(4, "four")
	
	count := 0
	l.Range(func(key int64, value interface{}) bool {
		count++
		return true
	})
	
	if count != 3 { // 2, 3, 4
		t.Errorf("Expected to visit 3 elements, got %d", count)
	}
}

// TestConcurrentListOperations tests concurrent operations on lists
func TestConcurrentListOperations(t *testing.T) {
	lists := []struct {
		name string
		list interface{}
	}{
		{"LockFree", NewLockFreeList()},
		{"Optimistic", NewOptimisticList()},
		{"Lazy", NewLazyList()},
		{"Harris", NewHarrisLockFreeList()},
	}
	
	for _, l := range lists {
		t.Run(l.name, func(t *testing.T) {
			var wg sync.WaitGroup
			goroutines := 10
			operationsPerGoroutine := 100
			
			// Insert phase
			for i := 0; i < goroutines; i++ {
				wg.Add(1)
				go func(id int) {
					defer wg.Done()
					for j := 0; j < operationsPerGoroutine; j++ {
						key := int64(id*operationsPerGoroutine + j)
						value := key * 10
						switch list := l.list.(type) {
						case *LockFreeList:
							list.Insert(key, value)
						case *OptimisticList:
							list.Insert(key, value)
						case *LazyList:
							list.Insert(key, value)
						case *HarrisLockFreeList:
							list.Insert(key, value)
						}
					}
				}(i)
			}
			
			wg.Wait()
			
			// Verify all elements were inserted
			for i := 0; i < goroutines*operationsPerGoroutine; i++ {
				var contains bool
				switch list := l.list.(type) {
				case *LockFreeList:
					contains = list.Contains(int64(i))
				case *OptimisticList:
					contains = list.Contains(int64(i))
				case *LazyList:
					contains = list.Contains(int64(i))
				case *HarrisLockFreeList:
					contains = list.Contains(int64(i))
				}
				
				if !contains {
					t.Errorf("List should contain key %d", i)
				}
			}
			
			// Delete phase
			for i := 0; i < goroutines; i++ {
				wg.Add(1)
				go func(id int) {
					defer wg.Done()
					for j := 0; j < operationsPerGoroutine; j++ {
						key := int64(id*operationsPerGoroutine + j)
						switch list := l.list.(type) {
						case *LockFreeList:
							list.Delete(key)
						case *OptimisticList:
							list.Delete(key)
						case *LazyList:
							list.Delete(key)
						case *HarrisLockFreeList:
							list.Delete(key)
						}
					}
				}(i)
			}
			
			wg.Wait()
			
			// Verify all elements were deleted
			var isEmpty bool
			switch list := l.list.(type) {
			case *LockFreeList:
				isEmpty = list.IsEmpty()
			case *OptimisticList:
				isEmpty = list.IsEmpty()
			case *LazyList:
				isEmpty = list.IsEmpty()
			case *HarrisLockFreeList:
				isEmpty = list.IsEmpty()
			}
			
			if !isEmpty {
				t.Error("List should be empty after deleting all elements")
			}
		})
	}
}

// TestListOrdering tests that lists maintain sorted order
func TestListOrdering(t *testing.T) {
	lists := []struct {
		name string
		list interface{}
	}{
		{"LockFree", NewLockFreeList()},
		{"Optimistic", NewOptimisticList()},
		{"Lazy", NewLazyList()},
		{"Harris", NewHarrisLockFreeList()},
	}
	
	for _, l := range lists {
		t.Run(l.name, func(t *testing.T) {
			// Insert elements in random order
			keys := []int64{5, 2, 8, 1, 9, 3, 7, 4, 6}
			for _, key := range keys {
				switch list := l.list.(type) {
				case *LockFreeList:
					list.Insert(key, key*10)
				case *OptimisticList:
					list.Insert(key, key*10)
				case *LazyList:
					list.Insert(key, key*10)
				case *HarrisLockFreeList:
					list.Insert(key, key*10)
				}
			}
			
			// Verify all keys are present
			for i := int64(1); i <= 9; i++ {
				var contains bool
				switch list := l.list.(type) {
				case *LockFreeList:
					contains = list.Contains(i)
				case *OptimisticList:
					contains = list.Contains(i)
				case *LazyList:
					contains = list.Contains(i)
				case *HarrisLockFreeList:
					contains = list.Contains(i)
				}
				
				if !contains {
					t.Errorf("List should contain key %d", i)
				}
			}
			
			// Test range operation for Harris list
			if harris, ok := l.list.(*HarrisLockFreeList); ok {
				prevKey := int64(-1)
				harris.Range(func(key int64, value interface{}) bool {
					if key <= prevKey {
						t.Errorf("Keys should be in ascending order, got %d after %d", key, prevKey)
					}
					prevKey = key
					return true
				})
			}
		})
	}
}

// TestListStressTest performs stress testing on lists
func TestListStressTest(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping stress test in short mode")
	}
	
	l := NewLockFreeList()
	var wg sync.WaitGroup
	goroutines := 50
	operationsPerGoroutine := 1000
	
	// Mixed operations: insert, delete, contains
	for i := 0; i < goroutines; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			for j := 0; j < operationsPerGoroutine; j++ {
				key := int64(id*operationsPerGoroutine + j)
				switch j % 3 {
				case 0:
					l.Insert(key, key*10)
				case 1:
					l.Contains(key)
				case 2:
					l.Delete(key)
				}
			}
		}(i)
	}
	
	wg.Wait()
	
	// List should still be functional
	l.Insert(999999, "test")
	if !l.Contains(999999) {
		t.Error("List should be functional after stress test")
	}
	
	value, exists := l.Get(999999)
	if !exists || value != "test" {
		t.Errorf("Expected 'test', got %v, exists=%v", value, exists)
	}
}

// TestListMemoryReclamation tests memory reclamation
func TestListMemoryReclamation(t *testing.T) {
	l := NewLockFreeList()
	
	// Insert many elements
	for i := 0; i < 10000; i++ {
		l.Insert(int64(i), i*10)
	}
	
	// Delete all elements
	for i := 0; i < 10000; i++ {
		l.Delete(int64(i))
	}
	
	// List should be empty
	if !l.IsEmpty() {
		t.Error("List should be empty after deleting all elements")
	}
	
	if l.Size() != 0 {
		t.Errorf("Expected size 0, got %d", l.Size())
	}
	
	// Should be able to insert again
	l.Insert(42, "test")
	if !l.Contains(42) {
		t.Error("List should contain newly inserted element")
	}
	
	value, exists := l.Get(42)
	if !exists || value != "test" {
		t.Errorf("Expected 'test', got %v, exists=%v", value, exists)
	}
}

// TestListEdgeCases tests edge cases
func TestListEdgeCases(t *testing.T) {
	l := NewLockFreeList()
	
	// Test operations on empty list
	if l.Contains(1) {
		t.Error("Empty list should not contain any keys")
	}
	
	if l.Delete(1) {
		t.Error("Delete should fail on empty list")
	}
	
	_, exists := l.Get(1)
	if exists {
		t.Error("Get should fail on empty list")
	}
	
	if l.Update(1, "test") {
		t.Error("Update should fail on empty list")
	}
	
	// Test with negative keys
	l.Insert(-1, "negative")
	l.Insert(-5, "more_negative")
	
	if !l.Contains(-1) {
		t.Error("List should contain negative key")
	}
	
	if !l.Contains(-5) {
		t.Error("List should contain negative key")
	}
	
	// Test with zero key
	l.Insert(0, "zero")
	if !l.Contains(0) {
		t.Error("List should contain zero key")
	}
	
	// Test large keys
	l.Insert(1000000, "large")
	if !l.Contains(1000000) {
		t.Error("List should contain large key")
	}
}