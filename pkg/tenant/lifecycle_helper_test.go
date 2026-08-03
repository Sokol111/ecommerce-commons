package tenant

import (
	"context"
	"fmt"

	"go.uber.org/zap"
)

type migrationRunner interface {
	MigrateTenant(slug string) error
}

type LifecycleInternal struct {
	repo   Repository
	runner migrationRunner
	log    *zap.Logger
}

func NewLifecycleInternal(repo Repository, runner migrationRunner, log *zap.Logger) *LifecycleInternal {
	return &LifecycleInternal{repo: repo, runner: runner, log: log}
}

func (l *LifecycleInternal) Create(ctx context.Context, slug string) error {
	l.log.Info("Creating tenant")
	if err := l.repo.Upsert(ctx, slug); err != nil {
		return fmt.Errorf("failed to register tenant %q: %w", slug, err)
	}
	if err := l.runner.MigrateTenant(slug); err != nil {
		return fmt.Errorf("failed to migrate tenant %q: %w", slug, err)
	}
	return nil
}

func (l *LifecycleInternal) Delete(ctx context.Context, slug string) error {
	return NewLifecycle(l.repo, nil, l.log).Delete(ctx, slug)
}
