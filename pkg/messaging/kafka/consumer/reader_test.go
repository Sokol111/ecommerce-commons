package consumer

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/confluentinc/confluent-kafka-go/v2/kafka"
	"github.com/stretchr/testify/assert"
	"go.uber.org/zap"
)

// mockMessageReader is a test implementation of MessageReader
type mockMessageReader struct {
	readMessageFunc func(timeout time.Duration) (*kafka.Message, error)
}

func (m *mockMessageReader) ReadMessage(timeout time.Duration) (*kafka.Message, error) {
	if m.readMessageFunc != nil {
		return m.readMessageFunc(timeout)
	}
	return nil, kafka.NewError(kafka.ErrTimedOut, "timeout", false)
}

func TestNewReader(t *testing.T) {
	mockConsumer := &mockMessageReader{}
	messagesChan := make(chan *kafka.Message, 10)
	log := zap.NewNop()

	r := newReader(mockConsumer, messagesChan, log)

	assert.NotNil(t, r)
	assert.Equal(t, mockConsumer, r.consumer)
	assert.Equal(t, log, r.log)
	assert.NotNil(t, r.throttler)
}

func TestReader_Run_ContextCancellation(t *testing.T) {
	mockConsumer := &mockMessageReader{
		readMessageFunc: func(timeout time.Duration) (*kafka.Message, error) {
			time.Sleep(10 * time.Millisecond)
			return nil, kafka.NewError(kafka.ErrTimedOut, "timeout", false)
		},
	}

	messagesChan := make(chan *kafka.Message, 10)
	log := zap.NewNop()
	r := newReader(mockConsumer, messagesChan, log)

	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Cancel immediately

	errChan := make(chan error, 1)
	go func() {
		errChan <- r.Run(ctx)
	}()

	select {
	case err := <-errChan:
		assert.NoError(t, err)
	case <-time.After(1 * time.Second):
		t.Fatal("Run did not return in time")
	}
}

func TestReader_Run_ProcessesMessage(t *testing.T) {
	topic := "test-topic"
	testMessage := &kafka.Message{
		TopicPartition: kafka.TopicPartition{Topic: &topic, Partition: 0, Offset: 100},
		Key:            []byte("test-key"),
		Value:          []byte("test-value"),
	}

	callCount := 0
	mockConsumer := &mockMessageReader{
		readMessageFunc: func(timeout time.Duration) (*kafka.Message, error) {
			callCount++
			if callCount == 1 {
				return testMessage, nil
			}
			// After first message, return timeout to allow context check
			return nil, kafka.NewError(kafka.ErrTimedOut, "timeout", false)
		},
	}

	messagesChan := make(chan *kafka.Message, 10)
	log := zap.NewNop()
	r := newReader(mockConsumer, messagesChan, log)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	errChan := make(chan error, 1)
	go func() {
		errChan <- r.Run(ctx)
	}()

	// Wait for message
	select {
	case msg := <-messagesChan:
		assert.Equal(t, testMessage.Value, msg.Value)
		assert.Equal(t, testMessage.Key, msg.Key)
		cancel()
	case <-time.After(1 * time.Second):
		t.Fatal("Did not receive message in time")
	}

	select {
	case err := <-errChan:
		assert.NoError(t, err)
	case <-time.After(1 * time.Second):
		t.Fatal("Run did not return after context cancellation")
	}
}

func TestReader_Run_HandlesTimeout(t *testing.T) {
	timeoutCount := 0
	mockConsumer := &mockMessageReader{
		readMessageFunc: func(timeout time.Duration) (*kafka.Message, error) {
			timeoutCount++
			return nil, kafka.NewError(kafka.ErrTimedOut, "timeout", false)
		},
	}

	messagesChan := make(chan *kafka.Message, 10)
	log := zap.NewNop()
	r := newReader(mockConsumer, messagesChan, log)

	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()

	err := r.Run(ctx)

	assert.NoError(t, err)
	assert.Greater(t, timeoutCount, 0)
}

func TestReader_Run_HandlesFatalError(t *testing.T) {
	fatalErr := kafka.NewError(kafka.ErrFatal, "fatal error", true)
	mockConsumer := &mockMessageReader{
		readMessageFunc: func(timeout time.Duration) (*kafka.Message, error) {
			return nil, fatalErr
		},
	}

	messagesChan := make(chan *kafka.Message, 10)
	log := zap.NewNop()
	r := newReader(mockConsumer, messagesChan, log)

	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
	defer cancel()

	err := r.Run(ctx)

	assert.Error(t, err)
}

func TestReader_Run_HandlesNonKafkaError(t *testing.T) {
	callCount := 0
	mockConsumer := &mockMessageReader{
		readMessageFunc: func(timeout time.Duration) (*kafka.Message, error) {
			callCount++
			if callCount == 1 {
				return nil, errors.New("non-kafka error")
			}
			return nil, kafka.NewError(kafka.ErrTimedOut, "timeout", false)
		},
	}

	messagesChan := make(chan *kafka.Message, 10)
	log := zap.NewNop()
	r := newReader(mockConsumer, messagesChan, log)

	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()

	err := r.Run(ctx)

	assert.NoError(t, err)
	assert.Greater(t, callCount, 1) // Continued after non-kafka error
}

func TestReader_Run_HandlesUnknownTopicError(t *testing.T) {
	callCount := 0
	mockConsumer := &mockMessageReader{
		readMessageFunc: func(timeout time.Duration) (*kafka.Message, error) {
			callCount++
			if callCount <= 2 {
				return nil, kafka.NewError(kafka.ErrUnknownTopicOrPart, "unknown topic", false)
			}
			return nil, kafka.NewError(kafka.ErrTimedOut, "timeout", false)
		},
	}

	messagesChan := make(chan *kafka.Message, 10)
	log := zap.NewNop()
	r := newReader(mockConsumer, messagesChan, log)

	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()

	err := r.Run(ctx)

	assert.NoError(t, err)
	assert.Greater(t, callCount, 2) // Continued after unknown topic error
}

func TestReader_Run_HandlesRetriableError(t *testing.T) {
	callCount := 0
	mockConsumer := &mockMessageReader{
		readMessageFunc: func(timeout time.Duration) (*kafka.Message, error) {
			callCount++
			if callCount <= 2 {
				// Use a retriable error that is not fatal
				return nil, kafka.NewError(kafka.ErrAllBrokersDown, "all brokers down", false)
			}
			return nil, kafka.NewError(kafka.ErrTimedOut, "timeout", false)
		},
	}

	messagesChan := make(chan *kafka.Message, 10)
	log := zap.NewNop()
	r := newReader(mockConsumer, messagesChan, log)

	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()

	err := r.Run(ctx)

	assert.NoError(t, err)
	assert.Greater(t, callCount, 2)
}

func TestReader_Run_HandlesUnknownKafkaError(t *testing.T) {
	callCount := 0
	mockConsumer := &mockMessageReader{
		readMessageFunc: func(timeout time.Duration) (*kafka.Message, error) {
			callCount++
			if callCount == 1 {
				// Non-retriable, non-fatal, non-timeout error
				return nil, kafka.NewError(kafka.ErrBadMsg, "bad message", false)
			}
			return nil, kafka.NewError(kafka.ErrTimedOut, "timeout", false)
		},
	}

	messagesChan := make(chan *kafka.Message, 10)
	log := zap.NewNop()
	r := newReader(mockConsumer, messagesChan, log)

	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()

	err := r.Run(ctx)

	assert.NoError(t, err)
	assert.Greater(t, callCount, 1)
}

func TestReader_Run_ContextCancelledWhileSendingMessage(t *testing.T) {
	topic := "test-topic"
	testMessage := &kafka.Message{
		TopicPartition: kafka.TopicPartition{Topic: &topic, Partition: 0, Offset: 100},
		Value:          []byte("test-value"),
	}

	mockConsumer := &mockMessageReader{
		readMessageFunc: func(timeout time.Duration) (*kafka.Message, error) {
			return testMessage, nil
		},
	}

	// Unbuffered channel - will block
	messagesChan := make(chan *kafka.Message)
	log := zap.NewNop()
	r := newReader(mockConsumer, messagesChan, log)

	ctx, cancel := context.WithCancel(context.Background())

	errChan := make(chan error, 1)
	go func() {
		errChan <- r.Run(ctx)
	}()

	// Give reader time to get message and try to send
	time.Sleep(10 * time.Millisecond)
	cancel()

	select {
	case err := <-errChan:
		assert.NoError(t, err)
	case <-time.After(1 * time.Second):
		t.Fatal("Run did not return after context cancellation")
	}
}
