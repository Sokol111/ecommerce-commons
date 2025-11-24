package consumer

import (
	"context"
	"math"
	"time"
)

// sleep паузує виконання на вказану тривалість або до скасування контексту
func sleep(ctx context.Context, d time.Duration) {
	timer := time.NewTimer(d)
	defer timer.Stop()

	select {
	case <-ctx.Done():
		return
	case <-timer.C:
		return
	}
}

// backoffDuration розраховує тривалість затримки для експоненційного backoff
func backoffDuration(attempt int, max time.Duration) time.Duration {
	duration := time.Duration(math.Pow(2, float64(attempt))) * time.Second
	if duration > max {
		return max
	}
	return duration
}
