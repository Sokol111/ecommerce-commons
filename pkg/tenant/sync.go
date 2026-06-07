package tenant

import (
	"context"
	"fmt"
	"time"

	"github.com/samber/lo"
	"go.uber.org/zap"
)

const deletionDelay = 5 * time.Minute

// tenantSyncer fetches tenant slugs and reconciles them with the local registry.
type tenantSyncer struct {
	provider SlugsProvider
	repo     repository
	log      *zap.Logger
}

func newTenantSyncer(provider SlugsProvider, repo repository, log *zap.Logger) *tenantSyncer {
	return &tenantSyncer{provider: provider, repo: repo, log: log}
}

// sync fetches tenant slugs and reconciles them with the local registry.
// Returns the list of active slugs.
func (s *tenantSyncer) sync(ctx context.Context) ([]string, error) {
	localSlugs, err := s.slugsFromRegistry(ctx)
	if err != nil {
		return nil, err
	}

	apiSlugs, err := s.provider.GetSlugs(ctx)
	if err != nil {
		s.log.Warn("Failed to fetch tenant slugs from API, falling back to local registry", zap.Error(err))
		return localSlugs, nil
	}

	if err := s.syncRegistry(ctx, apiSlugs, localSlugs); err != nil {
		return nil, fmt.Errorf("failed to sync tenant registry: %w", err)
	}

	return apiSlugs, nil
}

// slugsFromRegistry reads active tenant slugs from the local registry.
func (s *tenantSyncer) slugsFromRegistry(ctx context.Context) ([]string, error) {
	records, err := s.repo.FindActive(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to read local tenant registry: %w", err)
	}
	return lo.Map(records, func(r Record, _ int) string { return r.Slug }), nil
}

// syncRegistry compares API slugs with local slugs and reconciles:
//   - Slugs in API but not local → upsert as active
//   - Slugs in local but not API → mark for deletion
func (s *tenantSyncer) syncRegistry(ctx context.Context, apiSlugs, localSlugs []string) error {
	// New tenants from API (not in local) → upsert as active
	toCreate := lo.Without(apiSlugs, localSlugs...)
	for _, slug := range toCreate {
		if err := s.repo.Upsert(ctx, slug); err != nil {
			return fmt.Errorf("failed to upsert tenant %q: %w", slug, err)
		}
	}

	// Local tenants not in API → mark for deletion
	toDelete := lo.Without(localSlugs, apiSlugs...)
	deleteAfter := time.Now().Add(deletionDelay)
	for _, slug := range toDelete {
		s.log.Info("Tenant not in API, marking for deletion",
			zap.String("tenant", slug),
			zap.Time("delete_after", deleteAfter),
		)
		if err := s.repo.MarkForDeletion(ctx, slug, deleteAfter); err != nil {
			return fmt.Errorf("failed to mark tenant %q for deletion: %w", slug, err)
		}
	}

	return nil
}
