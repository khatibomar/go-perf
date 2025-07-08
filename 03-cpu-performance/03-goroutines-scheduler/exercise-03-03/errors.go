package main

import "errors"

// Common errors used throughout the system
var (
	ErrNoWorkersAvailable = errors.New("no workers available")
	ErrWorkerNotFound     = errors.New("worker not found")
	ErrWorkerBusy         = errors.New("worker is busy")
	ErrQueueFull          = errors.New("worker queue is full")
	ErrInvalidWork        = errors.New("invalid work item")
	ErrDistributorStopped = errors.New("distributor is stopped")
	ErrWorkerStopped      = errors.New("worker is stopped")
	ErrInvalidConfiguration = errors.New("invalid configuration")
	ErrTimeout            = errors.New("operation timed out")
	ErrCapacityExceeded   = errors.New("capacity exceeded")
)