package consumer

import (
	"context"
	"fmt"
	"sync"

	"github.com/Sokol111/ecommerce-commons/pkg/messaging/kafka/avro/deserialization"
	"github.com/confluentinc/confluent-kafka-go/v2/kafka"
	"go.uber.org/zap"
)

// MessageEnvelope contains deserialized event with original Kafka message metadata
type MessageEnvelope struct {
	Event   any
	Message *kafka.Message
}

// messageDeserializer reads raw Kafka messages, deserializes them, and sends to output channel
type messageDeserializer struct {
	inputChan    <-chan *kafka.Message
	outputChan   chan<- *MessageEnvelope
	deserializer deserialization.Deserializer
	log          *zap.Logger
	tracer       MessageTracer
	dlqHandler   DLQHandler

	ctx        context.Context
	cancelFunc context.CancelFunc
	wg         sync.WaitGroup
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

func (d *messageDeserializer) start() {
	d.log.Info("starting message deserializer")
	d.ctx, d.cancelFunc = context.WithCancel(context.Background())
	d.wg.Add(1)
	go d.run()
}

func (d *messageDeserializer) stop() {
	d.log.Info("stopping message deserializer")
	if d.cancelFunc != nil {
		d.cancelFunc()
	}
	d.wg.Wait()
	d.log.Info("message deserializer stopped")
}

func (d *messageDeserializer) run() {
	defer func() {
		d.log.Info("message deserializer worker stopped")
		d.wg.Done()
	}()

	for {
		select {
		case <-d.ctx.Done():
			return
		case msg := <-d.inputChan:
			if d.ctx.Err() != nil {
				return
			}
			d.deserializeAndSend(msg)
		}
	}
}

func (d *messageDeserializer) deserializeAndSend(message *kafka.Message) {
	event, err := d.deserializer.Deserialize(message.Value)
	if err != nil {
		// Deserialization error is permanent - send to DLQ
		d.log.Error("failed to deserialize message - sending to DLQ",
			zap.String("key", string(message.Key)),
			zap.Int32("partition", message.TopicPartition.Partition),
			zap.Int64("offset", int64(message.TopicPartition.Offset)),
			zap.Error(err))

		ctx := d.tracer.ExtractContext(d.ctx, message)
		d.dlqHandler.SendToDLQ(ctx, message, fmt.Errorf("deserialization failed: %w", err))
		return
	}

	envelope := &MessageEnvelope{
		Event:   event,
		Message: message,
	}

	select {
	case <-d.ctx.Done():
		return
	case d.outputChan <- envelope:
	}
}
