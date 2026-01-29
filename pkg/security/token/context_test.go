package token

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestContextWithClaims_And_ClaimsFromContext(t *testing.T) {
	t.Run("store and retrieve claims", func(t *testing.T) {
		claims := &Claims{
			UserID:      "user-123",
			Role:        "admin",
			Permissions: []string{"read", "write"},
		}

		ctx := context.Background()
		ctx = ContextWithClaims(ctx, claims)

		retrieved := ClaimsFromContext(ctx)
		assert.NotNil(t, retrieved)
		assert.Equal(t, claims.UserID, retrieved.UserID)
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
		claims1 := &Claims{UserID: "user-1"}
		claims2 := &Claims{UserID: "user-2"}

		ctx := context.Background()
		ctx = ContextWithClaims(ctx, claims1)
		ctx = ContextWithClaims(ctx, claims2)

		retrieved := ClaimsFromContext(ctx)
		assert.Equal(t, "user-2", retrieved.UserID)
	})
}

func TestContextWithToken_And_FromContext(t *testing.T) {
	t.Run("store and retrieve token", func(t *testing.T) {
		tokenString := "v4.public.eyJzdWIiOiJ1c2VyLTEyMyJ9..."

		ctx := context.Background()
		ctx = ContextWithToken(ctx, tokenString)

		retrieved := FromContext(ctx)
		assert.Equal(t, tokenString, retrieved)
	})

	t.Run("returns empty string when no token in context", func(t *testing.T) {
		ctx := context.Background()
		retrieved := FromContext(ctx)
		assert.Equal(t, "", retrieved)
	})

	t.Run("store empty token", func(t *testing.T) {
		ctx := context.Background()
		ctx = ContextWithToken(ctx, "")

		retrieved := FromContext(ctx)
		assert.Equal(t, "", retrieved)
	})

	t.Run("overwrite token", func(t *testing.T) {
		ctx := context.Background()
		ctx = ContextWithToken(ctx, "token-1")
		ctx = ContextWithToken(ctx, "token-2")

		retrieved := FromContext(ctx)
		assert.Equal(t, "token-2", retrieved)
	})
}

func TestContextWithBothClaimsAndToken(t *testing.T) {
	claims := &Claims{UserID: "user-123"}
	tokenString := "v4.public.token"

	ctx := context.Background()
	ctx = ContextWithClaims(ctx, claims)
	ctx = ContextWithToken(ctx, tokenString)

	retrievedClaims := ClaimsFromContext(ctx)
	retrievedToken := FromContext(ctx)

	assert.NotNil(t, retrievedClaims)
	assert.Equal(t, "user-123", retrievedClaims.UserID)
	assert.Equal(t, tokenString, retrievedToken)
}
