package outbox

import (
	"context"
	"fmt"

	"github.com/Sokol111/ecommerce-commons/pkg/messaging/kafka/producer"
	"github.com/confluentinc/confluent-kafka-go/v2/kafka"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/trace"
	"go.uber.org/zap"
)

type sender struct {
	producer     producer.Producer
	entitiesChan <-chan *outboxEntity
	deliveryChan chan kafka.Event
	logger       *zap.Logger
}

func newSender(
	producer producer.Producer,
	entitiesChan <-chan *outboxEntity,
	deliveryChan chan kafka.Event,
	logger *zap.Logger,
) *sender {
	return &sender{
		producer:     producer,
		entitiesChan: entitiesChan,
		deliveryChan: deliveryChan,
		logger:       logger,
	}
}

func (s *sender) Run(ctx context.Context) error {
	defer s.logger.Info("sender worker stopped")

	for {
		select {
		case <-ctx.Done():
			return nil
		default:
		}

		select {
		case <-ctx.Done():
			return nil
		case entity := <-s.entitiesChan:
			if err := s.send(entity); err != nil {
				s.logger.Error("failed to send outbox message",
					zap.String("id", entity.ID),
					zap.Error(err))
				continue
			}
			s.logger.Debug("outbox sent to kafka", zap.String("id", entity.ID))
		}
	}
}

func (s *sender) send(entity *outboxEntity) error {
	ctx := s.extractTraceContext(entity)
	ctx, span := s.startProducerSpan(ctx, entity)
	defer span.End()
	return s.produceToKafka(ctx, entity)
}

func (s *sender) extractTraceContext(entity *outboxEntity) context.Context {
	ctx := context.Background()
	if len(entity.Headers) > 0 {
		propagator := otel.GetTextMapPropagator()
		carrier := propagation.MapCarrier(entity.Headers)
		ctx = propagator.Extract(ctx, carrier)
	}
	return ctx
}

func (s *sender) startProducerSpan(ctx context.Context, entity *outboxEntity) (context.Context, trace.Span) {
	tracer := otel.Tracer("kafka.producer")
	return tracer.Start(ctx, "kafka.produce.buffer",
		trace.WithSpanKind(trace.SpanKindProducer),
		trace.WithAttributes(
			attribute.String("messaging.system", "kafka"),
			attribute.String("messaging.destination", entity.Topic),
			attribute.String("messaging.message.id", entity.ID),
		),
	)
}

func (s *sender) produceToKafka(ctx context.Context, entity *outboxEntity) error {
	// Update headers with the current span context (child span)
	updatedHeaders := s.injectTraceContext(ctx, entity.Headers)

	// Convert to Kafka headers
	var kafkaHeaders []kafka.Header
	for key, value := range updatedHeaders {
		kafkaHeaders = append(kafkaHeaders, kafka.Header{
			Key:   key,
			Value: []byte(value),
		})
	}

	// Produce message to Kafka
	err := s.producer.Produce(&kafka.Message{
		TopicPartition: kafka.TopicPartition{Topic: &entity.Topic, Partition: kafka.PartitionAny},
		Opaque:         entity.ID,
		Value:          entity.Payload, // Already serialized: [0x00][schema_id][avro_data]
		Key:            []byte(entity.Key),
		Headers:        kafkaHeaders,
	}, s.deliveryChan)

	if err != nil {
		return fmt.Errorf("failed to send outbox message with id %v: %w", entity.ID, err)
	}

	return nil
}

func (s *sender) injectTraceContext(ctx context.Context, headers map[string]string) map[string]string {
	updatedHeaders := make(map[string]string)
	for k, v := range headers {
		updatedHeaders[k] = v
	}
	propagator := otel.GetTextMapPropagator()
	carrier := propagation.MapCarrier(updatedHeaders)
	propagator.Inject(ctx, carrier)
	return updatedHeaders
}
