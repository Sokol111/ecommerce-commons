package validation

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"connectrpc.com/connect"
	"github.com/Sokol111/ecommerce-commons/pkg/http/connect/interceptor"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

// AuthInterceptorPriority is the recommended priority for the auth interceptor.
// It must run AFTER tenant resolution (18) but BEFORE tenant validation (26)
// so that claims are available for tenant validation.
const AuthInterceptorPriority = 22

// ProcedurePermissions maps Connect procedure names to required permission strings.
// Each service provides its own ProcedurePermissions via FX.
type ProcedurePermissions map[string][]string

// newAuthInterceptorModule provides the auth Connect interceptor via FX.
// It expects ProcedurePermissions to be provided by the service module.
func newAuthInterceptorModule() fx.Option {
	return fx.Provide(
		fx.Annotate(
			func(
				validator Validator,
				perms ProcedurePermissions,
				log *zap.Logger,
			) interceptor.Interceptor {
				return interceptor.Interceptor{
					Priority: AuthInterceptorPriority,
					Handler:  NewAuthInterceptor(validator, perms, log),
				}
			},
			fx.ResultTags(`group:"connect_interceptor"`),
		),
	)
}

// NewAuthInterceptor creates a Connect-RPC interceptor that validates bearer
// tokens, stores claims in context, and enforces required permissions per
// procedure. The procedurePermissions map maps procedure names (e.g.
// "/tenant.v1.TenantService/CreateTenant") to required permission strings.
func NewAuthInterceptor(
	validator Validator,
	procedurePermissions ProcedurePermissions,
	log *zap.Logger,
) connect.UnaryInterceptorFunc {
	return func(next connect.UnaryFunc) connect.UnaryFunc {
		return func(ctx context.Context, req connect.AnyRequest) (connect.AnyResponse, error) {
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

			perms := procedurePermissions[req.Spec().Procedure]
			if len(perms) > 0 && !claims.HasAnyPermission(perms) {
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
