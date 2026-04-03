package tenant

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestContextWithSlug_and_SlugFromContext(t *testing.T) {
	ctx := ContextWithSlug(context.Background(), "tenant-123")

	slug, ok := SlugFromContext(ctx)
	require.True(t, ok)
	assert.Equal(t, "tenant-123", slug)
}

func TestSlugFromContext_NoTenant(t *testing.T) {
	slug, ok := SlugFromContext(context.Background())
	assert.False(t, ok)
	assert.Empty(t, slug)
}

func TestSlugFromContext_EmptyTenant(t *testing.T) {
	ctx := ContextWithSlug(context.Background(), "")

	slug, ok := SlugFromContext(ctx)
	assert.False(t, ok)
	assert.Empty(t, slug)
}

func TestMustSlugFromContext_Success(t *testing.T) {
	ctx := ContextWithSlug(context.Background(), "tenant-456")

	slug := MustSlugFromContext(ctx)
	assert.Equal(t, "tenant-456", slug)
}

func TestMustSlugFromContext_Panics(t *testing.T) {
	assert.Panics(t, func() {
		MustSlugFromContext(context.Background())
	})
}
