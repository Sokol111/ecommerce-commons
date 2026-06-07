package tenant

import "context"

// SlugsProvider fetches the list of active tenant slugs at startup.
// Implementations typically call tenant-service API.
type SlugsProvider interface {
	GetSlugs(ctx context.Context) ([]string, error)
}
