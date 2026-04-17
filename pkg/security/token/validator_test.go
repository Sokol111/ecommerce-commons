package token

import (
	"crypto/rand"
	"crypto/rsa"
	"encoding/base64"
	"encoding/json"
	"math/big"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// testJWKS creates an RSA key pair and returns the private key and an httptest
// server serving the public key as a JWKS endpoint.
func testJWKS(t *testing.T) (*rsa.PrivateKey, *httptest.Server) {
	t.Helper()
	privateKey, err := rsa.GenerateKey(rand.Reader, 2048)
	require.NoError(t, err)

	jwks := map[string]any{
		"keys": []map[string]any{
			{
				"kty": "RSA",
				"kid": "test-key",
				"use": "sig",
				"alg": "RS256",
				"n":   base64URLUint(privateKey.N),
				"e":   base64URLUint(big.NewInt(int64(privateKey.E))),
			},
		},
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(jwks) //nolint:errcheck
	}))
	t.Cleanup(server.Close)

	return privateKey, server
}

// base64URLUint encodes a big.Int as unpadded base64url (for JWK "n" and "e").
func base64URLUint(n *big.Int) string {
	return base64.RawURLEncoding.EncodeToString(n.Bytes())
}

// createTestJWT creates a signed JWT for testing.
func createTestJWT(privateKey *rsa.PrivateKey, subject, role string, permissions []string, expiry time.Time) string {
	claims := jwt.MapClaims{
		"sub":         subject,
		"role":        role,
		"permissions": permissions,
		"iat":         jwt.NewNumericDate(time.Now()),
		"exp":         jwt.NewNumericDate(expiry),
		"nbf":         jwt.NewNumericDate(time.Now().Add(-1 * time.Minute)),
	}
	token := jwt.NewWithClaims(jwt.SigningMethodRS256, claims)
	token.Header["kid"] = "test-key"
	signed, err := token.SignedString(privateKey)
	if err != nil {
		panic(err)
	}
	return signed
}

func TestNewTokenValidator(t *testing.T) {
	_, server := testJWKS(t)

	t.Run("valid JWKS URL", func(t *testing.T) {
		cfg := Config{JwksURL: server.URL}
		validator, err := newTokenValidator(cfg)

		assert.NoError(t, err)
		assert.NotNil(t, validator)
	})

	t.Run("invalid JWKS URL - tokens fail validation", func(t *testing.T) {
		cfg := Config{JwksURL: "http://localhost:1/nonexistent"}
		validator, err := newTokenValidator(cfg)

		// keyfunc creates the validator even if initial fetch fails (lazy refresh).
		// But token validation will fail because no keys are available.
		assert.NoError(t, err)
		assert.NotNil(t, validator)

		privateKey, _ := rsa.GenerateKey(rand.Reader, 2048)
		tokenString := createTestJWT(privateKey, "user-1", "admin", nil, time.Now().Add(1*time.Hour))
		claims, validErr := validator.ValidateToken(tokenString)
		assert.ErrorIs(t, validErr, ErrInvalidToken)
		assert.Nil(t, claims)
	})

	t.Run("validator works with valid token", func(t *testing.T) {
		privateKey, svr := testJWKS(t)
		cfg := Config{JwksURL: svr.URL}
		validator, err := newTokenValidator(cfg)
		require.NoError(t, err)

		tokenString := createTestJWT(privateKey, "user-123", "admin", []string{"read", "write"}, time.Now().Add(1*time.Hour))
		claims, err := validator.ValidateToken(tokenString)

		assert.NoError(t, err)
		assert.NotNil(t, claims)
	})
}

func TestTokenValidator_ValidateToken(t *testing.T) {
	privateKey, server := testJWKS(t)

	cfg := Config{JwksURL: server.URL}
	validator, err := newTokenValidator(cfg)
	require.NoError(t, err)

	t.Run("valid token with all claims", func(t *testing.T) {
		permissions := []string{"read", "write", "delete"}
		tokenString := createTestJWT(privateKey, "user-456", "super_admin", permissions, time.Now().Add(1*time.Hour))

		claims, err := validator.ValidateToken(tokenString)

		assert.NoError(t, err)
		require.NotNil(t, claims)
		assert.Equal(t, "super_admin", claims.Role)
		assert.Equal(t, permissions, claims.Permissions)
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
		differentKey, err := rsa.GenerateKey(rand.Reader, 2048)
		require.NoError(t, err)
		tokenString := createTestJWT(differentKey, "user-123", "admin", nil, time.Now().Add(1*time.Hour))

		claims, validErr := validator.ValidateToken(tokenString)

		assert.ErrorIs(t, validErr, ErrInvalidToken)
		assert.Nil(t, claims)
	})

	t.Run("token without subject", func(t *testing.T) {
		claims := jwt.MapClaims{
			"role": "admin",
			"iat":  jwt.NewNumericDate(time.Now()),
			"exp":  jwt.NewNumericDate(time.Now().Add(1 * time.Hour)),
		}
		token := jwt.NewWithClaims(jwt.SigningMethodRS256, claims)
		token.Header["kid"] = "test-key"
		tokenString, err := token.SignedString(privateKey)
		require.NoError(t, err)

		result, validErr := validator.ValidateToken(tokenString)

		assert.ErrorIs(t, validErr, ErrInvalidToken)
		assert.Nil(t, result)
	})

	t.Run("token with empty permissions", func(t *testing.T) {
		tokenString := createTestJWT(privateKey, "user-123", "viewer", []string{}, time.Now().Add(1*time.Hour))

		claims, err := validator.ValidateToken(tokenString)

		assert.NoError(t, err)
		require.NotNil(t, claims)
		assert.Empty(t, claims.Permissions)
	})
}
