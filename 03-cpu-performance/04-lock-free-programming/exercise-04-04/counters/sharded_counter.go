package counters

import (
	"runtime"
	"sync/atomic"
)

// ShardedCounter uses per-core shards to minimize contention
type ShardedCounter struct {
	shards    []paddedCounter
	numShards int
}

func NewShardedCounter() *ShardedCounter {
	numShards := runtime.NumCPU() * 2 // 2x CPU cores for better distribution
	return &ShardedCounter{
		shards:    make([]paddedCounter, numShards),
		numShards: numShards,
	}
}

func NewShardedCounterWithShards(numShards int) *ShardedCounter {
	return &ShardedCounter{
		shards:    make([]paddedCounter, numShards),
		numShards: numShards,
	}
}

func (c *ShardedCounter) Add(delta int64) {
	shard := c.getShardIndex()
	atomic.AddInt64(&c.shards[shard].value, delta)
}

func (c *ShardedCounter) Inc() {
	c.Add(1)
}

func (c *ShardedCounter) Dec() {
	c.Add(-1)
}

func (c *ShardedCounter) Get() int64 {
	var total int64
	for i := range c.shards {
		total += atomic.LoadInt64(&c.shards[i].value)
	}
	return total
}

func (c *ShardedCounter) GetShardValues() []int64 {
	values := make([]int64, len(c.shards))
	for i := range c.shards {
		values[i] = atomic.LoadInt64(&c.shards[i].value)
	}
	return values
}

func (c *ShardedCounter) Reset() {
	for i := range c.shards {
		atomic.StoreInt64(&c.shards[i].value, 0)
	}
}

func (c *ShardedCounter) getShardIndex() int {
	// Use a combination of goroutine ID and CPU to distribute load
	gid := runtime.NumGoroutine()
	return gid % c.numShards
}

func (c *ShardedCounter) GetNumShards() int {
	return c.numShards
}