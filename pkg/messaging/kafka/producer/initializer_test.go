package producer

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/confluentinc/confluent-kafka-go/v2/kafka"
	"github.com/stretchr/testify/assert"
	"go.uber.org/zap"
)

// mockMetadataProvider is a mock implementation of metadataProvider interface for testing.
type mockMetadataProvider struct {
	getMetadataFunc func(topic *string, allTopics bool, timeoutMs int) (*kafka.Metadata, error)
}

func (m *mockMetadataProvider) GetMetadata(topic *string, allTopics bool, timeoutMs int) (*kafka.Metadata, error) {
	if m.getMetadataFunc != nil {
		return m.getMetadataFunc(topic, allTopics, timeoutMs)
	}
	return &kafka.Metadata{Brokers: []kafka.BrokerMetadata{{ID: 1}}}, nil
}

func TestWaitForBrokers(t *testing.T) {
	t.Run("returns nil when brokers are available", func(t *testing.T) {
		mock := &mockMetadataProvider{
			getMetadataFunc: func(topic *string, allTopics bool, timeoutMs int) (*kafka.Metadata, error) {
				return &kafka.Metadata{Brokers: []kafka.BrokerMetadata{{ID: 1}}}, nil
			},
		}

		err := waitForBrokers(context.Background(), mock, zap.NewNop(), 5, true)

		assert.NoError(t, err)
	})

	t.Run("returns error when timeout and failOnError is true", func(t *testing.T) {
		mock := &mockMetadataProvider{
			getMetadataFunc: func(topic *string, allTopics bool, timeoutMs int) (*kafka.Metadata, error) {
				return nil, errors.New("no brokers")
			},
		}

		err := waitForBrokers(context.Background(), mock, zap.NewNop(), 1, true)

		assert.Error(t, err)
		assert.Equal(t, context.DeadlineExceeded, err)
	})

	t.Run("returns nil when timeout and failOnError is false", func(t *testing.T) {
		mock := &mockMetadataProvider{
			getMetadataFunc: func(topic *string, allTopics bool, timeoutMs int) (*kafka.Metadata, error) {
				return nil, errors.New("no brokers")
			},
		}

		err := waitForBrokers(context.Background(), mock, zap.NewNop(), 1, false)

		assert.NoError(t, err)
	})

	t.Run("respects context cancellation", func(t *testing.T) {
		mock := &mockMetadataProvider{
			getMetadataFunc: func(topic *string, allTopics bool, timeoutMs int) (*kafka.Metadata, error) {
				return nil, errors.New("no brokers")
			},
		}

		ctx, cancel := context.WithCancel(context.Background())
		cancel()

		err := waitForBrokers(ctx, mock, zap.NewNop(), 0, true)

		assert.Error(t, err)
		assert.Equal(t, context.Canceled, err)
	})
}

func TestPollBrokers(t *testing.T) {
	t.Run("returns nil when brokers found", func(t *testing.T) {
		mock := &mockMetadataProvider{
			getMetadataFunc: func(topic *string, allTopics bool, timeoutMs int) (*kafka.Metadata, error) {
				return &kafka.Metadata{Brokers: []kafka.BrokerMetadata{{ID: 1}, {ID: 2}}}, nil
			},
		}

		err := pollBrokers(context.Background(), mock)

		assert.NoError(t, err)
	})

	t.Run("returns error on context cancellation", func(t *testing.T) {
		mock := &mockMetadataProvider{
			getMetadataFunc: func(topic *string, allTopics bool, timeoutMs int) (*kafka.Metadata, error) {
				return nil, errors.New("no brokers")
			},
		}

		ctx, cancel := context.WithCancel(context.Background())
		cancel()

		err := pollBrokers(ctx, mock)

		assert.Error(t, err)
		assert.Equal(t, context.Canceled, err)
	})

	t.Run("returns error on context timeout", func(t *testing.T) {
		mock := &mockMetadataProvider{
			getMetadataFunc: func(topic *string, allTopics bool, timeoutMs int) (*kafka.Metadata, error) {
				return nil, errors.New("no brokers")
			},
		}

		ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
		defer cancel()

		err := pollBrokers(ctx, mock)

		assert.Error(t, err)
		assert.Equal(t, context.DeadlineExceeded, err)
	})

	t.Run("keeps polling until brokers available", func(t *testing.T) {
		callCount := 0
		mock := &mockMetadataProvider{
			getMetadataFunc: func(topic *string, allTopics bool, timeoutMs int) (*kafka.Metadata, error) {
				callCount++
				if callCount < 3 {
					return nil, errors.New("no brokers")
				}
				return &kafka.Metadata{Brokers: []kafka.BrokerMetadata{{ID: 1}}}, nil
			},
		}

		err := pollBrokers(context.Background(), mock)

		assert.NoError(t, err)
		assert.Equal(t, 3, callCount)
	})

	t.Run("continues polling when metadata returns empty brokers", func(t *testing.T) {
		callCount := 0
		mock := &mockMetadataProvider{
			getMetadataFunc: func(topic *string, allTopics bool, timeoutMs int) (*kafka.Metadata, error) {
				callCount++
				if callCount < 2 {
					return &kafka.Metadata{Brokers: []kafka.BrokerMetadata{}}, nil
				}
				return &kafka.Metadata{Brokers: []kafka.BrokerMetadata{{ID: 1}}}, nil
			},
		}

		err := pollBrokers(context.Background(), mock)

		assert.NoError(t, err)
		assert.Equal(t, 2, callCount)
	})
}
