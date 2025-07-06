package data

import (
	"encoding/json"
	"fmt"
	"math"
	"math/rand"
	"sort"
	"strconv"
	"strings"
	"time"
)

// DataStructure represents different types of data structures for performance testing
type DataStructure interface {
	Insert(key string, value interface{}) error
	Get(key string) (interface{}, bool)
	Delete(key string) bool
	Size() int
	Iterate(func(key string, value interface{}) bool)
	Clear()
	GetType() string
}

// HashMap implementation
type HashMap struct {
	data map[string]interface{}
	size int
}

func NewHashMap() *HashMap {
	return &HashMap{
		data: make(map[string]interface{}),
		size: 0,
	}
}

func (hm *HashMap) Insert(key string, value interface{}) error {
	if _, exists := hm.data[key]; !exists {
		hm.size++
	}
	hm.data[key] = value
	return nil
}

func (hm *HashMap) Get(key string) (interface{}, bool) {
	value, exists := hm.data[key]
	return value, exists
}

func (hm *HashMap) Delete(key string) bool {
	if _, exists := hm.data[key]; exists {
		delete(hm.data, key)
		hm.size--
		return true
	}
	return false
}

func (hm *HashMap) Size() int {
	return hm.size
}

func (hm *HashMap) Iterate(fn func(key string, value interface{}) bool) {
	for key, value := range hm.data {
		if !fn(key, value) {
			break
		}
	}
}

func (hm *HashMap) Clear() {
	hm.data = make(map[string]interface{})
	hm.size = 0
}

func (hm *HashMap) GetType() string {
	return "HashMap"
}

// BinarySearchTree implementation
type BSTNode struct {
	Key   string
	Value interface{}
	Left  *BSTNode
	Right *BSTNode
}

type BinarySearchTree struct {
	root *BSTNode
	size int
}

func NewBinarySearchTree() *BinarySearchTree {
	return &BinarySearchTree{
		root: nil,
		size: 0,
	}
}

func (bst *BinarySearchTree) Insert(key string, value interface{}) error {
	if bst.root == nil {
		bst.root = &BSTNode{Key: key, Value: value}
		bst.size++
		return nil
	}

	inserted := bst.insertNode(bst.root, key, value)
	if inserted {
		bst.size++
	}
	return nil
}

func (bst *BinarySearchTree) insertNode(node *BSTNode, key string, value interface{}) bool {
	if key < node.Key {
		if node.Left == nil {
			node.Left = &BSTNode{Key: key, Value: value}
			return true
		}
		return bst.insertNode(node.Left, key, value)
	} else if key > node.Key {
		if node.Right == nil {
			node.Right = &BSTNode{Key: key, Value: value}
			return true
		}
		return bst.insertNode(node.Right, key, value)
	} else {
		// Key already exists, update value
		node.Value = value
		return false
	}
}

func (bst *BinarySearchTree) Get(key string) (interface{}, bool) {
	node := bst.findNode(bst.root, key)
	if node != nil {
		return node.Value, true
	}
	return nil, false
}

func (bst *BinarySearchTree) findNode(node *BSTNode, key string) *BSTNode {
	if node == nil {
		return nil
	}
	if key == node.Key {
		return node
	} else if key < node.Key {
		return bst.findNode(node.Left, key)
	} else {
		return bst.findNode(node.Right, key)
	}
}

func (bst *BinarySearchTree) Delete(key string) bool {
	if bst.root == nil {
		return false
	}

	newRoot, deleted := bst.deleteNode(bst.root, key)
	bst.root = newRoot
	if deleted {
		bst.size--
	}
	return deleted
}

func (bst *BinarySearchTree) deleteNode(node *BSTNode, key string) (*BSTNode, bool) {
	if node == nil {
		return nil, false
	}

	if key < node.Key {
		newLeft, deleted := bst.deleteNode(node.Left, key)
		node.Left = newLeft
		return node, deleted
	} else if key > node.Key {
		newRight, deleted := bst.deleteNode(node.Right, key)
		node.Right = newRight
		return node, deleted
	} else {
		// Found the node to delete
		if node.Left == nil {
			return node.Right, true
		} else if node.Right == nil {
			return node.Left, true
		} else {
			// Node has two children
			successor := bst.findMin(node.Right)
			node.Key = successor.Key
			node.Value = successor.Value
			newRight, _ := bst.deleteNode(node.Right, successor.Key)
			node.Right = newRight
			return node, true
		}
	}
}

func (bst *BinarySearchTree) findMin(node *BSTNode) *BSTNode {
	for node.Left != nil {
		node = node.Left
	}
	return node
}

func (bst *BinarySearchTree) Size() int {
	return bst.size
}

func (bst *BinarySearchTree) Iterate(fn func(key string, value interface{}) bool) {
	bst.inorderTraversal(bst.root, fn)
}

func (bst *BinarySearchTree) inorderTraversal(node *BSTNode, fn func(key string, value interface{}) bool) bool {
	if node == nil {
		return true
	}

	if !bst.inorderTraversal(node.Left, fn) {
		return false
	}

	if !fn(node.Key, node.Value) {
		return false
	}

	return bst.inorderTraversal(node.Right, fn)
}

func (bst *BinarySearchTree) Clear() {
	bst.root = nil
	bst.size = 0
}

func (bst *BinarySearchTree) GetType() string {
	return "BinarySearchTree"
}

// LinkedList implementation
type ListNode struct {
	Key   string
	Value interface{}
	Next  *ListNode
}

type LinkedList struct {
	head *ListNode
	size int
}

func NewLinkedList() *LinkedList {
	return &LinkedList{
		head: nil,
		size: 0,
	}
}

func (ll *LinkedList) Insert(key string, value interface{}) error {
	// Check if key already exists
	current := ll.head
	for current != nil {
		if current.Key == key {
			current.Value = value
			return nil
		}
		current = current.Next
	}

	// Insert at the beginning
	newNode := &ListNode{Key: key, Value: value, Next: ll.head}
	ll.head = newNode
	ll.size++
	return nil
}

func (ll *LinkedList) Get(key string) (interface{}, bool) {
	current := ll.head
	for current != nil {
		if current.Key == key {
			return current.Value, true
		}
		current = current.Next
	}
	return nil, false
}

func (ll *LinkedList) Delete(key string) bool {
	if ll.head == nil {
		return false
	}

	if ll.head.Key == key {
		ll.head = ll.head.Next
		ll.size--
		return true
	}

	current := ll.head
	for current.Next != nil {
		if current.Next.Key == key {
			current.Next = current.Next.Next
			ll.size--
			return true
		}
		current = current.Next
	}
	return false
}

func (ll *LinkedList) Size() int {
	return ll.size
}

func (ll *LinkedList) Iterate(fn func(key string, value interface{}) bool) {
	current := ll.head
	for current != nil {
		if !fn(current.Key, current.Value) {
			break
		}
		current = current.Next
	}
}

func (ll *LinkedList) Clear() {
	ll.head = nil
	ll.size = 0
}

func (ll *LinkedList) GetType() string {
	return "LinkedList"
}

// Array-based structure for comparison
type ArrayStructure struct {
	keys   []string
	values []interface{}
	size   int
}

func NewArrayStructure() *ArrayStructure {
	return &ArrayStructure{
		keys:   make([]string, 0),
		values: make([]interface{}, 0),
		size:   0,
	}
}

func (as *ArrayStructure) Insert(key string, value interface{}) error {
	// Check if key already exists
	for i, k := range as.keys {
		if k == key {
			as.values[i] = value
			return nil
		}
	}

	// Add new key-value pair
	as.keys = append(as.keys, key)
	as.values = append(as.values, value)
	as.size++
	return nil
}

func (as *ArrayStructure) Get(key string) (interface{}, bool) {
	for i, k := range as.keys {
		if k == key {
			return as.values[i], true
		}
	}
	return nil, false
}

func (as *ArrayStructure) Delete(key string) bool {
	for i, k := range as.keys {
		if k == key {
			// Remove by shifting elements
			as.keys = append(as.keys[:i], as.keys[i+1:]...)
			as.values = append(as.values[:i], as.values[i+1:]...)
			as.size--
			return true
		}
	}
	return false
}

func (as *ArrayStructure) Size() int {
	return as.size
}

func (as *ArrayStructure) Iterate(fn func(key string, value interface{}) bool) {
	for i := 0; i < len(as.keys); i++ {
		if !fn(as.keys[i], as.values[i]) {
			break
		}
	}
}

func (as *ArrayStructure) Clear() {
	as.keys = make([]string, 0)
	as.values = make([]interface{}, 0)
	as.size = 0
}

func (as *ArrayStructure) GetType() string {
	return "ArrayStructure"
}

// DataGenerator generates test data for performance testing
type DataGenerator struct {
	rand *rand.Rand
}

func NewDataGenerator() *DataGenerator {
	return &DataGenerator{
		rand: rand.New(rand.NewSource(time.Now().UnixNano())),
	}
}

// GenerateKeys generates a slice of random keys
func (dg *DataGenerator) GenerateKeys(count int, keyType string) []string {
	keys := make([]string, count)

	switch keyType {
	case "sequential":
		for i := 0; i < count; i++ {
			keys[i] = fmt.Sprintf("key_%06d", i)
		}
	case "random":
		for i := 0; i < count; i++ {
			keys[i] = dg.generateRandomString(10)
		}
	case "uuid":
		for i := 0; i < count; i++ {
			keys[i] = dg.generateUUID()
		}
	case "numeric":
		for i := 0; i < count; i++ {
			keys[i] = strconv.Itoa(dg.rand.Intn(count * 10))
		}
	case "mixed":
		for i := 0; i < count; i++ {
			switch dg.rand.Intn(4) {
			case 0:
				keys[i] = fmt.Sprintf("seq_%d", i)
			case 1:
				keys[i] = dg.generateRandomString(8)
			case 2:
				keys[i] = strconv.Itoa(dg.rand.Intn(1000))
			default:
				keys[i] = dg.generateUUID()[:8]
			}
		}
	default:
		// Default to sequential
		for i := 0; i < count; i++ {
			keys[i] = fmt.Sprintf("key_%d", i)
		}
	}

	return keys
}

// GenerateValues generates test values of different types
func (dg *DataGenerator) GenerateValues(count int, valueType string) []interface{} {
	values := make([]interface{}, count)

	switch valueType {
	case "integer":
		for i := 0; i < count; i++ {
			values[i] = dg.rand.Intn(10000)
		}
	case "float":
		for i := 0; i < count; i++ {
			values[i] = dg.rand.Float64() * 1000
		}
	case "string":
		for i := 0; i < count; i++ {
			values[i] = dg.generateRandomString(20)
		}
	case "json":
		for i := 0; i < count; i++ {
			values[i] = dg.generateJSONObject()
		}
	case "struct":
		for i := 0; i < count; i++ {
			values[i] = dg.generateTestStruct()
		}
	case "mixed":
		for i := 0; i < count; i++ {
			switch dg.rand.Intn(5) {
			case 0:
				values[i] = dg.rand.Intn(1000)
			case 1:
				values[i] = dg.rand.Float64() * 100
			case 2:
				values[i] = dg.generateRandomString(15)
			case 3:
				values[i] = dg.generateJSONObject()
			default:
				values[i] = dg.generateTestStruct()
			}
		}
	default:
		// Default to integers
		for i := 0; i < count; i++ {
			values[i] = i
		}
	}

	return values
}

// generateRandomString creates a random string of specified length
func (dg *DataGenerator) generateRandomString(length int) string {
	const charset = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
	b := make([]byte, length)
	for i := range b {
		b[i] = charset[dg.rand.Intn(len(charset))]
	}
	return string(b)
}

// generateUUID creates a simple UUID-like string
func (dg *DataGenerator) generateUUID() string {
	return fmt.Sprintf("%08x-%04x-%04x-%04x-%012x",
		dg.rand.Uint32(),
		dg.rand.Uint32()&0xffff,
		dg.rand.Uint32()&0xffff,
		dg.rand.Uint32()&0xffff,
		dg.rand.Uint64()&0xffffffffffff)
}

// generateJSONObject creates a JSON-serializable object
func (dg *DataGenerator) generateJSONObject() map[string]interface{} {
	return map[string]interface{}{
		"id":          dg.rand.Intn(10000),
		"name":        dg.generateRandomString(10),
		"value":       dg.rand.Float64() * 100,
		"active":      dg.rand.Intn(2) == 1,
		"timestamp":   time.Now().Unix(),
		"tags":        dg.generateRandomTags(),
		"metadata":    dg.generateMetadata(),
		"coordinates": []float64{dg.rand.Float64() * 180, dg.rand.Float64() * 90},
	}
}

// generateTestStruct creates a test struct
func (dg *DataGenerator) generateTestStruct() TestStruct {
	return TestStruct{
		ID:          dg.rand.Intn(10000),
		Name:        dg.generateRandomString(15),
		Score:       dg.rand.Float64() * 100,
		Active:      dg.rand.Intn(2) == 1,
		CreatedAt:   time.Now(),
		Tags:        dg.generateRandomTags(),
		Attributes:  dg.generateAttributes(),
		NestedData:  dg.generateNestedData(),
	}
}

// generateRandomTags creates random tags
func (dg *DataGenerator) generateRandomTags() []string {
	tagCount := dg.rand.Intn(5) + 1
	tags := make([]string, tagCount)
	possibleTags := []string{"urgent", "important", "low-priority", "archived", "active", "pending", "completed", "draft"}

	for i := 0; i < tagCount; i++ {
		tags[i] = possibleTags[dg.rand.Intn(len(possibleTags))]
	}

	return tags
}

// generateMetadata creates metadata map
func (dg *DataGenerator) generateMetadata() map[string]string {
	return map[string]string{
		"source":    fmt.Sprintf("system_%d", dg.rand.Intn(10)),
		"version":   fmt.Sprintf("v%d.%d.%d", dg.rand.Intn(3)+1, dg.rand.Intn(10), dg.rand.Intn(10)),
		"region":    []string{"us-east", "us-west", "eu-central", "ap-south"}[dg.rand.Intn(4)],
		"env":       []string{"dev", "staging", "prod"}[dg.rand.Intn(3)],
		"checksum":  dg.generateRandomString(32),
	}
}

// generateAttributes creates attributes map
func (dg *DataGenerator) generateAttributes() map[string]interface{} {
	return map[string]interface{}{
		"priority":    dg.rand.Intn(10) + 1,
		"category":    []string{"A", "B", "C", "D"}[dg.rand.Intn(4)],
		"weight":      dg.rand.Float64() * 10,
		"enabled":     dg.rand.Intn(2) == 1,
		"max_retries": dg.rand.Intn(5) + 1,
		"timeout":     time.Duration(dg.rand.Intn(30)+1) * time.Second,
	}
}

// generateNestedData creates nested data structure
func (dg *DataGenerator) generateNestedData() NestedData {
	return NestedData{
		Level1: Level1Data{
			Value: dg.generateRandomString(10),
			Count: dg.rand.Intn(100),
			Level2: Level2Data{
				Items: dg.generateItems(),
				Sum:   dg.rand.Float64() * 1000,
				Level3: Level3Data{
					Data:      dg.generateRandomString(20),
					Timestamp: time.Now(),
					Flags:     dg.generateFlags(),
				},
			},
		},
		Alternate: dg.generateAlternateData(),
	}
}

// generateItems creates random items
func (dg *DataGenerator) generateItems() []string {
	count := dg.rand.Intn(10) + 1
	items := make([]string, count)
	for i := 0; i < count; i++ {
		items[i] = fmt.Sprintf("item_%s", dg.generateRandomString(5))
	}
	return items
}

// generateFlags creates random flags
func (dg *DataGenerator) generateFlags() map[string]bool {
	return map[string]bool{
		"feature_a": dg.rand.Intn(2) == 1,
		"feature_b": dg.rand.Intn(2) == 1,
		"feature_c": dg.rand.Intn(2) == 1,
		"debug":     dg.rand.Intn(2) == 1,
		"verbose":   dg.rand.Intn(2) == 1,
	}
}

// generateAlternateData creates alternate data
func (dg *DataGenerator) generateAlternateData() []AlternateItem {
	count := dg.rand.Intn(5) + 1
	items := make([]AlternateItem, count)
	for i := 0; i < count; i++ {
		items[i] = AlternateItem{
			Key:    dg.generateRandomString(8),
			Value:  dg.rand.Intn(1000),
			Weight: dg.rand.Float64(),
		}
	}
	return items
}

// Data structure definitions
type TestStruct struct {
	ID         int                    `json:"id"`
	Name       string                 `json:"name"`
	Score      float64                `json:"score"`
	Active     bool                   `json:"active"`
	CreatedAt  time.Time              `json:"created_at"`
	Tags       []string               `json:"tags"`
	Attributes map[string]interface{} `json:"attributes"`
	NestedData NestedData             `json:"nested_data"`
}

type NestedData struct {
	Level1    Level1Data      `json:"level1"`
	Alternate []AlternateItem `json:"alternate"`
}

type Level1Data struct {
	Value  string     `json:"value"`
	Count  int        `json:"count"`
	Level2 Level2Data `json:"level2"`
}

type Level2Data struct {
	Items  []string   `json:"items"`
	Sum    float64    `json:"sum"`
	Level3 Level3Data `json:"level3"`
}

type Level3Data struct {
	Data      string            `json:"data"`
	Timestamp time.Time         `json:"timestamp"`
	Flags     map[string]bool   `json:"flags"`
}

type AlternateItem struct {
	Key    string  `json:"key"`
	Value  int     `json:"value"`
	Weight float64 `json:"weight"`
}

// Performance testing utilities

// BenchmarkDataStructure performs benchmark tests on data structures
func BenchmarkDataStructure(ds DataStructure, keys []string, values []interface{}, operations []string) map[string]time.Duration {
	results := make(map[string]time.Duration)

	// Benchmark insertions
	start := time.Now()
	for i, key := range keys {
		ds.Insert(key, values[i%len(values)])
	}
	results["insert"] = time.Since(start)

	// Benchmark lookups
	start = time.Now()
	for _, key := range keys {
		ds.Get(key)
	}
	results["lookup"] = time.Since(start)

	// Benchmark iteration
	start = time.Now()
	count := 0
	ds.Iterate(func(key string, value interface{}) bool {
		count++
		return true
	})
	results["iterate"] = time.Since(start)

	// Benchmark deletions
	start = time.Now()
	for i, key := range keys {
		if i%2 == 0 { // Delete every other key
			ds.Delete(key)
		}
	}
	results["delete"] = time.Since(start)

	return results
}

// GenerateWorkload creates a realistic workload for testing
func (dg *DataGenerator) GenerateWorkload(size int, workloadType string) ([]string, []interface{}, []string) {
	var keys []string
	var values []interface{}
	var operations []string

	switch workloadType {
	case "read_heavy":
		keys = dg.GenerateKeys(size, "sequential")
		values = dg.GenerateValues(size, "integer")
		operations = dg.generateOperations(size*5, []string{"get", "get", "get", "get", "insert"})

	case "write_heavy":
		keys = dg.GenerateKeys(size, "random")
		values = dg.GenerateValues(size, "struct")
		operations = dg.generateOperations(size*3, []string{"insert", "insert", "insert", "get", "delete"})

	case "mixed":
		keys = dg.GenerateKeys(size, "mixed")
		values = dg.GenerateValues(size, "mixed")
		operations = dg.generateOperations(size*2, []string{"insert", "get", "delete", "get", "insert"})

	case "cache_simulation":
		keys = dg.GenerateKeys(size/10, "sequential") // Smaller key space for cache hits
		values = dg.GenerateValues(size, "json")
		operations = dg.generateOperations(size*10, []string{"get", "get", "get", "get", "get", "get", "insert", "delete"})

	case "database_simulation":
		keys = dg.GenerateKeys(size, "uuid")
		values = dg.GenerateValues(size, "struct")
		operations = dg.generateOperations(size*4, []string{"insert", "get", "get", "get", "insert", "get", "delete"})

	default:
		keys = dg.GenerateKeys(size, "sequential")
		values = dg.GenerateValues(size, "integer")
		operations = dg.generateOperations(size*2, []string{"insert", "get", "delete"})
	}

	return keys, values, operations
}

// generateOperations creates a sequence of operations
func (dg *DataGenerator) generateOperations(count int, opTypes []string) []string {
	operations := make([]string, count)
	for i := 0; i < count; i++ {
		operations[i] = opTypes[dg.rand.Intn(len(opTypes))]
	}
	return operations
}

// ExecuteWorkload executes a workload against a data structure
func ExecuteWorkload(ds DataStructure, keys []string, values []interface{}, operations []string) WorkloadResult {
	result := WorkloadResult{
		OperationCounts: make(map[string]int),
		OperationTimes:  make(map[string]time.Duration),
		StartTime:       time.Now(),
	}

	keyIndex := 0
	valueIndex := 0

	for _, op := range operations {
		start := time.Now()
		result.OperationCounts[op]++

		switch op {
		case "insert":
			key := keys[keyIndex%len(keys)]
			value := values[valueIndex%len(values)]
			ds.Insert(key, value)
			keyIndex++
			valueIndex++

		case "get":
			key := keys[keyIndex%len(keys)]
			ds.Get(key)
			keyIndex++

		case "delete":
			key := keys[keyIndex%len(keys)]
			ds.Delete(key)
			keyIndex++

		case "iterate":
			count := 0
			ds.Iterate(func(key string, value interface{}) bool {
				count++
				return count < 100 // Limit iteration for performance
			})
		}

		duration := time.Since(start)
		result.OperationTimes[op] += duration
		result.TotalOperations++
	}

	result.EndTime = time.Now()
	result.TotalDuration = result.EndTime.Sub(result.StartTime)
	result.FinalSize = ds.Size()

	return result
}

// WorkloadResult contains the results of workload execution
type WorkloadResult struct {
	OperationCounts map[string]int
	OperationTimes  map[string]time.Duration
	TotalOperations int
	TotalDuration   time.Duration
	StartTime       time.Time
	EndTime         time.Time
	FinalSize       int
}

// String returns a string representation of the workload result
func (wr WorkloadResult) String() string {
	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("Workload Results:\n"))
	sb.WriteString(fmt.Sprintf("  Total Operations: %d\n", wr.TotalOperations))
	sb.WriteString(fmt.Sprintf("  Total Duration: %v\n", wr.TotalDuration))
	sb.WriteString(fmt.Sprintf("  Final Size: %d\n", wr.FinalSize))
	sb.WriteString(fmt.Sprintf("  Operations per second: %.2f\n", float64(wr.TotalOperations)/wr.TotalDuration.Seconds()))

	sb.WriteString("\n  Operation Breakdown:\n")
	for op, count := range wr.OperationCounts {
		avgTime := wr.OperationTimes[op] / time.Duration(count)
		sb.WriteString(fmt.Sprintf("    %s: %d operations, avg time: %v\n", op, count, avgTime))
	}

	return sb.String()
}

// CompareDataStructures compares the performance of different data structures
func CompareDataStructures(structures []DataStructure, workloadSize int, workloadType string) map[string]WorkloadResult {
	dg := NewDataGenerator()
	keys, values, operations := dg.GenerateWorkload(workloadSize, workloadType)

	results := make(map[string]WorkloadResult)

	for _, ds := range structures {
		// Clear the data structure
		ds.Clear()

		// Execute workload
		result := ExecuteWorkload(ds, keys, values, operations)
		results[ds.GetType()] = result
	}

	return results
}

// MemoryIntensiveOperations performs memory-intensive operations for profiling
func MemoryIntensiveOperations(size int) {
	dg := NewDataGenerator()

	// Create large data structures
	hashMap := NewHashMap()
	bst := NewBinarySearchTree()
	linkedList := NewLinkedList()
	arrayStruct := NewArrayStructure()

	// Generate large amounts of data
	keys := dg.GenerateKeys(size, "mixed")
	values := dg.GenerateValues(size, "struct")

	// Populate all structures
	for i, key := range keys {
		value := values[i%len(values)]
		hashMap.Insert(key, value)
		bst.Insert(key, value)
		linkedList.Insert(key, value)
		arrayStruct.Insert(key, value)
	}

	// Perform intensive operations
	for i := 0; i < size/10; i++ {
		key := keys[i%len(keys)]

		// Multiple lookups
		hashMap.Get(key)
		bst.Get(key)
		linkedList.Get(key)
		arrayStruct.Get(key)

		// Iterations
		if i%100 == 0 {
			count := 0
			hashMap.Iterate(func(k string, v interface{}) bool {
				count++
				return count < 50
			})
		}
	}

	// Create additional memory pressure
	largeSlices := make([][]byte, 100)
	for i := range largeSlices {
		largeSlices[i] = make([]byte, 1024*1024) // 1MB each
		for j := range largeSlices[i] {
			largeSlices[i][j] = byte(i + j)
		}
	}

	// JSON marshaling/unmarshaling
	for i := 0; i < size/100; i++ {
		value := values[i%len(values)]
		if data, err := json.Marshal(value); err == nil {
			var unmarshaled interface{}
			json.Unmarshal(data, &unmarshaled)
		}
	}

	// String operations
	var largeString strings.Builder
	for i := 0; i < size/10; i++ {
		largeString.WriteString(dg.generateRandomString(100))
		if i%1000 == 0 {
			// Force string operations
			str := largeString.String()
			strings.ToUpper(str[:len(str)/2])
			strings.Split(str, "a")
		}
	}
}

// CPUIntensiveOperations performs CPU-intensive operations for profiling
func CPUIntensiveOperations(iterations int) {
	dg := NewDataGenerator()

	// Mathematical computations
	for i := 0; i < iterations; i++ {
		// Complex mathematical operations
		x := float64(i)
		result := math.Sin(x) * math.Cos(x) * math.Tan(x)
		result = math.Pow(result, 2) + math.Sqrt(math.Abs(result))
		result = math.Log(math.Abs(result)+1) * math.Exp(result/1000)

		// Prime number calculations
		if i%100 == 0 {
			calculatePrimesUpTo(1000)
		}

		// Sorting operations
		if i%50 == 0 {
			data := make([]int, 1000)
			for j := range data {
				data[j] = dg.rand.Intn(10000)
			}
			sort.Ints(data)
		}

		// String processing
		if i%25 == 0 {
			str := dg.generateRandomString(1000)
			processString(str)
		}
	}
}

// calculatePrimesUpTo calculates all prime numbers up to n
func calculatePrimesUpTo(n int) []int {
	if n < 2 {
		return []int{}
	}

	sieve := make([]bool, n+1)
	for i := 2; i <= n; i++ {
		sieve[i] = true
	}

	for i := 2; i*i <= n; i++ {
		if sieve[i] {
			for j := i * i; j <= n; j += i {
				sieve[j] = false
			}
		}
	}

	primes := make([]int, 0)
	for i := 2; i <= n; i++ {
		if sieve[i] {
			primes = append(primes, i)
		}
	}

	return primes
}

// processString performs various string processing operations
func processString(s string) {
	// Convert to uppercase and lowercase
	upper := strings.ToUpper(s)
	lower := strings.ToLower(upper)

	// Split and join operations
	words := strings.Split(lower, "")
	joined := strings.Join(words, "-")

	// Replace operations
	replaced := strings.ReplaceAll(joined, "a", "@")
	replaced = strings.ReplaceAll(replaced, "e", "3")
	replaced = strings.ReplaceAll(replaced, "i", "1")

	// Substring operations
	if len(replaced) > 10 {
		substr := replaced[5 : len(replaced)-5]
		_ = substr
	}

	// Character counting
	charCount := make(map[rune]int)
	for _, char := range s {
		charCount[char]++
	}
}