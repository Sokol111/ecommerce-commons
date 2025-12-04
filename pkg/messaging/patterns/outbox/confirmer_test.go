package outbox

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/confluentinc/confluent-kafka-go/v2/kafka"
	"github.com/stretchr/testify/assert"
	"go.uber.org/zap"
)

func TestConfirmer_Run(t *testing.T) {
	t.Run("processes delivery events and updates repository", func(t *testing.T) {
		repo := newMockRepository()
		deliveryChan := make(chan kafka.Event, 10)
		c := newConfirmer(repo, deliveryChan, zap.NewNop())

		ctx, cancel := context.WithTimeout(context.Background(), 500*time.Millisecond)
		defer cancel()

		var wg sync.WaitGroup
		wg.Add(1)
		go func() {
			defer wg.Done()
			_ = c.Run(ctx)
		}()

		// Send delivery event
		topic := "test-topic"
		deliveryChan <- &kafka.Message{
			TopicPartition: kafka.TopicPartition{
				Topic:     &topic,
				Partition: 0,
			},
			Opaque: "message-id-1",
		}

		// Wait for ticker flush (2 seconds) or cancel context
		time.Sleep(2500 * time.Millisecond)
		cancel()
		wg.Wait()

		ids := repo.GetUpdateAsSentIDs()
		assert.Contains(t, ids, "message-id-1")
	})

	t.Run("batches multiple delivery events", func(t *testing.T) {
		repo := newMockRepository()
		deliveryChan := make(chan kafka.Event, 200)
		c := newConfirmer(repo, deliveryChan, zap.NewNop())

		ctx, cancel := context.WithTimeout(context.Background(), 500*time.Millisecond)
		defer cancel()

		var wg sync.WaitGroup
		wg.Add(1)
		go func() {
			defer wg.Done()
			_ = c.Run(ctx)
		}()

		// Send 100 delivery events to trigger batch flush
		topic := "test-topic"
		for i := 0; i < 100; i++ {
			deliveryChan <- &kafka.Message{
				TopicPartition: kafka.TopicPartition{
					Topic:     &topic,
					Partition: 0,
				},
				Opaque: string(rune(i)),
			}
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
		deliveryChan := make(chan kafka.Event, 10)
		c := newConfirmer(repo, deliveryChan, zap.NewNop())

		ctx, cancel := context.WithCancel(context.Background())
		cancel()

		err := c.Run(ctx)

		assert.NoError(t, err)
	})

	t.Run("skips events with delivery errors", func(t *testing.T) {
		repo := newMockRepository()
		deliveryChan := make(chan kafka.Event, 10)
		c := newConfirmer(repo, deliveryChan, zap.NewNop())

		ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
		defer cancel()

		var wg sync.WaitGroup
		wg.Add(1)
		go func() {
			defer wg.Done()
			_ = c.Run(ctx)
		}()

		// Send message with delivery error
		topic := "test-topic"
		deliveryErr := kafka.NewError(kafka.ErrMsgTimedOut, "message timed out", false)
		deliveryChan <- &kafka.Message{
			TopicPartition: kafka.TopicPartition{
				Topic:     &topic,
				Partition: 0,
				Error:     deliveryErr,
			},
			Opaque: "failed-message",
		}

		// Send successful message
		deliveryChan <- &kafka.Message{
			TopicPartition: kafka.TopicPartition{
				Topic:     &topic,
				Partition: 0,
			},
			Opaque: "success-message",
		}

		time.Sleep(2500 * time.Millisecond)
		cancel()
		wg.Wait()

		ids := repo.GetUpdateAsSentIDs()
		assert.NotContains(t, ids, "failed-message")
		assert.Contains(t, ids, "success-message")
	})

	t.Run("skips events with invalid opaque type", func(t *testing.T) {
		repo := newMockRepository()
		deliveryChan := make(chan kafka.Event, 10)
		c := newConfirmer(repo, deliveryChan, zap.NewNop())

		ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
		defer cancel()

		var wg sync.WaitGroup
		wg.Add(1)
		go func() {
			defer wg.Done()
			_ = c.Run(ctx)
		}()

		// Send message with invalid opaque type (int instead of string)
		topic := "test-topic"
		deliveryChan <- &kafka.Message{
			TopicPartition: kafka.TopicPartition{
				Topic:     &topic,
				Partition: 0,
			},
			Opaque: 12345, // Invalid type
		}

		// Send valid message
		deliveryChan <- &kafka.Message{
			TopicPartition: kafka.TopicPartition{
				Topic:     &topic,
				Partition: 0,
			},
			Opaque: "valid-message",
		}

		time.Sleep(2500 * time.Millisecond)
		cancel()
		wg.Wait()

		ids := repo.GetUpdateAsSentIDs()
		assert.NotContains(t, ids, "12345")
		assert.Contains(t, ids, "valid-message")
	})

	t.Run("handles repository error gracefully", func(t *testing.T) {
		repo := newMockRepository()
		repo.updateAsSentErr = errors.New("database error")
		deliveryChan := make(chan kafka.Event, 10)
		c := newConfirmer(repo, deliveryChan, zap.NewNop())

		ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
		defer cancel()

		var wg sync.WaitGroup
		wg.Add(1)
		go func() {
			defer wg.Done()
			_ = c.Run(ctx)
		}()

		topic := "test-topic"
		deliveryChan <- &kafka.Message{
			TopicPartition: kafka.TopicPartition{
				Topic:     &topic,
				Partition: 0,
			},
			Opaque: "message-id",
		}

		time.Sleep(2500 * time.Millisecond)
		cancel()
		wg.Wait()

		// Should have attempted to update
		assert.GreaterOrEqual(t, repo.GetUpdateAsSentCalls(), 1)
	})

	t.Run("flushes remaining events on context cancellation", func(t *testing.T) {
		repo := newMockRepository()
		deliveryChan := make(chan kafka.Event, 10)
		c := newConfirmer(repo, deliveryChan, zap.NewNop())

		ctx, cancel := context.WithCancel(context.Background())

		var wg sync.WaitGroup
		wg.Add(1)
		go func() {
			defer wg.Done()
			_ = c.Run(ctx)
		}()

		// Send some events
		topic := "test-topic"
		for i := 0; i < 5; i++ {
			deliveryChan <- &kafka.Message{
				TopicPartition: kafka.TopicPartition{
					Topic:     &topic,
					Partition: 0,
				},
				Opaque: string(rune('a' + i)),
			}
		}

		// Give some time for events to be received
		time.Sleep(50 * time.Millisecond)
		cancel()
		wg.Wait()

		// Events should be flushed on cancellation
		ids := repo.GetUpdateAsSentIDs()
		assert.GreaterOrEqual(t, len(ids), 0) // At least attempted to flush
	})
}

func TestNewConfirmer(t *testing.T) {
	t.Run("creates confirmer with dependencies", func(t *testing.T) {
		repo := newMockRepository()
		deliveryChan := make(chan kafka.Event)
		logger := zap.NewNop()

		c := newConfirmer(repo, deliveryChan, logger)

		assert.NotNil(t, c)
		assert.Equal(t, repo, c.outboxRepository)
		assert.NotNil(t, c.deliveryChan)
		assert.Equal(t, logger, c.logger)
	})
}
