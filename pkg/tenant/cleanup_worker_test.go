package tenant

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

type fakeCleaner struct {
	cleaned []string
	err     error
}

func (c *fakeCleaner) CleanupTenant(ctx context.Context, slug string) error {
	c.cleaned = append(c.cleaned, slug)
	if c.err != nil {
		return c.err
	}
	return nil
}

func TestDefaultCleanupWorker_processExpiredTenants(t *testing.T) {
	t.Parallel()

	repo := &fakeRepository{
		pendingRecords: []Record{{Slug: "shop1"}, {Slug: "shop2"}},
	}
	cleaner := &fakeCleaner{}
	worker := NewDefaultCleanupWorker(repo, []Cleaner{cleaner}, zap.NewNop())

	worker.processExpiredTenants(context.Background())

	assert.Equal(t, []string{"shop1", "shop2"}, cleaner.cleaned)
	assert.Equal(t, []string{"shop1", "shop2"}, repo.removeCalls)
}

func TestDefaultCleanupWorker_processExpiredTenants_FindError(t *testing.T) {
	t.Parallel()

	repo := &fakeRepository{findPendingErr: errors.New("find failed")}
	cleaner := &fakeCleaner{}
	worker := NewDefaultCleanupWorker(repo, []Cleaner{cleaner}, zap.NewNop())

	worker.processExpiredTenants(context.Background())

	assert.Empty(t, cleaner.cleaned)
	assert.Empty(t, repo.removeCalls)
}

func TestDefaultCleanupWorker_cleanupTenant_CleanerError(t *testing.T) {
	t.Parallel()

	repo := &fakeRepository{}
	cleaner := &fakeCleaner{err: errors.New("cleanup failed")}
	worker := NewDefaultCleanupWorker(repo, []Cleaner{cleaner}, zap.NewNop())

	worker.cleanupTenant(context.Background(), "shop")

	assert.Equal(t, []string{"shop"}, cleaner.cleaned)
	assert.Empty(t, repo.removeCalls)
}

func TestDefaultCleanupWorker_cleanupTenant_RemoveError(t *testing.T) {
	t.Parallel()

	repo := &fakeRepository{removeErr: errors.New("remove failed")}
	cleaner := &fakeCleaner{}
	worker := NewDefaultCleanupWorker(repo, []Cleaner{cleaner}, zap.NewNop())

	worker.cleanupTenant(context.Background(), "shop")

	assert.Equal(t, []string{"shop"}, cleaner.cleaned)
	assert.Equal(t, []string{"shop"}, repo.removeCalls)
}

func TestDefaultCleanupWorker_Run_StopsOnContext(t *testing.T) {
	t.Parallel()

	repo := &fakeRepository{}
	worker := NewDefaultCleanupWorker(repo, nil, zap.NewNop())

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	err := worker.Run(ctx)

	require.NoError(t, err)
}

func TestNoopCleanupWorker_Run(t *testing.T) {
	t.Parallel()

	w := &NoopCleanupWorker{}
	require.NoError(t, w.Run(context.Background()))
}
