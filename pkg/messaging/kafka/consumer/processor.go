package consumer

import (
	"context"
	"math"
	"sync"
	"time"

	"github.com/confluentinc/confluent-kafka-go/v2/kafka"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/trace"
	"go.uber.org/zap"
)

type processor struct {
	consumer     *kafka.Consumer
	messagesChan <-chan *kafka.Message
	handler      Handler
	deserializer Deserializer
	subject      string // Topic subject for Schema Registry
	log          *zap.Logger

	ctx        context.Context
	cancelFunc context.CancelFunc
	wg         sync.WaitGroup
}

func newProcessor(
	consumer *kafka.Consumer,
	messagesChan <-chan *kafka.Message,
	handler Handler,
	deserializer Deserializer,
	subject string,
	log *zap.Logger,
) *processor {
	return &processor{
		consumer:     consumer,
		messagesChan: messagesChan,
		handler:      handler,
		deserializer: deserializer,
		subject:      subject,
		log:          log,
	}
}

func (p *processor) start() {
	p.log.Info("starting processor")
	p.ctx, p.cancelFunc = context.WithCancel(context.Background())
	p.wg.Add(1)
	go p.run()
}

func (p *processor) stop() {
	p.log.Info("stopping processor")
	if p.cancelFunc != nil {
		p.cancelFunc()
	}
	p.wg.Wait()

	// Final commit before shutdown
	if _, commitErr := p.consumer.Commit(); commitErr != nil {
		kafkaErr, ok := commitErr.(kafka.Error)
		if !ok || kafkaErr.Code() != kafka.ErrNoOffset {
			p.log.Warn("failed to commit offsets on shutdown", zap.Error(commitErr))
		}
	} else {
		p.log.Debug("final commit successful")
	}

	p.log.Info("processor stopped")
}

func (p *processor) run() {
	defer func() {
		p.log.Info("processor worker stopped")
		p.wg.Done()
	}()

	for {
		select {
		case <-p.ctx.Done():
			return
		case msg := <-p.messagesChan:
			if p.ctx.Err() != nil {
				return
			}
			p.handleMessage(msg)
		}
	}
}

func (p *processor) handleMessage(message *kafka.Message) {
	// Extract trace context from Kafka headers
	ctx := p.extractTraceContext(message)

	// Create span for message processing
	ctx, span := p.startConsumerSpan(ctx, message)
	defer span.End()

	for attempt := 1; ctx.Err() == nil; attempt++ {
		event, err := p.deserializer.Deserialize(p.subject, message.Value)

		if err != nil {
			span.RecordError(err)
			p.log.Error("failed to deserialize message",
				zap.String("key", string(message.Key)),
				zap.Int32("partition", message.TopicPartition.Partition),
				zap.Int32("offset", int32(message.TopicPartition.Offset)),
				zap.Error(err))
			sleep(ctx, backoffDuration(attempt, 10*time.Second))
			continue
		}

		// Now process the deserialized event with trace context
		err = p.handler.Process(ctx, event)

		if err != nil {
			// Handler error - log and retry
			span.RecordError(err)
			p.log.Error("failed to process message",
				zap.String("key", string(message.Key)),
				zap.Int32("partition", message.TopicPartition.Partition),
				zap.Int32("offset", int32(message.TopicPartition.Offset)),
				zap.Error(err))
			sleep(ctx, backoffDuration(attempt, 10*time.Second))
			continue
		}

		// Success - store offset
		span.SetStatus(codes.Ok, "message processed successfully")
		span.AddEvent("message.processed",
			trace.WithAttributes(
				attribute.Int("attempts", attempt),
			),
		)
		p.storeOffset(message)
		return
	}

	// Context cancelled during retries
	span.SetStatus(codes.Error, "context cancelled during message processing")
}

// extractTraceContext extracts OpenTelemetry trace context from Kafka message headers
func (p *processor) extractTraceContext(message *kafka.Message) context.Context {
	ctx := p.ctx

	if len(message.Headers) == 0 {
		return ctx
	}

	// Convert Kafka headers to map for propagator
	headersMap := make(map[string]string)
	for _, header := range message.Headers {
		headersMap[header.Key] = string(header.Value)
	}

	// Extract trace context using OpenTelemetry propagator
	propagator := otel.GetTextMapPropagator()
	carrier := propagation.MapCarrier(headersMap)
	ctx = propagator.Extract(ctx, carrier)

	return ctx
}

func (p *processor) startConsumerSpan(ctx context.Context, message *kafka.Message) (context.Context, trace.Span) {
	tracer := otel.Tracer("kafka-consumer")
	return tracer.Start(ctx, "kafka.consume",
		trace.WithSpanKind(trace.SpanKindConsumer),
		trace.WithAttributes(
			attribute.String("messaging.system", "kafka"),
			attribute.String("messaging.destination", *message.TopicPartition.Topic),
			attribute.Int("messaging.partition", int(message.TopicPartition.Partition)),
			attribute.Int64("messaging.offset", int64(message.TopicPartition.Offset)),
			attribute.String("messaging.message.key", string(message.Key)),
		),
	)
}

func (p *processor) storeOffset(message *kafka.Message) {
	_, err := p.consumer.StoreMessage(message)
	if err != nil {
		p.log.Error("failed to store offset",
			zap.String("key", string(message.Key)),
			zap.Int32("partition", message.TopicPartition.Partition),
			zap.Int32("offset", int32(message.TopicPartition.Offset)),
			zap.Error(err))
	} else {
		p.log.Debug("offset stored",
			zap.Int32("partition", message.TopicPartition.Partition),
			zap.Int32("offset", int32(message.TopicPartition.Offset)))
	}
}

func backoffDuration(attempt int, max time.Duration) time.Duration {
	var backoff time.Duration
	var duration = time.Duration(math.Pow(2, float64(attempt))) * time.Second
	if duration > max {
		backoff = max
	}
	return backoff
}

func sleep(ctx context.Context, d time.Duration) {
	timer := time.NewTimer(d)
	defer timer.Stop()

	select {
	case <-ctx.Done():
		return
	case <-timer.C:
		return
	}
}
