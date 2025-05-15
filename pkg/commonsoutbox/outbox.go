package commonsoutbox

import (
	"context"
	"errors"
	"fmt"
	"github.com/Sokol111/ecommerce-commons/pkg/commonskafka"
	"github.com/confluentinc/confluent-kafka-go/v2/kafka"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"log/slog"
	"sync"
	"time"
)

type OutboxInterface interface {
	SaveAndSend(entity OutboxEntity)
}

type Outbox struct {
	producer   commonskafka.ProducerInterface
	repository OutboxRepository

	logger *slog.Logger

	deliveryChan chan kafka.Event

	wg         sync.WaitGroup
	cancelFunc context.CancelFunc
	ctx        context.Context
	startOnce  sync.Once
	stopOnce   sync.Once
}

func NewOutbox(ctx context.Context, producer commonskafka.ProducerInterface, repository OutboxRepository) *Outbox {
	ctx, cancelFunc := context.WithCancel(ctx)
	return &Outbox{
		cancelFunc:   cancelFunc,
		ctx:          ctx,
		producer:     producer,
		repository:   repository,
		deliveryChan: make(chan kafka.Event, 1000),
		logger:       slog.Default().With("component", "outbox"),
	}
}

func (o *Outbox) Start() {
	o.startOnce.Do(func() {
		o.logger.Info("starting outbox workers")
		go o.startFetchingWorker()
		o.wg.Add(1)
		go o.startConfirmationWorker()
	})
}

func (o *Outbox) Stop() {
	o.stopOnce.Do(func() {
		o.logger.Info("stopping outbox")
		o.cancelFunc()
		o.wg.Wait()
		o.logger.Info("outbox stopped")
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
	defer o.logger.Info("fetching worker stopped")
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
				o.logger.Error("failed to get outbox entity", "err", err)
				time.Sleep(5 * time.Second)
				continue
			}
			err = o.send(entity)

			if err != nil {
				o.logger.Error("failed to send outbox message", "id", entity.ID.Hex(), "err", err)
				time.Sleep(2 * time.Second)
			}
		}
	}
}

func (o *Outbox) startConfirmationWorker() {
	defer o.logger.Info("confirmation worker stopped")
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
			o.logger.Warn("skipping confirmation",
				"reason", "unexpected event type",
				"got", fmt.Sprintf("%T", event),
				"expected", "*kafka.Message")
			continue
		}
		if msg.TopicPartition.Error != nil {
			o.logger.Warn("skipping confirmation",
				"reason", "topic partition error",
				"opaque", fmt.Sprintf("%#v", msg.Opaque),
				"error", msg.TopicPartition.Error,
				"topic", *msg.TopicPartition.Topic,
				"partition", msg.TopicPartition.Partition,
				"offset", msg.TopicPartition.Offset)
			continue
		}
		id, ok := msg.Opaque.(primitive.ObjectID)
		if !ok {
			o.logger.Warn("skipping confirmation",
				"reason", "failed to cast Opaque to ObjectID",
				"opaque", fmt.Sprintf("%#v", msg.Opaque))
			continue
		}
		ids = append(ids, id)
	}

	if len(ids) == 0 {
		return
	}

	err := o.repository.UpdateAsSentByIds(o.ctx, ids)
	if err != nil {
		o.logger.Error("failed to update confirmation", "err", err)
	}
}
