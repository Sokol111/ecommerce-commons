package consumer

import (
	"context"
	"errors"
	"fmt"
	"runtime/debug"
	"time"

	"github.com/Sokol111/ecommerce-commons/pkg/messaging/kafka/config"
	"github.com/cenkalti/backoff/v4"
	"go.uber.org/zap"
)

// PanicError represents an error that occurred due to panic
type panicError struct {
	Panic interface{}
	Stack []byte
}

func (e *panicError) Error() string {
	return fmt.Sprintf("panic: %v", e.Panic)
}

type processor struct {
	envelopeChan  <-chan *MessageEnvelope
	handler       Handler
	log           *zap.Logger
	resultHandler *resultHandler
	tracer        MessageTracer

	// Retry configuration
	maxRetries        uint64
	initialBackoff    time.Duration
	maxBackoff        time.Duration
	processingTimeout time.Duration
}

func newProcessor(
	envelopeChan <-chan *MessageEnvelope,
	handler Handler,
	log *zap.Logger,
	resultHandler *resultHandler,
	tracer MessageTracer,
	consumerConf config.ConsumerConfig,
) *processor {
	return &processor{
		envelopeChan:      envelopeChan,
		handler:           handler,
		log:               log,
		resultHandler:     resultHandler,
		tracer:            tracer,
		maxRetries:        uint64(consumerConf.MaxRetryAttempts - 1), // backoff counts retries, not attempts
		initialBackoff:    consumerConf.InitialBackoff,
		maxBackoff:        consumerConf.MaxBackoff,
		processingTimeout: consumerConf.ProcessingTimeout,
	}
}

func (p *processor) run(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			return
		case envelope := <-p.envelopeChan:
			if ctx.Err() != nil {
				return
			}
			p.processMessage(ctx, envelope)
		}
	}
}

func (p *processor) processMessage(ctx context.Context, envelope *MessageEnvelope) {
	// Витягуємо trace context з Kafka headers
	ctx = p.tracer.ExtractContext(ctx, envelope.Message)

	// Створюємо span для обробки повідомлення
	ctx, span := p.tracer.StartConsumerSpan(ctx, envelope.Message)
	defer span.End()

	// Обробляємо повідомлення з retry логікою
	err := p.executeWithRetry(ctx, envelope.Event)

	// Класифікуємо результат та застосовуємо відповідну стратегію
	p.resultHandler.handle(ctx, err, envelope.Message, span)
}

// executeWithRetry executes the handler with exponential backoff retry logic
func (p *processor) executeWithRetry(ctx context.Context, event any) error {
	expBackoff := backoff.NewExponentialBackOff(
		backoff.WithInitialInterval(p.initialBackoff),
		backoff.WithMaxInterval(p.maxBackoff),
		backoff.WithMaxElapsedTime(0), // No time limit, use max retries instead
	)

	b := backoff.WithContext(
		backoff.WithMaxRetries(expBackoff, p.maxRetries),
		ctx,
	)

	var attempt uint64
	return backoff.Retry(func() error {
		attempt++
		err := p.process(ctx, event)

		if err == nil {
			return nil
		}

		// Skip and permanent errors should not be retried
		if errors.Is(err, ErrSkipMessage) || errors.Is(err, ErrPermanent) {
			return backoff.Permanent(err)
		}

		// Log retriable error: debug for intermediate attempts, error for the last one
		logFields := []zap.Field{
			zap.Uint64("attempt", attempt),
			zap.Uint64("maxAttempts", p.maxRetries+1),
			zap.Error(err),
		}

		if attempt > p.maxRetries {
			p.log.Error("failed to process message", logFields...)
		} else {
			p.log.Debug("failed to process message, retrying", logFields...)
		}

		return err
	}, b)
}

// safeProcess executes the handler with panic recovery
func (p *processor) process(ctx context.Context, event any) (err error) {
	// Apply processing timeout to prevent hanging
	ctx, cancel := context.WithTimeout(ctx, p.processingTimeout)
	defer cancel()

	defer func() {
		if rec := recover(); rec != nil {
			err = fmt.Errorf("%w: %v", ErrPermanent, &panicError{
				Panic: rec,
				Stack: debug.Stack(),
			})
		}
	}()

	return p.handler.Process(ctx, event)
}
