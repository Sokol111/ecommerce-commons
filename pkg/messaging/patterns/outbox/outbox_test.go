package outbox

import (
	"context"
	"errors"
	"testing"

	"github.com/Sokol111/ecommerce-commons/pkg/core/logger"
	"github.com/Sokol111/ecommerce-commons/pkg/messaging/kafka/events"
	"github.com/confluentinc/confluent-kafka-go/v2/kafka"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/otel/trace"
	"go.uber.org/zap"
)

// mockEvent is a mock implementation of events.Event for testing
type mockEvent struct {
	metadata events.EventMetadata
}

func (m *mockEvent) GetMetadata() *events.EventMetadata {
	return &m.metadata
}

func (m *mockEvent) GetTopic() string {
	return "mock.topic"
}

func (m *mockEvent) GetSchemaName() string {
	return "com.test.MockEvent"
}

func (m *mockEvent) GetSchema() []byte {
	return []byte(`{"type": "record", "name": "MockEvent", "fields": []}`)
}

// mockMetadataPopulator is a mock implementation of events.MetadataPopulator
type mockMetadataPopulator struct {
	populateFunc func(ctx context.Context, event events.Event) string
}

func (m *mockMetadataPopulator) PopulateMetadata(ctx context.Context, event events.Event) string {
	if m.populateFunc != nil {
		return m.populateFunc(ctx, event)
	}
	event.GetMetadata().EventID = "generated-event-id"
	event.GetMetadata().EventType = "MockEvent"
	event.GetMetadata().Source = "test-service"
	return "generated-event-id"
}

// mockSerializer is a mock implementation of serde.Serializer
type mockSerializer struct {
	serializeFunc func(event events.Event) ([]byte, error)
}

func (m *mockSerializer) Serialize(event events.Event) ([]byte, error) {
	if m.serializeFunc != nil {
		return m.serializeFunc(event)
	}
	return []byte("serialized"), nil
}

// mockTracePropagator is a mock implementation of tracePropagator
type mockTracePropagator struct {
	saveTraceContextFunc func(ctx context.Context, headers map[string]string) map[string]string
}

func (m *mockTracePropagator) SaveTraceContext(ctx context.Context, headers map[string]string) map[string]string {
	if m.saveTraceContextFunc != nil {
		return m.saveTraceContextFunc(ctx, headers)
	}
	if headers == nil {
		headers = make(map[string]string)
	}
	headers["traceparent"] = "00-test-trace-id-00"
	return headers
}

func (m *mockTracePropagator) StartKafkaProducerSpan(headers map[string]string, topic, messageID string) (context.Context, trace.Span, []kafka.Header) {
	return context.Background(), noopTestSpan{}, nil
}

// noopTestSpan implements trace.Span for testing
type noopTestSpan struct {
	trace.Span
}

func (n noopTestSpan) End(options ...trace.SpanEndOption) {}

func TestOutbox_Create(t *testing.T) {
	t.Run("successfully creates outbox message", func(t *testing.T) {
		repo := newMockRepository()
		entitiesChan := make(chan *outboxEntity, 10)
		serializer := &mockSerializer{}
		propagator := &mockTracePropagator{}
		metadataPopulator := &mockMetadataPopulator{}
		log := zap.NewNop()

		o := newOutbox(log, repo, entitiesChan, serializer, propagator, metadataPopulator)

		msg := Message{
			Event:   &mockEvent{},
			Key:     "partition-key",
			Headers: map[string]string{"custom": "header"},
		}

		ctx := logger.With(context.Background(), log)
		sendFunc, err := o.Create(ctx, msg)

		require.NoError(t, err)
		assert.NotNil(t, sendFunc)
		assert.Len(t, repo.created, 1)
		assert.Equal(t, "generated-event-id", repo.created[0].ID)
		assert.Equal(t, "partition-key", repo.created[0].Key)
		assert.Equal(t, "mock.topic", repo.created[0].Topic)
	})

	t.Run("serialization error returns error", func(t *testing.T) {
		repo := newMockRepository()
		entitiesChan := make(chan *outboxEntity, 10)
		serializer := &mockSerializer{
			serializeFunc: func(event events.Event) ([]byte, error) {
				return nil, errors.New("serialization failed")
			},
		}
		propagator := &mockTracePropagator{}
		metadataPopulator := &mockMetadataPopulator{}
		log := zap.NewNop()

		o := newOutbox(log, repo, entitiesChan, serializer, propagator, metadataPopulator)

		msg := Message{
			Event: &mockEvent{},
		}

		ctx := logger.With(context.Background(), log)
		sendFunc, err := o.Create(ctx, msg)

		assert.Error(t, err)
		assert.Nil(t, sendFunc)
		assert.Contains(t, err.Error(), "failed to serialize outbox message")
	})

	t.Run("repository error returns error", func(t *testing.T) {
		repo := newMockRepository()
		repo.createErr = errors.New("database error")
		entitiesChan := make(chan *outboxEntity, 10)
		serializer := &mockSerializer{}
		propagator := &mockTracePropagator{}
		metadataPopulator := &mockMetadataPopulator{}
		log := zap.NewNop()

		o := newOutbox(log, repo, entitiesChan, serializer, propagator, metadataPopulator)

		msg := Message{
			Event: &mockEvent{},
		}

		ctx := logger.With(context.Background(), log)
		sendFunc, err := o.Create(ctx, msg)

		assert.Error(t, err)
		assert.Nil(t, sendFunc)
		assert.Contains(t, err.Error(), "failed to create outbox message")
	})

	t.Run("headers are propagated with trace context", func(t *testing.T) {
		repo := newMockRepository()
		entitiesChan := make(chan *outboxEntity, 10)
		serializer := &mockSerializer{}
		propagator := &mockTracePropagator{
			saveTraceContextFunc: func(ctx context.Context, headers map[string]string) map[string]string {
				if headers == nil {
					headers = make(map[string]string)
				}
				headers["traceparent"] = "injected-trace"
				return headers
			},
		}
		metadataPopulator := &mockMetadataPopulator{}
		log := zap.NewNop()

		o := newOutbox(log, repo, entitiesChan, serializer, propagator, metadataPopulator)

		msg := Message{
			Event:   &mockEvent{},
			Headers: map[string]string{"existing": "header"},
		}

		ctx := logger.With(context.Background(), log)
		_, err := o.Create(ctx, msg)

		require.NoError(t, err)
		assert.Contains(t, repo.created[0].Headers, "traceparent")
		assert.Equal(t, "injected-trace", repo.created[0].Headers["traceparent"])
		assert.Equal(t, "header", repo.created[0].Headers["existing"])
	})
}

func TestOutbox_SendFunc(t *testing.T) {
	t.Run("sends entity to channel", func(t *testing.T) {
		repo := newMockRepository()
		entitiesChan := make(chan *outboxEntity, 10)
		serializer := &mockSerializer{}
		propagator := &mockTracePropagator{}
		metadataPopulator := &mockMetadataPopulator{}
		log := zap.NewNop()

		o := newOutbox(log, repo, entitiesChan, serializer, propagator, metadataPopulator)

		msg := Message{
			Event: &mockEvent{},
		}

		ctx := logger.With(context.Background(), log)
		sendFunc, err := o.Create(ctx, msg)
		require.NoError(t, err)

		err = sendFunc(context.Background())
		require.NoError(t, err)

		select {
		case entity := <-entitiesChan:
			assert.Equal(t, "generated-event-id", entity.ID)
		default:
			t.Fatal("expected entity in channel")
		}
	})

	t.Run("returns error on context cancellation", func(t *testing.T) {
		repo := newMockRepository()
		entitiesChan := make(chan *outboxEntity) // unbuffered channel
		serializer := &mockSerializer{}
		propagator := &mockTracePropagator{}
		metadataPopulator := &mockMetadataPopulator{}
		log := zap.NewNop()

		o := newOutbox(log, repo, entitiesChan, serializer, propagator, metadataPopulator)

		msg := Message{
			Event: &mockEvent{},
		}

		ctx := logger.With(context.Background(), log)
		sendFunc, err := o.Create(ctx, msg)
		require.NoError(t, err)

		ctx2, cancel := context.WithCancel(context.Background())
		cancel()

		err = sendFunc(ctx2)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "outbox didn't sent")
	})

	t.Run("returns error when channel is full", func(t *testing.T) {
		repo := newMockRepository()
		entitiesChan := make(chan *outboxEntity) // unbuffered, no receiver
		serializer := &mockSerializer{}
		propagator := &mockTracePropagator{}
		metadataPopulator := &mockMetadataPopulator{}
		log := zap.NewNop()

		o := newOutbox(log, repo, entitiesChan, serializer, propagator, metadataPopulator)

		msg := Message{
			Event: &mockEvent{},
		}

		ctx := logger.With(context.Background(), log)
		sendFunc, err := o.Create(ctx, msg)
		require.NoError(t, err)

		// sendFunc also uses logger from context
		err = sendFunc(ctx)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "entitiesChan is full")
	})
}

func TestOutbox_NilHeaders(t *testing.T) {
	t.Run("handles nil headers gracefully", func(t *testing.T) {
		repo := newMockRepository()
		entitiesChan := make(chan *outboxEntity, 10)
		serializer := &mockSerializer{}
		propagator := &mockTracePropagator{}
		metadataPopulator := &mockMetadataPopulator{}
		log := zap.NewNop()

		o := newOutbox(log, repo, entitiesChan, serializer, propagator, metadataPopulator)

		msg := Message{
			Event:   &mockEvent{},
			Headers: nil,
		}

		ctx := logger.With(context.Background(), log)
		sendFunc, err := o.Create(ctx, msg)

		require.NoError(t, err)
		assert.NotNil(t, sendFunc)
		assert.NotNil(t, repo.created[0].Headers)
	})
}
