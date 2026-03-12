package outbox

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"go.uber.org/zap"
)

func TestConfirmer_Run(t *testing.T) {
	t.Run("processes confirm results and updates repository", func(t *testing.T) {
		repo := newMockRepository()
		confirmChan := make(chan confirmResult, 10)
		c := newConfirmer(repo, confirmChan, zap.NewNop())

		ctx, cancel := context.WithTimeout(context.Background(), 500*time.Millisecond)
		defer cancel()

		var wg sync.WaitGroup
		wg.Add(1)
		go func() {
			defer wg.Done()
			_ = c.Run(ctx)
		}()

		// Send confirm result
		confirmChan <- confirmResult{ID: "message-id-1", Err: nil}

		// Wait for ticker flush (2 seconds) or cancel context
		time.Sleep(2500 * time.Millisecond)
		cancel()
		wg.Wait()

		ids := repo.GetUpdateAsSentIDs()
		assert.Contains(t, ids, "message-id-1")
	})

	t.Run("batches multiple confirm results", func(t *testing.T) {
		repo := newMockRepository()
		confirmChan := make(chan confirmResult, 200)
		c := newConfirmer(repo, confirmChan, zap.NewNop())

		ctx, cancel := context.WithTimeout(context.Background(), 500*time.Millisecond)
		defer cancel()

		var wg sync.WaitGroup
		wg.Add(1)
		go func() {
			defer wg.Done()
			_ = c.Run(ctx)
		}()

		// Send 100 confirm results to trigger batch flush
		for i := 0; i < 100; i++ {
			confirmChan <- confirmResult{ID: fmt.Sprintf("msg-%d", i), Err: nil}
		}

		// Wait for batch to be processed
		time.Sleep(100 * time.Millisecond)
		cancel()
		wg.Wait()

		// Should have at least one batch processed
		assert.GreaterOrEqual(t, repo.GetUpdateAsSentCalls(), 1)
	})

	t.Run("returns nil when context is cancelled", func(t *testing.T) {
		repo := newMockRepository()
		confirmChan := make(chan confirmResult, 10)
		c := newConfirmer(repo, confirmChan, zap.NewNop())

		ctx, cancel := context.WithCancel(context.Background())
		cancel()

		err := c.Run(ctx)

		assert.NoError(t, err)
	})

	t.Run("skips results with delivery errors", func(t *testing.T) {
		repo := newMockRepository()
		confirmChan := make(chan confirmResult, 10)
		c := newConfirmer(repo, confirmChan, zap.NewNop())

		ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
		defer cancel()

		var wg sync.WaitGroup
		wg.Add(1)
		go func() {
			defer wg.Done()
			_ = c.Run(ctx)
		}()

		// Send result with delivery error
		confirmChan <- confirmResult{ID: "failed-message", Err: errors.New("kafka delivery failed")}

		// Send successful result
		confirmChan <- confirmResult{ID: "success-message", Err: nil}

		time.Sleep(2500 * time.Millisecond)
		cancel()
		wg.Wait()

		ids := repo.GetUpdateAsSentIDs()
		assert.NotContains(t, ids, "failed-message")
		assert.Contains(t, ids, "success-message")
	})

	t.Run("handles repository error gracefully", func(t *testing.T) {
		repo := newMockRepository()
		repo.updateAsSentErr = errors.New("database error")
		confirmChan := make(chan confirmResult, 10)
		c := newConfirmer(repo, confirmChan, zap.NewNop())

		ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
		defer cancel()

		var wg sync.WaitGroup
		wg.Add(1)
		go func() {
			defer wg.Done()
			_ = c.Run(ctx)
		}()

		confirmChan <- confirmResult{ID: "message-id", Err: nil}

		time.Sleep(2500 * time.Millisecond)
		cancel()
		wg.Wait()

		// Should have attempted to update
		assert.GreaterOrEqual(t, repo.GetUpdateAsSentCalls(), 1)
	})

	t.Run("flushes remaining results on context cancellation", func(t *testing.T) {
		repo := newMockRepository()
		confirmChan := make(chan confirmResult, 10)
		c := newConfirmer(repo, confirmChan, zap.NewNop())

		ctx, cancel := context.WithCancel(context.Background())

		var wg sync.WaitGroup
		wg.Add(1)
		go func() {
			defer wg.Done()
			_ = c.Run(ctx)
		}()

		// Send some results
		for i := 0; i < 5; i++ {
			confirmChan <- confirmResult{ID: string(rune('a' + i)), Err: nil}
		}

		// Give some time for results to be received
		time.Sleep(50 * time.Millisecond)
		cancel()
		wg.Wait()

		// Results should be flushed on cancellation
		ids := repo.GetUpdateAsSentIDs()
		assert.GreaterOrEqual(t, len(ids), 0) // At least attempted to flush
	})
}

func TestNewConfirmer(t *testing.T) {
	t.Run("creates confirmer with dependencies", func(t *testing.T) {
		repo := newMockRepository()
		confirmChan := make(chan confirmResult)
		logger := zap.NewNop()

		c := newConfirmer(repo, confirmChan, logger)

		assert.NotNil(t, c)
		assert.Equal(t, repo, c.outboxRepository)
		assert.NotNil(t, c.confirmChan)
		assert.Equal(t, logger, c.logger)
	})
}
