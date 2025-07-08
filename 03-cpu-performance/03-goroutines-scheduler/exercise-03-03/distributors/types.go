package distributors

import (
	"fmt"
)

// Error variables
var (
	ErrNoWorkersAvailable    = fmt.Errorf("no workers available")
	ErrWorkerNotFound        = fmt.Errorf("worker not found")
	ErrQueueFull             = fmt.Errorf("worker queue is full")
	ErrDistributorStopped    = fmt.Errorf("distributor is stopped")
	ErrWorkerStopped         = fmt.Errorf("worker is stopped")
	ErrInvalidConfiguration  = fmt.Errorf("invalid configuration")
)