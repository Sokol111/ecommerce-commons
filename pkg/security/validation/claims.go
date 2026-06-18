package validation

import (
	"slices"
)

// Claims represents the token claims.
// This is used across all services for authentication.
type Claims struct {
	// Tenant is the tenant slug this token was issued for.
	// Empty for service tokens (they operate cross-tenant).
	Tenant string
	// Role is the user's role (e.g., "super_admin", "catalog_manager", "viewer").
	Role string
	// Permissions is the list of permissions granted to the user.
	Permissions []string
}

// WildcardPermission grants access to all permissions.
const WildcardPermission = "*"

// HasAnyPermission checks if the user has at least one of the required permissions.
// Returns true if permissions is empty (no specific permission required),
// if the user holds the wildcard permission, or if the user has any of the listed permissions.
func (c *Claims) HasAnyPermission(permissions []string) bool {
	if len(permissions) == 0 {
		return true
	}
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

// IsTenantScoped returns true if the token is bound to a specific tenant.
// Returns false for service accounts and platform admins (they operate cross-tenant).
func (c *Claims) IsTenantScoped() bool {
	return c.Tenant != ""
}
