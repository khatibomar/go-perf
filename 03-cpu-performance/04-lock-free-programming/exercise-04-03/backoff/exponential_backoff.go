package backoff

import (
	"math/rand"
	"runtime"
	"sync/atomic"
	"time"
)

// ExponentialBackoff implements exponential backoff strategy
type ExponentialBackoff struct {
	initialDelay time.Duration
	maxDelay     time.Duration
	multiplier   float64
	jitter       bool
	currentDelay time.Duration
	attempts     int64
}

func NewExponentialBackoff(initial, max time.Duration, multiplier float64, jitter bool) *ExponentialBackoff {
	return &ExponentialBackoff{
		initialDelay: initial,
		maxDelay:     max,
		multiplier:   multiplier,
		jitter:       jitter,
		currentDelay: initial,
	}
}

// Backoff performs the backoff delay
func (eb *ExponentialBackoff) Backoff() {
	atomic.AddInt64(&eb.attempts, 1)
	
	delay := eb.currentDelay
	
	if eb.jitter {
		// Add random jitter (±25%)
		jitterRange := int64(float64(delay) * 0.25)
		jitterOffset := rand.Int63n(2*jitterRange) - jitterRange
		delay = time.Duration(int64(delay) + jitterOffset)
		if delay < 0 {
			delay = eb.initialDelay
		}
	}
	
	if delay > 0 {
		time.Sleep(delay)
	} else {
		runtime.Gosched()
	}
	
	// Update delay for next time
	nextDelay := time.Duration(float64(eb.currentDelay) * eb.multiplier)
	if nextDelay > eb.maxDelay {
		nextDelay = eb.maxDelay
	}
	eb.currentDelay = nextDelay
}

// Reset resets the backoff to initial state
func (eb *ExponentialBackoff) Reset() {
	eb.currentDelay = eb.initialDelay
	atomic.StoreInt64(&eb.attempts, 0)
}

// GetAttempts returns the number of backoff attempts
func (eb *ExponentialBackoff) GetAttempts() int64 {
	return atomic.LoadInt64(&eb.attempts)
}

// GetCurrentDelay returns the current delay duration
func (eb *ExponentialBackoff) GetCurrentDelay() time.Duration {
	delay := eb.currentDelay
	
	if eb.jitter {
		// Add random jitter (±25%)
		jitterRange := int64(float64(delay) * 0.25)
		jitterOffset := rand.Int63n(2*jitterRange) - jitterRange
		delay = time.Duration(int64(delay) + jitterOffset)
		if delay < 0 {
			delay = eb.initialDelay
		}
	}
	
	return delay
}

// BackoffWithLimit performs backoff with attempt limit
func (eb *ExponentialBackoff) BackoffWithLimit(maxAttempts int64) bool {
	if atomic.LoadInt64(&eb.attempts) >= maxAttempts {
		return false
	}
	eb.Backoff()
	return true
}

// SimpleExponentialBackoff is a stateless version that returns the calculated delay
func SimpleExponentialBackoff(initial time.Duration, multiplier float64, attempt int) time.Duration {
	delay := initial
	for i := 0; i < attempt; i++ {
		delay = time.Duration(float64(delay) * multiplier)
	}
	return delay
}

// ExponentialBackoffWithJitter adds jitter to prevent thundering herd
func ExponentialBackoffWithJitter(attempt int, initial, max time.Duration, multiplier float64) {
	delay := initial
	for i := 0; i < attempt; i++ {
		delay = time.Duration(float64(delay) * multiplier)
		if delay > max {
			delay = max
			break
		}
	}
	
	// Add jitter (±50%)
	if delay > 0 {
		jitterRange := int64(float64(delay) * 0.5)
		jitterOffset := rand.Int63n(2*jitterRange) - jitterRange
		finalDelay := time.Duration(int64(delay) + jitterOffset)
		if finalDelay < 0 {
			finalDelay = initial
		}
		time.Sleep(finalDelay)
	} else {
		runtime.Gosched()
	}
}

// CASWithExponentialBackoff performs CAS with exponential backoff
func CASWithExponentialBackoff(addr *int64, old, new int64, initial, max time.Duration, multiplier float64, maxAttempts int) bool {
	backoff := NewExponentialBackoff(
		initial,
		max,
		multiplier,
		true,
	)
	
	for i := 0; i < maxAttempts; i++ {
		if atomic.CompareAndSwapInt64(addr, old, new) {
			return true
		}
		
		// Check if value changed
		current := atomic.LoadInt64(addr)
		if current != old {
			return false
		}
		
		backoff.Backoff()
	}
	
	return false
}