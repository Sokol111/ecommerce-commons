package outbox

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"sync/atomic"
	"time"

	"github.com/Sokol111/ecommerce-commons/pkg/kafka"
	confluent "github.com/confluentinc/confluent-kafka-go/v2/kafka"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.uber.org/zap"
)

type OutboxInterface interface {
	Create(ctx context.Context, payload string, key string, topic string) error
}

type Outbox struct {
	producer kafka.Producer
	store    Store

	log *zap.Logger

	entitiesChan chan OutboxEntity
	deliveryChan chan confluent.Event

	wg         sync.WaitGroup
	cancelFunc context.CancelFunc
	ctx        context.Context

	startOnce sync.Once
	stopOnce  sync.Once
	started   atomic.Bool
}

func NewOutbox(log *zap.Logger, producer kafka.Producer, store Store) *Outbox {
	return &Outbox{
		producer:     producer,
		store:        store,
		entitiesChan: make(chan OutboxEntity, 100),
		deliveryChan: make(chan confluent.Event, 1000),
		log:          log.With(zap.String("component", "outbox")),
	}
}

func (o *Outbox) Start() {
	o.startOnce.Do(func() {
		o.log.Info("starting outbox workers")
		o.ctx, o.cancelFunc = context.WithCancel(context.Background())
		go o.startFetchingWorker()
		go o.startSendingWorker()
		o.wg.Add(1)
		go o.startConfirmationWorker()
		o.started.Store(true)
		o.log.Info("Outbox started")
	})
}

func (o *Outbox) Stop(ctx context.Context) {
	if !o.started.Load() {
		o.log.Warn("outbox not started, skipping stop")
		return
	}

	o.stopOnce.Do(func() {
		o.log.Info("stopping outbox")
		o.cancelFunc()

		done := make(chan struct{})
		go func() {
			defer close(done)
			o.wg.Wait()
		}()

		select {
		case <-done:
		case <-ctx.Done():
			o.log.Warn("shutdown timed out")
		}

		o.log.Info("outbox stopped")
	})
}

func (o *Outbox) Create(ctx context.Context, payload string, key string, topic string) error {
	entity, err := o.store.Create(ctx, payload, key, topic)
	if err != nil {
		return fmt.Errorf("failed to create outbox message: %w", err)
	}
	select {
	case o.entitiesChan <- entity:
		return nil
	default:
		o.log.Warn("entitiesChan is full, dropping message", zap.String("id", entity.ID.Hex()))
		return nil
	}
}

func (o *Outbox) send(entity OutboxEntity) error {
	err := o.producer.Produce(&confluent.Message{
		TopicPartition: confluent.TopicPartition{Topic: &entity.Topic, Partition: confluent.PartitionAny},
		Opaque:         entity.ID,
		Value:          []byte(entity.Payload),
		Key:            []byte(entity.Key),
	}, o.deliveryChan)
	if err != nil {
		return fmt.Errorf("failed to send outbox message with id %v: %w", entity.ID, err)
	}
	return nil
}

func (o *Outbox) startFetchingWorker() {
	defer o.log.Info("fetching worker stopped")
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
				o.log.Error("failed to get outbox entity", zap.Error(err))
				time.Sleep(5 * time.Second)
				continue
			}
			o.entitiesChan <- entity
		}
	}
}

func (o *Outbox) startSendingWorker() {
	defer o.log.Info("sending worker stopped")
	for {
		select {
		case <-o.ctx.Done():
			return
		case entity := <-o.entitiesChan:
			err := o.send(entity)
			if err != nil {
				o.log.Error("failed to send outbox message", zap.String("id", entity.ID.Hex()), zap.Error(err))
			}
		}
	}
}

func (o *Outbox) startConfirmationWorker() {
	defer o.log.Info("confirmation worker stopped")
	defer o.wg.Done()
	events := make([]confluent.Event, 0, 100)

	flush := func() {
		if len(events) == 0 {
			return
		}
		copySlice := make([]confluent.Event, len(events))
		copy(copySlice, events)
		o.wg.Add(1)
		go o.handleConfirmation(copySlice)
		events = events[:0]
	}

	timer := time.NewTimer(2 * time.Second)
	defer timer.Stop()

	for {
		select {
		case <-o.ctx.Done():
			flush()
			return
		case event := <-o.deliveryChan:
			events = append(events, event)
			if len(events) == 100 {
				flush()
				timer.Reset(2 * time.Second)
			}
		case <-timer.C:
			flush()
		}
	}
}

func (o *Outbox) handleConfirmation(events []confluent.Event) {
	defer o.wg.Done()
	ids := make([]primitive.ObjectID, 0, len(events))
	for _, event := range events {
		msg, ok := event.(*confluent.Message)
		if !ok {
			o.log.Warn("skipping confirmation",
				zap.String("reason", "unexpected event type"),
				zap.String("got", fmt.Sprintf("%T", event)),
				zap.String("expected", "*kafka.Message"))
			continue
		}
		if msg.TopicPartition.Error != nil {
			o.log.Warn("skipping confirmation",
				zap.String("reason", "topic partition error"),
				zap.String("opaque", fmt.Sprintf("%#v", msg.Opaque)),
				zap.Error(msg.TopicPartition.Error),
				zap.String("topic", *msg.TopicPartition.Topic),
				zap.Int32("partition", msg.TopicPartition.Partition),
				zap.Any("offset", msg.TopicPartition.Offset))
			continue
		}
		id, ok := msg.Opaque.(primitive.ObjectID)
		if !ok {
			o.log.Warn("skipping confirmation",
				zap.String("reason", "failed to cast Opaque to ObjectID"),
				zap.String("opaque", fmt.Sprintf("%#v", msg.Opaque)))
			continue
		}
		ids = append(ids, id)
	}

	if len(ids) == 0 {
		return
	}

	err := o.store.UpdateAsSentByIds(o.ctx, ids)
	if err != nil {
		o.log.Error("failed to update confirmation", zap.Error(err))
	}
}
