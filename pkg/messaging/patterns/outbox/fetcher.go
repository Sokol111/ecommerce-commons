package outbox

import (
	"context"
	"errors"
	"time"

	"github.com/Sokol111/ecommerce-commons/pkg/core/health"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

type fetcher struct {
	store        Store
	entitiesChan chan<- *outboxEntity
	logger       *zap.Logger
	readiness    health.Readiness

	ctx        context.Context
	cancelFunc context.CancelFunc
}

func newFetcher(store Store, entitiesChan chan<- *outboxEntity, logger *zap.Logger, readiness health.Readiness) *fetcher {
	return &fetcher{
		store:        store,
		entitiesChan: entitiesChan,
		logger:       logger.With(zap.String("component", "outbox")),
		readiness:    readiness,
	}
}

func provideFetcher(lc fx.Lifecycle, store Store, channels *channels, logger *zap.Logger, readiness health.Readiness) *fetcher {
	f := newFetcher(store, channels.entities, logger, readiness)

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

	// Wait for Kubernetes readiness before starting to fetch entities
	f.logger.Info("waiting for Kubernetes readiness before fetching outbox entities")
	if err := f.readiness.WaitKubernetesReady(f.ctx); err != nil {
		f.logger.Info("context cancelled while waiting for Kubernetes readiness")
		return
	}
	f.logger.Info("Kubernetes readiness achieved, starting to fetch outbox entities")

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
