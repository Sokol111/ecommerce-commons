package consumer

import (
	"context"
	"errors"
	"math"
	"sync"
	"time"

	"github.com/confluentinc/confluent-kafka-go/v2/kafka"
	"go.uber.org/zap"
)

type processor struct {
	consumer     *kafka.Consumer
	messagesChan <-chan *kafka.Message
	handler      Handler
	deserializer Deserializer
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
	log *zap.Logger,
) *processor {
	return &processor{
		consumer:     consumer,
		messagesChan: messagesChan,
		handler:      handler,
		deserializer: deserializer,
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
	for attempt := 1; p.ctx.Err() == nil; attempt++ {
		// First, deserialize the message using the provided deserializer
		event, err := p.deserializer(message.Value, message.Headers)

		if err != nil {
			// If deserializer wants to skip this message, commit offset and return
			if errors.Is(err, ErrSkipMessage) {
				p.log.Debug("message skipped by deserializer",
					zap.String("key", string(message.Key)),
					zap.Int32("partition", message.TopicPartition.Partition),
					zap.Int32("offset", int32(message.TopicPartition.Offset)))
				p.storeOffset(message)
				return
			}

			// Deserialization error - log and retry
			p.log.Error("failed to deserialize message",
				zap.String("key", string(message.Key)),
				zap.Int32("partition", message.TopicPartition.Partition),
				zap.Int32("offset", int32(message.TopicPartition.Offset)),
				zap.Error(err))
			sleep(p.ctx, backoffDuration(attempt, 10*time.Second))
			continue
		}

		// Now process the deserialized event
		err = p.handler.Process(p.ctx, event)

		if err != nil {
			// Handler error - log and retry
			p.log.Error("failed to process message",
				zap.String("key", string(message.Key)),
				zap.Int32("partition", message.TopicPartition.Partition),
				zap.Int32("offset", int32(message.TopicPartition.Offset)),
				zap.Error(err))
			sleep(p.ctx, backoffDuration(attempt, 10*time.Second))
			continue
		}

		// Success - store offset
		p.storeOffset(message)
		return
	}
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
