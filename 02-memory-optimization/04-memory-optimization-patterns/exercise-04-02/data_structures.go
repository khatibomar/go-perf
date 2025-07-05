package main

import (
	"fmt"
	"unsafe"
)

// Array of Structs (AoS) - traditional approach
type PersonAoS struct {
	ID       uint32
	Age      uint8
	Salary   uint32
	Active   bool
	Name     string
	Email    string
}

type PersonCollectionAoS struct {
	Persons []PersonAoS
}

func NewPersonCollectionAoS(capacity int) *PersonCollectionAoS {
	return &PersonCollectionAoS{
		Persons: make([]PersonAoS, 0, capacity),
	}
}

func (pc *PersonCollectionAoS) Add(id uint32, age uint8, salary uint32, active bool, name, email string) {
	pc.Persons = append(pc.Persons, PersonAoS{
		ID:     id,
		Age:    age,
		Salary: salary,
		Active: active,
		Name:   name,
		Email:  email,
	})
}

func (pc *PersonCollectionAoS) GetTotalSalary() uint64 {
	var total uint64
	for _, person := range pc.Persons {
		if person.Active {
			total += uint64(person.Salary)
		}
	}
	return total
}

func (pc *PersonCollectionAoS) GetAverageAge() float64 {
	if len(pc.Persons) == 0 {
		return 0
	}
	var total uint32
	for _, person := range pc.Persons {
		total += uint32(person.Age)
	}
	return float64(total) / float64(len(pc.Persons))
}

// Struct of Arrays (SoA) - cache-friendly approach
type PersonCollectionSoA struct {
	IDs     []uint32
	Ages    []uint8
	Salaries []uint32
	Active  []bool
	Names   []string
	Emails  []string
	count   int
}

func NewPersonCollectionSoA(capacity int) *PersonCollectionSoA {
	return &PersonCollectionSoA{
		IDs:      make([]uint32, 0, capacity),
		Ages:     make([]uint8, 0, capacity),
		Salaries: make([]uint32, 0, capacity),
		Active:   make([]bool, 0, capacity),
		Names:    make([]string, 0, capacity),
		Emails:   make([]string, 0, capacity),
	}
}

func (pc *PersonCollectionSoA) Add(id uint32, age uint8, salary uint32, active bool, name, email string) {
	pc.IDs = append(pc.IDs, id)
	pc.Ages = append(pc.Ages, age)
	pc.Salaries = append(pc.Salaries, salary)
	pc.Active = append(pc.Active, active)
	pc.Names = append(pc.Names, name)
	pc.Emails = append(pc.Emails, email)
	pc.count++
}

func (pc *PersonCollectionSoA) GetTotalSalary() uint64 {
	var total uint64
	for i := 0; i < pc.count; i++ {
		if pc.Active[i] {
			total += uint64(pc.Salaries[i])
		}
	}
	return total
}

func (pc *PersonCollectionSoA) GetAverageAge() float64 {
	if pc.count == 0 {
		return 0
	}
	var total uint32
	for i := 0; i < pc.count; i++ {
		total += uint32(pc.Ages[i])
	}
	return float64(total) / float64(pc.count)
}

// Optimized struct with better field ordering
type PersonOptimized struct {
	// 8-byte aligned fields first
	Name   string // 16 bytes (pointer + length)
	Email  string // 16 bytes (pointer + length)
	
	// 4-byte aligned fields
	ID     uint32 // 4 bytes
	Salary uint32 // 4 bytes
	
	// 1-byte fields grouped together
	Age    uint8  // 1 byte
	Active bool   // 1 byte
	// 6 bytes padding to next 8-byte boundary
}

// Bit-packed struct for flags and small values
type PersonCompact struct {
	ID     uint32
	Salary uint32
	Name   string
	Email  string
	
	// Pack age (7 bits, max 127) and active flag (1 bit) into single byte
	AgeAndFlags uint8 // bits 0-6: age, bit 7: active
}

func NewPersonCompact(id uint32, age uint8, salary uint32, active bool, name, email string) PersonCompact {
	ageAndFlags := age & 0x7F // Mask to 7 bits
	if active {
		ageAndFlags |= 0x80 // Set bit 7
	}
	return PersonCompact{
		ID:          id,
		Salary:      salary,
		Name:        name,
		Email:       email,
		AgeAndFlags: ageAndFlags,
	}
}

func (p *PersonCompact) GetAge() uint8 {
	return p.AgeAndFlags & 0x7F
}

func (p *PersonCompact) IsActive() bool {
	return (p.AgeAndFlags & 0x80) != 0
}

// Cache-friendly hash table with open addressing
type HashTable struct {
	buckets []HashEntry
	size    int
	count   int
}

type HashEntry struct {
	key   uint32
	value uint32
	empty bool
}

func NewHashTable(size int) *HashTable {
	// Ensure size is power of 2 for efficient modulo
	if size&(size-1) != 0 {
		size = nextPowerOf2(size)
	}
	buckets := make([]HashEntry, size)
	for i := range buckets {
		buckets[i].empty = true
	}
	return &HashTable{
		buckets: buckets,
		size:    size,
	}
}

func nextPowerOf2(n int) int {
	n--
	n |= n >> 1
	n |= n >> 2
	n |= n >> 4
	n |= n >> 8
	n |= n >> 16
	n++
	return n
}

func (ht *HashTable) hash(key uint32) int {
	// Simple hash function
	return int(key) & (ht.size - 1)
}

func (ht *HashTable) Put(key, value uint32) bool {
	if ht.count >= ht.size*3/4 { // Load factor > 0.75
		return false // Table full
	}
	
	index := ht.hash(key)
	for {
		if ht.buckets[index].empty {
			ht.buckets[index] = HashEntry{key: key, value: value, empty: false}
			ht.count++
			return true
		}
		if ht.buckets[index].key == key {
			ht.buckets[index].value = value
			return true
		}
		index = (index + 1) & (ht.size - 1) // Linear probing
	}
}

func (ht *HashTable) Get(key uint32) (uint32, bool) {
	index := ht.hash(key)
	for {
		if ht.buckets[index].empty {
			return 0, false
		}
		if ht.buckets[index].key == key {
			return ht.buckets[index].value, true
		}
		index = (index + 1) & (ht.size - 1)
	}
}

// Compact binary tree with array-based storage
type CompactBinaryTree struct {
	nodes []TreeNode
	size  int
}

type TreeNode struct {
	value uint32
	used  bool
}

func NewCompactBinaryTree(capacity int) *CompactBinaryTree {
	return &CompactBinaryTree{
		nodes: make([]TreeNode, capacity),
	}
}

func (cbt *CompactBinaryTree) Insert(value uint32) bool {
	if cbt.size >= len(cbt.nodes) {
		return false
	}
	
	if cbt.size == 0 {
		cbt.nodes[0] = TreeNode{value: value, used: true}
		cbt.size++
		return true
	}
	
	index := 0
	for {
		if !cbt.nodes[index].used {
			cbt.nodes[index] = TreeNode{value: value, used: true}
			cbt.size++
			return true
		}
		
		if value < cbt.nodes[index].value {
			leftChild := 2*index + 1
			if leftChild >= len(cbt.nodes) {
				return false
			}
			index = leftChild
		} else {
			rightChild := 2*index + 2
			if rightChild >= len(cbt.nodes) {
				return false
			}
			index = rightChild
		}
	}
}

func (cbt *CompactBinaryTree) Search(value uint32) bool {
	if cbt.size == 0 {
		return false
	}
	
	index := 0
	for index < len(cbt.nodes) && cbt.nodes[index].used {
		if cbt.nodes[index].value == value {
			return true
		}
		
		if value < cbt.nodes[index].value {
			index = 2*index + 1
		} else {
			index = 2*index + 2
		}
	}
	return false
}

// Memory usage analysis functions
func GetStructSize(v interface{}) uintptr {
	return unsafe.Sizeof(v)
}

func AnalyzeMemoryLayout() {
	fmt.Printf("PersonAoS size: %d bytes\n", unsafe.Sizeof(PersonAoS{}))
	fmt.Printf("PersonOptimized size: %d bytes\n", unsafe.Sizeof(PersonOptimized{}))
	fmt.Printf("PersonCompact size: %d bytes\n", unsafe.Sizeof(PersonCompact{}))
	fmt.Printf("HashEntry size: %d bytes\n", unsafe.Sizeof(HashEntry{}))
	fmt.Printf("TreeNode size: %d bytes\n", unsafe.Sizeof(TreeNode{}))
}

// Sink variables to prevent compiler optimizations
var (
	SinkUint64 uint64
	SinkFloat64 float64
	SinkBool bool
	SinkUint32 uint32
)