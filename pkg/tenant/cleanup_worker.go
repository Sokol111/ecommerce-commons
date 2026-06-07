package tenant

import (
	"context"
	"time"

	"go.uber.org/zap"
)

const cleanupWorkerInterval = 1 * time.Minute

type cleanupWorker struct {
	repo     repository
	cleaners []Cleaner
	log      *zap.Logger
}

func newCleanupWorker(repo repository, cleaners []Cleaner, log *zap.Logger) *cleanupWorker {
	return &cleanupWorker{repo: repo, cleaners: cleaners, log: log}
}

func (w *cleanupWorker) Run(ctx context.Context) error {
	ticker := time.NewTicker(cleanupWorkerInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return nil
		case <-ticker.C:
			w.processExpiredTenants(ctx)
		}
	}
}

func (w *cleanupWorker) processExpiredTenants(ctx context.Context) {
	records, err := w.repo.FindPendingDeletion(ctx)
	if err != nil {
		w.log.Error("failed to find pending deletion tenants", zap.Error(err))
		return
	}

	for _, record := range records {
		w.cleanupTenant(ctx, record.Slug)
	}
}

func (w *cleanupWorker) cleanupTenant(ctx context.Context, slug string) {
	w.log.Info("cleaning up expired tenant", zap.String("tenant", slug))

	for _, cleaner := range w.cleaners {
		if err := cleaner.CleanupTenant(ctx, slug); err != nil {
			w.log.Error("tenant cleanup failed", zap.String("tenant", slug), zap.Error(err))
			return
		}
	}

	if err := w.repo.Remove(ctx, slug); err != nil {
		w.log.Error("failed to remove tenant from repository after cleanup", zap.String("tenant", slug), zap.Error(err))
		return
	}

	w.log.Info("tenant cleanup completed", zap.String("tenant", slug))
}
