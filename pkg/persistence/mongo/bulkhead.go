package mongo

import (
	"context"
	"time"

	"go.uber.org/zap"
	"golang.org/x/sync/semaphore"
)

// Bulkhead limits concurrent access to MongoDB operations using golang.org/x/sync/semaphore
type Bulkhead struct {
	semaphore *semaphore.Weighted
	timeout   time.Duration
	log       *zap.Logger
}

// NewBulkhead creates a new bulkhead with the specified limit and timeout
func NewBulkhead(limit int, timeout time.Duration, log *zap.Logger) *Bulkhead {
	log.Info("bulkhead initialized",
		zap.Int("limit", limit),
		zap.Duration("timeout", timeout),
	)

	return &Bulkhead{
		semaphore: semaphore.NewWeighted(int64(limit)),
		timeout:   timeout,
		log:       log,
	}
}

// Execute executes the given function within the bulkhead protection
func (b *Bulkhead) Execute(ctx context.Context, fn func() error) error {
	// Create timeout context
	timeoutCtx, cancel := context.WithTimeout(ctx, b.timeout)
	defer cancel()

	// Try to acquire semaphore slot
	if err := b.semaphore.Acquire(timeoutCtx, 1); err != nil {
		b.log.Warn("bulkhead acquisition failed",
			zap.Duration("timeout", b.timeout),
			zap.Error(err),
		)
		return err
	}
	defer b.semaphore.Release(1)

	// Execute the actual function
	return fn()
}
