package token

import (
	"slices"
	"time"

	"aidanwoods.dev/go-paseto"
)

// Claims represents the token claims.
// This is used across all services for authentication.
type Claims struct {
	// UserID is the unique identifier of the user (subject).
	UserID string
	// Role is the user's role (e.g., "super_admin", "catalog_manager", "viewer").
	Role string
	// Permissions is the list of permissions granted to the user.
	Permissions []string
	// Type is the token type (e.g., "access", "refresh").
	Type string
	// IssuedAt is the time when the token was issued.
	IssuedAt time.Time
	// ExpiresAt is the time when the token expires.
	ExpiresAt time.Time
	// NotBefore is the time before which the token is not valid.
	NotBefore time.Time

	// token holds the underlying PASETO token for custom claims access.
	token *paseto.Token
}

// WildcardPermission grants access to all permissions.
const WildcardPermission = "*"

// HasPermission checks if the user has a specific permission.
// Returns true if the user has the exact permission or the wildcard permission.
func (c *Claims) HasPermission(permission string) bool {
	return slices.Contains(c.Permissions, WildcardPermission) ||
		slices.Contains(c.Permissions, permission)
}

// HasAnyPermission checks if the user has at least one of the required permissions.
// Returns true if the user has any of the specified permissions or the wildcard permission.
func (c *Claims) HasAnyPermission(permissions []string) bool {
	// Wildcard grants all permissions
	if slices.Contains(c.Permissions, WildcardPermission) {
		return true
	}
	for _, perm := range permissions {
		if slices.Contains(c.Permissions, perm) {
			return true
		}
	}
	return false
}

// GetString returns a custom string claim from the token.
func (c *Claims) GetString(key string) (string, error) {
	if c.token == nil {
		return "", ErrInvalidToken
	}
	return c.token.GetString(key)
}

// Get unmarshals a custom claim from the token into the provided value.
func (c *Claims) Get(key string, v any) error {
	if c.token == nil {
		return ErrInvalidToken
	}
	return c.token.Get(key, v)
}

// IsExpired checks if the token has expired.
func (c *Claims) IsExpired() bool {
	return time.Now().After(c.ExpiresAt)
}

// IsAccess returns true if the token is an access token.
func (c *Claims) IsAccess() bool {
	return c.Type == "access"
}

// IsRefresh returns true if the token is a refresh token.
func (c *Claims) IsRefresh() bool {
	return c.Type == "refresh"
}
