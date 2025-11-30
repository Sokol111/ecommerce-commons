package consumer

import (
	"context"
	"errors"
	"sync/atomic"
	"testing"
	"time"

	"github.com/confluentinc/confluent-kafka-go/v2/kafka"
	"github.com/stretchr/testify/assert"
	"go.uber.org/zap"
)

// mockDeserializer is a test implementation of deserialization.Deserializer
type mockDeserializer struct {
	deserializeFunc func(data []byte) (interface{}, error)
}

func (m *mockDeserializer) Deserialize(data []byte) (interface{}, error) {
	if m.deserializeFunc != nil {
		return m.deserializeFunc(data)
	}
	return nil, nil
}

// mockDLQHandler is a test implementation of DLQHandler
type mockDLQHandler struct {
	sendToDLQFunc func(ctx context.Context, message *kafka.Message, processingErr error)
	callCount     atomic.Int32
}

func (m *mockDLQHandler) SendToDLQ(ctx context.Context, message *kafka.Message, processingErr error) {
	m.callCount.Add(1)
	if m.sendToDLQFunc != nil {
		m.sendToDLQFunc(ctx, message, processingErr)
	}
}

func TestNewMessageDeserializer(t *testing.T) {
	t.Run("creates deserializer with correct fields", func(t *testing.T) {
		inputChan := make(chan *kafka.Message)
		outputChan := make(chan *MessageEnvelope)
		deserializer := &mockDeserializer{}
		log := zap.NewNop()
		tracer := newMockTracer()
		dlqHandler := &mockDLQHandler{}

		d := newMessageDeserializer(inputChan, outputChan, deserializer, log, tracer, dlqHandler)

		assert.NotNil(t, d)
	})
}

func TestMessageDeserializer_Run(t *testing.T) {
	t.Run("deserializes messages and sends to output", func(t *testing.T) {
		inputChan := make(chan *kafka.Message, 1)
		outputChan := make(chan *MessageEnvelope, 1)
		expectedEvent := map[string]string{"key": "value"}
		deserializer := &mockDeserializer{
			deserializeFunc: func(data []byte) (interface{}, error) {
				return expectedEvent, nil
			},
		}
		log := zap.NewNop()
		tracer := newMockTracer()
		dlqHandler := &mockDLQHandler{}

		d := newMessageDeserializer(inputChan, outputChan, deserializer, log, tracer, dlqHandler)

		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()

		go func() {
			_ = d.Run(ctx)
		}()

		msg := createTestMessage()
		inputChan <- msg

		select {
		case envelope := <-outputChan:
			assert.Equal(t, expectedEvent, envelope.Event)
			assert.Equal(t, msg, envelope.Message)
		case <-time.After(100 * time.Millisecond):
			t.Fatal("envelope was not sent to output channel")
		}

		cancel()
	})

	t.Run("sends to DLQ on deserialization error", func(t *testing.T) {
		inputChan := make(chan *kafka.Message, 1)
		outputChan := make(chan *MessageEnvelope, 1)
		deserializer := &mockDeserializer{
			deserializeFunc: func(data []byte) (interface{}, error) {
				return nil, errors.New("deserialization failed")
			},
		}
		log := zap.NewNop()
		tracer := newMockTracer()

		dlqCalled := make(chan *kafka.Message, 1)
		dlqHandler := &mockDLQHandler{
			sendToDLQFunc: func(ctx context.Context, message *kafka.Message, processingErr error) {
				dlqCalled <- message
			},
		}

		d := newMessageDeserializer(inputChan, outputChan, deserializer, log, tracer, dlqHandler)

		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()

		go func() {
			_ = d.Run(ctx)
		}()

		msg := createTestMessage()
		inputChan <- msg

		select {
		case sentMsg := <-dlqCalled:
			assert.Equal(t, msg, sentMsg)
		case <-time.After(100 * time.Millisecond):
			t.Fatal("DLQ handler was not called")
		}

		// Output channel should be empty
		select {
		case <-outputChan:
			t.Fatal("envelope should not be sent on deserialization error")
		default:
			// Expected
		}

		cancel()
	})

	t.Run("stops on context cancellation", func(t *testing.T) {
		inputChan := make(chan *kafka.Message)
		outputChan := make(chan *MessageEnvelope)
		deserializer := &mockDeserializer{}
		log := zap.NewNop()
		tracer := newMockTracer()
		dlqHandler := &mockDLQHandler{}

		d := newMessageDeserializer(inputChan, outputChan, deserializer, log, tracer, dlqHandler)

		ctx, cancel := context.WithCancel(context.Background())

		done := make(chan error)
		go func() {
			done <- d.Run(ctx)
		}()

		cancel()

		select {
		case err := <-done:
			assert.NoError(t, err)
		case <-time.After(100 * time.Millisecond):
			t.Fatal("deserializer did not stop on context cancellation")
		}
	})

	t.Run("handles multiple messages", func(t *testing.T) {
		inputChan := make(chan *kafka.Message, 3)
		outputChan := make(chan *MessageEnvelope, 3)
		counter := atomic.Int32{}
		deserializer := &mockDeserializer{
			deserializeFunc: func(data []byte) (interface{}, error) {
				return counter.Add(1), nil
			},
		}
		log := zap.NewNop()
		tracer := newMockTracer()
		dlqHandler := &mockDLQHandler{}

		d := newMessageDeserializer(inputChan, outputChan, deserializer, log, tracer, dlqHandler)

		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()

		go func() {
			_ = d.Run(ctx)
		}()

		// Send 3 messages
		for i := 0; i < 3; i++ {
			inputChan <- createTestMessage()
		}

		// Receive 3 envelopes
		for i := 0; i < 3; i++ {
			select {
			case envelope := <-outputChan:
				assert.NotNil(t, envelope.Event)
			case <-time.After(100 * time.Millisecond):
				t.Fatalf("envelope %d was not received", i+1)
			}
		}

		cancel()
	})
}

func TestMessageDeserializer_DeserializeAndSend(t *testing.T) {
	t.Run("extracts trace context from message", func(t *testing.T) {
		inputChan := make(chan *kafka.Message)
		outputChan := make(chan *MessageEnvelope, 1)
		deserializer := &mockDeserializer{
			deserializeFunc: func(data []byte) (interface{}, error) {
				return nil, errors.New("error")
			},
		}
		log := zap.NewNop()

		extractCalled := atomic.Bool{}
		tracer := newMockTracer()
		tracer.extractContextFunc = func(ctx context.Context, message *kafka.Message) context.Context {
			extractCalled.Store(true)
			return ctx
		}

		dlqHandler := &mockDLQHandler{}

		d := newMessageDeserializer(inputChan, outputChan, deserializer, log, tracer, dlqHandler)

		d.deserializeAndSend(context.Background(), createTestMessage())

		assert.True(t, extractCalled.Load())
	})

	t.Run("respects context cancellation during send", func(t *testing.T) {
		inputChan := make(chan *kafka.Message)
		outputChan := make(chan *MessageEnvelope) // unbuffered - will block
		deserializer := &mockDeserializer{
			deserializeFunc: func(data []byte) (interface{}, error) {
				return "event", nil
			},
		}
		log := zap.NewNop()
		tracer := newMockTracer()
		dlqHandler := &mockDLQHandler{}

		d := newMessageDeserializer(inputChan, outputChan, deserializer, log, tracer, dlqHandler)

		ctx, cancel := context.WithCancel(context.Background())
		cancel() // Cancel before sending

		// Should not block
		done := make(chan struct{})
		go func() {
			d.deserializeAndSend(ctx, createTestMessage())
			close(done)
		}()

		select {
		case <-done:
			// Success
		case <-time.After(100 * time.Millisecond):
			t.Fatal("deserializeAndSend blocked on cancelled context")
		}
	})
}
