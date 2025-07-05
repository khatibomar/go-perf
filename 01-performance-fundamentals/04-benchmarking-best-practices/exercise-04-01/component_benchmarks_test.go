package benchmarks

import (
	"container/heap"
	"container/list"
	"fmt"
	"math/rand"
	"sync"
	"testing"
)

// Component benchmarks for data structures and larger components

// Stack implementation benchmarks
type Stack struct {
	items []interface{}
	mu    sync.Mutex
}

func (s *Stack) Push(item interface{}) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.items = append(s.items, item)
}

func (s *Stack) Pop() interface{} {
	s.mu.Lock()
	defer s.mu.Unlock()
	if len(s.items) == 0 {
		return nil
	}
	item := s.items[len(s.items)-1]
	s.items = s.items[:len(s.items)-1]
	return item
}

func (s *Stack) Size() int {
	s.mu.Lock()
	defer s.mu.Unlock()
	return len(s.items)
}

// Lock-free stack for comparison
type LockFreeStack struct {
	items []interface{}
}

func (s *LockFreeStack) Push(item interface{}) {
	s.items = append(s.items, item)
}

func (s *LockFreeStack) Pop() interface{} {
	if len(s.items) == 0 {
		return nil
	}
	item := s.items[len(s.items)-1]
	s.items = s.items[:len(s.items)-1]
	return item
}

func BenchmarkStackOperations(b *testing.B) {
	stack := &Stack{}
	b.ResetTimer()
	
	for i := 0; i < b.N; i++ {
		// Push operations
		for j := 0; j < 100; j++ {
			stack.Push(j)
		}
		
		// Pop operations
		for j := 0; j < 100; j++ {
			stack.Pop()
		}
	}
}

func BenchmarkLockFreeStackOperations(b *testing.B) {
	stack := &LockFreeStack{}
	b.ResetTimer()
	
	for i := 0; i < b.N; i++ {
		// Push operations
		for j := 0; j < 100; j++ {
			stack.Push(j)
		}
		
		// Pop operations
		for j := 0; j < 100; j++ {
			stack.Pop()
		}
	}
}

// Queue implementation benchmarks
type Queue struct {
	items []interface{}
	mu    sync.Mutex
}

func (q *Queue) Enqueue(item interface{}) {
	q.mu.Lock()
	defer q.mu.Unlock()
	q.items = append(q.items, item)
}

func (q *Queue) Dequeue() interface{} {
	q.mu.Lock()
	defer q.mu.Unlock()
	if len(q.items) == 0 {
		return nil
	}
	item := q.items[0]
	q.items = q.items[1:]
	return item
}

// Circular buffer queue for better performance
type CircularQueue struct {
	items []interface{}
	head  int
	tail  int
	size  int
	cap   int
	mu    sync.Mutex
}

func NewCircularQueue(capacity int) *CircularQueue {
	return &CircularQueue{
		items: make([]interface{}, capacity),
		cap:   capacity,
	}
}

func (q *CircularQueue) Enqueue(item interface{}) bool {
	q.mu.Lock()
	defer q.mu.Unlock()
	if q.size == q.cap {
		return false // Queue full
	}
	q.items[q.tail] = item
	q.tail = (q.tail + 1) % q.cap
	q.size++
	return true
}

func (q *CircularQueue) Dequeue() interface{} {
	q.mu.Lock()
	defer q.mu.Unlock()
	if q.size == 0 {
		return nil
	}
	item := q.items[q.head]
	q.head = (q.head + 1) % q.cap
	q.size--
	return item
}

func BenchmarkQueueOperations(b *testing.B) {
	queue := &Queue{}
	b.ResetTimer()
	
	for i := 0; i < b.N; i++ {
		// Enqueue operations
		for j := 0; j < 100; j++ {
			queue.Enqueue(j)
		}
		
		// Dequeue operations
		for j := 0; j < 100; j++ {
			queue.Dequeue()
		}
	}
}

func BenchmarkCircularQueueOperations(b *testing.B) {
	queue := NewCircularQueue(1000)
	b.ResetTimer()
	
	for i := 0; i < b.N; i++ {
		// Enqueue operations
		for j := 0; j < 100; j++ {
			queue.Enqueue(j)
		}
		
		// Dequeue operations
		for j := 0; j < 100; j++ {
			queue.Dequeue()
		}
	}
}

func BenchmarkStandardListOperations(b *testing.B) {
	l := list.New()
	b.ResetTimer()
	
	for i := 0; i < b.N; i++ {
		// Push operations
		for j := 0; j < 100; j++ {
			l.PushBack(j)
		}
		
		// Pop operations
		for j := 0; j < 100; j++ {
			if e := l.Front(); e != nil {
				l.Remove(e)
			}
		}
	}
}

// Priority Queue benchmarks
type Item struct {
	value    string
	priority int
	index    int
}

type PriorityQueue []*Item

func (pq PriorityQueue) Len() int { return len(pq) }

func (pq PriorityQueue) Less(i, j int) bool {
	return pq[i].priority > pq[j].priority
}

func (pq PriorityQueue) Swap(i, j int) {
	pq[i], pq[j] = pq[j], pq[i]
	pq[i].index = i
	pq[j].index = j
}

func (pq *PriorityQueue) Push(x interface{}) {
	n := len(*pq)
	item := x.(*Item)
	item.index = n
	*pq = append(*pq, item)
}

func (pq *PriorityQueue) Pop() interface{} {
	old := *pq
	n := len(old)
	item := old[n-1]
	old[n-1] = nil
	item.index = -1
	*pq = old[0 : n-1]
	return item
}

func BenchmarkPriorityQueueOperations(b *testing.B) {
	pq := make(PriorityQueue, 0)
	heap.Init(&pq)
	b.ResetTimer()
	
	for i := 0; i < b.N; i++ {
		// Push operations
		for j := 0; j < 100; j++ {
			item := &Item{
				value:    fmt.Sprintf("item%d", j),
				priority: rand.Intn(100),
			}
			heap.Push(&pq, item)
		}
		
		// Pop operations
		for j := 0; j < 100; j++ {
			if pq.Len() > 0 {
				heap.Pop(&pq)
			}
		}
	}
}

// Cache implementation benchmarks
type LRUCache struct {
	capacity int
	cache    map[string]*Node
	head     *Node
	tail     *Node
	mu       sync.RWMutex
}

type Node struct {
	key   string
	value interface{}
	prev  *Node
	next  *Node
}

func NewLRUCache(capacity int) *LRUCache {
	cache := &LRUCache{
		capacity: capacity,
		cache:    make(map[string]*Node),
		head:     &Node{},
		tail:     &Node{},
	}
	cache.head.next = cache.tail
	cache.tail.prev = cache.head
	return cache
}

func (c *LRUCache) Get(key string) (interface{}, bool) {
	c.mu.Lock()
	defer c.mu.Unlock()
	
	if node, exists := c.cache[key]; exists {
		c.moveToHead(node)
		return node.value, true
	}
	return nil, false
}

func (c *LRUCache) Put(key string, value interface{}) {
	c.mu.Lock()
	defer c.mu.Unlock()
	
	if node, exists := c.cache[key]; exists {
		node.value = value
		c.moveToHead(node)
	} else {
		newNode := &Node{key: key, value: value}
		c.cache[key] = newNode
		c.addToHead(newNode)
		
		if len(c.cache) > c.capacity {
			tail := c.removeTail()
			delete(c.cache, tail.key)
		}
	}
}

func (c *LRUCache) addToHead(node *Node) {
	node.prev = c.head
	node.next = c.head.next
	c.head.next.prev = node
	c.head.next = node
}

func (c *LRUCache) removeNode(node *Node) {
	node.prev.next = node.next
	node.next.prev = node.prev
}

func (c *LRUCache) moveToHead(node *Node) {
	c.removeNode(node)
	c.addToHead(node)
}

func (c *LRUCache) removeTail() *Node {
	last := c.tail.prev
	c.removeNode(last)
	return last
}

// Simple map cache for comparison
type SimpleCache struct {
	cache map[string]interface{}
	mu    sync.RWMutex
}

func NewSimpleCache() *SimpleCache {
	return &SimpleCache{
		cache: make(map[string]interface{}),
	}
}

func (c *SimpleCache) Get(key string) (interface{}, bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()
	value, exists := c.cache[key]
	return value, exists
}

func (c *SimpleCache) Put(key string, value interface{}) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.cache[key] = value
}

func BenchmarkLRUCacheOperations(b *testing.B) {
	cache := NewLRUCache(1000)
	keys := make([]string, 1000)
	for i := range keys {
		keys[i] = fmt.Sprintf("key%d", i)
	}
	b.ResetTimer()
	
	for i := 0; i < b.N; i++ {
		// Put operations
		for j := 0; j < 100; j++ {
			key := keys[j%len(keys)]
			cache.Put(key, fmt.Sprintf("value%d", j))
		}
		
		// Get operations
		for j := 0; j < 100; j++ {
			key := keys[j%len(keys)]
			cache.Get(key)
		}
	}
}

func BenchmarkSimpleCacheOperations(b *testing.B) {
	cache := NewSimpleCache()
	keys := make([]string, 1000)
	for i := range keys {
		keys[i] = fmt.Sprintf("key%d", i)
	}
	b.ResetTimer()
	
	for i := 0; i < b.N; i++ {
		// Put operations
		for j := 0; j < 100; j++ {
			key := keys[j%len(keys)]
			cache.Put(key, fmt.Sprintf("value%d", j))
		}
		
		// Get operations
		for j := 0; j < 100; j++ {
			key := keys[j%len(keys)]
			cache.Get(key)
		}
	}
}

// Tree structure benchmarks
type TreeNode struct {
	Value int
	Left  *TreeNode
	Right *TreeNode
}

func (t *TreeNode) Insert(value int) {
	if value < t.Value {
		if t.Left == nil {
			t.Left = &TreeNode{Value: value}
		} else {
			t.Left.Insert(value)
		}
	} else {
		if t.Right == nil {
			t.Right = &TreeNode{Value: value}
		} else {
			t.Right.Insert(value)
		}
	}
}

func (t *TreeNode) Search(value int) bool {
	if t == nil {
		return false
	}
	if t.Value == value {
		return true
	}
	if value < t.Value {
		return t.Left.Search(value)
	}
	return t.Right.Search(value)
}

func (t *TreeNode) InorderTraversal(result *[]int) {
	if t != nil {
		t.Left.InorderTraversal(result)
		*result = append(*result, t.Value)
		t.Right.InorderTraversal(result)
	}
}

func BenchmarkBinaryTreeOperations(b *testing.B) {
	root := &TreeNode{Value: 50}
	values := generateRandomSlice(1000)
	
	// Pre-populate tree
	for _, v := range values[:500] {
		root.Insert(v)
	}
	b.ResetTimer()
	
	for i := 0; i < b.N; i++ {
		// Insert operations
		for j := 0; j < 10; j++ {
			root.Insert(values[500+j%500])
		}
		
		// Search operations
		for j := 0; j < 10; j++ {
			root.Search(values[j%len(values)])
		}
	}
}

func BenchmarkBinaryTreeTraversal(b *testing.B) {
	root := &TreeNode{Value: 50}
	values := generateRandomSlice(1000)
	
	// Pre-populate tree
	for _, v := range values {
		root.Insert(v)
	}
	b.ResetTimer()
	
	for i := 0; i < b.N; i++ {
		var result []int
		root.InorderTraversal(&result)
		_ = result
	}
}

// Concurrent data structure benchmarks
func BenchmarkConcurrentMapAccess(b *testing.B) {
	m := make(map[int]int)
	var mu sync.RWMutex
	
	// Pre-populate map
	for i := 0; i < 1000; i++ {
		m[i] = i * 2
	}
	b.ResetTimer()
	
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			key := rand.Intn(1000)
			
			// 80% reads, 20% writes
			if rand.Float32() < 0.8 {
				mu.RLock()
				value := m[key]
				mu.RUnlock()
				_ = value
			} else {
				mu.Lock()
				m[key] = key * 3
				mu.Unlock()
			}
		}
	})
}

func BenchmarkSyncMapAccess(b *testing.B) {
	var m sync.Map
	
	// Pre-populate map
	for i := 0; i < 1000; i++ {
		m.Store(i, i*2)
	}
	b.ResetTimer()
	
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			key := rand.Intn(1000)
			
			// 80% reads, 20% writes
			if rand.Float32() < 0.8 {
				value, _ := m.Load(key)
				_ = value
			} else {
				m.Store(key, key*3)
			}
		}
	})
}

// Component size scaling benchmarks
func BenchmarkDataStructureScaling(b *testing.B) {
	sizes := []int{100, 1000, 10000}
	
	for _, size := range sizes {
		b.Run(fmt.Sprintf("Stack_Size%d", size), func(b *testing.B) {
			stack := &Stack{}
			b.ResetTimer()
			
			for i := 0; i < b.N; i++ {
				for j := 0; j < size/10; j++ {
					stack.Push(j)
				}
				for j := 0; j < size/10; j++ {
					stack.Pop()
				}
			}
		})
		
		b.Run(fmt.Sprintf("Queue_Size%d", size), func(b *testing.B) {
			queue := &Queue{}
			b.ResetTimer()
			
			for i := 0; i < b.N; i++ {
				for j := 0; j < size/10; j++ {
					queue.Enqueue(j)
				}
				for j := 0; j < size/10; j++ {
					queue.Dequeue()
				}
			}
		})
		
		b.Run(fmt.Sprintf("LRUCache_Size%d", size), func(b *testing.B) {
			cache := NewLRUCache(size)
			b.ResetTimer()
			
			for i := 0; i < b.N; i++ {
				for j := 0; j < size/10; j++ {
					key := fmt.Sprintf("key%d", j)
					cache.Put(key, j)
					cache.Get(key)
				}
			}
		})
	}
}