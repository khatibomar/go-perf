package counters

import (
	"runtime"
	"sync/atomic"
)

// NUMACounter implements NUMA-aware counter design
type NUMACounter struct {
	nodes    []numaNode
	numNodes int
}

type numaNode struct {
	counters []paddedCounter
	_        [56]byte // Padding
}

func NewNUMACounter() *NUMACounter {
	// Simulate NUMA topology - in real implementation, this would query actual NUMA info
	numNodes := max(1, runtime.NumCPU()/4) // Assume 4 cores per NUMA node
	countersPerNode := 4                   // Multiple counters per node
	
	nc := &NUMACounter{
		nodes:    make([]numaNode, numNodes),
		numNodes: numNodes,
	}
	
	for i := range nc.nodes {
		nc.nodes[i].counters = make([]paddedCounter, countersPerNode)
	}
	
	return nc
}

func (nc *NUMACounter) Add(delta int64) {
	nodeID, counterID := nc.getOptimalCounter()
	atomic.AddInt64(&nc.nodes[nodeID].counters[counterID].value, delta)
}

func (nc *NUMACounter) Inc() {
	nc.Add(1)
}

func (nc *NUMACounter) Dec() {
	nc.Add(-1)
}

func (nc *NUMACounter) Get() int64 {
	var total int64
	for nodeIdx := range nc.nodes {
		for counterIdx := range nc.nodes[nodeIdx].counters {
			total += atomic.LoadInt64(&nc.nodes[nodeIdx].counters[counterIdx].value)
		}
	}
	return total
}

func (nc *NUMACounter) GetByNode() []int64 {
	nodeTotals := make([]int64, nc.numNodes)
	for nodeIdx := range nc.nodes {
		var nodeTotal int64
		for counterIdx := range nc.nodes[nodeIdx].counters {
			nodeTotal += atomic.LoadInt64(&nc.nodes[nodeIdx].counters[counterIdx].value)
		}
		nodeTotals[nodeIdx] = nodeTotal
	}
	return nodeTotals
}

func (nc *NUMACounter) Reset() {
	for nodeIdx := range nc.nodes {
		for counterIdx := range nc.nodes[nodeIdx].counters {
			atomic.StoreInt64(&nc.nodes[nodeIdx].counters[counterIdx].value, 0)
		}
	}
}

func (nc *NUMACounter) getOptimalCounter() (int, int) {
	// Get current CPU (simplified)
	// In real implementation, this would use CPU affinity and NUMA topology
	gid := runtime.NumGoroutine()
	nodeID := gid % nc.numNodes
	counterID := (gid / nc.numNodes) % len(nc.nodes[nodeID].counters)
	return nodeID, counterID
}

func (nc *NUMACounter) GetNumNodes() int {
	return nc.numNodes
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}