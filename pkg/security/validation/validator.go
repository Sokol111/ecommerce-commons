package validation

import (
	"errors"
	"strings"

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
// issued by an OIDC provider (e.g. Logto).
type jwtValidator struct {
	jwks     keyfunc.Keyfunc
	audience string
}

// jwtClaims is the JWT claims structure expected from the OIDC provider.
// Standard OIDC scopes are in the "scope" claim (space-separated).
// Custom claims "role" and "tenant" are injected via custom JWT claims script.
type jwtClaims struct {
	jwt.RegisteredClaims
	Role   string `json:"role"`
	Tenant string `json:"tenant"`
	Scope  string `json:"scope"`
}

// oidcScopes are standard OIDC scopes that should be filtered out from permissions.
var oidcScopes = map[string]bool{
	"openid":         true,
	"profile":        true,
	"email":          true,
	"offline_access": true,
	"all":            true,
}

// newTokenValidator creates a new JWT validator that fetches keys from a JWKS endpoint.
func newTokenValidator(config Config) (Validator, error) {
	jwks, err := keyfunc.NewDefault([]string{config.JwksURL})
	if err != nil {
		return nil, errors.New("invalid public key")
	}

	return &jwtValidator{
		jwks:     jwks,
		audience: config.Audience,
	}, nil
}

// ValidateToken validates a JWT and returns the claims.
func (v *jwtValidator) ValidateToken(tokenString string) (*Claims, error) {
	parserOpts := []jwt.ParserOption{
		jwt.WithValidMethods([]string{"RS256", "ES256", "ES384"}),
	}
	if v.audience != "" {
		parserOpts = append(parserOpts, jwt.WithAudience(v.audience))
	}

	token, err := jwt.ParseWithClaims(tokenString, &jwtClaims{}, v.jwks.Keyfunc, parserOpts...)
	if err != nil || !token.Valid {
		return nil, errors.New("invalid token")
	}

	c, ok := token.Claims.(*jwtClaims)
	if !ok || c.Subject == "" {
		return nil, errors.New("invalid token claims")
	}

	return &Claims{
		Tenant:      c.Tenant,
		Role:        c.Role,
		Permissions: parseScopes(c.Scope),
	}, nil
}

// parseScopes splits the space-separated scope string and filters out standard OIDC scopes.
func parseScopes(scope string) []string {
	if scope == "" {
		return nil
	}
	parts := strings.Split(scope, " ")
	permissions := make([]string, 0, len(parts))
	for _, s := range parts {
		if s != "" && !oidcScopes[s] {
			permissions = append(permissions, s)
		}
	}
	return permissions
}
