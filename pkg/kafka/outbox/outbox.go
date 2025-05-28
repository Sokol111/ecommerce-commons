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
	"github.com/confluentinc/confluent-kafka-go/v2/kafka"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.uber.org/zap"
)

type Outbox interface {
	Create(ctx context.Context, event any, key string, topic string) error
	Start()
	Stop(ctx context.Context) error
}

type outbox struct {
	producer producer.Producer
	store    Store

	log *zap.Logger

	entitiesChan chan OutboxEntity
	deliveryChan chan kafka.Event

	wg         sync.WaitGroup
	cancelFunc context.CancelFunc
	ctx        context.Context

	startOnce sync.Once
	stopOnce  sync.Once
	started   atomic.Bool
}

func NewOutbox(log *zap.Logger, producer producer.Producer, store Store) Outbox {
	return &outbox{
		producer:     producer,
		store:        store,
		entitiesChan: make(chan OutboxEntity, 100),
		deliveryChan: make(chan kafka.Event, 1000),
		log:          log.With(zap.String("component", "outbox")),
	}
}

func (o *outbox) Start() {
	o.startOnce.Do(func() {
		o.log.Info("starting outbox workers")
		o.ctx, o.cancelFunc = context.WithCancel(context.Background())
		go o.startFetchingWorker()
		go o.startSendingWorker()
		o.wg.Add(1)
		go o.startConfirmationWorker()
		o.started.Store(true)
		o.log.Info("outbox started")
	})
}

func (o *outbox) Stop(ctx context.Context) error {
	if !o.started.Load() {
		o.log.Warn("outbox not started, skipping stop")
		return nil
	}

	var err error

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
			err = fmt.Errorf("shutdown timed out")
		}

		o.log.Info("outbox stopped")
	})
	return err
}

func (o *outbox) Create(ctx context.Context, event any, key string, topic string) error {
	eventStr, err := json.Marshal(event)
	if err != nil {
		return fmt.Errorf("failed to marshal event: %w", err)
	}
	entity, err := o.store.Create(ctx, string(eventStr), key, topic)
	if err != nil {
		return fmt.Errorf("failed to create outbox message: %w", err)
	}
	select {
	case o.entitiesChan <- entity:
		return nil
	default:
		o.log.Warn("entitiesChan is full, dropping message", zap.String("id", entity.ID.Hex()))
		err = o.store.UpdateLockExpiresAt(ctx, entity.ID, time.Now().UTC())
		if err != nil {
			o.log.Warn("failed to update lockExpiresAt", zap.Error(err))
		}
		return nil
	}
}

func (o *outbox) send(entity OutboxEntity) error {
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
	o.log.Info("starting fetching worker")
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

func (o *outbox) startSendingWorker() {
	o.log.Info("starting sending worker")
	defer o.log.Info("sending worker stopped")
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
				o.log.Error("failed to send outbox message", zap.String("id", entity.ID.Hex()), zap.Error(err))
			}
		}
	}
}

func (o *outbox) startConfirmationWorker() {
	o.log.Info("starting confirmation worker")
	defer func() {
		o.log.Info("confirmation worker stopped")
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
