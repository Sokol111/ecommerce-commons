package commonsoutbox

import (
	"context"
	"errors"
	"fmt"
	"github.com/Sokol111/ecommerce-commons/pkg/commonskafka"
	"github.com/confluentinc/confluent-kafka-go/v2/kafka"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.uber.org/zap"
	"sync"
	"time"
)

type OutboxInterface interface {
	SaveAndSend(entity OutboxEntity)
}

type Outbox struct {
	producer   commonskafka.ProducerInterface
	repository OutboxRepository

	log *zap.Logger

	deliveryChan chan kafka.Event

	wg         sync.WaitGroup
	cancelFunc context.CancelFunc
	ctx        context.Context
	startOnce  sync.Once
	stopOnce   sync.Once
}

func NewOutbox(ctx context.Context, log *zap.Logger, producer commonskafka.ProducerInterface, repository OutboxRepository) *Outbox {
	ctx, cancelFunc := context.WithCancel(ctx)
	return &Outbox{
		cancelFunc:   cancelFunc,
		ctx:          ctx,
		producer:     producer,
		repository:   repository,
		deliveryChan: make(chan kafka.Event, 1000),
		log:          log.With(zap.String("component", "outbox")),
	}
}

func (o *Outbox) Start() {
	o.startOnce.Do(func() {
		o.log.Info("starting outbox workers")
		go o.startFetchingWorker()
		o.wg.Add(1)
		go o.startConfirmationWorker()
	})
}

func (o *Outbox) Stop() {
	o.stopOnce.Do(func() {
		o.log.Info("stopping outbox")
		o.cancelFunc()
		o.wg.Wait()
		o.log.Info("outbox stopped")
	})
}

func (o *Outbox) Create(payload string, key string, topic string) error {
	_, err := o.repository.Create(o.ctx, payload, key, topic)
	if err != nil {
		return fmt.Errorf("failed to create outbox message: %w", err)
	}
	return nil
}

func (o *Outbox) send(entity OutboxEntity) error {
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

func (o *Outbox) startFetchingWorker() {
	defer o.log.Info("fetching worker stopped")
	for {
		select {
		case <-o.ctx.Done():
			return
		default:
			entity, err := o.repository.FetchAndLock(o.ctx)
			if err != nil {
				if errors.Is(err, EntityNotFoundError) {
					time.Sleep(2 * time.Second)
					continue
				}
				o.log.Error("failed to get outbox entity", zap.Error(err))
				time.Sleep(5 * time.Second)
				continue
			}
			err = o.send(entity)

			if err != nil {
				o.log.Error("failed to send outbox message", zap.String("id", entity.ID.Hex()), zap.Error(err))
				time.Sleep(2 * time.Second)
			}
		}
	}
}

func (o *Outbox) startConfirmationWorker() {
	defer o.log.Info("confirmation worker stopped")
	defer o.wg.Done()
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

func (o *Outbox) handleConfirmation(events []kafka.Event) {
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

	err := o.repository.UpdateAsSentByIds(o.ctx, ids)
	if err != nil {
		o.log.Error("failed to update confirmation", zap.Error(err))
	}
}
