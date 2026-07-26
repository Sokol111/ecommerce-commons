package tenant

import (
	"context"
	"time"

	"go.uber.org/zap"
)

type CleanupWorker interface {
	Run(ctx context.Context) error
}

type NoopCleanupWorker struct{}

func (w *NoopCleanupWorker) Run(ctx context.Context) error {
	return nil
}

const cleanupWorkerInterval = 1 * time.Minute

type DefaultCleanupWorker struct {
	repo     Repository
	cleaners []Cleaner
	log      *zap.Logger
}

func NewDefaultCleanupWorker(repo Repository, cleaners []Cleaner, log *zap.Logger) *DefaultCleanupWorker {
	return &DefaultCleanupWorker{repo: repo, cleaners: cleaners, log: log}
}

func (w *DefaultCleanupWorker) Run(ctx context.Context) error {
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

func (w *DefaultCleanupWorker) processExpiredTenants(ctx context.Context) {
	records, err := w.repo.FindPendingDeletion(ctx)
	if err != nil {
		w.log.Error("failed to find pending deletion tenants", zap.Error(err))
		return
	}

	for _, record := range records {
		w.cleanupTenant(ctx, record.Slug)
	}
}

func (w *DefaultCleanupWorker) cleanupTenant(ctx context.Context, slug string) {
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
