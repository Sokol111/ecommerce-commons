package token

import (
	"context"
	"encoding/hex"
	"testing"
	"time"

	"aidanwoods.dev/go-paseto"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// mockValidator is a test double for TokenValidator.
type mockValidator struct {
	claims *Claims
	err    error
}

func (m *mockValidator) ValidateToken(_ string) (*Claims, error) {
	return m.claims, m.err
}

func TestHandleBearerAuth(t *testing.T) {
	t.Run("successful authentication without required permissions", func(t *testing.T) {
		claims := &Claims{
			UserID:      "user-123",
			Role:        "admin",
			Permissions: []string{"read", "write"},
			Type:        "access",
		}
		validator := &mockValidator{claims: claims}
		ctx := context.Background()

		resultCtx, resultClaims, err := HandleBearerAuth(validator, ctx, "valid-token", nil)

		assert.NoError(t, err)
		assert.Equal(t, claims, resultClaims)

		// Verify claims are stored in context
		storedClaims := ClaimsFromContext(resultCtx)
		assert.Equal(t, claims, storedClaims)

		// Verify token is stored in context
		storedToken := TokenFromContext(resultCtx)
		assert.Equal(t, "valid-token", storedToken)
	})

	t.Run("successful authentication with matching permission", func(t *testing.T) {
		claims := &Claims{
			UserID:      "user-123",
			Permissions: []string{"catalog:read", "catalog:write"},
			Type:        "access",
		}
		validator := &mockValidator{claims: claims}
		ctx := context.Background()

		resultCtx, resultClaims, err := HandleBearerAuth(validator, ctx, "valid-token", []string{"catalog:read"})

		assert.NoError(t, err)
		assert.Equal(t, claims, resultClaims)
		assert.NotNil(t, ClaimsFromContext(resultCtx))
	})

	t.Run("successful authentication with one of multiple required permissions", func(t *testing.T) {
		claims := &Claims{
			UserID:      "user-123",
			Permissions: []string{"catalog:write"},
			Type:        "access",
		}
		validator := &mockValidator{claims: claims}
		ctx := context.Background()

		resultCtx, resultClaims, err := HandleBearerAuth(validator, ctx, "valid-token", []string{"catalog:read", "catalog:write"})

		assert.NoError(t, err)
		assert.Equal(t, claims, resultClaims)
		assert.NotNil(t, ClaimsFromContext(resultCtx))
	})

	t.Run("fails when token validation fails", func(t *testing.T) {
		validator := &mockValidator{err: ErrInvalidToken}
		ctx := context.Background()

		resultCtx, resultClaims, err := HandleBearerAuth(validator, ctx, "invalid-token", nil)

		assert.ErrorIs(t, err, ErrInvalidToken)
		assert.Nil(t, resultClaims)
		// Original context should be returned
		assert.Nil(t, ClaimsFromContext(resultCtx))
	})

	t.Run("fails when token is not access type", func(t *testing.T) {
		claims := &Claims{
			UserID: "user-123",
			Type:   "refresh", // Not an access token
		}
		validator := &mockValidator{claims: claims}
		ctx := context.Background()

		resultCtx, resultClaims, err := HandleBearerAuth(validator, ctx, "refresh-token", nil)

		assert.ErrorIs(t, err, ErrInvalidToken)
		assert.Nil(t, resultClaims)
		assert.Nil(t, ClaimsFromContext(resultCtx))
	})

	t.Run("fails when user lacks required permissions", func(t *testing.T) {
		claims := &Claims{
			UserID:      "user-123",
			Permissions: []string{"catalog:read"},
			Type:        "access",
		}
		validator := &mockValidator{claims: claims}
		ctx := context.Background()

		resultCtx, resultClaims, err := HandleBearerAuth(validator, ctx, "valid-token", []string{"catalog:delete", "admin:manage"})

		assert.ErrorIs(t, err, ErrInsufficientPermissions)
		assert.Nil(t, resultClaims)
		assert.Nil(t, ClaimsFromContext(resultCtx))
	})

	t.Run("fails when user has no permissions but permissions are required", func(t *testing.T) {
		claims := &Claims{
			UserID:      "user-123",
			Permissions: nil,
			Type:        "access",
		}
		validator := &mockValidator{claims: claims}
		ctx := context.Background()

		resultCtx, resultClaims, err := HandleBearerAuth(validator, ctx, "valid-token", []string{"any:permission"})

		assert.ErrorIs(t, err, ErrInsufficientPermissions)
		assert.Nil(t, resultClaims)
		assert.Nil(t, ClaimsFromContext(resultCtx))
	})

	t.Run("succeeds with empty token type when checking IsAccess", func(t *testing.T) {
		claims := &Claims{
			UserID: "user-123",
			Type:   "", // Empty type
		}
		validator := &mockValidator{claims: claims}
		ctx := context.Background()

		_, resultClaims, err := HandleBearerAuth(validator, ctx, "token", nil)

		// Empty type means IsAccess() returns false
		assert.ErrorIs(t, err, ErrInvalidToken)
		assert.Nil(t, resultClaims)
	})
}

func TestHandleBearerAuth_Integration(t *testing.T) {
	// Integration test with real token validator
	secretKey := paseto.NewV4AsymmetricSecretKey()
	publicKey := secretKey.Public()
	publicKeyHex := hex.EncodeToString(publicKey.ExportBytes())

	cfg := Config{PublicKey: publicKeyHex}
	validator, err := newTokenValidator(cfg)
	require.NoError(t, err)

	t.Run("full flow with real validator", func(t *testing.T) {
		// Create a real token
		token := paseto.NewToken()
		token.SetSubject("user-integration")
		token.SetString("role", "catalog_manager")
		token.SetString("type", "access")
		token.Set("permissions", []string{"catalog:read", "catalog:write"})
		token.SetIssuedAt(time.Now())
		token.SetExpiration(time.Now().Add(1 * time.Hour))
		tokenString := token.V4Sign(secretKey, nil)

		ctx := context.Background()
		resultCtx, claims, err := HandleBearerAuth(validator, ctx, tokenString, []string{"catalog:write"})

		assert.NoError(t, err)
		require.NotNil(t, claims)
		assert.Equal(t, "user-integration", claims.UserID)
		assert.Equal(t, "catalog_manager", claims.Role)

		// Verify context
		storedClaims := ClaimsFromContext(resultCtx)
		assert.Equal(t, "user-integration", storedClaims.UserID)

		storedToken := TokenFromContext(resultCtx)
		assert.Equal(t, tokenString, storedToken)
	})
}
