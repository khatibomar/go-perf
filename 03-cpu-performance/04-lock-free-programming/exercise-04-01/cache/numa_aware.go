package cache

import (
	"runtime"
	"sync/atomic"
	"unsafe"
)

// NUMACounter provides NUMA-aware atomic operations
type NUMACounter struct {
	counters []int64
	numNodes int
}

func NewNUMACounter() *NUMACounter {
	// Use number of CPUs as approximation for NUMA nodes
	numNodes := runtime.NumCPU()
	if numNodes > 8 {
		numNodes = 8 // Reasonable upper bound
	}
	
	// Allocate one counter per NUMA node with padding
	counters := make([]int64, numNodes*8) // 8 int64s per cache line
	
	return &NUMACounter{
		counters: counters,
		numNodes: numNodes,
	}
}

// getNodeIndex returns the NUMA node index for current goroutine
func (n *NUMACounter) getNodeIndex() int {
	// Simple hashing based on goroutine ID (approximation)
	// In real implementation, you'd use actual NUMA topology
	return int(getGoroutineID()) % n.numNodes
}

// Increment increments the counter for the current NUMA node
func (n *NUMACounter) Increment() {
	nodeIndex := n.getNodeIndex()
	atomic.AddInt64(&n.counters[nodeIndex*8], 1)
}

// Add adds delta to the counter for the current NUMA node
func (n *NUMACounter) Add(delta int64) {
	nodeIndex := n.getNodeIndex()
	atomic.AddInt64(&n.counters[nodeIndex*8], delta)
}

// Load returns the total count across all NUMA nodes
func (n *NUMACounter) Load() int64 {
	var total int64
	for i := 0; i < n.numNodes; i++ {
		total += atomic.LoadInt64(&n.counters[i*8])
	}
	return total
}

// LoadNode returns the count for a specific NUMA node
func (n *NUMACounter) LoadNode(nodeIndex int) int64 {
	if nodeIndex >= 0 && nodeIndex < n.numNodes {
		return atomic.LoadInt64(&n.counters[nodeIndex*8])
	}
	return 0
}

// Reset resets all counters
func (n *NUMACounter) Reset() {
	for i := 0; i < n.numNodes; i++ {
		atomic.StoreInt64(&n.counters[i*8], 0)
	}
}

// GetNumNodes returns the number of NUMA nodes
func (n *NUMACounter) GetNumNodes() int {
	return n.numNodes
}

// NUMALocalData provides NUMA-local data structures
type NUMALocalData struct {
	data     [][]int64
	numNodes int
}

func NewNUMALocalData(size int) *NUMALocalData {
	numNodes := runtime.NumCPU()
	if numNodes > 8 {
		numNodes = 8
	}
	
	// Allocate separate data arrays for each NUMA node
	data := make([][]int64, numNodes)
	for i := range data {
		data[i] = make([]int64, size)
	}
	
	return &NUMALocalData{
		data:     data,
		numNodes: numNodes,
	}
}

// Load loads from NUMA-local data
func (n *NUMALocalData) Load(index int) int64 {
	nodeIndex := n.getNodeIndex()
	if nodeIndex < len(n.data) && index >= 0 && index < len(n.data[nodeIndex]) {
		return atomic.LoadInt64(&n.data[nodeIndex][index])
	}
	return 0
}

// Store stores to NUMA-local data
func (n *NUMALocalData) Store(index int, val int64) {
	nodeIndex := n.getNodeIndex()
	if nodeIndex < len(n.data) && index >= 0 && index < len(n.data[nodeIndex]) {
		atomic.StoreInt64(&n.data[nodeIndex][index], val)
	}
}

// Add adds to NUMA-local data
func (n *NUMALocalData) Add(index int, delta int64) int64 {
	nodeIndex := n.getNodeIndex()
	if nodeIndex < len(n.data) && index >= 0 && index < len(n.data[nodeIndex]) {
		return atomic.AddInt64(&n.data[nodeIndex][index], delta)
	}
	return 0
}

func (n *NUMALocalData) getNodeIndex() int {
	return int(getGoroutineID()) % n.numNodes
}

// LoadGlobal loads the sum across all NUMA nodes for a given index
func (n *NUMALocalData) LoadGlobal(index int) int64 {
	var total int64
	for nodeIndex := 0; nodeIndex < n.numNodes; nodeIndex++ {
		if index >= 0 && index < len(n.data[nodeIndex]) {
			total += atomic.LoadInt64(&n.data[nodeIndex][index])
		}
	}
	return total
}

// CPUAffinityCounter binds operations to specific CPUs
type CPUAffinityCounter struct {
	counters []int64
	numCPUs  int
}

func NewCPUAffinityCounter() *CPUAffinityCounter {
	numCPUs := runtime.NumCPU()
	// Allocate one counter per CPU with cache line padding
	counters := make([]int64, numCPUs*8)
	
	return &CPUAffinityCounter{
		counters: counters,
		numCPUs:  numCPUs,
	}
}

// Increment increments the counter for the current CPU
func (c *CPUAffinityCounter) Increment() {
	cpuIndex := getCPUIndex()
	atomic.AddInt64(&c.counters[cpuIndex*8], 1)
}

// Add adds delta to the counter for the current CPU
func (c *CPUAffinityCounter) Add(delta int64) {
	cpuIndex := getCPUIndex()
	atomic.AddInt64(&c.counters[cpuIndex*8], delta)
}

// Load returns the total count across all CPUs
func (c *CPUAffinityCounter) Load() int64 {
	var total int64
	for i := 0; i < c.numCPUs; i++ {
		total += atomic.LoadInt64(&c.counters[i*8])
	}
	return total
}

// LoadCPU returns the count for a specific CPU
func (c *CPUAffinityCounter) LoadCPU(cpuIndex int) int64 {
	if cpuIndex >= 0 && cpuIndex < c.numCPUs {
		return atomic.LoadInt64(&c.counters[cpuIndex*8])
	}
	return 0
}

// Helper functions (simplified implementations)
func getGoroutineID() uint64 {
	// Simplified goroutine ID extraction
	// In real implementation, you'd use runtime internals or other methods
	return uint64(uintptr(unsafe.Pointer(&struct{}{})))
}

func getCPUIndex() int {
	// Simplified CPU index (would use actual CPU affinity in real implementation)
	return int(getGoroutineID()) % runtime.NumCPU()
}