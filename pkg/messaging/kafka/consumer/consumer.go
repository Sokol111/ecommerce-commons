package consumer

import (
	"context"
	"sync"

	"go.uber.org/zap"
)

type Consumer interface {
	Start() error
	Stop(ctx context.Context) error
}

type consumer struct {
	reader    *reader
	processor *processor
	log       *zap.Logger

	startOnce sync.Once
	stopOnce  sync.Once
}

func newConsumer(
	reader *reader,
	processor *processor,
	log *zap.Logger,
) Consumer {
	return &consumer{
		reader:    reader,
		processor: processor,
		log:       log,
	}
}

func (c *consumer) Start() error {
	var startErr error
	c.startOnce.Do(func() {
		c.log.Info("starting consumer")

		c.reader.start()
		c.processor.start()

		c.log.Info("consumer started")
	})
	return startErr
}

func (c *consumer) Stop(ctx context.Context) error {
	var resultErr error

	c.stopOnce.Do(func() {
		c.log.Info("stopping consumer")

		// Stop in reverse order: processor -> reader
		c.processor.stop()
		c.reader.stop()

		c.log.Info("consumer stopped")
	})

	return resultErr
}
