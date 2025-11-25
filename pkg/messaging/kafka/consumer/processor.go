package consumer

import (
	"context"
	"errors"
	"fmt"
	"sync"

	"github.com/confluentinc/confluent-kafka-go/v2/kafka"
	"go.uber.org/zap"
)

type processor struct {
	consumer      *kafka.Consumer
	messagesChan  <-chan *kafka.Message
	handler       Handler
	deserializer  Deserializer
	log           *zap.Logger
	resultHandler *resultHandler
	retryExecutor RetryExecutor
	tracer        MessageTracer

	ctx        context.Context
	cancelFunc context.CancelFunc
	wg         sync.WaitGroup
}

func newProcessor(
	consumer *kafka.Consumer,
	messagesChan <-chan *kafka.Message,
	handler Handler,
	deserializer Deserializer,
	log *zap.Logger,
	resultHandler *resultHandler,
	retryExecutor RetryExecutor,
	tracer MessageTracer,
) *processor {
	return &processor{
		consumer:      consumer,
		messagesChan:  messagesChan,
		handler:       handler,
		deserializer:  deserializer,
		log:           log,
		resultHandler: resultHandler,
		retryExecutor: retryExecutor,
		tracer:        tracer,
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
	// Витягуємо trace context з Kafka headers
	ctx := p.tracer.ExtractContext(p.ctx, message)

	// Створюємо span для обробки повідомлення
	ctx, span := p.tracer.StartConsumerSpan(ctx, message)
	defer span.End()

	// Обробляємо повідомлення з retry логікою
	err := p.handleMessage(ctx, message)

	// Класифікуємо результат та застосовуємо відповідну стратегію
	p.resultHandler.classifyAndHandle(ctx, err, message, span)
}

func (p *processor) handleMessage(ctx context.Context, message *kafka.Message) error {
	// Десеріалізуємо повідомлення один раз перед retry циклом
	event, err := p.deserializer.Deserialize(message.Value)
	if err != nil {
		// Помилка десеріалізації - перманентна, не можна retry
		return fmt.Errorf("%w: deserialization failed: %v", ErrPermanent, err)
	}

	// Використовуємо RetryExecutor для обробки з повторними спробами
	return p.retryExecutor.Execute(ctx, func(ctx context.Context) error {
		return p.handler.Process(ctx, event)
	})
}
