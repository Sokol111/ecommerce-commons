package tenant

import (
	"context"
)

// tenantKey is the context key for storing the tenant slug.
type tenantKey struct{}

// ContextWithSlug returns a new context with the tenant slug stored.
func ContextWithSlug(ctx context.Context, slug string) context.Context {
	return context.WithValue(ctx, tenantKey{}, slug)
}

// SlugFromContext retrieves the tenant slug from the context.
// Returns the slug and true if present, empty string and false otherwise.
func SlugFromContext(ctx context.Context) (string, bool) {
	slug, ok := ctx.Value(tenantKey{}).(string)
	if !ok || slug == "" {
		return "", false
	}
	return slug, true
}

// MustSlugFromContext retrieves the tenant slug from the context.
// Panics if tenant slug is not present — use only in code paths
// where tenant middleware has already validated the presence.
func MustSlugFromContext(ctx context.Context) string {
	slug, ok := SlugFromContext(ctx)
	if !ok {
		panic("tenant: MustSlugFromContext called without tenant in context")
	}
	return slug
}
