package tenant

import (
	"context"
	"fmt"
	"time"

	"go.uber.org/zap"
)

const eventDeletionDelay = 5 * time.Minute

// Lifecycle manages tenant creation and deletion.
// Used by event handlers to delegate tenant lifecycle logic to the commons package.
type Lifecycle interface {
	// Create registers a tenant and runs its database migrations.
	Create(ctx context.Context, slug string) error

	// Delete marks a tenant for deferred deletion.
	Delete(ctx context.Context, slug string) error
}

type lifecycle struct {
	repo   repository
	runner *migrationRunner
	log    *zap.Logger
}

func newLifecycle(repo repository, runner *migrationRunner, log *zap.Logger) Lifecycle {
	return &lifecycle{repo: repo, runner: runner, log: log}
}

func (l *lifecycle) Create(ctx context.Context, slug string) error {
	l.log.Info("Creating tenant", zap.String("tenant", slug))

	if err := l.repo.Upsert(ctx, slug); err != nil {
		return fmt.Errorf("failed to register tenant %q: %w", slug, err)
	}

	if err := l.runner.migrateTenant(slug); err != nil {
		return fmt.Errorf("failed to migrate tenant %q: %w", slug, err)
	}

	return nil
}

func (l *lifecycle) Delete(ctx context.Context, slug string) error {
	l.log.Info("Scheduling tenant deletion", zap.String("tenant", slug))

	deleteAfter := time.Now().Add(eventDeletionDelay)
	if err := l.repo.MarkForDeletion(ctx, slug, deleteAfter); err != nil {
		return fmt.Errorf("failed to mark tenant %q for deletion: %w", slug, err)
	}

	return nil
}
