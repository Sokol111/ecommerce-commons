package tenant

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

func TestLifecycle_Create(t *testing.T) {
	t.Parallel()

	repo := &fakeRepository{}
	runner := &fakeMigrationRunner{}
	lifecycle := NewLifecycleInternal(repo, runner, zap.NewNop())

	err := lifecycle.Create(context.Background(), "shop")

	require.NoError(t, err)
	assert.Equal(t, []string{"shop"}, repo.upsertCalls)
	assert.Equal(t, []string{"shop"}, runner.migrated)
}

func TestLifecycle_Create_UpsertError(t *testing.T) {
	t.Parallel()

	repo := &fakeRepository{upsertErr: errors.New("upsert failed")}
	runner := &fakeMigrationRunner{}
	lifecycle := NewLifecycleInternal(repo, runner, zap.NewNop())

	err := lifecycle.Create(context.Background(), "shop")

	require.Error(t, err)
	assert.ErrorContains(t, err, "upsert failed")
	assert.Empty(t, runner.migrated)
}

func TestLifecycle_Create_MigrationError(t *testing.T) {
	t.Parallel()

	repo := &fakeRepository{}
	runner := &fakeMigrationRunner{err: errors.New("migration failed")}
	lifecycle := NewLifecycleInternal(repo, runner, zap.NewNop())

	err := lifecycle.Create(context.Background(), "shop")

	require.Error(t, err)
	assert.ErrorContains(t, err, "migration failed")
	assert.Equal(t, []string{"shop"}, repo.upsertCalls)
}

func TestLifecycle_Delete(t *testing.T) {
	t.Parallel()

	repo := &fakeRepository{}
	lifecycle := NewLifecycle(repo, nil, zap.NewNop())

	err := lifecycle.Delete(context.Background(), "shop")

	require.NoError(t, err)
	require.Len(t, repo.markForDeletionCalls, 1)
	assert.Equal(t, "shop", repo.markForDeletionCalls[0].slug)
}

func TestLifecycle_Delete_MarkError(t *testing.T) {
	t.Parallel()

	repo := &fakeRepository{markErr: errors.New("mark failed")}
	lifecycle := NewLifecycle(repo, nil, zap.NewNop())

	err := lifecycle.Delete(context.Background(), "shop")

	require.Error(t, err)
	assert.ErrorContains(t, err, "mark failed")
}
