package token

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestContextWithClaims_And_ClaimsFromContext(t *testing.T) {
	t.Run("store and retrieve claims", func(t *testing.T) {
		claims := &Claims{
			Tenant:      "tenant-1",
			Role:        "admin",
			Permissions: []string{"read", "write"},
		}

		ctx := context.Background()
		ctx = ContextWithClaims(ctx, claims)

		retrieved := ClaimsFromContext(ctx)
		assert.NotNil(t, retrieved)
		assert.Equal(t, claims.Tenant, retrieved.Tenant)
		assert.Equal(t, claims.Role, retrieved.Role)
		assert.Equal(t, claims.Permissions, retrieved.Permissions)
	})

	t.Run("returns nil when no claims in context", func(t *testing.T) {
		ctx := context.Background()
		retrieved := ClaimsFromContext(ctx)
		assert.Nil(t, retrieved)
	})

	t.Run("store nil claims", func(t *testing.T) {
		ctx := context.Background()
		ctx = ContextWithClaims(ctx, nil)

		retrieved := ClaimsFromContext(ctx)
		assert.Nil(t, retrieved)
	})

	t.Run("overwrite claims", func(t *testing.T) {
		claims1 := &Claims{Tenant: "tenant-1"}
		claims2 := &Claims{Tenant: "tenant-2"}

		ctx := context.Background()
		ctx = ContextWithClaims(ctx, claims1)
		ctx = ContextWithClaims(ctx, claims2)

		retrieved := ClaimsFromContext(ctx)
		assert.Equal(t, "tenant-2", retrieved.Tenant)
	})
}
