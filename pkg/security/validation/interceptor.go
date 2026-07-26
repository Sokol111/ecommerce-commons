package validation

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"connectrpc.com/connect"
	"go.uber.org/zap"
)

// ProcedurePermissions maps Connect procedure names to required permission strings.
// Each service provides its own ProcedurePermissions via FX.
type ProcedurePermissions map[string][]string

// NewAuthInterceptor creates a Connect-RPC interceptor that validates bearer
// tokens, stores claims in context, and enforces required permissions per
// procedure. The procedurePermissions map maps procedure names (e.g.
// "/tenant.v1.TenantService/CreateTenant") to required permission strings.
//
// Permission semantics:
//   - nil  → public endpoint, authentication is skipped entirely
//   - []string{} → authenticated, no specific permissions required
//   - []string{"perm"} → authenticated and must have at least one of the listed permissions
func NewAuthInterceptor(
	validator Validator,
	procedurePermissions ProcedurePermissions,
	log *zap.Logger,
) connect.UnaryInterceptorFunc {
	return func(next connect.UnaryFunc) connect.UnaryFunc {
		return func(ctx context.Context, req connect.AnyRequest) (connect.AnyResponse, error) {
			perms, registered := procedurePermissions[req.Spec().Procedure]
			if !registered {
				return nil, connect.NewError(connect.CodePermissionDenied, fmt.Errorf("procedure not registered: %s", req.Spec().Procedure))
			}
			if perms == nil {
				return next(ctx, req)
			}

			auth := req.Header().Get("Authorization")
			token := strings.TrimPrefix(auth, "Bearer ")

			if token == "" {
				return nil, connect.NewError(connect.CodeUnauthenticated, errors.New("missing bearer token"))
			}

			claims, err := validator.ValidateToken(token)
			if err != nil {
				log.Warn("Auth failed",
					zap.String("procedure", req.Spec().Procedure),
					zap.Error(err),
				)
				return nil, connect.NewError(connect.CodeUnauthenticated, fmt.Errorf("invalid token: %w", err))
			}

			if !claims.HasAnyPermission(perms) {
				log.Warn("Permission denied",
					zap.String("procedure", req.Spec().Procedure),
					zap.Strings("required", perms),
					zap.Strings("granted", claims.Permissions),
				)
				return nil, connect.NewError(connect.CodePermissionDenied, fmt.Errorf("missing required permissions: %v", perms))
			}

			ctx = ContextWithClaims(ctx, claims)
			return next(ctx, req)
		}
	}
}
