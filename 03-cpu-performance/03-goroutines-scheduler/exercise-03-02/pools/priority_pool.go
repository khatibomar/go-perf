package pools

import (
	"container/heap"
	"context"
	"sync"
	"time"
)

// Priority levels
const (
	PriorityLow    = 0
	PriorityNormal = 1
	PriorityHigh   = 2
	PriorityCritical = 3
)

// PriorityPool implements a priority-based worker pool
type PriorityPool struct {
	workers      int
	priorityQueues []*PriorityQueue
	wg           sync.WaitGroup
	ctx          context.Context
	cancel       context.CancelFunc
	metrics      *PoolMetrics
	scheduler    *PriorityScheduler
	started      bool
	mu           sync.RWMutex
}

// PriorityTask represents a task with priority
type PriorityTask struct {
	Task      func()
	Priority  int
	SubmitTime time.Time
	Index     int // heap index
}

// PriorityQueue implements a priority queue for tasks
type PriorityQueue struct {
	tasks []*PriorityTask
	mu    sync.RWMutex
}

// PriorityScheduler manages task scheduling and starvation prevention
type PriorityScheduler struct {
	starvationThreshold time.Duration
	boostInterval       time.Duration
	lastBoost           time.Time
	mu                  sync.RWMutex
}

// NewPriorityPool creates a new priority-based worker pool
func NewPriorityPool(workers int, queueSizes []int) *PriorityPool {
	ctx, cancel := context.WithCancel(context.Background())
	
	// Create priority queues (4 levels: Low, Normal, High, Critical)
	queues := make([]*PriorityQueue, 4)
	for i := 0; i < 4; i++ {
		size := 100 // default size
		if i < len(queueSizes) {
			size = queueSizes[i]
		}
		queues[i] = &PriorityQueue{
			tasks: make([]*PriorityTask, 0, size),
		}
	}
	
	return &PriorityPool{
		workers:        workers,
		priorityQueues: queues,
		ctx:            ctx,
		cancel:         cancel,
		metrics:        &PoolMetrics{},
		scheduler: &PriorityScheduler{
			starvationThreshold: time.Second * 30, // Boost low priority tasks after 30s
			boostInterval:       time.Second * 10, // Check for starvation every 10s
		},
	}
}

// Start initializes and starts the priority pool
func (p *PriorityPool) Start() error {
	p.mu.Lock()
	defer p.mu.Unlock()
	
	if p.started {
		return nil
	}
	
	// Start workers
	for i := 0; i < p.workers; i++ {
		p.wg.Add(1)
		go p.worker(i)
	}
	
	// Start starvation prevention scheduler
	go p.starvationPrevention()
	
	p.started = true
	return nil
}

// Submit adds a task with specified priority to the pool
func (p *PriorityPool) Submit(task func(), priority int) error {
	if priority < PriorityLow || priority > PriorityCritical {
		priority = PriorityNormal
	}
	
	p.metrics.mu.Lock()
	p.metrics.TasksSubmitted++
	p.metrics.mu.Unlock()
	
	priorityTask := &PriorityTask{
		Task:       task,
		Priority:   priority,
		SubmitTime: time.Now(),
	}
	
	queue := p.priorityQueues[priority]
	queue.mu.Lock()
	defer queue.mu.Unlock()
	
	heap.Push(queue, priorityTask)
	return nil
}

// worker processes tasks from priority queues
func (p *PriorityPool) worker(id int) {
	defer p.wg.Done()
	
	for {
		task := p.getNextTask()
		if task == nil {
			select {
			case <-p.ctx.Done():
				return
			default:
				// No task available, brief pause
				time.Sleep(time.Millisecond * 10)
				continue
			}
		}
		
		p.executeTask(task)
	}
}

// getNextTask retrieves the next task based on priority
func (p *PriorityPool) getNextTask() *PriorityTask {
	// Check queues from highest to lowest priority
	for i := PriorityCritical; i >= PriorityLow; i-- {
		queue := p.priorityQueues[i]
		queue.mu.Lock()
		if queue.Len() > 0 {
			task := heap.Pop(queue).(*PriorityTask)
			queue.mu.Unlock()
			return task
		}
		queue.mu.Unlock()
	}
	return nil
}

// executeTask executes a task and updates metrics
func (p *PriorityPool) executeTask(task *PriorityTask) {
	// Check if task is nil to prevent panic
	if task == nil {
		return
	}
	
	start := time.Now()
	
	p.metrics.mu.Lock()
	p.metrics.TasksInProgress++
	p.metrics.mu.Unlock()
	
	// Check if task.Task is nil to prevent panic
	if task.Task != nil {
		task.Task()
	}
	
	duration := time.Since(start)
	waitTime := start.Sub(task.SubmitTime)
	
	p.metrics.mu.Lock()
	p.metrics.TasksCompleted++
	p.metrics.TasksInProgress--
	p.metrics.AverageLatency = (p.metrics.AverageLatency + duration) / 2
	p.metrics.mu.Unlock()
	
	// Track wait time for priority analysis
	_ = waitTime // Could be used for metrics
}

// starvationPrevention monitors and prevents low-priority task starvation
func (p *PriorityPool) starvationPrevention() {
	ticker := time.NewTicker(p.scheduler.boostInterval)
	defer ticker.Stop()
	
	for {
		select {
		case <-ticker.C:
			p.checkAndBoostStarvedTasks()
		case <-p.ctx.Done():
			return
		}
	}
}

// checkAndBoostStarvedTasks promotes long-waiting low-priority tasks
func (p *PriorityPool) checkAndBoostStarvedTasks() {
	p.scheduler.mu.Lock()
	defer p.scheduler.mu.Unlock()
	
	now := time.Now()
	if now.Sub(p.scheduler.lastBoost) < p.scheduler.boostInterval {
		return
	}
	
	// Check low and normal priority queues for starved tasks
	for priority := PriorityLow; priority <= PriorityNormal; priority++ {
		queue := p.priorityQueues[priority]
		queue.mu.Lock()
		
		// Find tasks that have been waiting too long
		var tasksToBoost []*PriorityTask
		for _, task := range queue.tasks {
			if now.Sub(task.SubmitTime) > p.scheduler.starvationThreshold {
				tasksToBoost = append(tasksToBoost, task)
			}
		}
		
		// Remove starved tasks from current queue
		for _, task := range tasksToBoost {
			p.removeTaskFromQueue(queue, task)
		}
		
		queue.mu.Unlock()
		
		// Boost priority and re-submit
		for _, task := range tasksToBoost {
			newPriority := task.Priority + 1
			if newPriority > PriorityCritical {
				newPriority = PriorityCritical
			}
			task.Priority = newPriority
			
			newQueue := p.priorityQueues[newPriority]
			newQueue.mu.Lock()
			heap.Push(newQueue, task)
			newQueue.mu.Unlock()
		}
	}
	
	p.scheduler.lastBoost = now
}

// removeTaskFromQueue removes a specific task from the queue
func (p *PriorityPool) removeTaskFromQueue(queue *PriorityQueue, taskToRemove *PriorityTask) {
	for i, task := range queue.tasks {
		if task == taskToRemove {
			heap.Remove(queue, i)
			break
		}
	}
}

// Shutdown gracefully shuts down the pool
func (p *PriorityPool) Shutdown(timeout time.Duration) error {
	p.mu.Lock()
	defer p.mu.Unlock()
	
	if !p.started {
		return nil
	}
	
	p.cancel()
	
	done := make(chan struct{})
	go func() {
		p.wg.Wait()
		close(done)
	}()
	
	select {
	case <-done:
		return nil
	case <-time.After(timeout):
		return ErrShutdownTimeout
	}
}

// GetMetrics returns current pool metrics
func (p *PriorityPool) GetMetrics() PoolMetrics {
	p.metrics.mu.RLock()
	defer p.metrics.mu.RUnlock()
	return *p.metrics
}

// GetPriorityQueueDepths returns queue depths for each priority level
func (p *PriorityPool) GetPriorityQueueDepths() []int {
	depths := make([]int, len(p.priorityQueues))
	for i, queue := range p.priorityQueues {
		queue.mu.RLock()
		depths[i] = queue.Len()
		queue.mu.RUnlock()
	}
	return depths
}

// QueueDepth returns the total queue depth across all priorities
func (p *PriorityPool) QueueDepth() int {
	total := 0
	for _, queue := range p.priorityQueues {
		queue.mu.RLock()
		total += queue.Len()
		queue.mu.RUnlock()
	}
	return total
}

// WorkerCount returns the number of workers
func (p *PriorityPool) WorkerCount() int {
	return p.workers
}

// Priority Queue implementation for heap interface
func (pq *PriorityQueue) Len() int { return len(pq.tasks) }

func (pq *PriorityQueue) Less(i, j int) bool {
	// Higher priority first, then FIFO for same priority
	if pq.tasks[i].Priority == pq.tasks[j].Priority {
		return pq.tasks[i].SubmitTime.Before(pq.tasks[j].SubmitTime)
	}
	return pq.tasks[i].Priority > pq.tasks[j].Priority
}

func (pq *PriorityQueue) Swap(i, j int) {
	pq.tasks[i], pq.tasks[j] = pq.tasks[j], pq.tasks[i]
	pq.tasks[i].Index = i
	pq.tasks[j].Index = j
}

func (pq *PriorityQueue) Push(x interface{}) {
	n := len(pq.tasks)
	task := x.(*PriorityTask)
	task.Index = n
	pq.tasks = append(pq.tasks, task)
}

func (pq *PriorityQueue) Pop() interface{} {
	old := pq.tasks
	n := len(old)
	task := old[n-1]
	old[n-1] = nil
	task.Index = -1
	pq.tasks = old[0 : n-1]
	return task
}