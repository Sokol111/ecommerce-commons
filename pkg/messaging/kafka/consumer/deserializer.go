package consumer

import (
	"context"
	"fmt"

	"github.com/Sokol111/ecommerce-commons/pkg/messaging/kafka/serde"
	"github.com/twmb/franz-go/pkg/kgo"
	"go.uber.org/zap"
)

// MessageEnvelope contains deserialized event with original Kafka record metadata.
type MessageEnvelope struct {
	Event  any
	Record *kgo.Record
}

// messageDeserializer reads raw Kafka records, deserializes them, and sends to output channel.
type messageDeserializer struct {
	inputChan    <-chan *kgo.Record
	outputChan   chan<- *MessageEnvelope
	deserializer serde.Deserializer
	log          *zap.Logger
	tracer       MessageTracer
	dlqHandler   DLQHandler
}

func newMessageDeserializer(
	inputChan chan *kgo.Record,
	outputChan chan *MessageEnvelope,
	deserializer serde.Deserializer,
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
		case record := <-d.inputChan:
			d.deserializeAndSend(ctx, record)
		}
	}
}

func (d *messageDeserializer) deserializeAndSend(ctx context.Context, record *kgo.Record) {
	event, err := d.deserializer.Deserialize(record.Value)
	if err != nil {
		// Deserialization error is permanent - send to DLQ
		d.log.Error("failed to deserialize message - sending to DLQ",
			zap.String("key", string(record.Key)),
			zap.Int32("partition", record.Partition),
			zap.Int64("offset", record.Offset),
			zap.Error(err))

		tracedCtx := d.tracer.ExtractContext(ctx, record)
		d.dlqHandler.SendToDLQ(tracedCtx, record, fmt.Errorf("deserialization failed: %w", err))
		return
	}

	envelope := &MessageEnvelope{
		Event:  event,
		Record: record,
	}

	select {
	case <-ctx.Done():
		return
	case d.outputChan <- envelope:
	}
}
