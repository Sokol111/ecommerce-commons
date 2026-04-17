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

// newSecurityHandler creates a SecurityHandler with the given validator.
// The validator determines the behavior:
//   - tokenValidator: production PASETO validation
//   - testValidator: base64 JSON tokens for e2e tests
//   - noopValidator: always returns admin claims (bypasses security)
func newSecurityHandler(validator Validator) SecurityHandler {
	return &securityHandler{validator: validator}
}

// HandleBearerAuth validates a token and checks permissions.
// Note: tenant claim validation is NOT done here because ogen executes
// security handlers before the middleware chain, so tenant slug is not
// yet available in the context. Tenant validation is handled by the
// tenant middleware after both security and tenant resolution have run.
func (s *securityHandler) HandleBearerAuth(
	ctx context.Context,
	tokenString string,
	requiredPermissions []string,
) (context.Context, *Claims, error) {
	claims, err := s.validator.ValidateToken(tokenString)
	if err != nil {
		return ctx, nil, err
	}

	// Check permissions if required by the operation
	if len(requiredPermissions) > 0 {
		if !claims.HasAnyPermission(requiredPermissions) {
			return ctx, nil, ErrInsufficientPermissions
		}
	}

	// Store claims in context
	ctx = ContextWithClaims(ctx, claims)

	return ctx, claims, nil
}
