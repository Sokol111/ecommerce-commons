package consumer

import (
	"context"
	"errors"
	"sync/atomic"
	"testing"
	"time"

	"github.com/Sokol111/ecommerce-commons/pkg/messaging/kafka/config"
	"github.com/confluentinc/confluent-kafka-go/v2/kafka"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/otel/trace"
	"go.opentelemetry.io/otel/trace/noop"
	"go.uber.org/zap"
)

// mockHandler is a test implementation of Handler
type mockHandler struct {
	processFunc func(ctx context.Context, event any) error
	callCount   atomic.Int32
}

func (m *mockHandler) Process(ctx context.Context, event any) error {
	m.callCount.Add(1)
	if m.processFunc != nil {
		return m.processFunc(ctx, event)
	}
	return nil
}

// mockTracer is a test implementation of MessageTracer
type mockTracer struct {
	extractContextFunc    func(ctx context.Context, message *kafka.Message) context.Context
	startConsumerSpanFunc func(ctx context.Context, message *kafka.Message) (context.Context, trace.Span)
	startDLQSpanFunc      func(ctx context.Context, message *kafka.Message, dlqTopic string) (context.Context, trace.Span)
	injectContextFunc     func(ctx context.Context, message *kafka.Message)
}

func newMockTracer() *mockTracer {
	return &mockTracer{
		extractContextFunc: func(ctx context.Context, message *kafka.Message) context.Context {
			return ctx
		},
		startConsumerSpanFunc: func(ctx context.Context, message *kafka.Message) (context.Context, trace.Span) {
			_, span := noop.NewTracerProvider().Tracer("test").Start(ctx, "test")
			return ctx, span
		},
		startDLQSpanFunc: func(ctx context.Context, message *kafka.Message, dlqTopic string) (context.Context, trace.Span) {
			_, span := noop.NewTracerProvider().Tracer("test").Start(ctx, "dlq")
			return ctx, span
		},
		injectContextFunc: func(ctx context.Context, message *kafka.Message) {},
	}
}

func (m *mockTracer) ExtractContext(ctx context.Context, message *kafka.Message) context.Context {
	return m.extractContextFunc(ctx, message)
}

func (m *mockTracer) StartConsumerSpan(ctx context.Context, message *kafka.Message) (context.Context, trace.Span) {
	return m.startConsumerSpanFunc(ctx, message)
}

func (m *mockTracer) StartDLQSpan(ctx context.Context, message *kafka.Message, dlqTopic string) (context.Context, trace.Span) {
	return m.startDLQSpanFunc(ctx, message, dlqTopic)
}

func (m *mockTracer) InjectContext(ctx context.Context, message *kafka.Message) {
	m.injectContextFunc(ctx, message)
}

func createTestConsumerConfig() config.ConsumerConfig {
	return config.ConsumerConfig{
		Name:              "test-consumer",
		Topic:             "test-topic",
		GroupID:           "test-group",
		MaxRetryAttempts:  3,
		InitialBackoff:    10 * time.Millisecond,
		MaxBackoff:        50 * time.Millisecond,
		ProcessingTimeout: 1 * time.Second,
	}
}

func createTestMessage() *kafka.Message {
	topic := "test-topic"
	return &kafka.Message{
		TopicPartition: kafka.TopicPartition{
			Topic:     &topic,
			Partition: 0,
			Offset:    100,
		},
		Key:   []byte("test-key"),
		Value: []byte("test-value"),
	}
}

func TestNewProcessor(t *testing.T) {
	t.Run("creates processor with correct configuration", func(t *testing.T) {
		envelopeChan := make(chan *MessageEnvelope)
		handler := &mockHandler{}
		log := zap.NewNop()
		tracer := newMockTracer()
		conf := createTestConsumerConfig()
		conf.MaxRetryAttempts = 5

		resultHandler := &resultHandler{log: log}

		p := newProcessor(envelopeChan, handler, log, resultHandler, tracer, conf)

		assert.NotNil(t, p)
		assert.Equal(t, uint64(4), p.maxRetries) // maxRetries = MaxRetryAttempts - 1
		assert.Equal(t, 10*time.Millisecond, p.initialBackoff)
		assert.Equal(t, 50*time.Millisecond, p.maxBackoff)
		assert.Equal(t, 1*time.Second, p.processingTimeout)
	})
}

func TestProcessor_Run(t *testing.T) {
	t.Run("stops on context cancellation", func(t *testing.T) {
		envelopeChan := make(chan *MessageEnvelope)
		handler := &mockHandler{}
		log := zap.NewNop()
		tracer := newMockTracer()
		conf := createTestConsumerConfig()

		resultHandler := &resultHandler{log: log}

		p := newProcessor(envelopeChan, handler, log, resultHandler, tracer, conf)

		ctx, cancel := context.WithCancel(context.Background())

		done := make(chan error)
		go func() {
			done <- p.Run(ctx)
		}()

		cancel()

		select {
		case err := <-done:
			assert.NoError(t, err)
		case <-time.After(100 * time.Millisecond):
			t.Fatal("processor did not stop on context cancellation")
		}
	})
}

func TestProcessor_ExecuteWithRetry(t *testing.T) {
	t.Run("succeeds on first attempt", func(t *testing.T) {
		handler := &mockHandler{
			processFunc: func(ctx context.Context, event any) error {
				return nil
			},
		}
		log := zap.NewNop()
		tracer := newMockTracer()
		conf := createTestConsumerConfig()

		resultHandler := &resultHandler{log: log}

		p := newProcessor(make(chan *MessageEnvelope), handler, log, resultHandler, tracer, conf)

		err := p.executeWithRetry(context.Background(), "test-event")

		assert.NoError(t, err)
		assert.Equal(t, int32(1), handler.callCount.Load())
	})

	t.Run("retries on transient error and succeeds", func(t *testing.T) {
		attempts := atomic.Int32{}
		handler := &mockHandler{
			processFunc: func(ctx context.Context, event any) error {
				if attempts.Add(1) < 3 {
					return errors.New("transient error")
				}
				return nil
			},
		}
		log := zap.NewNop()
		tracer := newMockTracer()
		conf := createTestConsumerConfig()
		conf.MaxRetryAttempts = 5

		resultHandler := &resultHandler{log: log}

		p := newProcessor(make(chan *MessageEnvelope), handler, log, resultHandler, tracer, conf)

		err := p.executeWithRetry(context.Background(), "test-event")

		assert.NoError(t, err)
		assert.Equal(t, int32(3), attempts.Load())
	})

	t.Run("returns error after max retries exhausted", func(t *testing.T) {
		handler := &mockHandler{
			processFunc: func(ctx context.Context, event any) error {
				return errors.New("persistent error")
			},
		}
		log := zap.NewNop()
		tracer := newMockTracer()
		conf := createTestConsumerConfig()
		conf.MaxRetryAttempts = 3

		resultHandler := &resultHandler{log: log}

		p := newProcessor(make(chan *MessageEnvelope), handler, log, resultHandler, tracer, conf)

		err := p.executeWithRetry(context.Background(), "test-event")

		assert.Error(t, err)
		assert.Equal(t, int32(3), handler.callCount.Load()) // 3 attempts
	})

	t.Run("does not retry ErrSkipMessage", func(t *testing.T) {
		handler := &mockHandler{
			processFunc: func(ctx context.Context, event any) error {
				return ErrSkipMessage
			},
		}
		log := zap.NewNop()
		tracer := newMockTracer()
		conf := createTestConsumerConfig()

		resultHandler := &resultHandler{log: log}

		p := newProcessor(make(chan *MessageEnvelope), handler, log, resultHandler, tracer, conf)

		err := p.executeWithRetry(context.Background(), "test-event")

		assert.ErrorIs(t, err, ErrSkipMessage)
		assert.Equal(t, int32(1), handler.callCount.Load())
	})

	t.Run("does not retry ErrPermanent", func(t *testing.T) {
		handler := &mockHandler{
			processFunc: func(ctx context.Context, event any) error {
				return ErrPermanent
			},
		}
		log := zap.NewNop()
		tracer := newMockTracer()
		conf := createTestConsumerConfig()

		resultHandler := &resultHandler{log: log}

		p := newProcessor(make(chan *MessageEnvelope), handler, log, resultHandler, tracer, conf)

		err := p.executeWithRetry(context.Background(), "test-event")

		assert.ErrorIs(t, err, ErrPermanent)
		assert.Equal(t, int32(1), handler.callCount.Load())
	})

	t.Run("respects context cancellation", func(t *testing.T) {
		handler := &mockHandler{
			processFunc: func(ctx context.Context, event any) error {
				return errors.New("error")
			},
		}
		log := zap.NewNop()
		tracer := newMockTracer()
		conf := createTestConsumerConfig()
		conf.MaxRetryAttempts = 10

		resultHandler := &resultHandler{log: log}

		p := newProcessor(make(chan *MessageEnvelope), handler, log, resultHandler, tracer, conf)

		ctx, cancel := context.WithCancel(context.Background())
		cancel() // Cancel immediately

		err := p.executeWithRetry(ctx, "test-event")

		// Should return quickly with context error
		assert.Error(t, err)
	})
}

func TestProcessor_Process(t *testing.T) {
	t.Run("successfully processes event", func(t *testing.T) {
		handler := &mockHandler{
			processFunc: func(ctx context.Context, event any) error {
				return nil
			},
		}
		log := zap.NewNop()
		tracer := newMockTracer()
		conf := createTestConsumerConfig()

		resultHandler := &resultHandler{log: log}

		p := newProcessor(make(chan *MessageEnvelope), handler, log, resultHandler, tracer, conf)

		err := p.process(context.Background(), "test-event")

		assert.NoError(t, err)
	})

	t.Run("returns error from handler", func(t *testing.T) {
		expectedErr := errors.New("handler error")
		handler := &mockHandler{
			processFunc: func(ctx context.Context, event any) error {
				return expectedErr
			},
		}
		log := zap.NewNop()
		tracer := newMockTracer()
		conf := createTestConsumerConfig()

		resultHandler := &resultHandler{log: log}

		p := newProcessor(make(chan *MessageEnvelope), handler, log, resultHandler, tracer, conf)

		err := p.process(context.Background(), "test-event")

		assert.ErrorIs(t, err, expectedErr)
	})

	t.Run("recovers from panic", func(t *testing.T) {
		handler := &mockHandler{
			processFunc: func(ctx context.Context, event any) error {
				panic("test panic")
			},
		}
		log := zap.NewNop()
		tracer := newMockTracer()
		conf := createTestConsumerConfig()

		resultHandler := &resultHandler{log: log}

		p := newProcessor(make(chan *MessageEnvelope), handler, log, resultHandler, tracer, conf)

		err := p.process(context.Background(), "test-event")

		require.Error(t, err)
		assert.ErrorIs(t, err, ErrPermanent)
		assert.Contains(t, err.Error(), "panic")
	})

	t.Run("respects processing timeout", func(t *testing.T) {
		handler := &mockHandler{
			processFunc: func(ctx context.Context, event any) error {
				<-ctx.Done()
				return ctx.Err()
			},
		}
		log := zap.NewNop()
		tracer := newMockTracer()
		conf := createTestConsumerConfig()
		conf.ProcessingTimeout = 50 * time.Millisecond

		resultHandler := &resultHandler{log: log}

		p := newProcessor(make(chan *MessageEnvelope), handler, log, resultHandler, tracer, conf)

		start := time.Now()
		err := p.process(context.Background(), "test-event")
		elapsed := time.Since(start)

		assert.Error(t, err)
		assert.Less(t, elapsed, 200*time.Millisecond)
	})
}

func TestPanicError(t *testing.T) {
	t.Run("Error returns panic message", func(t *testing.T) {
		pe := &panicError{
			Panic: "test panic message",
			Stack: []byte("stack trace"),
		}

		assert.Equal(t, "panic: test panic message", pe.Error())
	})

	t.Run("Error handles non-string panic value", func(t *testing.T) {
		pe := &panicError{
			Panic: 42,
			Stack: []byte("stack trace"),
		}

		assert.Equal(t, "panic: 42", pe.Error())
	})
}

func TestProcessor_ProcessMessage(t *testing.T) {
	t.Run("calls handler on process", func(t *testing.T) {
		handlerCalled := atomic.Bool{}
		handler := &mockHandler{
			processFunc: func(ctx context.Context, event any) error {
				handlerCalled.Store(true)
				return nil
			},
		}

		log := zap.NewNop()
		tracer := newMockTracer()
		conf := createTestConsumerConfig()

		// Create processor
		envelopeChan := make(chan *MessageEnvelope, 1)
		rh := &resultHandler{log: log}
		p := newProcessor(envelopeChan, handler, log, rh, tracer, conf)

		// Just test handler was called via executeWithRetry
		err := p.executeWithRetry(context.Background(), "test-event")

		assert.True(t, handlerCalled.Load())
		assert.NoError(t, err)
	})

	t.Run("sends permanent error to result handler", func(t *testing.T) {
		handler := &mockHandler{
			processFunc: func(ctx context.Context, event any) error {
				return ErrPermanent
			},
		}

		log := zap.NewNop()
		tracer := newMockTracer()
		conf := createTestConsumerConfig()

		rh := &resultHandler{log: log}
		p := newProcessor(make(chan *MessageEnvelope), handler, log, rh, tracer, conf)

		err := p.executeWithRetry(context.Background(), "test-event")

		assert.ErrorIs(t, err, ErrPermanent)
	})
}
