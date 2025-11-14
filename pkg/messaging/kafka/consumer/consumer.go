package consumer

import (
	"context"
	"sync"

	"go.uber.org/zap"
)

type Consumer interface {
	Start(ctx context.Context) error
	Stop(ctx context.Context) error
}

type consumer struct {
	reader      *reader
	processor   *processor
	initializer *initializer
	log         *zap.Logger

	startOnce sync.Once
	stopOnce  sync.Once
}

func newConsumer(reader *reader, processor *processor, initializer *initializer, log *zap.Logger) Consumer {
	return &consumer{
		reader:      reader,
		processor:   processor,
		initializer: initializer,
		log:         log,
	}
}

func (c *consumer) Start(ctx context.Context) error {
	var err error
	c.startOnce.Do(func() {
		c.log.Info("starting consumer")

		// Initialize consumer (subscribe + wait for readiness)
		err = c.initializer.Initialize(ctx)
		if err != nil {
			return
		}

		// Start reader
		c.reader.start()

		// Start processor
		c.processor.start()
	})
	return err
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
