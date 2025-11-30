package producer

import (
	"errors"
	"testing"

	"github.com/confluentinc/confluent-kafka-go/v2/kafka"
	"github.com/stretchr/testify/assert"
	"go.uber.org/zap"
)

// mockKafkaProducer is a mock implementation of kafkaProducer interface for testing.
type mockKafkaProducer struct {
	produceFunc func(msg *kafka.Message, deliveryChan chan kafka.Event) error
}

func (m *mockKafkaProducer) Produce(msg *kafka.Message, deliveryChan chan kafka.Event) error {
	if m.produceFunc != nil {
		return m.produceFunc(msg, deliveryChan)
	}
	return nil
}

func TestNewProducer(t *testing.T) {
	t.Run("creates producer with dependencies", func(t *testing.T) {
		mock := &mockKafkaProducer{}
		log := zap.NewNop()

		p := newProducer(mock, log)

		assert.NotNil(t, p)
	})
}

func TestProducer_Produce(t *testing.T) {
	topic := "test-topic"

	t.Run("successfully produces message", func(t *testing.T) {
		var capturedMessage *kafka.Message
		var capturedChan chan kafka.Event

		mock := &mockKafkaProducer{
			produceFunc: func(msg *kafka.Message, deliveryChan chan kafka.Event) error {
				capturedMessage = msg
				capturedChan = deliveryChan
				return nil
			},
		}
		p := newProducer(mock, zap.NewNop())

		message := &kafka.Message{
			TopicPartition: kafka.TopicPartition{Topic: &topic, Partition: kafka.PartitionAny},
			Key:            []byte("test-key"),
			Value:          []byte("test-value"),
		}
		deliveryChan := make(chan kafka.Event, 1)

		err := p.Produce(message, deliveryChan)

		assert.NoError(t, err)
		assert.Equal(t, message, capturedMessage)
		assert.Equal(t, deliveryChan, capturedChan)
	})

	t.Run("wraps error with topic information", func(t *testing.T) {
		originalErr := errors.New("kafka error")

		mock := &mockKafkaProducer{
			produceFunc: func(msg *kafka.Message, deliveryChan chan kafka.Event) error {
				return originalErr
			},
		}
		p := newProducer(mock, zap.NewNop())

		message := &kafka.Message{
			TopicPartition: kafka.TopicPartition{Topic: &topic, Partition: kafka.PartitionAny},
			Value:          []byte("test"),
		}

		err := p.Produce(message, nil)

		assert.Error(t, err)
		assert.Contains(t, err.Error(), "failed to send message to topic")
		assert.ErrorIs(t, err, originalErr)
	})

	t.Run("handles nil delivery channel", func(t *testing.T) {
		var capturedChan chan kafka.Event

		mock := &mockKafkaProducer{
			produceFunc: func(msg *kafka.Message, deliveryChan chan kafka.Event) error {
				capturedChan = deliveryChan
				return nil
			},
		}
		p := newProducer(mock, zap.NewNop())

		message := &kafka.Message{
			TopicPartition: kafka.TopicPartition{Topic: &topic, Partition: kafka.PartitionAny},
			Value:          []byte("test-value"),
		}

		err := p.Produce(message, nil)

		assert.NoError(t, err)
		assert.Nil(t, capturedChan)
	})

	t.Run("handles message with headers", func(t *testing.T) {
		var capturedMessage *kafka.Message

		mock := &mockKafkaProducer{
			produceFunc: func(msg *kafka.Message, deliveryChan chan kafka.Event) error {
				capturedMessage = msg
				return nil
			},
		}
		p := newProducer(mock, zap.NewNop())

		headers := []kafka.Header{
			{Key: "correlation-id", Value: []byte("123")},
			{Key: "content-type", Value: []byte("application/json")},
		}
		message := &kafka.Message{
			TopicPartition: kafka.TopicPartition{Topic: &topic, Partition: kafka.PartitionAny},
			Key:            []byte("test-key"),
			Value:          []byte("test-value"),
			Headers:        headers,
		}

		err := p.Produce(message, nil)

		assert.NoError(t, err)
		assert.Equal(t, headers, capturedMessage.Headers)
	})

	t.Run("handles empty message value", func(t *testing.T) {
		var capturedMessage *kafka.Message

		mock := &mockKafkaProducer{
			produceFunc: func(msg *kafka.Message, deliveryChan chan kafka.Event) error {
				capturedMessage = msg
				return nil
			},
		}
		p := newProducer(mock, zap.NewNop())

		message := &kafka.Message{
			TopicPartition: kafka.TopicPartition{Topic: &topic, Partition: kafka.PartitionAny},
			Key:            []byte("test-key"),
			Value:          []byte{},
		}

		err := p.Produce(message, nil)

		assert.NoError(t, err)
		assert.Empty(t, capturedMessage.Value)
	})
}
