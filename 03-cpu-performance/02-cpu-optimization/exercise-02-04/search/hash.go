package search

import (
	"hash/fnv"
	"math"
)

// KeyValue represents a key-value pair
type KeyValue struct {
	Key   string
	Value interface{}
}

// HashTable represents a basic hash table
type HashTable struct {
	buckets [][]KeyValue
	size    int
	count   int
}

// NewHashTable creates a new hash table
func NewHashTable() *HashTable {
	initialSize := 16
	return &HashTable{
		buckets: make([][]KeyValue, initialSize),
		size:    initialSize,
		count:   0,
	}
}

// Insert adds a key-value pair to the hash table
func (ht *HashTable) Insert(key string, value interface{}) {
	// Resize if load factor > 0.75
	if float64(ht.count)/float64(ht.size) > 0.75 {
		ht.resize()
	}

	hash := ht.hash(key)
	bucket := hash % ht.size

	// Check if key already exists
	for i, kv := range ht.buckets[bucket] {
		if kv.Key == key {
			ht.buckets[bucket][i].Value = value
			return
		}
	}

	// Add new key-value pair
	ht.buckets[bucket] = append(ht.buckets[bucket], KeyValue{Key: key, Value: value})
	ht.count++
}

// Search finds a value by key
func (ht *HashTable) Search(key string) (interface{}, bool) {
	hash := ht.hash(key)
	bucket := hash % ht.size

	for _, kv := range ht.buckets[bucket] {
		if kv.Key == key {
			return kv.Value, true
		}
	}

	return nil, false
}

// Delete removes a key-value pair
func (ht *HashTable) Delete(key string) bool {
	hash := ht.hash(key)
	bucket := hash % ht.size

	for i, kv := range ht.buckets[bucket] {
		if kv.Key == key {
			// Remove element
			ht.buckets[bucket] = append(ht.buckets[bucket][:i], ht.buckets[bucket][i+1:]...)
			ht.count--
			return true
		}
	}

	return false
}

// hash computes hash value for a key
func (ht *HashTable) hash(key string) int {
	h := fnv.New32a()
	h.Write([]byte(key))
	return int(h.Sum32())
}

// resize doubles the hash table size
func (ht *HashTable) resize() {
	oldBuckets := ht.buckets
	ht.size *= 2
	ht.buckets = make([][]KeyValue, ht.size)
	ht.count = 0

	// Rehash all elements
	for _, bucket := range oldBuckets {
		for _, kv := range bucket {
			ht.Insert(kv.Key, kv.Value)
		}
	}
}

// OptimizedHashTable represents an optimized hash table
type OptimizedHashTable struct {
	buckets [][]KeyValue
	size    int
	mask    int // Power of 2 - 1 for fast modulo
	count   int
}

// NewOptimizedHashTable creates a new optimized hash table
func NewOptimizedHashTable() *OptimizedHashTable {
	initialSize := 16 // Must be power of 2
	return &OptimizedHashTable{
		buckets: make([][]KeyValue, initialSize),
		size:    initialSize,
		mask:    initialSize - 1,
		count:   0,
	}
}

// Insert adds a key-value pair to the optimized hash table
func (ht *OptimizedHashTable) Insert(key string, value interface{}) {
	// Resize if load factor > 0.75
	if float64(ht.count)/float64(ht.size) > 0.75 {
		ht.resize()
	}

	hash := fastHash(key)
	bucket := hash & ht.mask // Fast modulo for power of 2

	// Check if key already exists
	for i, kv := range ht.buckets[bucket] {
		if kv.Key == key {
			ht.buckets[bucket][i].Value = value
			return
		}
	}

	// Add new key-value pair
	ht.buckets[bucket] = append(ht.buckets[bucket], KeyValue{Key: key, Value: value})
	ht.count++
}

// Search finds a value by key with optimizations
func (ht *OptimizedHashTable) Search(key string) (interface{}, bool) {
	hash := fastHash(key)
	bucket := hash & ht.mask // Fast modulo for power of 2

	// Linear probing with prefetching
	for _, kv := range ht.buckets[bucket] {
		if kv.Key == key {
			return kv.Value, true
		}
	}

	return nil, false
}

// resize doubles the optimized hash table size
func (ht *OptimizedHashTable) resize() {
	oldBuckets := ht.buckets
	ht.size *= 2
	ht.mask = ht.size - 1
	ht.buckets = make([][]KeyValue, ht.size)
	ht.count = 0

	// Rehash all elements
	for _, bucket := range oldBuckets {
		for _, kv := range bucket {
			ht.Insert(kv.Key, kv.Value)
		}
	}
}

// fastHash implements a fast hash function
func fastHash(key string) int {
	hash := uint32(2166136261) // FNV offset basis
	for i := 0; i < len(key); i++ {
		hash ^= uint32(key[i])
		hash *= 16777619 // FNV prime
	}
	return int(hash)
}

// RobinHoodHashTable implements Robin Hood hashing
type RobinHoodHashTable struct {
	entries []RobinHoodEntry
	size    int
	mask    int
	count   int
}

type RobinHoodEntry struct {
	Key      string
	Value    interface{}
	Distance int // Distance from ideal position
	Occupied bool
}

// NewRobinHoodHashTable creates a new Robin Hood hash table
func NewRobinHoodHashTable() *RobinHoodHashTable {
	initialSize := 16
	return &RobinHoodHashTable{
		entries: make([]RobinHoodEntry, initialSize),
		size:    initialSize,
		mask:    initialSize - 1,
		count:   0,
	}
}

// Insert adds a key-value pair using Robin Hood hashing
func (ht *RobinHoodHashTable) Insert(key string, value interface{}) {
	if float64(ht.count)/float64(ht.size) > 0.75 {
		ht.resizeRobinHood()
	}

	hash := fastHash(key)
	pos := hash & ht.mask
	distance := 0

	entry := RobinHoodEntry{
		Key:      key,
		Value:    value,
		Distance: distance,
		Occupied: true,
	}

	for {
		if !ht.entries[pos].Occupied {
			ht.entries[pos] = entry
			ht.count++
			return
		}

		if ht.entries[pos].Key == key {
			ht.entries[pos].Value = value
			return
		}

		// Robin Hood: if current entry is richer, swap
		if ht.entries[pos].Distance < entry.Distance {
			entry, ht.entries[pos] = ht.entries[pos], entry
		}

		pos = (pos + 1) & ht.mask
		entry.Distance++
	}
}

// Search finds a value using Robin Hood hashing
func (ht *RobinHoodHashTable) Search(key string) (interface{}, bool) {
	hash := fastHash(key)
	pos := hash & ht.mask
	distance := 0

	for {
		if !ht.entries[pos].Occupied {
			return nil, false
		}

		if ht.entries[pos].Key == key {
			return ht.entries[pos].Value, true
		}

		// If we've traveled further than the stored entry, key doesn't exist
		if distance > ht.entries[pos].Distance {
			return nil, false
		}

		pos = (pos + 1) & ht.mask
		distance++
	}
}

func (ht *RobinHoodHashTable) resizeRobinHood() {
	oldEntries := ht.entries
	ht.size *= 2
	ht.mask = ht.size - 1
	ht.entries = make([]RobinHoodEntry, ht.size)
	ht.count = 0

	for _, entry := range oldEntries {
		if entry.Occupied {
			ht.Insert(entry.Key, entry.Value)
		}
	}
}

// CuckooHashTable implements Cuckoo hashing
type CuckooHashTable struct {
	table1 []CuckooEntry
	table2 []CuckooEntry
	size   int
	count  int
}

type CuckooEntry struct {
	Key      string
	Value    interface{}
	Occupied bool
}

// NewCuckooHashTable creates a new Cuckoo hash table
func NewCuckooHashTable() *CuckooHashTable {
	initialSize := 16
	return &CuckooHashTable{
		table1: make([]CuckooEntry, initialSize),
		table2: make([]CuckooEntry, initialSize),
		size:   initialSize,
		count:  0,
	}
}

// Insert adds a key-value pair using Cuckoo hashing
func (ht *CuckooHashTable) Insert(key string, value interface{}) {
	if float64(ht.count)/float64(ht.size) > 0.5 {
		ht.resizeCuckoo()
	}

	entry := CuckooEntry{Key: key, Value: value, Occupied: true}
	maxIterations := int(math.Log2(float64(ht.size))) + 1

	for i := 0; i < maxIterations; i++ {
		// Try table1
		pos1 := fastHash(entry.Key) % ht.size
		if !ht.table1[pos1].Occupied {
			ht.table1[pos1] = entry
			ht.count++
			return
		}

		// Evict from table1
		entry, ht.table1[pos1] = ht.table1[pos1], entry

		// Try table2
		pos2 := fastHash2(entry.Key) % ht.size
		if !ht.table2[pos2].Occupied {
			ht.table2[pos2] = entry
			ht.count++
			return
		}

		// Evict from table2
		entry, ht.table2[pos2] = ht.table2[pos2], entry
	}

	// If we reach here, rehash is needed
	ht.resizeCuckoo()
	ht.Insert(entry.Key, entry.Value)
}

// Search finds a value using Cuckoo hashing
func (ht *CuckooHashTable) Search(key string) (interface{}, bool) {
	// Check table1
	pos1 := fastHash(key) % ht.size
	if ht.table1[pos1].Occupied && ht.table1[pos1].Key == key {
		return ht.table1[pos1].Value, true
	}

	// Check table2
	pos2 := fastHash2(key) % ht.size
	if ht.table2[pos2].Occupied && ht.table2[pos2].Key == key {
		return ht.table2[pos2].Value, true
	}

	return nil, false
}

func (ht *CuckooHashTable) resizeCuckoo() {
	oldTable1 := ht.table1
	oldTable2 := ht.table2
	ht.size *= 2
	ht.table1 = make([]CuckooEntry, ht.size)
	ht.table2 = make([]CuckooEntry, ht.size)
	ht.count = 0

	for _, entry := range oldTable1 {
		if entry.Occupied {
			ht.Insert(entry.Key, entry.Value)
		}
	}
	for _, entry := range oldTable2 {
		if entry.Occupied {
			ht.Insert(entry.Key, entry.Value)
		}
	}
}

// fastHash2 implements a second hash function for Cuckoo hashing
func fastHash2(key string) int {
	hash := uint32(2166136261)
	for i := len(key) - 1; i >= 0; i-- {
		hash ^= uint32(key[i])
		hash *= 16777619
	}
	return int(hash)
}