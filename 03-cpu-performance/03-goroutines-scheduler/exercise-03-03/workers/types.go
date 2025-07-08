package workers

import (
	"fmt"
)

// Error variables
var (
	ErrWorkerStopped = fmt.Errorf("worker is stopped")
	ErrQueueFull     = fmt.Errorf("worker queue is full")
)