package tenant

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

type fakeRepository struct {
	activeRecords        []Record
	pendingRecords       []Record
	upsertCalls          []string
	markForDeletionCalls []markForDeletionCall
	removeCalls          []string
	findActiveErr        error
	findPendingErr       error
	upsertErr            error
	markErr              error
	removeErr            error
}

type markForDeletionCall struct {
	slug        string
	deleteAfter time.Time
}

func (r *fakeRepository) Upsert(ctx context.Context, slug string) error {
	r.upsertCalls = append(r.upsertCalls, slug)
	if r.upsertErr != nil {
		return r.upsertErr
	}
	return nil
}

func (r *fakeRepository) MarkForDeletion(ctx context.Context, slug string, deleteAfter time.Time) error {
	r.markForDeletionCalls = append(r.markForDeletionCalls, markForDeletionCall{slug: slug, deleteAfter: deleteAfter})
	if r.markErr != nil {
		return r.markErr
	}
	return nil
}

func (r *fakeRepository) FindPendingDeletion(ctx context.Context) ([]Record, error) {
	if r.findPendingErr != nil {
		return nil, r.findPendingErr
	}
	return r.pendingRecords, nil
}

func (r *fakeRepository) FindActive(ctx context.Context) ([]Record, error) {
	if r.findActiveErr != nil {
		return nil, r.findActiveErr
	}
	return r.activeRecords, nil
}

func (r *fakeRepository) Remove(ctx context.Context, slug string) error {
	r.removeCalls = append(r.removeCalls, slug)
	if r.removeErr != nil {
		return r.removeErr
	}
	return nil
}

type fakeSlugsProvider struct {
	slugs []string
	err   error
}

func (p *fakeSlugsProvider) GetSlugs(ctx context.Context) ([]string, error) {
	if p.err != nil {
		return nil, p.err
	}
	return p.slugs, nil
}

func TestTenantSyncer_Sync_NoChanges(t *testing.T) {
	t.Parallel()

	repo := &fakeRepository{activeRecords: []Record{{Slug: "shop1"}, {Slug: "shop2"}}}
	provider := &fakeSlugsProvider{slugs: []string{"shop1", "shop2"}}
	syncer := NewTenantSyncer(provider, repo, zap.NewNop())

	slugs, err := syncer.Sync(context.Background())

	require.NoError(t, err)
	assert.Equal(t, []string{"shop1", "shop2"}, slugs)
	assert.Empty(t, repo.upsertCalls)
	assert.Empty(t, repo.markForDeletionCalls)
}

func TestTenantSyncer_Sync_CreatesMissingTenants(t *testing.T) {
	t.Parallel()

	repo := &fakeRepository{activeRecords: []Record{{Slug: "shop1"}}}
	provider := &fakeSlugsProvider{slugs: []string{"shop1", "shop2", "shop3"}}
	syncer := NewTenantSyncer(provider, repo, zap.NewNop())

	slugs, err := syncer.Sync(context.Background())

	require.NoError(t, err)
	assert.Equal(t, []string{"shop1", "shop2", "shop3"}, slugs)
	assert.ElementsMatch(t, []string{"shop2", "shop3"}, repo.upsertCalls)
	assert.Empty(t, repo.markForDeletionCalls)
}

func TestTenantSyncer_Sync_MarksExtraLocalForDeletion(t *testing.T) {
	t.Parallel()

	repo := &fakeRepository{activeRecords: []Record{{Slug: "shop1"}, {Slug: "shop2"}}}
	provider := &fakeSlugsProvider{slugs: []string{"shop1"}}
	syncer := NewTenantSyncer(provider, repo, zap.NewNop())

	slugs, err := syncer.Sync(context.Background())

	require.NoError(t, err)
	assert.Equal(t, []string{"shop1"}, slugs)
	assert.Empty(t, repo.upsertCalls)
	require.Len(t, repo.markForDeletionCalls, 1)
	assert.Equal(t, "shop2", repo.markForDeletionCalls[0].slug)
	assert.NotNil(t, repo.markForDeletionCalls[0].deleteAfter)
}

func TestTenantSyncer_Sync_FallsBackToLocalOnProviderError(t *testing.T) {
	t.Parallel()

	repo := &fakeRepository{activeRecords: []Record{{Slug: "local1"}}}
	provider := &fakeSlugsProvider{err: errors.New("api unreachable")}
	syncer := NewTenantSyncer(provider, repo, zap.NewNop())

	slugs, err := syncer.Sync(context.Background())

	require.NoError(t, err)
	assert.Equal(t, []string{"local1"}, slugs)
	assert.Empty(t, repo.upsertCalls)
	assert.Empty(t, repo.markForDeletionCalls)
}

func TestTenantSyncer_Sync_PropagatesRepositoryReadError(t *testing.T) {
	t.Parallel()

	repo := &fakeRepository{findActiveErr: errors.New("db down")}
	provider := &fakeSlugsProvider{slugs: []string{"shop1"}}
	syncer := NewTenantSyncer(provider, repo, zap.NewNop())

	_, err := syncer.Sync(context.Background())

	require.Error(t, err)
	assert.ErrorContains(t, err, "db down")
	assert.Empty(t, repo.upsertCalls)
}

func TestTenantSyncer_Sync_PropagatesUpsertError(t *testing.T) {
	t.Parallel()

	repo := &fakeRepository{
		activeRecords: []Record{{Slug: "shop1"}},
		upsertErr:     errors.New("upsert failed"),
	}
	provider := &fakeSlugsProvider{slugs: []string{"shop1", "shop2"}}
	syncer := NewTenantSyncer(provider, repo, zap.NewNop())

	_, err := syncer.Sync(context.Background())

	require.Error(t, err)
	assert.ErrorContains(t, err, "upsert failed")
}

func TestTenantSyncer_Sync_PropagatesMarkForDeletionError(t *testing.T) {
	t.Parallel()

	repo := &fakeRepository{
		activeRecords: []Record{{Slug: "shop1"}, {Slug: "shop2"}},
		markErr:       errors.New("mark failed"),
	}
	provider := &fakeSlugsProvider{slugs: []string{"shop1"}}
	syncer := NewTenantSyncer(provider, repo, zap.NewNop())

	_, err := syncer.Sync(context.Background())

	require.Error(t, err)
	assert.ErrorContains(t, err, "mark failed")
}

func TestTenantSyncer_slugsFromRegistry(t *testing.T) {
	t.Parallel()

	repo := &fakeRepository{
		activeRecords: []Record{{Slug: "a"}, {Slug: "b"}, {Slug: "c"}},
	}
	syncer := NewTenantSyncer(&fakeSlugsProvider{}, repo, zap.NewNop())

	slugs, err := syncer.slugsFromRegistry(context.Background())

	require.NoError(t, err)
	assert.Equal(t, []string{"a", "b", "c"}, slugs)
}

type fakeMigrationRunner struct {
	migrated []string
	err      error
}

func (r *fakeMigrationRunner) MigrateTenant(slug string) error {
	r.migrated = append(r.migrated, slug)
	if r.err != nil {
		return r.err
	}
	return nil
}
