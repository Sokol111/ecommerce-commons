package token

import (
	"encoding/hex"
	"testing"
	"time"

	"aidanwoods.dev/go-paseto"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// generateTestKeyPair generates a new PASETO v4 asymmetric key pair for testing.
func generateTestKeyPair() (paseto.V4AsymmetricSecretKey, paseto.V4AsymmetricPublicKey) {
	secretKey := paseto.NewV4AsymmetricSecretKey()
	publicKey := secretKey.Public()
	return secretKey, publicKey
}

// createTestToken creates a signed PASETO token for testing.
func createTestToken(secretKey paseto.V4AsymmetricSecretKey, subject, role, tokenType string, permissions []string, expiry time.Time) string {
	token := paseto.NewToken()
	token.SetSubject(subject)
	token.SetString("role", role)
	token.SetString("type", tokenType)
	token.Set("permissions", permissions)
	token.SetIssuedAt(time.Now())
	token.SetExpiration(expiry)
	token.SetNotBefore(time.Now().Add(-1 * time.Minute))

	return token.V4Sign(secretKey, nil)
}

func TestNewTokenValidator(t *testing.T) {
	secretKey, publicKey := generateTestKeyPair()
	validPublicKeyHex := hex.EncodeToString(publicKey.ExportBytes())

	tests := []struct {
		name      string
		publicKey string
		wantErr   error
	}{
		{
			name:      "valid public key",
			publicKey: validPublicKeyHex,
			wantErr:   nil,
		},
		{
			name:      "invalid hex encoding",
			publicKey: "not-valid-hex",
			wantErr:   ErrInvalidPublicKey,
		},
		{
			name:      "wrong key length - too short",
			publicKey: hex.EncodeToString([]byte("short")),
			wantErr:   ErrInvalidPublicKey,
		},
		{
			name:      "wrong key length - too long",
			publicKey: hex.EncodeToString(make([]byte, 64)),
			wantErr:   ErrInvalidPublicKey,
		},
		{
			name:      "empty public key",
			publicKey: "",
			wantErr:   ErrInvalidPublicKey,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := Config{PublicKey: tt.publicKey}
			validator, err := newTokenValidator(cfg)

			if tt.wantErr != nil {
				assert.ErrorIs(t, err, tt.wantErr)
				assert.Nil(t, validator)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, validator)
			}
		})
	}

	// Verify the validator actually works with the generated key
	t.Run("validator works with valid token", func(t *testing.T) {
		cfg := Config{PublicKey: validPublicKeyHex}
		validator, err := newTokenValidator(cfg)
		require.NoError(t, err)

		tokenString := createTestToken(secretKey, "user-123", "admin", "access", []string{"read", "write"}, time.Now().Add(1*time.Hour))
		claims, err := validator.ValidateToken(tokenString)

		assert.NoError(t, err)
		assert.NotNil(t, claims)
		assert.Equal(t, "user-123", claims.UserID)
	})
}

func TestTokenValidator_ValidateToken(t *testing.T) {
	secretKey, publicKey := generateTestKeyPair()
	publicKeyHex := hex.EncodeToString(publicKey.ExportBytes())

	cfg := Config{PublicKey: publicKeyHex}
	validator, err := newTokenValidator(cfg)
	require.NoError(t, err)

	t.Run("valid token with all claims", func(t *testing.T) {
		permissions := []string{"read", "write", "delete"}
		tokenString := createTestToken(secretKey, "user-456", "super_admin", "access", permissions, time.Now().Add(1*time.Hour))

		claims, err := validator.ValidateToken(tokenString)

		assert.NoError(t, err)
		require.NotNil(t, claims)
		assert.Equal(t, "user-456", claims.UserID)
		assert.Equal(t, "super_admin", claims.Role)
		assert.Equal(t, "access", claims.Type)
		assert.Equal(t, permissions, claims.Permissions)
		assert.False(t, claims.IsExpired())
		assert.True(t, claims.IsAccess())
	})

	t.Run("valid refresh token", func(t *testing.T) {
		tokenString := createTestToken(secretKey, "user-789", "user", "refresh", nil, time.Now().Add(24*time.Hour))

		claims, err := validator.ValidateToken(tokenString)

		assert.NoError(t, err)
		require.NotNil(t, claims)
		assert.Equal(t, "refresh", claims.Type)
		assert.True(t, claims.IsRefresh())
		assert.False(t, claims.IsAccess())
	})

	t.Run("invalid token format", func(t *testing.T) {
		claims, err := validator.ValidateToken("invalid-token")

		assert.ErrorIs(t, err, ErrInvalidToken)
		assert.Nil(t, claims)
	})

	t.Run("empty token", func(t *testing.T) {
		claims, err := validator.ValidateToken("")

		assert.ErrorIs(t, err, ErrInvalidToken)
		assert.Nil(t, claims)
	})

	t.Run("token signed with different key", func(t *testing.T) {
		differentSecretKey := paseto.NewV4AsymmetricSecretKey()
		tokenString := createTestToken(differentSecretKey, "user-123", "admin", "access", nil, time.Now().Add(1*time.Hour))

		claims, err := validator.ValidateToken(tokenString)

		assert.ErrorIs(t, err, ErrInvalidToken)
		assert.Nil(t, claims)
	})

	t.Run("token without subject", func(t *testing.T) {
		token := paseto.NewToken()
		token.SetString("role", "admin")
		token.SetString("type", "access")
		tokenString := token.V4Sign(secretKey, nil)

		claims, err := validator.ValidateToken(tokenString)

		// GetSubject returns error for missing subject
		assert.ErrorIs(t, err, ErrInvalidToken)
		assert.Nil(t, claims)
	})

	t.Run("token with empty permissions", func(t *testing.T) {
		tokenString := createTestToken(secretKey, "user-123", "viewer", "access", []string{}, time.Now().Add(1*time.Hour))

		claims, err := validator.ValidateToken(tokenString)

		assert.NoError(t, err)
		require.NotNil(t, claims)
		assert.Empty(t, claims.Permissions)
	})

	t.Run("claims token field is set for custom claims", func(t *testing.T) {
		token := paseto.NewToken()
		token.SetSubject("user-123")
		token.SetString("role", "admin")
		token.SetString("type", "access")
		token.SetString("custom_field", "custom_value")
		token.SetIssuedAt(time.Now())
		token.SetExpiration(time.Now().Add(1 * time.Hour))
		tokenString := token.V4Sign(secretKey, nil)

		claims, err := validator.ValidateToken(tokenString)

		assert.NoError(t, err)
		require.NotNil(t, claims)

		// Verify we can access custom claims
		customValue, err := claims.GetString("custom_field")
		assert.NoError(t, err)
		assert.Equal(t, "custom_value", customValue)
	})
}
