package consumer

import (
	"context"
	"fmt"

	"github.com/Sokol111/ecommerce-commons/pkg/messaging/kafka/avro/deserialization"
	"github.com/confluentinc/confluent-kafka-go/v2/kafka"
	"go.uber.org/zap"
)

// MessageEnvelope contains deserialized event with original Kafka message metadata.
type MessageEnvelope struct {
	Event   any
	Message *kafka.Message
}

// messageDeserializer reads raw Kafka messages, deserializes them, and sends to output channel.
type messageDeserializer struct {
	inputChan    <-chan *kafka.Message
	outputChan   chan<- *MessageEnvelope
	deserializer deserialization.Deserializer
	log          *zap.Logger
	tracer       MessageTracer
	dlqHandler   DLQHandler
}

func newMessageDeserializer(
	inputChan <-chan *kafka.Message,
	outputChan chan<- *MessageEnvelope,
	deserializer deserialization.Deserializer,
	log *zap.Logger,
	tracer MessageTracer,
	dlqHandler DLQHandler,
) *messageDeserializer {
	return &messageDeserializer{
		inputChan:    inputChan,
		outputChan:   outputChan,
		deserializer: deserializer,
		log:          log,
		tracer:       tracer,
		dlqHandler:   dlqHandler,
	}
}

func (d *messageDeserializer) Run(ctx context.Context) error {
	for {
		select {
		case <-ctx.Done():
			return nil
		case msg := <-d.inputChan:
			d.deserializeAndSend(ctx, msg)
		}
	}
}

func (d *messageDeserializer) deserializeAndSend(ctx context.Context, message *kafka.Message) {
	event, err := d.deserializer.Deserialize(message.Value)
	if err != nil {
		// Deserialization error is permanent - send to DLQ
		d.log.Error("failed to deserialize message - sending to DLQ",
			zap.String("key", string(message.Key)),
			zap.Int32("partition", message.TopicPartition.Partition),
			zap.Int64("offset", int64(message.TopicPartition.Offset)),
			zap.Error(err))

		tracedCtx := d.tracer.ExtractContext(ctx, message)
		d.dlqHandler.SendToDLQ(tracedCtx, message, fmt.Errorf("deserialization failed: %w", err))
		return
	}

	envelope := &MessageEnvelope{
		Event:   event,
		Message: message,
	}

	select {
	case <-ctx.Done():
		return
	case d.outputChan <- envelope:
	}
}
