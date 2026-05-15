package validation

import "errors"

var (
	// ErrInvalidToken is returned when the token cannot be parsed or verified.
	ErrInvalidToken = errors.New("invalid token")
	// ErrTenantMismatch is returned when the token's tenant claim doesn't match the request tenant.
	ErrTenantMismatch = errors.New("token tenant mismatch")
	// ErrInvalidPublicKey is returned when the public key is invalid.
	ErrInvalidPublicKey = errors.New("invalid public key")
	// ErrInsufficientPermissions is returned when user lacks required permissions.
	ErrInsufficientPermissions = errors.New("insufficient permissions")
)
