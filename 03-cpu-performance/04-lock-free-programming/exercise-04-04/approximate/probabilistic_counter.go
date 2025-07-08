package approximate

import (
	"math"
	"math/rand"
	"sync/atomic"
	"time"
)

// ProbabilisticCounter implements probabilistic counting for high throughput
type ProbabilisticCounter struct {
	value       int64
	probability float64
	scale       float64
	rng         *rand.Rand
	_           [32]byte // Padding
}

func NewProbabilisticCounter(probability float64) *ProbabilisticCounter {
	return &ProbabilisticCounter{
		probability: probability,
		scale:       1.0 / probability,
		rng:         rand.New(rand.NewSource(time.Now().UnixNano())),
	}
}

func (pc *ProbabilisticCounter) Add(delta int64) {
	for i := int64(0); i < delta; i++ {
		if pc.rng.Float64() < pc.probability {
			atomic.AddInt64(&pc.value, 1)
		}
	}
}

func (pc *ProbabilisticCounter) Inc() {
	if pc.rng.Float64() < pc.probability {
		atomic.AddInt64(&pc.value, 1)
	}
}

func (pc *ProbabilisticCounter) Dec() {
	if pc.rng.Float64() < pc.probability {
		atomic.AddInt64(&pc.value, -1)
	}
}

func (pc *ProbabilisticCounter) Get() int64 {
	observed := atomic.LoadInt64(&pc.value)
	return int64(float64(observed) * pc.scale)
}

func (pc *ProbabilisticCounter) GetExact() int64 {
	return atomic.LoadInt64(&pc.value)
}

func (pc *ProbabilisticCounter) Reset() {
	atomic.StoreInt64(&pc.value, 0)
}

func (pc *ProbabilisticCounter) GetProbability() float64 {
	return pc.probability
}

func (pc *ProbabilisticCounter) GetScale() float64 {
	return pc.scale
}

// GetErrorBounds returns the theoretical error bounds
func (pc *ProbabilisticCounter) GetErrorBounds(confidence float64) (float64, float64) {
	observed := float64(atomic.LoadInt64(&pc.value))
	estimate := observed * pc.scale
	
	// Standard error for probabilistic counting
	stdError := math.Sqrt(observed * (1 - pc.probability) / pc.probability)
	marginOfError := stdError * pc.scale
	
	// Z-score for confidence level (simplified)
	zScore := 1.96 // 95% confidence
	if confidence > 0.99 {
		zScore = 2.58
	} else if confidence < 0.9 {
		zScore = 1.64
	}
	
	errorMargin := zScore * marginOfError
	return estimate - errorMargin, estimate + errorMargin
}