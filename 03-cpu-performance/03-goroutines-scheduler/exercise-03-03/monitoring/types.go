package monitoring

import (
	. "exercise-03-03/types"
)





// SystemMetrics contains overall system metrics
type SystemMetrics struct {
	Overall OverallMetrics
	Distributors map[string]DistributorMetrics
	Workers map[string]WorkerMetrics
}