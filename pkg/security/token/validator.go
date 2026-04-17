package token

import (
	"github.com/MicahParks/keyfunc/v3"
	"github.com/golang-jwt/jwt/v5"
)

// Validator validates tokens and returns claims.
type Validator interface {
	// ValidateToken validates a token and returns the claims.
	ValidateToken(token string) (*Claims, error)
}

// jwtValidator validates JWT tokens using a JWKS endpoint.
// This is intended for use in microservices that need to validate tokens
// issued by Zitadel (or any OIDC provider).
type jwtValidator struct {
	jwks keyfunc.Keyfunc
}

// jwtClaims is the JWT claims structure expected from Zitadel tokens.
type jwtClaims struct {
	jwt.RegisteredClaims
	Role        string   `json:"role"`
	Tenant      string   `json:"tenant"`
	Permissions []string `json:"permissions"`
}

// newTokenValidator creates a new JWT validator that fetches keys from a JWKS endpoint.
func newTokenValidator(config Config) (Validator, error) {
	jwks, err := keyfunc.NewDefault([]string{config.JwksURL})
	if err != nil {
		return nil, ErrInvalidPublicKey
	}

	return &jwtValidator{
		jwks: jwks,
	}, nil
}

// ValidateToken validates a JWT and returns the claims.
func (v *jwtValidator) ValidateToken(tokenString string) (*Claims, error) {
	token, err := jwt.ParseWithClaims(tokenString, &jwtClaims{}, v.jwks.Keyfunc,
		jwt.WithValidMethods([]string{"RS256"}),
	)
	if err != nil || !token.Valid {
		return nil, ErrInvalidToken
	}

	c, ok := token.Claims.(*jwtClaims)
	if !ok || c.Subject == "" {
		return nil, ErrInvalidToken
	}

	return &Claims{
		Tenant:      c.Tenant,
		Role:        c.Role,
		Permissions: c.Permissions,
	}, nil
}
