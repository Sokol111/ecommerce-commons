package outbox

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/twmb/franz-go/pkg/kgo"
	"go.opentelemetry.io/otel/trace"
	"go.uber.org/zap"
)

// mockProducer is a mock implementation of producer.Producer
type mockProducer struct {
	mu          sync.Mutex
	produceFunc func(ctx context.Context, record *kgo.Record, promise func(*kgo.Record, error))
	records     []*kgo.Record
}

func (m *mockProducer) Produce(ctx context.Context, record *kgo.Record, promise func(*kgo.Record, error)) {
	m.mu.Lock()
	m.records = append(m.records, record)
	fn := m.produceFunc
	m.mu.Unlock()
	if fn != nil {
		fn(ctx, record, promise)
	} else if promise != nil {
		promise(record, nil)
	}
}

func (m *mockProducer) GetRecords() []*kgo.Record {
	m.mu.Lock()
	defer m.mu.Unlock()
	result := make([]*kgo.Record, len(m.records))
	copy(result, m.records)
	return result
}

// mockSenderTracePropagator is a mock for tracePropagator in sender tests
type mockSenderTracePropagator struct {
	kafkaHeaders []kgo.RecordHeader
}

func (m *mockSenderTracePropagator) SaveTraceContext(ctx context.Context, headers map[string]string) map[string]string {
	return headers
}

func (m *mockSenderTracePropagator) StartKafkaProducerSpan(headers map[string]string, topic, messageID string) (context.Context, trace.Span, []kgo.RecordHeader) {
	if m.kafkaHeaders != nil {
		return context.Background(), senderNoopSpan{}, m.kafkaHeaders
	}
	return context.Background(), senderNoopSpan{}, []kgo.RecordHeader{
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
		confirmChan := make(chan confirmResult, 10)
		propagator := &mockSenderTracePropagator{}

		s := newSender(producer, entitiesChan, confirmChan, zap.NewNop(), propagator)

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

		records := producer.GetRecords()
		require.Len(t, records, 1)
		assert.Equal(t, []byte("test-payload"), records[0].Value)
		assert.Equal(t, []byte("test-key"), records[0].Key)
		assert.Equal(t, "test-topic", records[0].Topic)

		// Verify confirm result was sent
		select {
		case result := <-confirmChan:
			assert.Equal(t, "test-id", result.ID)
			assert.NoError(t, result.Err)
		default:
			t.Fatal("confirm result was not sent")
		}
	})

	t.Run("returns nil when context is cancelled", func(t *testing.T) {
		producer := &mockProducer{}
		entitiesChan := make(chan *outboxEntity, 10)
		confirmChan := make(chan confirmResult, 10)
		propagator := &mockSenderTracePropagator{}

		s := newSender(producer, entitiesChan, confirmChan, zap.NewNop(), propagator)

		ctx, cancel := context.WithCancel(context.Background())
		cancel()

		err := s.Run(ctx)

		assert.NoError(t, err)
	})

	t.Run("sends multiple entities", func(t *testing.T) {
		producer := &mockProducer{}
		entitiesChan := make(chan *outboxEntity, 10)
		confirmChan := make(chan confirmResult, 10)
		propagator := &mockSenderTracePropagator{}

		s := newSender(producer, entitiesChan, confirmChan, zap.NewNop(), propagator)

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

		records := producer.GetRecords()
		assert.Len(t, records, 5)
	})

	t.Run("includes kafka headers from trace propagator", func(t *testing.T) {
		producer := &mockProducer{}
		entitiesChan := make(chan *outboxEntity, 10)
		confirmChan := make(chan confirmResult, 10)
		propagator := &mockSenderTracePropagator{
			kafkaHeaders: []kgo.RecordHeader{
				{Key: "custom-trace", Value: []byte("custom-value")},
			},
		}

		s := newSender(producer, entitiesChan, confirmChan, zap.NewNop(), propagator)

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

		records := producer.GetRecords()
		require.Len(t, records, 1)
		require.Len(t, records[0].Headers, 1)
		assert.Equal(t, "custom-trace", records[0].Headers[0].Key)
		assert.Equal(t, []byte("custom-value"), records[0].Headers[0].Value)
	})
}

func TestNewSender(t *testing.T) {
	t.Run("creates sender with dependencies", func(t *testing.T) {
		producer := &mockProducer{}
		entitiesChan := make(chan *outboxEntity)
		confirmChan := make(chan confirmResult)
		log := zap.NewNop()
		propagator := &mockSenderTracePropagator{}

		s := newSender(producer, entitiesChan, confirmChan, log, propagator)

		assert.NotNil(t, s)
		assert.Equal(t, producer, s.producer)
		assert.NotNil(t, s.entitiesChan)
		assert.NotNil(t, s.confirmChan)
		assert.Equal(t, log, s.logger)
	})
}
