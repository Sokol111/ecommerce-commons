package outbox

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"sync"
	"sync/atomic"
	"time"

	"github.com/Sokol111/ecommerce-commons/pkg/kafka/producer"
	"github.com/Sokol111/ecommerce-commons/pkg/logger"
	"github.com/confluentinc/confluent-kafka-go/v2/kafka"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.uber.org/zap"
)

type Outbox interface {
	Create(ctx context.Context, event any, key string, topic string) (SendFunc, error)
	Start()
	Stop(ctx context.Context) error
}

type outbox struct {
	producer producer.Producer
	store    Store

	logger *zap.Logger

	entitiesChan chan *outboxEntity
	deliveryChan chan kafka.Event

	wg         sync.WaitGroup
	cancelFunc context.CancelFunc
	ctx        context.Context

	startOnce sync.Once
	stopOnce  sync.Once
	started   atomic.Bool
}

func newOutbox(logger *zap.Logger, producer producer.Producer, store Store) Outbox {
	return &outbox{
		producer:     producer,
		store:        store,
		entitiesChan: make(chan *outboxEntity, 100),
		deliveryChan: make(chan kafka.Event, 1000),
		logger:       logger.With(zap.String("component", "outbox")),
	}
}

func (o *outbox) Start() {
	o.startOnce.Do(func() {
		o.logger.Info("starting outbox workers")
		o.ctx, o.cancelFunc = context.WithCancel(context.Background())
		go o.startFetchingWorker()
		go o.startSendingWorker()
		o.wg.Add(1)
		go o.startConfirmationWorker()
		o.started.Store(true)
		o.logger.Info("outbox started")
	})
}

func (o *outbox) Stop(ctx context.Context) error {
	if !o.started.Load() {
		o.logger.Warn("outbox not started, skipping stop")
		return nil
	}

	var err error

	o.stopOnce.Do(func() {
		o.logger.Info("stopping outbox")
		o.cancelFunc()

		done := make(chan struct{})
		go func() {
			defer close(done)
			o.wg.Wait()
		}()

		select {
		case <-done:
		case <-ctx.Done():
			err = fmt.Errorf("shutdown timed out")
		}

		o.logger.Info("outbox stopped")
	})
	return err
}

type SendFunc func(ctx context.Context) error

func (o *outbox) Create(ctx context.Context, event any, key string, topic string) (SendFunc, error) {
	eventStr, err := json.Marshal(event)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal event: %w", err)
	}
	entity, err := o.store.Create(ctx, string(eventStr), key, topic)
	if err != nil {
		return nil, fmt.Errorf("failed to create outbox message: %w", err)
	}
	o.log(ctx).Debug("outbox created", zap.String("id", entity.ID.Hex()))

	return SendFunc(func(ctx context.Context) error {
		timer := time.NewTimer(1 * time.Second)
		defer timer.Stop()
		select {
		case <-ctx.Done():
			return fmt.Errorf("outbox didn't sent: %w", ctx.Err())
		case o.entitiesChan <- entity:
			return nil
		case <-timer.C:
			o.log(ctx).Warn("entitiesChan is full, dropping message", zap.String("id", entity.ID.Hex()))
			return fmt.Errorf("entitiesChan is full")
		}
	}), nil
}

func (o *outbox) send(entity *outboxEntity) error {
	err := o.producer.Produce(&kafka.Message{
		TopicPartition: kafka.TopicPartition{Topic: &entity.Topic, Partition: kafka.PartitionAny},
		Opaque:         entity.ID,
		Value:          []byte(entity.Payload),
		Key:            []byte(entity.Key),
	}, o.deliveryChan)
	if err != nil {
		return fmt.Errorf("failed to send outbox message with id %v: %w", entity.ID, err)
	}
	return nil
}

func (o *outbox) startFetchingWorker() {
	o.logger.Info("starting fetching worker")
	defer o.logger.Info("fetching worker stopped")
	for {
		select {
		case <-o.ctx.Done():
			return
		default:
			entity, err := o.store.FetchAndLock(o.ctx)
			if err != nil {
				if errors.Is(err, errEntityNotFound) {
					time.Sleep(2 * time.Second)
					continue
				}
				o.logger.Error("failed to get outbox entity", zap.Error(err))
				time.Sleep(5 * time.Second)
				continue
			}
			o.entitiesChan <- entity
		}
	}
}

func (o *outbox) startSendingWorker() {
	o.logger.Info("starting sending worker")
	defer o.logger.Info("sending worker stopped")
	for {
		select {
		case <-o.ctx.Done():
			return
		case entity := <-o.entitiesChan:
			if o.ctx.Err() != nil {
				return
			}
			err := o.send(entity)
			if err != nil {
				o.logger.Error("failed to send outbox message", zap.String("id", entity.ID.Hex()), zap.Error(err))
			}
			o.logger.Debug("outbox sent to kafka", zap.String("id", entity.ID.Hex()))
		}
	}
}

func (o *outbox) startConfirmationWorker() {
	o.logger.Info("starting confirmation worker")
	defer func() {
		o.logger.Info("confirmation worker stopped")
		o.wg.Done()
	}()
	events := make([]kafka.Event, 0, 100)

	flush := func() {
		if len(events) == 0 {
			return
		}
		copySlice := make([]kafka.Event, len(events))
		copy(copySlice, events)
		o.wg.Add(1)
		go o.handleConfirmation(copySlice)
		events = events[:0]
	}

	ticker := time.NewTicker(2 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-o.ctx.Done():
			flush()
			return
		case event := <-o.deliveryChan:
			if o.ctx.Err() != nil {
				flush()
				return
			}
			events = append(events, event)
			if len(events) == 100 {
				flush()
			}
		case <-ticker.C:
			flush()
		}
	}
}

func (o *outbox) handleConfirmation(events []kafka.Event) {
	defer o.wg.Done()
	ids := make([]primitive.ObjectID, 0, len(events))
	for _, event := range events {
		msg, ok := event.(*kafka.Message)
		if !ok {
			o.logger.Warn("skipping confirmation",
				zap.String("reason", "unexpected event type"),
				zap.String("got", fmt.Sprintf("%T", event)),
				zap.String("expected", "*kafka.Message"))
			continue
		}
		if msg.TopicPartition.Error != nil {
			o.logger.Warn("skipping confirmation",
				zap.String("reason", "topic partition error"),
				zap.Any("opaque", msg.Opaque),
				zap.Error(msg.TopicPartition.Error),
				zap.Any("topic", msg.TopicPartition.Topic),
				zap.Int32("partition", msg.TopicPartition.Partition),
				zap.Any("offset", msg.TopicPartition.Offset))
			continue
		}
		id, ok := msg.Opaque.(primitive.ObjectID)
		if !ok {
			o.logger.Warn("skipping confirmation",
				zap.String("reason", "failed to cast Opaque to ObjectID"),
				zap.Any("opaque", msg.Opaque))
			continue
		}
		ids = append(ids, id)
	}

	if len(ids) == 0 {
		return
	}

	err := o.store.UpdateAsSentByIds(o.ctx, ids)
	if err != nil {
		o.logger.Error("failed to update confirmation", zap.Error(err))
	}

	o.logger.Debug("outbox sending confirmed", zap.Any("ids", ids))
}

func (o *outbox) log(ctx context.Context) *zap.Logger {
	return logger.CombineLogger(o.logger, ctx)
}
