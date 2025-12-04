package outbox

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/confluentinc/confluent-kafka-go/v2/kafka"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/otel/trace"
	"go.uber.org/zap"
)

// mockProducer is a mock implementation of producer.Producer
type mockProducer struct {
	mu          sync.Mutex
	produceFunc func(msg *kafka.Message, deliveryChan chan kafka.Event) error
	messages    []*kafka.Message
}

func (m *mockProducer) Produce(msg *kafka.Message, deliveryChan chan kafka.Event) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.messages = append(m.messages, msg)
	if m.produceFunc != nil {
		return m.produceFunc(msg, deliveryChan)
	}
	return nil
}

func (m *mockProducer) GetMessages() []*kafka.Message {
	m.mu.Lock()
	defer m.mu.Unlock()
	result := make([]*kafka.Message, len(m.messages))
	copy(result, m.messages)
	return result
}

// mockSenderTracePropagator is a mock for tracePropagator in sender tests
type mockSenderTracePropagator struct {
	kafkaHeaders []kafka.Header
}

func (m *mockSenderTracePropagator) SaveTraceContext(ctx context.Context, headers map[string]string) map[string]string {
	return headers
}

func (m *mockSenderTracePropagator) StartKafkaProducerSpan(headers map[string]string, topic, messageID string) (context.Context, trace.Span, []kafka.Header) {
	if m.kafkaHeaders != nil {
		return context.Background(), senderNoopSpan{}, m.kafkaHeaders
	}
	return context.Background(), senderNoopSpan{}, []kafka.Header{
		{Key: "traceparent", Value: []byte("test-trace")},
	}
}

// senderNoopSpan implements trace.Span for testing
type senderNoopSpan struct {
	trace.Span
}

func (n senderNoopSpan) End(options ...trace.SpanEndOption) {}

func TestSender_Run(t *testing.T) {
	t.Run("sends entity to kafka", func(t *testing.T) {
		producer := &mockProducer{}
		entitiesChan := make(chan *outboxEntity, 10)
		deliveryChan := make(chan kafka.Event, 10)
		propagator := &mockSenderTracePropagator{}

		s := newSender(producer, entitiesChan, deliveryChan, zap.NewNop(), propagator)

		ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
		defer cancel()

		var wg sync.WaitGroup
		wg.Add(1)
		go func() {
			defer wg.Done()
			_ = s.Run(ctx)
		}()

		entity := &outboxEntity{
			ID:      "test-id",
			Payload: []byte("test-payload"),
			Key:     "test-key",
			Topic:   "test-topic",
			Headers: map[string]string{"header": "value"},
		}
		entitiesChan <- entity

		// Wait a bit for the message to be processed
		time.Sleep(50 * time.Millisecond)
		cancel()
		wg.Wait()

		messages := producer.GetMessages()
		require.Len(t, messages, 1)
		assert.Equal(t, []byte("test-payload"), messages[0].Value)
		assert.Equal(t, []byte("test-key"), messages[0].Key)
		assert.Equal(t, "test-topic", *messages[0].TopicPartition.Topic)
		assert.Equal(t, "test-id", messages[0].Opaque)
	})

	t.Run("returns nil when context is cancelled", func(t *testing.T) {
		producer := &mockProducer{}
		entitiesChan := make(chan *outboxEntity, 10)
		deliveryChan := make(chan kafka.Event, 10)
		propagator := &mockSenderTracePropagator{}

		s := newSender(producer, entitiesChan, deliveryChan, zap.NewNop(), propagator)

		ctx, cancel := context.WithCancel(context.Background())
		cancel()

		err := s.Run(ctx)

		assert.NoError(t, err)
	})

	t.Run("continues after producer error", func(t *testing.T) {
		callCount := 0
		producer := &mockProducer{
			produceFunc: func(msg *kafka.Message, deliveryChan chan kafka.Event) error {
				callCount++
				if callCount == 1 {
					return errors.New("producer error")
				}
				return nil
			},
		}
		entitiesChan := make(chan *outboxEntity, 10)
		deliveryChan := make(chan kafka.Event, 10)
		propagator := &mockSenderTracePropagator{}

		s := newSender(producer, entitiesChan, deliveryChan, zap.NewNop(), propagator)

		ctx, cancel := context.WithTimeout(context.Background(), 200*time.Millisecond)
		defer cancel()

		var wg sync.WaitGroup
		wg.Add(1)
		go func() {
			defer wg.Done()
			_ = s.Run(ctx)
		}()

		// Send two entities
		entitiesChan <- &outboxEntity{ID: "entity-1", Topic: "topic", Payload: []byte("p1")}
		entitiesChan <- &outboxEntity{ID: "entity-2", Topic: "topic", Payload: []byte("p2")}

		time.Sleep(100 * time.Millisecond)
		cancel()
		wg.Wait()

		// Both messages should have been attempted
		assert.GreaterOrEqual(t, callCount, 2)
	})

	t.Run("sends multiple entities", func(t *testing.T) {
		producer := &mockProducer{}
		entitiesChan := make(chan *outboxEntity, 10)
		deliveryChan := make(chan kafka.Event, 10)
		propagator := &mockSenderTracePropagator{}

		s := newSender(producer, entitiesChan, deliveryChan, zap.NewNop(), propagator)

		ctx, cancel := context.WithTimeout(context.Background(), 200*time.Millisecond)
		defer cancel()

		var wg sync.WaitGroup
		wg.Add(1)
		go func() {
			defer wg.Done()
			_ = s.Run(ctx)
		}()

		for i := 0; i < 5; i++ {
			entitiesChan <- &outboxEntity{
				ID:      string(rune('a' + i)),
				Topic:   "test-topic",
				Payload: []byte("payload"),
			}
		}

		time.Sleep(100 * time.Millisecond)
		cancel()
		wg.Wait()

		messages := producer.GetMessages()
		assert.Len(t, messages, 5)
	})

	t.Run("includes kafka headers from trace propagator", func(t *testing.T) {
		producer := &mockProducer{}
		entitiesChan := make(chan *outboxEntity, 10)
		deliveryChan := make(chan kafka.Event, 10)
		propagator := &mockSenderTracePropagator{
			kafkaHeaders: []kafka.Header{
				{Key: "custom-trace", Value: []byte("custom-value")},
			},
		}

		s := newSender(producer, entitiesChan, deliveryChan, zap.NewNop(), propagator)

		ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
		defer cancel()

		var wg sync.WaitGroup
		wg.Add(1)
		go func() {
			defer wg.Done()
			_ = s.Run(ctx)
		}()

		entitiesChan <- &outboxEntity{
			ID:      "test-id",
			Topic:   "test-topic",
			Payload: []byte("payload"),
		}

		time.Sleep(50 * time.Millisecond)
		cancel()
		wg.Wait()

		messages := producer.GetMessages()
		require.Len(t, messages, 1)
		require.Len(t, messages[0].Headers, 1)
		assert.Equal(t, "custom-trace", messages[0].Headers[0].Key)
		assert.Equal(t, []byte("custom-value"), messages[0].Headers[0].Value)
	})
}

func TestNewSender(t *testing.T) {
	t.Run("creates sender with dependencies", func(t *testing.T) {
		producer := &mockProducer{}
		entitiesChan := make(chan *outboxEntity)
		deliveryChan := make(chan kafka.Event)
		log := zap.NewNop()
		propagator := &mockSenderTracePropagator{}

		s := newSender(producer, entitiesChan, deliveryChan, log, propagator)

		assert.NotNil(t, s)
		assert.Equal(t, producer, s.producer)
		assert.NotNil(t, s.entitiesChan)
		assert.NotNil(t, s.deliveryChan)
		assert.Equal(t, log, s.logger)
	})
}
