package token

import (
	"context"
	"errors"
)

// ErrInsufficientPermissions is returned when user lacks required permissions.
var ErrInsufficientPermissions = errors.New("insufficient permissions")

// HandleBearerAuth validates a token and checks permissions.
// It returns the context with claims stored, the claims, and any error.
//
// Usage in your service:
//
//	func (s *securityHandler) HandleBearerAuth(ctx context.Context, operationName httpapi.OperationName, t httpapi.BearerAuth) (context.Context, error) {
//	    ctx, _, err := token.HandleBearerAuth(s.validator, ctx, t.Token, t.Roles)
//	    return ctx, err
//	}
func HandleBearerAuth(
	validator TokenValidator,
	ctx context.Context,
	tokenString string,
	requiredPermissions []string,
) (context.Context, *Claims, error) {
	claims, err := validator.ValidateToken(tokenString)
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

	// Store claims in context
	return ContextWithClaims(ctx, claims), claims, nil
}
