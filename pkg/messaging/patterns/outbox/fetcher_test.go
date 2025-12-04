package outbox

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

func TestFetcher_Run(t *testing.T) {
	t.Run("fetches entity and sends to channel", func(t *testing.T) {
		repo := newMockRepository()
		entity := &outboxEntity{
			ID:      "test-id",
			Payload: []byte("test-payload"),
			Topic:   "test-topic",
		}
		repo.SetFetchAndLockEntity(entity)

		entitiesChan := make(chan *outboxEntity, 10)
		f := newFetcher(repo, entitiesChan, zap.NewNop())

		ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
		defer cancel()

		var wg sync.WaitGroup
		wg.Add(1)
		go func() {
			defer wg.Done()
			_ = f.Run(ctx)
		}()

		// Wait for entity to be sent to channel
		select {
		case received := <-entitiesChan:
			assert.Equal(t, entity.ID, received.ID)
			assert.Equal(t, entity.Payload, received.Payload)
		case <-time.After(50 * time.Millisecond):
			t.Fatal("expected entity in channel")
		}

		cancel()
		wg.Wait()
	})

	t.Run("returns nil when context is cancelled", func(t *testing.T) {
		repo := newMockRepository()
		entitiesChan := make(chan *outboxEntity, 10)
		f := newFetcher(repo, entitiesChan, zap.NewNop())

		ctx, cancel := context.WithCancel(context.Background())
		cancel()

		err := f.Run(ctx)

		assert.NoError(t, err)
	})

	t.Run("waits when no entity found", func(t *testing.T) {
		repo := newMockRepository()
		repo.SetFetchAndLockError(errEntityNotFound)

		entitiesChan := make(chan *outboxEntity, 10)
		f := newFetcher(repo, entitiesChan, zap.NewNop())

		ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
		defer cancel()

		start := time.Now()
		err := f.Run(ctx)
		elapsed := time.Since(start)

		assert.NoError(t, err)
		// Should wait for context timeout (not the full 5 second wait)
		assert.Less(t, elapsed, 500*time.Millisecond)
	})

	t.Run("multiple entities are fetched sequentially", func(t *testing.T) {
		fetchCount := 0
		entities := []*outboxEntity{
			{ID: "entity-1"},
			{ID: "entity-2"},
			{ID: "entity-3"},
		}
		repo := newMockRepositoryWithFetchFunc(func(ctx context.Context) (*outboxEntity, error) {
			fetchCount++
			if fetchCount <= len(entities) {
				return entities[fetchCount-1], nil
			}
			return nil, errEntityNotFound
		})

		entitiesChan := make(chan *outboxEntity, 10)
		f := newFetcher(repo, entitiesChan, zap.NewNop())

		ctx, cancel := context.WithTimeout(context.Background(), 200*time.Millisecond)
		defer cancel()

		var wg sync.WaitGroup
		wg.Add(1)
		go func() {
			defer wg.Done()
			_ = f.Run(ctx)
		}()

		received := make([]*outboxEntity, 0)
		timeout := time.After(150 * time.Millisecond)
	loop:
		for {
			select {
			case e := <-entitiesChan:
				received = append(received, e)
				if len(received) >= 3 {
					break loop
				}
			case <-timeout:
				break loop
			}
		}

		cancel()
		wg.Wait()

		require.Len(t, received, 3)
		assert.Equal(t, "entity-1", received[0].ID)
		assert.Equal(t, "entity-2", received[1].ID)
		assert.Equal(t, "entity-3", received[2].ID)
	})
}

func TestNewFetcher(t *testing.T) {
	t.Run("creates fetcher with dependencies", func(t *testing.T) {
		repo := newMockRepository()
		entitiesChan := make(chan *outboxEntity)
		logger := zap.NewNop()

		f := newFetcher(repo, entitiesChan, logger)

		assert.NotNil(t, f)
		assert.Equal(t, repo, f.outboxRepository)
		assert.NotNil(t, f.entitiesChan)
		assert.Equal(t, logger, f.logger)
	})
}
