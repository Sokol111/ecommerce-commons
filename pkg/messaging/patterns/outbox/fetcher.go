package outbox

import (
	"context"
	"errors"
	"time"

	"go.uber.org/zap"
)

type fetcher struct {
	outboxRepository repository
	entitiesChan     chan<- *outboxEntity
	logger           *zap.Logger
}

func newFetcher(outboxRepository repository, entitiesChan chan *outboxEntity, logger *zap.Logger) *fetcher {
	return &fetcher{
		outboxRepository: outboxRepository,
		entitiesChan:     entitiesChan,
		logger:           logger,
	}
}

func (f *fetcher) Run(ctx context.Context) error {
	for {
		select {
		case <-ctx.Done():
			return nil
		default:
		}

		entity, err := f.outboxRepository.FetchAndLock(ctx)
		if err != nil {
			if errors.Is(err, errEntityNotFound) {
				select {
				case <-ctx.Done():
					return nil
				case <-time.After(5 * time.Second):
				}
				continue
			}
			f.logger.Error("failed to get outbox entity", zap.Error(err))
			select {
			case <-ctx.Done():
				return nil
			case <-time.After(10 * time.Second):
			}
			continue
		}

		select {
		case <-ctx.Done():
			return nil
		case f.entitiesChan <- entity:
		}
	}
}
