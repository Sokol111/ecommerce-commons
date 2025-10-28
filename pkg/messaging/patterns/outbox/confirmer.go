package outbox

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/confluentinc/confluent-kafka-go/v2/kafka"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

type confirmer struct {
	store        Store
	deliveryChan <-chan kafka.Event
	logger       *zap.Logger

	ctx        context.Context
	cancelFunc context.CancelFunc
	wg         sync.WaitGroup
}

func newConfirmer(
	store Store,
	deliveryChan <-chan kafka.Event,
	logger *zap.Logger,
) *confirmer {
	return &confirmer{
		store:        store,
		deliveryChan: deliveryChan,
		logger:       logger.With(zap.String("component", "outbox")),
	}
}

func provideConfirmer(lc fx.Lifecycle, store Store, channels *channels, logger *zap.Logger) *confirmer {
	c := newConfirmer(store, channels.delivery, logger)

	lc.Append(fx.Hook{
		OnStart: func(ctx context.Context) error {
			c.start()
			return nil
		},
		OnStop: func(ctx context.Context) error {
			c.stop()
			return nil
		},
	})

	return c
}

func (c *confirmer) start() {
	c.logger.Info("starting confirmer")
	c.ctx, c.cancelFunc = context.WithCancel(context.Background())
	c.wg.Add(1)
	go c.run()
}

func (c *confirmer) stop() {
	c.logger.Info("stopping confirmer")
	if c.cancelFunc != nil {
		c.cancelFunc()
	}
	c.wg.Wait()
	c.logger.Info("confirmer stopped")
}

func (c *confirmer) run() {
	defer c.wg.Done()

	events := make([]kafka.Event, 0, 100)

	flush := func() {
		if len(events) == 0 {
			return
		}
		copySlice := make([]kafka.Event, len(events))
		copy(copySlice, events)
		c.wg.Add(1)
		go c.handleConfirmation(copySlice)
		events = events[:0]
	}

	ticker := time.NewTicker(2 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-c.ctx.Done():
			flush()
			return
		default:
		}

		select {
		case <-c.ctx.Done():
			flush()
			return
		case event := <-c.deliveryChan:
			events = append(events, event)
			if len(events) == 100 {
				flush()
			}
		case <-ticker.C:
			flush()
		}
	}
}

func (c *confirmer) handleConfirmation(events []kafka.Event) {
	defer c.wg.Done()

	ids := make([]string, 0, len(events))
	for _, event := range events {
		msg, ok := event.(*kafka.Message)
		if !ok {
			c.logger.Error("skipping confirmation",
				zap.String("reason", "unexpected event type"),
				zap.String("got", fmt.Sprintf("%T", event)),
				zap.String("expected", "*kafka.Message"))
			continue
		}
		if msg.TopicPartition.Error != nil {
			c.logger.Error("skipping confirmation",
				zap.String("reason", "topic partition error"),
				zap.Any("opaque", msg.Opaque),
				zap.Error(msg.TopicPartition.Error),
				zap.Any("topic", msg.TopicPartition.Topic),
				zap.Int32("partition", msg.TopicPartition.Partition),
				zap.Any("offset", msg.TopicPartition.Offset))
			continue
		}
		id, ok := msg.Opaque.(string)
		if !ok {
			c.logger.Error("skipping confirmation",
				zap.String("reason", "failed to cast Opaque to string"),
				zap.Any("opaque", msg.Opaque))
			continue
		}
		ids = append(ids, id)
	}

	if len(ids) == 0 {
		return
	}

	err := c.store.UpdateAsSentByIds(c.ctx, ids)
	if err != nil {
		c.logger.Error("failed to update confirmation", zap.Error(err))
		return
	}

	c.logger.Debug("outbox sending confirmed", zap.Any("ids", ids))
}
