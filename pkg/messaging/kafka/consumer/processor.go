package consumer

import (
	"context"
	"errors"
	"fmt"
	"math"
	"runtime/debug"
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

type panicError struct {
	Panic interface{}
	Stack []byte
}

func (e *panicError) Error() string {
	return fmt.Sprintf("panic: %v", e.Panic)
}

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
		var kafkaErr kafka.Error
		if !errors.As(commitErr, &kafkaErr) || kafkaErr.Code() != kafka.ErrNoOffset {
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
			p.processMessage(msg)
		}
	}
}

func (p *processor) processMessage(message *kafka.Message) {
	// Extract trace context from Kafka headers
	ctx := p.extractTraceContext(message)

	// Create span for message processing
	ctx, span := p.startConsumerSpan(ctx, message)
	defer span.End()

	// Handle message with retry logic
	err := p.handleMessage(ctx, message)

	// Analyze error and decide what to do
	if err == nil {
		// Success - store offset
		span.SetStatus(codes.Ok, "message processed successfully")
		p.storeOffset(message)
		return
	}

	// Check error type
	if errors.Is(err, ErrSkipMessage) {
		// Skip message and commit offset
		span.SetStatus(codes.Ok, "message skipped")
		p.log.Info("skipping message",
			zap.String("key", string(message.Key)),
			zap.Int32("partition", message.TopicPartition.Partition),
			zap.Int32("offset", int32(message.TopicPartition.Offset)))
		p.storeOffset(message)
		return
	}

	if errors.Is(err, ErrPermanent) {
		// Permanent error - send to DLQ (TODO: implement DLQ)
		span.SetStatus(codes.Error, "permanent error - should send to DLQ")
		p.log.Error("permanent error - message should be sent to DLQ",
			zap.String("key", string(message.Key)),
			zap.Int32("partition", message.TopicPartition.Partition),
			zap.Int32("offset", int32(message.TopicPartition.Offset)),
			zap.Error(err))
		// For now, just commit to avoid blocking
		// TODO: Send to DLQ before committing
		p.storeOffset(message)
		return
	}

	// Context cancelled or retryable error exhausted retries
	span.RecordError(err)
	span.SetStatus(codes.Error, "message processing failed")
	p.log.Error("message processing failed after retries",
		zap.String("key", string(message.Key)),
		zap.Int32("partition", message.TopicPartition.Partition),
		zap.Int32("offset", int32(message.TopicPartition.Offset)),
		zap.Error(err))
	// TODO: Send to DLQ before committing
	p.storeOffset(message)
}

func (p *processor) handleMessage(ctx context.Context, message *kafka.Message) error {
	// Deserialize message once before retry loop
	event, err := p.deserializer.Deserialize(p.subject, message.Value)
	if err != nil {
		// Deserialization error is permanent - cannot retry
		return fmt.Errorf("%w: deserialization failed: %v", ErrPermanent, err)
	}

	// Retry processing with backoff
	maxAttempts := 5
	for attempt := 1; attempt <= maxAttempts && ctx.Err() == nil; attempt++ {
		// Process the deserialized event with panic recovery
		err = func() (err error) {
			defer func() {
				if r := recover(); r != nil {
					// Panic is a permanent error - indicates a bug in the code
					err = fmt.Errorf("%w: %v", ErrPermanent, &panicError{
						Panic: r,
						Stack: debug.Stack(),
					})
				}
			}()
			return p.handler.Process(ctx, event)
		}()

		if err == nil {
			// Success
			return nil
		}

		// Check if message should be skipped
		if errors.Is(err, ErrSkipMessage) {
			return err
		}

		// Check if error is permanent
		if errors.Is(err, ErrPermanent) {
			return err
		}

		// Log error
		logFields := []zap.Field{
			zap.String("key", string(message.Key)),
			zap.Int32("partition", message.TopicPartition.Partition),
			zap.Int32("offset", int32(message.TopicPartition.Offset)),
			zap.Int("attempt", attempt),
			zap.Int("maxAttempts", maxAttempts),
		}

		// Add panic-specific fields if it's a panic error
		var panicErr *panicError
		if errors.As(err, &panicErr) {
			logFields = append(logFields,
				zap.Any("panic", panicErr.Panic),
				zap.ByteString("stack", panicErr.Stack),
			)
		} else {
			logFields = append(logFields, zap.Error(err))
		}

		p.log.Error("failed to process message", logFields...)

		// If this is the last attempt, return error
		if attempt >= maxAttempts {
			return fmt.Errorf("max retry attempts reached: %w", err)
		}

		// Sleep with backoff before next retry
		sleep(ctx, backoffDuration(attempt, 10*time.Second))
	}

	// Context cancelled during retries
	if ctx.Err() != nil {
		return ctx.Err()
	}

	return fmt.Errorf("unexpected end of retry loop")
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
	}
}

func backoffDuration(attempt int, max time.Duration) time.Duration {
	duration := time.Duration(math.Pow(2, float64(attempt))) * time.Second
	if duration > max {
		return max
	}
	return duration
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
