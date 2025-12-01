package outbox

import (
	"context"
	"errors"
	"time"

	"go.uber.org/zap"
)

type fetcher struct {
	store        Store
	entitiesChan chan<- *outboxEntity
	logger       *zap.Logger
}

func newFetcher(store Store, entitiesChan chan<- *outboxEntity, logger *zap.Logger) *fetcher {
	return &fetcher{
		store:        store,
		entitiesChan: entitiesChan,
		logger:       logger,
	}
}

func (f *fetcher) Run(ctx context.Context) error {
	for {
		select {
		case <-ctx.Done():
			return nil
		default:
			entity, err := f.store.FetchAndLock(ctx)
			if err != nil {
				if errors.Is(err, errEntityNotFound) {
					time.Sleep(2 * time.Second)
					continue
				}
				f.logger.Error("failed to get outbox entity", zap.Error(err))
				time.Sleep(5 * time.Second)
				continue
			}
			f.entitiesChan <- entity
		}
	}
}
