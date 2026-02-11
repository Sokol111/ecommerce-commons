package token

import (
	"context"
	"errors"
)

// ErrInsufficientPermissions is returned when user lacks required permissions.
var ErrInsufficientPermissions = errors.New("insufficient permissions")

// SecurityHandler handles bearer token authentication and authorization.
// Different validators provide different behaviors for production vs testing.
type SecurityHandler interface {
	// HandleBearerAuth validates a token and checks permissions.
	// Returns the context with claims stored, the claims, and any error.
	HandleBearerAuth(ctx context.Context, token string, requiredPermissions []string) (context.Context, *Claims, error)
}

// securityHandler validates tokens using a Validator.
type securityHandler struct {
	validator Validator
}

// NewSecurityHandler creates a SecurityHandler with the given validator.
// The validator determines the behavior:
//   - tokenValidator: production PASETO validation
//   - testValidator: base64 JSON tokens for e2e tests
//   - noopValidator: always returns admin claims (bypasses security)
func NewSecurityHandler(validator Validator) SecurityHandler {
	return &securityHandler{validator: validator}
}

// HandleBearerAuth validates a token and checks permissions.
func (s *securityHandler) HandleBearerAuth(
	ctx context.Context,
	tokenString string,
	requiredPermissions []string,
) (context.Context, *Claims, error) {
	claims, err := s.validator.ValidateToken(tokenString)
	if err != nil {
		return ctx, nil, err
	}

	// Ensure it's an access token
	if !claims.IsAccess() {
		return ctx, nil, ErrInvalidToken
	}

	// Check permissions if required by the operation
	if len(requiredPermissions) > 0 {
		if !claims.HasAnyPermission(requiredPermissions) {
			return ctx, nil, ErrInsufficientPermissions
		}
	}

	// Store claims and token in context
	ctx = ContextWithClaims(ctx, claims)
	ctx = ContextWithToken(ctx, tokenString)

	return ctx, claims, nil
}
