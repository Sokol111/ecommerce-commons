package consumer

import (
	"context"
	"errors"
	"sync"

	"github.com/confluentinc/confluent-kafka-go/v2/kafka"
	"go.uber.org/zap"
)

type processor struct {
	consumer      *kafka.Consumer
	envelopeChan  <-chan *MessageEnvelope
	handler       Handler
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
	envelopeChan <-chan *MessageEnvelope,
	handler Handler,
	log *zap.Logger,
	resultHandler *resultHandler,
	retryExecutor RetryExecutor,
	tracer MessageTracer,
) *processor {
	return &processor{
		consumer:      consumer,
		envelopeChan:  envelopeChan,
		handler:       handler,
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
		case envelope := <-p.envelopeChan:
			if p.ctx.Err() != nil {
				return
			}
			p.processMessage(envelope)
		}
	}
}

func (p *processor) processMessage(envelope *MessageEnvelope) {
	// Витягуємо trace context з Kafka headers
	ctx := p.tracer.ExtractContext(p.ctx, envelope.Message)

	// Створюємо span для обробки повідомлення
	ctx, span := p.tracer.StartConsumerSpan(ctx, envelope.Message)
	defer span.End()

	// Обробляємо повідомлення з retry логікою
	err := p.retryExecutor.Execute(ctx, func(ctx context.Context) error {
		return p.handler.Process(ctx, envelope.Event)
	})

	// Класифікуємо результат та застосовуємо відповідну стратегію
	p.resultHandler.handle(ctx, err, envelope.Message, span)
}
