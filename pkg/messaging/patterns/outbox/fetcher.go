package outbox

import (
	"context"
	"errors"
	"time"

	"go.uber.org/fx"
	"go.uber.org/zap"
)

type fetcher struct {
	store        Store
	entitiesChan chan<- *outboxEntity
	logger       *zap.Logger

	ctx        context.Context
	cancelFunc context.CancelFunc
}

func newFetcher(store Store, entitiesChan chan<- *outboxEntity, logger *zap.Logger) *fetcher {
	return &fetcher{
		store:        store,
		entitiesChan: entitiesChan,
		logger:       logger.With(zap.String("component", "outbox")),
	}
}

func provideFetcher(lc fx.Lifecycle, store Store, channels *channels, logger *zap.Logger) *fetcher {
	f := newFetcher(store, channels.entities, logger)

	lc.Append(fx.Hook{
		OnStart: func(ctx context.Context) error {
			f.start()
			return nil
		},
		OnStop: func(ctx context.Context) error {
			f.stop()
			return nil
		},
	})

	return f
}

func (f *fetcher) start() {
	f.logger.Info("starting fetcher")
	f.ctx, f.cancelFunc = context.WithCancel(context.Background())
	go f.run()
}

func (f *fetcher) stop() {
	f.logger.Info("stopping fetcher")
	if f.cancelFunc != nil {
		f.cancelFunc()
	}
}

func (f *fetcher) run() {
	defer f.logger.Info("fetcher worker stopped")

	for {
		select {
		case <-f.ctx.Done():
			return
		default:
			entity, err := f.store.FetchAndLock(f.ctx)
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
